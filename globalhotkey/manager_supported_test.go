//go:build windows || ((darwin || linux) && cgo)

package globalhotkey

import (
	"testing"
	"time"
)

func TestLockPlatformTimesOutWithoutOverlappingOperation(t *testing.T) {
	manager := NewManager()
	manager.platformMu.Lock()
	defer manager.platformMu.Unlock()

	if manager.lockPlatform(5 * time.Millisecond) {
		t.Fatal("lockPlatform acquired a gate held by an in-flight platform operation")
	}
}

func TestLockPlatformAcquiresAfterOperationCompletes(t *testing.T) {
	manager := NewManager()
	manager.platformMu.Lock()
	go func() {
		time.Sleep(5 * time.Millisecond)
		manager.platformMu.Unlock()
	}()

	if !manager.lockPlatform(time.Second) {
		t.Fatal("lockPlatform did not acquire the gate after the prior operation completed")
	}
	manager.platformMu.Unlock()
}
