# 13: Reject late picture-frame advances

**What to build:** Leaving picture-frame mode prevents queued advances from changing the closed or subsequent session.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: draft — awaiting breakdown approval
Scope: required
Source: [Maintainability specification](../spec.md), AC03 / MA-003; plan work package I (picture-frame mode).
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [ ] Hold an advance callback, exit or restart the mode, then release it and assert no stale image advance.
- [ ] Recheck request generation at UI application and remove shared worker/UI stale-result locals.
- [ ] Cancellation stops a worker waiting on queued work without blocking the event loop; completion is observable.
- [ ] Preserve interval, shuffle, mode-exit and geometry-restoration behavior.
- [ ] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/ui/slideshow ./internal/ui -count=1`

These commands are acceptance requirements for future implementation; they have not been run as proof that this draft is implemented. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

Independent from ticket 12: the owning state machines are separate, even though the queue-testing pattern is useful to both.

## Completion boundary

This ticket closes only its named part of MA-003. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Draft prepared for breakdown approval; no implementation or publication has occurred.
