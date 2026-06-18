# Phase 24: Session-Aware Execution - Context

**Gathered:** 2026-06-15
**Status:** Ready for planning

<domain>
## Phase Boundary

Redirect RunCommand from the existing output-pane executor to the active terminal session's PTY. Commands resolve variables via the existing VariablePrompt flow, apply working directory, then write the resolved command into the terminal's interactive shell. Output streams naturally through the terminal's PTY events (Phase 21), with Ctrl+C handled by the PTY's foreground process group.

**Phase 24 delivers:** Session-aware command execution — Run button writes to active terminal session instead of output pane. Removes OutputPane and RunInTerminal.

**Not in scope:** Session persistence (Phase 22 skipped), terminal tab UI (Phase 23), session creation (Phase 21).
</domain>

<decisions>
## Implementation Decisions

### Execution Dispatch
- **D-01:** Run button always executes in the active terminal session. No option to run in output pane.
- **D-02:** If the active session's shell is stopped, auto-start it via `Start(sessionId)` before executing the command. Transparent to user.
- **D-03:** Remove `RunInTerminal` (external terminal app launch). Terminal sessions replace it entirely.
- **D-04:** Remove `OutputPane.tsx` and all `cmd-output` event subscriptions. All command output is now in the terminal.

### Command Construction
- **D-05:** Use existing RunCommand variable resolution + `TerminalService.Write(sessionId, data)` to send the resolved command to the PTY. No new execution mechanism needed.

### Working Directory
- **D-06:** Existing working directory fallback chain (per-command → global default → session cwd) applies. Cd behavior in PTY is agent discretion.

### Variable Resolution
- **D-07:** Reuse existing VariablePrompt modal flow as-is. No changes to variable handling.
- **D-08:** Keep preset persistence — variable values from previous runs are pre-filled.
- **D-09:** Do NOT record `ExecutionRecord` in SQLite for terminal session execution. Terminal runs are transient.

### the agent's Discretion
- Exact command string construction for PTY writing (cd prefix, subshell wrapping, newline handling)
- Working directory cd strategy (permanent cd vs subshell)
- How to wire resolved command from RunCommand flow into `TerminalService.Write`
- Cleanup of dead code: OutputPane component, cmd-output event handlers, RunInTerminal bindings
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Roadmap & Requirements
- `.planning/ROADMAP.md` — Phase 24 definition, success criteria (5 items), depends on Phase 21 + Phase 23
- `.planning/REQUIREMENTS.md` — EXEC-01, EXEC-02, EXEC-03, EXEC-05, EXEC-06

### Prior Phase Context
- `.planning/phases/21-backend-session-foundation/21-CONTEXT.md` — Session CRUD, TerminalService.Write, event namespacing, active session management, shell lifecycle
- `.planning/phases/23-frontend-tabbed-terminal/23-CONTEXT.md` — Tab bar UI, TerminalComponent with sessionId prop, activeSessionId state

### Existing Code (must study before implementing)
- `execution_service.go` — `RunCommand` method: command loading, variable resolution, script construction, execution dispatch
- `terminal_service.go` — TerminalService: `Write(sessionId, data)`, `Start(sessionId)`, `GetActiveSession()`, PTY readLoop, event emission
- `executor.go` — Executor: `ExecuteScript`, temp script lifecycle, CEL default evaluation
- `script.go` — `ReplaceTemplateVars`, `ExtractTemplateVars`, `GenerateScript`
- `event_service.go` — EventNames struct, event emission pattern
- `models.go` — Command, ExecutionResult, VariableDefinition types
- `frontend/src/App.tsx` — RunCommand call sites, VariablePrompt modal, output pane state, activeSessionId
- `frontend/src/components/Terminal.tsx` — TerminalComponent with `sessionId` prop, event subscriptions
- `frontend/src/components/OutputPane.tsx` — To be removed

### Reference Documents
- `.planning/codebase/ARCHITECTURE.md` — Service patterns, data flow for RunCommand
- `.planning/codebase/CONVENTIONS.md` — Go and TypeScript coding patterns
</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`RunCommand` in `execution_service.go`**: Already handles command loading, variable resolution, working directory lookup, and script construction. Modify to redirect output to TerminalService.Write instead of ExecuteScript.
- **`TerminalService.Write(sessionId, data)`**: Accepts raw bytes and writes to PTY. Already handles auto-resume via Start() on stopped sessions.
- **`TerminalService.GetActiveSession()`**: Returns the current active session ID. Source of truth for where to route execution.
- **`ReplaceTemplateVars` in `script.go`**: Resolves `{{var}}` placeholders with provided values. Unchanged for Phase 24.
- **`VariablePrompt` modal in `App.tsx`**: Existing flow: detect unresolved variables → show modal → collect values → call RunCommand. Unchanged for Phase 24.

### Established Patterns
- **Wails event emission**: `wailsApp.Event.Emit(eventNames.X, data)`. Session events use namespaced format: `pty-output:{sessionId}`.
- **Frontend async call pattern**: `RunCommand(id, vars).then(result => ...)`. Will need to change since no ExecutionRecord is returned for terminal runs.
- **PTY write pattern**: `Write(sessionId, stringData)` — sends bytes to the PTY's stdin. Shell parses and executes.

### Integration Points
- **`RunCommand` in `execution_service.go`**: Currently creates temp script → ExecuteScript → emits cmd-output events → returns ExecutionRecord. Must change to: resolve command → Write to active session PTY → return (no ExecutionRecord).
- **`App.tsx` Run button handler**: Currently calls `RunCommand` and updates output pane state. Must change to call RunCommand and NOT update output pane. Result handling changes (no ExecutionRecord).
- **`cmd-output` event**: Currently subscribed in App.tsx for output pane. Remove subscription.
- **`OutputPane.tsx`**: Remove component entirely. No replacement needed.
- **`RunInTerminal`**: Remove from `execution_service.go` and generated bindings.
</code_context>

<specifics>
## Specific Ideas

User wants terminal sessions to be the natural command execution surface. The mental model is: "I have a terminal open, I click Run on a command, it runs there." No dual-mode execution, no output pane fallback, no execution history clutter.
</specifics>

<deferred>
## Deferred Ideas

- **Execution history for terminal runs:** Rejected — terminal is transient. If needed later, could add opt-in recording.
- **External terminal launch:** Removed in this phase. Could be re-added as a separate feature if user demand emerges.

None — discussion stayed within phase scope.
</deferred>

---

*Phase: 24-session-aware-execution*
*Context gathered: 2026-06-15*
