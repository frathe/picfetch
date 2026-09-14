package ui

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"slices"
	"sync/atomic"
	"testing"

	"fyne.io/fyne/v2/driver/desktop"

	"github.com/frathe/picfetch/internal/filesort"
	"github.com/frathe/picfetch/internal/similarity"
	explorerui "github.com/frathe/picfetch/internal/ui/explorer"
	searchui "github.com/frathe/picfetch/internal/ui/visualsearch"
	"github.com/frathe/picfetch/internal/uitest"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/storage"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func findSearchMenu(t *testing.T, v *viewer) *fyne.MenuItem {
	t.Helper()
	for _, menu := range v.win.MainMenu().Items {
		for _, item := range menu.Items {
			if item.Label == lang.L("Find more like this") {
				return item
			}
		}
	}
	t.Fatal("Find more like this is missing from Actions")
	return nil
}
func TestFindMoreLikeThisInitialAdmission(t *testing.T) {
	for _, changed := range []bool{false, true} {
		t.Run(fmt.Sprintf("setup reference changed=%v", changed), func(t *testing.T) {
			v := openGridWith(t, "a.jpg", "b.jpg")
			v.grid.Close()
			v.ShowImage(0)
			waitUntilLoaded(t, v)
			configureExplorer(v, func(options *explorerui.Options) {
				options.Settings.IntroSeen = false
				options.Supported, options.AssetsReady = true, true
			})
			v.visualsearch.Configure(searchui.Options{Queue: &uitest.UIQueue{}, Provider: func(ctx context.Context, _ similarity.SearchRequest, _ <-chan similarity.SearchQuery, _ func(similarity.SearchEvent)) error {
				<-ctx.Done()
				return ctx.Err()
			}})
			generation := v.Generation()
			v.findMoreLikeThis()
			if !v.explorer.State().SetupOpen {
				t.Fatal("first-use setup did not open")
			}
			if changed {
				v.ShowImage(1)
				waitUntilLoaded(t, v)
			}
			if v.Generation() != generation {
				t.Fatal("navigation unexpectedly replaced the collection")
			}
			fynetest.Tap(explorerDialogButton(t, v, "Continue"))
			if v.searchActive() == changed {
				t.Fatalf("setup admitted wrong reference: changed=%v active=%v", changed, v.searchActive())
			}
			if changed {
				v.findMoreLikeThis()
				if !v.searchActive() || v.visualsearch.State().Visit.ReferencePath != v.FileAt(1).Path() {
					t.Fatal("new explicit search did not admit the displayed reference")
				}
			}
		})
	}

	t.Run("ranked menus disable duplicate commands", func(t *testing.T) {
		v, publish := streamingSearch(t)
		publish(similarity.SearchPartial, 2, 1)
		if !v.menus.Actions().Hide().Disabled || !v.menus.Actions().ShowVariant().Disabled {
			t.Fatal("ranked search enabled duplicate commands")
		}
		v.visualsearch.Exit()
		if v.menus.Actions().Hide().Disabled {
			t.Fatal("exiting search left duplicate hiding disabled")
		}
	})

	v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
	item := findSearchMenu(t, v)
	if item.Disabled {
		t.Fatal("single highlighted reference is disabled")
	}
	v.grid.SelectAll()
	if !item.Disabled {
		t.Fatal("ambiguous multi-selection admitted")
	}
}

func streamingSearch(t *testing.T) (*viewer, func(similarity.SearchKind, ...int)) {
	t.Helper()
	v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg")
	configureExplorer(v, func(options *explorerui.Options) {
		options.Supported, options.AssetsReady, options.Settings.IntroSeen = true, true, true
	})
	queue := &uitest.UIQueue{}
	events := make(chan similarity.SearchEvent)
	delivered := make(chan struct{})
	v.visualsearch.Configure(searchui.Options{Queue: queue, Provider: func(ctx context.Context, request similarity.SearchRequest, queries <-chan similarity.SearchQuery, emit func(similarity.SearchEvent)) error {
		query := <-queries
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case event := <-events:
				event.SessionID, event.QueryID = request.SessionID, query.ID
				emit(event)
				delivered <- struct{}{}
			}
		}
	}})
	v.findMoreLikeThis()
	var revision uint64
	return v, func(kind similarity.SearchKind, indexes ...int) {
		t.Helper()
		revision++
		event := similarity.SearchEvent{Kind: kind, Revision: revision, Processed: 4, Total: 4}
		for _, i := range indexes {
			event.Matches = append(event.Matches, similarity.Match{Path: v.FileAt(i).Path()})
		}
		events <- event
		<-delivered
		queue.Drain()
	}
}

func TestFindMoreLikeThisProgressiveForegroundIdentity(t *testing.T) {
	v, publish := streamingSearch(t)
	publish(similarity.SearchPartial, 2, 1)
	v.grid.SimulateHover(1)
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
	waitUntilLoaded(t, v)
	if got, want := v.preloadCandidates(), []fyne.URI{v.FileAt(1), v.FileAt(0)}; !slices.EqualFunc(got, want, func(a, b fyne.URI) bool { return a.String() == b.String() }) {
		t.Fatalf("preloads left the captured search order: %v", got)
	}
	publish(similarity.SearchFinal, 3, 2)
	if v.grid.Visible() || v.state.index != 2 {
		t.Fatal("progress replaced opened image")
	}
	v.StepImage(1)
	waitUntilLoaded(t, v)
	if v.state.index != 1 {
		t.Fatalf("opened visit navigation changed to %d", v.state.index)
	}
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if !slices.Equal(v.grid.ResultIndexes(), []int{0, 3, 2}) {
		t.Fatalf("latest result was not restored: %v", v.grid.ResultIndexes())
	}
}

func TestFindMoreLikeThisActionsCaptureRankedSources(t *testing.T) {
	t.Run("favorite-opened-list-scans-once", func(t *testing.T) {
		v, publish := streamingSearch(t)
		publish(similarity.SearchFinal, 2, 1)
		v.grid.SimulateHover(1)
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitUntilLoaded(t, v)
		v.display.Settle()
		// Keep the real opened images at the end of a large captured collection.
		// No decoding or scan is needed to observe the naming-dialog admission.
		const prefix = 4096
		files := make([]fyne.URI, prefix, prefix+len(v.state.files))
		dir := t.TempDir()
		for i := range files {
			files[i] = storage.NewFileURI(filepath.Join(dir, fmt.Sprintf("%04d.jpg", i)))
		}
		v.state.files = append(files, v.state.files...)
		v.state.index += prefix
		var calls atomic.Int64
		for i, uri := range v.state.files {
			v.state.files[i] = searchCountingURI{URI: uri, calls: &calls}
		}
		v.favorites.AddCurrentList()
		if got := calls.Load(); got > int64(2*len(v.state.files)) {
			t.Fatalf("Favorite naming repeatedly scanned the original collection: %d path reads for %d files", got, len(v.state.files))
		}
	})
	for _, action := range []string{"copy", "trash", "compare", "favorite-list"} {
		t.Run(action, func(t *testing.T) {
			v, publish := streamingSearch(t)
			publish(similarity.SearchPartial, 2, 1)
			v.grid.SelectAll()
			want := []string{v.FileAt(0).Path(), v.FileAt(1).Path(), v.FileAt(2).Path()}
			var captured []string
			switch action {
			case "favorite-list":
				v.grid.HandleRune('/')
				v.grid.HandleRune('b')
				h := favoriteListHost{v}
				for _, uri := range h.CurrentFiles() {
					captured = append(captured, uri.Path())
				}
				want = []string{v.FileAt(1).Path()}
			case "copy":
				uitest.StubClipboardCopyFiles(t, func(paths []string) error { captured = slices.Clone(paths); return nil })
				v.copyGridSelection()
				publish(similarity.SearchFinal, 3, 1)
				waitForClipboard(t, v)
			case "trash":
				uitest.StubTrashMove(t, func(path string) error { captured = append(captured, path); return nil })
				v.deleteGridSelection()
				publish(similarity.SearchFinal, 3, 1)
				if !slices.Equal(v.grid.ResultIndexes(), []int{0, 2, 1}) {
					t.Fatal("deletion prompt's ordering changed")
				}
				v.deletion.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
				v.deletion.HandleKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
				v.deletion.Settle()
			case "compare":
				v.grid.ClearSelection()
				for _, id := range []int{1, 2} {
					v.grid.SimulateHover(id)
					v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
				}
				v.compareSelected()
				if err := v.compare.Settle(context.Background()); err != nil {
					t.Fatal(err)
				}
				publish(similarity.SearchFinal, 3, 1)
				if !slices.Equal(v.grid.ResultIndexes(), []int{0, 2, 1}) {
					t.Fatal("comparison's underlying Grid changed")
				}
				v.compare.Close()
				if !slices.Equal(v.grid.ResultIndexes(), []int{0, 3, 1}) {
					t.Fatal("comparison close did not apply latest result")
				}
				return
			}
			slices.Sort(captured)
			slices.Sort(want)
			if !slices.Equal(captured, want) {
				t.Fatalf("%s targets = %v, want %v", action, captured, want)
			}
		})
	}
}

type searchCountingURI struct {
	fyne.URI
	calls *atomic.Int64
}

func (u searchCountingURI) Path() string {
	u.calls.Add(1)
	return u.URI.Path()
}

func TestFindMoreLikeThisSourceAndSortRetirement(t *testing.T) {
	for _, change := range []string{"remove", "sort"} {
		t.Run(change, func(t *testing.T) {
			v, publish := streamingSearch(t)
			publish(similarity.SearchFinal, 2, 1)
			if change == "remove" {
				v.RemoveFile(2)
			} else {
				v.SetSortMode(filesort.BySize)
				waitForSort(t, v)
			}
			if v.searchActive() || !v.grid.Visible() {
				t.Fatal("source transition did not restore origin")
			}
			want := 4
			if change == "remove" {
				want = 3
			}
			if len(v.grid.ResultIndexes()) != want {
				t.Fatalf("reconciled origin = %v", v.grid.ResultIndexes())
			}
		})
	}
}

func TestFindMoreLikeThisInitialRoundTrip(t *testing.T) {
	for _, dismissal := range []string{"escape", "outside"} {
		t.Run("overlay-"+dismissal, func(t *testing.T) {
			v, publish := streamingSearch(t)
			publish(similarity.SearchPartial, 2, 1)
			previous := slices.Clone(v.grid.ResultIndexes())
			popup := widget.NewPopUpMenu(fyne.NewMenu("", fyne.NewMenuItem("Example", func() {})), v.win.Canvas())
			popup.ShowAtPosition(fyne.NewPos(10, 10))
			publish(similarity.SearchFinal, 3, 2)
			if !slices.Equal(v.grid.ResultIndexes(), previous) {
				t.Fatal("overlay lost its captured result")
			}
			wait := v.searchView.overlay
			if wait == nil {
				t.Fatal("pending final result has no overlay-return delivery")
			}
			if dismissal == "escape" {
				popup.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
			} else {
				fynetest.TapCanvas(v.win.Canvas(), fyne.NewPos(v.win.Canvas().Size().Width-1, v.win.Canvas().Size().Height-1))
			}
			for v.searchView.pending != nil {
				<-wait.notice
				v.searchView.overlayUI.Drain()
			}
			if !slices.Equal(v.grid.ResultIndexes(), []int{0, 3, 2}) {
				t.Fatalf("dismissed overlay kept stale ranking: %v", v.grid.ResultIndexes())
			}
		})
	}
	t.Run("cohort-origin", func(t *testing.T) {
		v := explorerFixture(t)
		paths := []string{v.FileAt(0).Path(), v.FileAt(1).Path()}
		v.OpenSimilarityCohort(paths)
		camera := v.explorer.Surface().View()
		v.visualsearch.Configure(searchui.Options{Queue: &uitest.UIQueue{}, Provider: func(ctx context.Context, request similarity.SearchRequest, queries <-chan similarity.SearchQuery, emit func(similarity.SearchEvent)) error {
			query := <-queries
			emit(similarity.SearchEvent{SessionID: request.SessionID, QueryID: query.ID, Revision: 1, Kind: similarity.SearchFinal, Matches: []similarity.Match{{Path: paths[1]}}})
			<-ctx.Done()
			return ctx.Err()
		}})
		v.findMoreLikeThis()
		v.visualsearch.Settle()
		v.visualsearch.Exit()
		if !slices.Equal(explorerGridPaths(v), paths) {
			t.Fatal("search exit lost its cohort origin")
		}
		v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
		if !v.explorerMapActive() || !reflect.DeepEqual(v.explorer.Surface().View(), camera) {
			t.Fatalf("cohort Back: map active=%v grid visible=%v, camera=%+v, want %+v", v.explorerMapActive(), v.grid.Visible(), v.explorer.Surface().View(), camera)
		}
	})
	for _, shortcut := range []bool{false, true} {
		t.Run(map[bool]string{false: "menu", true: "shortcut"}[shortcut], func(t *testing.T) {
			v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
			v.grid.SimulateHover(1)
			paths := []string{v.FileAt(0).Path(), v.FileAt(1).Path(), v.FileAt(2).Path()}
			configureExplorer(v, func(options *explorerui.Options) {
				options.AssetsReady = true
				options.Supported = true
				options.Settings.IntroSeen = true
			})
			var captured similarity.SearchRequest
			var reference string
			v.visualsearch.Configure(searchui.Options{Queue: &uitest.UIQueue{}, Provider: func(ctx context.Context, request similarity.SearchRequest, queries <-chan similarity.SearchQuery, emit func(similarity.SearchEvent)) error {
				captured = request
				query := <-queries
				reference = query.ReferencePath
				emit(similarity.SearchEvent{SessionID: request.SessionID, QueryID: query.ID, Revision: 1, Kind: similarity.SearchFinal, Processed: 3, Total: 3, Matches: []similarity.Match{{Path: paths[2], Score: 0.9}, {Path: paths[0], Score: 0.8}}})
				<-ctx.Done()
				return ctx.Err()
			}})
			if shortcut {
				handler := &fyne.ShortcutHandler{}
				wireGlobalShortcuts(handler, v)
				handler.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyL, Modifier: fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift})
			} else {
				findSearchMenu(t, v).Action()
			}
			v.visualsearch.Settle()
			if reference != paths[1] || !slices.Equal(captured.Paths, paths) {
				t.Fatalf("reference/scope %q %v", reference, captured.Paths)
			}
			if !v.grid.Visible() || !slices.Equal(v.grid.ResultIndexes(), []int{1, 2, 0}) {
				t.Fatalf("ranked Grid %v", v.grid.ResultIndexes())
			}
			v.grid.SimulateHover(1)
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
			waitUntilLoaded(t, v)
			if v.state.index != 2 || v.grid.Visible() {
				t.Fatal("opening did not use the ranked source")
			}
			v.StepImage(1)
			waitUntilLoaded(t, v)
			if v.state.index != 0 {
				t.Fatalf("stepping left ranked order: %d", v.state.index)
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			if !v.grid.Visible() || !v.searchActive() {
				t.Fatal("image Escape did not return to search Grid")
			}
			v.visualsearch.Back()
			if v.searchActive() || !v.grid.Visible() || !slices.Equal(v.grid.ResultIndexes(), []int{0, 1, 2}) || v.grid.Highlight() != 1 {
				t.Fatalf("origin not restored: %v highlight %d", v.grid.ResultIndexes(), v.grid.Highlight())
			}
		})
	}
}
