package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type ptyWinsize struct {
	Rows uint16
	Cols uint16
}

// TerminalService manages a PTY-backed shell process for the integrated terminal.
type TerminalService struct {
	mu              sync.Mutex
	ptmx            *os.File
	cmd             *exec.Cmd
	shellPath       string
	shellFlag       string
	lastSize        ptyWinsize
	stopCh          chan struct{}
	running         bool
	starting        bool
	intentionalStop bool

	// readerWg tracks the readLoop goroutine lifetime. startLocked calls
	// readerWg.Wait() after closing the old PTY fd, guaranteeing the old
	// goroutine has fully exited before a new one starts — preventing two
	// concurrent readers on the same fd from producing interleaved output.
	readerWg sync.WaitGroup

	outputCh  chan string
	outputSeq uint64
	emitterWg sync.WaitGroup
}

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

func (s *TerminalService) startEmitter() {
	s.outputCh = make(chan string, 512)
	s.emitterWg.Add(1)

	go func() {
		defer s.emitterWg.Done()

		var buf strings.Builder
		ticker := time.NewTicker(8 * time.Millisecond)
		defer ticker.Stop()

		flush := func() {
			if buf.Len() == 0 {
				return
			}
			seq := atomic.AddUint64(&s.outputSeq, 1)
			if wailsApp != nil {
				wailsApp.Event.Emit(eventNames.PtyOutput, map[string]interface{}{
					"data": buf.String(),
					"seq":  seq,
				})
			}
			buf.Reset()
		}

		for {
			select {
			case data, ok := <-s.outputCh:
				if !ok {
					flush()
					return
				}
				buf.WriteString(data)
				if buf.Len() >= 32*1024 {
					flush()
				}

			case <-ticker.C:
				flush()
			}
		}
	}()
}

func (s *TerminalService) stopEmitter() {
	if s.outputCh != nil {
		close(s.outputCh)
		s.emitterWg.Wait()
		s.outputCh = nil
	}
}

func (s *TerminalService) enqueueOutput(data string) {
	ch := s.outputCh
	if ch == nil {
		return
	}
	select {
	case ch <- data:
	default:
	}
}

func (s *TerminalService) readLoop(ptmx *os.File, stopCh chan struct{}) {
	defer s.readerWg.Done()

	buf := make([]byte, 8192)
	var leftover []byte

	for {
		select {
		case <-stopCh:
			return
		default:
		}

		n, err := ptmx.Read(buf)
		if n > 0 {
			var data []byte
			if len(leftover) > 0 {
				data = make([]byte, len(leftover)+n)
				copy(data, leftover)
				copy(data[len(leftover):], buf[:n])
				leftover = nil
			} else {
				data = buf[:n]
			}

			split := len(data)
			for i := 0; i < utf8.UTFMax && split > 0 && !utf8.Valid(data[:split]); i++ {
				split--
			}

			s.enqueueOutput(string(data[:split]))

			if split < len(data) {
				leftover = append(leftover[:0], data[split:]...)
			}
		}

		if err != nil {
			if len(leftover) > 0 {
				s.enqueueOutput(string(leftover))
			}
			return
		}
	}
}

func (s *TerminalService) monitorExit(cmd *exec.Cmd, ptmx *os.File, stopCh chan struct{}) {
	err := cmd.Wait()

	select {
	case <-stopCh:
		return
	default:
	}

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	s.mu.Lock()
	intentional := s.intentionalStop || exitCode == 0
	cols := int(s.lastSize.Cols)
	rows := int(s.lastSize.Rows)
	s.mu.Unlock()

	if wailsApp != nil {
		wailsApp.Event.Emit(eventNames.PtyExit, map[string]interface{}{
			"exitCode":       exitCode,
			"wasIntentional": intentional,
		})
	}

	if intentional {
		s.enqueueOutput(fmt.Sprintf("\r\n[shell exited with code %d]\r\n", exitCode))
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		return
	}

	s.enqueueOutput(fmt.Sprintf("\r\n[shell exited with code %d — restarting...]\r\n", exitCode))

	time.Sleep(100 * time.Millisecond)
	_ = s.Start(cols, rows)
}

func (s *TerminalService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	terminalSvc = s
	s.startEmitter()
	return s.Start(80, 24)
}

func (s *TerminalService) ServiceShutdown() error {
	err := s.Stop()
	s.stopEmitter()
	return err
}

func (s *TerminalService) stopLocked() {
	if s.stopCh != nil {
		close(s.stopCh)
		s.stopCh = nil
	}
}

func (s *TerminalService) startLocked(cols, rows int) error {
	if s.starting {
		return nil
	}
	s.starting = true

	if cols < 10 {
		cols = 80
	}
	if rows < 3 {
		rows = 24
	}

	s.stopLocked()
	oldPtmx := s.ptmx
	oldCmd := s.cmd
	s.ptmx = nil
	s.cmd = nil
	s.running = false
	s.intentionalStop = false

	shellPath, shellFlag := detectShell()

	s.mu.Unlock()

	if oldPtmx != nil {
		oldPtmx.Close()
	}
	if oldCmd != nil {
		killProcessGroup(oldCmd)
	}

	s.readerWg.Wait()

	ptmx, cmd, err := ptyStart(shellPath, shellFlag, rows, cols)

	s.mu.Lock()
	s.starting = false

	if err != nil {
		return err
	}

	s.shellPath = shellPath
	s.shellFlag = shellFlag
	s.lastSize = ptyWinsize{Rows: uint16(rows), Cols: uint16(cols)}
	s.ptmx = ptmx
	s.cmd = cmd
	s.stopCh = make(chan struct{})
	s.running = true

	stopCh := s.stopCh
	s.readerWg.Add(1)
	go s.readLoop(ptmx, stopCh)
	go s.monitorExit(cmd, ptmx, stopCh)

	return nil
}

func (s *TerminalService) Start(cols, rows int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.startLocked(cols, rows)
}

func (s *TerminalService) Stop() error {
	s.mu.Lock()

	s.intentionalStop = true
	s.stopLocked()

	oldPtmx := s.ptmx
	oldCmd := s.cmd
	s.ptmx = nil
	s.cmd = nil
	s.running = false

	s.mu.Unlock()

	if oldPtmx != nil {
		oldPtmx.Close()
	}
	if oldCmd != nil {
		killProcessGroup(oldCmd)
	}

	s.readerWg.Wait()

	return nil
}

func (s *TerminalService) Write(data string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		if err := s.startLocked(int(s.lastSize.Cols), int(s.lastSize.Rows)); err != nil {
			return err
		}
	}

	if s.ptmx == nil {
		return fmt.Errorf("terminal not started")
	}

	b := []byte(data)
	for len(b) > 0 {
		n, err := s.ptmx.Write(b)
		if err != nil {
			return err
		}
		b = b[n:]
	}
	return nil
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

func (s *TerminalService) Clear() error {
	return s.Write("\x1b[H\x1b[2J\x1b[3J")
}
