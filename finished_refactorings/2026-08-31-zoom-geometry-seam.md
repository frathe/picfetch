# Zoom geometry seam

Route: Standard

Deliverable: expose `internal/ui/zoom` presentation geometry, shared scroll
handling, and one guarded geometry-change callback without changing zoom,
window-resize, cursor, or vector-raster behavior.

Non-goals: Copy Selection composition, viewer changes, translations, manuals,
and architecture-map changes.

## Acceptance criteria

1. `Geometry` reports literal fit, manual zoom, cursor-anchored zoom, drag pan,
   Shift-scroll pan, reset-to-fit, and viewport-resize presentation geometry.
2. `HandleScroll` and the image widget produce identical zoom and pan results.
3. The geometry callback reports each real position/size change once, including
   pan, and stays silent for identical geometry.
4. Existing `onChanged`, `onScaleChanged`, and cursor contracts remain green.

Verify all criteria: `go test -race ./internal/ui/zoom/...`

## Task 1 - Geometry and callback tracer

Owner: T0 inline
Files: `internal/ui/zoom/zoom.go`, `internal/ui/zoom/zoom_test.go`
Depends: none
Contract: `Geometry`, `(*Zoom).Geometry`, and one per-instance callback setter.
Test: literal fit geometry, then real and duplicate callback notifications.
Verify: focused geometry tests, then package command.
Budget: 0 spawns; 1 review round; full suite no.

## Task 2 - Shared scroll entry point

Owner: T0 inline
Files: `internal/ui/zoom/zoom.go`, `internal/ui/zoom/widget.go`,
`internal/ui/zoom/zoom_test.go`
Depends: Task 1
Contract: `(*Zoom).HandleScroll(*fyne.ScrollEvent)` owns existing scroll logic;
the widget only delegates.
Test: compare public forwarding and widget paths for wheel zoom and Shift-pan.
Verify: package command.
Budget: 0 spawns; 1 review round; full suite no.

## Task 3 - Review and ticket resolution

Owner: T0 inline
Files: ticket status/answer and this ledger
Depends: Tasks 1 and 2
Test: deliberately disable identical-geometry suppression, observe callback
guard fail, restore, then rerun verification.
Verify: ticket command followed by repository final gate.
Budget: 0 spawns; 1 review round; full suite yes, once.

Task graph: Task 1 -> Task 2 -> Task 3. All work stays inline because three
files share one state contract; scripting does not apply.

## Cost ledger

| Task | Spawns (budget/actual) | Review rounds | Full suite | Notes |
|------|------------------------|---------------|------------|-------|
| 1 | 0 / 0 | 1 | no | Geometry + callback already on branch; Lead verified |
| 2 | 0 / 0 | 1 | no | HandleScroll already delegates from the image widget |
| 3 | 0 / 0 | 1 | yes | Negative guard seen failing. `make verify` fails on darwin/arm64 `TestE2E_CopySelection` (0.356% raster vs Docker master); linux/amd64 Docker run passes. |
