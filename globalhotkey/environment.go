package globalhotkey

import (
	"os"
	"runtime"
	"strings"
)

// EnvironmentWarning returns a human-readable caveat about global hotkey
// reliability in the current desktop session, or "" when there is nothing to
// report.
//
// The case that matters in practice is Linux under Wayland: the X11 XGrabKey
// call used to register the shortcut is proxied through XWayland, so it only
// ever sees keystrokes delivered to X11 clients. Registration appears to
// succeed but the shortcut never fires while a native Wayland application has
// focus. Wayland exposes no portable global-shortcut protocol that this Wails
// alpha can reach, so the honest thing to do is say so rather than let the
// setting look active while doing nothing.
func EnvironmentWarning() string {
	if runtime.GOOS != "linux" {
		return ""
	}
	sessionType := strings.ToLower(os.Getenv("XDG_SESSION_TYPE"))
	if sessionType == "wayland" || os.Getenv("WAYLAND_DISPLAY") != "" {
		return "Wayland session detected: the shortcut is grabbed through XWayland, so it will not fire while a native Wayland window has focus. Running CmDex in an X11 session is currently the only way to get a reliable global shortcut on Linux."
	}
	if os.Getenv("DISPLAY") == "" {
		return "No X11 display detected: the global shortcut cannot be registered."
	}
	return ""
}
