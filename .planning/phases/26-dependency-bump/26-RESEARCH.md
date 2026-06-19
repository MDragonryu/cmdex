# Phase 26: Dependency Version Bump — Research

**Researched:** 2026-06-19
**Domain:** Go module dependency upgrade / Wails v3 framework API compatibility
**Confidence:** HIGH

## Summary

This phase bumps two dependencies: Wails from v3.0.0-alpha.74 to v3.0.0-alpha2.104 (a jump of ~30 alpha releases, 820 commits, and a version-scheme change from `alpha.NNN` to `alpha2.NNN`), and the Go toolchain directive from `go 1.25.0` to `go 1.26.0`.

The primary risk is Wails v3 API breakage. After direct source-code comparison of every API surface the project consumes (application boot, service registration, window creation, menu, events, dialogs, and types), **no breaking API changes were found**. The version-scheme change at v104 is cosmetic only. The upgrade should require only `go.mod` edits, `go mod tidy`, and a binding regeneration — no Go source-code changes.

The Go toolchain bump from 1.25.0 to 1.26.0 is safe: Wails v3.0.0-alpha2.104 requires `go 1.25.0`, and the system has Go 1.26.4 installed.

**Primary recommendation:** Edit `go.mod`, run `go mod tidy`, run `go build ./...`, regenerate frontend bindings — no source edits expected.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Go module versioning | Build System | — | go.mod directive and require block |
| Wails framework API | Go Backend (main) | Frontend (bindings) | All service structs, window/menu/event code |
| Frontend binding generation | Build Tooling | — | `wails3 generate build-assets` |
| Indirect dependency resolution | Build System | — | `go mod tidy` |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Wails v3 | v3.0.0-alpha2.104 | Desktop app framework | Project's chosen framework; latest alpha in the series |
| Go toolchain | 1.26.0 | Compiler/runtime | System Go 1.26.4; Wails requires ≥1.25.0 |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| modernc.org/sqlite | v1.47.0 | SQLite (CGo-free) | Project direct dependency; Wails go.mod lists v1.44.3; our direct dep wins |
| github.com/creack/pty | v1.1.24 | PTY pseudo-terminals | Unchanged |
| github.com/google/cel-go | v0.27.0 | CEL expression evaluation | Unchanged |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| v3.0.0-alpha2.104 | Staying at alpha.74 | Misses ~30 releases of bug fixes; the breakage window only widens over time |

**Installation:**
```bash
# After go.mod edit:
go get github.com/wailsapp/wails/v3@v3.0.0-alpha2.104
go mod tidy
wails3 generate build-assets
```

**Version verification:**
- Wails v3.0.0-alpha2.104: confirmed via GitHub API tree (commit `d639cef`) and raw source inspection
- Go 1.26.4: confirmed via `go version` (darwin/arm64)
- Wails go.mod requires `go 1.25.0` — compatible with 1.26.x

## Package Legitimacy Audit

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| github.com/wailsapp/wails/v3 | Go modules | ~3 yrs (v3 alpha series) | N/A (Go proxy) | github.com/wailsapp/wails | [OK] | Approved — well-known project, 34.9k stars |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

*All packages are existing project dependencies; no new packages are introduced by this bump beyond the Wails version update itself and its transitive dependency updates.*

## API Changes Summary

### Verified: No Breaking Changes

Every Wails v3 API surface used by this project was cross-referenced against the v3.0.0-alpha2.104 source. Results:

| API Surface | alpha.74 Usage | alpha2.104 Status | Source |
|------------|----------------|-------------------|--------|
| `application.New(Options{…})` | main.go:16 | ✓ Identical signature | [VERIFIED: raw source] |
| `application.NewService(&T{})` | main.go:19–25 | ✓ Now generic `NewService[T any](*T) Service`; type inference backward-compatible | [VERIFIED: raw source — services.go] |
| `Options.Name`, `.Services`, `.Assets` | main.go:17–27 | ✓ All fields present | [VERIFIED: raw source — application_options.go] |
| `Options.Mac` (`MacOptions`) | main.go:30–32 | ✓ `MacOptions` struct unchanged; `ApplicationShouldTerminateAfterLastWindowClosed` present | [VERIFIED: raw source — application_options.go] |
| `AssetOptions.Handler` | main.go:28 | ✓ Present; `BundledAssetFileServer(fs.FS) http.Handler` unchanged | [VERIFIED: raw source — application_options.go] |
| `application.WebviewWindowOptions` | main.go:65–73, app.go:92–106 | ✓ All fields present (Title, Width, Height, MinWidth, MinHeight, UseApplicationMenu, BackgroundColour, HideOnEscape, DisableResize, MinimiseButtonState, MaximiseButtonState, URL, Name, InitialPosition, X, Y, Hidden) | [VERIFIED: raw source — webview_window_options.go] |
| `application.NewRGBA(r,g,b,a) RGBA` | main.go:72, app.go:95 | ✓ Return type is `RGBA` value type; usage identical | [VERIFIED: raw source — webview_window_options.go] |
| `application.WindowCentered` | app.go:104 | ✓ Constant `WindowCentered WindowStartPosition = 0` unchanged | [VERIFIED: raw source — webview_window_options.go] |
| `application.ButtonDisabled` | app.go:100–101 | ✓ Constant `ButtonDisabled ButtonState = 1` unchanged | [VERIFIED: raw source — webview_window_options.go] |
| `events.Common.WindowClosing` | main.go:75, app.go:109 | ✓ `WindowClosing WindowEventType = 1030` unchanged | [VERIFIED: raw source — events.go] |
| `RegisterHook(WindowingEventType, func(*WindowEvent))` | main.go:75–77 | ✓ Signature unchanged in `Window` interface | [VERIFIED: raw source — window.go] |
| `OnWindowEvent(WindowEventType, func(*WindowEvent))` | app.go:109 | ✓ Signature unchanged in `Window` interface | [VERIFIED: raw source — window.go] |
| `app.Window.Current()` → `Window` | main.go:57 | ✓ `WindowManager` present on `App`; `Window` interface has `OpenDevTools()` | [VERIFIED: raw source — application.go, window.go] |
| `w.OpenDevTools()` | main.go:59 | ✓ Present on `Window` interface | [VERIFIED: raw source — window.go] |
| `app.NewMenu()` | main.go:35 | ✓ Delegates to `a.Menu.New()` | [VERIFIED: raw source — menu.go] |
| Menu: `AddSubmenu`, `AddRole`, `Add`, `SetAccelerator`, `OnClick` | main.go:37–48 | ✓ All present; `AddRole` accepts `Role` type with constants `About`, `Hide`, `HideOthers`, `Reload`, `Quit`, `EditMenu` | [VERIFIED: raw source — menu.go] |
| `app.Run()` | main.go:79 | ✓ Signature unchanged `func (a *App) Run() error` | [VERIFIED: raw source — application.go] |
| `application.Get()` | app.go:29 | ✓ Returns `*App`; calls `globalApplication` singleton | [VERIFIED: raw source — application.go] |
| `ServiceStartup(ctx, ServiceOptions) error` | All services | ✓ Interface unchanged; `ServiceOptions` struct present | [VERIFIED: raw source — services.go] |
| `ServiceShutdown() error` | app.go:40, terminal_service.go:165 | ✓ Interface unchanged | [VERIFIED: raw source — services.go] |
| `app.Quit()` | main.go:76 | ✓ Present | [VERIFIED: raw source — application.go] |
| `Dialog.OpenFile()`, `.SaveFile()` | app.go:71, importexport_service.go:24,76,179 | ✓ `DialogManager` present on `App` | [VERIFIED: raw source — application.go] |
| `app.Event.Emit(name, data)` | terminal_service.go:493,566,676 | ✓ `EventManager` present on `App` | [VERIFIED: raw source — application.go, events.go] |

### What Changed (non-impact)

These are changes between alpha.74 and alpha2.104 that do **not** affect this project:

| Change | Version | Impact |
|--------|---------|--------|
| Version scheme `alpha.NNN` → `alpha2.NNN` | alpha2.104 | Cosmetic only; Go module path unchanged |
| macOS coordinate system normalisation | alpha.91 | Project does not use `GetScreens`/`SetPosition`/`Position` |
| Tri-state `ButtonState` replaces boolean fullscreen API | alpha.84 | Project already uses `ButtonDisabled` constant |
| iOS/Android platform managers | alpha.103 | Project is desktop-only |
| New optional `WebviewWindowOptions` fields (Permissions, DevToolsEnabled, ContentProtectionEnabled, etc.) | various | Opt-in; defaults preserve existing behavior |
| `Service` is now a generic struct `Service` wrapping instance, not an interface | — | `NewService[T any](*T) Service` inferred from argument; no code change |
| Mobile bridge event renaming (`native:*` → `common:*`/`ios:*`/`android:*`) | alpha.103 | Project does not use mobile events |
| Garble obfuscation support | alpha.96 | Opt-in build flag; no source change |

## Affected Files

No Go source files should require editing. The only files touched are:

| File | Change | Reason |
|------|--------|--------|
| `go.mod` | Edit: `go 1.26.0`, `wails/v3 v3.0.0-alpha2.104` | Direct version bump |
| `go.sum` | Auto-generated | `go mod tidy` output |
| `frontend/bindings/` | Auto-regenerated | `wails3 generate build-assets` output; never hand-edited |

## Migration Steps

### Step 1: Edit go.mod
```bash
# Change line 3: go 1.25.0 → go 1.26.0
# Change line 9: v3.0.0-alpha.74 → v3.0.0-alpha2.104
```

### Step 2: Resolve dependencies
```bash
go mod tidy
```

### Step 3: Verify Go compilation
```bash
go build ./...
```

If compilation fails, the most likely cause is an indirect dependency conflict. Run:
```bash
go mod why <conflicting-package>
```
To diagnose. The only known risk is `modernc.org/sqlite` version mismatch (see Risks below).

### Step 4: Regenerate frontend bindings
```bash
wails3 generate build-assets
```

### Step 5: Verify TypeScript compilation
```bash
make check
# or: cd frontend && pnpm tsc --noEmit
```

### Step 6: Run tests (if any)
```bash
go test ./...
```

## Go 1.26 Compatibility

**Status: Compatible — HIGH confidence.**

- Wails v3.0.0-alpha2.104 `go.mod` specifies `go 1.25.0` [VERIFIED: raw source]
- System Go is 1.26.4 darwin/arm64 [VERIFIED: command `go version`]
- Go 1.26 is backward-compatible with Go 1.25 code
- No language changes in Go 1.26 that would break existing code patterns in this project
- Other direct dependencies (`creack/pty v1.1.24`, `cel-go v0.27.0`, `uuid v1.6.0`, `modernc.org/sqlite v1.47.0`) all have Go version requirements ≤1.25

## Risks & Mitigations

### Risk 1: Indirect Dependency Version Conflicts
**Severity:** Low | **Likelihood:** Medium

The project has `modernc.org/sqlite v1.47.0` as a direct dependency, while Wails v3.0.0-alpha2.104 lists `modernc.org/sqlite v1.44.3` in its own `go.mod`. Go modules MVS (Minimal Version Selection) will honor the project's direct requirement (v1.47.0). However, `modernc.org/libc` (a transitive dep of sqlite) may also differ (project: v1.70.0, Wails: v1.67.6).

**Mitigation:** `modernc.org/sqlite` minor versions are API-compatible (it's a CGo-free SQLite implementation wrapping the same C source). If `go build ./...` passes, this is resolved. If it fails, check for struct field or function signature changes in the sqlite package.

### Risk 2: Generated Binding Format Change
**Severity:** Low | **Likelihood:** Low

The `wails3 generate build-assets` output format may have changed between alpha.74 and alpha2.104. The TypeScript bindings in `frontend/bindings/` are generated and never hand-edited.

**Mitigation:** Run `wails3 generate build-assets` and verify `pnpm tsc --noEmit` passes. If TypeScript compilation fails, the bindings format changed and the frontend import paths or call signatures need updating. This is unlikely since the Go method signatures haven't changed.

### Risk 3: Wails3 CLI Version Mismatch
**Severity:** Medium | **Likelihood:** Low

The `wails3` CLI binary must match the library version. If the system `wails3` CLI is still at an older version, `wails3 generate build-assets` may produce incompatible bindings.

**Mitigation:** Install the matching CLI before regeneration:
```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha2.104
```

### Risk 4: Behavioral Changes from Bug Fixes
**Severity:** Low | **Likelihood:** Low

Several bug fixes between alpha.74 and alpha2.104 may subtly change runtime behavior:
- macOS menu state fixes (alpha.89) — menus now update synchronously
- Windows callback batching (alpha.90) — may affect event timing
- GTK thread safety fixes (alpha.86, alpha.93) — Linux desktop only

**Mitigation:** Manual smoke testing after the bump (open app, create/edit/delete a command, open terminal session, open settings window, close app).

## Recommendations

1. **Atomic bump** — Edit both `go` directive and Wails version in a single commit. Do not split across commits; the intermediate state would be invalid.

2. **Regenerate bindings immediately** — After `go mod tidy` + `go build ./...` passes, run `wails3 generate build-assets` before any further frontend work. Stale bindings cause silent type mismatches.

3. **CLI version check** — Verify `wails3 --version` reports `v3.0.0-alpha2.104` or later. If not, install the matching CLI.

4. **Smoke test before committing** — Run `wails3 dev` and verify the app launches, settings window opens, terminal session works, and commands execute. The `go build` check only verifies compilation, not runtime behavior.

5. **Commit message:**
   ```
   chore(deps): bump Wails to v3.0.0-alpha2.104, Go to 1.26.0
   
   - Wails v3.0.0-alpha.74 → v3.0.0-alpha2.104 (820 commits)
   - Go toolchain 1.25.0 → 1.26.0
   - No API breakage; all consumed Wails APIs verified compatible
   - Regenerated frontend bindings via `wails3 generate build-assets`
   ```

## Architecture Patterns

No architecture changes are introduced by this bump. The existing Wails v3 service-registration pattern (six service structs registered as `application.Service` in `main.go`) is preserved exactly.

### Anti-Patterns to Avoid
- **Manually editing `frontend/bindings/`:** These are generated output. Any issues must be fixed in Go service method signatures, then regenerated.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Version resolution | Manual go.sum editing | `go mod tidy` | MVS algorithm; hand-edits cause checksum mismatches |
| Binding generation | Hand-writing TypeScript stubs | `wails3 generate build-assets` | Type-safe, auto-updated from Go method signatures |
| Dependency conflict resolution | `replace` directives | `go mod why` + `go mod graph` | `replace` permanently pins versions and masks breakage |

## Common Pitfalls

### Pitfall 1: Stale wails3 CLI Binary
**What goes wrong:** `wails3 generate build-assets` succeeds but produces bindings for an older API surface. Frontend TypeScript compilation fails with type errors on the generated bindings.
**Why it happens:** The system `wails3` CLI was installed from an older version and wasn't updated alongside the library bump.
**How to avoid:** Run `wails3 version` before regenerating and confirm the version matches. If not, run `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha2.104`.
**Warning signs:** `pnpm tsc --noEmit` fails on files in `frontend/bindings/` after regeneration.

### Pitfall 2: Forgetting to Run `go mod tidy`
**What goes wrong:** `go build ./...` passes locally but fails in CI because `go.sum` is missing entries for new indirect dependencies.
**Why it happens:** `go build` doesn't automatically update `go.sum` for all transitive deps; `go mod tidy` does.
**How to avoid:** Always run `go mod tidy` after editing `go.mod` before committing.
**Warning signs:** CI fails with "missing go.sum entry" errors.

### Pitfall 3: Platform-Specific Build Tag Breakage
**What goes wrong:** `go build ./...` passes on macOS but fails on Windows/Linux-specific files (e.g., `pty_backend_windows.go`).
**Why it happens:** Cross-platform build tags and platform-specific dependencies may have changed between Wails versions.
**How to avoid:** Run cross-compilation checks if CI doesn't catch them:
```bash
GOOS=linux GOARCH=amd64 go build ./...
GOOS=windows GOARCH=amd64 go build ./...
```
**Warning signs:** Build succeeds on one OS but fails on another.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Compilation | ✓ | 1.26.4 | — |
| wails3 CLI | Binding generation, dev server | ? | Unknown | Install via `go install` if missing |
| Node.js / pnpm | Frontend build, TypeScript check | ? | Unknown | Required for `make check` |

**Missing dependencies with no fallback:**
- `wails3` CLI must be at version ≥ v3.0.0-alpha2.104 for binding generation. If not installed, install it.

**Missing dependencies with fallback:**
- None critical — both Go and Node toolchains are project prerequisites already.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go standard `testing` |
| Config file | none |
| Quick run command | `go test ./...` |
| Full suite command | `go test -count=1 ./...` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| — | `go build ./...` passes | compile | `go build ./...` | N/A |
| — | `make check` passes (Go + frontend) | compile | `make check` | N/A |
| — | DB migration tests pass | unit | `go test -run TestFreshDBMigrations -v ./...` | ✅ db_test.go |
| — | App launches without panic | manual | `wails3 dev` | N/A |

### Sampling Rate
- **Per task commit:** `go build ./...`
- **Per wave merge:** `make check`
- **Phase gate:** Full suite green + manual smoke test

### Wave 0 Gaps
- [ ] No frontend test infrastructure exists — but this phase doesn't modify frontend code
- [ ] No integration/E2E tests — manual smoke test required

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | All consumed Wails v3 APIs are backward-compatible from alpha.74 to alpha2.104 | API Changes Summary | Go compilation fails; requires source edits to affected files |
| A2 | `modernc.org/sqlite v1.47.0` (project) is compatible with Wails compiled against v1.44.3 | Risks | Runtime SQLite behavior difference; unlikely given CGo-free implementation |
| A3 | `wails3 generate build-assets` output format hasn't changed in a way that breaks `pnpm tsc --noEmit` | Risks | Frontend TypeScript compilation fails; trivial to fix |
| A4 | Go 1.26.0 is fully backward-compatible for this codebase | Go 1.26 Compatibility | Compilation fails on Go 1.26-specific changes (very unlikely for Go) |

## Sources

### Primary (HIGH confidence)
- Wails v3.0.0-alpha2.104 source code at `github.com/wailsapp/wails` (commit `d639cef`):
  - `v3/pkg/application/application.go` — App struct, New(), Run(), Get(), Quit()
  - `v3/pkg/application/application_options.go` — Options, MacOptions, AssetOptions
  - `v3/pkg/application/webview_window_options.go` — WebviewWindowOptions, RGBA, ButtonState, WindowStartPosition
  - `v3/pkg/application/services.go` — Service struct, NewService, ServiceOptions, ServiceStartup, ServiceShutdown
  - `v3/pkg/application/window.go` — Window interface (RegisterHook, OnWindowEvent, OpenDevTools)
  - `v3/pkg/application/menu.go` — Menu, NewMenu, AddSubmenu, AddRole
  - `v3/pkg/application/events.go` — EventProcessor, CustomEvent, Emit
  - `v3/pkg/application/bindings.go` — Bindings, getMethods, internalServiceMethods
  - `v3/pkg/events/events.go` — Common events, WindowEventType, WindowClosing
  - `v3/go.mod` — Go version requirement and dependency list
- GitHub Releases — release notes for alpha.84 through alpha2.104
- System: `go version go1.26.4 darwin/arm64`

### Secondary (MEDIUM confidence)
- GitHub compare `v3.0.0-alpha.74...v3.0.0-alpha2.104` — 820 commits, 4,575 files changed (comparison timed out; used individual source file inspection instead)
- Release notes page 1 (alpha.91–alpha2.104) and page 2 (alpha.84–alpha.93)

### Tertiary (LOW confidence)
- None — all claims verified against authoritative sources.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — Wails version confirmed via GitHub API tree and raw source; Go version confirmed via system tool
- Architecture: HIGH — direct source-code comparison of every consumed API surface
- Pitfalls: MEDIUM — based on prior dependency bump experience and Wails v3 alpha volatility

**Research date:** 2026-06-19
**Valid until:** 2026-07-19 (Wails v3 alpha releases are nightly; a newer version may exist within 30 days)
