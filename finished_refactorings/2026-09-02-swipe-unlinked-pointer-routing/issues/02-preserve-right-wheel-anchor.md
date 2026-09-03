# 02 - Preserve the right Swipe wheel anchor

Status: resolved
Blocked by: 01

## Contract

After Ticket 01 makes the right pane input start at the divider, preserve the
full-viewport image point beneath an unmodified wheel gesture. Copy each
non-nil `fyne.ScrollEvent`, add the input widget's reveal offset to the copied
event position, and forward that viewport-relative event. Never mutate the
event supplied by the caller.

Files: `internal/ui/compare/compare_test.go` and
`internal/ui/compare/input.go`.

## Red / green

1. Add `TestCompareSwipeUnlinkedRightWheelPreservesViewportAnchor` through the
   overlay's pane input seam after Ticket 01 is green.
2. Observe the right photo zoom around the reveal-local coordinate instead of
   the full-viewport cursor position.
3. Add the scroll-coordinate translation and observe the point beneath the
   cursor remain fixed.
4. Verify the original event is unchanged and nil events remain inert.

## Acceptance

`go test ./internal/ui/compare -run '^TestCompareSwipeUnlinkedRightWheelPreservesViewportAnchor$' -count=1`

## Constraints

- Preserve left-pane, side-by-side, linked-wheel, and Shift+wheel behavior.
- Do not expose pane internals or add a second scroll path.
- Do not mutate caller-owned input events.

## Comments

- Red: with reveal-local x=100 forwarded unchanged, the normalized point under
  full-viewport x=500 moved from `0.625` to `0.5774` during wheel zoom.
- Green: translating a copied event by the input origin preserved the anchor;
  the original event remained unchanged and nil stayed inert.
