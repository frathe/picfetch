# 08: Decode Windows copied-file lists explicitly as UTF-8

**What to build:** Copy Windows file references containing accented, CJK and emoji names without changing their paths at the PowerShell boundary.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: resolved
Scope: required
Source: [Maintainability specification](../spec.md), AC12 / MA-012; plan work package F (clipboard encoding).
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [x] The BOM-less UTF-8 list is explicitly decoded at the Windows PowerShell consumer.
- [x] Execute the generated decoding boundary on Windows with accented, CJK and emoji temporary paths and assert exact decoded names; stub clipboard mutation.
- [x] Exercise an ANSI-default configuration or otherwise establish explicit decoding independently of the host's convenient UTF-8 settings.
- [x] Retain list cleanup, ordering and failure behavior; actual native test execution is required, not only script-string inspection.
- [x] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/clipboard -count=1 -v`

These commands are acceptance requirements; an unchecked ticket has not yet supplied implementation evidence. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

Requires native Windows PowerShell. Cross-compilation and skipped tests do not close this ticket.

## Completion boundary

This ticket closes only its named part of MA-012. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Published after the user requested `/implement sdd tdd` on the parent specification. Conditional and watch activation rules remain in force.

Implementation and portable guards complete; Windows native validation remains open. The consumer explicitly requests UTF8 and uses terminating PowerShell errors so read failure cannot report success. The builder regression failed before this change, and removal of either explicit encoding or error propagation fails the guard. `TestCopyFilesWindows_DecodesUTF8WithNonUTF8Default` runs the production command with only Set-Clipboard replaced, supplies a non-UTF8 default, and checks Unicode order, single selection, read failure and list cleanup. It cross-compiles; execution on Windows remains required. Native clipboard package and shared `make verify` pass, including all 625 Linux UI tests.

Resolved 2026-09-07: actual Windows 11 ARM64 execution now supplies the missing native evidence. Eight complete accepted package runs passed, 254 top-level executions and no skipped tests; the production validator accepted all 14 required Windows/Store guards. The wallpaper XDG parser correction was red on Windows before the fix, then green on Windows and macOS. The updater binary passed unchanged after moving its working directory from WebDAV to a local test folder. The follow-up `make verify` gate passed all 665 UI tests and remaining race packages; Windows/amd64 cross-vet also passed. See [native results](../evidence/25-native-guards.md#offline-windows-execution) and [common gate](../evidence/25-windows-followup-verify.txt).
