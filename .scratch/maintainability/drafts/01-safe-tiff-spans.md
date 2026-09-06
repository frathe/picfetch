# 01: Reject malformed TIFF spans safely

**What to build:** Open malformed EXIF-bearing images and RAW previews without a panic while retaining valid metadata and orientation.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: draft — awaiting breakdown approval
Scope: required
Source: [Maintainability specification](../spec.md), AC01 / MA-001; plan work package A.
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [ ] Check root, nested and next-IFD offsets, entry/value overflow, near-end spans, truncation and MaxUint32 in both byte orders before slicing.
- [ ] Valid metadata, orientation and RAW embedded previews retain their current results and tolerant error policy; reader and writer malformed-block policies stay distinct.
- [ ] Add the specified FuzzTIFFReaders target with deterministic seeds for the affected reader entry points; record the reproduced failure before the fix and the bounded fuzz result.
- [ ] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/imaging -count=1`
2. `go test ./internal/imaging -run '^$' -fuzz '^FuzzTIFFReaders$' -fuzztime=30s`

These commands are acceptance requirements for future implementation; they have not been run as proof that this draft is implemented. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Completion boundary

This ticket closes only its named part of MA-001. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Draft prepared for breakdown approval; no implementation or publication has occurred.
