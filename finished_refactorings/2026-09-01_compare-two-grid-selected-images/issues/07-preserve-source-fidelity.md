# 07: Preserve source fidelity across supported formats

**What to build:** Finish comparison with the same source-format fidelity and
safety guarantees as the normal viewer, including vector rerasterization,
canonical orientation, RAW previews, frozen animation, and explicit failures
under existing input limits.

**Blocked by:** 06: Preserve comparison state across transitions

**Status:** ready-for-human

## Acceptance criteria

- [x] Raster sources retain their full decoded resolution throughout zoom,
  pan, resize, layout switching, and Swap. Comparison never substitutes a grid
  thumbnail or silently reduces source quality.
  Verify: `go test ./internal/ui/... -run 'CompareRasterFidelity' -count=1`
- [x] Each SVG is rerasterized for its current effective display size as zoom,
  layout, or window size changes. Superseded work is cancellable/stale-safe and
  cannot paint after another change or after comparison closes.
  Verify: `go test ./internal/ui/... ./internal/imaging/... -run 'CompareVector|RasterAt' -count=1`
- [x] RAW files use the embedded preview selected by the normal imaging path,
  and animated inputs stay frozen on their first decoded frame for the entire
  comparison session.
  Verify: `go test ./internal/ui/... ./internal/imaging/... -run 'Compare(RAW|Animated)|RAWPreview' -count=1`
- [x] Both sources use their canonical EXIF-corrected orientation. Temporary
  viewer-only rotation is ignored and cannot leak into comparison geometry or
  pixels.
  Verify: `go test ./internal/ui/... ./internal/imaging/... -run 'CompareOrientation|Orientation' -count=1`
- [x] Existing encoded-input and vector-raster limits remain enforced. Two
  simultaneous full-resolution decodes may consume their combined memory; if
  either cannot complete, comparison follows its normal failure path instead
  of downsampling or removing a source.
  Verify: `go test ./internal/ui/... ./internal/imaging/... -run 'Compare(Memory|InputLimit|Failure)|InputTooLarge' -count=1`
- [x] All comparison background work participates in deterministic test
  cleanup, and cancellation leaves no late UI completion or running worker.
  Verify: `go test -race ./internal/ui/... -run 'Compare(Cancel|Settle|Stale)' -count=1`
- [x] Both manuals describe the complete comparison workflow and all introduced
  strings have locale parity with no Unicode arrows in app-drawn content.
  Verify: `go test ./... -run 'Translations|Manual|UnicodeArrows' -count=1`

## Comments

- 2026-09-01: Comparison now retains pane-local rendered frames without
  mutating the cached `LoadedImage`. Raster formats keep the canonical decoded
  first frame; SVG panes rerasterize at their current device-pixel display
  target after zoom, layout, resize, and Swap, with the existing pixel clamp,
  debounce, cancellation, and stale-target rejection.
- 2026-09-01: The comparison feature now publishes load and vector completions
  through a per-instance `UIQueue`. `Settle` waits load workers and both pane
  raster waitgroups, drains completions, and repeats when applying a load starts
  downstream vector work. The assembled test viewer installs the drainable
  queue, keeping Fyne's inline test driver race-free.
- 2026-09-01: Real-loader guards cover RAW embedded previews, a frozen first GIF
  frame, canonical EXIF orientation despite viewer-only rotation, two full
  decodes beyond a one-byte cache budget, and encoded-input-limit failure with
  both selected files preserved. Both manuals now document these guarantees.
- 2026-09-01: Every new guard was observed failing against a deliberate source,
  reraster, limit, stale/cancel, settle, format, or manual regression and then
  restored green. All seven acceptance commands and the complete comparison
  suite passed. `make verify` passed TUF validation, vet, build, and the full
  Linux/amd64 race suite (`internal/ui` 649.604s; `internal/ui/compare`
  14.701s).
