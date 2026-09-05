# 02: Export size limit

**What to build:** The export prompt gains a size control offering Original,
2400, 1600 and 1000. The number is a ceiling on the **longest edge**, aspect
preserved, and it never enlarges: choosing 2400 for an 1800px photo exports it at
1800px, not blown up.

The Original option is labelled with the frame's real longest edge, computed from
the image in hand. This is doing two jobs. It tells the user at a glance whether a
smaller rung would change anything, and it makes the RAW case honest for free —
the frame on screen for a RAW file is the camera's embedded JPEG preview, so
"Original" means the preview's dimensions rather than the sensor's, and showing
the number says so without a special case.

Scaling uses an export-grade interpolator. The imaging module's existing
scale-to-fit is tuned for filling a thumbnail cache and is visibly soft on a photo
someone is about to email; this ticket adds an exported entry point using a
higher-quality kernel and leaves the thumbnail path's faster one alone. The
longest-edge fitting rule itself is already implemented in that module and should
be reused, not reimplemented.

When a limit actually applied, the filename the save panel opens pre-filled with
carries the size, so a resized copy does not collide with the original it was
exported from. At Original size the suggestion is unchanged. Likewise the
completion toast reports the size only when it differs from the default, so a
routine export keeps today's short message.

**Blocked by:** 01

**Status:** ready-for-agent

- [ ] The export prompt offers Original, 2400, 1600 and 1000 as an export size limit
- [ ] The limit applies to the longest edge with aspect ratio preserved
- [ ] A photo already inside the chosen ceiling is exported at its own size and never enlarged
- [ ] The Original option's label states the frame's actual longest edge in pixels
- [ ] A RAW file's Original label reports the embedded preview's dimensions, matching what is actually written
- [ ] Scaling for export uses a higher-quality interpolator than the thumbnail path, which keeps its existing kernel unchanged
- [ ] The longest-edge fitting rule is reused from the imaging module rather than reimplemented
- [ ] The suggested filename carries the applied size when a limit changed the pixels, and is unchanged at Original size
- [ ] The completion toast reports the size only when it differs from the default
- [ ] The option resets to Original every time the prompt opens, and is never persisted
- [ ] New labels have translation entries in both shipped languages
