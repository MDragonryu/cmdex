# Phase 24: Session-Aware Execution - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-06-15
**Phase:** 24-session-aware-execution
**Areas discussed:** Execution dispatch, Command construction, Working directory, Variable resolution

---

## Execution Dispatch

| Option | Description | Selected |
|--------|-------------|----------|
| Always in active terminal session | Run button always writes to active session PTY. Output pane retired. | ✓ |
| Terminal when available, fallback | Fall back to output pane if no session or stopped shell | |
| User chooses per execution | Dropdown/split button for terminal vs output | |

**User's choice:** Always in active terminal session.
**Notes:** Terminal sessions are the primary execution surface. No dual-mode.

### Sub-decision: Stopped shell behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Auto-start the shell, then execute | Call Start(sessionId), wait for readiness, then write command | ✓ |
| Show an error | Toast: 'Shell is not running. Start session first.' | |

**User's choice:** Auto-start the shell.

### Sub-decision: External terminal (RunInTerminal)

| Option | Description | Selected |
|--------|-------------|----------|
| Remove it | Terminal sessions replace external terminal launch | ✓ |
| Keep it | Add separate UI control for external terminal | |

**User's choice:** Remove RunInTerminal entirely.

### Sub-decision: Output pane

| Option | Description | Selected |
|--------|-------------|----------|
| Remove it entirely | All output in terminal. Remove OutputPane.tsx and cmd-output events | ✓ |
| Keep for non-execution output | Repurpose for system messages, import/export logs | |

**User's choice:** Remove OutputPane entirely.

---

## Command Construction

| Option | Description | Selected |
|--------|-------------|----------|
| Agent discretion | Existing RunCommand resolution + Write(sessionId, data) is sufficient — no new spec needed | ✓ |

**User's choice:** Already handled by existing infrastructure. Agent decides exact wiring.
**Notes:** Command construction reuses existing pieces without reimplementing.

---

## Working Directory

| Option | Description | Selected |
|--------|-------------|----------|
| Agent discretion | Existing working directory fallback chain already implemented — no new spec needed | ✓ |

**User's choice:** Skip — existing infrastructure handles this.
**Notes:** Per-command → global default → session cwd chain from Phase 10-13 applies.

---

## Variable Resolution

### Prompt flow

| Option | Description | Selected |
|--------|-------------|----------|
| Reuse existing VariablePrompt flow as-is | Same modal, same behavior, same variable handling | ✓ |
| Streamline — auto-resolve CEL/env defaults | Only prompt for variables without defaults | |

**User's choice:** Reuse existing VariablePrompt flow as-is.

### Preset persistence

| Option | Description | Selected |
|--------|-------------|----------|
| Keep presets — remember last values | Variable values saved as presets, pre-filled next run | ✓ |
| Don't persist — fresh each run | Start with defaults only | |

**User's choice:** Keep presets.

### Execution history

| Option | Description | Selected |
|--------|-------------|----------|
| No — terminal is transient | Don't record ExecutionRecord for terminal runs | ✓ |
| Yes — record all executions | Preserve audit trail | |

**User's choice:** No execution history for terminal runs.

---

## the agent's Discretion

- Exact command string construction for PTY writing (cd prefix, subshell wrapping, newline handling)
- Working directory cd strategy (permanent cd vs subshell)
- How to wire resolved command from RunCommand flow into `TerminalService.Write`
- Cleanup of dead code: OutputPane component, cmd-output event handlers, RunInTerminal bindings

## Deferred Ideas

- Execution history for terminal runs (rejected — could be opt-in later)
- External terminal launch (removed — could return if demand emerges)
