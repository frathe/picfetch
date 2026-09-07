# 14: Keep chooser admission and results on UI

**What to build:** Native file choosing remains background work while admission checks, comparison refusal and result handling run on UI.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: resolved
Scope: required
Source: [Maintainability specification](../spec.md), AC03 / MA-003; plan work package I (chooser admission).
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [x] Direct, menu and shortcut entry check comparison state on UI before external work.
- [x] Activate comparison or supersede/close the request while the chooser is held: revalidate its returned result on UI and discard obsolete delivery.
- [x] Worker code neither reads mode state nor paints a refusal toast directly; errors are reported once on the correct current request.
- [x] Cancellation and delayed result application remain observable without waiting cyclically on UI.
- [x] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/ui -count=1`

These commands are acceptance requirements; an unchecked ticket has not yet supplied implementation evidence. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

Independent from ticket 06's lossless path representation. Preserve the current open/drop path.

## Completion boundary

This ticket closes only its named part of MA-003. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Published after the user requested `/implement sdd tdd` on the parent specification. Conditional and watch activation rules remain in force.

## Implementation evidence

Lead-owned red/green, negative guard and contract review evidence is recorded in the [active plan](../../../plans/2026-09-06-maintainability-plan.md). The shared `make verify` gate passed formatting/TUF/Qodana, native vet/build, all 637 named Linux UI tests and every remaining race package. Log: `/private/tmp/picfetch-maintainability-12-14-verify.log`. New guards ran without skips. This completes tickets 12–14 and MA-003 in the working tree; no commit was created.
