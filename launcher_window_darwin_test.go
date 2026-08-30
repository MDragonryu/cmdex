//go:build darwin

package main

import "testing"

func TestPresentLauncherWindowNativeMissingWindowIsSafe(t *testing.T) {
	if presentLauncherWindowNative("cmdex-launcher-test-window-does-not-exist", launcherWidth, launcherHeight, launcherTopFraction, true) {
		t.Fatal("native presenter reported success for a missing window")
	}
}

func TestPrepareLauncherWindowNativeMissingWindowIsSafe(t *testing.T) {
	if prepareLauncherWindowNative("cmdex-launcher-test-window-does-not-exist") {
		t.Fatal("native preparer reported success for a missing window")
	}
}
