---
phase: 24
slug: session-aware-execution
status: verified
threats_open: 0
asvs_level: 2
created: 2026-06-16
verified: 2026-06-16
---

# Phase 24 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| Backend service-to-service (in-process) | `ExecutionService` invokes `TerminalService` via the `terminalSvc` package-level global pointer — no IPC, no auth | Resolved command line, session ID |
| Test setup to production state | `testWithTerminalSvc` saves/restores the `terminalSvc` global — tests must not leak terminal state into other tests | TerminalService pointer |
| Frontend ↔ Backend (Wails IPC) | Generated bindings (executionservice.js, eventservice.js, models.js) form the contract between React frontend and Go services. Regenerated to match Plan 01. | Method calls, struct shapes |
| E2E test mocks ↔ real runtime | e2e/mocks/runtime.ts is a Playwright-only in-memory backend. Handlers mirror the live Go service surface. | Mock method IDs |

---

## Threat Register

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-24-01 | Denial of Service | TerminalService.Write called with malicious long input | accept | cmdLine is built from user-saved script content (trusted source). D-09 makes terminal runs transient. Length bounded by script content field. | closed |
| T-24-02 | Elevation of Privilege | Race between TerminalService.ServiceShutdown and RunCommand | mitigate | Nil checks in RunCommand: `execution_service.go:138-144` (terminalSvc) and `execution_service.go:146-152` (session). Both return `ExecutionRecord{Error, ExitCode: -1}` without panicking. | closed |
| T-24-03 | Tampering (Test Isolation) | testWithTerminalSvc leaks terminalSvc between tests | mitigate | `execution_service_test.go:61` saves prevTerminalSvc; cleanup at lines 68-72 restores it via deferred call. Same pattern as TestTerminalService_ServiceStartupAssignsTerminalSvc. | closed |
| T-24-04 | Information Disclosure | record.Error may leak internal Go error message to frontend | accept | Frontend's runCommandDirect (App.tsx:990-994) already surfaces result.error via toast. Pre-existing behavior, not a regression. | closed |
| T-24-05 | Tampering (Test Skip logic) | New test in CI without PTY could fail instead of skip | mitigate | `execution_service_test.go:58-60` skips on testing.Short(); lines 64-67 skip on ServiceStartup error. | closed |
| T-24-06 | Tampering (Bindings Drift) | Stale import of RunInTerminal or eventNames.cmdExecuting after bindings regen | mitigate | Pre-flight grep + post-edit grep for RunInTerminal/cmdExecuting in frontend/src/ returned 0. `pnpm tsc --noEmit` passes. | closed |
| T-24-07 | Tampering (CSS Orphan) | .output-pane* rules left in style.css with no DOM target | accept | Pure dead-CSS — no runtime impact, no security surface. Removed for code hygiene. | closed |
| T-24-08 | Information Disclosure | t('common.copyLastOutput') i18n key removal | accept | Terminal copy button at App.tsx:1572 still uses this key. Verified preserved at en.json:10. | closed |
| T-24-09 | Denial of Service (E2E Mocks) | e2e mock with stale method IDs | mitigate | Handlers with IDs 1736747747/2752844091/3022740230 (RunInTerminal/GetExecutionHistory/ClearExecutionHistory) removed from runtime.ts. | closed |
| T-24-10 | Tampering (Test Isolation) | e2e mock state leak via executionHistory | mitigate | executionHistory array declaration + 3 reset references removed from runtime.ts. | closed |

*Status: closed · open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| R-24-01 | T-24-01 | User-saved script content is the trusted source for cmdLine. Terminal runs are transient (no persistent execution record). Length bounded by SQLite script content field. DoS risk bounded by user input. | gsd-security-auditor | 2026-06-16 |
| R-24-02 | T-24-04 | Error surface to frontend (result.error via toast) is pre-existing behavior since Phase 18. Not a regression introduced by Phase 24. Internal Go error messages may leak file paths or Go internals — low impact in single-user desktop app context. | gsd-security-auditor | 2026-06-16 |
| R-24-03 | T-24-07 | Pure dead-CSS removal (no DOM target, no runtime impact, no security surface). Code hygiene only. | gsd-security-auditor | 2026-06-16 |
| R-24-04 | T-24-08 | i18n key removal is the inverse of info disclosure — removing orphan keys that were never displayed. The terminal copy button retains its i18n key. | gsd-security-auditor | 2026-06-16 |

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-06-16 | 10 | 10 | 0 | gsd-security-auditor |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-06-16
