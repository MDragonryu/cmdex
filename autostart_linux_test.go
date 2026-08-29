//go:build linux

package main

import "testing"

func TestQuoteDesktopEntryArg(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want string
	}{
		{name: "spaces", arg: "/opt/CmDex App/cmdex", want: `"/opt/CmDex App/cmdex"`},
		{name: "field code marker", arg: "/opt/100%/cmdex", want: `"/opt/100%%/cmdex"`},
		{name: "desktop metacharacters", arg: "/opt/CmDex\\\"" + "`" + "/cmdex", want: `"` + `/opt/CmDex\\\"` + "\\`" + `/cmdex"`},
		{name: "control characters", arg: "/opt/CmDex\tApp\ncmdex", want: `"/opt/CmDex\tApp\ncmdex"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := quoteDesktopEntryArg(tt.arg); got != tt.want {
				t.Fatalf("quoteDesktopEntryArg(%q) = %q, want %q", tt.arg, got, tt.want)
			}
		})
	}
}
