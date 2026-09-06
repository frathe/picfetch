# 03: Correct the dimension tags instead of dropping them

**What to build:** When the trigger from ticket 02 fires, rewrite the tags
that can be made true to the written frame's real size, and drop only the
ones that cannot. Today all eight go.

The trigger itself is settled and must not change: `dimensionTagsInvalidated`
already answers "do the pixels being written still match the frame the
source's own header records", and covers a resize, a viewer rotation and an
Orientation 5-8 source. This ticket changes only what happens *when it says
yes*.

## Patched, not dropped

Every one of these is a claim about the image's size, and the written frame
is the answer:

- IFD0: `0x0100` ImageWidth, `0x0101` ImageLength
- Exif SubIFD: `0xA002` PixelXDimension, `0xA003` PixelYDimension
- Interoperability IFD: `0x1001` RelatedImageWidth, `0x1002` RelatedImageLength

All six are count-1 SHORT or LONG, so the value lives inline in the entry's
own 4 bytes and the patch is the same in-place overwrite `patchSavedTIFF`
already does for Orientation - no entry shifting, no count change, no
pointer relocation.

The Interop pair is the least obvious of the six: those tags describe the
related image in an interoperability set rather than this file directly.
They are patched on the same reasoning anyway - they are a dimension claim
about the same picture, and a reader that trusts them should not be handed a
number that disagrees with the frame header.

## Still dropped

- `0x9214` SubjectArea and `0xA214` SubjectLocation. These are *coordinates
  inside* the frame, not dimensions. Correcting them needs the actual
  transform - rotation angle, scale factor - which `Export` deliberately does
  not know: it infers that something changed from the pixels, not what
  changed. A rotated coordinate cannot be derived from a new width and
  height, so these keep going.

## Still dropped as a fallback

Removal does not go away; it becomes the answer for any entry that cannot
hold the truth honestly:

- a declared type that is neither SHORT nor LONG,
- a count other than 1,
- a SHORT entry whose new value exceeds 65535.

**A tag must never be left holding a wrong number.** If it cannot be
corrected, it is removed - that is the whole safety property this ticket
trades against, since a confidently wrong dimension is worse than an absent
one.

## Contract

The plumbing currently threads a `bool` from `Export` down to the patcher.
Widen it to carry the truth instead of a flag, with the zero value meaning
"the tags still describe this file, leave them alone" - a written frame is
never 0x0, so the zero value is unambiguous:

```go
func encodeJPEGPreservingMetadata(w io.Writer, img image.Image, orig []byte, corrected image.Point) error
func normalizeSavedExif(app1 []byte, corrected image.Point) []byte
func patchSavedTIFF(tiff []byte, corrected image.Point)

// patchIFDDimension overwrites the inline value of tag in the IFD at
// ifdOffset with v, honouring the entry's declared type. It reports false
// when the entry cannot hold v honestly - an unexpected type, a count other
// than 1, or a SHORT that v overflows - so the caller removes the entry
// rather than leaving a wrong number in it.
func patchIFDDimension(tiff []byte, bo binary.ByteOrder, ifdOffset uint64, tag uint16, v int) bool
```

and in `Export`, where the bool is computed today:

```go
var corrected image.Point
if dimensionTagsInvalidated(out.Bounds(), orig, SizeLimitApplies(img.Bounds(), opts.MaxEdge)) {
    corrected = image.Pt(out.Bounds().Dx(), out.Bounds().Dy())
}
```

`SaveRotated` passes the zero value, exactly as it passes `false` today.

## Existing tests this deliberately changes

These assert today that the tags are **absent** after an invalidating
export. Six of the eight will now be **present and correct** instead, so
their expectations change with the behaviour - this is the intended change,
not a regression to work around:

- `TestExport_ResizeDropsTheDimensionTagsItInvalidated`
- `TestExport_RotatedFrameAtOriginalSizeDropsTheDimensionTags`
- `TestExport_Orientation5Through8SourceAtOriginalSizeDropsTheDimensionTags`
- `TestExport_FallsBackToTheSizeLimitAnswerWhenTheFrameHeaderCannotBeRead` -
  its signal is currently "is `0x0100` present"; it becomes "does `0x0100`
  read the source's size or the written one", which is a sharper test
- the `assertDimensionTagsDropped` helper in `save_test.go`
- `TestExportAs_RotationDropsDimensionTagsButNotFilenameOrToast` in
  `internal/ui/export_test.go`

`TestExport_KeepsDimensionTagsWhenThePixelsDidNotChange`,
`TestExport_180DegreeTurnDropsNothing` and
`TestExport_UnrotatedOrientation1OriginalSizeExportDropsNothing` must keep
passing untouched: when nothing was invalidated, nothing is patched either.

**Blocked by:** 02

**Status:** done

- [x] A resized export's `0x0100`/`0x0101`/`0xA002`/`0xA003`/`0x1001`/`0x1002` all read the written frame's real size, read back off the file
- [x] A rotated Original-size export's six read the written (transposed) size
- [x] An Orientation 5-8 source's six read the written (upright) size
- [x] `0x9214` SubjectArea and `0xA214` SubjectLocation are still removed on every one of those
- [x] A SHORT entry too small to hold the new value is removed rather than left wrong, proven against the patcher with a corrected edge of 70000 (an *export* cannot reach this: JPEG itself caps a frame at 65535)
- [x] An entry with an unexpected type or a count other than 1 is removed rather than patched
- [x] A patched SHORT stays SHORT and a patched LONG stays LONG - the entry's declared type is not rewritten
- [x] Camera, lens, exposure, date, GPS, MakerNote and the resolution/DPI tags still survive
- [x] An export that invalidated nothing patches nothing and removes nothing
- [x] `SaveRotated`'s written output is unchanged
- [x] Every surviving entry's value is still readable afterwards, including ones whose value lives in the value area at an absolute offset
- [x] `go test ./internal/imaging/ -run 'TestExport_|TestSaveRotated|TestNormalizeSavedExif|TestRemoveIFDEntries' -v` and `go test ./internal/ui/ -run 'TestExportAs_' -v` pass
- [x] `make test` passes (Linux/amd64 Docker)

## Why this is worth doing

The file stops being merely honest and becomes useful: anything that reads
Exif dimensions rather than the JPEG frame header - `exiftool`, catalogue
importers, some CMS upload paths - gets the right answer instead of nothing.

The original export-options spec listed "rewriting dimension tags to correct
values" as out of scope, on the grounds that it would mean maintaining a
second source of truth. That argument is thinner than it looked: the value is
derived from the frame being written, at the moment it is written, so there
is nothing to keep in sync afterwards. What survives from that reasoning is
the failure mode, and this ticket answers it directly - anything that cannot
be corrected is removed, so a bug can still only ever cost a tag, never
invent a wrong one.

## Comments

2026-09-06 - Implemented inline (the Lead already held this code's context, so
a subagent would have rebuilt it cold). One thing the ticket did not predict:
patching walks entry by entry, so it would have rewritten bytes inside a
*truncated* IFD that `removeIFDEntries` refuses to touch - caught by
`TestNormalizeSavedExif_CorrectionSurvivesAMalformedBlock`, which was written
for the removal path and turned out to bind this one too. `patchIFDDimension`
now applies the same all-or-nothing rule: an IFD whose declared entries do not
all fit is left completely alone.

Review then found a real defect: the first draft patched only the *first*
entry matching a tag, while removal took every match - so a file repeating
0x0100 would have kept the second copy at the old size and reported success,
the exact outcome the safety rule forbids. Every match is now corrected, one
failure condemns them all, and
TestPatchSavedTIFF_CorrectsEveryCopyOfARepeatedTag pins both halves. The same
review pass unified the three hand-rolled IFD walks behind ifdEntryOffsets,
which also closed an inconsistency where savedIFDPointer would follow a
pointer out of a truncated IFD the writers refuse to touch.
