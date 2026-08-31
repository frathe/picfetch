# 05: Add swipe comparison

**What to build:** Add a swipe layout that overlays both synchronized images
and lets the user reveal either side with a pointer- or keyboard-controlled
divider while ordinary drags continue to pan the linked view.

**Blocked by:** 04: Link side-by-side zoom and pan

**Status:** ready-for-agent

## Acceptance criteria

- [ ] The permanent toolbar adds **Swipe** in side-by-side mode and changes its
  label to **Side by side** while swipe is active. The toggle remains disabled
  until both images are ready.
  Verify: `go test ./internal/ui/... -run 'CompareSwipeToggle' -count=1`
- [ ] Swipe draws both images across the full comparison viewport, revealing
  the logical left image left of a vertical divider and the logical right
  image to its right. Identity badges remain at the bottom-left and
  bottom-right corners and the toolbar remains at the top right.
  Verify: `go test ./internal/ui/... -run 'CompareSwipeLayout' -count=1`
- [ ] The divider itself is the drag target. Dragging it changes only the reveal
  percentage; dragging elsewhere pans both images through the linked transform
  without accidentally moving the divider.
  Verify: `go test ./internal/ui/... -run 'CompareSwipePointer' -count=1`
- [ ] In swipe mode, `Left` and `Right` move the divider by 5 percentage points,
  Shift+Left and Shift+Right move it by 1 point, and `Home` and `End` move it to
  0% and 100%. Values are clamped to that range. These keys do nothing in
  side-by-side mode.
  Verify: `go test ./internal/ui/... -run 'CompareDividerKeys' -count=1`
- [ ] Switching between layouts retains the current shared transform and
  divider value. Every newly opened comparison starts side by side with its
  inactive divider reset to 50%.
  Verify: `go test ./internal/ui/... -run 'Compare(LayoutToggle|DividerReset)' -count=1`
- [ ] Swipe labels and controls are localized in every catalogue and both
  manuals document pointer and keyboard divider control.
  Verify: `go test ./... -run 'Translations|Manual|UnicodeArrows' -count=1`

## Comments

