package main

import (
	"io"
	"os/exec"
	"sync"
	"time"
)

// ptyBackend is the seam between TerminalService and the OS PTY layer.
// It exists so the OS-specific implementation (creack/pty on darwin/linux,
// conpty on windows) can be swapped at test time via the darwin mock.
type ptyBackend interface {
	Start(shellPath, shellFlag, workingDir string, rows, cols int) (ptyHandle, ptyProcess, error)
	Resize(handle ptyHandle, cols, rows int) error
}

// ptyHandle is the I/O surface a started PTY exposes to TerminalService.
// *os.File satisfies this interface (it implements io.ReadWriteCloser), so
// the production code path is unchanged for darwin; the darwin-side mock
// also satisfies it via mockPtyHandle.
type ptyHandle interface {
	io.ReadWriteCloser
}

// ptyProcess is the process half of a started PTY. Windows ConPTY clients are
// spawned with CreateProcess + STARTUPINFOEX, which os/exec cannot express, so
// the seam cannot hand TerminalService an *exec.Cmd — it gets this instead.
type ptyProcess interface {
	// Wait blocks until the process exits and reports its exit code. It is
	// safe to call concurrently and more than once; every caller observes the
	// same result.
	Wait() (int, error)
	// Exited reports whether the process has already been reaped.
	Exited() bool
	// Kill terminates the process and its descendants, then waits for it to
	// be reaped.
	Kill() error
	// Pid returns the OS process id.
	Pid() int
}

// killGracePeriod is how long a process gets to go away after a polite
// terminate before it is force-killed.
const killGracePeriod = 2 * time.Second

// execProcess adapts an *exec.Cmd to ptyProcess. It owns the single Wait on
// the command: the reaping goroutine started by newExecProcess is the only
// caller of cmd.Wait, so exit monitoring and Kill can both observe the result
// without racing for it.
type execProcess struct {
	cmd  *exec.Cmd
	done chan struct{}

	mu       sync.Mutex
	exited   bool
	exitCode int
	waitErr  error
}

func newExecProcess(cmd *exec.Cmd) *execProcess {
	p := &execProcess{cmd: cmd, done: make(chan struct{})}
	go func() {
		err := cmd.Wait()
		p.mu.Lock()
		p.exited = true
		p.exitCode = exitCodeFromWait(err)
		p.waitErr = err
		p.mu.Unlock()
		close(p.done)
	}()
	return p
}

func (p *execProcess) Wait() (int, error) {
	<-p.done
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.exitCode, p.waitErr
}

func (p *execProcess) Exited() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.exited
}

func (p *execProcess) Pid() int {
	if p.cmd == nil || p.cmd.Process == nil {
		return 0
	}
	return p.cmd.Process.Pid
}

func (p *execProcess) Kill() error {
	if p.cmd == nil || p.cmd.Process == nil {
		return nil
	}
	if p.Exited() {
		return nil
	}

	_ = hangupProcessGroup(p.cmd)

	select {
	case <-p.done:
		return nil
	case <-time.After(killGracePeriod):
		_ = killProcessGroup(p.cmd)
		<-p.done
		return nil
	}
}

// exitCodeFromWait maps the error returned by (*exec.Cmd).Wait to an exit code.
// A non-ExitError failure (the process could not be reaped at all) reports -1,
// matching the convention used by ExecutionRecord.
func exitCodeFromWait(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return -1
}
