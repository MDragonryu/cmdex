package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type ptyWinsize struct {
	Rows uint16
	Cols uint16
}

// TerminalService manages a PTY-backed shell process for the integrated terminal.
type TerminalService struct {
	mu        sync.Mutex
	ptmx      *os.File
	cmd       *exec.Cmd
	shellPath string
	shellFlag string
	lastSize  ptyWinsize
	stopCh    chan struct{}
}

// detectShell returns the shell path and flag for the current OS.
// On Unix: respects $SHELL env var, falls back to /bin/sh with -l flag.
// On Windows: chains pwsh → powershell → cmd (pwsh preferred with -NoLogo flag).
// Note: D-01 requires full Windows detection chain that NewExecutor() does not support,
// and D-07 advises reusing NewExecutor(). This function implements D-01 fully.
func detectShell() (path, flag string) {
	if runtime.GOOS == "windows" {
		for _, shell := range []string{"pwsh", "powershell"} {
			if lp, err := exec.LookPath(shell); err == nil {
				return lp, "-NoLogo"
			}
		}
		return "cmd", ""
	}

	path = os.Getenv("SHELL")
	if path == "" {
		path = "/bin/sh"
	}
	flag = "-l"
	return path, flag
}

func (s *TerminalService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	return s.Start(80, 24)
}

func (s *TerminalService) ServiceShutdown() error {
	return s.Stop()
}

func (s *TerminalService) Start(cols, rows int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ptmx != nil {
		s.Stop()
	}

	shellPath, shellFlag := detectShell()
	s.shellPath = shellPath
	s.shellFlag = shellFlag
	s.lastSize = ptyWinsize{Rows: uint16(rows), Cols: uint16(cols)}

	ptmx, cmd, err := ptyStart(shellPath, shellFlag, rows, cols)
	if err != nil {
		return err
	}

	s.ptmx = ptmx
	s.cmd = cmd
	s.stopCh = make(chan struct{}, 1)

	return nil
}

func (s *TerminalService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopCh != nil {
		close(s.stopCh)
	}

	if s.cmd != nil && s.ptmx != nil {
		s.ptmx.Close()
		killProcessGroup(s.cmd)
	}

	s.ptmx = nil
	s.cmd = nil
	s.stopCh = nil

	return nil
}

func (s *TerminalService) Write(data string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ptmx == nil {
		return fmt.Errorf("terminal not started")
	}

	_, err := s.ptmx.Write([]byte(data))
	return err
}

func (s *TerminalService) Resize(cols, rows int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.ptmx == nil {
		return fmt.Errorf("terminal not started")
	}

	s.lastSize = ptyWinsize{Rows: uint16(rows), Cols: uint16(cols)}

	return ptyResize(s.ptmx, cols, rows)
}
