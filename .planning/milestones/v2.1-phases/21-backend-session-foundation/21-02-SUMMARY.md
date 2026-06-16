---
phase: 21-backend-session-foundation
plan: 02
subsystem: terminal
tags: [pty, goroutines, wails, events, sessions, mutex]

# Dependency graph
requires:
  - phase: 21-01
    provides: sessionState struct, SessionInfo type, TerminalService CRUD, resolveSession helper
provides:
  - Per-session PTY lifecycle (start/stop/read/emit/monitor)
  - SessionId-parameter dispatch methods (Write/Resize/Clear/Start/Stop)
  - Namespaced PTY events (pty-output:{id}, pty-exit:{id}, pty-cleared:{id})
  - Full session shutdown with goroutine cleanup
affects: [22-db-persistence, 23-frontend-tabbed-ui, 24-execution-integration]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - unlock-before-blocking deadlock prevention (ss.mu.Unlock before PTY close/wait)
    - namespaced Wails event emission ("pty-output:" + sessionId)
    - sessionId-parameter dispatch via resolveSession helper
    - snapshot-then-iterate pattern for shutdown without holding locks

key-files:
  created: []
  modified:
    - terminal_service.go (added 12 methods: startSessionLocked, stopSessionLocked, readLoop, startEmitter, stopEmitter, enqueueOutput, monitorExit, Write, Resize, Clear, Start, Stop; updated CreateSession and CloseSession)
    - event_service.go (removed PtyOutput, PtyExit, PtyCleared fields from EventNames struct and eventNames var)

key-decisions:
  - "monitorExit kept on *TerminalService (not *sessionState) for access to s.Start(id, cols, rows) auto-restart"
  - "Emitter buffer is local bytes.Buffer in goroutine (not a sessionState field) — same as old singleton pattern"
  - "Event namespacing uses inline format strings at call sites, not EventNames struct fields — per D-04 clean break"

patterns-established:
  - "ss.mu unlock-before-blocking: unguard ss.mu before PTY close, killProcessGroup, readerWg.Wait; re-acquire after"
  - "Snapshot-then-iterate: snapshot session IDs under s.mu, release lock, then iterate without holding"
  - "Namespaced events: pty-output:{id}, pty-exit:{id}, pty-cleared:{id} — inline format, no global EventNames fields"

requirements-completed: [EXEC-04]

# Metrics
duration: ~10min
completed: 2026-06-10
---

# Phase 21 Plan 02: Per-Session PTY Lifecycle and Namespaced Events Summary

**Independent per-session PTY lifecycle with unlock-before-blocking deadlock prevention, sessionId-parameter dispatch, and namespaced Wails events per D-04**

## Performance

- **Duration:** ~10 min
- **Started:** 2026-06-10T07:26:00Z
- **Completed:** 2026-06-10T07:36:47Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- `startSessionLocked(ss, cols, rows)` with critical unlock-before-blocking pattern: releases `ss.mu` before PTY close/process kill/wait, re-acquires after — same deadlock prevention as old singleton
- `readLoop`, `startEmitter`, `stopEmitter`, `enqueueOutput` extracted as `*sessionState` methods with identical logic to old TerminalService versions
- `Write(sessionId)` auto-resumes stopped shell via `startSessionLocked`; `Resize(sessionId)` validates 1..65535 range; `Clear(sessionId)` emits namespaced clear event
- `Start(sessionId)` and `Stop(sessionId)` dispatch to correct session's PTY with full cleanup on stop (kill process group, wait reader goroutine)
- `CreateSession` now calls `startEmitter()` + `startSessionLocked(ss, 80, 24)` — new sessions start with running PTY
- `CloseSession` performs full PTY teardown: stops stopCh, kills process group, waits `readerWg` and `emitterWg`
- `monitorExit` on `*TerminalService` with `ss *sessionState` parameter: emits `"pty-exit:"+ss.id` with exit code, auto-restarts via `s.Start(ss.id, cols, rows)` on crash
- `event_service.go` cleaned — `PtyOutput`, `PtyExit`, `PtyCleared` removed from both `EventNames` struct and `eventNames` var; events now inline `"pty-output:"+sessionId`

## Task Commits

Each task was committed atomically:

1. **Task 1: Implement per-session PTY lifecycle and dispatch** - `440376d` (feat)
2. **Task 2: Remove global PTY event names from EventNames** - `6d41e81` (docs)

## Files Modified
- `terminal_service.go` — Added 12 methods: `startSessionLocked`, `stopSessionLocked`, `readLoop`, `startEmitter`, `stopEmitter`, `enqueueOutput`, `monitorExit`, `Write(sessionId)`, `Resize(sessionId)`, `Clear(sessionId)`, `Start(sessionId)`, `Stop(sessionId)`; updated `CreateSession` and `CloseSession`
- `event_service.go` — Removed `PtyOutput`, `PtyExit`, `PtyCleared` fields from `EventNames` struct and `eventNames` var

## Decisions Made
- **monitorExit on TerminalService:** Kept receiver as `*TerminalService` (not `*sessionState`) because auto-restart needs `s.Start(ss.id, cols, rows)` — the `s` receiver gives access to dispatch methods
- **Local buffer in emitter:** Used local `bytes.Buffer` in emitter goroutine rather than a `sessionState` field — same pattern as old singleton code, no per-session field inflation
- **Inline event names:** Session-keyed events use inline format strings (`"pty-output:"+ss.id`) at each call site rather than precomputed fields — per D-04 clean break from global constants

## Deviations from Plan

None — plan executed exactly as written.

## Known Stubs

None — all methods fully implemented with complete PTY lifecycle, event emission, and goroutine cleanup.

## Issues Encountered

None.

## Threat Mitigations Verified

| Threat | Mitigation | Status |
|--------|-----------|--------|
| T-21-02 (zombie processes on shutdown) | `ServiceShutdown` snapshots IDs under lock, `CloseSession` kills process group, waits readerWg + emitterWg | ✅ |
| T-21-03 (goroutine leak) | `startSessionLocked` tracks via `ss.readerWg`; `CloseSession` waits both readerWg and emitterWg | ✅ |
| T-21-04 (deadlock) | `startSessionLocked` unlocks `ss.mu` before PTY close/wait, re-locks after; `CloseSession` releases `s.mu` before blocking ops | ✅ |
| T-21-05 (cross-session output leak) | Each session has own `outputCh`, `ptmx`, emitter goroutine; events namespaced with session ID | ✅ |
| T-21-06 (shell injection) | Accepted — PTY input is user-controlled by design | ✅ |

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Phase 21 Plan 03 (tests) can validate all 12 new methods against the old singleton behavior
- Phase 22 (db persistence) can layer on `terminal_sessions` table and migration
- Phase 23 (frontend tabbed UI) can connect to `Write(sessionId)`, `Resize(sessionId)`, `Clear(sessionId)` dispatch and subscribe to namespaced events

---
## Self-Check: PASSED

- File checks: `terminal_service.go` ✅, `event_service.go` ✅, `21-02-SUMMARY.md` ✅
- Commits: `440376d` (feat) ✅, `6d41e81` (docs) ✅
- Build: `go build ./...` ✅
- Vet: `go vet ./...` ✅
- Event cleanup: No `PtyOutput`/`PtyExit`/`PtyCleared` in `event_service.go` ✅

---
*Phase: 21-backend-session-foundation*
*Completed: 2026-06-10*
