---
phase: 19-terminal-polish
plan: "01"
subsystem: ui
tags: [xtermjs, css-variables, theme-sync, font-transition]

requires:
  - phase: 18-execution-integration-and-interactivity
    provides: "xterm.js Terminal component with FitAddon, WebglAddon, WebLinksAddon, PTY I/O, keystroke buffering"
provides:
  - CSS variable → ITheme hot-swap for terminal theme sync (no recreate, no flicker)
  - 150ms opacity transition on font change (no fade on first mount)
affects: []

tech-stack:
  added: []
  patterns:
    - "CSS variable mapping: read --background, --foreground, --primary from getComputedStyle, map to xterm ITheme"
    - "Hex-to-RGBA helper for selectionBackground opacity"

key-files:
  created: []
  modified:
    - frontend/src/components/Terminal.tsx - theme prop, hexToRgba helper, theme useEffect, isFirstMountRef, opacity transition
    - frontend/src/App.tsx - theme={theme} prop passed to TerminalComponent
    - frontend/src/style.css - opacity: 1 on .terminal-container

key-decisions:
  - "ANSI 16-color palette preserved via ...term.options.theme spread — only bg/fg/cursor/selection overridden"
  - "Selection background uses primary color at 0.4 opacity via hexToRgba inline helper"
  - "Transition set inline (not CSS class) to avoid animating first mount and collapse/expand toggles"

patterns-established:
  - "Theme prop → useEffect → getComputedStyle → term.options.theme hot swap — no terminal recreate"
  - "isFirstMountRef → skipTransition guard — prevents initial mount fade animation"

requirements-completed: [POL-01, POL-02, POL-03, POL-04]

duration: 6min
completed: 2026-05-22
---

# Phase 19 Plan 01: Terminal Polish Summary

**CSS variable → ITheme hot-swap for real-time terminal theme sync, plus 150ms font-change opacity transition**

## Performance

- **Duration:** 6 min
- **Started:** 2026-05-22T03:47:00Z
- **Completed:** 2026-05-22T03:52:59Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- Terminal theme hot-swaps on Cmdex theme change via `getComputedStyle` CSS variable mapping — reads `--background`, `--foreground`, `--primary` and maps to xterm `ITheme` fields via `term.options.theme = {...}`
- ANSI 16-color palette preserved across all themes via `...term.options.theme` spread — only bg/fg/cursor/selection overridden
- Font change triggers 150ms opacity fade transition on `.terminal-container` — no fade on initial mount via `isFirstMountRef` guard
- Copy/paste confirmed working via xterm.js v6.0.0 built-in clipboard support — no code changes needed (POL-03, POL-04 verified)

## Task Commits

Each task was committed atomically:

1. **Task 1: Theme sync** - `133c998` (feat(19-01): sync terminal theme via CSS var mapping to xterm ITheme)
2. **Task 2: Font change opacity transition** - `f1b68c4` (feat(19-01): add 150ms opacity transition on terminal font change)

## Files Created/Modified

- `frontend/src/components/Terminal.tsx` - Added `theme` prop to interface, `hexToRgba` helper, `useEffect([theme])` with CSS var → ITheme mapping, `isFirstMountRef` guard, opacity transition logic in `[monoFont]` effect
- `frontend/src/App.tsx` - Added `theme={theme}` prop to `<TerminalComponent>`
- `frontend/src/style.css` - Added `opacity: 1` to `.terminal-container`

## Decisions Made

- Used `...term.options.theme` spread to preserve hardcoded ANSI 16-color palette (D-03)
- Selection background uses `--primary` at 0.4 opacity via inline `hexToRgba()` helper (RESEARCH recommendation)
- Opacity transition set inline via `containerRef.current.style.transition` only during font changes — avoids animating first mount and collapse/expand toggles
- `getComputedStyle` values `.trim()`'d to handle browser whitespace (RESEARCH Pitfall 1)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Phase 19 complete — terminal is fully theme-aware with smooth font transitions
- Copy/paste confirmed native (xterm.js v6.0.0 built-in) — no further work needed for POL-03, POL-04
- SearchAddon (POL-05) deferred per D-09 — ready for future phase when scrollback search is desired
- All verification criteria met: `make check` passes (go build + pnpm tsc --noEmit)

---
*Phase: 19-terminal-polish*
*Completed: 2026-05-22*
