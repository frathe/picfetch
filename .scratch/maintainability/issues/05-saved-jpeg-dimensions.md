# 05: Save truthful rotated JPEG dimensions

**What to build:** Save Changes writes JPEG dimension metadata that agrees with the encoded frame after rotation and orientation normalization.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: resolved
Scope: required
Source: [Maintainability specification](../spec.md), AC07 / MA-007; plan work package E.
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [x] Replace the old leave-dimensions-alone assertion with numeric IFD0, Exif and Interop dimension checks for both quarter turns and orientations 5-8.
- [x] Correct representable dimension tags; remove tags that cannot be made true and invalidated coordinate tags under the existing dimension policy.
- [x] Retain unrelated camera metadata, DPI, color profiles and unchanged-geometry behavior; preserve tolerant malformed-input fallback.
- [x] Exercise the viewer Save Changes path and retain its failure behavior; explicitly document the superseded metadata policy.
- [x] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/imaging -run 'Test.*(SaveRotated|Export|JPEG|Exif|EXIF)' -count=1 -v`
2. `go test ./internal/ui -run 'Test.*Save' -count=1 -v`

These commands are acceptance requirements; an unchecked ticket has not yet supplied implementation evidence. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

Ticket 01 is useful prior work, not a blocker: reuse the current checked writer boundary without waiting for reader hardening.

## Completion boundary

This ticket closes only its named part of MA-007. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Published after the user requested `/implement sdd tdd` on the parent specification. Conditional and watch activation rules remain in force.

Resolution: Completed by reusing the existing dimension-invalidated policy in SaveRotated. The former presence-only/leave-dimensions-alone assertion is superseded by TestSaveRotated_CorrectsDimensionTags: numeric IFD0/Exif/Interop dimensions match both quarter turns and orientations 5–8, with ICC and unrelated camera/DPI metadata retained. Unchanged geometry, malformed IFDs, unreadable-header fallback and viewer write failure are covered. TestSaveRotation_CorrectsJPEGDimensionValues exercises actual Save Changes. Original source and incorrect fallback overlays fail their guards. Lead standards/spec review closed; shared `make verify` passed (623 Linux UI tests plus other race packages).

Common gate log: `/private/tmp/picfetch-maintainability-03-05-verify.log`.
