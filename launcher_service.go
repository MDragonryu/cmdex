package main

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// DefaultLauncherShortcut is the out-of-the-box global shortcut: Cmd+Shift+K on
// macOS, Ctrl+Shift+K elsewhere. It deliberately avoids Spotlight (Cmd+Space),
// the macOS emoji picker (Ctrl+Cmd+Space) and the Windows/Linux window menu
// (Alt+Space).
const DefaultLauncherShortcut = "CmdOrCtrl+Shift+K"

const (
	launcherWindowName  = "launcher"
	launcherWindowTitle = "CmDex Launcher"
	launcherWindowURL   = "/?window=launcher"

	launcherWidth         = 720
	launcherHeight        = 460
	launcherCenterDivisor = 2

	launcherBackgroundRed           = 15
	launcherBackgroundGreen         = 15
	launcherBackgroundBlue          = 20
	launcherBackgroundAlpha         = 255
	launcherMinimumAcceleratorParts = 2
	// launcherExpandedHeight is used once the inline terminal is revealed.
	launcherExpandedHeight = 660
	// launcherSessionRecoveryTimeout bounds frontend waiters while the eager
	// launcher shell is starting (or being recovered after a failed startup).
	// The creator itself is never abandoned: if it eventually succeeds after a
	// timeout, it publishes the session for the next invocation.
	launcherSessionRecoveryTimeout = 5 * time.Second

	// launcherMacCollectionBehavior lets the panel overlay an unrelated
	// fullscreen app while remaining available on every Space. MoveToActiveSpace
	// is intentionally omitted: it can move the launcher into Cmdex's Space.
	launcherMacCollectionBehavior = application.MacWindowCollectionBehaviorCanJoinAllSpaces |
		application.MacWindowCollectionBehaviorFullScreenAuxiliary |
		application.MacWindowCollectionBehaviorIgnoresCycle |
		application.MacWindowCollectionBehaviorStationary

	// launcherTopFraction positions the window in the upper portion of the
	// screen's work area, the way Spotlight and Raycast do.
	launcherTopFraction = 0.16

	// launcherBlurGrace ignores focus-loss events fired immediately after a
	// Show, which some window managers emit while the window is still being
	// raised. Without it the launcher can hide itself the moment it appears.
	launcherBlurGrace = 300 * time.Millisecond
)

func launcherMacOptions() application.MacWindow {
	return application.MacWindow{
		// Wails creates a real non-activating NSPanel on macOS. Show and Focus
		// then order and key only this panel without activating CmDex or
		// replacing the active app's menu bar.
		WindowClass: application.MacWindowClassPanel,
		PanelPreferences: application.MacPanelPreferences{
			NonActivating:          true,
			BecomesKeyOnlyIfNeeded: false,
			FloatingPanel:          true,
		},
		WindowLevel:             application.MacWindowLevelFloating,
		CollectionBehavior:      launcherMacCollectionBehavior,
		InvisibleTitleBarHeight: 0,
	}
}

// LauncherStatus describes the state of the global shortcut for the settings UI.
type LauncherStatus struct {
	// Supported is false when Wails cannot provide global shortcuts on this
	// platform/build. Registration failures are surfaced separately in Error.
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

// launcherHotkeyManager is the small part of Wails' global shortcut manager
// used by the launcher. Keeping the service dependent on this interface lets
// tests verify registration behavior without starting a desktop application.
type launcherHotkeyManager interface {
	Register(string, func()) error
	Unregister(string) error
	IsRegistered(string) bool
}

type wailsLauncherHotkeyManager struct {
	manager *application.GlobalShortcutManager
}

func (m wailsLauncherHotkeyManager) Register(accelerator string, callback func()) error {
	return m.manager.Register(accelerator, callback)
}

func (m wailsLauncherHotkeyManager) Unregister(accelerator string) error {
	return m.manager.Unregister(accelerator)
}

func (m wailsLauncherHotkeyManager) IsRegistered(accelerator string) bool {
	return m.manager.IsRegistered(accelerator)
}

// Validate uses Wails' accelerator parser through MenuItem.SetAccelerator.
// Wails beta.12 does not export the parser, so the temporary item provides the
// same validation path without registering an OS shortcut.
func (m wailsLauncherHotkeyManager) Validate(accelerator string) error {
	item := application.NewMenuItem("launcher-shortcut-validation")
	defer item.Destroy()
	item.SetAccelerator(accelerator)
	if item.GetAccelerator() == "" {
		return fmt.Errorf("invalid accelerator %q", accelerator)
	}

	// Global shortcuts must include a modifier. Wails permits bare menu keys,
	// but registering those globally would make the key unusable elsewhere.
	parts := strings.Split(accelerator, "+")
	if len(parts) < launcherMinimumAcceleratorParts {
		return fmt.Errorf("shortcut %q needs at least one modifier", accelerator)
	}
	for _, part := range parts[:len(parts)-1] {
		switch strings.ToLower(strings.TrimSpace(part)) {
		case "cmdorctrl", "commandorcontrol", "cmd", "command", "ctrl",
			"control", "optionoralt", "alt", "option", "opt", "shift",
			"super", "meta", "win", "windows":
		default:
			return fmt.Errorf("shortcut %q needs at least one modifier", accelerator)
		}
	}
	return nil
}

var newLauncherHotkeyManager = func() launcherHotkeyManager {
	if wailsApp == nil || wailsApp.GlobalShortcut == nil {
		return nil
	}
	return wailsLauncherHotkeyManager{manager: wailsApp.GlobalShortcut}
}

// LauncherService owns the global quick launcher: its always-on-top window, the
// system-wide shortcut that toggles it, and the dedicated terminal session its
// commands run in. The window and its terminal session are created once and
// then only shown and hidden, so opening the launcher never rebuilds React
// state or respawns a shell.
type LauncherService struct {
	mu sync.Mutex
	// sessionMu serializes launcher-session creation/replacement independently
	// from window and shortcut state. This is intentionally single-flight: the
	// eager startup and concurrent GetSessionID calls must never create two
	// hidden PTYs.
	sessionMu          sync.Mutex
	sessionCreating    bool
	sessionDone        chan struct{}
	sessionShutdown    chan struct{}
	sessionShuttingDown bool
	sessionErr         string
	createSessionFn    func() (*SessionInfo, error)
	closeSessionFn     func(string) error
	sessionExistsFn    func(string) bool
	// applyMu serializes the complete settings application transaction. In
	// particular, the settings read must stay ordered with unregister,
	// register, and the final status publication when startup and frontend
	// settings changes overlap.
	applyMu            sync.Mutex
	loginMu            sync.Mutex
	loginOp            uint64
	window             *application.WebviewWindow
	hotkeys            launcherHotkeyManager
	stopStart          func()
	registeredShortcut string
	sessionID          string
	shownAt            time.Time
	status             LauncherStatus
}

type launcherSessionResult struct {
	info *SessionInfo
	err  error
}

// ServiceStartup creates the launcher window up front (hidden) and applies the
// persisted launcher settings.
func (s *LauncherService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
	s.hotkeys = newLauncherHotkeyManager()
	launcherSvc = s
	s.sessionMu.Lock()
	if s.sessionShutdown == nil {
		s.sessionShutdown = make(chan struct{})
	}
	s.sessionMu.Unlock()

	// App.ServiceStartup normally initializes wailsApp first. Keeping window
	// creation conditional makes the lifecycle seam safe for unit tests too.
	if wailsApp != nil {
		s.mu.Lock()
		s.createWindowLocked()
		s.mu.Unlock()
	}

	// Register after Wails has started its platform loop. This makes OS-level
	// failures available to ApplySettings instead of deferring them to Wails'
	// generic error handler as happens for pre-Run registrations.
	if wailsApp != nil && wailsApp.Event != nil {
		s.stopStart = wailsApp.Event.OnApplicationEvent(
			events.Common.ApplicationStarted,
			func(*application.ApplicationEvent) {
				// TerminalService is registered before LauncherService. Starting
				// here ensures its shell integration and default user session are
				// ready, while keeping app startup responsive.
				go s.ensureSession(launcherSessionRecoveryTimeout)
				s.ApplySettings()
			},
		)
	} else {
		// The nil-app path is used by unit tests which inject the manager seam.
		go s.ensureSession(launcherSessionRecoveryTimeout)
		go s.ApplySettings()
	}
	return nil
}

// ServiceShutdown releases the global shortcut.
func (s *LauncherService) ServiceShutdown() error {
	if s.stopStart != nil {
		s.stopStart()
		s.stopStart = nil
	}
	s.applyMu.Lock()
	if s.hotkeys != nil && s.registeredShortcut != "" {
		_ = s.hotkeys.Unregister(s.registeredShortcut)
		s.registeredShortcut = ""
	}
	s.applyMu.Unlock()

	s.sessionMu.Lock()
	if !s.sessionShuttingDown {
		s.sessionShuttingDown = true
		if s.sessionShutdown != nil {
			close(s.sessionShutdown)
		}
	}
	id := s.sessionID
	s.sessionID = ""
	s.sessionMu.Unlock()
	if id != "" {
		s.closeLauncherSession(id)
	}
	// A creator that is already in the backend cannot be cancelled safely, but
	// it observes sessionShuttingDown and closes any late result. Waiters are
	// woken by sessionShutdown above; do not block shutdown on PTY startup.
	return nil
}

// createWindowLocked builds the hidden launcher window. Caller must hold s.mu.
func (s *LauncherService) createWindowLocked() {
	if s.window != nil {
		return
	}

	options := application.WebviewWindowOptions{
		Title:              launcherWindowTitle,
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
		Mac: launcherMacOptions(),
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

// Show reveals and focuses the launcher. On macOS Wails' non-activating panel
// semantics keep the previously active app active while the native positioning
// helper selects the display under the pointer. Other platforms use the Wails
// screen API fallback.
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
		var targetDisplay uint32
		presentLauncherWindow(
			func() { targetDisplay = launcherDisplayUnderMouseNative() },
			func() { w.Show() },
			func() bool {
				return positionLauncherWindowNative(
					w.NativeWindow(),
					launcherWidth,
					launcherHeight,
					launcherTopFraction,
					targetDisplay,
				)
			},
			func() { s.positionWindow(w, launcherHeight) },
			func() { w.Focus() },
		)
		// Tell the UI to reset: focus the search field and select existing text.
		wailsApp.Event.Emit(eventNames.LauncherShown)
	})
}

// Hide conceals only the launcher panel without activating or focusing the
// main Cmdex window, and without destroying the panel or its terminal session.
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
		if !positionLauncherWindowNative(w.NativeWindow(), launcherWidth, height, launcherTopFraction, 0) {
			s.positionWindow(w, height)
		}
	})
}

// positionWindow is the cross-platform fallback for the native macOS
// presenter. On macOS the native presenter chooses the display containing the
// current pointer (or the existing window screen when the pointer is between
// displays), because Wails does not expose a public cursor-position API.
func (s *LauncherService) positionWindow(w *application.WebviewWindow, height int) {
	var screen *application.Screen
	if current, err := w.GetScreen(); err == nil {
		screen = current
	}
	if screen == nil && wailsApp != nil && wailsApp.Screen != nil {
		screen = wailsApp.Screen.GetPrimary()
	}
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

// GetSessionID returns the launcher's eagerly-created dedicated terminal
// session. If eager startup failed, one bounded single-flight recovery attempt
// is made on demand. A session removed from TerminalService is never returned
// as a stale ID.
func (s *LauncherService) GetSessionID() (string, error) {
	return s.ensureSession(launcherSessionRecoveryTimeout)
}

// ensureSession is the single-flight session creator used by eager startup
// and GetSessionID. It deliberately does not hold sessionMu while starting a
// PTY: the bounded waiters can observe shutdown and the backend can perform
// its blocking shell setup without blocking window operations.
func (s *LauncherService) ensureSession(timeout time.Duration) (string, error) {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	for recovery := 0; recovery < 2; recovery++ {
		s.sessionMu.Lock()
		if s.sessionShuttingDown {
			s.sessionMu.Unlock()
			return "", errors.New("launcher service is shutting down")
		}
		if s.sessionID != "" && s.sessionExists(s.sessionID) {
			id := s.sessionID
			s.sessionMu.Unlock()
			return id, nil
		}
		staleID := s.sessionID
		if staleID != "" {
			s.sessionID = ""
		}
		if s.sessionCreating {
			done := s.sessionDone
			shutdown := s.sessionShutdown
			s.sessionMu.Unlock()
			if staleID != "" {
				s.closeLauncherSession(staleID)
			}
			select {
			case <-done:
				continue
			case <-shutdown:
				return "", errors.New("launcher service is shutting down")
			case <-deadline.C:
				return "", errors.New("launcher terminal startup timed out")
			}
		}

		s.sessionCreating = true
		s.sessionDone = make(chan struct{})
		done := s.sessionDone
		sessionShutdown := s.sessionShutdown
		create := s.createLauncherSession
		s.sessionMu.Unlock()

		if staleID != "" {
			s.closeLauncherSession(staleID)
		}
		result := make(chan launcherSessionResult, 1)
		go func() {
			info, err := create()
			s.completeSessionCreate(done, info, err)
			result <- launcherSessionResult{info: info, err: err}
		}()
		var info *SessionInfo
		var err error
		select {
		case created := <-result:
			info, err = created.info, created.err
		case <-sessionShutdown:
			return "", errors.New("launcher service is shutting down")
		case <-deadline.C:
			return "", errors.New("launcher terminal startup timed out")
		}
		if err != nil {
			return "", fmt.Errorf("create launcher terminal session: %w", err)
		}
		if info == nil {
			return "", errors.New("create launcher terminal session: empty session response")
		}
		if sessionShutdown != nil {
			select {
			case <-sessionShutdown:
				return "", errors.New("launcher service is shutting down")
			default:
			}
		}
		return info.ID, nil
	}
	return "", errors.New("launcher terminal startup timed out")
}

func (s *LauncherService) completeSessionCreate(done chan struct{}, info *SessionInfo, err error) {
	s.sessionMu.Lock()
	s.sessionCreating = false
	if err == nil && info != nil && !s.sessionShuttingDown {
		s.sessionID = info.ID
	} else if err != nil {
		s.sessionErr = fmt.Sprintf("launcher terminal unavailable: %v", err)
	} else if info == nil {
		s.sessionErr = "launcher terminal unavailable: empty session response"
	}
	shuttingDown := s.sessionShuttingDown
	close(done)
	s.sessionMu.Unlock()

	if err != nil {
		s.publishSessionError(fmt.Sprintf("launcher terminal unavailable: %v", err))
	} else if info == nil {
		s.publishSessionError("launcher terminal unavailable: empty session response")
	} else if !shuttingDown {
		s.publishSessionError("")
	}
	if shuttingDown && err == nil && info != nil {
		s.closeLauncherSession(info.ID)
	}
}

func (s *LauncherService) createLauncherSession() (*SessionInfo, error) {
	if s.createSessionFn != nil {
		return s.createSessionFn()
	}
	if terminalSvc == nil {
		return nil, errors.New("terminal service not initialized")
	}
	return terminalSvc.CreateInternalSession("Launcher")
}

func (s *LauncherService) closeLauncherSession(id string) {
	if id == "" {
		return
	}
	closeFn := s.closeSessionFn
	if closeFn == nil && terminalSvc != nil {
		closeFn = terminalSvc.CloseSession
	}
	if closeFn != nil {
		if err := closeFn(id); err != nil {
			fmt.Printf("close launcher terminal session %s: %v\n", id, err)
		}
	}
}

func (s *LauncherService) sessionExists(id string) bool {
	// Existence, rather than the transient Running bit, is the ownership
	// signal here. TerminalService auto-restarts unintentional shell exits and
	// Write lazily restarts an intentionally stopped shell; replacing during
	// either hand-off would create an unnecessary second PTY.
	if s.sessionExistsFn != nil {
		return s.sessionExistsFn(id)
	}
	return terminalSvc != nil && terminalSvc.hasSession(id)
}

func (s *LauncherService) publishSessionError(message string) {
	s.sessionMu.Lock()
	previous := s.sessionErr
	s.sessionErr = message
	s.sessionMu.Unlock()

	s.mu.Lock()
	if message != "" || s.status.Error == previous {
		s.status.Error = message
	}
	s.mu.Unlock()
}

// ========== Settings & shortcut registration ==========

// ValidateShortcut reports whether an accelerator string is a usable global
// shortcut, so the settings UI can reject it before saving.
func (s *LauncherService) ValidateShortcut(accelerator string) error {
	if validator, ok := s.hotkeys.(interface{ Validate(string) error }); ok {
		return validator.Validate(accelerator)
	}
	if wailsApp != nil && wailsApp.GlobalShortcut != nil {
		return (wailsLauncherHotkeyManager{manager: wailsApp.GlobalShortcut}).Validate(accelerator)
	}
	return errors.New("global shortcut manager is not initialized")
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
	s.applyMu.Lock()
	defer s.applyMu.Unlock()

	status := LauncherStatus{
		Supported: s.hotkeys != nil,
		Platform:  runtime.GOOS,
		Shortcut:  DefaultLauncherShortcut,
	}
	s.sessionMu.Lock()
	status.Error = s.sessionErr
	s.sessionMu.Unlock()

	settings, err := db.GetSettings()
	if err != nil {
		status.Error = fmt.Sprintf("read settings: %v", err)
		s.setStatus(status)
		return status
	}

	// The launcher is opt-in. A nil value can occur in legacy/hand-edited
	// settings and must remain disabled rather than unexpectedly registering a
	// global shortcut during startup.
	status.Enabled = settings.LauncherEnabled != nil && *settings.LauncherEnabled
	status.LaunchAtLogin = autostartEnabled()
	if settings.LauncherShortcut != "" {
		status.Shortcut = settings.LauncherShortcut
	}

	if s.hotkeys != nil && s.registeredShortcut != "" {
		_ = s.hotkeys.Unregister(s.registeredShortcut)
		s.registeredShortcut = ""
	}

	if !status.Enabled {
		s.setStatus(status)
		return status
	}
	if s.hotkeys == nil {
		status.Supported = false
		status.Error = "Global shortcuts are unavailable on this platform."
		s.setStatus(status)
		return status
	}

	if validator, ok := s.hotkeys.(interface{ Validate(string) error }); ok {
		if err := validator.Validate(status.Shortcut); err != nil {
			status.Error = err.Error()
			s.setStatus(status)
			return status
		}
	}

	if err := s.hotkeys.Register(status.Shortcut, s.Toggle); err != nil {
		status.Error = registrationHint(err)
		if strings.Contains(strings.ToLower(err.Error()), "not supported") {
			status.Supported = false
		}
		s.setStatus(status)
		return status
	}

	status.Registered = s.hotkeys.IsRegistered(status.Shortcut)
	if status.Registered {
		s.registeredShortcut = status.Shortcut
	}
	s.setStatus(status)
	return status
}

// SetLaunchAtLogin installs or removes the platform login item and persists the
// preference.
func (s *LauncherService) SetLaunchAtLogin(enabled bool) error {
	// The OS item and its persisted preference are one transaction from the
	// user's perspective. Serialize independent Wails callers so two toggles
	// cannot observe/restore each other's intermediate state.
	op := atomic.AddUint64(&s.loginOp, 1)
	s.loginMu.Lock()
	defer s.loginMu.Unlock()
	// If a newer request arrived while this one waited for the lock, let that
	// request own the final state rather than applying an older user intent.
	if op != atomic.LoadUint64(&s.loginOp) {
		return nil
	}

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

// registrationHint turns a raw Wails registration failure into text suitable
// for the launcher settings status line.
func registrationHint(err error) string {
	return err.Error()
}
