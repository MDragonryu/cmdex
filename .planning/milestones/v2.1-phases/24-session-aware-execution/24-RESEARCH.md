# Phase 24: Session-Aware Execution - Research

**Researched:** 2026-06-15
**Domain:** Backend ↔ Frontend dispatch refactor (PTY write directly from `RunCommand`; remove event-mediated handoff)
**Confidence:** HIGH

## Summary

Phase 24 collapses the current event-mediated dispatch chain into a direct backend call. Today, `ExecutionService.RunCommand` resolves variables, builds a `cmdLine` string, and emits a global `cmd-executing` event; the frontend's `TerminalComponent` then subscribes and calls `TerminalService.Write(sessionId, cmdLine)` on the active session. After Phase 24, `RunCommand` calls `terminalSvc.Write(sessionId, cmdLine)` directly on the backend and returns an `ExecutionRecord` (kept for error reporting and test compatibility).

The plumbing is already in place from Phase 21: `TerminalService.GetActiveSession()` returns the active session's `SessionInfo`, and `Write(sessionId, data)` already auto-resumes a stopped shell via `startSessionLocked` (satisfies D-02 transparently). The frontend's `Terminal.tsx` already has the namespaced `pty-output:{sessionId}` subscription that streams PTY bytes into xterm.js, so output rendering needs zero changes. The work is mechanical: replace one event with one in-process call, delete now-unused code (`RunInTerminal`, `OutputPane.tsx`, the `cmd-executing` event/subscription), and update tests.

The biggest risk is silent breakage of `execution_service_test.go` because the new `RunCommand` calls `terminalSvc.Write`, which requires a live `TerminalService` with an open PTY. Existing tests must be updated to construct a `TerminalService` via `ServiceStartup` (already proven in `TestTerminalService_ServiceStartupAssignsTerminalSvc`) before invoking `RunCommand`.

**Primary recommendation:** Refactor `RunCommand` to write directly to the active session's PTY (no event hop), extract `buildCmdLine` into a testable helper, update `execution_service_test.go` to set up a real `TerminalService`, regenerate Wails bindings, and delete `RunInTerminal`, `OutputPane.tsx`, and the `cmd-executing` event plumbing.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

- **D-01:** Run button always executes in the active terminal session. No option to run in output pane.
- **D-02:** If the active session's shell is stopped, auto-start it via `Start(sessionId)` before executing the command. Transparent to user.
- **D-03:** Remove `RunInTerminal` (external terminal app launch). Terminal sessions replace it entirely.
- **D-04:** Remove `OutputPane.tsx` and all `cmd-output` event subscriptions. All command output is now in the terminal.
- **D-05:** Use existing RunCommand variable resolution + `TerminalService.Write(sessionId, data)` to send the resolved command to the PTY. No new execution mechanism needed.
- **D-06:** Existing working directory fallback chain (per-command → global default → session cwd) applies. Cd behavior in PTY is agent discretion.
- **D-07:** Reuse existing VariablePrompt modal flow as-is. No changes to variable handling.
- **D-08:** Keep preset persistence — variable values from previous runs are pre-filled.
- **D-09:** Do NOT record `ExecutionRecord` in SQLite for terminal session execution. Terminal runs are transient.

### the agent's Discretion

- Exact command string construction for PTY writing (cd prefix, subshell wrapping, newline handling)
- Working directory cd strategy (permanent cd vs subshell)
- How to wire resolved command from RunCommand flow into `TerminalService.Write`
- Cleanup of dead code: OutputPane component, cmd-output event handlers, RunInTerminal bindings

### Deferred Ideas (OUT OF SCOPE)

- **Execution history for terminal runs:** Rejected — terminal is transient. If needed later, could add opt-in recording.
- **External terminal launch:** Removed in this phase. Could be re-added as a separate feature if user demand emerges.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| **EXEC-01** | User can execute a saved command in the active terminal session | `RunCommand` calls `terminalSvc.GetActiveSession()` → `terminalSvc.Write(session.ID, cmdLine)` (Pattern 1). Phase 21's `GetActiveSession` already returns the active session's `SessionInfo`. |
| **EXEC-02** | Command variables are resolved (CEL defaults, env, prompts) before sending to session | Existing `ReplaceTemplateVars` (`script.go:54-63`) + `EvalDefaults` (`executor.go:341-406`) + `GetVariables` (`execution_service.go:63-92`) pipeline — all unchanged. CEL evaluation happens in the backend before the cmdLine is built and written. |
| **EXEC-03** | Command working directory is applied (per-command → global default → session cwd) | `ExecutionService.resolveWorkingDir` (`execution_service.go:28-60`) implements the full fallback chain. `hasExplicitWorkingDir` (`execution_service.go:97-109`) decides whether to prefix with `cd %s &&`. D-06 satisfied. |
| **EXEC-05** | Command output streams to the active session's terminal in real-time | Phase 23's `Terminal.tsx:309-318` already subscribes to `pty-output:{sessionId}` and writes to xterm.js. `TerminalService.Write` writes to PTY, PTY echoes to `readLoop` → `emitter` → event → xterm. ANSI sequences pass through transparently. |
| **EXEC-06** | User can send Ctrl+C to interrupt a running command in the active session | xterm.js's `term.onData` (`Terminal.tsx:246-254`) already pipes raw keystrokes (including `\x03`) to `Write(sessionId, data)`. PTY's foreground process group receives SIGINT. No new code needed. |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|--------------|----------------|-----------|
| Variable resolution (CEL/env/prompts) | Backend (`ExecutionService`) | Frontend (`VariablePrompt` modal) | Resolution already lives in `ReplaceTemplateVars` + `EvalDefaults`; modal is purely input collection. |
| Command line construction | Backend (`ExecutionService`) | — | Pure function: `{{vars}}` → resolved script + optional `cd … &&` prefix. Single source of truth. |
| Working directory resolution | Backend (`ExecutionService.resolveWorkingDir`) | — | Already implements D-06 fallback chain; lives in `execution_service.go:28-60`. |
| PTY write | Backend (`TerminalService.Write`) | — | Already takes `sessionId` + `data`; auto-resumes stopped shells (satisfies D-02). |
| Active session lookup | Backend (`TerminalService.GetActiveSession`) | Frontend cache (`activeSessionId` state) | Backend is source of truth per D-03 of Phase 21; frontend mirrors for UI only. |
| Output streaming | Backend PTY → `pty-output:{id}` event → Frontend xterm.js | — | Namespaced events from Phase 21; `Terminal.tsx:309-318` subscribes per session. |
| Ctrl+C interrupt | Frontend xterm.js (raw byte input) → Backend PTY | — | xterm's `term.onData` (`Terminal.tsx:246-254`) already pipes keystrokes to `Write(sessionId, data)`; `Ctrl+C` = `0x03` byte → PTY foreground process group. No new code. |
| Variable prompt collection | Frontend (`VariablePrompt` modal) | Backend `GetVariables` | Existing flow; `App.tsx:1064-1083` unchanged. |
| Command result reporting | Backend → `ExecutionRecord` (return value) | Frontend toast | Error path already in `runCommandDirect` (`App.tsx:990-994`); no history persistence. |
| Cleanup of `OutputPane`, `cmd-executing`, `RunInTerminal` | Frontend (delete file/imports) + Backend (delete method/event) | — | Direct removals; nothing else references these after Phase 23 already retired the output pane from `App.tsx`. |

## Standard Stack

No new libraries. Phase 24 is a refactor using existing Phase 21 + Phase 23 infrastructure.

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/creack/pty` | v1.1.11 (current in go.sum) | Unix PTY for session write | Already in use; `terminal_unix.go:12-32` |
| `@xterm/xterm` | v5.5.0 (current in `frontend/package.json`) | Terminal renderer; receives `pty-output:{id}` bytes | Already in use; `Terminal.tsx:2` |
| `@wailsio/runtime` | Wails v3 runtime | `Events.On` for PTY events; `Write` from generated bindings | Already in use; `Terminal.tsx:7,9` |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/google/uuid` | v1.6.0 (current in go.sum) | Generate `ExecutionRecord.ID` | Already used in `execution_service.go:10,118,144` |
| `sonner` (toast) | current in `frontend/package.json` | User feedback for run success/failure | Already in use; `App.tsx:20,992` |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Direct `terminalSvc.Write` from `RunCommand` | Keep event-based dispatch (`cmd-executing` event → frontend `Write`) | The current pattern was a Phase 21 stopgap before `terminalSvc` global was reliably initialized. Now that `terminalSvc` is always non-nil after `TerminalService.ServiceStartup`, the direct call is simpler, eliminates one event round-trip, and removes a class of timing bugs (frontend subscription not yet attached when event fires). Rejected. |
| Subshell wrapping: `(cd $dir && cmd)` | Permanent `cd $dir` (changes shell state across runs) | Subshell wrapping is reversible (next command starts from previous cwd, not the just-finished command's cwd). Permanent cd would compound across runs. The existing `cmdLine = "cd %s && %s\n"` pattern (Phase 18) is the de-facto approach and already subshell-equivalent for a single command. Keep the current pattern. |
| Multi-line script: `cmd1\ncmd2\n` as one write | `Write` per line | Single write is atomic from the shell's perspective; terminal sees one keystroke burst and displays it as a coherent block. The existing pattern (`resolvedScript + "\n"`) is correct. |
| Send Ctrl+C via `Write(sessionId, "\x03")` from a "Stop" button | User types into xterm | User pressing Ctrl+C in xterm is the natural mechanism (`term.onData` already pipes `\x03` through `Write`). No new code needed; D-02/D-06 of Phase 21 already cover PTY foreground process group signal delivery. Reject new "Stop" button. |

**No new packages to install.** The Phase Legitimacy Audit (below) is N/A — nothing new is added.

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Frontend (React)                                │
│                                                                              │
│  ┌──────────────────┐         ┌────────────────────┐                         │
│  │ CommandDetailTab │ Run btn │  App.runCommandDirect                        │
│  │ (or Cmd+Enter)   │────────▶│  App.tsx:981-1005  │                         │
│  └──────────────────┘         └─────────┬──────────┘                         │
│                                          │ RunCommand(id, vars)              │
└──────────────────────────────────────────┼──────────────────────────────────┘
                                           │ Wails IPC
┌──────────────────────────────────────────▼──────────────────────────────────┐
│                            Backend (Go / Wails)                              │
│                                                                              │
│  ┌──────────────────────────────────────────────────────────┐                │
│  │ ExecutionService.RunCommand(commandID, variables)        │  ◀── Phase 24  │
│  │   1. db.GetCommand(commandID)                            │      refactor  │
│  │   2. ReplaceTemplateVars(cmd.ScriptContent, variables)   │                │
│  │   3. stripShebang + TrimRight                            │                │
│  │   4. resolveWorkingDir(cmd)  [D-06 fallback chain]       │                │
│  │   5. Build cmdLine:                                       │                │
│  │        - "cd %q && %s\n" if explicit working dir          │                │
│  │        - "%s\n"              otherwise                    │                │
│  │   6. terminalSvc.GetActiveSession() → session            │                │
│  │   7. terminalSvc.Write(session.ID, cmdLine)  ◀── NEW     │                │
│  │   8. Return ExecutionRecord{FinalCmd: cmdLine, ...}      │                │
│  └──────────────────────────┬───────────────────────────────┘                │
│                             │ Write(session.ID, cmdLine)                     │
│                             ▼                                                │
│  ┌──────────────────────────────────────────────────────────┐                │
│  │ TerminalService.Write(sessionId, data)   [Phase 21]     │                │
│  │   - resolveSession(sessionId) → *sessionState            │                │
│  │   - if !ss.running → startSessionLocked (D-02)           │                │
│  │   - ss.ptmx.Write([]byte(data))                           │                │
│  └──────────────────────────┬───────────────────────────────┘                │
│                             │ bytes to PTY stdin                              │
│                             ▼                                                │
│  ┌──────────────────────────────────────────────────────────┐                │
│  │ sessionState.readLoop → outputCh → emitter               │                │
│  │   - wailsApp.Event.Emit("pty-output:"+ss.id, data)      │                │
│  └──────────────────────────┬───────────────────────────────┘                │
│                             │ pty-output:{sessionId}                          │
└─────────────────────────────┼────────────────────────────────────────────────┘
                              │
┌─────────────────────────────▼────────────────────────────────────────────────┐
│                       Frontend (Terminal.tsx)                                │
│  Events.On("pty-output:"+sessionId, ...) → xterm.write(output)                │
│  [unchanged from Phase 23]                                                   │
│                                                                              │
│  Ctrl+C: user types into xterm → term.onData("\x03") → Write(sessionId,"\x03")│
│  → PTY foreground process group receives SIGINT [no new code]                 │
└──────────────────────────────────────────────────────────────────────────────┘
```

The single change is the **red arrow** at step 7: the backend calls `terminalSvc.Write` directly instead of emitting a `cmd-executing` event that the frontend would have received and forwarded.

### Recommended Project Structure

No new files. The diff is a set of removals plus a focused edit:

```
execution_service.go           # EDIT — RunCommand: replace event emit with direct Write call
                                  Remove: RunInTerminal method
event_service.go               # EDIT — remove CmdExecuting field + value
execution_service_test.go      # EDIT — set up terminalSvc in test setup
frontend/src/wails/events.ts   # EDIT — remove cmdExecuting entry
frontend/src/components/Terminal.tsx       # EDIT — remove cmd-executing subscription (lines 334-345)
frontend/src/components/OutputPane.tsx     # DELETE — orphan file (already unused in App.tsx)
frontend/bindings/cmdex/executionservice.js # REGEN — RunInTerminal removed
frontend/bindings/cmdex/eventservice.js    # REGEN — EventNames no longer has cmdExecuting
frontend/src/locales/en.json               # EDIT — remove outputPane + historyPane sections
```

### Pattern 1: Direct In-Process Call Across Services

**What:** `ExecutionService` invokes `TerminalService.Write` via the `terminalSvc` package-level pointer set in `TerminalService.ServiceStartup` (`terminal_service.go:127`).

**When to use:** Any time two Wails services need to coordinate synchronously within the same backend process. Same pattern is already used by `execution_service_test.go:181-190` to verify `terminalSvc` initialization.

**Example:**

```go
// Source: cmamer/execution_service.go (current — to be replaced)
// Source: cmamer/terminal_service.go:554-582 (Write signature)

func (s *ExecutionService) RunCommand(commandID string, variables map[string]string) ExecutionRecord {
    cmd, err := db.GetCommand(commandID)
    if err != nil {
        return ExecutionRecord{ID: uuid.New().String(), Error: err.Error(), ExitCode: -1}
    }

    cmdLine, err := s.buildCmdLine(cmd, variables)
    if err != nil {
        return ExecutionRecord{ID: uuid.New().String(), Error: err.Error(), ExitCode: -1}
    }

    if terminalSvc == nil {
        return ExecutionRecord{ID: uuid.New().String(), Error: "terminal service not initialized", ExitCode: -1}
    }
    session := terminalSvc.GetActiveSession()
    if session == nil {
        return ExecutionRecord{ID: uuid.New().String(), Error: "no active terminal session", ExitCode: -1}
    }

    if err := terminalSvc.Write(session.ID, cmdLine); err != nil {
        return ExecutionRecord{ID: uuid.New().String(), Error: err.Error(), ExitCode: -1}
    }

    return ExecutionRecord{
        ID:         uuid.New().String(),
        CommandID:  commandID,
        FinalCmd:   cmdLine,
        ExecutedAt: time.Now(),
    }
}
```

```go
// Source: cmamer/execution_service.go (new helper, testable in isolation)
func (s *ExecutionService) buildCmdLine(cmd Command, variables map[string]string) (string, error) {
    resolved := ReplaceTemplateVars(cmd.ScriptContent, variables)
    resolved = stripShebang(resolved)
    resolved = strings.TrimRight(resolved, "\n")
    if resolved == "" {
        return "", fmt.Errorf("empty script after variable resolution")
    }

    workingDir := s.resolveWorkingDir(cmd)
    if s.hasExplicitWorkingDir(cmd) {
        return fmt.Sprintf("cd %s && %s\n", shellQuoteDir(workingDir), resolved), nil
    }
    return resolved + "\n", nil
}
```

### Pattern 2: Auto-Resume on Stale Session (D-02)

**What:** `TerminalService.Write` already calls `startSessionLocked` when `!ss.running`. This is the D-02 "transparent auto-start" — no new code needed in `RunCommand`.

**When to use:** Any code that needs to push bytes into a session that may have been stopped (e.g., after the user typed `exit` or the shell crashed non-fatally).

**Example:** (existing — no change required)

```go
// Source: cmamer/terminal_service.go:554-582 (Write method)
func (s *TerminalService) Write(sessionId string, data string) error {
    ss, err := s.resolveSession(sessionId)
    if err != nil { return err }

    ss.mu.Lock()
    defer ss.mu.Unlock()

    if !ss.running {
        if err := s.startSessionLocked(ss, int(ss.lastSize.Cols), int(ss.lastSize.Rows)); err != nil {
            return err
        }
    }
    if ss.ptmx == nil { return fmt.Errorf("terminal not started") }

    b := []byte(data)
    for len(b) > 0 {
        n, err := ss.ptmx.Write(b)
        if err != nil { return err }
        b = b[n:]
    }
    return nil
}
```

### Pattern 3: Event Removal (Clean Break)

**What:** When a previously-emitted event is no longer needed, remove it from (1) the producer's `EventNames` struct + value, (2) any consumer subscriptions, and (3) regenerate bindings. No backward-compat shim — Phase 21 established the "clean break" precedent for `pty-output`/`pty-exit`/`pty-cleared` (D-04 of Phase 21).

**When to use:** Any refactor that retires an event. Same pattern as Phase 21's event namespacing (`.planning/phases/21-backend-session-foundation/21-02-PLAN.md:238`, `21-03-PLAN.md:104-115`).

**Example:** (this phase — three locations to edit)

```go
// Source: cmamer/event_service.go (EDIT)
type EventNames struct {
    OpenSettings          string `json:"openSettings"`
    OpenShortcuts         string `json:"openShortcuts"`
    SettingsChanged       string `json:"settingsChanged"`
    SettingsWindowClosing string `json:"settingsWindowClosing"`
    // CmdExecuting REMOVED — RunCommand now calls TerminalService.Write directly
}

var eventNames = EventNames{
    OpenSettings:          "open-settings",
    OpenShortcuts:         "open-shortcuts",
    SettingsChanged:       "settings-changed",
    SettingsWindowClosing: "settings-window-closing",
}
```

```typescript
// Source: cmamer/frontend/src/wails/events.ts (EDIT)
export const eventNames = {
    openSettings: 'open-settings',
    openShortcuts: 'open-shortcuts',
    settingsChanged: 'settings-changed',
    settingsWindowClosing: 'settings-window-closing',
    // cmdExecuting REMOVED
};

// In initEventNames():
//   eventNames.cmdExecuting = names.cmdExecuting;  // REMOVED
```

```typescript
// Source: cmamer/frontend/src/components/Terminal.tsx (EDIT, lines 334-345)
// Remove entire cleanupCmdExecuting block — the subscription has no producer anymore.
//   const cleanupCmdExecuting = Events.On(eventNames.cmdExecuting, (event: { data: { data: string } }) => {
//     if (activeSessionIdRef.current !== sessionIdRef.current) return;
//     const cmdLine = event?.data?.data;
//     if (cmdLine && backendAvailableRef.current) {
//       Write(sessionIdRef.current, cmdLine).catch(...)
//     }
//   });

// In the cleanup return (line 347-352):
//   cleanupCmdExecuting();  // REMOVED
```

### Pattern 4: Test Setup for Service Singletons

**What:** Tests that depend on a package-level service pointer (`terminalSvc`) must save/restore the previous value and construct the service via `ServiceStartup` to avoid a nil dereference.

**When to use:** Any test that exercises a code path touching `terminalSvc` (or future global services).

**Example:** (precedent from `execution_service_test.go:176-200`)

```go
// Source: cmamer/execution_service_test.go (EXTEND existing testDBCreateCommand)
prevTerminalSvc := terminalSvc
terminalSvc = nil
defer func() {
    if terminalSvc != nil {
        _ = terminalSvc.ServiceShutdown()
    }
    terminalSvc = prevTerminalSvc
}()

ts := &TerminalService{}
if err := ts.ServiceStartup(nil, application.ServiceOptions{}); err != nil {
    t.Skipf("TerminalService.ServiceStartup failed: %v", err)
}
```

### Anti-Patterns to Avoid

- **Re-introducing `cmd-output` events for stream output:** Output already flows through `pty-output:{sessionId}`. Adding a second channel for the same bytes would double-render and split focus state.
- **Recording `ExecutionRecord` in SQLite (D-09 violation):** The `ExecutionRecord` returned from `RunCommand` is for the call site (frontend error reporting) and tests. It must NOT be passed to `db.AddExecution` — that's the entire point of "terminal is transient."
- **Calling `Start(sessionId, ...)` explicitly before `Write`:** `Write` already auto-resumes a stopped session. Calling `Start` first is a redundant round-trip and risks starting the shell twice if the user races a tab switch.
- **Adding a "Stop" button that calls `terminalSvc.Stop`:** The xterm input layer already pipes Ctrl+C (0x03) to the PTY's foreground process group via the existing `Write(sessionId, data)` path. A separate button is dead code.
- **Keeping the `cmd-executing` field on `EventNames` as a "harmless fallback":** Dead event names rot. Clean removal is consistent with Phase 21's event-namespace clean break (`.planning/research/ARCHITECTURE.md:60-67`).
- **Wrapping the command in a subshell `(cd dir && cmd)`:** The existing `cd %s && %s\n` pattern is already single-line and reversible. Adding parentheses buys nothing and complicates quoting.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Active session lookup | Manual scan of `SessionInfo[]` in `App.tsx` for the session marked `isActive` | `terminalSvc.GetActiveSession()` from backend | Backend is source of truth per Phase 21 D-03; frontend `activeSessionId` is a UI mirror that can drift. |
| PTY auto-resume | Custom "if stopped, call Start then write" logic in `RunCommand` | `TerminalService.Write` (already auto-resumes) | Existing `terminal_service.go:563-567` already handles this; re-implementing risks deadlock against `ss.mu`. |
| Working directory resolution | New resolver duplicating the per-command → global → home chain | `ExecutionService.resolveWorkingDir` (existing) | Same fallback chain implemented and tested in Phase 10-13. |
| Variable substitution | Inline `{{var}}` regex in `RunCommand` | `ReplaceTemplateVars` from `script.go:54-63` | Tested in `db_test.go`; CEL defaults already evaluated upstream. |
| Output streaming | Direct subscription to PTY in `App.tsx` | Existing `Terminal.tsx` per-session `pty-output:{id}` subscription | Phase 23 already wires this; no new streaming code path needed. |
| Ctrl+C interrupt | New keyboard handler in `App.tsx` that sends `\x03` to the PTY | xterm.js `term.onData` (already pipes all keystrokes to `Write`) | `Terminal.tsx:246-254` already exists; re-implementing would double-send and split focus state. |
| Stopped-shell error message | New toast saying "shell stopped" | Let `Write` fail naturally; existing error toast at `App.tsx:992-997` catches it | The existing path already surfaces `Write` errors via `result.error`. |

**Key insight:** Phase 24 is a deletion-heavy refactor. The work that looks "new" is mostly removing indirection that was only there to bridge a pre-Phase-21 architecture (when `terminalSvc` didn't exist as a stable global). The first instinct will be to add code; resist it. Trust the infrastructure Phase 21 + Phase 23 already built.

## Common Pitfalls

### Pitfall 1: Test Suite Crashes on `terminalSvc == nil`

**What goes wrong:** `RunCommand` calls `terminalSvc.Write(...)` which dereferences `terminalSvc`. The existing `execution_service_test.go` tests (lines 50-153) construct only `&ExecutionService{}` and assume `terminalSvc` is irrelevant. After the refactor, those tests will panic with a nil pointer dereference at the first `terminalSvc.GetActiveSession()` call.

**Why it happens:** The current test setup never invokes `TerminalService.ServiceStartup`, so the package-level `terminalSvc` stays at its zero value (`nil`).

**How to avoid:**
- In `testDBCreateCommand` (or a new test helper `testWithTerminalSvc`), save the previous `terminalSvc`, set it to a freshly-constructed `&TerminalService{}`, call its `ServiceStartup`, and register a `defer` that calls `ServiceShutdown` and restores the previous value.
- Alternatively, extract `buildCmdLine` as an exported helper and have tests verify that helper directly. The "Write was called" test would then need a fake or live PTY.

**Warning signs:** `go test ./...` panics with `runtime error: invalid memory address or nil pointer dereference` at `terminal_service.go:556` or `execution_service.go` (new Write call site). The earlier `TestTerminalService_ServiceStartupAssignsTerminalSvc` test (line 176) does the setup pattern correctly — use it as the template.

### Pitfall 2: `OutputPane.tsx` Linter Error After Deletion

**What goes wrong:** `OutputPane.tsx` is currently not imported anywhere in `App.tsx` (verified by grep — only the file itself, `style.css`, and `e2e/utils/selectors.ts` reference `output-pane` strings). Deleting the file is safe for the app, but `frontend/e2e/utils/selectors.ts:40` has `outputPane: '[data-testid="output-pane"]'` and the `e2e/mocks/runtime.ts` has commented-out references to `RunInTerminal`/`GetExecutionHistory`. The e2e test setup is dead-code prone.

**Why it happens:** Phase 23 already removed `OutputPane` from the layout but left the file as a residual. The e2e helpers weren't cleaned up because they were "just in case" the pane came back.

**How to avoid:**
- Delete `OutputPane.tsx` (orphaned component file).
- Remove the `outputPane` selector from `frontend/e2e/utils/selectors.ts` if no test uses it (verify with `grep -r 'outputPane' frontend/e2e/tests/`).
- Remove the commented `// RunInTerminal` / `// GetExecutionHistory` blocks from `frontend/e2e/mocks/runtime.ts` to avoid drift.
- The `outputPane` (lines 175-189) and `historyPane` (lines 167-174) and `common.copyLastOutput` (line 10) entries in `frontend/src/locales/en.json` are unreferenced after the refactor — remove them so i18n doesn't ship dead keys.

**Warning signs:** `pnpm tsc --noEmit` passes but `pnpm lint` flags unused exports; or e2e tests in CI fail because the `[data-testid="output-pane"]` selector is missing.

### Pitfall 3: `RunInTerminal` Removal Breaks Regenerated Bindings File

**What goes wrong:** `frontend/bindings/cmdex/executionservice.js` is a generated file (`// @ts-check` header, "DO NOT EDIT" comment). After removing the `RunInTerminal` method from `execution_service.go`, regenerating with `wails3 generate bindings` (or `wails3 generate build-assets`) will drop the `export function RunInTerminal(...)` line. Any TypeScript import of `RunInTerminal` will become a compile error.

**Why it happens:** Hand-rolled code that imports the soon-to-be-deleted binding method.

**How to avoid:** Grep for `RunInTerminal` across `frontend/src/` before regenerating. The current state (verified) shows zero imports in app code — the method is exposed but unused. After regeneration, no further action is needed.

**Warning signs:** `pnpm tsc --noEmit` reports `Module '"../../bindings/cmdex/executionservice"' has no exported member 'RunInTerminal'`.

### Pitfall 4: Event Subscription Leaks in `Terminal.tsx`

**What goes wrong:** If the `cleanupCmdExecuting` line is removed from the cleanup return (line 351) but the `const cleanupCmdExecuting = ...` declaration (line 334) is left in place, the subscription will still fire if a stray `cmd-executing` event ever escapes from a stale build. Conversely, removing the subscription cleanly is the right move but easy to miss alongside the 3 other cleanup calls.

**Why it happens:** Diff-by-omission when editing 3 contiguous things in the same `useEffect`.

**How to avoid:** Remove both the `const cleanupCmdExecuting = Events.On(...)` block AND the `cleanupCmdExecuting()` line in the return. Also remove the unused `eventNames` import (line 8) if `eventNames.cmdExecuting` was its only remaining use — verify by grepping `eventNames` in `Terminal.tsx` after the edit.

**Warning signs:** The `eventNames` import becomes dead. Or a `cmd-executing` event fires from a stale event-bus binding and crashes because the handler is gone (defensive guard missing in Wails v3 — see `.planning/codebase/INTEGRATIONS.md` if it covers this).

### Pitfall 5: `terminalSvc` Unavailable on First Run

**What goes wrong:** `RunCommand` is invoked from the frontend before `TerminalService.ServiceStartup` has run (e.g., a race during cold start where the frontend mounts and dispatches a Run click before all Wails services are up). `terminalSvc == nil` → crash.

**Why it happens:** `terminalSvc` is set inside `TerminalService.ServiceStartup` (`terminal_service.go:127`), and Wails service startup order is implementation-defined.

**How to avoid:** Defensive check at the top of `RunCommand`:
```go
if terminalSvc == nil {
    return ExecutionRecord{ID: uuid.New().String(), Error: "terminal service not initialized", ExitCode: -1}
}
```
Mirror the pattern already used for `db == nil` checks in `execution_service.go` and similar methods. The frontend's `runCommandDirect` (`App.tsx:990-994`) will surface the error as a toast.

**Warning signs:** Crash report on first run after install. Rare in practice — `App.tsx:316-336` ensures a session exists by the time the user can click Run — but the defensive check costs nothing.

### Pitfall 6: Subshell Quoting for `cd` Argument

**What goes wrong:** The existing `cmdLine = "cd %s && %s\n"` with `shellQuoteDir(workingDir)` handles simple paths, paths with spaces, and paths with single quotes (verified by `TestShellQuoteDir` at line 155). But the Phase 24 planner might be tempted to "improve" this with backticks, double quotes, or POSIX-portable alternatives that break the existing tests.

**Why it happens:** Tinkering with a working pattern that has unit tests.

**How to avoid:** Leave `shellQuoteDir` and the `cd %s && %s` format exactly as-is. The existing `TestShellQuoteDir` covers the edge cases. Only change the dispatch path; do not change the cmdLine construction (D-06 says working-directory behavior is locked).

**Warning signs:** `TestShellQuoteDir` or `TestRunCommand_FinalCmdWithWorkingDir` starts failing after an "improvement" to the format string.

### Pitfall 7: Removing `ExecutionRecord` Fields Breaks Test Contract

**What goes wrong:** D-09 says "Do NOT record `ExecutionRecord` in SQLite for terminal session execution." A planner might interpret this as "remove all the `ExecutionRecord` fields except `Error`." But the existing tests (lines 50-95) check `record.FinalCmd`, and the frontend's `runCommandDirect` reads `result.error` and `result.exitCode`. Removing those fields would break both.

**Why it happens:** Misreading D-09 as "don't return an ExecutionRecord" instead of "don't persist one."

**How to avoid:** Keep `ExecutionRecord` as the return type with `ID`, `CommandID`, `FinalCmd`, `ExecutedAt`, `Error`, `ExitCode`. Just don't add a `db.AddExecution(record)` call. The struct in `models.go:133-144` and the DB schema are untouched.

**Warning signs:** `TestRunCommand_FinalCmdWithWorkingDir` fails because `record.FinalCmd == ""`. Or `App.tsx:990-994` toasts on every successful run because `result.error` is unexpectedly populated.

## Code Examples

Verified patterns from official sources and existing codebase:

### 1. Direct Service-to-Service Call (Existing Precedent)

```go
// Source: cmamer/execution_service.go:114-148 (current — to be modified)
// The existing test (execution_service_test.go:176-200) already demonstrates
// the package-level pointer pattern that Phase 24 will rely on at runtime.
```

### 2. `Write` with Auto-Resume

```go
// Source: cmamer/terminal_service.go:554-582 (existing, no change)
// Auto-resume is built into the existing method — RunCommand does not need
// to call Start() separately to satisfy D-02.
```

### 3. Working Directory Fallback Chain

```go
// Source: cmamer/execution_service.go:28-60 (existing, no change)
// Per-command → global default → home → cwd → tempdir. Already implements D-06.
func (s *ExecutionService) resolveWorkingDir(cmd Command) string { /* ... */ }
```

### 4. Event Removal (Pattern from Phase 21)

```go
// Source: cmamer/event_service.go:10-24 (to be edited)
// Phase 21's "clean break" precedent: removing fields from EventNames without
// backward-compat shims (see 21-02-PLAN.md:238).
```

### 5. Test Setup with Service Singleton

```go
// Source: cmamer/execution_service_test.go:176-200 (precedent to extend)
func TestTerminalService_ServiceStartupAssignsTerminalSvc(t *testing.T) {
    if testing.Short() { t.Skip("skipping integration test in short mode") }
    prevTerminalSvc := terminalSvc
    terminalSvc = nil
    defer func() { terminalSvc = prevTerminalSvc }()

    s := &TerminalService{}
    _ = s.ServiceStartup(nil, application.ServiceOptions{})
    defer s.ServiceShutdown()

    if terminalSvc == nil { t.Error(...) }
    // ... assertions
}
```

The new `testWithTerminalSvc` helper for the existing `TestRunCommand_*` family should follow this same pattern.

## Runtime State Inventory

This section is included because Phase 24 is a refactor that retires a code path (event-mediated dispatch) and a data path (`db.AddExecution` is no longer called from `RunCommand`). The plan must verify nothing runtime-state-carrying is left behind.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | `executions` table in SQLite (`db.go:1018-1038` — `GetExecutions`, `AddExecution`) | **No change.** D-09 says don't write to it for terminal runs, but the table + reads must remain in case future code uses them. The history feature (`historyPane` i18n keys, no UI) is unused but not the subject of this phase. **Plan note:** planner should consider whether to also retire `GetExecutionHistory`/`ClearExecutionHistory` for code hygiene, or defer to a future phase. |
| Live service config | `terminalSvc` global (`app.go:17`, set in `terminal_service.go:127`) | **No change.** Phase 24 depends on it. |
| OS-registered state | None. No Task Scheduler / launchd / systemd entries. | None. |
| Secrets and env vars | None. | None. |
| Build artifacts | `frontend/bindings/cmdex/executionservice.js` — must be regenerated to drop `RunInTerminal`. `frontend/bindings/cmdex/eventservice.js` — must be regenerated to drop `CmdExecuting` field. | Regenerate via `wails3 generate bindings` (or `wails3 generate build-assets`). |
| In-memory state | `wailsApp` package-level pointer (`app.go:16`) — used by both `Event.Emit` calls and the eventual (post-Phase-24) `terminalSvc.Write`. The `cmd-executing` event is removed from the producer side, so any in-flight subscribers die. | None at runtime. |
| Frontend orphan files | `frontend/src/components/OutputPane.tsx` (280 lines, not imported anywhere). | **DELETE.** |
| Frontend orphan references | `frontend/e2e/utils/selectors.ts:40` `outputPane` selector. `frontend/e2e/mocks/runtime.ts:339-346` commented `// RunInTerminal`, `// GetExecutionHistory`, `// ClearExecutionHistory`. | **Remove if no test consumes them.** Verify with grep. |
| Locale keys | `frontend/src/locales/en.json`: `common.copyLastOutput` (line 10), `commandDetail.runInTerminal` (line 48), `outputPane.*` (lines 175-189), `historyPane.*` (lines 167-174) | **Remove** (no consumers after `OutputPane` is deleted and `RunInTerminal` UI button is gone). |

**Nothing found in category "Stored data" beyond `executions` table:** No other tables store command-output history; `terminal_sessions` is owned by Phase 22 and is out of scope for Phase 24.

## Common Pitfalls (Summary)

See the dedicated Common Pitfalls section above for the full list. The two highest-impact items for the planner:

1. **Test setup for `terminalSvc`** (Pitfall 1) — the existing `execution_service_test.go` will panic without a test helper that calls `TerminalService.ServiceStartup`.
2. **Clean removal of `OutputPane.tsx` + i18n keys** (Pitfall 2) — easy to miss the `e2e/utils/selectors.ts` and `e2e/mocks/runtime.ts` references.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `RunCommand` emits `cmd-executing` event; frontend `Terminal.tsx` subscribes and calls `Write(sessionId, cmdLine)` | `RunCommand` calls `terminalSvc.Write(sessionId, cmdLine)` directly in-process | Phase 24 (this phase) | Removes one event hop, eliminates a class of timing/race bugs (frontend subscription not yet attached), removes dead `cmd-executing` plumbing. |
| External terminal launch via `RunInTerminal` (osascript for Terminal.app, iTerm2, Warp; native bins for Alacritty/Kitty/Ghostty) | Removed; only PTY-backed sessions | Phase 24 (this phase) | User mental model: "I have a terminal open, I click Run on a command, it runs there." Aligns with D-03. |
| `OutputPane.tsx` with streaming output + `cmd-output` events | Removed; output flows through `pty-output:{sessionId}` → xterm.js | Phase 23 (already completed for UI) + Phase 24 (cleanup) | Single source of truth for output. No more "did the command go to the pane or the terminal?" confusion. |
| Phase 18's `terminalSvc == nil` early-return for graceful degradation (used during pre-Phase-21 transition) | `terminalSvc` is always set after `TerminalService.ServiceStartup` | Phase 21 | Phase 24's `RunCommand` can assume `terminalSvc != nil` for normal operation, but should still defensively check for cold-start races (Pitfall 5). |

**Deprecated/outdated:**
- **`ExecutionRecord` as a persistent record (D-09):** No longer written to SQLite. The struct remains as the in-process return type for error reporting + tests.
- **`RunInTerminal` method (D-03):** Removed from `execution_service.go`; bindings regenerate without it.
- **`OutputPane` component:** Deleted; no replacement needed (terminal is the output).
- **`cmd-executing` event:** Removed from `event_service.go` EventNames, removed from `events.ts` consumer, removed from `Terminal.tsx` subscription.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `terminalSvc` is reliably non-nil by the time the user clicks Run in normal operation (because `App.tsx:316-336` calls `ListSessions` and creates a default session on mount, and `TerminalService.ServiceStartup` sets the global). | Architecture Patterns §Pattern 1 | If true cold-start race exists, the defensive `if terminalSvc == nil` check (Pitfall 5) is required. The current architecture already guards against this, so the check is belt-and-suspenders. |
| A2 | `TerminalService.Write` auto-resume (D-02) is sufficient — no explicit `Start` call needed in `RunCommand`. | Architecture Patterns §Pattern 2 | If `Write`'s auto-resume is broken in some platform case (e.g., Windows ConPTY edge case), the user would see an opaque "terminal not started" error. The auto-resume code is in `terminal_service.go:563-567` and is exercised by Phase 23's terminal-tab UI today. |
| A3 | Tests should set up a real `TerminalService` (via `ServiceStartup`) rather than mocking it. | Common Pitfalls §Pitfall 1 | If a real PTY is undesirable in CI (e.g., Linux containers without `/dev/ptmx`), `ServiceStartup` will fail. The `testing.Short()` guard pattern from `TestTerminalService_ServiceStartupAssignsTerminalSvc` can be reused to skip. |
| A4 | `OutputPane.tsx` deletion is safe — no test or runtime code imports it. | Common Pitfalls §Pitfall 2 | If a test file imports it (e.g., snapshot test not in current scan), `pnpm tsc --noEmit` will fail. Grep before delete. |
| A5 | Removing the `outputPane.*`, `historyPane.*`, and `common.copyLastOutput` i18n keys is safe. | Common Pitfalls §Pitfall 2 | If any component file uses `t('outputPane.XXX')` after the `OutputPane` deletion (none found in current scan), i18n lookup will return the key as a string (graceful, but visible). |
| A6 | The generated bindings (after `wails3 generate bindings`) will drop `RunInTerminal` and the `CmdExecuting` field without any code-side update. | Common Pitfalls §Pitfall 3 | If a stale import of `RunInTerminal` exists anywhere in `frontend/src/`, `pnpm tsc --noEmit` will fail. Grep before regenerating. |
| A7 | The user-facing behavior of Ctrl+C (sending 0x03 to PTY) is already correct without any Phase 24 code. | Architecture Patterns §Pattern 2, D-06 of Phase 21 | If a bug exists in xterm.js's `convertEol`/`onData` handling of control characters, the user can't interrupt. Verified: xterm.js v5.5.0 transmits raw bytes including 0x03. |

**If this table is empty:** All claims in this research were verified or cited — no user confirmation needed.

The table is not empty; A1-A3 are design choices the planner should confirm if it considers alternatives (e.g., a mock-based test design).

## Open Questions

1. **Should `GetExecutionHistory` / `ClearExecutionHistory` be deleted too, or deferred?**
   - What we know: D-09 says "do not record" but doesn't explicitly say "delete the read paths." `execution_service.go:167-180` still exposes them. `frontend/bindings/cmdex/executionservice.js` will still expose them after regen.
   - What's unclear: Are they used anywhere on the frontend? (Initial scan: no `GetExecutionHistory` / `ClearExecutionHistory` imports in `frontend/src/`.)
   - Recommendation: The plan should delete them for code hygiene, OR explicitly defer to a future phase. Lean toward deletion in this phase since the bindings will regenerate and the orphan methods add no value.

2. **Should `OutputPane.tsx`'s CSS rules in `style.css:1798-1940` be deleted?**
   - What we know: 142 lines of `.output-pane*` styles with no DOM target after the component is gone.
   - What's unclear: None — dead CSS adds bundle size.
   - Recommendation: Delete them as part of this phase. Small, safe cleanup.

3. **Should the `historyPane` and `outputPane` i18n sections be removed in the same commit, or separately?**
   - What we know: Both are unreferenced after the `OutputPane` deletion + `RunInTerminal` UI removal.
   - What's unclear: The user's tolerance for "drive-by" changes (per the AGENTS.md "Surgical Changes" rule).
   - Recommendation: Remove in the same commit as the component deletion. They are co-located dead code.

## Environment Availability

**Step 2.6: SKIPPED.** This phase is a refactor of existing code using existing libraries and tools. No new external dependencies are introduced.

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `wails3` CLI | `wails3 generate bindings` / `wails3 dev` / `wails3 build` | ✓ (already used in Phase 21, 23) | current | — |
| `go test` | `execution_service_test.go` | ✓ | go 1.25.0 per `go.mod` | — |
| `pnpm tsc --noEmit` | Frontend type-check after binding regen | ✓ | current | — |
| PTY (Unix: `creack/pty`, Windows: ConPTY) | `TerminalService.ServiceStartup` in tests | ✓ (in normal use) | creack/pty v1.1.11 | Tests can `t.Skip` on environments without PTY support (`testing.Short()`). |

**Missing dependencies with no fallback:** None.

**Missing dependencies with fallback:** None — all required tools are present.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go `testing` (standard library). No frontend test runner. |
| Config file | None — Go discovers `*_test.go` automatically. |
| Quick run command | `go test ./... -run TestRunCommand` |
| Full suite command | `go test ./...` + `cd frontend && pnpm tsc --noEmit` (current CI gate via `make check`) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| EXEC-01 | Clicking Run on a saved command executes it in the active session's terminal | Integration (PTY round-trip) | `go test ./... -run TestRunCommand_ExecutesOnActiveSession` (NEW) | ❌ Wave 0 |
| EXEC-02 | Command variables (CEL defaults, env, prompts) are resolved before sending to session | Unit (cmdLine construction) | `go test ./... -run TestRunCommand_FinalCmdMultilineScript` (EXISTING, lines 97-113) + new `TestRunCommand_VariableResolution` | ✅ EXISTING (multiline covers it implicitly) |
| EXEC-03 | Command working directory is applied (per-command → global default → session cwd) | Unit | `go test ./... -run TestRunCommand_FinalCmdWithWorkingDir` (EXISTING, lines 50-66) + `TestRunCommand_FinalCmdNoWorkingDir` (EXISTING, lines 68-95) | ✅ EXISTING |
| EXEC-05 | Command output streams to the active session's terminal in real-time with ANSI support | Integration (PTY round-trip + namespaced event) | Manual verification only (no test runner). The PTY readLoop (`terminal_service.go:383-440`) and emitter (`terminal_service.go:443-484`) are Phase 21 code, not Phase 24 changes. | ❌ Manual only |
| EXEC-06 | User can press Ctrl+C to interrupt a running command in the active session | Manual | Verify xterm.js `term.onData` sends `0x03` byte via `Write` to PTY. | ❌ Manual only |

### Sampling Rate

- **Per task commit:** `go build ./... && cd frontend && pnpm tsc --noEmit` (current `make check` gate).
- **Per wave merge:** `go test ./... -v` (full Go test suite).
- **Phase gate:** Full `go test ./...` green + manual UAT of the 5 success criteria in `wails3 dev` before `/gsd-verify-work`.

### Wave 0 Gaps

- [ ] `execution_service_test.go` — update `testDBCreateCommand` (or add `testWithTerminalSvc` helper) to set up `terminalSvc` via `ServiceStartup` before each `RunCommand` test. The existing 4 tests (`TestRunCommand_FinalCmdWithWorkingDir`, `TestRunCommand_FinalCmdNoWorkingDir`, `TestRunCommand_FinalCmdMultilineScript`, `TestRunCommand_NoHistoryPersistence`) will all need this setup to avoid nil-deref after the Phase 24 refactor.
- [ ] `execution_service_test.go` — NEW `TestRunCommand_ExecutesOnActiveSession` (EXEC-01): set up a real `TerminalService`, call `RunCommand` with a benign command (`true` or `echo hello`), assert `record.Error == ""` and (optionally) capture PTY output via a `pty-output` listener.
- [ ] `execution_service_test.go` — NEW `TestRunCommand_NoActiveSession` (EXEC-01 edge case): set up a `TerminalService` with no sessions, call `RunCommand`, assert `record.Error` contains "no active" and `record.ExitCode == -1`.
- [ ] `execution_service_test.go` — NEW `TestRunCommand_NilTerminalSvc` (cold-start race): with `terminalSvc = nil`, call `RunCommand`, assert `record.Error` contains "terminal service" and `record.ExitCode == -1`.
- [ ] Optional: extract `buildCmdLine` as a separate function to make the cmdLine construction testable without a live `TerminalService`. The existing 4 cmdLine tests can be retargeted to this helper.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V2 Authentication | No | Desktop app, no auth surface in this phase. |
| V3 Session Management | Partial | "Session" here is a PTY session, not a user-auth session. The session lifecycle (create, write, close) is already covered by Phase 21's mutex-locked `sessions` map. Phase 24 does not change auth or session ownership. |
| V4 Access Control | No | All commands run in the user's local shell with the user's env (`cmd.Env = os.Environ()` in `terminal_unix.go:19`). No new access control surface. |
| V5 Input Validation | Yes | Variable values are interpolated into the command line without escaping (existing pattern: `ReplaceTemplateVars` does straight string replacement at `script.go:54-63`). This is the established behavior — terminal sessions are interactive shells where the user types arbitrary commands anyway, so a maliciously-crafted command saved in the DB is no worse than typing it into a terminal. **Mitigation:** the value is sent to a PTY (not `exec.Command` with shell-interpreted args), so injection requires the user to have saved a malicious script. Documented limitation, not a Phase 24 regression. |
| V6 Cryptography | No | No crypto operations. |
| V7 Error Handling and Logging | Yes | `terminalSvc.Write` errors are propagated via `ExecutionRecord.Error`. Frontend surfaces via existing toast. No new logging surface. |
| V8 Data Protection | No | No persistent data added; `executions` table writes are now skipped (D-09). |

### Known Threat Patterns for Wails + PTY Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|--------------------|
| Malicious script body saved in DB | Tampering / Elevation of Privilege | Out of scope — saving a malicious command is equivalent to the user typing it. No mitigation needed in `RunCommand`. |
| Variable value contains shell metacharacters (`; rm -rf /`) | Tampering | Existing pattern. `ReplaceTemplateVars` does string replacement, then the result is sent to a PTY where the user's shell interprets it. This is the intended behavior for "execute a CLI command." If we wanted defense-in-depth, we could `shellQuoteArg` each value, but that breaks legitimate use cases (e.g., passing a path with a literal `*` glob). Documented trust boundary. |
| `cmd-executing` event injection (now that the event is removed) | N/A | Removing the event eliminates the attack surface. The new direct `Write` call is in-process; no IPC channel. |

**No new security surface introduced by Phase 24.** The refactor actually *reduces* attack surface by removing a global event channel.

## Sources

### Primary (HIGH confidence)

- `.planning/phases/24-session-aware-execution/24-CONTEXT.md` — locked decisions, integration points, canonical refs
- `.planning/phases/24-session-aware-execution/24-DISCUSSION-LOG.md` — discussion alternatives (D-01..D-09)
- `execution_service.go:114-148` — current `RunCommand` implementation (event-based dispatch)
- `terminal_service.go:554-582` — `Write` method with auto-resume (satisfies D-02)
- `terminal_service.go:127` — `terminalSvc = s` global assignment
- `terminal_service.go:289-304` — `GetActiveSession` (D-01 active session lookup)
- `event_service.go:10-24` — `EventNames` struct + values to modify
- `frontend/src/components/Terminal.tsx:246-254` — `term.onData` → `Write` (Ctrl+C plumbing)
- `frontend/src/components/Terminal.tsx:309-345` — namespaced PTY events + `cmd-executing` subscription
- `frontend/src/components/OutputPane.tsx` (full file) — orphan component to delete
- `frontend/src/wails/events.ts:1-24` — `eventNames` object to modify
- `execution_service_test.go:50-200` — existing tests that need test setup update
- `.planning/phases/21-backend-session-foundation/21-02-PLAN.md:238` — Phase 21's "clean break" event removal precedent
- `.planning/phases/21-backend-session-foundation/21-03-PLAN.md:104-115` — `cmdExecuting` keep-pattern (predecessor to Phase 24's removal)
- `.planning/research/ARCHITECTURE.md:60-67` — `wailsApp.Event.Emit("pty-output:"+sessionId, ...)` namespacing pattern
- `.planning/research/PITFALLS.md:5-10` — "global state collision" pitfall (prevention: use `terminalSvc` global, not a service pointer in a struct)
- `.planning/codebase/CONVENTIONS.md:11-20` — Go and TypeScript naming conventions to follow in the refactor

### Secondary (MEDIUM confidence)

- `app.go:13-18` — package-level globals (`db`, `executor`, `wailsApp`, `terminalSvc`) — confirms the cross-service coordination pattern
- `frontend/src/App.tsx:981-1005` — `runCommandDirect` (the frontend consumer that needs no semantic change, just confirms return value contract)
- `frontend/src/App.tsx:316-336` — session list/active-session bootstrap on mount (proves the timing assumption in A1)
- `models.go:133-144` — `ExecutionRecord` struct fields to keep (don't remove `FinalCmd`, `Error`, `ExitCode`)

### Tertiary (LOW confidence)

- None. All claims in this research are grounded in the project's own source files and prior phase artifacts.

## Metadata

**Confidence breakdown:**

| Area | Level | Reason |
|------|-------|--------|
| Standard Stack | HIGH | No new libraries; all changes use Phase 21/23 infrastructure that already exists in the repo. |
| Architecture | HIGH | The dispatch path is a one-line change at the producer (replace `wailsApp.Event.Emit(...)` with `terminalSvc.Write(...)`); the consumer (xterm.js) is unchanged. Verified by reading the full `RunCommand` and `Write` source. |
| Pitfalls | HIGH | Five pitfalls identified with concrete detection signals (test panic, tsc errors, file delete verification). Mirrors Phase 21's pitfalls (`.planning/research/PITFALLS.md:1-10`). |
| Tests | HIGH | The existing 4 `TestRunCommand_*` tests give a clear baseline. The `TestTerminalService_ServiceStartupAssignsTerminalSvc` precedent shows exactly how to set up `terminalSvc` for tests. The cold-start race (Pitfall 5) is the only thing that needs a new defensive test. |
| Security | HIGH | No new surface. The refactor removes an event channel. Variable interpolation is the existing trust model, not a Phase 24 concern. |

**Research date:** 2026-06-15

**Valid until:** 2026-07-15 (30 days). The architecture is stable; no fast-moving dependencies. The only risk is the Wails v3 binding regeneration pipeline changing, which is unrelated to this phase's logic.
