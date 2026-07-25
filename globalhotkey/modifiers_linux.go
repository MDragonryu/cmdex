//go:build linux && cgo

package globalhotkey

import "golang.design/x/hotkey"

// X11 exposes modifiers as generic Mod1..Mod5 bits rather than named keys.
// By near-universal convention Mod1 is Alt and Mod4 is Super/Meta.
func platformModifiers(c Chord) ([]hotkey.Modifier, error) {
	var mods []hotkey.Modifier
	if c.Cmd {
		mods = append(mods, hotkey.Mod4)
	}
	if c.Ctrl {
		mods = append(mods, hotkey.ModCtrl)
	}
	if c.Alt {
		mods = append(mods, hotkey.Mod1)
	}
	if c.Shift {
		mods = append(mods, hotkey.ModShift)
	}
	return mods, nil
}
