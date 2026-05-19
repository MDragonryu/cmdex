---
phase: 16-pty-backend-foundation
plan: 02
subsystem: terminal
tags: [pty, event-streaming, goroutine, batching]

requires:
  - phase: 16-01
    provides: TerminalService struct, ptyStart/ptmx/cmd, stopCh, eventNames.PtyOutput/PtyExit

provides:
  - readLoop goroutine with 16ms batched PTY output via bufio reader
  - emitOutput method emitting pty-output Wails events with {data: string} payload
  - monitorExit goroutine detecting shell exit, emitting pty-exit events, auto-restart
  - Graceful drain of buffered output on stopCh signal and read error
  - Auto-restart suppression during app shutdown via stopCh check

affects:
  - 16-03 (test coverage for readLoop batching and monitorExit flow)

tech-stack:
  added: []
  patterns:
    - "Goroutine lifecycle: readLoop + monitorExit launched in Start(), signaled via stopCh closure in Stop()"
    - "16ms ticker batching with bytes.Buffer accumulation and 64KB hard-cap immediate emit"

key-files:
  modified:
    - terminal_service.go - Added emitOutput, readLoop, monitorExit; goroutine launch in Start()

key-decisions:
  - "16ms ticker (time.NewTicker) chosen for batching interval per PTY-03"
  - "64KB bufio.NewReaderSize and bytes.Buffer.Grow cap — prevents event bridge saturation"
  - "50ms sleep after close(stopCh) allows readLoop to drain buffered output before PTY close"
  - "100ms restart delay prevents tight restart loops on immediate shell crash"
  - "Exit code 0 treated as intentional (user typed exit/Ctrl+D), non-zero as crash"

requirements-completed: [PTY-03, PTY-06]

duration: 10min
completed: 2026-05-19
---

# Phase 16 Plan 02: Output Streaming & Exit Detection Summary

**16ms batched PTY output reader and shell exit monitor with auto-restart via Wails event bridge**

## Performance

- **Duration:** 10 min
- **Started:** 2026-05-19T04:40:00Z
- **Completed:** 2026-05-19T04:50:00Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments
- readLoop goroutine drains PTY output with 16ms batching and 64KB max chunks
- emitOutput bridges PTY data to frontend via `pty-output` Wails events
- monitorExit detects shell termination, emits `pty-exit` with exit code and intent flag
- Shell auto-restarts after exit with last known terminal size, suppressed during app shutdown

## Task Commits

1. **Task 1: Add readLoop goroutine with 16ms batching and emitOutput** - `feat(16-02): add readLoop goroutine with 16ms batching and emitOutput`
2. **Task 2: Add monitorExit goroutine with auto-restart and exit event** - `feat(16-02): add monitorExit goroutine with auto-restart and pty-exit event`

## Files Modified
- `terminal_service.go` - Added emitOutput, readLoop, monitorExit methods; goroutine launch in Start()

## Decisions Made
- 16ms ticker interval matches PTY-03 spec and prevents event bridge flooding (~62 max events/sec)
- 64KB reader and buffer cap provides hard limit on event size
- 50ms drain window after stopCh to capture trailing PTY output before kill
- stopCh check in monitorExit prevents shell restart during app termination (T-16-07 mitigation)

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## Next Phase Readiness
- TerminalService is now fully bidirectional: keystrokes go in via Write(), output comes out via readLoop, shell lifecycle handled by monitorExit
- Ready for Plan 16-03 (test suite and Windows PTY integration)

---
*Phase: 16-pty-backend-foundation*
*Completed: 2026-05-19*
