package exifwin

import (
	"context"
	"fmt"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/completion"
)

type stripMutation struct {
	generation uint64
	cancel     context.CancelFunc
	source     string
	pending    bool
	workers    sync.WaitGroup
	done       completion.Signal
}

// performStrip captures the confirmed URI and leaves source work to a worker.
// A successful commit still notifies the host after navigation/close; only its
// current panel request owns the busy state and toast.
func (w *Window) performStrip(u fyne.URI) {
	w.pending = nil
	if w.stopped || w.text == nil || w.stripWork.pending {
		return
	}
	w.cancelStrip()
	ctx, cancel := context.WithCancel(context.Background())
	w.stripWork.cancel = cancel
	w.stripWork.pending = true
	w.stripWork.source = u.String()
	generation := w.stripWork.generation
	done := w.stripWork.done.Begin()
	w.syncStripVisible()
	strip := w.stripFile
	w.stripWork.workers.Go(func() {
		result, err := strip(ctx, u)
		if ctx.Err() != nil && !result.Committed {
			cancel()
			done()
			return
		}
		w.ui.Do(func() {
			defer done()
			defer cancel()
			current := ctx.Err() == nil && generation == w.stripWork.generation && !w.stopped && w.text != nil
			if current {
				w.stripWork.pending = false
				w.syncStripVisible()
			}
			if err != nil {
				if current {
					fyne.LogError("failed to remove metadata", err)
					w.host.ShowToast(fmt.Sprintf(lang.L("could not remove metadata from %q: %v"), u.Name(), err))
				}
				return
			}
			if !w.stopped && (current || result.Committed) {
				// Some hosts already refresh through their notification. Keep
				// exactly one fallback read for hosts that do not.
				metadataGeneration := w.metadata.generation
				w.host.AfterMetadataRemoved(u, result)
				if metadataGeneration == w.metadata.generation {
					w.Refresh()
				}
			}
			if current {
				w.host.ShowToast(lang.L("Metadata removed"))
			}
		})
	})
}

func (w *Window) cancelStrip() {
	w.stripWork.generation++
	if w.stripWork.cancel != nil {
		w.stripWork.cancel()
		w.stripWork.cancel = nil
	}
	w.stripWork.pending = false
}

// MutationDone includes the current removal's causal UI delivery. Settle waits
// all removal workers, including superseded or already-committed requests.
func (w *Window) MutationDone() *completion.Signal { return &w.stripWork.done }
