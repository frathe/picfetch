package ui

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

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
