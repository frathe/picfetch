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
