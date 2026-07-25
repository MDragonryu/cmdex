//go:build darwin && cgo

package globalhotkey

import "golang.design/x/hotkey"

func platformModifiers(c Chord) ([]hotkey.Modifier, error) {
	var mods []hotkey.Modifier
	if c.Cmd {
		mods = append(mods, hotkey.ModCmd)
	}
	if c.Ctrl {
		mods = append(mods, hotkey.ModCtrl)
	}
	if c.Alt {
		mods = append(mods, hotkey.ModOption)
	}
	if c.Shift {
		mods = append(mods, hotkey.ModShift)
	}
	return mods, nil
}
