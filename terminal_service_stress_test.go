//go:build darwin

package main

import (
	"runtime"
	"testing"
	"time"
)

// TestTerminalService_StressCreateClose verifies D-08: a 100-cycle
// CreateSession/CloseSession loop against the darwin-side mock leaves the
// TerminalService.sessions map empty after every close and the goroutine
// count stable within 5 of baseline. The test uses mockPtyBackend from
// pty_backend_mock_test.go (also //go:build darwin) so no real shells are
// spawned — only the orchestration path is exercised.
//
// Deviations from PLAN: 25-02 task 2 says the test goes in
// terminal_service_test.go. That file has no build tag; referencing
// newTestTerminalServiceWithMock (which is in a //go:build darwin file)
// from a non-tagged _test.go would break cross-platform test compilation.
// The test is therefore placed in this //go:build darwin file. The
// behavior under test is unchanged.
func TestTerminalService_StressCreateClose(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only test (uses mockPtyBackend from Plan 25-01)")
	}

	s := newTestTerminalServiceWithMock(t)
	defer s.ServiceShutdown()

	// Warmup cycle: run one full create+close before capturing baseline.
	// Capturing the baseline right after service construction can be flaky
	// under CI load because the Go runtime may still be allocating
	// background goroutines (GC, sysmon) for the first few hundred ms of
	// test process lifetime. Sleeping 200ms lets the runtime settle.
	warmup, err := s.CreateSession()
	if err != nil {
		t.Fatalf("warmup CreateSession failed: %v", err)
	}
	if err := s.CloseSession(warmup.ID); err != nil {
		t.Fatalf("warmup CloseSession failed: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	baseline := runtime.NumGoroutine()

	const cycles = 100
	for i := 0; i < cycles; i++ {
		info, err := s.CreateSession()
		if err != nil {
			t.Fatalf("cycle %d: CreateSession failed: %v", i, err)
		}
		if err := s.CloseSession(info.ID); err != nil {
			t.Fatalf("cycle %d: CloseSession failed: %v", i, err)
		}
		s.mu.RLock()
		count := len(s.sessions)
		s.mu.RUnlock()
		if count != 0 {
			t.Fatalf("cycle %d: sessions map not empty after close (len=%d)", i, count)
		}
	}

	// Allow monitorExit and emitter goroutines from the most recent close
	// to drain before sampling the post-loop goroutine count.
	time.Sleep(200 * time.Millisecond)

	after := runtime.NumGoroutine()
	drift := after - baseline
	if drift > 5 {
		t.Fatalf("goroutine drift %d exceeds limit of 5 (baseline=%d, after=%d)",
			drift, baseline, after)
	}
}
