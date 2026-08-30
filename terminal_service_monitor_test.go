package main

import (
	"testing"
	"time"
)

func TestWaitForAutoRestartStopsDuringDelay(t *testing.T) {
	stopCh := make(chan struct{})
	result := make(chan bool, 1)
	timer := time.NewTimer(time.Hour)
	go func() { result <- waitForAutoRestartTimer(stopCh, timer) }()

	close(stopCh)

	select {
	case restarted := <-result:
		if restarted {
			t.Fatal("waitForAutoRestart reported a restart after the session was stopped")
		}
	case <-time.After(autoRestartDelay):
		t.Fatal("waitForAutoRestart did not observe the stop signal")
	}
}
