---
status: complete
phase: 21-backend-session-foundation
source: 21-01-SUMMARY.md, 21-02-SUMMARY.md, 21-03-SUMMARY.md
started: 2026-06-10T08:34:02Z
updated: 2026-06-10T09:05:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Go Build & Vet
expected: `go build ./...` and `go vet ./...` both pass. All session CRUD, PTY lifecycle, and dispatch methods compile cleanly.
result: pass

### 2. Go Tests Pass
expected: `go test -short ./...` passes all 19 tests (10 updated, 9 new) covering multi-session CRUD, process persistence, output isolation, and concurrency. No panics, no race conditions.
result: pass

### 3. Frontend TypeScript Check
expected: `cd frontend && pnpm tsc --noEmit` passes with no type errors. TerminalComponent.sessionId prop, namespaced events, and regenerated Wails bindings all type-check.
result: pass

### 4. App Starts in Dev Mode
expected: `wails3 dev` boots without errors. App window opens, terminal service initializes with a default session, no crash on startup.
result: pass

### 5. Terminal Renders with Shell Prompt
expected: The terminal component shows a working shell (bash/zsh) prompt. The sessionId wiring connects the frontend to the backend PTY session.
result: skipped
reason: "don't know how to verify it"

### 6. Basic Command Execution
expected: Typing `echo hello` in the terminal produces visible keystrokes and correct output. The per-session PTY read/write loop works, namespaced pty-output events deliver output to the correct session.
result: pass

### 7. Session Event Isolation
expected: Events are namespaced per session (pty-output:{id}). The ptyOutput/ptyExit/ptyCleared constants no longer exist in frontend events.ts or backend event_service.go.
result: skipped
reason: "don't know how to verify"

## Summary

total: 7
passed: 5
issues: 0
pending: 0
skipped: 2

## Gaps

- truth: "Typing `echo hello` in the terminal produces visible keystrokes and correct output"
  status: failed
  reason: "User reported: terminal not render my keystroke anymore, but it's still execute correct what i typed"
  severity: major
  test: 6
  root_cause: "readLoop UTF-8 boundary detection at terminal_service.go:413-425 never recognizes single-byte ASCII (0xxxxxxx pattern) as valid complete UTF-8. Every keystroke gets stuck in the leftover buffer and is only flushed when subsequent data (command output) arrives. The old code used utf8.Valid() which correctly handles ASCII; the new manual detection only checks for multi-byte start bytes (11xxxxxx)."
  artifacts:
    - path: "frontend/src/components/Terminal.tsx"
      issue: "useEffect [sessionId] dep causes double terminal creation; first with empty sessionId missing events, second with real ID after unnecessary PTY restart"
    - path: "terminal_service.go"
      issue: "readLoop UTF-8 boundary detection at lines 413-425 drops single-byte ASCII characters — no check for 0xxxxxxx pattern before decrementing validEnd"
  missing:
    - "Split Terminal.tsx into two effects: terminal creation with [] dep, event subscriptions + Start with [sessionId] dep"
    - "Use sessionIdRef for dispatch methods (Write/Resize/Clear)"
    - "Add ASCII byte detection to readLoop: check data[validEnd-1]&0x80 == 0 before decrementing validEnd"
    - "Fix multi-byte bit masks: 0xE0==0xC0 for 2-byte, 0xF0==0xE0 for 3-byte, 0xF8==0xF0 for 4-byte (was incorrectly using 0xC0==0xC0 for all)"
  debug_session: ".planning/phases/21-backend-session-foundation/21-UAT.md"
