# 26: Maintain and exercise production comparison GL smoke checks

**What to build:** Verify comparison's actual shader output and interactions in a documented, repeatable native desktop smoke procedure.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: resolved
Scope: required
Source: [Maintainability specification](../spec.md), AC17 / MA-017; plan work package R (production renderer).
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [x] Use temporary fixtures and record OS/GPU, standard/high display density, source sizes, shader compilation/output and screenshots.
- [x] Exercise linked/unlinked pan, zoom, rotation, side-by-side/swipe transitions and large-source detail against the intended geometry/fidelity.
- [x] Exercise close during work and record shutdown behavior; preserve deterministic canvas-reference and Linux golden tests.
- [x] Provide exact fixture/setup/launch/action/result instructions so another maintainer can repeat the run; app launch alone and headless shader-object tests are insufficient.
- [x] Record environment limitations explicitly, and retain any required unsupported-platform result as open instead of declaring full native coverage.
- [x] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `make run`
2. `go test ./internal/ui/compare -count=1`

These commands are acceptance requirements; an unchecked ticket has not yet supplied implementation evidence. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

Can start against the current production renderer. Each later lifecycle fix still needs its own native evidence; no circular blocking edge is introduced back to those fixes.

## Completion boundary

This ticket closes only its named part of MA-017. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Published after the user requested `/implement sdd tdd` on the parent specification. Conditional and watch activation rules remain in force.

Native progress (2026-09-07): [repeatable setup, actions, screenshots and limitations](../evidence/26-native-comparison-smoke.md) retain actual Retina GLSL output, linked/unlinked zoom, swipe endpoints, swap, canonical orientation and large-source detail. The native quit run reproduced an AppKit menu-thread crash; the existing shutdown integration regression was strengthened, five negative guards were rejected, and the fixed native comparison quit exits 0. The full common gate passed with all 665 Linux UI tests and remaining race packages. A temporary per-renderer held-work overlay confirms both tile calls cancel on Escape and on Cmd+Q, with native process exit 0. The subsequent user-operated ordinary Windows ARM64 package run passes linked/unlinked pan, divider dragging, linked wheel zoom and unlinked large-source detail. Returned process records identify PID 6672, window DPI 96 and clean exit 0 with empty stdout/stderr. The read-only environment report confirms Windows 11 Pro ARM64, VirtIO GPU DOD driver 22.7.38.43, 1728x1043 virtual monitor at 100% scale. This completes representative standard-density coverage alongside the earlier Mac Retina run. The repeated native setup/action/result procedure and limitations are retained. Ticket 26 and MA-017 are resolved; ticket 27 retains the unrelated x64 package startup/WACK/Linux requirements.
