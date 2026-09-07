# 10: Invalidate favorite previews on subsecond edits

**What to build:** Refresh a favorite preview after a same-size source edit whose available modification timestamp differs within one second.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: resolved
Scope: required
Source: [Maintainability specification](../spec.md), AC18 / MA-018; plan work package H (preview identity).
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [x] Different same-size bytes with different subsecond mtimes produce distinct preview identities.
- [x] Old-format cache entries become misses and remain sweepable after a complete pass; normal cache hits still work.
- [x] Cancelled passes retain unvisited previews.
- [x] Identify filesystems lacking timestamp precision and cover the key/arithmetic contract with controlled inputs instead of silently skipping all evidence.
- [x] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/favthumbs -count=1`

These commands are acceptance requirements; an unchecked ticket has not yet supplied implementation evidence. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Completion boundary

This ticket closes only its named part of MA-018. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Published after the user requested `/implement sdd tdd` on the parent specification. Conditional and watch activation rules remain in force.

Resolved in the working tree. `TestEntryNamePreservesSubsecondPrecision`, `TestReadMissesSameSizeSubsecondEdit` and `TestSyncReplacesLegacyPreviewAfterCompletePass` reproduced whole-second collisions, stale actual preview reuse and accepted legacy entries. The host filesystem preserves .1s and .9s in one second, with same-size changed bytes. Separate seconds/nanoseconds fields retain precision outside UnixNano range. Legacy keys miss, complete sync sweeps them, and cancellation preserves unvisited entries. Full favthumbs passes natively (1.144s) and under Linux race (1.491s); timestamp and premature-sweep mutations fail the guards. Lead standards/spec review closed with no remaining findings. Shared `make verify` passes: formatting/TUF/Qodana, native vet/build, Linux Docker race suite and all 625 UI tests. Gate log: `/private/tmp/picfetch-maintainability-06-11-verify.log`. No commit created.
