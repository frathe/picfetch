# Render source images and frame styles

Type: task
Status: resolved
Priority: P0
Blocked by: 01, 04

## Goal

Implement the deep module's one generation operation, reusing PicFetch's
canonical decode behavior and returning exact target-sized pixels.

## Existing code anchors

- `imaging.ReadAndProbe(ctx, uri)` reads bytes and returns EXIF-oriented
  display bounds; `imaging.DecodeLoaded(ctx, data, 0)` applies orientation,
  uses RAW previews, and intentionally freezes animated GIFs at the first
  frame.
- SVG is the exception: `LoadedImage.Vector.RasterAt(w, h)` can render at the
  card's required pixel size. Raster decoders do not support decode-at-size,
  so a full raster decode is sometimes unavoidable.
- `imaging.Export` is an output encoder, not a generation primitive, and copies
  JPEG metadata only when given a non-nil source. Mosaic generation must retain
  no source metadata.

## Scope

- Add `mosaic.Generate(context.Context, Request) (Result, error)` as the
  module's single operation, with implementation split into package-private
  loader/render files as needed.
- Shuffle the URI snapshot before any probe. Probe and decode lazily in that
  order, stopping once the layout is covered; a 10,000-entry pool must not be
  fully opened or decoded for a typical display.
- Reuse `ReadAndProbe` and `DecodeLoaded(..., 0)`. Rasterize SVG directly for
  the required card pixels. Downsample decoded rasters with `x/image/draw` and
  release source bytes/full-size pixels as soon as the last placement using
  them is rendered.
- Keep a generation-local cache only for sources actually repeated by the
  covering plan. Do not introduce another general decoder, viewer cache, or
  unbounded pool-wide cache.
- Skip unreadable/disappeared sources, continue into the shuffled tail, and
  return a typed aggregate error only if no source can be rendered. Preserve
  `context.Canceled`/`DeadlineExceeded` rather than wrapping them as source
  failures.
- Render with `draw.Over`, preserving source aspect ratio and alpha without an
  internal crop or stretch. Apply the selected frame to every card and ensure
  the exact border/footer/shadow footprint matches ticket 04's occupancy.
- Return an ordinary 8-bit Go image interpreted as sRGB by the existing
  PNG/JPEG encoders. Do not claim ICC-profile conversion: PicFetch's current
  decode pipeline performs none.
- Add deterministic image fixtures/goldens under `internal/mosaic/testdata`
  and add every new test path to Qodana.

## Acceptance Criteria

- Output bounds are exactly `(0,0)-(targetWidth,targetHeight)` and the rendered
  result has no untouched canvas pixel after compositing.
- EXIF orientations, the first animated frame, SVG at card resolution, RAW
  preview, transparent raster, portrait, and landscape fixtures follow the
  same behavior as `internal/imaging`.
- None, Thin light, Thin dark, and Polaroid have deterministic reference images;
  frame and shadow pixels agree with their layout footprints.
- A broken source followed by a readable source succeeds; an all-broken pool
  reports the attempted sources without exposing implementation internals.
- A counting loader proves excess entries in a 10,000-entry pool are not probed
  or decoded after coverage.
- Source files are byte-for-byte unchanged before and after generation.

```sh
go test ./internal/mosaic -run 'TestGenerate_(SourceFidelity|FrameStyles|Unreadable|LazyPool|SourcesUnchanged)'
```

## Non-Goals

- Reusing or mutating the viewer's `imgCache`
- Color-profile conversion beyond the existing decoder behavior
- Export or wallpaper I/O

## Comments

The original "decode at needed size" wording was narrowed: only SVG supports
that today. Raster memory is controlled by lazy loading, bounded concurrency,
downsampling, and prompt release.

Implemented and verified on 2026-09-04: canonical EXIF/GIF/SVG/RAW/alpha loading,
lazy 10,000-source behavior, source immutability, and fixed frame references are green.
