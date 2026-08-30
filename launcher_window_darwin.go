//go:build darwin

package main

/*
#cgo CFLAGS: -mmacosx-version-min=10.13 -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <AppKit/AppKit.h>
#import <Cocoa/Cocoa.h>
#include <stdbool.h>
#include <stdlib.h>

static NSWindow* launcherWindowForTitle(const char* title) {
	if (title == NULL) {
		return nil;
	}
	NSString* expectedTitle = [NSString stringWithUTF8String:title];
	if (expectedTitle == nil) {
		return nil;
	}
	for (NSWindow* window in [[NSApplication sharedApplication] windows]) {
		// Wails beta.12 does not expose its native NSWindow handle. The title is
		// an exact, app-owned constant and only one launcher window is created.
		// Restricting the search to NSApp's windows avoids another application's
		// similarly named window being raised.
		if ([window.title isEqualToString:expectedTitle]) {
			return window;
		}
	}
	return nil;
}

static NSScreen* screenForMouseOrWindow(NSWindow* window) {
	NSPoint mouse = [NSEvent mouseLocation];
	for (NSScreen* screen in [NSScreen screens]) {
		if (NSPointInRect(mouse, screen.frame)) {
			return screen;
		}
	}
	if (window.screen != nil) {
		return window.screen;
	}
	return [NSScreen mainScreen];
}

static bool presentLauncherWindowNative(const char* title, int height, double topFraction) {
	(void)height;
	NSWindow* window = launcherWindowForTitle(title);
	NSScreen* screen = screenForMouseOrWindow(window);
	if (window == nil || screen == nil) {
		return false;
	}

	// This function is called from application.InvokeAsync, Wails' AppKit
	// main-thread dispatcher. Keep all AppKit operations here on that thread.
	window.collectionBehavior = NSWindowCollectionBehaviorMoveToActiveSpace |
		NSWindowCollectionBehaviorFullScreenAuxiliary |
		NSWindowCollectionBehaviorIgnoresCycle |
		NSWindowCollectionBehaviorTransient;
	window.level = NSFloatingWindowLevel;

	NSRect workArea = screen.visibleFrame;
	NSRect frame = window.frame;
	CGFloat x = NSMidX(workArea) - (frame.size.width / 2.0);
	CGFloat topOffset = workArea.size.height * (CGFloat)topFraction;
	CGFloat y = NSMaxY(workArea) - topOffset - frame.size.height;
	if (y < NSMinY(workArea)) {
		y = NSMinY(workArea);
	}
	[window setFrameOrigin:NSMakePoint(x, y)];

	// Re-order after activation so MoveToActiveSpace is evaluated against the
	// Space that was active when the shortcut was pressed. orderOut clears any
	// stale ordering from the launcher's previous Space.
	[window orderOut:nil];
	[[NSApplication sharedApplication] activateIgnoringOtherApps:YES];
	[window orderFrontRegardless];
	[window makeKeyAndOrderFront:nil];
	[window makeKeyWindow];
	return true;
}
*/
import "C"
import "unsafe"

// presentLauncherWindowNative activates and raises the launcher on the
// AppKit screen under the pointer. The title is safe here because it is the
// unique constant used when creating the one launcher window; Wails beta.12
// does not expose the native NSWindow pointer publicly.
func presentLauncherWindowNative(title string, height int, topFraction float64) bool {
	nativeTitle := C.CString(title)
	defer C.free(unsafe.Pointer(nativeTitle))
	return bool(C.presentLauncherWindowNative(nativeTitle, C.int(height), C.double(topFraction)))
}

// presentLauncherWindow keeps the show/placement/focus sequence injectable so
// the global-shortcut path can be tested without starting a desktop app.
func presentLauncherWindow(show func(), nativePlace func() bool, fallbackPlace func(), focus func()) {
	show()
	if !nativePlace() {
		fallbackPlace()
	}
	focus()
}
