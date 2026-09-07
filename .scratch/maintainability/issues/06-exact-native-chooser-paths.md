# 06: Preserve exact native chooser paths

**What to build:** Open exactly the selected files and save to exactly the confirmed destination, preserving every path boundary supported by each native backend.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: resolved
Scope: required
Source: [Maintainability specification](../spec.md), AC05 / MA-005; plan work package F (chooser transport).
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [x] Keep one exact save destination distinct from multi-open results; preserve legal POSIX embedded newlines, trailing CR, spaces, Unicode and selection order.
- [x] Cover cancellation, empty selection and execution errors distinctly; unsupported transport fails explicitly before a path is used.
- [x] Exercise the actual changed transport on its native platform, including the macOS bridge, using temporary paths and no user-file overwrite.
- [x] Validate chooser-to-open/export behavior and maintain existing backend support; inspect Windows chooser stdout separately from the clipboard encoding defect.
- [x] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/filepicker -count=1 -v`
2. `go test ./internal/ui -run 'Test.*(Chooser|OpenFile|ExportAs)' -count=1 -v`

These commands are acceptance requirements; an unchecked ticket has not yet supplied implementation evidence. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

Run the package tests on every changed platform and record actual native transport cases. Ticket 14 concerns thread admission and can land independently.

## Completion boundary

This ticket closes only its named part of MA-005. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Published after the user requested `/implement sdd tdd` on the parent specification. Conditional and watch activation rules remain in force.

Implementation and lead standards review complete; Windows native validation remains open. Darwin NSURL transport and Linux Zenity transport pass exact path/cancellation cases, including replay through the production decoder. See [native evidence](../evidence/README.md). Viewer and mosaic exports preserve the confirmed destination while the selected encoder handles an unsupported/absent suffix. Full picker/imaging/mosaic and chooser UI tests pass; 17 shared negative overlays reject the relevant guards. `make verify` passes all 625 Linux UI tests. `TestWindowsPickerTransport_EmitsUTF8PathArrays` cross-compiles; no Windows runtime has been supplied yet.

Resolved 2026-09-07: actual Windows 11 ARM64 execution now supplies the missing native evidence. Eight complete accepted package runs passed, 254 top-level executions and no skipped tests; the production validator accepted all 14 required Windows/Store guards. The wallpaper XDG parser correction was red on Windows before the fix, then green on Windows and macOS. The updater binary passed unchanged after moving its working directory from WebDAV to a local test folder. The follow-up `make verify` gate passed all 665 UI tests and remaining race packages; Windows/amd64 cross-vet also passed. See [native results](../evidence/25-native-guards.md#offline-windows-execution) and [common gate](../evidence/25-windows-followup-verify.txt).
