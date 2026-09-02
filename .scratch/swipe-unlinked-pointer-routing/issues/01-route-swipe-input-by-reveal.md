# 01 - Route Swipe input by revealed pane

Status: resolved

## Contract

Through `compare.Feature.Overlay()` in a real Fyne test window, prove that
Swipe + Unlinked comparison assigns pointer input to the photo occupying the
revealed region under the pointer. Hit regions must follow the current divider
without changing either photo's full-viewport render geometry.

Add a private `layoutPaneInput(index, input)` helper driven by
`paneVisibleArea`. Apply it during pane layout and every reveal/divider update,
and remove the full-viewport input reset from transform application. The
divider remains the exclusive drag target in its hit area, and a fully hidden
pane has no interactive area.

Files: `internal/ui/compare/compare_test.go`,
`internal/ui/compare/transform.go`, and `internal/ui/compare/swipe.go`.

## Red / green

1. Add `TestCompareSwipeUnlinkedCanvasRoutesPointerByReveal` using actual
   canvas hover, drag, and wheel dispatch at the default divider and after
   moving it to 75%.
2. Observe the current implementation report `Unlinked: Right` while the
   pointer is over the visible left photo.
3. Implement reveal-aligned pane input bounds.
4. Verify Left/Right status, pane-local gestures, and subsequent transform keys
   affect only the revealed target. Retain the last target after leaving a
   photo region.

## Acceptance

`go test ./internal/ui/compare -run '^TestCompareSwipeUnlinkedCanvasRoutesPointerByReveal$' -count=1`

## Constraints

- Do not change renderer viewports, reveal clips, image transforms, tile
  planning, shaders, caches, or divider behavior.
- Do not add an assembled-viewer duplicate of this regression.
- Do not add exported APIs or user-visible strings.

## Comments

- Red: the permanent canvas test reported `Unlinked: Right` while the pointer
  was at x=200 in the visible left reveal.
- Green: reveal-aligned pane input bounds passed the focused acceptance command,
  including divider movement, both extremes, gestures, and transform keys.
