# 06: Preserve comparison state across transitions

**What to build:** Make resize, layout switching, and Swap behave as continuous
changes to one comparison session, preserving the user's point of interest and
scale while still resetting cleanly for the next session.

**Blocked by:** 02: Identify and swap compared images; 05: Add swipe comparison

**Status:** ready-for-agent

## Acceptance criteria

- [ ] Switching between side-by-side and swipe preserves the normalized center,
  zoom multiplier, and divider position while recomputing each image's fitted
  scale for the destination viewport.
  Verify: `go test ./internal/ui/... -run 'CompareLayoutTransition' -count=1`
- [ ] Resizing the main window preserves the normalized center and zoom
  multiplier, recomputes fitted scales for the new pane or swipe viewport, and
  reclamps the shared center without exposing blank space.
  Verify: `go test ./internal/ui/... -run 'CompareResize' -count=1`
- [ ] Actual-size mode remains an absolute 100% pixel scale through resize and
  layout changes rather than being reinterpreted as a fitted-size multiplier.
  Its shared center remains as close as possible after valid-range clamping.
  Verify: `go test ./internal/ui/... -run 'CompareActualSizeTransition' -count=1`
- [ ] Swap exchanges images, badges, title order, and swipe reveal roles while
  preserving the current layout, normalized center, effective transform, and
  divider position.
  Verify: `go test ./internal/ui/... -run 'CompareSwapState' -count=1`
- [ ] No transition or swap mutates grid selection, file order, filter,
  highlight, or scroll state, and returning to Grid View restores its title.
  Verify: `go test ./internal/ui/... -run 'CompareTransitionPreservesGrid' -count=1`
- [ ] Closing and opening a new comparison never leaks prior session state: the
  new session starts fitted, centered, side by side, and with a 50% inactive
  divider regardless of the preceding layout or transform.
  Verify: `go test ./internal/ui/... -run 'CompareSessionReset' -count=1`

## Comments

