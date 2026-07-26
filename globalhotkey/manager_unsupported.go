//go:build !windows && !((darwin || linux) && cgo)

package globalhotkey

// Manager is the no-op implementation used on build/platform combinations that
// cannot register global hotkeys — in practice a CGO_ENABLED=0 Unix build.
//
// The underlying golang.design/x/hotkey package ships a stub for that case
// whose Register panics, so this package deliberately does not import it here
// and returns ErrUnsupported instead. The app then starts normally with the
// launcher's global shortcut disabled.
type Manager struct{}

// NewManager returns a Manager that cannot register hotkeys.
func NewManager() *Manager { return &Manager{} }

// Supported always reports false for this build.
func (m *Manager) Supported() bool { return false }

// Register always fails with ErrUnsupported.
func (m *Manager) Register(_ Chord, _ func()) error { return ErrUnsupported }

// Unregister is a no-op.
func (m *Manager) Unregister() {}
