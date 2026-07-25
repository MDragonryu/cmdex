package globalhotkey

import (
	"runtime"
	"testing"
)

func TestParseChordValid(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Chord
	}{
		{"default shortcut", "CmdOrCtrl+Shift+K", cmdOrCtrl(Chord{Shift: true, Key: "K"})},
		{"lowercase", "ctrl+shift+k", Chord{Ctrl: true, Shift: true, Key: "K"}},
		{"spaces around tokens", " Ctrl + Shift + K ", Chord{Ctrl: true, Shift: true, Key: "K"}},
		{"space key", "Ctrl+Alt+Space", Chord{Ctrl: true, Alt: true, Key: "SPACE"}},
		{"option alias", "Option+Space", Chord{Alt: true, Key: "SPACE"}},
		{"super alias", "Super+K", Chord{Cmd: true, Key: "K"}},
		{"esc alias", "Ctrl+Esc", Chord{Ctrl: true, Key: "ESCAPE"}},
		{"enter alias", "Ctrl+Enter", Chord{Ctrl: true, Key: "RETURN"}},
		{"function key", "Ctrl+F12", Chord{Ctrl: true, Key: "F12"}},
		{"digit key", "Ctrl+Alt+7", Chord{Ctrl: true, Alt: true, Key: "7"}},
		{"all modifiers", "Cmd+Ctrl+Alt+Shift+K", Chord{Cmd: true, Ctrl: true, Alt: true, Shift: true, Key: "K"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseChord(tt.input)
			if err != nil {
				t.Fatalf("ParseChord(%q) returned error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ParseChord(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseChordInvalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"whitespace only", "   "},
		{"no modifier", "K"},
		{"modifiers only", "Ctrl+Shift"},
		{"two regular keys", "Ctrl+K+J"},
		{"unknown key", "Ctrl+PrintScreen"},
		{"empty component", "Ctrl++K"},
		{"trailing plus", "Ctrl+K+"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := ParseChord(tt.input); err == nil {
				t.Errorf("ParseChord(%q) = %+v, want error", tt.input, got)
			}
		})
	}
}

// A bare key must never be accepted: grabbing it system-wide would make that
// key unusable in every other application.
func TestParseChordRejectsAllBareKeys(t *testing.T) {
	for key := range canonicalKeys {
		if _, err := ParseChord(key); err == nil {
			t.Errorf("ParseChord(%q) accepted a bare key without modifiers", key)
		}
	}
}

// Every key the parser accepts must have a platform code, otherwise a user
// could save a shortcut that then fails to register.
func TestEveryCanonicalKeyIsRegisterable(t *testing.T) {
	for _, canonical := range canonicalKeys {
		chord, err := ParseChord("Ctrl+Shift+" + canonical)
		if err != nil {
			t.Fatalf("ParseChord for canonical key %q failed: %v", canonical, err)
		}
		if chord.Key != canonical {
			t.Errorf("canonical key %q parsed to %q", canonical, chord.Key)
		}
	}
}

func TestChordRoundTrip(t *testing.T) {
	original := "Ctrl+Shift+K"
	chord, err := ParseChord(original)
	if err != nil {
		t.Fatalf("ParseChord failed: %v", err)
	}
	reparsed, err := ParseChord(chord.String())
	if err != nil {
		t.Fatalf("ParseChord(%q) failed: %v", chord.String(), err)
	}
	if reparsed != chord {
		t.Errorf("round trip changed chord: %+v -> %q -> %+v", chord, chord.String(), reparsed)
	}
}

func TestCmdOrCtrlIsPlatformSpecific(t *testing.T) {
	chord, err := ParseChord("CmdOrCtrl+K")
	if err != nil {
		t.Fatalf("ParseChord failed: %v", err)
	}
	if runtime.GOOS == "darwin" {
		if !chord.Cmd || chord.Ctrl {
			t.Errorf("on darwin CmdOrCtrl should map to Cmd, got %+v", chord)
		}
	} else if chord.Cmd || !chord.Ctrl {
		t.Errorf("on %s CmdOrCtrl should map to Ctrl, got %+v", runtime.GOOS, chord)
	}
}

// cmdOrCtrl sets whichever of Cmd/Ctrl "CmdOrCtrl" resolves to on this platform.
func cmdOrCtrl(c Chord) Chord {
	if runtime.GOOS == "darwin" {
		c.Cmd = true
	} else {
		c.Ctrl = true
	}
	return c
}
