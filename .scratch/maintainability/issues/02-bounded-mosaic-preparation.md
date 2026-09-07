# 02: Bound mosaic preparation without changing composition

**What to build:** Generate a mosaic from extremely wide or tall source images under an explicit scratch-memory budget, preserving placement and edge quality.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: resolved
Scope: required
Source: [Maintainability specification](../spec.md), AC02 / MA-002; plan work package B.
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [x] Declare and enforce an aggregate scratch budget including simultaneous resampling buffers and masks; check byte arithmetic before allocation and account for output/source/repeat-cache memory separately.
- [x] Use observed allocation plans for wide and tall ratios through 10000, including the audited 1920x1080/defaults/seed-42 case; never attempt a real OOM.
- [x] Ordinary seeded layouts, crop, aspect ratio, rotated-edge fidelity and repeat-cache behavior remain correct through the generation window.
- [x] Cancellation stops further preparation; stale generation results cannot install.
- [x] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/mosaic ./internal/ui/mosaicwin -count=1`

These commands are acceptance requirements; an unchecked ticket has not yet supplied implementation evidence. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Completion boundary

This ticket closes only its named part of MA-002. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Published after the user requested `/implement sdd tdd` on the parent specification. Conditional and watch activation rules remain in force.

## Answer

Implemented a 64 MiB live scratch budget with checked preparation plans, bounded destination tiles, clipped floating-point masks, and bounded additional SVG rasterization. The observer reproduced 9.98/10.03 GB raster requests and a 157 MB hidden resampling intermediate before the fix; negative full-only and cancellation overlays fail as expected without large allocation. Panoramic pixels and patterned enlargement/minification at 0, 7 and -12 degrees pass (maximum premultiplied channel difference 4/255, mean <= 0.1/255 versus retained full preparation). Both focused packages pass, and `make verify` passes with the Docker race suite and all 617 UI tests. Lead standards/spec review closed. The budget excludes output/layout and already-decoded/cache memory; in-progress SVG rasterization remains noninterruptible. No commit was made.
