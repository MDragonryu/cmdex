//go:build darwin

package main

import "testing"

func TestPresentLauncherWindowNativeMissingWindowIsSafe(t *testing.T) {
	if presentLauncherWindowNative("cmdex-launcher-test-window-does-not-exist", launcherHeight, launcherTopFraction) {
		t.Fatal("native presenter reported success for a missing window")
	}
}
