# Export options: export size limit + metadata omission

Status: done

## Problem Statement

Exporting a photo out of PicFetch today asks exactly one question: PNG or JPEG.
Everything else about the written file is fixed. That leaves two things a person
routinely wants and cannot get:

1. **The file is too big to send.** A modern camera or phone photo exports at full
   sensor resolution. Mailing it, attaching it to a ticket, or dropping it into a
   web page means taking it somewhere else first to shrink it.
2. **The file carries more than the picture.** A JPEG export currently receives a
   normalized copy of the source's metadata segments — deliberately, so a rotate
   round-trip does not discard camera settings. But that means GPS coordinates,
   capture timestamps, and camera serial data ride along to whoever receives the
   file, with nothing in the export flow saying so or offering otherwise.

The app already knows how to remove metadata — but only as **metadata removal**:
an irreversible in-place rewrite of the original, reached from the EXIF window
behind a "this cannot be undone" confirmation. That is the wrong instrument for
"send a clean copy to someone and keep my original intact."

## Solution

The export prompt grows two controls beside the format choice:

- **Export size limit** — Original / 2400 / 1600 / 1000, applied to the longest
  edge. A photo already inside the ceiling exports at its own size; nothing is
  ever enlarged to meet the limit.
- **Include camera metadata** — checked by default, preserving today's behavior.
  Unchecking it performs **metadata omission**: the exported copy is written
  without the source's identifying tags, and the source file keeps everything
  it had.

Together these turn export from "write this in another format" into "prepare
this photo for the web or for mail" — one dialog, one keystroke to commit,
original untouched either way.

## User Stories

1. As someone mailing a photo to family, I want to cap the exported copy at 1600px, so that the attachment sends without bouncing off a size limit.
2. As someone posting to a web page, I want a 2400px export, so that the image looks sharp on a high-DPI screen without shipping a 24-megapixel file.
3. As someone attaching a screenshot-like reference to a ticket, I want a 1000px export, so that the file is small enough to upload quickly and still readable.
4. As someone exporting a photo I intend to print or archive, I want an Original option, so that the export path never silently costs me resolution.
5. As someone who picks a 2400px limit on a photo that is already 1800px wide, I want the photo exported at 1800px, so that the app never invents pixels that were not captured.
6. As a privacy-conscious sender, I want to uncheck "Include camera metadata", so that the recipient does not receive the GPS coordinates of my home.
7. As a privacy-conscious sender, I want that choice to leave my original file untouched, so that I keep my own capture data while sharing a clean copy.
8. As someone who has used the EXIF window's "Remove Metadata", I want export's control to be worded differently, so that I do not believe exporting is about to rewrite my original.
9. As someone exporting to PNG, I want the metadata control to say it applies to JPEG only, so that I am not left guessing whether my choice did anything.
10. As an existing user, I want the prompt to open with today's behavior selected, so that an upgrade does not silently change what my exports contain.
11. As someone who exports a resized copy, then exports a different photo, I want the second export to open at Original again, so that a limit I set once cannot silently shrink a photo months later.
12. As someone who exports a metadata-free copy, then exports again, I want the checkbox back at its default, so that the prompt is never in a state I have forgotten about.
13. As a keyboard-first user, I want the format buttons to still commit the export, so that Cmd/Ctrl+E followed by Return remains a two-keystroke export.
14. As a keyboard-first user, I want to reach the size and metadata controls without a mouse, so that the whole flow stays on the keyboard.
15. As a keyboard-first user, I want Escape to cancel the whole prompt, so that backing out costs one key.
16. As someone exporting a RAW file, I want the Original option to show the actual pixel dimensions, so that I can see it means the embedded preview's size rather than the sensor's.
17. As anyone using the prompt, I want the Original option to state the image's real longest edge, so that I can tell at a glance whether a smaller rung would change anything.
18. As someone exporting a resized copy into the source's own folder, I want the suggested filename to carry the size, so that it does not collide with the original I am exporting from.
19. As someone exporting at Original size, I want the suggested filename unchanged, so that the common case behaves exactly as it does today.
20. As someone who just exported, I want the confirmation to state the size and metadata outcome when they differ from the default, so that I have a receipt of what actually went out.
21. As someone who exported at defaults, I want the short confirmation I get today, so that routine exports are not noisier than before.
22. As someone who typed a filename ending in a different format than the button I pressed, I want the typed extension to still win, so that a file's bytes always match its name.
23. As someone whose typed extension overrode my format choice, I want the confirmation to report what was actually written, so that the override is visible rather than silent.
24. As someone exporting a resized JPEG with metadata included, I want the embedded dimension tags dropped rather than left stale, so that no program reads the file as claiming a size it does not have.
25. As someone exporting a resized JPEG, I want my camera, lens, exposure, and GPS tags preserved when I asked for them, so that dropping the dimension tags does not cost me the metadata I chose to keep.
26. As someone exporting at Original size, I want every metadata tag preserved exactly as today, so that nothing is dropped when nothing has changed.
27. As someone who picked a size limit larger than my photo, I want no tags dropped, so that the file is only modified when its pixels actually were.
28. As someone using Save Changes after a rotation, I want that path completely unaffected, so that a feature added to export cannot change what saving a rotation writes.
29. As someone exporting a photo with an embedded color profile, I want the profile preserved even when I omit metadata, so that the colors do not shift for the recipient.
30. As someone exporting a photo whose orientation tag is not 1, I want the exported copy upright, so that omitting metadata does not leave the picture sideways.
31. As someone exporting an animated image, I want the frame on screen written as a still, so that the behavior matches what export already does today.
32. As someone exporting a WebP or HEIC, I want the size and metadata controls to work the same way, so that decode-only formats are not second-class in the export flow.
33. As a maintainer, I want the resize to use an export-grade interpolation, so that a photo someone is about to email is not visibly soft.
34. As a maintainer, I want the thumbnail cache's own scaling left on its faster kernel, so that adding export quality does not slow down grid population.
35. As someone who opens the export prompt while the delete confirmation is up, I want the existing guards to still hold, so that pressing Return cannot hit a hidden "Move to Trash".
36. As someone pressing Shift+Delete while the export prompt is up, I want that still refused, so that the two prompts cannot fight over the keyboard.
37. As a non-English user, I want the new labels translated, so that the prompt reads in my own language.
38. As someone exporting during a comparison, I want the prompt still refused, so that the new controls do not open a path that was deliberately closed.
39. As someone exporting mid-load, I want the prompt still refused, so that I cannot export the previous photo's pixels under the new photo's name.

## Implementation Decisions

### Scope

- **Single image only.** Export continues to act on the frame in the single-image
  viewer. Batch export over a **Grid selection** is explicitly a later feature: it
  needs a destination-folder picker, a collision-naming policy, and progress and
  cancellation for N re-encodes, none of which exist.
- **No JPEG quality control.** The re-encode quality stays fixed. PicFetch is a
  viewer, not an editor; a quality slider is a separate decision if export file
  size proves to be the real complaint.

### Export size limit

- Semantics are longest-edge, aspect preserved, **never upscale** — identical to
  the rule the thumbnail path's edge-fitting helper already implements, which is
  reused rather than reimplemented.
- Rungs: Original, 2400, 1600, 1000.
- The Original rung's label carries the image's real longest edge, computed from
  the frame in hand. This is what makes a RAW export honest: the on-screen frame
  for a RAW is the camera's embedded JPEG preview, so "Original" means the
  preview's dimensions, and showing the number says so without a special case.
- Scaling for export uses a high-quality interpolator (CatmullRom class), **not**
  the approximate-bilinear kernel the thumbnail cache uses. The thumbnail path
  keeps its faster kernel unchanged; this is a new exported entry point in the
  imaging module, not a change to the existing one.

### Metadata omission

- Included metadata is today's behavior and the default: the source JPEG's
  segments are copied onto the exported copy with orientation normalized.
- Omitted metadata reuses the existing ICC-preserving encode path: the color
  profile is kept, identifying tags are not. Note the existing constraint that
  the Adobe APP14 segment must not be spliced back, since it would misdeclare
  the encoder's color transform.
- PNG carries no metadata either way. Rather than dynamically disabling the
  control, its label states the limitation permanently — "Include camera
  metadata (JPEG only)" — which stays honest without costing the fast path a
  keystroke.

### Dropping tags that would lie

When metadata is included **and the pixels actually changed size**, the tags that
would now make a false claim are removed rather than rewritten. Removing them
lets a reader fall back to the JPEG frame header, which carries the true size for
free; rewriting them would mean computing and maintaining a second source of
truth.

The set is closed, and deliberately narrow:

- **IFD0**: `0x0100` ImageWidth, `0x0101` ImageLength
- **Exif SubIFD** (via the `0x8769` pointer): `0xA002` PixelXDimension,
  `0xA003` PixelYDimension, `0x9214` SubjectArea, `0xA214` SubjectLocation
- **Interoperability IFD** (via the `0xA005` pointer, which lives in the Exif
  SubIFD, not IFD0): `0x1001` RelatedImageWidth, `0x1002` RelatedImageLength

Explicitly **kept**: MakerNote and the resolution/DPI tags. MakerNote may contain
pixel geometry but cannot be audited, and removing it on suspicion would discard
lens and body detail a user asked to keep. DPI states intended print density
rather than a pixel count; dropping it would land photos at a default density in
layout applications.

Triggering rules:

- The trigger is **pixels actually changed**, not "a limit was selected". Choosing
  2400 for an 1800px photo drops nothing.
- The rotate-and-save path is untouched. It resizes nothing, so its existing
  metadata normalization keeps today's behavior exactly.

Mechanically this extends the existing in-place TIFF patcher, which today only
overwrites the orientation value and zeroes the next-IFD pointer. Entry removal
is new: decrement the entry count, move following entries back one entry width,
and rewrite the next-IFD pointer at its new position. Entry value offsets are
absolute from the TIFF start, so they survive the shift untouched, and the freed
trailing bytes are left dead — consistent with the existing function's documented
choice not to compact freed bytes.

### Prompt shape and defaults

- The export prompt **stays a choice card** and grows an optional area for extra
  rows above its button row. It is deliberately *not* ported to the app's
  buttonless-custom-dialog idiom. The card's overlay is a member of the window's
  own content stack, whose paint order is documented as load-bearing so the prompt
  appears over an open grid; the key dispatcher routes to it through an explicit
  visibility check, separate from the generic "a Fyne dialog owns the keyboard"
  branch; and eight test files reach the prompt through the card's API, one of
  them asserting its index within the overlay stack. A dialog port breaks all of
  that for no gain.
- Vertical navigation is available without a port: while the card is visible the
  dispatcher hands it every key and returns, and the card's key handler ignores Up
  and Down today. They are free in exactly the state that needs them.
- The choice card is shared with the delete confirmation, so the extra-rows slot
  and Up/Down delegation are optional and deletion passes neither, leaving its own
  card byte-identical.
- **Format remains the committing action.** The bottom row stays PNG / JPEG, so
  Cmd/Ctrl+E followed by Return is still a two-keystroke export. Escape cancels.
- Options **reset to defaults every time the prompt opens** — not persisted to
  preferences, not remembered within a session. The prompt always states the
  whole truth about what it is about to write.
- Defaults are today's behavior: Original size, metadata included.

### Destination and reporting

- The rule that a typed extension overrides the chosen format is unchanged: a
  `.png` file must contain a PNG. No new prompt argues with the user about a name
  they already typed.
- The suggested filename gains the applied limit as a suffix when a limit
  actually applied; at Original size the suggestion is unchanged. The
  pre-existing hazard that an Original-size export pre-fills a name colliding
  with the source is out of scope here.
- The completion toast reports the **outcome**, not the request, and only when it
  differs from the defaults — so a typed extension that overrode the format shows
  up in it, while a routine export keeps today's short message.

### Module interfaces

- The imaging module's export entry point widens to carry an options value
  (size limit and metadata choice) alongside its existing destination, image, and
  source arguments. It has two call sites; the wallpaper path passes defaults.
- The viewer's internal export runner carries the same options value in place of
  its current bare extension argument.
- A new exported scale-to-fit entry point in the imaging module, distinct from
  the unexported thumbnail-grade one.

## Testing Decisions

**Verification runs under the Docker test target, never a bare native `go test`.**
The end-to-end suite includes golden-image comparisons, and Fyne's software
rasterizer produces different anti-aliased pixels per CPU architecture — the
Makefile documents this directly. On a developer machine that is not
Linux/amd64, one golden test fails as a standing platform artifact; that failure
is the pre-existing baseline, not a regression. An agent must not "fix" it by
accepting a locally rendered master, which passes locally and breaks CI.

A good test here asserts on **what was written to disk** — the bytes, their
dimensions, and the tags readable back out of them — and never on how the code
arrived there. No test should assert that a particular helper was called, that a
scaling kernel was selected, or that an options struct held a particular field.

**No new seams are introduced.** Both seams used already exist:

1. **Prompt-to-file, in the UI module.** The save-panel entry point is already a
   swappable package-level variable, and the export runner is already split from
   its dispatching wrapper specifically so a test can drive the whole path on one
   goroutine. Every user-visible rule — which rung produced which pixel
   dimensions, whether metadata survived, what the suggested filename was, what
   the toast said, that a typed extension overrode the format — is testable here
   against a real file. Prior art: the existing export tests that assert a picked
   path receives the displayed frame, that a JPEG source keeps its GPS tags, that
   the suggested path uses the source's own folder, and that cancelling writes
   nothing.
2. **Encode-level, in the imaging module.** The tag-dropping rules want direct
   assertions on written bytes: dimension tags absent after a real resize, camera
   and GPS tags still present, MakerNote and DPI still present, nothing dropped
   when the pixels did not change, and the rotate-and-save path unchanged. Prior
   art: the existing test that a JPEG source keeps metadata on a JPEG
   destination, and the metadata-removal test family, which already reads tags
   back out of written files and covers orientation, ICC retention, permissions,
   and symlink targets.

Specific coverage the seams above should carry:

- Each rung produces the expected longest edge, and a photo smaller than the rung
  is written at its own size.
- Metadata included versus omitted, asserted by reading tags back.
- ICC survives omission; the image stays upright when the source orientation is
  not 1.
- The closed tag set is dropped on a real resize and only then.
- The prompt opens at defaults after a previous export used non-defaults.
- The existing mutual guards between the export prompt and the delete
  confirmation still hold after the move from card to dialog — there is already a
  test in each direction, and both must keep passing.
- The keyboard story: the prompt's own keys respond while navigation keys are
  swallowed. There is already a test asserting exactly this for the card; it
  needs to hold for the dialog.

## Out of Scope

- **Batch export over a Grid selection.** Deferred as its own feature.
- **A JPEG quality control.** The re-encode quality stays as it is.
- **Cropping or exporting an image-region selection.** Export writes the whole
  frame; the region path remains copy-to-clipboard only.
- **Changing what the EXIF window's metadata removal does.** It stays an
  irreversible in-place operation behind its confirmation.
- **Rewriting dimension tags to correct values.** They are dropped, not patched.
- **Dropping MakerNote or DPI tags.**
- **The pre-existing collision** where an Original-size export pre-fills a
  filename matching the source. Only the resized case gains a suffix.
- **Persisting export options** to preferences, in any form.
- **Changing the rotate-and-save path** in any way.

## Further Notes

- Three terms were added to the domain glossary while specifying this: **metadata
  removal** (in place, irreversible, the EXIF window's operation), **metadata
  omission** (a copy written without the source's tags, the export operation), and
  **export size limit** (the longest-edge ceiling, never enlarging). The UI wording
  should follow them — export never says "strip" or "remove".
- The decision to **drop** lying dimension tags rather than patch them is the one
  choice here that a future reader is most likely to question, and it had real
  alternatives (patch the values; refuse to combine a size limit with included
  metadata; ship stale tags). It is a reasonable candidate for the repo's first
  ADR if one is wanted.
- The original TODO entry claimed the metadata work was something the save module
  "already gets close to". It is in fact complete and shipped, wired to the EXIF
  window. The remaining work is making an existing default optional on the export
  path, not building a stripper.
- Export currently *preserves* metadata by deliberate design, so this feature
  makes an existing behavior optional rather than adding a capability.
