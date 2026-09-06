# 02: Drop dimension tags whenever the written frame stopped matching them

**What to build:** `imaging.Export` currently decides whether the source's
dimension tags have been invalidated by asking whether its *own* scaling pass
changed anything:

```go
resized := SizeLimitApplies(img.Bounds(), opts.MaxEdge)
```

Replace that with the question the tags actually answer - whether the pixels
about to be written are the size the source's own frame header records:

```go
// dimensionTagsInvalidated reports whether written - the bounds of the
// pixels about to be encoded - differs from the frame orig's own header
// records, i.e. whether orig's dimension tags would now describe a file that
// no longer matches them. fallback is the answer for a source whose frame
// header cannot be read.
func dimensionTagsInvalidated(written image.Rectangle, orig []byte, fallback bool) bool
```

called as

```go
dimensionTagsInvalidated(out.Bounds(), orig, SizeLimitApplies(img.Bounds(), opts.MaxEdge))
```

where `out` is the already-scaled image `Export` writes and `orig` the source
bytes it already read. `jpegFrameSize` (ticket 01) is how it reads the frame.

This widens the trigger to three causes from one, all of which produce a file
whose tags lie today:

- an export size limit that resized (what it already caught),
- **a rotation in the viewer** - `Export` writes the frame on screen and
  `v.img.Image` is already rotated,
- **a source whose EXIF Orientation is 5-8** - the decode path applies it, so
  the frame handed over is transposed relative to the stored bytes, and the
  export normalizes Orientation to 1 on the way out.

Nothing else changes: same closed tag set, same removal machinery, MakerNote
and DPI still kept, `SaveRotated` still passing `false`, and the UI's
suggested filename and toast still reporting the *size limit* specifically -
a rotation is not a size limit and must not add a `-2400` suffix to anything.

**Blocked by:** 01

**Status:** done

- [x] A rotated frame exported at Original size with metadata included carries none of IFD0 `0x0100`/`0x0101`, Exif SubIFD `0xA002`/`0xA003`/`0x9214`/`0xA214`, Interop `0x1001`/`0x1002`
- [x] A source with Orientation 6 exported at Original size drops the same set
- [x] Camera, lens, exposure, date, GPS, MakerNote and the resolution/DPI tags survive both
- [x] An unrotated Original-size export of an Orientation-1 source still drops nothing
- [x] A 180-degree turn drops nothing: it changes no dimension
- [x] A source whose frame header cannot be read still exports, falling back to the size-limit answer
- [x] The viewer path is covered end to end: rotate, export at Original size, read the tags back off the written file
- [x] The suggested filename and the completion toast are unchanged by a rotation
- [x] `go test ./internal/imaging/ -run 'TestExport_|TestSaveRotated' -v` and `go test ./internal/ui/ -run 'TestExportAs_|TestExportPrompt_' -v` pass
- [x] Every test added to `internal/ui` gets a row in `.github/testshards/internal-ui.tsv` and every new `_test.go` file a `qodana.yaml` entry (`make sync-qodana-test-exclusions`)

## Comments

2026-09-06 - Delivered by a go-expert subagent, test-first. It also caught a shadowing trap while wiring this up: the encode closure took a parameter named img, which shadowed Export's own img - the pre-scale bounds the fallback needs - with the already-scaled frame; the parameter is now named frame. On review the predicate was hoisted out of the closure so it is computed once under names for both frames, and the fixture and tag reader the viewer-level test needed moved into internal/uitest (DimensionTaggedJPEG, ExifIFD0HasTag) rather than being hand-rolled a third time in internal/ui.
