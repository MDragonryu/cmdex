//go:build !windows

package main

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// readCollector drains a PTY handle into a buffer that tests can snapshot.
type readCollector struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (c *readCollector) drain(h ptyHandle) {
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := h.Read(buf)
			if n > 0 {
				c.mu.Lock()
				c.buf.Write(buf[:n])
				c.mu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()
}

func (c *readCollector) takeAfter(d time.Duration) string {
	time.Sleep(d)
	c.mu.Lock()
	defer c.mu.Unlock()
	s := c.buf.String()
	c.buf.Reset()
	return s
}

// TestPtyBackspaceEchoesEraseSequence is the regression test for the reported
// bug: with no TERM in the child environment the shell degrades to "dumb" line
// editing and echoes a lone space for Backspace, which advances the cursor
// instead of erasing. A correctly described terminal echoes a cursor-left.
func TestPtyBackspaceEchoesEraseSequence(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns a real login shell")
	}

	shellPath, shellFlag := detectShell()
	backend := newPtyBackend()

	handle, proc, err := backend.Start(shellPath, shellFlag, "", 24, 80)
	if err != nil {
		t.Skipf("cannot start PTY shell %s: %v", shellPath, err)
	}
	defer func() {
		_ = proc.Kill()
		_ = handle.Close()
	}()

	collector := &readCollector{}
	collector.drain(handle)

	// Let the login shell finish printing its prompt, then discard it.
	collector.takeAfter(1500 * time.Millisecond)

	if _, err := handle.Write([]byte("abc")); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	typed := collector.takeAfter(400 * time.Millisecond)
	if !strings.Contains(typed, "abc") {
		t.Skipf("shell did not echo typed input (%q); environment cannot run this check", typed)
	}

	if _, err := handle.Write([]byte{0x7f}); err != nil { // DEL, what xterm.js sends for Backspace
		t.Fatalf("write failed: %v", err)
	}
	erased := collector.takeAfter(400 * time.Millisecond)

	if !strings.Contains(erased, "\b") {
		t.Errorf("Backspace echoed %q, which contains no cursor-left; the cursor would advance instead of erasing", erased)
	}
}

// TestPtyStartUsesWorkingDir guards that the session's working directory is
// actually applied to the shell process rather than only reported in
// SessionInfo.
func TestPtyStartUsesWorkingDir(t *testing.T) {
	dir := t.TempDir()

	_, cmd, err := ptyStart("/bin/sh", "-c", dir, 24, 80, "exit 0")
	if err != nil {
		t.Fatalf("ptyStart failed: %v", err)
	}
	defer func() { _ = killProcessGroup(cmd) }()

	if cmd.Dir != dir {
		t.Errorf("cmd.Dir = %q, want %q", cmd.Dir, dir)
	}
}

// TestPtyExecutesCarriageReturnTerminatedCommand is the regression test for a
// command that streamed into the terminal but sat at the prompt until the user
// pressed Enter themselves. Shells submit a line on CR — the byte xterm.js
// sends for Enter — so a written command must be CR-terminated too. A bare LF
// happens to work on unix because the line discipline translates it, which is
// what hid the bug until it reached Windows, where the ConPTY input parser
// does not. This test pins the unix half: the CR form must keep executing here.
func TestPtyExecutesCarriageReturnTerminatedCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("spawns a real login shell")
	}

	shellPath, shellFlag := detectShell()
	backend := newPtyBackend()

	handle, proc, err := backend.Start(shellPath, shellFlag, "", 24, 80)
	if err != nil {
		t.Skipf("cannot start PTY shell %s: %v", shellPath, err)
	}
	defer func() {
		_ = proc.Kill()
		_ = handle.Close()
	}()

	collector := &readCollector{}
	collector.drain(handle)

	// Let the login shell finish printing its prompt, then discard it.
	collector.takeAfter(1500 * time.Millisecond)

	if _, err := handle.Write([]byte(toTerminalInput("echo cmdex_ran\n"))); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	out := collector.takeAfter(1200 * time.Millisecond)

	// The shell echoes the typed line back, so the sentinel appearing once
	// only proves it reached the prompt. Executing it prints a second copy.
	if strings.Count(out, "cmdex_ran") < 2 {
		t.Errorf("command was echoed but never executed; PTY output was %q", out)
	}
}
