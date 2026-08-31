# 07: Preserve source fidelity across supported formats

**What to build:** Finish comparison with the same source-format fidelity and
safety guarantees as the normal viewer, including vector rerasterization,
canonical orientation, RAW previews, frozen animation, and explicit failures
under existing input limits.

**Blocked by:** 06: Preserve comparison state across transitions

**Status:** ready-for-agent

## Acceptance criteria

- [ ] Raster sources retain their full decoded resolution throughout zoom,
  pan, resize, layout switching, and Swap. Comparison never substitutes a grid
  thumbnail or silently reduces source quality.
  Verify: `go test ./internal/ui/... -run 'CompareRasterFidelity' -count=1`
- [ ] Each SVG is rerasterized for its current effective display size as zoom,
  layout, or window size changes. Superseded work is cancellable/stale-safe and
  cannot paint after another change or after comparison closes.
  Verify: `go test ./internal/ui/... ./internal/imaging/... -run 'CompareVector|RasterAt' -count=1`
- [ ] RAW files use the embedded preview selected by the normal imaging path,
  and animated inputs stay frozen on their first decoded frame for the entire
  comparison session.
  Verify: `go test ./internal/ui/... ./internal/imaging/... -run 'Compare(RAW|Animated)|RAWPreview' -count=1`
- [ ] Both sources use their canonical EXIF-corrected orientation. Temporary
  viewer-only rotation is ignored and cannot leak into comparison geometry or
  pixels.
  Verify: `go test ./internal/ui/... ./internal/imaging/... -run 'CompareOrientation|Orientation' -count=1`
- [ ] Existing encoded-input and vector-raster limits remain enforced. Two
  simultaneous full-resolution decodes may consume their combined memory; if
  either cannot complete, comparison follows its normal failure path instead
  of downsampling or removing a source.
  Verify: `go test ./internal/ui/... ./internal/imaging/... -run 'Compare(Memory|InputLimit|Failure)|InputTooLarge' -count=1`
- [ ] All comparison background work participates in deterministic test
  cleanup, and cancellation leaves no late UI completion or running worker.
  Verify: `go test -race ./internal/ui/... -run 'Compare(Cancel|Settle|Stale)' -count=1`
- [ ] Both manuals describe the complete comparison workflow and all introduced
  strings have locale parity with no Unicode arrows in app-drawn content.
  Verify: `go test ./... -run 'Translations|Manual|UnicodeArrows' -count=1`

## Comments
