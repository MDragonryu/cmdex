---
phase: 18-execution-integration-and-interactivity
plan: 02
subsystem: terminal
tags: [xterm.js, keystroke-buffering, forwardRef, pty, ctrl-c]

# Dependency graph
requires:
  - phase: 18-01
    provides: "RunCommand via PTY Write, cd sandwich working directory integration"
provides:
  - "term.onData keystroke buffering (50ms idle / immediate on Enter)"
  - "TerminalHandle interface with clear() via forwardRef"
  - "Clear Terminal button in terminal divider bar"
  - "Ctrl+C (\\x03) forwarding through keystroke buffer to PTY stdin"
affects: ["19-terminal-polish", "terminal-interactivity"]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "forwardRef + useImperativeHandle for terminal control from parent"
    - "Client-side keystroke buffering with flush-on-Enter / 50ms idle timeout"

key-files:
  created: []
  modified:
    - "frontend/src/components/Terminal.tsx"
    - "frontend/src/App.tsx"
    - "frontend/src/style.css"

key-decisions:
  - "forwardRef + useImperativeHandle pattern for exposing terminal.clear() to parent"
  - "50ms idle timeout for keystroke buffering (matches readLoop 16ms batching principle)"
  - "Clear button calls both term.clear() (xterm.js scrollback) AND Write('clear\\r') (shell screen)"
  - "Ctrl+C flows through same keystroke buffer path — no separate Interrupt() method needed"

patterns-established:
  - "Keystroke batching: accumulate in string buffer, flush on Enter or 50ms idle"
  - "forwardRef + useImperativeHandle for cross-component imperative control"

requirements-completed: [EXEC-02, EXEC-03, LAY-03]

# Metrics
duration: 6min
completed: 2026-05-21
---

# Phase 18 Plan 02: Keystroke Forwarding and Terminal Interactivity Summary

**xterm.js term.onData keystroke buffering with 50ms idle batched writes to PTY, TerminalHandle forwardRef with clear(), and Clear Terminal button in divider bar**

## Performance

- **Duration:** 6 min
- **Started:** 2026-05-21T04:16:42Z
- **Completed:** 2026-05-21T04:22:57Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- Keystroke buffering via `term.onData` accumulates chars and flushes to `TerminalService.Write` on Enter (`\r`) or after 50ms idle timeout
- Ctrl+C (`\x03`) automatically flows through the same buffer path — PTY driver translates to SIGINT for foreground process
- `TerminalHandle` interface exported from `Terminal.tsx` with `clear()` method exposed via `useImperativeHandle`
- `clear()` calls both `term.clear()` (xterm.js scrollback reset) and `Write('clear\r')` (shell screen clear)
- Clear Terminal button added to terminal divider bar, wired to `terminalRef.current?.clear()`
- CSS for `.terminal-clear-btn` matches existing `.terminal-collapse-btn` styling with right-side positioning

## Task Commits

Each task was committed atomically:

1. **Task 1: Add term.onData keystroke buffering + forwardRef clear to Terminal.tsx** - `145413c` (feat)
2. **Task 2: Wire terminalRef + Clear Terminal button in App.tsx terminal pane** - `1ece40f` (feat)

**Plan metadata:** (to be committed after SUMMARY.md)

## Files Modified
- `frontend/src/components/Terminal.tsx` - Added forwardRef/useImperativeHandle, TerminalHandle interface, term.onData keystroke buffering, Write import
- `frontend/src/App.tsx` - Added TerminalHandle type import, terminalRef declaration, ref prop on TerminalComponent, Clear button in terminal divider
- `frontend/src/style.css` - Added `.terminal-clear-btn` CSS matching `.terminal-collapse-btn` styling

## Decisions Made
- Used `forwardRef` + `useImperativeHandle` for terminal control from parent (per RESEARCH.md Q2 resolution — keeps clear logic encapsulated in terminal component)
- Clear button calls both `term.clear()` AND `Write('clear\r')` — both approaches per Agent Discretion for complete screen reset
- 50ms idle timeout for keystroke buffering mirrors the 16ms batching principle used in readLoop
- No separate `Interrupt()` Go method needed — `\x03` byte flows through `term.onData` → buffer → `Write` path

## Deviations from Plan

None — plan executed exactly as written.

## Issues Encountered
- Initial TypeScript compilation error (TS1128) on first attempt — missing closing brace for forwardRef arrow function body. Fixed immediately in Task 1.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness
- Terminal is now fully interactive — users can type commands freely and Ctrl+C works
- Ready for Phase 19 Terminal Polish (theme sync, font, copy/paste, search)
- Plan 18-01 (RunCommand PTY Write + cd sandwich) must be completed for the Run button to write commands to this terminal

---

*Phase: 18-execution-integration-and-interactivity*
*Completed: 2026-05-21*

## Self-Check: PASSED

- [x] SUMMARY.md exists on disk
- [x] Task commits verified: `145413c` (Task 1), `1ece40f` (Task 2)
- [x] `pnpm tsc --noEmit` passes with no TypeScript errors
- [x] All acceptance criteria verified for both tasks
