# 21: Move whole-image clipboard encoding off UI

**What to build:** Copy the captured displayed image while input and redraw remain responsive throughout PNG encoding and clipboard dispatch.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: resolved
Scope: required
Source: [Maintainability specification](../spec.md), AC14 / MA-014; plan work package N (clipboard).
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [x] Gate PNG encoding before OS dispatch and show that an unrelated queued UI interaction still completes.
- [x] Capture the intended image/pixels and own a cancellable request with stale-result checks, including navigation or close during encoding.
- [x] Error and success effects are applied once on UI; completion includes the promised callback effects and new work is included in harness drain.
- [x] Preserve Grid selection and image-region selection routing and existing image-copy failure reporting; tests use clipboard stubs.
- [x] Define overlapping clipboard requests explicitly so an older completion cannot close the wrong operation's signal or overwrite newer presentation.
- [x] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/ui ./internal/clipboard -count=1`

These commands are acceptance requirements; an unchecked ticket has not yet supplied implementation evidence. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

Ticket 07 is a recommended independent fix to error fidelity, not a prerequisite for moving encoding. Preserve its primary-error contract whenever present; do not expand this task into a global worker registry.

## Completion boundary

This ticket closes only its named part of MA-014. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Published after the user requested `/implement sdd tdd` on the parent specification. Conditional and watch activation rules remain in force.

## Implementation evidence

The actual PNG-read tracer reproduced UI blocking (1.331s); navigation reproduced a cancelled OS dispatch (2.092s), and competing grid/region copies replaced the active Signal (4.501s). Per-viewer captured pixels, cancellable encoding and one-operation admission now pass the focused UI/clipboard race selection (72.961s/1.280s). Eight named Clipboard/Shutdown regression tests cover literal pixels, queued error/current/stale delivery, output cancellation, retries and actual OS-worker completion. Fourteen deliberate violations were rejected; `/private/tmp/picfetch-maintainability-21-negative.log`. Clipboard dispatch was stubbed throughout.

Shared `make verify` passed formatting/TUF/Qodana, native vet/build and all Linux race partitions: 653 UI tests (212/213/228), shard times 332.664s/287.301s/255.054s. Log: `/private/tmp/picfetch-maintainability-21-22-verify.log`. No commit was created. MA-014 remains open for ticket 23.
