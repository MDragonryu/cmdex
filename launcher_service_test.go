package main

import (
	"sync"
	"testing"
	"time"
)

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
