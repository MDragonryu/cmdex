# Release Cheatsheet

Quick reference for cutting a build. For the full pipeline description see [DEPLOYMENT.md](DEPLOYMENT.md).

This project uses **Wails v3 + [Task](https://taskfile.dev)**. The `Makefile` is a thin
wrapper; `Taskfile.yml` is the real entry point.

---

## 0. Every time — PATH

`wails3` installs to `~/go/bin`, which is usually not on `$PATH`:

```bash
export PATH="$HOME/go/bin:$PATH"
wails3 version    # expect v3.0.0-alpha.74
```

If missing entirely:

```bash
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-alpha.74
```

---

## 1. The real release: push a tag

`.github/workflows/release.yml` triggers on `v*.*.*` and builds all three platforms on
native runners, then publishes a GitHub Release with auto-generated notes.

```bash
# 1. bump `version:` in build/config.yml
# 2. commit it
git tag v0.1.0
git push origin v0.1.0
```

Building natively per-platform is what you want for anything users will download —
local cross-builds are for testing.

---

## 2. macOS — `.app` + `.dmg`

```bash
task package:universal      # arm64 + amd64, ~2x build time
task package                # native arch only, faster
```

Output: `bin/cmdex.app`, `bin/cmdex.dmg`

Requires Xcode Command Line Tools (CGO is on for darwin).

**Only ad-hoc signed** (`codesign --sign -`). Works on your machine; Gatekeeper blocks
it everywhere else. Recipients need:

```bash
xattr -dr com.apple.quarantine /Applications/cmdex.app
```

For real distribution, fill `SIGN_IDENTITY` / `KEYCHAIN_PROFILE` in
`build/darwin/Taskfile.yml` (vars block at the top), then `task darwin:sign:notarize`.

---

## 3. Windows — NSIS installer (cross-builds fine from macOS)

One-time:

```bash
brew install makensis
```

```bash
LANG=en_US.UTF-8 task windows:package ARCH=amd64
```

Output: `bin/CmDex-amd64-installer.exe`

Two gotchas, both easy to hit:

| Symptom | Cause | Fix |
|---|---|---|
| `Bad text encoding: project.nsi:26` | Shell has `LC_CTYPE=C`; makensis can't decode the `©` in a comment on line 26 | Prefix with `LANG=en_US.UTF-8` |
| Installer named `-arm64-`, `.exe` is `PE32+ Aarch64` | `ARCH` defaults to *host* arch, so an Apple Silicon Mac targets ARM64 Windows | Pass `ARCH=amd64` explicitly |

Verify what you actually built before shipping:

```bash
file bin/cmdex.exe    # want: PE32+ executable (GUI) x86-64
```

No CGO needed for Windows (`CGO_ENABLED=0`, pure-Go SQLite), so plain Go
cross-compilation handles it — no Docker required.

**Unsigned** → SmartScreen warning for users. Needs an Authenticode cert plus
`SIGN_CERTIFICATE`/`SIGN_THUMBPRINT` in `build/windows/Taskfile.yml`, then
`task windows:sign:installer`. There is no local workaround like macOS ad-hoc signing.

---

## 4. Linux — AppImage / deb / rpm

```bash
task package        # run this ON Linux
```

Output: `bin/*.AppImage`, `bin/*.deb`, `bin/*.rpm`

Needs `libgtk-3-dev` + `libwebkit2gtk-4.1-dev`. The packaging steps are Linux-native, so
from macOS use CI. You can still cross-build the raw *binary* via Docker:

```bash
task setup:docker   # ~800MB, one-time
task linux:build
```

---

## 5. Sanity checks

```bash
make check                  # go build ./... + tsc --noEmit
go test ./...
cd frontend && pnpm test:e2e
```

Note you cannot smoke-test a Windows or Linux build from macOS — cross-compiling proves
it links, not that it launches. CI's native runners are the real signal.

---

## Where things live

| What | File |
|---|---|
| Version, product name, bundle ID | `build/config.yml` |
| macOS bundle + signing tasks | `build/darwin/Taskfile.yml` |
| Windows build + NSIS + signing | `build/windows/Taskfile.yml` |
| NSIS installer script | `build/windows/nsis/project.nsi` |
| Linux packaging | `build/linux/Taskfile.yml` |
| Release workflow | `.github/workflows/release.yml` |
