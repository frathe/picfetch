# Preserve comparison state across transitions

Status: complete

Route: Standard. This ticket hardens the existing comparison feature through
`internal/ui/compare` and its assembled-viewer seam. It adds no package,
dependency, exported API, preference, or user-visible string.

Deliverable: layout changes, window resizes, Swap, and session replacement
preserve or reset comparison state exactly as specified without mutating the
covered grid.

## Locked decisions

| Decision | Contract |
|---|---|
| Test seam | Precise geometry is observed through `compare.Feature` and `Overlay`; the assembled viewer proves grid and title preservation. |
| Relayout | Fit-relative scale is recomputed from each destination viewport while the shared multiplier and closest valid center survive. |
| Actual size | `1` remains an absolute 100% pixel scale through resize and layout changes. |
| Swap | Swap exchanges presentation roles only; layout, transform, divider, loading generation, and covered grid remain unchanged. |
| Session reset | Every `Open` starts fitted, centered, side by side, in source order, with the inactive divider at 50%. |

Non-goals: source-format fidelity assigned to ticket 07, a preferred persisted
layout, new controls, translations, and manual changes.

## Tasks

### Task 1 - Component transition guards

Owner: T0 inline

Status: complete

Files: `internal/ui/compare/compare_test.go`; production comparison files only
if a guard exposes a concrete mismatch.

Contract: cover fit-relative layout changes, both resize layouts and clamping,
absolute-size transitions, transformed swipe Swap, and complete session reset.

Test: `CompareLayoutTransition`, `CompareResize`,
`CompareActualSizeTransition`, `CompareSwapState`, and
`CompareSessionReset` as vertical behavior slices.

Verify: the first four and sixth ticket acceptance commands.

Budget: 0 spawns; at most 2 review rounds; no full suite.

### Task 2 - Assembled-viewer preservation guard

Owner: T0 inline

Status: complete

Files: `internal/ui/compare_test.go`.

Contract: a real comparison transition/resize/Swap round trip leaves file
order, filter, selection, highlight, scroll, and restored grid title unchanged.

Test: `CompareTransitionPreservesGrid` through the existing viewer harness.

Verify: the fifth ticket acceptance command.

Budget: 0 spawns; at most 2 review rounds; no full suite.

### Task 3 - Landing records and final gate

Owner: T0 inline

Status: complete

Files: source ticket, `todos.md`, and this plan.

Contract: record only negatively verified and green acceptance claims while
preserving unrelated user edits already present in `todos.md`.

Verify: all six ticket commands, the complete focused comparison suite, then
`make verify` once.

Budget: 0 spawns; at most 1 review round; full suite once.

Task graph: Task 1 -> Task 2 -> Task 3. Every new guard is observed failing
against a deliberate behavior mutation before final acceptance.

## Delegation gate

No task is delegated. The state and seam decisions are hot cross-file context,
and the final review/gate remain with T0. Budget: 0 spawns.

## Cost ledger

| Task | Spawns budget/actual | Review rounds | Full suite | Notes |
|---|---:|---:|---|---|
| T1 | 0 / 0 | 1 | no | Five component guards failed under deliberate mutations, then passed restored. |
| T2 | 0 / 0 | 1 | no | The viewer guard rejected a deliberate covered-selection mutation. |
| T3 | 0 / 0 | 1 | no | Acceptance commands, issue, and TODO records complete. |
| gate | - | - | yes | `make verify` passed; Linux race: `internal/ui` 646.227s. |

## Verification

All six ticket acceptance commands and the complete focused `Compare` suite
passed on the restored tree. Deliberate mutations proved every new guard rejects
the behavior regression it names. `make verify` passed TUF validation, vet,
build, and the Linux/amd64 race suite (`internal/ui` 646.227s;
`internal/ui/compare` 14.657s).
