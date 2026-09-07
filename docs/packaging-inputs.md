# Reviewed packaging inputs

`packaging/tools.mk` owns the Fyne CLI and fyne-cross versions, the Windows and
Linux container image digests, and the common engine/cache defaults. Local
Make targets, the release workflow and the Store workflow use these same inputs.
Tools install into version-specific, ignored `.tools/` directories; a globally
installed CLI does not select the version used by these targets.

The image digests name multiarchitecture indexes, supporting amd64 and arm64
build hosts. An image upgrade must retain both host architectures. The container
contains its own Fyne CLI: pinning the host CLI alone does not pin cross-packaging.

Every package target reports the actual host CLI's Go module metadata.
Cross-package targets also report the resolved local image ID, repository
digests, OS/architecture, container Go version and container Fyne CLI version.
Warmup uses the same UID, image and cache as packaging, with `GOTOOLCHAIN=auto`
so the project's Go requirement takes precedence over the image's bundled Go.
Container, compiler and artifact-copy failures stop the target immediately.

## Reviewing an upgrade

1. Change the versions/digests in `packaging/tools.mk`. Inspect each image's
   multiarchitecture index and the CLI/toolchain inside the actual selected
   image. Retain the inspection output with the build logs.
2. Run `go test ./scripts/msixstage ./scripts/plistdoctypes -count=1`. The guards
   execute all ordinary/debug/Store cross routes with controlled external
   tools, require provenance and distribution flags, and reject failed builds
   or artifact copies. Workflow guards require shared installation targets.
3. Copy the complete proposed source into a disposable checkout. Run
   `make install-tools`, `make package-mac`, then
   `make package-windows package-windows-store package-linux`, retaining output.
   macOS packaging runs natively; the cross builds require Docker. Avoid
   simultaneous package targets sharing generated Fyne resources/output paths.
4. Inspect the actual outputs with `file` and `go version -m`, the macOS
   `Info.plist`, and staged MSIX manifests. Confirm the requested architectures,
   app identity, version, file associations and Store-only build tag. Keep
   `appID`, package ID and Store identity unchanged unless separately approved.
5. Launch the built artifacts on macOS, Windows and Linux. Record the OS,
   architecture, graphics driver, observed image rendering and clean shutdown.
   A cross-build or `--help` run alone is not native graphical startup evidence.
   For the existing Windows 11 test VM, place a distinctly named executable on
   its Desktop alongside its GL libraries, preserving the existing executable
   and DLLs.
6. On Windows, stage both Store executables with `scripts/msixstage` and execute
   the existing MakeAppx pack/bundle, temporary test-signature verification and
   WACK steps from `.github/workflows/microsoft-store.yml`. Retain the bundle
   metadata and certification report. Use a disposable Windows environment for
   temporary certificate trust. See [Store packaging](microsoft-store.md).
7. Run `make verify` before handoff. Record any unavailable native or SDK check
   as open; do not substitute dry-run output or a skipped test.

Image, engine and cache overrides remain available for deliberate experiments
and are visible in the command/provenance output. An override's evidence applies
to that override, not automatically to the reviewed defaults. Package validation
does not authorize a push, release, workflow publication or Store submission.
Pins make input changes reviewable; timestamped or signed artifacts are not
promised to be byte-identical.
