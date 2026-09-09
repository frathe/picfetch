//go:build explorertrial

package ui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/similarity"
)

// The real worker enforces outbound denial before it reads image pixels. This
// explicit suite fails on absent local assets; it never silently substitutes a
// provider or skips the native integration.
func TestVisualSimilarityExplorerLocal(t *testing.T) {
	t.Run("offline", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg", "e.jpg", "f.jpg")
		assets, err := filepath.Abs("../../.scratch/visual-similarity-explorer/assets")
		if err != nil {
			t.Fatal(err)
		}
		client := similarity.Client{Assets: assets}
		var result similarity.Event
		v.explorerAnalyze = func(ctx context.Context, paths []string, emit func(similarity.Event)) error {
			return client.Analyze(ctx, paths, func(event similarity.Event) {
				if event.Complete {
					result = event
				}
				emit(event)
			})
		}
		explorerMenu(t, v).Action()
		v.settleExplorer()
		if !result.Complete || !result.OfflineVerified || result.Successful != 6 || len(result.Items) != 6 {
			t.Fatalf("real offline engine did not deliver all six inputs: complete=%v denied=%v ready=%d", result.Complete, result.OfflineVerified, result.Successful)
		}
		for _, item := range result.Items {
			if len(item.Embedding) != 768 || len(item.Position) != 2 || item.Cohort == "" {
				t.Fatal("real result has no representation or map assignment")
			}
		}
		piles := explorerPiles(v)
		if len(piles) > 0 {
			fynetest.Tap(piles[0])
		} else {
			explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
				if b, ok := o.(*widget.Button); ok && b.Text == fmt.Sprintf(lang.L("Unassigned (%d)"), 6) {
					fynetest.Tap(b)
				}
			})
		}
		if !v.grid.Visible() || len(explorerGridPaths(v)) == 0 {
			t.Fatal("real completed map cannot open a cohort")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitUntilLoaded(t, v)
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		if !v.grid.Visible() {
			t.Fatal("real cohort browsing did not return to Grid View")
		}
	})
	t.Run("failed_source", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
		missing := v.FileAt(2).Path()
		if err := os.Remove(missing); err != nil {
			t.Fatal(err)
		}
		assets, err := filepath.Abs("../../.scratch/visual-similarity-explorer/assets")
		if err != nil {
			t.Fatal(err)
		}
		var result similarity.Event
		err = (similarity.Client{Assets: assets}).Analyze(context.Background(), []string{v.FileAt(0).Path(), missing}, func(e similarity.Event) {
			if e.Complete {
				result = e
			}
		})
		if err != nil {
			t.Fatal(err)
		}
		if result.Successful != 1 || result.Failed != 1 || len(result.Items) != 2 || result.Items[1].Error == "" || len(result.Items[1].Embedding) != 0 {
			t.Fatal("missing source was not accounted separately")
		}
	})
	t.Run("cancel_worker", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg", "e.jpg", "f.jpg")
		assets, err := filepath.Abs("../../.scratch/visual-similarity-explorer/assets")
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		paths := make([]string, 0, 64)
		for range 64 {
			paths = append(paths, v.FileAt(0).Path())
		}
		completed := false
		progress := false
		err = (similarity.Client{Assets: assets}).Analyze(ctx, paths, func(e similarity.Event) {
			completed = completed || e.Complete
			if e.Successful > 0 {
				progress = true
				cancel()
			}
		})
		if !progress || completed || !errors.Is(err, context.Canceled) {
			t.Fatalf("real worker did not stop after progress: progress=%v complete=%v err=%v", progress, completed, err)
		}
	})
}
