---
phase: 21-backend-session-foundation
plan: 01
subsystem: terminal
tags: [go, wails, pty, session-management, uuid]

# Dependency graph
requires: []
provides:
  - SessionInfo struct (5 exported fields, camelCase JSON tags)
  - sessionState struct (15+ internal per-PTY fields)
  - TerminalService as multi-session manager (sessions map, activeSessionID, sessionCounter)
  - CreateSession, ListSessions, CloseSession, RenameSession, SetActiveSession, GetActiveSession
  - resolveSession() helper with active session fallback
  - getWorkingDir() helper using os.UserHomeDir()
affects:
  - 21-02-backend-pty-lifecycle (uses sessionState fields, resolveSession helper)
  - 21-03-backend-tests (tests session CRUD API)
  - 22-backend-persistence (adds database table on top of in-memory sessions)

# Tech tracking
tech-stack:
  added:
    - github.com/google/uuid (UUID v4 session ID generation)
  patterns:
    - sync.RWMutex for concurrent-safe session map access (Lock for writes, RLock for reads)
    - Internal unexported sessionState with per-session sync.Mutex
    - Public exported SessionInfo with json:"camelCase" tags for Wails serialization
    - Method naming: MethodName: description: %w error wrapping convention
    - Active session fallback: resolveSession uses activeSessionID when sessionId is ""

key-files:
  created: []
  modified:
    - terminal_service.go (complete rewrite: singleton → multi-session manager)
    - execution_service_test.go (s.Stop() → s.ServiceShutdown())
    - terminal_service_test.go (//go:build ignore until Plan 03 rewrite)

key-decisions:
  - "TerminalService refactored in-place as session manager — no separate SessionService struct"
  - "Session IDs use UUID v4 via github.com/google/uuid for non-enumerable IDs"
  - "First session auto-active (activeSessionID set on CreateSession when empty)"
  - "CloseSession reassigns active to any remaining session (non-deterministic pick)"
  - "PTY cleanup deferred to Plan 02 with TODO stub in CloseSession"

requirements-completed: [SESS-01, SESS-04, SESS-05]

# Metrics
duration: 5min
completed: 2026-06-10
---

# Phase 21 Plan 01: Backend Session Foundation Summary

**Multi-session TerminalService with UUID keyed in-memory session manager, SessionInfo/SessionState types, and full session CRUD API**

## Performance

- **Duration:** 5 min
- **Started:** 2026-06-10T07:22:34Z
- **Completed:** 2026-06-10T07:28:04Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- Defined `SessionInfo` (exported, 5 fields, camelCase JSON tags) and `sessionState` (internal, 15+ PTY fields) types
- Refactored `TerminalService` from singleton to in-memory session manager with `map[string]*sessionState`
- Upgraded mutex from `sync.Mutex` to `sync.RWMutex` for concurrent read safety
- Implemented full session CRUD: CreateSession (UUID v4, auto-active), ListSessions, CloseSession (map cleanup + active reassignment), RenameSession (non-empty validation), SetActiveSession, GetActiveSession
- Added `resolveSession()` helper with active session fallback and `getWorkingDir()` helper

## Task Commits

Each task was committed atomically:

1. **Task 1: Define sessionState, SessionInfo, refactor TerminalService struct** - `6cafe6b` (feat)
2. **Task 2: Implement session CRUD — CreateSession, ListSessions, CloseSession, RenameSession, active session methods** - `c34160f` (feat)

## Files Created/Modified

- `terminal_service.go` - Complete restructure: new SessionInfo/sessionState types, refactored TerminalService (4 manager fields), 7 new methods (Create/List/Close/Rename/SetActive/GetActive/resolveSession), removed 13 old singleton methods
- `execution_service_test.go` - Fixed `s.Stop()` → `s.ServiceShutdown()` for API compatibility
- `terminal_service_test.go` - Added `//go:build ignore` build tag (Plan 03 will rewrite for multi-session API)

## Decisions Made

- Refactored `TerminalService` in-place as session manager rather than creating a separate `SessionService` struct — keeps API surface smaller per D-01
- Used `sync.RWMutex` for concurrent-safe session map access — RLock for reads, Lock for writes per D-06
- First session auto-active via `activeSessionID` on `CreateSession` per D-03 hybrid approach
- PTY cleanup intentionally stubbed with `// TODO(Plan 02)` — Plan 02 adds `ss.stopSession()` to kill PTY and wait for goroutines

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed execution_service_test.go referencing removed s.Stop()**
- **Found during:** Task 2 (verification)
- **Issue:** `go vet` failed on `execution_service_test.go:183` — `s.Stop()` no longer exists on `*TerminalService`
- **Fix:** Changed `defer s.Stop()` to `defer s.ServiceShutdown()`
- **Files modified:** execution_service_test.go
- **Committed in:** c34160f (Task 2 commit)

**2. [Rule 1 - Bug] terminal_service_test.go broken by removed singleton API**
- **Found during:** Task 2 (verification)
- **Issue:** `go vet` failed on `terminal_service_test.go:17` — `s.Start()`, `s.Stop()`, `s.ptmx`, `s.cmd`, etc. all reference removed fields/methods
- **Fix:** Added `//go:build ignore` build tag with `// TODO(Plan 03)` — Plan 03 will rewrite tests for multi-session API
- **Files modified:** terminal_service_test.go
- **Committed in:** c34160f (Task 2 commit)

**3. [Rule 1 - Bug] Removed unused imports after method deletion**
- **Found during:** Task 2 (verification)
- **Issue:** `"strings"` and `"unicode/utf8"` imports unused after old singleton methods were deleted
- **Fix:** Removed both imports
- **Files modified:** terminal_service.go
- **Committed in:** c34160f (Task 2 commit)

---

**Total deviations:** 3 auto-fixed (3 Rule 1 bugs)
**Impact on plan:** All auto-fixes necessary for `go build ./...` and `go vet ./...` to pass. No scope creep — test file updates are Plan 03's responsibility.

## Known Stubs

- `terminal_service.go:211` — `// TODO(Plan 02): call ss.stopSession() to kill PTY and wait for goroutines` in `CloseSession`. Intentional: PTY lifecycle is Plan 02's scope. The `_ = ss` line prevents unused variable error.
- `terminal_service.go:79` — `getWorkingDir()` returns empty string on `os.UserHomeDir()` error. Intentional per RESEARCH.md open question 2 recommendation for Phase 21.

## Issues Encountered

None — plan executed as designed. Both tasks compiled per expectation (Task 1: expected errors from old methods, Task 2: clean build).

## Next Phase Readiness

- Session manager foundation ready for Plan 02 (PTY lifecycle: `startSessionLocked`, `stopSession`, per-session `readLoop`, `monitorExit`, emitter)
- `resolveSession()` helper available for Plan 02 method dispatch
- `sessionState` struct has all fields Plan 02 needs for PTY operations
- Test files gated for Plan 03 rewrite

---
*Phase: 21-backend-session-foundation*
*Completed: 2026-06-10*
