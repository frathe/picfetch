# 04: Construct complete shared image-cache records

**What to build:** Show consistent size, EXIF information and canonical pixels regardless of whether foreground loading, preloading or comparison first cached an image.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: resolved
Scope: required
Source: [Maintainability specification](../spec.md), AC06 / MA-006; plan work package D.
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [x] All three paths construct equivalent file-size/EXIF facts for the same encoded bytes, including a GPS JPEG.
- [x] Comparison-first normal navigation exposes the real size and EXIF affordance in the actual UI tree.
- [x] Preserve EXIF-corrected pixels, RAW preview markers, original animation frames, encoded-size limits and stale-request checks.
- [x] Preserve explicit foreground Add versus speculative AddIfFits behavior and separate thumbnail/full-image cache budgets.
- [x] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/ui ./internal/imaging -count=1`

These commands are acceptance requirements; an unchecked ticket has not yet supplied implementation evidence. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Completion boundary

This ticket closes only its named part of MA-006. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Published after the user requested `/implement sdd tdd` on the parent specification. Conditional and watch activation rules remain in force.

Resolution: Completed through imaging.DecodeRecord, used by foreground, preload and comparison. TestImageCacheWriters_PreserveCompleteRecords passes across ordinary/GPS/oriented JPEG, RAW and animated GIF; TestCompareFirstNavigation_ShowsCachedSizeAndEXIF verifies real bytes and the EXIF link in the window tree after comparison populated the cache. Original comparison source fails both guards. Cache/decode/fidelity regressions and full imaging pass; Add/AddIfFits, read limits and caller staleness checks are preserved. Lead standards/spec review closed; shared `make verify` passed (623 Linux UI tests plus other race packages).

Common gate log: `/private/tmp/picfetch-maintainability-03-05-verify.log`.
