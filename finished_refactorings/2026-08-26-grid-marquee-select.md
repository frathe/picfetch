# Grid marquee selection Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Controller extra (this session):** After every task, the parent agent reviews the diff and fixes it before dispatching the next task. Do not start Task N+1 until that review lands. Do not commit (`AGENTS.md`). End with a suggested commit message for the user. Do not dispatch Task 1 until Florian has confirmed the locked product decisions (or explicitly said to proceed with the defaults).

**Goal:** In the grid overview, click-and-drag draws a selection rectangle and picks every thumbnail the rectangle intersects, using the existing `internal/selection` set and cell tints.

**Architecture:** A transparent `fyne.Draggable` catcher sits *under* `GridWrap` in the grid body stack so cell taps, hover, wheel-scroll, and the scrollbar keep working. A `WithoutLayout` layer on top of the wrap draws the rectangle. Hit-testing is pure geometry against `cellSize` + theme padding + `GridWrap.GetScrollOffset()`, returning display indices that `fileIndex` maps to host indices (same as Shift+click). Unmodified drag replaces the selection; Shift or the shortcut modifier unions with the selection that existed when the drag began.

**Tech Stack:** Go 1.26, Fyne v2.8 `widget.GridWrap` / `fyne.Draggable` / `canvas.Rectangle`, existing `internal/selection.Set`, `internal/ui/grid` tests via `openGrid` + `uitest.UIQueue`.

## Why this approach

GridWrap items are `Tappable` + `Hoverable`, not `Draggable`. The wrap's `VScroll` is not draggable either (only the scrollbar thumb is). Fyne's hit-test walks every visible object and keeps the *last* match, so a drag on a thumbnail today can land on the zoom widget *under* the opaque grid overlay and pan a hidden image. A catcher in the grid stack that is itself `Draggable` both implements the feature and stops that leak.

Alternatives rejected:

- **Draggable cell wrappers:** cannot start a drag in the padding or the empty region below the last row; more objects to keep in sync with recycling.
- **Catcher on top of GridWrap:** last `Draggable` wins, so the overlay would steal scrollbar-thumb drags.
- **Modifier-only marquee:** unmodified drag would stay a no-op (or keep leaking to zoom). The Windows Ctrl+click Fyne bug makes an unmodified drag the reliable mouse path.

## Locked product decisions

These are the defaults. Change them only if Florian says so before Task 1.

1. **Unmodified click-drag** replaces the selection with every *visible* cell the rectangle intersects (including cells the rect only clips). A movement below Fyne's 2px drag threshold remains a tap and still opens the image.
2. **Shift+drag** and **Cmd/Ctrl+drag** union the rectangle's cells with the selection frozen at mouse-down. Cmd/Ctrl+click and Shift+click are unchanged.
3. **No auto-scroll** when the pointer hits the edge. Wheel/trackpad still scroll; v1 does not pan the grid during the drag.
4. **Escape during an in-progress drag** restores the pre-drag selection and hides the rectangle (it does not also run the normal "clear selection" stage on that same press).
5. **On drag end:** keep the grid open, `Unfocus()`, move the highlight ring to the display cell under the drag origin (or the first selected cell if the origin sits in a gutter). Do not call `ShowImage`.
6. **Filtered / hide-duplicates / browse-duplicates grids** marquee in *display* space, then map through `fileIndex`, matching Shift+click.
7. **No new `lang.L` strings.** The existing `"%d selected"` bar is enough. Manual EN/DE get a sentence each.
8. **No golden screenshot regeneration.**

## Global Constraints

- Do not commit. `AGENTS.md`: “Do not run `git commit`. End with a suggested commit message for the user.”
- Do not add `TODO`/`FIXME` comments. Open work stays in `todos.md`.
- Every user-visible string is `lang.L("English text")` with the same key in every `translations/*.json` bundle. This feature adds none.
- Feature packages own widgets/state and declare a narrow `Host`. Do not pass `appState`. Marquee stays inside `internal/ui/grid`.
- `internal/selection` stays Fyne-free. Geometry that needs `cellSize` / padding lives in `internal/ui/grid`.
- Grid tests use `newOverview` / `openGrid` and `g.ui` (`uitest.UIQueue`). Do not switch completions to a raw `fyne.Do`.
- Drive gestures by calling methods (`Dragged` / `DragEnd` / `applyMarquee`), not `time.Sleep`. Fyne's `test.Drag` is one-shot and not useful for a live marquee.
- Preserve `gofmt` / `goimports -local github.com/frathe/picfetch`. Tabs, not spaces.
- Update `ARCHITECTURE.md` in the same change that adds `marquee.go`.
- Subagents must not start Task N+1 themselves. They stop after their task's verification and report.
- Do not "fix" the Windows Ctrl+click Fyne bug. Out of scope.

## Subagent models

Use the least powerful listed model that can handle the role. Available slugs: `composer-2.5-fast`, `cursor-grok-4.5-high-fast`, `cursor-grok-4.6-xhigh`, `claude-opus-5-thinking-high`.

| Role | Model | Why |
|------|--------|-----|
| Task 1 implementer | `cursor-grok-4.5-high-fast` | Pure geometry + tests; complete code in the brief. |
| Task 2 implementer | `cursor-grok-4.5-high-fast` | One method on `selection.Set`. |
| Task 3 implementer | `cursor-grok-4.6-xhigh` | Selection apply path, filter mapping, live refresh. |
| Task 4 implementer | `cursor-grok-4.6-xhigh` | Fyne hit-testing / overlay order. If the implementer reports `BLOCKED` on events not reaching the catcher, re-dispatch this task only with `claude-opus-5-thinking-high`. |
| Task 5 implementer | `cursor-grok-4.5-high-fast` | Manual / ARCHITECTURE / todos copy. |
| Task reviewer (Tasks 1, 2, 5) | `cursor-grok-4.5-high-fast` | Mid-tier floor. |
| Task reviewer (Tasks 3, 4) | `cursor-grok-4.6-xhigh` | Coordinate and event-order bugs are easy to miss. |
| Parent review / fix after each task | this session (do not dispatch) | Review and fix after every step. |
| Final whole-branch review | `claude-opus-5-thinking-high` | Cross-task: click still opens, drag does not, scrollbar/wheel still work. |

Subagent type: `generalPurpose` for implementers and reviewers. Do not use `go-expert` to write the code (it is for design questions). Do not dispatch two implementers in parallel.

## File structure

- Create: `internal/ui/grid/marquee.go` — `marqueeRect`, `cellsIntersecting`, `marqueeCatcher`, `applyMarquee`, drag lifecycle on `Overview`
- Create: `internal/ui/grid/marquee_test.go` — geometry tests (Task 1) then apply/drag tests (Tasks 3–4)
- Modify: `internal/selection/selection.go` — `SetAnchor`
- Modify: `internal/selection/selection_test.go`
- Modify: `internal/ui/grid/grid.go` — `Overview` fields; body stack order in `New`
- Modify: `internal/ui/grid/nav.go` — `escape` cancels an in-progress marquee first
- Modify: `internal/ui/widgets/style.go` — marquee fill/stroke constants + `NewMarqueeRect`
- Modify: `internal/ui/help/manual.md`, `internal/ui/help/manual_de.md`
- Modify: `ARCHITECTURE.md` — grid package row + “Where to look”
- Modify: `todos.md` — point this item at this plan (do not move it to Done)

Do not split `grid.go` in this work. Do not add an `internal/ui` glue file; batch actions already consume `Targets()`.

## Overlay order (load-bearing)

Inside `New`, the grid *body* stack must be exactly:

```
catcher                // back: transparent fyne.Draggable
Padded(wrap)           // GridWrap (taps, hover, wheel, scrollbar)
Center(empty)          // "No file names match"
WithoutLayout(rect)    // front: marquee stroke, hidden until a drag
```

Walk is back-to-front and the last match wins:

| Event | Last match | Why |
|-------|------------|-----|
| Tap | `gridWrapItem` | catcher is not `Tappable` |
| Hover | `gridWrapItem` | catcher is not `Hoverable` |
| Wheel | `VScroll` | catcher is not `Scrollable` |
| Drag on a cell / gutter | catcher | wrap items are not `Draggable` |
| Drag on scrollbar thumb | scrollbar | walked after the catcher, still contains the point |
| Drag on zoom under the grid | catcher | later in the window tree than `zoom.imageWidget` |

The search bar stays the Border *top* slot, outside this stack. Do not marquee over it.

---

### Task 1: Cell–rectangle intersection

**Files:**
- Create: `internal/ui/grid/marquee.go`
- Test: `internal/ui/grid/marquee_test.go`

**Interfaces:**
- Consumes: `cellSize` (`120`) in `internal/ui/grid/grid.go`
- Produces: `marqueeRect`, `normRect`, `cellsIntersecting` as specified below. Later tasks call these; do not rename.

- [ ] **Step 1: Write the failing tests**

Create `internal/ui/grid/marquee_test.go` with a `testing` import only (no Fyne widgets, no `openGrid`):

```go
package grid

import (
	"slices"
	"testing"

	"fyne.io/fyne/v2"
)

func TestNormRect_OrdersCorners(t *testing.T) {
	got := normRect(fyne.NewPos(80, 90), fyne.NewPos(10, 20))
	if got.minX != 10 || got.minY != 20 || got.maxX != 80 || got.maxY != 90 {
		t.Errorf("normRect = %+v, want min=(10,20) max=(80,90)", got)
	}
}

func TestCellsIntersecting_PitchAndGaps(t *testing.T) {
	// 3 columns, cell 120, pad 4, pitch 124. Ten cells → last row has one.
	grid := marqueeGrid{cols: 3, count: 10, cell: 120, pad: 4}

	tests := []struct {
		name     string
		a, b     fyne.Position
		want     []int
	}{
		{"single cell from its origin", fyne.NewPos(0, 0), fyne.NewPos(120, 120), []int{0}},
		{"clip into neighbour column", fyne.NewPos(10, 10), fyne.NewPos(130, 10), []int{0, 1}},
		{"gutter only selects nothing", fyne.NewPos(121, 0), fyne.NewPos(123, 120), nil},
		{"partial overlap still selects", fyne.NewPos(119, 0), fyne.NewPos(121, 10), []int{0}},
		{"two rows two cols", fyne.NewPos(10, 10), fyne.NewPos(130, 130), []int{0, 1, 3, 4}},
		{"beyond last cell clamps", fyne.NewPos(0, 124*3), fyne.NewPos(50, 124*3+50), []int{9}},
		{"empty grid", fyne.NewPos(0, 0), fyne.NewPos(400, 400), nil},
		{"zero cols", fyne.NewPos(0, 0), fyne.NewPos(10, 10), nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := grid
			if tt.name == "empty grid" {
				g.count = 0
			}
			if tt.name == "zero cols" {
				g.cols = 0
			}
			got := cellsIntersecting(normRect(tt.a, tt.b), g)
			if tt.want == nil {
				if len(got) != 0 {
					t.Errorf("cellsIntersecting = %v, want empty", got)
				}
				return
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("cellsIntersecting = %v, want %v", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -run 'TestNormRect_OrdersCorners|TestCellsIntersecting_PitchAndGaps' ./internal/ui/grid/`

Expected: FAIL with `normRect` / `cellsIntersecting` undefined.

- [ ] **Step 3: Write the geometry**

Create `internal/ui/grid/marquee.go`:

```go
package grid

import "fyne.io/fyne/v2"

// marqueeRect is an axis-aligned rectangle in wrap-content coordinates
// (origin at the top-left of cell 0, y increasing down, scroll already added).
type marqueeRect struct {
	minX, minY, maxX, maxY float32
}

func normRect(a, b fyne.Position) marqueeRect {
	return marqueeRect{
		minX: min(a.X, b.X),
		minY: min(a.Y, b.Y),
		maxX: max(a.X, b.X),
		maxY: max(a.Y, b.Y),
	}
}

// marqueeGrid is the laid-out cell lattice cellsIntersecting tests against.
// cols is GridWrap.ColumnCount(); cell and pad must match itemMin and
// theme padding (GridWrap lays out at pitch cell+pad).
type marqueeGrid struct {
	cols, count int
	cell, pad   float32
}

func (g marqueeGrid) pitch() float32 { return g.cell + g.pad }

// cellsIntersecting returns display indices whose cell boxes overlap r,
// ascending. A rect that only covers the padding gutter between cells
// selects nothing. A purely horizontal or vertical drag is a line
// (max == min on one axis); inflate that axis by 1px so it still hits
// the row or column the pointer is in.
func cellsIntersecting(r marqueeRect, g marqueeGrid) []int {
	if g.cols < 1 || g.count < 1 || g.cell <= 0 {
		return nil
	}
	if r.maxX == r.minX {
		r.maxX++
	}
	if r.maxY == r.minY {
		r.maxY++
	}

	pitch := g.pitch()
	rows := (g.count + g.cols - 1) / g.cols

	col0 := max(0, int(r.minX/pitch))
	col1 := min(g.cols-1, int(r.maxX/pitch))
	row0 := max(0, int(r.minY/pitch))
	row1 := min(rows-1, int(r.maxY/pitch))

	out := make([]int, 0)
	for row := row0; row <= row1; row++ {
		for col := col0; col <= col1; col++ {
			id := row*g.cols + col
			if id < 0 || id >= g.count {
				continue
			}
			x0 := float32(col) * pitch
			y0 := float32(row) * pitch
			x1 := x0 + g.cell
			y1 := y0 + g.cell
			if r.maxX <= x0 || r.minX >= x1 || r.maxY <= y0 || r.minY >= y1 {
				continue
			}
			out = append(out, id)
		}
	}
	return out
}
```

Do not add the catcher widget or any `Overview` methods in this task.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -run 'TestNormRect_OrdersCorners|TestCellsIntersecting_PitchAndGaps' ./internal/ui/grid/`

Expected: PASS.

- [ ] **Step 5: Stop**

Do not commit. Report: files created, test command, pass/fail.

---

### Task 2: `Set.SetAnchor`

**Files:**
- Modify: `internal/selection/selection.go`
- Test: `internal/selection/selection_test.go`

**Interfaces:**
- Consumes: existing `Set` (`Toggle` already writes `anchor` / `hasAnchor`)
- Produces: `func (s *Set) SetAnchor(i int)` — sets the range-extension anchor to `i` without changing membership. `Anchor()` afterwards returns `(i, true)`.

- [ ] **Step 1: Write the failing test**

Append to `internal/selection/selection_test.go`:

```go
func TestSetAnchor_DoesNotChangeMembership(t *testing.T) {
	s := New()
	s.Toggle(1)
	s.Add(2)

	s.SetAnchor(9)

	if want := []int{1, 2}; !slices.Equal(s.Indices(), want) {
		t.Errorf("Indices() = %v after SetAnchor(9), want %v", s.Indices(), want)
	}
	if a, ok := s.Anchor(); !ok || a != 9 {
		t.Errorf("Anchor() = (%d, %v), want (9, true)", a, ok)
	}
}

func TestSetAnchor_WorksOnAnEmptySet(t *testing.T) {
	s := New()

	s.SetAnchor(3)

	if s.Len() != 0 {
		t.Errorf("Len() = %d, want 0", s.Len())
	}
	if a, ok := s.Anchor(); !ok || a != 3 {
		t.Errorf("Anchor() = (%d, %v), want (3, true)", a, ok)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run 'TestSetAnchor' ./internal/selection/`

Expected: FAIL with `SetAnchor` undefined.

- [ ] **Step 3: Implement**

Add to `internal/selection/selection.go`, immediately after `Anchor`:

```go
// SetAnchor names i as the index a later range extension measures from,
// without adding or removing members. A marquee uses this so a following
// Shift+click extends from where the drag started rather than from whatever
// Toggle last happened to touch.
func (s *Set) SetAnchor(i int) {
	s.anchor = i
	s.hasAnchor = true
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -run 'TestSetAnchor|TestToggle|TestReplace' ./internal/selection/`

Expected: PASS. Existing `Replace` still leaves the previous anchor alone; marquee will call `SetAnchor` itself.

- [ ] **Step 5: Stop**

Do not commit. Report files and test output.

---

### Task 3: `applyMarquee` on the overview

**Files:**
- Modify: `internal/ui/grid/marquee.go` (add `applyMarquee` only)
- Modify: `internal/ui/grid/marquee_test.go` (add `openGrid` tests)
- Modify: `internal/ui/grid/grid.go` only if you must add unexported fields used by `applyMarquee`. Do **not** change the overlay stack yet.

**Interfaces:**
- Consumes: `cellsIntersecting`, `normRect`, `marqueeGrid` (Task 1); `(*Set).Replace`, `(*Set).SetAnchor` (Task 2); `Overview.fileIndex`, `count`, `syncTopBar`, `Host.ForceRepaint`
- Produces: `func (g *Overview) applyMarquee(origin, at fyne.Position, add bool)` with wrap-content coordinates (padding and scroll already applied by the caller). `cellAtPoint` and `unionSorted` as unexported helpers in `marquee.go`.

Coordinate contract for `applyMarquee`: `origin` and `at` are in **wrap-content space** — `(0,0)` is the top-left of display cell 0, y includes scroll (a scrolled-down grid uses larger y). Task 4 is what translates catcher-local events into this space.

Semantics:

- `ids := cellsIntersecting(...)` mapped through `fileIndex`, skipping `< 0`.
- `add == false`: `g.sel.Replace(hostIDs)`.
- `add == true`: `Replace` the union of `g.marqueeSaved` and `hostIDs`. If `g.marqueeSaved` is nil, treat it as empty.
- Then `SetAnchor` to the host index of the display cell under `origin` when that cell is a real cell (`fileIndex` of `cellAtPoint(origin)`), otherwise to the first host id in the new set if any.
- `g.wrap.Refresh(); g.syncTopBar(); g.host.ForceRepaint()`.
- Do not `Close` or `ShowImage`. Do not `Unfocus` here (Task 4 does that on `DragEnd`).
- Do not move the highlight here (Task 4 on `DragEnd`).

`g.marqueeSaved` is `[]int`, the host indices frozen at drag start. Task 3 tests may set it directly (same package).

- [ ] **Step 1: Write the failing tests**

Append to `internal/ui/grid/marquee_test.go`. These need the Fyne test app (`TestMain` in `harness_test.go` already provides it) and `openGrid`:

```go
func TestApplyMarquee_ReplacesTheSelection(t *testing.T) {
	g, host := openGrid(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg", "e.jpg", "f.jpg")
	g.wrap.Resize(fyne.NewSize(cellSize*3+8, cellSize*2+4))
	if g.wrap.ColumnCount() != 3 {
		t.Fatalf("ColumnCount() = %d, want 3; applyMarquee tests assume this geometry", g.wrap.ColumnCount())
	}

	// Origin in cell 0, corner in cell 4 (display 0,1,3,4) at 3 columns.
	g.applyMarquee(fyne.NewPos(10, 10), fyne.NewPos(cellSize+10, cellSize+10), false)

	if want := []int{0, 1, 3, 4}; !slices.Equal(g.Selection(), want) {
		t.Errorf("Selection() = %v, want %v", g.Selection(), want)
	}
	if len(host.shown) != 0 {
		t.Errorf("ShowImage calls = %v, want none", host.shown)
	}
	if !g.Visible() {
		t.Error("a marquee must leave the grid open")
	}
}

func TestApplyMarquee_AddUnionsWithTheSnapshot(t *testing.T) {
	g, host := openGrid(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg")
	g.wrap.Resize(fyne.NewSize(cellSize*2+4, cellSize*2+4))
	if g.wrap.ColumnCount() != 2 {
		t.Fatalf("ColumnCount() = %d, want 2", g.wrap.ColumnCount())
	}
	click(g, host, 0, fyne.KeyModifierShortcutDefault)
	g.marqueeSaved = g.Selection()

	// Cell 3 only, union with the snapshot's 0.
	g.applyMarquee(fyne.NewPos(cellSize+10, cellSize+10), fyne.NewPos(cellSize+20, cellSize+20), true)

	if want := []int{0, 3}; !slices.Equal(g.Selection(), want) {
		t.Errorf("Selection() = %v, want %v", g.Selection(), want)
	}
}

func TestApplyMarquee_UsesDisplayIndicesWhenFiltered(t *testing.T) {
	g, _ := openGrid(t, "sun1.jpg", "moon.jpg", "sun2.jpg", "star.jpg", "sun3.jpg")
	typeQuery(g, "sun") // display 0,1,2 → host 0,2,4
	g.wrap.Resize(fyne.NewSize(cellSize*3+8, cellSize))
	if g.wrap.ColumnCount() != 3 {
		t.Fatalf("ColumnCount() = %d, want 3", g.wrap.ColumnCount())
	}

	g.applyMarquee(fyne.NewPos(10, 10), fyne.NewPos(cellSize*2+10, 20), false)

	if want := []int{0, 2, 4}; !slices.Equal(g.Selection(), want) {
		t.Errorf("Selection() = %v, want %v (host indices of the three suns)", g.Selection(), want)
	}
}

func TestApplyMarquee_SetsTheAnchorToTheOriginCell(t *testing.T) {
	g, host := openGrid(t, "a.jpg", "b.jpg", "c.jpg")
	g.wrap.Resize(fyne.NewSize(cellSize*3+8, cellSize))
	if g.wrap.ColumnCount() != 3 {
		t.Fatalf("ColumnCount() = %d, want 3", g.wrap.ColumnCount())
	}

	g.applyMarquee(fyne.NewPos(cellSize+10, 10), fyne.NewPos(cellSize*2+10, 20), false)

	click(g, host, 0, fyne.KeyModifierShift)
	if want := []int{0, 1, 2}; !slices.Equal(g.Selection(), want) {
		t.Errorf("Selection() = %v, want 0..2 extending from origin cell 1", g.Selection())
	}
}
```

`openGrid` does not put the overlay in a window; `g.wrap.Resize` is what `ColumnCount` reads. The `cellSize*3+8` width matches GridWrap's pitch (`itemMin+padding`, default pad 4): `floor((368+4)/124) = 3`. If `ColumnCount` is not 3, fail the test rather than asserting the wrong cells.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -run TestApplyMarquee ./internal/ui/grid/`

Expected: FAIL with `applyMarquee` undefined.

- [ ] **Step 3: Implement `applyMarquee`**

In `marquee.go` (same package, so it can use `g.sel`, `g.wrap`, `g.marqueeSaved`):

Add to `Overview` in `grid.go` (fields only):

```go
// marqueeSaved is the host-index selection frozen at mouse-down, so a
// Shift/Cmd drag unions against what the user started with rather than
// against a live set that the previous Dragged already replaced.
marqueeSaved []int
```

Implementation:

```go
func (g *Overview) applyMarquee(origin, at fyne.Position, add bool) {
	cols := g.wrap.ColumnCount()
	pad := g.wrap.Theme().Size(theme.SizeNamePadding)
	hostIDs := make([]int, 0)
	for _, d := range cellsIntersecting(normRect(origin, at), marqueeGrid{
		cols: cols, count: g.count(), cell: cellSize, pad: pad,
	}) {
		if i := g.fileIndex(d); i >= 0 {
			hostIDs = append(hostIDs, i)
		}
	}

	if add {
		hostIDs = unionSorted(g.marqueeSaved, hostIDs)
	}
	g.sel.Replace(hostIDs)

	if i := g.fileIndex(cellAtPoint(origin, marqueeGrid{cols: cols, count: g.count(), cell: cellSize, pad: pad})); i >= 0 {
		g.sel.SetAnchor(i)
	} else if len(hostIDs) > 0 {
		g.sel.SetAnchor(hostIDs[0])
	}

	g.wrap.Refresh()
	g.syncTopBar()
	g.host.ForceRepaint()
}

func cellAtPoint(p fyne.Position, grid marqueeGrid) int {
	ids := cellsIntersecting(marqueeRect{minX: p.X, minY: p.Y, maxX: p.X + 1, maxY: p.Y + 1}, grid)
	if len(ids) == 0 {
		return -1
	}
	return ids[0]
}

func unionSorted(a, b []int) []int {
	seen := make(map[int]struct{}, len(a)+len(b))
	for _, xs := range [][]int{a, b} {
		for _, i := range xs {
			seen[i] = struct{}{}
		}
	}
	out := make([]int, 0, len(seen))
	for i := range seen {
		out = append(out, i)
	}
	slices.Sort(out)
	return out
}
```

Import `slices` and `fyne.io/fyne/v2/theme`. `cellAtPoint` uses a 1×1 rect so a point on a cell counts as an intersection; a point in the gutter returns -1.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -run 'TestApplyMarquee|TestOnSelected_PlainClickStillOpensTheImage' ./internal/ui/grid/`

Expected: PASS. The plain-click regression must still pass; this task must not touch `OnSelected`.

- [ ] **Step 5: Stop**

Do not commit.

---

### Task 4: Catcher widget, rectangle, overlay, Escape

**Files:**
- Modify: `internal/ui/grid/marquee.go` — `marqueeCatcher`, drag start/move/end
- Modify: `internal/ui/grid/grid.go` — `New` body stack; `Overview` fields
- Modify: `internal/ui/grid/nav.go` — `escape` first stage
- Modify: `internal/ui/widgets/style.go` — `NewMarqueeRect`
- Modify: `internal/ui/grid/marquee_test.go` — drag + escape tests
- Modify: `internal/ui/grid/selection_test.go` only if a comment should mention the marquee; prefer not to touch it

**Interfaces:**
- Consumes: `applyMarquee` (Task 3)
- Produces: catcher as `fyne.Draggable`; `Overview.marqueeDragging bool`; visual rect hidden except while dragging

#### Event translation

`DragEvent.Position` is catcher-local. The wrap sits in a `container.NewPadded` sibling, so wrap-content origin is `(theme.Padding(), theme.Padding())` inside the catcher, plus `g.wrap.GetScrollOffset()` on y:

```
content := fyne.NewPos(
    ev.Position.X - pad,
    ev.Position.Y - pad + g.wrap.GetScrollOffset(),
)
```

`theme.Padding()` is what `layout.PaddedLayout` uses (not `SizeNameInnerPadding`). Use the same source in production and tests: `g.wrap.Theme().Size(theme.SizeNamePadding)` is the GridWrap pitch padding and matches default `theme.Padding()` in tests.

First `Dragged` of a gesture:

```
g.marqueeOrigin = content.Subtract(fyne.NewPos(ev.Dragged.DX, ev.Dragged.DY))
g.marqueeSaved = append([]int(nil), g.sel.Indices()...)
g.marqueeDragging = true
```

(The first event's `Dragged` is the delta that crossed Fyne's 2px threshold; subtracting it recovers the press point closely enough.)

Each `Dragged` (including the first):

```
add := false
if toggle, extend := pickModifier(g.host.Modifiers()); toggle || extend {
    add = true
}
g.applyMarquee(g.marqueeOrigin, content, add)
g.placeMarqueeRect(originCatcher, atCatcher) // catcher-local, not content
```

`DragEnd`:

```
g.marqueeDragging = false
g.marqueeBox: rect.Hide(); rect.Refresh()
g.host.Unfocus()
if d := cellAtPoint(g.marqueeOrigin, ...); d >= 0 { g.setHighlight(d) }
g.marqueeSaved = nil
```

Do not call `ShowImage`. Do not `Close`.

#### Visual

Add to `internal/ui/widgets/style.go`:

```go
const (
    MarqueeStrokeWidth float32 = 1
    MarqueeFillAlpha   uint8   = 40
)

func NewMarqueeRect() *canvas.Rectangle {
    c := color.NRGBAModel.Convert(theme.Color(theme.ColorNamePrimary)).(color.NRGBA)
    fill := c
    fill.A = MarqueeFillAlpha
    r := canvas.NewRectangle(fill)
    r.StrokeColor = c
    r.StrokeWidth = MarqueeStrokeWidth
    r.CornerRadius = RingRadius
    r.Hide()
    return r
}
```

`placeMarqueeRect` Move/Resizes `g.marqueeRect` inside `g.marqueeBox` (`container.NewWithoutLayout`). Show the rect while dragging; hide on end and on Escape-cancel. Stack layout must not be used for the rect itself.

#### Overlay wiring in `New`

After creating `g.wrap` / search bar / empty, before `g.overlay = ...`:

```go
g.catcher = newMarqueeCatcher(g)
g.marqueeRect = widgets.NewMarqueeRect()
g.marqueeBox = container.NewWithoutLayout(g.marqueeRect)

body := container.NewStack(
    g.catcher,
    container.NewPadded(g.wrap),
    container.NewCenter(g.empty),
    g.marqueeBox,
)
g.overlay = container.NewStack(backdrop, container.NewBorder(g.searchBar, nil, nil, nil, body))
```

`newMarqueeCatcher`:

```go
type marqueeCatcher struct {
    widget.BaseWidget
    g *Overview
}

func newMarqueeCatcher(g *Overview) *marqueeCatcher {
    c := &marqueeCatcher{g: g}
    c.ExtendBaseWidget(c)
    return c
}

func (c *marqueeCatcher) CreateRenderer() fyne.WidgetRenderer {
    return widget.NewSimpleRenderer(canvas.NewRectangle(color.Transparent))
}

func (c *marqueeCatcher) Dragged(ev *fyne.DragEvent) { c.g.marqueeDragged(ev) }
func (c *marqueeCatcher) DragEnd()                 { c.g.marqueeDragEnd() }
```

Declare `var _ fyne.Draggable = (*marqueeCatcher)(nil)`. Do **not** implement `Tappable`, `Hoverable`, `Scrollable`, or `desktop.Mouseable`.

#### Escape

In `nav.go` `escape`, add a stage *before* `g.sel.Len() > 0`:

```go
case g.marqueeDragging:
    g.cancelMarquee()
```

`cancelMarquee` restores `g.sel.Replace(g.marqueeSaved)`, `SetAnchor` is left as-is (or restored by Replace's "keep anchor" — the snapshot is membership only; restoring membership then `wrap.Refresh` / `syncTopBar` / hide rect / `marqueeDragging=false` / `marqueeSaved=nil` is enough).

#### Close

`Close` already `g.sel.Clear()`. Also set `marqueeDragging=false`, hide the rect, nil `marqueeSaved`, so a defensive Close during a drag cannot leave the catcher armed.

- [ ] **Step 1: Write the failing tests**

```go
func TestMarqueeDrag_SelectsWithoutOpening(t *testing.T) {
    g, host := openGrid(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg")
    layoutMarquee(t, g, 3)

    pad := g.wrap.Theme().Size(theme.SizeNamePadding)
    start := fyne.NewPos(pad+10, pad+10)
    end := fyne.NewPos(pad+cellSize+10, pad+cellSize+10)
    g.catcher.Dragged(&fyne.DragEvent{
        PointEvent: fyne.PointEvent{Position: start},
        Dragged:    fyne.NewDelta(8, 8),
    })
    g.catcher.Dragged(&fyne.DragEvent{
        PointEvent: fyne.PointEvent{Position: end},
        Dragged:    fyne.NewDelta(end.X-start.X, end.Y-start.Y),
    })
    g.catcher.DragEnd()

    if g.SelectionCount() < 2 {
        t.Errorf("Selection() = %v, want at least two cells", g.Selection())
    }
    if len(host.shown) != 0 {
        t.Errorf("ShowImage = %v, want none", host.shown)
    }
    if host.unfocused == 0 {
        t.Error("DragEnd should Unfocus, or later keys are swallowed")
    }
    if g.marqueeRect.Visible() {
        t.Error("the rectangle must hide when the drag ends")
    }
}

func TestMarqueeDrag_PlainClickPathUntouched(t *testing.T) {
    g, host := openGrid(t, "a.jpg", "b.jpg")
    click(g, host, 1, 0)
    if !slices.Equal(host.shown, []int{1}) {
        t.Errorf("ShowImage = %v, want [1]", host.shown)
    }
}

func TestEscape_CancelsAnInProgressMarquee(t *testing.T) {
    g, host := openGrid(t, "a.jpg", "b.jpg", "c.jpg")
    layoutMarquee(t, g, 3)
    click(g, host, 0, fyne.KeyModifierShortcutDefault)

    pad := g.wrap.Theme().Size(theme.SizeNamePadding)
    g.catcher.Dragged(&fyne.DragEvent{
        PointEvent: fyne.PointEvent{Position: fyne.NewPos(pad+cellSize+10, pad+10)},
        Dragged:    fyne.NewDelta(8, 0),
    })
    if g.SelectionCount() == 1 {
        t.Fatal("precondition: the drag should already have changed the selection")
    }

    g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyEscape})

    if want := []int{0}; !slices.Equal(g.Selection(), want) {
        t.Errorf("Selection() = %v after Escape, want the pre-drag snapshot %v", g.Selection(), want)
    }
    if !g.Visible() {
        t.Error("cancelling a marquee must not close the grid")
    }
}

func layoutMarquee(t *testing.T, g *Overview, cols int) {
    t.Helper()
    pad := float32(4)
    g.wrap.Resize(fyne.NewSize(cellSize*float32(cols)+pad*float32(cols-1), cellSize*2+pad))
    g.catcher.Resize(fyne.NewSize(g.wrap.Size().Width+2*pad, g.wrap.Size().Height+2*pad))
    if g.wrap.ColumnCount() != cols {
        t.Fatalf("ColumnCount() = %d, want %d", g.wrap.ColumnCount(), cols)
    }
}
```

`layoutMarquee` must Resize **both** wrap and catcher because tests never put `Overlay()` in a window.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -run 'TestMarqueeDrag|TestEscape_CancelsAnInProgressMarquee' ./internal/ui/grid/`

Expected: FAIL (`catcher` nil or methods missing).

- [ ] **Step 3: Implement**

Follow the overlay order, catcher, `marqueeDragged` / `marqueeDragEnd` / `cancelMarquee`, `NewMarqueeRect`, and `escape` stage exactly. `placeMarqueeRect` takes two catcher-local positions, `normRect`s them, `Move`s the rect to `(minX,minY)`, `Resize`s to `(maxX-minX, maxY-minY)`, `Show`, `Refresh`. Skip Show if the size is empty.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -race -count=1 ./internal/ui/grid/`

Expected: PASS, including `TestOnSelected_PlainClickStillOpensTheImage`, Shift+click, Space, SelectAll, and the new marquee tests.

- [ ] **Step 5: Stop**

Do not commit. If `go test ./internal/ui/grid/` fails because `ColumnCount` is 1 (renderer never measured `itemMin`), call `g.wrap.Refresh()` inside `layoutMarquee` after Resize, or `win.SetContent(g.Overlay()); win.Resize(...)` using the window `newOverview` already created. Do not skip the test.

---

### Task 5: Docs

**Files:**
- Modify: `internal/ui/help/manual.md`
- Modify: `internal/ui/help/manual_de.md`
- Modify: `ARCHITECTURE.md`
- Modify: `todos.md`

**Interfaces:**
- Consumes: the behaviour Tasks 1–4 implemented (unmodified drag replaces; Shift/Cmd drag adds; Esc cancels in-progress drag)
- Produces: no code

- [ ] **Step 1: Manual (English)**

In `internal/ui/help/manual.md`, the bullet that starts **Select several at once** (~line 395), after the Cmd/Ctrl+click sentence, add one sentence:

> Drag a rectangle across the thumbnails to select everything it touches; hold Shift or Cmd/Ctrl while dragging to add to what was already picked.

In the keyboard reference (~line 562) change the Cmd/Ctrl+click / Shift+click line to also name the drag:

> **`Cmd`/`Ctrl+click`** / **`Shift+click`** / **click-and-drag** — (grid only) add one thumbnail / select the range / select every thumbnail the rectangle touches (Shift or Cmd/Ctrl+drag adds rather than replacing)

Same addition in the “Select in the grid” summary bullet (~line 932).

- [ ] **Step 2: Manual (German)**

In `internal/ui/help/manual_de.md`, matching **Mehrere auf einmal auswählen** (~line 446):

> Ziehen Sie ein Rechteck über die Miniaturansichten, um alles auszuwählen, das es berührt; halten Sie dabei Shift oder Cmd/Strg, um zur bestehenden Auswahl hinzuzufügen.

Keyboard reference (~line 638) and the summary **Im Raster auswählen** (~line 1069): add the same drag clause.

Do not bump the manual Version header.

- [ ] **Step 3: ARCHITECTURE.md**

Grid feature-package row: mention `marquee.go` (drag rectangle → `Targets()`).

“How do I act on several images at once?”: add `grid/marquee.go`.

- [ ] **Step 4: todos.md**

Leave the item under `## TODO`. Add a plan pointer:

```
- possibility to drag a selection rectangle in grid view to select multiple images.
  Plan: `docs/superpowers/plans/2026-08-26-grid-marquee-select.md`.
```

Do not move it to Done.

- [ ] **Step 5: Verify**

Run: `go test -run Locale ./` (or `go test -count=1 .` at the module root — `main_test.go` locale parity). Expected: PASS, because this task adds no `lang.L` keys.

Run: `gofmt -l internal/ui/grid/marquee.go internal/selection/selection.go internal/ui/widgets/style.go` — empty output.

- [ ] **Step 6: Stop**

Do not commit.

---

## Verification (controller, after Task 5)

From the repository root:

```
make fmt-check
go vet ./...
go build ./...
go test -race ./internal/ui/grid/ ./internal/selection/
go test -race ./...
```

`make fmt-check` is `goimports -local github.com/frathe/picfetch`.

Suggested commit message (user runs git commit):

```
Add click-and-drag rectangle selection in the grid overview.

Unmodified drag replaces the selection with every thumbnail the
rectangle intersects; Shift or Cmd/Ctrl+drag adds. Cell taps, hover,
wheel-scroll, and the scrollbar keep working.
```

## Progress ledger

Controller: append to `.superpowers/sdd/progress.md` after each approved task:

```
Task N: complete (no commit; review clean)
```
