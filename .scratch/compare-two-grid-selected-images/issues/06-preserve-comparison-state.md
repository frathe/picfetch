# 06: Preserve comparison state across transitions

**What to build:** Make resize, layout switching, and Swap behave as continuous
changes to one comparison session, preserving the user's point of interest and
scale while still resetting cleanly for the next session.

**Blocked by:** 02: Identify and swap compared images; 05: Add swipe comparison

**Status:** ready-for-human

## Acceptance criteria

- [x] Switching between side-by-side and swipe preserves the normalized center,
  zoom multiplier, and divider position while recomputing each image's fitted
  scale for the destination viewport.
  Verify: `go test ./internal/ui/... -run 'CompareLayoutTransition' -count=1`
- [x] Resizing the main window preserves the normalized center and zoom
  multiplier, recomputes fitted scales for the new pane or swipe viewport, and
  reclamps the shared center without exposing blank space.
  Verify: `go test ./internal/ui/... -run 'CompareResize' -count=1`
- [x] Actual-size mode remains an absolute 100% pixel scale through resize and
  layout changes rather than being reinterpreted as a fitted-size multiplier.
  Its shared center remains as close as possible after valid-range clamping.
  Verify: `go test ./internal/ui/... -run 'CompareActualSizeTransition' -count=1`
- [x] Swap exchanges images, badges, title order, and swipe reveal roles while
  preserving the current layout, normalized center, effective transform, and
  divider position.
  Verify: `go test ./internal/ui/... -run 'CompareSwapState' -count=1`
- [x] No transition or swap mutates grid selection, file order, filter,
  highlight, or scroll state, and returning to Grid View restores its title.
  Verify: `go test ./internal/ui/... -run 'CompareTransitionPreservesGrid' -count=1`
- [x] Closing and opening a new comparison never leaks prior session state: the
  new session starts fitted, centered, side by side, and with a 50% inactive
  divider regardless of the preceding layout or transform.
  Verify: `go test ./internal/ui/... -run 'CompareSessionReset' -count=1`

## Comments

- 2026-09-01: Added component guards for fit-relative layout transitions,
  side-by-side and swipe resizing/clamping, absolute-size transitions,
  transformed swipe Swap, and complete session reset, plus an assembled-viewer
  guard for unchanged grid state and restored title.
- The existing comparison state model already satisfied all six behaviors, so
  no production source change was needed. Each new guard was observed failing
  against a deliberate transform, relayout, actual-scale, Swap, reset, or grid
  mutation, restored, and rerun green. All six acceptance commands and the
  complete focused comparison suite pass.
- Final gate: `make verify` passed, including the Linux/amd64 race suite
  (`internal/ui` in 646.227s; `internal/ui/compare` in 14.657s).
