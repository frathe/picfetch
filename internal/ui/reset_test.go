package ui

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/uitest"
)

// This file owns reset itself: viewer.go's Escape-triggered "start over",
// which with files loaded wipes the session back to exactly the state the
// viewer launched in - no files, no image, welcome art and the drop zone
// showing again, window back to its start size and title. reset is not
// clearToDropzone - it calls clearToDropzone and then goes further,
// clearing any in-flight vector rasterization, restoring the launch-time
// welcome look, and forcing a repaint - see both functions in viewer.go.
//
// What a reset does to one particular feature's own state lives with that
// feature instead of here: TestViewerReset_ReshowsRestoreLinkWhenSessionUnconsumed
// and TestViewerReset_DoesNotReshowRestoreLinkOnceConsumed are in
// session_test.go, TestClearToDropzone_HidesInfoCardButKeepsThePreference is
// in info_test.go, and TestClearToDropzone_PurgesTheImageCache is in
// imgcache_test.go. What stays here is the reset itself.

func TestViewerReset(t *testing.T) {
	v := newTestViewer(t)

	jpegURI := uitest.TempJPEGURI(t, "one.jpg", 10, 10, color.RGBA{R: 255, A: 255})
	dropAndWait(t, v, jpegURI)

	if v.img.Image == nil {
		t.Fatal("expected an image to be loaded before reset")
	}

	v.reset()

	if v.state.Observe().DisplayFiles() != nil {
		t.Errorf("files = %v, want nil after reset", v.state.Observe().DisplayFiles())
	}
	if v.state.Observe().index != 0 {
		t.Errorf("index = %d, want 0 after reset", v.state.Observe().index)
	}
	if v.img.Image != nil {
		t.Error("image should be cleared after reset")
	}
	if v.img.Visible() {
		t.Error("image should be hidden after reset")
	}
	if !v.dropzone.Visible() {
		t.Error("dropzone should be visible again after reset")
	}
	if !v.welcomeArt.Visible() {
		t.Error("welcomeArt should be visible again after reset, matching the just-launched state")
	}
	if v.emptyStateArt.Visible() {
		t.Error("emptyStateArt should be hidden after reset")
	}
	if got, want := v.hint.Text, lang.L("Drop images here"); got != want {
		t.Errorf("hint text = %q, want %q after reset", got, want)
	}
	if size := v.win.Canvas().Size(); !uitest.ApproxEqual(size.Width, startW) || !uitest.ApproxEqual(size.Height, startH) {
		t.Errorf("window size = %v, want %vx%v after reset", size, startW, startH)
	}
}

// nativeResetWindow models the window-manager boundary: a maximized native
// window rejects a requested size even if Fyne changes its logical canvas.
// This is the mismatch observed on Linux, which Canvas().Size() cannot detect.
type nativeResetWindow struct {
	fyne.Window
	nativeSize fyne.Size
	maximized  bool
}

func (w *nativeResetWindow) Resize(size fyne.Size) {
	w.Window.Resize(size)
	if !w.maximized {
		w.nativeSize = size
	}
}

func TestEscapeResetRestoresNativeWindow(t *testing.T) {
	for _, tc := range []struct {
		name       string
		staticSize bool
		gridVisit  bool
		closeFiles bool
	}{
		{name: "ordinary_maximize"},
		{name: "close_files", closeFiles: true},
		{name: "grid_maximize", gridVisit: true},
		{name: "fixed_ordinary_maximize", staticSize: true},
		{name: "fixed_grid_maximize", staticSize: true, gridVisit: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := newTestViewer(t)
			u := uitest.TempJPEGURI(t, "one.jpg", 800, 600, color.White)
			dropAndWait(t, v, u)
			v.SetStaticWindowSize(tc.staticSize)
			if tc.gridVisit {
				v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyG})
				v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyG})
			}

			restoredSize := v.win.Canvas().Size()
			maximizedSize := fyne.NewSize(1600, 1000)
			v.win.Resize(maximizedSize)
			native := &nativeResetWindow{Window: v.win, nativeSize: maximizedSize, maximized: true}
			v.win = native
			v.unmaximizeWindow = func(window fyne.Window) {
				if window != native {
					t.Fatal("reset restored a different native window")
				}
				if native.maximized {
					native.maximized = false
					native.nativeSize = restoredSize
					native.Window.Resize(restoredSize)
				}
			}

			if tc.closeFiles {
				v.closeFiles()
			} else {
				v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			}

			wantSize := fyne.NewSize(startW, startH)
			wantMaximized := tc.staticSize && !tc.gridVisit
			if wantMaximized {
				wantSize = maximizedSize
			} else if tc.staticSize {
				wantSize = restoredSize
			}
			if native.maximized != wantMaximized || native.nativeSize != wantSize {
				t.Errorf("native window after reset: maximized=%v size=%v; want maximized=%v size=%v (logical canvas=%v)",
					native.maximized, native.nativeSize, wantMaximized, wantSize, native.Canvas().Size())
			}
			if v.state.Observe().Count() != 0 || !v.dropzone.Visible() {
				t.Fatal("reset did not return to the empty welcome view")
			}
		})
	}
}
