package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type recordingLauncherHotkeyManager struct {
	mu              sync.Mutex
	registers       int
	unregisters     int
	registered      bool
	shortcut        string
	callback        func()
	registrationErr error
}

func (m *recordingLauncherHotkeyManager) Register(shortcut string, callback func()) error {
	m.mu.Lock()
	m.registers++
	err := m.registrationErr
	if err == nil {
		m.registered = true
		m.shortcut = shortcut
		m.callback = callback
	}
	m.mu.Unlock()
	return err
}

func (m *recordingLauncherHotkeyManager) Unregister(_ string) error {
	m.mu.Lock()
	m.unregisters++
	m.registered = false
	m.mu.Unlock()
	return nil
}

func (m *recordingLauncherHotkeyManager) IsRegistered(_ string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.registered
}

func (m *recordingLauncherHotkeyManager) Validate(shortcut string) error {
	if !strings.Contains(shortcut, "+") {
		return fmt.Errorf("shortcut %q needs at least one modifier", shortcut)
	}
	return nil
}

func (m *recordingLauncherHotkeyManager) registerCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.registers
}

func (m *recordingLauncherHotkeyManager) unregisterCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.unregisters
}

func (m *recordingLauncherHotkeyManager) invoke() {
	m.mu.Lock()
	callback := m.callback
	m.mu.Unlock()
	if callback != nil {
		callback()
	}
}

func launcherTestDB(t *testing.T) *DB {
	t.Helper()
	previousDB := db
	testDB := newTestDB(t)
	if err := testDB.runMigrations(); err != nil {
		t.Fatalf("runMigrations failed: %v", err)
	}
	db = testDB
	t.Cleanup(func() {
		db = previousDB
		_ = testDB.Close()
	})
	return testDB
}

func useLauncherHotkeyManager(t *testing.T, manager launcherHotkeyManager) {
	t.Helper()
	previousFactory := newLauncherHotkeyManager
	newLauncherHotkeyManager = func() launcherHotkeyManager { return manager }
	t.Cleanup(func() { newLauncherHotkeyManager = previousFactory })
}

func TestFreshSettingsDisableLauncher(t *testing.T) {
	testDB := launcherTestDB(t)

	settings, err := testDB.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings failed: %v", err)
	}
	if settings.LauncherEnabled == nil {
		t.Fatal("fresh settings LauncherEnabled is nil, want false")
	}
	if *settings.LauncherEnabled {
		t.Fatal("fresh settings LauncherEnabled = true, want false")
	}
}

func TestLauncherMacCollectionBehaviorSupportsFullscreenAcrossSpaces(t *testing.T) {
	if launcherMacCollectionBehavior&application.MacWindowCollectionBehaviorCanJoinAllSpaces == 0 {
		t.Fatal("launcher collection behavior must join all Spaces")
	}

	want := application.MacWindowCollectionBehaviorCanJoinAllSpaces |
		application.MacWindowCollectionBehaviorFullScreenAuxiliary |
		application.MacWindowCollectionBehaviorIgnoresCycle |
		application.MacWindowCollectionBehaviorTransient
	if launcherMacCollectionBehavior != want {
		t.Fatalf("launcher collection behavior = %d, want %d", launcherMacCollectionBehavior, want)
	}
}

func TestSettingsPreserveExplicitLauncherValues(t *testing.T) {
	testDB := launcherTestDB(t)
	enabled := true
	if err := testDB.SetSettings(AppSettings{LauncherEnabled: &enabled}); err != nil {
		t.Fatalf("enable launcher: %v", err)
	}
	settings, err := testDB.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings after enable failed: %v", err)
	}
	if settings.LauncherEnabled == nil || !*settings.LauncherEnabled {
		t.Fatal("explicit true launcher setting was not persisted")
	}

	disabled := false
	if err := testDB.SetSettings(AppSettings{LauncherEnabled: &disabled}); err != nil {
		t.Fatalf("disable launcher: %v", err)
	}
	settings, err = testDB.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings after disable failed: %v", err)
	}
	if settings.LauncherEnabled == nil || *settings.LauncherEnabled {
		t.Fatal("explicit false launcher setting was not persisted")
	}
}

func TestLauncherServiceApplySettingsTreatsNilAsDisabled(t *testing.T) {
	testDB := launcherTestDB(t)
	if _, err := testDB.conn.Exec(`UPDATE app_settings SET data = '{"launcherEnabled":null}'`); err != nil {
		t.Fatalf("seed nil launcher setting: %v", err)
	}

	manager := &recordingLauncherHotkeyManager{}
	service := &LauncherService{hotkeys: manager}
	status := service.ApplySettings()
	if status.Enabled {
		t.Fatal("nil LauncherEnabled produced an enabled status")
	}
	if status.Registered {
		t.Fatal("nil LauncherEnabled registered a global shortcut")
	}
	if got := manager.registerCount(); got != 0 {
		t.Fatalf("Register called %d times for nil LauncherEnabled, want 0", got)
	}
}

func TestLauncherServiceApplySettingsExplicitEnableRegisters(t *testing.T) {
	testDB := launcherTestDB(t)
	enabled := true
	if err := testDB.SetSettings(AppSettings{LauncherEnabled: &enabled}); err != nil {
		t.Fatalf("enable launcher: %v", err)
	}

	manager := &recordingLauncherHotkeyManager{}
	service := &LauncherService{hotkeys: manager}
	status := service.ApplySettings()
	if !status.Enabled {
		t.Fatal("explicit true LauncherEnabled produced a disabled status")
	}
	if !status.Registered {
		t.Fatal("explicit true LauncherEnabled did not register the shortcut")
	}
	if got := manager.registerCount(); got != 1 {
		t.Fatalf("Register called %d times for explicit true LauncherEnabled, want 1", got)
	}
}

func TestLauncherServiceApplySettingsReregistersAndShutdownUnregisters(t *testing.T) {
	testDB := launcherTestDB(t)
	enabled := true
	if err := testDB.SetSettings(AppSettings{LauncherEnabled: &enabled}); err != nil {
		t.Fatalf("enable launcher: %v", err)
	}

	manager := &recordingLauncherHotkeyManager{}
	service := &LauncherService{hotkeys: manager}
	if status := service.ApplySettings(); !status.Registered {
		t.Fatalf("initial ApplySettings status = %+v, want registered", status)
	}
	if status := service.ApplySettings(); !status.Registered {
		t.Fatalf("second ApplySettings status = %+v, want registered", status)
	}
	if got := manager.registerCount(); got != 2 {
		t.Fatalf("Register called %d times, want 2", got)
	}
	if got := manager.unregisterCount(); got != 1 {
		t.Fatalf("Unregister called %d times before shutdown, want 1", got)
	}

	if err := service.ServiceShutdown(); err != nil {
		t.Fatalf("ServiceShutdown failed: %v", err)
	}
	if got := manager.unregisterCount(); got != 2 {
		t.Fatalf("Unregister called %d times after shutdown, want 2", got)
	}
}

func TestLauncherServiceApplySettingsReportsUnsupportedManager(t *testing.T) {
	testDB := launcherTestDB(t)
	enabled := true
	if err := testDB.SetSettings(AppSettings{LauncherEnabled: &enabled}); err != nil {
		t.Fatalf("enable launcher: %v", err)
	}

	manager := &recordingLauncherHotkeyManager{
		registrationErr: errors.New("global shortcuts are not supported on this platform"),
	}
	service := &LauncherService{hotkeys: manager}
	status := service.ApplySettings()
	if status.Supported {
		t.Fatalf("unsupported manager status = %+v, want Supported=false", status)
	}
	if status.Registered {
		t.Fatalf("unsupported manager status = %+v, want Registered=false", status)
	}
	if status.Error == "" {
		t.Fatal("unsupported manager returned no status error")
	}
}

func TestLauncherServiceValidateShortcutUsesManagerValidator(t *testing.T) {
	manager := &recordingLauncherHotkeyManager{}
	service := &LauncherService{hotkeys: manager}
	if err := service.ValidateShortcut("Ctrl+Shift+K"); err != nil {
		t.Fatalf("valid shortcut rejected: %v", err)
	}
	if err := service.ValidateShortcut("K"); err == nil {
		t.Fatal("bare key accepted as a global shortcut")
	}
}

func TestLauncherServiceStartupDisabledDoesNotRegister(t *testing.T) {
	launcherTestDB(t)
	manager := &recordingLauncherHotkeyManager{}
	useLauncherHotkeyManager(t, manager)

	previousWailsApp := wailsApp
	wailsApp = nil
	t.Cleanup(func() { wailsApp = previousWailsApp })

	service := &LauncherService{}
	if err := service.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatalf("ServiceStartup failed: %v", err)
	}

	deadline := time.Now().Add(time.Second)
	for service.GetStatus().Platform == "" && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := manager.registerCount(); got != 0 {
		t.Fatalf("startup called Register %d times with launcher disabled, want 0", got)
	}
	if status := service.GetStatus(); status.Enabled || status.Registered {
		t.Fatalf("startup status = %+v, want disabled and unregistered", status)
	}
}

func TestLauncherServiceApplySettingsWaitsForPreviousApplication(t *testing.T) {
	previousDB := db
	testDB := newTestDB(t)
	if err := testDB.runMigrations(); err != nil {
		t.Fatalf("runMigrations failed: %v", err)
	}
	db = testDB
	t.Cleanup(func() {
		db = previousDB
		_ = testDB.Close()
	})

	service := &LauncherService{}
	service.applyMu.Lock()
	result := make(chan LauncherStatus, 1)
	go func() { result <- service.ApplySettings() }()

	select {
	case <-result:
		t.Fatal("ApplySettings completed while another settings application was active")
	case <-time.After(25 * time.Millisecond):
	}

	service.applyMu.Unlock()
	select {
	case <-result:
	case <-time.After(time.Second):
		t.Fatal("ApplySettings did not proceed after the previous application completed")
	}

	// Startup and frontend settings changes can arrive concurrently. Exercise
	// that path repeatedly so the race-enabled test also covers status writes
	// while applications are queued behind applyMu.
	const concurrentCalls = 32
	var wg sync.WaitGroup
	statuses := make(chan LauncherStatus, concurrentCalls)
	for i := 0; i < concurrentCalls; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			statuses <- service.ApplySettings()
		}()
	}
	wg.Wait()
	close(statuses)
	for status := range statuses {
		if status.Platform == "" {
			t.Error("concurrent ApplySettings returned an empty platform")
		}
	}
}
