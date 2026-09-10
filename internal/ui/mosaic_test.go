package ui

import (
	"image"
	"image/color"
	"slices"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	fynetest "fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/displays"
	"github.com/frathe/picfetch/internal/ui/mosaicwin"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestMosaicMenu_FollowsGridResultImmediately(t *testing.T) {
	v, _, _ := newTestUI(t)
	item := v.menus.Window().Mosaic()
	if !item.Disabled {
		t.Fatal("mosaic menu enabled without loaded images")
	}
	dropAndWait(t, v, uitest.TempJPEGURI(t, "sun.jpg", 4, 4, color.White))
	if item.Disabled {
		t.Fatal("mosaic menu stayed disabled after loading images")
	}

	v.grid.Toggle()
	if item.Disabled {
		t.Fatal("mosaic menu stayed disabled for a non-empty Grid result")
	}
	v.grid.HandleRune('/')
	v.grid.HandleRune('z')
	if !item.Disabled {
		t.Fatal("mosaic menu stayed enabled for a no-match Grid result")
	}
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeyBackspace})
	if item.Disabled {
		t.Fatal("mosaic menu stayed disabled after the Grid result widened")
	}
	v.grid.Close()
	if item.Disabled {
		t.Fatal("mosaic menu stayed disabled after returning to the loaded collection")
	}
}

func TestMosaicWindow_OpensFromMenuWithDefensiveSnapshot(t *testing.T) {
	v, _, _ := newTestUI(t)
	dropAndWait(t, v,
		uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White),
		uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White),
	)
	v.grid.Toggle()
	inspections := 0
	uitest.StubDisplays(t, func(fyne.Window) (displays.Snapshot, error) {
		inspections++
		return displays.Snapshot{
			Displays: []displays.Display{{ID: "main", Name: "Main", Bounds: image.Rect(0, 0, 80, 50)}},
			Default:  "main",
		}, nil
	})

	v.menus.Window().Mosaic().Action()
	if !v.mosaicWin.Opened() {
		t.Fatal("mosaic menu did not open the secondary window")
	}
	first := v.mosaicWin.Snapshot()
	v.grid.HandleRune('/')
	v.grid.HandleRune('a')
	v.menus.Window().Mosaic().Action()
	if inspections != 1 {
		t.Fatalf("already-open mosaic reinspected displays %d times", inspections)
	}
	if got := uriNames(v.mosaicWin.Snapshot().Sources); !slices.Equal(got, uriNames(first.Sources)) {
		t.Fatalf("already-open mosaic retargeted sources from %v to %v", uriNames(first.Sources), got)
	}
	v.mosaicWin.Close()
}

func TestMosaicWindow_DirectInvocationGuardsEmptySources(t *testing.T) {
	v, _, _ := newTestUI(t)
	inspections := 0
	uitest.StubDisplays(t, func(fyne.Window) (displays.Snapshot, error) {
		inspections++
		return displays.Snapshot{}, nil
	})

	v.showMosaic()
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	v.grid.Toggle()
	v.grid.HandleRune('/')
	v.grid.HandleRune('z')
	v.showMosaic()
	if v.mosaicWin.Opened() || inspections != 0 {
		t.Fatalf("guarded invocation opened=%v inspections=%d", v.mosaicWin.Opened(), inspections)
	}
}

func TestMosaicKeyboard(t *testing.T) {
	v := openGridWith(t, "a.jpg", "b.jpg")
	uitest.StubDisplays(t, func(_ fyne.Window) (displays.Snapshot, error) {
		return displays.Snapshot{
			Displays: []displays.Display{{ID: "main", Name: "Main", Bounds: image.Rect(0, 0, 80, 50)}},
			Default:  "main",
		}, nil
	})
	stubKeyModifiers(t, v, fyne.KeyModifierShift)
	v.grid.HandleRune('/')
	v.win.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyM})
	if v.mosaicWin.Opened() {
		t.Fatal("Shift+M opened Mosaic while typing a Grid search")
	}
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	modal := dialog.NewInformation("Test", "Keyboard belongs to this dialog", v.win)
	modal.Show()
	v.win.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyM})
	if v.mosaicWin.Opened() {
		t.Fatal("Shift+M opened Mosaic behind a dialog")
	}
	modal.Hide()
	v.win.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyM})
	if !v.mosaicWin.Opened() {
		t.Fatal("Shift+M did not open the mosaic window from Grid View")
	}
	if got := uriNames(v.mosaicWin.Snapshot().Sources); !slices.Equal(got, []string{"a.jpg", "b.jpg"}) {
		t.Fatalf("keyboard mosaic sources = %v, want the complete Grid result", got)
	}
	v.mosaicWin.Close()
	v.grid.Close()
	beforeMerge := v.state.MergeMode()
	v.win.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyM})
	if !v.mosaicWin.Opened() {
		t.Fatal("Shift+M did not open Mosaic for the loaded collection in the viewer")
	}
	if got := uriNames(v.mosaicWin.Snapshot().Sources); !slices.Equal(got, []string{"a.jpg", "b.jpg"}) {
		t.Fatalf("viewer mosaic sources = %v, want the loaded collection", got)
	}
	if v.mosaicWin.Snapshot().Kind != mosaicwin.SourceKind("files") {
		t.Fatal("viewer mosaic incorrectly describes its sources as a Grid result")
	}
	if v.state.MergeMode() != beforeMerge {
		t.Fatal("Shift+M changed merge mode")
	}
	v.mosaicWin.Close()
	stubKeyModifiers(t, v, 0)
	v.win.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyM})
	if v.state.MergeMode() == beforeMerge {
		t.Fatal("plain M no longer toggles merge mode")
	}
	mapViewer := explorerFixture(t)
	topology := uiMosaicSnapshot(t).Displays
	uitest.StubDisplays(t, func(_ fyne.Window) (displays.Snapshot, error) { return topology, nil })
	if mapViewer.menus.Window().Mosaic().Disabled {
		t.Fatal("Mosaic is disabled on a loaded Explorer map")
	}
	mapViewer.menus.Window().Mosaic().Action()
	if !mapViewer.mosaicWin.Opened() || len(mapViewer.mosaicWin.Snapshot().Sources) != 18 || mapViewer.mosaicWin.Snapshot().Kind != mosaicwin.SourceKind("files") {
		t.Fatal("Explorer menu did not open Mosaic with the entire loaded collection")
	}
	mapViewer.mosaicWin.Close()
	stubKeyModifiers(t, mapViewer, fyne.KeyModifierShift)
	mapViewer.win.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyM})
	if !mapViewer.mosaicWin.Opened() || !mapViewer.explorerMapActive() {
		t.Fatal("Shift+M did not open Mosaic while preserving the Explorer map")
	}
}

func TestMosaicSources_UsesExplicitSelectionExclusively(t *testing.T) {
	v, _, _ := newTestUI(t)
	dropAndWait(t, v,
		uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White),
		uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White),
		uitest.TempJPEGURI(t, "c.jpg", 4, 4, color.White),
	)
	v.grid.Toggle()
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	v.grid.HandleRune('/')
	v.grid.HandleRune('a') // the selected b.jpg is now hidden by search

	got, err := v.mosaicSources()
	if err != nil {
		t.Fatalf("mosaicSources() = %v", err)
	}
	if len(got) != 1 || got[0].Name() != "b.jpg" {
		t.Fatalf("mosaicSources() = %v, want selected b.jpg only", uriNames(got))
	}
}

func TestMosaicSources_UsesCompleteFilteredResultWithoutSelection(t *testing.T) {
	v, _, _ := newTestUI(t)
	dropAndWait(t, v,
		uitest.TempJPEGURI(t, "sun-a.jpg", 4, 4, color.White),
		uitest.TempJPEGURI(t, "moon.jpg", 4, 4, color.White),
		uitest.TempJPEGURI(t, "sun-b.jpg", 4, 4, color.White),
	)
	v.grid.Toggle()
	v.grid.HandleRune('/')
	for _, r := range "sun" {
		v.grid.HandleRune(r)
	}

	got, err := v.mosaicSources()
	if err != nil {
		t.Fatalf("mosaicSources() = %v", err)
	}
	if want := []string{"sun-a.jpg", "sun-b.jpg"}; !slices.Equal(uriNames(got), want) {
		t.Fatalf("mosaicSources() = %v, want %v", uriNames(got), want)
	}
}

func TestMosaicSources_HiddenDuplicatesUseHighestResolution(t *testing.T) {
	setup := func(t *testing.T, selectRepresentative bool) (*viewer, fyne.URI, fyne.URI) {
		t.Helper()
		v, _, _ := newTestUI(t)
		small := uitest.PatternedJPEGURISize(t, "a-small.jpg", 1, 64, 48)
		large := uitest.PatternedJPEGURISize(t, "b-large.jpg", 1, 192, 144)
		dropAndWait(t, v, small, large)
		if err := v.grid.Warm(); err != nil {
			t.Fatalf("Warm: %v", err)
		}
		v.grid.Toggle()
		v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
		if selectRepresentative {
			v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
			v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
		}
		v.grid.SetHideDuplicates(true)
		v.grid.Settle()
		waitUntilLoaded(t, v)
		if !v.dupes.IsHiddenExtra(0) || v.dupes.RepresentativeOf(0) != 1 {
			t.Fatalf("setup did not hide the smaller duplicate: hidden=%v representative=%d", v.dupes.IsHiddenExtra(0), v.dupes.RepresentativeOf(0))
		}
		return v, small, large
	}

	t.Run("selected hidden member is substituted", func(t *testing.T) {
		v, _, large := setup(t, false)
		uitest.StubDisplays(t, func(fyne.Window) (displays.Snapshot, error) {
			return displays.Snapshot{
				Displays: []displays.Display{{ID: "main", Name: "Main", Bounds: image.Rect(0, 0, 80, 50)}},
				Default:  "main",
			}, nil
		})

		v.menus.Window().Mosaic().Action()
		if got, want := uriNames(v.mosaicWin.Snapshot().Sources), []string{large.Name()}; !slices.Equal(got, want) {
			t.Fatalf("mosaic sources = %v, want highest-resolution representative %v", got, want)
		}
		v.mosaicWin.Close()
	})

	t.Run("selected group is collapsed", func(t *testing.T) {
		v, _, large := setup(t, true)
		got, err := v.mosaicSources()
		if err != nil {
			t.Fatal(err)
		}
		if want := []string{large.Name()}; !slices.Equal(uriNames(got), want) {
			t.Fatalf("mosaic sources = %v, want one representative %v", uriNames(got), want)
		}
	})

	t.Run("browsed variants keep the explicit selection", func(t *testing.T) {
		v, small, _ := setup(t, false)
		v.grid.SetBrowsingDuplicates(true)
		v.grid.Settle()
		if !v.grid.BrowsingDuplicates() {
			t.Fatal("setup did not enter duplicate browsing")
		}
		got, err := v.mosaicSources()
		if err != nil {
			t.Fatal(err)
		}
		if want := []string{small.Name()}; !slices.Equal(uriNames(got), want) {
			t.Fatalf("mosaic sources while browsing = %v, want explicit selection %v", uriNames(got), want)
		}
	})
}

func TestMosaicSnapshot_DoesNotRetargetAfterGridMutation(t *testing.T) {
	v, _, _ := newTestUI(t)
	dropAndWait(t, v,
		uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White),
		uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White),
	)
	v.grid.Toggle()

	snapshot, err := v.mosaicSources()
	if err != nil {
		t.Fatalf("mosaicSources() = %v", err)
	}
	v.RemoveFiles([]int{0})
	v.grid.Close()

	if want := []string{"a.jpg", "b.jpg"}; !slices.Equal(uriNames(snapshot), want) {
		t.Fatalf("snapshot = %v after mutation, want %v", uriNames(snapshot), want)
	}
}

func TestMosaicDrain_ClosesWindowAndClearsTransientState(t *testing.T) {
	v, _, _ := newTestUI(t)
	v.mosaicWin.Show(uiMosaicSnapshot(t))
	drain(t, v)

	if v.mosaicWin.Opened() || len(v.mosaicWin.Snapshot().Sources) != 0 {
		t.Fatal("test drain retained the mosaic window or its source snapshot")
	}
}

func TestMosaicShutdown_ClosesWindowBeforeTheEventLoopStops(t *testing.T) {
	application := fynetest.NewApp()
	v, win := buildStartupViewer(application)
	v.grid.SetUIQueue(&uitest.UIQueue{})
	v.compare.SetUIQueue(&uitest.UIQueue{})
	v.mosaicWin.SetUIQueue(&uitest.UIQueue{})
	t.Cleanup(win.Close)
	t.Cleanup(func() { drain(t, v) })
	v.mosaicWin.Show(uiMosaicSnapshot(t))

	lifecycle, ok := application.Lifecycle().(interface{ OnStopped() func() })
	if !ok {
		t.Skip("test app lifecycle does not expose its stopped hook")
	}
	original := lifecycle.OnStopped()
	registerShutdown(application, v)
	shutdown := lifecycle.OnStopped()
	application.Lifecycle().SetOnStopped(original)
	if shutdown == nil {
		t.Fatal("registerShutdown did not install a stopped hook")
	}

	shutdown()
	if v.mosaicWin.Opened() || len(v.mosaicWin.Snapshot().Sources) != 0 {
		t.Fatal("shutdown retained the mosaic window or its source snapshot")
	}
}

func uiMosaicSnapshot(t *testing.T) mosaicwin.Snapshot {
	t.Helper()
	snapshot, err := mosaicwin.NewSnapshot(
		[]fyne.URI{uitest.TempJPEGURI(t, "mosaic.jpg", 8, 6, color.White)},
		mosaicwin.SourceResult,
		displays.Snapshot{
			Displays: []displays.Display{{ID: "main", Name: "Main", Bounds: image.Rect(0, 0, 80, 50)}},
			Default:  "main",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func uriNames(uris []fyne.URI) []string {
	names := make([]string, len(uris))
	for i, uri := range uris {
		names[i] = uri.Name()
	}

	return names
}
