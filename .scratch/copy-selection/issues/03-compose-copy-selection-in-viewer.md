# 03: Compose Copy Selection in the viewer

**Spec:** [Copy an Image-Region Selection](../spec.md)

**What to build:** Construct the Copy Selection module in production and test
viewers, adapt zoom geometry into its `View`, place its overlay at the required
paint depth, and provide viewer-owned start/end coordination. This ticket adds
directly testable viewer behavior but no menu item or global shortcut yet.

**Blocked by:** 01, 02

**Status:** ready-for-agent

## Integration contract

- Add a `regionCopy *copyselection.Feature` field to `viewer`; do not reuse the
  existing `copySelection()` name, which routes ordinary `Cmd`/`Ctrl+C` between
  Grid selection and the displayed image.
- Construct `regionCopy` immediately after zoom in `registerFeatures`, using
  closures that resolve the viewer at interaction time.
- Add viewer methods `startRegionCopy`, `finishRegionCopy`, and
  `cancelRegionCopy`. Keep all information-overlay and repaint coordination in
  these viewer methods rather than teaching the feature module about other
  features.
- Adapt `zoom.Geometry` plus the oriented displayed-image bounds into
  `copyselection.View` in one `internal/ui` function.
- Place the Copy Selection overlay above the normal image and information
  overlay, and below Grid View, deletion, export prompt, and toast. Preserve
  the existing load-bearing tail order.
- Geometry notifications must update the selection through the Fyne-safe path
  documented by ticket 02; do not add an untracked goroutine.

## Behavior checklist

- [ ] Start with a failing real-viewer test proving direct activation shows an
      active selection overlay only when a decoded static image is present.
- [ ] Every activation starts blank and repeated activation is a no-op.
- [ ] Hide the information overlay on entry and restore exactly its prior
      visibility on cancel or successful completion.
- [ ] Forward overlay scroll to the existing zoom behavior and update the
      on-screen rectangle after every geometry notification.
- [ ] Keep the same image-region selection through zoom, pan, window resize,
      and HiDPI geometry changes, including partially off-screen regions.
- [ ] Ensure direct cancellation removes all selection visuals and the button
      without clearing files, changing zoom, or touching the clipboard.
- [ ] Update `newTestUI` construction and cleanup for every new per-viewer seam;
      do not add mutable package-level test configuration.

## Files

- Modify: `internal/ui/features.go`
- Modify: `internal/ui/build.go`
- Modify: `internal/ui/viewer.go`
- Create: `internal/ui/copyselection.go`
- Create or modify focused tests under `internal/ui/`
- Do not modify: menus, shortcuts, translations, manuals,
  `internal/clipboard`, or `ARCHITECTURE.md`

## Verification

```sh
go test -race ./internal/ui/... -run 'TestCopySelection(Activation|InfoOverlay|ZoomPanResize|Cancel)$'
```

Run the complete `internal/ui` package without race once the focused slice is
green. Before resolving, set `Status: resolved` and record the result under
`## Answer`. Do not commit.

## Answer

Pending.
