# Resolve the mosaic source pool from Grid View

Type: task
Status: resolved
Priority: P0
Blocked by: none

## Goal

Snapshot exactly the host files represented by the current Grid result without
teaching Grid View about mosaics.

## Existing code anchors

- `grid.Overview.Selection()` already returns defensive, ascending host file
  indices and deliberately survives search changes.
- `grid.Overview.Targets()` falls back to only the highlighted cell. That is
  correct for copy/delete and incorrect for mosaics.
- `grid.count()` plus `grid.fileIndex()` already define the complete displayed
  result after search, hide-duplicates, or duplicate-group browsing, including
  virtualized cells outside the scroll viewport.
- `internal/ui/batch.go` and `compare.go` show that selection-to-file decisions
  belong in `internal/ui`, where both Grid state and `viewer.FileAt` exist.

## Scope

- Add a narrowly named defensive query such as
  `(*grid.Overview).ResultIndexes() []int`; do not add the ambiguous
  `Result() []int` proposed by the original plan.
- Implement it by walking display indices through the existing `count` and
  `fileIndex` mapping. Do not duplicate search or duplicate-visibility rules.
- Add a Grid result-change observer and fire it after result membership can
  change (search/backspace/clear, hide/browse changes, hash-driven refilters,
  and file-set shrink). This lets the menu matrix notice a zero-result search;
  `syncMenus` will avoid a native menu rebuild when availability did not move.
- In a new `internal/ui/mosaic.go`, add the cross-feature resolver that chooses
  `Selection()` when non-empty and otherwise `ResultIndexes()`, validates each
  host index, and copies the corresponding `fyne.URI` values immediately.
- Keep the resolver synchronous on the UI goroutine. Later Grid mutation must
  not alter the returned slice.
- Add any new `_test.go` paths to Qodana in this change.

## Acceptance Criteria

- Ten explicitly selected files in a 100-file result yield exactly those ten,
  even if a later search hides some of them.
- No selection yields every member of the current result, not only the
  highlighted cell or virtualized viewport.
- Search, hide-duplicates, and duplicate-group browse all use the same mapping
  the Grid is currently drawing; a no-match search yields an empty pool.
- Reorder, removal, search, selection, and Grid close after resolution cannot
  retarget the copied URI snapshot.
- Existing copy/delete `Targets()` behavior does not change.

```sh
go test ./internal/ui/grid -run 'Test(ResultIndexes|ResultChanged|Targets)' -count=1 &&
go test ./internal/ui -run 'TestMosaicSources|TestMosaicSnapshot' -count=1 &&
make check-qodana-test-exclusions
```

## Non-Goals

- Opening the mosaic window
- Decoding or validating source files
- Changing `Targets()`

## Comments

This ticket intentionally has no dependency on `internal/mosaic`; it exposes
existing Grid state and performs app-level composition only.

Implemented and verified on 2026-09-04: defensive filtered-result indices,
change notification, selection exclusivity, and command-entry snapshots are green.
