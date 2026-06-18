---
phase: 24
slug: session-aware-execution
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-15
---

# Phase 24 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` (standard library). No frontend test runner. |
| **Config file** | None — Go discovers `*_test.go` automatically. |
| **Quick run command** | `go test ./... -run TestRunCommand` |
| **Full suite command** | `go test ./...` + `cd frontend && pnpm tsc --noEmit` |
| **Estimated runtime** | ~10 seconds (PTY-backed tests may add 1–2s) |

---

## Sampling Rate

- **After every task commit:** `go build ./... && cd frontend && pnpm tsc --noEmit`
- **After every plan wave:** `go test ./... -v` (full Go test suite)
- **Before `/gsd-verify-work`:** Full suite must be green + manual UAT of 5 success criteria in `wails3 dev`
- **Max feedback latency:** 10 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 24-01-01 | 01 | 1 | EXEC-01 | T-1 / — | terminalSvc is non-nil and active session exists | Integration | `go test ./... -run TestRunCommand_ExecutesOnActiveSession` | ❌ W0 | ⬜ pending |
| 24-01-02 | 01 | 1 | EXEC-01 | T-1 / — | If terminalSvc nil, return ExecutionRecord{Error: "terminal service not initialized"} | Integration | `go test ./... -run TestRunCommand_NilTerminalSvc` | ❌ W0 | ⬜ pending |
| 24-01-03 | 01 | 1 | EXEC-01 | T-1 / — | If no active session, return ExecutionRecord{Error: "no active session", ExitCode: -1} | Integration | `go test ./... -run TestRunCommand_NoActiveSession` | ❌ W0 | ⬜ pending |
| 24-01-04 | 01 | 1 | EXEC-01, EXEC-02 | T-1, T-2 / — | RunCommand resolves variables then sends to PTY | Integration | `go test ./... -run TestRunCommand_FinalCmdMultilineScript` | ✅ EXISTING | ⬜ pending |
| 24-01-05 | 01 | 1 | EXEC-03 | T-2 / — | Per-command working dir applied via `cd /path && cmd\n` | Unit | `go test ./... -run TestRunCommand_FinalCmdWithWorkingDir` | ✅ EXISTING | ⬜ pending |
| 24-01-06 | 01 | 1 | EXEC-03 | T-2 / — | Fallback to global default / home when no per-command dir | Unit | `go test ./... -run TestRunCommand_FinalCmdNoWorkingDir` | ✅ EXISTING | ⬜ pending |
| 24-01-07 | 01 | 1 | — | — | No `executions` row written after a successful RunCommand | Integration | `go test ./... -run TestRunCommand_NoHistoryPersistence` | ✅ EXISTING | ⬜ pending |
| 24-02-01 | 02 | 1 | — | — | OutputPane.tsx deleted, no remaining imports in src/ | Lint | `cd frontend && pnpm tsc --noEmit` | ✅ EXISTING | ⬜ pending |
| 24-02-02 | 02 | 1 | — | — | `outputPane.*` and `historyPane.*` i18n keys removed | Lint | `cd frontend && pnpm tsc --noEmit` (no string lookup warnings) | ✅ EXISTING | ⬜ pending |
| 24-03-01 | 03 | 1 | — | — | `RunInTerminal` removed from execution_service.go | Compile | `go build ./...` | ✅ EXISTING | ⬜ pending |
| 24-03-02 | 03 | 1 | — | — | Regenerated bindings drop `RunInTerminal` | Lint | `cd frontend && pnpm tsc --noEmit` | ✅ EXISTING | ⬜ pending |
| 24-04-01 | 04 | 1 | — | — | `cmd-executing` removed from event_service.go EventNames | Compile | `go build ./...` | ✅ EXISTING | ⬜ pending |
| 24-04-02 | 04 | 1 | — | — | Terminal.tsx no longer subscribes to `cmd-executing` | Lint | `cd frontend && pnpm tsc --noEmit` | ✅ EXISTING | ⬜ pending |
| 24-05-01 | 05 | 1 | — | — | Orphan `.output-pane*` CSS removed from style.css | Lint | `cd frontend && pnpm tsc --noEmit && pnpm lint` | ✅ EXISTING | ⬜ pending |
| 24-05-02 | 05 | 1 | — | — | e2e selectors and mocks cleaned up (no outputPane selector, no commented RunInTerminal) | Lint | `cd frontend && pnpm tsc --noEmit` | ✅ EXISTING | ⬜ pending |
| 24-06-01 | 06 | 2 | EXEC-05 | — | Real-time ANSI output streams to active session terminal | Manual | Run `echo -e "\033[31mRED\033[0m"` via saved command; verify colored output in terminal tab | ❌ Manual | ⬜ pending |
| 24-06-02 | 06 | 2 | EXEC-06 | — | Ctrl+C interrupts running command in active session | Manual | Run `sleep 30` via saved command; press Ctrl+C in terminal; verify shell returns to prompt | ❌ Manual | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `execution_service_test.go` — update `testDBCreateCommand` (or add `testWithTerminalSvc` helper) to set up `terminalSvc` via `ServiceStartup` before each `RunCommand` test. The existing 4 tests will all need this setup to avoid nil-deref after the Phase 24 refactor.
- [ ] `execution_service_test.go` — NEW `TestRunCommand_ExecutesOnActiveSession` (EXEC-01): set up a real `TerminalService`, call `RunCommand` with a benign command, assert `record.Error == ""`.
- [ ] `execution_service_test.go` — NEW `TestRunCommand_NoActiveSession` (EXEC-01 edge case): set up a `TerminalService` with no sessions, call `RunCommand`, assert `record.Error` contains "no active" and `record.ExitCode == -1`.
- [ ] `execution_service_test.go` — NEW `TestRunCommand_NilTerminalSvc` (cold-start race): with `terminalSvc = nil`, call `RunCommand`, assert `record.Error` contains "terminal service" and `record.ExitCode == -1`.
- [ ] Optional: extract `buildCmdLine` as a separate function to make cmdLine construction testable without a live `TerminalService`.

*If none: "Existing infrastructure covers all phase requirements."*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Real-time ANSI output streaming | EXEC-05 | Requires live PTY + xterm.js rendering in browser view | Open `wails3 dev`, run `echo -e "\033[31mRED\033[0m"` via saved command, verify colored output in active session tab |
| Ctrl+C interrupt | EXEC-06 | Requires user input + live PTY | Open `wails3 dev`, run `sleep 30` via saved command, press Ctrl+C in terminal tab, verify shell returns to prompt within 1s |

*If none: "All phase behaviors have automated verification."*

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 10s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
