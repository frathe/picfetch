# 04: Drop dimension tags a resize invalidated

**What to build:** When a user exports with camera metadata **included** and an
export size limit **actually changed the pixels**, the exported file must not
carry tags claiming the original's dimensions. Those tags are **removed**, not
rewritten to new values: a reader then falls back to the JPEG frame header, which
already carries the true size, so the file stops making a claim rather than making
a second one that has to be kept correct.

The set is closed and deliberately narrow:

- IFD0: ImageWidth (0x0100), ImageLength (0x0101)
- Exif SubIFD, reached through the 0x8769 pointer: PixelXDimension (0xA002),
  PixelYDimension (0xA003), SubjectArea (0x9214), SubjectLocation (0xA214)
- Interoperability IFD, reached through the 0xA005 pointer — which lives in the
  Exif SubIFD, not IFD0: RelatedImageWidth (0x1001), RelatedImageLength (0x1002)

MakerNote and the resolution/DPI tags are explicitly **kept**. MakerNote may
contain pixel geometry but cannot be audited, and removing it on suspicion would
discard lens and body detail a user asked to keep. DPI states intended print
density rather than a pixel count, and dropping it would land photos at a default
density in layout applications.

Two triggering rules matter as much as the tag list. The trigger is that the
pixels actually changed, not that a rung was selected — choosing 2400 for an
1800px photo drops nothing. And the rotate-and-save path is untouched: it resizes
nothing, so its existing metadata normalization must keep behaving exactly as it
does today.

This is the only genuinely new machinery in the feature. The imaging module's
in-place TIFF patcher has so far only ever overwritten a value and zeroed a
pointer; removing an entry means decrementing the entry count, moving the
following entries back by one entry width, and rewriting the next-IFD pointer at
its new position. Entry value offsets are absolute from the TIFF start, so they
survive the shift untouched, and the freed trailing bytes are left dead —
consistent with that function's documented choice not to compact freed bytes.

If this proves too large for one sitting, it splits cleanly at the Interop hop:
IFD0 and Exif SubIFD tags first, the 0xA005 chase second.

**Blocked by:** 02

**Status:** ready-for-agent

- [ ] A resized export with metadata included carries none of the listed dimension tags, verified by reading tags back out of the written file
- [ ] Camera, lens, exposure, date and GPS tags are still present on that same file
- [ ] MakerNote is still present
- [ ] Resolution/DPI tags are still present
- [ ] An export at Original size drops nothing
- [ ] An export whose chosen limit exceeded the photo's own size drops nothing
- [ ] The Interoperability IFD's dimension tags are dropped, reached through the pointer in the Exif SubIFD rather than IFD0
- [ ] Removing an entry leaves every surviving entry's value readable, confirming absolute offsets survived the shift
- [ ] The rotate-and-save path's written output is unchanged, covered by its existing tests
- [ ] A malformed or truncated metadata block leaves the block unchanged rather than panicking, matching the existing walker's failure mode
