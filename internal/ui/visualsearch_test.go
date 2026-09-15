package ui

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"fyne.io/fyne/v2/driver/desktop"

	"github.com/frathe/picfetch/internal/filesort"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/similarity"
	"github.com/frathe/picfetch/internal/ui/analysiscache"
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
	t.Run("cache-pressure-continues", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
		v.analysisDir = t.TempDir()
		v.settings.looseAnalysisCache, v.settings.analysisCacheMiB = true, 1
		v.analysisCache.Configure(analysiscache.Options{Roots: v.analysisRoots(), Queue: &uitest.UIQueue{}})
		v.analysisCache.SetPolicy(true, 1)
		records := filepath.Join(v.analysisDir, "v1")
		if err := os.Mkdir(records, 0700); err != nil {
			t.Fatal(err)
		}
		record := filepath.Join(records, strings.Repeat("a", 64)+".json")
		if err := os.WriteFile(record, make([]byte, 1024*1024), 0600); err != nil {
			t.Fatal(err)
		}
		configureExplorer(v, func(options *explorerui.Options) {
			options.Supported, options.AssetsReady, options.Settings.IntroSeen = true, true, true
		})
		var calls atomic.Int32
		firstStopped := make(chan struct{})
		preparing := make(chan struct{})
		finish := make(chan struct{})
		ready := make(chan struct{}, 1)
		queue := &uitest.UIQueue{}
		paths := []string{v.FileAt(0).Path(), v.FileAt(1).Path(), v.FileAt(2).Path()}
		v.visualsearch.Configure(searchui.Options{Queue: queue, Provider: func(ctx context.Context, request similarity.SearchRequest, queries <-chan similarity.SearchQuery, emit func(similarity.SearchEvent)) error {
			first := calls.Add(1) == 1
			if first {
				defer close(firstStopped)
			}
			var revision uint64
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case query := <-queries:
					if revision == 0 && first {
						revision++
						emit(similarity.SearchEvent{SessionID: request.SessionID, QueryID: query.ID, Revision: revision, Kind: similarity.SearchPartial, Processed: 2, Total: 3, CachePressureBytes: 1, Matches: []similarity.Match{{Path: paths[1]}}})
						close(preparing)
						select {
						case <-ctx.Done():
							return ctx.Err()
						case <-finish:
						}
					}
					revision++
					emit(similarity.SearchEvent{SessionID: request.SessionID, QueryID: query.ID, Revision: revision, Kind: similarity.SearchFinal, Processed: 3, Total: 3, Matches: []similarity.Match{{Path: paths[1]}, {Path: paths[2]}}})
					revision++
					emit(similarity.SearchEvent{SessionID: request.SessionID, QueryID: query.ID, Revision: revision, Kind: similarity.SearchReady, Processed: 3, Total: 3, CachePressureBytes: 1})
					ready <- struct{}{}
				}
			}
		}})
		v.findMoreLikeThis()
		<-preparing
		queue.Drain()
		if !v.visualsearch.State().Preparing || v.analysisMaintenanceBusy() {
			t.Fatal("capacity pressure interrupted preparation or started early eviction")
		}
		if _, err := os.Stat(record); err != nil {
			t.Fatalf("cache record was removed before readiness: %v", err)
		}
		close(finish)
		<-ready
		v.visualsearch.Settle()
		v.analysisCache.Settle()
		if _, err := os.Stat(record); !errors.Is(err, os.ErrNotExist) || v.analysisMaintenanceBusy() {
			t.Fatalf("automatic eviction did not finish: %v", err)
		}
		select {
		case <-firstStopped:
			t.Fatal("automatic eviction canceled the fully prepared producer")
		default:
		}
		firstSession := v.visualsearch.State().SessionID
		v.grid.SimulateHover(1)
		v.findMoreLikeThis()
		<-ready
		v.visualsearch.Settle()
		if calls.Load() != 1 || v.visualsearch.State().SessionID != firstSession || v.visualsearch.State().Preparing || v.analysisMaintenanceBusy() {
			t.Fatal("next reference repeated preparation or handled cache pressure")
		}
		// Explicit cleanup still retires the completed producer.
		v.analysisCache.Content(true, 1)
		v.analysisCache.Settle()
		v.analysisCache.Clean(similarity.ClearAll)
		v.analysisCache.Settle()
		select {
		case <-firstStopped:
		default:
			t.Fatal("explicit cleanup retained an admitted producer")
		}
	})
	t.Run("limit-increase-before-inspection", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
		v.settings.looseAnalysisCache = true
		publish := streamingSearchFrom(t, v)
		publish(similarity.SearchPartial, 2, 1)
		before := v.visualsearch.State().Visit
		oldLimit := v.settings.analysisCacheMiB
		provider := &searchLimitInspection{started: make(chan similarity.CacheRetuneRequest, 1), release: make(chan struct{})}
		t.Cleanup(func() {
			select {
			case <-provider.release:
			default:
				close(provider.release)
			}
		})
		v.analysisCache.Configure(analysiscache.Options{Provider: provider, Queue: &uitest.UIQueue{}})
		v.analysisCache.Content(true, oldLimit)
		v.analysisCache.Settle()
		if !v.visualsearch.State().Preparing {
			t.Fatal("read-only usage inspection retired search")
		}
		v.analysisCache.Retune(oldLimit * 2)
		if v.visualsearch.State().Preparing {
			t.Fatal("increased limit began inspection with the old search producer active")
		}
		request := <-provider.started
		if request.LimitBytes != uint64(oldLimit*2)*1024*1024 || request.RetireWriters || v.settings.analysisCacheMiB != oldLimit {
			t.Fatal("increase changed lease policy or committed before inspection completed")
		}
		v.visualsearch.Settle()
		if !v.searchActive() || !slices.Equal(v.visualsearch.State().Visit.Paths, before.Paths) || !v.analysisMaintenanceBusy() {
			t.Fatal("retiring the captured limit lost browsing or maintenance ownership")
		}
		close(provider.release)
		v.analysisCache.Settle()
		if v.settings.analysisCacheMiB != oldLimit*2 || v.visualsearch.State().Preparing || provider.calls.Load() != 1 {
			t.Fatal("limit acceptance restarted search or queued additional maintenance")
		}
	})
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
		hidden := v.dupes.HideDuplicates()
		v.toggleActionsHideDuplicates()
		if v.dupes.HideDuplicates() != hidden {
			t.Fatal("direct duplicate command bypassed ranked menu admission")
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
	return v, streamingSearchFrom(t, v)
}

func streamingSearchFrom(t *testing.T, v *viewer) func(similarity.SearchKind, ...int) {
	t.Helper()
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
			case next := <-queries:
				query = next
			case event := <-events:
				// An explicit Explore can enqueue its query immediately before
				// this event. Associate publication with the latest admitted query.
				select {
				case next := <-queries:
					query = next
				default:
				}
				event.SessionID, event.QueryID = request.SessionID, query.ID
				emit(event)
				delivered <- struct{}{}
			}
		}
	}})
	v.findMoreLikeThis()
	var revision uint64
	return func(kind similarity.SearchKind, indexes ...int) {
		t.Helper()
		revision++
		event := similarity.SearchEvent{Kind: kind, Revision: revision, Processed: v.FileCount(), Total: v.FileCount()}
		if kind == similarity.SearchPartial {
			event.Processed /= 2
		}
		for _, i := range indexes {
			event.Matches = append(event.Matches, similarity.Match{Path: v.FileAt(i).Path()})
		}
		events <- event
		<-delivered
		queue.Drain()
	}
}

func TestFindMoreLikeThisProgressiveForegroundIdentity(t *testing.T) {
	t.Run("open-reference-before-first-result", func(t *testing.T) {
		v, publish := streamingSearch(t)
		v.grid.HandleRune('/')
		v.grid.HandleRune('a')
		v.grid.SimulateHover(0)
		v.grid.SelectAll()
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitUntilLoaded(t, v)
		publish(similarity.SearchFinal, 2, 1)
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		if v.grid.Query() != "a" || !slices.Equal(v.grid.Selection(), []int{0}) || !slices.Equal(v.grid.ResultIndexes(), []int{0}) {
			t.Fatalf("initial Grid state was lost: query=%q selected=%v results=%v", v.grid.Query(), v.grid.Selection(), v.grid.ResultIndexes())
		}
	})
	v, publish := streamingSearch(t)
	publish(similarity.SearchPartial, 2, 1)
	v.grid.SimulateHover(1)
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
	waitUntilLoaded(t, v)
	hidden := v.dupes.HideDuplicates()
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyD})
	if v.dupes.HideDuplicates() != hidden {
		t.Fatal("opened ranked image admitted duplicate hiding through the keyboard")
	}
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

type searchLimitInspection struct {
	similarity.CacheManager
	started chan similarity.CacheRetuneRequest
	release chan struct{}
	calls   atomic.Int32
}

func (*searchLimitInspection) Inspect(_ context.Context, _ similarity.CacheRoots, _ func(similarity.CacheProgress)) (similarity.CacheUsage, error) {
	return similarity.CacheUsage{}, nil
}

func (p *searchLimitInspection) Retune(ctx context.Context, request similarity.CacheRetuneRequest, _ func(similarity.CacheProgress)) (similarity.CacheReport, error) {
	p.calls.Add(1)
	p.started <- request
	select {
	case <-p.release:
		return similarity.CacheReport{AppliedLimit: request.LimitBytes}, nil
	case <-ctx.Done():
		return similarity.CacheReport{}, ctx.Err()
	}
}

type searchCacheInspection struct {
	similarity.CacheManager
	started chan struct{}
}

func (p searchCacheInspection) Inspect(ctx context.Context, _ similarity.CacheRoots, _ func(similarity.CacheProgress)) (similarity.CacheUsage, error) {
	close(p.started)
	<-ctx.Done()
	return similarity.CacheUsage{}, ctx.Err()
}

func TestFindMoreLikeThisActionsCaptureRankedSources(t *testing.T) {
	t.Run("favorite-save-during-inspection", func(t *testing.T) {
		for _, retired := range []bool{false, true} {
			t.Run(fmt.Sprintf("retired=%v", retired), func(t *testing.T) {
				v := openGridWith(t, "a.jpg", "b.jpg")
				v.settings.looseAnalysisCache = false
				v.favorites.SetDir(t.TempDir())
				configureExplorer(v, func(options *explorerui.Options) {
					options.Supported, options.AssetsReady, options.Settings.IntroSeen = true, true, true
					options.Settings.CacheFavorites = true
				})
				queue := &uitest.UIQueue{}
				started := make(chan (<-chan similarity.SearchQuery), 1)
				v.visualsearch.Configure(searchui.Options{Queue: queue, Provider: func(ctx context.Context, request similarity.SearchRequest, queries <-chan similarity.SearchQuery, emit func(similarity.SearchEvent)) error {
					query := <-queries
					emit(similarity.SearchEvent{SessionID: request.SessionID, QueryID: query.ID, Revision: 1, Kind: similarity.SearchPartial,
						Processed: 1, Total: 2, Matches: []similarity.Match{{Path: request.Paths[1]}}})
					started <- queries
					<-ctx.Done()
					return ctx.Err()
				}})
				v.findMoreLikeThis()
				queries := <-started
				queue.Drain()
				inspection := searchCacheInspection{started: make(chan struct{})}
				v.analysisCache.Configure(analysiscache.Options{Roots: v.analysisRoots(), Provider: inspection, Queue: &uitest.UIQueue{}})
				v.analysisCache.Content(false, 2048)
				<-inspection.started
				if !v.analysisCache.Busy() || !v.visualsearch.State().Preparing {
					t.Fatal("inspection did not overlap active preparation")
				}
				if retired {
					<-v.visualsearch.Suspend()
				}
				v.favorites.AddCurrentList()
				entry, ok := v.win.Canvas().Focused().(interface {
					SetText(string)
					TypedKey(*fyne.KeyEvent)
				})
				if !ok {
					t.Fatal("Favorite naming entry unavailable")
				}
				entry.SetText("Saved during inspection")
				entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
				select {
				case query := <-queries:
					if retired || query.CacheRevision != 1 {
						t.Fatalf("unexpected save notification: retired=%v query=%+v", retired, query)
					}
				default:
					if !retired {
						t.Fatal("read-only inspection dropped the committed Favorite save")
					}
				}
				if len(started) != 0 || retired && v.visualsearch.State().Preparing {
					t.Fatal("Favorite save restarted a retired producer")
				}
			})
		}
	})
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
		_ = v.preloadCandidates()
		if got := calls.Load(); got > int64(len(v.state.files)+16) {
			t.Fatalf("preloads captured the ranked order more than once: %d path reads for %d files", got, len(v.state.files))
		}
		calls.Store(0)
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
	for _, gridOrigin := range []bool{false, true} {
		t.Run(fmt.Sprintf("comparison-committed-write-grid=%v", gridOrigin), func(t *testing.T) {
			v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
			v.ShowImage(0)
			waitUntilLoaded(t, v)
			original := v.FileAt(0)
			if !gridOrigin {
				v.grid.Close()
			}
			publish := streamingSearchFrom(t, v)
			publish(similarity.SearchFinal, 2, 1)
			v.grid.ClearSelection()
			for _, i := range []int{0, 1} {
				v.grid.SimulateHover(i)
				v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
			}
			v.compareSelected()
			if err := v.compare.Settle(context.Background()); err != nil {
				t.Fatal(err)
			}
			result, err := imaging.SaveRotatedContext(context.Background(), original, image.NewRGBA(image.Rect(0, 0, 19, 13)))
			if err != nil || !result.Committed {
				t.Fatalf("source write failed: %v", err)
			}
			// Save/export workers invalidate decoded content at disk commit,
			// before their tracked UI reconciliation is admitted.
			v.imgCache.Purge()
			v.afterFileWrite(result, true, true, func() {})
			drainFileWork(t, v)
			if err := v.compare.Settle(context.Background()); err != nil {
				t.Fatal(err)
			}
			if v.searchActive() || v.grid.Visible() != gridOrigin || v.comparisonActive() != gridOrigin {
				t.Fatal("committed write did not reconcile comparison with the restored origin")
			}
			if !gridOrigin {
				waitUntilLoaded(t, v)
				if got, ok := v.DisplayedFile(); !ok || got.String() != original.String() || v.img.Image.Bounds().Size() != image.Pt(19, 13) {
					t.Fatal("committed write restored stale image pixels")
				}
			}
		})
	}
	for _, gridOrigin := range []bool{false, true} {
		t.Run(fmt.Sprintf("load-failure-owner-grid=%v", gridOrigin), func(t *testing.T) {
			v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg")
			v.ShowImage(0)
			waitUntilLoaded(t, v)
			if !gridOrigin {
				v.grid.Close()
			}
			origin, failed, next := v.FileAt(0), v.FileAt(2), v.FileAt(3)
			publish := streamingSearchFrom(t, v)
			publish(similarity.SearchFinal, 2, 1)
			v.display.WaitPreloads()
			if err := os.WriteFile(failed.Path(), []byte("unreadable image"), 0600); err != nil {
				t.Fatal(err)
			}
			v.imgCache.Purge()
			revision := v.display.RequestRevision()
			v.grid.SimulateHover(1)
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
			waitUntilLoaded(t, v)
			want := origin
			if gridOrigin {
				want = next
			}
			current, _, ok := v.CurrentFile()
			displayed, shown := v.DisplayedFile()
			if !ok || !shown || current.String() != want.String() || displayed.String() != want.String() {
				t.Fatalf("load recovery disagrees with browsing: current=%v displayed=%v want=%v", current, displayed, want)
			}
			if v.display.RequestRevision() != revision+1 {
				t.Fatal("source restoration started a competing load instead of using display's retry")
			}
			if v.searchActive() || v.grid.Visible() != gridOrigin || v.FileCount() != 3 {
				t.Fatal("load recovery lost the source change or origin surface")
			}
		})
	}
	for _, operation := range []string{"ordinary", "committed-trash", "source-failure"} {
		t.Run("grid-origin-batch-"+operation, func(t *testing.T) {
			names := make([]string, 40)
			for i := range names {
				names[i] = fmt.Sprintf("%02d.jpg", i)
			}
			v := openGridWith(t, names...)
			v.grid.SimulateHover(3)
			v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
			v.grid.HandleRune('/')
			v.grid.HandleRune('.')
			for range 2 {
				v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeyPageDown})
			}
			origin := v.grid.CaptureVisit()
			if origin.ScrollOffset <= 0 || len(origin.Selected) != 1 {
				t.Fatal("Grid origin lacks a scrolled selection")
			}
			publish := streamingSearchFrom(t, v)
			publish(similarity.SearchFinal, 0, 1)
			removed := []fyne.URI{v.FileAt(0), v.FileAt(2)}
			for _, source := range removed {
				if err := os.Remove(source.Path()); err != nil {
					t.Fatal(err)
				}
			}
			switch operation {
			case "ordinary":
				v.RemoveFiles([]int{0, 2})
			case "committed-trash":
				v.ReconcileDeletedFiles(removed)
			case "source-failure":
				searchHost{v}.Failed(searchui.SessionError{Err: errors.New("search source changed")})
				drainFileWork(t, v)
			}
			got := v.grid.CaptureVisit()
			if !got.Visible || !slices.Equal(got.Selected, origin.Selected) || got.Highlight != origin.Highlight || got.Query != origin.Query || got.ScrollOffset != origin.ScrollOffset {
				t.Fatalf("reconciliation overwrote restored Grid: got=%+v origin=%+v", got, origin)
			}
		})
	}
	for _, gridOrigin := range []bool{false, true} {
		for _, replace := range []bool{false, true} {
			t.Run(fmt.Sprintf("comparison-source-grid=%v-replaced=%v", gridOrigin, replace), func(t *testing.T) {
				v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg")
				v.ShowImage(0)
				waitUntilLoaded(t, v)
				if !gridOrigin {
					v.grid.Close()
				}
				publish := streamingSearchFrom(t, v)
				publish(similarity.SearchFinal, 2, 1)
				original, survivor := v.FileAt(0), v.FileAt(1)
				v.grid.ClearSelection()
				for _, i := range []int{0, 1} {
					v.grid.SimulateHover(i)
					v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
				}
				v.compareSelected()
				if err := v.compare.Settle(context.Background()); err != nil {
					t.Fatal(err)
				}
				if !v.comparisonActive() {
					t.Fatal("comparison was not opened")
				}
				if replace {
					data := uitest.EncodeJPEG(t, 16, 16, color.Black)
					if err := os.WriteFile(original.Path(), data, 0600); err != nil {
						t.Fatal(err)
					}
				} else if err := os.Remove(original.Path()); err != nil {
					t.Fatal(err)
				}
				searchHost{v}.Failed(searchui.SessionError{Err: errors.New("search source changed")})
				drainFileWork(t, v)
				if v.comparisonActive() {
					t.Fatal("comparison blocked source reconciliation")
				}
				waitUntilLoaded(t, v)
				want := survivor
				if replace {
					want = original
					if v.img.Image.Bounds().Dx() != 16 {
						t.Fatal("replaced source retained stale displayed pixels")
					}
				}
				if got, ok := v.DisplayedFile(); !ok || got.String() != want.String() || v.searchActive() || v.grid.Visible() != gridOrigin {
					t.Fatalf("source reconciliation restored stale presentation: %v", got)
				}
			})
		}
	}
	for _, operation := range []string{"ordinary", "committed-trash", "source-failure"} {
		t.Run("image-origin-batch-"+operation, func(t *testing.T) {
			v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg")
			v.grid.Close()
			v.ShowImage(0)
			waitUntilLoaded(t, v)
			publish := streamingSearchFrom(t, v)
			publish(similarity.SearchFinal, 2, 1)
			removed := []fyne.URI{v.FileAt(0), v.FileAt(2)}
			kept := []fyne.URI{v.FileAt(1), v.FileAt(3)}
			for _, source := range removed {
				if err := os.Remove(source.Path()); err != nil {
					t.Fatal(err)
				}
			}
			switch operation {
			case "ordinary":
				v.RemoveFiles([]int{0, 2})
			case "committed-trash":
				if !v.ReconcileDeletedFiles(removed) {
					t.Fatal("completed removals were ignored")
				}
			case "source-failure":
				searchHost{v}.Failed(searchui.SessionError{Err: errors.New("search source changed")})
				drainFileWork(t, v)
			}
			if got := v.display.Snapshot().Requested.Source; got == nil || got.String() != kept[0].String() {
				t.Fatalf("search restored before removals finished: requested=%v want=%v", got, kept[0])
			}
			waitUntilLoaded(t, v)
			if v.searchActive() || v.grid.Visible() || !slices.EqualFunc(v.state.files, kept, func(a, b fyne.URI) bool { return a.String() == b.String() }) {
				t.Fatalf("batch restoration lost healthy survivors: files=%v", v.state.files)
			}
			if got, ok := v.DisplayedFile(); !ok || got.String() != kept[0].String() {
				t.Fatalf("batch restored the wrong displayed source: %v", got)
			}
		})
	}
	for _, change := range []string{"deleted", "replaced", "all-deleted", "new-collection", "new-query", "closed-search"} {
		t.Run("external-"+change, func(t *testing.T) {
			v, publish := streamingSearch(t)
			publish(similarity.SearchFinal, 2, 1)
			original := v.FileAt(0)
			imageWriter, thumbWriter := v.imgCache.Capture(), v.grid.CaptureThumbs()
			switch change {
			case "deleted", "new-collection":
				if err := os.Remove(original.Path()); err != nil {
					t.Fatal(err)
				}
			case "all-deleted":
				for _, source := range v.state.files {
					if err := os.Remove(source.Path()); err != nil {
						t.Fatal(err)
					}
				}
			case "replaced":
				replacement := uitest.TempJPEGURI(t, "replacement.jpg", 16, 16, color.Black)
				data, err := os.ReadFile(replacement.Path())
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(original.Path(), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			searchHost{v}.Failed(searchui.SessionError{Err: errors.New("search source changed")})
			if change == "new-query" || change == "closed-search" {
				v.fileWork.workers.Wait()
				if change == "new-query" {
					v.startVisualSearch(v.FileAt(1).Path())
				} else {
					v.visualsearch.Exit()
				}
				drainFileWork(t, v)
				if v.searchActive() != (change == "new-query") || !imageWriter.Current() || !thumbWriter.Current() {
					t.Fatal("obsolete reconciliation changed later browsing")
				}
				return
			}
			if change == "new-collection" {
				v.fileWork.workers.Wait()
				replacement := uitest.TempJPEGURI(t, "new-collection.jpg", 12, 12, color.White)
				dropAndWait(t, v, replacement)
				drainFileWork(t, v)
				if v.FileCount() != 1 || v.FileAt(0).String() != replacement.String() || v.searchActive() {
					t.Fatal("obsolete reconciliation replaced the new collection")
				}
				return
			}
			drainFileWork(t, v)
			if v.searchActive() || imageWriter.Current() || thumbWriter.Current() {
				t.Fatal("terminal source failure retained its session or derived pixels")
			}
			want := 3
			if change == "all-deleted" {
				want = 0
			} else if change == "replaced" {
				want = 4
			}
			if v.FileCount() != want || len(v.grid.ResultIndexes()) != want || v.grid.Visible() != (want > 0) {
				t.Fatalf("origin retained stale source mappings: files=%d results=%v visible=%v", v.FileCount(), v.grid.ResultIndexes(), v.grid.Visible())
			}
			if change == "replaced" {
				waitUntilLoaded(t, v)
				if v.img.Image.Bounds().Dx() != 16 {
					t.Fatal("replaced source retained displayed pixels after reconciliation")
				}
			}
		})
	}
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
	for _, action := range []string{"exit", "back", "remove-before", "remove-occurrence"} {
		t.Run("image-origin-occurrence/"+action, func(t *testing.T) {
			v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
			duplicate := v.FileAt(1)
			v.SetMergeMode(true)
			dropAndWait(t, v, duplicate)
			origin, occurrences := -1, 0
			for i, uri := range v.state.files {
				if uri.Path() == duplicate.Path() {
					origin = i
					occurrences++
				}
			}
			if occurrences != 2 || origin == 0 {
				t.Fatal("merge fixture did not retain two image occurrences")
			}
			v.grid.Close()
			v.ShowImage(origin)
			waitUntilLoaded(t, v)
			publish := streamingSearchFrom(t, v)
			publish(similarity.SearchFinal, 0, v.FileCount()-1)
			switch action {
			case "exit":
				v.visualsearch.Exit()
			case "back":
				v.visualsearch.Back()
			case "remove-before":
				v.RemoveFiles([]int{0})
				origin--
			case "remove-occurrence":
				v.RemoveFiles([]int{origin - 1})
				origin--
			}
			waitUntilLoaded(t, v)
			if v.searchActive() || v.grid.Visible() || v.state.index != origin {
				t.Fatalf("image origin restored index %d, want occurrence at %d", v.state.index, origin)
			}
			if got, ok := v.DisplayedFile(); !ok || got.Path() != duplicate.Path() {
				t.Fatal("image origin lost its source")
			}
			v.StepImage(1)
			waitUntilLoaded(t, v)
			if v.state.index != origin+1 {
				t.Fatal("navigation did not continue after the restored occurrence")
			}
		})
	}
	t.Run("deferred-back-observes-later-preparation", func(t *testing.T) {
		v, publish := streamingSearch(t)
		publish(similarity.SearchPartial, 2, 1)
		v.grid.SimulateHover(1)
		v.findMoreLikeThis()
		publish(similarity.SearchPartial, 0, 1)
		popup := widget.NewPopUpMenu(fyne.NewMenu("", fyne.NewMenuItem("Example", func() {})), v.win.Canvas())
		popup.ShowAtPosition(fyne.NewPos(10, 10))
		v.visualsearch.Back()
		publish(similarity.SearchReady)
		wait := v.searchView.overlay
		if wait == nil {
			t.Fatal("deferred restoration has no completion")
		}
		popup.Hide()
		for v.searchView.pending != nil {
			<-wait.notice
			v.searchView.overlayUI.Drain()
		}
		visibleProgress := false
		explorerWalk(v.win.Content(), func(object fyne.CanvasObject) {
			if _, ok := object.(*widget.ProgressBar); ok {
				visibleProgress = true
			}
		})
		if !v.visualsearch.State().Progress.Complete || visibleProgress {
			t.Fatal("deferred Back restored stale progress after preparation completed")
		}
	})
	t.Run("cohort-stream-history-overlay-exit", func(t *testing.T) {
		v := explorerFixture(t)
		paths := []string{v.FileAt(0).Path(), v.FileAt(1).Path()}
		v.OpenSimilarityCohort(paths)
		v.grid.SimulateHover(1)
		camera := v.explorer.Surface().View()
		publish := streamingSearchFrom(t, v)
		publish(similarity.SearchPartial, 2, 0)
		v.grid.SimulateHover(1)
		v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitUntilLoaded(t, v)
		opened := v.state.index
		frozen := favoriteListHost{v}.CurrentFiles()
		publish(similarity.SearchFinal, 3, 2)
		if v.grid.Visible() || v.state.index != opened || !slices.Equal(frozen, favoriteListHost{v}.CurrentFiles()) {
			t.Fatal("final publication retargeted the opened visit")
		}
		v.returnToSearchGrid()
		if !slices.Equal(v.grid.ResultIndexes(), []int{1, 3, 2}) || !slices.Equal(v.grid.Selection(), []int{2}) {
			t.Fatalf("return lost live rank or selected identity: %v / %v", v.grid.ResultIndexes(), v.grid.Selection())
		}
		v.grid.SimulateHover(1)
		first := v.grid.CaptureVisit()
		v.findMoreLikeThis()
		publish(similarity.SearchPartial, 0, 2)
		popup := widget.NewPopUpMenu(fyne.NewMenu("", fyne.NewMenuItem("Example", func() {})), v.win.Canvas())
		popup.ShowAtPosition(fyne.NewPos(10, 10))
		publish(similarity.SearchFinal, 4, 2)
		before := v.grid.CaptureVisit()
		// History restoration is also a delivery: it must respect the modal
		// surface, while retaining the session's completed preparation status.
		v.visualsearch.Back()
		during := v.grid.CaptureVisit()
		if !slices.Equal(during.Results, before.Results) || !slices.Equal(during.Selected, before.Selected) || during.Highlight != before.Highlight {
			t.Fatal("Back mutated the Grid under an open overlay")
		}
		wait := v.searchView.overlay
		if wait == nil {
			t.Fatal("deferred Back has no overlay completion")
		}
		popup.Hide()
		for v.searchView.pending != nil {
			<-wait.notice
			v.searchView.overlayUI.Drain()
		}
		restored := v.grid.CaptureVisit()
		if !slices.Equal(restored.Results, first.Results) || !slices.Equal(restored.Selected, first.Selected) || restored.Highlight != first.Highlight || !v.visualsearch.State().Progress.Complete {
			t.Fatal("Back lost the first rank or restored stale preparation progress")
		}
		v.visualsearch.Exit()
		if !slices.Equal(explorerGridPaths(v), paths) || v.grid.Highlight() != 1 || !v.explorer.Surface().Visible() {
			t.Fatal("Exit did not restore the complete cohort surface")
		}
		v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
		if !v.explorerMapActive() || !reflect.DeepEqual(v.explorer.Surface().View(), camera) {
			t.Fatal("cohort Back lost the original Explorer camera")
		}
	})
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
