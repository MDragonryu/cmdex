# Phase 25 Runbook — Manual Verification

This runbook covers manual verification procedures that complement the automated
Go stress test in `terminal_service_stress_test.go`. Phase 25 is a polish
phase — automated tests cover the backend session lifecycle, while this runbook
captures the visual / DOM-level checks that are best performed by a human on a
running build.

## Manual xterm leak smoke test

Per **D-09** in the Phase 25 CONTEXT, this smoke test verifies that opening
many terminal tabs and closing them does not leave orphaned xterm DOM nodes.
The Go stress test (`TestTerminalService_StressCreateClose`) covers the
backend lifecycle (PTY file descriptors, monitorExit, emitter goroutines);
this manual test covers the xterm.js DOM side.

### Prerequisites

- `wails3 dev` running (or a built `cmdex.app` launched from Finder).
- The app is on the home screen with one default terminal tab visible.
- Chrome DevTools access (Help → Open Dev Tools, or right-click → Inspect).

### Procedure

1. Open the Cmdex app and confirm one default terminal tab is visible.
2. Press **Ctrl+T** twenty times to create twenty additional terminal tabs.
3. Open DevTools (Help → Open Dev Tools, or right-click anywhere → Inspect).
4. In the DevTools Console, run:
   ```js
   document.querySelectorAll('.xterm').length
   ```
   Record this value as **count-A** (expected ≈ 21: the original tab plus
   twenty new tabs).
5. Press **Ctrl+W** twenty times to close the new tabs. The last tab stays
   open by design (last-tab close guard in `TabBar.tsx`).
6. In the DevTools Console, run the same query:
   ```js
   document.querySelectorAll('.xterm').length
   ```
   Record this value as **count-B**.
7. Verify `count-B <= 1` and `count-A ≈ 21`. If the numbers match, the
   smoke test passes.

### Pass criterion

`count-B <= 1` — after closing all but the protected last tab, at most
one xterm node remains in the DOM. (`count-A` should be approximately
21; a small variance is acceptable as long as it is in the right
ballpark.)

### Failure interpretation

If `count-B > 1`, orphaned xterm DOM nodes are present after tab close.
Each orphaned node still owns a `Terminal` instance and its event
listeners, which is a memory leak.

The fix is in the xterm cleanup useEffect of
`frontend/src/components/Terminal.tsx` (around lines 270–274) where
`term.dispose()` is called. Confirm:

- The cleanup function calls `term.dispose()`.
- `term.dispose()` runs on unmount (when the tab is removed from
  `App.tsx` state, not just hidden via `display: none`).
- No other code path holds a reference to the disposed terminal.

If `count-B` is exactly 1, the smoke test passes — the last tab
stays mounted by design (last-tab close guard).

### Notes

- This is a manual smoke test, not a CI assertion. Phase 25 explicitly
  deferred adding frontend test infrastructure (Playwright, etc.) per
  the phase's deferred-items list.
- The Go stress test (`TestTerminalService_StressCreateClose`) is the
  automated counterpart for the backend lifecycle. Both should pass
  before declaring Phase 25 done.
- `count-A ≈ 21` is approximate because the test does not enforce an
  exact count — what matters is that opening 20 tabs produces roughly
  20 xterm nodes, and closing 20 tabs returns the DOM to ≈ 1 node.
- If you observe a leak but only in the *hidden* tabs (display:none
  xterm nodes), that is expected during the session — xterm instances
  are kept alive across tab switches. The leak is only when tabs are
  *removed* from the tab list.
