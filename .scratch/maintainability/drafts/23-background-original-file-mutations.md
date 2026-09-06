# 23: Run original-file mutations in serialized background work

**What to build:** Save Changes and metadata removal leave the UI responsive and cannot overlap unsafely on the same original source.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: draft — awaiting breakdown approval
Scope: required
Source: [Maintainability specification](../spec.md), AC14 / MA-014; plan work package N (Save Changes and metadata removal).
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [ ] Gate Save Changes and metadata removal independently while input/redraw continue; capture source identity and intended pixels before background work.
- [ ] Serialize conflicting operations across their complete read-transform-write transaction using the resolved source identity, including symlink aliases; atomic rename alone is not transaction serialization.
- [ ] Ensure existing write routes participate when they can target that same resolved file, including an Export destination aliasing the source; preserve independent-file concurrency and avoid a universal task registry.
- [ ] Preserve permissions, atomic replacement, correct existing image-record/dimension policy, and error behavior; a failed save retains its rotation state.
- [ ] Cover cancellation before write and completion after replacement committed; navigation/close cannot retarget writes or install old UI state and must not pretend to undo a completed disk mutation.
- [ ] Refresh/invalidate the appropriate caches and current metadata/info once after success; expose all workers and causal UI completion to shutdown/harness drain.
- [ ] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/ui ./internal/ui/exifwin ./internal/imaging -count=1`

These commands are acceptance requirements for future implementation; they have not been run as proof that this draft is implemented. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

These two mutation actions stay together because making either asynchronous creates a real shared-source race. Tickets 04/05/12-17/22 are useful precedents and their established contracts must be preserved; they do not supply a required new interface that blocks starting this slice.

## Completion boundary

This ticket closes only its named part of MA-014. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Draft prepared for breakdown approval; no implementation or publication has occurred.
