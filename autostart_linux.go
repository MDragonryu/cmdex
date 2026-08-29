//go:build linux

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func autostartDesktopPath() (string, error) {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "autostart", "cmdex.desktop"), nil
}

// setAutostart writes or removes an XDG autostart .desktop entry, which is
// honoured by GNOME, KDE, XFCE and other freedesktop-compliant environments.
func setAutostart(enabled bool) error {
	path, err := autostartDesktopPath()
	if err != nil {
		return err
	}

	if !enabled {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove login item: %w", err)
		}
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("resolve executable symlinks: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create autostart directory: %w", err)
	}

	entry := strings.Join([]string{
		"[Desktop Entry]",
		"Type=Application",
		"Name=CmDex",
		"Comment=CLI command manager with variable placeholders",
		// Exec is split on spaces by the spec, so quote the binary path.
		fmt.Sprintf("Exec=%q %s", exe, backgroundFlag),
		"Terminal=false",
		"X-GNOME-Autostart-enabled=true",
		"",
	}, "\n")

	if err := os.WriteFile(path, []byte(entry), 0o600); err != nil {
		return fmt.Errorf("write login item: %w", err)
	}
	return nil
}

// autostartEnabled reports whether the autostart .desktop entry exists.
func autostartEnabled() bool {
	path, err := autostartDesktopPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}
