# 26: Maintain and exercise production comparison GL smoke checks

**What to build:** Verify comparison's actual shader output and interactions in a documented, repeatable native desktop smoke procedure.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: draft — awaiting breakdown approval
Scope: required
Source: [Maintainability specification](../spec.md), AC17 / MA-017; plan work package R (production renderer).
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [ ] Use temporary fixtures and record OS/GPU, standard/high display density, source sizes, shader compilation/output and screenshots.
- [ ] Exercise linked/unlinked pan, zoom, rotation, side-by-side/swipe transitions and large-source detail against the intended geometry/fidelity.
- [ ] Exercise close during work and record shutdown behavior; preserve deterministic canvas-reference and Linux golden tests.
- [ ] Provide exact fixture/setup/launch/action/result instructions so another maintainer can repeat the run; app launch alone and headless shader-object tests are insufficient.
- [ ] Record environment limitations explicitly, and retain any required unsupported-platform result as open instead of declaring full native coverage.
- [ ] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `make run`
2. `go test ./internal/ui/compare -count=1`

These commands are acceptance requirements for future implementation; they have not been run as proof that this draft is implemented. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

Can start against the current production renderer. Each later lifecycle fix still needs its own native evidence; no circular blocking edge is introduced back to those fixes.

## Completion boundary

This ticket closes only its named part of MA-017. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Draft prepared for breakdown approval; no implementation or publication has occurred.
