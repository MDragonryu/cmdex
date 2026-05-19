---
phase: 17-xterm-js-terminal-and-split-pane-layout
plan: "01"
subsystem: ui
tags: [xterm.js, react, terminal, split-pane, css]

requires:
  - phase: 16-pty-backend-foundation
    provides: "TerminalService with PTY lifecycle and pty-output/pty-exit Wails events"
provides:
  - "xterm.js Terminal.tsx React component with FitAddon, WebglAddon, WebLinksAddon"
  - "CSS classes for terminal split pane layout (terminal-pane, divider, collapse, split)"
affects: [17-02 (App.tsx integration), 17-03 (PTY event wiring)]

tech-stack:
  added:
    - "@xterm/xterm@^6.0.0"
    - "@xterm/addon-fit@^0.11.0"
    - "@xterm/addon-webgl@^0.19.0"
    - "@xterm/addon-web-links@^0.12.0"
  patterns:
    - "useRef + useEffect + ResizeObserver for xterm.js React lifecycle"
    - "StrictMode ref guard (terminalRef.current === term) for safe dispose"
    - "CSS display toggle (isVisible prop → display: flex/none) for DOM persistence"
    - "try/catch WebglAddon with onContextLoss handler for GPU fallback"

key-files:
  created:
    - "frontend/src/components/Terminal.tsx"
  modified:
    - "frontend/package.json"
    - "frontend/src/style.css"

key-decisions:
  - "Hardcoded dark palette: background #1e1e1e, foreground #d4d4d4, full ANSI 16-color set (VS Code terminal default)"
  - "Addon init order: FitAddon → WebLinksAddon → WebglAddon (try/catch) — follows xterm.js best practice"
  - "ResizeObserver on container (not window.resize) feeds FitAddon — covers divider drag"
  - "requestAnimationFrame before initial fitAddon.fit() — prevents zero cols/rows race"
  - "Fixed 14px fontSize, monoFont read on mount only per D-14/D-16"

patterns-established:
  - "Terminal React lifecycle: useRef for Terminal/FitAddon/container, useEffect for create/open/dispose"
  - "CSS split pane: center-area-split (flex column) → center-area-editor (flex 1) + divider + terminal-pane"
  - "Divider: ns-resize cursor, 8px hit area, 1px → 2px line on hover/drag using var(--primary)"

requirements-completed: [TERM-01, TERM-02, TERM-03, TERM-04]

duration: 12min
completed: 2026-05-19
---

# Phase 17 Plan 01: Terminal.tsx Component and CSS Foundation

**xterm.js v6 Terminal React component with FitAddon/WebglAddon/WebLinksAddon, hardcoded dark palette, and split-pane CSS classes**

## Performance

- **Duration:** 12 min
- **Started:** 2026-05-19T12:45:00Z
- **Completed:** 2026-05-19T12:57:00Z
- **Tasks:** 3
- **Files modified/created:** 3

## Accomplishments
- Installed four @xterm/* packages at verified versions (xterm v6.0.0, addon-fit 0.11.0, addon-webgl 0.19.0, addon-web-links 0.12.0)
- Created Terminal.tsx with complete xterm.js lifecycle: useRef for instance/addon storage, useEffect for create/open/dispose, ResizeObserver for container-based resizing, StrictMode ref guard for safe cleanup
- Added 11 CSS classes for terminal split pane layout: pane container, xterm host div, horizontal divider with hover/drag states, collapse button with opacity transition, collapsed rail, split layout containers, and scrollbar overrides

## Task Commits

Each task was committed atomically:

1. **Task 1: Install xterm.js npm packages** - `f5773fb` (feat)
2. **Task 2: Create Terminal.tsx component** - `d1ac505` (feat)
3. **Task 3: Add Terminal CSS to style.css** - `a2d7d5b` (feat)

## Files Created/Modified
- `frontend/package.json` - Added @xterm/xterm, @xterm/addon-fit, @xterm/addon-webgl, @xterm/addon-web-links
- `frontend/src/components/Terminal.tsx` - xterm.js Terminal component with FitAddon, WebglAddon (try/catch), WebLinksAddon, ResizeObserver, StrictMode guard, CSS display toggle
- `frontend/src/style.css` - Added .terminal-pane, .terminal-container, .terminal-divider*, .terminal-collapse-btn*, .terminal-collapsed-rail*, .center-area-split, .center-area-editor, xterm viewport scrollbar overrides

## Decisions Made
- Used `cursorStyle: 'block'` for the terminal cursor (matches VS Code default)
- Used `allowProposedApi: true` (required for addons in xterm.js v6)
- Used `convertEol: true` for CR+LF compatibility with PTY output
- xterm viewport scrollbar hidden via `scrollbar-width: none` and `::-webkit-scrollbar { display: none }`
- Existing `.output-pane` CSS preserved (unused after 17-02, but safe to keep until then)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## Next Phase Readiness

Ready for 17-02 (App.tsx integration) — Terminal component is ready to be dropped into the split pane layout. Terminal accepts `monoFont` and `isVisible` props, matching the App.tsx integration contract.

---
*Phase: 17-xterm-js-terminal-and-split-pane-layout*
*Plan: 01*
*Completed: 2026-05-19*
