package grid

import (
	"fmt"
	"slices"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

func TestScrollFollower_CanvasScrollKeepsRingVisible(t *testing.T) {
	g, host := scrollOverview(t, 60)
	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	mountScrollOverview(t, g, 244)
	g.wrap.ScrollToOffset(0)
	selection := g.Selection()
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(g.wrap).Add(fyne.NewPos(10, 10))

	test.Scroll(g.win.Canvas(), pos, 0, -124)
	if got := g.ScrollOffset(); got != 124 {
		t.Fatalf("scroll offset = %v, want 124 from canvas wheel event", got)
	}
	if got := g.Highlight(); got != 4 {
		t.Fatalf("ring = %d, want 4 (first fully visible row, column 1)", got)
	}
	if host.last() != 4 {
		t.Errorf("highlight notification = %d, want host 4", host.last())
	}
	if !slices.Equal(g.Selection(), selection) || len(host.shown) != 0 || host.index != 0 {
		t.Errorf("scroll changed selection/image: selection=%v shown=%v current=%d", g.Selection(), host.shown, host.index)
	}
	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	if g.Highlight() != 5 {
		t.Errorf("Right after scroll = %d, want 5", g.Highlight())
	}
}

func TestScrollFollower_EdgeRows(t *testing.T) {
	tests := []struct {
		name                   string
		count, ring, wantRing  int
		height, offset, deltaY float32
		wantOffset             float32
	}{
		{"retain fully visible cell", 60, 4, 4, 244, 0, -10, 10},
		{"down past partial top row", 60, 4, 7, 244, 124, -1, 125},
		{"up past partial bottom row", 60, 7, 4, 244, 124, 1, 123},
		{"down several rows", 60, 2, 11, 244, 0, -300, 300},
		{"up several rows", 60, 17, 8, 244, 496, 300, 196},
		{"exact bottom remains visible", 60, 7, 7, 244, 125, 1, 124},
		{"short final row clamps column", 10, 2, 9, 120, 0, -1000, 372},
		{"down in short viewport", 60, 1, 1, 80, 0, -10, 10},
		{"up in short viewport", 60, 10, 7, 80, 372, 80, 248},
		{"down after native scrollbar left ring below", 60, 40, 7, 368, 124, -1, 125},
		{"up after native scrollbar left ring above", 60, 1, 13, 368, 372, 1, 371},
		{"horizontal delta is inert", 60, 1, 1, 244, 124, 0, 124},
		{"clamped top is inert", 60, 10, 10, 244, 0, 100, 0},
		{"clamped bottom is inert", 60, 1, 1, 244, 2232, -100, 2232},
		{"content fits without scrolling", 2, 1, 1, 244, 0, -124, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, host := scrollOverview(t, tt.count)
			mountScrollOverview(t, g, max(120, tt.height))
			if tt.height < 120 {
				g.wrap.Resize(fyne.NewSize(g.wrap.Size().Width, tt.height))
			}
			g.wrap.Highlight(tt.ring)
			g.wrap.ScrollToOffset(tt.offset)
			pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(g.wrap).Add(fyne.NewPos(10, 10))
			test.Scroll(g.win.Canvas(), pos, 20, tt.deltaY)
			if g.Highlight() != tt.wantRing || g.ScrollOffset() != tt.wantOffset {
				t.Errorf("ring=%d offset=%v, want ring=%d offset=%v", g.Highlight(), g.ScrollOffset(), tt.wantRing, tt.wantOffset)
			}
			if len(host.shown) != 0 || host.index != 0 || g.SelectionCount() != 0 {
				t.Errorf("scroll changed selection/image: selection=%v shown=%v current=%d", g.Selection(), host.shown, host.index)
			}
		})
	}
}

func TestScrollFollower_OnlyScrollInput(t *testing.T) {
	f := newScrollFollower(nil)
	if _, ok := any(f).(fyne.Scrollable); !ok {
		t.Fatal("follower must receive scroll input")
	}
	for _, forbidden := range []struct {
		name    string
		matches bool
	}{
		{"tap", implements[fyne.Tappable](f)},
		{"secondary tap", implements[fyne.SecondaryTappable](f)},
		{"double tap", implements[fyne.DoubleTappable](f)},
		{"drag", implements[fyne.Draggable](f)},
		{"hover", implements[desktop.Hoverable](f)},
		{"mouse", implements[desktop.Mouseable](f)},
		{"focus", implements[fyne.Focusable](f)},
	} {
		if forbidden.matches {
			t.Errorf("scroll follower must not receive %s input", forbidden.name)
		}
	}
}

func implements[T any](value any) bool {
	_, ok := value.(T)
	return ok
}

func TestScrollFollower_CanvasPointerRouting(t *testing.T) {
	t.Run("hover", func(t *testing.T) {
		g, host := scrollOverview(t, 60)
		mountScrollOverview(t, g, 244)
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(g.wrap).Add(fyne.NewPos(134, 10))
		test.MoveMouse(g.win.Canvas(), pos)
		if g.Highlight() != 1 || host.last() != 1 {
			t.Fatalf("hover ring=%d notification=%d, want 1", g.Highlight(), host.last())
		}
	})
	t.Run("cell tap", func(t *testing.T) {
		g, host := scrollOverview(t, 60)
		mountScrollOverview(t, g, 244)
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(g.wrap).Add(fyne.NewPos(134, 10))
		test.TapCanvas(g.win.Canvas(), pos)
		if !slices.Equal(host.shown, []int{1}) || g.Visible() {
			t.Errorf("cell tap shown=%v grid visible=%v, want image 1 and closed grid", host.shown, g.Visible())
		}
	})
	t.Run("marquee", func(t *testing.T) {
		g, host := scrollOverview(t, 60)
		mountScrollOverview(t, g, 244)
		end := fyne.CurrentApp().Driver().AbsolutePositionForObject(g.wrap).Add(fyne.NewPos(134, 10))
		test.Drag(g.win.Canvas(), end, 124, 0)
		if !slices.Equal(g.Selection(), []int{0, 1}) || len(host.shown) != 0 {
			t.Errorf("canvas marquee selection=%v shown=%v, want selection [0 1] and no image load", g.Selection(), host.shown)
		}
	})
	for _, action := range []string{"thumb", "track"} {
		t.Run("scrollbar "+action, func(t *testing.T) {
			g, host := scrollOverview(t, 60)
			mountScrollOverview(t, g, 244)
			pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(g.wrap).Add(fyne.NewPos(g.wrap.Size().Width-1, 10))
			test.MoveMouse(g.win.Canvas(), pos)
			if action == "thumb" {
				test.Drag(g.win.Canvas(), pos, 0, 40)
			} else {
				test.TapCanvas(g.win.Canvas(), pos.Add(fyne.NewPos(0, 180)))
			}
			if g.ScrollOffset() <= 0 {
				t.Fatal("native scrollbar did not move the viewport")
			}
			if g.Highlight() != 0 || g.SelectionCount() != 0 || len(host.shown) != 0 {
				t.Errorf("scrollbar changed ring/selection/image: ring=%d selection=%v shown=%v", g.Highlight(), g.Selection(), host.shown)
			}
		})
	}
}

func scrollOverview(t *testing.T, count int) (*Overview, *fakeHost) {
	t.Helper()
	names := make([]string, count)
	for i := range names {
		names[i] = fmt.Sprintf("scroll-%02d.jpg", i)
	}
	return openGrid(t, names...)
}

func mountScrollOverview(t *testing.T, g *Overview, height float32) {
	t.Helper()
	g.win.SetPadded(false)
	g.win.SetContent(g.Overlay())
	bar := float32(0)
	if g.searchBar.Visible() {
		bar = g.searchBar.MinSize().Height + 4
	}
	g.win.Resize(fyne.NewSize(376, height+8+bar))
	if g.wrap.ColumnCount() != 3 || g.wrap.Size().Height != height {
		t.Fatalf("scroll geometry: columns=%d size=%v, want 3 columns and height %v", g.wrap.ColumnCount(), g.wrap.Size(), height)
	}
}

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
		name string
		a, b fyne.Position
		want []int
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

func TestMarqueeDrag_SelectsWithoutOpening(t *testing.T) {
	g, host := openGrid(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg")
	layoutMarquee(t, g, 3)

	pad := g.wrap.Theme().Size(theme.SizeNamePadding)
	start := fyne.NewPos(pad+10, pad+10)
	end := fyne.NewPos(pad+cellSize+10, pad+cellSize+10)
	g.catcher.Dragged(&fyne.DragEvent{
		Position: start,
		Dragged:  fyne.NewDelta(8, 8),
	})
	g.catcher.Dragged(&fyne.DragEvent{
		Position: end,
		Dragged:  fyne.NewDelta(end.X-start.X, end.Y-start.Y),
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

func TestMarqueeCatcher_IsOnlyDraggable(t *testing.T) {
	c := newMarqueeCatcher(nil)
	if _, ok := any(c).(fyne.Draggable); !ok {
		t.Error("catcher must be Draggable")
	}
	if _, ok := any(c).(fyne.Tappable); ok {
		t.Error("Tappable would steal cell taps")
	}
	if _, ok := any(c).(fyne.SecondaryTappable); ok {
		t.Error("SecondaryTappable would steal right-clicks")
	}
	if _, ok := any(c).(fyne.DoubleTappable); ok {
		t.Error("DoubleTappable would steal double-clicks")
	}
	if _, ok := any(c).(fyne.Scrollable); ok {
		t.Error("Scrollable would steal the wheel")
	}
	if _, ok := any(c).(desktop.Hoverable); ok {
		t.Error("Hoverable would steal cell hover")
	}
	if _, ok := any(c).(desktop.Mouseable); ok {
		t.Error("Mouseable would steal mouse events from GridWrap")
	}
}

func TestMarqueeDrag_UsesScrollOffset(t *testing.T) {
	names := make([]string, 12)
	for i := range names {
		names[i] = fmt.Sprintf("f%d.jpg", i)
	}
	g, host := openGrid(t, names...)
	layoutMarquee(t, g, 3)

	pad := g.wrap.Theme().Size(theme.SizeNamePadding)
	pitch := cellSize + pad
	g.wrap.ScrollToOffset(pitch)
	if g.wrap.GetScrollOffset() != pitch {
		t.Fatalf("GetScrollOffset() = %v, want %v", g.wrap.GetScrollOffset(), pitch)
	}

	start := fyne.NewPos(pad+10, pad+10)
	g.catcher.Dragged(&fyne.DragEvent{
		Position: start,
		Dragged:  fyne.NewDelta(8, 8),
	})
	g.catcher.DragEnd()

	if want := []int{3}; !slices.Equal(g.Selection(), want) {
		t.Errorf("Selection() = %v, want %v (row 1 col 0 once scrolled by one pitch)", g.Selection(), want)
	}
	if len(host.shown) != 0 {
		t.Errorf("ShowImage = %v, want none", host.shown)
	}
}

func TestMarqueeDrag_ShiftUnionsAgainstMouseDownSet(t *testing.T) {
	g, host := openGrid(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg")
	layoutMarquee(t, g, 3)
	click(g, host, 0, fyne.KeyModifierShortcutDefault)
	host.mods = fyne.KeyModifierShift

	pad := g.wrap.Theme().Size(theme.SizeNamePadding)
	start := fyne.NewPos(pad+10, pad+10)
	wide := fyne.NewPos(pad+cellSize+10, pad+10)
	g.catcher.Dragged(&fyne.DragEvent{
		Position: start,
		Dragged:  fyne.NewDelta(8, 0),
	})
	g.catcher.Dragged(&fyne.DragEvent{
		Position: wide,
		Dragged:  fyne.NewDelta(wide.X-start.X, 0),
	})
	g.catcher.Dragged(&fyne.DragEvent{
		Position: start,
		Dragged:  fyne.NewDelta(start.X-wide.X, 0),
	})
	g.catcher.DragEnd()

	if want := []int{0}; !slices.Equal(g.Selection(), want) {
		t.Errorf("Selection() = %v, want %v (union with the mouse-down set, not cells the rect already left)", g.Selection(), want)
	}
}

func TestEscape_CancelsAnInProgressMarquee(t *testing.T) {
	g, host := openGrid(t, "a.jpg", "b.jpg", "c.jpg")
	layoutMarquee(t, g, 3)
	click(g, host, 0, fyne.KeyModifierShortcutDefault)

	pad := g.wrap.Theme().Size(theme.SizeNamePadding)
	g.catcher.Dragged(&fyne.DragEvent{
		Position: fyne.NewPos(pad+cellSize+10, pad+10),
		Dragged:  fyne.NewDelta(8, 0),
	})
	if slices.Equal(g.Selection(), []int{0}) {
		t.Fatal("precondition: the drag should already have changed the selection")
	}

	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyEscape})

	if want := []int{0}; !slices.Equal(g.Selection(), want) {
		t.Errorf("Selection() = %v after Escape, want the pre-drag snapshot %v", g.Selection(), want)
	}
	if !g.Visible() {
		t.Error("cancelling a marquee must not close the grid")
	}

	// The driver keeps delivering Dragged until mouse-up. Those events
	// must not start a fresh replace, and the eventual DragEnd must not
	// move the ring as if the cancelled gesture had finished.
	ring := g.Highlight()
	notices := len(host.highlighted)
	unfocused := host.unfocused
	g.catcher.Dragged(&fyne.DragEvent{
		Position: fyne.NewPos(pad+cellSize*2+10, pad+10),
		Dragged:  fyne.NewDelta(8, 0),
	})
	if want := []int{0}; !slices.Equal(g.Selection(), want) {
		t.Errorf("Selection() = %v after a post-Escape Dragged, want the snapshot %v", g.Selection(), want)
	}
	g.catcher.DragEnd()
	if g.Highlight() != ring {
		t.Errorf("Highlight() = %d after cancelled DragEnd, want %d", g.Highlight(), ring)
	}
	if len(host.highlighted) != notices {
		t.Errorf("HighlightChanged after cancelled DragEnd: got %v", host.highlighted[notices:])
	}
	if host.unfocused != unfocused {
		t.Errorf("Unfocus calls after cancelled DragEnd = %d, want %d", host.unfocused, unfocused)
	}
}

func TestClose_DisarmsAnInProgressMarquee(t *testing.T) {
	g, _ := openGrid(t, "a.jpg", "b.jpg", "c.jpg")
	layoutMarquee(t, g, 3)

	pad := g.wrap.Theme().Size(theme.SizeNamePadding)
	g.catcher.Dragged(&fyne.DragEvent{
		Position: fyne.NewPos(pad+cellSize+10, pad+10),
		Dragged:  fyne.NewDelta(8, 0),
	})
	if g.SelectionCount() == 0 {
		t.Fatal("precondition: the drag should have selected something")
	}

	g.Close()
	g.catcher.Dragged(&fyne.DragEvent{
		Position: fyne.NewPos(pad+10, pad+10),
		Dragged:  fyne.NewDelta(8, 8),
	})
	if g.SelectionCount() != 0 {
		t.Errorf("Selection() = %v after Close then Dragged, want empty", g.Selection())
	}
	g.catcher.DragEnd()
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
