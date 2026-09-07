# 30: Consolidate repeated command admission when routes are touched

**What to build:** Preserve command behavior across menus, keyboard and direct entry while simplifying repeated admission decisions in touched routes.

**Blocked by:** [12: Pace animation after queued frame application](12-queued-animation-pacing.md); [13: Reject late picture-frame advances](13-queued-picture-frame-advances.md); [14: Keep chooser admission and results on UI](14-chooser-ui-admission.md)

Status: ready-for-agent
Scope: conditional
Source: [Maintainability specification](../spec.md), AC22 / MA-022; plan work package Q.
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [ ] Capture the current matrix for comparison, Copy Selection mode, Grid View, inspect and picture-frame mode before consolidating.
- [ ] Test Escape priority, menu/keyboard/direct-entry parity and intentional exceptions, including comparison Help and Open refusal.
- [ ] Centralize only repeated decisions when that reduces duplication; preserve feature state ownership and explicit composition.
- [ ] Retain a tested matrix without forcing consolidation where it does not simplify the touched routes; do not reopen the separate chooser-thread defect.
- [ ] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/ui ./internal/ui/menus -count=1`

These commands are acceptance requirements; an unchecked ticket has not yet supplied implementation evidence. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

Activation: only when these routes are selected for adjacent work. Blockers retain the supplied plan's I-before-Q ordering; they are the three parts of I, not a new universal mode framework.

## Completion boundary

This ticket closes only its named part of MA-022. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Published after the user requested `/implement sdd tdd` on the parent specification. Conditional and watch activation rules remain in force.
