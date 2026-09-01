# 04: Link side-by-side zoom and pan

**What to build:** Give both side-by-side panes one synchronized view so zooming
or panning either image changes both without exposing blank space or allowing
their centers to drift apart.

**Blocked by:** 01: Open and close a fitted side-by-side comparison

**Status:** ready-for-human

## Acceptance criteria

- [x] Both images share one normalized image-space center and one zoom
  multiplier relative to each image's own fitted scale. Images with different
  pixel dimensions or aspect ratios remain visually synchronized.
  Verify: `go test ./internal/ui/... -run 'CompareLinkedTransform' -count=1`
- [x] Wheel and `+` / `-` zoom both images around the shared view. `0` fits and
  centers both images; `1` shows each at actual pixel size rather than applying
  the same raw canvas scale to differently sized inputs.
  Verify: `go test ./internal/ui/... -run 'Compare(Zoom|Fit|ActualSize)' -count=1`
- [x] Dragging either image pans both. Shift+wheel pans both using the same
  modifier behavior as the normal viewer, and comparison command isolation
  admits these inputs without yielding to the grid or viewer.
  Verify: `go test ./internal/ui/... -run 'ComparePanInputs' -count=1`
- [x] The shared center is clamped to the intersection of both images' valid
  pan ranges after every zoom or pan. Neither pane can expose blank overscroll,
  and repeated input cannot produce visible drift between the images.
  Verify: `go test ./internal/ui/... -run 'Compare(Clamp|NoDrift|NoOverscroll)' -count=1`
- [x] Returning to fit establishes the canonical centered state from which
  subsequent normalized zoom and pan operations proceed.
  Verify: `go test ./internal/ui/... -run 'CompareFitReset' -count=1`
- [x] Both manuals document linked zoom/pan and the `0`, `1`, `+`, and `-`
  controls without introducing Unicode arrows.
  Verify: `go test ./... -run 'Manual|UnicodeArrows' -count=1`

## Comments

- 2026-09-01: Implemented one normalized center for both clipped panes, with a
  shared fit-relative multiplier and an explicit absolute-scale mode for `1`.
  Wheel/key zoom, drag/Shift+wheel pan, cursor anchoring, shared-range clamping,
  fit reset, and comparison-only keyboard/pointer routing are covered at the
  feature and assembled-viewer seams.
- 2026-09-01: All six acceptance commands passed after mutation checks proved
  the guards fail for broken fit-relative scaling, actual-size mode, clamping,
  keyboard routing, drag and Shift+wheel routing, fit reset, and manual text.
  `make verify` passed formatting, TUF validation, vet, build, and the complete
  Linux/amd64 race suite.
