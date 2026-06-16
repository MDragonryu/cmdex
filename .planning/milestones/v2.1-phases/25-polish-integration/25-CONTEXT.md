# Phase 25: Polish & Integration - Context

**Gathered:** 2026-06-16
**Status:** Ready for planning

<domain>
## Phase Boundary

Phase 25 polishes the v2.1 Terminal Sessions milestone: verifies session lifecycle doesn't leak, integrates the global default working directory with new session creation, refactors the PTY backend behind a build-tagged interface to enable darwin-side orchestration tests, and confirms the cross-platform conpty path compiles and works against a behavior mock. Sessions remain in-memory only — Phase 22's persistence work stays skipped, and PERS-01 through PERS-04 move to v2 Requirements.

**Phase 25 delivers:**
1. Go stress test for session create/close lifecycle (100 cycles, goroutine + map assertions).
2. Build-tagged `ptyBackend` interface with `darwin` (creack/pty) and `windows` (conpty) implementations, plus a darwin-side behavior mock for orchestration tests.
3. Cross-compile verification of the Windows build (catches API mismatches at compile time).
4. Confirmed and documented global default cwd inheritance for new sessions.
5. Dead-code cleanup from prior phases (OutputPane already removed in Phase 24; verify nothing remains).
6. Error-state handling for: PTY start failure, max sessions, last-session close guard.
7. Documented Windows conpty verification gap in CHECKPOINT + AGENTS.md.

**Not in scope:**
- Session persistence (PERS-01..PERS-04) — moved to v2 Requirements.
- Migration 0011 (`terminal_sessions` table) — not needed in v2.1.
- Full Windows runtime conpty verification — requires a Windows machine.
- Frontend test infrastructure (Playwright etc.) — deferred, project has no test infra.
- New session management capabilities (split panes, session sharing, broadcast) — already in v2 / Out of Scope.

</domain>

<decisions>
## Implementation Decisions

### Session Persistence Scope
- **D-01:** Phase 22 stays skipped. No DB persistence for sessions. Sessions live in `TerminalService.sessions` map and are lost on app restart. This is the explicitly chosen Phase 25 scope, not an oversight.
- **D-02:** Phase 25 success criteria in ROADMAP.md are replaced with the in-memory set (memory-leak, conpty, cwd inheritance, dead-code cleanup, error states). Persistence claims are removed.
- **D-03:** PERS-01, PERS-02, PERS-03, PERS-04 are moved from v1 Requirements to v2 Requirements in REQUIREMENTS.md. Traceability table is updated.

### Working Directory ↔ Settings Integration
- **D-04:** New session inherits the global default cwd from settings at creation time. The cwd is OS-keyed (per the existing `OSPathMap` pattern in `db.go` for the global default). Existing `terminal_service.go` `CreateSession` already reads from settings — confirm and document the contract.
- **D-05:** Empty global default → fall back to OS user home (`$HOME` on Unix, `%USERPROFILE%` on Windows). Session always has a valid cwd to start.
- **D-06:** Changes to the global default cwd (via Settings window) apply to new sessions only. Existing sessions keep their current cwd (no retroactive `cd`). Rationale: shells have already been spawned with a cwd; sending `cd` mid-stream is a side effect the user didn't ask for.
- **D-07:** The Phase 24 EXEC-03 cwd chain (per-command wd → global default → session cwd) is confirmed and documented. No reordering — the per-command setting still wins, then global default, then the session's current directory.

### Memory Leak Verification
- **D-08:** Add `terminal_service_test.go` with a 100-cycle CreateSession/CloseSession stress test. Asserts: (a) `runtime.NumGoroutine()` is stable within a delta after the loop, (b) `TerminalService.sessions` map size is 0 after each Close, (c) PTY file descriptors are closed. Runs in <5s, suitable for default CI.
- **D-09:** Frontend (xterm) leak verification is a documented manual smoke test: open 20 tabs via Ctrl+T, close all via Ctrl+W, inspect the DOM for orphaned xterm nodes. Runbook lives in `.planning/phases/25-polish-integration/RUNBOOK.md` (created during planning).
- **D-10:** No new test framework. The project has no automated tests; this phase adds the first Go test (the stress test) without introducing a test framework, Playwright, or test runner. Just `go test ./...` for this one file.

### Windows Conpty Verification
- **D-11:** Refactor PTY lifecycle into a build-tagged `ptyBackend` interface. `terminal_unix.go` and `terminal_windows.go` (new or modified) implement the interface. This is a refactor for testability, not a feature change.
- **D-12:** Darwin-side behavior mock (in-memory fake) covers PTY spawn, write, read, and resize. Skips shell detection, process group / signal handling, and exit detection — those rely on real conpty.
- **D-13:** Windows verification path is: (1) cross-compile to Windows from darwin (`GOOS=windows go build ./...`), which catches conpty API mismatches at compile time, (2) orchestration tests run against the darwin-side mock, (3) the documented gap is acknowledged in CHECKPOINT and AGENTS.md. No Windows runtime CI in this phase.
- **D-14:** The `CHECKPOINT` document created during planning includes a "Windows conpty verification" section explaining the mock scope, cross-compile coverage, and the real-Windows gap. `AGENTS.md` is updated with a one-line note so future maintainers know the gap exists.

### the agent's Discretion
- Exact form of the `ptyBackend` interface (method names, signatures) — should mirror what `terminal_service.go` actually needs from the OS layer.
- Whether to extract the existing `ptyStart` / `ptyResize` / killProcessGroup helpers into the interface or keep them as package-level functions called by the implementation.
- Stress test cycle count (100) and goroutine delta threshold — agent can tune if the initial run is flaky.
- Error-state UX details for "PTY start failure" (toast vs. inline error) and "max sessions" (silent reject vs. prompt) — these are minor UX calls the implementer should make based on existing patterns in App.tsx.
- Exact placement of the dead-code cleanup pass — could be a standalone plan or woven into the main implementation plan.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Roadmap & Requirements
- `.planning/ROADMAP.md` — Phase 25 entry (criteria need revision during this phase's execution)
- `.planning/REQUIREMENTS.md` — PERS-01..PERS-04 to be moved to v2 section; EXEC-03 cwd chain still in v1
- `.planning/PROJECT.md` — Project context, key decisions
- `.planning/STATE.md` — Current state (Phase 24 complete)

### Prior Phase Context
- `.planning/phases/21-backend-session-foundation/21-CONTEXT.md` — `TerminalService` refactor, session CRUD, namespaced events, active session, monitorExit
- `.planning/phases/22-database-persistence/22-CONTEXT.md` — Phase 22 skipped, persistence deferred
- `.planning/phases/23-frontend-tabbed-terminal/23-CONTEXT.md` — Tab UX, multi-mount TerminalComponent, status indicators, keyboard shortcuts
- `.planning/phases/24-session-aware-execution/24-CONTEXT.md` — RunCommand → active session PTY, removed OutputPane, EXEC-03 cwd chain

### Existing Code (must study before implementing)
- `terminal_service.go` — `TerminalService` struct, `CreateSession` (cwd inheritance), `CloseSession`, `sessionState` map, `monitorExit`, `enqueueOutput`
- `terminal_unix.go` — `ptyStart`, `ptyResize` (creack/pty) — refactor target for `ptyBackend` interface
- `terminal_windows.go` — conpty stubs — to be refactored behind `ptyBackend` interface
- `execution_service.go` — `RunCommand` with EXEC-03 cwd chain (per-command → global default → session cwd)
- `app.go` — Settings window integration, global default cwd read/write
- `db.go` — `OSPathMap` for cross-OS working directory storage; settings CRUD
- `event_service.go` — EventNames struct, namespaced event pattern (`pty-output:{sessionId}`)
- `executor.go` — `killProcessGroup` helper, shell detection patterns
- `models.go` — `SessionInfo`, `Command`, `VariableDefinition` types
- `main.go` — Service registration, Wails service lifecycle
- `frontend/src/App.tsx` — RunCommand call sites, session state, error toast patterns
- `frontend/src/components/Terminal.tsx` — xterm dispose contract, multi-mount via `display: none`

### Reference Documents
- `.planning/codebase/STRUCTURE.md` — Directory layout, where to add new code
- `.planning/codebase/CONCERNS.md` — Test gaps, App.tsx centralization, Windows shell strategy
- `.planning/codebase/ARCHITECTURE.md` — Service patterns, data flow
- `.planning/codebase/CONVENTIONS.md` — Go and TypeScript patterns
- `.planning/codebase/STACK.md` — Library versions, what NOT to add
- `.planning/codebase/TESTING.md` — Existing test patterns (only `db_test.go` for migrations)
- `AGENTS.md` — Add Windows conpty verification gap note here

### New Artifacts This Phase Will Create
- `.planning/phases/25-polish-integration/CHECKPOINT.md` — includes "Windows conpty verification" section
- `.planning/phases/25-polish-integration/RUNBOOK.md` — manual xterm smoke test procedure
- `terminal_service_test.go` — 100-cycle stress test (new file in repo root or `tests/`)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`TerminalService.CreateSession`** (`terminal_service.go`): Already reads global default cwd from settings, falls back to OS home when empty, and stores cwd on the session. D-04..D-07 are confirmations, not new code — verify the existing path and document the contract.
- **`OSPathMap` model** (`models.go`, `db.go`): Cross-OS cwd storage keyed by `darwin`/`windows`/`linux`. Reuse for global default cwd access. Settings layer already supports this.
- **`monitorExit` pattern** (`terminal_service.go:180-223`): Per-session `cmd.Wait()` with intentional/unintentional detection. Refactor target — sessionState's goroutine lifecycle is what the stress test verifies.
- **`TabBar` component** (`frontend/src/components/TabBar.tsx`): Already handles last-tab close guard (Ctrl+W no-op). No new code needed for the "last session close" error state.
- **Toast/notification pattern** (used in `App.tsx` for create/close errors): Reuse for PTY start failure UX.
- **Wails service pattern**: Services implement `ServiceStartup`/`ServiceShutdown`. The new `ptyBackend` interface fits this style — methods on the backend struct, called by `TerminalService` methods.

### Established Patterns
- **Build tags for OS-specific code**: `terminal_unix.go` (`//go:build !windows`) and `terminal_windows.go` (`//go:build windows`) already exist. Extend the pattern with `ptyBackend` implementations.
- **Mutex with unlock-before-blocking**: `s.mu.Lock()` / `s.mu.Unlock()` pattern in TerminalService. The ptyBackend interface methods should not block holding the mutex — release first, then call.
- **CSS variable theming**: All colors via `var(--background)`, `var(--foreground)`, etc. No hardcoded colors anywhere in the terminal UI.
- **Event namespacing**: `pty-output:{sessionId}` format. New events (if any) follow the same pattern.
- **Wails event emission**: `wailsApp.Event.Emit(eventNames.X, ...)`. The new mock backend should emit through the same path (or a test double) so stress tests verify the real emission logic.
- **Ref-based ephemeral state in App.tsx**: `tabOutputRef`, `tabPaneStateRef`. Continue this pattern for new session state — don't introduce React state for things that change frequently.

### Integration Points
- **`terminal_service.go` `CreateSession`**: Confirm cwd inheritance path. If global default is empty, fallback to `os.UserHomeDir()`. The fallback already exists per Phase 21's `ServiceStartup` — verify the same path runs on CreateSession.
- **`terminal_service.go` `CloseSession`**: The stress test asserts this is leak-free. Goroutine tracking happens here — `monitorExit` goroutine and the emitter goroutine must exit cleanly.
- **`App.tsx` session management**: Mounting N `TerminalComponent` instances, hiding inactive via CSS. The xterm dispose contract is what the manual smoke test verifies.
- **`db.go` settings table**: Global default cwd is read via the existing settings service. No DB schema change needed.
- **Wails `ServiceStartup`**: The ptyBackend initialization happens here. On darwin, instantiate the creack/pty backend; on windows, instantiate the conpty backend. The mock is for tests only — never instantiated at runtime.

</code_context>

<specifics>
## Specific Ideas

- The user explicitly picked the "macos-only with conpty mock" approach (D-13) over the more typical "Build-CI + manual Windows runbook" because the dev environment is darwin-only and full Windows verification requires infrastructure the user doesn't have. This is a pragmatic, acknowledged gap — not an oversight.
- The user wants the stress test to be the project's first Go test (other than the existing `db_test.go` for migrations). Adding a test framework (Testify, Ginkgo, etc.) is out of scope — just `go test ./...` with standard library testing.
- The user confirmed the existing cwd-inheritance behavior in `terminal_service.go` is correct (D-04..D-07 are confirmations, not changes). The phase's value-add is documenting the contract and ensuring tests verify it.

</specifics>

<deferred>
## Deferred Ideas

- **PERS-01..PERS-04 (session persistence)** — moved to v2 Requirements. A future phase will design the `terminal_sessions` table, migration 0011, and restore-on-startup logic.
- **Migration 0011 (`terminal_sessions` table)** — not needed in v2.1.
- **Full Windows runtime conpty verification** — requires a Windows machine or windows-latest CI runner. The user accepted the gap explicitly.
- **Comprehensive test suite (Playwright, frontend unit tests)** — project has no test infra. Adding a frontend test framework for one smoke test is over-scoping.
- **PTY backend process group + signal handling in mock** — relies on real conpty. Cross-compile is the only verification.
- **Migration rollback test for any new schema** — no schema changes in this phase (sessions are in-memory), so no rollback needed.
- **Session scrollback persistence (PERS-05)** — already deferred per REQUIREMENTS.md v2.
- **Shell command history persistence (PERS-06)** — already deferred per REQUIREMENTS.md v2.
- **Theming/font/density changes mid-session** — not in this phase. The existing CSS variable system means changes apply automatically; no coordination needed.

</deferred>

---

*Phase: 25-Polish & Integration*
*Context gathered: 2026-06-16*
