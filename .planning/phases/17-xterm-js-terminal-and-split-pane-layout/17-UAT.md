---
status: diagnosed
phase: 17-xterm-js-terminal-and-split-pane-layout
source: 17-01-SUMMARY.md, 17-02-SUMMARY.md, 17-03-SUMMARY.md
started: 2026-05-19T12:00:00Z
updated: 2026-05-19T12:00:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Terminal Split Pane Layout
expected: Terminal pane visible below editor, separated by horizontal divider with ns-resize cursor on hover
result: pass
note: Terminal keyboard input is Phase 18 scope, not Phase 17

### 2. Terminal Visual Appearance
expected: Dark background (#1e1e1e), light text (#d4d4d4), block cursor blinking, scrollbar hidden
result: pass

### 3. Resizable Divider
expected: Dragging the divider resizes the terminal. Terminal height respects min (100px) and max (85% viewport) bounds. Editor height adjusts proportionally.
result: pass

### 4. Collapse/Expand Terminal
expected: Collapse button (on divider or rail) hides the terminal pane. A thin collapsed rail appears with an expand button. Clicking expand restores the terminal. Collapse state persists across page reload.
result: pass

### 5. PTY Output in Terminal
expected: Running a command (any saved command) shows its output in the xterm.js terminal pane instead of the old output pane
result: issue
reported: "not see anything in terminal pane"
severity: major

### 6. Terminal Persists Across Tab Switches
expected: Switching between multiple command tabs does not destroy or remount the terminal. PTY output continues to flow regardless of active tab.
result: pass

## Summary

total: 6
passed: 5
issues: 1
pending: 0
skipped: 0
blocked: 0

## Gaps

- truth: "Running a command via 'Run' button displays output in the xterm.js terminal pane"
  status: diagnosed
  reason: "User reported: not see anything in terminal pane"
  severity: major
  test: 5
  root_cause: "Event stream mismatch: RunCommand emits cmd-output events → streamLines state, but OutputPane (only consumer) was removed in 17-02. xterm.js terminal only subscribes to pty-output events for the interactive PTY shell. streamLines is still populated but has no render target."
  closure_plan: 17-04-PLAN.md
  artifacts:
    - path: "frontend/src/App.tsx"
      issue: "streamLines state populated by cmd-output events but not rendered anywhere after OutputPane removal"
    - path: "frontend/src/components/Terminal.tsx"
      issue: "Only subscribes to pty-output events, not cmd-output — RunCommand output never reaches xterm.js"
  missing:
    - "Route RunCommand output to Terminal.tsx (either via TerminalService PTY or additional cmd-output event subscription)"
  debug_session: ""
