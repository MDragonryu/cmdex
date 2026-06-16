//go:build darwin

package main

import "testing"

// newTestTerminalServiceWithMock returns a TerminalService wired to the
// in-memory mockPtyBackend. Orchestration tests that want to exercise
// TerminalService behavior without spawning a real shell use this helper.
func newTestTerminalServiceWithMock(t *testing.T) *TerminalService {
	t.Helper()
	s := &TerminalService{ptyBackend: mockPtyBackend{}}
	s.sessions = make(map[string]*sessionState)
	return s
}
