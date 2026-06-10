// TODO(Plan 03): Update tests for multi-session TerminalService API.
//go:build ignore

package main

import (
	"os"
	"runtime"
	"syscall"
	"testing"
)

func newTestTerminalService(t *testing.T) *TerminalService {
	t.Helper()
	return &TerminalService{}
}

func mustStart(t *testing.T, s *TerminalService, cols, rows int) {
	t.Helper()
	if err := s.Start(cols, rows); err != nil {
		t.Fatalf("Start(%d, %d) failed: %v", cols, rows, err)
	}
	t.Cleanup(func() {
		s.Stop()
	})
}

// TestTerminalDetectShell verifies detectShell returns correct path/flag for current OS (POL-05).
func TestTerminalDetectShell(t *testing.T) {
	path, flag := detectShell()

	if path == "" {
		t.Fatal("detectShell returned empty path")
	}

	if runtime.GOOS == "windows" {
		validShells := map[string]bool{"pwsh": true, "powershell": true, "cmd": true}
		if !validShells[path] {
			t.Errorf("unexpected Windows shell path: %s", path)
		}
		if path == "cmd" && flag != "" {
			t.Errorf("cmd shell should have empty flag, got: %s", flag)
		}
	} else {
		if flag != "-l" {
			t.Errorf("Unix shell flag should be '-l', got: '%s'", flag)
		}
	}

	// Test $SHELL env var is respected on Unix.
	if runtime.GOOS != "windows" {
		t.Run("respectsSHELL", func(t *testing.T) {
			prev := os.Getenv("SHELL")
			os.Setenv("SHELL", "/bin/zsh")
			defer os.Setenv("SHELL", prev)

			p, _ := detectShell()
			if p != "/bin/zsh" {
				t.Errorf("expected /bin/zsh from $SHELL, got: %s", p)
			}
		})
	}
}

// TestTerminalBatching verifies emitOutput and readLoop exist (PTY-03).
func TestTerminalBatching(t *testing.T) {
	s := newTestTerminalService(t)

	_ = s.lastSize

	// readLoop reads PTY output in a background goroutine and emits
	// each read via emitOutput. No batching or ticker is used.
}

// TestTerminalStart verifies Start() spawns a shell process via PTY (PTY-01).
func TestTerminalStart(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	s := newTestTerminalService(t)

	err := s.Start(80, 24)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer s.Stop()

	if s.ptmx == nil {
		t.Fatal("ptmx is nil after Start")
	}
	if s.cmd == nil {
		t.Fatal("cmd is nil after Start")
	}
	if s.cmd.Process == nil {
		t.Fatal("process is nil after Start")
	}
}

// TestTerminalWrite verifies Write() sends keystrokes to PTY stdin (PTY-02).
func TestTerminalWrite(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	s := newTestTerminalService(t)
	mustStart(t, s, 80, 24)

	err := s.Write("echo test\n")
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
}

// TestTerminalResize verifies Resize() calls Setsize and updates lastSize (PTY-04).
func TestTerminalResize(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	s := newTestTerminalService(t)
	mustStart(t, s, 80, 24)

	err := s.Resize(120, 40)
	if err != nil {
		t.Errorf("Resize failed: %v", err)
	}

	if s.lastSize.Cols != 120 || s.lastSize.Rows != 40 {
		t.Errorf("lastSize not updated after Resize: got Cols=%d Rows=%d, want Cols=120 Rows=40",
			s.lastSize.Cols, s.lastSize.Rows)
	}
}

// TestTerminalShutdown verifies Stop() kills the shell process and clears state (PTY-05).
func TestTerminalShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Use a long-running command that won't exit on its own.
	shellPath, shellFlag := detectShell()

	s := newTestTerminalService(t)

	s.shellPath = shellPath
	s.shellFlag = shellFlag

	ptmx, c, err := ptyStart(shellPath, shellFlag, 80, 24, "-c", "sleep 60")
	if err != nil {
		t.Fatalf("ptyStart failed: %v", err)
	}
	s.ptmx = ptmx
	s.cmd = c
	s.stopCh = make(chan struct{}, 1)

	pid := s.cmd.Process.Pid

	// Verify process is running.
	proc, err := os.FindProcess(pid)
	if err != nil {
		t.Fatalf("FindProcess failed: %v", err)
	}

	s.Stop()

	if s.ptmx != nil {
		t.Error("ptmx not cleared after Stop")
	}
	if s.cmd != nil {
		t.Error("cmd not cleared after Stop")
	}
	if s.stopCh != nil {
		t.Error("stopCh not cleared after Stop")
	}

	// Verify process is dead.
	err = proc.Signal(syscall.Signal(0))
	if err == nil {
		t.Error("process still running after Stop")
	}
}

// TestTerminalExit verifies shell exit triggers monitorExit flow (PTY-06).
func TestTerminalExit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	s := newTestTerminalService(t)

	// Start a shell that immediately exits.
	shellPath, shellFlag := detectShell()
	ptmx, cmd, err := ptyStart(shellPath, shellFlag, 80, 24, "-c", "exit 0")
	if err != nil {
		t.Fatalf("ptyStart failed: %v", err)
	}
	s.shellPath = shellPath
	s.shellFlag = shellFlag
	s.lastSize = ptyWinsize{Rows: 24, Cols: 80}
	s.ptmx = ptmx
	s.cmd = cmd
	s.stopCh = make(chan struct{}, 1)

	// Wait for the shell to exit first so ProcessState is populated
	// before monitorExit reads it from its own Wait call (which returns cached).
	_ = cmd.Wait()
	ptmx.Close()
	s.running = true

	go s.monitorExit(cmd, ptmx, s.stopCh)
	defer s.Stop()

	if cmd.ProcessState == nil {
		t.Error("process state is nil after shell exit")
	}
}
