# 04: Link side-by-side zoom and pan

**What to build:** Give both side-by-side panes one synchronized view so zooming
or panning either image changes both without exposing blank space or allowing
their centers to drift apart.

**Blocked by:** 01: Open and close a fitted side-by-side comparison

**Status:** ready-for-agent

## Acceptance criteria

- [ ] Both images share one normalized image-space center and one zoom
  multiplier relative to each image's own fitted scale. Images with different
  pixel dimensions or aspect ratios remain visually synchronized.
  Verify: `go test ./internal/ui/... -run 'CompareLinkedTransform' -count=1`
- [ ] Wheel and `+` / `-` zoom both images around the shared view. `0` fits and
  centers both images; `1` shows each at actual pixel size rather than applying
  the same raw canvas scale to differently sized inputs.
  Verify: `go test ./internal/ui/... -run 'Compare(Zoom|Fit|ActualSize)' -count=1`
- [ ] Dragging either image pans both. Shift+wheel pans both using the same
  modifier behavior as the normal viewer, and comparison command isolation
  admits these inputs without yielding to the grid or viewer.
  Verify: `go test ./internal/ui/... -run 'ComparePanInputs' -count=1`
- [ ] The shared center is clamped to the intersection of both images' valid
  pan ranges after every zoom or pan. Neither pane can expose blank overscroll,
  and repeated input cannot produce visible drift between the images.
  Verify: `go test ./internal/ui/... -run 'Compare(Clamp|NoDrift|NoOverscroll)' -count=1`
- [ ] Returning to fit establishes the canonical centered state from which
  subsequent normalized zoom and pan operations proceed.
  Verify: `go test ./internal/ui/... -run 'CompareFitReset' -count=1`
- [ ] Both manuals document linked zoom/pan and the `0`, `1`, `+`, and `-`
  controls without introducing Unicode arrows.
  Verify: `go test ./... -run 'Manual|UnicodeArrows' -count=1`

## Comments

