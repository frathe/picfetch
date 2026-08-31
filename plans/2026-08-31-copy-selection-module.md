# Copy Selection module implementation

Route: Standard

Deliverable: the standalone `internal/ui/copyselection` feature module from
`.scratch/copy-selection/issues/01-build-copy-selection-module.md`, with its
public lifecycle, pointer interaction, visuals, geometry transforms, and PNG
crop encoder covered through the same seam the viewer will use.

Non-goals: viewer/zoom composition, clipboard dispatch, menus, shortcuts,
translations, manuals, and `ARCHITECTURE.md`; those belong to tickets 02-07.

## Acceptance criteria

1. The public feature seam draws, replaces, moves, and resizes a clamped
   image-region selection, including crossed handles.
   Verify: `go test ./internal/ui/copyselection -run 'Test(DrawSelection|InvalidReplacement|MoveSelection|ResizeSelection|CrossedResizeHandle)$'`
2. Pixel bounds round outward and remain stable across `ViewChanged` geometry.
   Verify: `go test ./internal/ui/copyselection -run 'Test(PixelBounds|ViewportTransform|HiDPIGeometry)$'`
3. Lifecycle, cursor, visual, button, scroll, busy, success, and retry behavior
   matches ticket 01.
   Verify: `go test ./internal/ui/copyselection -run 'Test(ModeActivation|VisualState|CursorState|CopyButtonVisibility|CopyLifecycle|ScrollForwarding)$'`
4. PNG cropping validates bounds, produces exact selected pixels, preserves
   alpha, and contains no source metadata.
   Verify: `go test ./internal/ui/copyselection -run 'TestPNG'`
5. The complete ticket package is race-clean.
   Verify: `go test -race ./internal/ui/copyselection/...`

## Tasks

### Task 1 - Public seam and draw/commit tracer

Owner: T0 inline
Files: create `internal/ui/copyselection/*.go`, `*_test.go`
Depends: none
Contract: ticket 01's exact exported types and methods; `PNG(image.Image,
image.Rectangle) ([]byte, error)` is the package crop function.
Test: drag in both directions through the overlay and commit literal pixel
bounds through the copy callback.
Verify: focused draw test.
Budget: 0 spawns; 1 review round; full suite no.

### Task 2 - Editing, viewport, visuals, and lifecycle

Owner: T0 inline
Files: same package only
Depends: Task 1
Contract: no additional exported geometry or renderer state.
Test: vertical behavior slices for replacement, move, eight handles/crossing,
viewport/HiDPI changes, cursors, visuals, scroll, and completion.
Verify: criteria 1-3 commands.
Budget: 0 spawns; 1 review round; full suite no.

### Task 3 - PNG crop

Owner: T0 inline
Files: same package only
Depends: Task 1
Contract: `PNG` returns zero-origin PNG bytes or a recoverable validation or
encoding error.
Test: literal multicolor and alpha pixels plus invalid rectangles.
Verify: criterion 4 command.
Budget: 0 spawns; 1 review round; full suite no.

### Task 4 - Review and final gate

Owner: T0 inline
Files: ticket status/answer and this ledger
Depends: Tasks 1-3
Test: negatively verify one geometry guard, then restore it.
Verify: ticket command followed by repository final gate.
Budget: 0 spawns; 1 review round; full suite yes, once.

Task graph: Task 1 -> Tasks 2 and 3 -> Task 4. Tasks stay inline because their
files and contract overlap; scripting does not apply.

## Cost ledger

| Task | Spawns (budget/actual) | Review rounds | Full suite | Notes |
|------|------------------------|---------------|------------|-------|
| 1 | 0 / 0 | 0 | no | pending |
| 2 | 0 / 0 | 0 | no | pending |
| 3 | 0 / 0 | 0 | no | pending |
| 4 | 0 / 0 | 0 | yes | pending |
