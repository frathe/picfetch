package exifwin

import (
	"context"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/completion"
	"github.com/frathe/picfetch/internal/imaging"
)

// metadataRead belongs to the panel, independently of map-warm generations.
// Only workers and captured contexts cross the UI boundary.
type metadataRead struct {
	generation uint64
	cancel     context.CancelFunc
	workers    sync.WaitGroup
	done       completion.Signal
}

// Invalidate cancels reads and clears their presentation when the viewer starts
// changing the displayed source. Refresh admits a read once new pixels arrive.
func (w *Window) Invalidate() {
	w.cancelStrip()
	w.cancelMetadata()
	w.hideConfirm()
	w.pending = nil
	if w.text == nil {
		return
	}
	w.text.SetText("")
	w.canStrip = false
	w.syncStripVisible()
	w.showLocation(imaging.Metadata{})
}

// Refresh captures the displayed source and reads its metadata off UI. Old
// text, location and strip availability disappear immediately; a current
// result installs through the panel's UI queue. A closed panel does no work.
func (w *Window) Refresh() {
	if w.stopped || w.text == nil {
		return
	}
	w.cancelMetadata()
	w.dismissStalePending()
	w.canStrip = false
	w.syncStripVisible()
	w.showLocation(imaging.Metadata{})
	u, ok := w.host.DisplayedFile()
	if w.stripWork.pending && (!ok || u == nil || u.String() != w.stripWork.source) {
		w.cancelStrip()
	}
	if !ok || u == nil {
		w.text.SetText("")
		return
	}
	w.text.SetText(lang.L("Loading..."))
	ctx, cancel := context.WithCancel(context.Background())
	w.metadata.cancel = cancel
	generation := w.metadata.generation
	done := w.metadata.done.Begin()
	w.metadata.workers.Go(func() {
		data, _, err := imaging.ReadAndProbe(ctx, u)
		var metadata imaging.Metadata
		var canStrip bool
		if err == nil && ctx.Err() == nil {
			metadata = imaging.ReadMetadata(data)
			if ctx.Err() == nil {
				canStrip = imaging.CanStripJPEGMetadata(data) && !metadata.Empty()
			}
		}
		if ctx.Err() != nil {
			cancel()
			done()
			return
		}
		w.ui.Do(func() {
			defer done()
			defer cancel()
			if ctx.Err() != nil || generation != w.metadata.generation || w.stopped || w.text == nil {
				return
			}
			if err != nil {
				fyne.LogError("failed to read image metadata", err)
				w.text.SetText(lang.L("Could not read this file's metadata."))
				return
			}
			w.text.SetText(formatExifMetadata(metadata))
			w.showLocation(metadata)
			w.canStrip = canStrip
			w.syncStripVisible()
		})
	})
}

func (w *Window) cancelMetadata() {
	w.metadata.generation++
	if w.metadata.cancel != nil {
		w.metadata.cancel()
		w.metadata.cancel = nil
	}
}

// MetadataDone names the current metadata read through UI result delivery.
// Settle waits all reads (including superseded ones) and drains that delivery.
func (w *Window) MetadataDone() *completion.Signal { return &w.metadata.done }
