---
phase: 23-frontend-tabbed-terminal
plan: 01
subsystem: ui
tags: [react, typescript, dnd-kit, shadcn, context-menu, terminal, sessions]

# Dependency graph
requires: []
provides:
  - SessionInfo TypeScript interface (id, name, running, shellPath, workingDir)
  - TerminalTabBar React component with DnD reorder, context menu, status dots, '+' button
  - CSS classes for tab status dots, drag handle, and '+' button
affects:
  - 23-02 (App.tsx integration — mounts TerminalTabBar above terminal pane)

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "DnD with PointerSensor activationConstraint distance:5 to prevent click/drag interference"
    - "Separate drag handle (GripVertical) inside sortable tab — attributes/listeners on handle, not tab div"
    - "Status dot uses CSS modifiers (.running / .stopped) with var() color references"
    - "Context menu via shadcn ContextMenu/ContextMenuTrigger/ContextMenuContent/ContextMenuItem"

key-files:
  created:
    - frontend/src/components/TerminalTabBar.tsx
  modified:
    - frontend/src/types.ts
    - frontend/src/style.css

key-decisions:
  - "Separate drag handle with 5px activation distance prevents DnD/click interference"
  - "Close button hidden (not disabled) on last tab; context menu Close item disabled on last tab"
  - "Window.prompt() for rename — simple, no modal component needed for this scope"
  - "All CSS colors use var() references — no hardcoded hex values"

patterns-established:
  - "TerminalTabBar is standalone — does not extend/wrap existing TabBar component"
  - "SortableTerminalTab is an internal sub-component (not exported)"

requirements-completed: [SESS-02, SESS-06, UI-01, UI-02, UI-05]

# Metrics
duration: 5min
completed: 2026-06-10
---

# Phase 23 Plan 01: Terminal Tab Bar Component Summary

**TerminalTabBar component with drag-and-drop reorder via @dnd-kit, right-click context menu via shadcn, green/gray status indicator dots, and '+' new-session button — all theme-driven via CSS custom properties**

## Performance

- **Duration:** 5 min
- **Started:** 2026-06-10T10:44:07Z
- **Completed:** 2026-06-10T10:49:11Z
- **Tasks:** 3
- **Files modified:** 3

## Accomplishments

- Added `SessionInfo` TypeScript interface to `types.ts` with all 5 Go-backed fields
- Created 5 CSS classes (`.tab-status-dot`, `.tab-status-dot.running`, `.tab-status-dot.stopped`, `.tab-drag-handle`, `.tab-new-session-btn`) using CSS custom properties exclusively
- Built `TerminalTabBar` component (196 lines) with DnD sortable tabs, context menu, status indicators, close buttons with last-tab protection, '+' button, and auto-scroll-to-active

## Task Commits

Each task was committed atomically:

1. **Task 1: Add SessionInfo TypeScript interface to types.ts** - `9238804` (feat)
2. **Task 2: Add Terminal Tab Bar CSS classes to style.css** - `328318c` (feat)
3. **Task 3: Create TerminalTabBar.tsx component** - `bcc3fb5` (feat)

## Files Created/Modified

- `frontend/src/types.ts` - Added `SessionInfo` interface with id, name, running, shellPath, workingDir fields
- `frontend/src/style.css` - Added `.tab-status-dot`, `.tab-status-dot.running`, `.tab-status-dot.stopped`, `.tab-drag-handle`, `.tab-new-session-btn` classes (47 lines)
- `frontend/src/components/TerminalTabBar.tsx` - New 196-line component with DnD, context menu, status dots, '+' button

## Decisions Made

- Drag handle uses dedicated `<span>` with all dnd-kit `attributes`/`listeners` — tab click-to-select is separate from drag, preventing DnD/click interference (RESEARCH.md Pitfall 2)
- Close button hidden (not rendered) on last tab — context menu Close item `disabled={isLastTab}` for same protection
- Rename uses `window.prompt()` — avoids modal complexity for this scope; validates non-empty trimmed value before calling `onRenameSession`
- All colors reference CSS variables — no hardcoded hex values in any new CSS rule blocks

## Deviations from Plan

### Acceptance Criteria Tolerance

**1. Minor criteria mismatch — `tab-status-dot` grep count**
- **Found during:** Task 3 verification
- **Issue:** Plan acceptance criteria expected `grep -c "tab-status-dot"` to return ≥2, expecting separate occurrences for className usage and running/stopped conditional. Implementation places both on a single line: `` className={`tab-status-dot ${session.running ? 'running' : 'stopped'}`} ``, yielding grep count of 1.
- **Fix:** None needed — functional intent fully satisfied (both base class and conditional present). Noted as plan criteria overspecification.
- **Verification:** Visual review of line 85 confirms both class name and conditional.

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Fixed `onDragEnd` event type incompatibility**
- **Found during:** Task 3 (TypeScript check)
- **Issue:** Inline event type `{ active: { id: string }; over: { id: string } | null }` incompatible with dnd-kit's `DragEndEvent` which uses `UniqueIdentifier` (string | number) for `active.id`
- **Fix:** Imported `DragEndEvent` from `@dnd-kit/core` and cast `active.id`/`over.id` with `String()` in `findIndex` calls
- **Files modified:** frontend/src/components/TerminalTabBar.tsx
- **Committed in:** bcc3fb5 (Task 3)

---

**Total deviations:** 2 (1 blocking auto-fix, 1 minor criteria mismatch)
**Impact on plan:** Blocking fix essential for TypeScript compilation. Criteria mismatch is cosmetic — functional intent fully met.

## Issues Encountered

None — all three tasks executed cleanly. Single TypeScript type error caught and fixed in Task 3 verification cycle.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

Ready for Plan 02 (App.tsx integration):
- `SessionInfo` type and `TerminalTabBar` component are in place
- CSS classes are defined and use-theme-agnostic (all `var()` references)
- Component accepts the full props interface App.tsx will wire up (sessions, activeSessionId, onSelectTab, onCloseTab, onReorderTabs, onCreateSession, onRenameSession)
- No blockers — all TypeScript checks pass

## Threat Surface

No new trust boundaries introduced. Threat T-23-01 (rename prompt validation) mitigated — `window.prompt()` result validated for non-empty-trimmed before calling `onRenameSession`. Threats T-23-02 and T-23-03 accepted per plan.

## Self-Check: PASSED

- [x] `frontend/src/types.ts` exists with `export interface SessionInfo` (1 match)
- [x] `frontend/src/components/TerminalTabBar.tsx` exists with `export default function TerminalTabBar` (1 match)
- [x] `frontend/src/style.css` contains `.tab-status-dot` classes (3 matches)
- [x] `cd frontend && pnpm tsc --noEmit` exits 0
- [x] All 3 commits verified in git log

---

*Phase: 23-frontend-tabbed-terminal*
*Completed: 2026-06-10*
