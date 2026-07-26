package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func envMap(t *testing.T, env []string) map[string]string {
	t.Helper()
	m := make(map[string]string, len(env))
	for _, kv := range env {
		key, value, ok := strings.Cut(kv, "=")
		if !ok {
			t.Fatalf("malformed env entry: %q", kv)
		}
		m[key] = value
	}
	return m
}

// A GUI-launched app inherits no TERM. Without one the shell falls back to
// "dumb" line editing, where erasing a character echoes a bare space and the
// cursor advances instead of erasing — the Backspace bug this guards.
func TestPtyEnvSetsTermWhenHostHasNone(t *testing.T) {
	t.Setenv("TERM", "")
	os.Unsetenv("TERM")

	m := envMap(t, ptyEnv())

	if m["TERM"] != ptyTerm {
		t.Errorf("TERM = %q, want %q", m["TERM"], ptyTerm)
	}
	if m["COLORTERM"] != "truecolor" {
		t.Errorf("COLORTERM = %q, want truecolor", m["COLORTERM"])
	}
}

// An inherited TERM describes the host terminal, not xterm.js, so it must be
// replaced rather than passed through.
func TestPtyEnvOverridesInheritedTerm(t *testing.T) {
	t.Setenv("TERM", "dumb")

	m := envMap(t, ptyEnv())

	if m["TERM"] != ptyTerm {
		t.Errorf("TERM = %q, want %q", m["TERM"], ptyTerm)
	}
}

func TestPtyEnvStripsStaleSizeVars(t *testing.T) {
	t.Setenv("LINES", "10")
	t.Setenv("COLUMNS", "40")

	m := envMap(t, ptyEnv())

	if _, ok := m["LINES"]; ok {
		t.Error("LINES leaked into PTY env; it pins the shell to a stale size")
	}
	if _, ok := m["COLUMNS"]; ok {
		t.Error("COLUMNS leaked into PTY env; it pins the shell to a stale size")
	}
}

func TestPtyEnvPreservesConfiguredLocale(t *testing.T) {
	t.Setenv("LANG", "de_DE.UTF-8")

	m := envMap(t, ptyEnv())

	if m["LANG"] != "de_DE.UTF-8" {
		t.Errorf("LANG = %q, want the host value de_DE.UTF-8", m["LANG"])
	}
}

func TestPtyEnvFallsBackToUTF8Locale(t *testing.T) {
	t.Setenv("LANG", "")
	os.Unsetenv("LANG")

	m := envMap(t, ptyEnv())

	if !strings.HasSuffix(m["LANG"], "UTF-8") {
		t.Errorf("LANG = %q, want a UTF-8 locale fallback", m["LANG"])
	}
}

func TestPtyEnvHasNoDuplicateKeys(t *testing.T) {
	seen := make(map[string]int)
	for _, kv := range ptyEnv() {
		key, _, _ := strings.Cut(kv, "=")
		seen[key]++
	}
	for key, n := range seen {
		if n > 1 {
			t.Errorf("env key %q appears %d times", key, n)
		}
	}
}

func TestResolvePtyDir(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"existing directory", dir, dir},
		{"empty", "", ""},
		{"missing", filepath.Join(dir, "nope"), ""},
		{"regular file", file, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolvePtyDir(tt.in); got != tt.want {
				t.Errorf("resolvePtyDir(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
