//go:build !darwin

package main

// presentLauncherWindowNative is intentionally a no-op outside macOS. The
// caller supplies the Wails screen-based fallback for those platforms.
func presentLauncherWindowNative(_ string, _ int, _ float64) bool {
	return false
}

// presentLauncherWindow keeps the show/placement/focus sequence injectable so
// the global-shortcut path can be tested without starting a desktop app.
func presentLauncherWindow(show func(), nativePlace func() bool, fallbackPlace func(), focus func()) {
	show()
	if !nativePlace() {
		fallbackPlace()
	}
	focus()
}
