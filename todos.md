# PicFetch — TODOs

## Done

### What's Changed

#### New Features

#### Bugfix

#### Internal

## ACTIVE DEVELOPMENT

## TODO

### Compare two grid-selected images

- **Entry:** `Cmd+D` on macOS and `Ctrl+D` elsewhere runs **Actions -> Compare
  selected images**. Enable it only in Grid View with exactly two explicitly
  selected images. Otherwise remain in Grid View and show **Select exactly 2
  images to compare**; never substitute the highlighted image or silently pick
  two from a larger selection. A selected image remains eligible when the
  current filename or duplicate filter hides its thumbnail.
- **Surface and restoration:** Show comparison as an opaque main-window overlay
  above the still-open grid, not as a separate window. `Escape` and **Back to
  Grid** remove only that overlay and reveal the grid with both selections,
  filter, highlight, and scroll position unchanged.
- **Initial layout:** Every comparison starts fitted and centered in fixed
  50/50 side-by-side panes. The image earlier in grid/file order starts on the
  left; selection order is not changed or persisted. Each new comparison resets
  to side-by-side mode, while its inactive swipe divider starts at 50%.
- **Controls:** Keep a compact, translucent toolbar permanently visible at the
  top right of the comparison content. Its labeled buttons are **Swipe** (or
  **Side by side** when swipe is active), **Swap**, and **Back to Grid**. Native
  title-bar controls are explicitly out of scope because Fyne does not support
  them portably.
- **Swipe:** Display both images in the full comparison viewport with the left
  image revealed to the left of a draggable vertical divider and the right
  image to its right. The divider itself is the drag target; dragging elsewhere
  pans both images. In swipe mode, `Left` / `Right` move the divider by 5
  percentage points, `Shift+Left` / `Shift+Right` move it by 1 point, and
  `Home` / `End` move it to 0% / 100%. Those keys do nothing in side-by-side
  mode.
- **Linked view:** Both images share a normalized image-space center and the
  same zoom multiplier relative to their respective fitted sizes. Wheel and
  `+` / `-` zoom both images; dragging either image and `Shift`+wheel pan both;
  `0` fits both; `1` displays both at actual pixel size. Clamp the shared center
  to the intersection of both images' valid pan ranges so neither image exposes
  blank overscroll or drifts out of sync.
- **Mode, resize, and swap state:** Switching layouts and resizing the window
  preserve the shared center and zoom multiplier while recomputing fitted scale
  for the new viewport. Actual-size mode preserves its absolute 100% scale.
  **Swap** exchanges the images, badges, title order, and swipe roles while
  preserving the layout, transform, and divider position; it never changes the
  grid selection or file order.
- **Identity:** Show a translucent base-filename badge at the bottom-left and
  bottom-right corners in both layouts. When the base names match, use the
  shortest distinguishing `folder/file` suffix. Set the window title to
  `Compare: left.jpg | right.jpg - PicFetch`, update it after a swap, and
  restore the grid's highlighted-file title on exit.
- **Loading:** Open the overlay immediately and decode both sources concurrently
  with a spinner in each pane. Keep **Back to Grid** enabled but disable **Swipe**
  and **Swap** until both images are ready. `Escape` cancels pending work. If
  either source fails, return to the untouched grid, preserve the selections,
  show a non-blocking error, and do not remove either file from the set.
- **Command isolation:** While comparison is active, allow only its controls,
  linked zoom/pan, `Escape`, F1 help, and normal window closing. Disable or
  ignore unrelated viewer, grid, navigation, rotation, sorting, copy, delete,
  export, favorite, and picture-frame commands. Refuse drops, file-dialog
  opens, and native Open With deliveries with **Return to Grid View before
  opening files** rather than queueing or replacing the comparison.
- **Fidelity:** Keep raster images at full decoded resolution, rerasterize each
  SVG as zoom changes, and use embedded RAW previews as the normal viewer does.
  Freeze animated images on their first decoded frame. Ignore temporary
  viewer-only rotation and use each image's canonical EXIF-corrected
  orientation.
- **Honest limit:** Comparing two full-resolution decodes can require roughly
  their combined decoded memory. Keep the existing input-size safeguards and
  report a load failure instead of silently reducing comparison quality.

## not deemed worth implementing (edge cases)

- Windows releases are not Authenticode-signed. Controlled Folder Access and
  SmartScreen both judge by signature and reputation as well as by which
  program is writing, so an unsigned `picfetch.exe` can still be blocked
  even with the in-process swap (see Done → Bugfix above, where the block
  would now name `picfetch.exe` instead of `cmd.exe`). The real remaining
  fix is signing the Windows release build — Azure Trusted Signing or a
  purchased certificate — in `.github/workflows/release.yml`, which runs no
  `signtool` today.

- There is a bug in the Windows Version: WHen in Gridview, multiselect via
  the space key works, but when trying it with mouse and Ctrl key, it does not.
  Holding the Ctrl key down and clicking on an image does not select it but
  instead opens it. Observation, when pushing the Ctrl key at exactly the same time
  as clicking on the image, it actually works, and the image is selected.
  (this seems to be a bug in fyne, created an issue, sorry Windows users)

### Qodana drops detected duplicates during serialisation (upstream)

At `210fee5` (run `33270269940`), the IDE reports 71 `DuplicatedCode`
fragments and the CI SARIF reports 63, with CI's 63 a strict subset of the
IDE's 71. The 8 fragments CI is missing are 7 in
`internal/imaging/loader_test.go` and 1 at
`internal/update/tufroot_test.go:173`. That run's own `log/idea.log` carries
exactly 3 `#o.j.q.s.i.r.g.DuplicatesProblem` "Can't find duplicate problem in
db" warnings, naming exactly those two files and no others, emitted
immediately after the line `The Project analysis stage completed in 41s` —
so Qodana's own log shows detection succeeded and serialisation into the
report/SARIF failed afterwards. This is an upstream defect, not a picfetch
config problem: nothing here suppresses or excludes those two files, and the
drop happens before any project-side filtering runs.

`qodana.yaml`'s new `_test.go` exclusion (see Done → Internal above) makes
this defect invisible going forward in this repository, because every
dropped fragment happens to live in a test file that the exclusion now
removes from the inspection entirely — recorded here so the defect is not
lost along with the rule that used to surface it. Of the 12-fragment
CSV-to-SARIF gap at `210fee5`, these 8 serialisation losses are one part; the
other 4 are the source-suppressed production fragments in the orientation
pixel loops recorded above, so nothing about that gap is left open — only
the underlying serialisation defect itself is. See
`finished_refactorings/2026-08-29-qodana-evidence.md` for the decoded byte
offsets and anchoring detail, and `plans/2026-08-29-qodana-serialisation-bug-report.md`,
Task 8's draft of the upstream report text — as of this writing not yet
submitted to JetBrains; check that file for whether it has been sent since.
