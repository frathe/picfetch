# 02: Bound mosaic preparation without changing composition

**What to build:** Generate a mosaic from extremely wide or tall source images under an explicit scratch-memory budget, preserving placement and edge quality.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: draft — awaiting breakdown approval
Scope: required
Source: [Maintainability specification](../spec.md), AC02 / MA-002; plan work package B.
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [ ] Declare and enforce an aggregate scratch budget including simultaneous resampling buffers and masks; check byte arithmetic before allocation and account for output/source/repeat-cache memory separately.
- [ ] Use observed allocation plans for wide and tall ratios through 10000, including the audited 1920x1080/defaults/seed-42 case; never attempt a real OOM.
- [ ] Ordinary seeded layouts, crop, aspect ratio, rotated-edge fidelity and repeat-cache behavior remain correct through the generation window.
- [ ] Cancellation stops further preparation; stale generation results cannot install.
- [ ] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/mosaic ./internal/ui/mosaicwin -count=1`

These commands are acceptance requirements for future implementation; they have not been run as proof that this draft is implemented. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Completion boundary

This ticket closes only its named part of MA-002. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Draft prepared for breakdown approval; no implementation or publication has occurred.
