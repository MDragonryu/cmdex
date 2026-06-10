# Roadmap: Cmdex v2.1 Terminal Sessions

**Milestone:** v2.1
**Name:** Terminal Sessions
**Goal:** Enable multiple terminal sessions where commands execute on the user's active (selected) session, allowing long-running CLI processes to run alongside other commands.

---

## Phase 21: Backend Session Foundation

**Goal:** Backend can manage multiple terminal sessions with isolated PTYs, and long-running processes persist across session switches.

**Depends on:** v2.0 Terminal Integration (phases 16-20 complete)

**Requirements:** SESS-01, SESS-04, SESS-05, EXEC-04

**Success Criteria** (what must be TRUE):
1. Backend can create a new terminal session with a default name and return its session ID
2. Backend can list all active sessions with their metadata (name, status, working directory, shell)
3. Backend can rename a session by ID
4. Backend can close a session by ID, cleaning up its PTY and process group
5. When switching active session, the previous session's PTY process continues running (verify: `sleep 30` in session A, switch to B, session A still running)
6. Namespaced events (`pty-output:{sessionId}`, `pty-exit:{sessionId}`, `pty-cleared:{sessionId}`) route output correctly per session
7. SessionService uses a mutex-protected map — no global state collision

**Plans:** 3 plans

Plans:
- [ ] 21-01-PLAN.md — Session types, struct refactoring, CRUD API (CreateSession, ListSessions, CloseSession, RenameSession, active session)
- [ ] 21-02-PLAN.md — Per-session PTY lifecycle, namespaced events, ServiceStartup/Shutdown
- [ ] 21-03-PLAN.md — Frontend event wiring, Wails bindings, multi-session Go tests

**UI hint**: no

---

## Phase 22: Database Persistence

**Goal:** Terminal sessions persist across app restarts with all metadata restored and active session remembered.

**Depends on:** Phase 21

**Requirements:** PERS-01, PERS-02, PERS-03, PERS-04

**Success Criteria** (what must be TRUE):
1. On app startup, previous sessions are loaded from SQLite and available via SessionService
2. Session metadata (name, working directory, shell) is correctly restored for each session
3. The previously active session is marked active and auto-selected in the tab bar
4. Working directory fallback chain works: per-command → global default → OS home

**Plans:** TBD

**UI hint**: no

---

## Phase 23: Frontend Tabbed Terminal

**Goal:** Users can manage and interact with multiple terminal sessions through a tabbed interface with full keyboard and mouse support.

**Depends on:** Phase 21, Phase 22

**Requirements:** SESS-02, SESS-03, SESS-06, UI-01, UI-02, UI-03, UI-04, UI-05, UI-06

**Success Criteria** (what must be TRUE):
1. User sees a tab bar listing all terminal sessions with names
2. User can switch sessions by clicking tabs — terminal output updates instantly
3. User can reorder tabs via drag-and-drop
4. Each tab shows session name and status indicator (idle/running/busy)
5. Right-clicking a tab shows context menu with rename, close, duplicate options
6. Keyboard shortcuts work: Ctrl+T (new), Ctrl+W (close), Ctrl+Tab (next), Ctrl+Shift+Tab (prev)
7. Each session preserves 5000 lines of scrollback independently
8. Terminal theme matches app theme via CSS variables (no hardcoded colors)
9. Clear button clears only the active session's terminal

**Plans:** TBD

**UI hint**: yes

---

## Phase 24: Session-Aware Execution

**Goal:** Users can execute saved commands in the active terminal session with full variable resolution, working directory support, and real-time output streaming.

**Depends on:** Phase 21, Phase 23

**Requirements:** EXEC-01, EXEC-02, EXEC-03, EXEC-05, EXEC-06

**Success Criteria** (what must be TRUE):
1. Clicking Run on a saved command executes it in the active session's terminal
2. Command variables (CEL defaults, env, prompts) are resolved before sending to session
3. Command working directory is applied (per-command → global default → session cwd)
4. Command output streams to the active session's terminal in real-time with ANSI support
5. User can press Ctrl+C to interrupt a running command in the active session

**Plans:** TBD

**UI hint**: no

---

## Phase 25: Polish & Integration

**Goal:** All session features work cohesively with settings, persistence, edge cases handled, and cross-platform verified.

**Depends on:** Phase 22, Phase 23, Phase 24

**Requirements:** (integrates all prior — no new requirements)

**Success Criteria** (what must be TRUE):
1. Sessions created via UI persist across app restarts and restore correctly (end-to-end)
2. Session working directory integrates with global default setting from Settings window
3. Active session selection persists across restarts
4. No memory leaks: rapid create/close cycles don't leak xterm instances or PTY processes
5. Windows conpty compatibility verified (PTY spawn, resize, I/O, shell detection)

**Plans:** TBD

**UI hint**: no

---

## Progress

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 21. Backend Session Foundation | 0/3 | Not started | - |
| 22. Database Persistence | 0/0 | Not started | - |
| 23. Frontend Tabbed Terminal | 0/0 | Not started | - |
| 24. Session-Aware Execution | 0/0 | Not started | - |
| 25. Polish & Integration | 0/0 | Not started | - |

**Execution order:** 21 → 22 → 23 → 24 → 25 (serial — each phase depends on the prior)

---

*Last updated: 2026-06-09*