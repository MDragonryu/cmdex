---
phase: 17-xterm-js-terminal-and-split-pane-layout
plan: "03"
subsystem: terminal
tags: [xterm.js, pty, wails-events, events.ts]

requires:
  - phase: 17-xterm-js-terminal-and-split-pane-layout
    provides: Terminal component foundation with xterm.js (17-01)
provides:
  - PTY output event wiring from Go TerminalService backend to xterm.js rendering
  - Event name constants (ptyOutput, ptyExit) with hardcoded fallback and Go backend population
  - Regenerated Wails bindings exposing TerminalService methods and EventNames fields
affects: [18-pty-input, 19-terminal-theme]

tech-stack:
  added: []
  patterns:
    - "Wails v3 event subscription pattern: Events.On(eventNames.X, handler) with cleanup return"
    - "Double-unwrap for Wails v3 PTY events: event.data.data (Wails wrapper + Go payload)"
    - "Separate useEffect for event subscriptions with empty deps after terminal ref is set"

key-files:
  created: []
  modified:
    - frontend/src/wails/events.ts
    - frontend/src/components/Terminal.tsx
    - frontend/bindings/cmdex/eventservice.js (regenerated, gitignored)
    - frontend/bindings/cmdex/terminalservice.js (regenerated, gitignored)
    - frontend/bindings/cmdex/models.js (regenerated, gitignored)

key-decisions:
  - "None - followed plan as specified"

patterns-established:
  - "Pattern 1: PTY output events use double-unwrap (event.data.data) because Wails v3 wraps in {name, data, sender} and Go emits {data: string}"
  - "Pattern 2: pty-exit event handler is informational only — backend already writes exit message to PTY output before emitting the event"

requirements-completed:
  - TERM-01
  - TERM-02

duration: 3min
completed: 2026-05-19
---

# Phase 17 Plan 03: Wire PTY Backend to xterm.js Terminal Summary

**Event name constants for PTY output/exit registered in events.ts, Wails bindings regenerated with TerminalService exports, and Terminal.tsx subscribed to live PTY data flow via pty-output/pty-exit Wails events.**

## Performance

- **Duration:** 3 min
- **Started:** 2026-05-19T10:15:00Z
- **Completed:** 2026-05-19T10:18:00Z
- **Tasks:** 2
- **Files modified:** 2 (committed) + 3 (regenerated, gitignored)

## Accomplishments

- events.ts now has `ptyOutput: 'pty-output'` and `ptyExit: 'pty-exit'` constants with both hardcoded fallback values and `initEventNames()` population from Go backend
- `wails3 generate build-assets` regenerated bindings: EventNames type includes PtyOutput/PtyExit, TerminalService methods (Start, Stop, Write, Resize) available to frontend
- Terminal.tsx subscribes to `eventNames.ptyOutput` — writes raw PTY bytes to xterm.js via `term.write(event.data.data)` using Wails v3 double-unwrap pattern
- Terminal.tsx subscribes to `eventNames.ptyExit` — logs exit code, no frontend UI action needed (backend writes exit message to PTY output)
- Event subscriptions properly cleaned up via `cleanupOutput()` and `cleanupExit()` in useEffect return, preventing listener accumulation on hot reload

## Task Commits

1. **Task 1: Add ptyOutput/ptyExit to events.ts + regenerate Wails bindings** - `dfa1506` (feat)
2. **Task 2: Add pty-output/pty-exit event subscriptions to Terminal.tsx** - `9be0bea` (feat)

## Files Created/Modified

- `frontend/src/wails/events.ts` - Added `ptyOutput`/`ptyExit` to eventNames object (lines 9-10) and initEventNames() population (lines 22-23)
- `frontend/src/components/Terminal.tsx` - Added Events/eventNames imports (lines 7-8), new useEffect for PTY event subscriptions with cleanup (lines 97-117)
- `frontend/bindings/cmdex/eventservice.js` - Regenerated with EventNames.PtyOutput/PtyExit (gitignored)
- `frontend/bindings/cmdex/terminalservice.js` - New TerminalService bindings with Start, Stop, Write, Resize (gitignored)
- `frontend/bindings/cmdex/models.js` - Regenerated EventNames type with ptyOutput/ptyExit fields (gitignored)

## Decisions Made

None - followed plan as specified.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

- All 28 TypeScript errors in `pnpm tsc --noEmit` are pre-existing in App.tsx (unrelated removed variables like `tabOutputRef`, `OutputPane`). Terminal.tsx and events.ts compiled cleanly.
- Generated Wails bindings (`frontend/bindings/`) are gitignored and cannot be committed — this is expected project behavior. Bindings are regenerated during `wails3 dev`/`wails3 build`.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- PTY output data flow complete: Go TerminalService → Wails event bridge → React Terminal → xterm.js rendering
- Ready for Phase 18: PTY input wiring (TerminalService.Write + xterm.js onData → keystrokes to PTY)
- Ready for Phase 19: Terminal theme sync with Cmdex theme system

---
*Phase: 17-xterm-js-terminal-and-split-pane-layout*
*Completed: 2026-05-19*
