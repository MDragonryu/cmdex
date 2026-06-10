//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/creack/pty"
)

func ptyStart(shellPath, shellFlag string, rows, cols int, extraArgs ...string) (*os.File, *exec.Cmd, error) {
	args := []string{shellFlag}
	args = append(args, extraArgs...)
	cmd := exec.Command(shellPath, args...)
	cmd.Env = os.Environ()

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
	if err != nil {
		return nil, nil, fmt.Errorf("pty.StartWithSize: %w", err)
	}

	return ptmx, cmd, nil
}

func ptyResize(ptmx *os.File, cols, rows int) error {
	return pty.Setsize(ptmx, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
}

func killProcessGroup(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	pid := cmd.Process.Pid

	_ = syscall.Kill(-pid, syscall.SIGHUP)

	if cmd.ProcessState != nil {
		return nil
	}

	done := make(chan struct{}, 1)
	go func() {
		_ = cmd.Wait()
		done <- struct{}{}
	}()

	select {
	case <-done:
		return nil
	case <-time.After(2 * time.Second):
		_ = syscall.Kill(-pid, syscall.SIGKILL)
		<-done
		return nil
	}
}
