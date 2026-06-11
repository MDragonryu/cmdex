# Phase 23: Frontend Tabbed Terminal - Context

**Gathered:** 2026-06-10
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can manage and interact with multiple terminal sessions through a tabbed interface with full keyboard and mouse support. Phase 21 already provides the multi-session backend (CreateSession, ListSessions, CloseSession, RenameSession, SetActiveSession, GetActiveSession, namespaced events). This phase builds the frontend tab UI on top.

**Phase 23 delivers:** Tab bar UI, tab switching (click + keyboard), tab create/close/rename, per-tab terminal output with independent scrollback, status indicators, drag-and-drop reorder.

**Not in scope:** Session persistence (skipped in Phase 22), command execution integration (Phase 24).
</domain>

<decisions>
## Implementation Decisions

### Tab Creation & Closing
- **D-01:** '+' button at end of tab bar always visible, calls `CreateSession()`. Also Ctrl+T keyboard shortcut.
- **D-02:** Close via per-tab close button (x), Ctrl+W shortcut, or right-click context menu. Last remaining tab is not closeable (close button hidden/grayed, Ctrl+W no-op on last tab).
- **D-03:** Right-click context menu on a tab shows two actions: Rename (opens dialog/inline edit, calls `RenameSession`) and Close (grayed out for last tab). No Duplicate option.

### Tab Switching
- **D-04:** Click a tab to switch active session (calls `SetActiveSession`). Frontend caches `activeSessionId` locally for immediate UI, syncs with backend.
- **D-05:** Keyboard shortcuts: Ctrl+Tab (next tab), Ctrl+Shift+Tab (previous tab). Wrap around at ends.

### the Agent's Discretion
- **Tab bar implementation:** Leverage existing `TabBar` component (`frontend/src/components/TabBar.tsx`) with terminal-specific extensions (status indicator, right-click menu) rather than building a completely new component. The existing TabBar already handles reorder (drag-and-drop) and close.
- **Terminal rendering:** Mount one `TerminalComponent` per session, hide inactive via CSS (`display: none`). This preserves independent scrollback per session naturally — each xterm.js instance maintains its own buffer. Phase 21 already has `TerminalComponent` accepting `sessionId` prop and subscribing to namespaced events.
- **Tab reorder:** Drag-and-drop via existing TabBar mechanism. Reorder is visual-only (frontend state) since Phase 22 persistence was skipped.
- **Status indicators:** Each tab shows session name + running status (green dot = shell alive, gray dot = stopped). Status derived from `SessionInfo.running` field. Update on `pty-exit:{sessionId}` and `pty-output:{sessionId}` events.
- **Right-click menu:** Use Radix ContextMenu (already in shadcn/ui) or a custom dropdown positioned at cursor. Simple implementation with two items (Rename, Close).
- **Scrollback:** 5000 lines per session as configured in `Terminal.tsx` (`scrollback: 5000`). Each mounted TerminalComponent preserves its own scrollback when hidden.
- **Tab overflow:** When many tabs exist, tab bar scrolls horizontally (overflow-x: auto with hidden scrollbar, or arrow buttons). Follow existing TabBar overflow behavior.
- **Clear button:** Existing clear button in app clears only the active session's terminal (already works since `Clear(sessionId)` is called with the active session ID).
</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Roadmap & Requirements
- `.planning/ROADMAP.md` — Phase 23 definition, success criteria (9 items), dependencies
- `.planning/REQUIREMENTS.md` — SESS-02, SESS-03, SESS-06, UI-01 through UI-06

### Prior Phase Context
- `.planning/phases/21-backend-session-foundation/21-CONTEXT.md` — Session CRUD API, event namespacing, SessionInfo fields, active session management
- `.planning/phases/22-database-persistence/22-CONTEXT.md` — Phase 22 skipped, no persistence

### Existing Code (must study before implementing)
- `frontend/src/App.tsx` — Root state, existing TabBar usage, TerminalComponent mounting, `activeSessionId` state
- `frontend/src/components/Terminal.tsx` — TerminalComponent with `sessionId` prop, event subscriptions, method dispatch
- `frontend/src/components/TabBar.tsx` — Existing tab bar (command editor tabs), reorder, close, drag-and-drop
- `frontend/src/hooks/useKeyboardShortcuts.ts` — Keyboard shortcut registration pattern
- `frontend/src/style.css` — CSS variables for theming, existing tab bar styles
- `frontend/bindings/cmdex/terminalservice.js` — Generated bindings: CreateSession, ListSessions, CloseSession, RenameSession, SetActiveSession, GetActiveSession, Write, Resize, Start, Stop, Clear

### Reference Documents
- `.planning/codebase/STRUCTURE.md` — Frontend directory layout, component organization
- `.planning/codebase/CONVENTIONS.md` — React patterns, CSS conventions, component naming
</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- **`TabBar` component** (`frontend/src/components/TabBar.tsx`): Supports tab list with names, close buttons, drag-and-drop reorder, and active tab highlighting. Can be extended with status indicators and right-click context menu. Props: `tabs: Tab[]`, `activeTabId`, callbacks for `onSelectTab`, `onCloseTab`, `onReorderTabs`.
- **`TerminalComponent`** (`frontend/src/components/Terminal.tsx`): Already accepts `sessionId` prop, subscribes to namespaced `pty-output:{id}` events, dispatches `Write(sessionId)/Resize(sessionId)/Clear(sessionId)/Start(sessionId)`. Ready for multi-instance mounting.
- **`useKeyboardShortcuts`** (`frontend/src/hooks/useKeyboardShortcuts.ts`): Global shortcut registration with `cmdOrCtrl` helper. Add Ctrl+T, Ctrl+W, Ctrl+Tab, Ctrl+Shift+Tab handlers.
- **shadcn/ui components** (`frontend/src/components/ui/`): ContextMenu, Dialog, Button, tooltip available for right-click menu and rename dialog.

### Established Patterns
- **CSS variable theming:** All colors use `var(--background)`, `var(--foreground)`, `var(--primary)`, etc. No hardcoded colors. Tab bar must follow this pattern.
- **Event subscriptions:** `Events.On(eventName, handler)` with cleanup returned. Per-session events use `'pty-output:' + sessionId` format.
- **Ref-based state:** Terminal output and pane state use refs (not React state) to avoid re-render loops.

### Integration Points
- **App.tsx:** TerminalComponent is mounted once with `activeSessionId` state. Must change to mount N TerminalComponents (one per session), showing only the active one. `activeSessionId` is set from `SetActiveSession()` call.
- **TabBar:** Currently used for command editor tabs. Terminal tab bar can be a separate instance with terminal-specific behavior or the same component extended.
- **Wails bindings:** All session management methods are already generated in `frontend/bindings/cmdex/terminalservice.js`. No regeneration needed for this phase.
</code_context>

<specifics>
## Specific Ideas

User wants standard terminal tab UX: '+' button for new tabs, close buttons per tab, right-click context menu for rename/close, keyboard shortcuts matching common terminal apps (Ctrl+T, Ctrl+W, Ctrl+Tab). No persistence across restarts (Phase 22 skipped).
</specifics>

<deferred>
## Deferred Ideas

- **SESS-05 (close with confirmation):** Close confirmation when a process is running was not discussed. Default: close without confirmation (backend handles PTY cleanup). Can be added later.
- **UI-02 (duplicate):** Duplicate action in context menu was rejected by user. Not needed for this phase.
- **Session persistence (PERS-01 through PERS-04):** All deferred per Phase 22 skip decision.
</deferred>

---

*Phase: 23-Frontend Tabbed Terminal*
*Context gathered: 2026-06-10*
