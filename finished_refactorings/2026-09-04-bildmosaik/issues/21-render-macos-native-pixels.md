# Generate macOS mosaics at native panel pixels

Type: task
Status: resolved
Priority: P1
Blocked by: 02

## Goal

Make the macOS display inspector report physical pixel dimensions so a mosaic
generated for a Retina panel fills that panel at its real resolution instead of
half of it.

## Evidence

- `internal/displays/darwin.go:31` reads `CGDisplayBounds(ident)`, which returns
  the display's rectangle in the global *point* space. On a 2x panel a
  3456x2234-pixel screen is reported as 1728x1117.
- The spec requires "Detect attached displays, their stable session identifiers,
  native pixel dimensions, and aspect ratios" and "Render one image at the native
  pixel dimensions of the selected display". Ticket 02 states "Pixel bounds are
  physical/native bounds, including HiDPI displays."
- The consequence is twofold: generation targets half-resolution bounds, and the
  picker label built at `internal/ui/mosaicwin/window.go:343` advertises point
  dimensions to the user as native resolution.
- Windows (`internal/displays/windows.go:99`, `IDesktopWallpaper::GetMonitorRECT`)
  and Linux (`internal/displays/linux.go:39`, xrandr) already report physical
  pixels, so macOS is the only inconsistent backend.

## Decisions

- Derive the pixel size from the display's current mode
  (`CGDisplayCopyDisplayMode` plus `CGDisplayModeGetPixelWidth` /
  `CGDisplayModeGetPixelHeight`), or equivalently from
  `-[NSScreen convertRectToBacking:]`, and keep `CGDisplayBounds` only for the
  origin used to pick the display under the PicFetch window.
- Keep the origin in point space, since window-overlap comparison happens in
  point space; do not mix the two coordinate systems in one rectangle.
- Leave the Windows and Linux backends unchanged.

## Acceptance Criteria

- The macOS inspector reports pixel width and height that match the panel's
  current display mode, not its point size.
- Display-under-window selection still resolves correctly with a scaled panel
  and a second unscaled panel attached.
- A mosaic generated for a 2x display produces an image whose bounds equal the
  panel's physical pixel dimensions.

```sh
go test ./internal/displays -run 'TestInspect' -count=1 &&
go test ./internal/ui/mosaicwin -run 'TestMosaicTarget' -count=1
```

## Non-Goals

- Changing how the viewer or Grid View scale content
- Adding a user-facing scale factor setting
- Altering Windows or Linux display detection

## Comments

Found by a spec-axis review of the branch on 2026-09-04. Needs native macOS
hardware with a Retina panel to confirm; the automated part can only assert the
mode-based path is used.

## Answer

The Darwin adapter now obtains width and height from the current CoreGraphics
display mode and releases the copied mode. It publishes a zero-origin local
native-pixel rectangle, while the unchanged AppKit window/screen intersection
continues to choose the default target entirely in point space. A portable
adapter-source guard was observed failing against the old point-size path and
passing after the change; the mosaic target test also pins a 3456x2234 request.

Verified with the ticket acceptance command on macOS. Physical scaled plus
unscaled multi-panel confirmation remains in the supervised smoke matrix.
