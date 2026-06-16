# Phase 24: Session-Aware Execution - Pattern Map

**Mapped:** 2026-06-15
**Files analyzed:** 13 new/modified + 3 deleted/regenerated
**Analogs found:** 11 / 13

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `execution_service.go` | service | request-response (Wails IPC) | self (refactor) | exact (in-place edit) |
| `event_service.go` | service | const registry | self (refactor) | exact (in-place edit) |
| `execution_service_test.go` | test | service-singleton setup | `TestTerminalService_ServiceStartupAssignsTerminalSvc` (in same file, lines 176-204) | exact |
| `frontend/src/components/Terminal.tsx` | component | event subscription cleanup | self (in-place edit, lines 334-345, 351) | exact |
| `frontend/src/wails/events.ts` | utility | const init | self (in-place edit) | exact |
| `frontend/src/locales/en.json` | config (i18n) | key removal | self (in-place edit) | exact |
| `frontend/src/style.css` | config (CSS) | rule removal | self (in-place edit) | exact |
| `frontend/src/App.tsx` | component | run-button flow (no change to logic) | self (in-place edit at lines 981-1005) | exact |
| `frontend/e2e/utils/selectors.ts` | test utility | selector removal | self (in-place edit at line 40) | exact |
| `frontend/e2e/mocks/runtime.ts` | test mock | binding mock removal | self (in-place edit at lines 339-349) | exact |
| `frontend/src/components/OutputPane.tsx` | component | (deleted) | n/a (entire file removal) | deletion |
| `frontend/bindings/cmdex/executionservice.js` | binding (generated) | regen after `RunInTerminal` removal | n/a (auto-generated) | regenerated |
| `frontend/bindings/cmdex/eventservice.js` | binding (generated) | regen after `CmdExecuting` removal | n/a (auto-generated) | regenerated |

> **Phase 24 is a deletion-heavy refactor.** Almost every change is a removal or a small in-place edit. The "new" code is one call-site replacement (`wailsApp.Event.Emit(...)` → `terminalSvc.Write(...)`) and one new test helper.

---

## Pattern Assignments

### `execution_service.go` (service, request-response) — EDIT

**Analog:** self — `execution_service.go:114-148` (the current `RunCommand` that emits the event).

**File to edit:** `execution_service.go`

**The change in `RunCommand` (lines 114-148):** Replace the `wailsApp.Event.Emit(eventNames.CmdExecuting, ...)` block with a direct call to `terminalSvc.Write`.

**Existing imports to keep** (lines 1-12):
```go
import (
    "context"
    "fmt"
    "os"
    "strings"
    "time"

    "github.com/google/uuid"
    "github.com/wailsapp/wails/v3/pkg/application"
)
```

**Helper functions to keep verbatim (no edits):**

`resolveWorkingDir` — `execution_service.go:28-60` — Per-command → global default → home → cwd → tempdir. Satisfies D-06.

`hasExplicitWorkingDir` — `execution_service.go:97-109` — Decides whether to prefix with `cd %s &&`.

**`shellQuoteDir` helper** — `executor.go:75-81` — Used as-is. `TestShellQuoteDir` (line 155) covers edge cases.
```go
// Source: executor.go:75-81
func shellQuoteDir(dir string) string {
    if !strings.Contains(dir, `'`) {
        return `'` + dir + `'`
    }
    escaped := strings.ReplaceAll(dir, `'`, `'"'"'`)
    return `'` + escaped + `'`
}
```

**`stripShebang` helper** — `executor.go:40-49` — Used as-is.

**Refactored `RunCommand` (template to copy from — based on `terminal_service.go:554-582` Write signature and `execution_service.go:114-148`):**
```go
// Source: cmamer/terminal_service.go:554-582 (Write signature) — analog for the new dispatch
// Source: cmamer/execution_service.go:114-148 (RunCommand body to preserve) — analog for cmdLine construction
func (s *ExecutionService) RunCommand(commandID string, variables map[string]string) ExecutionRecord {
    cmd, err := db.GetCommand(commandID)
    if err != nil {
        return ExecutionRecord{ID: uuid.New().String(), Error: err.Error(), ExitCode: -1}
    }

    resolvedScript := ReplaceTemplateVars(cmd.ScriptContent, variables)
    resolvedScript = stripShebang(resolvedScript)
    resolvedScript = strings.TrimRight(resolvedScript, "\n")
    workingDir := s.resolveWorkingDir(cmd)

    var cmdLine string
    if s.hasExplicitWorkingDir(cmd) {
        cmdLine = fmt.Sprintf("cd %s && %s\n", shellQuoteDir(workingDir), resolvedScript)
    } else {
        cmdLine = resolvedScript + "\n"
    }

    // NEW: direct in-process call to terminalSvc (replaces the cmd-executing event)
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

**Methods to DELETE (per D-03, D-04, and the Open Question #1 in RESEARCH.md):**
- `RunInTerminal` — `execution_service.go:151-165` — Removed entirely. The frontend bindings regenerate without it.
- `GetExecutionHistory` — `execution_service.go:168-175` — Removed (Open Question #1: lean toward deletion).
- `ClearExecutionHistory` — `execution_service.go:178-180` — Removed (same reasoning).

**Imports to remove after deletions:** `application` is still used by `ServiceStartup`; keep. `fmt` may become unused if you delete `GetExecutionHistory` (it has a `fmt.Println`); verify with `go build`.

---

### `event_service.go` (service, const registry) — EDIT

**Analog:** self — `event_service.go:10-24` (the current `EventNames` struct and `eventNames` var).

**The change:** Remove the `CmdExecuting` field and its value (D-04 from Phase 21 clean-break pattern).

**Before (lines 10-24):**
```go
type EventNames struct {
    CmdExecuting          string `json:"cmdExecuting"`
    OpenSettings          string `json:"openSettings"`
    OpenShortcuts         string `json:"openShortcuts"`
    SettingsChanged       string `json:"settingsChanged"`
    SettingsWindowClosing string `json:"settingsWindowClosing"`
}

var eventNames = EventNames{
    CmdExecuting:          "cmd-executing",
    OpenSettings:          "open-settings",
    OpenShortcuts:         "open-shortcuts",
    SettingsChanged:       "settings-changed",
    SettingsWindowClosing: "settings-window-closing",
}
```

**After:**
```go
type EventNames struct {
    OpenSettings          string `json:"openSettings"`
    OpenShortcuts         string `json:"openShortcuts"`
    SettingsChanged       string `json:"settingsChanged"`
    SettingsWindowClosing string `json:"settingsWindowClosing"`
}

var eventNames = EventNames{
    OpenSettings:          "open-settings",
    OpenShortcuts:         "open-shortcuts",
    SettingsChanged:       "settings-changed",
    SettingsWindowClosing: "settings-window-closing",
}
```

`GetEventNames` (lines 35-37) is unchanged — it just returns the struct.

---

### `execution_service_test.go` (test) — EDIT

**Analog:** `TestTerminalService_ServiceStartupAssignsTerminalSvc` at `execution_service_test.go:176-204` (same file) — the existing precedent for setting up `terminalSvc` in tests.

**The change:** Add a `testWithTerminalSvc` helper that wraps `testDBCreateCommand` and sets up a real `TerminalService`. Update all 4 existing `TestRunCommand_*` tests to use it. Add 3 new tests (NilTerminalSvc, NoActiveSession, ExecutesOnActiveSession).

**Existing `testDBCreateCommand` (lines 16-48) — keep as the base, wrap it.**

**New test helper to add (template to copy from `execution_service_test.go:181-190`):**
```go
// Source: cmamer/execution_service_test.go:181-190 (precedent for save/restore of terminalSvc)
func testWithTerminalSvc(t *testing.T) func() {
    t.Helper()
    if testing.Short() {
        t.Skip("skipping integration test in short mode (requires real PTY)")
    }
    prevTerminalSvc := terminalSvc
    terminalSvc = nil
    ts := &TerminalService{}
    if err := ts.ServiceStartup(nil, application.ServiceOptions{}); err != nil {
        terminalSvc = prevTerminalSvc
        t.Skipf("TerminalService.ServiceStartup failed: %v", err)
    }
    return func() {
        _ = ts.ServiceShutdown()
        terminalSvc = prevTerminalSvc
    }
}
```

**Existing tests to update** (all four call `RunCommand` which will now need a non-nil `terminalSvc`):
- `TestRunCommand_FinalCmdWithWorkingDir` (lines 50-66) — after `testDBCreateCommand`, call `defer testWithTerminalSvc(t)()`.
- `TestRunCommand_FinalCmdNoWorkingDir` (lines 68-95) — same.
- `TestRunCommand_FinalCmdMultilineScript` (lines 97-113) — same.
- `TestRunCommand_NoHistoryPersistence` (lines 137-153) — same. **Note:** D-09 says no history persistence; the test will now pass trivially because `terminalSvc.Write` is called and there's no `db.AddExecution` anywhere. Keep the test to lock in the no-persistence contract.

**`TestShellQuoteDir` (lines 155-174) — NO change.** Pure function test, doesn't touch `terminalSvc`.

**`TestRunCommand_GetCommandError` (lines 115-135) — needs the new helper too**, because the path through `RunCommand` still hits `terminalSvc` after `db.GetCommand` succeeds; but since the test expects early return on `GetCommand` failure, the `terminalSvc` check is never reached. Still, for safety add the helper.

**New tests to add (per RESEARCH.md Wave 0 Gaps):**
```go
// Source: cmamer/execution_service_test.go:176-204 (analog for service setup)
// Plus: execution_service.go:202-211 (analog for the nil/session error patterns)
func TestRunCommand_NilTerminalSvc(t *testing.T) {
    prevTerminalSvc := terminalSvc
    terminalSvc = nil
    defer func() { terminalSvc = prevTerminalSvc }()

    initDB, err := NewDB()
    if err != nil { t.Skipf("cannot open test DB: %v", err) }
    defer initDB.Close()
    prevDB := db; db = initDB
    defer func() { db = prevDB }()

    _, cleanup := testDBCreateCommand(t, "test-cat-nilsvc-24", "test-cmd-nilsvc-24", "T", "T", "echo hi", `{}`)
    defer cleanup()

    svc := &ExecutionService{}
    record := svc.RunCommand("test-cmd-nilsvc-24", nil)
    if record.Error == "" {
        t.Error("expected Error to be set when terminalSvc is nil")
    }
    if record.ExitCode != -1 {
        t.Errorf("ExitCode = %d, want -1", record.ExitCode)
    }
}

func TestRunCommand_NoActiveSession(t *testing.T) {
    defer testWithTerminalSvc(t)()
    // After testWithTerminalSvc, terminalSvc has 1 active session.
    // To test the "no active" path, manually clear activeSessionID:
    //   ts := terminalSvc
    //   ts.mu.Lock(); ts.activeSessionID = ""; ts.mu.Unlock()
    // (Cast via the file's own terminalSvc global, which is package-level.)
    // ... call RunCommand, assert Error contains "no active", ExitCode == -1
}
```

> **Test design constraint:** the `terminalSvc` global is shared across tests in this package. Use `t.Cleanup()` to restore state, not just `defer`. Reference: `execution_service_test.go:181-183` shows the `prevTerminalSvc` pattern.

---

### `frontend/src/components/Terminal.tsx` (component) — EDIT

**Analog:** self — `Terminal.tsx:334-345, 351` (the `cleanupCmdExecuting` subscription block to remove).

**The change:** Remove the `cmd-executing` event subscription entirely. The backend now calls `terminalSvc.Write` directly, so this consumer has no producer.

**Block to REMOVE (lines 334-345):**
```tsx
// Source: cmamer/frontend/src/components/Terminal.tsx:334-345
const cleanupCmdExecuting = Events.On(eventNames.cmdExecuting, (event: { data: { data: string } }) => {
  if (activeSessionIdRef.current !== sessionIdRef.current) return;
  const cmdLine = event?.data?.data;
  if (cmdLine && backendAvailableRef.current) {
    Write(sessionIdRef.current, cmdLine).catch((err) => {
      console.error('TerminalService.Write failed:', err);
      if (backendAvailableRef.current) {
        backendAvailableRef.current = false;
      }
    });
  }
});
```

**Also remove (line 351):** `cleanupCmdExecuting();` from the cleanup `return`.

**Cleanup pass after the edit:**
- `eventNames` import (line 8) — verify with `grep -n eventNames frontend/src/components/Terminal.tsx` after the edit. If `eventNames` is no longer referenced anywhere in this file, remove the import too. **No other use in this file** — the only `Events.On` calls left use hardcoded event names (`'pty-output:' + sessionId`, `'pty-exit:' + sessionId`, `'pty-cleared:' + sessionId`).
- `activeSessionId` prop (line 16) and `activeSessionIdRef` (line 40-43) — verify with `grep -n activeSessionId frontend/src/components/Terminal.tsx`. **The only use was the guard in `cleanupCmdExecuting`.** Remove the prop from the interface, from the destructuring (line 28), and from the call sites in `App.tsx:1601`. This is a breaking change to the `TerminalComponent` API — must update `App.tsx`.

**Keep as-is:**
- Lines 246-254: `term.onData` → `Write(sessionIdRef.current, data)` — Ctrl+C keystroke path. Unchanged. (D-06 of Phase 21, EXEC-06.)
- Lines 305-318: `pty-output:{sessionId}` subscription. Unchanged. (EXEC-05.)
- Lines 320-328: `pty-exit:{sessionId}` subscription. Unchanged.
- Lines 330-332: `pty-cleared:{sessionId}` subscription. Unchanged.

---

### `frontend/src/wails/events.ts` (utility) — EDIT

**Analog:** self — `events.ts:1-24` (the whole file is small enough to be a single edit).

**The change:** Remove `cmdExecuting` from the const map and from `initEventNames()`.

**Before (line 8):**
```typescript
cmdExecuting: 'cmd-executing',
```

**Before (line 19):**
```typescript
eventNames.cmdExecuting = names.cmdExecuting;
```

**After:** Delete both lines. The TypeScript type of the `eventNames` const is inferred — no type annotation to update.

---

### `frontend/src/locales/en.json` (config, i18n) — EDIT

**Analog:** self — top-level keys to delete are co-located dead code.

**Keys to DELETE (per RESEARCH.md Runtime State Inventory):**
- Line 10: `"copyLastOutput": "Copy last output",` (in `common` block)
- Line 48: `"runInTerminal": "Run in Terminal",` (in `commandDetail` block — verify exact location, then delete)
- Lines 167-174: entire `historyPane` block
- Lines 175-189: entire `outputPane` block

**Verification before deleting:** Grep `frontend/src/` for `t('outputPane.`, `t('historyPane.`, `t('common.copyLastOutput')`, `t('commandDetail.runInTerminal')` — must return zero matches (the components that consumed them are deleted). Per RESEARCH.md initial scan: zero consumers expected.

**Reference for app's existing toast pattern (line 992, App.tsx) — uses `t('toast.commandFailed')`, kept:**
```typescript
// Source: App.tsx:992 — toast pattern to preserve after the refactor
toast.error(t('toast.commandFailed', { code: result.exitCode ?? -1 }));
```

---

### `frontend/src/style.css` (config, CSS) — EDIT

**Analog:** self — `.output-pane*` rules at lines 1798-1940 (142 lines, no DOM target after component deletion).

**Action:** DELETE the entire `.output-pane` CSS block. Verify with `grep -n 'output-pane\|output-' frontend/src/` — should return zero matches after the file deletion.

---

### `frontend/src/App.tsx` (component) — MINOR EDIT

**Analog:** self — `App.tsx:981-1005` (the `runCommandDirect` function — keep the body, but no callers in this file reference `OutputPane` or `cmd-output`).

**No change to `runCommandDirect` itself.** The flow `RunCommand` → toast on error is correct for Phase 24's "write to PTY, no history" model.

**Edits required to keep the build green after sibling changes:**

1. **`TerminalComponent` prop removal** — `App.tsx:1524` and `App.tsx:1601` pass `activeSessionId={activeSessionId}` to `<TerminalComponent>`. After removing that prop from the component's interface, remove these two prop assignments:
   ```tsx
   // REMOVE from App.tsx:1524
   activeSessionId={activeSessionId}

   // REMOVE from App.tsx:1601
   activeSessionId={activeSessionId}
   ```
   **Keep** `sessionId={id}` (line 1598) — that prop is still used.

2. **`copyLastOutput` tooltip** — `App.tsx:1572`:
   ```tsx
   {t('common.copyLastOutput')}
   ```
   Delete this line and the surrounding tooltip wrapper (D-04 — no more output pane to copy from). The exact surrounding context should be reviewed with `read -offset 1560 -limit 30 frontend/src/App.tsx` before editing.

3. **Imports** — verify `OutputPane` is not imported in `App.tsx` already (initial scan shows no matches — Phase 23 already removed it from the layout). No import to remove.

**Keep as-is:**
- `runCommandDirect` body (lines 981-1005) — unchanged. The `setIsExecuting`/`expandTerminal` state changes remain because they drive the "is this tab currently running" UI in `CommandDetailTab.tsx`.
- `ListSessions` → `GetActiveSession` → `CreateSession` chain (lines 316-334) — unchanged. Already creates a default session on mount.
- `createTerminalSession` / `closeTerminalSession` / `selectTerminalSession` callbacks (lines 193-260) — unchanged.

---

### `frontend/e2e/utils/selectors.ts` (test utility) — EDIT

**Analog:** self — `frontend/e2e/utils/selectors.ts:40`.

**The change:** Delete the `outputPane` selector entry.

**Before (line 40):**
```typescript
outputPane: '[data-testid="output-pane"]',
```

**Before deleting, verify no e2e test consumes it:**
```bash
grep -rn 'outputPane' frontend/e2e/tests/
```
(Expected output: zero matches. If matches exist, the e2e test is dead code that should also be removed — but that is out of scope for Phase 24's code; the deletion is safe regardless.)

---

### `frontend/e2e/mocks/runtime.ts` (test mock) — EDIT

**Analog:** self — `frontend/e2e/mocks/runtime.ts:339-349`.

**The change:** Remove the commented `RunInTerminal` / `GetExecutionHistory` / `ClearExecutionHistory` blocks.

**Before (lines 339-349):**
```typescript
// RunInTerminal(commandID, variables)
1736747747: () => {},

// ── History ──────────────────────────────────────────────
// GetExecutionHistory
2752844091: () => executionHistory,

// ClearExecutionHistory
3022740230: () => {
  executionHistory = [];
},
```

**After:** Delete the entire block. Also remove any now-unused local state (e.g., the `executionHistory` array and the `now()` helper if their only consumer was the deleted mock block — verify with `grep -n executionHistory frontend/e2e/mocks/runtime.ts`).

---

### `frontend/src/components/OutputPane.tsx` (component) — DELETE

**Analog:** none — the file is the orphan. Reference: `.planning/codebase/ARCHITECTURE.md` and the RESEARCH.md Runtime State Inventory.

**Action:** `rm frontend/src/components/OutputPane.tsx`

**Pre-delete verification (Pitfall 2 from RESEARCH.md):**
```bash
grep -rn "from.*OutputPane" frontend/src/   # must be zero
grep -rn "OutputPane" frontend/src/         # must be zero
```

The initial scan (RESEARCH.md) shows zero consumers. Delete is safe.

---

### `frontend/bindings/cmdex/executionservice.js` (binding, generated) — REGEN

**Analog:** self — current file has `RunInTerminal` (lines 67-69), `GetExecutionHistory` (lines 30-34), `ClearExecutionHistory` (lines 22-24).

**Action:** Run `wails3 generate build-assets` (or `wails3 generate bindings`) after editing `execution_service.go`. The regenerated file will drop the three deleted methods.

**Pre-regen verification (Pitfall 3 from RESEARCH.md):**
```bash
grep -rn "RunInTerminal\|GetExecutionHistory\|ClearExecutionHistory" frontend/src/
```
Expected: zero matches. If a stale import exists, `pnpm tsc --noEmit` will fail with a missing export error.

---

### `frontend/bindings/cmdex/eventservice.js` (binding, generated) — REGEN

**Analog:** self — current file exports `GetEventNames` which returns an `EventNames` struct. After regen, the struct drops the `cmdExecuting` field.

**Action:** Same `wails3 generate build-assets` invocation regenerates this file too.

**Cascade check:** `frontend/bindings/cmdex/models.js` (21.6K) has a generated `EventNames.createFrom` type definition. The regen will update it to drop the `cmdExecuting` field. The frontend `events.ts` no longer references `names.cmdExecuting` after our edit, so no compile error.

---

## Shared Patterns

### Pattern A: Package-Level Service Singleton (`terminalSvc`)

**Source:** `app.go:13-18` + `terminal_service.go:126-136`

**Apply to:** `execution_service.go` (the new dispatch in `RunCommand`), `execution_service_test.go` (test setup helper).

```go
// Source: app.go:13-18
var (
    db          *DB
    executor    *Executor
    wailsApp    *application.App
    terminalSvc *TerminalService
)

// Source: terminal_service.go:126-136
func (s *TerminalService) ServiceStartup(ctx context.Context, options application.ServiceOptions) error {
    terminalSvc = s           // global assignment on startup
    s.sessions = make(map[string]*sessionState)
    _, err := s.CreateSession()
    if err != nil {
        fmt.Printf("TerminalService: CreateSession failed (graceful degradation): %v\n", err)
    }
    return nil
}
```

**Defensive check pattern** (mirror the existing `db == nil` style in `execution_service.go`):
```go
if terminalSvc == nil {
    return ExecutionRecord{ID: uuid.New().String(), Error: "terminal service not initialized", ExitCode: -1}
}
```

### Pattern B: Save/Restore Test Globals

**Source:** `execution_service_test.go:181-183` (the `prevTerminalSvc` precedent for `terminalSvc`), `execution_service_test.go:41-47` (the `prevDB` precedent for `db`).

**Apply to:** New `testWithTerminalSvc` helper.

```go
// Source: execution_service_test.go:41-47 (analog for the db save/restore pattern)
prevDB := db
db = initDB

return initDB, func() {
    db = prevDB
    initDB.Close()
}
```

**Pattern:** save the previous value of the global, replace it, and return a cleanup function that restores. The new helper does the same for `terminalSvc`.

### Pattern C: TerminalService.Write Auto-Resume (D-02 satisfaction)

**Source:** `terminal_service.go:554-582` (existing, no change required)

**Apply to:** `execution_service.go:RunCommand` — the new dispatch delegates auto-resume to `Write`. **Do NOT call `Start()` separately before `Write()`.**

```go
// Source: terminal_service.go:554-582 — the auto-resume is built in
func (s *TerminalService) Write(sessionId string, data string) error {
    ss, err := s.resolveSession(sessionId)
    if err != nil { return err }

    ss.mu.Lock()
    defer ss.mu.Unlock()

    if !ss.running {                                                // ← D-02 transparent auto-start
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

### Pattern D: `Events.On` Cleanup Pairing

**Source:** `Terminal.tsx:309-352` (the existing per-session subscription pattern with cleanup)

**Apply to:** The remaining 3 subscriptions in `Terminal.tsx` (pty-output, pty-exit, pty-cleared) — the pattern is preserved. Only `cleanupCmdExecuting` is removed.

```tsx
// Source: Terminal.tsx:309-332 (the preserved pattern)
const cleanupOutput = Events.On(ptyOutputEvent, (event: { data: { data: string } }) => { ... });
const cleanupExit = Events.On(ptyExitEvent, (event: { data: { exitCode: number; wasIntentional: boolean } }) => { ... });
const cleanupCleared = Events.On(ptyClearedEvent, () => { ... });

return () => {
    cleanupOutput();   // ← all three cleanup calls preserved
    cleanupExit();
    cleanupCleared();
    // cleanupCmdExecuting();  // ← only this one removed
};
```

### Pattern E: Wails Event Removal (Clean Break)

**Source:** Phase 21's "clean break" precedent (referenced in `.planning/phases/21-backend-session-foundation/21-PATTERNS.md`).

**Apply to:** `event_service.go`, `events.ts`, `Terminal.tsx`, regenerated bindings.

When a previously-emitted event has no remaining consumers:
1. Remove the field from the producer's `EventNames` struct + value.
2. Remove all consumer subscriptions.
3. Regenerate Wails bindings.
4. **No backward-compat shim** — same precedent as Phase 21's `pty-output`/`pty-exit`/`pty-cleared` namespacing.

---

## No Analog Found

Files with no close match in the codebase (planner should use RESEARCH.md patterns instead):

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `frontend/bindings/cmdex/executionservice.js` (regen) | generated | n/a | Auto-generated by `wails3 generate build-assets` — no analog needed; just regenerate. |
| `frontend/bindings/cmdex/eventservice.js` (regen) | generated | n/a | Same as above. |

---

## Metadata

**Analog search scope:**
- `*.go` at repo root (services, models, executor, script)
- `frontend/src/components/*.tsx`
- `frontend/src/wails/*.ts`
- `frontend/src/locales/en.json`
- `frontend/src/style.css`
- `frontend/src/App.tsx`
- `frontend/e2e/**`
- `frontend/bindings/cmdex/*.js`

**Files scanned:** 18 Go files + 23 frontend component/util files + 2 generated bindings + 1 e2e selector file + 1 e2e mock file.

**Pattern extraction date:** 2026-06-15

**Key constraints surfaced during mapping:**
1. **`terminalSvc` is set by `TerminalService.ServiceStartup`** — every `RunCommand` test must initialize a `TerminalService` first. The `prevTerminalSvc` save/restore pattern in `execution_service_test.go:181-183` is the direct template.
2. **The `cmdLine` construction in `RunCommand` stays verbatim** — `shellQuoteDir` + `cd %s && %s\n` format. `TestShellQuoteDir` (line 155) and 3 existing `TestRunCommand_FinalCmd*` tests lock the contract. Do not "improve" it.
3. **`activeSessionId` prop removal from `TerminalComponent` is a breaking change** — the prop has exactly one consumer (the `cleanupCmdExecuting` guard at line 335) and is purely a UI mirror of the backend's authoritative `terminalSvc.GetActiveSession()`. Remove it from the `TerminalComponent` interface and from `App.tsx:1524, 1601`.
4. **`RunInTerminal` removal cascades to regenerated bindings** — pre-regen grep must show zero consumers in `frontend/src/`.
5. **The `OutputPane.tsx` deletion is a deletion of orphan code** — initial scan shows zero imports. Per Pitfall 2 of RESEARCH.md, the file is safe to delete. Cascading cleanup: `style.css` lines 1798-1940, `e2e/utils/selectors.ts:40`, `e2e/mocks/runtime.ts:339-349`, `locales/en.json` lines 10/48/167-189.

**Coordination notes for the planner:**
- The `Open Question #1` from RESEARCH.md (whether to also delete `GetExecutionHistory`/`ClearExecutionHistory`) is resolved toward deletion in this map because:
  1. The `executions` SQLite table can be left untouched (no schema migration needed).
  2. `db.AddExecution` is no longer called from `RunCommand` (D-09).
  3. No frontend imports `GetExecutionHistory`/`ClearExecutionHistory` (initial scan).
  4. The method-binding mock numbers in `runtime.ts` (2752844091, 3022740230) become orphan entries — clean them up in the same edit.
- The `Open Question #2` (delete `.output-pane*` CSS) is resolved toward deletion (142 lines, no DOM target after component removal).
- The `Open Question #3` (commit i18n cleanup with the component deletion) is resolved toward "same commit" because they are co-located dead code.
