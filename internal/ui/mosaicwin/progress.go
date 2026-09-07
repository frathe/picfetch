package mosaicwin

import (
	"context"
	"image"
	"sync"

	"github.com/frathe/picfetch/internal/mosaic"
)

// progressReporter belongs to one generation. Its synchronous submissions stay
// inside that generation's tracked worker, and a held UI queue retains at most
// one progress callback and its latest preview. No old session can update a
// reopened window; published snapshots are read-only once handed to this queue.
func (w *Window) progressReporter(ctx context.Context, revision uint64) func(mosaic.Progress) {
	var mu sync.Mutex
	latest := 0.0
	var latestPreview image.Image
	pending := false
	queue := w.ui
	return func(progress mosaic.Progress) {
		if ctx.Err() != nil || progress.TotalPixels <= 0 {
			return
		}
		value := min(1, max(0, float64(progress.CoveredPixels)/float64(progress.TotalPixels)))
		mu.Lock()
		if value < latest || value == latest && progress.Preview == nil {
			mu.Unlock()
			return
		}
		latest = value
		if progress.Preview != nil {
			latestPreview = progress.Preview
		}
		if pending {
			mu.Unlock()
			return
		}
		pending = true
		mu.Unlock()
		queue.Do(func() {
			mu.Lock()
			value := latest
			preview := latestPreview
			latestPreview = nil
			pending = false
			mu.Unlock()
			if !w.lifecycle.current(revision) || !w.win.Open() || !w.generationBusy {
				return
			}
			w.loading.SetValue(value)
			if preview != nil {
				w.preview.Image = preview
				w.preview.Refresh()
				if !w.previewPanel.Visible() {
					w.config.Hide()
					w.previewPanel.Show()
					w.root.Refresh()
					w.win.Window().Canvas().Focus(w.previewCancelButton)
				}
			}
		})
	}
}

// restoreFinishedPreview discards any partial canvas without copying the
// full-size result again. It is called only on UI at generation boundaries.
func (w *Window) restoreFinishedPreview() {
	w.preview.Image = w.finishedPreview
	w.preview.Refresh()
	if w.hasResult {
		w.config.Hide()
		w.previewPanel.Show()
	} else {
		w.previewPanel.Hide()
		w.config.Show()
	}
}

func (w *Window) focusAfterGeneration() {
	if w.hasResult {
		w.win.Window().Canvas().Focus(w.startOverButton)
	} else {
		w.win.Window().Canvas().Focus(w.displaySelect)
	}
}
