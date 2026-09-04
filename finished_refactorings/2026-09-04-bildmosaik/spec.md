# Generate Image Mosaic

Status: ready-for-human

## Problem Statement

PicFetch can display, export, and set one image as the desktop wallpaper, but it
cannot combine a Grid View result into a screen-sized collage. Users who want a
wallpaper from several photos must currently leave PicFetch and assemble it in
another application.

## Solution

Add **Generate Image Mosaic...** to the Actions menu while Grid View is open.
The command snapshots either the explicit Grid selection or, when there is no
selection, every image in the current filtered Grid result. It opens a mosaic
window where the target display is chosen directly and all visual options live
behind the Advanced disclosure.

PicFetch generates a native-pixel image for the target display. Photo cards
keep their source aspect ratio, have approximately equal visual area, overlap
slightly, and receive small random offsets and rotations. Rotated card and
frame edges are anti-aliased. The canvas must be fully covered. The preview can
be regenerated with the same settings, exported as PNG or JPEG, or handed to
the existing wallpaper integration.

All processing stays local and never modifies a source image.

## Resolved Product Decisions

These decisions are final for this feature and should not be relitigated during
implementation.

### Platforms and target display

- Support the Windows, macOS, and Linux desktop builds.
- Detect attached displays, their stable session identifiers, native pixel
  dimensions, and aspect ratios.
- Default to the display containing the largest part of the PicFetch window.
- Let the user select another detected display before generation.
- Render one image at the native pixel dimensions of the selected display.
- If the selected display disappears before generation, require a new choice;
  do not silently render for another display.

### Source pool

- The command is available only while Grid View is visible and its current
  result contains at least one image.
- If one or more Grid cells are explicitly selected, use only those host files.
- While duplicate hiding is active, resolve every explicitly selected duplicate
  to its group's highest-resolution representative and include that
  representative only once. This representative substitution is the sole
  exception to explicit-selection exclusivity.
- If nothing is selected, use every host file in the current Grid result after
  search and duplicate filtering, including cells outside the scroll viewport.
- Snapshot URIs at command entry so later navigation, search, sorting, or file
  removal cannot retarget an in-flight generation.
- Collapse repeated occurrences of the same source URI, then shuffle the pool
  for every generation. Use every distinct readable URI at most once before
  reuse. Once the canvas is covered, discard the unused tail.
- If the distinct readable pool is exhausted before coverage is complete,
  repeat images from the same pool. Never add an unselected image to an
  explicit selection except for the hidden-duplicate representative
  substitution above.
- Skip an unreadable or disappeared source. Fail only when no readable source
  remains.

### Layout and rendering

- Use an adaptive, jittered layout rather than unconstrained random placement.
  The implementation may maintain an occupancy mask or equivalent internal
  coverage model; this stays hidden behind the mosaic module's interface.
- Preserve every source image's aspect ratio. Do not stretch or crop inside a
  card.
- Cards may extend beyond and be clipped by the outer canvas.
- No background pixel may remain visible in the final image.
- Keep overlap light and avoid routinely hiding a large part of a motif.
- Treat overlap as the intended linear inset between neighboring card bodies,
  measured against the shorter unrotated image edge. Rotation may vary the
  exact intersecting area, but an interior primary card must not be buried into
  a narrow sliver. Coverage-repair cards render underneath the primary layout.
- Respect EXIF orientation and size raster cards from the oriented decoded
  pixels, not pre-orientation probe dimensions. Animated images contribute
  their first composited frame. SVG uses a raster appropriate for the required
  card size. Camera RAW contributes the embedded preview PicFetch already
  supports.
- Produce sRGB output without copying source metadata.
- A frame choice applies to every card in one mosaic.

Default settings:

| Setting | Default | Allowed range |
| --- | ---: | ---: |
| Minimum shorter card edge | 18% of the target display's shorter edge | 10-30% |
| Size variation | plus or minus 12% | 0-25% |
| Overlap | approximately 8% | 0-20% |
| Maximum rotation | plus or minus 7 degrees | 0-12 degrees |
| Drop shadow | on | off/on |

The minimum size is measured on the unrotated card before clipping at the outer
canvas. Frame thickness and an enabled shadow participate in layout bounds.
The translucent shadow never counts as opaque image coverage.

Frame choices for the first version:

- None
- Thin light
- Thin dark
- Polaroid

### Window and workflow

- Open a dedicated secondary mosaic window instead of adding another overlay
  to the load-bearing main-window stack.
- The initial view contains the target-display choice with **Refresh Displays**,
  the full-width **Advanced** toggle, **Generate**, and **Cancel**.
- The initially collapsed **Advanced** section contains minimum image size,
  frame, size variation, overlap, maximum rotation, and drop shadow.
- Generation runs outside the UI goroutine. Show an activity indicator while it
  is running and marshal every UI mutation through Fyne's UI queue.
- A newer generation supersedes the previous one. A stale result may finish
  internally but must never replace the current preview.
- The preview has a top-left **Start Over** action plus **Regenerate**, **Set as
  Wallpaper**, **Save Image**, and **Close**.
- **Start Over** returns to configuration, discards the finished result and
  status, and preserves the source snapshot, selected display, visual settings,
  and export format. This makes choosing another display and generating its
  wallpaper a continuous workflow.
- Regeneration keeps every setting but uses a new random seed and reshuffles the
  pool.
- The preview has no per-card movement, resizing, rotation, removal, or other
  manual editing in this version.
- Remember the last valid visual settings. Do not persist source URIs or a Grid
  selection.

### Export and wallpaper

- PNG is the default export format; JPEG is the alternative.
- Reuse the existing native save picker and `imaging.Export` path.
- Suggest `PicFetch-Mosaic-YYYYMMDD-HHmmss` with the selected extension.
- Save and wallpaper actions consume the exact current render result and never
  trigger an implicit regeneration.
- Reuse the existing persistent wallpaper-copy lifecycle. Extend the wallpaper
  seam with an optional target-display identifier instead of creating a second
  wallpaper implementation.
- Windows and macOS should address the selected display where their native
  integration can do so reliably.
- GNOME and KDE wallpaper tools may expose only desktop-wide application. When
  a selected-display operation is unavailable, report the limitation before
  applying globally; do not pretend that only one display changed.
- An unsupported or failed wallpaper operation leaves the preview open and the
  export action usable.

## Architecture Decisions

- Add `internal/mosaic` as the deep, viewer-independent module. Its small
  external interface accepts a validated request and returns one immutable
  rendered result. It hides shuffling, probing, decode-at-needed-size, adaptive
  placement, coverage verification, frame rendering, and failure aggregation.
- Add `internal/displays` as the seam for native display enumeration. It returns
  opaque display identifiers plus user-facing names and native pixel bounds.
  Build-tagged adapters own platform details; tests replace the dispatch through
  the established `internal/uitest` style.
- Add `internal/ui/mosaicwin` as the secondary-window feature module. It owns
  configuration widgets, preview state, its request lifecycle, the activity
  indicator, and the five preview actions. It reaches back through a narrow
  consumer-side Host interface.
- Keep the cross-feature source decision in `internal/ui`: it joins Grid
  selection/current result to the mosaic window and snapshots URIs. The Grid
  package exposes current result indices but does not learn about mosaics.
- Reuse `internal/imaging`, `internal/filepicker`, `internal/preferences`, and
  `internal/wallpaper`; do not create parallel decode, export, settings, or
  wallpaper paths.
- Change the wallpaper interface once to accept a target descriptor. Existing
  single-image wallpaper and new mosaic wallpaper must use the same seam.
- Every new background operation needs cancellation or staleness handling, an
  observable completion signal, a `Settle` path for feature tests, and a drain
  hook in `internal/ui/harness_test.go`.
- Add each new `_test.go` path to Qodana's exact duplicated-code exclusion list.
- Update `ARCHITECTURE.md` in the same change that introduces new packages.

## Acceptance Criteria

### AC1 - Menu availability and source snapshot

The Actions menu enables **Generate Image Mosaic...** only for a non-empty Grid
result. Explicit selection is exclusive; otherwise the complete current result
is used, including filtered cells outside the viewport. A later Grid mutation
does not alter an open mosaic request.

```sh
go test ./internal/ui/... -run 'TestMosaic(Menu|Sources|Snapshot)' -count=1
```

### AC2 - Deterministic, bounded layout

For a fixed seed, source metadata, target size, and settings, layout is stable.
Every rotation and minimum size stays within its configured bounds. Extra pool
entries are unused after coverage; a short pool repeats only its own entries.

```sh
go test ./internal/mosaic/... -run 'TestGenerate_(Deterministic|Bounds|Pool)' -count=1
```

### AC3 - Full coverage and source fidelity

The final canvas has the requested native-pixel bounds and no uncovered pixel.
Portrait, landscape, transparent, EXIF-rotated, animated, SVG, and RAW fixtures
retain the defined orientation and aspect behavior without source mutation.

```sh
go test ./internal/mosaic/... -run 'TestGenerate_(Coverage|SourceFidelity|SourcesUnchanged)' -count=1
```

### AC4 - Frame styles

All four frame choices render consistently, participate in layout bounds, and
are covered by deterministic image comparisons.

```sh
go test ./internal/mosaic/... -run 'TestGenerate_FrameStyles' -count=1
```

### AC5 - Display enumeration and selection

Display adapters return native pixel dimensions and stable session IDs. The
window-containing display is the default, explicit selection survives refresh,
and removal requires a new choice.

```sh
go test ./internal/displays/... ./internal/ui/mosaicwin/... -run 'Test(Display|Target)' -count=1
```

### AC6 - Lifecycle and stale-result rejection

Generation does not block the UI. Regeneration preserves settings, supersedes
older work, and only the newest result can reach the preview. Closing or
cancelling leaves no background work that can mutate later UI state.

```sh
go test ./internal/ui/mosaicwin/... ./internal/ui/... -run 'TestMosaic(Generate|Regenerate|Cancel|Drain)' -count=1
```

### AC7 - Preview fidelity and export

PNG and JPEG use the exact current preview result at the target resolution,
with the specified filename pattern. Cancelling the picker writes nothing; a
write failure leaves the preview usable.

```sh
go test ./internal/ui/mosaicwin/... ./internal/ui/... -run 'TestMosaic(Preview|Export)' -count=1
```

### AC8 - Wallpaper routing and honest fallback

The current preview is copied to persistent wallpaper storage and routed with
the selected display ID. Supported adapters target that display. An adapter
that cannot target one display returns a typed limitation; failures and
unsupported environments retain the preview and export path.

```sh
go test ./internal/wallpaper/... ./internal/ui/mosaicwin/... ./internal/ui/... -run 'Test(MosaicWallpaper|SetTarget|TargetUnsupported)' -count=1
```

### AC9 - Persistence, localization, and accessibility

Visual settings round-trip through preferences, invalid values normalize to
defaults, and source URIs never persist. All new controls are keyboard
reachable, have accessible names, and use localized English keys present in
every catalogue.

```sh
go test ./internal/preferences/... ./internal/ui/mosaicwin/... ./internal/ui/... -run 'Test(MosaicPreferences|MosaicAccessibility|Translations)' -count=1
```

### AC10 - Large-pool behavior and final gate

A 10,000-entry pool does not decode every source, rapid regeneration exposes
only the final result, all source checksums remain unchanged, and repository
verification stays green.

```sh
go test ./internal/mosaic/... ./internal/ui/mosaicwin/... -run 'Test(Generate_(LazyPool|SourcesUnchanged)|MosaicSupersede_ReverseCompletionPublishesOnlyNewest)' -count=1 &&
make fmt-check &&
go vet ./... &&
go build ./... &&
go test -timeout 30m -race ./... -count=1 &&
GOOS=windows GOARCH=amd64 go vet ./internal/...
```

### AC11 - Collage polish and compact configuration

The reported 7% overlap scenario retains at least 45% of every substantially
in-canvas primary card while coverage repairs stay underneath it. Rotated card
and frame edges contain anti-aliased transition pixels. Drop shadow defaults on,
can be disabled, persists both values, and does not count as coverage. Before
Advanced is expanded, target display is the only visible visual configuration;
after expansion every visual control is visible and keyboard reachable.

```sh
go test ./internal/mosaic/... -run 'Test(Layout_(ConfiguredOverlap|ShadowDoesNotCount)|Generate_(RepairCards|DropShadow)|RenderPlacement_(Antialiases|Interpolates))' -count=1 &&
go test ./internal/preferences/... -run 'TestMosaicPreferences' -count=1 &&
go test ./internal/ui/mosaicwin/... ./internal/ui/... -run 'TestMosaic(Controls|Configuration|Keyboard|Preferences)' -count=1 &&
go test . ./internal/ui/help -run 'Test(Translations|Manual)' -count=1
```

### AC12 - Area-covered rotated edges

Rotated card silhouettes use actual destination-pixel area coverage instead of
only interpolating a point sample from a larger rectangular bitmap. Against an
independent polygon-clipping oracle, no materially partial pixel on the
reported 12-degree edge becomes binary, mean geometric-coverage error stays
within 16 of 255 alpha levels, and no pixel differs by more than 48. A tiny
center-heavy Gaussian filter may soften only this one-channel mask.
The same mask clips the frame and photo, so filtering does not blur photo
interiors. Generation remains deterministic, cancellable, and produces the
same exact target-sized CPU image for preview, export, and wallpaper use.

```sh
go test ./internal/mosaic -run 'TestRenderPlacement_(AreaCoverage|StopsBetweenTransformBands)' -count=1 &&
go test ./internal/mosaic -run 'TestGenerate_(DeterministicAndCoverage|FrameStyles|SourceFidelity)' -count=1
```

### AC13 - Return from preview to configuration

After a mosaic finishes, a localized, accessible **Start Over** button appears
at the preview's top-left. Activating it returns to the configuration screen,
clears the finished result and status without generating again, preserves the
source snapshot, target, visual settings, and export format, and focuses the
target-display control. The user can then choose another attached display and
generate a result at that display's native resolution. Preview actions,
including **Start Over**, remain unavailable while another preview action is
busy.

```sh
go test ./internal/ui/mosaicwin -run 'TestMosaic(StartOver|Keyboard|Accessibility)' -count=1 &&
go test . ./internal/ui/help -run 'Test(Translations|Manual)' -count=1
```

### AC14 - Oriented and unique source selection

A raster card uses the display-ready decoded orientation and matching oriented
pixel dimensions throughout layout and rendering. During one generation,
repeated occurrences of one URI cannot cause that image to be selected again
while another distinct readable URI remains unused. When duplicate hiding is
active, an explicit selection that still references a hidden duplicate resolves
to the group's highest-resolution representative, with each representative
appearing only once in the source snapshot.

```sh
go test ./internal/mosaic -run 'TestGenerate_(RespectsDecodedOrientation|UsesDistinctSourceURIsBeforeReuse)' -count=1 &&
go test ./internal/ui -run 'TestMosaicSources_HiddenDuplicatesUseHighestResolution' -count=1
```

## Non-Goals

- Mobile platforms
- Manual card editing in the preview
- Mixed frame styles inside one result
- User-authored frame styles
- Cloud generation or upload
- A cross-platform promise that every Linux desktop can change one monitor's
  wallpaper independently
- Changing the behavior of the existing single-image export command
- A GPU-only mosaic path whose output depends on a window, graphics driver, or
  readback support

## Honest Limits

- Linux display enumeration and per-monitor wallpaper control vary by display
  server and desktop environment. The feature can generate the correct image
  for a detected display and can support tested GNOME/KDE paths, but it cannot
  truthfully guarantee per-monitor application on every Linux desktop. The
  typed fallback and retained export path are the accepted behavior.
- Native macOS and Windows wallpaper behavior cannot be fully proven by the
  Linux CI runner. Build-selected logic and adapter contracts are automated;
  release acceptance still requires the documented native smoke-test matrix.
- Decoding a very large raster may still temporarily require memory near the
  source decoder's needs because existing decoders do not all support streamed
  region or resolution-selective decoding. The module must bound concurrency
  and release each source promptly rather than claim zero large allocations.
- A native-resolution raster still consists of pixels and can show individual
  steps when magnified substantially. Area coverage removes the avoidable
  binary staircase without applying a soft-focus filter to the photographs.
