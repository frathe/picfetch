# 11: Handle declining RSS in the optional HEIC check

**What to build:** Treat declining or unchanged RSS as no positive growth instead of unsigned-underflow evidence of a leak.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: draft — awaiting breakdown approval
Scope: required
Source: [Maintainability specification](../spec.md), AC19 / MA-019; plan work package H (RSS arithmetic).
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [ ] Add the specified deterministic TestRSSGrowth covering increasing, equal and decreasing uint64 samples.
- [ ] The arithmetic regression runs without enabling the long RSS experiment.
- [ ] Retain heicleak build selection, PICFETCH_HEIC_LEAK_TEST opt-in and the existing HEIC mitigation.
- [ ] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test -tags=heicleak ./internal/imaging -run '^TestRSSGrowth$' -count=1 -v`

These commands are acceptance requirements for future implementation; they have not been run as proof that this draft is implemented. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

Run on Linux. The named arithmetic test is to be added. The optional long experiment is not required for this arithmetic-only fix.

## Completion boundary

This ticket closes only its named part of MA-019. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Draft prepared for breakdown approval; no implementation or publication has occurred.
