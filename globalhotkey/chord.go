// Package globalhotkey provides a small, platform-neutral wrapper around
// system-wide (global) keyboard shortcut registration.
//
// Wails v3 alpha.74 has no global hotkey facility of its own — its KeyBindings
// only fire while a Wails window already has focus — so registration is handled
// by golang.design/x/hotkey behind this package's interface.
//
// Platform support is decided at build time:
//
//   - Windows: always available (pure syscall, no cgo required).
//   - macOS/Linux: available when built with cgo, which is already the case for
//     every CmDex desktop build because Wails itself requires it.
//   - Anything else (notably a CGO_ENABLED=0 Unix build): Register returns
//     ErrUnsupported instead of panicking, so the app still starts normally.
package globalhotkey

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// ErrUnsupported is returned by Register when global hotkeys cannot work on the
// current platform/build combination. Callers should treat it as "feature off"
// rather than as a fatal error.
var ErrUnsupported = errors.New("global hotkeys are not supported by this build")

// Chord is a platform-neutral description of a global shortcut: one or more
// modifiers plus exactly one regular key.
type Chord struct {
	Cmd   bool // Command on macOS, Super/Windows key elsewhere
	Ctrl  bool
	Alt   bool // Option on macOS
	Shift bool
	Key   string // canonical upper-case key name, e.g. "K", "SPACE", "F1"
}

// canonicalKeys maps accepted key spellings to the canonical name used by the
// per-platform lookup tables. Only keys present on all three platforms are
// accepted so a shortcut configured on one OS stays valid on the others.
var canonicalKeys = buildCanonicalKeys()

func buildCanonicalKeys() map[string]string {
	keys := map[string]string{
		"SPACE": "SPACE",
		"ENTER": "RETURN", "RETURN": "RETURN",
		"TAB": "TAB",
		"ESC": "ESCAPE", "ESCAPE": "ESCAPE",
		"DEL": "DELETE", "DELETE": "DELETE",
		"UP": "UP", "DOWN": "DOWN", "LEFT": "LEFT", "RIGHT": "RIGHT",
	}
	for c := byte('A'); c <= 'Z'; c++ {
		keys[string(c)] = string(c)
	}
	for c := byte('0'); c <= '9'; c++ {
		keys[string(c)] = string(c)
	}
	for i := 1; i <= 20; i++ {
		name := fmt.Sprintf("F%d", i)
		keys[name] = name
	}
	return keys
}

// ParseChord parses an accelerator string such as "CmdOrCtrl+Shift+K" into a
// Chord. Tokens are separated by "+" and are case-insensitive.
//
// "CmdOrCtrl" resolves to Command on macOS and Control on Windows/Linux, which
// matches the accelerator convention already used by the CmDex application menu.
//
// A chord must carry at least one modifier — grabbing a bare key system-wide
// would make that key unusable in every other application.
func ParseChord(accelerator string) (Chord, error) {
	var c Chord
	if strings.TrimSpace(accelerator) == "" {
		return c, errors.New("shortcut is empty")
	}

	for raw := range strings.SplitSeq(accelerator, "+") {
		token := strings.ToUpper(strings.TrimSpace(raw))
		if token == "" {
			return Chord{}, fmt.Errorf("shortcut %q has an empty component", accelerator)
		}

		switch token {
		case "CMDORCTRL", "COMMANDORCONTROL":
			if runtime.GOOS == "darwin" {
				c.Cmd = true
			} else {
				c.Ctrl = true
			}
		case "CMD", "COMMAND", "SUPER", "META", "WIN", "WINDOWS":
			c.Cmd = true
		case "CTRL", "CONTROL":
			c.Ctrl = true
		case "ALT", "OPT", "OPTION":
			c.Alt = true
		case "SHIFT":
			c.Shift = true
		default:
			if c.Key != "" {
				return Chord{}, fmt.Errorf("shortcut %q has more than one non-modifier key", accelerator)
			}
			key, ok := canonicalKeys[token]
			if !ok {
				return Chord{}, fmt.Errorf("shortcut %q uses unsupported key %q", accelerator, raw)
			}
			c.Key = key
		}
	}

	if c.Key == "" {
		return Chord{}, fmt.Errorf("shortcut %q has no key", accelerator)
	}
	if !c.Cmd && !c.Ctrl && !c.Alt && !c.Shift {
		return Chord{}, fmt.Errorf("shortcut %q needs at least one modifier", accelerator)
	}
	return c, nil
}

// String renders the chord back into a canonical accelerator string.
func (c Chord) String() string {
	parts := make([]string, 0)
	if c.Ctrl {
		parts = append(parts, "Ctrl")
	}
	if c.Alt {
		if runtime.GOOS == "darwin" {
			parts = append(parts, "Option")
		} else {
			parts = append(parts, "Alt")
		}
	}
	if c.Shift {
		parts = append(parts, "Shift")
	}
	if c.Cmd {
		if runtime.GOOS == "darwin" {
			parts = append(parts, "Cmd")
		} else {
			parts = append(parts, "Super")
		}
	}
	return strings.Join(append(parts, c.Key), "+")
}
