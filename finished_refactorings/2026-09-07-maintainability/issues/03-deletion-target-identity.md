# 03: Reconcile confirmed deletions by file identity

**What to build:** Delete the confirmed files and reconcile only successful removals with the current Grid result even when its ordering or generation changes.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: resolved
Scope: required
Source: [Maintainability specification](../spec.md), AC04 / MA-004; plan work package C.
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [x] Prompt for A in [A,B], reorder to [B,A], then confirm: only A leaves temporary disk and the current list.
- [x] Cover fresh drops during prompting, generation changes during trash work, partial failure and overlapping confirmations; retain failed and unrelated files.
- [x] Snapshot targets so caller mutation cannot retarget the action; never interpret a stale index as the current target.
- [x] Use temporary-file trash stubs and assert actual file/list identity, with observable worker completion.
- [x] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/ui/deletion ./internal/ui -count=1`

These commands are acceptance requirements; an unchecked ticket has not yet supplied implementation evidence. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Completion boundary

This ticket closes only its named part of MA-004. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Published after the user requested `/implement sdd tdd` on the parent specification. Conditional and watch activation rules remain in force.

Resolution: Completed with immutable, deduplicated URI targets; successful moves reconcile against the current list, with per-operation results, queued completion and shutdown ownership. Temp-file identity, reorder/replacement, partial/overlapping moves, caller mutation, duplicate URIs, cache eviction and comparison/shutdown regressions pass. Deliberate identity/queue/cancellation violations failed their guards. Lead standards/spec review closed. Shared `make verify` passed, including 623 Linux UI tests and deletion under race. The unrelated native Copy Selection golden mismatch was reproduced against pre-03 source; Linux goldens pass.

Common gate log: `/private/tmp/picfetch-maintainability-03-05-verify.log`.
