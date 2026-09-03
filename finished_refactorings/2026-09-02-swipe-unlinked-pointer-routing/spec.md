# Spec: unlinked swipe pointer routing

Status: complete

## Problem

In Swipe comparison, both scrollable pane inputs occupy the full viewport.
Fyne treats each scrollable as a new hit-test clip, so the topmost right pane
captures pointer input even over the visible left photo. While the panes are
unlinked, hover therefore reports Right and the user can interact only with
the right photo.

## Decisions

- Use **Linked comparison** and **Unlinked comparison** as the canonical terms;
  avoid locked/unlocked comparison.
- In Swipe + Unlinked comparison, hover, drag, wheel, cursor, and
  last-hovered transform-key targeting follow the currently revealed photo.
- Pane hit regions track the movable divider. At the 0% and 100% extremes,
  the fully hidden pane has no interactive area.
- Leaving a revealed photo for the divider, toolbar, or window edge retains
  the last target for `0`, `1`, `+`, and `-`.
- Moving the divider across a stationary cursor changes the target on the
  next pointer event; divider movement alone does not replace the last target.
- The divider remains the exclusive drag target within its own hit area.
- Full-viewport photo rendering and reveal clipping remain unchanged.
- A right-pane wheel event is translated from its reveal-local coordinates
  into full-viewport coordinates so zoom remains anchored beneath the cursor.
- Regression tests use `compare.Feature.Overlay()` in a real Fyne test window.
  The assembled-viewer suite is regression coverage, not a duplicate test
  seam.

## Acceptance criteria

1. Actual canvas hover over either revealed Swipe region reports the matching
   Left or Right target at the default divider and after moving the divider.
   Verify:
   `go test ./internal/ui/compare -run '^TestCompareSwipeUnlinkedCanvasRoutesPointerByReveal$' -count=1`
2. Canvas drag, wheel input, and subsequent transform keys affect only the
   targeted photo while unlinked.
   Verify: the canvas-routing command above.
3. Wheel zoom over the right reveal preserves the full-viewport image point
   beneath the cursor and does not mutate the supplied event.
   Verify:
   `go test ./internal/ui/compare -run '^TestCompareSwipeUnlinkedRightWheelPreservesViewportAnchor$' -count=1`
4. Linked comparison, side-by-side input, divider precedence, layout
   transitions, and command isolation retain their existing behavior.
   Verify: `go test ./internal/ui/compare -count=1`
   Verify:
   `go test ./internal/ui -run 'Compare(LinkToggle|SwipePointer)' -count=1`
5. `CONTEXT.md` records the canonical comparison terms, `ARCHITECTURE.md`
   records the reveal-aligned input invariant, and `todos.md` records the
   bugfix using linking/unlinking terminology.
   Verify: `rg -n 'Linked comparison|Unlinked comparison' CONTEXT.md`
   Verify: `rg -n 'reveal|revealed' ARCHITECTURE.md todos.md`
6. Formatting, vet, build, and the complete race suite pass.
   Verify: `make verify`

## Non-goals

- Changing image render geometry, reveal clipping, tile planning, caches, or
  GPU shaders.
- Changing linked-camera behavior, photo transforms, divider semantics, Swap,
  or comparison lifecycle behavior.
- Adding exported APIs, preferences, localization keys, or new user controls.
- Retargeting from divider movement without a subsequent pointer event.
- Duplicating the regression through the assembled viewer or requiring a
  manual native UI smoke test.
- Rewriting the already-correct comparison manuals or creating an ADR.

## Honest limit

The deterministic regression uses Fyne's test driver rather than a manual UI
session. The test and native drivers share the diagnosed hit-test walker, and
the reported native symptom matches the test failure, so focused native tests
plus `make verify` are the acceptance boundary.

## Outcome

Swipe pane inputs now track their reveal bounds while both photo render
viewports remain full-size. Right-pane wheel events are copied and translated
back into viewport coordinates. Both regression guards were observed failing
against deliberate reversions, all focused acceptance commands passed, and
`make verify` completed successfully.
