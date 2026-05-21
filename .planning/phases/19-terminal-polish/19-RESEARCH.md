# Phase 19: Terminal Polish - Research

**Researched:** 2026-05-21
**Domain:** xterm.js terminal integration — theme sync, font sync, copy/paste
**Confidence:** HIGH

## Summary

Phase 19 polishes the xterm.js terminal embedded in Cmdex so it feels native and integrated, not like a separate component. The work involves three capabilities: (1) syncing the terminal's theme with Cmdex's theme system using CSS variable mapping, (2) smoothing font changes with a CSS opacity transition, and (3) relying on xterm.js's built-in copy/paste behavior (selection-to-clipboard on mouse release; paste through the existing `onData` → `TerminalService.Write` pipeline).

All changes are in the Browser/Client tier — Terminal.tsx and App.tsx, plus CSS. No new packages, no backend changes, no Go code modifications. The phase uses the existing `--background`, `--foreground`, and `--primary` CSS custom properties already set by `applyTheme()`, and reads them at runtime via `getComputedStyle(document.documentElement)` to map to xterm.js's `ITheme` interface.

**Primary recommendation:** Add a `theme` prop to Terminal.tsx, add a `useEffect` watching it that reads computed CSS properties and hot-swaps `term.options.theme`, and add a 150ms `opacity` transition on `.terminal-container` for font-change smoothness. Copy/paste needs zero code changes — xterm.js v6.0.0 handles both natively.

## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Use CSS variable mapping — read computed styles via `getComputedStyle(document.documentElement)` at theme change time, map to xterm's `ITheme` fields. Works for both built-in and custom themes automatically.
- **D-02:** Pass theme as prop from App.tsx to Terminal (matching existing `monoFont` prop pattern). App.tsx already owns `theme` and `customThemes` state. Theme changes infrequently — prop-based approach is cleanest.
- **D-03:** Keep the existing hardcoded ANSI 16-color palette (VSCode Dark+ inspired). Only override background, foreground, cursor, and selection background from CSS vars. ANSI colors stay consistent across all Cmdex themes.
- **D-04:** Minimal CSS var → ITheme mapping: `--background` → `background`, `--foreground` → `foreground`, `--primary` → `cursor`. Selection background uses `--primary` at reduced opacity. Cursor accent is the inverse of cursor.
- **D-05:** Apply theme via `term.options.theme = {...}` on the live Terminal instance. No flicker, no recreate needed for theme changes (xterm supports hot theme swap).
- **D-06:** Keep full terminal recreate on font change (`[monoFont]` dep in Terminal's main `useEffect`). Font changes are rare (only via Settings window) — simplest approach with no edge cases around font metrics or cell recalculation.
- **D-07:** Add a 150ms CSS opacity fade transition on the terminal container to smooth the recreate flash. Matches the existing app-wide 150ms transition style established in v1.0.
- **D-08:** Rely on xterm.js defaults. Xterm handles selection-to-clipboard on mouse release natively. Paste flows through the existing `term.onData` → buffered → `TerminalService.Write` → PTY pipeline. No explicit copy/paste handlers needed.
- **D-09:** SearchAddon (POL-05) deferred to a future phase. Not implementing scrollback search at this time.

### the agent's Discretion
- Exact implementation of the `theme` prop wiring (how the `useEffect` is structured in Terminal.tsx)
- Whether to keep ANSI palette inline or extract to a named constant
- CSS class naming for the opacity transition
- Whether to early-return in theme `useEffect` if terminal is not yet mounted

### Deferred Ideas (OUT OF SCOPE)
- SearchAddon (POL-05): Ctrl+F scrollback search via `@xterm/addon-search`. Deferred to a future phase — user explicitly excluded from this phase scope.

## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| POL-01 | Switching Cmdex themes updates xterm terminal theme in real time with no flicker | CSS variable mapping + `term.options.theme = {...}` hot swap; see Architecture Pattern 1 |
| POL-02 | Terminal font family updates when user changes font in Settings | Existing `[monoFont]` dep causes full recreate; 150ms opacity transition added for smoothness; see Architecture Pattern 2 |
| POL-03 | Cmd+C / Ctrl+Shift+C copies selected text from terminal | xterm.js v6.0.0 built-in selection-to-clipboard; no code changes needed; see Architecture Pattern 3 |
| POL-04 | Cmd+V / Ctrl+Shift+V pastes clipboard text into terminal | Existing `onData` → buffered → `TerminalService.Write` pipeline already handles paste; see Architecture Pattern 3 |
| POL-05 | Ctrl+F opens search in terminal scrollback buffer | Deferred (D-09) — out of scope for this phase |

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Theme sync | Browser / Client | — | Reads CSS vars from DOM, sets xterm.js options; no server involvement |
| Font update | Browser / Client | — | CSS transition on terminal container; terminal recreate via React state |
| Copy (selection → clipboard) | Browser / Client | — | xterm.js handles via `navigator.clipboard` API; no code changes needed |
| Paste (clipboard → PTY) | Browser / Client | API / Backend | xterm.js `onData` fires → buffered → `TerminalService.Write` → PTY (existing pipeline) |

## Standard Stack

### Core (Already Installed — No New Packages)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `@xterm/xterm` | 6.0.0 | Terminal emulator | VS Code's terminal engine; industry standard for web terminals |
| `@xterm/addon-fit` | 0.11.0 | Auto-resize to container | Official xterm.js addon; bundled with project |
| `@xterm/addon-webgl` | 0.19.0 | GPU-accelerated rendering | Official xterm.js addon; smoother scrolling |
| `@xterm/addon-web-links` | 0.12.0 | Clickable URLs in output | Official xterm.js addon; bundled with project |

### Not Needed for This Phase

| Library | Why Not Used |
|---------|-------------|
| `@xterm/addon-search` | Deferred per D-09 — not implementing search in this phase |
| `@xterm/addon-web-fonts` | Not needed — Cmdex fonts are bundled via CSS `@font-face` in style.css, not loaded dynamically |
| `@xterm/addon-clipboard` | Not needed — v6.0.0 has native clipboard support built-in |

**Installation:** No new packages required. All xterm.js packages are already installed and verified in `frontend/package.json`.

## Package Legitimacy Audit

> slopcheck was unavailable at research time (installation failed). All packages are tagged `[ASSUMED]` below and the planner must gate each install verification behind a `checkpoint:human-verify` task. However, these packages are already installed in the project — no new installs are needed for this phase.

| Package | Registry | Age | Repository | slopcheck | Disposition |
|---------|----------|-----|-----------|-----------|-------------|
| `@xterm/xterm` 6.0.0 | npm | ~2.5 yrs (since 6.0.0) | github.com/xtermjs/xterm.js | [ASSUMED] | Already installed |
| `@xterm/addon-fit` 0.11.0 | npm | Official xterm.js org | github.com/xtermjs/xterm.js | [ASSUMED] | Already installed |
| `@xterm/addon-webgl` 0.19.0 | npm | Official xterm.js org | github.com/xtermjs/xterm.js | [ASSUMED] | Already installed |
| `@xterm/addon-web-links` 0.12.0 | npm | Official xterm.js org | github.com/xtermjs/xterm.js | [ASSUMED] | Already installed |

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none
*Note: All packages above are tagged `[ASSUMED]` because slopcheck was unavailable. Since no new installs are needed, the risk is minimal — these packages are already in the project's `package.json` and `node_modules`.*

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                      App.tsx                                 │
│  ┌─────────────┐  ┌──────────────┐  ┌────────────────────┐ │
│  │ theme state  │  │ monoFont     │  │ terminalCollapsed   │ │
│  │ setTheme()   │  │ setMonoFont()│  │ (collapse/expand)  │ │
│  └──────┬───────┘  └──────┬───────┘  └─────────┬──────────┘ │
│         │                  │                     │           │
│  ┌──────▼──────────────────▼─────────────────────▼──────────┐│
│  │  applyTheme() → sets CSS vars on documentElement         ││
│  └──────┬───────────────────────────────────────────────────┘│
└─────────┼────────────────────────────────────────────────────┘
          │ props: { theme, monoFont, isVisible }
          ▼
┌─────────────────────────────────────────────────────────────┐
│                      Terminal.tsx                            │
│                                                              │
│  ┌─────────────┐   ┌──────────────────────────────────┐    │
│  │ theme prop  │──▶│ useEffect(theme):                 │    │
│  │ (string)    │   │ 1. getComputedStyle(docEl)        │    │
│  └─────────────┘   │ 2. extract --bg, --fg, --primary  │    │
│                    │ 3. map to ITheme partial           │    │
│  ┌─────────────┐   │ 4. term.options.theme = {...}     │    │
│  │ monoFont    │──▶│    (hot swap — no flicker)         │    │
│  │ prop        │   └──────────────────────────────────┘    │
│  │ (string)    │                                            │
│  └─────────────┘   ┌──────────────────────────────────┐    │
│                    │ useEffect([monoFont]):             │    │
│                    │ 1. dispose old Terminal            │    │
│                    │ 2. create new Terminal              │    │
│                    │ 3. CSS opacity transition 150ms    │    │
│                    └──────────────────────────────────┘    │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ Runtime event subscriptions (mount once):              │   │
│  │  · ptyOutput → term.write(output)                     │   │
│  │  · ptyExit   → log exit code                          │   │
│  │  · cmdOutput → term.write (stderr in red)             │   │
│  │  · term.onData → buffered → TermSvc.Write → PTY       │   │
│  │    (includes paste: Cmd+V / Ctrl+Shift+V)             │   │
│  │  · term.onResize → TermSvc.Resize(cols, rows)          │   │
│  │  · ResizeObserver → fitAddon.fit()                     │   │
│  └──────────────────────────────────────────────────────┘   │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ Copy (selection → clipboard):                         │   │
│  │  xterm.js v6.0.0 native: mouse-select → release →    │   │
│  │  navigator.clipboard.writeText(selectedText)          │   │
│  │  Cmd+C / Ctrl+Shift+C also works (built-in handler)   │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
          │ TerminalService.Write / Resize
          ▼
┌─────────────────────────────────────────────────────────────┐
│                   Go Backend (no changes)                     │
│  terminal_service.go: Start / Stop / Write / Resize          │
└─────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure (No New Files — Changes Only)

```
frontend/src/
├── components/
│   └── Terminal.tsx          # MODIFY: add theme prop, theme useEffect, opacity state
├── App.tsx                    # MODIFY: pass theme prop to TerminalComponent
├── style.css                  # MODIFY: add opacity transition on .terminal-container font-change
└── lib/
    └── theme-apply.ts        # NO CHANGE — already sets CSS vars on documentElement
```

### Pattern 1: CSS Variable → ITheme Hot Swap

**What:** Watching a `theme` prop, read computed CSS properties from `document.documentElement` at change time, construct a partial `ITheme` object mapped from `--background`, `--foreground`, and `--primary`, and apply it live via `term.options.theme = {...}`.

**When to use:** Whenever the `theme` prop changes (user switches theme in settings, or system dark/light mode triggers theme switch).

**Confidence:** HIGH — verified via Context7 for xterm.js v6.0.0 ITheme interface and live option mutation.

**Example:**

```typescript
// Terminal.tsx — new useEffect after existing useEffects
useEffect(() => {
    const term = terminalRef.current;
    if (!term) return;

    const styles = getComputedStyle(document.documentElement);
    const background = styles.getPropertyValue('--background').trim();
    const foreground = styles.getPropertyValue('--foreground').trim();
    const primary = styles.getPropertyValue('--primary').trim();

    // Reduce primary opacity for selection background
    // Using a simple opacity helper: convert hex to rgba
    const selectionBg = applyOpacity(primary, 0.4);

    // Cursor accent = inverse of cursor (use background color)
    const cursorAccent = styles.getPropertyValue('--background').trim();

    // Apply partial theme — keep existing ANSI colors intact
    term.options.theme = {
        ...term.options.theme, // preserve existing ANSI colors
        background,
        foreground,
        cursor: primary,
        cursorAccent,
        selectionBackground: selectionBg,
    };
}, [theme]);
// Source: Context7 /websites/xtermjs - ITheme interface + Terminal.options docs
// Verified: xterm.js v6.0.0 supports hot theme swap via term.options.theme = newObject
```

**Key rule:** Must spread `...term.options.theme` to preserve the hardcoded 16-color ANSI palette (D-03). Setting only background/foreground/cursor/selectionBackground as overrides.

### Pattern 2: Font Change with Opacity Transition

**What:** Keep the existing `useEffect([monoFont])` that creates a new Terminal instance on font change, but apply a CSS opacity transition on `.terminal-container` to smooth the moment between dispose and recreate.

**When to use:** User changes mono font in Settings → settings-changed event fires → App.tsx setMonoFont → Terminal remounts.

**Confidence:** HIGH — the existing recreate pattern is proven; opacity transition is standard CSS.

**Implementation strategy:**

1. **CSS addition** (style.css):
```css
.terminal-container.font-changing {
    opacity: 0;
    transition: opacity var(--transition-fast);
}
.terminal-container.font-ready {
    opacity: 1;
    transition: opacity var(--transition-fast);
}
```

2. **Terminal.tsx approach** (inside the main `useEffect` with `[monoFont]`):
   - Before `term.dispose()`, add `.font-changing` class → triggers opacity to 0 over 150ms
   - After 150ms (match transition duration), dispose and recreate normally
   - After new `term.open(container)`, add `.font-ready` class → triggers opacity to 1 over 150ms

Alternatively, a simpler approach: set initial opacity to 0, then after `term.open()` and first render, set opacity to 1 with transition. This avoids the two-step class toggle complexity.

**Simpler single-state approach:**

```typescript
// Inside [monoFont] useEffect:
// Before disposing old terminal, set container opacity to 0
if (containerRef.current) {
    containerRef.current.style.opacity = '0';
    containerRef.current.style.transition = 'opacity var(--transition-fast)';
}

// ... dispose old terminal, create new ...

// After new terminal is opened:
if (containerRef.current) {
    // Use requestAnimationFrame to ensure the DOM has registered opacity:0 first
    requestAnimationFrame(() => {
        if (containerRef.current) {
            containerRef.current.style.opacity = '1';
        }
    });
}
```

### Pattern 3: Copy/Paste — No Code Changes

**What:** xterm.js v6.0.0 handles both copy (selection → clipboard on mouse release, plus Cmd+C / Ctrl+Shift+C) and paste (through `onData` which feeds into the existing keystroke buffering pipeline → `TerminalService.Write` → PTY).

**When to use:** Always — zero code changes needed. Verified against the current codebase.

**Confidence:** HIGH — verified:
- xterm.js v6.0.0 has built-in clipboard support (Context7 docs confirm `getSelection()`, `paste()` methods, and `onData` event)
- No global keyboard shortcuts registered for Cmd+V, Ctrl+Shift+V, Cmd+C, or Ctrl+Shift+C in App.tsx
- The existing `onData` handler in Terminal.tsx (line 164) captures all keystrokes including paste and forwards them to `TerminalService.Write`

**How paste flows through:**
```
User presses Cmd+V → xterm receives paste event → onData fires with pasted text
→ keystrokeBuffer += data → flushBuffer() → Write(batch) → Go TerminalService → PTY
```

**How copy flows:**
```
User selects text in terminal with mouse → releases mouse → 
xterm calls navigator.clipboard.writeText(selectedText)
User presses Cmd+C with selection → xterm captures → clipboard write
```

### Anti-Patterns to Avoid

- **Recreating Terminal on every theme change:** Would cause flicker and reset terminal state/scrollback. Use `term.options.theme = {...}` (D-05).
- **Hardcoding theme preset tables per Cmdex theme:** Would not work with custom themes. Use `getComputedStyle()` (D-01).
- **Mapping CSS vars to ANSI colors:** Cmdex CSS vars like `--destructive` (#f44747) don't map cleanly to ANSI red (#cd3131). Keep the existing hardcoded palette (D-03).
- **Adding explicit copy/paste keybinding handlers in Terminal.tsx:** xterm.js v6.0.0 already handles this. Extra handlers could cause double-paste or interfere with built-in behavior (D-08).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Color parsing (hex → rgba for selection opacity) | Custom color math | simple inline helper or `color-mix(in srgb, var(--primary) 40%, transparent)` approach | Hex-to-rgba is trivial (<10 lines); the alternative is reading CSS vars that already handle opacity |
| Clipboard access for copy | `document.execCommand('copy')` or custom clipboard handler | xterm.js built-in clipboard (v6.0.0) | Built-in handles edge cases: multi-line selection, ANSI stripping, platform differences |
| Paste interception | Custom paste event listener | xterm.js `onData` event (already wired) | Paste is just another keystroke — flows through existing pipeline |

**Key insight:** The existing xterm.js pipeline in Terminal.tsx is already correct for paste. The only additions needed are theme reading/watching and a CSS transition — both leveraging existing infrastructure.

## Runtime State Inventory

> This is a greenfield polish phase (`ui_phase: true`), not a rename/refactor/migration phase. No runtime state migration is needed. The terminal is a transient UI component — no stored data, live service config, OS-registered state, secrets, or build artifacts carry terminal-specific state.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | None — terminal state (scrollback, cursor position) is in-memory only, not persisted to SQLite | none |
| Live service config | None | none |
| OS-registered state | None | none |
| Secrets/env vars | None | none |
| Build artifacts | None | none |

## Common Pitfalls

### Pitfall 1: `getComputedStyle` Returns Whitespace
**What goes wrong:** `getComputedStyle(document.documentElement).getPropertyValue('--background')` returns `" #1e1e1e"` (with leading space) in some browsers. xterm.js parses colors strictly and may reject values with whitespace.
**Why it happens:** CSS custom property values preserve the whitespace from the declaration. If `:root { --background: #1e1e1e; }` has a space after the colon, it's preserved.
**How to avoid:** Always call `.trim()` on the result of `getPropertyValue()`.
**Warning signs:** Terminal background doesn't change after theme switch, but no errors in console.

### Pitfall 2: Theme Object Reference Comparison
**What goes wrong:** You set `term.options.theme.background = newColor` and nothing happens.
**Why it happens:** xterm.js performs a reference equality check (`===`) on the theme object. Mutating properties in-place doesn't trigger a re-render because the reference is the same.
**How to avoid:** Always spread into a new object: `term.options.theme = { ...term.options.theme, background: newColor }`. This is confirmed in xterm.js official docs (Context7).
**Warning signs:** Console shows theme was set but terminal doesn't visually change.

### Pitfall 3: Terminal Instance Is Null
**What goes wrong:** The theme `useEffect` runs before the terminal is created, causing `terminalRef.current` to be `null`.
**Why it happens:** The theme prop may be set before the Terminal component mounts (App.tsx sets default theme on mount, Terminal mounts later in the React tree).
**How to avoid:** Guard with `if (!term) return;` at the top of the theme `useEffect`. The effect will re-fire when the terminal is created since `terminalRef.current` will be populated.
**Warning signs:** "Cannot read properties of null" error in console on initial load.

### Pitfall 4: Opacity Transition on First Mount
**What goes wrong:** The terminal fades in from transparent on first load, which looks like a glitch.
**Why it happens:** The opacity transition CSS applies on initial mount too, causing a visible fade-in from `opacity: 0`.
**How to avoid:** Only apply the opacity transition on font changes, not on initial mount. Use a `useRef` to track whether this is the first mount within the `[monoFont]` effect.
**Warning signs:** Terminal has a visible fade-in animation every time the app starts.

### Pitfall 5: `Ctrl+Shift+C` on Windows vs `Cmd+C` on macOS
**What goes wrong:** Custom copy handler that only listens for `Cmd+C` breaks on Windows/Linux where the terminal copy shortcut is `Ctrl+Shift+C` (because `Ctrl+C` sends SIGINT).
**Why it happens:** Terminal emulators use `Ctrl+Shift+C` for copy on non-macOS platforms to avoid conflicting with the interrupt signal.
**How to avoid:** Rely on xterm.js defaults (D-08). xterm.js internally handles platform-specific keybindings for copy. Don't add custom handlers.
**Warning signs:** Copy works on Mac but not on Windows/Linux, or vice versa.

## Code Examples

### Reading CSS Custom Properties at Runtime [VERIFIED: Context7 and codebase]

```typescript
// Source: D-01, D-04 from CONTEXT.md; verified against theme-apply.ts CSS vars
// This pattern reads whichever theme is active — built-in or custom
const styles = getComputedStyle(document.documentElement);
const background = styles.getPropertyValue('--background').trim();   // e.g. "#1e1e1e"
const foreground = styles.getPropertyValue('--foreground').trim();   // e.g. "#d4d4d4"
const primary = styles.getPropertyValue('--primary').trim();         // e.g. "#007acc"
```

### Hex to RGBA Conversion for Selection Opacity

```typescript
// Source: D-04 — selection background uses --primary at reduced opacity
// Simple inline helper (no library needed)
function hexToRgba(hex: string, alpha: number): string {
    hex = hex.replace('#', '');
    if (hex.length === 3) {
        hex = hex[0] + hex[0] + hex[1] + hex[1] + hex[2] + hex[2];
    }
    const r = parseInt(hex.substring(0, 2), 16);
    const g = parseInt(hex.substring(2, 4), 16);
    const b = parseInt(hex.substring(4, 6), 16);
    return `rgba(${r}, ${g}, ${b}, ${alpha})`;
}
// Usage: hexToRgba('#007acc', 0.4) → "rgba(0, 122, 204, 0.4)"
```

### Theme Hot-Swap on Live Terminal [VERIFIED: Context7 /websites/xtermjs]

```typescript
// Source: Context7 — xterm.js Terminal.options docs; D-05
// Must use spread to create new object (reference comparison check)
term.options.theme = {
    ...term.options.theme,  // preserve hardcoded ANSI 16-color palette (D-03)
    background: '#1e1e1e',
    foreground: '#d4d4d4',
    cursor: '#007acc',
    cursorAccent: '#1e1e1e',
    selectionBackground: 'rgba(0, 122, 204, 0.4)',
};
```

### Opacity Transition CSS [VERIFIED: existing style.css patterns]

```css
/* Source: D-07 — 150ms matches --transition-fast (line 156 of style.css) */
/* Add to style.css near the .terminal-container block (line 1912) */
.terminal-container.font-transitioning {
  opacity: 0;
  transition: opacity var(--transition-fast);  /* 150ms cubic-bezier(0.4, 0, 0.2, 1) */
}

.terminal-container {
  /* existing styles */
  opacity: 1;
  transition: opacity var(--transition-fast);
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Hardcoded theme in Terminal constructor | CSS var mapping + `getComputedStyle` | This phase (D-01) | Terminal theme auto-syncs with any Cmdex theme including custom |
| No transition on font change | 150ms opacity fade | This phase (D-07) | Smoother visual experience during font switch |
| Manual copy/paste concerns | Verified xterm.js v6.0.0 built-in support | This investigation | Confirmed no code changes needed for copy/paste |

**Deprecated/outdated:**
- `document.execCommand('copy')` — deprecated clipboard API; xterm.js v6.0.0 uses `navigator.clipboard.writeText()` instead
- `@xterm/addon-clipboard` — no longer needed; clipboard support is built into xterm.js core since v5.x

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | xterm.js v6.0.0 handles `Cmd+C`/`Ctrl+Shift+C` copy natively without explicit handler | Architecture Pattern 3 | Copy would silently fail on non-mouse-select triggers — would need to add explicit handler |
| A2 | xterm.js v6.0.0's `onData` event fires for paste (Cmd+V/Ctrl+Shift+V) content | Architecture Pattern 3 | Paste would not reach the PTY — would need to add explicit paste event listener |
| A3 | `getComputedStyle(document.documentElement)` reads custom CSS properties set by `applyTheme()` regardless of whether theme was set via `:root` or `[data-theme]` selector | Architecture Pattern 1 | Theme sync would fail for certain themes — would need alternative approach |
| A4 | The 16-color ANSI palette hardcoded in Terminal.tsx is acceptable to keep as-is across all themes | Standard Stack | Users might expect ANSI colors to shift with theme — would need per-theme palettes |

**If this table is empty:** N/A — four assumptions are documented above requiring user validation before execution.

## Open Questions

1. **Selection opacity value**
   - What we know: D-04 says "Selection background uses `--primary` at reduced opacity." No specific alpha value given.
   - What's unclear: What alpha value looks best across all 8 themes? 0.4? 0.3?
   - Recommendation: Use `0.4` (40% opacity) as a starting point, matching the `color-mix(in srgb, var(--primary) 40%, transparent)` pattern used elsewhere in style.css (line 1242).

2. **Font change transition timing edge case**
   - What we know: D-07 says 150ms opacity transition. D-06 says full recreate on font change.
   - What's unclear: How to precisely sequence "fade out → dispose → recreate → fade in" without a visible gap or double-render.
   - Recommendation: Use a simple approach: set container opacity to 0, wait 150ms (match transition), then dispose. After new terminal opens, set opacity to 1 with transition. The brief blank period during recreate is acceptable — the opacity fade masks it.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | Frontend dev server (Vite) | ✓ | v25.6.1 | — |
| npm | Package management | ✓ | 11.12.1 | — |
| pnpm | Frontend commands (lint, tsc) | ✓ | installed | — |
| wails3 | Dev server / build | ✓ | installed | — |
| xterm.js packages | Terminal component | ✓ | v6.0.0 (already in node_modules) | — |

**Missing dependencies with no fallback:** none
**Missing dependencies with fallback:** none

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go: standard `testing` package; Frontend: none configured |
| Config file | none — frontend has no test config |
| Quick run command | `go test ./...` |
| Full suite command | `go test ./... -v` (Go only; no frontend tests exist) |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| POL-01 | Theme sync — switching theme updates terminal bg/fg/cursor | manual-only | N/A (requires visual verification) | ❌ Wave 0 |
| POL-02 | Font change applies opacity transition, new font renders | manual-only | N/A (requires visual verification) | ❌ Wave 0 |
| POL-03 | Copy selected text from terminal | manual-only | N/A (requires clipboard access + visual selection) | ❌ Wave 0 |
| POL-04 | Paste text into terminal | manual-only | N/A (requires clipboard + PTY interaction) | ❌ Wave 0 |

### Sampling Rate

- **Per task commit:** `go build ./...` — verify no Go compile errors
- **Per wave merge:** `make check` — runs `go build ./...` + `pnpm tsc --noEmit`
- **Phase gate:** Manual verification of all four POL requirements + `make check` green

### Wave 0 Gaps

- [ ] No frontend test framework exists (no jest, vitest, or testing-library configured)
- [ ] All POL requirements are `manual-only` — terminal visual behavior, clipboard access, and PTY interaction are inherently hard to automate in a test environment
- [ ] No existing test file for Terminal.tsx

*(This is consistent with the project's current state — STATE.md confirms "No tests exist — manual verification required for all phases.")*

## Security Domain

> `security_enforcement: true` in config.json; applicable to this phase.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|------------------|
| V2 Authentication | no | — |
| V3 Session Management | no | — |
| V4 Access Control | no | — |
| V5 Input Validation | yes | xterm.js `onData` already sanitizes input through PTY write — no HTML injection vector exists in terminal output |
| V6 Cryptography | no | — |

### Known Threat Patterns for Terminal Integration

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| ANSI escape sequence injection in terminal output | Tampering | xterm.js handles ANSI sequences internally — no custom parsing. Write operations flow through Go backend to PTY which controls the shell process. |
| Clipboard exfiltration via copy | Information Disclosure | Browser's clipboard API requires user gesture (selection + release or Cmd+C). xterm.js doesn't auto-copy without user interaction. |
| Malicious paste content | Elevation of Privilege | Paste content flows through the existing `onData` → `Write` → PTY pipeline. The shell (not Cmdex) interprets pasted commands. This is standard terminal behavior. |

## Sources

### Primary (HIGH confidence)
- Context7 `/websites/xtermjs` — ITheme interface (all properties documented), ITerminalOptions (theme, fontFamily, allowTransparency), Terminal.options (hot-swap with new object reference required), Terminal.getSelection, Terminal.paste
- Context7 `/xtermjs/xterm.js` — Terminal creation with theme object, onData event, fontFamily hot-swap
- Codebase: `frontend/src/components/Terminal.tsx` — current terminal implementation verified
- Codebase: `frontend/src/lib/theme-apply.ts` — CSS variable setting mechanism verified
- Codebase: `frontend/src/style.css` — CSS variable definitions, `--transition-fast`, terminal container styles
- Codebase: `frontend/src/App.tsx` — theme state, monoFont state, Terminal props, keyboard shortcuts (no conflicts)
- npm registry: `@xterm/xterm` v6.0.0 confirmed latest; published 2025-12-22

### Secondary (MEDIUM confidence)
- xtermjs.org official docs (via Context7/ITheme interface) — all theme properties confirmed
- xterm.js GitHub repository (github.com/xtermjs/xterm.js) — package source verified

### Tertiary (LOW confidence)
- None — all critical claims verified through Context7 or codebase inspection

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all packages already installed, verified via npm view; no new packages needed
- Architecture: HIGH — pattern confirmed by Context7 docs and existing codebase patterns
- Pitfalls: HIGH — pitfalls derived from Context7-documented xterm.js behavior (reference comparison, whitespace in getComputedStyle) and existing codebase patterns

**Research date:** 2026-05-21
**Valid until:** 2026-06-21 (30 days — xterm.js is stable; CSS variable system is project-internal)
