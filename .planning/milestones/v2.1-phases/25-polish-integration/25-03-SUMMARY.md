---
phase: 25-polish-integration
plan: 03
subsystem: terminal
tags: [conpty, max-sessions, guard, cross-compile, windows, i18n, toaster, mock-backend]

# Dependency graph
requires:
  - phase: 25-polish-integration
    plan: 01
    provides: "ptyBackend interface + darwin-side mockPtyBackend + newTestTerminalServiceWithMock"
  - phase: 25-polish-integration
    plan: 02
    provides: "conpty stub in pty_backend_windows.go + 100-cycle stress test pattern"
provides:
  - "MaxSessions = 10 limit enforced by CreateSession (resource-exhaustion guard)"
  - "Localized toast.maxSessionsReached surfaces max-sessions error to users"
  - "Cross-compile verification GOOS=windows go build ./... passes from darwin (D-13)"
  - "CHECKPOINT.md documents the Windows conpty verification scope, gap, and future work (D-14)"
  - "AGENTS.md one-line pointer to CHECKPOINT.md for future maintainers (D-14)"
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Package-level const + lock-guarded capacity check at top of CRUD method"
    - "Error string match (msg.includes) on the frontend to switch between localized and generic toasts"
    - "Build-tagged darwin _test.go file for tests that reference darwin-only helpers (mirrors 25-02 pattern)"
    - "Phase checkpoint document scoped to one verification concern (cross-compile coverage)"

key-files:
  created:
    - path: "terminal_service_max_sessions_test.go"
      purpose: "TestTerminalService_MaxSessionsLimit (darwin-tagged, uses newTestTerminalServiceWithMock)"
    - path: ".planning/phases/25-polish-integration/CHECKPOINT.md"
      purpose: "Phase 25 checkpoint with Windows conpty verification section + D-01..D-14 decisions list"
  modified:
    - path: "terminal_service.go"
      purpose: "Added const MaxSessions = 10 + len(s.sessions) >= MaxSessions guard at top of CreateSession"
    - path: "frontend/src/locales/en.json"
      purpose: "Added toast.maxSessionsReached with {{limit}} interpolation"
    - path: "frontend/src/App.tsx"
      purpose: "createTerminalSession catch block now detects 'max sessions reached' and surfaces localized toast"
    - path: "AGENTS.md"
      purpose: "One-line bullet in ## Tests section pointing to CHECKPOINT.md"

key-decisions:
  - "Placed TestTerminalService_MaxSessionsLimit in new //go:build darwin file (terminal_service_max_sessions_test.go) instead of terminal_service_test.go because newTestTerminalServiceWithMock is darwin-only — same deviation pattern as 25-02 stress test"
  - "MaxSessions = 10 chosen as a reasonable cap (typical use is 1-3 sessions) that prevents accidental resource exhaustion"
  - "Error string 'max sessions reached' used as the frontend matching key — simple and stable; survives Go error wrapping"
  - "{{limit}} interpolation in en.json hard-codes 10 on the frontend call site (t('toast.maxSessionsReached', { limit: 10 })) — single source of truth is the Go constant, with the integer mirrored in the call site"

patterns-established:
  - "Capacity guard pattern: const at package level + len(s.sessions) >= MaxSessions check after s.mu.Lock() + early return with formatted error"

requirements-completed: []

# Metrics
duration: 4min
completed: 2026-06-16
---

# Phase 25 Plan 3: Max-Sessions Guard + Windows Conpty Verification Summary

**MaxSessions = 10 limit enforced in `CreateSession`, localized max-sessions toast, and CHECKPOINT.md documenting the Windows conpty cross-compile-only coverage.**

## Performance

- **Duration:** 4 min
- **Started:** 2026-06-16T07:36:44Z
- **Completed:** 2026-06-16T07:40:51Z
- **Tasks:** 2
- **Files modified:** 6 (1 new test, 1 new checkpoint, 4 edited)

## Accomplishments

- **`MaxSessions = 10` guard** added to `CreateSession` — the first 10 calls succeed, the 11th returns `CreateSession: max sessions reached (10)` and the function never touches `sessionCounter` or the `sessions` map. Protects against accidental resource exhaustion from buggy frontend loops.
- **`TestTerminalService_MaxSessionsLimit`** (darwin-tagged) asserts the guard fires on the 11th call and the error string contains `max sessions reached`. Uses `newTestTerminalServiceWithMock` so the loop does not depend on a real PTY.
- **Localized `toast.maxSessionsReached`** with `{{limit}}` interpolation added to `en.json`; `App.tsx:createTerminalSession` catch block string-matches the new error and surfaces the new toast, falling back to the existing generic message for other errors.
- **Cross-compile verification** — `GOOS=windows go build ./...` from darwin exits 0. The conpty stub in `pty_backend_windows.go` compiles cleanly; no conpty API mismatches at compile time. This is the explicit D-13 verification path.
- **`CHECKPOINT.md`** created at the phase directory with: `## Summary` (7 deliverables), `## Windows conpty verification` (scope, out-of-scope gap, future work subsections), `## Files` (new/modified/removed), `## Test coverage` (4 new tests + 12 existing), `## Decisions honored` (D-01..D-14 with one-line summaries).
- **`AGENTS.md` one-line pointer** to `CHECKPOINT.md` added in the `## Tests` section so future maintainers know the Windows conpty gap exists and where to look.
- **Existing error states confirmed in place** (no edits): `App.tsx:closeTerminalSession` line 214 still has the last-tab guard; `TerminalTabBar.tsx` line 131 still hides the close `X` for the last tab; `Terminal.tsx:292` still has the PTY start failure toast.

## Task Commits

Each task was committed atomically:

1. **Task 1: MaxSessions guard + test + i18n + frontend catch** — `903f4b3` (feat)
2. **Task 2: Cross-compile + CHECKPOINT + AGENTS.md** — `be9564c` (docs)

## Files Created/Modified

- `terminal_service.go` — Added `const MaxSessions = 10` above `SessionInfo`; added `if n := len(s.sessions); n >= MaxSessions { ... return ... }` guard at the top of `CreateSession`.
- `terminal_service_max_sessions_test.go` (new) — `TestTerminalService_MaxSessionsLimit` (darwin-tagged, uses `newTestTerminalServiceWithMock`).
- `frontend/src/locales/en.json` — Added `maxSessionsReached: "Maximum number of terminal sessions ({{limit}}) reached. Close a session to create a new one."` to the `toast` object.
- `frontend/src/App.tsx` — `createTerminalSession` catch block (lines 202-211) now: captures `msg = String(err)`, calls `toast.error(t('toast.maxSessionsReached', { limit: 10 }))` if `msg.includes('max sessions reached')`, falls back to the existing generic message otherwise.
- `AGENTS.md` — `## Tests` section now lists the new stress and max-sessions test files, plus a bold `**Windows conpty verification gap:**` bullet pointing to `CHECKPOINT.md`.
- `.planning/phases/25-polish-integration/CHECKPOINT.md` (new) — Phase 25 checkpoint document with `## Windows conpty verification` section (scope/out-of-scope/future work) and `## Decisions honored` section listing D-01..D-14.

## Decisions Made

- **Test placement deviation from PLAN:** PLAN 25-03 Task 1 Step 3 says "Add `TestTerminalService_MaxSessionsLimit` to `terminal_service_test.go`." That file has no build tag; referencing `newTestTerminalServiceWithMock` (which is in a `//go:build darwin` file) from a non-tagged `_test.go` would break cross-platform test compilation. The test is therefore placed in a new `//go:build darwin` file `terminal_service_max_sessions_test.go`. This mirrors the documented deviation in PLAN 25-02 (the stress test was placed in `terminal_service_stress_test.go` for the same reason). The behavior under test is unchanged.
- **`MaxSessions = 10` cap value:** The plan specifies `MaxSessions = 10`. The plan's rationale is documented in the code comment: "exceeds normal user workflows (typical use is 1-3 sessions) while preventing accidental resource exhaustion."
- **Error string as the frontend matching key:** The frontend uses `msg.includes('max sessions reached')` rather than a typed Go error (e.g., `errors.Is(err, ErrMaxSessions)`). This is simpler and survives Go error wrapping (`fmt.Errorf("...%w", err)`); the cost is that the match is on a free-form string. The risk is low because the error is package-internal and the matching text is stable. A future hardening pass could introduce a typed error if more callers need to branch on it.
- **Hard-coded `limit: 10` in the frontend call site:** The `MaxSessions` constant is defined in Go; the frontend hard-codes the same value when calling `t('toast.maxSessionsReached', { limit: 10 })`. This is acceptable because: (a) the constant rarely changes, (b) the value is exposed to the user as a string for UX, and (c) a future Phase could expose `MaxSessions` via the existing `GetEventNames`-style Go-to-TS binding pattern. Out of scope for v2.1.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Moved `TestTerminalService_MaxSessionsLimit` to a darwin-tagged file**
- **Found during:** Task 1 (writing the test)
- **Issue:** PLAN 25-03 Task 1 Step 3 says to add the test to `terminal_service_test.go`, but that file has no build tag. Referencing `newTestTerminalServiceWithMock` (which lives in `pty_backend_mock_test.go`, a `//go:build darwin` file) from a non-tagged test file would break `go test ./...` on non-darwin platforms. This is the same compile-time constraint documented as a deviation in PLAN 25-02 (the stress test was placed in `terminal_service_stress_test.go` for the same reason).
- **Fix:** Created new `terminal_service_max_sessions_test.go` with `//go:build darwin` build tag mirroring the stress test pattern. The test logic, acceptance criteria, and Go test function name are unchanged.
- **Files modified:** `terminal_service_max_sessions_test.go` (new)
- **Verification:** `go test -run TestTerminalService_MaxSessionsLimit -v ./...` passes; `go test ./...` (full suite) reports 45 passed in 2 packages
- **Committed in:** `903f4b3` (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Same deviation pattern as PLAN 25-02. No scope creep; the test function and contract are unchanged. The plan's intent (a darwin-only max-sessions test against the mock) is preserved.

## Issues Encountered

- **Linux cross-compile sanity check fails:** The plan's acceptance criteria include `GOOS=linux go build ./... from darwin exits 0 (sanity — the unix build is the default and should also still work)`. This fails with 11 errors, all in the upstream `wailsapp/wails/v3` alpha dependency: `menu_linux.go`, `webview_window_linux.go`, and `dialogs_linux.go` reference undefined CGO types (`pointer`, `uintptr`, etc.). This is a pre-existing upstream limitation in Wails v3 alpha (no Linux cross-compile support), not introduced by this plan. The primary D-13 acceptance criterion (`GOOS=windows go build ./...` exits 0) PASSES. Logged as a deferred item — fixing it requires either an upstream Wails v3 alpha fix or pinning to a specific Wails version that does not have this issue, both out of scope for the polish phase. The project's CI does not currently run `GOOS=linux` cross-compile, so this is a non-blocker.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 25 is now complete. Plan 25-03 closes the last code-bearing plan in this phase; the existing 25-01, 25-02, 25-04 plans already have summaries. The four-plan phase delivers all seven context-document items.
- Future work for closing the Windows conpty gap is documented in `CHECKPOINT.md` (Future work section): add a conpty binding, replace the stub, add a `windows-latest` CI runner, add a Windows-tagged variant of the orchestration tests. None of these are blockers for v2.1 release.
- The `MaxSessions = 10` limit closes a previously-unbounded resource exhaustion vector. The frontend surfaces the new error gracefully (localized toast) instead of crashing on a generic error.

---

*Phase: 25-polish-integration*
*Completed: 2026-06-16*

## Self-Check: PASSED

Verified artifacts:
- `.planning/phases/25-polish-integration/25-03-SUMMARY.md` exists (11.6K)
- `.planning/phases/25-polish-integration/CHECKPOINT.md` exists (12.8K)
- `terminal_service_max_sessions_test.go` exists (1.8K)
- Commit `903f4b3` (Task 1: feat) verified in git log
- Commit `be9564c` (Task 2: docs) verified in git log
- `go test -run TestTerminalService_MaxSessionsLimit -v ./...` → 1 passed
- `go test ./...` (full suite) → 45 passed in 2 packages
- `GOOS=windows go build ./...` → EXIT 0
- `go vet ./...` → no issues
- `pnpm tsc --noEmit` in `frontend/` → no errors
- Existing last-tab guard at `App.tsx:closeTerminalSession` (line 214) preserved
- Existing `!isLastTab` UI guard in `TerminalTabBar.tsx` (line 131) preserved
- Existing `toast.error('Terminal start failed')` in `Terminal.tsx:292` preserved
