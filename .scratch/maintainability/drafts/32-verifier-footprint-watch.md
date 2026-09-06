# 32: Measure verifier footprint at its next major upgrade

**What to build:** At a future authorized major verifier upgrade, measure dependency cost and retain the update trust/provenance contract through any justified reduction.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: draft — awaiting breakdown approval
Scope: watch
Source: [Maintainability specification](../spec.md), AC24 / MA-024; plan work package Accepted watch.
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [ ] Activation requires a separate major verifier-upgrade task; no immediate dependency-removal work or recurring monitor is requested.
- [ ] Record reachable dependencies, build time and binary size with platform and cache conditions.
- [ ] Consider only supported reductions preserving signature/provenance validation, traversal/symlink defenses, download bounds and rollback ordering.
- [ ] Retain the existing verifier if no supported improvement is established; any actual change passes updater and common gates and records before/after evidence.
- [ ] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go list -deps ./internal/update`
2. `time make build`
3. `wc -c bin/picfetch`
4. `go test ./internal/update ./internal/ui/autoupdate -count=1`

These commands are acceptance requirements for future implementation; they have not been run as proof that this draft is implemented. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

Activation: next separately authorized major verifier upgrade. No blockers means no numbered prerequisite, not immediate authorization to alter dependencies.

## Completion boundary

This ticket closes only its named part of MA-024. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Draft prepared for breakdown approval; no implementation or publication has occurred.
