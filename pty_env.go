package main

import (
	"os"
	"strings"
)

// ptyTerm is the terminfo name the frontend emulator (xterm.js) implements.
// It must be advertised to the shell explicitly: a GUI-launched app inherits no
// TERM from launchd/Finder, and with TERM unset a shell falls back to "dumb"
// line editing — where erasing a character is echoed as a bare space instead of
// the "\b \b" sequence, so the cursor visibly advances on Backspace.
const ptyTerm = "xterm-256color"

// ptyEnvBlocklist are variables that must not leak from the host process into
// the PTY child. TERM/COLORTERM are re-set below; LINES/COLUMNS would pin the
// shell to a stale size and defeat SIGWINCH resizing.
var ptyEnvBlocklist = []string{
	"TERM",
	"COLORTERM",
	"LINES",
	"COLUMNS",
	"TERM_PROGRAM",
	"TERM_PROGRAM_VERSION",
}

// ptyEnv builds the environment for a PTY-hosted shell. It starts from the
// host environment, strips terminal-describing variables that would be wrong or
// stale, and declares what the frontend emulator actually supports.
func ptyEnv() []string {
	base := os.Environ()
	env := make([]string, 0, len(base)+5)

	hasLang := false
	for _, kv := range base {
		key, value, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		if isBlockedPtyEnv(key) {
			continue
		}
		if key == "LANG" && value != "" {
			hasLang = true
		}
		env = append(env, kv)
	}

	env = append(env,
		"TERM="+ptyTerm,
		"COLORTERM=truecolor",
		"TERM_PROGRAM=cmdex",
	)

	// A GUI-launched app also inherits no locale, which leaves the shell in the
	// C locale and mangles multi-byte input and output. Only fill the gap —
	// never override a locale the user has configured.
	if !hasLang {
		env = append(env, "LANG=en_US.UTF-8")
	}

	return env
}

func isBlockedPtyEnv(key string) bool {
	for _, blocked := range ptyEnvBlocklist {
		if key == blocked {
			return true
		}
	}
	return false
}

// resolvePtyDir returns dir if it is an existing directory, otherwise "" so the
// caller leaves exec.Cmd.Dir unset and inherits the process working directory.
func resolvePtyDir(dir string) string {
	if dir == "" {
		return ""
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return ""
	}
	return dir
}
