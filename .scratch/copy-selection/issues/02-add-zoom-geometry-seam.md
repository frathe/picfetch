# 02: Add the zoom geometry seam

**Spec:** [Copy an Image-Region Selection](../spec.md)

**What to build:** Give `internal/ui/zoom` one narrow interface through which
the viewer can translate image coordinates into the current canvas geometry,
forward scroll gestures from the selection overlay, and learn when zoom, pan,
or viewport layout changes that geometry. Preserve all existing zoom behavior.

**Blocked by:** 01

**Status:** ready-for-agent

## Interface contract

- Add a value `Geometry` containing the displayed image position and size in
  the zoom widget's coordinate space. It reports presentation geometry, not
  Copy Selection state.
- Add `Geometry() Geometry` on `*Zoom`.
- Add `HandleScroll(*fyne.ScrollEvent)` on `*Zoom`; the existing image widget
  and the selection overlay must delegate to the same implementation so wheel
  zoom and `Shift+scroll` pan cannot drift.
- Add one per-instance geometry-change callback on `*Zoom`. Fire it when
  `apply` changes image position or size because of fit, zoom, pan, or viewport
  layout, and suppress identical notifications.
- A callback fired from renderer layout must not synchronously mutate another
  widget. Document this constraint next to the interface and preserve a
  deterministic test path.
- Do not import `internal/ui/copyselection` into `internal/ui/zoom`;
  `internal/ui` is the adapter between the two modules.

## Behavior checklist

- [ ] Start with a failing test showing that `Geometry` matches the actual
      canvas image position and size at fit.
- [ ] Cover manual zoom, cursor-anchored wheel zoom, drag pan,
      `Shift+scroll` pan, reset-to-fit, and viewport resize with literal
      geometry expectations.
- [ ] Prove the existing image widget and the new forwarding method produce
      identical scroll behavior.
- [ ] Prove the change callback fires for real geometry movement, including
      pan, and stays silent for identical geometry.
- [ ] Keep the existing `onChanged` and vector `onScaleChanged` contracts
      intact; do not make pan trigger window auto-resize or duplicate vector
      raster requests.
- [ ] Keep zoom's cursor behavior unchanged outside Copy Selection mode.

## Files

- Modify: `internal/ui/zoom/zoom.go`
- Modify: `internal/ui/zoom/widget.go`
- Modify: `internal/ui/zoom/zoom_test.go`
- Do not modify: `internal/ui/copyselection`, other `internal/ui` files,
  translations, manuals, or `ARCHITECTURE.md`

## Verification

```sh
go test -race ./internal/ui/zoom/...
```

Before resolving the ticket, negatively verify the geometry callback guard,
restore it, rerun the command, set `Status: resolved`, and record the result
under `## Answer`. Do not commit.

## Answer

Pending.
