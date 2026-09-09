# Spec: Grid scroll follows the viewport during duplicate merging

Status: resolved

## Problem

With a large Grid View open, pressing `D` begins asynchronous duplicate
grouping. A user can then scroll away from the active ring with a mouse wheel
or trackpad. When a group update hides another duplicate, the grid restores the
old highlighted host through:

`groupsReady -> applyVisibleFilter(false, keepHost) -> restoreHighlight -> GridWrap.Highlight`

Fyne's `GridWrap.Highlight` reveals an offscreen cell. Each group update can
therefore pull the viewport back to the stale ring, making it impossible to
browse the changing result. Page Up and Page Down do not show the problem
because they already move the ring with the viewport.

## Decisions

- Keep this inside `internal/ui/grid`; do not fork, replace, or upgrade Fyne.
- Add a transparent, scroll-only sibling above `GridWrap`. It receives Fyne's
  common wheel/trackpad scroll event, forwards the vertical delta through
  `GridWrap.ScrollToOffset`, then reconciles the ring. It implements only
  `fyne.Scrollable`, so taps, hover, marquee drag, and scrollbar controls keep
  their existing targets.
- Retain the ring while its cell is fully visible under Fyne's own visibility
  rule. Once it becomes clipped, move it to the nearest fully visible edge row:
  the first row when scrolling down and the last row when scrolling up. Preserve
  its column when possible; clamp on a short final row.
- On a non-reset filter reflow, preserve the viewport rather than an old
  highlighted host. Retain that host only if its remapped cell is still fully
  visible; otherwise select the nearest visible edge cell. This covers duplicate
  removals above the viewport as well as ordinary live hash completions.
- Grid title and menu state continue to follow the ring. Scrolling does not
  load a new image, change the underlying current image, or alter Grid
  selection.
- Explicit filter operations keep their current reset-to-top behavior: opening
  or changing search, toggling `D`, clearing filters, and changing duplicate
  distance.
- Scrollbar thumb/track/page actions retain native behavior. They cannot update
  the ring immediately because Fyne exposes no `GridWrap` scroll callback, but
  the next filter reflow reconciles the ring before it can snap the viewport
  back. The proxy may not trigger macOS's transient auto-hidden scrollbar
  indicator; this limitation is accepted.
- If no full row fits in an unusually short viewport, use the nearest
  intersecting edge row and allow Fyne's minimum corrective reveal.

## Acceptance criteria

1. A real Fyne canvas wheel/trackpad event scrolls Grid View and moves an
   offscreen ring to a fully visible edge row in the same column, without
   changing the selected files or loading an image.
   Verify: `go test ./internal/ui/grid -run '^TestScrollFollower_' -count=1`
2. The scroll proxy receives only scroll input; cell tapping, hover, marquee
   drag, and native scrollbar interaction retain their present routing.
   Verify: `go test ./internal/ui/grid -run '^(TestScrollFollower_|TestMarquee)' -count=1`
3. When duplicate grouping hides extras above a manually scrolled viewport,
   the viewport does not snap to the former highlighted host. The resulting
   ring is fully visible and explicit Grid selection is unchanged.
   Verify: `go test ./internal/ui/grid -run '^TestRebuildFilter_PreservesViewportWhenExtrasDisappearBeforeRing$' -count=1`
4. Existing initial filter/reset behavior, keyboard navigation, and title
   synchronization remain correct.
   Verify: `go test ./internal/ui/grid -run 'Test(HandleKey|HighlightChanged|SetHideDuplicates|RebuildFilter)' -count=1`
5. Formatting, vetting, build, race tests, and repository gates pass.
   Verify: `make verify`

## Non-goals

- Adding a preference, user-visible control, translation key, or exported API.
- Treating scroll as image navigation or changing Grid selection.
- Changing keyboard Page Up/Page Down, arrow-key, hover, search, duplicate
  toggle, or explicit reset semantics.
- Making scrollbar thumb/track/page movement update the ring immediately.
- Forking Fyne solely to expose its private GridWrap scroll callback.

## Honest limit

Fyne reports wheel and trackpad input through the same event and exposes no
device-specific distinction. The implementation covers both reported paths,
but cannot observe a native scrollbar's offset changes until a later grid
reflow. Its viewport-first reconciliation prevents the reported later snap.

## Comments

Implemented with Standard SDD and red/green TDD; verification completed on
2026-09-09. Focused grid tests, negative routing/restoration guards, and the full
`make verify` Linux/amd64 race gate passed. Implementation record:
`finished_refactorings/2026-09-08-grid-scroll-follow-duplicate-merges.md`.
