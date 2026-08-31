# Compare two grid-selected images

Status: approved

## Problem

Grid View can explicitly select multiple images, but PicFetch cannot hold two
of those images on screen for a detailed visual comparison. The comparison
must be a temporary presentation of the existing grid selection: entering,
using, and leaving it must not mutate the file set, grid order, selection,
filter, highlight, or scroll position.

## Decisions

- `Cmd+D` on macOS and `Ctrl+D` elsewhere runs **Actions -> Compare selected
  images**. The action is available only while Grid View has exactly two
  explicitly selected images. An invalid invocation stays in Grid View and
  reports **Select exactly 2 images to compare**. It never substitutes the
  highlighted image or takes an arbitrary pair from a larger selection.
- Selection identity is based on the grid's host-file indices. A selected file
  remains eligible while the filename or duplicate filter hides its thumbnail.
  The earlier file in grid/file order starts on the left; selection gesture
  order is neither used nor persisted.
- Comparison is an opaque main-window overlay above the still-open grid, never
  a separate window. **Back to Grid** and `Escape` remove only that overlay and
  reveal the grid exactly as it was.
- Every comparison starts fitted and centered in fixed 50/50 side-by-side
  panes. It starts in side-by-side mode even if the preceding comparison used
  swipe, and its inactive swipe divider starts at 50%.
- A compact translucent toolbar remains visible at the top right. Its controls
  are **Swipe** (or **Side by side** while swipe is active), **Swap**, and
  **Back to Grid**. Native title-bar controls are out of scope because Fyne
  does not support them portably.
- Swipe uses the full comparison viewport. The left image is revealed left of
  a draggable vertical divider and the right image to its right. The divider
  itself is the drag target; dragging elsewhere pans both images. `Left` and
  `Right` move the divider by 5 percentage points, their Shift variants move it
  by 1 point, and `Home` and `End` move it to 0% and 100%. Those keys do nothing
  in side-by-side mode.
- Both images share a normalized image-space center and a zoom multiplier
  relative to their respective fitted sizes. Wheel and `+` / `-` zoom both;
  dragging either image and Shift+wheel pan both; `0` fits both; `1` displays
  both at actual pixel size. The center is clamped to the intersection of both
  images' valid pan ranges so neither exposes blank overscroll or drifts out of
  sync.
- Layout changes and window resizes preserve the normalized center and zoom
  multiplier while recomputing fitted scales for the new viewport. Actual-size
  mode preserves its absolute 100% scale. **Swap** exchanges the images,
  badges, title order, and swipe roles while preserving layout, transform, and
  divider position. It never changes grid selection or file order.
- Both layouts show translucent identity badges at their bottom-left and
  bottom-right corners. A badge normally contains the base filename. When the
  base names match, both use the shortest distinguishing `folder/file` suffix.
  The window title is `Compare: left.jpg | right.jpg - PicFetch`, follows a
  swap, and returns to the grid's highlighted-file title on exit.
- The overlay opens before decoding starts and shows one spinner per pane.
  Both sources decode concurrently. **Back to Grid** stays enabled while
  **Swipe** and **Swap** stay disabled until both sources are ready. `Escape`
  cancels pending work. If either load fails, comparison closes, the untouched
  grid returns, a non-blocking error is shown, and neither source is removed
  from the set.
- While comparison is active, only its controls, linked zoom/pan, `Escape`, F1
  help, and normal window closing are admitted. Viewer, grid, navigation,
  rotation, sorting, copy, delete, export, favorite, wallpaper, merge,
  information, and picture-frame commands are disabled or ignored. Drops,
  file-dialog opens, and native Open With deliveries are refused with
  **Return to Grid View before opening files**; they are not queued and do not
  replace the comparison.
- Raster images retain their full decoded resolution. SVGs rerasterize as zoom
  changes. RAW files use the same embedded preview as the normal viewer.
  Animated images remain frozen on their first decoded frame. Comparison uses
  each source's canonical EXIF-corrected orientation and ignores temporary
  viewer-only rotation.

## Acceptance criteria and verification

1. Entry accepts exactly two explicit grid selections, including selections
   hidden by an active filter, and assigns sides by file order.
   Verify: `go test ./internal/ui/... -run 'Compare(Entry|Selection|Order)' -count=1`
2. Comparison is a main-window overlay whose exit restores the unchanged grid
   selection, filter, highlight, scroll position, and title.
   Verify: `go test ./internal/ui/... -run 'Compare(Overlay|Restoration|Exit)' -count=1`
3. A new comparison starts fitted and centered in fixed 50/50 side-by-side
   panes, with reset layout and divider state.
   Verify: `go test ./internal/ui/... -run 'Compare(Initial|SideBySide|Reset)' -count=1`
4. The permanent toolbar, identity badges, title, and Swap behavior match the
   decisions above.
   Verify: `go test ./internal/ui/... -run 'Compare(Toolbar|Identity|Title|Swap)' -count=1`
5. Swipe rendering, divider dragging, and divider keyboard control match the
   decisions above without stealing ordinary pan drags.
   Verify: `go test ./internal/ui/... -run 'Compare(Swipe|Divider|Layout)' -count=1`
6. Zoom, pan, fit, actual-size, and center clamping remain linked for images of
   different dimensions and aspect ratios.
   Verify: `go test ./internal/ui/... -run 'Compare(Linked|Zoom|Pan|Clamp|Actual)' -count=1`
7. Layout changes, resizes, and swaps preserve the specified state, including
   absolute 100% scale in actual-size mode.
   Verify: `go test ./internal/ui/... -run 'Compare(Transition|Resize|Preserve)' -count=1`
8. Concurrent loading, cancellation, and either-side failure are observable,
   stale-safe, and leave the grid and file set untouched.
   Verify: `go test ./internal/ui/... -run 'Compare(Loading|Cancel|Failure|Stale)' -count=1`
9. Every unrelated command path is disabled or ignored, and every file-open
   path refuses input rather than queueing it while comparison is active.
   Verify: `go test ./internal/ui/... -run 'Compare(Command|Isolation|OpenRefusal)' -count=1`
10. Raster, SVG, RAW, animated, and oriented inputs retain the specified
    fidelity and existing input-size protection.
    Verify: `go test ./internal/ui/... ./internal/imaging/... -run 'Compare|Vector|RAW|Orientation|InputTooLarge' -count=1`
11. Every introduced user-visible string is localized, both manuals describe
    the finished comparison workflow, and no Unicode arrow reaches app-drawn
    content.
    Verify: `go test ./... -run 'Translations|Manual' -count=1`
12. The completed feature passes the repository's full formatting, vet, build,
    and race-test gate.
    Verify: `make verify`

## Non-goals

- A separate comparison window or native title-bar controls.
- Inferring a pair from the highlighted cell or a selection larger than two.
- Persisting selection gesture order or changing grid/file order after Swap.
- Applying the normal viewer's temporary rotation to either comparison source.
- Reducing source quality or silently downsampling to satisfy memory pressure.

## Honest limit

Two full-resolution decodes can require roughly their combined decoded memory.
PicFetch keeps its existing encoded-input safeguards and reports a load failure
when the comparison cannot be completed; it does not silently reduce quality.

## Ticket map

1. Open and close a fitted side-by-side comparison.
2. Identify and swap compared images.
3. Isolate comparison-mode commands.
4. Link side-by-side zoom and pan.
5. Add swipe comparison.
6. Preserve comparison state across transitions.
7. Preserve source fidelity across supported formats.
