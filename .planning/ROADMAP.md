# Roadmap: Cmdex v2.0 Terminal Integration

**Milestone:** v2.0
**Name:** Terminal Integration
**Goal:** Replace the static output pane with an xterm.js-based PTY terminal with full ANSI support, interactive input, and freeform command typing in a split-pane layout.

---

## Phase 16: PTY Backend Foundation

**Goal:** A persistent shell process runs in a Go-managed PTY with bidirectional I/O, resize support, and clean lifecycle management. The frontend can send keystrokes and receive output via Wails events.

**Requirements:** PTY-01, PTY-02, PTY-03, PTY-04, PTY-05, PTY-06, POL-05

**Success criteria:**
1. Shell spawns in PTY on app startup (bash on macOS/Linux, powershell/cmd on Windows)
2. Keystrokes sent from Go to PTY stdin produce correct shell echo in PTY output
3. PTY output streams via Wails events with 16ms batching and 64KB max chunks per emit
4. Calling `pty.Setsize` with new cols/rows changes shell dimensions (verify via `tput cols; tput lines`)
5. App shutdown kills the shell process group — no orphaned bash/cmd processes
6. Shell exit (user types `exit` or receives Ctrl+D) is detected and reported
7. Shell auto-detection correctly picks bash on macOS/Linux and powershell/cmd on Windows

<details>
<summary>Plans (3 plans)</summary>

- [x] 16-01-PLAN.md — TerminalService struct, service registration, event constants, Unix PTY lifecycle (creack/pty), Windows stubs, shell detection (Wave 1)
- [x] 16-02-PLAN.md — PTY output streaming with 16ms batching, 64KB chunks, pty-output/pty-exit events, shell exit detection, auto-restart (Wave 2)
- [x] 16-03-PLAN.md — Test suite for all 7 requirements, Windows go-winpty integration, package legitimacy gate (Wave 2)

</details>

---

## Phase 17: xterm.js Terminal and Split Pane Layout

**Goal:** The current OutputPane is replaced by an xterm.js Terminal component in a split pane layout. The terminal renders PTY output, auto-fits to container size, and stays mounted across tab switches.

**Requirements:** TERM-01, TERM-02, TERM-03, TERM-04, LAY-01, LAY-02, LAY-04

**Success criteria:**
1. xterm.js terminal renders in the bottom pane with ANSI color support (verify via `ls --color`)
2. FitAddon resizes terminal when window or divider is resized — no scrollbar gaps
3. WebglAddon activates on supported hardware, falls back gracefully to canvas renderer
4. URLs in terminal output are underlined and clickable (WebLinksAddon)
5. Layout is a vertical split: command editor on top, terminal on bottom, with drag-to-resize divider
6. Switching command tabs does not unmount the Terminal component (CSS display toggle, not React unmount)
7. The old OutputPane toggle behavior is fully removed from App.tsx

<details>
<summary>Plans (TBD by /gsd-plan-phase)</summary>

- 17-01: Create Terminal.tsx component with xterm.js, FitAddon, WebglAddon, WebLinksAddon
- 17-02: Replace OutputPane with split pane layout in App.tsx (editor + terminal)
- 17-03: Wire terminal to PTY backend via Wails bindings and events

</details>

---

## Phase 18: Execution Integration and Interactivity

**Goal:** Clicking Run writes the resolved command to the terminal. Users can type commands freely. Ctrl+C interrupts running processes. Working directory is respected.

**Requirements:** EXEC-01, EXEC-02, EXEC-03, EXEC-04, LAY-03

**Success criteria:**
1. Clicking Run on a saved command writes resolved command text + newline to PTY stdin via TerminalService.Write
2. Command output appears in the terminal with full ANSI rendering (replaces static OutputPane)
3. User can type any custom command directly in the terminal and execute it freely
4. Ctrl+C sends SIGINT and interrupts the foreground process (verify with `sleep 30` then Ctrl+C)
5. Shell starts in or changes to the command's resolved working directory when a command is loaded
6. Clear button resets the terminal scrollback buffer

<details>
<summary>Plans (TBD by /gsd-plan-phase)</summary>

- 18-01: Replace RunCommand flow — write resolved command to TerminalService.Write instead of old executor
- 18-02: Implement Ctrl+C handling in terminal (SIGINT to PTY process group)
- 18-03: Integrate working directory resolution with terminal shell CWD

</details>

---

## Phase 19: Terminal Polish

**Goal:** Terminal theme syncs with Cmdex themes, font matches app font, copy/paste works per-platform, search is available, and the terminal feels native.

**Requirements:** POL-01, POL-02, POL-03, POL-04

**Success criteria:**
1. Switching Cmdex themes updates xterm terminal theme in real time with no flicker
2. Terminal font family updates when user changes font in Settings
3. Cmd+C / Ctrl+Shift+C copies selected text from terminal
4. Cmd+V / Ctrl+Shift+V pastes clipboard text into terminal
5. Ctrl+F opens search in terminal scrollback buffer (SearchAddon)

<details>
<summary>Plans (TBD by /gsd-plan-phase)</summary>

- 19-01: Theme sync — derive xterm ITheme from Cmdex CSS variables, apply on theme change
- 19-02: Font, copy/paste, and SearchAddon integration

</details>

---

## Progress

| Phase | Milestone | Requirements | Status |
|-------|-----------|-------------|--------|
| 1-5 | v1.0 Premium Polish | — | Shipped 2026-04-13 |
| 6-7 | v1.1 Build Settings Window | — | Shipped |
| 8-9 | v1.2 DB Migration Refactor | — | Shipped |
| 10-13 | v1.3 Working Directory | 14 | Shipped 2026-04-23 |
| 14 | v1.4 Editor Multi-Mount Refactor | — | Shipped 2026-04-23 |
| 15 | v1.5 Cross-Platform Execution | — | Shipped 2026-05-04 |
| 16 | 3/3 | Complete   | 2026-05-19 |
| 17 | v2.0 Terminal Integration | 7 | Pending |
| 18 | v2.0 Terminal Integration | 6 | Pending |
| 19 | v2.0 Terminal Integration | 4 | Pending |

**Execution order:** 16 -> 17 -> 18 -> 19 (serial — each phase depends on the prior)

---

*Last updated: 2026-05-18*
