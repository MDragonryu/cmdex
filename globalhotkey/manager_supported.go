//go:build windows || ((darwin || linux) && cgo)

package globalhotkey

import (
	"fmt"
	"sync"
	"time"

	"golang.design/x/hotkey"
)

// unregisterTimeout bounds how long Unregister waits for the platform call.
// See runOffMainThread for why it can block, and Unregister for why giving up
// is safe.
const unregisterTimeout = 2 * time.Second

// Manager owns at most one registered global hotkey at a time.
type Manager struct {
	mu   sync.Mutex
	hk   *hotkey.Hotkey
	stop chan struct{}
}

// runOffMainThread runs fn on a fresh goroutine and waits for its result.
//
// The macOS backend performs a dispatch_sync onto the main queue for both
// register and unregister. Issuing that from the main thread — which is
// exactly where Wails runs ServiceStartup and ServiceShutdown — is an
// immediate self-deadlock that libdispatch aborts with SIGTRAP. A newly
// spawned goroutine is never scheduled onto Wails' locked main thread, so the
// dispatch_sync can complete there.
//
// The call blocks until the main run loop services its queue, so callers must
// not invoke it from the main thread before application.Run has started.
func runOffMainThread(fn func() error) error {
	done := make(chan error, 1)
	go func() { done <- fn() }()
	return <-done
}

// NewManager returns a Manager with no hotkey registered.
func NewManager() *Manager { return &Manager{} }

// Supported reports whether this build can register global hotkeys.
func (m *Manager) Supported() bool { return true }

// Register grabs the chord system-wide and calls onTrigger on every keydown.
// Any previously registered hotkey is released first, so Register doubles as
// "change the shortcut". The returned error is non-nil when the combination is
// already taken by another application, or (on macOS) when the process has not
// been granted Accessibility permission.
func (m *Manager) Register(c Chord, onTrigger func()) error {
	mods, err := platformModifiers(c)
	if err != nil {
		return err
	}
	key, ok := keyTable[c.Key]
	if !ok {
		return fmt.Errorf("unsupported key %q", c.Key)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.unregisterLocked()

	hk := hotkey.New(mods, key)
	if err := runOffMainThread(hk.Register); err != nil {
		return fmt.Errorf("register %s: %w", c, err)
	}

	stop := make(chan struct{})
	m.hk, m.stop = hk, stop

	go func() {
		down := hk.Keydown()
		for {
			select {
			case <-stop:
				return
			case _, open := <-down:
				if !open {
					return
				}
				onTrigger()
			}
		}
	}()
	return nil
}

// Unregister releases the currently registered hotkey, if any. It is safe to
// call when nothing is registered.
func (m *Manager) Unregister() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.unregisterLocked()
}

func (m *Manager) unregisterLocked() {
	if m.stop != nil {
		close(m.stop)
		m.stop = nil
	}
	hk := m.hk
	m.hk = nil
	if hk == nil {
		return
	}

	// During shutdown the main run loop may already have stopped servicing its
	// queue, leaving the platform call with nothing to complete against. Give
	// up rather than hang the app on quit: the OS tears the hotkey down with
	// the process anyway.
	done := make(chan struct{})
	go func() {
		_ = hk.Unregister()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(unregisterTimeout):
	}
}

// keyTable maps canonical key names to hotkey key codes. The constant names are
// identical across the darwin, windows and x11 backends, so one table serves all.
var keyTable = map[string]hotkey.Key{
	"A": hotkey.KeyA, "B": hotkey.KeyB, "C": hotkey.KeyC, "D": hotkey.KeyD,
	"E": hotkey.KeyE, "F": hotkey.KeyF, "G": hotkey.KeyG, "H": hotkey.KeyH,
	"I": hotkey.KeyI, "J": hotkey.KeyJ, "K": hotkey.KeyK, "L": hotkey.KeyL,
	"M": hotkey.KeyM, "N": hotkey.KeyN, "O": hotkey.KeyO, "P": hotkey.KeyP,
	"Q": hotkey.KeyQ, "R": hotkey.KeyR, "S": hotkey.KeyS, "T": hotkey.KeyT,
	"U": hotkey.KeyU, "V": hotkey.KeyV, "W": hotkey.KeyW, "X": hotkey.KeyX,
	"Y": hotkey.KeyY, "Z": hotkey.KeyZ,

	"0": hotkey.Key0, "1": hotkey.Key1, "2": hotkey.Key2, "3": hotkey.Key3,
	"4": hotkey.Key4, "5": hotkey.Key5, "6": hotkey.Key6, "7": hotkey.Key7,
	"8": hotkey.Key8, "9": hotkey.Key9,

	"F1": hotkey.KeyF1, "F2": hotkey.KeyF2, "F3": hotkey.KeyF3, "F4": hotkey.KeyF4,
	"F5": hotkey.KeyF5, "F6": hotkey.KeyF6, "F7": hotkey.KeyF7, "F8": hotkey.KeyF8,
	"F9": hotkey.KeyF9, "F10": hotkey.KeyF10, "F11": hotkey.KeyF11, "F12": hotkey.KeyF12,
	"F13": hotkey.KeyF13, "F14": hotkey.KeyF14, "F15": hotkey.KeyF15, "F16": hotkey.KeyF16,
	"F17": hotkey.KeyF17, "F18": hotkey.KeyF18, "F19": hotkey.KeyF19, "F20": hotkey.KeyF20,

	"SPACE": hotkey.KeySpace, "RETURN": hotkey.KeyReturn, "TAB": hotkey.KeyTab,
	"ESCAPE": hotkey.KeyEscape, "DELETE": hotkey.KeyDelete,
	"UP": hotkey.KeyUp, "DOWN": hotkey.KeyDown,
	"LEFT": hotkey.KeyLeft, "RIGHT": hotkey.KeyRight,
}
