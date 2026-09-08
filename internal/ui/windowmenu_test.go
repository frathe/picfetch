// Window menu (windowmenu.go): grey-out of surfaces already showing, and
// the show/enter actions (Viewer leaves overlay modes; Grid and
// picture-frame never Toggle off). Do not open the manual here - rendering
// manual.md panics under Fyne's test theme.

package ui

import (
	"image/color"
	"testing"
	"time"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/uitest"
)

// Menu surface actions are idempotent; G/P are toggles. Grid keyboard ownership
// also swallows P, while the menu can explicitly switch to picture-frame mode.
func TestWindowCommandAdmissionMatrix(t *testing.T) {
	type state struct{ grid, frame, inspect, copy bool }
	for _, tc := range []struct {
		name, initial, command string
		key, action            state
	}{
		{"open grid", "viewer", "grid", state{grid: true}, state{grid: true}},
		{"grid toggle versus show", "grid", "grid", state{}, state{grid: true}},
		{"frame refuses grid", "frame", "grid", state{frame: true}, state{frame: true}},
		{"inspect reopens variants", "inspect", "grid", state{grid: true}, state{grid: true}},
		{"open frame", "viewer", "frame", state{frame: true}, state{frame: true}},
		{"grid owns P key", "grid", "frame", state{grid: true}, state{frame: true}},
		{"frame toggle versus show", "frame", "frame", state{}, state{frame: true}},
		{"inspect refuses frame", "inspect", "frame", state{inspect: true}, state{inspect: true}},
		{"viewer leaves grid", "grid", "viewer", state{}, state{}},
		{"viewer leaves frame", "frame", "viewer", state{}, state{}},
		{"viewer preserves inspect", "inspect", "viewer", state{inspect: true}, state{inspect: true}},
		{"grid yields copy selection", "copy", "grid", state{grid: true}, state{grid: true}},
	} {
		for _, route := range []string{"key", "menu", "direct command"} {
			t.Run(tc.name+"/"+route, func(t *testing.T) {
				var v *viewer
				if tc.initial == "inspect" {
					v = loadBrowsePair(t)
					v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
					v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
					waitUntilLoaded(t, v)
					if !v.dupes.Inspecting() {
						t.Fatal("premise: inspect did not start")
					}
				} else {
					v = newTestViewer(t)
					dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 40, 20, color.White))
					warmThumbs(t, v)
				}
				v.slides.SetInterval(time.Hour)
				t.Cleanup(func() { settleSlideshow(t, v) })
				switch tc.initial {
				case "grid":
					v.showWindowGrid()
				case "frame":
					v.showWindowPictureFrame()
				case "copy":
					v.startRegionCopy()
					if !v.regionCopy.State().Active {
						t.Fatal("premise: Copy Selection did not start")
					}
				}
				key, action, direct := fyne.KeyG, v.menus.Window().Grid().Action, v.showWindowGrid
				switch tc.command {
				case "frame":
					key, action, direct = fyne.KeyP, v.menus.Window().PictureFrame().Action, v.showWindowPictureFrame
				case "viewer":
					key, action, direct = fyne.KeyV, v.menus.Window().Viewer().Action, v.showViewer
				}
				want := tc.action
				switch route {
				case "key":
					want = tc.key
					v.handleKeyEvent(&fyne.KeyEvent{Name: key})
				case "menu":
					action()
				case "direct command":
					v.RunCommand(direct)
				}
				v.grid.Settle()
				got := state{v.grid.Visible(), v.slides.Active(), v.dupes.Inspecting(), v.regionCopy.State().Active}
				if got != want {
					t.Fatalf("grid/frame/inspect/copy = %+v, want %+v", got, want)
				}
				if tc.initial == "inspect" && tc.command == "grid" && !v.grid.BrowsingDuplicates() {
					t.Fatal("inspect reopened the ordinary grid instead of variants")
				}
			})
		}
	}
}

func windowMenu(v *viewer) *fyne.Menu {
	bar := v.win.MainMenu()
	if bar == nil || len(bar.Items) < 4 {
		return nil
	}
	return bar.Items[3] // File, Favorites, Actions, Window
}

func assertWindowMenuDisabled(t *testing.T, v *viewer, viewer, exif, grid, pf, help bool) {
	t.Helper()

	m := windowMenu(v)
	if m == nil || len(m.Items) < 5 {
		t.Fatal("Window menu missing from the main menu bar")
	}

	got := []bool{
		m.Items[0].Disabled,
		m.Items[1].Disabled,
		m.Items[2].Disabled,
		m.Items[3].Disabled,
		m.Items[4].Disabled,
	}
	want := []bool{viewer, exif, grid, pf, help}
	labels := []string{"Viewer", "EXIF Data", "Grid View", "Picture-frame mode", "Help"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s Disabled = %v, want %v", labels[i], got[i], want[i])
		}
	}
}

func TestWindowMenu_FreshViewerDisablesSurfacesExceptHelp(t *testing.T) {
	v := newTestViewer(t)

	assertWindowMenuDisabled(t, v, true, true, true, true, false)

	v.syncMenus()
	if v.menus.Window().Help().Disabled != v.help.ManualOpen() {
		t.Errorf("Window > Help Disabled = %v, want ManualOpen() (%v)", v.menus.Window().Help().Disabled, v.help.ManualOpen())
	}
}

func TestWindowMenu_AfterOneJPEGDropEnablesExifGridAndPictureFrame(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))

	assertWindowMenuDisabled(t, v, true, false, false, false, false)
}

func TestWindowMenu_ExifDisabledWhileOpenAndEnabledAfterClose(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))

	v.exif.Show()
	if !v.exif.Open() {
		t.Fatal("exif.Show should open the panel")
	}
	// Opens are not close hooks — apply after Show the way E / the info
	// link / showWindowExif do. Close below is what SetOnClosed covers.
	v.syncMenus()
	if !v.menus.Window().Exif().Disabled {
		t.Error("EXIF Data should be disabled while the panel is open")
	}

	win := v.exif.Window()
	if win == nil {
		t.Fatal("open EXIF panel has no window")
	}
	win.Close()
	if v.exif.Open() {
		t.Fatal("closing the EXIF window should leave it closed")
	}
	if v.menus.Window().Exif().Disabled {
		t.Error("EXIF Data should be enabled again after close")
	}
}

func TestWindowMenu_GridToggleGreysGridAndEnablesViewer(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	warmThumbs(t, v)

	v.grid.Toggle()
	if !v.grid.Visible() {
		t.Fatal("Toggle should open the grid")
	}
	assertWindowMenuDisabled(t, v, false, false, true, false, false)

	v.grid.Toggle()
	if v.grid.Visible() {
		t.Fatal("a second Toggle should close the grid")
	}
	assertWindowMenuDisabled(t, v, true, false, false, false, false)
}

func TestWindowMenu_PictureFrameToggleGreysPFEnablesViewerDisablesGrid(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	t.Cleanup(func() { settleSlideshow(t, v) })

	v.togglePictureFrameMode()
	if !v.slides.Active() {
		t.Fatal("togglePictureFrameMode should enter picture-frame mode")
	}
	assertWindowMenuDisabled(t, v, false, false, true, true, false)

	v.togglePictureFrameMode()
	if v.slides.Active() {
		t.Fatal("a second togglePictureFrameMode should leave picture-frame mode")
	}
	assertWindowMenuDisabled(t, v, true, false, false, false, false)
}

func TestWindowMenu_GKeyWhileGridClosedMatchesToggle(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	warmThumbs(t, v)

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyG})
	if !v.grid.Visible() {
		t.Fatal("G should open the grid")
	}
	assertWindowMenuDisabled(t, v, false, false, true, false, false)

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyG})
	if v.grid.Visible() {
		t.Fatal("a second G should close the grid")
	}
	assertWindowMenuDisabled(t, v, true, false, false, false, false)
}

func TestWindowMenu_PictureFrameLeavesGridItemDisabled(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	t.Cleanup(func() { settleSlideshow(t, v) })

	if v.menus.Window().Grid().Disabled {
		t.Fatal("Grid View should be enabled after a file is loaded")
	}

	v.togglePictureFrameMode()
	if !v.slides.Active() {
		t.Fatal("togglePictureFrameMode should enter picture-frame mode")
	}
	if !v.menus.Window().Grid().Disabled {
		t.Error("Grid View should stay disabled while picture-frame mode is on")
	}
}

func TestWindowMenu_CloseFilesDisablesGridPictureFrameExifAndViewer(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))

	assertWindowMenuDisabled(t, v, true, false, false, false, false)

	v.closeFiles()

	assertWindowMenuDisabled(t, v, true, true, true, true, false)
}

func TestWindowMenu_ViewerLeavesGrid(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	warmThumbs(t, v)
	v.grid.Toggle()
	if !v.grid.Visible() {
		t.Fatal("premises: grid up")
	}
	v.menus.Window().Viewer().Action()
	if v.grid.Visible() {
		t.Error("Viewer should close the grid")
	}
	if !v.menus.Window().Viewer().Disabled {
		t.Error("Viewer should grey once the image view is back")
	}
}

func TestWindowMenu_ViewerLeavesPictureFrame(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	t.Cleanup(func() { settleSlideshow(t, v) })
	v.togglePictureFrameMode()
	if !v.slides.Active() {
		t.Fatal("premises: picture-frame on")
	}
	v.menus.Window().Viewer().Action()
	if v.slides.Active() {
		t.Error("Viewer should exit picture-frame mode")
	}
	// The action carries no syncMenus of its own; slideshow's
	// SetOnActiveChanged observer is what must resync the matrix here.
	assertWindowMenuDisabled(t, v, true, false, false, false, false)
}

func TestWindowMenu_GridActionOpensAndDoesNotToggleOff(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	warmThumbs(t, v)
	v.menus.Window().Grid().Action()
	if !v.grid.Visible() {
		t.Fatal("Grid View should open the grid")
	}
	v.menus.Window().Grid().Action() // even if Disabled, Action is callable from tests
	if !v.grid.Visible() {
		t.Error("Grid View must not toggle the grid closed")
	}
}

func TestWindowMenu_PictureFrameActionEntersAndDoesNotToggleOff(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	t.Cleanup(func() { settleSlideshow(t, v) })
	v.menus.Window().PictureFrame().Action()
	if !v.slides.Active() {
		t.Fatal("should enter picture-frame")
	}
	v.menus.Window().PictureFrame().Action()
	if !v.slides.Active() {
		t.Error("Picture-frame mode must not toggle off from the menu")
	}
}

func TestWindowMenu_GridActionNoopsWithoutFiles(t *testing.T) {
	v := newTestViewer(t)
	v.menus.Window().Grid().Action()
	if v.grid.Visible() {
		t.Error("no files: grid must stay closed")
	}
}

func TestWindowMenu_ExifActionOpensWhenAFileIsDisplayed(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	v.menus.Window().Exif().Action()
	if !v.exif.Open() {
		t.Error("EXIF Data should open the panel")
	}
	if !v.menus.Window().Exif().Disabled {
		t.Error("EXIF item should grey while the panel is open")
	}
}

func TestWindowMenu_VKeyLeavesGrid(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	warmThumbs(t, v)
	v.grid.Toggle()
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyV})
	if v.grid.Visible() {
		t.Error("V should leave the grid, same as Window -> Viewer")
	}
}

func TestWindowMenu_VKeyLeavesPictureFrame(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	t.Cleanup(func() { settleSlideshow(t, v) })
	v.togglePictureFrameMode()
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyV})
	if v.slides.Active() {
		t.Error("V should leave picture-frame mode, same as Window -> Viewer")
	}
}

func TestWindowMenu_GridActionNoopsDuringPictureFrame(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	t.Cleanup(func() { settleSlideshow(t, v) })
	v.togglePictureFrameMode()
	v.menus.Window().Grid().Action()
	if v.grid.Visible() {
		t.Error("must not open the grid over picture-frame mode")
	}
}

func TestWindowMenu_PictureFrameDisabledDuringVariantsSession(t *testing.T) {
	v := loadBrowsePair(t)
	if !v.menus.Window().PictureFrame().Disabled {
		t.Fatal("Picture-frame should be disabled while browsing variants")
	}

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
	waitUntilLoaded(t, v)
	if !v.menus.Window().PictureFrame().Disabled {
		t.Fatal("Picture-frame should be disabled while inspecting")
	}
}
