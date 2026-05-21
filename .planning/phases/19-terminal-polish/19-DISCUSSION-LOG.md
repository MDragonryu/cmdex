# Phase 19: Terminal Polish - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-21
**Phase:** 19-terminal-polish
**Areas discussed:** Theme Sync, Font Update, Copy/paste, SearchAddon

---

## Theme Sync

| Option | Description | Selected |
|--------|-------------|----------|
| CSS variable mapping | Read computed CSS custom properties at runtime, map each to the closest xterm ITheme field. Works for custom themes automatically. | ✓ |
| Explicit preset table | Define a static xterm theme preset for each of the 8 built-in Cmdex themes. | |
| Minimal mapping | Only sync background, foreground, cursor, and selection highlight from CSS vars. | |

**User's choice:** CSS variable mapping
**Notes:** User chose the most flexible approach — works for both built-in and custom themes without extra logic.

| Option | Description | Selected |
|--------|-------------|----------|
| Terminal listens directly | Terminal.tsx subscribes to settings-changed event itself. | |
| Pass as prop from App.tsx | App.tsx passes current theme as props to Terminal. | ✓ |

**User's choice:** Pass as prop from App.tsx
**Notes:** User deferred to best choice — prop pattern is consistent with existing monoFont prop.

| Option | Description | Selected |
|--------|-------------|----------|
| Keep existing palette | Keep hardcoded ANSI 16-color palette, only override bg/fg/cursor/selection from CSS vars. | ✓ |
| Derive from CSS vars | Map Cmdex CSS vars to ANSI slots (destructive→red, success→green, etc.). | |
| Built-in theme presets | Explicit ANSI palettes per built-in theme with fallback for custom themes. | |

**User's choice:** Keep existing palette
**Notes:** ANSI colors stay consistent across all themes. Only the UI-relevant fields change.

| Option | Description | Selected |
|--------|-------------|----------|
| Read computed styles | getComputedStyle(documentElement) — reads resolved CSS var values regardless of source. | ✓ |
| Use prop values directly | App.tsx passes customColors as a prop object, Terminal maps from it. | |

**User's choice:** Read computed styles
**Notes:** Automatically handles custom themes without needing to pass extra props.

| Option | Description | Selected |
|--------|-------------|----------|
| Minimal: bg + fg + cursor | Map only --background, --foreground, --primary (cursor). Selection at reduced opacity. | ✓ |
| Full: bg+fg+cursor+selection+tab | Also map --accent, --tab-active-bg, --border. | |

**User's choice:** Minimal mapping
**Notes:** Covers 90% of visual integration. Cmdex CSS vars don't have direct xterm equivalents for everything.

---

## Font Update

| Option | Description | Selected |
|--------|-------------|----------|
| Hot-swap fontFamily | Separate useEffect that calls term.options.fontFamily live — no recreate, no flicker. | |
| Keep full recreate | Current behavior — font change causes terminal disposal and recreation. | ✓ |

**User's choice:** Keep full recreate
**Notes:** Font changes are rare (only via Settings window). Simple, no edge cases with font metrics or cell recalc.

| Option | Description | Selected |
|--------|-------------|----------|
| Add fade transition | 150ms CSS opacity fade-in/out on terminal container when font changes. | ✓ |
| No transition | Instant recreate, brief flash. | |

**User's choice:** Add fade transition
**Notes:** 150ms matches existing app-wide transition duration.

---

## Copy/paste

| Option | Description | Selected |
|--------|-------------|----------|
| Rely on xterm defaults | No explicit copy/paste handlers — xterm.js handles selection copy, paste flows through onData → PTY. | ✓ |
| Add explicit keybindings | Explicit Cmd+C / Ctrl+Shift+C copy and Cmd+V / Ctrl+Shift+V paste handlers. | |

**User's choice:** Rely on xterm defaults
**Notes:** Existing pipeline already handles paste (keystrokes → onData → buffered → Write). Copy via selection works natively.

---

## SearchAddon

| Option | Description | Selected |
|--------|-------------|----------|
| Standard Ctrl/Cmd+F | Ctrl/Cmd+F opens SearchAddon bar with xterm's built-in keyboard handling. | |
| Custom shortcut mapping | Map to project keyboard shortcut system. | |
| Don't support | Defer search to future phase. | ✓ |

**User's choice:** Deferred
**Notes:** POL-05 (Ctrl+F search) explicitly excluded from Phase 19 scope. May return in a future phase.

---

## Deferred Ideas

- **SearchAddon (POL-05):** Ctrl+F scrollback search via `@xterm/addon-search` — deferred to future phase.
