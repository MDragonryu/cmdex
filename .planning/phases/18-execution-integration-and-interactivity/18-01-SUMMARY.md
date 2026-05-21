---
phase: 18-execution-integration-and-interactivity
plan: 01
subsystem: execution
tags: [pty, terminal, wails, react, typescript, go]

# Dependency graph
requires:
  - phase: 16-pty-backend-foundation
    provides: TerminalService with Write/Start/Stop methods, PTY spawning, output streaming
  - phase: 17-xterm-js-terminal-and-split-pane-layout
    provides: xterm.js terminal component, pty-output event consumption, split-pane layout
provides:
  - PTY-write execution path (replaces temp-script subprocess)
  - cd-sandwich working directory injection at Run time
  - Fire-and-forget Run command (no ExecutionRecord persistence)
  - Package-level terminalSvc for cross-service PTY access
affects: [future-interrupt-and-keystroke]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "cd sandwich: cd <workingDir> && <command> && cd ~"
    - "Package-level service var for cross-service access (terminalSvc)"
    - "Fire-and-forget execution: write to PTY, don't persist ExecutionRecord"

key-files:
  created:
    - execution_service_test.go (10 tests for PTY Write behavior)
  modified:
    - execution_service.go (RunCommand rewrite, hasExplicitWorkingDir, cd sandwich)
    - app.go (terminalSvc package-level var)
    - terminal_service.go (terminalSvc = s in ServiceStartup)
    - frontend/src/App.tsx (runCommandDirect simplification)

key-decisions:
  - "Used existing shellQuoteDir from executor.go instead of duplicating (would cause redeclaration error)"
  - "Added hasExplicitWorkingDir helper to control cd sandwich — only when per-command or global default is explicitly set, not when falling back to home"
  - "Kept selectedRecord/historyPaneOpen state for HistoryPane; only removed their use from Run path"

patterns-established:
  - "PTY Write execution: ReplaceTemplateVars → cd sandwich → terminalSvc.Write → fire-and-forget"
  - "No db.AddExecution from RunCommand — history pane is independent of terminal execution"

requirements-completed:
  - EXEC-01
  - EXEC-04

# Metrics
duration: 20m
completed: 2026-05-21
---

# Phase 18 Plan 01: PTY Write execution + fire-and-forget Run path Summary

**Replaced temp-script subprocess execution with direct PTY Write via cd-sandwich command chaining, establishing fire-and-forget Run path per D-01 design constraints.**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-05-21T03:50:00Z
- **Completed:** 2026-05-21T04:09:58Z
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments

- RunCommand now writes resolved command text directly to PTY terminal via `terminalSvc.Write(cmdLine)` instead of spawning a temp-script subprocess via `executor.ExecuteScript`
- Working directory injected at Run time via shell command chaining (`cd <dir> && <cmd> && cd ~`) using existing `shellQuoteDir` for path escaping
- Package-level `terminalSvc *TerminalService` var declared in `app.go`, assigned during `TerminalService.ServiceStartup`
- `runCommandDirect` simplified to fire-and-forget — no `setSelectedRecord`, `setHistoryPaneOpen`, `loadHistory`, `exitCode` logic, or `errRecord` construction
- 10 new Go tests covering: PTY write with/without working dir, db error handling, write error handling, ServiceStartup assignment, shellQuoteDir escaping, and no history persistence

## Task Commits

Each task was committed atomically:

1. **Task 1 (TDD - RED): Add failing tests for PTY Write + terminalSvc** — `9ebe27a` (test)
   - 10 tests covering PTY Write behavior, error paths, and ServiceStartup assignment
   - Added `terminalSvc *TerminalService` to `app.go` package-level vars

2. **Task 1 (TDD - GREEN): Implement PTY Write + terminalSvc** — `20fd36c` (feat)
   - Rewrote `RunCommand` in `execution_service.go` to use `terminalSvc.Write` with cd sandwich
   - Added `hasExplicitWorkingDir` helper (cd sandwich only when dir explicitly set)
   - Assigned `terminalSvc = s` in `TerminalService.ServiceStartup`
   - Removed `db.AddExecution` call — fire-and-forget per D-01

3. **Task 2: Simplify runCommandDirect to fire-and-forget** — `28a9c39` (fix)
   - Removed `setSelectedRecord`, `setHistoryPaneOpen`, `loadHistory`, `exitCode` logic, `errRecord` construction from Run path
   - Kept executing state tracking and active-tab toast gating
   - Simplified dependency array from `[t, loadHistory]` to `[t]`
   - Preserved `selectedRecord`/`historyPaneOpen` for HistoryPane

## Files Created/Modified

- `execution_service_test.go` — 10 new tests for PTY Write execution behavior
- `execution_service.go` — Rewritten `RunCommand` method, new `hasExplicitWorkingDir` helper
- `app.go` — Added `terminalSvc *TerminalService` to package-level vars
- `terminal_service.go` — Added `terminalSvc = s` in `ServiceStartup`
- `frontend/src/App.tsx` — Simplified `runCommandDirect` to fire-and-forget

## Decisions Made

- **Reused existing `shellQuoteDir` from `executor.go`** instead of duplicating — creating a duplicate would cause a Go redeclaration error (same `package main`). The acceptance criteria referencing a `shellQuoteDir` in `execution_service.go` was a plan oversight.
- **Added `hasExplicitWorkingDir` helper** to determine when cd sandwich should apply. The plan's `workingDir != ""` condition would always be true because `resolveWorkingDir` has a mandatory 5-step fallback chain (always returns a path). The cd sandwich now only applies when per-command or global default working dir is explicitly set — not when falling back to home.
- **Preserved `selectedRecord`/`historyPaneOpen` state declarations** — used by `HistoryPane` and `handleSelectRecord`, not by the Run path. Only removed their usage from `runCommandDirect`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 — Bug] Avoided duplicate `shellQuoteDir` function**
- **Found during:** Task 1 GREEN (implementation)
- **Issue:** Plan requested adding `shellQuoteDir` to `execution_service.go`, but it already exists in `executor.go` (same `package main`). Duplicating would cause a Go redeclaration compile error.
- **Fix:** Used the existing `shellQuoteDir` from `executor.go` instead. No duplicate added. Did not add unused `"strings"` import.
- **Files modified:** `execution_service.go` (no duplicate added)
- **Committed in:** `20fd36c`

**2. [Rule 1 — Bug] Added `hasExplicitWorkingDir` to control cd sandwich**
- **Found during:** Task 1 GREEN (test failure analysis)
- **Issue:** Plan's condition `workingDir != ""` always evaluates to `true` because `resolveWorkingDir` has a mandatory fallback chain (per-command → global default → home → cwd → tmp). The "no cd sandwich" behavior (Test 2) could never occur.
- **Fix:** Added `hasExplicitWorkingDir(cmd Command) bool` helper that checks only explicit settings (per-command dir or global default), not fallback values. Cd sandwich only applies when an explicit dir exists.
- **Files modified:** `execution_service.go`
- **Committed in:** `20fd36c`

**3. [Rule 3 — Blocking] Fixed test DB/panic issues**
- **Found during:** Task 1 GREEN (test execution)
- **Issue:** (a) Setting `db = nil` caused nil pointer panic in `RunCommand` instead of graceful error return. (b) Shared production DB had pre-existing global default working dir, causing false cd sandwich in "no working dir" test.
- **Fix:** (a) Used nonexistent command ID with real DB to trigger `GetCommand` error. (b) Added settings cleanup to clear global default before test.
- **Files modified:** `execution_service_test.go`
- **Committed in:** `20fd36c`

---

**Total deviations:** 3 auto-fixed (2 Rule 1 bugs, 1 Rule 3 blocking)
**Impact on plan:** All auto-fixes necessary for correctness. No scope creep. Plan shipped as designed — deviations were implementation-level corrections to plan logic.

## Issues Encountered

- `TestTerminalExit` pre-existing test failure (shell exit timing) — unrelated to this plan, not addressed.
- Plan's "no working dir" test expectation conflicted with `resolveWorkingDir`'s guaranteed fallback behavior — resolved via `hasExplicitWorkingDir` helper.

## Threat Flags

None — all security-relevant changes are within the existing `<threat_model>`. The `hasExplicitWorkingDir` helper and cd sandwich logic follow T-18-01 (template var replacement) and T-18-02 (shell quote directory) mitigations.

## Known Stubs

None — all execution paths are fully implemented. The `RunCommand` method is complete with error handling for both `GetCommand` failures and `Write` failures.

## Next Phase Readiness

- PTY Write execution path complete — commands now execute inside the visible xterm.js terminal
- Fire-and-forget Run path ready for keystroke forwarding and Ctrl+C handling (plan 18-02)
- Ready for next plan: 18-02 (keystroke forwarding + interrupt)

---
*Phase: 18-execution-integration-and-interactivity*
*Completed: 2026-05-21*
