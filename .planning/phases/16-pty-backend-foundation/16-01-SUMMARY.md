---
phase: 16-pty-backend-foundation
plan: 01
subsystem: terminal
tags: [pty, creack, shell, wails, service]

requires: []
provides:
  - TerminalService struct with Wails-bound Start/Stop/Write/Resize methods
  - Unix PTY lifecycle via creack/pty (start, resize, process group kill)
  - Windows PTY stubs with taskkill-based process termination
  - Cross-platform shell detection (pwsh → powershell → cmd on Windows, $SHELL on Unix)
  - PtyOutput and PtyExit Wails event name constants

affects:
  - 16-02 (readLoop and monitorExit goroutines)
  - 16-03 (test suite and Windows PTY go-winpty integration)

tech-stack:
  added: [github.com/creack/pty v1.1.24]
  patterns:
    - "Build-tagged platform files (//go:build !windows / windows) with identical function names"
    - "Wails v3 service struct pattern with ServiceStartup/ServiceShutdown lifecycle"
    - "Package-level wailsApp + eventNames access for event emission"

key-files:
  created:
    - terminal_service.go - TerminalService struct, detectShell, Start/Stop/Write/Resize
    - terminal_unix.go - creack/pty integration: ptyStart, ptyResize, killProcessGroup
    - terminal_windows.go - Stub PTY functions, working taskkill process kill
  modified:
    - main.go - TerminalService registered in Services slice
    - event_service.go - PtyOutput and PtyExit event constants
    - go.mod - creack/pty v1.1.24 dependency

key-decisions:
  - "Used build-tagged identical function names (ptyStart, ptyResize, killProcessGroup) instead of runtime.GOOS branches — Go compiler checks both branches even at runtime, so platform-specific function names don't compile"
  - "detectShell() implements full D-01 Windows detection chain (pwsh→powershell→cmd) diverging from D-07's NewExecutor() reuse advice, as NewExecutor only detects cmd on Windows"
  - "ptmx.Close() called in Stop() before process kill to unblock pending PTY reads (Common Pitfall 3)"

requirements-completed: [PTY-01, PTY-02, PTY-04, PTY-05, POL-05]

duration: 15min
completed: 2026-05-19
---

# Phase 16 Plan 01: TerminalService Backend Foundation Summary

**creack/pty-backed TerminalService with Unix PTY lifecycle, Windows stubs, cross-platform shell detection, and Wails v3 service registration**

## Performance

- **Duration:** 15 min
- **Started:** 2026-05-19T04:23:00Z
- **Completed:** 2026-05-19T04:38:00Z
- **Tasks:** 3
- **Files modified:** 6 (3 created, 3 modified)

## Accomplishments
- TerminalService struct registered as Wails v3 service with Start/Stop/Write/Resize methods
- Unix PTY creates interactive shell via creack/pty.StartWithSize with process group kill (SIGHUP→SIGKILL)
- Windows platform file with function stubs and working taskkill-based process termination
- Cross-platform shell detection: pwsh→powershell→cmd on Windows, $SHELL env var on Unix
- PtyOutput and PtyExit event constants defined in event_service.go

## Task Commits

Each task was committed atomically:

1. **Task 1: Package legitimacy check — creack/pty v1.1.24** - checkpoint (no code changes)
2. **Task 2: Add event constants, register service, and install creack/pty** - `feat(16-01): add PTY event constants, register TerminalService, install creack/pty v1.1.24`
3. **Task 3: Create platform files and TerminalService struct with lifecycle** - `feat(16-01): create TerminalService with Unix PTY lifecycle, Windows stubs, shell detection`

## Files Created/Modified
- `terminal_service.go` - TerminalService struct, detectShell(), Start/Stop/Write/Resize methods
- `terminal_unix.go` - creack/pty integration with build tag `//go:build !windows`
- `terminal_windows.go` - Stub platform functions with build tag `//go:build windows`
- `main.go` - TerminalService registered in Services slice after App
- `event_service.go` - PtyOutput ("pty-output") and PtyExit ("pty-exit") constants
- `go.mod` / `go.sum` - github.com/creack/pty v1.1.24 dependency

## Decisions Made
- **Build-tagged identical function names:** Plan specified `ptyStartUnix`/`ptyStartWindows` with runtime.GOOS dispatch, but Go compiler checks both branches. Used identical names (`ptyStart`, `ptyResize`, `killProcessGroup`) with build tags for correct compilation
- **detectShell vs NewExecutor():** D-01 required pwsh→powershell→cmd chain that NewExecutor() doesn't support, so standalone detectShell() was implemented per D-01
- **ptmx.Close() in Stop():** Plan mentioned closing PTY master before process kill; added s.ptmx.Close() in Stop() before killProcessGroup call

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Platform function naming prevented cross-compilation**
- **Found during:** Task 3 (build verification)
- **Issue:** Plan specified `ptyStartUnix`/`ptyStartWindows` called via `if runtime.GOOS == "windows"` branches, but Go compiler checks all branches regardless. On macOS, `ptyStartWindows` is undefined.
- **Fix:** Renamed all platform functions to identical names (`ptyStart`, `ptyResize`, `killProcessGroup`) and removed runtime.GOOS branches. Build tags handle dispatch.
- **Files modified:** terminal_unix.go, terminal_windows.go, terminal_service.go
- **Verification:** `go build ./...` and `go vet ./...` pass on macOS
- **Committed in:** Task 3 commit

---

**Total deviations:** 1 auto-fixed (compilation bug)
**Impact on plan:** Function names changed from plan but functionality is identical. All plan requirements met.

## Issues Encountered
- `go mod tidy` removed creack/pty after Task 2's `go get` since no file imported it yet — re-added in Task 3 when terminal_unix.go was created
- Plan's verification grep patterns expect `ptyStartUnix`/`ptyResizeUnix` names — actual names are `ptyStart`/`ptyResize` due to compilation fix

## Next Phase Readiness
- TerminalService skeleton ready for Plan 16-02 (readLoop and monitorExit goroutines)
- Platform abstraction ready for Plan 16-03 (test suite and Windows go-winpty integration)

---
*Phase: 16-pty-backend-foundation*
*Completed: 2026-05-19*
