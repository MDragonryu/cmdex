---
phase: 21
slug: backend-session-foundation
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-10
---

# Phase 21 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go `testing` package (stdlib, no testify) |
| **Config file** | None — tests use Go's standard `_test.go` convention |
| **Quick run command** | `go test -race -count=1 -run 'TestTerminal' -v ./...` |
| **Full suite command** | `go test -race -count=1 -v ./...` |
| **Estimated runtime** | ~30 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test -race -count=1 -run '<current task test name>' -v ./...`
- **After every plan wave:** Run `go test -race -count=1 -v ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 21-01-01 | 01 | 1 | SESS-01 | — | CreateSession returns UUID v4, default name, running=true | unit | `go test -run TestTerminalService_CreateSession -v ./...` | ❌ W0 | ⬜ pending |
| 21-01-02 | 01 | 1 | SESS-01 | — | ListSessions returns all created sessions | unit | `go test -run TestTerminalService_ListSessions -v ./...` | ❌ W0 | ⬜ pending |
| 21-01-03 | 01 | 1 | SESS-04 | T-21-01 | RenameSession rejects empty string, accepts non-empty | unit | `go test -run TestTerminalService_RenameSession -v ./...` | ❌ W0 | ⬜ pending |
| 21-01-04 | 01 | 1 | SESS-05 | T-21-02 | CloseSession cleans up PTY, removes from map | unit | `go test -run TestTerminalService_CloseSession -v ./...` | ❌ W0 | ⬜ pending |
| 21-01-05 | 01 | 2 | EXEC-04 | T-21-03 | Long-running process persists across session switch | integration | `go test -run TestTerminalService_ProcessPersistAcrossSessionSwitch -v ./...` | ❌ W0 | ⬜ pending |
| 21-01-06 | 01 | 2 | EXEC-04 | T-21-04 | Output isolation — session A output not in session B channel | unit | `go test -run TestTerminalService_OutputIsolation -v ./...` | ❌ W0 | ⬜ pending |
| 21-01-07 | 01 | 2 | — | — | Namespaced events fire with correct session ID | unit | `go test -run TestTerminalService_NamespacedEvents -v ./...` | ❌ W0 | ⬜ pending |
| 21-01-08 | 01 | 2 | — | T-21-05 | ServiceShutdown cleans all sessions, no zombie processes | unit | `go test -run TestTerminalService_ShutdownCleansAll -v ./...` | ❌ W0 | ⬜ pending |
| 21-01-09 | 01 | 3 | — | T-21-06 | Concurrent access does not panic (race detector) | unit | `go test -race -run TestTerminalService_ConcurrentAccess -v ./...` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] `terminal_service_test.go` — expand with multi-session test cases (currently has single-session tests)
- [ ] `execution_service_test.go` — update singleton assignment test for new session manager role
- [ ] Test helper: `newTestSessionState(t)` — creates a sessionState with a mock PTY for unit testing PTY lifecycle
- [ ] Add `go test -race` to `make check` or document as required manual step

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Terminal emulator renders session output | EXEC-04 | Requires GUI — PTY output rendering depends on xterm.js frontend | Open app, create two sessions, run `echo hello` in each, verify output appears in correct tabs |
| Shell auto-restart on crash | — | Requires process kill signal | Start session, `kill -9` its shell PID from another terminal, verify shell automatically restarts |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
