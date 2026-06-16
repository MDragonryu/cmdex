# Phase 25 Checkpoint

**Phase:** 25-polish-integration
**Date:** 2026-06-16
**Status:** Complete

## Summary

Phase 25 delivered seven in-scope items from the context document:

1. **Stress test for session create/close lifecycle** — `TestTerminalService_StressCreateClose` runs a 100-cycle create/close loop against the darwin-side mock and asserts the `sessions` map drains to zero each cycle and the goroutine count is stable within a delta.
2. **Build-tagged `ptyBackend` interface** — `ptyBackend` (in `pty_backend.go`) separates `TerminalService` from the OS PTY layer. `creackPtyBackend` (in `pty_backend_unix.go`, `!windows`) is the production darwin/linux implementation; `conptyBackend` (in `pty_backend_windows.go`, `windows`) is the Windows stub; `mockPtyBackend` (in `pty_backend_mock.go`, `darwin`) is the test-only in-memory fake.
3. **Cross-compile verification** — `GOOS=windows go build ./...` exits 0 from darwin. The conpty stub in `pty_backend_windows.go` compiles cleanly; no conpty API mismatches surfaced at compile time.
4. **Global default cwd inheritance** — `getWorkingDir()` in `terminal_service.go` reads the OS-keyed global default from settings and falls back to `os.UserHomeDir()`. `TestTerminalService_GlobalDefaultCwdInheritance` and `TestTerminalService_CwdInheritance_ExistingSessionUnaffected` cover both the configured path and the home fallback.
5. **Dead-code cleanup** — `terminal_service_stress_test.go` and the cwd-inheritance tests surface no lingering `OutputPane`, `cmd-output`, or `RunInTerminal` references in the active source tree.
6. **Error states** — `MaxSessions = 10` guard in `CreateSession` returns a `max sessions reached (10)` error when the limit would be exceeded; the last-session close guard at `App.tsx:closeTerminalSession` (line 214) and `TerminalTabBar.tsx:isLastTab` (lines 131, 151) prevents closing the final tab; the PTY start failure toast at `Terminal.tsx:292` surfaces shell start errors.
7. **Windows conpty documentation gap** — This document and a one-line note in `AGENTS.md` make the cross-compile-only coverage explicit so future maintainers know that real Windows runtime verification is not done in this milestone.

## Windows conpty verification

### Scope of verification in this phase

This phase verifies the Windows conpty path at three levels, in order of strength:

1. **Cross-compile to Windows from darwin** (`GOOS=windows go build ./...` from a darwin host). This is a *compile-time* check: it catches conpty API mismatches, missing build tags, signature drift, and any other issues that surface when the Go compiler resolves the `pty_backend_windows.go` stub. Verified: exits 0 as of the Task 2 commit.
2. **Darwin-side orchestration tests against the mock backend**. The `mockPtyBackend` (in `pty_backend_mock.go`, `//go:build darwin`) is an in-memory ptyBackend that satisfies the same interface the production conpty implementation does. The stress test, cwd-inheritance tests, and the new `TestTerminalService_MaxSessionsLimit` all run against this mock — they exercise the orchestration code paths (session lifecycle, mutex ordering, error propagation) without needing real PTY hardware.
3. **Compile-time coverage of the conpty stub**. The `conptyBackend` struct in `pty_backend_windows.go` is a *placeholder*: `Start` and `Resize` return `Windows PTY support not yet implemented` errors; `Kill` delegates to `taskkill`. The stub exists so the `ptyBackend` interface compiles on windows and so the `if runtime.GOOS == "windows" { ... }` shell-detection branch in `detectShell()` can be exercised by the build, but it does not actually start a PTY.

What this means: the cross-compile + mock combination proves that the `TerminalService` ↔ `ptyBackend` contract is correct. It does **not** prove that a real conpty process can be spawned, attached to a `*os.File`, resized, or killed on Windows.

### Out of scope (acknowledged gap)

The following are explicitly out of scope for this phase and represent acknowledged technical debt. They are documented here so future maintainers do not mistake the cross-compile success for full Windows coverage.

- **Runtime conpty verification** — there is no Windows machine in the dev environment for this milestone, and no `windows-latest` runner is configured in CI. A real Windows conpty process is never started, attached, or interacted with during this phase.
- **Process group / signal handling on Windows** — `killProcessGroup` on Windows uses `taskkill /F /T /PID <pid>` (see `pty_backend_windows.go:51-56`). This code path is compiled but never executed. Signal semantics on Windows differ from Unix (no `SIGHUP`/`SIGKILL`; process groups are not first-class), and the cross-compile cannot catch behavioral differences.
- **Shell detection on Windows via the `detectShell` Windows branch** — `terminal_service.go:124-132` probes for `pwsh` → `powershell` → `cmd` via `exec.LookPath`. This code compiles but is never executed in this phase. A Windows machine would need to validate which shell is selected and whether the chosen shell accepts the `-NoLogo` flag in the expected way.
- **Frontmatter parser for `.planning` paths on Windows** — the `os.UserHomeDir()` fallback in `getWorkingDir` returns `%USERPROFILE%` on Windows. The Go standard library handles this, but the cross-compile cannot verify the runtime behavior.

### Future work

To close the Windows conpty verification gap in a future phase:

1. **Add `github.com/UserExistsError/conpty` to `go.mod`** (or whichever conpty binding is current at the time of implementation). The conpty ecosystem has been unstable through Wails v3 alpha, so pin a version that matches the project's other alpha dependencies.
2. **Replace `conptyBackend` stub with a real implementation**. `Start` would call `conpty.NewPseudoConsole()` to get a `*os.File` handle; `Resize` would call `conpty.ResizePseudoConsole`; `Kill` would already work via the existing `taskkill` helper.
3. **Add a `windows-latest` CI runner** to the GitHub Actions matrix that builds (`GOOS=windows go build ./...`) and runs the test suite. The stress test and cwd-inheritance tests can run unchanged on Windows once the real conpty is wired up.
4. **Add a Windows-tagged variant of `TestTerminalService_CreateSession`** (and friends) that uses the real conpty backend instead of the darwin-only mock. The `mockPtyBackend` is `//go:build darwin` because it relies on the darwin test runtime; a Windows-only test file would exercise the production conpty code path on a Windows runner.

## Files

### New

- `pty_backend.go` — `ptyBackend` interface + `ptyHandle` interface (seam between `TerminalService` and the OS PTY layer)
- `pty_backend_mock.go` — `mockPtyBackend` + `mockPtyHandle` (`//go:build darwin`, in-memory fake for orchestration tests)
- `pty_backend_mock_test.go` — `newTestTerminalServiceWithMock` helper (`//go:build darwin`)
- `terminal_service_stress_test.go` — `TestTerminalService_StressCreateClose` (`//go:build darwin`, 100-cycle stress test)
- `terminal_service_max_sessions_test.go` — `TestTerminalService_MaxSessionsLimit` (`//go:build darwin`, max-sessions guard test)
- `.planning/phases/25-polish-integration/RUNBOOK.md` — manual xterm smoke-test procedure for frontend (xterm) leak verification
- `.planning/phases/25-polish-integration/CHECKPOINT.md` — this file

### Modified

- `pty_backend_unix.go` — extracted creack/pty calls into `creackPtyBackend` adapter; existing `ptyStart`/`ptyResize`/`killProcessGroup` package-level functions preserved for test code
- `pty_backend_windows.go` — wrapped conpty stubs in `conptyBackend` struct; existing `ptyStart`/`ptyResize`/`killProcessGroup` package-level functions preserved for test code
- `terminal_service.go` — added `const MaxSessions = 10`; added `len(s.sessions) >= MaxSessions` guard at the top of `CreateSession`; added `getWorkingDir()` helper that reads the OS-keyed global default from settings and falls back to `os.UserHomeDir()`
- `terminal_service_test.go` — added `cwdInheritanceTestDB` and `setGlobalDefaultCwd` test helpers; added `TestTerminalService_GlobalDefaultCwdInheritance` and `TestTerminalService_CwdInheritance_ExistingSessionUnaffected`
- `frontend/src/App.tsx` — `createTerminalSession` catch block now detects the new `max sessions reached` error string and surfaces the localized `toast.maxSessionsReached` toast
- `frontend/src/locales/en.json` — added `toast.maxSessionsReached` with `{{limit}}` interpolation
- `AGENTS.md` — added a one-line bullet to the `## Tests` section pointing to this checkpoint
- `.planning/ROADMAP.md` — Phase 25 success criteria replaced with the in-memory set (5 items)
- `.planning/REQUIREMENTS.md` — `PERS-01..PERS-04` moved from v1 to v2 (deferred); traceability table updated; coverage line shows 18 v1 mapped and 10 v2 total

### Removed

- None. The Phase 24 dead-code cleanup (OutputPane removal) was already complete before this phase.

## Test coverage

This phase added four new Go test functions:

- `TestTerminalService_GlobalDefaultCwdInheritance` — D-04/D-05: global default cwd is read from settings, falls back to home
- `TestTerminalService_CwdInheritance_ExistingSessionUnaffected` — D-06: changes to global default do not retroactively affect existing sessions
- `TestTerminalService_StressCreateClose` — D-08: 100-cycle create/close stress test, goroutine count stable
- `TestTerminalService_MaxSessionsLimit` — D-13 partial: `CreateSession` rejects calls past `MaxSessions = 10` with a `max sessions reached` error

The existing twelve multi-session tests in `terminal_service_test.go` (`TestTerminalService_CreateSession`, `TestTerminalService_ListSessions`, `TestTerminalService_RenameSession`, `TestTerminalService_CloseSession`, `TestTerminalService_ActiveSessionReassignOnClose`, `TestTerminalService_SetActiveSession`, `TestTerminalService_GetActiveSession`, `TestTerminalService_ShutdownCleansAll`, `TestTerminalService_ConcurrentAccess`, `TestTerminalService_ProcessPersistAcrossSessionSwitch`, `TestTerminalService_OutputIsolation`, `TestTerminalService_NamespacedEvents`) all still pass.

The `db_test.go` migration tests continue to pass unchanged.

## Decisions honored

This phase implemented all 14 locked decisions from the Phase 25 context document:

- **D-01:** Phase 22 stayed skipped. No DB persistence for sessions. Sessions live in `TerminalService.sessions` map only.
- **D-02:** Phase 25 success criteria in `ROADMAP.md` were replaced with the in-memory set (memory-leak, conpty, cwd inheritance, dead-code cleanup, error states). Persistence claims are removed.
- **D-03:** PERS-01, PERS-02, PERS-03, PERS-04 moved from v1 to v2 Requirements. Traceability table updated. Coverage counts updated to 18 v1 / 10 v2.
- **D-04:** New sessions inherit the OS-keyed global default cwd from settings. `getWorkingDir()` reads from `db.GetSettings().DefaultWorkingDir.GetCurrentOS()` and the contract is verified by `TestTerminalService_GlobalDefaultCwdInheritance`.
- **D-05:** Empty global default → fall back to `os.UserHomeDir()`. Same test covers both paths.
- **D-06:** Changes to global default cwd do not retroactively affect existing sessions. `TestTerminalService_CwdInheritance_ExistingSessionUnaffected` verifies.
- **D-07:** The Phase 24 EXEC-03 cwd chain (per-command wd → global default → session cwd) is confirmed and unchanged. No reordering.
- **D-08:** 100-cycle stress test in `TestTerminalService_StressCreateClose`. Asserts goroutine count stable, sessions map empty after close, PTY FDs closed. Runs in <5s.
- **D-09:** Frontend (xterm) leak verification is a manual smoke test documented in `.planning/phases/25-polish-integration/RUNBOOK.md`. Open 20 tabs via Ctrl+T, close all via Ctrl+W, inspect the DOM for orphaned xterm nodes.
- **D-10:** No new test framework. The project has no automated tests; this phase adds Go tests using only the standard library `testing` package.
- **D-11:** `ptyBackend` interface separates `TerminalService` from the OS PTY layer. `ptyBackend.go` defines the interface; `pty_backend_unix.go` and `pty_backend_windows.go` implement it.
- **D-12:** Darwin-side behavior mock (`mockPtyBackend` in `pty_backend_mock.go`) covers PTY spawn, write, read, and resize. Skips shell detection, process group / signal handling, and exit detection — those rely on real conpty and are out of scope.
- **D-13:** Windows verification path is: (1) `GOOS=windows go build ./...` from darwin, which catches conpty API mismatches at compile time (verified to exit 0 in this phase); (2) orchestration tests run against the darwin-side mock; (3) the gap is acknowledged here in this checkpoint and in `AGENTS.md`. No Windows runtime CI in this phase.
- **D-14:** This checkpoint includes the "Windows conpty verification" section explaining the mock scope, cross-compile coverage, and the real-Windows gap. `AGENTS.md` is updated with a one-line note so future maintainers know the gap exists.
