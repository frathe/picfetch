package exifwin

import (
	"context"
	"errors"
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
	w.setRemovalStatus("")
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
	w.setRemovalStatus("")
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
		var inspection imaging.JPEGMetadataInspection
		if err == nil && ctx.Err() == nil {
			metadata = imaging.ReadMetadata(data)
			if ctx.Err() == nil {
				inspection = imaging.InspectJPEGMetadata(ctx, data)
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
			w.canStrip = inspection.State == imaging.JPEGMetadataRemovable
			if inspection.State == imaging.JPEGMetadataClean {
				w.setRemovalStatus(lang.L("Metadata removal: nothing to remove."))
			} else if inspection.Err != nil && !errors.Is(inspection.Err, imaging.ErrJPEGMetadataNotJPEG) {
				w.setRemovalStatus(removalErrorText(inspection.Err))
			}
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

func (w *Window) setRemovalStatus(text string) {
	if w.removalStatus == nil || w.north == nil {
		return
	}
	w.removalStatus.SetText(text)
	if text == "" {
		w.north.Remove(w.removalStatus)
	} else if !northHolds(w.north, w.removalStatus) {
		w.north.Add(w.removalStatus)
	}
}

func removalErrorText(err error) string {
	switch {
	case errors.Is(err, imaging.ErrJPEGMetadataStructure):
		return lang.L("Metadata removal is unavailable: this JPEG is incomplete or invalid.")
	case errors.Is(err, imaging.ErrJPEGMetadataProfile):
		return lang.L("Metadata removal is unavailable for this color profile.")
	case errors.Is(err, imaging.ErrJPEGMetadataOrientation):
		return lang.L("Metadata removal is unavailable because orientation could not be verified.")
	case errors.Is(err, imaging.ErrJPEGMetadataProcess), errors.Is(err, imaging.ErrJPEGMetadataNotJPEG):
		return lang.L("Metadata removal is unavailable for this image's encoding or color model.")
	default:
		return lang.L("Could not process this file. Check access, available space, and the file size limit.")
	}
}
