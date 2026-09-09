# 01 - Keep the Grid View ring visible while scrolling

Status: resolved
Blocked by: None
Source: [Grid scroll follows the viewport during duplicate merging](../spec.md), AC1 and AC2.

## Contract

Add a transparent private `scrollFollower` above the padded `GridWrap` in the
Grid View body. It implements only `fyne.Scrollable`, forwards the vertical
wheel/trackpad delta through `GridWrap.ScrollToOffset`, and reconciles the ring
after an actual offset change.

The reconciliation keeps the ring while its cell is fully visible under
Fyne's own row geometry. Once clipped, it selects the nearest fully visible
edge row: first after downward movement and last after upward movement. It
retains the current column where possible and clamps a short final row. When no
full row fits, it chooses the nearest intersecting edge row and permits Fyne's
minimum reveal correction.

Use the existing highlight path so title, menu, and keyboard-cursor state stay
in sync. Do not load an image or change Grid selection. The proxy must remain
separate from `marqueeCatcher` and implement no tap, hover, mouse, drag, or
secondary-click interfaces.

Files: `internal/ui/grid/grid.go`, new
`internal/ui/grid/scrollfollow.go`, and `internal/ui/grid/marquee_test.go`.

## Red / green

1. Mount the real Grid View overlay in a Fyne test window and send `test.Scroll`
   to an interior cell.
2. Observe the current widget move its offset while leaving the ring offscreen.
3. Add the scroll-only proxy and full-visibility helpers; assert the same event
   moves the ring to a visible same-column cell without `ShowImage` or selection
   changes.
4. Add deterministic edge-row cases for both directions, partial rows, a short
   final row, and a viewport shorter than one cell.
5. Add interface and canvas-routing guards proving cell tap, hover, marquee
   drag, and scrollbar drag/tap retain their current targets.

## Acceptance

`go test ./internal/ui/grid -run '^(TestScrollFollower_|TestMarquee)' -count=1`

## Constraints

- Do not fork, upgrade, or patch Fyne.
- Do not add an exported API, preference, translation key, worker, timer, or
  mutable package-level test seam.
- Do not create a second scrolling model or bypass GridWrap's public offset API.
- Scrollbar thumb/track/page movement stays native and does not update the ring
  immediately; this ticket must not claim otherwise.
- Keep tests in existing files, avoiding a Qodana exclusion and root-UI shard
  update.

## Comments

Published from the approved parent specification on 2026-09-08.

Implemented with Standard SDD and red/green TDD; verification completed on
2026-09-09. Focused grid tests, negative routing/restoration guards, and the full
`make verify` Linux/amd64 race gate passed. Implementation record:
`finished_refactorings/2026-09-08-grid-scroll-follow-duplicate-merges.md`.
