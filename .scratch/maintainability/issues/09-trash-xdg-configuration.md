# 09: Respect ordinary XDG Trash configuration

**What to build:** Honor the user's normal Trash data directory while retaining the identified sandbox-redirection workaround.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: resolved
Scope: required
Source: [Maintainability specification](../spec.md), AC13 / MA-013; plan work package G (Trash).
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [x] Cover absent/default settings, a legitimate custom XDG_DATA_HOME, and identified sandbox redirection.
- [x] Preserve ordinary overrides; normalize only the demonstrated redirected environment.
- [x] Assert the adapter environment with OS operations stubbed and preserve package-specific environment semantics.
- [x] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/trash ./internal/wallpaper -count=1`

These commands are acceptance requirements; an unchecked ticket has not yet supplied implementation evidence. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Completion boundary

This ticket closes only its named part of MA-013. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Published after the user requested `/implement sdd tdd` on the parent specification. Conditional and watch activation rules remain in force.

Resolved in the working tree. `TestMoveLinux_PreservesOrdinaryXDGAndNormalizesSnap` reproduced blanket overriding through both gio and trash-put stubs. Absent, empty, default and legitimate custom settings now survive; only the identified canonical home/snap/app/revision/.local/share shape is normalized. Numeric revisions, current/common and unrelated near misses are covered; parent and unrelated environment values are unchanged. Trash/wallpaper suites pass natively (0.573s/0.799s) and under Linux race (1.061s/1.110s). Both over-normalizing custom paths and omitting real snap normalization fail negative overlays. Lead standards/spec review closed with no remaining findings. Shared `make verify` passes: formatting/TUF/Qodana, native vet/build, Linux Docker race suite and all 625 UI tests. Gate log: `/private/tmp/picfetch-maintainability-06-11-verify.log`. No commit created.
