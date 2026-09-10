package ui

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2/canvas"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	explorerui "github.com/frathe/picfetch/internal/ui/explorer"
	"github.com/frathe/picfetch/internal/uitest"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/favstore"
	"github.com/frathe/picfetch/internal/filesort"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/preferences"
	"github.com/frathe/picfetch/internal/similarity"
)

func explorerMenu(t *testing.T, v *viewer) *fyne.MenuItem {
	t.Helper()
	for _, menu := range v.win.MainMenu().Items {
		for _, item := range menu.Items {
			if item.Label == lang.L("Visual Similarity Explorer") {
				return item
			}
		}
	}
	t.Fatal("Visual Similarity Explorer is missing from the main menu")
	return nil
}

func explorerFixture(t *testing.T) *viewer { return explorerFixtureScale(t, 1) }

func explorerLargeFixture(t *testing.T) (*viewer, func(int, bool)) {
	t.Helper()
	names := make([]string, 144)
	for i := range names {
		names[i] = fmt.Sprintf("%03d.jpg", i)
	}
	v, publish := streamingExplorerEvents(t, names...)
	v.win.Resize(fyne.NewSize(1100, 700))
	preview := uitest.EncodeJPEG(t, 128, 96, color.White)
	return v, func(count int, complete bool) {
		items := make([]similarity.Item, count)
		var merges []similarity.CohortMerge
		for i := range items {
			group := max(0, i-15)
			items[i] = similarity.Item{Path: v.FileAt(i).Path(), Cohort: fmt.Sprintf("%03d", group), Position: []float32{float32(group % 12), float32(group / 12)}, Preview: preview}
			if group > 0 {
				merges = append(merges, similarity.CohortMerge{Left: "000", Right: items[i].Cohort})
			}
		}
		publish(similarity.Event{Total: v.FileCount(), Successful: count, Complete: complete, Items: items, Merges: merges})
	}
}

func explorerSamples(pile *explorerui.Pile) []string {
	var names []string
	explorerWalk(pile, func(o fyne.CanvasObject) {
		if img, ok := o.(*canvas.Image); ok && img.Resource != nil {
			names = append(names, img.Resource.Name())
		}
	})
	return names
}

func explorerFixtureScale(t *testing.T, factor float32) *viewer {
	t.Helper()
	names := make([]string, 18)
	for i := range names {
		names[i] = fmt.Sprintf("%02d.jpg", i)
	}
	v := openGridWith(t, names...)
	previews := make([][]byte, 18)
	for i := range previews {
		previews[i] = uitest.EncodeJPEG(t, 128, 96, color.NRGBA{R: uint8(50 + i*9), G: uint8(70 + (i*19)%160), B: uint8(90 + (i*31)%150), A: 255})
	}
	v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
		var items []similarity.Item
		for i, path := range paths {
			group := "a"
			x := float32(-1)
			if i == 16 {
				group = "b"
				x = 1
			}
			if i == 17 {
				group = "unassigned"
			}
			items = append(items, similarity.Item{Path: path, Cohort: group, Position: []float32{x * factor, 0}, Preview: previews[i]})
		}
		emit(similarity.Event{Total: len(paths), Successful: len(paths), Complete: true, Items: items})
		return nil
	}
	item := explorerMenu(t, v)
	if item.Disabled {
		t.Fatal("explorer unavailable with opened images")
	}
	item.Action()
	v.settleExplorer()
	return v
}

func explorerWalk(o fyne.CanvasObject, visit func(fyne.CanvasObject)) {
	if !o.Visible() {
		return
	}
	visit(o)
	switch o := o.(type) {
	case *fyne.Container:
		for _, child := range o.Objects {
			explorerWalk(child, visit)
		}
	case fyne.Widget:
		for _, child := range fynetest.WidgetRenderer(o).Objects() {
			explorerWalk(child, visit)
		}
	}
}
func explorerPiles(v *viewer) []*explorerui.Pile {
	var piles []*explorerui.Pile
	explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
		if p, ok := o.(*explorerui.Pile); ok {
			piles = append(piles, p)
		}
	})
	return piles
}
func explorerGridPaths(v *viewer) []string {
	var paths []string
	for _, i := range v.grid.ResultIndexes() {
		paths = append(paths, v.FileAt(i).Path())
	}
	return paths
}

func explorerButton(t *testing.T, v *viewer, label string) *widget.Button {
	t.Helper()
	var found *widget.Button
	explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
		if button, ok := o.(*widget.Button); ok && button.Text == lang.L(label) {
			found = button
		}
	})
	if found == nil {
		t.Fatalf("missing visible button %q", label)
	}
	return found
}

func explorerTag(t *testing.T, v *viewer, label string, count int) (*widget.Check, *widget.Hyperlink) {
	t.Helper()
	var found *widget.Check
	var number *widget.Hyperlink
	explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
		row, ok := o.(*fyne.Container)
		if !ok {
			return
		}
		var check *widget.Check
		var link *widget.Hyperlink
		for _, child := range row.Objects {
			switch child := child.(type) {
			case *widget.Check:
				check = child
			case *widget.Hyperlink:
				link = child
			}
		}
		if check != nil && check.Text == lang.L(label) && link != nil && link.Text == fmt.Sprintf("(%d)", count) {
			found, number = check, link
		}
	})
	if found == nil {
		t.Fatalf("missing visible tag %q with clickable count %d", label, count)
	}
	return found, number
}

// A publication is observed only after its provider callback has queued it and
// the owning UI queue has delivered it. The worker remains held between maps.
func streamingExplorer(t *testing.T) (*viewer, func([]string, bool)) {
	t.Helper()
	v, publish := streamingExplorerEvents(t)
	preview := uitest.EncodeJPEG(t, 128, 96, color.White)
	return v, func(groups []string, complete bool) {
		var items []similarity.Item
		for i, group := range groups {
			items = append(items, similarity.Item{Path: v.FileAt(i).Path(), Cohort: group, Position: []float32{float32(i * 100), 0}, Preview: preview})
		}
		publish(similarity.Event{Items: items, Successful: len(items), Total: v.FileCount(), Complete: complete, Stage: "encoding"})
	}
}

func streamingExplorerEvents(t *testing.T, names ...string) (*viewer, func(similarity.Event)) {
	t.Helper()
	if len(names) == 0 {
		names = []string{"a.jpg", "b.jpg", "c.jpg", "d.jpg", "e.jpg", "f.jpg", "g.jpg", "h.jpg"}
	}
	v := openGridWith(t, names...)
	events := make(chan similarity.Event)
	published := make(chan struct{})
	v.explorerAnalyze = func(ctx context.Context, _ []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case event := <-events:
				emit(event)
				published <- struct{}{}
				if event.Complete {
					return nil
				}
			}
		}
	}
	explorerMenu(t, v).Action()
	return v, func(event similarity.Event) {
		events <- event
		<-published
		v.explorer.ui.Drain()
	}
}

func TestVisualSimilarityExplorer(t *testing.T) {
	before := preferences.Load(testApp)
	t.Cleanup(func() { preferences.Save(testApp, before) })
	for _, key := range []string{"similarityFavoriteCache", "similarityAutoFit", "similarityAutoUpdate"} {
		testApp.Preferences().RemoveValue(key)
	}

	t.Run("source_changes", func(t *testing.T) {
		t.Run("missing_cohort_member", func(t *testing.T) {
			v := explorerFixture(t)
			fynetest.Tap(explorerPiles(v)[0])
			members := explorerGridPaths(v)
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
			waitUntilLoaded(t, v)
			v.preloads.Wait()
			if err := os.Remove(members[len(members)-1]); err != nil {
				t.Fatal(err)
			}
			v.imgCache.Purge()
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEnd})
			waitUntilLoaded(t, v)
			remaining := members[:len(members)-1]
			if current, _, _ := v.CurrentFile(); !slices.Contains(remaining, current.Path()) {
				t.Fatalf("missing member sent navigation outside its cohort: %s", current.Path())
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			if !v.grid.Visible() || !slices.Equal(explorerGridPaths(v), remaining) {
				t.Fatal("missing file remained actionable in the frozen cohort")
			}
		})

		t.Run("remove_after_reorder", func(t *testing.T) {
			v := openGridWith(t, "c.jpg", "a.jpg", "b.jpg")
			preview := uitest.EncodeJPEG(t, 32, 24, color.White)
			queued, release := make(chan struct{}), make(chan struct{})
			unblock := sync.OnceFunc(func() { close(release) })
			t.Cleanup(unblock)
			v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
				items := []similarity.Item{
					{Path: paths[0], Cohort: "a", Preview: preview},
					{Path: paths[1], Cohort: "a", Preview: preview},
					{Path: paths[2], Cohort: "b", Preview: preview},
				}
				emit(similarity.Event{Items: items, Successful: 3, Total: 3})
				close(queued)
				<-release
				emit(similarity.Event{Items: items, Successful: 3, Total: 3, Complete: true})
				return nil
			}
			explorerMenu(t, v).Action()
			<-queued
			v.explorer.ui.Drain()
			fynetest.Tap(explorerPiles(v)[0])
			members := explorerGridPaths(v)
			moving, moveRelease := make(chan struct{}), make(chan struct{})
			finishMove := sync.OnceFunc(func() { close(moveRelease) })
			t.Cleanup(finishMove)
			uitest.StubTrashMove(t, func(path string) error {
				close(moving)
				<-moveRelease
				return os.Remove(path)
			})
			v.requestDelete()
			v.deletion.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
			v.deletion.HandleKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
			<-moving
			v.menus.Actions().Sort()[filesort.ByDropOrder].Action()
			waitForSort(t, v)
			waitUntilLoaded(t, v)
			if got := explorerGridPaths(v); !slices.Equal(got, members) {
				t.Fatalf("reorder changed cohort identities: %v", got)
			}
			finishMove()
			v.deletion.Settle()
			if got := explorerGridPaths(v); !slices.Equal(got, members[1:]) {
				t.Fatalf("deletion after reorder targeted the wrong member: %v", got)
			}
			unblock()
			v.settleExplorer()
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
			waitUntilLoaded(t, v)
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
			waitUntilLoaded(t, v)
			if current, _, _ := v.CurrentFile(); current.Path() != members[1] {
				t.Fatal("navigation escaped the surviving frozen cohort")
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			fynetest.Tap(explorerButton(t, v, "Back to map"))
			if len(explorerPiles(v)) != 0 {
				t.Fatal("removed source survived in the map through a late publication")
			}
		})

		for _, destination := range []string{"source", "alias", "unrelated"} {
			t.Run(destination, func(t *testing.T) {
				v := explorerFixture(t)
				fynetest.Tap(explorerPiles(v)[0])
				members := explorerGridPaths(v)
				v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
				waitUntilLoaded(t, v)
				source, _, _ := v.CurrentFile()
				dest := source
				if destination != "source" {
					dest = storage.NewFileURI(filepath.Join(t.TempDir(), "export.png"))
				}
				if destination == "alias" {
					if err := os.Rename(source.Path(), dest.Path()); err != nil {
						t.Fatal(err)
					}
					if err := os.Symlink(dest.Path(), source.Path()); err != nil {
						t.Fatal(err)
					}
				}
				entered, release := make(chan struct{}), make(chan struct{})
				unblock := sync.OnceFunc(func() { close(release) })
				t.Cleanup(unblock)
				v.fileWork.export = func(ctx context.Context, dest fyne.URI, pixels image.Image, src fyne.URI, opts imaging.ExportOptions) (imaging.WriteResult, error) {
					result, err := imaging.ExportContext(ctx, dest, pixels, src, opts)
					close(entered)
					<-release
					return result, err
				}
				uitest.StubSaveChooser(t, func(_ string) (fyne.URI, error) { return dest, nil })
				v.rotateBy(1)
				v.exportAs(".png")
				<-entered
				v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEnd})
				waitUntilLoaded(t, v)
				current, _, _ := v.CurrentFile()
				unblock()
				settleChooser(t, v)
				if got, _, _ := v.CurrentFile(); got.String() != current.String() {
					t.Fatal("stale export changed the currently viewed file")
				}
				v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
				if !v.grid.Visible() || !slices.Equal(explorerGridPaths(v), members) {
					t.Fatal("committed export changed the frozen browsing cohort")
				}
				fynetest.Tap(explorerButton(t, v, "Back to map"))
				if destination == "unrelated" {
					if len(explorerPiles(v)) != 2 {
						t.Fatal("unrelated export invalidated source analysis")
					}
				} else if len(explorerPiles(v)) != 0 {
					t.Fatal("committed source write retained stale map results")
				}
			})
		}
	})

	t.Run("lifecycle", func(t *testing.T) {
		for _, action := range []string{"exit", "restart", "shutdown"} {
			t.Run(action, func(t *testing.T) {
				v := openGridWith(t, "a.jpg", "b.jpg")
				preview := uitest.EncodeJPEG(t, 32, 24, color.White)
				queued := make(chan struct{})
				v.explorerAnalyze = func(ctx context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
					emit(similarity.Event{Complete: true, Successful: 1, Total: 2, Items: []similarity.Item{
						{Path: paths[0], Cohort: "old", Preview: preview},
					}})
					close(queued)
					<-ctx.Done()
					return ctx.Err()
				}
				explorerMenu(t, v).Action()
				<-queued
				oldQueue := v.explorer.ui
				if action == "shutdown" {
					lifecycle, ok := testApp.Lifecycle().(interface{ OnStopped() func() })
					if !ok {
						t.Fatal("test app has no stopped hook")
					}
					previous := lifecycle.OnStopped()
					registerShutdown(testApp, v)
					shutdown := lifecycle.OnStopped()
					testApp.Lifecycle().SetOnStopped(previous)
					shutdown()
				} else {
					v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
				}
				v.explorer.workers.Wait()
				v.explorer.ui = &uitest.UIQueue{}
				v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
					emit(similarity.Event{Complete: true, Successful: 1, Total: 2, Items: []similarity.Item{
						{Path: paths[1], Cohort: "new", Preview: preview},
					}})
					return nil
				}
				if action != "exit" {
					explorerMenu(t, v).Action()
				}
				v.settleExplorer()
				oldQueue.Drain()
				piles := explorerPiles(v)
				if action != "restart" {
					if len(piles) != 0 || v.explorer.surface.Visible() || v.FileCount() != 2 {
						t.Fatal("retired analysis mutated or reopened the viewer")
					}
					return
				}
				if len(piles) != 1 {
					t.Fatalf("restarted map has %d piles", len(piles))
				}
				fynetest.Tap(piles[0])
				if got := explorerGridPaths(v); !slices.Equal(got, []string{v.FileAt(1).Path()}) {
					t.Fatalf("old queued result replaced the new cohort: %v", got)
				}
			})
		}
	})

	t.Run("recovery", func(t *testing.T) {
		for _, failure := range []string{"setup", "after_partial", "after_final", "incomplete"} {
			t.Run(failure, func(t *testing.T) {
				v := openGridWith(t, "a.jpg", "b.jpg")
				preview := uitest.EncodeJPEG(t, 32, 24, color.White)
				v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
					if failure != "setup" {
						emit(similarity.Event{Complete: failure == "after_final", Successful: 1, Total: 2, Items: []similarity.Item{
							{Path: paths[0], Cohort: "failed", Preview: preview},
						}})
					}
					if failure == "incomplete" {
						return nil
					}
					return io.ErrUnexpectedEOF
				}
				explorerMenu(t, v).Action()
				v.settleExplorer()
				failed := false
				explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
					if label, ok := o.(*widget.Label); ok && label.Text == lang.L("Analysis failed. Open the explorer to retry.") {
						failed = true
					}
				})
				if !failed {
					t.Fatal("analysis failure has no visible recovery feedback")
				}
				v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
					emit(similarity.Event{Complete: true, Successful: 1, Failed: 1, Total: 2, Items: []similarity.Item{
						{Path: paths[1], Cohort: "retried", Preview: preview},
						{Path: paths[0], Error: "unreadable"},
					}})
					return nil
				}
				explorerMenu(t, v).Action()
				v.settleExplorer()
				piles := explorerPiles(v)
				if len(piles) != 1 {
					t.Fatalf("retry produced %d piles, want one readable cohort", len(piles))
				}
				fynetest.Tap(piles[0])
				if got := explorerGridPaths(v); !slices.Equal(got, []string{v.FileAt(1).Path()}) {
					t.Fatalf("retry retained failed analysis: %v", got)
				}
			})
		}
	})

	t.Run("granularity", func(t *testing.T) {
		v, publish := streamingExplorerEvents(t)
		preview := uitest.EncodeJPEG(t, 32, 24, color.White)
		var items []similarity.Item
		for i, group := range []string{"a", "a", "b", "b", "c", "unassigned"} {
			items = append(items, similarity.Item{Path: v.FileAt(i).Path(), Cohort: group, Tags: []string{"bird"}, Preview: preview, Position: []float32{float32(i), 0}})
		}
		merges := []similarity.CohortMerge{{Left: "a", Right: "b"}, {Left: "b", Right: "c"}}
		publish(similarity.Event{Items: items, Merges: merges, Successful: 6, Total: 8})
		var slider *widget.Slider
		explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
			if s, ok := o.(*widget.Slider); ok {
				slider = s
			}
		})
		if slider == nil {
			t.Fatal("map is missing its granularity slider")
		}
		if len(explorerPiles(v)) != 3 {
			t.Fatal("default granularity changed the original cohorts")
		}
		for _, x := range []float32{0, slider.Size().Width, slider.Size().Width / 2} {
			slider.Dragged(&fyne.DragEvent{Position: fyne.NewPos(x, slider.Size().Height/2)})
			if len(explorerPiles(v)) != 3 {
				t.Fatal("dragging granularity rebuilt the map before release")
			}
		}
		if slider.Value != 50 {
			t.Fatal("granularity thumb did not follow the drag")
		}
		publish(similarity.Event{Items: items, Merges: merges, Successful: 6, Total: 8})
		if len(explorerPiles(v)) != 3 || slider.Value != 50 {
			t.Fatal("analysis publication applied unfinished granularity or moved the thumb")
		}
		slider.DragEnd()
		if len(explorerPiles(v)) != 2 || v.win.Canvas().Focused() != nil {
			t.Fatal("broader granularity did not combine the nearest cohorts or release focus")
		}
		fynetest.Tap(explorerPiles(v)[0])
		want := []string{v.FileAt(0).Path(), v.FileAt(1).Path(), v.FileAt(2).Path(), v.FileAt(3).Path()}
		if !slices.Equal(explorerGridPaths(v), want) {
			t.Fatalf("broader cohort membership: %v", explorerGridPaths(v))
		}
		items = append(items, similarity.Item{Path: v.FileAt(6).Path(), Cohort: "a", Tags: []string{"bird"}, Preview: preview})
		publish(similarity.Event{Items: items, Merges: merges, Successful: 7, Total: 8, Complete: true})
		if !slices.Equal(explorerGridPaths(v), want) {
			t.Fatal("new hierarchy changed the open grid")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		if len(explorerPiles(v)) != 2 || slider.Value != 50 {
			t.Fatal("publication or grid return reset granularity")
		}
		explorerTag(t, v, "Bird", 7)
		fynetest.TapAt(slider, fyne.NewPos(0, slider.Size().Height/2))
		if len(explorerPiles(v)) != 1 {
			t.Fatal("broadest granularity did not join the assigned groups")
		}
		fynetest.Tap(explorerButton(t, v, "Unassigned (1)"))
		if !slices.Equal(explorerGridPaths(v), []string{v.FileAt(5).Path()}) {
			t.Fatal("granularity moved unassigned images into a cohort")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		slider.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
		if len(explorerPiles(v)) != 2 {
			t.Fatal("keyboard granularity did not apply its completed change")
		}
		fynetest.TapAt(slider, fyne.NewPos(slider.Size().Width, slider.Size().Height/2))
		if len(explorerPiles(v)) != 3 {
			t.Fatal("fine granularity did not restore the original groups")
		}
		fynetest.TapAt(slider, fyne.NewPos(0, slider.Size().Height/2))
		v.LeaveSimilarityMap()
		v.explorerAnalyze = func(_ context.Context, _ []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			emit(similarity.Event{Items: items, Merges: merges, Successful: 7, Total: 8, Complete: true})
			return nil
		}
		explorerMenu(t, v).Action()
		v.settleExplorer()
		if len(explorerPiles(v)) != 3 || slider.Value != 100 {
			t.Fatal("reopening the explorer retained the previous applied granularity")
		}
	})

	t.Run("keyboard", func(t *testing.T) {
		v, publish := streamingExplorerEvents(t)
		v.win.Resize(fyne.NewSize(1100, 700))
		preview := uitest.EncodeJPEG(t, 32, 24, color.White)
		var items []similarity.Item
		for i, position := range [][]float32{{-1, -1}, {1, -1}, {-1, 1}, {1, 1}} {
			items = append(items, similarity.Item{Path: v.FileAt(i).Path(), Cohort: fmt.Sprint(i), Position: position, Tags: []string{"bird"}, Preview: preview})
		}
		publish(similarity.Event{Items: items, Successful: 4, Total: 8})
		piles := explorerPiles(v)
		for _, label := range []string{"+", "-", "Fit map"} {
			fynetest.Tap(explorerButton(t, v, label))
			if v.win.Canvas().Focused() != nil {
				t.Fatalf("%s captured keyboard focus instead of returning it to the map", label)
			}
		}
		width := piles[0].Size().Width
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyPlus})
		if piles[0].Size().Width <= width {
			t.Fatal("keyboard + did not zoom the map")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyMinus})
		if math.Abs(float64(piles[0].Size().Width-width)) > .01 {
			t.Fatal("keyboard - did not undo zoom")
		}
		highlight := func(want *explorerui.Pile) {
			t.Helper()
			for _, pile := range explorerPiles(v) {
				visible := false
				explorerWalk(pile, func(o fyne.CanvasObject) {
					if frame, ok := o.(*canvas.Rectangle); ok && frame.StrokeWidth >= 2 && frame.Size().Width > 0 {
						visible = true
					}
				})
				if visible != (pile == want) {
					t.Fatal("arrow navigation did not visibly highlight exactly the active stack")
				}
			}
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
		highlight(piles[0])
		for _, step := range []struct {
			key   fyne.KeyName
			index int
		}{{fyne.KeyRight, 1}, {fyne.KeyDown, 3}, {fyne.KeyLeft, 2}, {fyne.KeyUp, 0}} {
			v.handleKeyEvent(&fyne.KeyEvent{Name: step.key})
			highlight(piles[step.index])
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		if !v.grid.Visible() || !slices.Equal(explorerGridPaths(v), []string{v.FileAt(0).Path()}) {
			t.Fatal("Enter did not open the highlighted stack")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		highlight(piles[0])
		items[0].Cohort = "renamed"
		publish(similarity.Event{Items: items, Successful: 4, Total: 8, Complete: true})
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		if !slices.Equal(explorerGridPaths(v), []string{v.FileAt(0).Path()}) {
			t.Fatal("publication lost the selected source identity")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		fynetest.Tap(explorerButton(t, v, "Clear tags"))
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		if v.grid.Visible() {
			t.Fatal("keyboard opened a filtered-out stack")
		}
	})

	t.Run("navigation_preview_reuse", func(t *testing.T) {
		names := make([]string, 30)
		for i := range names {
			names[i] = fmt.Sprintf("%02d.jpg", i)
		}
		v, publish := streamingExplorerEvents(t, names...)
		v.win.Resize(fyne.NewSize(1100, 700))
		items := make([]similarity.Item, len(names))
		for i := range items {
			items[i] = similarity.Item{
				Path: v.FileAt(i).Path(), Cohort: fmt.Sprint(i / 15), Position: []float32{float32(i / 15), 0},
				Preview: uitest.EncodeJPEG(t, 160, 120, color.NRGBA{R: uint8(40 + i*6), G: 120, B: 180, A: 255}),
			}
		}
		publish(similarity.Event{Items: items, Successful: len(items), Total: len(items), Complete: true})
		v.settleExplorer()
		for _, key := range []fyne.KeyName{fyne.KeyRight, fyne.KeyRight, fyne.KeyLeft} {
			v.handleKeyEvent(&fyne.KeyEvent{Name: key})
		}
		piles := explorerPiles(v)
		if len(piles) != 2 || len(explorerSamples(piles[0])) != 15 || len(explorerSamples(piles[1])) != 15 {
			t.Fatal("navigation fixture must display two complete 15-sample piles")
		}
		snapshot := func() []byte {
			var b bytes.Buffer
			if err := png.Encode(&b, v.win.Canvas().Capture()); err != nil {
				t.Fatal(err)
			}
			return b.Bytes()
		}
		before := snapshot()
		var start, end runtime.MemStats
		runtime.ReadMemStats(&start)
		for range 20 {
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyLeft})
		}
		runtime.ReadMemStats(&end)
		allocated := end.TotalAlloc - start.TotalAlloc
		t.Logf("40 selections allocated %.2f MiB", float64(allocated)/(1<<20))
		if allocated > 8<<20 {
			t.Errorf("selection allocated %.2f MiB for unchanged previews; want under 8 MiB", float64(allocated)/(1<<20))
		}
		if !bytes.Equal(before, snapshot()) {
			t.Fatal("returning selection to its starting pile changed the rendered map")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		if len(explorerGridPaths(v)) != 15 {
			t.Fatal("keyboard selection lost the complete cohort")
		}
	})

	t.Run("viewport_previews", func(t *testing.T) {
		names := make([]string, 300)
		for i := range names {
			names[i] = fmt.Sprintf("%03d.jpg", i)
		}
		v, publish := streamingExplorerEvents(t, names...)
		v.win.Resize(fyne.NewSize(1100, 700))
		items := make([]similarity.Item, len(names))
		for i := range items {
			group := i / 15
			items[i] = similarity.Item{
				Path: v.FileAt(i).Path(), Cohort: fmt.Sprintf("%02d", group), Position: []float32{float32(group % 5), float32(group / 5)},
				Preview: uitest.EncodeJPEG(t, 160, 120, color.NRGBA{R: uint8(i), G: 120, B: 180, A: 255}),
			}
		}
		publish(similarity.Event{Items: items, Successful: len(items), Total: len(items), Complete: true})
		v.settleExplorer()
		snapshot := func() []byte {
			var b bytes.Buffer
			if err := png.Encode(&b, v.win.Canvas().Capture()); err != nil {
				t.Fatal(err)
			}
			return b.Bytes()
		}
		before := snapshot()
		allocated := func() uint64 {
			runtime.GC()
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			return m.HeapAlloc
		}
		loaded := allocated()
		v.explorer.surface.Dragged(&fyne.DragEvent{Dragged: fyne.Delta{DX: 10000, DY: 10000}})
		after := allocated()
		t.Logf("loaded heap %.2f MiB; away %.2f MiB", float64(loaded)/(1<<20), float64(after)/(1<<20))
		if after+4<<20 > loaded {
			t.Errorf("panning away did not release at least 4 MiB of decoded previews")
		}
		fynetest.Tap(explorerButton(t, v, "Fit map"))
		piles := explorerPiles(v)
		if len(piles) != 20 {
			t.Fatalf("viewport eviction lost cohorts: got %d", len(piles))
		}
		for _, pile := range piles {
			if len(explorerSamples(pile)) != 15 {
				t.Fatal("returning to the map did not restore every sample")
			}
			explorerWalk(pile, func(o fyne.CanvasObject) {
				if img, ok := o.(*canvas.Image); ok && img.Image == nil {
					t.Fatal("returning to the viewport left a preview waiting for pixels")
				}
			})
		}
		if !bytes.Equal(before, snapshot()) {
			t.Fatal("viewport round trip changed the map's rendered appearance")
		}
		fynetest.Tap(piles[0])
		if len(explorerGridPaths(v)) != 15 {
			t.Fatal("viewport eviction lost full cohort membership")
		}
	})

	t.Run("viewport_margin", func(t *testing.T) {
		v := explorerFixture(t)
		pile := explorerPiles(v)[0]
		// Start with a decoded pile one pile-width beyond the left viewport
		// edge, then cross that preparation boundary repeatedly.
		v.explorer.surface.Dragged(&fyne.DragEvent{Dragged: fyne.Delta{DX: -2*pile.Size().Width - pile.Position().X}})
		var start, end runtime.MemStats
		runtime.ReadMemStats(&start)
		for range 20 {
			v.explorer.surface.Dragged(&fyne.DragEvent{Dragged: fyne.Delta{DX: -1}})
			v.explorer.surface.Dragged(&fyne.DragEvent{Dragged: fyne.Delta{DX: 1}})
		}
		runtime.ReadMemStats(&end)
		allocated := end.TotalAlloc - start.TotalAlloc
		if allocated > 1<<20 {
			t.Fatalf("small reversals at the viewport margin churned %.2f MiB of previews", float64(allocated)/(1<<20))
		}
	})

	t.Run("tags_collapse", func(t *testing.T) {
		v := explorerFixture(t)
		v.win.Resize(fyne.NewSize(1100, 700))
		v.ForceRepaint()
		before := v.explorer.surface.Size()
		piles := explorerPiles(v)
		fynetest.Tap(explorerButton(t, v, "Hide tags"))
		v.ForceRepaint()
		if v.explorer.surface.Size().Width <= before.Width {
			t.Fatal("collapsing tags did not give their width back to the map")
		}
		explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
			if b, ok := o.(*widget.Button); ok && b.Text == lang.L("Clear tags") {
				t.Fatal("collapsed tags still expose sidebar controls")
			}
		})
		fynetest.Tap(explorerButton(t, v, "Show tags"))
		v.ForceRepaint()
		if v.explorer.surface.Size() != before || !slices.Equal(explorerPiles(v), piles) || v.win.Canvas().Focused() != nil {
			t.Fatal("restoring tags changed map contents, geometry or input focus")
		}
	})

	t.Run("tags_browse", func(t *testing.T) {
		v, publish := streamingExplorerEvents(t)
		preview := uitest.EncodeJPEG(t, 32, 24, color.White)
		items := []similarity.Item{
			{Path: v.FileAt(0).Path(), Cohort: "a", Tags: []string{"bird", "bird"}, Preview: preview},
			{Path: v.FileAt(1).Path(), Cohort: "a", Tags: []string{"dog"}, Preview: preview},
			{Path: v.FileAt(2).Path(), Cohort: "b", Tags: []string{"bird"}, Preview: preview},
			{Path: v.FileAt(3).Path(), Cohort: "unassigned", Tags: []string{"bird"}, Preview: preview},
		}
		items = append(items, items[0])
		publish(similarity.Event{Items: items, Successful: 4, Total: 8})
		check, number := explorerTag(t, v, "Bird", 3)
		fynetest.Tap(check)
		before := explorerPiles(v)[0].Position()
		fynetest.TapAt(number, fyne.NewPos(number.MinSize().Width/2, number.MinSize().Height/2))
		want := []string{v.FileAt(0).Path(), v.FileAt(2).Path(), v.FileAt(3).Path()}
		if !v.grid.Visible() || !slices.Equal(explorerGridPaths(v), want) {
			t.Fatalf("tag link did not open exactly the matching distinct images: %v", explorerGridPaths(v))
		}
		items = append(items, similarity.Item{Path: v.FileAt(4).Path(), Cohort: "b", Tags: []string{"bird"}, Preview: preview})
		publish(similarity.Event{Items: items, Successful: 5, Total: 8, Complete: true})
		if !slices.Equal(explorerGridPaths(v), want) {
			t.Fatal("publication changed the tag grid being browsed")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		check, number = explorerTag(t, v, "Bird", 4)
		if check.Checked || explorerPiles(v)[0].Position() != before {
			t.Fatal("tag grid return lost filtering or map position")
		}
		fynetest.TapAt(number, fyne.NewPos(number.MinSize().Width/2, number.MinSize().Height/2))
		want = append(want, v.FileAt(4).Path())
		if !slices.Equal(explorerGridPaths(v), want) {
			t.Fatal("reopened tag grid did not include new matches")
		}
	})

	t.Run("tags_filter", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg", "e.jpg")
		paths := make([]string, v.FileCount())
		for i := range paths {
			paths[i] = v.FileAt(i).Path()
		}
		preview := uitest.EncodeJPEG(t, 128, 96, color.White)
		v.explorerAnalyze = func(_ context.Context, _ []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			items := []similarity.Item{
				{Path: paths[0], Cohort: "a", Tags: []string{"bird", "dog", "bird"}},
				{Path: paths[1], Cohort: "a", Tags: []string{"bird"}},
				{Path: paths[2], Cohort: "b", Tags: []string{"dog"}},
				{Path: paths[3], Cohort: "c"},
				{Path: paths[4], Cohort: "unassigned", Tags: []string{"bird"}},
			}
			for i := range items {
				items[i].Preview = preview
			}
			// Repeated source identities and repeated labels count only once.
			items = append(items, items[0])
			emit(similarity.Event{Items: items, Total: 5, Successful: 5, Complete: true})
			return nil
		}
		explorerMenu(t, v).Action()
		v.settleExplorer()
		check := func(label string, count int) *widget.Check {
			t.Helper()
			check, _ := explorerTag(t, v, label, count)
			return check
		}
		bird, dog, untagged := check("Bird", 3), check("Dog", 2), check("Untagged", 1)
		if !bird.Checked || !dog.Checked || !untagged.Checked || len(explorerPiles(v)) != 3 {
			t.Fatal("tags must start checked with every cohort reachable")
		}
		if path := os.Getenv("PICFETCH_EXPLORER_TAG_QA"); path != "" {
			v.win.Resize(fyne.NewSize(1100, 700))
			v.ForceRepaint()
			v.explorer.surface.Fit()
			f, err := os.Create(path)
			if err != nil {
				t.Fatal(err)
			}
			err = png.Encode(f, v.win.Canvas().Capture())
			closeErr := f.Close()
			if err != nil || closeErr != nil {
				t.Fatalf("tag render: %v/%v", err, closeErr)
			}
		}
		first := explorerPiles(v)[0]
		position, size := first.Position(), first.Size()
		fynetest.Tap(untagged)
		fynetest.Tap(dog)
		if v.win.Canvas().Focused() != nil {
			t.Fatal("tag checkbox captured keyboard focus and blocked map Escape")
		}
		if len(explorerPiles(v)) != 1 || explorerPiles(v)[0] != first {
			t.Fatal("Bird must show only its assigned cohort; filtering must preserve piles")
		}
		var clearTags *widget.Button
		explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
			if b, ok := o.(*widget.Button); ok && b.Text == lang.L("Clear tags") {
				clearTags = b
			}
		})
		if clearTags == nil {
			t.Fatal("tag list needs a clear action to avoid toggling every tag individually")
		}
		fynetest.Tap(clearTags)
		if v.win.Canvas().Focused() != nil {
			t.Fatal("clear-tags button captured keyboard focus and blocked map Escape")
		}
		if len(explorerPiles(v)) != 0 {
			t.Fatal("all tags off must hide every pile")
		}
		explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
			if b, ok := o.(*widget.Button); ok && b.Text == fmt.Sprintf(lang.L("Unassigned (%d)"), 1) {
				t.Fatal("all tags off left Unassigned reachable")
			}
		})
		fynetest.Tap(dog)
		if len(explorerPiles(v)) != 2 || first.Position() != position || first.Size() != size {
			t.Fatal("OR filtering changed the map camera or cohort positions")
		}
		check("Bird", 3)
		check("Dog", 2)
		check("Untagged", 1)
		fynetest.Tap(first)
		if got := explorerGridPaths(v); !slices.Equal(got, paths[:2]) {
			t.Fatalf("filter changed cohort membership: %v, want %v", got, paths[:2])
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		if check("Bird", 3).Checked || !check("Dog", 2).Checked || len(explorerPiles(v)) != 2 {
			t.Fatal("returning from the cohort lost filter choices")
		}
		var allTags *widget.Button
		explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
			if b, ok := o.(*widget.Button); ok && b.Text == lang.L("All tags") {
				allTags = b
			}
		})
		if allTags == nil {
			t.Fatal("tag list needs an action to restore every tag")
		}
		fynetest.Tap(allTags)
		if !check("Bird", 3).Checked || !check("Untagged", 1).Checked || len(explorerPiles(v)) != 3 {
			t.Fatal("All tags did not restore every cohort")
		}
		fynetest.Tap(clearTags)
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		explorerMenu(t, v).Action()
		v.settleExplorer()
		if !check("Bird", 3).Checked || !check("Untagged", 1).Checked || len(explorerPiles(v)) != 3 {
			t.Fatal("a fresh explorer session must reset all tags to checked")
		}
	})

	t.Run("tags_progressive", func(t *testing.T) {
		v, publish := streamingExplorerEvents(t)
		preview := uitest.EncodeJPEG(t, 128, 96, color.White)
		item := func(i int, cohort string, tags ...string) similarity.Item {
			return similarity.Item{Path: v.FileAt(i).Path(), Cohort: cohort, Tags: tags, Preview: preview}
		}
		check := func(label string) *widget.Check {
			t.Helper()
			check, _ := explorerTag(t, v, label, 1)
			return check
		}
		publish(similarity.Event{Items: []similarity.Item{item(0, "a", "bird"), item(1, "b", "dog"), item(2, "c")}, Successful: 3, Total: 8})
		fynetest.Tap(check("Bird"))
		fynetest.Tap(explorerPiles(v)[0])
		frozen := []string{v.FileAt(1).Path()}
		if !slices.Equal(explorerGridPaths(v), frozen) {
			t.Fatal("did not open the dog cohort")
		}
		publish(similarity.Event{Items: []similarity.Item{item(1, "b", "dog"), item(2, "b", "cat")}, Successful: 2, Total: 8})
		if !slices.Equal(explorerGridPaths(v), frozen) {
			t.Fatal("tagged publication changed an open cohort")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		if !check("Cat").Checked {
			t.Fatal("a newly discovered tag must start checked")
		}
		failed := item(4, "failed", "bird")
		failed.Error = "unreadable"
		publish(similarity.Event{Items: []similarity.Item{item(0, "a", "bird"), item(1, "b", "dog"), item(2, "b", "cat"), item(3, "c", "unknown-label"), failed}, Successful: 4, Failed: 1, Total: 8, Complete: true})
		if check("Bird").Checked || !check("Cat").Checked || !check("Untagged").Checked || len(explorerPiles(v)) != 2 {
			t.Fatal("publication lost choices, counted failed sources, or hid unknown content")
		}
		fynetest.Tap(check("Dog"))
		fynetest.Tap(check("Untagged"))
		if len(explorerPiles(v)) != 1 {
			t.Fatal("cohort must stay visible while any member tag is active")
		}
		fynetest.Tap(explorerPiles(v)[0])
		if got := explorerGridPaths(v); !slices.Equal(got, []string{v.FileAt(1).Path(), v.FileAt(2).Path()}) {
			t.Fatalf("reopened tagged cohort did not use current complete membership: %v", got)
		}
	})

	t.Run("release_on_exit", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg", "e.jpg", "f.jpg")
		allocated := func() uint64 { runtime.GC(); var m runtime.MemStats; runtime.ReadMemStats(&m); return m.HeapAlloc }
		baseline := allocated()
		v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			for pass := range 2 {
				var items []similarity.Item
				for i, path := range paths {
					preview := make([]byte, 8<<20)
					copy(preview, uitest.EncodeJPEG(t, 16, 16, color.White))
					items = append(items, similarity.Item{Path: path, Cohort: fmt.Sprint(i), Preview: preview})
				}
				emit(similarity.Event{Complete: pass == 1, Successful: len(items), Total: len(items), Items: items})
			}
			return nil
		}
		explorerMenu(t, v).Action()
		v.settleExplorer()
		loaded := allocated()
		if loaded < baseline+40<<20 {
			t.Fatal("fixture did not retain its map resources")
		}
		if loaded > baseline+64<<20 {
			t.Fatalf("map rebuild retained superseded resources: baseline %.1f MiB, mapped %.1f MiB", float64(baseline)/(1<<20), float64(loaded)/(1<<20))
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		v.settleExplorer()
		after := allocated()
		t.Logf("heap baseline %.1f MiB, current map %.1f MiB, after exit %.1f MiB", float64(baseline)/(1<<20), float64(loaded)/(1<<20), float64(after)/(1<<20))
		if after > baseline+16<<20 {
			t.Fatalf("leaving Explorer retained map memory: baseline %.1f MiB, mapped %.1f MiB, exited %.1f MiB", float64(baseline)/(1<<20), float64(loaded)/(1<<20), float64(after)/(1<<20))
		}
		explorerMenu(t, v).Action()
		v.settleExplorer()
		if len(explorerPiles(v)) != 6 {
			t.Fatal("reopening the released explorer did not rebuild a map")
		}
	})
	t.Run("settings", func(t *testing.T) {
		before := preferences.Load(testApp)
		t.Cleanup(func() { preferences.Save(testApp, before) })
		for _, key := range []string{"similarityFavoriteCache", "similarityAutoUpdate", "similarityAutoFit"} {
			testApp.Preferences().RemoveValue(key)
		}
		labels := []string{"Save analysis for favorites", "Auto-update every 30 images", "Fit new stacks into view"}
		findChecks := func(v *viewer) (fyne.Window, map[string]*widget.Check) {
			v.settingsWin.Show(v.settingsState(), false)
			checks := map[string]*widget.Check{}
			for _, win := range v.app.Driver().AllWindows() {
				if win.Title() != lang.L("Settings") {
					continue
				}
				explorerWalk(win.Content(), func(o fyne.CanvasObject) {
					if c, ok := o.(*widget.Check); ok {
						checks[c.Text] = c
					}
				})
				return win, checks
			}
			t.Fatal("settings window did not open")
			return nil, nil
		}
		v := newTestViewer(t)
		win, checks := findChecks(v)
		defaults := []bool{true, false, true}
		for i, label := range labels {
			c := checks[lang.L(label)]
			if c == nil {
				t.Fatalf("settings is missing %s", label)
			}
			if c.Checked != defaults[i] {
				t.Fatalf("wrong default for %s", label)
			}
			fynetest.Tap(c)
		}
		win.Close()
		preferences.Save(testApp, v.currentPreferences())
		reopened := newTestViewer(t)
		win, checks = findChecks(reopened)
		defer win.Close()
		for i, label := range labels {
			if checks[lang.L(label)].Checked == defaults[i] {
				t.Fatalf("%s was not persisted", label)
			}
		}
	})
	t.Run("manual_controls", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg")
		events := make(chan similarity.Event)
		published := make(chan struct{})
		seen := make(chan similarity.Control, 8)
		v.explorerAnalyze = func(ctx context.Context, _ []string, controls <-chan similarity.Control, emit func(similarity.Event)) error {
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case control := <-controls:
					seen <- control
				case event := <-events:
					emit(event)
					published <- struct{}{}
					if event.Complete {
						return nil
					}
				}
			}
		}
		explorerMenu(t, v).Action()
		var update *widget.Button
		var automatic *widget.Check
		explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
			if b, ok := o.(*widget.Button); ok && b.Text == lang.L("Update map") {
				update = b
			}
			if c, ok := o.(*widget.Check); ok && c.Text == lang.L("Auto-update every 30 images") {
				automatic = c
			}
		})
		if update == nil || automatic == nil {
			t.Fatal("map is missing manual update controls")
		}
		if !update.Disabled() || automatic.Checked {
			t.Fatal("map must begin in manual mode with no data to rebuild")
		}
		receive := func() similarity.Control {
			select {
			case c := <-seen:
				return c
			case <-time.After(testTimeout):
				t.Fatal("map control did not reach analysis")
				return similarity.Control{}
			}
		}
		if c := receive(); c.Automatic || c.Update {
			t.Fatal("analysis did not start in manual mode")
		}
		publish := func(event similarity.Event) { events <- event; <-published; v.explorer.ui.Drain() }
		publish(similarity.Event{Successful: 1, Total: 2, Stage: "encoding"})
		if update.Disabled() {
			t.Fatal("available data cannot be rebuilt")
		}
		fynetest.Tap(update)
		if c := receive(); !c.Update || c.Automatic {
			t.Fatal("button did not request a manual rebuild")
		}
		if !update.Disabled() {
			t.Fatal("pending rebuild allows duplicate requests")
		}
		fynetest.Tap(automatic)
		if c := receive(); !c.Automatic || c.Update {
			t.Fatal("automatic option was not delivered")
		}
		publish(similarity.Event{Successful: 1, Total: 2, Items: []similarity.Item{{Path: v.FileAt(0).Path(), Cohort: "a"}}})
		if !update.Disabled() {
			t.Fatal("up-to-date map can be rebuilt without new data")
		}
		publish(similarity.Event{Successful: 2, Total: 2, Stage: "encoding"})
		if update.Disabled() {
			t.Fatal("new data did not re-enable manual update")
		}
		fynetest.Tap(automatic)
		if c := receive(); c.Automatic || c.Update {
			t.Fatal("automatic updates cannot be disabled")
		}
		publish(similarity.Event{Successful: 2, Total: 2, Complete: true, Items: []similarity.Item{{Path: v.FileAt(0).Path(), Cohort: "a"}, {Path: v.FileAt(1).Path(), Cohort: "a"}}})
		v.settleExplorer()
		if !update.Disabled() {
			t.Fatal("completed map still offers a rebuild")
		}
	})
	t.Run("pending_duplicates", func(t *testing.T) {
		for _, action := range []string{"finish", "cancel", "replace"} {
			t.Run(action, func(t *testing.T) {
				v := newTestViewer(t)
				small := uitest.PatternedJPEGURISize(t, "a.jpg", 1, 64, 48)
				large := uitest.PatternedJPEGURISize(t, "b.jpg", 1, 192, 144)
				unique := uitest.PatternedJPEGURI(t, "c.jpg", 99)
				data, err := os.ReadFile(large.Path())
				if err != nil {
					t.Fatal(err)
				}
				var armed atomic.Bool
				var once sync.Once
				release := make(chan struct{})
				defer once.Do(func() { close(release) })
				held := uitest.ReaderURI(large, func() (io.ReadCloser, error) {
					r := bytes.NewReader(data)
					return uitest.ReadCloser{ReadFunc: func(p []byte) (int, error) {
						if armed.Load() {
							<-release
						}
						return r.Read(p)
					}, CloseFunc: func() error { return nil }}, nil
				})
				dropAndWait(t, v, small, held, unique)
				started := make(chan []string, 1)
				v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
					started <- paths
					emit(similarity.Event{Complete: true, Total: len(paths)})
					return nil
				}
				func() {
					armed.Store(true)
					v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyG})
					v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyD})
					explorerMenu(t, v).Action()
					v.explorer.workers.Wait()
					v.explorer.ui.Drain()
					select {
					case <-started:
						t.Fatal("analysis started before duplicate facts were ready")
					default:
					}
					if !v.explorerMapActive() {
						t.Fatal("waiting for duplicate facts hid the map")
					}
					if action == "cancel" {
						v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
					}
					if action == "replace" {
						v.handleDrop([]fyne.URI{small, unique})
					}
					once.Do(func() { close(release) })
					v.settleExplorer()
				}()
				if action == "replace" {
					waitForScan(t, v)
					waitForSort(t, v)
					waitUntilLoaded(t, v)
					v.settleExplorer()
				}
				select {
				case paths := <-started:
					if action != "finish" {
						t.Fatal("cancelled duplicate preparation started analysis")
					}
					if !slices.Equal(paths, []string{large.Path(), unique.Path()}) {
						t.Fatalf("pending hashes did not elect highest quality: %v", paths)
					}
				default:
					if action == "finish" {
						t.Fatal("finished duplicate preparation did not start analysis")
					}
				}
			})
		}
	})
	t.Run("duplicate_representatives", func(t *testing.T) {
		v := newTestViewer(t)
		small := uitest.PatternedJPEGURISize(t, "a.jpg", 1, 64, 48)
		large := uitest.PatternedJPEGURISize(t, "b.jpg", 1, 192, 144)
		unique := uitest.PatternedJPEGURI(t, "c.jpg", 99)
		dropAndWait(t, v, small, large, unique)
		warmThumbs(t, v)
		v.grid.Toggle()
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyD})
		v.grid.Settle()
		waitUntilLoaded(t, v)
		if !slices.Equal(explorerGridPaths(v), []string{large.Path(), unique.Path()}) {
			t.Fatal("premise: D did not filter the smaller duplicate")
		}
		v.grid.HandleRune('/')
		v.grid.HandleRune('c')
		v.grid.SelectAll()
		var got []string
		v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			got = append([]string(nil), paths...)
			emit(similarity.Event{Complete: true, Total: len(paths)})
			return nil
		}
		explorerMenu(t, v).Action()
		v.settleExplorer()
		if !slices.Equal(got, []string{large.Path(), unique.Path()}) {
			t.Fatalf("filtered analysis must use the highest-resolution representative plus unique sources: %v", got)
		}
		if v.FileCount() != 3 {
			t.Fatal("representative analysis changed the opened file set")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyD})
		explorerMenu(t, v).Action()
		v.settleExplorer()
		if !slices.Equal(got, []string{small.Path(), large.Path(), unique.Path()}) {
			t.Fatal("disabling duplicate filtering did not restore all analysis inputs")
		}
	})
	t.Run("minimum_zoom_inputs", func(t *testing.T) {
		v, publish := explorerLargeFixture(t)
		publish(144, true)
		v.settleExplorer()
		pile := explorerPiles(v)[0]
		// Sampling is observed when the target is on screen; distant piles
		// may release their decoded pixels while retaining their identities.
		viewport := v.explorer.surface.Size()
		v.explorer.surface.Dragged(&fyne.DragEvent{Dragged: fyne.Delta{DX: viewport.Width/2 - pile.Position().X - pile.Size().Width/2, DY: viewport.Height/2 - pile.Position().Y - pile.Size().Height/2}})
		samples := explorerSamples(pile)
		if len(samples) != 15 {
			t.Fatalf("large map must retain all 15 sampled thumbnails, got %d", len(samples))
		}
		for range 12 {
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyPlus})
		}
		for range 40 {
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyMinus})
		}
		if got := pile.Size().Width; math.Abs(float64(got-190)) > .01 {
			t.Fatalf("large-map zoom-out must stop at a 190px pile, got %.2fpx", got)
		}
		pos, size := pile.Position(), pile.Size()
		fynetest.Tap(explorerButton(t, v, "-"))
		v.explorer.surface.Scrolled(&fyne.ScrollEvent{Position: fyne.NewPos(81, 93), Scrolled: fyne.Delta{DY: -1000}})
		if pile.Position() != pos || pile.Size() != size || !slices.Equal(explorerSamples(pile), samples) {
			t.Fatal("zoom-out at the floor changed the camera or sampled thumbnails")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyPlus})
		if pile.Size().Width <= size.Width {
			t.Fatal("zoom-in stopped working at the floor")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyMinus})
		// Bring the sampled cohort onto the screen using the ordinary pan input.
		viewport = v.explorer.surface.Size()
		v.explorer.surface.Dragged(&fyne.DragEvent{Dragged: fyne.Delta{DX: viewport.Width/2 - pile.Position().X - pile.Size().Width/2, DY: viewport.Height/2 - pile.Position().Y - pile.Size().Height/2}})
		pos, size = pile.Position(), pile.Size()
		fynetest.Tap(pile)
		if len(explorerGridPaths(v)) != 16 {
			t.Fatal("large-map pile did not open its complete cohort")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		if pile.Position() != pos || pile.Size() != size || !slices.Equal(explorerSamples(pile), samples) {
			t.Fatal("cohort return lost the camera or full sample set")
		}

		small := explorerFixture(t)
		for range 40 {
			small.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyMinus})
		}
		if got := explorerPiles(small)[0].Size().Width; math.Abs(float64(got-11.4)) > .01 {
			t.Fatalf("small map lost its original 0.03x zoom floor: pile %.2fpx", got)
		}
	})

	t.Run("minimum_zoom_fit", func(t *testing.T) {
		v, publish := explorerLargeFixture(t)
		publish(120, false)
		if got := explorerPiles(v)[0].Size().Width; math.Abs(float64(got-190)) > .01 {
			t.Fatalf("initial large-map fit bypassed the floor: %.2fpx", got)
		}
		v.explorer.surface.Dragged(&fyne.DragEvent{Dragged: fyne.Delta{DX: 23, DY: -17}})
		before := explorerPiles(v)[0].Position()
		publish(144, false)
		pile := explorerPiles(v)[0]
		if math.Abs(float64(pile.Size().Width-190)) > .01 || pile.Position() != before {
			t.Fatalf("discovery at the floor moved the camera: %v -> %v, width %.2f", before, pile.Position(), pile.Size().Width)
		}
		fynetest.Tap(explorerButton(t, v, "Fit map"))
		if got := pile.Size().Width; math.Abs(float64(got-190)) > .01 {
			t.Fatalf("manual large-map fit bypassed the floor: %.2fpx", got)
		}
		// Keyboard navigation must still bring distant piles into view.
		for range 20 {
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
		}
		var selected *explorerui.Pile
		for _, p := range explorerPiles(v) {
			explorerWalk(p, func(o fyne.CanvasObject) {
				if frame, ok := o.(*canvas.Rectangle); ok && frame.StrokeWidth >= 2 {
					selected = p
				}
			})
		}
		if selected == nil {
			t.Fatal("large-map navigation lost selection")
		}
		pos, size, viewport := selected.Position(), selected.Size(), v.explorer.surface.Size()
		if pos.X < 0 || pos.Y < 0 || pos.X+size.Width > viewport.Width || pos.Y+size.Height > viewport.Height {
			t.Fatal("keyboard navigation left the selected large-map pile off screen")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		if !v.grid.Visible() || len(explorerGridPaths(v)) == 0 {
			t.Fatal("keyboard could not open the distant cohort")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})

		var slider *widget.Slider
		explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
			if s, ok := o.(*widget.Slider); ok {
				slider = s
			}
		})
		if slider == nil {
			t.Fatal("missing granularity control")
		}
		slider.SetValue(0)
		for range 40 {
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyMinus})
		}
		if got := explorerPiles(v)[0].Size().Width; math.Abs(float64(got-11.4)) > .01 {
			t.Fatal("reducing a large map to one pile did not release its zoom limit")
		}
		slider.SetValue(100)
		if got := explorerPiles(v)[0].Size().Width; math.Abs(float64(got-190)) > .01 {
			t.Fatal("restoring a large map retained the small-map zoom")
		}
		v.win.Resize(fyne.NewSize(700, 400))
		v.ForceRepaint()
		if explorerPiles(v)[0].Size().Width < 190 {
			t.Fatal("window resize bypassed the large-map zoom floor")
		}
		publish(115, true) // Exactly 100 cohorts, including the 16-image pile.
		v.settleExplorer()
		for range 40 {
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyMinus})
		}
		if got := explorerPiles(v)[0].Size().Width; math.Abs(float64(got-11.4)) > .01 {
			t.Fatalf("100-pile map incorrectly received the large-map zoom floor: %.2fpx", got)
		}
	})

	t.Run("auto_fit_off", func(t *testing.T) {
		v, publish := streamingExplorer(t)
		v.win.Resize(fyne.NewSize(1100, 700))
		prev := v.settingsState()
		next := prev
		next.SimilarityAutoFit = false
		v.ApplySettings(prev, next)
		publish([]string{"a", "a"}, false)
		before := explorerPiles(v)[0].Size()
		publish([]string{"a", "a", "b", "c", "d", "e", "f", "g"}, true)
		if explorerPiles(v)[0].Size() != before {
			t.Fatal("disabled automatic fitting still changed zoom")
		}
		explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
			if b, ok := o.(*widget.Button); ok && b.Text == lang.L("Fit map") {
				fynetest.Tap(b)
			}
		})
		if explorerPiles(v)[0].Size() == before {
			t.Fatal("manual Fit map no longer works with automatic fitting disabled")
		}
		viewport := v.explorer.surface.Size()
		for _, p := range explorerPiles(v) {
			pos, size := p.Position(), p.Size()
			if pos.X < 0 || pos.Y < 0 || pos.X+size.Width > viewport.Width || pos.Y+size.Height > viewport.Height {
				t.Fatal("manual fitting left a stack outside the map")
			}
		}
	})
	t.Run("discovery_expands_view", func(t *testing.T) {
		v, publish := streamingExplorer(t)
		v.win.Resize(fyne.NewSize(1100, 700))
		publish([]string{"a", "a"}, false)
		before := explorerPiles(v)[0].Size()
		publish([]string{"a", "a", "b", "c", "d", "e", "f", "g"}, false)
		piles := explorerPiles(v)
		if len(piles) != 7 {
			t.Fatal("discovery lost newly created piles")
		}
		viewport := v.explorer.surface.Size()
		for _, pile := range piles {
			pos, size := pile.Position(), pile.Size()
			if pos.X < 0 || pos.Y < 0 || pos.X+size.Width > viewport.Width || pos.Y+size.Height > viewport.Height {
				t.Fatalf("new discovery left a pile outside the viewport: %v/%v in %v", pos, size, viewport)
			}
		}
		zoomedOut := piles[0].Size()
		if zoomedOut.Width >= before.Width {
			t.Fatal("map did not zoom out to include new piles")
		}
		publish([]string{"a", "a", "a", "a", "a", "a", "a", "a"}, true)
		if got := explorerPiles(v)[0].Size(); got != zoomedOut {
			t.Fatal("a later grouping automatically zoomed back in")
		}
		v.settleExplorer()
	})
	t.Run("progressive_exploration", func(t *testing.T) {
		v, publish := streamingExplorer(t)
		publish([]string{"a", "a", "a", "a"}, false)
		piles := explorerPiles(v)
		if len(piles) != 1 {
			t.Fatal("partial map is unavailable while analysis remains pending")
		}
		progress := false
		explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
			if label, ok := o.(*widget.Label); ok && label.Text == fmt.Sprintf(lang.L("%d ready, %d failed, %d total"), 4, 0, 8) {
				progress = true
			}
		})
		if !progress {
			t.Fatal("partial analysis counts are missing from the visible surface")
		}
		fynetest.Tap(piles[0])
		if len(explorerGridPaths(v)) != 4 {
			t.Fatal("partial cohort is not browsable")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitUntilLoaded(t, v)
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		publish([]string{"a", "a", "a", "a", "b", "b", "b", "b"}, true)
		v.settleExplorer()
		if len(explorerPiles(v)) != 2 {
			t.Fatal("final discovery did not extend the partial map")
		}
	})
	t.Run("pile_spacing", func(t *testing.T) {
		for name, positions := range map[string][][]float32{
			"coincident":      {{0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}, {0, 0}},
			"cell_boundaries": {{-1, -1}, {-.99, 1}, {.5, -.1}, {1, 1}, {-.5, -1}, {.7, .75}},
		} {
			t.Run(name, func(t *testing.T) {
				v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg", "e.jpg", "f.jpg")
				preview := uitest.EncodeJPEG(t, 32, 24, color.White)
				v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
					var items []similarity.Item
					for i, path := range paths {
						items = append(items, similarity.Item{Path: path, Cohort: fmt.Sprint(i), Position: positions[i], Preview: preview})
					}
					emit(similarity.Event{Complete: true, Items: items})
					return nil
				}
				explorerMenu(t, v).Action()
				v.settleExplorer()
				piles := explorerPiles(v)
				if len(piles) != 6 {
					t.Fatal("missing close/coincident cohorts")
				}
				for i, a := range piles {
					for _, b := range piles[i+1:] {
						dx := float32(math.Abs(float64(a.Position().X-b.Position().X))) - a.Size().Width
						dy := float32(math.Abs(float64(a.Position().Y-b.Position().Y))) - a.Size().Height
						if max(dx, dy) < a.Size().Width*.08 {
							t.Fatalf("piles overlap or lack breathing room: horizontal gap %.1f, vertical gap %.1f", dx, dy)
						}
					}
				}
			})
		}
	})
	t.Run("navigation", func(t *testing.T) {
		v, publish := streamingExplorer(t)
		publish([]string{"a", "a", "b", "b"}, false)
		center := fyne.NewPos(v.win.Canvas().Size().Width/2, v.win.Canvas().Size().Height/2)
		fynetest.Drag(v.win.Canvas(), center, 50, 35)
		fynetest.Scroll(v.win.Canvas(), center, 0, 80)
		before := explorerPiles(v)
		delta := before[1].Position().Subtract(before[0].Position())
		delta.X /= before[0].Size().Width
		delta.Y /= before[0].Size().Height
		publish([]string{"new-a", "new-a", "new-b", "new-b", "new-c", "new-c", "new-c", "new-c"}, false)
		piles := explorerPiles(v)
		if len(piles) != 3 {
			t.Fatal("discovery lost a continuing pile")
		}
		after := piles[1].Position().Subtract(piles[0].Position())
		if math.Abs(float64(after.X/piles[0].Size().Width-delta.X)) > .001 || math.Abs(float64(after.Y/piles[0].Size().Height-delta.Y)) > .001 {
			t.Fatal("discovery moved continuing piles relative to each other")
		}
		position := piles[0].Position()
		fynetest.Drag(v.win.Canvas(), center, 30, 20)
		if piles[0].Position() == position {
			t.Fatal("map stopped responding while further analysis remained held")
		}
	})
	t.Run("fit_uses_window", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg", "e.jpg", "f.jpg")
		v.win.Resize(fyne.NewSize(1100, 700))
		preview := uitest.EncodeJPEG(t, 128, 96, color.White)
		v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			var items []similarity.Item
			for i, path := range paths {
				items = append(items, similarity.Item{Path: path, Cohort: fmt.Sprint(i), Position: []float32{0, float32(i * 100)}, Preview: preview})
			}
			emit(similarity.Event{Complete: true, Items: items})
			return nil
		}
		explorerMenu(t, v).Action()
		v.settleExplorer()
		for _, pile := range explorerPiles(v) {
			if pile.Size().Width < v.win.Canvas().Size().Width*.2 {
				t.Fatalf("a tall projection wastes the wide window: pile width %.1f", pile.Size().Width)
			}
		}
	})
	t.Run("open_cohort", func(t *testing.T) {
		v, publish := streamingExplorer(t)
		publish([]string{"a", "a", "a", "a", "b", "b"}, false)
		fynetest.Tap(explorerPiles(v)[0])
		frozen := explorerGridPaths(v)
		publish([]string{"new-a", "new-a", "new-b", "new-b", "new-b", "new-b", "new-c", "new-c"}, true)
		if !slices.Equal(explorerGridPaths(v), frozen) {
			t.Fatal("regrouping changed the already open cohort")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitUntilLoaded(t, v)
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEnd})
		waitUntilLoaded(t, v)
		current, _, _ := v.CurrentFile()
		if current.Path() != frozen[len(frozen)-1] {
			t.Fatal("regrouping changed image navigation in the frozen cohort")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		fynetest.Tap(explorerPiles(v)[0])
		if got := explorerGridPaths(v); !slices.Equal(got, frozen[:2]) {
			t.Fatal("reopening a pile did not use its current membership")
		}
		v.settleExplorer()
	})
	t.Run("non_overlapping_cohorts", func(t *testing.T) {
		v, publish := streamingExplorer(t)
		publish([]string{"a", "a", "a", "a", "b", "b"}, false)
		fynetest.Tap(explorerPiles(v)[0])
		publish([]string{"new-a", "new-a", "new-b", "new-b", "new-b", "new-b", "new-c", "new-c"}, true)
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		owners := map[string]bool{}
		for _, pile := range explorerPiles(v) {
			fynetest.Tap(pile)
			for _, path := range explorerGridPaths(v) {
				if owners[path] {
					t.Fatal("current map assigns one source to multiple cohorts")
				}
				owners[path] = true
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		}
		if len(owners) != 8 {
			t.Fatal("current map lost members during regrouping")
		}
		v.settleExplorer()
	})
	t.Run("cohort_piles", func(t *testing.T) {
		v := explorerFixture(t)
		if path := os.Getenv("PICFETCH_EXPLORER_QA"); path != "" {
			v.win.Resize(fyne.NewSize(1100, 700))
			v.ForceRepaint()
			v.explorer.surface.Fit()
			f, err := os.Create(path)
			if err != nil {
				t.Fatal(err)
			}
			err = png.Encode(f, v.win.Canvas().Capture())
			closeErr := f.Close()
			if err != nil || closeErr != nil {
				t.Fatalf("render: %v/%v", err, closeErr)
			}
		}
		piles := explorerPiles(v)
		if len(piles) != 2 {
			t.Fatalf("want two cohort piles in the surface, got %d", len(piles))
		}
		for i, pile := range piles {
			samples := map[string]bool{}
			count := 0
			explorerWalk(pile, func(o fyne.CanvasObject) {
				if img, ok := o.(*canvas.Image); ok && img.Resource != nil {
					samples[img.Resource.Name()] = true
					count++
				}
			})
			want := 15
			if i == 1 {
				want = 1
			}
			if count != want || len(samples) != want {
				t.Fatalf("pile %d has %d images/%d distinct sources; want %d", i, count, len(samples), want)
			}
			fynetest.Tap(pile)
			members := explorerGridPaths(v)
			expected := 16
			if i == 1 {
				expected = 1
			}
			if len(members) != expected {
				t.Fatalf("opening pile has %d members; want %d including unsampled", len(members), expected)
			}
			for sample := range samples {
				if !slices.Contains(members, sample) {
					t.Fatal("pile sample is outside cohort")
				}
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		}
		found := false
		explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
			if button, ok := o.(*widget.Button); ok && button.Text == fmt.Sprintf(lang.L("Unassigned (%d)"), 1) {
				found = true
				fynetest.Tap(button)
			}
		})
		if !found || len(explorerGridPaths(v)) != 1 {
			t.Fatal("Unassigned must open its own collection")
		}
	})
	t.Run("merged_source_samples", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
		v.SetMergeMode(true)
		dropAndWait(t, v, v.FileAt(0))
		preview := uitest.EncodeJPEG(t, 32, 24, color.White)
		v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			var items []similarity.Item
			for _, path := range paths {
				items = append(items, similarity.Item{Path: path, Cohort: "a", Position: []float32{0, 0}, Preview: preview})
			}
			emit(similarity.Event{Complete: true, Items: items})
			return nil
		}
		explorerMenu(t, v).Action()
		v.settleExplorer()
		piles := explorerPiles(v)
		if len(piles) != 1 {
			t.Fatal("missing merged cohort")
		}
		samples := map[string]bool{}
		count := 0
		explorerWalk(piles[0], func(o fyne.CanvasObject) {
			if img, ok := o.(*canvas.Image); ok && img.Resource != nil {
				samples[img.Resource.Name()] = true
				count++
			}
		})
		if count != 3 || len(samples) != 3 {
			t.Fatalf("merged source repeated in preview: %d samples / %d distinct", count, len(samples))
		}
		fynetest.Tap(piles[0])
		if len(explorerGridPaths(v)) != 4 || v.FileCount() != 4 {
			t.Fatal("distinct previews must preserve all opened occurrences in Grid View")
		}
	})
	t.Run("scattered_previews", func(t *testing.T) {
		v := explorerFixture(t)
		pile := explorerPiles(v)[0]
		var pictures []*canvas.Image
		explorerWalk(pile, func(o fyne.CanvasObject) {
			if img, ok := o.(*canvas.Image); ok {
				pictures = append(pictures, img)
			}
		})
		for i, a := range pictures {
			if ratio := a.Size().Width / a.Size().Height; math.Abs(float64(ratio-4.0/3)) > .01 {
				t.Fatalf("preview frame reserves unused letterbox space: ratio %.3f, want 4:3", ratio)
			}
			for _, b := range pictures[i+1:] {
				w := max(float32(0), min(a.Position().X+a.Size().Width, b.Position().X+b.Size().Width)-max(a.Position().X, b.Position().X))
				h := max(float32(0), min(a.Position().Y+a.Size().Height, b.Position().Y+b.Size().Height)-max(a.Position().Y, b.Position().Y))
				if w*h > a.Size().Width*a.Size().Height*.7 {
					t.Fatal("two sampled images overlap by more than 70 percent")
				}
			}
		}
		objects := fynetest.WidgetRenderer(pile).Objects()
		for _, img := range pictures {
			for _, obj := range objects {
				border, ok := obj.(*canvas.Rectangle)
				if !ok || border.Position().Y > img.Position().Y || border.Position().X > img.Position().X {
					continue
				}
				if abs := img.Position().X - border.Position().X; abs < img.Size().Width*.04 && img.Position().Y-border.Position().Y < img.Size().Height*.04 {
					if border.Size().Width-img.Size().Width > img.Size().Width*.025 {
						t.Fatal("preview border is too thick")
					}
				}
			}
		}
	})
	t.Run("projection_scale", func(t *testing.T) {
		v := explorerFixtureScale(t, 1000000)
		piles := explorerPiles(v)
		if len(piles) != 2 {
			t.Fatal("missing cohorts")
		}
		for _, pile := range piles {
			if pile.Size().Width < 100 {
				t.Fatalf("projection units made a pile unreadable: %v", pile.Size())
			}
		}
	})
	t.Run("completed_map", func(t *testing.T) {
		v := explorerFixture(t)
		piles := explorerPiles(v)
		if len(piles) != 2 {
			t.Fatal("map not delivered")
		}
		fynetest.Tap(piles[0])
		want := explorerGridPaths(v)
		if len(want) != 16 {
			t.Fatal("cohort missing members")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitUntilLoaded(t, v)
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEnd})
		waitUntilLoaded(t, v)
		current, _, _ := v.CurrentFile()
		if current.Path() != want[len(want)-1] {
			t.Fatal("End escaped the cohort")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
		waitUntilLoaded(t, v)
		current, _, _ = v.CurrentFile()
		if current.Path() != want[0] {
			t.Fatal("next image did not wrap within the cohort")
		}
		if v.FileCount() != 18 {
			t.Fatal("cohort browsing replaced the opened file set")
		}
	})
	t.Run("cold_cohort_size", func(t *testing.T) {
		v := explorerFixture(t)
		v.settings.staticWindowSize = false
		v.win.Resize(fyne.NewSize(1100, 700))
		want := v.win.Canvas().Size()
		fynetest.Tap(explorerPiles(v)[0])
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitUntilLoaded(t, v)
		v.imgCache.Purge()
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEnd})
		waitUntilLoaded(t, v)
		if got := v.win.Canvas().Size(); got != want {
			t.Fatalf("uncached cohort image resized the explorer window: %v -> %v", want, got)
		}
	})
	t.Run("return_to_map", func(t *testing.T) {
		v := explorerFixture(t)
		piles := explorerPiles(v)
		if len(piles) != 2 {
			t.Fatal("map missing")
		}
		before, size := piles[0].Position(), piles[0].Size()
		center := fyne.NewPos(v.win.Canvas().Size().Width/2, v.win.Canvas().Size().Height/2)
		fynetest.Drag(v.win.Canvas(), center, 50, 35)
		fynetest.Scroll(v.win.Canvas(), center, 0, 80)
		moved, scaled := piles[0].Position(), piles[0].Size()
		if moved == before || scaled == size {
			t.Fatalf("canvas drag/scroll did not move and zoom the map: canvas=%v map=%v/%v pile %v/%v -> %v/%v", v.win.Canvas().Size(), v.explorer.surface.Position(), v.explorer.surface.Size(), before, size, moved, scaled)
		}
		fynetest.Tap(piles[0])
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitUntilLoaded(t, v)
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		if !v.grid.Visible() || len(explorerGridPaths(v)) != 16 {
			t.Fatal("Escape did not reopen cohort Grid View")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		returned := explorerPiles(v)
		if v.grid.Visible() || len(returned) != 2 || returned[0].Position() != moved || returned[0].Size() != scaled {
			t.Fatal("return changed the map camera")
		}
	})
	t.Run("shift_pan", func(t *testing.T) {
		v := explorerFixture(t)
		pile := explorerPiles(v)[0]
		before, size := pile.Position(), pile.Size()
		v.keyModifiers = func() fyne.KeyModifier { return fyne.KeyModifierShift }
		center := fyne.NewPos(v.win.Canvas().Size().Width/2, v.win.Canvas().Size().Height/2)
		fynetest.Scroll(v.win.Canvas(), center, 40, 60)
		if pile.Position() == before || pile.Size() != size {
			t.Fatal("Shift-scroll must pan the map without zooming")
		}
		v.keyModifiers = func() fyne.KeyModifier { return 0 }
		fynetest.Scroll(v.win.Canvas(), center, 0, 60)
		if pile.Size() == size {
			t.Fatal("unmodified scrolling must still zoom")
		}
	})

	t.Run("cancellation", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg")
		started := make(chan struct{})
		stopped := make(chan struct{})
		v.explorerAnalyze = func(ctx context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			close(started)
			<-ctx.Done()
			emit(similarity.Event{Complete: true, Total: len(paths), Successful: len(paths)})
			close(stopped)
			return ctx.Err()
		}
		explorerMenu(t, v).Action()
		<-started
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		select {
		case <-stopped:
		case <-time.After(testTimeout):
			t.Fatal("leaving map did not cancel analysis")
		}
		v.settleExplorer()
		if len(explorerPiles(v)) != 0 || v.FileCount() != 2 {
			t.Fatal("canceled work changed the opened viewer")
		}
	})
	t.Run("map_commands", func(t *testing.T) {
		v := explorerFixture(t)
		if !v.menus.Actions().Trash().Disabled || !v.menus.Actions().Copy().Disabled {
			t.Fatal("image actions remain enabled behind the map")
		}
		v.menus.Actions().Trash().Action()
		if v.deletion.Visible() {
			t.Fatal("map command targeted a covered image")
		}
		v.menus.Window().Viewer().Action()
		if len(explorerPiles(v)) != 0 {
			t.Fatal("Window -> Viewer did not leave the map")
		}
	})

	t.Run("opened_files", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
		var got []string
		v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			got = append([]string(nil), paths...)
			emit(similarity.Event{Complete: true, Total: len(paths)})
			return nil
		}
		want := []string{v.FileAt(0).Path(), v.FileAt(1).Path(), v.FileAt(2).Path()}
		v.grid.HandleRune('/')
		v.grid.HandleRune('a')
		v.grid.SelectAll()
		explorerMenu(t, v).Action()
		v.settleExplorer()
		if !slices.Equal(got, want) {
			t.Fatal("analysis used Grid search/selection instead of all opened sources")
		}
		v.SetMergeMode(true)
		extra := uitest.TempJPEGURI(t, "d.jpg", 4, 4, color.White)
		dropAndWait(t, v, extra)
		want = append(want, extra.Path())
		slices.Sort(want)
		explorerMenu(t, v).Action()
		v.settleExplorer()
		slices.Sort(got)
		if !slices.Equal(got, want) {
			t.Fatal("merged sources not included in analysis")
		}
		v.SetMergeMode(false)
		dir := t.TempDir()
		favorite := uitest.TempJPEGURI(t, "favorite.jpg", 4, 4, color.White)
		if err := favstore.Save(dir, "Trip", []fyne.URI{favorite}); err != nil {
			t.Fatal(err)
		}
		v.favorites.SetDir(dir)
		v.favorites.Menu().Items[2].Action()
		waitForScan(t, v)
		waitForSort(t, v)
		waitUntilLoaded(t, v)
		explorerMenu(t, v).Action()
		v.settleExplorer()
		if !slices.Equal(got, []string{favorite.Path()}) {
			t.Fatal("Favorite did not replace the analysis input")
		}
	})
	t.Run("replacement_discards_late_map", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg")
		started := make(chan struct{})
		release := make(chan struct{})
		preview := uitest.EncodeJPEG(t, 32, 24, color.White)
		v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			close(started)
			<-release
			emit(similarity.Event{Complete: true, Items: []similarity.Item{{Path: paths[0], Cohort: "old", Position: []float32{0, 0}, Preview: preview}}})
			return nil
		}
		explorerMenu(t, v).Action()
		<-started
		replacement := uitest.TempJPEGURI(t, "new.jpg", 4, 4, color.White)
		dropAndWait(t, v, replacement)

		delivered := make(chan struct{})
		v.explorerAnalyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
			emit(similarity.Event{Complete: true, Items: []similarity.Item{{Path: paths[0], Cohort: "new", Position: []float32{0, 0}, Preview: preview}}})
			close(delivered)
			return nil
		}
		explorerMenu(t, v).Action()
		<-delivered
		v.explorer.ui.Drain()
		close(release)
		v.settleExplorer()
		piles := explorerPiles(v)
		if len(piles) != 1 {
			t.Fatal("new map disappeared")
		}
		fynetest.Tap(piles[0])
		if !slices.Equal(explorerGridPaths(v), []string{replacement.Path()}) {
			t.Fatal("late analysis overwrote the new map")
		}

	})

}
