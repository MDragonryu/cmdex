//go:build !darwin

package main

// presentLauncherWindowNative is intentionally a no-op outside macOS. The
// caller supplies the Wails screen-based fallback for those platforms.
func prepareLauncherWindowNative(_ string) bool {
	return false
}

func presentLauncherWindowNative(_ string, _, _ int, _ float64, _ bool) bool {
	return false
}

func hideLauncherWindowNative(_ string) bool {
	return false
}

// presentLauncherWindow keeps the show/placement/focus sequence injectable so
// the global-shortcut path can be tested without starting a desktop app.
func presentLauncherWindow(prepare func(), show func(), nativePlace func() bool, fallbackPlace func(), focus func()) {
	prepare()
	show()
	if !nativePlace() {
		fallbackPlace()
	}
	focus()
}
