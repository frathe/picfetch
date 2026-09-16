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
smoke test. The installed-MSIX entry is `TestNativeInstalledHEICActivation`,
compiled with `heicnative,microsoftstore` alongside the actual Store executable.
Windows activates its declared test application through
[IApplicationActivationManager](https://learn.microsoft.com/en-us/windows/win32/api/shobjidl_core/nn-shobjidl_core-iapplicationactivationmanager).
It requires actual package identity, Store-managed behavior, a nonadministrator
token and unchanged installed helper bytes/ACLs.

CI's `packaging/heic/qualify-windows.ps1` provisions a disposable local standard
account on a GitHub-hosted runner. For MSIX it adds a separate test application,
uses a disposable signing certificate, and records setup separately from
application execution. `qualify-windows-child.ps1` performs the unelevated work
and removes its test package. The parent removes its account, owned workspace
and certificate/trust entry. Neither script is called by PicFetch. CI runs both
architectures for standalone and installed-MSIX scenarios. Runtime or permission
query failures fail the gate; there is no successful skip or elevated substitute.
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

Focused preference/Settings/admission/Explorer and helper-publication tests pass
locally. Native macOS arm64 application-constructor activation passes with the
unchanged helper entitlement and finite limits. Windows amd64/arm64 test binaries
compile. Native Windows, installed-MSIX, Linux and macOS amd64 results for these
new paths remain unverified until their new CI jobs execute. Production GUI
smoke tests remain distinct from the test-driver fixture. See the plan for exact
commands, recorded red/green results and inspections as they complete.

All prior sandbox, integrity and resource controls remain. macOS still has the
previously accepted absence of a guaranteed total native-memory cap; WASM,
input/output and deadline limits remain mandatory. Decoder source and dependency
versions are unchanged. Distribution/licensing clearance (including patent and
source traceability questions), production signing and broader camera/color
qualification remain release gates. Successful decoding is not color fidelity.
