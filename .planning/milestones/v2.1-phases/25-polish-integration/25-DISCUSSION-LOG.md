# Phase 25: Polish & Integration - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-16
**Phase:** 25-polish-integration
**Areas discussed:** Session persistence approach, Working dir ↔ settings integration, Memory leak verification, Windows conpty verification scope

---

## Session Persistence Approach

| Option | Description | Selected |
|--------|-------------|----------|
| Full DB persistence | Re-enable Phase 22's full persistence (terminal_sessions table, migration 0011, full CRUD persistence) | |
| Minimal active-only persistence | Save only active session ID + last session count + global default cwd | |
| Update success criteria | Replace Phase 25 success criteria to match in-memory reality | ✓ |
| Restore-from-scratch persistence | Save session names + cwds as a 'session preset' list, recreate from scratch | |

**User's choice:** "remove/skip phase 22" — keep persistence out, update Phase 25 success criteria.

| Option | Description | Selected |
|--------|-------------|----------|
| Replace with in-memory criteria | Rewrite the 5 criteria to focus on what Phase 25 actually delivers | ✓ |
| Keep criteria + add deferral note | Keep existing 5 criteria, add a note about Phase 22 unskip | |
| Remove persistence criteria | Delete items 1, 2, 3 from the criteria list | |

**User's choice:** Replace with in-memory criteria.

| Option | Description | Selected |
|--------|-------------|----------|
| Use proposed 5 | (1) memory-leak, (2) conpty, (3) cwd inheritance, (4) dead-code cleanup, (5) error states | ✓ |
| Trim to 3 | Drop 'dead-code cleanup' and 'error states' | |
| Add automated tests criterion | Add a 6th criterion for automated tests | |

**User's choice:** Use proposed 5.

| Option | Description | Selected |
|--------|-------------|----------|
| Move PERS to v2 | Move PERS-01..PERS-04 to the v2 Requirements section | ✓ |
| Add Out of Scope note | Keep in v1, add deferral note | |
| Leave as-is | Leave REQUIREMENTS.md alone, note in CONTEXT.md | |

**User's choice:** Move PERS to v2.

---

## Working Dir ↔ Settings Integration

| Option | Description | Selected |
|--------|-------------|----------|
| Inherit global default at creation | Backend CreateSession reads global default cwd from settings (OS-keyed) | ✓ |
| Always start at OS home | Ignore global default, always start at OS user home | |
| Copy from active session | New session's cwd = active session's cwd at creation | |
| Frontend picks cwd at creation | Frontend prompts user for cwd with a 'use default' option | |

**User's choice:** Inherit global default at creation. Confirmed existing behavior in `terminal_service.go`.

| Option | Description | Selected |
|--------|-------------|----------|
| Empty → OS home | Fall back to OS user home ($HOME / %USERPROFILE%) | ✓ |
| Empty → shell's default cwd | Use whatever the shell process has at spawn | |
| Empty → show error | Show error and prompt user to set one in Settings | |

**User's choice:** Empty → OS home.

| Option | Description | Selected |
|--------|-------------|----------|
| New sessions only | Default changes affect new sessions only | ✓ |
| Retroactively update all | Send `cd {newDir}` to every running session | |
| Show 'apply to all' prompt | Prompt user when default changes | |

**User's choice:** New sessions only. No retroactive coordination.

| Option | Description | Selected |
|--------|-------------|----------|
| Confirm chain | per-command wd → global default → session cwd (no change) | ✓ |
| Reorder: session cwd takes priority | per-command → session cwd → global default | |
| Reorder: per-command wins, no fallback | per-command → session cwd (drop global default) | |

**User's choice:** Confirm chain. Matches Phase 24's EXEC-03.

---

## Memory Leak Verification

| Option | Description | Selected |
|--------|-------------|----------|
| Go stress test + manual smoke | Go test (100 cycles) + manual xterm smoke | ✓ |
| Manual smoke test only | Open/close 50 sessions, watch Activity Monitor | |
| Runtime pprof + dev menu | Add runtime/pprof HTTP endpoint in dev mode | |
| Add a full test suite | Comprehensive Go + frontend tests with CI | |

**User's choice:** Go stress test + manual smoke.

| Option | Description | Selected |
|--------|-------------|----------|
| Backend lifecycle | Goroutine count stable, sessions map size 0 after each Close, pty FDs closed | ✓ |
| Backend + event channel leak check | Same + assert no events emitted for closed sessions | |
| Full integration | Backend + event channels + Wails event registration | |

**User's choice:** Backend lifecycle.

| Option | Description | Selected |
|--------|-------------|----------|
| Manual xterm smoke | Open 20 tabs, close all, check DOM for orphaned nodes | ✓ |
| Add Playwright for xterm | Playwright test asserting xterm DOM count = 0 | |
| Add xterm dispose() assertions | Explicit dispose() in cleanup useEffect + contract | |

**User's choice:** Manual xterm smoke.

| Option | Description | Selected |
|--------|-------------|----------|
| 100 cycles | Runs <5s, catches most leaks, suitable for default CI | ✓ |
| 1000 cycles | Higher confidence, ~30-60s, separate CI job | |
| 10 cycles (smoke) | Quick smoke test, catches gross leaks only | |

**User's choice:** 100 cycles.

---

## Windows Conpty Verification Scope

| Option | Description | Selected |
|--------|-------------|----------|
| Build-CI + manual Windows runbook | GitHub Actions build matrix + manual Windows runbook | |
| Runtime test in CI | windows-latest runner with full conpty test | |
| Code review + runbook only | Inspect terminal_windows.go, document expected behavior | |
| macos-only with conpty mock | Cross-compile to Windows from darwin, run orchestration tests via darwin-side mock | ✓ |

**User's choice:** macos-only with conpty mock. Pragmatic given darwin-only dev environment.

| Option | Description | Selected |
|--------|-------------|----------|
| Behavior mock in sessionState | Build-tagged interface, darwin + windows impls, darwin-side mock | ✓ |
| No mock, document gap | Just cross-compile, document the gap | |
| Inject fakes for test | Testable interface, inject fakes in tests | |

**User's choice:** Behavior mock in sessionState.

| Option | Description | Selected |
|--------|-------------|----------|
| PTY spawn + I/O + resize | Covers most common paths, skips shell detection and signals | ✓ |
| PTY + I/O + resize + exit detection | Adds the exit-handling path | |
| Full mock (PTY + I/O + resize + exit + signals) | Highest confidence, largest scope | |

**User's choice:** PTY spawn + I/O + resize.

| Option | Description | Selected |
|--------|-------------|----------|
| Document gap in CHECKPOINT + AGENTS.md | Add 'Windows conpty verification' section to CHECKPOINT + note in AGENTS.md | ✓ |
| Inline comments in terminal_windows.go only | Comments flagging untested paths, no doc file | |
| Don't document | Trust cross-compile + mock is sufficient | |

**User's choice:** Document gap in CHECKPOINT + AGENTS.md.

---

## the agent's Discretion

- Form of the `ptyBackend` interface (method names, signatures) — should mirror what `terminal_service.go` actually needs from the OS layer.
- Whether to extract existing `ptyStart` / `ptyResize` / `killProcessGroup` helpers into the interface or keep them as package-level functions.
- Stress test cycle count (100) and goroutine delta threshold — agent can tune.
- Error-state UX details for "PTY start failure" (toast vs. inline) and "max sessions" (silent vs. prompt) — minor UX calls based on existing patterns.
- Exact placement of the dead-code cleanup pass.

## Deferred Ideas

- PERS-01..PERS-04 (session persistence) — moved to v2 Requirements.
- Migration 0011 (`terminal_sessions` table) — not needed in v2.1.
- Full Windows runtime conpty verification — requires a Windows machine.
- Comprehensive test suite (Playwright, frontend unit tests) — project has no test infra.
- PTY backend process group + signal handling in mock — relies on real conpty.
- Migration rollback test for any new schema — no schema changes.
- Session scrollback persistence (PERS-05) — already deferred.
- Shell command history persistence (PERS-06) — already deferred.
- Theming/font/density changes mid-session — not in this phase, CSS variables handle it.
