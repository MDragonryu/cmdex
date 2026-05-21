# Phase 19: Terminal Polish - Context

**Gathered:** 2026-05-21
**Status:** Ready for planning

<domain>
## Phase Boundary

Sync the xterm.js terminal's visual appearance (theme, font) with Cmdex's existing theme and font settings. Ensure copy/paste works reliably. Terminal feels native and integrated, not like a separate embedded component.
</domain>

<decisions>
## Implementation Decisions

### Theme Sync
- **D-01:** Use CSS variable mapping — read computed styles via `getComputedStyle(document.documentElement)` at theme change time, map to xterm's `ITheme` fields. Works for both built-in and custom themes automatically.
- **D-02:** Pass theme as prop from App.tsx to Terminal (matching existing `monoFont` prop pattern). App.tsx already owns `theme` and `customThemes` state. Theme changes infrequently — prop-based approach is cleanest.
- **D-03:** Keep the existing hardcoded ANSI 16-color palette (VSCode Dark+ inspired). Only override background, foreground, cursor, and selection background from CSS vars. ANSI colors stay consistent across all Cmdex themes.
- **D-04:** Minimal CSS var → ITheme mapping: `--background` → `background`, `--foreground` → `foreground`, `--primary` → `cursor`. Selection background uses `--primary` at reduced opacity. Cursor accent is the inverse of cursor.
- **D-05:** Apply theme via `term.options.theme = {...}` on the live Terminal instance. No flicker, no recreate needed for theme changes (xterm supports hot theme swap).

### Font Update
- **D-06:** Keep full terminal recreate on font change (`[monoFont]` dep in Terminal's main `useEffect`). Font changes are rare (only via Settings window) — simplest approach with no edge cases around font metrics or cell recalculation.
- **D-07:** Add a 150ms CSS opacity fade transition on the terminal container to smooth the recreate flash. Matches the existing app-wide 150ms transition style established in v1.0.

### Copy/Paste
- **D-08:** Rely on xterm.js defaults. Xterm handles selection-to-clipboard on mouse release natively. Paste flows through the existing `term.onData` → buffered → `TerminalService.Write` → PTY pipeline. No explicit copy/paste handlers needed.

### SearchAddon
- **D-09:** SearchAddon (POL-05) deferred to a future phase. Not implementing scrollback search at this time.
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Terminal Component
- `frontend/src/components/Terminal.tsx` — xterm.js Terminal component: addons (Fit/Webgl/WebLinks), event subscriptions (pty-output, pty-exit, cmd-output), keystroke buffering, forwardRef clear() API, hardcoded theme and font
- `frontend/src/wails/events.ts` — Event name constants: ptyOutput, ptyExit, settingsChanged

### Theme System
- `frontend/src/lib/theme-apply.ts` — `applyTheme(themeId, customColors?)`: sets data-theme attr + CSS custom properties on documentElement
- `frontend/src/style.css` — CSS variable definitions (--background, --foreground, --primary, etc.), @theme mapping for Tailwind, bundled @font-face declarations for all 7 mono fonts
- `frontend/src/types.ts` — `THEMES` constant (8 built-in themes), `CustomTheme` interface

### Settings & Sync
- `frontend/src/App.tsx` — theme state management (setTheme, settingsRef), settings-changed event handler (line 515-555), monoFont state, Terminal rendering with monoFont prop (line 1477-1481), `applyTheme`/`applyDensity`/`applyFonts` usage
- `frontend/src/components/SettingsPage.tsx` — Settings window theme picker, font picker, emits settings-changed on change

### Go Backend
- `terminal_service.go` — TerminalService: Start, Stop, Write, Resize (PTY lifecycle)
- `event_service.go` — EventNames struct registration
- `main.go` — Service registration for TerminalService
</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **Terminal.tsx**: xterm.js instance with FitAddon, WebglAddon, WebLinksAddon. Ready to add theme-aware useEffect. Already receives `monoFont` prop. `forwardRef` + `useImperativeHandle` exposes clear().
- **theme-apply.ts**: `applyTheme(themeId, customColors?)` already sets all CSS vars on `document.documentElement`. Terminal can read these directly via `getComputedStyle`.
- **THEMES constant (types.ts)**: 8 themes with id/label/type. Can be used to detect dark vs light for cursor defaults.

### Established Patterns
- **Wails event sync:** Settings window emits `settings-changed` event → App.tsx listens and updates state → triggers side effects (applyTheme, applyFonts). Terminal should follow this same downstream-consumer pattern.
- **150ms transitions:** App-wide CSS transition duration. Terminal fade-in/out should match.
- **Tab persistence:** Terminal stays mounted across tab switches via CSS `display: none` toggle, not unmount. Theme/font changes apply to the single mounted terminal instance.
- **xterm hot config:** `term.options.theme` and `term.options.fontFamily` can be set on a live Terminal instance without recreation. Theme uses this; font does not (by decision D-06).

### Integration Points
- **App.tsx → Terminal props:** Currently `monoFont` and `isVisible`. Add `theme` (string) prop. Terminal's theme useEffect watches this prop, reads computed CSS styles, applies ITheme.
- **Theme change flow:** Settings window → settings-changed event → App.tsx setTheme → applyTheme (CSS vars on document) → Terminal prop update → terminal theme sync.
- **Font change flow:** Settings window → settings-changed → App.tsx setMonoFont → applyFonts + Terminal remount via [monoFont] dep → 150ms fade transition.
</code_context>

<specifics>
## Specific Ideas

No specific references or external examples — standard implementation.
</specifics>

<deferred>
## Deferred Ideas

- **SearchAddon (POL-05):** Ctrl+F scrollback search via `@xterm/addon-search`. Deferred to a future phase — user explicitly excluded from this phase scope.

None otherwise — discussion stayed within phase scope.
</deferred>

---

*Phase: 19-terminal-polish*
*Context gathered: 2026-05-21*
