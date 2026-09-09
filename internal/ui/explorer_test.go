package ui

import (
	"context"
	"fmt"
	"image/color"
	"image/png"
	"os"
	"slices"
	"testing"
	"time"

	"fyne.io/fyne/v2/canvas"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	explorerui "github.com/frathe/picfetch/internal/ui/explorer"
	"github.com/frathe/picfetch/internal/uitest"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/favstore"
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
	v.explorerAnalyze = func(_ context.Context, paths []string, emit func(similarity.Event)) error {
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
func TestVisualSimilarityExplorer(t *testing.T) {
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
		v.explorerAnalyze = func(_ context.Context, paths []string, emit func(similarity.Event)) error {
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

	t.Run("cancellation", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg")
		started := make(chan struct{})
		stopped := make(chan struct{})
		v.explorerAnalyze = func(ctx context.Context, paths []string, emit func(similarity.Event)) error {
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
		v.explorerAnalyze = func(_ context.Context, paths []string, emit func(similarity.Event)) error {
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
		v.explorerAnalyze = func(_ context.Context, paths []string, emit func(similarity.Event)) error {
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
		v.explorerAnalyze = func(_ context.Context, paths []string, emit func(similarity.Event)) error {
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
