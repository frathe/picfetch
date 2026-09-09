# 02 - Preserve the viewport across live duplicate reflows

Status: resolved
Blocked by: 01
Source: [Grid scroll follows the viewport during duplicate merging](../spec.md), AC3 through AC5.

## Contract

Change only non-reset Grid View filter restoration. After a duplicate-group
update rebuilds `matches` and refreshes the grid, retain the previous host's
ring only when its remapped display cell is still fully visible. If duplicate
extras disappearing above or below the viewport move that host offscreen, use
Ticket 01's visible-edge reconciliation instead of asking `GridWrap.Highlight`
to reveal the stale host.

Initial and explicit filtering continue to reset to cell zero. Grouping,
duplicate notifications, inspect/browse behavior, title synchronization,
selection, and cancellation ordering remain unchanged.

Files: `internal/ui/grid/search.go` and `internal/ui/grid/dupes_test.go`.

## Red / green

1. Replace the old highlighted-host preservation guard with
   `TestRebuildFilter_PreservesViewportWhenExtrasDisappearBeforeRing`.
2. Build a large deterministic duplicate snapshot, scroll past several rows,
   then make an extra before the viewport disappear through the normal grouping
   completion path.
3. Observe the current implementation remap the old host and snap the viewport
   back to it.
4. Reconcile to a fully visible edge ring after refresh. Assert the offset is
   preserved when normal clamping does not apply, explicit Grid selection is
   unchanged, and no image is loaded.

## Acceptance

1. `go test ./internal/ui/grid -run '^TestRebuildFilter_PreservesViewportWhenExtrasDisappearBeforeRing$' -count=1`
2. `go test ./internal/ui/grid -run 'Test(HandleKey|HighlightChanged|SetHideDuplicates|RebuildFilter|ScrollFollower|Marquee)' -count=1`
3. `go test ./internal/ui/grid -count=1`
4. `make verify`

## Constraints

- Preserve every existing `resetView` reset-to-top path.
- Do not reintroduce per-file duplicate-model reads or move grouping work onto
  the UI goroutine.
- Use the existing `UIQueue` and `Settle` helpers; do not use sleeps.
- Do not promise immediate ring tracking for native scrollbar thumb/track/page
  changes or macOS's transient auto-hidden scrollbar indicator.

## Comments

Published from the approved parent specification on 2026-09-08.

Implemented with Standard SDD and red/green TDD; verification completed on
2026-09-09. Focused grid tests, negative routing/restoration guards, and the full
`make verify` Linux/amd64 race gate passed. Implementation record:
`finished_refactorings/2026-09-08-grid-scroll-follow-duplicate-merges.md`.
