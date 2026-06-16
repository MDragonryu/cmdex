# Cmdex — Project

## What This Is

Cmdex is a cross-platform desktop app for saving, organizing, and executing CLI commands as bash scripts with dynamic variable arguments. Built with Go + Wails v2 (backend) and React + TypeScript + Vite (frontend), with SQLite local storage.

## Core Value

Users can organize commands by project context, execute them with variable placeholders, and share them with the community — all in a clean, customizable interface.

## Current State (v2.1 Shipped)

**Milestone:** v2.1 Terminal Sessions — SHIPPED 2026-06-16
**Tech Stack:** Go + Wails v3 + React 19 + TypeScript + Vite + SQLite (`modernc.org/sqlite`)
**Status:** All phases complete. Multiple terminal sessions with refactored PTY backend, in-memory only (session persistence across restarts is deferred to a future milestone — see v2 deferred requirements).

### v2.1 Features Delivered
- ✅ Multiple terminal sessions with active-session routing (Phases 21-24)
- ✅ Command execution routes directly to `terminalSvc.Write` on the active session (Phase 24)
- ✅ ptyBackend interface with build-tagged darwin (`creack/pty`) and windows (conpty stub) implementations (Phase 25)
- ✅ Darwin-only `mockPtyBackend` for hermetic orchestration tests (Phase 25)
- ✅ Global default working directory inheritance for new sessions with home fallback (Phase 25)
- ✅ 100-cycle CreateSession/CloseSession stress test passing with goroutine drift ≤ 5 (Phase 25)
- ✅ `MaxSessions = 10` resource-exhaustion guard with localized toast (Phase 25)
- ✅ Dead-code sweep — no remnants of `OutputPane`, `cmd-output`, or `RunInTerminal` in source (Phase 25)
- ✅ `GOOS=windows go build ./...` cross-compile verification (Phase 25)
- ✅ Documented Windows conpty runtime gap in CHECKPOINT.md and AGENTS.md (Phase 25)
- ✅ ROADMAP/REQUIREMENTS doc sync — persistence claims removed, PERS-01..04 moved to v2 deferred (Phase 25)

### v1.4 Features Delivered
- ✅ React.memo-wrapped CommandDetail (skip reconciliation on inactive tabs)
- ✅ Per-tab useCallback factory functions keyed by tabId (stable callbacks)
- ✅ Iterated per-tab mounts with CSS display:none toggle
- ✅ onResolvedValuesChange gated to active tab only
- ✅ Draft-based variable fallback for inactive tabs
- ✅ Welcome and loading views preserved across tab switches

### v1.0–v1.3 Features Delivered
- ✅ OSPathMap model for cross-OS working directory storage (Phase 10)
- ✅ Native directory picker via Wails binding (Phase 11)
- ✅ Executor runs commands in resolved working directory with fallback chain (Phase 11)
- ✅ Global default working directory setting in Settings window (Phase 12)
- ✅ Command Editor working directory input with browse/clear (Phase 13)
- ✅ UI transparency — no OS keys or JSON ever exposed (Phase 13)
- ✅ Responsive sidebar (auto-collapse at ≤600px)
- ✅ Inline delete confirmation (no modals)
- ✅ 150ms transitions (tabs, sidebar, output pane)
- ✅ Unified script block (template/preview toggle)
- ✅ Theme engine (8 built-in themes, custom colors)
- ✅ OS dark/light preference sync
- ✅ Font selection (7 bundled fonts)
- ✅ Layout density (compact/comfortable/spacious)
- ✅ Import/Export (JSON with variables/presets)
- ✅ Settings as separate Wails window
- ✅ Per-file database migration pattern with rollback

### Technical Debt

- No automated tests exist (manual verification only)

## Next Milestone: TBD

The v2.1 Terminal Sessions milestone is complete. The next direction is not yet decided — candidates include:
- **v2.2 Persistence** — Implement PERS-01..PERS-06 (session restoration across restarts, scrollback history, command history)
- **v2.0 Workspaces** — Named project contexts with sidebar switcher, cloud sync, OAuth, command sharing
- **Windows conpty runtime** — Replace the conpty stub in `pty_backend_windows.go` with a real implementation and add a `windows-latest` CI runner (per CHECKPOINT.md Future Work)

Use `/gsd-add-phase` or `/gsd-new-milestone` to plan the next milestone.

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

## Current Milestone: v2.1 Terminal Sessions

**Goal:** Enable multiple terminal sessions where commands execute on the user's active (selected) session, allowing long-running CLI processes to run alongside other commands.

**Target features:**
- Multiple terminal tabs/sessions
- Execute command on active session
- Long-running process support (servers, watchers, tails persist)
- Session management UI (create/close/rename, status)

## Next Milestone Goals (v2.0)

- **Workspaces** — Named project contexts with sidebar switcher
- **Cloud sync** — Cloudflare Workers + D1 + R2 backend
- **OAuth** — Google + GitHub sign-in
- **Command sharing** — Generate shareable links

## Out of Scope

- Real-time collaboration (sync is eventual, not live)
- Mobile app (desktop-first)
- Email/password auth (OAuth-only)
- Team/organization management (personal tool)

## Constraints

- Cloud: Cloudflare services (Workers + D1 + R2)
- Auth: OAuth only (Google/GitHub)
- Desktop: Wails v2

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

---

*Last updated: 2026-06-16 — v2.1 Terminal Sessions milestone complete (Phase 25 polish-integration shipped)*