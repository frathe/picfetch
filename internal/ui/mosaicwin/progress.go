package mosaicwin

import (
	"context"
	"sync"

	"github.com/frathe/picfetch/internal/mosaic"
)

// progressReporter belongs to one generation. Its synchronous submissions stay
// inside that generation's tracked worker, and a held UI queue retains at most
// one progress callback. No old session can update a reopened window.
func (w *Window) progressReporter(ctx context.Context, revision uint64) func(mosaic.Progress) {
	var mu sync.Mutex
	latest := 0.0
	pending := false
	queue := w.ui
	return func(progress mosaic.Progress) {
		if ctx.Err() != nil || progress.TotalPixels <= 0 {
			return
		}
		value := min(1, max(0, float64(progress.CoveredPixels)/float64(progress.TotalPixels)))
		mu.Lock()
		if value <= latest {
			mu.Unlock()
			return
		}
		latest = value
		if pending {
			mu.Unlock()
			return
		}
		pending = true
		mu.Unlock()
		queue.Do(func() {
			mu.Lock()
			value := latest
			pending = false
			mu.Unlock()
			if !w.lifecycle.current(revision) || !w.win.Open() || !w.generationBusy {
				return
			}
			w.loading.SetValue(value)
		})
	}
}
