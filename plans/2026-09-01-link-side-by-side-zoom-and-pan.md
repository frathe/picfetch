# Link side-by-side zoom and pan

Status: complete

Route: Standard. This ticket deepens the existing comparison feature across
`internal/ui/compare` and its `internal/ui` adapter. It adds no package,
dependency, menu item, or user-visible string.

Deliverable: both side-by-side comparison panes share one normalized center
and linked scale, accept the established zoom/pan controls, and clamp against
both sources so neither pane can overscroll or drift.

## Locked decisions

| Decision | Contract |
|---|---|
| Test seam | Precise geometry and pointer behavior are observed through `compare.Feature` and `Overlay`; the assembled viewer tests only keyboard/modifier routing and covered-grid isolation. |
| Transform | One private normalized center drives both panes. Fitted mode uses each image's fit scale times one multiplier; actual mode uses one absolute pixels-to-points multiplier. |
| Actual size | `1` matches the normal viewer: center both at 100%, then continue wheel/key zoom from that absolute baseline until `0`. |
| Rendering | Each explicit image geometry is clipped to its own pane; the existing full-surface shield continues blocking unrelated input. |
| Clamping | Intersect both images' valid normalized center ranges after every zoom, pan, load, and relayout. |

Non-goals: swipe layout, explicit cross-layout/session transition guarantees,
SVG rerasterization, and broader source-fidelity work assigned to tickets
05-07.

## Tasks

### Task 1 - Linked transform and clipped panes

Owner: T0 inline

Status: complete

Files: `internal/ui/compare`

Contract: add `Callbacks.Modifiers` and `Feature.HandleKey`; render both panes
from one normalized transform and clamp it to their shared valid range.

Test: `CompareLinkedTransform`, `CompareZoom`, `CompareFit`,
`CompareActualSize`, `CompareClamp`, `CompareNoDrift`,
`CompareNoOverscroll`, and `CompareFitReset` as vertical red/green slices.

Verify: the first, second, fourth, and fifth ticket commands.

Budget: 0 spawns; at most 3 review rounds; no full suite.

### Task 2 - Pointer and keyboard routing

Owner: T0 inline

Status: complete

Files: `internal/ui/compare`, `internal/ui/keys.go`, `internal/ui/features.go`,
and focused comparison tests.

Contract: pane drag and Shift+wheel pan the comparison; wheel and comparison
keys alter only the comparison while all unrelated input remains isolated.

Test: `ComparePanInputs` through real pointer dispatch and assembled-viewer
state snapshots.

Verify: the second and third ticket commands.

Budget: 0 spawns; at most 2 review rounds; no full suite.

### Task 3 - Documentation and landing records

Owner: T0 inline

Status: complete

Files: both manuals, their guards, architecture/TODO records, this plan, and
the source ticket.

Contract: document linked controls with ASCII-only key notation, then record
only verified completion claims.

Verify: the sixth ticket command, all focused commands after negative guard
checks, then `make verify` once.

Budget: 0 spawns; at most 1 review round; full suite once.

Task graph: Task 1 -> Task 2 -> Task 3. Tasks 1 and 2 interleave one failing
behavior test with its minimum implementation.

## Delegation gate

No task is delegated. The transform/interface decisions and cross-package
routing are hot context (G4/G5), and user-visible documentation is never
delegated. Budget: 0 spawns.

## Cost ledger

| Task | Spawns budget/actual | Review rounds | Full suite | Notes |
|---|---:|---:|---|---|
| T1 | 0 / 0 | 1 | no | Linked transform, clipping, clamping, actual-size mode, and focused tests complete. |
| T2 | 0 / 0 | 1 | no | Keyboard, drag, wheel, modifier, and isolation routing complete. |
| T3 | 0 / 0 | 1 | no | Manuals, architecture, TODO, plan, and issue records complete. |
| gate | - | - | - | yes | `make verify` passed. |

## Verification record

- All six issue acceptance commands passed on the restored final tree.
- Mutation checks made the focused guards fail for broken fit-relative scale,
  actual-size mode, shared clamping, keyboard routing, drag routing,
  Shift+wheel routing, fit reset, and manual coverage; every mutation was
  restored before the acceptance rerun.
- `make verify` passed goimports, TUF expiry validation, `go vet`, `go build`,
  and the Docker Linux/amd64 race suite. The `internal/ui` race package took
  629.722 seconds.
