package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"cmdex/globalhotkey"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type recordingLauncherHotkeyManager struct {
	mu              sync.Mutex
	registers       int
	unregisters     int
	supported       bool
	registrationErr error
}

func (m *recordingLauncherHotkeyManager) Supported() bool { return m.supported }

func (m *recordingLauncherHotkeyManager) Register(globalhotkey.Chord, func()) error {
	m.mu.Lock()
	m.registers++
	err := m.registrationErr
	m.mu.Unlock()
	return err
}

func (m *recordingLauncherHotkeyManager) Unregister() {
	m.mu.Lock()
	m.unregisters++
	m.mu.Unlock()
}

func (m *recordingLauncherHotkeyManager) registerCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.registers
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

	manager := &recordingLauncherHotkeyManager{supported: true}
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

	manager := &recordingLauncherHotkeyManager{supported: true}
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

func TestLauncherServiceStartupDisabledDoesNotRegister(t *testing.T) {
	launcherTestDB(t)
	manager := &recordingLauncherHotkeyManager{supported: true}
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
