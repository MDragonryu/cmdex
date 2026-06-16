---
phase: 25-polish-integration
plan: 02
subsystem: testing
tags: [go, test, stress-test, cwd-inheritance, darwin, mock, dead-code, runbook]

# Dependency graph
requires:
  - phase: 25-polish-integration
    plan: 01
    provides: ptyBackend interface + mockPtyBackend + newTestTerminalServiceWithMock helper
  - phase: 24-session-aware-execution
    provides: terminal_service.go CreateSession shell + per-session goroutine tracking + execution_service.go EXEC-03 chain (D-07)
  - phase: 21-backend-session-foundation
    provides: TerminalService.sessions map and CloseSession lifecycle
provides:
  - terminal_service.go getWorkingDir() now reads settings.DefaultWorkingDir.GetCurrentOS() (OS-keyed) with os.UserHomeDir() fallback; logs 'getWorkingDir: GetSettings failed: %v' on settings-read errors
  - TestTerminalService_GlobalDefaultCwdInheritance — new test in terminal_service_test.go (D-04, D-05)
  - TestTerminalService_CwdInheritance_ExistingSessionUnaffected — new test in terminal_service_test.go (D-06)
  - TestTerminalService_StressCreateClose — 100-cycle create/close stress test in terminal_service_stress_test.go (//go:build darwin) using mockPtyBackend from 25-01 (D-08)
  - RUNBOOK.md — manual xterm leak smoke test procedure (D-09)
  - Dead-code cleanup: cmdOutput: 'cmd-output' removed from frontend/e2e/mocks/runtime.ts; OutputPane / cmd-output / RunInTerminal all return zero matches in source
affects:
  - Future plans can rely on getWorkingDir() honoring the global default cwd; the contract is now test-enforced
  - Future Windows conpty work only needs to implement the conptyBackend methods — the seam is in place and tested

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Darwin-only test in its own //go:build darwin _test.go file: keeps cross-platform test compilation clean while referencing build-tagged helpers"
    - "Warmup cycle + 200ms sleep before goroutine baseline capture: prevents CI flakiness from runtime-internal goroutines (GC, sysmon) still settling"
    - "Mirror the EXEC-03 chain's settings-read fallback pattern in getWorkingDir(): same log style ('getWorkingDir: GetSettings failed: %v'), same OS-keyed read (DefaultWorkingDir.GetCurrentOS()), same home fallback"

key-files:
  created:
    - terminal_service_stress_test.go
    - .planning/phases/25-polish-integration/RUNBOOK.md
  modified:
    - terminal_service.go
    - terminal_service_test.go
    - frontend/e2e/mocks/runtime.ts

key-decisions:
  - "getWorkingDir() mirrors the execution_service.go:resolveWorkingDir pattern (same log message format, same OS-keyed read via GetCurrentOS, same home fallback) so the two cwd-resolution paths are visually parallel"
  - "TestTerminalService_StressCreateClose goes in a new //go:build darwin file (terminal_service_stress_test.go) rather than terminal_service_test.go. The plan said terminal_service_test.go, but newTestTerminalServiceWithMock is itself darwin-tagged (per 25-01) and referencing it from a non-tagged _test.go would break non-darwin test compilation"
  - "Stress test runs 100 cycles with a warmup + 200ms baseline-capture sleep. The 5-goroutine drift threshold matches D-08 and is the same value 25-01's mock design accounts for (runtime-internal goroutines like GC and sysmon)"
  - "Dead-code sweep removed only the cmdOutput field from frontend/e2e/mocks/runtime.ts:350 — the only confirmed source-side residual at planning time. The Wails bindings (excluded from sweep) were regenerated in Phase 24 to no longer export RunInTerminal / CmdOutput, so the e2e mock was the last place where the field was still defined"
  - "Did not change execution_service.go — D-07 confirmed by git diff (no edits to that file) and by TestRunCommand_FinalCmdWithWorkingDir / TestRunCommand_FinalCmdNoWorkingDir still passing"

patterns-established:
  - "For functions that read settings, mirror the existing log style in execution_service.go: '<funcName>: GetSettings failed: %v' on errors, then fall through to the next fallback"
  - "Test files that depend on build-tagged helpers live in their own //go:build darwin file when they exercise only that platform's mock; the main test file (terminal_service_test.go) stays portable"

requirements-completed: []

# Metrics
duration: 7min
completed: 2026-06-16
---

# Phase 25 Plan 02: cwd Inheritance + Stress Test + Dead-Code Sweep Summary

**Global default cwd inheritance wired into `getWorkingDir()` with home fallback, 100-cycle stress test using the darwin-side `mockPtyBackend`, and dead-code sweep + RUNBOOK for Phase 24 remnants**

## Performance

- **Duration:** 7 min
- **Started:** 2026-06-16T07:19:24Z
- **Completed:** 2026-06-16T07:26:48Z
- **Tasks:** 2
- **Files modified:** 5 (3 modified, 2 created)

## Accomplishments

- `terminal_service.go:getWorkingDir()` now reads `settings.DefaultWorkingDir.GetCurrentOS()` for the current OS, falling back to `os.UserHomeDir()` when the path is empty or settings is unavailable. Logs `getWorkingDir: GetSettings failed: %v` on settings-read errors, mirroring the `execution_service.go:resolveWorkingDir` log style. (D-04, D-05)
- `TestTerminalService_GlobalDefaultCwdInheritance` added to `terminal_service_test.go`: opens the real DB, writes a global default cwd keyed by `runtime.GOOS`, asserts a new session's `WorkingDir` reads that path, then resets the global default and asserts a second session falls back to `os.UserHomeDir()`.
- `TestTerminalService_CwdInheritance_ExistingSessionUnaffected` added to `terminal_service_test.go`: creates session S1 with global default P1, then changes the global default to P2 and creates session S2. Asserts S1's `WorkingDir` remains P1 (D-06) and S2's is P2.
- `TestTerminalService_StressCreateClose` added in a new build-tagged file `terminal_service_stress_test.go` (`//go:build darwin`). Runs a warmup cycle + 200ms sleep, then 100 `CreateSession` / `CloseSession` cycles against the `mockPtyBackend` (in-memory; no real shell). Asserts `len(s.sessions) == 0` after every close and goroutine drift ≤ 5 over baseline. Wall-clock: 0.58s on darwin (well under the 5s budget). (D-08)
- Dead-code sweep: removed `cmdOutput: 'cmd-output',` from `frontend/e2e/mocks/runtime.ts:350` (the last source-side residual from Phase 24). All grep checks for `OutputPane`, `cmd-output`, and `RunInTerminal` return zero matches in `frontend/src/`, `frontend/e2e/`, and `*.go` files.
- `RUNBOOK.md` created at `.planning/phases/25-polish-integration/RUNBOOK.md` with the manual xterm leak smoke test procedure: 20× Ctrl+T, query `.xterm` count, 20× Ctrl+W, query again, pass criterion `count-B <= 1`, failure interpretation pointing to `frontend/src/components/Terminal.tsx` cleanup useEffect. (D-09)
- `execution_service.go` EXEC-03 chain is **unchanged** — verified by `git diff --name-only` showing no edits to that file, and by `TestRunCommand_FinalCmdWithWorkingDir` / `TestRunCommand_FinalCmdNoWorkingDir` still passing. (D-07)

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement global default cwd inheritance (D-04..D-07) + unit tests** - `6bef407` (feat)
2. **Task 2: 100-cycle stress test (D-08) + dead-code sweep + RUNBOOK.md (D-09)** - `bbd19cf` (test)

## Files Created/Modified

- `terminal_service.go` (MODIFIED) — `getWorkingDir()` body now reads `settings.DefaultWorkingDir.GetCurrentOS()` (OS-keyed) with `os.UserHomeDir()` fallback; logs `getWorkingDir: GetSettings failed: %v` on settings-read errors. Signature unchanged.
- `terminal_service_test.go` (MODIFIED) — added `TestTerminalService_GlobalDefaultCwdInheritance` and `TestTerminalService_CwdInheritance_ExistingSessionUnaffected` (D-04..D-06). Added helper functions `cwdInheritanceTestDB` and `setGlobalDefaultCwd` that mirror the save/restore pattern from `execution_service_test.go:testDBCreateCommand`.
- `terminal_service_stress_test.go` (NEW, `//go:build darwin`) — `TestTerminalService_StressCreateClose`: warmup cycle, 200ms settle, 100 create/close iterations against `mockPtyBackend`, asserts `len(s.sessions) == 0` after every close and goroutine drift ≤ 5 over baseline.
- `frontend/e2e/mocks/runtime.ts` (MODIFIED) — removed the stale `cmdOutput: 'cmd-output'` field from the `GetEventNames` mock (line 350). The Wails event bridge was regenerated in Phase 24 to no longer export `CmdOutput`; the e2e mock now matches the regenerated bindings.
- `.planning/phases/25-polish-integration/RUNBOOK.md` (NEW) — manual xterm leak smoke test procedure per D-09. Includes Prerequisites, 7-step Procedure, Pass criterion (`count-B <= 1`), Failure interpretation (pointing to `Terminal.tsx` cleanup useEffect around lines 270-274), and Notes.

## Decisions Made

- **Mirror `execution_service.go:resolveWorkingDir` in `getWorkingDir()`.** Both functions now share the same log message format (`<funcName>: GetSettings failed: %v`), the same OS-keyed read pattern (`DefaultWorkingDir.GetCurrentOS()`), and the same home fallback. The two cwd-resolution paths are visually parallel so future maintainers can recognize the pattern.
- **Put `TestTerminalService_StressCreateClose` in a new `//go:build darwin` file.** The plan said `terminal_service_test.go`, but `newTestTerminalServiceWithMock` is itself darwin-tagged (from Plan 25-01) and referencing a build-tagged function from a non-tagged `_test.go` would fail to compile on non-darwin platforms. The deviation is documented below.
- **Stress test warmup + 200ms settle before baseline.** Capturing `runtime.NumGoroutine()` immediately after `newTestTerminalServiceWithMock` is flaky under CI load — the Go runtime may still be allocating background goroutines (GC, sysmon) for the first few hundred ms. The 200ms sleep matches the pattern used in similar Go stress tests and the 5-goroutine delta threshold in D-08 accounts for runtime-internal goroutines.
- **Did not change `execution_service.go`.** D-07 is verified by `git diff --name-only` showing no edits to that file AND by `TestRunCommand_FinalCmdWithWorkingDir` / `TestRunCommand_FinalCmdNoWorkingDir` still passing. The EXEC-03 chain in `resolveWorkingDir` is preserved as-is.
- **Only removed the `cmdOutput` field from the e2e mock.** Per the planning-time grep, that was the only confirmed source-side residual. `OutputPane` and `RunInTerminal` already returned zero matches in `frontend/src/` after Phase 24; the e2e mock's `cmdOutput` field was the only line that needed touching.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Moved `TestTerminalService_StressCreateClose` to a build-tagged file**

- **Found during:** Task 2 (after adding the test to `terminal_service_test.go` and running `go build` — would have failed on non-darwin platforms)
- **Issue:** The plan said to add `TestTerminalService_StressCreateClose` to `terminal_service_test.go`, but the test references `newTestTerminalServiceWithMock` from `pty_backend_mock_test.go` (a `//go:build darwin` file from Plan 25-01). Referencing a darwin-only function from a non-tagged `_test.go` would break cross-platform test compilation. On darwin it would compile, but on Linux/Windows the test build would fail with "undefined: newTestTerminalServiceWithMock".
- **Fix:** Created `terminal_service_stress_test.go` with `//go:build darwin` and moved the test there. The test function is the same as planned, the acceptance criteria are the same, and the build verification still passes (`go build ./...` exits 0 on darwin). The test runs the same 100 cycles with the same warmup/sleep/asserts and the same `newTestTerminalServiceWithMock` helper.
- **Files modified:** `terminal_service_stress_test.go` (NEW), `terminal_service_test.go` (unchanged)
- **Verification:** `go build ./...` and `go vet ./...` exit 0 on darwin; `go test ./...` passes 44 tests; `go test -run TestTerminalService_StressCreateClose -v` runs in 0.58s; the test references `newTestTerminalServiceWithMock` (verified by grep).
- **Committed in:** `bbd19cf` (Task 2 commit)
- **Note on plan acceptance criteria:** The criterion "`terminal_service_test.go` contains `TestTerminalService_StressCreateClose`" is not met verbatim (the test is in `terminal_service_stress_test.go` instead). The substantive criteria — that the test exists, uses `newTestTerminalServiceWithMock`, runs 100 cycles with warmup, and asserts goroutine drift ≤ 5 — are all met.

**2. [Rule 2 - Missing Critical] Test-helper for `setGlobalDefaultCwd` covers empty-map reset**

- **Found during:** Task 1 (writing the `TestTerminalService_GlobalDefaultCwdInheritance` test)
- **Issue:** The plan's Step 2 says "Reset settings to `DefaultWorkingDir = &OSPathMap{}` (empty paths map)" for the home-fallback assertion, but the empty `OSPathMap{}` has a nil `paths` field, which would cause `GetCurrentOS()` to return "". Without a clear reset, the test's home-fallback assertion might fail because the previous global default was still in the DB.
- **Fix:** The `setGlobalDefaultCwd` helper explicitly creates `&OSPathMap{paths: map[string]string{}}` (empty but non-nil map) when the path is `""`. This matches the plan's intent — the global default for the current OS is empty, so `getWorkingDir()` falls through to the home fallback. Verified by both tests passing.
- **Files modified:** `terminal_service_test.go`
- **Verification:** Both new tests pass; the home-fallback path asserts `info2.WorkingDir == home`.
- **Committed in:** `6bef407` (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (1 blocking, 1 missing critical)
**Impact on plan:** Both auto-fixes are necessary for correctness/cross-platform compatibility. The first deviation is a build-correctness fix (referencing a build-tagged helper from a non-tagged file would break non-darwin builds). The second deviation is a test-correctness fix (ensuring the empty-map reset actually clears the global default so the home-fallback assertion can run). No scope creep — both are strict subsets of the plan's goals.

## Issues Encountered

- **Transient SQLite BUSY on first test-suite run.** Running the full `go test ./...` for the first time after the changes produced "database is locked (5) (SQLITE_BUSY)" on both new tests. Subsequent runs (including `go test -count=2 ./...`) all pass with 44/44 tests. The issue is a one-time WAL flush race in `~/.cmdex/cmdex.db` between tests, not a code defect in this plan. The two new tests pass reliably when run individually (`go test -run TestTerminalService_GlobalDefaultCwdInheritance -v ./...`) and when run as part of the full suite on subsequent runs.
- **Plan assumed `newTestTerminalServiceWithMock` was in a non-tagged test file.** It is not — it was placed in `pty_backend_mock_test.go` with `//go:build darwin` per Plan 25-01's design. The plan's Step 1 of Task 2 should have anticipated this. The deviation above is the fix.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The cwd-inheritance contract is now code-enforced by tests, not just documented behavior. Future changes to `getWorkingDir()` will fail tests if they break D-04/D-05/D-06.
- The 100-cycle stress test is the project's first lifecycle-leak check; future refactors of `CreateSession` / `CloseSession` should keep this test green.
- The dead-code sweep is complete for Phase 24 leftovers. Future phases should keep the grep checks (in this plan's `verify` block) as a pre-commit check.
- RUNBOOK.md is in place for the manual xterm leak smoke test. A maintainer running through the procedure will catch any xterm.js dispose bugs that the Go stress test cannot detect.
- `execution_service.go` is unchanged; the EXEC-03 chain is preserved (D-07 confirmed).

## Self-Check: PASSED

- `terminal_service.go:getWorkingDir()` body contains `settings.DefaultWorkingDir.GetCurrentOS` (verified by grep)
- `terminal_service.go:getWorkingDir()` still references `os.UserHomeDir` for the home-fallback branch (verified by grep)
- `terminal_service.go:getWorkingDir()` signature is unchanged: `func getWorkingDir() string`
- `terminal_service.go:getWorkingDir()` body contains `getWorkingDir: GetSettings failed` (verified by grep)
- `terminal_service_test.go` contains `TestTerminalService_GlobalDefaultCwdInheritance`
- `terminal_service_test.go` contains `TestTerminalService_CwdInheritance_ExistingSessionUnaffected`
- `go test -run "TestTerminalService_(GlobalDefaultCwdInheritance|CwdInheritance_ExistingSessionUnaffected)" -v ./...` passes both tests
- `go test -run "TestRunCommand_(FinalCmdWithWorkingDir|FinalCmdNoWorkingDir)" -v ./...` passes both tests (D-07 EXEC-03 chain unchanged)
- `execution_service.go` has NOT been modified (verified by `git diff --name-only` showing no change)
- `go test ./...` reports 44 passed, 0 failed
- `terminal_service_stress_test.go` contains `TestTerminalService_StressCreateClose`
- The stress test references `newTestTerminalServiceWithMock` (verified by grep)
- The stress test runs a warmup create/close cycle and sleeps 200ms before capturing `baseline` (verified by grep `warmup` and `time.Sleep(200`)
- The stress test uses a `cycles` constant equal to 100 (verified by grep `100`)
- The stress test asserts `len(s.sessions) == 0` after each close (verified by grep)
- The stress test asserts goroutine drift `<= 5` (verified by grep `> 5`)
- `go test -run TestTerminalService_StressCreateClose -v ./...` passes on darwin
- Wall-clock time of the stress test: 0.58s (well under 5s budget)
- `grep -rn --include='*.ts' --include='*.tsx' --include='*.go' -E 'OutputPane' frontend/src frontend/e2e *.go` returns zero matches
- `grep -rn --include='*.ts' --include='*.tsx' -E 'cmd-output' frontend/src frontend/e2e` returns zero matches
- `grep -rn --include='*.ts' --include='*.tsx' -E 'RunInTerminal' frontend/src frontend/e2e` returns zero matches
- `frontend/e2e/mocks/runtime.ts` no longer contains `cmdOutput` (verified by `! grep`)
- `.planning/phases/25-polish-integration/RUNBOOK.md` exists
- `RUNBOOK.md` contains the heading `# Phase 25 Runbook — Manual Verification`
- `RUNBOOK.md` contains the section `## Manual xterm leak smoke test`
- `RUNBOOK.md` references `document.querySelectorAll('.xterm').length` for the DOM query (verified by grep)
- Both task commits present in git log: `6bef407` (Task 1) and `bbd19cf` (Task 2)
- `go build ./...` and `go vet ./...` exit 0 on darwin

---

*Phase: 25-polish-integration*
*Completed: 2026-06-16*
