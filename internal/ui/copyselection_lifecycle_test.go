package ui

import (
	"image"
	"image/color"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/filesort"
	"github.com/frathe/picfetch/internal/ui/copyselection"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestCopySelectionKeyboard(t *testing.T) {
	t.Run("Escape cancels without copying or resetting", func(t *testing.T) {
		v := newTestViewer(t)
		loadTwoCopySelectionImages(t, v)
		v.toggleInfoOverlay()
		uitest.StubClipboardCopy(t, func([]byte) error {
			t.Error("Escape copied image pixels")
			return nil
		})

		selectRegion(t, v, copySelectionBounds)
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})

		if got := v.regionCopy.State(); got != (copyselection.State{}) {
			t.Fatalf("State() after Escape = %+v, want inactive", got)
		}
		if v.FileCount() != 2 {
			t.Fatalf("files after Escape = %d, want the session left loaded", v.FileCount())
		}
		if !v.info.Object().Visible() {
			t.Fatal("Escape did not restore the information overlay")
		}
		if v.clipboard.Begun() {
			t.Fatal("Escape started a clipboard copy")
		}
	})

	t.Run("Return and Enter copy only with a committed rectangle", func(t *testing.T) {
		v := newTestViewer(t)
		loadTwoCopySelectionImages(t, v)
		copies := 0
		uitest.StubClipboardCopy(t, func([]byte) error {
			copies++
			return nil
		})

		v.startRegionCopy()
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEnter})
		if copies != 0 || !v.regionCopy.State().Active {
			t.Fatalf("copy without a rectangle: copies=%d active=%v", copies, v.regionCopy.State().Active)
		}

		selectRegion(t, v, copySelectionBounds)
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitForClipboard(t, v)
		if copies != 1 || v.regionCopy.State().Active {
			t.Fatalf("Return copy = {copies:%d active:%v}, want 1 and inactive", copies, v.regionCopy.State().Active)
		}

		selectRegion(t, v, copySelectionBounds)
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEnter})
		waitForClipboard(t, v)
		if copies != 2 || v.regionCopy.State().Active {
			t.Fatalf("Enter copy = {copies:%d active:%v}, want 2 and inactive", copies, v.regionCopy.State().Active)
		}
	})

	t.Run("navigation keys and typed runes are suppressed", func(t *testing.T) {
		v := newTestViewer(t)
		loadTwoCopySelectionImages(t, v)
		start := v.CurrentIndex()
		selectRegion(t, v, copySelectionBounds)

		for _, key := range []fyne.KeyName{
			fyne.KeyRight, fyne.KeyLeft, fyne.KeyUp, fyne.KeyDown, fyne.KeyHome, fyne.KeyEnd,
		} {
			v.handleKeyEvent(&fyne.KeyEvent{Name: key})
		}
		v.handleTypedRune('a')
		v.handleTypedRune('s')

		if v.CurrentIndex() != start {
			t.Fatalf("index after navigation keys = %d, want %d", v.CurrentIndex(), start)
		}
		if got := v.regionCopy.State(); !got.Active || !got.HasSelection {
			t.Fatalf("State() after suppressed keys = %+v, want active with selection", got)
		}

		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
		waitUntilLoaded(t, v)
		if v.CurrentIndex() == start {
			t.Fatal("Right after leaving Copy Selection did not navigate")
		}
	})
}

func TestCopySelectionRepeatedActivation(t *testing.T) {
	v := newTestViewer(t)
	loadTwoCopySelectionImages(t, v)
	selectRegion(t, v, copySelectionBounds)
	before := v.regionCopy.State()

	v.copyActionsSelection()
	if got := v.regionCopy.State(); got != before {
		t.Fatalf("repeated activation changed state from %+v to %+v", before, got)
	}
}

func TestCopySelectionUnknownKeyCancels(t *testing.T) {
	v := newTestViewer(t)
	loadTwoCopySelectionImages(t, v)
	selectRegion(t, v, copySelectionBounds)

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyC})

	if got := v.regionCopy.State(); got.Active {
		t.Fatalf("Copy Selection stayed active after an unowned key: %+v", got)
	}
}

func TestCopySelectionFocusLoss(t *testing.T) {
	v := newTestViewer(t)
	loadTwoCopySelectionImages(t, v)
	selectRegion(t, v, copySelectionBounds)
	before := v.regionCopy.State()

	v.Unfocus()
	v.win.RequestFocus()
	v.win.Resize(fyne.NewSize(640, 400))

	if got := v.regionCopy.State(); got != before {
		t.Fatalf("State() after focus change and resize = %+v, want %+v", got, before)
	}
}

func TestCopySelectionCancelsBeforeOtherCommands(t *testing.T) {
	for _, tc := range []struct {
		name    string
		keep    bool
		act     func(*testing.T, *viewer)
		assert  func(*testing.T, *viewer)
		cleanup func(*testing.T, *viewer)
	}{
		{
			name: "zoom in",
			keep: true,
			act:  func(_ *testing.T, v *viewer) { v.menus.Actions().ZoomIn().Action() },
		},
		{
			name: "zoom out",
			keep: true,
			act:  func(_ *testing.T, v *viewer) { v.menus.Actions().ZoomOut().Action() },
		},
		{
			name: "key zoom and wheel pan stay in mode",
			keep: true,
			act: func(_ *testing.T, v *viewer) {
				v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.Key1})
				v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyPlus})
				test.Scroll(v.win.Canvas(), fyne.NewPos(80, 80), 0, 20)
			},
		},
		{
			name: "rotate",
			act:  func(_ *testing.T, v *viewer) { v.menus.Actions().Rotate().Action() },
			assert: func(t *testing.T, v *viewer) {
				if v.display.Rotation() != 1 {
					t.Fatalf("rotation = %d, want 1", v.display.Rotation())
				}
			},
		},
		{
			name: "grid view",
			act: func(t *testing.T, v *viewer) {
				warmThumbs(t, v)
				v.menus.Window().Grid().Action()
			},
			assert: func(t *testing.T, v *viewer) {
				if !v.grid.Visible() {
					t.Fatal("Grid View did not open")
				}
			},
			cleanup: func(_ *testing.T, v *viewer) { v.grid.Close() },
		},
		{
			name: "picture frame",
			act:  func(_ *testing.T, v *viewer) { v.menus.Window().PictureFrame().Action() },
			assert: func(t *testing.T, v *viewer) {
				if !v.slides.Active() {
					t.Fatal("Picture Frame did not start")
				}
			},
			cleanup: settleSlideshow,
		},
		{
			name: "close files",
			act:  func(_ *testing.T, v *viewer) { v.menus.CloseFiles().Action() },
			assert: func(t *testing.T, v *viewer) {
				if v.FileCount() != 0 {
					t.Fatalf("files = %d, want 0", v.FileCount())
				}
			},
		},
		{
			name: "drop",
			act: func(t *testing.T, v *viewer) {
				dropAndWait(t, v, uitest.TempJPEGURI(t, "c.jpg", 8, 8, color.White))
			},
			assert: func(t *testing.T, v *viewer) {
				if v.FileCount() != 1 {
					t.Fatalf("files after drop = %d, want 1", v.FileCount())
				}
			},
		},
		{
			name: "copy image",
			act: func(t *testing.T, v *viewer) {
				v.menus.Actions().Copy().Action()
				waitForClipboard(t, v)
			},
		},
		{
			name: "copy image shortcut",
			act: func(t *testing.T, v *viewer) {
				handler := &fyne.ShortcutHandler{}
				wireGlobalShortcuts(handler, v)
				handler.TypedShortcut(&fyne.ShortcutCopy{})
				waitForClipboard(t, v)
			},
		},
		{
			name: "copy path",
			act: func(t *testing.T, v *viewer) {
				v.menus.Actions().CopyPath().Action()
				if v.app.Clipboard().Content() == "" {
					t.Fatal("Copy image path left the text clipboard empty")
				}
			},
		},
		{
			name: "trash",
			act:  func(_ *testing.T, v *viewer) { v.menus.Actions().Trash().Action() },
			assert: func(t *testing.T, v *viewer) {
				if !v.deletion.Visible() {
					t.Fatal("Move image to Trash did not show the confirmation")
				}
			},
			cleanup: func(_ *testing.T, v *viewer) { v.deletion.Cancel() },
		},
		{
			name: "export",
			act:  func(_ *testing.T, v *viewer) { v.menus.Export().Action() },
			assert: func(t *testing.T, v *viewer) {
				if !v.exportPrompt.Visible() {
					t.Fatal("Export image did not show the format prompt")
				}
			},
			cleanup: func(_ *testing.T, v *viewer) { v.exportPrompt.Hide() },
		},
		{
			name: "exif",
			act:  func(_ *testing.T, v *viewer) { v.menus.Window().Exif().Action() },
			assert: func(t *testing.T, v *viewer) {
				if !v.exif.Open() {
					t.Fatal("EXIF Data did not open")
				}
			},
			cleanup: func(_ *testing.T, v *viewer) {
				if win := v.exif.Window(); win != nil {
					win.Close()
				}
			},
		},
		{
			name: "settings",
			act:  func(_ *testing.T, v *viewer) { v.win.MainMenu().Items[0].Items[5].Action() },
			assert: func(t *testing.T, v *viewer) {
				if !v.settingsWin.Open() {
					t.Fatal("Settings did not open")
				}
			},
			cleanup: nil,
		},
		{
			name: "merge mode",
			act:  func(_ *testing.T, v *viewer) { v.menus.Actions().Merge().Action() },
			assert: func(t *testing.T, v *viewer) {
				if !v.MergeMode() {
					t.Fatal("Toggle merge mode did not turn merge on")
				}
			},
		},
		{
			name: "info overlay",
			act:  func(_ *testing.T, v *viewer) { v.menus.Actions().Info().Action() },
			assert: func(t *testing.T, v *viewer) {
				if !v.info.Visible() {
					t.Fatal("Show/Hide info overlay did not turn the standing preference on")
				}
			},
		},
		{
			name: "sort",
			act: func(_ *testing.T, v *viewer) {
				for i, mode := range filesort.Modes() {
					if mode == filesort.BySize {
						v.menus.Actions().Sort()[i].Action()
						return
					}
				}
			},
			assert: func(t *testing.T, v *viewer) {
				if v.SortMode() != filesort.BySize {
					t.Fatalf("sort mode = %v, want BySize", v.SortMode())
				}
			},
		},
		{
			name: "G key",
			act: func(t *testing.T, v *viewer) {
				warmThumbs(t, v)
				v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyG})
			},
			assert: func(t *testing.T, v *viewer) {
				if !v.grid.Visible() {
					t.Fatal("G did not open the grid")
				}
			},
			cleanup: func(_ *testing.T, v *viewer) { v.grid.Close() },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := newTestViewer(t)
			loadTwoCopySelectionImages(t, v)
			uitest.StubClipboardCopy(t, func([]byte) error { return nil })
			selectRegion(t, v, copySelectionBounds)
			if tc.cleanup != nil {
				t.Cleanup(func() { tc.cleanup(t, v) })
			}

			tc.act(t, v)

			if tc.keep {
				if got := v.regionCopy.State(); !got.Active || !got.HasSelection {
					t.Fatalf("State() after viewport command = %+v, want active with selection", got)
				}
				return
			}
			if got := v.regionCopy.State(); got.Active {
				t.Fatalf("Copy Selection stayed active after %s: %+v", tc.name, got)
			}
			if tc.assert != nil {
				tc.assert(t, v)
			}
		})
	}
}

func TestCopySelectionBusyBlocksOtherCommands(t *testing.T) {
	v, win, closed := newTestUI(t)
	loadTwoCopySelectionImages(t, v)
	release := holdBusyRegionCopy(t, v)
	startIndex := v.CurrentIndex()
	startRotation := v.display.Rotation()

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyG})
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyR})
	v.menus.Actions().Rotate().Action()
	v.menus.CloseFiles().Action()
	v.handleDrop([]fyne.URI{uitest.TempJPEGURI(t, "busy-drop.jpg", 4, 4, color.White)})
	v.copyActionsSelection()

	if got := v.regionCopy.State(); got != (copySelectionBusyState()) {
		t.Fatalf("state while other commands ran = %+v, want busy", got)
	}
	if v.FileCount() != 2 || v.CurrentIndex() != startIndex {
		t.Fatalf("session while busy = count:%d index:%d, want 2 files at %d",
			v.FileCount(), v.CurrentIndex(), startIndex)
	}
	if v.display.Rotation() != startRotation {
		t.Fatalf("rotation while busy = %d, want %d", v.display.Rotation(), startRotation)
	}
	if v.grid.Visible() {
		t.Fatal("G opened the grid while Copy Selection was busy")
	}

	win.Close()
	if !closed() {
		t.Fatal("window close was blocked while Copy Selection was busy")
	}

	release()
	waitForClipboard(t, v)
}

var copySelectionBounds = image.Rect(4, 3, 16, 12)

func loadTwoCopySelectionImages(t *testing.T, v *viewer) {
	t.Helper()
	dropAndWait(t, v,
		regionCopyPNGURI(t, "a.png", markedRegionCopyImage(20, 15)),
		regionCopyPNGURI(t, "b.png", markedRegionCopyImage(20, 15)),
	)
}

func holdBusyRegionCopy(t *testing.T, v *viewer) func() {
	t.Helper()
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	t.Cleanup(func() { once.Do(func() { close(release) }) })
	uitest.StubClipboardCopy(t, func([]byte) error {
		close(started)
		<-release
		return nil
	})

	selectRegion(t, v, copySelectionBounds)
	v.regionCopy.HandleKey(fyne.KeyReturn)
	select {
	case <-started:
	case <-time.After(testTimeout):
		t.Fatal("timed out waiting for Copy Selection clipboard dispatch")
	}
	if got := v.regionCopy.State(); got != (copySelectionBusyState()) {
		t.Fatalf("state after copy request = %+v, want busy", got)
	}
	return func() { once.Do(func() { close(release) }) }
}
