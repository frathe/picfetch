# Microsoft Store MSIX delivery

## Problem

PicFetch has a reserved Microsoft Store identity and a draft Partner Center
submission, but no Store-compatible package or listing. Partner Center currently
shows Pricing and availability, Properties, and Age ratings as complete;
Packages is incomplete and Store listings is not started.

The ordinary Windows release is a pair of signed ZIP archives. Partner Center's
MSIX submission path instead needs packages whose manifest uses the exact Store
identity, whose version is valid and monotonically increasing, and whose two
architectures can be delivered as one bundle. A Store-installed build must also
leave updates to Microsoft: its installed executable is immutable, so PicFetch's
GitHub self-updater must not offer or apply an update there.

## Decisions

| Decision | Choice |
|---|---|
| Store identity | `OpenSourceDeveloperFloria.PicFetch` / `CN=D9654E56-586C-4C1E-ABC8-71CCDC33B78F` / `Open Source Developer Florian Rathe`, copied from Partner Center. |
| Package shape | One unsigned `.msixbundle` containing x64 and ARM64 application packages; Microsoft signs it after certification. |
| Store version | `1.0.<Fyne Build>.0`. `Build` is already monotonic; the first component is non-zero and the fourth remains Store-reserved zero. |
| Device family | `Windows.Desktop`, minimum Windows 10 build 19041, x64 and ARM64 only. |
| Trust model | Packaged classic desktop app at medium integrity with the required `runFullTrust` capability. |
| File associations | One `windows.fileTypeAssociation` generated from `imaging.SupportedExtensions()`. |
| Icons | Deterministically resize `assets/appIcon.png` into the package's required Store, tile, and target-size PNG assets. |
| Update ownership | A `microsoftstore` build tag marks only Store binaries. Those builds do not check GitHub, download stages, or apply staged binaries, and Settings explains that Microsoft Store manages updates. |
| Automation | A dedicated GitHub Actions workflow runs on version tags and manual dispatch, builds both tagged Windows binaries, validates/packages them with Windows SDK tools, runs WACK, and uploads the bundle as a workflow artifact. |
| Listing languages | English and German, matching the app's shipped translations. Reuse the existing valid desktop screenshots and generate a 300x300 Store icon. |
| Portal safety | Prepare all local and CI artifacts first. Upload/save listing data only at the user's browser-action checkpoint; submit for certification only after a separate final confirmation. |

## Acceptance criteria

1. The staging command emits a schema-shaped manifest with the exact Store
   identity, x64/ARM64 architecture mapping, `1.0.<Build>.0`, Desktop targeting,
   `runFullTrust`, and every supported extension.

   ```sh
   go test ./scripts/msixstage
   ```

2. Staging copies the requested executable and produces correctly sized PNG
   assets from the canonical app icon without modifying source assets.

   ```sh
   go test ./scripts/msixstage -run 'TestStage|TestRenderAssets'
   ```

3. Microsoft Store builds are distinguishable at compile time without changing
   ordinary release binaries.

   ```sh
   go test ./internal/distribution && go test -tags microsoftstore ./internal/distribution
   ```

4. Store-managed builds never begin GitHub update work or apply a staged binary,
   and their Settings Updates tab states that Microsoft Store manages updates.

   ```sh
   go test ./internal/ui/... -run 'StoreManaged|MicrosoftStore'
   ```

5. GitHub Actions can manually build the first package and automatically build
   later tag packages; it packages x64 and ARM64 with MakeAppx, bundles them,
   runs the Windows App Certification Kit, and uploads exactly one bundle
   artifact.

   ```sh
   ruby -e 'require "yaml"; YAML.safe_load(File.read(".github/workflows/microsoft-store.yml"), aliases: true)' &&
   rg -n 'workflow_dispatch|tags:|MakeAppx|appcert|msixbundle|picfetch-microsoft-store' .github/workflows/microsoft-store.yml
   ```

6. Store listing handoff material contains English and German descriptions,
   feature copy, screenshot captions, privacy/support/license URLs, the required
   restricted-capability explanation, and only screenshots meeting Microsoft's
   desktop minimum.

   ```sh
   test -f packaging/microsoft-store/listing.md &&
   go test ./scripts/msixstage -run TestStoreListingAssets
   ```

7. The full repository gate remains green.

   ```sh
   make verify
   ```

## Non-goals

- Publishing an EXE/MSI listing instead of MSIX.
- Adding x86, Xbox, Surface Hub, HoloLens, or mobile support.
- Automating Partner Center submission through long-lived API credentials.
- Shipping the unsigned Store bundle as a direct-download or GitHub Release
  asset; it is only for Partner Center, which re-signs it.
- Submitting for certification without the user's final review and confirmation.

## Honest limit

The bundle is created on a Windows GitHub-hosted runner because MakeAppx and the
Windows App Certification Kit are Windows SDK tools. This checkout can fully
test the generator and workflow shape, but a real bundle and Partner Center's
server-side validation exist only after the workflow is committed, pushed, and
run. Certification itself remains Microsoft's external review.

## Task graph

```text
T1 staging generator + tests
 |\
 | +--> T2 Store build marker + updater UI/runtime tests
 +----> T3 GitHub Actions packaging workflow
T1 ----> T4 listing handoff assets and copy
T2 + T3 + T4 --> T5 final verification and Partner Center checkpoint
```

### Task 1 - Generate MSIX staging trees

Owner: T0 inline

Files: create `scripts/msixstage/`; update `qodana.yaml`.

Depends: none.

Contract: `go run ./scripts/msixstage -arch <amd64|arm64> -exe <path> -out
<directory>` creates `AppxManifest.xml`, `picfetch.exe`, and `Assets/*.png`.

Test: manifest values, version validation, architecture validation, extension
parity, copied executable, and exact asset sizes.

Verify: `go test ./scripts/msixstage`.

Budget: 0 spawns; 1 review round; full suite no.

### Task 2 - Make Store update ownership explicit

Owner: T0 inline

Files: create `internal/distribution/`; modify `internal/ui` and
`internal/ui/settingswin`; update translations, tests, Qodana exclusions, and
`ARCHITECTURE.md`.

Depends: none.

Contract: `distribution.StoreManaged` is false normally and true under the
`microsoftstore` build tag. The viewer snapshots it into Settings and guards all
GitHub update entry/apply points.

Test: both build-tag values, disabled/absent update actions with explanatory
copy, and no automatic worker or apply request in a Store-managed viewer.

Verify: `go test ./internal/distribution && go test -tags microsoftstore
./internal/distribution && go test ./internal/ui/... -run
'StoreManaged|MicrosoftStore'`.

Budget: 0 spawns; 1 review round; full suite no.

### Task 3 - Build and validate bundles in GitHub Actions

Owner: T0 inline

Files: modify `Makefile` and `.github/workflows/ci.yml`; create
`.github/workflows/microsoft-store.yml` and `docs/microsoft-store.md`.

Depends: Tasks 1 and 2.

Contract: manual dispatch and `v*` tags build `microsoftstore` x64/ARM64
binaries, stage packages, use SHA-256 MakeAppx packaging, bundle them, run WACK,
and upload `picfetch-microsoft-store.msixbundle` as an Actions artifact. The
reusable CI concurrency key distinguishes its caller so the normal Release and
Store gates for the same tag cannot cancel each other.

Test: YAML parse plus focused workflow/Makefile assertions.

Verify: the AC5 command plus `make fmt-check`.

Budget: 0 spawns; 1 review round; full suite no.

### Task 4 - Prepare Store listing material

Owner: T0 inline

Files: create `packaging/microsoft-store/listing.md` and generated listing icon;
reuse `assets/screens/picture_galery.png` and `assets/screens/viewer.png`.

Depends: Task 1.

Contract: reviewer-ready English/German copy and captions, exact URLs, a
300x300 icon, and two qualifying screenshots.

Test: decode image dimensions and assert required sections/URLs.

Verify: `go test ./scripts/msixstage -run TestStoreListingAssets`.

Budget: 0 spawns; 1 review round; full suite no.

### Task 5 - Verify and hand off to Partner Center

Owner: T0 inline.

Files: all changed files.

Depends: Tasks 1-4.

Contract: all AC commands pass; inspect the diff; run `make verify`; after the
workflow produces a real bundle, upload it and complete the listings only with
the browser confirmation required at that action; stop again before Submit for
certification for final confirmation.

Verify: AC1-AC7 and Partner Center reports Packages and Store listings complete.

Budget: 0 spawns; 2 review rounds; full suite yes.
