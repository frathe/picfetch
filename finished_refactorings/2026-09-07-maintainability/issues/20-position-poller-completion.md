# 20: Make position-poller shutdown observable

**What to build:** Stopping window-position polling prevents queued native reads from updating a closed target and exposes actual worker completion.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: resolved
Scope: required
Source: [Maintainability specification](../spec.md), AC16 / MA-016; plan work package M.
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [x] Exercise the polling worker through an instance-owned native-read/queue seam, rather than a non-native-window no-op.
- [x] Cover stop before enqueue, after enqueue and during a gated native read; callbacks recheck cancellation before read and publication.
- [x] Cancellation requested and worker finished are distinguishable; cancelling from UI never waits for a callback needing that UI.
- [x] Integrate shutdown while the event loop can drain and retain harness completion for main and secondary windows.
- [x] Record a bounded native shutdown exercise; ticket 26's maintained GL procedure can be reused if already available.
- [x] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/winpos ./internal/ui/widgets ./internal/ui -count=1`
2. `make run`

These commands are acceptance requirements; an unchecked ticket has not yet supplied implementation evidence. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

The native run must record its OS, interaction and observed shutdown result; merely starting the app is insufficient.

## Completion boundary

This ticket closes only its named part of MA-016. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Published after the user requested `/implement sdd tdd` on the parent specification. Conditional and watch activation rules remain in force.

Shared verification passed: `make verify`, 644 Linux/amd64 UI tests across 212/204/228-test shards (325.882s/278.304s/260.193s), all remaining race packages, native vet/build and formatting/TUF/Qodana checks. Log: `/private/tmp/picfetch-maintainability-15-19-20-verify.log`. Lead review and negative guards are recorded in the active plan.

Native movement, secondary close/reopen and shutdown with Settings open passed on macOS 26.6.2 arm64. [Recorded native evidence](../evidence/20-native-poller-smoke.md).
