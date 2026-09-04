# Export the current mosaic as PNG or JPEG

Type: task
Status: resolved
Priority: P0
Blocked by: 08

## Goal

Save the latest completed mosaic pixels through PicFetch's existing native
picker and atomic image encoder without reading viewer display state.

## Existing code anchors

- `internal/ui/export.go` captures source/pixels on the UI goroutine, runs
  `filepicker.ChooseSave` off-thread, parses it with `ParseFileList`, and calls
  `imaging.Export`.
- `imaging.Export(dest, img, src)` writes atomically. Passing a JPEG source
  copies its metadata, so mosaic export must pass `nil` as `src`.
- Existing `exportDestination` is unexported and intentionally accepts every
  format `imaging` can encode. Mosaic's product interface is narrower and must
  not move this helper across packages merely to reuse it.

## Scope

- Put export behavior in `mosaicwin`, which owns the Save Image action. Offer
  PNG first/default and JPEG second using the existing focusable choice-panel
  pattern where a format choice is needed.
- Capture the current `mosaic.Result`, chosen format, source-snapshot directory,
  and suggested name before starting the worker. Include picker/encode work in
  ticket 06's worker/`Settle` accounting and marshal status/action changes via
  the window's `UIQueue`.
- Call `filepicker.ChooseSave` with a full suggested path. Use the first
  snapshotted source's directory, matching the existing export's useful start
  location, and basename `PicFetch-Mosaic-YYYYMMDD-HHmmss` from a per-window
  injectable clock.
- Parse the picker output with `filepicker.ParseFileList`. Accept typed `.png`,
  `.jpg`, or `.jpeg`; otherwise append the chosen extension so bytes never
  masquerade under an unsupported suffix.
- Call `imaging.Export(destination, resultPixels, nil)`. Do not regenerate,
  inspect `viewer.img`, or pass any source URI for metadata copying.
- Treat an empty picker result as cancellation with no toast/error. Log and
  present picker/encode failures without closing or replacing the preview.
- Add/translate status text in both catalogues in this ticket and update
  Qodana for new test paths.

## Acceptance Criteria

- PNG and JPEG decode at the exact target dimensions. PNG pixels equal the
  current result; JPEG is checked with a documented lossy tolerance rather
  than impossible byte/pixel equality.
- A fake exporter or fixture proves the captured pixels belong to the current
  completed generation even if regeneration starts after the click.
- Cancel writes nothing and reports no error; picker and write failures retain
  the preview and re-enable its actions.
- The fake clock/picker pins directory, timestamp, default format, typed
  PNG/JPEG override, missing extension, and unsupported extension behavior.
- Exported JPEG contains no EXIF copied from any source.

```sh
go test ./internal/ui/mosaicwin -run 'TestMosaicExport' &&
go test ./internal/imaging -run 'TestExport' &&
go test . -run 'TestTranslations_' &&
make check-qodana-test-exclusions
```

## Non-Goals

- Exporting GIF, BMP, TIFF, AVIF, or source metadata
- Refactoring ordinary viewer export without a proven shared abstraction

## Comments

`imaging.Export` already supplies atomic temp-file-then-rename behavior; the
ticket must reuse it rather than open/write the destination directly.

Implemented and verified on 2026-09-04: picker naming/extensions, captured-result
identity, exact PNG, tolerance-bounded JPEG, metadata stripping, and stale completion are green.
