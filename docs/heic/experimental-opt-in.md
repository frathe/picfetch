# Experimental HEIC activation

Status: implementation in progress, 2026-09-16. The
[active plan](../../plans/2026-09-16-experimental-heic-opt-in.md) owns current
verification; [qualification](qualification.md) preserves earlier helper/source
evidence. This record is not release approval.

## Settings and installation

Experimental is the last Settings tab, including when Cache is present. General
opens by default. “Experimental HEIC support” persists as `experimentalHEIC`,
false when absent, and takes effect only at the next startup. Colors may be
inaccurate; HDR display remains unsupported. Still `.heic` and `.heif` are the
only added formats; sequence rejection and the absence of HEIC encoding remain.

Startup reads preferences before constructing the shared viewer/preview/analysis
owner. The saved checkbox, immutable session capability and latest unavailable
status are separate. Missing or invalid packages retain the preference and show
an explanation in Experimental. Ordinary image formats continue to work. No
alternative decoder, runtime download, elevation prompt or relaxed sandbox is
available as a fallback. Package-level format queries and OS associations remain
unchanged. Saved Explorer HEIC rules remain editable when HEIC is disabled.
The live file-size setting applies to each new read through that reader. The
shared owner's fixed 64 MiB HEIC input ceiling still bounds every request, even
when the user chooses a higher limit for ordinary formats.

Development builds need the complete helper package. `go run .` or copying only
the main executable does not supply it. On macOS use `make package-mac`: the
running binary must be in `PicFetch.app/Contents/MacOS`, with the independently
signed helper in the fixed nested bundle. Linux and Windows require the `heic`
directory next to the main executable, with the architecture-matched helper,
manifest and notices produced by `scripts/heicpackage`. Preserve that directory
when assembling an installed package; packaging alone leaves the preference off.
Windows uses a verified private copy under the application's cache and never
changes immutable Store files or their permissions. Client shutdown releases
copy leases only after work joins; active obsolete copies defer cleanup.

The first transition from the released updater still requires a complete
reinstall or a separately qualified bridge. That updater cannot preserve the
new companion/helper package, and on macOS can invalidate the enclosing
signature. This feature does not qualify or implement that bridge.

## Native evidence commands

- `make heic-native-macos`: native Intel or Apple Silicon, independently signed
  helper and enclosing test app, shared application startup and native decode.
- `make heic-native-linux`: native amd64 or arm64; retains no-cgo helper seccomp
  and finite resource tests, then requires application startup and admission.
- `make heic-native-windows HEIC_MSIX_CONFIGURATION=<owned-config.json>`: run as
  a standard user; executes helper/cache/application guards, then installs and
  activates the disposable test-MSIX described by that configuration.

Each native suite also requires `TestNativeHEICAnalysisPixels`: successfully
decoded pixels reach an analysis subprocess and two preview queries in one
retained search subprocess through the shared owner, with cancellation/join
checks. This test uses the owned HEIC fixture and does not run model inference.

The application fixture is `TestNativePackagedHEICActivation` under `heicnative`.
It relocates the application test executable to the fixed package layout and
runs the shared startup/viewer harness. It exercises the real checkbox across
separate viewer lifetimes, HEIC/HEIF admission, native helper decoding, ordinary
viewing, cancellation and shutdown. Fyne's test driver supplies the surface;
this is application-constructor/package-boundary evidence, not a production GUI
smoke test. Its resumed coverage adds mixed-directory navigation, a foreground
source held after native readiness across a saved Settings edit and collection
replacement, and ordinary viewing after missing/invalid/wrong-target package
inputs. macOS also exercises an owned helper signed without App Sandbox: native
readiness refuses it before bulk source reads, and ordinary viewing recovers.
All damaged-package fixtures are disposable standalone copies; installed MSIX
files remain immutable. These additions pass locally on macOS arm64 with race
detection; the current plan records subsequent cross-platform CI results.
The installed-MSIX entry is `TestNativeInstalledHEICActivation`,
compiled with `heicnative,microsoftstore` alongside the actual Store executable.
The fixture launches the declared test executable directly from its installed
WindowsApps path. Windows can resolve package identity during this ordinary
process launch, as described by a [Microsoft Terminal maintainer](https://github.com/microsoft/terminal/discussions/20060).
It requires actual package identity, Store-managed behavior, a nonadministrator
token and unchanged installed helper bytes/ACLs.

CI's `packaging/heic/qualify-windows.ps1` provisions a disposable local standard
account on a GitHub-hosted runner. For MSIX it adds a separate test application,
uses a disposable signing certificate, and records setup separately from
application execution. The child replaces inherited runner profile variables
with the loaded standard user's native environment via
[CreateEnvironmentBlock](https://learn.microsoft.com/en-us/windows/win32/api/userenv/nf-userenv-createenvironmentblock).
`qualify-windows-child.ps1` performs the unelevated work
and removes its test package. The parent removes its account, owned workspace
and certificate/trust entry. Neither script is called by PicFetch. CI runs both
architectures for standalone and installed-MSIX scenarios. Runtime or permission
query failures fail the gate; there is no successful skip or elevated substitute.
At `69fef1a`, standalone standard-user qualification passes on both architectures.
Installed-MSIX activation remains blocked: IApplicationActivationManager returned
0x80070520, and ordinary installed-executable launch returns Access denied with
either the installed directory or the owned writable workspace as its working
directory. Both packages install successfully; their test process never starts.
Microsoft requires an [interactive user for packaged application execution](https://learn.microsoft.com/en-us/windows/msix/desktop/desktop-to-uwp-debug).
Qualification now needs native x64 and ARM64 environments with an interactive
standard-user session. Actual package identity and every existing guard must
pass there before this gate is complete. The hosted alternate-user fixture
does not establish installed Store helper activation, staging or sandbox behavior.
For a local MSIX run, the configuration supplies `Scenario: "msix"`, `Repository`,
`Go`, `Work`, `Evidence`, `Arch`, `Commit`, the signed `Package`, its `PackageName`
and architecture-matched `Dependency`. Provision only disposable test state.

The Windows cache uses Win32 sharing restrictions to retain leased executables
and serialize publishers; copied bytes are checked before publication and again
before launch. The underlying semantics are documented in
[CreateFile](https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-createfilew).
Native execution remains necessary to qualify ACLs, concurrent publication and
Store identity; cross-compilation is not that evidence.

## Current verification and remaining gates

At `c1b6890`, native Linux, macOS and standalone standard-user Windows pass on
both architectures on first attempt, including the expanded application
failure/navigation cases. All four Linux race partitions, ordinary Windows,
validation and Store input construction pass. Installed-MSIX remains the only
failing scenario in CI 35120395881: both packages install, but the test process
cannot start (Access denied). Standard-user desktop-session prerequisites and
Windows combined application-level ACL/query/concurrent-lifetime cases remain
open. The old intermittent ARM64 analysis failure did not recur; no root-cause
fix is claimed.

Fresh Codex code/security reviews report no findings for the implementation
commit. Final Qodana SARIF is empty; CodeQL contains only its two previously
assessed dismissed false positives. GitHub's separate dynamic AI scanner still
fails before analysis because its service rejects the configured model. That
scan remains unverified. Production GUI smoke tests remain distinct from the
test-driver fixture. See the plan and PR #28 for checkpoint and subsequent
documentation-only review/CI evidence.

All prior sandbox, integrity and resource controls remain. macOS still has the
previously accepted absence of a guaranteed total native-memory cap; WASM,
input/output and deadline limits remain mandatory. Decoder source and dependency
versions are unchanged. Distribution/licensing clearance (including patent and
source traceability questions), production signing and broader camera/color
qualification remain release gates. Successful decoding is not color fidelity.
