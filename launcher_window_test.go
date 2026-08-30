package main

import (
	"reflect"
	"testing"
)

func TestPresentLauncherWindowUsesNativePlacementBeforeFocus(t *testing.T) {
	var calls []string
	presentLauncherWindow(
		func() { calls = append(calls, "show") },
		func() bool { calls = append(calls, "native-place"); return true },
		func() { calls = append(calls, "fallback-place") },
		func() { calls = append(calls, "focus") },
	)

	want := []string{"show", "native-place", "focus"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("presentation sequence = %v, want %v", calls, want)
	}
}

func TestPresentLauncherWindowFallsBackWhenNativePlacementUnavailable(t *testing.T) {
	var calls []string
	presentLauncherWindow(
		func() { calls = append(calls, "show") },
		func() bool { calls = append(calls, "native-place"); return false },
		func() { calls = append(calls, "fallback-place") },
		func() { calls = append(calls, "focus") },
	)

	want := []string{"show", "native-place", "fallback-place", "focus"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("presentation sequence = %v, want %v", calls, want)
	}
}
