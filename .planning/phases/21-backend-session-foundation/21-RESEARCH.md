# Phase 21: Backend Session Foundation - Research

**Researched:** 2026-06-10
**Domain:** Multi-session PTY management, in-memory session lifecycle, namespaced event routing
**Confidence:** HIGH

## Summary

Phase 21 refactors the existing `TerminalService` singleton into an in-memory multi-session manager. Rather than creating a new `SessionService` struct, the existing `TerminalService` gains a `sessions map[string]*sessionState` and internal routing logic. Each session gets its own PTY, shell process, reader goroutine, output channel, and emitter goroutine — all extracted from the current single-instance fields. Events switch from global `pty-output`/`pty-exit`/`pty-cleared` to namespaced `pty-output:{sessionId}` format. No new external packages are needed — all dependencies are already in `go.mod`.

The critical architectural constraint is that the `TerminalService` struct remains a single registered Wails service — the refactoring is entirely internal. The frontend API surface changes (all methods gain a `sessionId` parameter), but the service registration in `main.go` does not change. The global `terminalSvc` package variable is retained for `execution_service.go` compatibility (Phase 24), but its type's internal structure changes completely.

**Primary recommendation:** Extract all per-session state from `TerminalService` into a `sessionState` struct, manage instances in a `sync.RWMutex`-protected `map[string]*sessionState`, and implement all existing methods as dispatchers that look up the session by ID. Namespace events per session. Remove global event name constants, replace with session-keyed format strings.

## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Refactor `TerminalService` (currently a singleton in `terminal_service.go`) to manage multiple sessions internally via `map[string]*sessionState`. TerminalService becomes the session manager — no separate `SessionService` struct needed. Each `sessionState` has its own PTY, reader goroutine, output channel, emitter goroutine, and shell process.
- **D-02:** Session IDs are UUID v4 generated via `crypto/rand`. Default session name format: `"Terminal 1"`, `"Terminal 2"` (incremental counter, in-memory, not persisted in Phase 21). Rename accepts any non-empty string.
- **D-03:** Hybrid approach. Backend `TerminalService` is source of truth with `SetActiveSession(id string)` and `GetActiveSession() *SessionInfo` methods. Frontend caches `activeSessionId` locally for immediate UI updates, syncs on startup. Future `RunCommand` accepts optional `sessionId` param, falls back to backend's `GetActiveSession()`.
- **D-04:** Namespaced events only — no global fallback. Format: `pty-output:{sessionId}`, `pty-exit:{sessionId}`, `pty-cleared:{sessionId}`. Frontend subscribes to active session's events, unsubscribes on tab switch. Legacy global event names (`pty-output`, `pty-exit`, `pty-cleared`) are removed entirely — clean break.
- **D-05:** Each session independently follows the existing `monitorExit` pattern: unintentional shell exit with non-zero code → wait 100ms → auto-restart shell. Intentional exit (user typed `exit` or `Stop()` called) → shell stays stopped, session marked inactive. Shell crashes never propagate to other sessions.
- **D-06:** Phase 21 is in-memory only (`map[string]*sessionState`). No database table, no migration, no persistence. Phase 22 will add the `terminal_sessions` table, migration `0011`, and CRUD persistence layer on top of this in-memory structure.
- **D-07:** `CreateSession` returns error if PTY cannot start. `CloseSession` always succeeds (force-kills PTY and process group via `Stop()`). `Rename` succeeds for any non-empty string. `Write` auto-resumes shell via `Start()` if session is stopped (existing behavior preserved). All errors propagate as Go error returns; frontend displays toast on failure.

### the agent's Discretion
- Shell dimensions: New sessions default to 80x24 cols/rows (matches existing `ServiceStartup` behavior).
- Session counter: Incremental counter lives in `TerminalService`, starts at 1, never resets (only in-memory in Phase 21).
- `SessionInfo` struct should expose fields sufficient for Phase 23 (tab UI): `id`, `name`, `running`, `shellPath`, `workingDir`.

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope. All feature expansions (split panes, session sharing, broadcast) are already in REQUIREMENTS.md v2/Out of Scope.

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SESS-01 | User can create a new terminal session with a default name | `CreateSession` method on `TerminalService` using `sessionState` extraction pattern, UUID v4 via `google/uuid`, counter for default name `"Terminal N"` |
| SESS-04 | User can rename a terminal session | `RenameSession(id, name string) error` — mutex-protected map update, validates non-empty per D-07 |
| SESS-05 | User can close a terminal session (with confirmation if process running) | `CloseSession(id string) error` — calls `session.Stop()`, waits for goroutines via `readerWg.Wait()`, removes from map, always succeeds per D-07 |
| EXEC-04 | Long-running processes (servers, watchers, tail) continue running when user switches sessions | Architecture ensures per-session PTY independence — each `sessionState` has its own `cmd *exec.Cmd`, `ptmx *os.File`, goroutines. No shared state. Active session tracking is metadata only — does not affect running processes. |

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Session creation & lifecycle (Create/Rename/Close) | API / Backend | — | `TerminalService` Go methods manage the in-memory `sessions map` |
| PTY management (start/stop/resize/write) | API / Backend | — | Per-session `sessionState` owns `ptmx`, `cmd`, goroutines; uses `creack/pty` for Unix, stubs for Windows |
| Event emission (pty-output/exit/cleared) | API / Backend | — | Each session's emitter goroutine emits via `wailsApp.Event.Emit` with session-keyed event name |
| Active session tracking | API / Backend | Browser / Client | Backend is source of truth per D-03; frontend caches for immediate UI |
| Event subscription & routing | Browser / Client | — | React `TerminalComponent` subscribes to `pty-output:{sessionId}` per active tab |
| Terminal rendering (xterm.js) | Browser / Client | — | Per-tab `Terminal` instance; Phase 23 concern |
| Shell process lifecycle | API / Backend | — | `monitorExit` goroutine per session; auto-restart on crash (D-05) |
| Session persistence | Database / Storage | — | Out of scope for Phase 21 (D-06); Phase 22 adds SQLite table |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/creack/pty` | v1.1.24 | Unix PTY creation, resize, process management | Established Go PTY library (9+ years, 4k+ stars); already in `go.mod` and `terminal_unix.go` |
| `github.com/google/uuid` | v1.6.0 | UUID v4 generation via `crypto/rand` | Official Google UUID package; already used throughout codebase (`command_service.go`, `db.go`, `execution_service.go`) |
| `github.com/wailsapp/wails/v3` | v3.0.0-alpha.74 | Desktop app framework, event system, service lifecycle | Project's runtime framework; `application.Service` pattern, `Event.Emit`, `application.Get()` |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Go `sync` (stdlib) | — | `sync.RWMutex` for session map, `sync.WaitGroup` for goroutine lifecycle | Always — no third-party mutex needed |
| Go `os/exec` (stdlib) | — | Shell process creation, `cmd.Wait()`, process group management | Always — reused from existing `terminal_service.go` |
| Go `crypto/rand` (stdlib) | — | Entropy source for UUID v4 | Used internally by `google/uuid` — no direct import needed |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Refactoring `TerminalService` in-place (D-01) | New `SessionService` struct | Extra service registration, coordination complexity, two services managing PTYs — rejected in discussion |
| `google/uuid` for UUID v4 (D-02) | Manual `crypto/rand` UUID assembly | More code, same result; `google/uuid` already a direct dependency with no issues |
| Namespaced events only (D-04) | Global + namespaced with fallback | Adds complexity for backward compat that isn't needed — clean break chosen |

**Installation:**
```bash
# No new packages to install. All dependencies already in go.mod.
# Confirm with:
go mod verify
```

**Version verification:**
```bash
# Confirmed via go list -m
go list -m github.com/creack/pty          # v1.1.24
go list -m github.com/google/uuid          # v1.6.0
go list -m github.com/wailsapp/wails/v3    # v3.0.0-alpha.74
```

## Package Legitimacy Audit

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| `github.com/creack/pty` | Go modules | 9+ yrs | — | github.com/creack/pty (4k+ stars) | OK | Approved — existing dependency |
| `github.com/google/uuid` | Go modules | 9+ yrs | — | github.com/google/uuid (6k+ stars) | OK | Approved — existing dependency |
| `github.com/wailsapp/wails/v3` | Go modules | 3+ yrs | — | github.com/wailsapp/wails | OK | Approved — project framework |

**Packages removed due to SLOP verdict:** None
**Packages flagged as suspicious SUS:** None

*No new packages are introduced by this phase. All three are established, widely-used Go libraries already present in `go.mod` as direct dependencies. Verified via `go list -m` showing correct versions in `go.sum`.*

## Architecture Patterns

### System Architecture Diagram

```
Frontend (Browser/Client)                    Backend (Go/Wails)
─────────────────────────────────            ────────────────────────────
                                             
App.tsx                                      TerminalService (single Wails service)
  │                                            │
  ├─ TerminalComponent(sessionId="A")          ├─ sessions map[string]*sessionState
  │   ├── Events.On("pty-output:A", ...)       │   ├─ activeSessionID string
  │   ├── Events.On("pty-exit:A", ...)         │   ├─ sessionCounter int
  │   ├── Events.On("pty-cleared:A", ...)      │   └─ mu sync.RWMutex
  │   ├── Write(sessionId="A", data) ────────► │
  │   ├── Resize(sessionId="A", cols, rows) ─► │   sessionState["A"]
  │   └── Clear(sessionId="A") ──────────────► │     ├── id: "uuid-A"
  │                                              │     ├── name: "Terminal 1"
  ├─ TerminalComponent(sessionId="B")            │     ├── running: true
  │   ├── Events.On("pty-output:B", ...)         │     ├── ptmx: *os.File
  │   └── ...                                    │     ├── cmd: *exec.Cmd
  │                                              │     ├── outputCh: chan string (512)
  │                                              │     ├── stopCh: chan struct{}
  │                                              │     ├── readerWg: sync.WaitGroup
  │                                              │     ├── emitterWg: sync.WaitGroup
  │                                              │     ├── readLoop goroutine ─► Read PTY
  │                                              │     ├── monitorExit goroutine ─► cmd.Wait()
  │                                              │     └── startEmitter goroutine ─► batch→Emit
  │                                              │
  │                                              │   sessionState["B"]
  │                                              │     ├── id: "uuid-B"
  │                                              │     ├── name: "Terminal 2"
  │                                              │     ├── running: true (sleep 30)
  │                                              │     └── ... (independent PTY)
  │                                              │
  │                                              │   ── Event Emission ──►
  │                                              │     pty-output:A → Only Component A
  │                                              │     pty-output:B → Only Component B
```

**Data flow for session switch:** User clicks tab B → Frontend sets `activeSessionId="B"` → unsubscribes from `pty-output:A` events → subscribes to `pty-output:B` events → calls `SetActiveSession("B")` on backend. Session A's `sleep 30` continues running — its PTY, goroutines, and output channel remain active. Output from session A is still emitted (no frontend listener), output from session B routes to visible terminal.

### Recommended Project Structure

```
├── terminal_service.go      # TerminalService: session manager (refactored)
│   ├── TerminalService struct: sessions map, activeSessionID, sessionCounter, mu
│   ├── sessionState struct: all per-session fields extracted from current TerminalService
│   ├── SessionInfo struct: public metadata (id, name, running, shellPath, workingDir)
│   ├── Public methods: CreateSession, CloseSession, RenameSession, ListSessions,
│   │   GetActiveSession, SetActiveSession, Write, Resize, Clear, Start (all with sessionId)
│   ├── Internal: createSessionLocked, closeSessionLocked, getSessionLocked
│   └── Lifecycle: ServiceStartup (creates default session), ServiceShutdown (closes all)
│
├── terminal_unix.go         # Unchanged: ptyStart, ptyResize, killProcessGroup
├── terminal_windows.go      # Unchanged: Windows stubs (no new Windows work in Phase 21)
├── event_service.go          # Modified: remove PtyOutput/PtyExit/PtyCleared fields;
│                             #   add event name format strings or session-keyed helper
├── app.go                    # Minor: remove terminalSvc global if no longer needed;
│                             #   terminalSvc retained for execution_service.go compatibility
├── execution_service_test.go # Updated: test no longer checks terminalSvc singleton assignment
|
├── frontend/
│   ├── src/
│   │   ├── components/Terminal.tsx    # Minor Phase 21 changes: import renamed bindings
│   │   └── wails/events.ts           # Updated: remove ptyOutput/ptyExit/ptyCleared,
│   │                                  #   add session event name builder
│   └── bindings/                     # Regenerated by wails3 dev after Go method changes
```

### Pattern 1: sessionState Extraction

**What:** Extract all per-instance fields from `TerminalService` into a new `sessionState` struct. `TerminalService` becomes a session registry/manager with only `sessions map`, `activeSessionID`, `sessionCounter`, and `mu`.

**When to use:** This is mandatory per D-01 — the entire phase is built on this extraction.

**Mapping of fields from TerminalService → sessionState:**

| Current TerminalService Field | Moves To | Notes |
|-------------------------------|----------|-------|
| `mu sync.Mutex` | `TerminalService.mu sync.RWMutex` | Upgraded to RWMutex for concurrent read safety on map |
| `ptmx *os.File` | `sessionState.ptmx` | Per session |
| `cmd *exec.Cmd` | `sessionState.cmd` | Per session |
| `shellPath string` | `sessionState.shellPath` | Per session |
| `shellFlag string` | `sessionState.shellFlag` | Per session |
| `lastSize ptyWinsize` | `sessionState.lastSize` | Per session |
| `stopCh chan struct{}` | `sessionState.stopCh` | Per session |
| `running bool` | `sessionState.running` | Per session |
| `starting bool` | `sessionState.starting` | Per session (guard against concurrent Start) |
| `intentionalStop bool` | `sessionState.intentionalStop` | Per session |
| `readerWg sync.WaitGroup` | `sessionState.readerWg` | Per session |
| `outputCh chan string` | `sessionState.outputCh` | Per session (512 buffer) |
| `outputSeq uint64` | `sessionState.outputSeq` | Per session (atomic) |
| `emitterWg sync.WaitGroup` | `sessionState.emitterWg` | Per session |
| `droppedCount atomic.Uint64` | `sessionState.droppedCount` | Per session |

**New fields added to TerminalService:**
```go
type TerminalService struct {
    mu              sync.RWMutex
    sessions        map[string]*sessionState  // key = UUID v4 session ID
    activeSessionID string                     // current active session per D-03
    sessionCounter  int                        // starts at 1, never resets
}
```

**New fields added to sessionState (beyond extracted fields):**
```go
type sessionState struct {
    id         string    // UUID v4
    name       string    // default "Terminal N"
    workingDir string    // inherited from app defaults (for Phase 22/23)
    createdAt  time.Time // in-memory only in Phase 21
    
    // ... all extracted fields from TerminalService ...
}
```

**Example:**
```go
// Source: project AGENTS.md architecture pattern, CONTEXT.md D-01
type sessionState struct {
    id              string
    name            string
    workingDir      string
    createdAt       time.Time
    
    mu              sync.Mutex
    ptmx            *os.File
    cmd             *exec.Cmd
    shellPath       string
    shellFlag       string
    lastSize        ptyWinsize
    stopCh          chan struct{}
    running         bool
    starting        bool
    intentionalStop bool
    
    readerWg        sync.WaitGroup
    outputCh        chan string
    outputSeq       uint64
    emitterWg       sync.WaitGroup
    droppedCount    atomic.Uint64
}
```

### Pattern 2: Mutex-Protected Session Map with RWMutex

**What:** `TerminalService.sessions` is a `map[string]*sessionState` protected by `sync.RWMutex`. Read operations (`ListSessions`, `GetActiveSession`, `GetSession`) use `RLock`; write operations (`CreateSession`, `CloseSession`, `RenameSession`) use `Lock`. Never unlock a mutex from a different goroutine than the one that locked it.

**When to use:** Every method that accesses the sessions map.

**Example:**
```go
// Source: Go stdlib sync.RWMutex pattern, verified against existing codebase mutex usage
func (s *TerminalService) CreateSession() (*SessionInfo, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    s.sessionCounter++
    name := fmt.Sprintf("Terminal %d", s.sessionCounter)
    id := uuid.New().String()
    
    ss := &sessionState{
        id:         id,
        name:       name,
        workingDir: getWorkingDir(),
        createdAt:  time.Now(),
    }
    
    // Extract existing startLocked logic for session
    if err := s.startSessionLocked(ss, 80, 24); err != nil {
        return nil, err
    }
    
    s.sessions[id] = ss
    
    // First session becomes active automatically
    if s.activeSessionID == "" {
        s.activeSessionID = id
    }
    
    return ss.info(), nil
}

func (s *TerminalService) ListSessions() []*SessionInfo {
    s.mu.RLock()
    defer s.mu.RUnlock()
    
    result := make([]*SessionInfo, 0, len(s.sessions))
    for _, ss := range s.sessions {
        result = append(result, ss.info())
    }
    return result
}
```

### Pattern 3: Namespaced Event Emission

**What:** Each session emits events with its session ID embedded in the event name string. Event format: `"pty-output:" + sessionId`, `"pty-exit:" + sessionId`, `"pty-cleared:" + sessionId`. Remove global event name constants from `EventNames` struct.

**When to use:** Every PTY event emission point in emitter goroutine (`startEmitter`), `monitorExit`, and `Clear`.

**Example:**
```go
// Source: Context7 Wails v3 docs (app.Event.Emit with dynamic name), CONTEXT.md D-04
// In per-session emitter goroutine:
func (ss *sessionState) flush() {
    if ss.outputBuf.Len() == 0 {
        return
    }
    seq := atomic.AddUint64(&ss.outputSeq, 1)
    if wailsApp != nil {
        // Namespaced event: pty-output:{sessionId}
        wailsApp.Event.Emit("pty-output:"+ss.id, map[string]interface{}{
            "data": ss.outputBuf.String(),
            "seq":  seq,
        })
    }
    ss.outputBuf.Reset()
}
```

### Pattern 4: Per-Session Goroutine Lifecycle

**What:** Each `sessionState` has three goroutines: `readLoop` (reads PTY → output channel), `startEmitter` (batches output → Wails events), `monitorExit` (watches `cmd.Wait()` → exit/restart logic). Each session's goroutines are tracked by its own `readerWg`, `emitterWg` — independent of other sessions.

**When to use:** Session creation (`Start`), session closure (`Stop`). The `readLoop` must check `ss.stopCh` for cancellation. The `monitorExit` must check session-level `stopCh` before taking action.

**Existing code that already follows this pattern (extract per session):**
- `readLoop` (`terminal_service.go:134-178`) — extract to `ss.readLoop(ptmx, stopCh)`
- `startEmitter` (`terminal_service.go:67-109`) — extract to `ss.startEmitter()`
- `monitorExit` (`terminal_service.go:180-223`) — extract to `ss.monitorExit(cmd, ptmx, stopCh)`
- `stopLocked` (`terminal_service.go:241-246`) — extract to `ss.stopLocked()`
- `startLocked` (`terminal_service.go:248-305`) — extract to session methods with `ss` receiver

### Pattern 5: Method Dispatch with Session ID

**What:** Public Wails-bound methods accept a `sessionId` parameter and dispatch to the appropriate session's internal method. If `sessionId` is empty string, use the active session.

**When to use:** `Write`, `Resize`, `Clear`, `Start` — all methods that operate on a specific session.

**Example:**
```go
// Source: CONTEXT.md D-03 (hybrid active session), D-01 (TerminalService as manager)
func (s *TerminalService) Write(sessionId string, data string) error {
    ss, err := s.getSession(sessionId)  // falls back to active if sessionId == ""
    if err != nil {
        return err
    }
    return ss.write(data)
}

func (s *TerminalService) getSession(sessionId string) (*sessionState, error) {
    if sessionId == "" {
        s.mu.RLock()
        sessionId = s.activeSessionID
        s.mu.RUnlock()
    }
    s.mu.RLock()
    ss, ok := s.sessions[sessionId]
    s.mu.RUnlock()
    if !ok {
        return nil, fmt.Errorf("session not found: %s", sessionId)
    }
    return ss, nil
}
```

### Anti-Patterns to Avoid

- **Global event fallback:** Do NOT emit both `pty-output` AND `pty-output:{id}`. D-04 mandates clean break — namespaced only. Remove `PtyOutput`, `PtyExit`, `PtyCleared` fields from `EventNames` struct entirely.
- **Unlock-before-blocking omitted:** The existing `startLocked` has a critical unlock-before-PTY-close pattern (`s.mu.Unlock()` before `oldPtmx.Close()` on line 271). Per-session code must preserve this to avoid deadlock with the reader goroutine. The session-level mutex must be unlocked before closing old PTY files or waiting on `readerWg`.
- **Sharing channels between sessions:** NEVER share `outputCh` or `stopCh` between sessions. Each `sessionState` creates its own channels in `startEmitter`/`startSessionLocked`.
- **Session map access without mutex:** Go's map is not goroutine-safe. Any concurrent access (even reads during a write) causes a fatal `concurrent map read and map write` panic.
- **Unlocking mutex from wrong goroutine:** The `mu.Unlock()` calls must happen in the same goroutine that called `mu.Lock()`. Do not pass locked mutexes to goroutines.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| UUID generation | Manual byte assembly with `crypto/rand` | `uuid.New()` from `github.com/google/uuid` | Already a direct dependency; `uuid.New()` uses `crypto/rand` internally and handles RFC 4122 version/variant bits correctly |
| PTY creation & resize | Raw `ioctl` syscalls | `pty.StartWithSize(cmd, winsize)` and `pty.Setsize(ptmx, winsize)` from `github.com/creack/pty` | Handles OS-specific ioctls, winsize struct layout, and platform differences; already battle-tested in `terminal_unix.go` |
| Process group kill | Iterating `/proc` or manual signal dispatch | Existing `killProcessGroup(cmd)` in `terminal_unix.go` | Already handles SIGHUP → 2s wait → SIGKILL escalation on Unix, `taskkill /F /T` on Windows |
| Event batching & throttling | Custom batching logic | Existing `startEmitter` pattern (8ms ticker, 32KB flush threshold) | Already tested, handles backpressure via non-blocking channel send + `droppedCount` |
| Mutex-protected map | Custom concurrent map implementation | `sync.RWMutex` + built-in `map` | Standard Go pattern; no need for `sync.Map` (which has weaker type safety and is optimized for different access patterns) |
| Shell detection | Hardcoded shell paths | Existing `detectShell()` function (`terminal_service.go:49-65`) | Already handles platform detection ($SHELL → /bin/sh on Unix, pwsh → powershell → cmd on Windows) |

**Key insight:** This phase is a refactoring — not a greenfield build. Every core capability (PTY start/resize/kill, shell detection, output batching, exit monitoring, mutex patterns) already exists in `terminal_service.go`. The engineering work is extraction and multiplexing, not invention.

## Runtime State Inventory

> This section is required because Phase 21 refactors the `TerminalService` singleton into a multi-session manager — a significant internal restructure.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — no SQLite table for terminal sessions exists. Phase 21 is in-memory only per D-06. | None |
| Live service config | `terminalSvc *TerminalService` package-level global in `app.go:17` — set by `ServiceStartup` on `terminal_service.go:226`. Referenced by `execution_service_test.go:177-179` and indirectly via Wails service registry in `main.go:20`. | Retain `terminalSvc` global variable; its type remains `*TerminalService` but the struct's internal fields change. Update `execution_service_test.go:TestTerminalService_ServiceStartupAssignsTerminalSvc` to reflect new singleton role (session manager, not single-PTY). |
| OS-registered state | None — no LaunchAgents, systemd units, or task scheduler entries for terminal sessions. Verified: macOS only, no OS registrations found via grep of repo. | None |
| Secrets/env vars | None — no SOPS keys, `.env` files, or environment variables reference terminal session IDs or PTY configuration. Terminal uses `os.Getenv("SHELL")` in `detectShell()` which is unchanged. | None |
| Build artifacts | **Frontend bindings** at `frontend/bindings/cmdex/terminalservice/` (generated by `wails3 generate build-assets`) — must be regenerated after Go method signatures change (all methods gain `sessionId` parameter). **No Go build artifacts** to clean — standard `go build` handles recompilation. | Run `wails3 generate build-assets` (or `wails3 dev`) after modifying Go method signatures. Bindings regeneration is automatic — never hand-edit `frontend/bindings/`. |

**Runtime state references to update:**

1. **`event_service.go:16-29`** — Remove `PtyOutput`, `PtyExit`, `PtyCleared` fields from `EventNames` struct and `eventNames` package var. Per D-04, these global event names are removed entirely. Session-keyed events are constructed as `"pty-output:" + sessionId` strings at call sites.

2. **`frontend/src/wails/events.ts:8-10,22-24`** — Remove `ptyOutput`, `ptyExit`, `ptyCleared` from fallback `eventNames` object and from `initEventNames()` assignment. Frontend will construct session-specific event names directly from `activeSessionId` (Phase 23 concern, but the constants must be cleaned up in Phase 21).

3. **`frontend/src/components/Terminal.tsx:265-288`** — Event subscriptions use `eventNames.ptyOutput` etc. These must change to use session-keyed event names. Phase 21 scope: the `Terminal.tsx` component's import of `eventNames.ptyOutput`/`ptyExit`/`ptyCleared` references will break when those fields are removed. **Planner must add a task to update Terminal.tsx for namespaced events, or defer the Terminal.tsx update to Phase 23 (noting that the component will be temporarily broken between phases).** Research recommends: update Terminal.tsx in Phase 21 to accept a `sessionId` prop and subscribe to namespaced events — this avoids a broken intermediate state.

4. **`execution_service_test.go:176-188`** — `TestTerminalService_ServiceStartupAssignsTerminalSvc` tests the singleton assignment. Update to verify the session map is non-nil and a default session is created after `ServiceStartup`.

**Nothing found in category:** Stored data, OS-registered state, Secrets/env vars — all verified empty through codebase grep and knowledge of Phase 21 scope.

## Common Pitfalls

### Pitfall 1: Concurrent Map Access Panic
**What goes wrong:** `fatal error: concurrent map read and map write` — Go runtime crash when one goroutine reads `s.sessions` while another writes to it (e.g., `CreateSession` runs while `ListSessions` reads the map).
**Why it happens:** Go maps are not safe for concurrent access. The existing `TerminalService` used `sync.Mutex` for single-instance state; the multi-session map is newly shared across method calls.
**How to avoid:** Use `sync.RWMutex` consistently. Every access to `s.sessions` or `s.activeSessionID` must be under lock. Audit: write methods (`CreateSession`, `CloseSession`, `RenameSession`, `SetActiveSession`) use `Lock()`, read methods (`ListSessions`, `GetActiveSession`, `GetSession`, `Write`/`Resize`/`Clear`/`Start` when they look up a session by ID) use `RLock()`. NEVER hold a read lock while calling a method that acquires a write lock (deadlock).
**Warning signs:** Tests passing individually but failing under `go test -race` or `go test -count=10`. The `-race` flag MUST be used during testing (see Validation Architecture).

### Pitfall 2: Mutex Held Across Blocking Operations
**What goes wrong:** Deadlock — e.g., holding `TerminalService.mu.Lock()` while calling `ss.Stop()` which waits on `ss.readerWg.Wait()`, but the reader goroutine is trying to emit an event that accesses `TerminalService.sessions`.
**Why it happens:** The existing `startLocked` method demonstrates the correct pattern: unlock `s.mu` before closing old PTY files and waiting on `readerWg.Wait()`. This pattern must be preserved when extracting per-session logic.
**How to avoid:** Always release `TerminalService.mu` before any blocking operation (`Wait()`, channel send, file close). Snapshot the session pointer under lock, then release, then operate on the snapshot. The sessionState's own `mu` protects its internal state independently.
**Warning signs:** App hangs on session close or create. `go test -timeout 10s` — any test exceeding 10 seconds indicates a deadlock. goroutine stack traces will show `sync.(*RWMutex).Lock` blocking.

### Pitfall 3: Unlock-Before-Blocking Pattern Omission
**What goes wrong:** Race condition where a new PTY is started before the old PTY's reader goroutine has fully exited, causing two goroutines reading the same `*os.File` descriptor — producing interleaved/corrupted output.
**Why it happens:** The existing code has `readerWg.Add(1)` before starting `readLoop`, `readerWg.Wait()` after closing the old PTY. This guarantee must be maintained per-session.
**How to avoid:** The `startSessionLocked` method for a session MUST follow the unlock-before-close pattern exactly as in `startLocked`:
```go
ss.mu.Unlock()                          // release session mutex
oldPtmx.Close()                         // close old PTY fd
killProcessGroup(oldCmd)                // kill old process group
ss.readerWg.Wait()                      // wait for old reader to exit
// ... then create new PTY, start new reader
ss.mu.Lock()
```
**Warning signs:** `go test -race` detects concurrent reads on `*os.File`. Output from session shows garbled/interleaved text.

### Pitfall 4: Event Name Mismatch Between Frontend and Backend
**What goes wrong:** Backend emits `"pty-output:abc-123"` but frontend listens on `"pty-output:xyz-456"` — output silently disappears (no error, no crash, just no visible output).
**Why it happens:** The event name is constructed dynamically from `sessionId`. If the frontend doesn't use the exact same session ID string, the subscription won't match.
**How to avoid:** The `SessionInfo` struct returned from `CreateSession` and `GetActiveSession` includes the exact `id` string. Frontend uses this exact string to construct event names. Add a backend `GetEventNames(sessionId string)` helper that returns `{"pty-output:"+sessionId, ...}` — or have the frontend construct the names using a utility function: `const ptyOutputEvent = (id: string) => 'pty-output:' + id;`
**Warning signs:** Terminal shows no output after creating a session. Check: backend logs show `wailsApp.Event.Emit` being called; frontend subscription uses correct event name. Log event names from both sides during dev.

### Pitfall 5: `ServiceShutdown` Leaves Sessions Running
**What goes wrong:** `ServiceShutdown` calls `Stop()` on only one session, leaving other PTYs and goroutines orphaned — zombie processes and goroutine leaks.
**Why it happens:** The current `ServiceShutdown` (`terminal_service.go:235-239`) only calls `s.Stop()` on the singleton. After refactoring, it must iterate all sessions.
**How to avoid:**
```go
func (s *TerminalService) ServiceShutdown() error {
    s.mu.Lock()
    sessionIDs := make([]string, 0, len(s.sessions))
    for id := range s.sessions {
        sessionIDs = append(sessionIDs, id)
    }
    s.mu.Unlock()
    
    for _, id := range sessionIDs {
        s.CloseSession(id) // CloseSession calls ss.Stop() internally
    }
    return nil
}
```
**Warning signs:** After app quit, `ps aux | grep bash` shows orphaned shell processes. `go test -count=1` goroutine leak detector (or manual inspection) shows outstanding goroutines.

### Pitfall 6: Active Session Becomes Invalid (closed session)
**What goes wrong:** User closes the active session → `activeSessionID` points to a deleted session → `Write`/`Resize` on active session fails silently or panics.
**Why it happens:** The session is removed from the map but `activeSessionID` is not updated.
**How to avoid:** In `CloseSession`, after removing the session from the map, check if it was the active session. If so, set `activeSessionID` to another session if any remain, or empty string if none.
```go
if s.activeSessionID == id {
    s.activeSessionID = ""
    for otherID := range s.sessions {
        s.activeSessionID = otherID
        break
    }
}
```
**Warning signs:** After closing the active tab, frontend sends `Write("", data)` (empty session ID → falls back to active) and gets "session not found" error.

## Code Examples

Verified patterns from official sources and existing codebase:

### CreateSession with PTY Start

```go
// Source: Derived from existing terminal_service.go ServiceStartup + startLocked pattern,
// adapted for per-session UUID-based creation per CONTEXT.md D-01/D-02.
func (s *TerminalService) CreateSession() (*SessionInfo, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    s.sessionCounter++
    name := fmt.Sprintf("Terminal %d", s.sessionCounter)
    id := uuid.New().String()
    
    ss := &sessionState{
        id:        id,
        name:      name,
        createdAt: time.Now(),
    }
    
    // startSessionLocked uses existing startLocked logic with ss receiver
    if err := s.startSessionLocked(ss, 80, 24); err != nil {
        return nil, fmt.Errorf("CreateSession: PTY start failed: %w", err)
    }
    
    s.sessions[id] = ss
    if s.activeSessionID == "" {
        s.activeSessionID = id
    }
    
    return ss.info(), nil
}
```

### Namespaced Event Emission (per-session emitter)

```go
// Source: Adapted from terminal_service.go:78-89 (flush function in startEmitter),
// namespaced per CONTEXT.md D-04. Verified: Context7 Wails v3 docs (app.Event.Emit).
func (ss *sessionState) emitFlush() {
    if ss.outputBuf.Len() == 0 {
        return
    }
    seq := atomic.AddUint64(&ss.outputSeq, 1)
    if wailsApp != nil {
        wailsApp.Event.Emit("pty-output:"+ss.id, map[string]interface{}{
            "data": ss.outputBuf.String(),
            "seq":  seq,
        })
    }
    ss.outputBuf.Reset()
}
```

### Session-Safe Write Method

```go
// Source: Adapted from terminal_service.go:339-361 (Write method),
// dispatching by sessionId per CONTEXT.md D-01/D-03.
func (s *TerminalService) Write(sessionId string, data string) error {
    ss, err := s.resolveSession(sessionId)
    if err != nil {
        return err
    }
    
    ss.mu.Lock()
    defer ss.mu.Unlock()
    
    if !ss.running {
        if err := s.startSessionLocked(ss, int(ss.lastSize.Cols), int(ss.lastSize.Rows)); err != nil {
            return err
        }
    }
    
    if ss.ptmx == nil {
        return fmt.Errorf("terminal not started")
    }
    
    b := []byte(data)
    for len(b) > 0 {
        n, err := ss.ptmx.Write(b)
        if err != nil {
            return err
        }
        b = b[n:]
    }
    return nil
}
```

### SessionInfo Struct for Frontend Consumption

```go
// Source: CONTEXT.md the agent's Discretion (fields required for Phase 23 tab UI).
// JSON tags use camelCase for Wails binding serialization (project convention, CONVENTIONS.md).
type SessionInfo struct {
    ID         string `json:"id"`
    Name       string `json:"name"`
    Running    bool   `json:"running"`
    ShellPath  string `json:"shellPath"`
    WorkingDir string `json:"workingDir"`
}

func (ss *sessionState) info() *SessionInfo {
    return &SessionInfo{
        ID:         ss.id,
        Name:       ss.name,
        Running:    ss.running,
        ShellPath:  ss.shellPath,
        WorkingDir: ss.workingDir,
    }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Single global `TerminalService` with one PTY | `TerminalService` as session manager with `map[string]*sessionState` | Phase 21 | Backend supports multiple concurrent terminal sessions |
| Global events (`pty-output`, `pty-exit`, `pty-cleared`) | Namespaced per session (`pty-output:{sessionId}`, etc.) | Phase 21 | Output routes to correct frontend terminal; clean break, no fallback |
| `terminalSvc` package-level singleton | Retained but internal structure changes — same type, different fields | Phase 21 | Minimal external API disruption; `execution_service.go` references remain valid |
| `sync.Mutex` for single-instance state | `sync.RWMutex` for session map + per-session `sync.Mutex` for PTY state | Phase 21 | Concurrent reads of session list possible; fine-grained per-session locking |

**Deprecated/outdated:**
- `EventNames.PtyOutput`, `EventNames.PtyExit`, `EventNames.PtyCleared` — replaced by dynamic `"pty-output:" + sessionId` strings. Remove from `event_service.go`.
- Global `TerminalService.Start()`, `Stop()`, `Write()`, `Resize()`, `Clear()` without `sessionId` param — replaced by methods accepting `sessionId string` as first parameter.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `uuid.New()` from `github.com/google/uuid` v1.6.0 uses `crypto/rand` as entropy source (matching D-02 requirement) | Standard Stack | If the package switches to `math/rand` in a future version, UUIDs would be predictable — but this is extremely unlikely for a security-sensitive package. Verified: package docs state v4 uses `crypto/rand`. |
| A2 | Frontend `Terminal.tsx` update for namespaced events is in Phase 21 scope | Runtime State Inventory | If deferred to Phase 23, the component breaks between phases (references deleted `eventNames.ptyOutput`). Planner should either include in Phase 21 or document the temporary breakage. |
| A3 | No other Go files reference `terminalSvc` beyond those found by grep | Runtime State Inventory | If missed, compilation may fail or behavior may change silently. grep confirmed all references in `app.go`, `terminal_service.go`, `execution_service_test.go`. |
| A4 | `detectShell()` call per session is correct — all sessions use the same shell | Standard Stack | Future feature may want per-session shell override. Current design is correct per discussion scope. |
| A5 | Default session counter starting at 1 and never resetting is acceptable (per discretion) | Architecture Patterns | If user expects counter to reset on app restart, Phase 22 persistence would need to save/resume counter. In-memory counter resets with app — this is expected behavior per D-06. |

## Open Questions

1. **Should Terminal.tsx be updated in Phase 21 or deferred to Phase 23?**
   - What we know: Removing `PtyOutput`/`PtyExit`/`PtyCleared` from `EventNames` breaks `Terminal.tsx:265-288`. The component imports `eventNames.ptyOutput` etc.
   - What's unclear: Whether Phase 21 should update Terminal.tsx to accept `sessionId` and subscribe to namespaced events, or leave it broken until Phase 23.
   - Recommendation: Include a minimal Terminal.tsx update in Phase 21 — add `sessionId` prop, subscribe to namespaced events, keep a single TerminalComponent instance for backward compatibility until Phase 23 adds TerminalTabs. This avoids broken intermediate state.

2. **What is the `getWorkingDir()` value for new sessions?**
   - What we know: `SessionInfo.WorkingDir` is required per the agent's Discretion. The existing codebase doesn't have a global working directory for terminal sessions.
   - What's unclear: Whether to use the app's `DefaultWorkingDir` from `AppSettings`, the current OS home directory, or leave empty.
   - Recommendation: Use OS home directory (`os.UserHomeDir()`) as fallback for Phase 21. Phase 22 can refine to use the full fallback chain (per-command → global default → OS home).

3. **Should `TerminalService` expose `sessionState` types to the frontend?**
   - What we know: Wails bindings require exported types. `SessionInfo` is exported. `sessionState` is internal.
   - What's unclear: Whether the Wails binding generator will correctly handle the new method signatures with `sessionId string` parameter and `*SessionInfo` return type.
   - Recommendation: Export `SessionInfo` struct with JSON tags. Keep `sessionState` unexported (internal). All public methods return `*SessionInfo` or `[]*SessionInfo`. Test with `wails3 generate build-assets` after implementation.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go | Build & run | ✓ | 1.25.8 | — |
| wails3 CLI | Frontend binding regeneration (`wails3 generate build-assets`) | ✓ | at `~/go/bin/wails3` | Can use `wails3 dev` which auto-regenerates |
| `github.com/creack/pty` | Unix PTY operations | ✓ | v1.1.24 | — (no alternative; core dependency) |
| `github.com/google/uuid` | UUID v4 session ID generation | ✓ | v1.6.0 | — (no alternative; core dependency) |
| `github.com/wailsapp/wails/v3` | Desktop framework, events, service lifecycle | ✓ | v3.0.0-alpha.74 | — (project runtime) |
| macOS (Darwin) | Unix PTY support | ✓ | Darwin/arm64 | Linux also supported; Windows is stubbed |
| Shell (`/bin/zsh` or `/bin/sh`) | PTY shell process | ✓ | `/bin/zsh` | `detectShell()` falls back to `/bin/sh` |

**Missing dependencies with no fallback:** None
**Missing dependencies with fallback:** None

*All dependencies are satisfied. This phase operates entirely within the existing Go + Wails stack with no external services or tools required beyond what's already installed.*

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go `testing` package (stdlib, no testify) |
| Config file | None — tests use Go's standard `_test.go` convention |
| Quick run command | `go test -race -count=1 -run 'TestTerminal' -v ./...` |
| Full suite command | `go test -race -count=1 -v ./...` |
| Race detector | **Required** — `-race` flag mandatory for all terminal session tests |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SESS-01 | `CreateSession` returns `*SessionInfo` with valid UUID, default name "Terminal N", running=true | unit | `go test -run TestTerminalService_CreateSession -v ./...` | ❌ Wave 0 |
| SESS-01 | `ListSessions` returns all created sessions | unit | `go test -run TestTerminalService_ListSessions -v ./...` | ❌ Wave 0 |
| SESS-04 | `RenameSession(id, "new-name")` updates name, succeeds for non-empty, returns error for empty | unit | `go test -run TestTerminalService_RenameSession -v ./...` | ❌ Wave 0 |
| SESS-05 | `CloseSession(id)` cleans up PTY, process group, removes from map | unit | `go test -run TestTerminalService_CloseSession -v ./...` | ❌ Wave 0 |
| EXEC-04 | Long-running process in session A continues after switching active session to B | integration | `go test -run TestTerminalService_ProcessPersistAcrossSessionSwitch -v ./...` | ❌ Wave 0 |
| EXEC-04 | Output from session A does not appear in session B's output channel | unit | `go test -run TestTerminalService_OutputIsolation -v ./...` | ❌ Wave 0 |
| — | Namespaced events fire with correct session ID in event name | unit | `go test -run TestTerminalService_NamespacedEvents -v ./...` | ❌ Wave 0 |
| — | `ServiceShutdown` closes all sessions (no zombie processes) | unit | `go test -run TestTerminalService_ShutdownCleansAll -v ./...` | ❌ Wave 0 |
| — | Concurrent session create/list/close does not panic (race detector) | unit | `go test -race -run TestTerminalService_ConcurrentAccess -v ./...` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test -race -count=1 -run '<current task test name>' -v ./...`
- **Per wave merge:** `go test -race -count=1 -v ./...` (full suite)
- **Phase gate:** Full suite green with `-race` before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `terminal_service_test.go` — expand with multi-session test cases (currently has single-session tests)
- [ ] `execution_service_test.go` — update `TestTerminalService_ServiceStartupAssignsTerminalSvc` for new session manager role
- [ ] Framework: Add `go test -race` to `make check` or document as required manual step
- [ ] Test helper: `newTestSessionState(t)` — creates a sessionState with a mock PTY for unit testing PTY lifecycle

**Existing test files to update:**
- `terminal_service_test.go` (10 tests, all single-session) — add multi-session tests, do not remove existing tests (they validate the extracted patterns still work)
- `execution_service_test.go:176-188` — update singleton assignment test

## Security Domain

> Security enforcement is enabled (no config.json `security_enforcement: false` found). However, this is a local desktop application with no network surface, authentication, or multi-user access. ASVS applicability is minimal.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | Local-only desktop app; no user authentication |
| V3 Session Management | No | Terminal "sessions" are PTY shells, not web sessions; no session tokens |
| V4 Access Control | No | Single-user desktop app; no access control boundaries |
| V5 Input Validation | Yes (lightweight) | Session name must be non-empty (D-07); PTY write data is passed verbatim to shell (by design) |
| V6 Cryptography | No | UUID v4 is not cryptographic; no encryption needed for in-memory session state |

### Known Threat Patterns for Go + PTY Desktop App

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Command injection via PTY Write | Tampering | PTY input is user-controlled by design (interactive terminal). No sanitization — this is expected behavior for a terminal emulator. Frontend keystrokes are forwarded as-is. |
| Zombie process accumulation | Denial of Service | `ServiceShutdown` must iterate all sessions and kill all process groups (see Pitfall 5) |
| Goroutine leak on rapid session create/close | Denial of Service | `WaitGroup.Wait()` on `readerWg` and `emitterWg` in session `Stop()`. `go test -race` will detect leaked goroutines. |
| Session ID enumeration | Information Disclosure | UUID v4 is not enumerable; session IDs are only exposed to the frontend (same process) |

## Sources

### Primary (HIGH confidence)
- [Context7 /creack/pty] — PTY API: `pty.StartWithSize`, `pty.Setsize`, `pty.Winsize` struct, `pty.InheritSize` [VERIFIED: Context7 — official library docs]
- [Context7 /websites/v3_wails_io] — Wails v3 Event emission: `app.Event.Emit(eventName, data)`, `application.Get()` global access [VERIFIED: Context7 — official Wails v3 documentation]
- [Codebase: terminal_service.go] — Existing PTY lifecycle: `startLocked`, `stopLocked`, `readLoop`, `monitorExit`, `startEmitter`, `stopEmitter`, `enqueueOutput`, `detectShell`, unlock-before-blocking patterns [VERIFIED: codebase grep]
- [Codebase: terminal_unix.go] — `ptyStart` using `pty.StartWithSize`, `ptyResize` using `pty.Setsize`, `killProcessGroup` with SIGHUP→SIGKILL escalation [VERIFIED: codebase grep]
- [Codebase: event_service.go + app.go] — `EventNames` struct pattern, `eventNames` package var, `wailsApp` global, `terminalSvc` global [VERIFIED: codebase grep]

### Secondary (MEDIUM confidence)
- [go.mod] — Confirmed dependency versions: `github.com/creack/pty v1.1.24`, `github.com/google/uuid v1.6.0`, `github.com/wailsapp/wails/v3 v3.0.0-alpha.74` [CITED: go.mod]
- [npm registry] — `@xterm/xterm` v6.0.0, addons at latest versions [VERIFIED: npm view]
- [CONTEXT.md] — All locked decisions (D-01 through D-07), the agent's discretion areas, and deferred ideas [CITED: .planning/phases/21-backend-session-foundation/21-CONTEXT.md]
- [REQUIREMENTS.md] — Phase requirements SESS-01, SESS-04, SESS-05, EXEC-04 [CITED: .planning/REQUIREMENTS.md]
- [.planning/research/PITFALLS.md] — Global state collision, event routing breakage, race on session close [CITED: research document]

### Tertiary (LOW confidence)
- None — all claims validated against codebase, Context7, or official documentation.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all dependencies already in go.mod, versions verified, no new packages
- Architecture: HIGH — extraction pattern is mechanical (moving fields from one struct to another), locked decisions unambiguous
- Pitfalls: HIGH — identified from examining existing mutex patterns and concurrent map access risks

**Research date:** 2026-06-10
**Valid until:** 2026-07-10 (stable; PTY and Wails APIs are mature, no breaking changes expected)
