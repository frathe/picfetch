# Copy an Image-Region Selection

Status: ready-for-agent

## Problem Statement

PicFetch can copy the complete displayed image to the system clipboard, but it
cannot copy only the part the user needs. Users must currently paste the whole
image into another application and crop it there. That interrupts quick
workflows such as sharing a detail, extracting a reference, or copying part of
a large image.

## Solution

Add **Copy selection** to the Actions menu. Activating it in the normal
single-image viewer starts a transient Copy Selection mode. The user draws one
rectangular image-region selection, can move or resize it, and copies its
full-resolution pixels through a **Copy to clipboard** button pinned to the
lower-right corner of the window.

The shortcut is `Option+Shift+C` on macOS and `Alt+Shift+C` elsewhere. Copying
successfully hides the selection and button and ends the mode. Cancelling does
not modify either the source file or the clipboard.

## Terminology

- **Grid selection** is a set of files selected in Grid View for a batch
  action.
- **Image-region selection** is one rectangular region of displayed image
  content in the image's oriented coordinate space.
- **Copy Selection mode** is the transient single-image-viewer interaction
  that defines and copies an image-region selection.
- **Crop** refers only to the bitmap generated for the clipboard. This feature
  never crops or otherwise modifies the source file.

These terms are also recorded in `CONTEXT.md`. Do not use bare "selection" in
new code or documentation where Grid selection and image-region selection
could be confused.

## User Stories

1. As a user viewing one image, I want to mark a rectangular region so that I
   can copy only the pixels I need.
2. As a keyboard user, I want a dedicated shortcut so that I can enter the mode
   without opening the Actions menu.
3. As a user making a precise selection, I want to move and resize the rectangle
   before copying it.
4. As a user working at a convenient zoom, I want the copied output to retain
   the image's resolution rather than the window's rendered resolution.
5. As a user correcting a selection, I want to redraw it without first finding
   a Reset control.
6. As a user whose clipboard integration fails, I want the selection retained
   so that I can retry.
7. As a user who changes my mind, I want `Escape` to leave the mode without
   changing the clipboard.
8. As a user of animated, vector, or RAW images, I want the copied pixels to
   correspond predictably to the image PicFetch shows me.

## Resolved Product Decisions

The following decisions are final for this feature and should not be
relitigated during implementation.

### Menu, Shortcut, and Availability

- Add **Copy selection** between the existing **Copy image** and **Copy image
  path** items in the Actions menu. Keep both existing actions unchanged.
- Display and bind `Option+Shift+C` on macOS and `Alt+Shift+C` elsewhere. The
  Fyne binding is `KeyModifierAlt | KeyModifierShift` with key `C`.
- Enable the action only when a decoded image is displayed in the normal
  single-image viewer.
- Disable it during the empty, initial, and loading states; while Grid View or
  Picture Frame mode is active; and while a modal prompt owns the window.
- Keep the menu item ordinary and unchecked while the mode is active. Invoking
  it again is a no-op; the crosshair and selection visuals communicate the
  active mode.

### Entering and Drawing

- Every activation begins without a rectangle. No selection is remembered per
  image, between activations, or between sessions.
- Hide the information overlay while the mode is active and restore its prior
  visibility when the mode ends or is cancelled.
- Show the crosshair only over selectable image content. Use the normal cursor
  over letterbox space and other window UI.
- A drag must begin inside the displayed image. It may proceed in any direction
  and shows a live rectangle while the pointer moves.
- Clamp the rectangle to image content; letterbox and window background pixels
  can never become part of it.
- Commit the rectangle on pointer release only when it covers at least one
  image pixel. An invalid click or sub-pixel drag creates no selection.
- Starting a valid drag outside an existing rectangle replaces it. Preserve the
  previous committed rectangle until the replacement becomes valid; an invalid
  replacement gesture must not destroy it.

### Moving and Resizing

- Dragging inside a committed rectangle moves it without changing its size and
  clamps it to the image boundary.
- Show eight resize handles after commit. Side handles change one axis; corner
  handles change both.
- Allow a dragged handle to cross its opposite edge. Normalize the rectangle
  and continue resizing from the new side instead of stopping abruptly.
- Keep the selection unconstrained. Do not add aspect-ratio locking, centered
  resizing, fixed presets, or a dimension readout.
- Show a move cursor inside the rectangle, directional resize cursors over its
  handles, and the crosshair over other selectable image content.

### Visual Treatment

- Dim only image content outside the selection. Do not dim the rest of the
  application window.
- Use a theme-aware, high-contrast border and eight theme-sized handles. Their
  usable on-screen thickness must remain stable across zoom and HiDPI display
  scaling.
- Show **Copy to clipboard** only after a valid rectangle is committed. Pin it
  inside the lower-right window corner with theme-standard padding and keep it
  visible while the rectangle is moved or resized.
- The button may visually overlap image content, but it must not affect the
  selected bounds or appear in the clipboard bitmap.
- Add a golden screenshot covering a committed selection, its dimming, border,
  handles, cursor-independent visuals, and lower-right button.

### Image Coordinates and Viewport Changes

- Store the selection in the oriented image's coordinate space, not in window
  or canvas coordinates.
- Keep it attached to the same image pixels through zoom, pan, window resizing,
  and HiDPI scaling. Transform only its on-screen representation.
- Keep existing `0`, `1`, `+`, `-`, wheel zoom, and `Shift+scroll` pan behavior
  available. Ordinary dragging belongs to selection drawing, movement, or
  resizing while the mode is active, rather than to zoom's drag-to-pan action.
- A selection may extend outside the current viewport after zooming or panning.
  Copy its complete image-space region, including the temporarily off-screen
  part.
- Resolve selection edges outward to integer image pixels: floor left and top,
  ceil right and bottom, then clamp to the image. The minimum committed crop is
  `1 x 1` pixel.

### Clipboard Pixels

- Copy a content crop at image resolution. Window dimensions, viewport scale,
  UI overlays, dimming, border, handles, and the Copy button must not appear in
  the result.
- Copy PNG image data through PicFetch's existing cross-platform image
  clipboard path. Preserve alpha where the displayed image has transparency;
  include no source metadata or file reference.
- Apply EXIF orientation and any unsaved view-only rotation exactly as shown.
- Freeze an animated image on the frame visible when the mode begins and copy
  that frame. Resume animation when the mode ends; exact animation timing phase
  need not be preserved.
- For SVG, map the selection through the image's logical dimensions and
  rasterize that region at logical resolution, subject to the existing vector
  raster safety cap.
- For camera RAW, crop the embedded preview PicFetch displays. This feature does
  not introduce full RAW development.

### Copying, Errors, and Exit

- `Escape` cancels the mode without copying. `Return` or `Enter` invokes Copy
  when a valid selection exists. Keep geometry editing pointer-based; do not
  repurpose arrow keys for movement or resizing.
- Suppress normal image-navigation keys while the mode owns keyboard input.
- Except for zoom and pan, any other PicFetch command cancels the mode before
  performing its normal action. This includes navigation, open/drop, close,
  rotate, Grid View, Picture Frame, secondary windows, and other Actions menu
  commands.
- Merely moving focus to another application preserves the selection.
- Clicking Copy atomically captures the frozen image and selected pixel bounds.
  Crop and PNG-encode off the UI thread, then call the existing clipboard
  dispatcher.
- While copying, disable the button and ignore selection editing, repeated copy,
  `Escape`, and other application commands. Normal window close remains
  available.
- On success, hide the selection and button, restore temporarily hidden state,
  end the mode, and show no success toast.
- On a recoverable crop, PNG encoding, or clipboard failure, use the existing
  clipboard-error reporting style: show an error toast, retain the committed
  selection, and unlock the mode for retry.
- Add no new crop-size limit. A very large crop may still fail because the
  process or operating system cannot allocate or accept it; the retained
  selection and error toast are the recovery path.

## Architecture Decisions

- Add `internal/ui/copyselection` as a feature package. It owns transient mode
  state, image-region geometry, coordinate transforms, pointer interaction,
  visual objects, cursors, and feature-level tests through a narrow
  consumer-side interface.
- Keep cross-feature composition in `internal/ui`: availability and menu state,
  cancellation before other features act, zoom/display coordination, animation
  freezing, information-overlay restoration, clipboard dispatch, and shortcut
  registration.
- Extend `internal/ui/zoom` only through the narrow state/transform or input
  hooks Copy Selection genuinely needs. Do not duplicate zoom calculations or
  build a parallel image display path.
- Keep `internal/clipboard` unchanged and continue to replace its dispatcher in
  tests through `internal/uitest`.
- Place the new selection overlay explicitly in `buildViewer`'s stack. Preserve
  the documented load-bearing order of Grid View, modal cards, and toast; the
  selection must be above the normal image but must not leak above unrelated
  modal surfaces.
- Any background crop/encode/copy work needs cancellation or staleness handling,
  a completion signal, and a `newTestUI` drain hook. Marshal UI changes through
  Fyne's UI queue.
- Update `ARCHITECTURE.md` in the implementation change when the new package is
  added.

## Acceptance Criteria

Each criterion names the command that will prove it. The test names below are
part of the implementation contract.

### AC1 - Menu, Accelerator, and Availability

The Actions menu contains **Copy selection** in the specified order, displays
and binds `Option`/`Alt+Shift+C`, and enables the command only in the settled
normal-viewer states.

```sh
go test ./internal/ui/menus ./internal/ui -run 'Test(ActionsMenu_CopySelection|CopySelectionAvailability|WireCopySelectionShortcut)$'
```

### AC2 - Mode Entry, Visuals, and Button

Activation starts a fresh mode, hides/restores the information overlay, applies
the required cursor and selection visuals, and shows the lower-right button
only for a committed rectangle. The real viewer stack matches its accepted
golden image.

```sh
go test ./internal/ui/copyselection -run 'Test(ModeActivation|VisualState|CursorState|CopyButtonVisibility)$' && go test ./internal/ui -run 'Test(E2E_CopySelection|CopySelectionInfoOverlay)$'
```

Regenerate a changed golden only with `make golden`, inspect
`internal/ui/testdata/failed/*.png`, and accept the intended render before
running the command above.

### AC3 - Drawing, Replacement, Movement, and Resizing

The selection obeys the complete pointer contract, image bounds, minimum size,
invalid-replacement rule, move clamping, eight handle roles, and handle crossing.

```sh
go test ./internal/ui/copyselection -run 'Test(DrawSelection|InvalidReplacement|MoveSelection|ResizeSelection|CrossedResizeHandle)$'
```

### AC4 - Pixel Bounds and Viewport Independence

The rectangle rounds outward to image pixels and stays attached to the same
pixel region through zoom, pan, resize, and display scaling, including when
part of it moves outside the viewport.

```sh
go test ./internal/ui/copyselection -run 'Test(PixelBounds|ViewportTransform|HiDPIGeometry)$' && go test ./internal/ui -run 'TestCopySelectionSurvivesZoomPanAndResize$'
```

### AC5 - Clipboard Pixel Fidelity

The PNG contains exactly the selected oriented pixels at content resolution,
preserves alpha, and excludes every UI element. Rotation, animation, SVG, and
RAW-preview rules are covered explicitly.

```sh
go test ./internal/ui/copyselection ./internal/ui -run 'TestCopySelection(Pixels|Transparency|Rotation|AnimatedFrame|SVG|RAWPreview)$'
```

### AC6 - Copy Lifecycle and Failure Recovery

Copy work runs without blocking the UI, locks the mode while pending, exits
silently only after success, and retains an editable selection after a
recoverable crop, encoding, or clipboard failure.

```sh
go test ./internal/ui/copyselection ./internal/ui -run 'TestCopySelection(Busy|Success|EncodeFailure|ClipboardFailure)$'
```

### AC7 - Keyboard and Cross-Feature Lifecycle

`Escape`, `Return`/`Enter`, navigation suppression, focus preservation, repeated
activation, and cancellation before every non-viewport command follow the
settled transient-mode rules.

```sh
go test ./internal/ui -run 'TestCopySelection(Keyboard|RepeatedActivation|FocusLoss|CancelsBeforeOtherCommands)$'
```

### AC8 - Translation and Manual Coverage

Every locale contains the new visible strings, English remains an identity map,
both manuals document the menu path, shortcut, interaction, output, and limits,
and no Unicode arrow reaches translated or manual content.

```sh
go test . -run 'TestTranslations_(EveryLocaleCoversEnglish|EnglishMapsEachKeyToItself|NoArrowFollowedByASpace)$' && go test ./internal/ui -run TestTranslationsHaveNoUnicodeArrows && go test ./internal/ui/help -run 'Test(ManualDocumentsCopySelection|ManualHasNoUnicodeArrows)$'
```

### AC9 - Architecture Map

The package map records the new feature package and its boundary.

```sh
rg -n 'internal/ui/copyselection' ARCHITECTURE.md
```

### AC10 - Complete Repository Gate

Formatting, the TUF-root check, vet, build, and the full race-enabled suite all
pass before handoff.

```sh
make verify
```

## Documentation Decisions

- Add `Copy selection` and `Copy to clipboard` as exact English keys in every
  translation bundle during implementation.
- Update both `internal/ui/help/manual.md` and
  `internal/ui/help/manual_de.md` during implementation. Document the Actions
  menu placement, platform shortcut names, drawing/editing interaction,
  `Escape` and `Return`, full-resolution output, and the format limitations.
- Use ASCII `->` for menu paths in manuals; do not introduce Unicode arrows.
- Do not update the manuals before the feature exists in the application.

## Out of Scope

- Cropping, saving, or otherwise modifying the source file.
- Selecting in Grid View or Picture Frame mode.
- Multiple rectangles, non-rectangular selections, aspect-ratio locks, centered
  resizing, fixed presets, or a dimension readout.
- Remembering a selection per image or across mode activations or sessions.
- Keyboard-driven selection creation, movement, or resizing.
- Copying a screenshot, UI overlays, file references, metadata, or additional
  clipboard formats.
- Adding a new clipboard implementation or changing the existing `Cmd+C` /
  `Ctrl+C` Copy image and Copy image path behavior.
- Full RAW development, uncapped SVG rasterization, exact restoration of an
  animated image's timing phase, or a new maximum crop size.

## Honest Limits

- Camera RAW output is limited to the embedded preview PicFetch already shows.
- SVG output remains subject to PicFetch's vector raster safety ceiling.
- An animated image resumes after the mode ends, but not necessarily at the
  exact timing phase it had before freezing.
- Very large crops still depend on available process memory and the operating
  system clipboard's capacity. Failure leaves the selection available for
  retry; this feature does not guarantee that every theoretically valid crop
  can be copied.
- Process-level memory exhaustion can terminate a Go process and cannot be
  guaranteed to reach the recoverable error-toast path.
- Linux continues to depend on the clipboard tools already documented for
  ordinary image copying.
