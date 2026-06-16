---
phase: 25-polish-integration
plan: 01
subsystem: testing
tags: [pty, refactor, mock, build-tags, darwin, windows, conpty, creack]

# Dependency graph
requires:
  - phase: 21-backend-session-foundation
    provides: TerminalService with CreateSession/CloseSession/Resize/Stop and ptyStart/ptyResize/killProcessGroup package-level helpers in terminal_unix.go + terminal_windows.go
provides:
  - ptyBackend interface (Start/Resize/Kill) and ptyHandle interface (io.ReadWriteCloser) in pty_backend.go
  - Build-tagged creack/pty implementation in pty_backend_unix.go (!windows)
  - Build-tagged conpty stub in pty_backend_windows.go (windows) — Start/Resize return "not yet implemented", Kill uses taskkill
  - darwin-only in-memory mockPtyBackend + mockPtyHandle in pty_backend_mock.go
  - darwin-only test helper newTestTerminalServiceWithMock in pty_backend_mock_test.go
  - terminal_service.go now routes Start/Resize/Kill through s.ptyBackend (no package-level helper calls)
  - sessionState.ptmx and readLoop/monitorExit parameter types are ptyHandle (not *os.File)
affects:
  - Future orchestration tests can use newTestTerminalServiceWithMock for hermetic testing on darwin
  - Future Windows conpty work only needs to implement the conptyBackend methods (Start/Resize) — the seam is in place

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Build-tagged Go interface seam: !windows and windows files implement the same interface, platform-portable code in non-tagged file"
    - "Adapter pattern (osFileHandle wraps *os.File) to satisfy io.ReadWriteCloser from a concrete type"
    - "darwin-only behavior mock in //go:build darwin — excluded from production darwin binary? Actually included on darwin; excluded from Windows build"
    - "Method delegation: backend struct methods are thin wrappers around package-level helpers, preserving existing semantics"

key-files:
  created:
    - pty_backend.go
    - pty_backend_unix.go
    - pty_backend_windows.go
    - pty_backend_mock.go
    - pty_backend_mock_test.go
  modified:
    - terminal_service.go
    - terminal_service_test.go
  removed:
    - terminal_unix.go
    - terminal_windows.go

key-decisions:
  - "ptyBackend and ptyHandle are lowercase (package-private) interfaces — no need to export them; the seam is internal to the main package"
  - "ptyHandle embeds io.ReadWriteCloser (not an interface with a third method) so *os.File satisfies it implicitly — source-compatible with the existing test file's direct ptyStart calls that return *os.File"
  - "Package-level ptyStart/ptyResize/killProcessGroup preserved as functions in pty_backend_unix.go (and pty_backend_windows.go for the kill helper). The creackPtyBackend methods delegate to them — this is the 'kept as package-level functions called by the implementation' variant from D-12's agent-discretion question"
  - "conptyBackend.Start and Resize return the same 'not yet implemented' errors as the old terminal_windows.go stubs, preserving existing behavior for cross-compile verification"
  - "mockPtyBackend.Start uses exec.Command('sleep', '0.05') and cmd.Start() so cmd.Wait() in monitorExit returns promptly. This lets orchestration tests exercise the full Start → readLoop → monitorExit path without a real shell or process group"
  - "newTestTerminalService helper updated to inject the real (darwin) backend via newPtyBackend(). This was required to prevent nil-interface panics — all 12 existing multi-session tests use this helper"

patterns-established:
  - "Interface seam for OS-specific code: define an interface in a non-build-tagged file; implement it in two build-tagged files (one per OS). The interface file is the public/internal seam; the implementations are the only place build tags live."
  - "Mock test helpers live in their own _test.go files with the same build tag as the mock implementation. The test helper does not need its own build tag if it lives in a _test.go file under the same package — but for clarity and to ensure the mock is excluded from non-darwin test runs, the helper file is also tagged darwin."

requirements-completed: []

# Metrics
duration: 9min
completed: 2026-06-16
---

# Phase 25 Plan 01: ptyBackend Interface + Mock Summary

**Build-tagged ptyBackend interface refactor — `pty_backend.go` defines the seam, `pty_backend_unix.go` implements creack/pty, `pty_backend_windows.go` stubs conpty, and `pty_backend_mock.go` provides a darwin-only in-memory mock for orchestration tests**

## Performance

- **Duration:** 9 min
- **Started:** 2026-06-16T06:55:16Z
- **Completed:** 2026-06-16T07:04:32Z
- **Tasks:** 2
- **Files modified:** 7 (5 created, 2 modified, 2 deleted = 9 net file operations, 7 files in the final state)

## Accomplishments

- `ptyBackend` (Start/Resize/Kill) and `ptyHandle` (io.ReadWriteCloser) interfaces defined in `pty_backend.go` — the only seam between `terminal_service.go` and the OS PTY layer.
- `creack/pty` implementation moved verbatim into `pty_backend_unix.go` (//go:build !windows) with a thin `osFileHandle` adapter wrapping `*os.File`.
- Conpty stub moved into `pty_backend_windows.go` (//go:build windows) — Start/Resize return the existing "not yet implemented" error, Kill uses the moved taskkill helper.
- `terminal_service.go` rewired: `TerminalService.ptyBackend` field initialized in `ServiceStartup` via `newPtyBackend()`; `startSessionLocked` calls `s.ptyBackend.Start`, `Resize` calls `s.ptyBackend.Resize`, `CloseSession`/`Stop`/`startSessionLocked` all call `s.ptyBackend.Kill`. No remaining package-level `ptyStart`/`ptyResize`/`killProcessGroup` references in `terminal_service.go`.
- `sessionState.ptmx`, `readLoop` parameter, and `monitorExit` parameter are now of type `ptyHandle` — `*os.File` still satisfies the interface, so the existing test file's direct `ptyStart` calls and `monitorExit(ss, cmd, ptmx, stopCh)` call sites compile unchanged.
- `terminal_unix.go` and `terminal_windows.go` deleted; their content lives in the new build-tagged files.
- `newTestTerminalService` test helper updated to inject the real backend (`s := &TerminalService{ptyBackend: newPtyBackend()}`) so the 12 existing multi-session tests do not panic on a nil backend interface.
- Darwin-only `mockPtyBackend` + `mockPtyHandle` added in `pty_backend_mock.go` (//go:build darwin). Mock provides PTY spawn (returns `exec.Command("sleep", "0.05")` + `cmd.Start()`), write, read, resize without a real long-lived process. `newTestTerminalServiceWithMock` helper added for future orchestration tests.
- All 12 existing `TestTerminalService_*` multi-session tests pass with no test-body changes. `go build ./...` and `go vet ./...` exit 0 on darwin. `GOOS=windows go build ./...` and `GOOS=windows go vet ./...` also exit 0 — Windows cross-compile is clean (the mock is excluded by the darwin build tag).

## Task Commits

Each task was committed atomically:

1. **Task 1: Define ptyBackend interface + build-tagged implementations + wire TerminalService through it** - `e32f174` (refactor)
2. **Task 2: Add darwin-side behavior mock + test helper for orchestration tests** - `f681a89` (test)

## Files Created/Modified

- `pty_backend.go` (NEW) — `ptyBackend` and `ptyHandle` interface definitions
- `pty_backend_unix.go` (NEW) — `//go:build !windows`. `creackPtyBackend` struct with `Start`/`Resize`/`Kill` methods; `osFileHandle` adapter (Read/Write/Close forwarding to `*os.File`); `newPtyBackend()` returns `creackPtyBackend{}`; package-level `ptyStart`/`ptyResize`/`killProcessGroup` moved verbatim from `terminal_unix.go`
- `pty_backend_windows.go` (NEW) — `//go:build windows`. `conptyBackend` struct with stub `Start`/`Resize` (return "not yet implemented") and `Kill` (calls moved taskkill helper); `newPtyBackend()` returns `conptyBackend{}`; package-level `ptyStart`/`ptyResize`/`killProcessGroup` moved verbatim from `terminal_windows.go`
- `pty_backend_mock.go` (NEW) — `//go:build darwin`. `mockPtyBackend` struct with `Start` (returns `*mockPtyHandle` + real `*exec.Cmd` running `sleep 0.05`), `Resize` (type-asserts to `*mockPtyHandle` and updates cols/rows), `Kill` (calls `cmd.Process.Kill()`); `mockPtyHandle` struct with mutex-protected `Read` (returns `io.EOF` when output empty, `os.ErrClosed` when closed), `Write` (returns `os.ErrClosed` when closed), `Close`
- `pty_backend_mock_test.go` (NEW) — `//go:build darwin`. `newTestTerminalServiceWithMock` test helper that injects `mockPtyBackend{}`
- `terminal_service.go` (MODIFIED) — `TerminalService.ptyBackend` field added; `ServiceStartup` initializes it; `sessionState.ptmx` is now `ptyHandle`; `readLoop` and `monitorExit` parameters are now `ptyHandle`; `startSessionLocked`/`Resize`/`CloseSession`/`Stop` all route through `s.ptyBackend.Start`/`Resize`/`Kill`
- `terminal_service_test.go` (MODIFIED) — `newTestTerminalService` helper initializes `s.ptyBackend = newPtyBackend()` to prevent nil-interface panics
- `terminal_unix.go` (DELETED) — content moved to `pty_backend_unix.go`
- `terminal_windows.go` (DELETED) — content moved to `pty_backend_windows.go`

## Decisions Made

- Kept the package-level `ptyStart`/`ptyResize`/`killProcessGroup` functions in the new build-tagged files (they are still called by the backend methods). This was one of the two options in the CONTEXT's agent-discretion section ("extract into interface" vs. "keep as package-level functions called by the implementation"). Keeping them as package-level functions was chosen because: (a) the test file already references `ptyStart` directly, (b) the moved bodies were preserved verbatim (less diff to review), and (c) it minimizes the risk of semantic drift in the kill/PTY-resize paths.
- `osFileHandle` is a value-type struct (not a pointer) with a `f *os.File` field. Its methods are on the value receiver, so both `osFileHandle` and `*osFileHandle` satisfy `io.ReadWriteCloser`. The creack backend's `Resize` does a type assertion `handle.(osFileHandle)` to extract the file for `ptyResize`.
- The `pty_backend.go` file has NO build tag — only the two implementation files are build-tagged. This means the interfaces compile on every platform, and the OS-specific implementations are selected by Go's normal build-tag resolution. The test file (not build-tagged) can reference `newPtyBackend()` and the interfaces from any platform; the OS implementation is selected per `GOOS`.
- Mock's `Start` returns a real `*exec.Cmd` (not a synthetic one) so that `monitorExit`'s `cmd.Wait()` returns promptly when the sleep finishes. This is enough to exercise the Start → readLoop → monitorExit orchestration path without a long-lived shell.
- Did NOT change the test bodies in `terminal_service_test.go` — only the `newTestTerminalService` helper. This keeps the diff minimal and proves the refactor preserves existing test behavior.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Also replaced killProcessGroup in startSessionLocked**

- **Found during:** Task 1 verification (`grep` for `killProcessGroup` after the planned updates to `CloseSession` and `Stop`)
- **Issue:** The plan's Step 4 explicitly listed `CloseSession` (line 249) and `Stop` (line 672) as the two call sites to update to `s.ptyBackend.Kill(oldCmd)`, but the must-have truth states "terminal_service.go calls s.ptyBackend.Start / Resize / Kill instead of package-level ptyStart / ptyResize / killProcessGroup". `startSessionLocked` at line 345 also had a `killProcessGroup(oldCmd)` call. To match the stated outcome, this third call site was also updated.
- **Fix:** Changed `killProcessGroup(oldCmd)` → `s.ptyBackend.Kill(oldCmd)` in `startSessionLocked` (line 345 of original `terminal_service.go`).
- **Files modified:** terminal_service.go
- **Verification:** `grep killProcessGroup terminal_service.go` returns 0 matches after the change. All 12 multi-session tests still pass.
- **Committed in:** e32f174 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 missing critical, 0 bug, 0 blocking, 0 architectural)
**Impact on plan:** The deviation was necessary to align the implementation with the plan's stated must-have truth. The Step 4 action list missed one of three call sites; the must-have truth was the binding requirement. No scope creep — this is a strict subset of the goal.

## Issues Encountered

None. The refactor compiled and passed all tests on first attempt after the third `killProcessGroup` call site was identified and updated.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The ptyBackend interface is in place. Future plans (25-02 onwards) can use `newTestTerminalServiceWithMock` for hermetic orchestration tests.
- The Windows conpty stub still returns "not yet implemented" — Windows runtime verification remains a documented gap per D-13. Cross-compile verification (GOOS=windows) passes, which catches API mismatches at compile time.
- The mock is darwin-only by design (D-12) — Windows-side orchestration tests would need a Windows-specific mock or a real conpty implementation. Out of scope for this phase.

## Self-Check: PASSED

- All 5 new files present on disk
- Both old files (terminal_unix.go, terminal_windows.go) absent
- Both task commits present in git log: e32f174 (Task 1) and f681a89 (Task 2)
- All 18 Task 1 acceptance criteria verified by source grep
- All 8 Task 2 acceptance criteria verified by source grep + build
- `go build ./...` exits 0 on darwin
- `go vet ./...` reports no issues on darwin
- `GOOS=windows go build ./...` exits 0
- `GOOS=windows go vet ./...` reports no issues
- All 12 `TestTerminalService_*` multi-session tests pass (no regression)
- All 7 `TestTerminal*` non-Service tests pass (no regression)
- Full test suite (`go test ./...`) passes

---

*Phase: 25-polish-integration*
*Completed: 2026-06-16*
