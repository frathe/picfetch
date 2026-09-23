# Windows HEIC codec runners (2026-09-23)

## Decision

The maintainer subsequently **closed the Windows/Store pre-release qualification
items** and will test them after rollout (2026-09-23). Native x64/ARM64 evidence,
installed-package behavior, codec recovery and final-package qualification are
accepted deferrals. They are no longer release blockers. The unchecked runtime
facts below remain evidence limits, not open pre-release tasks.

The maintainer chose to **keep x64 CI and exclude only installed-codec HEIC
tests there**. Switching CI to ARM is rejected. The runner now provides
`-skip-heic-codecs`, restricted to Windows/Store suites in GitHub Actions; both
hosted Windows commands select it. Local native commands and Linux/macOS CI
remain strict. Windows worker restrictions, alpha metadata and portable
regressions remain in CI. This explicit decision supersedes the original
proposal to require a codec-equipped runner before CI can pass.

The alternatives below record the investigation, not a pending CI migration.
They remain relevant if automated codec qualification is desired later.

The shortest path using existing equipment is to run the unchanged native guards
on the maintainer's codec-equipped Windows machines, then use a qualified,
isolated machine for the corresponding CI job. This is a recommendation,
conditional on those machines still being available. A manual run supplies
evidence but does not fix the failing required GitHub check by itself.

The simplest **candidate** for repeatable native ARM64 coverage is a small probe job on GitHub's standard `windows-11-arm` runner. GitHub documents this as native Windows 11 Arm64; no codec installation or custom image is needed to *try* it. Its [image inventory](https://github.com/actions/runner-images/blob/main/images/windows/Windows11-Arm64-Readme.md) does not list HEIF or HEVC extensions, so successful WIC activation and fixture decoding remain **unverified**. For x64, standard `windows-latest` is Windows Server 2025, and the project's observed `0x80040154` HEIF activation failure shows that current image is not a qualified decode runner. GitHub's [runner image table](https://github.com/actions/runner-images) and [hosted runner reference](https://docs.github.com/en/actions/reference/runners/github-hosted-runners) establish the OS and architecture labels, not codec availability.

Microsoft documents the [HEIF WIC codec](https://learn.microsoft.com/en-us/windows/win32/wic/heif-codec) as a Store extension. Its [support guidance](https://support.microsoft.com/en-us/windows/apps/photos/photos-app-video-editor-error-can-t-view-this-file-type) says HEIF Image Extensions are free, HEVC Video Extensions are $0.99, and both are needed even for HEIC photos. That guidance concerns Photos, so it supports the expected package dependency, but the app's WIC tests must prove actual decode behavior. Do not treat a successful package listing alone as qualification.

## Options

| Route | What official sources establish | Confidence / unresolved issue |
| --- | --- | --- |
| Standard `windows-11-arm` | GitHub offers a native Windows 11 Arm64 hosted VM; its [image inventory](https://github.com/actions/runner-images/blob/main/images/windows/Windows11-Arm64-Readme.md) does not list codecs. | **High** confidence in platform; **unknown** HEIF/HEVC installation and WIC activation. Run a read-only probe first. |
| Standard `windows-latest` / `windows-2025` | Both select Windows Server 2025 x64. Its [inventory](https://github.com/actions/runner-images/blob/main/images/windows/Windows2025-Readme.md) does not list HEIF/HEVC, and this project's CI observed `0x80040154`. | **High** confidence the present image fails the required probe. A future image update is possible; re-probe by image version. |
| Install Store extensions during a hosted job | Microsoft's [`winget install`](https://learn.microsoft.com/en-us/windows/package-manager/winget/install) can target the `msstore` source and accept source/package terms; [Microsoft's Store troubleshooting](https://learn.microsoft.com/en-us/troubleshoot/windows-client/shell-experience/troubleshooting-microsoft-store-apps-download-failure) requires user registration and reachable Store endpoints. Windows Server 2025 with Desktop Experience [includes winget](https://learn.microsoft.com/en-us/powershell/scripting/install/installing-powershell-on-windows), but that does **not** establish codec Store eligibility or HEVC purchase entitlement. | **Low** confidence as unattended CI setup. Test only after confirming license, package availability, runner account, and Store connectivity. No official evidence here that the paid HEVC extension can be acquired on an ephemeral hosted VM. |
| Download and sideload MSIX/Appx | Microsoft documents [`winget download`](https://learn.microsoft.com/en-us/windows/package-manager/winget/download) for Store packages, but its offline license retrieval requires Microsoft Entra ID authentication and an administrator role. [App package deployment](https://learn.microsoft.com/en-us/windows-hardware/manufacture/desktop/sideload-apps-with-dism-s14?view=windows-11) distinguishes current-user registration from provisioning for users. | **Low** confidence and more moving parts; do not use an unofficial codec package mirror or assume `Add-AppxPackage` settles Store licensing. |
| GitHub-hosted larger runner with Windows 11 desktop or custom image | GitHub lists a partner [Base Windows 11 desktop image](https://docs.github.com/en/actions/reference/runners/larger-runners) and [custom image support](https://docs.github.com/en/actions/how-tos/manage-runners/larger-runners/use-custom-images). Larger runners require Team/Enterprise Cloud and billing; custom-image creation lists **Windows x64**, not Windows ARM64. | **Medium** confidence as a supported x64 platform, **unknown** codec presence and paid HEVC entitlement. Cost and image maintenance make this a later option. It does not directly solve ARM64 custom imaging. |
| Self-hosted Windows 11 x64/ARM64 with licensed extensions | GitHub [supports Windows self-hosted runners](https://docs.github.com/en/actions/how-tos/manage-runners/self-hosted-runners/add-runners), including service configuration. Microsoft's [package model](https://learn.microsoft.com/en-us/windows-hardware/manufacture/desktop/sideload-apps-with-dism-s14?view=windows-11) makes user registration distinct from machine provisioning. Microsoft's [WinGet troubleshooting](https://learn.microsoft.com/en-us/windows/package-manager/winget/troubleshooting) says the CLI is unsupported as LocalSystem; a [WinGet maintainer](https://github.com/microsoft/winget-cli/discussions/3953) says a user running in service session 0 is unsupported for its COM server. | **Medium** confidence as the most controllable codec-equipped route, provided the *runner process* can WIC-decode fixtures under its actual account. A logged-in interactive runner is the first qualification experiment; do not assume a service inherits that user's codec registration. GitHub [warns against persistent self-hosted runners for public PR code](https://docs.github.com/en/actions/reference/security/secure-use). |

## Future runner experiment (not selected for this CI change)

Run the existing WIC provider/fixture probe on `windows-11-arm` and on any candidate x64 Windows 11 desktop runner **under the exact job account**. Record `ImageOS`/`ImageVersion`, OS architecture, `Get-AppxPackage` results for both Microsoft extensions, WIC activation HRESULT, loaded provider identity, and actual HEIC fixture results. A package's presence or Photos behavior is insufficient. If Arm64 passes and x64 Server still fails, use Arm64 hosted coverage while separately qualifying x64 on a licensed Windows 11 desktop machine. A self-hosted machine should run only trusted, controlled jobs unless isolated/reset between runs; no runner, extension, or paid image was provisioned for this research.

## PicFetch evidence checked

At PR head `a00e98a12029bda24fc1b6dfb8df87eb45e2b8d9`, the sole unresolved
[review thread](https://github.com/frathe/picfetch/pull/50#discussion_r4077282462)
requires official codecs or a qualified runner. The repository's runner API
returned zero registered self-hosted runners on 2026-09-23.

The [Windows job](https://github.com/frathe/picfetch/actions/runs/35795206229/job/106972927211)
used Windows Server 2025, image `windows-2025-vs2026`, version
`20260907.229.1`. Its downloaded raw test artifact (`10723783570`) confirms
Microsoft HEIF activation failed with `0x80040154`. Failing packages were
`internal/heic`, `internal/imaging`, and `internal/similarity`; their failing
top-level tests were the native HEIC qualification/corpus/primary/WIC/analysis
tests. **The Store and visual-search steps were skipped after that failure**,
so this run supplies no result for either step.

The maintainer reports functional testing on Windows and Windows ARM in this
conversation (2026-09-23). That should be retained as manual evidence. Exact
commit, native executable architecture, codec versions, tested operations and
portable versus installed MSIX/Store context were not supplied. Earlier
[Windows x64 evidence](../plans/2026-09-22-system-heic.md#windows-results-and-evidence)
records successful HEIC corpus/consumer tests with HEIF `1.2.30.0` and HEVC
`2.4.43.0`, but also unrelated failures in that earlier complete native run.
It is not a passing final-head CI run or a native ARM64 qualification record.

## Minimal qualification commands

On a Windows checkout of the selected commit, using the same user account as
the intended runner, start with this PowerShell probe. Use native Go for each
architecture; record `go env GOARCH`, since an x64 process running on ARM does
not test the ARM64 binary. The Windows WIC package does not require cgo or
analysis model assets for this initial probe.

```powershell
git rev-parse HEAD
Get-CimInstance Win32_OperatingSystem | Select-Object Caption, Version, OSArchitecture
go env GOOS GOARCH CGO_ENABLED CC
Get-AppxPackage Microsoft.HEIFImageExtension | Select-Object Name, Version, Architecture
Get-AppxPackage Microsoft.HEVCVideoExtension | Select-Object Name, Version, Architecture
$env:PICFETCH_HEIC_NATIVE_TEST = '1'
go test -tags no_emoji,nodynamic -count=1 -v ./internal/heic -run '^TestHEICWindowsWICProbe$'
if ($LASTEXITCODE -ne 0) { throw 'Microsoft WIC codec probe failed.' }
```

Once the probe passes, use the existing commands below. They require the native
C toolchain used by the app and `CGO_ENABLED=1` for real analysis. Run on x64 and
native ARM64, retaining output with the commit, OS and provider details above.
The installer downloads the project's existing pinned analysis assets; it does
not install HEIF/HEVC. These commands are proposed and were not run on Windows
during this research.

```powershell
$env:GOFLAGS = '-tags=no_emoji,nodynamic'
$env:CGO_ENABLED = '1'
$env:PICFETCH_SIMILARITY_ASSETS = Join-Path $PWD '.scratch/native-guard-assets'
go run ./scripts/explorereval -install -assets $env:PICFETCH_SIMILARITY_ASSETS
if ($LASTEXITCODE -ne 0) { throw 'Native analysis asset setup failed.' }
go run ./scripts/nativeguards -suite windows -capture windows-native.json
$windowsExit = $LASTEXITCODE
go run ./scripts/nativeguards -suite store -capture store-native.json
$storeExit = $LASTEXITCODE
if ($windowsExit -ne 0 -or $storeExit -ne 0) { throw 'Native qualification failed; retain both reports.' }
```

The runner itself enables the native-test opt-in and rejects missing/skipped
required cases. A successful short WIC probe does not replace these suites.
If automated in Actions, retain both reports with `if: always()`. On a public
repository, avoid attaching a personal workstation to arbitrary pull-request
execution; use an isolated environment and explicitly controlled trusted runs,
following [GitHub's runner security guidance](https://docs.github.com/en/actions/reference/security/secure-use).

## Recorded dispositions and post-rollout checks

| Item | Disposition / retained check |
| --- | --- |
| Review finding and failing Windows CI | Apply the maintainer's explicit CI-only exception and obtain passing Windows/Store checks on the latest PR commit. Document the resulting native-codec coverage gap in the thread; complete the fresh review/check cycle after disposition. No thread was changed in this session. |
| Native Windows x64/ARM64 qualification | Closed by maintainer decision; testing deferred until after rollout. Retain native reports, executable architecture and actual provider identity when testing. |
| Packaged Store HEIC behavior | Closed by maintainer decision; test installed-package HEIC opens/associations, child workers and image consumers after rollout. |
| Missing-codec recovery | Closed by maintainer decision; test absence, installation/recheck and removal recovery after rollout. |
| Final signed-package qualification | Closed by maintainer decision; the existing automated WACK workflow remains in place. Additional qualification evidence is deferred. |

[`nativeguards -suite store`](../scripts/nativeguards/main.go) adds the
`microsoftstore` **build tag** to test binaries. It does not install or activate
an MSIX. The separate [Store packaging workflow](../.github/workflows/microsoft-store.yml)
builds/signs a test bundle and runs WACK, but contains no explicit packaged HEIC
open/worker scenario. Passing either gate alone therefore does not establish
the packaged behavior required by [AC8](heic-system-decoding.md).

After the investigation, the maintainer authorized the CI-only exception above.
Only runner tooling, its tests, workflow configuration and evidence documents
were changed. No codec was installed or redistributed; no Windows runtime
qualification, review-thread disposition, commit or push is claimed here.
