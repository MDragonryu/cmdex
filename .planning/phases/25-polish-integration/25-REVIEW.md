---
phase: 25-polish-integration
reviewed: 2026-06-16T08:30:00Z
depth: standard
files_reviewed: 14
files_reviewed_list:
  - pty_backend.go
  - pty_backend_unix.go
  - pty_backend_windows.go
  - pty_backend_mock.go
  - pty_backend_mock_test.go
  - terminal_service.go
  - terminal_service_test.go
  - terminal_service_stress_test.go
  - terminal_service_max_sessions_test.go
  - frontend/src/App.tsx
  - frontend/src/locales/en.json
  - frontend/e2e/mocks/runtime.ts
  - AGENTS.md
  - frontend/src/components/TerminalTabBar.tsx (cross-ref only, no edits)
  - frontend/src/components/Terminal.tsx (cross-ref only, no edits)
findings:
  critical: 0
  warning: 2
  info: 5
  total: 7
status: issues_found
---

# Phase 25: Code Review Report

**Reviewed:** 2026-06-16T08:30:00Z
**Depth:** standard
**Files Reviewed:** 14
**Status:** issues_found

## Summary

Phase 25 is a testability and observability polish: a build-tagged `ptyBackend` interface refactor, a darwin-side in-memory mock, a 100-cycle stress test, global default cwd inheritance, dead-code sweep, a `MaxSessions` capacity guard, a cross-platform error toast, and documentation of the Windows conpty gap. The new code is generally well-scoped and the deviations documented in the summaries (e.g. stress/max-sessions tests placed in `//go:build darwin` files) are necessary for cross-platform compile correctness.

Direct code review found **0 BLOCKERs**, **2 WARNINGs** (one of which is a fragile design choice the team explicitly accepted), and **5 INFO** items. The pre-existing `cmd.Wait()` race and the pre-existing duplicate `readerWg.Wait()` in `startSessionLocked` were not introduced by this phase but are worth flagging because the refactor touched the surrounding code without addressing them.

### Concerns from the brief — all confirmed safe

1. **Mock `sleep 0.05` cmd lifecycle** — `cmd.Start()` is called in `mockPtyBackend.Start` (pty_backend_mock.go:28-31). The returned `*exec.Cmd` is stored in `ss.cmd` and the `monitorExit` goroutine calls `cmd.Wait()` on it. `CloseSession` calls `s.ptyBackend.Kill(oldCmd)` → `cmd.Process.Kill()`. The process is reaped correctly. **No leak** in the current tests because they `defer s.ServiceShutdown()`. *Caveat:* if a test forgets the defer, the cmd leaks; this is documented as INFO.
2. **MaxSessions lock release in error path** — verified at terminal_service.go:183-186. The `s.mu.Unlock()` precedes the `return` statement. **Correct.**
3. **`getWorkingDir()` nil-db safety** — verified at terminal_service.go:92. The function checks `if db != nil` before calling `db.GetSettings()`. Tests with nil `db` fall through to `os.UserHomeDir()`. **Correct.**
4. **TerminalTabBar.tsx:131 / Terminal.tsx:292 guards still in place** — verified. `{!isLastTab && (` at TerminalTabBar.tsx:131, `disabled={isLastTab}` at line 151, `toast.error('Terminal start failed')` at Terminal.tsx:292, and `if (sessions.length <= 1) return; // D-02: last tab not closeable` at App.tsx:214. **All confirmed.**
5. **`//go:build darwin` build tags** — confirmed on all three test files (pty_backend_mock_test.go:1, terminal_service_stress_test.go:1, terminal_service_max_sessions_test.go:1). Matches the verifier's documented deviation note. **Correct.**
6. **Mock Resize / Kill races** — `Resize` takes the per-handle mutex (pty_backend_mock.go:46-49). `Kill` does not touch mock state — only `cmd.Process.Kill()`. **No race.**
7. **Windows stub error pattern** — matches the prior `terminal_windows.go` pattern; *minor* inconsistency in error message suffix (see IN-03).

## Critical Issues

*None.*

## Warnings

### WR-01: Frontend matches Go error by free-form string substring

**File:** `frontend/src/App.tsx:204-206` (and Go source at `terminal_service.go:185`)
**Issue:** The frontend `createTerminalSession` callback detects the max-sessions error by checking `String(err).includes('max sessions reached')`. The Go side produces this error via `fmt.Errorf("CreateSession: max sessions reached (%d)", MaxSessions)` at terminal_service.go:185. There is no typed Go error (e.g. `var ErrMaxSessions = errors.New(...)`) and no exported constant for the error string.

The cost of this pattern:
- A Go-side typo, refactor that wraps the error (`fmt.Errorf("...: %w", err)`), or even just changing the message wording to "session limit reached" silently breaks the localized toast with no compile-time signal. The fallback generic message ("Could not create session. Check that the terminal backend is running.") would surface, but the user would not see the actionable "Close a session to create a new one" hint.
- The hard-coded `limit: 10` on the frontend (App.tsx:206) is also tied to the Go constant `MaxSessions = 10` (terminal_service.go:28). If `MaxSessions` is changed to e.g. 20, the toast still says "(10)" until a coordinated frontend change is made.

The 25-03 SUMMARY acknowledges this trade-off ("the cost is that the match is on a free-form string") and lists hardening as a future pass. The risk is real but low for v2.1 because the error is package-internal and the matching text is stable.

**Fix (future hardening):**

1. In Go, define a sentinel error:
   ```go
   // terminal_service.go
   var ErrMaxSessions = errors.New("max sessions reached")
   ```
2. Use it via wrapping:
   ```go
   return nil, fmt.Errorf("CreateSession: %w (%d)", ErrMaxSessions, MaxSessions)
   ```
3. Export a binding (e.g. via Wails `GetConstants()` or a static method on the service) that returns `{"MaxSessions": 10}` so the frontend can derive the toast value from the Go source of truth.
4. On the frontend, use `err.message.includes(ErrMaxSessionsGoString)` or (if Wails surfaces Go error types) `err.name === "ErrMaxSessions"`.

This is a maintenance-risk warning, not an active bug. The current code works.

### WR-02: Pre-existing `cmd.Wait()` race between `killProcessGroup` and `monitorExit` is exposed by more call sites

**File:** `terminal_service.go:279, 373, 702` (all route through `killProcessGroup` at `pty_backend_unix.go:76-103`); competing `cmd.Wait()` at `terminal_service.go:539` (`monitorExit`).

**Issue:** The Phase 25 refactor increased the number of call sites that route through `s.ptyBackend.Kill(cmd)` from 1 (the prior `CloseSession` site) to 3 (added `Stop` at line 702 and `startSessionLocked` cleanup at line 373). All three ultimately invoke `killProcessGroup` (pty_backend_unix.go:76-103), which itself spawns a goroutine that calls `_ = cmd.Wait()` (line 91) and waits for it before returning.

The same `*exec.Cmd` is concurrently waited on by `monitorExit` (terminal_service.go:539). Per `os/exec` documentation, calling `cmd.Wait()` more than once returns an error (`exec: Wait was already called`) — which is NOT an `*exec.ExitError`. In `monitorExit`, the error path is:

```go
exitCode := 0
if err != nil {
    if exitErr, ok := err.(*exec.ExitError); ok {
        exitCode = exitErr.ExitCode()
    } else {
        exitCode = -1  // <-- "Wait was already called" lands here
    }
}
```

Consequence: when a session is closed and `killProcessGroup`'s goroutine wins the race for `cmd.Wait()`, `monitorExit` may report `exitCode = -1` (an unknown / "crash") instead of the actual exit code. The downstream `wasIntentional := ss.intentionalStop || exitCode == 0` (line 557) would be `false` for `exitCode = -1`, which can lead to a brief auto-restart attempt before the session's `stopCh` is checked.

This is **pre-existing** in the original `killProcessGroup` and `monitorExit`; Phase 25 did not introduce the race. It is being flagged because:
- The refactor did not fix it despite the code being directly in scope.
- The refactor increased the surface area (3 call sites instead of 1) that can race with `monitorExit`'s `cmd.Wait()`.

**Fix (separate concern; out of scope for Phase 25 but should be tracked):**

Option A (cheap, partial): In `killProcessGroup`, instead of spawning a goroutine that calls `cmd.Wait()`, just send `SIGHUP` and rely on `monitorExit` to reap the process. The function would return as soon as the signal is sent.

Option B (correct): Use a sync.Once or atomic flag to ensure `cmd.Wait()` is called exactly once across all paths. Track the exit code in a struct shared between `killProcessGroup` and `monitorExit`.

Option C (pragmatic): In `monitorExit`, when `cmd.Wait()` returns an error that is not `*exec.ExitError`, check if `cmd.ProcessState != nil` and use `cmd.ProcessState.ExitCode()` if available, falling back to `-1` only if state is truly absent.

This is not a regression introduced by Phase 25 and does not block the v2.1 release, but the refactor was a good moment to fix it.

## Info

### IN-01: Mock `Start` leaks the sleep process if cleanup is skipped

**File:** `pty_backend_mock.go:27-38`
**Issue:** `mockPtyBackend.Start` calls `exec.Command("sleep", "0.05")` and `cmd.Start()`. The returned `*exec.Cmd` is the only handle to the spawned process; if a test does not defer `ServiceShutdown` (or otherwise call `CloseSession`/`Stop`), the process is reparented to init when the test process exits but the `*exec.Cmd` resource itself is never `Wait()`ed. In practice, the test binary exits and the process is gone, but during a long-running test session with skipped cleanup, you can see leftover `sleep` processes from `pgrep`.

Current tests are not affected (they all `defer s.ServiceShutdown()`).

**Fix:** Add a doc comment to `mockPtyBackend.Start` stating "callers must eventually call CloseSession or ServiceShutdown to reap the process". Already partially documented by the SUMMARY's "no real long-lived process" remark; would be clearer inline.

### IN-02: Mock `Read` returns `io.EOF` on first call (limitation for future tests)

**File:** `pty_backend_mock.go:73-83`
**Issue:** `mockPtyHandle.Read` returns `(0, io.EOF)` when the output buffer is empty. In `terminal_service.go:readLoop` (line 429-435), any non-nil error from `ptmx.Read` is treated as terminal, and the readLoop returns. The mock therefore exits the readLoop on the first iteration. The current tests (stress and max-sessions) do not read PTY output, so this is fine. Any future test that wants to drive PTY output from the mock must pre-fill the output buffer *before* the readLoop is scheduled, or use a coordination channel. This is a design constraint, not a bug, but worth noting for test authors.

**Fix:** Document the contract at `mockPtyHandle.Read`:
```go
// Read returns bytes from the output buffer. If the buffer is empty,
// Read returns io.EOF immediately (simulating "no output available"
// rather than blocking). Tests that need to drive output must pre-fill
// the output buffer.
```

### IN-03: Windows stub error messages are inconsistent

**File:** `pty_backend_windows.go:28, 33, 43, 48`
**Issue:** Four error returns across the conpty stub. Two carry the suffix `" — see Plan 16-03"` (line 28 in `conptyBackend.Start`, line 43 in the `ptyStart` package-level helper), and two do not (line 33 in `conptyBackend.Resize`, line 47 in the `ptyResize` package-level helper). The original `terminal_windows.go` (pre-Phase 25) used the shorter "not yet implemented" wording uniformly. The "see Plan 16-03" suffix is a stale historical reference (no such plan exists in the current project state). Functionally harmless, but log-grep noisy.

**Fix:** Pick one wording. Recommend:
```go
"windows conpty backend is not implemented in this milestone"
```
for all four call sites. Update both the conptyBackend methods and the package-level helpers (which are preserved for test compilation but are dead code in production).

### IN-04: Frontend `limit: 10` is a magic number not derived from the Go constant

**File:** `frontend/src/App.tsx:206`
**Issue:** The toast call is `toast.error(t('toast.maxSessionsReached', { limit: 10 }))`. The `10` is a hard-coded mirror of the Go `MaxSessions = 10` constant (terminal_service.go:28). If the Go constant changes, the toast will show a stale value. The SUMMARY documents this as an accepted trade-off (the constant rarely changes; a future Phase could expose it via a binding), but it is worth a tracking item.

**Fix (future):** Add a `TerminalService.GetConstants()` method (or extend `GetEventNames` to include constants) that returns `{"MaxSessions": 10}`. The frontend can then `const { MaxSessions } = await GetConstants();` once at app startup and use that value in the toast call.

### IN-05: Pre-existing duplicate `ss.readerWg.Wait()` preserved by refactor

**File:** `terminal_service.go:375-376`
**Issue:**
```go
ss.readerWg.Wait()
ss.readerWg.Wait()
```
The second call is a no-op (the WaitGroup counter is zero after the first call returns). This was already in the code before Phase 25 (e.g. terminal_service.go from commit 87b30f4: lines 345-346). The Phase 25 refactor at commit e32f174 modified the surrounding code (replacing `killProcessGroup` with `s.ptyBackend.Kill`, then `s.ptyBackend.Start`) but did not remove the duplicate. The refactor was a good opportunity to clean it up; harmless in operation but reads as accidental code.

**Fix:** Delete one of the two lines. Suggested change to terminal_service.go:375-376:
```go
ss.readerWg.Wait()
```
(remove the second line).

---

## Recommendations (non-blocking)

1. **Add a typed Go error for `ErrMaxSessions`** (see WR-01 fix). A 5-line change that prevents the most likely future regression in the new error path.
2. **Fix or document the `cmd.Wait()` race** (see WR-02 fix). Worth a follow-up issue; not a v2.1 blocker.
3. **Clean up the Windows stub error messages** (IN-03). Two minutes of work.
4. **Remove the duplicate `ss.readerWg.Wait()`** (IN-05). Two-character change.

---

_Reviewed: 2026-06-16T08:30:00Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_
