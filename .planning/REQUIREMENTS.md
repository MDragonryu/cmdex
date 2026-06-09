# Requirements: Cmdex — Terminal Sessions

**Defined:** 2026-06-08
**Core Value:** Users can run long-running CLI processes in persistent terminal sessions while executing other commands in parallel

## v1 Requirements

Requirements for v2.1 Terminal Sessions milestone. Each maps to roadmap phases.

### Session Management

- [ ] **SESS-01**: User can create a new terminal session with a default name
- [ ] **SESS-02**: User can see a list of all terminal sessions in a tab bar
- [ ] **SESS-03**: User can switch between terminal sessions by clicking tabs
- [ ] **SESS-04**: User can rename a terminal session
- [ ] **SESS-05**: User can close a terminal session (with confirmation if process running)
- [ ] **SESS-06**: User can reorder terminal session tabs via drag-and-drop

### Session Execution

- [ ] **EXEC-01**: User can execute a saved command in the active terminal session
- [ ] **EXEC-02**: Command variables are resolved (CEL defaults, env, prompts) before sending to session
- [ ] **EXEC-03**: Command working directory is applied (per-command → global default → session cwd)
- [ ] **EXEC-04**: Long-running processes (servers, watchers, tail) continue running when user switches sessions
- [ ] **EXEC-05**: Command output streams to the active session's terminal in real-time
- [ ] **EXEC-06**: User can send Ctrl+C to interrupt a running command in the active session

### Session UI

- [ ] **UI-01**: Terminal tabs show session name and status indicator (idle/running/busy)
- [ ] **UI-02**: Right-click tab shows context menu (rename, close, duplicate)
- [ ] **UI-03**: Keyboard shortcuts: New tab (Ctrl+T), Close tab (Ctrl+W), Next tab (Ctrl+Tab), Prev tab (Ctrl+Shift+Tab)
- [ ] **UI-04**: Terminal output preserves scrollback per session (5000 lines)
- [ ] **UI-05**: Session theme matches app theme (CSS variables)
- [ ] **UI-06**: Clear terminal button clears only the active session

### Session Persistence

- [ ] **PERS-01**: Terminal sessions persist across app restarts (name, working dir, shell)
- [ ] **PERS-02**: On startup, previous sessions are restored and available in tab bar
- [ ] **PERS-03**: Session working directory is restored (per-command → global default → OS home fallback)
- [ ] **PERS-04**: Active session is remembered and auto-selected on restart

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Advanced Session Features

- **SESS-07**: User can duplicate a session (clone working dir, env, history)
- **SESS-08**: User can split terminal pane horizontally/vertically (tmux-style)
- **SESS-09**: User can broadcast input to multiple sessions simultaneously

### Session Sharing

- **SESS-10**: User can export session layout as workspace file
- **SESS-11**: User can share session output via link (read-only)

### Enhanced Persistence

- **PERS-05**: Session scrollback history persisted across restarts
- **PERS-06**: Shell command history persisted per session

## Out of Scope

| Feature | Reason |
|---------|--------|
| Real-time collaboration / shared sessions | Multi-user, not personal tool |
| Session recording/replay | Future enhancement |
| Custom per-session shell configuration | Settings scope creep |
| Full tmux-style pane management | UI complexity, not requested |
| Background session notifications | Complex, low priority |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| SESS-01 | Phase 1 | Pending |
| SESS-02 | Phase 1 | Pending |
| SESS-03 | Phase 1 | Pending |
| SESS-04 | Phase 2 | Pending |
| SESS-05 | Phase 2 | Pending |
| SESS-06 | Phase 3 | Pending |
| EXEC-01 | Phase 2 | Pending |
| EXEC-02 | Phase 2 | Pending |
| EXEC-03 | Phase 2 | Pending |
| EXEC-04 | Phase 2 | Pending |
| EXEC-05 | Phase 2 | Pending |
| EXEC-06 | Phase 3 | Pending |
| UI-01 | Phase 3 | Pending |
| UI-02 | Phase 3 | Pending |
| UI-03 | Phase 3 | Pending |
| UI-04 | Phase 3 | Pending |
| UI-05 | Phase 3 | Pending |
| UI-06 | Phase 3 | Pending |
| PERS-01 | Phase 4 | Pending |
| PERS-02 | Phase 4 | Pending |
| PERS-03 | Phase 4 | Pending |
| PERS-04 | Phase 4 | Pending |

**Coverage:**
- v1 requirements: 22 total
- Mapped to phases: 22
- Unmapped: 0 ✓

---
*Requirements defined: 2026-06-08*
*Last updated: 2026-06-08 after initial definition*