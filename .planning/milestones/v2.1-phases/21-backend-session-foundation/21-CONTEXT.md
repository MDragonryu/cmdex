# Phase 21: Backend Session Foundation - Context

**Gathered:** 2026-06-09
**Status:** Ready for planning

<domain>
## Phase Boundary

Backend can manage multiple terminal sessions with isolated PTYs, each independently running shell processes. Long-running processes persist when the user switches between sessions. Namespaced events route per-session output to the correct frontend terminal instance.

**Phase 21 delivers:** In-memory multi-session backend — no database persistence (that's Phase 22), no frontend tabbed UI (that's Phase 23), no command execution integration (that's Phase 24).

</domain>

<decisions>
## Implementation Decisions

### Session Architecture
- **D-01:** Refactor `TerminalService` (currently a singleton in `terminal_service.go`) to manage multiple sessions internally via `map[string]*sessionState`. TerminalService becomes the session manager — no separate `SessionService` struct needed. Each `sessionState` has its own PTY, reader goroutine, output channel, emitter goroutine, and shell process.
- **D-02:** Session IDs are UUID v4 generated via `crypto/rand`. Default session name format: `"Terminal 1"`, `"Terminal 2"` (incremental counter, in-memory, not persisted in Phase 21). Rename accepts any non-empty string.

### Active Session
- **D-03:** Hybrid approach. Backend `TerminalService` is source of truth with `SetActiveSession(id string)` and `GetActiveSession() *SessionInfo` methods. Frontend caches `activeSessionId` locally for immediate UI updates, syncs on startup. Future `RunCommand` accepts optional `sessionId` param, falls back to backend's `GetActiveSession()`.

### Event Namespacing
- **D-04:** Namespaced events only — no global fallback. Format: `pty-output:{sessionId}`, `pty-exit:{sessionId}`, `pty-cleared:{sessionId}`. Frontend subscribes to active session's events, unsubscribes on tab switch. Legacy global event names (`pty-output`, `pty-exit`, `pty-cleared`) are removed entirely — clean break.

### Shell Lifecycle
- **D-05:** Each session independently follows the existing `monitorExit` pattern: unintentional shell exit with non-zero code → wait 100ms → auto-restart shell. Intentional exit (user typed `exit` or `Stop()` called) → shell stays stopped, session marked inactive. Shell crashes never propagate to other sessions.

### Persistence Boundary
- **D-06:** Phase 21 is in-memory only (`map[string]*sessionState`). No database table, no migration, no persistence. Phase 22 will add the `terminal_sessions` table, migration `0011`, and CRUD persistence layer on top of this in-memory structure.

### Error Handling
- **D-07:** `CreateSession` returns error if PTY cannot start. `CloseSession` always succeeds (force-kills PTY and process group via `Stop()`). `Rename` succeeds for any non-empty string. `Write` auto-resumes shell via `Start()` if session is stopped (existing behavior preserved). All errors propagate as Go error returns; frontend displays toast on failure.

### the agent's Discretion
- Shell dimensions: New sessions default to 80x24 cols/rows (matches existing `ServiceStartup` behavior).
- Session counter: Incremental counter lives in `TerminalService`, starts at 1, never resets (only in-memory in Phase 21).
- `SessionInfo` struct should expose fields sufficient for Phase 23 (tab UI): `id`, `name`, `running`, `shellPath`, `workingDir`.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Roadmap & Requirements
- `.planning/ROADMAP.md` — Phase 21 definition, success criteria, dependencies
- `.planning/REQUIREMENTS.md` — SESS-01, SESS-04, SESS-05, EXEC-04
- `.planning/PROJECT.md` — Project context, constraints, key decisions

### Existing Code (must study before implementing)
- `terminal_service.go` — Current `TerminalService` struct: all PTY management, readLoop, monitorExit, emitter pattern, mutex locking, Write/Resize/Clear/Start/Stop
- `terminal_unix.go` — `ptyStart`, `ptyResize` for Unix (`github.com/creack/pty`)
- `terminal_windows.go` — Windows stubs (`ptyStart`, `ptyResize`), `conpty` integration
- `event_service.go` — `EventNames` struct, event name constants, `PtyOutput`, `PtyExit`, `PtyCleared` fields (to be extended with namespacing)
- `main.go` — Service registration pattern (`application.Service`), Wails service lifecycle
- `executor.go` — `killProcessGroup` helper, shell detection patterns

### Reference Documents
- `.planning/codebase/ARCHITECTURE.md` — Service patterns, data flow, entry points
- `.planning/codebase/CONVENTIONS.md` — Go naming, error handling, import conventions
- `.planning/research/ARCHITECTURE.md` — Suggested build order, integration points
- `.planning/research/PITFALLS.md` — Global state collision, event routing, race conditions
- `.planning/research/STACK.md` — Library versions, what NOT to add

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`TerminalService` struct** (`terminal_service.go:24-47`): Contains `mu sync.Mutex`, `ptmx *os.File`, `cmd *exec.Cmd`, `lastSize ptyWinsize`, `stopCh`, `outputCh`, `emitterWg`, `readerWg`, `droppedCount`. All fields except ID/name are reusable per session.
- **`readLoop`** (`terminal_service.go:134-178`): Byte-buffered PTY read with UTF-8 boundary handling, leftover reassembly, `enqueueOutput` dispatch. Extract to `sessionState` without modification.
- **`startEmitter` / `stopEmitter`** (`terminal_service.go:67-117`): 8ms batching ticker, 32KB flush threshold, sequence numbering via `atomic.Uint64`, event emission via `wailsApp.Emit`. Each session needs its own emitter goroutine.
- **`monitorExit`** (`terminal_service.go:180-223`): `cmd.Wait()`, exit code extraction, intentional/unintentional detection, auto-restart on crash. Per-session with sessionId in events.
- **`detectShell`** (`terminal_service.go:49-65`): Platform-aware shell detection (Unix: `$SHELL` → `/bin/sh`, Windows: `pwsh` → `powershell` → `cmd`). Called per session on `Start()`.

### Established Patterns
- **Wails service pattern**: Services implement `ServiceStartup(ctx, options) error` and `ServiceShutdown() error`. Registered in `main.go` `Services` slice. Bindings generated via `wails3 generate build-assets`.
- **Event emission**: `wailsApp.Event.Emit(eventNames.X, map[string]interface{}{...})`. Event names must be added to `EventNames` struct in `event_service.go`.
- **Mutex patterns**: `s.mu.Lock()` / `s.mu.Unlock()` with careful unlock-before-blocking-calls pattern (see `startLocked` unlocking before PTY close/wait). Must preserve this when extracting per-session state.
- **Output channel**: Unbuffered? Actually buffered 512. `enqueueOutput` with `select`/`default` for non-blocking sends.

### Integration Points
- **`ServiceStartup`** (`terminal_service.go:225-233`): Currently starts one shell. Must change to: start emitter, initialize `sessions` map, create one default session.
- **Event names** (`event_service.go:16-29`): Current `PtyOutput`, `PtyExit`, `PtyCleared` emit globally. Must be replaced with session-keyed emission. Event pattern: `"pty-output:" + sessionId`.
- **Service registration** (`main.go`): TerminalService already registered. No new registration needed — only internal refactoring.

</code_context>

<specifics>
## Specific Ideas

All decisions emerged from discussion. No external references or "I want it like X" moments.

The user confirmed the existing TerminalService is the right foundation — refactor it internally rather than creating a parallel service. This keeps the API surface smaller and avoids coordination complexity between two terminal-related services.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope. All feature expansions (split panes, session sharing, broadcast) are already in REQUIREMENTS.md v2/Out of Scope.

</deferred>

---

*Phase: 21-Backend Session Foundation*
*Context gathered: 2026-06-09*
