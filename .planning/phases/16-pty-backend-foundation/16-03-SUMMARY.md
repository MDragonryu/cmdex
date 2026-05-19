---
phase: 16-pty-backend-foundation
plan: 03
subsystem: terminal
tags: [testing, windows, pty, go-winpty]

requires:
  - phase: 16-01
    provides: TerminalService struct, platform files, creack/pty dependency
  - phase: 16-02
    provides: readLoop goroutine, monitorExit goroutine, emitOutput

provides:
  - 7 test functions covering all Phase 16 requirements (PTY-01 through PTY-06, POL-05)
  - Integration tests gated behind testing.Short() for CI safety
  - Windows PTY refactoring TODO documenting go-winpty deprecation and conpty migration path

affects:
  - Phase 17 (frontend terminal integration relies on tested backend)

tech-stack:
  added: []
  patterns:
    - "Test helpers with t.Helper() and t.Cleanup() for automatic resource cleanup"
    - "testing.Short() gate for integration tests that spawn real processes"
    - "Package main test convention matching db_test.go"

key-files:
  created:
    - terminal_service_test.go - 7 test functions for PTY lifecycle/I/O/shell detection
  modified:
    - terminal_windows.go - Added TODO block documenting go-winpty deprecation and refactoring path

key-decisions:
  - "go-winpty v1.0.4 rejected — package is deprecated (pkg.go.dev banner recommends conpty). Windows PTY deferred to follow-up with conpty evaluation."
  - "Integration tests use non-interactive commands (sleep 60, exit 0) instead of full shell sessions for deterministic results"
  - "5 of 7 tests gated behind testing.Short() to prevent CI failures from shell-dependent tests"

requirements-completed: [PTY-01, PTY-02, PTY-03, PTY-04, PTY-05, PTY-06, POL-05]

duration: 10min
completed: 2026-05-19
---

# Phase 16 Plan 03: Test Suite & Windows PTY Summary

**7-test Go suite covering PTY lifecycle and I/O; Windows PTY deferred due to go-winpty deprecation**

## Performance

- **Duration:** 10 min
- **Started:** 2026-05-19T04:53:00Z
- **Completed:** 2026-05-19T05:03:00Z
- **Tasks:** 3
- **Files modified:** 2

## Accomplishments
- 7 test functions covering all Phase 16 requirements in terminal_service_test.go
- Integration tests (PTY spawning) gated behind testing.Short() for CI safety
- Test helper functions with t.Helper() and t.Cleanup() following db_test.go conventions
- Windows PTY path documented with clear TODO and conpty migration plan

## Task Commits

1. **Task 1: Package legitimacy check — go-winpty v1.0.4** - rejected (package deprecated)
2. **Task 2: Create terminal_service_test.go with all 7 requirement tests** - `test(16-03): add TerminalService test suite with 7 requirement tests`
3. **Task 3: Replace terminal_windows.go stubs with go-winpty implementation** - `docs(16-03): add Windows PTY refactoring TODO — go-winpty deprecated, deferring`

## Files Created/Modified
- `terminal_service_test.go` - 7 test functions: DetectShell, Batching, Start, Write, Resize, Shutdown, Exit
- `terminal_windows.go` - Added refactoring TODO block for Windows PTY support

## Decisions Made
- **go-winpty rejected:** Package deprecated since 2022 (recommends conpty). Windows PTY path deferred to follow-up with `UserExistsError/conpty` evaluation
- **TerminalService refactoring needed:** `ptmx *os.File` must become `io.ReadWriteCloser` for Windows dual-handle PTY support
- **Non-interactive test commands:** Tests use `sleep 60` / `exit 0` wrapped in shell flags for deterministic behavior

## Deviations from Plan

### Checkpoint rejection

**1. [Rule 4 - Architectural] go-winpty v1.0.4 rejected due to deprecation**
- **Found during:** Task 1 (package legitimacy check)
- **Issue:** pkg.go.dev shows DEPRECATED banner: "PTYs natively supported in Windows 10 1809+. Switch to conpty." Last release Jun 2022.
- **Fix:** Rejected by user. terminal_windows.go updated with TODO documenting conpty evaluation and required refactoring (ptmx → io.ReadWriteCloser).
- **Files modified:** terminal_windows.go
- **Verification:** `go build ./...` passes on macOS (terminal_windows.go excluded via build tag)
- **Committed in:** Task 3 commit

---

**Total deviations:** 1 (checkpoint rejection — architectural decision)
**Impact on plan:** Windows PTY support deferred. Unix PTY path unaffected. killProcessGroup (taskkill) already correct for Windows.

## Issues Encountered
- go-winpty deprecated — research identified correct replacement (conpty) but integration requires refactoring ptmx type
- TerminalService.Write uses `s.ptmx.Write()` (concrete type) — must become io.Writer before Windows PTY can work

## Next Phase Readiness
- All 7 Phase 16 requirements have test coverage on macOS
- 3 structural tests pass in short mode (DetectShell, Batching)
- 4 integration tests pass with full PTY (`go test -run TestTerminal -v .`)
- Windows PTY requires: evaluate conpty, refactor ptmx to io.ReadWriteCloser, test on Windows

---
*Phase: 16-pty-backend-foundation*
*Completed: 2026-05-19*
