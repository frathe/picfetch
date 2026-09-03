# Microsoft Store MSIX

## Frame

Deliver a Microsoft Store-ready x64/ARM64 MSIX bundle whose manifest uses the
Partner Center identity for PicFetch, registers every supported image type,
and whose Store build defers application updates to Microsoft Store.

Route: **Deep**. This adds a Windows shipping format, build-channel-specific
runtime behavior, release automation, and Store identity metadata.

## Problem

PicFetch currently publishes signed portable Windows executables inside ZIP
archives. Partner Center has an MSIX product reservation, but the repository
has no MSIX manifest, package assets, bundle builder, or Store distribution
channel. The existing in-app updater replaces its running executable, which is
not a valid update mechanism for an MSIX installed under WindowsApps.

## Decisions — do not relitigate

| Decision | Choice |
|---|---|
| Partner Center product name | `PicFetch` |
| Package identity name | `OpenSourceDeveloperFloria.PicFetch` |
| Package publisher | `CN=D9654E56-586C-4C1E-ABC8-71CCDC33B78F` |
| Publisher display name | `Open Source Developer Florian Rathe` |
| Store ID | `9P0DM0KTH01K` (documentation only; not a manifest field) |
| Package format | One `.msixbundle` containing x64 and ARM64 packages |
| Store package version | `1.0.<Fyne Build>.0`; for Build 440, `1.0.440.0` |
| Runtime model | Packaged classic desktop app, medium integrity, full trust |
| Minimum Windows | Windows 10 build 19041, required by the chosen uap10 manifest vocabulary |
| File associations | One alternate image-viewer association containing exactly `imaging.SupportedExtensions()` |
| Build selection | `microsoftstore` Go build tag; ordinary builds remain self-updating |
| Store Updates tab | Show version plus “Updates are managed by Microsoft Store.”; no update controls |
| Release behavior | Existing portable archives and Certum signing stay unchanged; add a separately built Store artifact |
| Store submission | Manual first upload; Partner Center API automation is deferred |

## Test seams confirmed by the user

1. `make package-windows-store` is the packaging command and produces one
   x64/ARM64 `.msixbundle` on a Windows host with MakeAppx.
2. `scripts/msix` is the deterministic package-layout seam: tests inspect its
   manifest, identity, package version, assets, architectures, and file types.
3. `settingswin.New(..., storeManagedUpdates)` is the presentation seam: Store
   mode shows the managed-update message and exposes no self-update controls.
4. `viewer` update entry points are the behavior seam: Store mode performs no
   automatic/manual check, requests no apply/relaunch, and skips staged apply
   on shutdown; portable mode retains the existing tests and behavior.

## Acceptance criteria

### AC1 — Store update behavior is inert

Store mode restores no self-update preference, starts no automatic or manual
worker, never applies a staged portable update during shutdown, and never quits
to self-update. Portable behavior is unchanged.

Verify:

```sh
go test ./internal/ui/... -run 'TestStoreManagedUpdates|TestManualUpdateFlow|TestCheckForUpdates'
```

### AC2 — Settings explains Store-managed updates

The Store Updates tab contains the version label and localized managed-update
message, but not the automatic checkbox or manual button. Portable mode keeps
all three existing objects.

Verify:

```sh
go test ./internal/ui/settingswin/... -run 'Test(StoreManagedUpdates|UpdatesTab|UpdateNow)'
go test . -run 'TestTranslations'
```

### AC3 — MSIX layout is derived and exact

The layout generator emits a valid XML manifest containing the exact Partner
Center identity, `PicFetch` display name, x64 or ARM64 architecture, version
`1.0.<Build>.0`, packaged-classic/full-trust declarations, all supported image
extensions, and every referenced PNG asset plus `picfetch.exe`.

Verify:

```sh
go test ./scripts/msix/...
```

### AC4 — The public packaging command builds both architectures

`make package-windows-store` builds Store-tagged x64 and ARM64 executables,
creates both MSIX packages with MakeAppx semantic validation enabled, bundles
them, and writes `bin/PicFetch-Store.msixbundle`. It fails clearly when run
without Windows SDK tooling.

Verify on Windows:

```powershell
make package-windows-store
Test-Path bin/PicFetch-Store.msixbundle
```

Verify statically on this host:

```sh
go test ./scripts/msix/... && make -n package-windows-store
```

### AC5 — Release and CI retain existing channels and add the Store artifact

The release workflow still publishes the existing signed ZIPs and also uploads
the Store bundle. CI compiles the Store-tagged UI path and tests the layout
generator on Windows.

Verify:

```sh
go test ./scripts/msix/... && rg -n 'package-windows-store|PicFetch-Store.msixbundle|microsoftstore' Makefile .github/workflows
```

### AC6 — Repository records match the new package map and workflow

`ARCHITECTURE.md`, `README.md`, and `todos.md` describe the Store channel,
package generator, release artifact, manual first submission, and Windows-only
validation limit.

Verify:

```sh
rg -n 'Microsoft Store|MSIX|scripts/msix|microsoftstore' ARCHITECTURE.md README.md todos.md
```

## Non-goals

- Operating Partner Center, uploading the first package, or submitting it for
  certification.
- Automating Partner Center submissions before the first manual certification.
- Replacing the portable ZIP/WinGet release channel or its updater.
- Migrating settings between an existing portable copy and the packaged copy.
- Adding Microsoft commerce, WNS, Xbox, or telemetry integrations.
- Producing an EXE/MSI installer.

## Honest limit

This host cannot install or launch an MSIX, run MakeAppx, or run the Windows App
Certification Kit. The generator, manifests, build graph, Go behavior, and
cross-compilation can be verified here; actual package creation, installation,
file-association activation, upgrade/uninstall behavior, and WACK results must
be verified by the Windows workflow and then on a clean Windows machine before
upload.

## File map

| Area | Files |
|---|---|
| Store build selection | create `internal/ui/distribution_default.go`, `internal/ui/distribution_microsoftstore.go`; modify `startup.go`, `viewer.go` |
| Update behavior | modify `internal/ui/autoupdate.go`, `run.go`, `features.go`, `autoupdate_test.go` |
| Settings presentation | modify `internal/ui/settingswin/settingswin.go`, `settingswin_test.go`, both translation bundles |
| Package generator | create `scripts/msix/main.go`, `scripts/msix/package.go`, `scripts/msix/package_test.go` |
| Packaging/release | modify `Makefile`, `.github/workflows/ci.yml`, `.github/workflows/release.yml` |
| Project records | modify `ARCHITECTURE.md`, `README.md`, `todos.md`, `qodana.yaml` |

## Task graph

```text
T1 Store update policy -----> T2 Settings presentation
          |                            |
          +------------+---------------+
                       v
T3 MSIX layout generator ---> T4 Makefile/workflows ---> T5 docs/final gate
```

### Task 1 — Store update policy

Owner: T0 inline
Files: `internal/ui/distribution_*.go`, `startup.go`, `viewer.go`, `features.go`,
`run.go`, `autoupdate.go`, `autoupdate_test.go`
Depends: none
Contract: `startupState.storeManagedUpdates` is immutable after construction;
the zero value preserves portable behavior.
Test: Store-mode viewer update entry points are inert and shutdown skips apply.
Verify: AC1 command
Budget: 0 spawns · ≤ 2 review rounds · full suite: no

### Task 2 — Store Settings presentation

Owner: T0 inline
Files: `internal/ui/settingswin/settingswin.go`, `settingswin_test.go`,
`translations/en.json`, `translations/de.json`
Depends: T1 field contract
Contract: `settingswin.New(app, host, storeManagedUpdates)`; portable layout
unchanged.
Test: Store Updates tab object identity and host non-interaction.
Verify: AC2 commands
Budget: 0 spawns · ≤ 2 review rounds · full suite: no

### Task 3 — MSIX layout generator

Owner: T0 inline
Files: `scripts/msix/*`, `qodana.yaml`
Depends: none
Contract: CLI `msix --arch amd64|arm64 --exe PATH --out DIR`; output is an
unpacked MSIX layout for MakeAppx.
Test: black-box `run` plus parsed XML and decoded asset dimensions.
Verify: AC3 command
Budget: 0 spawns · ≤ 3 review rounds · full suite: no

### Task 4 — Packaging and workflow integration

Owner: T0 inline
Files: `Makefile`, `.github/workflows/ci.yml`, `.github/workflows/release.yml`
Depends: T1, T3
Contract: `make package-windows-store` -> `bin/PicFetch-Store.msixbundle`.
Test: generator tests plus static dry-run/workflow assertions.
Verify: AC4 and AC5 commands
Budget: 0 spawns · ≤ 2 review rounds · full suite: no

### Task 5 — Records and final gate

Owner: T0 inline
Files: `ARCHITECTURE.md`, `README.md`, `todos.md`, this plan
Depends: T1–T4
Contract: no new package/file moves remain undocumented; first upload remains
manual and Windows runtime verification remains explicit.
Test: documentation grep, all AC commands, negative guard checks, final gate.
Verify: AC6 plus `make verify`
Budget: 0 spawns · ≤ 2 review rounds · full suite: yes, once

## Delegation routing

All tasks remain T0 inline. T1 and T2 fail G5 because the lead holds the active
architecture and packaging context. T3 and T4 also fail G5 and fall under the
workflow’s “platform-specific behavior that cannot be tested here” prohibition.
T5 is review/final-gate work and is never delegated. Rule S applies only to
generated PNG resizing and formatting; those deterministic operations run as
commands rather than model work.

## Cost ledger

| Task | Spawns (budget/actual) | Review rounds | Full suite | Notes |
|---|---:|---:|---|---|
| T1 | 0 / 0 | 0 | no | |
| T2 | 0 / 0 | 0 | no | |
| T3 | 0 / 0 | 0 | no | |
| T4 | 0 / 0 | 0 | no | |
| T5/gate | — | — | yes | |
