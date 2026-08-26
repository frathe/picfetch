package ui

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestStepImage_NextAndPrevWrap(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 8, 8, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 8, 8, color.White)
	dropAndWait(t, v, a, b)

	start := v.state.index
	v.StepImage(1)
	waitUntilLoaded(t, v)
	if v.state.index != (start+1)%2 {
		t.Fatalf("index after StepImage(1) = %d, want %d", v.state.index, (start+1)%2)
	}
	v.StepImage(1)
	waitUntilLoaded(t, v)
	if v.state.index != start {
		t.Fatalf("index after wrap = %d, want %d", v.state.index, start)
	}
	v.StepImage(-1)
	waitUntilLoaded(t, v)
	if v.state.index != (start+1)%2 {
		t.Fatalf("index after StepImage(-1) wrap = %d, want %d", v.state.index, (start+1)%2)
	}
}

func TestStepImage_NoopWithOneFile(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 8, 8, color.White)
	dropAndWait(t, v, a)
	v.StepImage(1)
	if v.state.index != 0 {
		t.Errorf("index = %d, want 0", v.state.index)
	}
}

func TestStepImage_SingleFileDropWalksFolderSiblings(t *testing.T) {
	v := newTestViewer(t)
	files := uitest.TempDirJPEGURIs(t, "c.jpg", "a.jpg", "b.jpg")
	var opened fyne.URI
	for _, u := range files {
		if u.Name() == "b.jpg" {
			opened = u
			break
		}
	}
	dropAndWait(t, v, opened)
	if v.state.files[v.state.index].Name() != "b.jpg" {
		t.Fatalf("setup: showing %q, want b.jpg", v.state.files[v.state.index].Name())
	}

	v.StepImage(1)
	waitUntilLoaded(t, v)
	if v.state.files[v.state.index].Name() != "c.jpg" {
		t.Fatalf("after StepImage(1) showing %q, want c.jpg (name-sort a,b,c)", v.state.files[v.state.index].Name())
	}

	v.StepImage(1)
	waitUntilLoaded(t, v)
	if v.state.files[v.state.index].Name() != "a.jpg" {
		t.Fatalf("after wrap showing %q, want a.jpg", v.state.files[v.state.index].Name())
	}

	v.StepImage(-1)
	waitUntilLoaded(t, v)
	if v.state.files[v.state.index].Name() != "c.jpg" {
		t.Fatalf("after StepImage(-1) showing %q, want c.jpg", v.state.files[v.state.index].Name())
	}
}

func TestHandleKeyEvent_LeftRightWalkFolderSiblings(t *testing.T) {
	v := newTestViewer(t)
	files := uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg")
	dropAndWait(t, v, files[0])
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
	waitUntilLoaded(t, v)
	if v.state.files[v.state.index].Name() != "b.jpg" {
		t.Fatalf("Right showing %q, want b.jpg", v.state.files[v.state.index].Name())
	}
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyLeft})
	waitUntilLoaded(t, v)
	if v.state.files[v.state.index].Name() != "a.jpg" {
		t.Fatalf("Left showing %q, want a.jpg", v.state.files[v.state.index].Name())
	}
}

func TestHandleKeyEvent_HomeEndOnFolderSiblings(t *testing.T) {
	v := newTestViewer(t)
	files := uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg", "c.jpg")
	var opened fyne.URI
	for _, u := range files {
		if u.Name() == "b.jpg" {
			opened = u
			break
		}
	}
	dropAndWait(t, v, opened)
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEnd})
	waitUntilLoaded(t, v)
	if v.state.files[v.state.index].Name() != "c.jpg" {
		t.Fatalf("End showing %q, want c.jpg", v.state.files[v.state.index].Name())
	}
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyHome})
	waitUntilLoaded(t, v)
	if v.state.files[v.state.index].Name() != "a.jpg" {
		t.Fatalf("Home showing %q, want a.jpg", v.state.files[v.state.index].Name())
	}
}

func TestAdvance_SingleFileDropWalksSiblings(t *testing.T) {
	v := newTestViewer(t)
	files := uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg")
	dropAndWait(t, v, files[0])
	v.Advance()
	waitUntilLoaded(t, v)
	if v.state.files[v.state.index].Name() != "b.jpg" {
		t.Fatalf("Advance showing %q, want b.jpg", v.state.files[v.state.index].Name())
	}
}

func TestStepImage_NoopWhileLoading(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 8, 8, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 8, 8, color.White)
	dropAndWait(t, v, a, b)
	start := v.state.index
	v.loading.Store(true)
	t.Cleanup(func() { v.loading.Store(false) })
	v.StepImage(1)
	if v.state.index != start {
		t.Errorf("index = %d, want %d while loading", v.state.index, start)
	}
}

func TestStepImage_NoopWhileDeleteConfirmVisible(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White)
	dropAndWait(t, v, a, b)
	v.deletion.Request()
	if !v.deletion.Visible() {
		t.Fatal("setup: delete confirmation should be visible")
	}
	start := v.state.index
	v.StepImage(1)
	if v.state.index != start {
		t.Errorf("index = %d, want %d while delete confirm is visible", v.state.index, start)
	}
}

func TestStepImage_NoopWhileExportPromptVisible(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White)
	dropAndWait(t, v, a, b)
	v.exportPrompt.Show(lang.L("Export as which format?"))
	if !v.exportPrompt.Visible() {
		t.Fatal("setup: export prompt should be visible")
	}
	start := v.state.index
	v.StepImage(1)
	if v.state.index != start {
		t.Errorf("index = %d, want %d while export prompt is visible", v.state.index, start)
	}
}

func TestStepImage_NoopWhileFyneDialogIsUp(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White)
	dropAndWait(t, v, a, b)
	v.showManageFavorites()
	if n := len(v.win.Canvas().Overlays().List()); n != 1 {
		t.Fatalf("setup: overlay count = %d, want Manage Favorites", n)
	}
	start := v.state.index
	v.StepImage(1)
	if v.state.index != start {
		t.Errorf("index = %d, want %d behind a Fyne dialog", v.state.index, start)
	}
}

func loadPatternedTriple(t *testing.T) *viewer {
	t.Helper()
	v := newTestViewer(t)
	a := uitest.PatternedJPEGURI(t, "a.jpg", 1)
	b := uitest.PatternedJPEGURI(t, "b.jpg", 1)
	c := uitest.PatternedJPEGURI(t, "c.jpg", 99)
	dropAndWait(t, v, a, b, c)
	if err := v.grid.Warm(); err != nil {
		t.Fatalf("Warm: %v", err)
	}
	return v
}

func TestStepImage_SkipsHiddenExtras(t *testing.T) {
	v := loadPatternedTriple(t)
	v.grid.SetHideDuplicates(true)
	waitUntilLoaded(t, v)
	if v.state.index != 0 {
		t.Fatalf("index = %d, want 0 (representative)", v.state.index)
	}

	v.StepImage(1)
	waitUntilLoaded(t, v)
	if v.state.index != 2 {
		t.Fatalf("index after StepImage(1) = %d, want 2 (skipped extra at 1)", v.state.index)
	}

	v.StepImage(1)
	waitUntilLoaded(t, v)
	if v.state.index != 0 {
		t.Fatalf("index after wrap = %d, want 0", v.state.index)
	}
}

func TestStepImage_HideDuplicatesShowsHighestResolution(t *testing.T) {
	v := newTestViewer(t)
	// ByName is the default; a/b/c keep drop order so index 0 is the
	// smaller copy, 1 the larger, 2 the unique shot.
	small := uitest.PatternedJPEGURISize(t, "a.jpg", 1, 64, 48)
	large := uitest.PatternedJPEGURISize(t, "b.jpg", 1, 192, 144)
	other := uitest.PatternedJPEGURI(t, "c.jpg", 99)
	dropAndWait(t, v, small, large, other)
	if err := v.grid.Warm(); err != nil {
		t.Fatalf("Warm: %v", err)
	}

	v.grid.SetHideDuplicates(true)
	v.grid.Settle()
	waitUntilLoaded(t, v)
	if v.grid.RepresentativeOf(0) != 1 || !v.grid.IsHiddenExtra(0) || v.grid.IsHiddenExtra(1) || v.grid.IsHiddenExtra(2) {
		t.Fatalf("same-seed pair did not group: extra(0)=%v extra(1)=%v extra(2)=%v rep=%d/%d Hamming=%d",
			v.grid.IsHiddenExtra(0), v.grid.IsHiddenExtra(1), v.grid.IsHiddenExtra(2),
			v.grid.RepresentativeOf(0), v.grid.RepresentativeOf(1),
			patternedHamming(t, small, large))
	}

	if v.state.index != 1 {
		t.Fatalf("index = %d, want 1 (larger copy of seed 1)", v.state.index)
	}

	v.StepImage(1)
	waitUntilLoaded(t, v)
	if v.state.index != 2 {
		t.Fatalf("after StepImage(1) index = %d, want 2 (skipped small extra at 0)", v.state.index)
	}

	v.StepImage(1)
	waitUntilLoaded(t, v)
	if v.state.index != 1 {
		t.Fatalf("after wrap index = %d, want 1", v.state.index)
	}
}

func patternedHamming(t *testing.T, a, b fyne.URI) int {
	t.Helper()
	ta, err := imaging.LoadThumbnail(a)
	if err != nil {
		t.Fatalf("thumb %s: %v", a.Name(), err)
	}
	tb, err := imaging.LoadThumbnail(b)
	if err != nil {
		t.Fatalf("thumb %s: %v", b.Name(), err)
	}
	return imaging.Hamming(imaging.DifferenceHash(ta), imaging.DifferenceHash(tb))
}

func TestHandleKeyEvent_DTogglesHideDuplicatesWhenGridClosed(t *testing.T) {
	v := loadPatternedTriple(t)
	if v.grid.Visible() {
		t.Fatal("setup: grid should be closed")
	}

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyD})
	if !v.grid.HideDuplicates() {
		t.Fatal("D with the grid closed should hide extras")
	}

	v.grid.SetHideDuplicates(false)
	v.ShowImage(1)
	waitUntilLoaded(t, v)
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyD})
	waitUntilLoaded(t, v)
	if v.state.index != 0 {
		t.Fatalf("index = %d, want 0 after hiding while on an extra", v.state.index)
	}
}

func TestHandleKeyEvent_ShiftDOpensGridOnCurrentGroup(t *testing.T) {
	v := loadPatternedTriple(t)
	stubKeyModifiers(t, v, fyne.KeyModifierShift)
	if v.grid.Visible() {
		t.Fatal("setup: grid closed")
	}

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyD})
	v.grid.Settle()

	if !v.grid.Visible() {
		t.Fatal("Shift+D on a duplicated file should open the grid")
	}
	if !v.grid.BrowsingDuplicates() {
		t.Fatal("grid should be in browse mode")
	}
	if v.grid.HideDuplicates() {
		t.Fatal("Shift+D must not turn hide on")
	}
}

func TestHandleKeyEvent_ShiftDNoopOnUniqueDoesNotOpenGrid(t *testing.T) {
	v := loadPatternedTriple(t)
	v.ShowImage(2)
	waitUntilLoaded(t, v)
	stubKeyModifiers(t, v, fyne.KeyModifierShift)

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyD})
	v.grid.Settle()

	if v.grid.Visible() {
		t.Fatal("Shift+D on a unique file must not open the grid")
	}
	if v.grid.BrowsingDuplicates() {
		t.Fatal("must not enter browse")
	}
}

func TestHandleKeyEvent_PlainDStillHidesWhenGridClosed(t *testing.T) {
	v := loadPatternedTriple(t)
	stubKeyModifiers(t, v, 0)
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyD})
	if !v.grid.HideDuplicates() || v.grid.Visible() {
		t.Fatal("plain D with grid closed should hide extras, not open the grid")
	}
}

func TestHandleKeyEvent_HomeEndSkipHiddenExtras(t *testing.T) {
	v := loadPatternedTriple(t)
	v.grid.SetHideDuplicates(true)
	waitUntilLoaded(t, v)

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEnd})
	waitUntilLoaded(t, v)
	if v.state.index != 2 {
		t.Fatalf("End index = %d, want 2 (last visible, not the extra)", v.state.index)
	}

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyHome})
	waitUntilLoaded(t, v)
	if v.state.index != 0 {
		t.Fatalf("Home index = %d, want 0", v.state.index)
	}
}

func TestHandleKeyEvent_LeftRightUseStepImage(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 8, 8, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 8, 8, color.White)
	dropAndWait(t, v, a, b)
	start := v.state.index
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
	waitUntilLoaded(t, v)
	if v.state.index != (start+1)%2 {
		t.Fatalf("Right via handleKeyEvent index = %d, want next", v.state.index)
	}
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyLeft})
	waitUntilLoaded(t, v)
	if v.state.index != start {
		t.Fatalf("Left via handleKeyEvent index = %d, want %d", v.state.index, start)
	}
}
