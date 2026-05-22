# Phase 20: Terminal Copy Buttons - Context

**Gathered:** 2026-05-22
**Status:** Ready for planning

<domain>
## Phase Boundary

Add Copy Command and Copy Output buttons to the terminal toolbar. Go emits a `cmd-executing` Wails event with the resolved command text when execution starts. Frontend stores this in a ref and exposes it via copy buttons. Copy Output uses xterm.js's built-in `getSelection()` method exposed through the TerminalHandle forwardRef.
</domain>

<decisions>
## Implementation Decisions

### Event-Based Command Tracking
- **D-01:** Go backend emits a `cmd-executing` Wails event with the resolved command text before writing to PTY. Frontend listens via `Events.On` and stores in a `useRef` (not state — no re-render needed).
- **D-02:** Event payload is the raw `cmdLine` string from execution_service.go, including the `cd <dir> &&` prefix when a working directory is set.

### Copy Buttons
- **D-03:** Two buttons added to the terminal toolbar (alongside Collapse and Clear): "Copy Cmd" copies the last executed command, "Copy Out" copies selected terminal text.
- **D-04:** Copy uses `navigator.clipboard.writeText()` with a toast confirmation on success.
- **D-05:** Copy Out uses xterm.js's `getSelection()` method — no addon needed (built into xterm.js v6.0.0 core).

### getSelection() Exposure
- **D-06:** `getSelection()` is added to the `TerminalHandle` interface and `useImperativeHandle`, so the parent can call `terminalRef.current.getSelection()` without accessing internal xterm instance refs.

### Button Visibility
- **D-07:** Buttons are inside `!terminalCollapsed` branch — only visible when terminal is expanded.
</decisions>

<canonical_refs>
## Canonical References

### Terminal Component
- `frontend/src/components/Terminal.tsx` — xterm.js Terminal component, TerminalHandle interface, useImperativeHandle
- `frontend/src/App.tsx` — Terminal toolbar area (lines 1451-1473), event listener patterns (lines 515-555)

### Backend
- `execution_service.go` — RunCommand (line 114-152) where cmdLine is resolved and written to terminal
- `event_service.go` — EventNames struct and event registration pattern

### Events
- `frontend/src/wails/events.ts` — Wails event name constants and initEventNames sync pattern
</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Patterns
- **Wails event flow:** execution_service → `application.Get().Emit(eventNames.X, payload)` → `Events.On(eventNames.X, handler)` in frontend. Use this exact pattern for cmd-executing.
- **Event name registration:** Add to EventNames struct → add to eventNames var → run `wails3 generate build-assets` → add to events.ts → add to initEventNames().
- **TerminalHandle forwardRef:** `useImperativeHandle` exposes `clear()`. Add `getSelection()` following the same pattern.
- **Terminal toolbar buttons:** Collapse (▼) and Clear buttons use `onMouseDown={e => e.stopPropagation()}` to prevent triggering the drag divider. New buttons must do the same.
</code_context>

<deferred>
## Deferred Ideas

None — phase scope is tightly defined: two copy buttons + event plumbing.
</deferred>

---
*Phase: 20-terminal-copy-buttons*
*Context gathered: 2026-05-22*
