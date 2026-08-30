package main

import (
	"io"
	"sync"
	"testing"
	"time"
)

type monitorTestHandle struct {
	closed bool
}

func (h *monitorTestHandle) Read([]byte) (int, error)    { return 0, io.EOF }
func (h *monitorTestHandle) Write(p []byte) (int, error) { return len(p), nil }
func (h *monitorTestHandle) Close() error {
	h.closed = true
	return nil
}

type monitorTestProcess struct{}

func (monitorTestProcess) Pid() int           { return 1 }
func (monitorTestProcess) Wait() (int, error) { return 1, nil }
func (monitorTestProcess) Exited() bool       { return true }

type monitorTestStableProcess struct{}

func (monitorTestStableProcess) Pid() int           { return 1 }
func (monitorTestStableProcess) Wait() (int, error) { return 0, nil }
func (monitorTestStableProcess) Exited() bool       { return true }

type resizeDuringRestartBackend struct {
	mu        sync.Mutex
	startCh   chan struct{}
	startRows int
	startCols int
}

func (b *resizeDuringRestartBackend) Start(_, _, _ string, rows, cols int, _ shellLaunchOpts) (ptyHandle, ptyProcess, error) {
	b.mu.Lock()
	b.startRows = rows
	b.startCols = cols
	b.mu.Unlock()
	close(b.startCh)
	return &monitorTestHandle{}, monitorTestStableProcess{}, nil
}

func (b *resizeDuringRestartBackend) Resize(ptyHandle, int, int) error { return nil }
func (b *resizeDuringRestartBackend) Kill(ptyProcess) error            { return nil }

func TestWaitForAutoRestartStopsDuringDelay(t *testing.T) {
	stopCh := make(chan struct{})
	result := make(chan bool, 1)
	timer := time.NewTimer(time.Hour)
	go func() { result <- waitForAutoRestartTimer(stopCh, timer) }()

	close(stopCh)

	select {
	case restarted := <-result:
		if restarted {
			t.Fatal("waitForAutoRestart reported a restart after the session was stopped")
		}
	case <-time.After(autoRestartDelay):
		t.Fatal("waitForAutoRestart did not observe the stop signal")
	}
}

func TestMonitorExitRestartUsesSizeResizedDuringDelay(t *testing.T) {
	backend := &resizeDuringRestartBackend{startCh: make(chan struct{})}
	s := &TerminalService{ptyBackend: backend, sessions: make(map[string]*sessionState)}
	stopCh := make(chan struct{})
	handle := &monitorTestHandle{}
	ss := &sessionState{
		id:       "resize-restart",
		ptmx:     handle,
		proc:     monitorTestProcess{},
		stopCh:   stopCh,
		running:  true,
		lastSize: ptyWinsize{Rows: 24, Cols: 80},
	}
	s.sessions[ss.id] = ss

	// Replace the real delay with a gate so Resize is guaranteed to happen
	// between the initial exit and the replacement Start call.
	originalWait := waitForAutoRestart
	delayReady := make(chan struct{})
	continueRestart := make(chan struct{})
	waitForAutoRestart = func(<-chan struct{}) bool {
		close(delayReady)
		<-continueRestart
		return true
	}
	t.Cleanup(func() { waitForAutoRestart = originalWait })

	done := make(chan struct{})
	go func() {
		s.monitorExit(ss, monitorTestProcess{}, stopCh)
		close(done)
	}()
	select {
	case <-delayReady:
	case <-time.After(time.Second):
		t.Fatal("monitorExit did not enter the restart delay")
	}

	if err := s.Resize(ss.id, 140, 42); err != nil {
		t.Fatalf("Resize during restart delay failed: %v", err)
	}
	close(continueRestart)

	select {
	case <-backend.startCh:
	case <-time.After(time.Second):
		t.Fatal("replacement PTY was not started")
	}
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("monitorExit did not finish")
	}

	backend.mu.Lock()
	rows, cols := backend.startRows, backend.startCols
	backend.mu.Unlock()
	if rows != 42 || cols != 140 {
		t.Fatalf("replacement PTY dimensions = rows=%d cols=%d, want rows=42 cols=140", rows, cols)
	}
}
