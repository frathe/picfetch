# Add swipe comparison

Status: complete

Route: Standard. This ticket extends the existing comparison feature across
`internal/ui/compare`, its assembled-viewer input seam, translations, and both
manuals. It adds no package, dependency, preference, or exported API.

Deliverable: comparison can switch between the existing linked side-by-side
view and a full-viewport swipe reveal with distinct divider and pan input.

## Locked decisions

| Decision | Contract |
|---|---|
| Test seam | Geometry and direct input are observed through `compare.Feature` and `Overlay`; the assembled viewer proves real canvas and modifier routing without reaching into comparison internals. |
| Layout | Side by side remains gapless 50/50. Swipe gives both panes the full viewport and clips their reveal at one normalized divider. |
| Input | The themed divider is the only reveal drag target; pane drags keep panning the linked transform. |
| State | Layout toggles retain the linked transform and divider subject to valid-range clamping; each `Open` resets side by side, fitted/centered, at 50%. |
| Keyboard | Left/Right step 5 points, Shift variants step 1, Home/End choose 0/100, and all are inert outside swipe. |

Non-goals: exhaustive resize/actual-size/swap transition guarantees assigned
to ticket 06, source-fidelity work assigned to ticket 07, touch gestures, and
persisting a preferred comparison layout.

## Tasks

### Task 1 - Toolbar and clipped swipe layout

Owner: T0 inline

Status: complete

Files: `internal/ui/compare`

Contract: add the ready-gated relabeling layout control and render both logical
panes in one aligned full viewport with left/right clipping and unchanged
chrome placement.

Test: `CompareSwipeToggle` and `CompareSwipeLayout` as separate red/green
slices, retaining the existing side-by-side and linked-transform guards.

Verify: the first and second ticket commands.

Budget: 0 spawns; at most 3 review rounds; no full suite.

### Task 2 - Divider input, keys, and reset

Owner: T0 inline

Status: complete

Files: `internal/ui/compare` and assembled comparison tests

Contract: divider drag and keys change only the clamped reveal value; ordinary
drag still pans; layout toggles and new sessions retain/reset the specified
state.

Test: `CompareSwipePointer`, `CompareDividerKeys`, `CompareLayoutToggle`, and
`CompareDividerReset` through feature and real-canvas seams.

Verify: the third through fifth ticket commands.

Budget: 0 spawns; at most 3 review rounds; no full suite.

### Task 3 - Localization, manuals, and landing records

Owner: T0 inline

Status: complete

Files: translation catalogues, both manuals and their guard, architecture/TODO
records, this plan, and the source ticket.

Contract: localize Swipe/Side by side, document divider pointer/keyboard
control in English and German, and record only verified completion claims.

Verify: the sixth ticket command, all focused commands after negative guard
checks, then `make verify` once.

Budget: 0 spawns; at most 1 review round; full suite once.

Task graph: Task 1 -> Task 2 -> Task 3. Each task interleaves one failing
behavior test with its minimum implementation.

## Delegation gate

No task is delegated. Layout/input contracts are hot cross-file context
(G4/G5), user-visible strings are never delegated, and the final gate remains
with T0. Budget: 0 spawns.

## Cost ledger

| Task | Spawns budget/actual | Review rounds | Full suite | Notes |
|---|---:|---:|---|---|
| T1 | 0 / 0 | 1 | no | Toggle/layout guards failed before implementation and pass after review. |
| T2 | 0 / 0 | 1 | no | Pointer/key guards failed before implementation; retention/reset rejected a deliberate 25% mutation. |
| T3 | 0 / 0 | 1 | no | Manual guard failed before documentation; locale parity rejected a missing German label mutation. |
| gate | - | - | - | yes | Passed all six acceptance commands and `make verify`; Linux/amd64 race: `internal/ui` 634.773s. |

## Verification

All ticket acceptance commands passed after formatting and review. `make
verify` then passed TUF validation, `go vet ./...`, `go build ./...`, and the
complete Linux/amd64 race suite (`internal/ui` 634.773s;
`internal/ui/compare` 2.960s; `internal/ui/help` 6.741s).
