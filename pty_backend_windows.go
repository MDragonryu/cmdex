//go:build windows

package main

// Windows pseudo console (ConPTY) backend. The client shell is spawned with
// CreateProcess + STARTUPINFOEX carrying PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE,
// which os/exec cannot express — that is why ptyBackend hands TerminalService a
// ptyProcess rather than an *exec.Cmd.
//
// ConPTY requires Windows 10 1809 (build 17763) or newer; on older builds
// CreatePseudoConsole fails and the error surfaces through CreateSession.

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// newPtyBackend returns the ConPTY-backed ptyBackend for windows.
func newPtyBackend() ptyBackend {
	return conptyBackend{}
}

// conptyBackend is the windows ConPTY ptyBackend implementation.
type conptyBackend struct{}

// Start creates a pseudo console sized to rows/cols and attaches a freshly
// spawned shell to it.
func (conptyBackend) Start(shellPath, shellFlag, workingDir string, rows, cols int) (ptyHandle, ptyProcess, error) {
	return startConpty(shellPath, shellFlag, workingDir, rows, cols)
}

// Resize updates the pseudo console buffer dimensions.
func (conptyBackend) Resize(handle ptyHandle, cols, rows int) error {
	h, ok := handle.(*conptyHandle)
	if !ok {
		return fmt.Errorf("conptyBackend.Resize: unexpected handle type %T", handle)
	}
	return h.resize(cols, rows)
}

// conptyHandle is the ptyHandle for a pseudo console: writes go to the client's
// stdin, reads drain the client's (VT-encoded) output.
type conptyHandle struct {
	mu     sync.Mutex
	hpc    windows.Handle
	in     *os.File
	out    *os.File
	closed bool
}

func (h *conptyHandle) Read(p []byte) (int, error) {
	return h.out.Read(p)
}

func (h *conptyHandle) Write(p []byte) (int, error) {
	return h.in.Write(p)
}

// Close tears the pseudo console down. ClosePseudoConsole blocks until the
// ConPTY has flushed its remaining output, so it runs on its own goroutine:
// the reader is still draining h.out at this point and must not be waiting on
// the same goroutine that unblocks it.
func (h *conptyHandle) Close() error {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return nil
	}
	h.closed = true
	hpc, in, out := h.hpc, h.in, h.out
	h.mu.Unlock()

	// Closing the input side signals EOF to the shell, which is enough for
	// well-behaved shells to exit on their own.
	_ = in.Close()

	go func() {
		windows.ClosePseudoConsole(hpc)
		_ = out.Close()
	}()
	return nil
}

func (h *conptyHandle) resize(cols, rows int) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return fmt.Errorf("conpty: pseudo console already closed")
	}
	return windows.ResizePseudoConsole(h.hpc, windows.Coord{X: int16(cols), Y: int16(rows)})
}

// conptyProcess is the ptyProcess for a ConPTY client. It owns the process
// handle and reaps the shell on a dedicated goroutine so Wait is idempotent.
type conptyProcess struct {
	pid  int
	done chan struct{}

	mu       sync.Mutex
	handle   windows.Handle
	exited   bool
	exitCode int
	waitErr  error
}

func newConptyProcess(handle windows.Handle, pid int) *conptyProcess {
	p := &conptyProcess{pid: pid, handle: handle, done: make(chan struct{})}

	go func() {
		code, err := waitForProcess(handle)

		p.mu.Lock()
		p.exited = true
		p.exitCode = code
		p.waitErr = err
		// The reaping goroutine is the last user of the handle: Kill only
		// touches it while p.exited is false, under this same lock.
		_ = windows.CloseHandle(p.handle)
		p.handle = windows.InvalidHandle
		p.mu.Unlock()

		close(p.done)
	}()

	return p
}

func waitForProcess(handle windows.Handle) (int, error) {
	event, err := windows.WaitForSingleObject(handle, windows.INFINITE)
	if err != nil {
		return -1, fmt.Errorf("WaitForSingleObject: %w", err)
	}
	if event != windows.WAIT_OBJECT_0 {
		return -1, fmt.Errorf("WaitForSingleObject: unexpected event 0x%x", event)
	}

	var code uint32
	if err := windows.GetExitCodeProcess(handle, &code); err != nil {
		return -1, fmt.Errorf("GetExitCodeProcess: %w", err)
	}
	return int(int32(code)), nil
}

func (p *conptyProcess) Wait() (int, error) {
	<-p.done
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.exitCode, p.waitErr
}

func (p *conptyProcess) Exited() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.exited
}

func (p *conptyProcess) Pid() int {
	return p.pid
}

// Kill terminates the shell and everything it spawned.
//
// Windows has no SIGHUP: the graceful signal is conptyHandle.Close, which every
// caller in TerminalService issues before this. So the grace period is spent
// waiting for the shell to act on that, and only then is the tree forced.
func (p *conptyProcess) Kill() error {
	if p.Exited() {
		return nil
	}

	select {
	case <-p.done:
		return nil
	case <-time.After(killGracePeriod):
	}

	_ = taskkillTree(p.pid)

	// taskkill can still miss a wedged process; TerminateProcess on the handle
	// we already hold is the last resort.
	p.mu.Lock()
	if !p.exited && p.handle != windows.InvalidHandle {
		_ = windows.TerminateProcess(p.handle, 1)
	}
	p.mu.Unlock()

	<-p.done
	return nil
}

// taskkillTree force-kills a process and its descendants. A shell's children
// are not in a process group the way they are on unix, so the tree has to be
// walked — /T does that.
func taskkillTree(pid int) error {
	return exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(pid)).Run()
}

// startConpty creates a pseudo console and attaches a shell to it. extraArgs
// are appended after shellFlag.
func startConpty(shellPath, shellFlag, workingDir string, rows, cols int, extraArgs ...string) (ptyHandle, ptyProcess, error) {
	if cols < 1 {
		cols = 80
	}
	if rows < 1 {
		rows = 24
	}

	// The ConPTY reads the client's input from inRead and writes its output to
	// outWrite; we keep the opposite end of each pipe.
	var inRead, inWrite windows.Handle
	if err := windows.CreatePipe(&inRead, &inWrite, nil, 0); err != nil {
		return nil, nil, fmt.Errorf("conpty: CreatePipe (input): %w", err)
	}

	var outRead, outWrite windows.Handle
	if err := windows.CreatePipe(&outRead, &outWrite, nil, 0); err != nil {
		_ = windows.CloseHandle(inRead)
		_ = windows.CloseHandle(inWrite)
		return nil, nil, fmt.Errorf("conpty: CreatePipe (output): %w", err)
	}

	var hpc windows.Handle
	size := windows.Coord{X: int16(cols), Y: int16(rows)}
	err := windows.CreatePseudoConsole(size, inRead, outWrite, 0, &hpc)

	// CreatePseudoConsole duplicates the ends it needs, so ours are released
	// either way — leaving outWrite open here would keep the output pipe from
	// ever reporting EOF.
	_ = windows.CloseHandle(inRead)
	_ = windows.CloseHandle(outWrite)

	if err != nil {
		_ = windows.CloseHandle(inWrite)
		_ = windows.CloseHandle(outRead)
		return nil, nil, fmt.Errorf("conpty: CreatePseudoConsole (requires Windows 10 1809+): %w", err)
	}

	proc, err := spawnConptyClient(hpc, shellPath, shellFlag, workingDir, extraArgs)
	if err != nil {
		windows.ClosePseudoConsole(hpc)
		_ = windows.CloseHandle(inWrite)
		_ = windows.CloseHandle(outRead)
		return nil, nil, err
	}

	return &conptyHandle{
		hpc: hpc,
		in:  os.NewFile(uintptr(inWrite), "conpty-input"),
		out: os.NewFile(uintptr(outRead), "conpty-output"),
	}, proc, nil
}

// spawnConptyClient starts the shell attached to hpc.
func spawnConptyClient(hpc windows.Handle, shellPath, shellFlag, workingDir string, extraArgs []string) (*conptyProcess, error) {
	argv := []string{shellPath}
	if shellFlag != "" {
		argv = append(argv, shellFlag)
	}
	argv = append(argv, extraArgs...)

	// appName stays nil so CreateProcess resolves bare names like "cmd"
	// against PATH the way exec.LookPath would.
	commandLine, err := windows.UTF16PtrFromString(windows.ComposeCommandLine(argv))
	if err != nil {
		return nil, fmt.Errorf("conpty: command line for %q: %w", shellPath, err)
	}

	var currentDir *uint16
	if dir := resolvePtyDir(workingDir); dir != "" {
		currentDir, err = windows.UTF16PtrFromString(dir)
		if err != nil {
			return nil, fmt.Errorf("conpty: working directory %q: %w", dir, err)
		}
	}

	envBlock, err := utf16EnvBlock(ptyEnv())
	if err != nil {
		return nil, fmt.Errorf("conpty: environment block: %w", err)
	}

	attrList, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		return nil, fmt.Errorf("conpty: NewProcThreadAttributeList: %w", err)
	}
	defer attrList.Delete()

	// PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE takes the HPCON by value, passed
	// through the lpValue pointer slot rather than as a pointer to it — this is
	// what the Win32 API expects, so `go vet -unsafeptr` flagging the
	// uintptr→unsafe.Pointer conversion here is a false positive.
	if err := attrList.Update(
		windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE,
		unsafe.Pointer(uintptr(hpc)),
		unsafe.Sizeof(hpc),
	); err != nil {
		return nil, fmt.Errorf("conpty: UpdateProcThreadAttribute: %w", err)
	}

	startupInfo := &windows.StartupInfoEx{
		StartupInfo:             windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfoEx{}))},
		ProcThreadAttributeList: attrList.List(),
	}

	var procInfo windows.ProcessInformation
	flags := uint32(windows.EXTENDED_STARTUPINFO_PRESENT | windows.CREATE_UNICODE_ENVIRONMENT)

	err = windows.CreateProcess(
		nil,
		commandLine,
		nil,
		nil,
		false,
		flags,
		envBlock,
		currentDir,
		&startupInfo.StartupInfo,
		&procInfo,
	)
	runtime.KeepAlive(envBlock)
	if err != nil {
		return nil, fmt.Errorf("conpty: CreateProcess %q: %w", shellPath, err)
	}

	_ = windows.CloseHandle(procInfo.Thread)

	return newConptyProcess(procInfo.Process, int(procInfo.ProcessId)), nil
}

// utf16EnvBlock builds the double-NUL-terminated UTF-16 environment block that
// CREATE_UNICODE_ENVIRONMENT expects. A nil result tells CreateProcess to
// inherit the parent environment.
func utf16EnvBlock(env []string) (*uint16, error) {
	if len(env) == 0 {
		return nil, nil
	}

	var block []uint16
	for _, entry := range env {
		if entry == "" {
			continue
		}
		encoded, err := windows.UTF16FromString(entry)
		if err != nil {
			// An embedded NUL makes the entry unrepresentable; dropping it is
			// better than failing the whole shell spawn.
			continue
		}
		block = append(block, encoded...)
	}
	if len(block) == 0 {
		return nil, nil
	}
	block = append(block, 0)

	return &block[0], nil
}

// hangupProcessGroup asks a process tree to terminate gracefully. Windows has
// no signal equivalent for a console-less child, so this asks taskkill nicely
// and lets execProcess.Kill escalate when the process ignores it.
func hangupProcessGroup(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	return exec.Command("taskkill", "/T", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
}

// killProcessGroup force-kills a process tree.
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	return taskkillTree(cmd.Process.Pid)
}
