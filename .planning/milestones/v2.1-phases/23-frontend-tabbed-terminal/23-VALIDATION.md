---
phase: 23
slug: frontend-tabbed-terminal
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-06-10
---

# Phase 23 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | None detected for frontend |
| **Config file** | none — see Wave 0 |
| **Quick run command** | `cd frontend && pnpm tsc --noEmit` |
| **Full suite command** | `cd frontend && pnpm tsc --noEmit` |
| **Estimated runtime** | ~5 seconds |

---

## Sampling Rate

- **After every task commit:** Run `cd frontend && pnpm tsc --noEmit`
- **After every plan wave:** Run `cd frontend && pnpm tsc --noEmit` + manual verification
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 23-{plan}-{task} | TBD | TBD | SESS-02 | — | N/A | manual | — | ❌ W0 | ⬜ pending |
| 23-{plan}-{task} | TBD | TBD | SESS-03 | — | N/A | manual | — | ❌ W0 | ⬜ pending |
| 23-{plan}-{task} | TBD | TBD | SESS-06 | — | N/A | manual | — | ❌ W0 | ⬜ pending |
| 23-{plan}-{task} | TBD | TBD | UI-01 | — | N/A | manual | — | ❌ W0 | ⬜ pending |
| 23-{plan}-{task} | TBD | TBD | UI-02 | — | N/A | manual | — | ❌ W0 | ⬜ pending |
| 23-{plan}-{task} | TBD | TBD | UI-03 | — | N/A | manual | — | ❌ W0 | ⬜ pending |
| 23-{plan}-{task} | TBD | TBD | UI-04 | — | N/A | manual | — | ❌ W0 | ⬜ pending |
| 23-{plan}-{task} | TBD | TBD | UI-05 | — | N/A | manual | — | ❌ W0 | ⬜ pending |
| 23-{plan}-{task} | TBD | TBD | UI-06 | — | N/A | manual | — | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

---

## Wave 0 Requirements

- [ ] No test framework configured for frontend (Playwright installed but no test files in `frontend/`)
- [ ] No test file for `TerminalTabBar` component
- [ ] No test file for terminal keyboard shortcut dispatch logic
- [ ] Framework install: `cd frontend && pnpm tsc --noEmit` is the only automated check

*All gaps noted — frontend testing infrastructure does not exist in this project per `frontend/package.json` and project conventions.*

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Tab bar lists all sessions | SESS-02 | Requires Wails backend + visual verification | Verify tab bar shows all sessions from ListSessions() |
| Click tab switches active session | SESS-03 | Requires Wails backend | Click tab, verify terminal output updates to that session |
| Drag-and-drop reorders tabs | SESS-06 | Visual + interaction verification | Drag tab to new position, verify order persists visually |
| Tab shows name + status dot | UI-01 | Visual verification | Check each tab has session name and green/gray dot |
| Right-click context menu | UI-02 | Visual + interaction | Right-click tab, verify rename/close menu appears |
| Keyboard shortcuts fire correctly | UI-03 | Requires global shortcuts + Wails backend | Press Ctrl+T, Ctrl+W, Ctrl+Tab, Ctrl+Shift+Tab, verify behavior |
| Scrollback preserved per session | UI-04 | Requires running shell | Type output in one session, switch tabs, switch back — scrollback intact |
| Theme CSS variables respected | UI-05 | Visual verification | Switch theme, verify terminal tabs match app theme |
| Clear clears only active session | UI-06 | Requires running shell | Type in two sessions, click clear, verify only active session clears |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 5s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
