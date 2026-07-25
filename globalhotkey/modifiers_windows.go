//go:build windows

package globalhotkey

import "golang.design/x/hotkey"

func platformModifiers(c Chord) ([]hotkey.Modifier, error) {
	var mods []hotkey.Modifier
	if c.Cmd {
		mods = append(mods, hotkey.ModWin)
	}
	if c.Ctrl {
		mods = append(mods, hotkey.ModCtrl)
	}
	if c.Alt {
		mods = append(mods, hotkey.ModAlt)
	}
	if c.Shift {
		mods = append(mods, hotkey.ModShift)
	}
	return mods, nil
}
