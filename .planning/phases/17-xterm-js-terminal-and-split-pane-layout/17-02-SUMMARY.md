---
phase: 17-xterm-js-terminal-and-split-pane-layout
plan: "02"
subsystem: ui
tags: [react, split-pane, useResizable, xterm.js, App.tsx]

requires:
  - phase: 17-xterm-js-terminal-and-split-pane-layout
    provides: TerminalComponent with xterm.js (17-01)
provides:
  - Vertical split pane layout: editor top, resizable terminal bottom
  - OutputPane component fully removed (state, refs, imports, keyboard shortcut)
  - Collapsible divider with localStorage persistence
affects: [18-pty-input, 19-terminal-theme]

tech-stack:
  added: []
  patterns:
    - "Split pane uses useResizable hook (axis: 'y', direction: -1) with localStorage-backed default"
    - "TerminalComponent always mounted in DOM (display toggle via isVisible prop)"
    - "Collapse state persisted to localStorage key `cmdex-terminal-height-collapsed`"

key-files:
  created: []
  modified:
    - frontend/src/App.tsx
    - frontend/src/lib/shortcuts.ts

key-decisions:
  - "Terminal always mounted (never conditional render) per D-12/LAY-04"
  - "Default terminal height 40% viewport, min 100px, max 85% per D-02/D-03/D-04"

patterns-established:
  - "Pattern 1: Terminal lives outside tab-switching container as sibling to preserve PTY subscriptions across tab changes"
  - "Pattern 2: Collapse uses CSS display:none on TerminalComponent, not conditional rendering"

requirements-completed:
  - LAY-01
  - LAY-02
  - LAY-04

duration: 5min
completed: 2026-05-19
---

# Phase 17 Plan 02: Split Pane Layout Summary

**Vertical split pane with resizable divider replaces OutputPane; Terminal always mounted in DOM with collapse/expand and localStorage persistence.**

## Performance

- **Duration:** 5 min
- **Started:** 2026-05-19T10:15:00Z
- **Completed:** 2026-05-19T10:20:00Z
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments

- OutputPane fully excised: import removed, outputPaneOpen state removed, tabOutputRef removed, outputPaneOpenRef removed, toggleOutput keyboard shortcut removed from App.tsx and shortcuts.ts
- New split pane layout: center-area-split wrapper with center-area-editor (top) + terminal-divider + terminal-pane (bottom) using useResizable hook
- TerminalComponent always mounted with monoFont prop and isVisible={!terminalCollapsed} — never unmounted across tab switches per LAY-04
- Collapse/expand wired: terminalCollapsed state, collapseTerminal/expandTerminal callbacks, localStorage persistence, collapsed rail button

## Task Commits

1. **Task 1: Remove OutputPane state, import, shortcut, and output tracking** - `c4d4753` (feat)
2. **Task 2: Add split pane layout with useResizable divider + Terminal component** - `1183abe` (feat)

## Files Created/Modified

- `frontend/src/App.tsx` - Removed OutputPane import/state/refs/tracking; added useResizable import, terminal state constants (TERM_STORAGE_KEY, MIN_TERM_HEIGHT, MAX_TERM_HEIGHT_PCT), useResizable hook call with axis:'y'/direction:-1, collapse/expand callbacks, and new JSX layout wrapping editor in center-area-split + center-area-editor, terminal in terminal-pane with TerminalComponent and collapsed rail
- `frontend/src/lib/shortcuts.ts` - Removed toggleOutput entry from SHORTCUTS registry

## Decisions Made

- None - followed plan as specified.

## Deviations from Plan

None - plan executed as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Split pane layout with persistent Terminal fully wired in App.tsx
- Ready for Phase 18: PTY input wiring (keystroke capture and TerminalService.Write)
- Ready for Phase 19: Terminal theme sync

---
*Phase: 17-xterm-js-terminal-and-split-pane-layout*
*Completed: 2026-05-19*
