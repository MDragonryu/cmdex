//go:build darwin

package main

/*
#cgo CFLAGS: -mmacosx-version-min=10.13 -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <AppKit/AppKit.h>
#import <Cocoa/Cocoa.h>
#include <stdbool.h>
#include <math.h>
#include <stdlib.h>

static NSWindow* launcherWindowForTitle(const char* title) {
	if (title == NULL) {
		return nil;
	}
	NSString* expectedTitle = [NSString stringWithUTF8String:title];
	if (expectedTitle == nil) {
		return nil;
	}
	NSWindow* match = nil;
	NSUInteger matches = 0;
	for (NSWindow* window in [[NSApplication sharedApplication] windows]) {
		// Wails beta.12 does not expose its native NSWindow handle. The title is
		// an exact, app-owned constant and only one launcher window is created.
		// Restricting the search to NSApp's windows avoids another application's
		// similarly named window being raised.
		if ([window.title isEqualToString:expectedTitle]) {
			match = window;
			matches++;
		}
	}
	return matches == 1 ? match : nil;
}

static NSScreen* screenForWindowOrMain(NSWindow* window) {
	if (window.screen != nil) {
		return window.screen;
	}
	return [NSScreen mainScreen];
}

static NSRunningApplication* launcherPreviousApplication = nil;
static NSScreen* launcherTargetScreen = nil;

static void capturePreviousFrontmostApplication(void) {
	// A repeated Show while the launcher is already active must not replace the
	// original owner with CmDex itself. Retain the object because the hide can
	// happen after the app's event callback returns.
	if (launcherPreviousApplication != nil && !launcherPreviousApplication.terminated) {
		return;
	}
	NSRunningApplication* frontmost = [[NSWorkspace sharedWorkspace] frontmostApplication];
	NSRunningApplication* current = [NSRunningApplication currentApplication];
	if (frontmost == nil || current == nil || frontmost.processIdentifier == current.processIdentifier) {
		return;
	}
	[launcherPreviousApplication release];
	launcherPreviousApplication = [frontmost retain];
}

static void reactivatePreviousApplication(void) {
	NSRunningApplication* previous = launcherPreviousApplication;
	launcherPreviousApplication = nil;
	if (previous != nil && !previous.terminated) {
		NSRunningApplication* current = [NSRunningApplication currentApplication];
		if (current == nil || previous.processIdentifier != current.processIdentifier) {
			[previous activateWithOptions:NSApplicationActivateIgnoringOtherApps];
		}
	}
	[previous release];
}

static NSScreen* screenContainingMouse(void) {
	NSPoint mouse = [NSEvent mouseLocation];
	for (NSScreen* screen in [NSScreen screens]) {
		if (NSPointInRect(mouse, screen.frame)) {
			return screen;
		}
	}
	return nil;
}

static bool prepareLauncherWindowNative(const char* title) {
	// This runs before Wails Show, whose makeKeyAndOrderFront may activate
	// CmDex. Capture both pieces of external state while the invoking app and
	// Space are still frontmost.
	NSWindow* window = launcherWindowForTitle(title);
	if (window == nil) {
		return false;
	}
	capturePreviousFrontmostApplication();
	NSScreen* screen = screenContainingMouse();
	if (screen == nil) {
		screen = screenForWindowOrMain(window);
	}
	[launcherTargetScreen release];
	launcherTargetScreen = screen == nil ? nil : [screen retain];
	return launcherTargetScreen != nil;
}

static bool presentLauncherWindowNative(const char* title, int width, int height, double topFraction, bool useMouseScreen) {
	// Capture the target before activation/order operations. The window screen
	// is intentionally preferred during resize so a pointer moving between
	// displays cannot make an expanded launcher jump to another screen.
	NSWindow* window = launcherWindowForTitle(title);
	NSScreen* screen = nil;
	if (useMouseScreen) {
		screen = launcherTargetScreen;
		if (screen == nil) {
			screen = screenContainingMouse();
		}
	}
	if (screen == nil) {
		screen = screenForWindowOrMain(window);
	}
	if (window == nil || screen == nil) {
		return false;
	}

	// Capture the app that owns the current Space before any operation that may
	// activate CmDex. Hide uses this retained reference to return focus without
	// showing or focusing CmDex's main window.
	if (useMouseScreen) {
		capturePreviousFrontmostApplication();
	}

	// This function is called from application.InvokeAsync, Wails' AppKit
	// main-thread dispatcher. Keep all AppKit operations here on that thread.
	// CanJoinAllSpaces + FullScreenAuxiliary is required for an auxiliary window
	// to appear over an unrelated app's fullscreen Space. MoveToActiveSpace is
	// deliberately omitted: it can move the launcher into Cmdex's Space when
	// Wails makes the window key.
	window.collectionBehavior = NSWindowCollectionBehaviorCanJoinAllSpaces |
		NSWindowCollectionBehaviorFullScreenAuxiliary |
		NSWindowCollectionBehaviorIgnoresCycle |
		NSWindowCollectionBehaviorTransient;
	window.level = NSFloatingWindowLevel;

	NSRect workArea = screen.visibleFrame;
	CGFloat x = NSMidX(workArea) - ((CGFloat)width / 2.0);
	CGFloat topOffset = workArea.size.height * (CGFloat)topFraction;
	CGFloat y = NSMaxY(workArea) - topOffset - (CGFloat)height;
	if (y < NSMinY(workArea)) {
		y = NSMinY(workArea);
	}
	NSRect intendedFrame = NSMakeRect(x, y, width, height);
	[window setFrame:intendedFrame display:YES animate:NO];

	// Make the native operations final. In particular, do not call Wails Focus,
	// SetPosition, or orderOut after this point: those calls can reassign the
	// window to Cmdex's old Space or restore stale dimensions.
	[window orderFrontRegardless];
	[window makeKeyAndOrderFront:nil];
	[window makeKeyWindow];
	[[NSApplication sharedApplication] activateIgnoringOtherApps:YES];

	// AppKit can adjust a window while ordering it (for example for a minimum
	// size). Re-apply and validate the intended frame so the presenter never
	// reports success for a clipped or stale-size launcher.
	NSRect committedFrame = window.frame;
	if (fabs(committedFrame.size.width - width) > 1.0 || fabs(committedFrame.size.height - height) > 1.0 ||
		fabs(committedFrame.origin.x - x) > 1.0 || fabs(committedFrame.origin.y - y) > 1.0) {
		[window setFrame:intendedFrame display:YES animate:NO];
		committedFrame = window.frame;
	}
	return fabs(committedFrame.size.width - width) <= 1.0 &&
		fabs(committedFrame.size.height - height) <= 1.0 &&
		fabs(committedFrame.origin.x - x) <= 1.0 &&
		fabs(committedFrame.origin.y - y) <= 1.0;
}

static bool hideLauncherWindowNative(const char* title) {
	NSWindow* window = launcherWindowForTitle(title);
	if (window != nil) {
		[window orderOut:nil];
	}
	// Restore only the app that was frontmost before the launcher presentation.
	// A terminated app is ignored, and the retained reference is always cleared.
	reactivatePreviousApplication();
	[launcherTargetScreen release];
	launcherTargetScreen = nil;
	return window != nil;
}
*/
import "C"
import "unsafe"

// presentLauncherWindowNative activates and raises the launcher on the
// AppKit screen under the pointer. The title is safe here because it is the
// unique constant used when creating the one launcher window; Wails beta.12
// does not expose the native NSWindow pointer publicly.
func prepareLauncherWindowNative(title string) bool {
	nativeTitle := C.CString(title)
	defer C.free(unsafe.Pointer(nativeTitle))
	return bool(C.prepareLauncherWindowNative(nativeTitle))
}

func presentLauncherWindowNative(title string, width int, height int, topFraction float64, useMouseScreen bool) bool {
	nativeTitle := C.CString(title)
	defer C.free(unsafe.Pointer(nativeTitle))
	return bool(C.presentLauncherWindowNative(nativeTitle, C.int(width), C.int(height), C.double(topFraction), C.bool(useMouseScreen)))
}

func hideLauncherWindowNative(title string) bool {
	nativeTitle := C.CString(title)
	defer C.free(unsafe.Pointer(nativeTitle))
	return bool(C.hideLauncherWindowNative(nativeTitle))
}

// presentLauncherWindow keeps the show/placement/focus sequence injectable so
// the global-shortcut path can be tested without starting a desktop app.
func presentLauncherWindow(prepare func(), show func(), nativePlace func() bool, fallbackPlace func(), focus func()) {
	prepare()
	show()
	if nativePlace() {
		return
	}
	fallbackPlace()
	focus()
}
