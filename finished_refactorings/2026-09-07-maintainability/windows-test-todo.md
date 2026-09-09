# Windows test follow-up

Status: closed by user acceptance on 2026-09-09 (ticket 27 / MA-020).

The user reported: “I did test on W11 arm and w11 x64 and it looked good,”
and instructed that the remaining checks be considered done as edge cases.
The export-window checkbox was also separately reported to work on x64.
This is user-reported validation; artifact hashes, exact test coverage and a
WACK report were not supplied for these later tests.

The remaining detailed artifact checks and Windows SDK/WACK evidence are
waived for this audit closeout. This acceptance supersedes the earlier
deferral and open status. It does not assert that WACK passed or change the
release workflow's validation controls. No additional Windows tests, VM
experiments, package/trust changes or publication were performed for closeout.

The original checklist below is retained as historical procedure. Its former
checked boxes have been removed so they cannot imply per-step execution.
Earlier logs retain their original artifact and environment scope.

## Historical native x64 package checklist

- Use a native x64 Windows system with a working OpenGL driver. Record Windows
  build, CPU/GPU, graphics-driver version and display scaling.
- Test the ordinary and Microsoft Store-tagged x64 executables separately.
  The final Phase 6 binaries and Alpha/Beta fixtures are in
  `bin/maintainability-validation/` in the project on the Mac. Copy this folder
  to the test machine; `SHA256SUMS` records all four executables.
- Verify the executable SHA256 against
  [the refreshed inspected artifacts](evidence/phase6-windows-artifacts.txt):
  ordinary `cbae58455b8c6bf7263d7682744769a53f3351bca6f56c08024b7739a6913c6c`;
  Store `989acdc3f6cbeb4bfce54d1b14f2e51cee7b0e7160235d9a5aa3ecf2b4ed8bbc`.
- Launch each with the generated `01-alpha.png` and `02-beta.png` fixtures
  from that same folder. Check all four labeled corners, correct 8192x6144
  dimensions, then Right to Beta and its magenta marker.
- Open Grid (G), select both images (Ctrl+A), then compare (Ctrl+D). Check
  linked/unlinked pan and zoom (Ctrl+L), side-by-side/swipe switching and divider
  drag, and full-source detail (1). See the
  [illustrated procedure](evidence/26-native-comparison-smoke.md).
- With a larger cold list containing duplicates, enable Hide duplicates.
  Confirm completed groups disappear successively while the remaining images
  are still analyzed, then confirm navigation skips extras. On a machine with
  a small CPU budget, check navigation while favorite previews are generated.
- Quit each application and retain its exit code and stdout/stderr alongside
  screenshots. A process starting or a window appearing briefly is not a pass.
  If a failure occurs, retain the exact artifact hash and reproduction steps.

Use the machine's normal matching graphics driver for this native x64 test.
The isolated Mesa DLL and `x64-sse2.cmd` were VM diagnostics; neither is a
production packaging change or a prerequisite for testing on native x64.

## Historical Store packaging and certification checklist

- Use a disposable Windows x64 environment with MakeAppx, SignTool and WACK
  available. The existing ARM64 VM lacks the required SDK tool set.
- Refresh both x64/ARM64 staging directories from the verified executables
  with `scripts/msixstage`; the earlier temporary stages predate the final
  executable rebuild.
- Run the existing pack/bundle, temporary test-signature verification and
  WACK steps from `.github/workflows/microsoft-store.yml` (or the retained
  [offline helper](evidence/27-offline-wack.ps1.txt) after checking prerequisites).
  Retain package identity/architecture metadata, command outputs and WACK report.
- Confirm cleanup of only the test package/certificate/trust entries created
  by the run. Do not alter a pre-existing PicFetch installation.

No release, Store submission or publication is part of this checklist.

The refreshed binaries include the progressive-hiding fix and CPU-aware preview
budget. Prior startup/graphics results below belong to the earlier artifact
hashes; the new binaries were rebuilt and inspected, not run in the Windows VM.
The original procedure required Store staging to use these refreshed binaries as well.

## Existing evidence: do not repeat unnecessarily

- Eight native Windows test-package runs and all 14 required Windows/Store
  guards pass without skips: [results](evidence/25-native-guards.md).
- Ordinary and Store ARM64 executables render and quit cleanly in Windows 11;
  comparison interactions pass at DPI 96 / 100% monitor scale.
- The ARM64 VM's existing Desktop graphics DLLs are ARM64. x64 executables
  initially failed to create an OpenGL context.
- A verified isolated x64 Mesa runtime creates a context, but both executables
  then fail during initial drawing with illegal instruction `0xc000001d`.
  [Raw results](evidence/27-windows-x64-illegal-instruction/processes.json).
  Restricting Mesa to SSE2 was prepared but has not been tested; that experiment
  was deferred and is now waived for audit closeout. No graphics-library
  replacement, registry change or SDK/trust installation was performed.
