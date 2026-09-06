# 10: Invalidate favorite previews on subsecond edits

**What to build:** Refresh a favorite preview after a same-size source edit whose available modification timestamp differs within one second.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: draft — awaiting breakdown approval
Scope: required
Source: [Maintainability specification](../spec.md), AC18 / MA-018; plan work package H (preview identity).
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [ ] Different same-size bytes with different subsecond mtimes produce distinct preview identities.
- [ ] Old-format cache entries become misses and remain sweepable after a complete pass; normal cache hits still work.
- [ ] Cancelled passes retain unvisited previews.
- [ ] Identify filesystems lacking timestamp precision and cover the key/arithmetic contract with controlled inputs instead of silently skipping all evidence.
- [ ] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/favthumbs -count=1`

These commands are acceptance requirements for future implementation; they have not been run as proof that this draft is implemented. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Completion boundary

This ticket closes only its named part of MA-018. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Draft prepared for breakdown approval; no implementation or publication has occurred.
