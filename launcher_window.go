package main

// launcherScreenBounds and launcherFrame are the platform-neutral part of
// launcher placement. Keeping the calculation here makes the AppKit geometry
// policy testable without requiring a running NSWindow or a particular monitor
// layout.
type launcherScreenBounds struct {
	X      int
	Y      int
	Width  int
	Height int
}

type launcherFrame struct {
	X      int
	Y      int
	Width  int
	Height int
}

func centeredLauncherFrame(screen launcherScreenBounds, width, height int, topFraction float64) launcherFrame {
	x := screen.X + (screen.Width-width)/launcherCenterDivisor
	y := screen.Y + screen.Height - int(float64(screen.Height)*topFraction) - height
	y = max(y, screen.Y)
	return launcherFrame{X: x, Y: y, Width: width, Height: height}
}
