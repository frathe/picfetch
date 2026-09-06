# Dimension tags survive a rotated or EXIF-oriented export

Status: done

## Problem Statement

`imaging.Export` drops the Exif tags that record a photo's pixel dimensions
when an export size limit changed those pixels, so no program reads the file
as claiming a size it does not have. The trigger it uses only sees its own
scaling pass:

```go
resized := SizeLimitApplies(img.Bounds(), opts.MaxEdge)
```

Two other things change the written frame's dimensions, and neither fires it:

1. **A rotation in the viewer.** Export writes the frame on screen, and
   `v.img.Image` is already rotated. Exporting a 900x600 JPEG at Original
   size after one 90-degree turn writes 600x900 pixels while IFD0 `0x0100`
   still reads 900. Verified against a written file:

   ```
   written pixels 600x900; IFD0 ImageWidth bytes = 84 03 00 00   (0x384 = 900)
   decoded bounds of the written file = (0,0)-(600,900)
   ```

2. **A source whose EXIF Orientation is 5-8.** The decode path applies the
   orientation, so the frame handed to Export is already transposed relative
   to the bytes stored in the file, and the export normalizes Orientation to
   1 on the way out. `0x0100`/`0x0101` and `0xA002`/`0xA003` are left
   describing the stored frame, which is now the wrong way round.

This is pre-existing: export has copied source metadata verbatim since long
before the export-options work, and the size-limit trigger added in
`.scratch/export-options/` narrowed correctly for what that ticket asked but
inherited the gap.

## Solution

Ask the question the tags actually answer. A dimension tag lies when the
pixels being written are not the size the **source's own frame header**
records - so compare the written bounds against the source JPEG's SOF, not
against the scaler's input. That one predicate catches all three causes
(resize, viewer rotation, orientation normalization) with no new plumbing
between the viewer and the imaging module.

## Implementation Decisions

| Decision | Why | Do not relitigate |
|---|---|---|
| Compare against the source's SOF frame size | It is the one thing in the file that states what the copied tags describe, and it is already sitting in bytes Export has read | yes |
| The closed tag set is unchanged | Same tags, same reasons; only the trigger widens | yes |
| Export stays viewer-independent | The viewer does not report "I rotated this"; Export infers it from the pixels, so nothing new crosses the module boundary | yes |
| An unreadable SOF falls back to the size-limit trigger | A malformed source must not silently start dropping tags, and must not fail the export either | yes |
| The rotate-and-save path stays untouched | `SaveRotated` re-encodes in place at the same size; nothing there is invalidated | yes |

## Acceptance Criteria

```
AC1  A JPEG's stored frame size is read from its SOF, including a progressive
     source, and an unreadable one reports not-ok rather than a wrong number.
     go test ./internal/imaging/ -run TestJPEGFrameSize -v

AC2  Exporting a rotated frame at Original size with metadata included
     corrects the six tags that state a size to that frame, drops the two
     that state a coordinate, and keeps
     camera/lens/exposure/GPS/MakerNote/DPI. (Ticket 03 turned this from
     dropping all eight into correcting six of them.)
     go test ./internal/imaging/ -run TestExport_ -v

AC3  Exporting a source whose Orientation is 5-8 does the same.
     go test ./internal/imaging/ -run TestExport_ -v

AC4  An export whose written frame matches the source's stored frame
     changes nothing - unrotated Original size, and a 180-degree turn, which
     changes no dimension.
     go test ./internal/imaging/ -run TestExport_ -v

AC5  A source whose frame header cannot be read still exports, falling back
     to the size-limit trigger.
     go test ./internal/imaging/ -run TestExport_ -v

AC6  The viewer path: rotate, export at Original size, read the tags back off
     the written file.
     go test ./internal/ui/ -run TestExportAs_ -v

AC7  SaveRotated's written output is unchanged.
     go test ./internal/imaging/ -run TestSaveRotated -v

AC8  Nothing else regressed, golden rendering included.
     make test
```

## Non-goals

- ~~**Rewriting the tags to correct values.** They are dropped, as before.~~
  **Superseded by ticket 03.** This was inherited from the export-options
  spec, which justified dropping as avoiding a second source of truth that
  would have to be kept correct. On review that argument does not hold: the
  value would be derived from the frame being written, at the moment it is
  written, so there is nothing to keep in sync afterwards. What does survive
  from it is the failure mode - a wrong dimension is worse than an absent
  one - and ticket 03 answers that by removing any tag it cannot correct
  honestly rather than guessing at one.
- **Widening the tag set.** MakerNote and DPI stay, for the reasons
  `.scratch/export-options/issues/04` gives.
- **Telling the imaging module that a rotation happened.** The pixels say it.
- **Changing `SaveRotated`, the wallpaper path, or the mosaic path.**
- **The UI's suggested filename and toast**, which report the size limit
  specifically and should keep doing exactly that: a rotation is not a size
  limit and must not add a `-2400` suffix.

## The honest limit

Ticket 03 narrows this: the six pure dimension tags are corrected rather
than dropped, so the limit below applies to the two coordinate tags only.

A 180-degree turn changes no dimension, so nothing is dropped - correct for
the width/height tags, but `SubjectArea` (`0x9214`) and `SubjectLocation`
(`0xA214`) are coordinates *within* the frame and a 180-degree turn does
invalidate those. Catching that would mean Export knowing that a rotation
happened rather than inferring it from dimensions, which is a viewer concept
crossing into a viewer-independent module. Both tags are rare, and the
dimensions they would be read against are still honest. Left alone
deliberately.
