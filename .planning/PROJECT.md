# Cmdex — Project

## What This Is

Cmdex is a cross-platform desktop app for saving, organizing, and executing CLI commands as bash scripts with dynamic variable arguments. Built with Go + Wails v3 (backend) and React 19 + TypeScript + Vite (frontend), with SQLite local storage (`modernc.org/sqlite`). Supports multiple persistent terminal sessions where commands execute on the user's active session.

## Core Value

Users can organize commands by project context, execute them with variable placeholders, run long-running processes in dedicated terminal sessions alongside other commands, and share them with the community — all in a clean, customizable interface.

## Current State (v2.1 Shipped)

**Milestone:** v2.1 Terminal Sessions — SHIPPED 2026-06-16
**Tech Stack:** Go + Wails v3 + React 19 + TypeScript + Vite + SQLite (`modernc.org/sqlite`)
**Status:** All 4 shippable phases complete (Phase 22 scoped out). Multiple terminal sessions with refactored PTY backend, in-memory only (session persistence across restarts deferred to a future milestone — see v2 deferred requirements).

### v2.1 Features Delivered
- ✅ Multiple terminal sessions with active-session routing (Phases 21, 23, 24)
- ✅ Command execution routes directly to `terminalSvc.Write` on the active session (Phase 24)
- ✅ `ptyBackend` interface with build-tagged darwin (`creack/pty`) and windows (conpty stub) implementations (Phase 25)
- ✅ Darwin-only `mockPtyBackend` for hermetic orchestration tests (Phase 25)
- ✅ Global default working directory inheritance for new sessions with home fallback (Phase 25)
- ✅ 100-cycle CreateSession/CloseSession stress test passing with goroutine drift ≤ 5 (Phase 25)
- ✅ `MaxSessions = 10` resource-exhaustion guard with localized toast (Phase 25)
- ✅ Dead-code sweep — no remnants of `OutputPane`, `cmd-output`, or `RunInTerminal` in source (Phase 25)
- ✅ `GOOS=windows go build ./...` cross-compile verification (Phase 25)
- ✅ Documented Windows conpty runtime gap in CHECKPOINT.md and AGENTS.md (Phase 25)
- ✅ ROADMAP/REQUIREMENTS doc sync — persistence claims removed, PERS-01..04 moved to v2 deferred (Phase 25)

### v1.4 Features Delivered
- React.memo-wrapped CommandDetail (skip reconciliation on inactive tabs)
- Per-tab useCallback factory functions keyed by tabId (stable callbacks)
- Iterated per-tab mounts with CSS display:none toggle
- onResolvedValuesChange gated to active tab only
- Draft-based variable fallback for inactive tabs
- Welcome and loading views preserved across tab switches

### v1.0–v1.3 Features Delivered
- OSPathMap model for cross-OS working directory storage
- Native directory picker via Wails binding
- Executor runs commands in resolved working directory with fallback chain
- Global default working directory setting in Settings window
- Command Editor working directory input with browse/clear
- UI transparency — no OS keys or JSON ever exposed
- Responsive sidebar (auto-collapse at ≤600px)
- Inline delete confirmation (no modals)
- 150ms transitions (tabs, sidebar, output pane)
- Unified script block (template/preview toggle)
- Theme engine (8 built-in themes, custom colors)
- OS dark/light preference sync
- Font selection (7 bundled fonts)
- Layout density (compact/comfortable/spacious)
- Import/Export (JSON with variables/presets)
- Settings as separate Wails window
- Per-file database migration pattern with rollback

### Technical Debt
- No frontend tests exist (manual verification only) — terminal UI features rely on UAT
- `getWorkingDir()` and `execution_service.go:resolveWorkingDir` are parallel implementations of the same contract (no shared helper)
- Mock PTY backend is darwin-only (`//go:build darwin`) — orchestration tests do not run on linux/windows CI yet
- Windows conpty runtime not verified (stub implementation; cross-compile only)

## Next Milestone: TBD

The v2.1 Terminal Sessions milestone is complete. The next direction is not yet decided.

Remaining technical work (from Phase 25 CHECKPOINT.md and the v2 deferred requirements):
- **PERS-01..PERS-06** — session persistence across restarts, scrollback history, command history
- **Windows conpty runtime** — replace the conpty stub in `pty_backend_windows.go` with a real implementation and add a `windows-latest` CI runner

Use `/gsd-new-milestone` to plan the next milestone.

---

## Key Decisions (v2.1)

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Multi-session over single-active-terminal | Long-running processes (servers, watchers) should survive command execution in parallel sessions | ✅ Good — delivers core value |
| `sync.RWMutex` for sessions map + per-session `sync.Mutex` | Concurrent-safe CRUD with minimal lock contention; per-session lock isolates PTY goroutine contention | ✅ Good — race detector clean across 19 tests |
| Namespaced events (`pty-output:{sessionId}`) | Avoid global event bus collisions; React subscriptions key by sessionId | ✅ Good — output isolation verified |
| Multi-mount TerminalComponents with `display:none` | Reuses Phase 14 CommandDetailTab pattern; preserves terminal scrollback and xterm state across tab switches | ✅ Good — pattern validated twice |
| Direct in-process `terminalSvc.Write` from RunCommand | No event hop; package-level singleton dispatches to active session | ✅ Good — simpler than cmd-executing event |
| Delete `OutputPane`/`RunInTerminal`/`CmdExecuting` (Phase 24) | All orphan after active-session routing; ~280 lines component + ~170 lines CSS removed | ✅ Good — clean break, no callers |
| `ptyBackend` interface with build-tags (Phase 25) | Separates TerminalService from OS PTY layer; enables hermetic tests on darwin and cross-compile on Windows | ✅ Good — seam in place, mock works |
| `MaxSessions = 10` resource-exhaustion guard | Unbounded session creation could leak FDs/goroutines; cap matches realistic personal-tool usage | ✅ Good — surfaced as localized toast |
| In-memory v2.1, persistence deferred to v2 (D-02/D-03) | Deliver multi-session UX now; persistence is separable concern with non-trivial design (restore order, scrollback vs replay, etc.) | ⚠️ Revisit — users will hit "where did my sessions go" after restart |
| `getWorkingDir()` mirrors `resolveWorkingDir` (visual parallelism) | If one is correct, the other should be too; refactor candidate if a third caller appears | — Pending — no third caller yet |

---

## Previous Milestone: v1.3 Working Directory

**Shipped:** 2026-04-23
**Goal:** Allow users to optionally specify a working directory per command, with a global default fallback, stored persistently and used during execution.

**Delivered:**
- OSPathMap data model for cross-OS working directory storage
- Native directory picker via Wails v3 binding
- Executor fallback chain: command → global default → OS home
- Settings UI for global default working directory
- Command Editor integration with fancy popup UI
- Complete UI transparency (no raw OS keys exposed)

---

## Previous Milestone: v1.2 DB Migration Refactor

**Goal:** Replace the monolithic inline `migrate()` function with a per-file up/down migration pattern — each migration in its own numbered Go file.

**Target features:**
- `migrations/` package with individual numbered files (e.g. `0001_initial.go`)
- Each file has `Up(tx *sql.Tx) error` and `Down(tx *sql.Tx) error`
- Migration runner handles discovery, ordering, and transaction wrapping automatically
- All 9 existing migrations ported to the new format
- `DB.RollbackTo(version int)` for dev/testing rollback

---

## Previous Milestone: v1.1 Build Settings Window

**Goal:** Convert the settings dialog from a popup/modal to a proper application Window using Wails window management.

**Target features:**
- Settings opened as separate application window (not dialog/popup)
- Window management (position, size, minimize, maximize)
- Settings persist and apply in real-time

---

## Out of Scope

- Real-time collaboration (sync is eventual, not live)
- Mobile app (desktop-first)
- Email/password auth (OAuth-only)
- Team/organization management (personal tool)
- Full tmux-style pane management (UI complexity, not requested)
- Background session notifications (complex, low priority)
- Session recording/replay (future enhancement)
- Custom per-session shell configuration (settings scope creep)

## Constraints

- **Desktop framework:** Wails v3 (not v2)
- **Frontend:** React 19 + TypeScript + Vite
- **Database:** SQLite via `modernc.org/sqlite` (pure Go, no CGo)
- **PTY layer (unix):** `creack/pty`
- **PTY layer (windows):** conpty stub (full implementation deferred)
- **State:** Wails v3 service registration (one struct per service, registered as `application.Service`)
- **Architecture:** Reactive props via callback factories keyed by tabId/sessionId

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state
5. Key Decisions table updated with outcomes

---

*Last updated: 2026-06-16 — v2.1 Terminal Sessions milestone archived*
