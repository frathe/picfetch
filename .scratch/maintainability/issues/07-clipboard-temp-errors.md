# 07: Preserve clipboard temporary-file errors

**What to build:** Report the original temporary PNG write or close error rather than apparent success after cleanup.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: resolved
Scope: required
Source: [Maintainability specification](../spec.md), AC11 / MA-011; plan work package G (clipboard).
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [x] Test failed writes and failed closes with successful removal; the original error remains the returned primary cause.
- [x] When cleanup also fails, preserve an inspectable primary cause, optionally joining the cleanup error.
- [x] Success returns a usable path and leaves existing clipboard behavior intact.
- [x] Fault tests reach the package helper through a local/instance seam or isolated subprocess and never change the real desktop clipboard.
- [x] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/clipboard -count=1`

These commands are acceptance requirements; an unchecked ticket has not yet supplied implementation evidence. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Completion boundary

This ticket closes only its named part of MA-011. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Published after the user requested `/implement sdd tdd` on the parent specification. Conditional and watch activation rules remain in force.

Resolved in the working tree. The fault-injected write/close transaction first lost all primary causes (six red cases including short writes). `TestWriteTempPNGFile_PreservesPrimaryFailures` now checks errors.Is, ordered close/removal, cleanup failures and actual temp-file removal. The full package passes natively (0.325s) and in the common race gate (1.072s). Removing the primary error or short-write guard with overlays makes the regression fail. Lead standards/spec review closed with no remaining findings. Shared `make verify` passes: formatting/TUF/Qodana, native vet/build, Linux Docker race suite and all 625 UI tests. Gate log: `/private/tmp/picfetch-maintainability-06-11-verify.log`. No commit created.
