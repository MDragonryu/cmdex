//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

func ptyStart(shellPath, shellFlag string, rows, cols int) (*os.File, *exec.Cmd, error) {
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
