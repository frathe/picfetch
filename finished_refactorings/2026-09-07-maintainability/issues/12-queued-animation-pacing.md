# 12: Pace animation after queued frame application

**What to build:** Play heterogeneous animation frame delays correctly when UI dispatch is delayed, and stop obsolete playback observably.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: resolved
Scope: required
Source: [Maintainability specification](../spec.md), AC03 / MA-003; plan work package I (animation).
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [x] A held per-instance UI queue proves the next delay follows the applied frame rather than the old frame.
- [x] Frame advancement has one goroutine owner with no worker/UI shared mutable locals.
- [x] Cancel while a callback or acknowledgement is pending: the worker stops without a UI-wait cycle and a later callback cannot update a new request.
- [x] Preserve animation pause/source capture for Copy Selection mode and register worker completion with the viewer harness.
- [x] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/ui -count=1`

These commands are acceptance requirements; an unchecked ticket has not yet supplied implementation evidence. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

Use the existing frame clock, request lifecycle and queue patterns; change the audit probe into a correct-pacing regression instead of preserving its expectation of premature scheduling.

## Completion boundary

This ticket closes only its named part of MA-003. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Published after the user requested `/implement sdd tdd` on the parent specification. Conditional and watch activation rules remain in force.

## Implementation evidence

Lead-owned red/green, negative guard and contract review evidence is recorded in the [active plan](../../../plans/2026-09-06-maintainability-plan.md). The shared `make verify` gate passed formatting/TUF/Qodana, native vet/build, all 637 named Linux UI tests and every remaining race package. Log: `/private/tmp/picfetch-maintainability-12-14-verify.log`. New guards ran without skips. This completes tickets 12–14 and MA-003 in the working tree; no commit was created.
