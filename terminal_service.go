package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

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

// emitOutput sends PTY output data to the frontend via Wails event.
func (s *TerminalService) emitOutput(data string) {
	wailsApp.Event.Emit(eventNames.PtyOutput, map[string]interface{}{
		"data": data,
	})
}

// readLoop drains PTY output in a background goroutine, batching reads at
// 16ms intervals with 64KB max chunks before emitting via emitOutput.
func (s *TerminalService) readLoop() {
	reader := bufio.NewReaderSize(s.ptmx, 64*1024)
	ticker := time.NewTicker(16 * time.Millisecond)
	defer ticker.Stop()

	var buf bytes.Buffer
	buf.Grow(64 * 1024)

	for {
		select {
		case <-s.stopCh:
			if buf.Len() > 0 {
				s.emitOutput(buf.String())
			}
			return
		case <-ticker.C:
			if buf.Len() > 0 {
				s.emitOutput(buf.String())
				buf.Reset()
			}
		default:
			chunk := make([]byte, 4096)
			n, err := reader.Read(chunk)
			if n > 0 {
				buf.Write(chunk[:n])
				if buf.Len() >= 64*1024 {
					s.emitOutput(buf.String())
					buf.Reset()
				}
			}
			if err != nil {
				if buf.Len() > 0 {
					s.emitOutput(buf.String())
				}
				return
			}
		}
	}
}

// monitorExit waits for the shell process to exit, emits a pty-exit event,
// writes a restart message to terminal output, and auto-restarts the shell.
func (s *TerminalService) monitorExit() {
	err := s.cmd.Wait()

	select {
	case <-s.stopCh:
		return
	default:
	}

	exitCode := 0
	wasIntentional := false

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	if exitCode == 0 {
		wasIntentional = true
	}

	wailsApp.Event.Emit(eventNames.PtyExit, map[string]interface{}{
		"exitCode":      exitCode,
		"wasIntentional": wasIntentional,
	})

	restartMsg := fmt.Sprintf("\r\n[shell exited with code %d — restarting...]\r\n", exitCode)
	s.emitOutput(restartMsg)

	time.Sleep(100 * time.Millisecond)

	_ = s.Start(int(s.lastSize.Cols), int(s.lastSize.Rows))
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

	go s.readLoop()
	go s.monitorExit()

	return nil
}

func (s *TerminalService) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopCh != nil {
		close(s.stopCh)
	}

	time.Sleep(50 * time.Millisecond)

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
