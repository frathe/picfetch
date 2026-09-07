# 17: Cancel grid thumbnails and native-size backfill

**What to build:** Superseding or closing grid work cancels obsolete thumbnail reads, hash work and native-size backfill without admitting cancelled decode-slot waiters.

**Blocked by:** [16: Cancel favorite thumbnail reads through imaging](16-favorite-thumbnail-cancellation.md)

Status: resolved
Scope: required
Source: [Maintainability specification](../spec.md), AC10 / MA-010; plan work package J (grid).
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [x] Use ticket 16's context-bearing thumbnail APIs for production grid callers, including the cached-thumbnail/missing-native-size fallback.
- [x] Own a per-instance grid work context with defined cancellation on supersede/close; replace Background contexts at pool admission and read boundaries.
- [x] Respect Pool.Go's acquired=false callback behavior: release owned claims and expose completion without decoding or publishing cancellation as a source failure.
- [x] Preserve queue dispatch inside owning decode workers and the existing Settle wait/drain contract.
- [x] Controlled reads and slot waits prove cancellation; retain compatibility wrappers only where remaining callers still need them.
- [x] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/imaging ./internal/ui/grid ./internal/ui -count=1`

These commands are acceptance requirements; an unchecked ticket has not yet supplied implementation evidence. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

Ticket 18 separately makes generation admission atomic. This ticket must not publish a cancelled operation as a real failure, but cancellation alone is not proof against all generation races.

## Completion boundary

This ticket closes only its named part of MA-010. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Published after the user requested `/implement sdd tdd` on the parent specification. Conditional and watch activation rules remain in force.

Implemented and lead-reviewed. Full grid race suite passes (2.707s), viewer grid/duplicate/inspect/shutdown race selection passes (83.097s); 17 negative mutations are rejected. Shared 15–18 common gate remains pending.

Common gate passed: `make verify` (641 Linux/amd64 UI tests, remaining race packages, native vet/build, formatting/TUF/Qodana). Evidence: `/private/tmp/picfetch-maintainability-15-18-verify.log`. Lead review and negative guards are recorded in the active plan. No commit was created.
