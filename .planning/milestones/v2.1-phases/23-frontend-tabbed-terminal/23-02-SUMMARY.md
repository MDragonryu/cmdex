---
phase: 23-frontend-tabbed-terminal
plan: 02
subsystem: ui
tags: [react, typescript, terminal, sessions, multi-instance, ref-map]

# Dependency graph
requires:
  - phase: 23-01
    provides: "TerminalTabBar component, SessionInfo type, CSS classes"
provides:
  - "sessions: useState<SessionInfo[]> with full CRUD callbacks"
  - "terminalRefs: useRef<Record<string, TerminalHandle>> per-session ref map"
  - "TerminalTabBar integration in App.tsx center-area layout"
  - "Multi-instance TerminalComponent mounting (one per session, keyed by session.id)"
  - "Clear/copy buttons refactored to target active session via terminalRefs[activeSessionId]"
  - "Mount-time session loading with default-session fallback"
  - "Per-session pty-exit event subscriptions for reactive status updates"
affects:
  - 23-03 (keyboard shortcuts and final integration)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Multi-instance mounting with key={session.id} and CSS display:none for inactive (same as Phase 14 CommandDetailTab pattern)"
    - "Ref callback pattern for per-session TerminalHandle registration into Record<string, TerminalHandle> map"
    - "Promise chain with intermediate data passthrough for mount-time loading (ListSessions → GetActiveSession → fallback CreateSession)"
    - "Events.On subscriptions with mapped dependency string (sessions.map(s => s.id).join(',')) for stable re-subscribe"
    - "Session CRUD with toast error UX on failure ('Could not create session...', 'Could not rename session...')"

key-files:
  created: []
  modified:
    - frontend/src/App.tsx

key-decisions:
  - "terminalRefs uses Record<string, TerminalHandle> keyed by session UUID — ref callback registers on mount, deletes on unmount"
  - "closeTerminalSession gates on sessions.length <= 1 (last tab not closeable), auto-selects nearest remaining on active-session close"
  - "renameTerminalSession validates non-empty trimmed name before calling RenameSession — silently rejects empty/whitespace per T-23-01"
  - "pty-exit event subscription uses sessions.map(s => s.id).join(',') as dependency — stable string prevents unnecessary re-subscriptions"
  - "onShellExit updates running:false locally AND conditionally collapses terminal only if the exited session was active"

patterns-established:
  - "Promise chain pattern: ListSessions().then(loaded => ... GetActiveSession().then(info => ({ loaded, info }))) for dependent async data"
  - "JSX fragment wrapper for multiple terminal-area elements inside conditional !terminalCollapsed guard"

requirements-completed: [SESS-03, UI-04, UI-06]

# Metrics
duration: 4min
completed: 2026-06-10
---

# Phase 23 Plan 02: App.tsx Session State Wiring Summary

**Added sessions state array, terminalRefs map, five CRUD callbacks, mount-time loading with fallback, pty-exit event subscriptions — then wired TerminalTabBar, per-session TerminalComponents, and refactored clear/copy buttons targeting the active session**

## Performance

- **Duration:** 4 min
- **Started:** 2026-06-10T10:51:42Z
- **Completed:** 2026-06-10T10:56:34Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments

- Added `sessions: useState<SessionInfo[]>` and `terminalRefs: useRef<Record<string, TerminalHandle>>` replacing the old single `terminalRef`
- Added five session CRUD callbacks: create, close (with last-tab gate), rename (with empty-name validation), switch, reorder
- Replaced mount-time GetActiveSession with full ListSessions + GetActiveSession chain, plus auto-create default session when none exist
- Added pty-exit event subscriptions that update per-session `running: false` status reactively
- Mounted TerminalTabBar inside the terminal pane's collapsed guard with all six callback props wired
- Replaced single TerminalComponent with per-session multi-mount pattern (key={session.id}, isVisible gates on active + not-collapsed)
- Refactored clear and copy buttons from `terminalRef.current` to `terminalRefs.current[activeSessionId]`
- All `terminalRef.current` references removed — 0 stale references remain

## Task Commits

Each task was committed atomically:

1. **Task 1: Add session state management to App.tsx — states, refs, callbacks, event subscriptions, mount-time loading** - `73a2efe` (feat)
2. **Task 2: Wire TerminalTabBar + multi-mount TerminalComponents + refactor clear/copy buttons** - `2adca7d` (feat)

## Files Created/Modified

- `frontend/src/App.tsx` — +136 lines total across both tasks. Added: sessions state, terminalRefs map, 5 CRUD callbacks, 2 new useEffects (mount-time loading + pty-exit subscriptions), TerminalTabBar import + mount, per-session TerminalComponent mapping, refactored clear/copy button handlers. Removed: single terminalRef, old GetActiveSession-only mount useEffect, single TerminalComponent mount.

## Decisions Made

- Used `Record<string, TerminalHandle>` for terminalRefs — ref callback pattern registers each TerminalHandle on mount via `terminalRefs.current[session.id] = el` and deletes on unmount
- closeTerminalSession depends on `[sessions, activeSessionId]` for the length gate — recreated when sessions change, ensuring accurate `sessions.length <= 1` check
- renameTerminalSession validates `name.trim()` is non-empty — silently returns when empty (per T-23-01 threat mitigation)
- pty-exit useEffect depends on `sessions.map(s => s.id).join(',')` — stable string prevents re-subscriptions on name/running updates
- Terminal components use `key={session.id}` — stable UUID keys prevent remount on reorder (per Pitfall 3: index-based keys destroy scrollback)
- onShellExit: updates local running status AND conditionally collapses terminal only if exited session === activeSessionId

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered

- **JSX fragment placement error:** `</>` was initially placed outside the `!terminalCollapsed && (` closing `)`, causing TS2657 "JSX expressions must have one parent element." Fixed by moving `</>` before `)}` — fragment must be fully inside the conditional expression.
- **AC4 (terminalRefs ≥ 2 matches) at Task 1 boundary:** After Task 1, `terminalRefs` had only 1 match (declaration only). Usage matches (ref callback, clear, copy) are added in Task 2. This is expected cross-task dependency — the criteria is fully satisfied after both tasks complete (5 matches total after Task 2).

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

Ready for Plan 03 (keyboard shortcuts and final integration):
- `sessions` state and all CRUD callbacks are fully wired
- `terminalRefs` map provides per-session access for clear/copy
- TerminalTabBar is mounted with all six callback props
- Multi-instance TerminalComponents preserve independent scrollback per session
- Clear and copy buttons target the active session correctly
- pty-exit events update running status reactively
- No blockers — TypeScript compiles clean, all 12+12 acceptance criteria pass

## Threat Surface

No new trust boundaries. Existing mitigations enforced:
- **T-23-01 (rename input validation):** `renameTerminalSession` validates `name.trim()` non-empty before calling `RenameSession` — silently rejects empty/whitespace names
- **T-23-02 (event listener cleanup):** pty-exit useEffect returns cleanup function that calls all registered `Events.On` cleanup handlers — listener count does not grow unbounded
- **T-23-03 (xterm.js memory):** Accepted — user controls session count in local desktop app
- **T-23-04 (hidden terminal output):** Accepted — only active session's output is accessible via clear/copy buttons

## Self-Check: PASSED

- [x] `cd frontend && pnpm tsc --noEmit` exits 0
- [x] `grep "terminalRef\.current" frontend/src/App.tsx` returns 0 (all stale refs removed)
- [x] `grep "terminalRefs\.current\[activeSessionId\]" frontend/src/App.tsx` returns 2 (clear + copy wired)
- [x] `grep "key={session\.id}" frontend/src/App.tsx` returns 1 (stable UUID keys)
- [x] `grep "<TerminalTabBar" frontend/src/App.tsx` returns 1 (component mounted)
- [x] All 6 TerminalTabBar callback props verified wired in source
- [x] Both commits verified in git log: `73a2efe`, `2adca7d`

---

*Phase: 23-frontend-tabbed-terminal*
*Completed: 2026-06-10*
