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
window where the target display, minimum card size, one frame style, and
optional advanced variation controls can be chosen.

PicFetch generates a native-pixel image for the target display. Photo cards
keep their source aspect ratio, have approximately equal visual area, overlap
slightly, and receive small random offsets and rotations. The canvas must be
fully covered. The preview can be regenerated with the same settings, exported
as PNG or JPEG, or handed to the existing wallpaper integration.

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
- If nothing is selected, use every host file in the current Grid result after
  search and duplicate filtering, including cells outside the scroll viewport.
- Snapshot URIs at command entry so later navigation, search, sorting, or file
  removal cannot retarget an in-flight generation.
- Shuffle the pool for every generation. Once the canvas is covered, discard
  the unused tail.
- If the pool is exhausted before coverage is complete, repeat images from the
  same pool. Never add an unselected image to an explicit selection.
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
- Respect EXIF orientation. Animated images contribute their first composited
  frame. SVG uses a raster appropriate for the required card size. Camera RAW
  contributes the embedded preview PicFetch already supports.
- Produce sRGB output without copying source metadata.
- A frame choice applies to every card in one mosaic.

Default settings:

| Setting | Default | Allowed range |
| --- | ---: | ---: |
| Minimum shorter card edge | 18% of the target display's shorter edge | 10-30% |
| Size variation | plus or minus 12% | 0-25% |
| Overlap | approximately 8% | 0-20% |
| Maximum rotation | plus or minus 7 degrees | 0-12 degrees |

The minimum size is measured on the unrotated card before clipping at the outer
canvas. Frame thickness and shadow participate in layout bounds.

Frame choices for the first version:

- None
- Thin light
- Thin dark
- Polaroid

### Window and workflow

- Open a dedicated secondary mosaic window instead of adding another overlay
  to the load-bearing main-window stack.
- The initial view contains target display, image-size control, frame choice,
  **Generate**, and **Cancel**.
- An initially collapsed **Advanced** section contains size variation, overlap,
  and maximum rotation.
- Generation runs outside the UI goroutine. Show an activity indicator while it
  is running and marshal every UI mutation through Fyne's UI queue.
- A newer generation supersedes the previous one. A stale result may finish
  internally but must never replace the current preview.
- Preview actions are **Regenerate**, **Set as Wallpaper**, **Save Image**, and
  **Close**.
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
  indicator, and the four preview actions. It reaches back through a narrow
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
go test ./internal/ui/... -run 'TestMosaic(Menu|Sources|Snapshot)'
```

### AC2 - Deterministic, bounded layout

For a fixed seed, source metadata, target size, and settings, layout is stable.
Every rotation and minimum size stays within its configured bounds. Extra pool
entries are unused after coverage; a short pool repeats only its own entries.

```sh
go test ./internal/mosaic/... -run 'TestGenerate_(Deterministic|Bounds|Pool)'
```

### AC3 - Full coverage and source fidelity

The final canvas has the requested native-pixel bounds and no uncovered pixel.
Portrait, landscape, transparent, EXIF-rotated, animated, SVG, and RAW fixtures
retain the defined orientation and aspect behavior without source mutation.

```sh
go test ./internal/mosaic/... -run 'TestGenerate_(Coverage|SourceFidelity|SourcesUnchanged)'
```

### AC4 - Frame styles

All four frame choices render consistently, participate in layout bounds, and
are covered by deterministic image comparisons.

```sh
go test ./internal/mosaic/... -run 'TestGenerate_FrameStyles'
```

### AC5 - Display enumeration and selection

Display adapters return native pixel dimensions and stable session IDs. The
window-containing display is the default, explicit selection survives refresh,
and removal requires a new choice.

```sh
go test ./internal/displays/... ./internal/ui/mosaicwin/... -run 'Test(Display|Target)'
```

### AC6 - Lifecycle and stale-result rejection

Generation does not block the UI. Regeneration preserves settings, supersedes
older work, and only the newest result can reach the preview. Closing or
cancelling leaves no background work that can mutate later UI state.

```sh
go test ./internal/ui/mosaicwin/... ./internal/ui/... -run 'TestMosaic(Generate|Regenerate|Cancel|Drain)'
```

### AC7 - Preview fidelity and export

PNG and JPEG use the exact current preview result at the target resolution,
with the specified filename pattern. Cancelling the picker writes nothing; a
write failure leaves the preview usable.

```sh
go test ./internal/ui/mosaicwin/... ./internal/ui/... -run 'TestMosaic(Preview|Export)'
```

### AC8 - Wallpaper routing and honest fallback

The current preview is copied to persistent wallpaper storage and routed with
the selected display ID. Supported adapters target that display. An adapter
that cannot target one display returns a typed limitation; failures and
unsupported environments retain the preview and export path.

```sh
go test ./internal/wallpaper/... ./internal/ui/mosaicwin/... ./internal/ui/... -run 'Test(MosaicWallpaper|SetTarget|TargetUnsupported)'
```

### AC9 - Persistence, localization, and accessibility

Visual settings round-trip through preferences, invalid values normalize to
defaults, and source URIs never persist. All new controls are keyboard
reachable, have accessible names, and use localized English keys present in
every catalogue.

```sh
go test ./internal/preferences/... ./internal/ui/mosaicwin/... ./internal/ui/... -run 'Test(MosaicPreferences|MosaicAccessibility|Translations)'
```

### AC10 - Large-pool behavior and final gate

A 10,000-entry pool does not decode every source, rapid regeneration exposes
only the final result, all source checksums remain unchanged, and repository
verification stays green.

```sh
go test ./internal/mosaic/... ./internal/ui/mosaicwin/... -run 'Test(MosaicLargePool|MosaicRapidRegeneration)' &&
make fmt-check &&
go vet ./... &&
go build ./... &&
go test -timeout 30m -race ./... &&
GOOS=windows GOARCH=amd64 go vet ./internal/...
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
