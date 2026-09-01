# 05: Add swipe comparison

**What to build:** Add a swipe layout that overlays both synchronized images
and lets the user reveal either side with a pointer- or keyboard-controlled
divider while ordinary drags continue to pan the linked view.

**Blocked by:** 04: Link side-by-side zoom and pan

**Status:** ready-for-human

## Acceptance criteria

- [x] The permanent toolbar adds **Swipe** in side-by-side mode and changes its
  label to **Side by side** while swipe is active. The toggle remains disabled
  until both images are ready.
  Verify: `go test ./internal/ui/... -run 'CompareSwipeToggle' -count=1`
- [x] Swipe draws both images across the full comparison viewport, revealing
  the logical left image left of a vertical divider and the logical right
  image to its right. Identity badges remain at the bottom-left and
  bottom-right corners and the toolbar remains at the top right.
  Verify: `go test ./internal/ui/... -run 'CompareSwipeLayout' -count=1`
- [x] The divider itself is the drag target. Dragging it changes only the reveal
  percentage; dragging elsewhere pans both images through the linked transform
  without accidentally moving the divider.
  Verify: `go test ./internal/ui/... -run 'CompareSwipePointer' -count=1`
- [x] In swipe mode, `Left` and `Right` move the divider by 5 percentage points,
  Shift+Left and Shift+Right move it by 1 point, and `Home` and `End` move it to
  0% and 100%. Values are clamped to that range. These keys do nothing in
  side-by-side mode.
  Verify: `go test ./internal/ui/... -run 'CompareDividerKeys' -count=1`
- [x] Switching between layouts retains the current shared transform and
  divider value. Every newly opened comparison starts side by side with its
  inactive divider reset to 50%.
  Verify: `go test ./internal/ui/... -run 'Compare(LayoutToggle|DividerReset)' -count=1`
- [x] Swipe labels and controls are localized in every catalogue and both
  manuals document pointer and keyboard divider control.
  Verify: `go test ./... -run 'Translations|Manual|UnicodeArrows' -count=1`

## Comments

- 2026-09-01: Implemented the ready-gated **Swipe** / **Side by side** toolbar
  control, aligned full-viewport reveal clips, themed draggable divider with
  horizontal-resize cursor, and swipe-only stepped/fine/end-point keyboard
  controls. Divider input changes only reveal geometry; pane input retains the
  existing linked pan transform. Layout toggles retain transform/divider state,
  while each new comparison resets side by side at 50%.
- 2026-09-01: All six acceptance commands passed at the feature and assembled
  viewer seams. Red/green and mutation checks covered toggle gating, reveal
  rendering, pointer separation, key routing, session reset, manual wording,
  and locale parity. `make verify` passed TUF validation, vet, build, and the
  complete Linux/amd64 race suite (`internal/ui` 634.773s;
  `internal/ui/compare` 2.960s).
- 2026-09-01: Fixed divider stutter by separating full swipe layout from the
  boundary-only pointer/key path. A red regression guard measured 100 owner
  repaints and 1,300 static-image bounds probes per pane across 100 divider
  events; the optimized path records zero for both. Separate mutations proved
  the guard rejects either recursive content refreshes or owner repaints.
- 2026-09-01: In the assembled-viewer profile, 25 pre-fix divider drags spent
  80 ms and allocated 41.94 MB below `dragDivider`, including hidden resource
  image decoding. The same post-fix profile records no CPU or allocation
  samples below `dragDivider`; the focused swipe/divider/layout suite passes.
