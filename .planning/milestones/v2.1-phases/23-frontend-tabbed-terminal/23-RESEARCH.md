# Phase 23: Frontend Tabbed Terminal - Research

**Researched:** 2026-06-10
**Domain:** React tabbed UI, drag-and-drop, keyboard shortcuts, xterm.js multi-instance
**Confidence:** HIGH

## Summary

Phase 23 builds the frontend tabbed UI on top of the Phase 21 backend session infrastructure. The backend already provides `CreateSession`, `ListSessions`, `CloseSession`, `RenameSession`, `SetActiveSession`, `GetActiveSession`, and namespaced events (`pty-output:{id}`, `pty-exit:{id}`, `pty-cleared:{id}`). The existing `TerminalComponent` already accepts a `sessionId` prop and subscribes to namespaced events.

This phase is purely frontend: no Go backend changes, no new Wails bindings, no package installation. All required libraries are already installed. The work consists of (1) creating a `TerminalTabBar` component with drag-and-drop reorder and right-click context menu, (2) mounting one `TerminalComponent` per session (hidden via `display: none` for inactive), and (3) wiring keyboard shortcuts for terminal tab navigation.

**Primary recommendation:** Extend the existing `TabBar` component pattern (not its code — it lacks reorder support) into a new `TerminalTabBar` component that uses `@dnd-kit/sortable` for drag-and-drop and the existing shadcn `ContextMenu` for right-click actions. Mount all sessions' `TerminalComponent` instances using the already-proven per-tab mount + `display: none` pattern from Phase 14 (Editor Multi-Mount Refactor).

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** '+' button at end of tab bar always visible, calls `CreateSession()`. Also Ctrl+T keyboard shortcut.
- **D-02:** Close via per-tab close button (x), Ctrl+W shortcut, or right-click context menu. Last remaining tab is not closeable (close button hidden/grayed, Ctrl+W no-op on last tab).
- **D-03:** Right-click context menu on a tab shows two actions: Rename (opens dialog/inline edit, calls `RenameSession`) and Close (grayed out for last tab). No Duplicate option.
- **D-04:** Click a tab to switch active session (calls `SetActiveSession`). Frontend caches `activeSessionId` locally for immediate UI, syncs with backend.
- **D-05:** Keyboard shortcuts: Ctrl+Tab (next tab), Ctrl+Shift+Tab (previous tab). Wrap around at ends.

### the agent's Discretion
- **Tab bar implementation:** Leverage existing `TabBar` component (`frontend/src/components/TabBar.tsx`) with terminal-specific extensions (status indicator, right-click menu) rather than building a completely new component. The existing TabBar already handles reorder (drag-and-drop) and close.
- **Terminal rendering:** Mount one `TerminalComponent` per session, hide inactive via CSS (`display: none`). This preserves independent scrollback per session naturally — each xterm.js instance maintains its own buffer. Phase 21 already has `TerminalComponent` accepting `sessionId` prop and subscribing to namespaced events.
- **Tab reorder:** Drag-and-drop via existing TabBar mechanism. Reorder is visual-only (frontend state) since Phase 22 persistence was skipped.
- **Status indicators:** Each tab shows session name + running status (green dot = shell alive, gray dot = stopped). Status derived from `SessionInfo.running` field. Update on `pty-exit:{sessionId}` and `pty-output:{sessionId}` events.
- **Right-click menu:** Use Radix ContextMenu (already in shadcn/ui) or a custom dropdown positioned at cursor. Simple implementation with two items (Rename, Close).
- **Scrollback:** 5000 lines per session as configured in `Terminal.tsx` (`scrollback: 5000`). Each mounted TerminalComponent preserves its own scrollback when hidden.
- **Tab overflow:** When many tabs exist, tab bar scrolls horizontally (overflow-x: auto with hidden scrollbar, or arrow buttons). Follow existing TabBar overflow behavior.
- **Clear button:** Existing clear button in app clears only the active session's terminal (already works since `Clear(sessionId)` is called with the active session ID).

### Deferred Ideas (OUT OF SCOPE)
- SESS-05 (close with confirmation): Close without confirmation (backend handles PTY cleanup).
- UI-02 (duplicate): Duplicate action in context menu was rejected by user.
- Session persistence (PERS-01 through PERS-04): All deferred per Phase 22 skip decision.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SESS-02 | User can see a list of all terminal sessions in a tab bar | `ListSessions()` returns `SessionInfo[]` with `id`, `name`, `running`. New `TerminalTabBar` component maps array to tab items. |
| SESS-03 | User can switch between terminal sessions by clicking tabs | Click handler calls `SetActiveSession(id)`, updates `activeSessionId` state, TerminalComponent visibility toggles via `display: none` pattern. |
| SESS-06 | User can reorder terminal session tabs via drag-and-drop | `@dnd-kit/sortable` v10.0.0 with `DndContext` + `SortableContext` + `useSortable` + `horizontalListSortingStrategy`. Visual-only reorder (no backend persistence). |
| UI-01 | Terminal tabs show session name and status indicator (idle/running/busy) | `SessionInfo.name` for tab label, `SessionInfo.running` for green/gray dot indicator. Update on `pty-output:{id}` and `pty-exit:{id}` events. |
| UI-02 | Right-click tab shows context menu (rename, close) | Existing shadcn `ContextMenu` component (`frontend/src/components/ui/context-menu.tsx`) wraps each tab. Two items: Rename (opens inline edit), Close (disabled for last tab). |
| UI-03 | Keyboard shortcuts: New tab (Ctrl+T), Close tab (Ctrl+W), Next tab (Ctrl+Tab), Prev tab (Ctrl+Shift+Tab) | Existing `useKeyboardShortcuts` hook. **Conflict**: These shortcuts are already bound for command tabs. Plan must resolve focus-dependent dispatch. |
| UI-04 | Terminal output preserves scrollback per session (5000 lines) | Already configured: `scrollback: 5000` in `Terminal.tsx` xterm.js constructor. Each TerminalComponent instance maintains own buffer; hidden instances retain buffer. |
| UI-05 | Session theme matches app theme (CSS variables) | Already implemented: `Terminal.tsx` reads `--background`, `--foreground`, `--primary` CSS variables in theme effect. Tab bar uses existing `--tab-bar-bg`, `--tab-active-bg` etc. |
| UI-06 | Clear terminal button clears only the active session | Already works: `terminalRef.current?.clear()` calls `Clear(sessionId)` on the active session's TerminalComponent. With multi-instance, need per-session ref map. |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Session lifecycle (create/close/rename) | API / Backend | — | Backend `TerminalService` owns session state. Frontend calls Go methods via Wails bindings. |
| Tab bar UI rendering | Browser / Client | — | Pure React rendering; tab bar is a visual-only component with no backend ownership. |
| Tab reorder (drag-and-drop) | Browser / Client | — | Reorder is visual-only frontend state since Phase 22 persistence was skipped. |
| Right-click context menu | Browser / Client | — | Radix ContextMenu renders a portal; purely client-side interaction. |
| Terminal output rendering | Browser / Client | — | xterm.js renders PTY output received via Wails events. Backend streams data, frontend writes to terminal. |
| Session scrollback | Browser / Client | — | xterm.js buffer (5000 lines) is client-side only. Hidden instances retain buffer. |
| Active session tracking | API / Backend | Browser / Client | Backend is source of truth via `SetActiveSession`/`GetActiveSession`. Frontend caches for immediate UI. |
| Keyboard shortcuts | Browser / Client | — | `useKeyboardShortcuts` hook handles global keydown events. |
| Status indicators (running/stopped) | Browser / Client | — | Derived from `SessionInfo.running` field + `pty-exit:{id}` events (both delivered by backend). |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| @dnd-kit/core | ^6.3.1 | Drag-and-drop engine (DndContext, sensors) | Already installed; industry-standard React DnD. Used for tab reorder. [VERIFIED: package.json] |
| @dnd-kit/sortable | ^10.0.0 | Sortable list primitives (SortableContext, useSortable, arrayMove) | Already installed; companion to @dnd-kit/core. Horizontal sortable for tab bar. [VERIFIED: package.json] |
| radix-ui (ContextMenu) | ^1.4.3 | Right-click context menu primitive (accessible, WAI-ARIA) | Already installed; wraps each tab for rename/close actions. Shadcn wrapper exists at `ui/context-menu.tsx`. [VERIFIED: package.json + file existence] |
| @xterm/xterm | ^6.0.0 | Terminal emulator (per-session instance) | Already installed; each TerminalComponent creates its own xterm.js instance with independent scrollback. [VERIFIED: package.json] |
| @wailsio/runtime | 3.0.0-alpha.79 | Wails event system (Events.On for namespaced pty events) | Already installed; used to subscribe to `pty-output:{id}`, `pty-exit:{id}`. [VERIFIED: package.json] |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| shadcn/ui Dialog | — | Rename session modal/dialog | When user selects "Rename" from context menu |
| lucide-react | ^0.576.0 | Icons (Plus, X, Terminal, etc.) | Tab bar buttons (add, close) and context menu icons |
| @dnd-kit/utilities | ^3.2.2 | CSS transform utilities for DnD | Drag overlay positioning |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| @dnd-kit/sortable v10 (already installed) | Custom drag-and-drop with HTML5 Drag API | Custom DnD is fragile, lacks accessibility, and requires complex touch support. @dnd-kit handles pointer, keyboard, and touch sensors out of the box. |
| shadcn ContextMenu (already installed) | Custom positioned dropdown | Custom dropdown requires manual positioning, focus management, and accessibility. Radix ContextMenu handles all of this. |

**Installation:** No new packages required. All dependencies are already installed via `pnpm`.

**Version verification:**
```bash
# All confirmed present in node_modules:
ls frontend/node_modules/@dnd-kit/sortable/package.json  # v10.0.0
ls frontend/node_modules/@dnd-kit/core/package.json      # v6.3.1
ls frontend/node_modules/radix-ui/package.json            # v1.4.3
ls frontend/node_modules/@xterm/xterm/package.json        # v6.0.0
```

## Package Legitimacy Audit

> No new packages are installed in this phase. All libraries are pre-existing in `package.json` and verified in `node_modules`.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| @dnd-kit/sortable | npm | 4+ yrs | 2M+/wk | github.com/clauderic/dnd-kit | [OK] — Pre-installed | Already in node_modules |
| @dnd-kit/core | npm | 4+ yrs | 2M+/wk | github.com/clauderic/dnd-kit | [OK] — Pre-installed | Already in node_modules |
| radix-ui | npm | 3+ yrs | 3M+/wk | github.com/radix-ui/primitives | [OK] — Pre-installed | Already in node_modules |
| @xterm/xterm | npm | 7+ yrs | 1M+/wk | github.com/xtermjs/xterm.js | [OK] — Pre-installed | Already in node_modules |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                        App.tsx                               │
│  ┌──────────┐  ┌──────────────────┐  ┌───────────────────┐ │
│  │ Sidebar  │  │  Center Area     │  │  Terminal Pane    │ │
│  │ (cmds)   │  │  ┌────────────┐  │  │  ┌─────────────┐  │ │
│  │          │  │  │  TabBar     │  │  │  │TerminalTab  │  │ │
│  │          │  │  │  (editors)  │  │  │  │  Bar        │  │ │
│  │          │  │  └────────────┘  │  │  │  (NEW)       │  │ │
│  │          │  │  ┌────────────┐  │  │  └─────────────┘  │ │
│  │          │  │  │Per-tab     │  │  │  ┌─────────────┐  │ │
│  │          │  │  │Command     │  │  │  │TermComp[id1]│  │ │
│  │          │  │  │DetailTabs  │  │  │  │(display:none)│  │ │
│  │          │  │  └────────────┘  │  │  ├─────────────┤  │ │
│  └──────────┘  └──────────────────┘  │  │TermComp[id2]│  │ │
│                                       │  │(active,      │  │ │
│                                       │  │ flex:1)      │  │ │
│                                       │  └─────────────┘  │ │
│                                       └───────────────────┘ │
└─────────────────────────────────────────────────────────────┘
         │                                          │
         ▼                                          ▼
┌─────────────────┐                    ┌────────────────────────┐
│  Wails Bindings │                    │  Wails Events          │
│  (Go backend)   │                    │  (@wailsio/runtime)    │
│                 │                    │                        │
│  ListSessions() │◄── polls ─────────│  Events.On(            │
│  CreateSession()│                    │    'pty-output:{id}',  │
│  CloseSession() │                    │    handler             │
│  RenameSession()│                    │  )                     │
│  SetActiveSess()│                    │                        │
│  GetActiveSess()│                    │  Events.On(            │
│  Clear()        │                    │    'pty-exit:{id}',    │
│  Write()        │                    │    handler             │
│  Resize()       │                    │  )                     │
└─────────────────┘                    └────────────────────────┘
```

**Data flow:**
1. User clicks '+' → `CreateSession()` → backend returns `SessionInfo` → added to `sessions` state → new tab appears
2. User clicks tab → `SetActiveSession(id)` → backend switches active → `activeSessionId` updates → inactive TerminalComponent hides, active TerminalComponent shows
3. PTY output: backend emits `pty-output:{id}` → `TerminalComponent` for that sessionId receives event → writes to xterm.js buffer
4. User drags tab → `onDragEnd` → `arrayMove` on `sessions` array → re-rendered tab order (visual-only)

### Recommended Project Structure

```
frontend/src/
├── components/
│   ├── TerminalTabBar.tsx     # NEW — terminal session tab bar with DnD + context menu
│   ├── TabBar.tsx             # EXISTING — command editor tab bar (unchanged)
│   ├── Terminal.tsx           # EXISTING — TerminalComponent (minor: expose per-session refs)
│   └── ui/
│       └── context-menu.tsx   # EXISTING — shadcn ContextMenu wrapper
├── App.tsx                    # MODIFY — add sessions state, multi-TerminalComponent mount,
│                              #          terminal keyboard shortcuts, terminalRefs map
├── hooks/
│   └── useKeyboardShortcuts.ts # MODIFY — add terminal-specific shortcut handlers
├── types.ts                   # MODIFY — add SessionInfo type mirror (if not already present)
└── style.css                  # MODIFY — terminal tab bar CSS (using existing --tab-* variables)
```

### Pattern 1: Per-Instance Mount with display:none (Phase 14 pattern)

**What:** Mount one component per data entity, hide inactive via CSS, preserving DOM state.

**When to use:** When each instance has expensive internal state (xterm.js buffer, scroll position) that must survive tab switches.

**Example (from existing App.tsx command tabs, adapted for terminals):**
```tsx
// Source: Phase 14 Editor Multi-Mount Refactor pattern, adapted for terminal sessions
{sessions.map((session) => (
  <TerminalComponent
    key={session.id}
    ref={(el) => { terminalRefs.current[session.id] = el; }}
    isVisible={session.id === activeSessionId && !terminalCollapsed}
    theme={theme}
    sessionId={session.id}
    onShellExit={() => {
      // Mark session as stopped
      setSessions(prev => prev.map(s =>
        s.id === session.id ? { ...s, running: false } : s
      ));
    }}
  />
))}
```

### Pattern 2: @dnd-kit Horizontal Sortable List

**What:** Drag-and-drop tab reorder using @dnd-kit/sortable v10 API.

**When to use:** For any horizontal list where users need to reorder items by dragging.

**Example:**
```tsx
// Source: @dnd-kit/sortable v10 API [VERIFIED: node_modules + Context7 docs]
import {
  DndContext,
  closestCenter,
  PointerSensor,
  useSensor,
  useSensors,
} from '@dnd-kit/core';
import {
  arrayMove,
  SortableContext,
  useSortable,
  horizontalListSortingStrategy,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';

function SortableTab({ id, tab }: { id: string; tab: SessionInfo }) {
  const { attributes, listeners, setNodeRef, transform, transition } =
    useSortable({ id });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  return (
    <div ref={setNodeRef} style={style} {...attributes} {...listeners}>
      {/* tab content */}
    </div>
  );
}

function TerminalTabBar({ sessions, activeSessionId, onReorder }) {
  const sensors = useSensors(useSensor(PointerSensor));

  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCenter}
      onDragEnd={({ active, over }) => {
        if (over && active.id !== over.id) {
          const oldIndex = sessions.findIndex(s => s.id === active.id);
          const newIndex = sessions.findIndex(s => s.id === over.id);
          onReorder(arrayMove(sessions, oldIndex, newIndex));
        }
      }}
    >
      <SortableContext
        items={sessions.map(s => s.id)}
        strategy={horizontalListSortingStrategy}
      >
        {sessions.map(session => (
          <SortableTab key={session.id} id={session.id} tab={session} />
        ))}
      </SortableContext>
    </DndContext>
  );
}
```

### Pattern 3: ContextMenu on Each Tab

**What:** Right-click each tab to show rename/close options using existing shadcn ContextMenu.

**When to use:** When tab actions (rename, close) need a secondary interaction surface beyond click and close button.

**Example:**
```tsx
// Source: shadcn/ui context-menu.tsx already in project [VERIFIED: file existence]
import {
  ContextMenu,
  ContextMenuTrigger,
  ContextMenuContent,
  ContextMenuItem,
} from '@/components/ui/context-menu';

<ContextMenu>
  <ContextMenuTrigger asChild>
    <div className="tab-item">
      {/* tab content */}
    </div>
  </ContextMenuTrigger>
  <ContextMenuContent>
    <ContextMenuItem onSelect={() => handleRename(session.id)}>
      Rename
    </ContextMenuItem>
    <ContextMenuItem
      disabled={sessions.length <= 1}
      onSelect={() => handleClose(session.id)}
    >
      Close
    </ContextMenuItem>
  </ContextMenuContent>
</ContextMenu>
```

### Anti-Patterns to Avoid

- **Single ref for multiple terminals:** Using a single `terminalRef` for all terminals loses per-session access. Use `terminalRefs: Record<string, TerminalHandle>` map instead. [CITED: App.tsx currently uses single `terminalRef`]
- **Unmounting inactive terminals:** Removing TerminalComponent from DOM on tab switch destroys xterm.js buffer, losing scrollback. Always mount all and hide via `display: none`. [CITED: Phase 14 pattern]
- **Rebuilding TabBar from scratch:** The existing `TabBar` has established CSS classes (`.tab-bar`, `.tab-item`, `.tab-title`, `.tab-close`) and theme variables. New `TerminalTabBar` should reuse these CSS classes for visual consistency.
- **Hardcoding keyboard shortcuts for terminal tabs without focus detection:** Ctrl+T currently opens a new command tab. The terminal shortcuts must either use focus-dependent dispatch or adjust the existing shortcut registration to distinguish terminal pane focus from editor focus. [CITED: App.tsx line 1154 vs CONTEXT.md D-01]

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Drag-and-drop tab reorder | Custom HTML5 drag API with mouse events | @dnd-kit/sortable (already installed) | Handles touch, keyboard, pointer sensors; accessibility (screen reader announcements); collision detection; smooth animations. Custom DnD is ~200 lines of fragile code. |
| Right-click context menu positioning | Manual absolute positioning with onContextMenu + state | shadcn ContextMenu (Radix) | Handles viewport collision detection, portal rendering, keyboard navigation, focus management, WAI-ARIA compliance. Custom positioning breaks in edge cases (near viewport edges). |
| Inline rename input | Build custom input with blur/Enter handling | Controlled `<input>` with auto-focus + onBlur/onKeyDown | Already standard pattern; no library needed here. Keep it simple. |

**Key insight:** The three non-trivial interaction patterns (drag-and-drop, context menu, keyboard shortcuts) all have existing solutions in the project's dependency tree. The only custom code needed is state management wiring (`sessions` array, `activeSessionId`, `terminalRefs` map) and the `TerminalTabBar` component composition — not interaction primitives.

## Common Pitfalls

### Pitfall 1: Keyboard Shortcut Conflicts Between Command Tabs and Terminal Tabs

**What goes wrong:** Ctrl+T, Ctrl+W, Ctrl+Tab, Ctrl+Shift+Tab are already bound in `useKeyboardShortcuts` for command tab operations. Adding terminal tab shortcuts on the same keys without scope detection causes both handlers to fire, creating broken behavior (e.g., pressing Ctrl+T opens a command tab AND creates a terminal session).

**Why it happens:** `useKeyboardShortcuts` registers global `keydown` listeners on `window`. There's no context to distinguish "user is interacting with terminal" from "user is in editor area."

**How to avoid:**
1. **Focus-based dispatch:** Check `document.activeElement` — if it's inside the `.terminal-pane` or a `.xterm-helper-textarea`, dispatch to terminal shortcuts; otherwise dispatch to command tab shortcuts.
2. **Or: different shortcuts for terminal tabs:** Use Ctrl+Shift+T (new terminal), Ctrl+Shift+W (close terminal) to avoid conflict entirely. The CONTEXT.md specifies Ctrl+T/Ctrl+W for terminal tabs, but the existing bindings for command tabs must be considered.

**Warning signs:** Pressing Ctrl+T opens both a new command tab AND a new terminal session.

### Pitfall 2: DnD Drag Handle Interference with Click-to-Select

**What goes wrong:** Using the entire tab as the drag handle makes it impossible to click-to-select without accidentally initiating a drag. Users get frustrated when clicking a tab to switch sessions triggers a drag.

**Why it happens:** `useSortable`'s `listeners` are spread on the same element as the `onClick` handler. Pointer down triggers both drag initiation and click.

**How to avoid:** Use a dedicated drag handle (grip icon or small area on the left edge of each tab) instead of making the entire tab draggable. Or use `activationConstraint: { distance: 5 }` in PointerSensor config so that small pointer movements (clicks) don't trigger drag.

**Warning signs:** Clicking a tab to switch sessions occasionally reorders the tab instead.

### Pitfall 3: TerminalComponent Key Prop Causes Remount on Reorder

**What goes wrong:** Using array index as `key` on TerminalComponent causes React to remount terminals when tabs are reordered, destroying scrollback buffers.

**Why it happens:** React uses `key` to identify instances across renders. Index-based keys change when array order changes, causing React to unmount the old instance and mount a new one.

**How to avoid:** Always use `session.id` (UUID v4) as the `key` prop on each TerminalComponent. UUIDs are stable across reorders.

**Warning signs:** Terminal scrollback resets after dragging a tab to a new position.

### Pitfall 4: Stale SessionInfo After Backend State Changes

**What goes wrong:** A session is closed by the backend (shell crashes, auto-restart fails), but the frontend `sessions` array still shows the session with `running: true`.

**Why it happens:** `pty-exit:{sessionId}` event fires with exit code, but the frontend may not update the `sessions` array's `running` field. `ListSessions()` is only called on mount.

**How to avoid:** Subscribe to `pty-exit:{sessionId}` and `pty-output:{sessionId}` events to update individual session status. Call `ListSessions()` on mount and after `CreateSession()`/`CloseSession()`. Keep `sessions` state in sync with backend via event handlers.

**Warning signs:** Tab shows green dot (running) but the shell process died minutes ago.

### Pitfall 5: ContextMenu and DnD Event Conflict

**What goes wrong:** Right-clicking a tab for context menu also triggers drag-and-drop initiation, or drag-and-drop prevents context menu from opening.

**Why it happens:** Both `useSortable` listeners and `ContextMenu` listen for pointer events on the same element. Without proper event isolation, they interfere.

**How to avoid:** The `ContextMenuTrigger` should wrap the tab element, and DnD listeners should be on a separate drag handle element (not the entire tab). If the entire tab must be draggable, ensure the `ContextMenu` uses `onContextMenu` (right-click only) while DnD uses left-click/pointer-down.

**Warning signs:** Right-clicking a tab sometimes drags it instead of opening context menu.

## Code Examples

Verified patterns from the codebase and official sources:

### Session State Management in App.tsx

```tsx
// Source: Pattern from existing App.tsx state management [VERIFIED: codebase]
// Adapted for multi-session terminal support

const [sessions, setSessions] = useState<SessionInfo[]>([]);
const [activeSessionId, setActiveSessionId] = useState<string>('');
const terminalRefs = useRef<Record<string, TerminalHandle>>({});

// Load sessions on mount
useEffect(() => {
  ListSessions().then(list => {
    const sessions = (list || []).filter(Boolean) as SessionInfo[];
    setSessions(sessions);
    GetActiveSession().then(info => {
      if (info) setActiveSessionId(info.id);
    }).catch(() => {});
  }).catch(() => {});
}, []);

// Create new session
const createTerminalSession = useCallback(async () => {
  const info = await CreateSession();
  if (info) {
    setSessions(prev => [...prev, info]);
    setActiveSessionId(info.id);
    SetActiveSession(info.id);
  }
}, []);

// Close session
const closeTerminalSession = useCallback(async (id: string) => {
  if (sessions.length <= 1) return; // Don't close last tab
  await CloseSession(id);
  setSessions(prev => prev.filter(s => s.id !== id));
  if (activeSessionId === id) {
    const remaining = sessions.filter(s => s.id !== id);
    const next = remaining[0];
    if (next) {
      setActiveSessionId(next.id);
      SetActiveSession(next.id);
    }
  }
}, [sessions, activeSessionId]);

// Rename session
const renameTerminalSession = useCallback(async (id: string, name: string) => {
  await RenameSession(id, name);
  setSessions(prev => prev.map(s => s.id === id ? { ...s, name } : s));
}, []);

// Switch session (click tab)
const switchTerminalSession = useCallback((id: string) => {
  setActiveSessionId(id);
  SetActiveSession(id);
}, []);

// Clear active terminal
const clearActiveTerminal = useCallback(() => {
  terminalRefs.current[activeSessionId]?.clear();
}, [activeSessionId]);
```

### Horizontal DnD Tab Bar with Context Menu

```tsx
// Source: @dnd-kit/sortable v10 + shadcn ContextMenu [VERIFIED: package versions]
// Key pattern: drag handle is separate from click-to-select target
// ContextMenu wraps the tab for right-click actions

function SortableTerminalTab({
  session,
  isActive,
  isLast,
  onSelect,
  onClose,
  onRename,
}: {
  session: SessionInfo;
  isActive: boolean;
  isLast: boolean;
  onSelect: () => void;
  onClose: () => void;
  onRename: (name: string) => void;
}) {
  const { attributes, listeners, setNodeRef, transform, transition } =
    useSortable({ id: session.id });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  return (
    <div ref={setNodeRef} style={style}>
      <ContextMenu>
        <ContextMenuTrigger asChild>
          <div
            className={`tab-item${isActive ? ' active' : ''}`}
            onClick={onSelect}
          >
            {/* Drag handle — only this element has DnD listeners */}
            <span className="tab-drag-handle" {...attributes} {...listeners}>
              <GripVertical size={12} />
            </span>
            {/* Status indicator */}
            <span className={`tab-status-dot ${session.running ? 'running' : 'stopped'}`} />
            {/* Tab title */}
            <span className="tab-title" title={session.name}>{session.name}</span>
            {/* Close button — hidden on last tab */}
            {!isLast && (
              <span className="tab-close" role="button" onClick={(e) => { e.stopPropagation(); onClose(); }}>
                <X size={12} />
              </span>
            )}
          </div>
        </ContextMenuTrigger>
        <ContextMenuContent>
          <ContextMenuItem onSelect={() => {
            // Open inline edit or dialog for rename
            const name = prompt('Rename session:', session.name);
            if (name?.trim()) onRename(name.trim());
          }}>
            Rename
          </ContextMenuItem>
          <ContextMenuItem disabled={isLast} onSelect={onClose}>
            Close
          </ContextMenuItem>
        </ContextMenuContent>
      </ContextMenu>
    </div>
  );
}
```

### Multi-TerminalComponent Mounting

```tsx
// Source: Phase 14 Editor Multi-Mount pattern [VERIFIED: App.tsx lines 1309-1355]
// Adapted for TerminalComponent

<div className="terminal-pane" style={{ height: terminalHeight }}>
  {sessions.map((session) => (
    <TerminalComponent
      key={session.id}  // Always use session.id, never index!
      ref={(el) => {
        if (el) terminalRefs.current[session.id] = el;
        else delete terminalRefs.current[session.id];
      }}
      isVisible={session.id === activeSessionId && !terminalCollapsed}
      theme={theme}
      sessionId={session.id}
      onShellExit={() => {
        setSessions(prev => prev.map(s =>
          s.id === session.id ? { ...s, running: false } : s
        ));
      }}
    />
  ))}
</div>
```

### Status Indicator Styling (CSS Variables)

```css
/* Source: Existing .tab-dirty-dot pattern [VERIFIED: style.css lines 627-638] */
.tab-status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  flex-shrink: 0;
  margin-right: 4px;
}

.tab-status-dot.running {
  background: var(--success, #4ec9b0);
}

.tab-status-dot.stopped {
  background: var(--muted-foreground, #858585);
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Single global terminal instance | Multi-session with per-session xterm.js instances | Phase 21 | Each session has independent PTY, event namespace, and scrollback |
| Single `terminalRef` for one TerminalComponent | `terminalRefs: Record<string, TerminalHandle>` map | Phase 23 (this phase) | Per-session access for clear, copy, etc. |
| Command tabs only (TabBar for editors) | Separate `TerminalTabBar` for session tabs + existing TabBar for editors | Phase 23 (this phase) | Two independent tab bars with different behaviors and shortcuts |
| @dnd-kit unused in TabBar | @dnd-kit/sortable for TerminalTabBar reorder | Phase 23 (this phase) | Drag-and-drop tab reorder with accessibility |

**Deprecated/outdated:** None. All patterns are additive.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The existing TabBar component does NOT support drag-and-drop reorder (contrary to CONTEXT.md's statement that it "already handles reorder") | Architecture Patterns | LOW — TabBar.tsx code clearly shows no DnD logic. The plan will build reorder from scratch using @dnd-kit regardless. |
| A2 | Keyboard shortcut conflict between command tab shortcuts (Ctrl+T/Ctrl+W/Ctrl+Tab) and terminal tab shortcuts can be resolved via focus-based dispatch | Common Pitfalls | MEDIUM — If focus detection proves unreliable (xterm.js textarea focus edge cases), the plan may need to use different shortcuts for terminal tabs. |
| A3 | `SessionInfo` type from Go backend (with `id`, `name`, `running`, `shellPath`, `workingDir`) is already generated in `frontend/bindings/cmdex/models.js` and usable as-is | Standard Stack | LOW — Confirmed by reading models.js lines 497-553, which show the SessionInfo class with all expected fields. |
| A4 | `@dnd-kit/sortable` v10 uses v1 API (`DndContext`, `SortableContext`, `useSortable`, `arrayMove`) not v2 API (`DragDropProvider`) | Architecture Patterns | LOW — Confirmed by reading node_modules version (10.0.0) and package.json. The v2 API is a separate package (`@dnd-kit/react`). |

## Open Questions (RESOLVED)

1. **Keyboard shortcut resolution for Ctrl+T / Ctrl+W / Ctrl+Tab / Ctrl+Shift+Tab**
   - What we know: These shortcuts are already bound for command tabs in App.tsx (lines 1154, 1162-1164, 1169, 1176). CONTEXT.md specifies them for terminal tabs.
   - What's unclear: Whether the user expects focus-dependent dispatch (in terminal pane → terminal action, in editor → command action) or separate shortcuts.
   - Recommendation: Implement focus-dependent dispatch. Check if `document.activeElement` is inside `.terminal-pane` or `.xterm-helper-textarea`. If the xterm.js textarea focus detection proves unreliable (it hides/shows the textarea dynamically), fall back to checking if the click target was inside the terminal pane area.

2. **Rename UX: inline edit vs. dialog**
   - What we know: CONTEXT.md says "opens dialog/inline edit." Shadcn Dialog is available.
   - What's unclear: Which UX pattern the user prefers for rename.
   - Recommendation: Use inline editing (replacing the tab title with an `<input>` on double-click or from context menu "Rename"). This matches terminal emulators like iTerm2 and Windows Terminal. Fall back to Dialog if inline edit is complex.

3. **Default session on first launch**
   - What we know: Phase 21 creates one default session on startup. Phase 23 starts with the backend's active session.
   - What's unclear: Should Phase 23 create an additional session if none exist, or trust the backend?
   - Recommendation: Trust the backend. Call `GetActiveSession()` on mount. If it returns null, call `CreateSession()` to create a default session (ensures user always sees at least one terminal tab).

## Environment Availability

> No external dependencies beyond the project's existing toolchain.

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js | Frontend build/dev | ✓ | — (via project) | — |
| pnpm | Package management | ✓ | — (via project) | — |
| Wails v3 | Backend bindings | ✓ | 3.0.0-alpha.79 | — |
| @dnd-kit/sortable | Tab drag-and-drop | ✓ (pre-installed) | 10.0.0 | — |
| @dnd-kit/core | DnD engine | ✓ (pre-installed) | 6.3.1 | — |
| radix-ui | ContextMenu | ✓ (pre-installed) | 1.4.3 | — |

**Missing dependencies with no fallback:** none
**Missing dependencies with fallback:** none

## Validation Architecture

> `workflow.nyquist_validation` is absent from config.json — treat as enabled.

### Test Framework
| Property | Value |
|----------|-------|
| Framework | None detected for frontend |
| Config file | none — see Wave 0 |
| Quick run command | `cd frontend && pnpm tsc --noEmit` |
| Full suite command | `cd frontend && pnpm tsc --noEmit` (no test runner configured) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| SESS-02 | Tab bar lists all sessions | manual | N/A — visual verification | ❌ Wave 0 |
| SESS-03 | Click tab switches active session | manual | N/A — requires Wails backend | ❌ Wave 0 |
| SESS-06 | Drag-and-drop reorders tabs | manual | N/A — visual + interaction | ❌ Wave 0 |
| UI-01 | Tab shows name + status dot | manual | N/A — visual verification | ❌ Wave 0 |
| UI-02 | Right-click shows rename/close menu | manual | N/A — visual verification | ❌ Wave 0 |
| UI-03 | Keyboard shortcuts fire correctly | manual | N/A — requires global shortcuts | ❌ Wave 0 |
| UI-04 | Scrollback preserved per session | manual | N/A — requires running shell | ❌ Wave 0 |
| UI-05 | Theme CSS variables respected | manual | N/A — visual verification | ❌ Wave 0 |
| UI-06 | Clear clears only active session | manual | N/A — requires running shell | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `cd frontend && pnpm tsc --noEmit`
- **Per wave merge:** `cd frontend && pnpm tsc --noEmit` + manual verification
- **Phase gate:** Manual verification of all 9 success criteria

### Wave 0 Gaps
- [ ] No test framework configured for frontend (Playwright installed but no test files in `frontend/`)
- [ ] No test file for `TerminalTabBar` component
- [ ] No test file for terminal keyboard shortcut dispatch logic
- [ ] Framework install: `cd frontend && pnpm tsc --noEmit` is the only automated check

*(All gaps noted — frontend testing infrastructure does not exist in this project per `frontend/package.json` and project conventions)*

## Security Domain

> `security_enforcement` is absent from config — treat as enabled.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | N/A — desktop app, no user auth |
| V3 Session Management | no | N/A — not web sessions |
| V4 Access Control | no | N/A — single-user desktop app |
| V5 Input Validation | yes — low risk | Session rename input: validate non-empty, strip control characters. Not a web attack surface. |
| V6 Cryptography | no | N/A — no crypto operations in this phase |

### Known Threat Patterns for React + Wails Desktop

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| XSS via session name in tab title | Information Disclosure | React auto-escapes JSX content. Session names come from backend (user's own input in a desktop app). No special escaping needed. |
| Event listener leaks (memory) | Denial of Service | Clean up `Events.On()` subscriptions in useEffect return. Already standard pattern in Terminal.tsx. |

**Assessment:** This phase has minimal security surface. Input (session rename) is user's own input in a local desktop app. No authentication, no network-bound data, no external content rendering.

## Sources

### Primary (HIGH confidence)
- [Context7: /clauderic/dnd-kit] — @dnd-kit/sortable v10 API reference, useSortable, SortableContext, arrayMove, horizontalListSortingStrategy
- [Context7: /websites/radix-ui_primitives] — Radix ContextMenu.Root, Trigger, Content, Item API with Portal rendering
- [codebase: frontend/src/components/TabBar.tsx] — Existing tab bar structure, CSS classes, props interface
- [codebase: frontend/src/components/Terminal.tsx] — TerminalComponent with sessionId prop, event subscriptions, scrollback config
- [codebase: frontend/src/App.tsx] — State management patterns, per-tab mount pattern, keyboard shortcut registration
- [codebase: frontend/src/hooks/useKeyboardShortcuts.ts] — Shortcut registration and key building logic
- [codebase: frontend/src/style.css] — CSS variables for theming, tab bar styles, terminal pane styles
- [codebase: frontend/bindings/cmdex/terminalservice.js] — Wails-generated bindings: CreateSession, ListSessions, CloseSession, RenameSession, SetActiveSession, GetActiveSession
- [codebase: frontend/bindings/cmdex/models.js] — SessionInfo class: id, name, running, shellPath, workingDir fields

### Secondary (MEDIUM confidence)
- [Context7: /llmstxt/ui_shadcn_llms_txt] — shadcn/ui component patterns (ContextMenu wrapper already exists in project)
- [Context7: /shadcn-ui/ui] — shadcn Dialog component for rename modal (alternative to inline edit)

### Tertiary (LOW confidence)
- None — all claims are verified against codebase or official documentation.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — all libraries are pre-installed, versions verified against package.json and node_modules
- Architecture: HIGH — patterns derived from existing codebase (Phase 14 multi-mount, App.tsx state management, Terminal.tsx event subscriptions)
- Pitfalls: HIGH — keyboard shortcut conflict, DnD click interference, and key-prop remount are documented patterns in React + dnd-kit ecosystems
- Package legitimacy: HIGH — no new packages; all pre-installed packages are well-established (>3 years, >1M weekly downloads)

**Research date:** 2026-06-10
**Valid until:** 2026-07-10 (30 days — the stack is stable, no breaking changes expected)
