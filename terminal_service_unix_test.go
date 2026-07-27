//go:build !windows

package main

// Unix-only terminal integration tests: they spawn a real PTY through the unix
// ptyStart helper and rely on POSIX shell syntax ("-c", "sleep 60") and
// signals. The windows ConPTY backend is exercised through the ptyBackend seam
// instead.

import (
	"os"
	"syscall"
	"testing"
)

// TestTerminalShutdown verifies Stop() kills the shell process and clears state.
func TestTerminalShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	shellPath, shellFlag := detectShell()

	s := newTestTerminalService(t)
	id := mustCreateAndStart(t, s)

	ss, _ := s.resolveSession(id)
	ss.mu.Lock()
	ss.shellPath = shellPath
	ss.shellFlag = shellFlag
	ss.mu.Unlock()

	s.Stop(id)
	ptmx, c, err := ptyStart(shellPath, shellFlag, "", 24, 80, "-c", "sleep 60")
	if err != nil {
		t.Fatalf("ptyStart failed: %v", err)
	}

	ss.mu.Lock()
	ss.ptmx = ptmx
	ss.proc = newExecProcess(c)
	ss.stopCh = make(chan struct{})
	ss.running = true
	pid := ss.proc.Pid()
	ss.mu.Unlock()

	proc, err := os.FindProcess(pid)
	if err != nil {
		t.Fatalf("FindProcess failed: %v", err)
	}

	s.Stop(id)

	ss.mu.Lock()
	defer ss.mu.Unlock()
	if ss.ptmx != nil {
		t.Error("ptmx not cleared after Stop")
	}
	if ss.proc != nil {
		t.Error("proc not cleared after Stop")
	}

	err = proc.Signal(syscall.Signal(0))
	if err == nil {
		t.Error("process still running after Stop")
	}
}

// TestTerminalExit verifies shell exit triggers monitorExit flow.
func TestTerminalExit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	s := newTestTerminalService(t)
	id := mustCreateAndStart(t, s)

	ss, _ := s.resolveSession(id)
	shellPath, shellFlag := detectShell()

	s.Stop(id)
	ptmx, cmd, err := ptyStart(shellPath, shellFlag, "", 24, 80, "-c", "exit 0")
	if err != nil {
		t.Fatalf("ptyStart failed: %v", err)
	}

	ss.mu.Lock()
	ss.shellPath = shellPath
	ss.shellFlag = shellFlag
	ss.lastSize = ptyWinsize{Rows: 24, Cols: 80}
	ss.ptmx = ptmx
	proc := newExecProcess(cmd)
	ss.proc = proc
	ss.stopCh = make(chan struct{})
	ss.running = true
	ss.mu.Unlock()

	_, _ = proc.Wait()
	ptmx.Close()

	go s.monitorExit(ss, proc, ptmx, ss.stopCh)
	defer s.Stop(id)

	if !proc.Exited() {
		t.Error("process not reaped after shell exit")
	}
}
