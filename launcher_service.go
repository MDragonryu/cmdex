package main

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"

	"cmdex/globalhotkey"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// DefaultLauncherShortcut is the out-of-the-box global shortcut: Cmd+Shift+K on
// macOS, Ctrl+Shift+K elsewhere. It deliberately avoids Spotlight (Cmd+Space),
// the macOS emoji picker (Ctrl+Cmd+Space) and the Windows/Linux window menu
// (Alt+Space).
const DefaultLauncherShortcut = "CmdOrCtrl+Shift+K"

const (
	launcherWindowName = "launcher"
	launcherWindowURL  = "/?window=launcher"

	launcherWidth         = 720
	launcherHeight        = 460
	launcherCenterDivisor = 2

	launcherBackgroundRed   = 15
	launcherBackgroundGreen = 15
	launcherBackgroundBlue  = 20
	launcherBackgroundAlpha = 255
	// launcherExpandedHeight is used once the inline terminal is revealed.
	launcherExpandedHeight = 660

	// launcherTopFraction positions the window in the upper portion of the
	// screen's work area, the way Spotlight and Raycast do.
	launcherTopFraction = 0.16

	// launcherBlurGrace ignores focus-loss events fired immediately after a
	// Show, which some window managers emit while the window is still being
	// raised. Without it the launcher can hide itself the moment it appears.
	launcherBlurGrace = 300 * time.Millisecond
)

// LauncherStatus describes the state of the global shortcut for the settings UI.
type LauncherStatus struct {
	// Supported is false when this build cannot register global hotkeys at all.
	Supported bool `json:"supported"`
	// Enabled mirrors the user's setting, regardless of registration success.
	Enabled bool `json:"enabled"`
	// Registered is true only when the OS actually granted the shortcut.
	Registered    bool   `json:"registered"`
	Shortcut      string `json:"shortcut"`
	Error         string `json:"error"`
	Warning       string `json:"warning"`
	LaunchAtLogin bool   `json:"launchAtLogin"`
	Platform      string `json:"platform"`
}

// LauncherService owns the global quick launcher: its always-on-top window, the
// system-wide shortcut that toggles it, and the dedicated terminal session its
// commands run in.
//
// The window and its terminal session are created once and then only shown and
// hidden, so opening the launcher never rebuilds React state or respawns a
// shell.
type LauncherService struct {
	mu        sync.Mutex
	window    *application.WebviewWindow
	hotkeys   *globalhotkey.Manager
	sessionID string
	shownAt   time.Time
	status    LauncherStatus
}

// ServiceStartup creates the launcher window up front (hidden) and applies the
// persisted launcher settings.
func (s *LauncherService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	s.hotkeys = globalhotkey.NewManager()
	launcherSvc = s

	s.mu.Lock()
	s.createWindowLocked()
	s.mu.Unlock()

	// Registration must not happen inline here. ServiceStartup runs on the main
	// thread before application.Run starts the platform run loop, and the macOS
	// hotkey backend blocks on the main queue — so doing this synchronously
	// would deadlock before the app ever appears.
	//
	// A failure only means the shortcut is unavailable; it must never stop the
	// app from starting. The reason is surfaced through GetStatus.
	go s.ApplySettings()
	return nil
}

// ServiceShutdown releases the global shortcut.
func (s *LauncherService) ServiceShutdown() error {
	if s.hotkeys != nil {
		s.hotkeys.Unregister()
	}
	return nil
}

// createWindowLocked builds the hidden launcher window. Caller must hold s.mu.
func (s *LauncherService) createWindowLocked() {
	if s.window != nil {
		return
	}

	options := application.WebviewWindowOptions{
		Title:              "CmDex Launcher",
		Name:               launcherWindowName,
		URL:                launcherWindowURL,
		Width:              launcherWidth,
		Height:             launcherHeight,
		Frameless:          true,
		AlwaysOnTop:        true,
		Hidden:             true,
		DisableResize:      true,
		UseApplicationMenu: false,
		// Escape is handled in the launcher UI so it can close the inline
		// terminal first; letting the platform hide the window would skip that.
		HideOnEscape: false,
		BackgroundColour: application.NewRGBA(
			launcherBackgroundRed,
			launcherBackgroundGreen,
			launcherBackgroundBlue,
			launcherBackgroundAlpha,
		),
		Mac: application.MacWindow{
			// Float above ordinary windows and follow the user onto whichever
			// Space or full-screen app is active, the way Spotlight does.
			WindowLevel: application.MacWindowLevelFloating,
			CollectionBehavior: application.MacWindowCollectionBehaviorCanJoinAllSpaces |
				application.MacWindowCollectionBehaviorFullScreenAuxiliary |
				application.MacWindowCollectionBehaviorIgnoresCycle,
			InvisibleTitleBarHeight: 0,
		},
	}

	w := wailsApp.Window.NewWithOptions(options)

	w.OnWindowEvent(events.Common.WindowLostFocus, func(event *application.WindowEvent) {
		s.mu.Lock()
		withinGrace := time.Since(s.shownAt) < launcherBlurGrace
		s.mu.Unlock()
		if withinGrace {
			return
		}
		s.Hide()
	})

	s.window = w
}

// ========== Window control ==========

// Show reveals the launcher, positions it on the primary display and focuses it.
func (s *LauncherService) Show() {
	s.mu.Lock()
	s.createWindowLocked()
	w := s.window
	s.shownAt = time.Now()
	s.mu.Unlock()

	if w == nil {
		return
	}

	application.InvokeAsync(func() {
		s.positionWindow(w, launcherHeight)
		w.Show()
		w.Focus()
		// Tell the UI to reset: focus the search field and select existing text.
		wailsApp.Event.Emit(eventNames.LauncherShown)
	})
}

// Hide conceals the launcher without destroying it or its terminal session.
func (s *LauncherService) Hide() {
	s.mu.Lock()
	w := s.window
	s.mu.Unlock()
	if w == nil {
		return
	}
	application.InvokeAsync(func() {
		w.Hide()
		wailsApp.Event.Emit(eventNames.LauncherHidden)
	})
}

// Toggle shows the launcher when hidden and hides it when visible. This is what
// the global shortcut invokes.
func (s *LauncherService) Toggle() {
	s.mu.Lock()
	w := s.window
	s.mu.Unlock()

	if w != nil && w.IsVisible() {
		s.Hide()
		return
	}
	s.Show()
}

// Resize switches the launcher between its compact and expanded heights, used
// when the inline terminal is revealed or dismissed.
func (s *LauncherService) Resize(expanded bool) {
	s.mu.Lock()
	w := s.window
	s.mu.Unlock()
	if w == nil {
		return
	}

	height := launcherHeight
	if expanded {
		height = launcherExpandedHeight
	}

	application.InvokeAsync(func() {
		w.SetSize(launcherWidth, height)
		s.positionWindow(w, height)
	})
}

// positionWindow centres the launcher horizontally and places it in the upper
// portion of the primary display's work area.
//
// Wails alpha.74 exposes no public cursor-position API, so "active display"
// cannot be resolved reliably across platforms; the primary display is used
// instead. See docs/CONFIGURATION.md for the limitation.
func (s *LauncherService) positionWindow(w *application.WebviewWindow, height int) {
	screen := wailsApp.Screen.GetPrimary()
	if screen == nil {
		w.Center()
		return
	}

	area := screen.WorkArea
	x := area.X + (area.Width-launcherWidth)/launcherCenterDivisor
	y := area.Y + int(float64(area.Height)*launcherTopFraction)

	if y+height > area.Y+area.Height {
		y = area.Y + max(0, area.Height-height)
	}
	w.SetPosition(x, y)
}

// ShowMainWindow brings the main CmDex window to the front. The launcher offers
// this so CmDex stays reachable when it was started in background mode.
func (s *LauncherService) ShowMainWindow() {
	s.Hide()
	application.InvokeAsync(func() {
		w, ok := wailsApp.Window.GetByName(mainWindowName)
		if !ok {
			return
		}
		w.Show()
		w.Focus()
	})
}

// ========== Terminal session ==========

// GetSessionID returns the launcher's dedicated terminal session, creating it on
// first use. The session is internal, so it never appears as a tab in the main
// window and never becomes the main window's active session.
func (s *LauncherService) GetSessionID() (string, error) {
	s.mu.Lock()
	existing := s.sessionID
	s.mu.Unlock()
	if existing != "" {
		return existing, nil
	}

	if terminalSvc == nil {
		return "", errors.New("terminal service not initialized")
	}

	info, err := terminalSvc.CreateInternalSession("Launcher")
	if err != nil {
		return "", fmt.Errorf("create launcher terminal session: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	// Another caller may have won the race; keep the first session and discard
	// this one so only a single launcher session ever exists.
	if s.sessionID != "" {
		go func() {
			if err := terminalSvc.CloseSession(info.ID); err != nil {
				fmt.Printf("close duplicate launcher terminal session %s: %v\n", info.ID, err)
			}
		}()
		return s.sessionID, nil
	}
	s.sessionID = info.ID
	return s.sessionID, nil
}

// ========== Settings & shortcut registration ==========

// ValidateShortcut reports whether an accelerator string is a usable global
// shortcut, so the settings UI can reject it before saving.
func (s *LauncherService) ValidateShortcut(accelerator string) error {
	_, err := globalhotkey.ParseChord(accelerator)
	return err
}

// GetStatus returns the current launcher/shortcut state.
func (s *LauncherService) GetStatus() LauncherStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

// ApplySettings re-reads the persisted launcher settings and re-registers the
// global shortcut accordingly. The frontend calls it after changing any
// launcher setting. It returns the resulting status rather than an error so the
// UI can show partial success (for example: enabled, but the combination is
// already taken).
func (s *LauncherService) ApplySettings() LauncherStatus {
	status := LauncherStatus{
		Supported: s.hotkeys != nil && s.hotkeys.Supported(),
		Platform:  runtime.GOOS,
		Shortcut:  DefaultLauncherShortcut,
		Warning:   globalhotkey.EnvironmentWarning(),
	}

	settings, err := db.GetSettings()
	if err != nil {
		status.Error = fmt.Sprintf("read settings: %v", err)
		s.setStatus(status)
		return status
	}

	status.Enabled = settings.LauncherEnabled == nil || *settings.LauncherEnabled
	status.LaunchAtLogin = autostartEnabled()
	if settings.LauncherShortcut != "" {
		status.Shortcut = settings.LauncherShortcut
	}

	if s.hotkeys != nil {
		s.hotkeys.Unregister()
	}

	if !status.Enabled {
		s.setStatus(status)
		return status
	}
	if !status.Supported {
		status.Error = "This build cannot register global shortcuts. Rebuild with CGO enabled."
		s.setStatus(status)
		return status
	}

	chord, err := globalhotkey.ParseChord(status.Shortcut)
	if err != nil {
		status.Error = err.Error()
		s.setStatus(status)
		return status
	}

	if err := s.hotkeys.Register(chord, s.Toggle); err != nil {
		status.Error = registrationHint(err)
		s.setStatus(status)
		return status
	}

	status.Registered = true
	s.setStatus(status)
	return status
}

// SetLaunchAtLogin installs or removes the platform login item and persists the
// preference.
func (s *LauncherService) SetLaunchAtLogin(enabled bool) error {
	previous := autostartEnabled()
	if err := setAutostart(enabled); err != nil {
		return err
	}
	if err := db.SetSettings(AppSettings{LaunchAtLogin: &enabled}); err != nil {
		// The login item is already installed or removed at this point. Undo it
		// so the OS state cannot disagree with the persisted preference.
		if rollbackErr := setAutostart(previous); rollbackErr != nil {
			return fmt.Errorf("persist launch-at-login: %w; restore login item: %w", err, rollbackErr)
		}
		return fmt.Errorf("persist launch-at-login: %w", err)
	}

	s.mu.Lock()
	s.status.LaunchAtLogin = autostartEnabled()
	s.mu.Unlock()
	return nil
}

func (s *LauncherService) setStatus(status LauncherStatus) {
	s.mu.Lock()
	s.status = status
	s.mu.Unlock()
}

// registrationHint turns a raw registration failure into something actionable,
// since the most common macOS cause is a missing permission rather than a
// genuine conflict.
func registrationHint(err error) string {
	if runtime.GOOS == "darwin" {
		return fmt.Sprintf("%v — if this persists, grant CmDex Accessibility access in "+
			"System Settings › Privacy & Security › Accessibility, then re-enable the launcher.", err)
	}
	return err.Error()
}
