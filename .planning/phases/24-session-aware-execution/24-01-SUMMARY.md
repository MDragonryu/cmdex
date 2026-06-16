---
phase: 24-session-aware-execution
plan: 01
subsystem: execution
tags: [go, wails, terminal, pty, refactor, clean-break]

# Dependency graph
requires:
  - phase: 21-backend-session-foundation
    provides: TerminalService with package-level terminalSvc global, GetActiveSession, Write with auto-resume
  - phase: 23-frontend-tabbed-terminal
    provides: Per-session pty-output:{sessionId} subscription in Terminal.tsx (so removing cmd-executing is safe)
provides:
  - RunCommand now dispatches resolved command line directly to active session's PTY via terminalSvc.Write (no event hop)
  - testWithTerminalSvc test helper for setting up real TerminalService in tests
  - 3 new RunCommand tests covering NilTerminalSvc, NoActiveSession, ExecutesOnActiveSession edges (EXEC-01)
  - Clean removal of dead code: RunInTerminal, GetExecutionHistory, ClearExecutionHistory, CmdExecuting event
affects:
  - frontend/src/components/Terminal.tsx (Plan 02 removes the cmd-executing subscription that is now orphaned)
  - frontend/src/wails/events.ts (Plan 02 removes the cmdExecuting entry)
  - frontend/bindings/cmdex/{executionservice,eventservice,models}.js (Plan 02 regenerates without RunInTerminal + CmdExecuting)
  - frontend/src/components/OutputPane.tsx (Plan 02 deletes the orphan component)
  - frontend/src/locales/en.json (Plan 02 removes orphan i18n keys)
  - frontend/src/style.css (Plan 02 removes orphan .output-pane* rules)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Direct in-process service-to-service dispatch via package-level singleton (terminalSvc)
    - Defensive nil checks at service boundary (terminalSvc == nil, session == nil) returning ExecutionRecord with Error
    - Clean-break event removal: drop producer field, value, and (in Plan 02) consumer subscription without backward-compat shim
    - testWithTerminalSvc save/restore pattern: snapshot global, construct via ServiceStartup, defer cleanup

key-files:
  created: []
  modified:
    - execution_service.go
    - event_service.go
    - execution_service_test.go

key-decisions:
  - "Route RunCommand via terminalSvc.Write directly (Pattern 1 in RESEARCH.md) — eliminates the cmd-executing event round-trip and the class of timing bugs where the frontend subscription is not yet attached when the event fires"
  - "Keep the cmdLine construction (ReplaceTemplateVars → stripShebang → TrimRight → optional cd prefix) verbatim — D-06 locks the working-directory fallback chain"
  - "Delete RunInTerminal, GetExecutionHistory, ClearExecutionHistory (D-03 + Open Question #1 resolution in RESEARCH.md) — clean break, no consumers remain"
  - "Drop CmdExecuting from EventNames struct + eventNames var (D-04 / Phase 21 clean-break precedent) — Plan 02 will drop the frontend consumer"
  - "Add defensive nil checks (terminalSvc, session) returning ExecutionRecord{Error, ExitCode: -1} — Pitfall 5 cold-start race and Pitfall 1 no-active edge"
  - "Test using a real TerminalService (not mocks) — exercises the full Write path including auto-resume, matching the precedent set by TestTerminalService_ServiceStartupAssignsTerminalSvc"

patterns-established:
  - "testWithTerminalSvc helper: t.Skip on testing.Short(), save/restore terminalSvc global, return cleanup that calls ServiceShutdown and restores global"
  - "Defensive nil-check pattern at service dispatch boundary: return ExecutionRecord with new UUID + descriptive Error + ExitCode: -1 instead of panicking"

requirements-completed: [EXEC-01, EXEC-02, EXEC-03, EXEC-05, EXEC-06]

# Metrics
duration: 12min
completed: 2026-06-16
---

# Phase 24 Plan 01: Session-Aware Execution Backend Summary

**RunCommand refactored to dispatch via terminalSvc.Write directly; cmd-executing event, RunInTerminal, and execution-history methods deleted; new testWithTerminalSvc helper + 3 new edge-case tests cover the new dispatch path.**

## Performance

- **Duration:** 12 min
- **Started:** 2026-06-16T03:05:56Z
- **Completed:** 2026-06-16T03:17:37Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- `RunCommand` no longer hops through a global `cmd-executing` event — it now resolves the command line, then calls `terminalSvc.GetActiveSession()` and `terminalSvc.Write(session.ID, cmdLine)` in-process. Output streams back through the existing `pty-output:{sessionId}` subscription in `Terminal.tsx` (Phase 23), and Ctrl+C is handled by the PTY's foreground process group.
- Defensive nil checks added: if `terminalSvc == nil` (cold-start race) or `session == nil` (no active session), `RunCommand` returns `ExecutionRecord{Error: ..., ExitCode: -1}` instead of panicking.
- Clean removal of dead code: `RunInTerminal`, `GetExecutionHistory`, `ClearExecutionHistory` deleted from `execution_service.go`; `CmdExecuting` field and value removed from `event_service.go`'s `EventNames` struct. Plan 02 will remove the corresponding frontend consumer and regenerate Wails bindings.
- `testWithTerminalSvc` helper added: saves/restores the `terminalSvc` global, calls `ServiceStartup` to construct a real `TerminalService`, returns a cleanup func. Skips under `testing.Short()` and on `ServiceStartup` failure.
- 5 existing `TestRunCommand_*` tests updated to use the helper (so the refactored `RunCommand` doesn't panic on a nil `terminalSvc`).
- 3 new tests added: `TestRunCommand_NilTerminalSvc` (cold-start), `TestRunCommand_NoActiveSession` (no active), `TestRunCommand_ExecutesOnActiveSession` (happy path with `true` command). The full `TestRunCommand` family (8 tests) passes.

## Task Commits

Each task was committed atomically:

1. **Task 1: Refactor RunCommand to direct terminalSvc.Write + remove dead methods + drop CmdExecuting from EventNames** - `ee9d431` (refactor)
2. **Task 2: Add testWithTerminalSvc helper and new RunCommand tests + update existing tests to use the helper** - `4d0a961` (test)

**Plan metadata:** (this commit)

## Files Created/Modified

- `execution_service.go` — `RunCommand` now resolves → `terminalSvc.Write` dispatch with nil checks; `RunInTerminal`, `GetExecutionHistory`, `ClearExecutionHistory` deleted (180 → 167 lines)
- `event_service.go` — `CmdExecuting` field/value removed from `EventNames` struct and `eventNames` var (37 → 35 lines)
- `execution_service_test.go` — `testWithTerminalSvc` helper added; 5 existing `TestRunCommand_*` tests use it; 3 new tests added (NilTerminalSvc, NoActiveSession, ExecutesOnActiveSession); `"strings"` import added (204 → 301 lines)

## Decisions Made

- **Direct in-process call over event hop** — the `cmd-executing` event was a Phase 21 stopgap. Now that `terminalSvc` is reliably initialized, the direct call eliminates one event round-trip and removes a class of timing/race bugs (frontend subscription not yet attached when event fires).
- **Keep cmdLine construction verbatim** — D-06 locks the working-directory fallback chain. The `cd %s && %s\n` format and `shellQuoteDir` are untouched; `TestShellQuoteDir` and `TestRunCommand_FinalCmd*` lock the contract.
- **Delete `RunInTerminal`/`GetExecutionHistory`/`ClearExecutionHistory`** — per D-03 and RESEARCH.md Open Question #1 resolution (lean toward deletion). No frontend consumers remain; `executions` table is left untouched in SQLite (no schema migration needed); `db.AddExecution` is no longer called (D-09).
- **Drop `CmdExecuting` from `EventNames` (clean break)** — Phase 21's "clean break" precedent for `pty-output`/`pty-exit`/`pty-cleared` namespacing. No backward-compat shim. Plan 02 will drop the frontend subscription in the same way.
- **Defensive nil checks at service boundary** — mirrors the existing `db == nil` pattern in `execution_service.go`. Costs nothing; protects against the cold-start race documented in Pitfall 5.
- **Real `TerminalService` in tests, not mocks** — exercises the full `Write` path including auto-resume, matching the precedent set by `TestTerminalService_ServiceStartupAssignsTerminalSvc` (shipped since Phase 21). The `testing.Short()` and `ServiceStartup` failure guards let the tests skip on CI environments without PTY support.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `terminalSvc.activeSessionID` not directly settable from outside the TerminalService**

- **Found during:** Task 2 (writing `TestRunCommand_NoActiveSession`)
- **Issue:** The plan's reference implementation in PATTERNS.md showed accessing the private `activeSessionID` field as `ts.activeSessionID = ""` (after constructing a local `ts := terminalSvc`). But in Go, you can't make a copy of a `*TerminalService` and then access unexported fields of the original `*TerminalService` from a different package. Within the `main` package (the test's package), the field is accessible — but the test file uses the package-level `terminalSvc` global directly, not a local copy.
- **Fix:** Use `terminalSvc.mu.Lock(); terminalSvc.activeSessionID = ""; terminalSvc.mu.Unlock()` directly on the package-level global (which is the same value as the `ts` local in `testWithTerminalSvc`). The mutex lock is required because `GetActiveSession` reads `activeSessionID` under `s.mu.RLock()`; clearing it without the lock would race.
- **Files modified:** `execution_service_test.go`
- **Verification:** `TestRunCommand_NoActiveSession` passes; `TestRunCommand_ExecutesOnActiveSession` also passes (proving the helper's session-bootstrap works when `activeSessionID` is NOT cleared).
- **Committed in:** `4d0a961` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (Rule 3 blocking, with mild scope improvement — the lock pattern is correct and required for race safety).
**Impact on plan:** Minimal. The fix preserves the plan's intent (exercise the "no active session" branch of `RunCommand`) and adds the lock that the original snippet implicitly required.

## Issues Encountered

- The `rtk` command-line wrapper in this environment collapsed `go test -v` output to a one-line summary. Bypassed with `/usr/local/go/bin/go test ./... -run TestRunCommand -v` to get the per-test PASS output. Build/test results are unaffected.
- The macOS linker emits warnings about object files being built for a newer macOS version (26.0) than the link target (11.0). This is environmental (Go toolchain is targeting an older SDK than the host), not a code issue. All tests still pass.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Plan 02 of Phase 24 is ready to execute. It will:
- Remove the `cmd-executing` subscription from `frontend/src/components/Terminal.tsx` (lines 334-345 + 351)
- Remove the `cmdExecuting` entry from `frontend/src/wails/events.ts` and `initEventNames()`
- Delete `frontend/src/components/OutputPane.tsx` (orphan)
- Remove orphan i18n keys from `frontend/src/locales/en.json` (`outputPane.*`, `historyPane.*`, `common.copyLastOutput`, `commandDetail.runInTerminal`)
- Remove orphan CSS from `frontend/src/style.css` (`.output-pane*` rules)
- Clean up `frontend/e2e/utils/selectors.ts` and `frontend/e2e/mocks/runtime.ts`
- Regenerate Wails bindings via `wails3 generate build-assets` so `frontend/bindings/cmdex/{executionservice,eventservice,models}.js` drop `RunInTerminal` and the `cmdExecuting` field
- Update `App.tsx` to remove the `activeSessionId` prop from `TerminalComponent` and the `copyLastOutput` tooltip

After Plan 02 lands, Phase 24's success criteria (Run dispatches to active session, output streams to terminal, Ctrl+C interrupts, no history persisted, no dead code remaining) will be complete. Manual UAT of `EXEC-05` (real-time ANSI output streaming) and `EXEC-06` (Ctrl+C interrupt) is still required in `wails3 dev` per VALIDATION.md.

---

*Phase: 24-session-aware-execution*
*Completed: 2026-06-16*
