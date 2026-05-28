//go:build windows

package main

// TODO(Phase 16-followup): Windows PTY support deferred.
//
// go-winpty v1.0.4 (github.com/iamacarpet/go-winpty) is DEPRECATED as of 2022 —
// the package README recommends github.com/UserExistsError/conpty for Windows 10
// 1809+ which has native ConPTY support.
//
// Refactoring steps needed to support Windows PTY:
//   1. Evaluate github.com/UserExistsError/conpty as the replacement for go-winpty.
//   2. Change TerminalService.ptmx from *os.File to io.ReadWriteCloser (or add a
//      separate winPTY struct field) because Windows PTY returns separate read/write
//      handles, unlike Unix PTY which uses a single bidirectional *os.File.
//   3. Update terminal_unix.go: ptyStart returns io.ReadWriteCloser instead of *os.File.
//   4. Update Write() → use io.Writer interface instead of *os.File.Write.
//   5. Update readLoop() → use io.Reader interface instead of bufio.NewReaderSize(*os.File).
//   6. Update Resize() → dispatch to platform-specific resize via interface assertion.
//   7. Test on actual Windows 10 1809+ with CGO enabled.
//
// The existing killProcessGroup (using taskkill /F /T /PID) is already correct
// and works independently of the PTY backend.

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

func ptyStart(shellPath, shellFlag string, rows, cols int, extraArgs ...string) (*os.File, *exec.Cmd, error) {
	return nil, nil, fmt.Errorf("Windows PTY support not yet implemented — see Plan 16-03")
}

func ptyResize(ptmx *os.File, cols, rows int) error {
	return fmt.Errorf("Windows PTY support not yet implemented")
}

func killProcessGroup(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	return exec.Command("taskkill", "/F", "/T", "/PID", strconv.Itoa(cmd.Process.Pid)).Run()
}
