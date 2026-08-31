# 01: Build the Copy Selection module

**Spec:** [Copy an Image-Region Selection](../spec.md)

**What to build:** Add the deep `internal/ui/copyselection` module that owns
Copy Selection mode state, image-region geometry, pointer gestures, visual
objects, cursor behavior, pixel rounding, and PNG crop generation. Keep the
module usable without a real viewer or operating-system clipboard so its tests
exercise the same interface production will use.

**Blocked by:** None (can start immediately)

**Status:** ready-for-agent

## Interface contract

Downstream tickets depend on these concepts and names. Keep the interface
small; geometry helpers and renderer details remain unexported.

- `View` describes the image bounds plus its position and size in the overlay's
  canvas coordinate space.
- `State` reports only `Active`, `Busy`, and `HasSelection`.
- `Callbacks` supplies an image-rectangle copy request, a mode-ended
  notification, and scroll forwarding. It must not expose internal gesture or
  renderer state.
- `New(Callbacks) *Feature` constructs the module.
- `Overlay() fyne.CanvasObject` returns the one object `internal/ui` composes.
- `Start(View)`, `ViewChanged(View)`, `Cancel()`, `State() State`,
  `HandleKey(fyne.KeyName) bool`, and `Complete(error)` are the viewer-facing
  lifecycle interface.
- A package function accepts an `image.Image` plus the selected
  `image.Rectangle` and returns PNG bytes or a recoverable error. Callers do not
  reimplement crop or encoding logic.

If implementation proves one identifier cannot carry this contract, stop and
record the conflict under `## Comments`; do not grow a parallel interface.

## Behavior checklist

- [ ] Start with a failing interface-level test for drawing and committing one
      image-bounded rectangle; run it and confirm the failure is behavioral.
- [ ] Implement drag-in-any-direction creation, image clamping, outward pixel
      rounding, and the `1 x 1` minimum.
- [ ] Preserve an existing committed rectangle after an invalid replacement;
      replace it only when a new valid rectangle commits.
- [ ] Move from the interior without resizing and clamp the whole rectangle to
      the image.
- [ ] Resize through four side and four corner handles; normalize cleanly when
      a handle crosses its opposite edge.
- [ ] Keep the selection attached to image coordinates when `ViewChanged`
      supplies a different zoom, pan, viewport, or HiDPI geometry.
- [ ] Implement the settled crosshair, move, and directional resize cursors;
      dim only image content outside the rectangle.
- [ ] Use theme-sized, high-contrast border and handle visuals. Show the
      lower-right `Copy to clipboard` button only after a valid commit.
- [ ] `Escape` cancels; `Return` and `Enter` request copying only with a valid
      rectangle. Suppress editing and repeated copy while busy.
- [ ] `Complete(nil)` ends the mode; `Complete(err)` retains the selection and
      unlocks it. Both outcomes are observable through the public interface.
- [ ] Generate pixel-exact PNG data, preserve alpha, reject invalid bounds, and
      include no UI or metadata.
- [ ] Use independent literal pixels and rectangles as expectations; do not
      recompute expected geometry with production formulas or mock internal
      collaborators.

## Files

- Create: `internal/ui/copyselection/*.go`
- Create: `internal/ui/copyselection/*_test.go`
- Do not modify: `internal/ui`, `internal/ui/zoom`, `internal/clipboard`, menus,
  translations, manuals, or `ARCHITECTURE.md`

## Verification

```sh
go test -race ./internal/ui/copyselection/...
```

Before resolving the ticket, observe at least one geometry guard fail after a
deliberate local behavior break, restore it, rerun the command, check every box,
set `Status: resolved`, and add the result under `## Answer`. Do not commit.

## Answer

Pending.
