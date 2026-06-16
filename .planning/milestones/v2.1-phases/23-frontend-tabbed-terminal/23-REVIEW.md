---
phase: 23-frontend-tabbed-terminal
reviewed: 2026-06-10T12:00:00Z
depth: standard
files_reviewed: 5
files_reviewed_list:
  - frontend/src/types.ts
  - frontend/src/components/TerminalTabBar.tsx
  - frontend/src/style.css
  - frontend/src/App.tsx
  - frontend/src/hooks/useKeyboardShortcuts.ts
findings:
  critical: 1
  warning: 3
  info: 3
  total: 7
status: issues_found
---

# Phase 23: Code Review Report

**Reviewed:** 2026-06-10T12:00:00Z
**Depth:** standard
**Files Reviewed:** 5
**Status:** issues_found

## Summary

Reviewed 5 source files changed during Phase 23 (frontend-tabbed-terminal). The phase adds multi-session terminal support with a DnD-reorderable tab bar, context menus, per-session terminal instances, and focus-dependent keyboard shortcuts. The implementation is thorough with good CSS variable usage and proper dnd-kit patterns (5px activation distance, separate drag handle).

**Key concern:** The Ctrl+W/Meta+W keyboard shortcut unconditionally swallows the key event when focus is in the xterm.js hidden textarea, preventing standard shell word-deletion (Ctrl+W). This destroys the active terminal session on a commonly-used shell shortcut.

1 critical issue (BLOCKER), 3 warnings, and 3 informational items were found.

---

## Critical Issues

### BL-01: Ctrl+W/Meta+W keyboard shortcuts destroy terminal sessions and swallow shell key events

**File:** `frontend/src/App.tsx:1276-1294`
**Also affected:** `frontend/src/hooks/useKeyboardShortcuts.ts:42-46`

**Issue:** When the user is actively typing in the terminal (focus is in xterm.js's hidden `.xterm-helper-textarea`), pressing Ctrl+W closes the active terminal session instead of allowing the shell to receive the key event for word deletion (standard shell behavior: delete previous word). The problem is a confluence of two design decisions:

1. `isFocusInTerminalPane()` (line 250-257) correctly includes `.xterm-helper-textarea` in its focus detection — this is correct for operations like Ctrl+T (new session) and Ctrl+Tab (cycle sessions).

2. `useKeyboardShortcuts` (line 42-46) **unconditionally** calls `e.preventDefault()` and `e.stopPropagation()` for ALL matched modifier shortcuts, **before** the handler even runs. This means:
   - The keydown event never reaches xterm.js or the shell
   - Even when the handler is a no-op (e.g., `sessions.length <= 1` guard on line 1279), the event is still consumed
   - Standard shell shortcuts like Ctrl+W are permanently disabled in the terminal

**User impact:** A developer typing a command in the terminal presses Ctrl+W to delete the last word (muscle memory from bash/zsh). Instead, the entire terminal session is closed and cannot be recovered. On the last remaining session, Ctrl+W is swallowed as a no-op (the close is gated), but the shell still never receives it for word deletion.

**Fix:** Two-part fix needed:

**Part A — Prevent default only when action is taken (useKeyboardShortcuts.ts):**

```typescript
// In useKeyboardShortcuts.ts, line 42-46, change from:
const hasModifier = e.metaKey || e.ctrlKey || e.altKey;
if (hasModifier || !isEditing) {
    e.preventDefault();
    e.stopPropagation();
    fn(e);
}

// To — allow the handler to signal whether it consumed the event:
const hasModifier = e.metaKey || e.ctrlKey || e.altKey;
if (hasModifier || !isEditing) {
    let consumed = true;
    fn(e, () => { consumed = false; });
    if (consumed) {
        e.preventDefault();
        e.stopPropagation();
    }
}
```

**Part B — Adjust Ctrl+W/Meta+W handler to not intercept when typing in terminal (App.tsx):**

```typescript
// For Ctrl+W/Meta+W, exclude .xterm-helper-textarea from focus detection
// so the shell receives the key event for word deletion.
const isFocusInTerminalChrome = useCallback((): boolean => {
    const el = document.activeElement;
    if (!el) return false;
    return !!((el as HTMLElement).closest?.('.terminal-pane') &&
              !(el as HTMLElement).closest?.('.xterm-helper-textarea'));
}, []);

// Use isFocusInTerminalChrome for Ctrl+W/Meta+W only
'ctrl+w': () => {
    if (isFocusInTerminalChrome()) {
        if (sessions.length > 1 && activeSessionId) {
            closeTerminalSession(activeSessionId);
        }
    } else {
        if (activeTabId) closeTab(activeTabId);
    }
},
```

**Alternative simpler fix:** Change Ctrl+W to only close when focus is in the tab bar chrome (`.tab-bar`), not the xterm textarea. Keep `isFocusInTerminalPane` for Ctrl+T and Ctrl+Tab which intentionally fire during terminal input.

---

## Warnings

### WR-01: `closeTerminalSession` computes `remaining` from stale closure `sessions` after `await`

**File:** `frontend/src/App.tsx:206-223`

**Issue:** After `await CloseSession(id)` resolves, the `remaining` computation on line 213 uses `sessions` from the closure, not from the latest state. Since `useCallback` has `[sessions, activeSessionId]` as deps, the callback is recreated when sessions change. However, if another state mutation occurs between the callback creation and the `await` resolution (e.g., a rapid second close), the `sessions` value could be stale.

```typescript
// Line 212-213 — sessions is from closure, not latest state:
if (activeSessionId === id) {
    const remaining = sessions.filter(s => s.id !== id);  // stale after await
```

**Fix:** Compute `remaining` from a snapshot taken before the `await`, or use the functional updater pattern:

```typescript
const closeTerminalSession = useCallback(async (id: string) => {
    if (sessions.length <= 1) return;
    // Snapshot before async operation
    const preCloseSessions = sessions;
    const wasActive = activeSessionId === id;
    try {
        await CloseSession(id);
        setSessions(prev => prev.filter(s => s.id !== id));
        if (wasActive) {
            const remaining = preCloseSessions.filter(s => s.id !== id);
            const next = remaining[0];
            if (next) {
                setActiveSessionId(next.id);
                SetActiveSession(next.id);
            }
        }
    } catch (err) {
        console.error('Failed to close terminal session:', err);
    }
}, [sessions, activeSessionId]);
```

### WR-02: `CreateSession` failure in mount-time `useEffect` has no user feedback

**File:** `frontend/src/App.tsx:323-329`

**Issue:** When the mount-time session loading detects zero sessions, it calls `CreateSession()` to create a default session. If this fails, the error is silently swallowed by `.catch(() => {})` (line 329). The user gets no toast, no error message, and the terminal pane remains in an unusable state (no sessions, no way to create one except the '+' button — which may also fail for the same reason).

```typescript
// Lines 323-329
CreateSession().then((newInfo) => {
    if (newInfo) {
        setSessions([newInfo]);
        setActiveSessionId(newInfo.id);
        SetActiveSession(newInfo.id);
    }
}).catch(() => {});  // <-- silent failure
```

**Fix:** Add a toast error on failure:

```typescript
CreateSession().then((newInfo) => {
    if (newInfo) {
        setSessions([newInfo]);
        setActiveSessionId(newInfo.id);
        SetActiveSession(newInfo.id);
    }
}).catch((err) => {
    console.error('Failed to create default session:', err);
    toast.error('Could not create terminal session. Check that the terminal backend is running.');
});
```

### WR-03: Terminal session tab order is not persisted through reorder

**File:** `frontend/src/App.tsx:244-246`

**Issue:** The DnD reorder handler `handleReorderTerminalTabs` updates React state but makes no backend call to persist the new order. When the app restarts or sessions are reloaded (e.g., via `ListSessions`), the original backend ordering is restored, losing any user-arranged tab order.

```typescript
const handleReorderTerminalTabs = useCallback((reordered: SessionInfo[]) => {
    setSessions(reordered);
    // No persistence call to backend
}, []);
```

**Fix:** If session ordering matters, add a backend call to persist the order. If ordering is ephemeral (acceptable to lose on restart), add a code comment documenting this decision so future maintainers know it's intentional:

```typescript
const handleReorderTerminalTabs = useCallback((reordered: SessionInfo[]) => {
    setSessions(reordered);
    // Note: Session order is ephemeral — resets to backend order on reload.
    // Backend does not support session ordering (sessions sorted by creation time).
}, []);
```

---

## Info

### IN-01: Misleading `isLastTab` variable name in TerminalTabBar

**File:** `frontend/src/components/TerminalTabBar.tsx:156`

**Issue:** The variable `isLastTab` is computed as `sessions.length <= 1` (line 156) and passed to every `SortableTerminalTab` instance. The name suggests "this specific tab is the last one," but it actually means "tabs cannot be closed at all." For a 3-tab setup, all three tabs receive `isLastTab = false`; for a 1-tab setup, the single tab receives `isLastTab = true`. The actual per-tab "is this the last tab?" check is performed in `closeTerminalSession` in App.tsx (which checks `sessions.length <= 1`).

**Fix:** Rename to something more descriptive:

```typescript
const isCloseDisabled = sessions.length <= 1;
```

### IN-02: `String()` type cast in `handleDragEnd` — fragile if dnd-kit changes ID types

**File:** `frontend/src/components/TerminalTabBar.tsx:146-147`

**Issue:** The drag handler casts `active.id` and `over.id` from dnd-kit's `UniqueIdentifier` (string | number) to string using `String()`. While this works with the current dnd-kit API and UUID session IDs, it relies on implicit type coercion. If dnd-kit ever changes `UniqueIdentifier` to include other types (symbols, objects), the `String()` conversion would produce garbage string values like `"Symbol(id)"` or `"[object Object]"` that would silently fail to match any session.

**Fix:** Add a type guard:

```typescript
const activeId = typeof active.id === 'string' ? active.id : String(active.id);
const overId = typeof over.id === 'string' ? over.id : String(over.id);
const oldIndex = sessions.findIndex((s) => s.id === activeId);
const newIndex = sessions.findIndex((s) => s.id === overId);
```

### IN-03: Unused `React` import in TerminalTabBar (React 19 JSX transform)

**File:** `frontend/src/components/TerminalTabBar.tsx:1`

**Issue:** The file imports `React` at line 1 (`import React, { useRef, useEffect } from 'react'`), but with React 19 and the automatic JSX runtime (configured in `tsconfig.json` as `"jsx": "react-jsx"`), `React` does not need to be in scope for JSX compilation. The named imports `useRef` and `useEffect` are used, but the default `React` import is unused.

**Fix:** Remove the unused default import:

```typescript
import { useRef, useEffect } from 'react';
```

---

_Reviewed: 2026-06-10T12:00:00Z_
_Reviewer: the agent (gsd-code-reviewer)_
_Depth: standard_
