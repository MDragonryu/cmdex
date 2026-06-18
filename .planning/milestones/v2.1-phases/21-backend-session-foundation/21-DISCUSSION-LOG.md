# Phase 21: Backend Session Foundation - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-09
**Phase:** 21-backend-session-foundation
**Areas discussed:** Active session tracking, Session ID/naming, Event namespacing, PTY lifecycle + integration, Auto-restart, In-memory vs persistence, Error handling

---

## Active Session Tracking

| Option | Description | Selected |
|--------|-------------|----------|
| Backend owns active session | SessionService has SetActiveSession/GetActiveSession. Frontend calls SetActiveSession(id) on tab click. RunCommand without sessionId uses GetActiveSession(). | |
| Frontend owns active session | Frontend tracks activeSessionId in state. RunCommand always passes sessionId explicitly. Backend has no active session concept. | |
| Hybrid: Backend owns, frontend caches | Backend has SetActiveSession/GetActiveSession as source of truth. Frontend caches activeSessionId locally for immediate UI updates, syncs on startup. RunCommand prefers explicit sessionId, falls back to backend GetActiveSession(). | ✓ |

**User's choice:** Hybrid: Backend owns, frontend caches
**Notes:** Backend is source of truth. Frontend caches for instant UI. RunCommand supports both explicit and fallback sessionId.

---

## Session ID & Naming Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| UUID v4 + "Terminal N" | Session ID = UUID v4 (crypto/rand). Default name = "Terminal 1", "Terminal 2" (incremental counter). Rename allows any string. | ✓ |
| Incremental ID + "Terminal N" | Session ID = simple incrementing integer. Default name matches ID. Simpler but not globally unique. | |
| UUID v4 + custom default | Session ID = UUID v4. Default name = "Session-{shortUUID}". | |

**User's choice:** UUID v4 + "Terminal N"
**Notes:** UUIDs for global uniqueness. Incremental counter for human-readable default names.

---

## Event Namespacing Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Namespaced only, no global | Only emit `pty-output:{sessionId}`, `pty-exit:{sessionId}`, `pty-cleared:{sessionId}`. Frontend subscribes to active session's events, unsubscribes on switch. | ✓ |
| Namespaced + transitional global | Emit BOTH namespaced AND legacy global events during transition. Deprecate global in Phase 23. | |
| Namespaced + global active | Always emit namespaced. Additionally emit global from active session only. | |

**User's choice:** Namespaced only, no global
**Notes:** Clean break from legacy global event names.

---

## PTY Lifecycle Per Session & Integration

| Option | Description | Selected |
|--------|-------------|----------|
| Extract TerminalSession, deprecate TerminalService | New TerminalSession struct with all PTY logic. TerminalService becomes thin wrapper for backward compat. | |
| Refactor TerminalService to be multi-session | Add `sessions map[string]*sessionState` to TerminalService. Manages session lifecycle internally. | ✓ |
| New SessionService, keep TerminalService as-is | Create new SessionService that manages multiple TerminalService instances. | |

**User's choice:** Refactor TerminalService to be multi-session
**Notes:** TerminalService becomes the session manager. No separate SessionService struct. Internal map of sessionStates.

---

## Shell Auto-Restart Per Session

| Option | Description | Selected |
|--------|-------------|----------|
| Yes, per-session restart | Each session follows the same monitorExit logic independently. Shell crash in session A doesn't affect session B. | ✓ |
| No, mark dead on exit | When shell exits, session is marked 'dead'. User must manually restart via UI. | |
| One retry, then dead | Unintentional exit auto-restarts once. If crashes again within 5s, mark dead. | |

**User's choice:** Yes, per-session restart
**Notes:** Preserves current auto-restart behavior, just per-session. monitorExit follows same pattern for each sessionState.

---

## In-Memory vs Persistence Boundaries

| Option | Description | Selected |
|--------|-------------|----------|
| In-memory only, Phase 22 adds DB | Phase 21: sessions in `map[string]*sessionState`. No DB table. Phase 22 adds terminal_sessions table and CRUD. | ✓ |
| DB from Phase 21, Phase 22 extends | Phase 21 includes DB migration and stores sessions. Phase 22 adds restore-on-startup. | |
| In-memory + interface for DB later | Phase 21: in-memory with SessionStore interface designed for future DB swap. | |

**User's choice:** In-memory only, Phase 22 adds DB
**Notes:** Clean separation. Phase 21 builds the session runtime. Phase 22 adds persistence layer on top.

---

## Error Handling Strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Standard: fail on create, force close, restore on write | CreateSession fails if PTY can't start. CloseSession always succeeds (force-kills). Rename always succeeds. Write auto-resumes shell if stopped. | ✓ |
| Close needs frontend confirmation | CloseSession with running process requires frontend confirmation. Backend signals close intent. | |
| Other | | |

**User's choice:** Standard: fail on create, force close, restore on write
**Notes:** CreateSession returns error on PTY failure. CloseSession is destructive (force-kill). Write auto-restarts shell (existing behavior).

---

## the agent's Discretion

- Shell dimensions: 80x24 cols/rows default for new sessions (matches existing ServiceStartup)
- Session counter: lives in TerminalService, starts at 1, never resets (in-memory only in Phase 21)
- SessionInfo struct exposes: id, name, running, shellPath, workingDir (sufficient for Phase 23 tab UI)

## Deferred Ideas

None — all discussion stayed within Phase 21 scope.
