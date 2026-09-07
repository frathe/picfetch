package mosaicwin

import (
	"context"
	"fmt"
	"image"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/filepicker"
	"github.com/frathe/picfetch/internal/imaging"
)

type ExportFormat string

const (
	ExportPNG  ExportFormat = ".png"
	ExportJPEG ExportFormat = ".jpg"
)

// SaveImage captures the current result before starting native picker and
// encode work. Later regeneration cannot retarget this export.
func (w *Window) SaveImage() {
	if !w.PreviewActionsEnabled() || len(w.snapshot.Sources) == 0 {
		return
	}
	pixels := w.result.Image()
	format := w.exportFormat
	clock := w.clock()
	suggested := suggestedMosaicPath(w.snapshot.Sources[0], clock, format)
	ctx, revision := w.actionLifecycle.begin()
	w.actionBusy = true
	w.setStatus(lang.L("Saving mosaic..."))
	w.syncActions()

	exporter, choose := w.exporter, filepicker.ChooseSave
	w.workers.Go(func() {
		destination, err := choose(suggested)
		if ctx.Err() != nil {
			return
		}
		var result imaging.WriteResult
		cancelled := false
		name := ""
		if err == nil {
			if destination == nil {
				cancelled = true
			} else {
				name = destination.Name()
				result, err = exporter(ctx, destination, pixels, nil, imaging.ExportOptions{FallbackExt: string(format)})
			}
		}
		if ctx.Err() != nil && !result.Committed {
			return
		}
		w.ui.Do(func() {
			if result.Committed {
				w.host.AfterFileExported(result)
			}
			if !w.actionLifecycle.current(revision) || !w.Opened() {
				return
			}
			w.actionBusy = false
			switch {
			case cancelled:
				w.setStatus("")
			case err != nil:
				fyne.LogError("failed to export mosaic", err)
				w.setStatus(fmt.Sprintf(lang.L("Could not save mosaic: %v"), err))
			default:
				w.setStatus(fmt.Sprintf(lang.L("Saved mosaic as %q"), name))
			}
			w.syncActions()
		})
	})
}

func suggestedMosaicPath(source fyne.URI, now time.Time, format ExportFormat) string {
	name := "PicFetch-Mosaic-" + now.Format("20060102-150405") + string(format)

	return filepath.Join(filepath.Dir(source.Path()), name)
}

func (w *Window) SetExportFormat(format ExportFormat) {
	if format == ExportJPEG {
		w.exportFormat = ExportJPEG
		if w.formatSelect != nil {
			w.formatSelect.SetSelected(lang.L("JPEG"))
		}
		return
	}
	w.exportFormat = ExportPNG
	if w.formatSelect != nil {
		w.formatSelect.SetSelected(lang.L("PNG"))
	}
}

func (w *Window) SetClock(clock func() time.Time) {
	if clock == nil {
		w.clock = time.Now
		return
	}
	w.clock = clock
}

func (w *Window) SetExporter(exporter func(context.Context, fyne.URI, image.Image, fyne.URI, imaging.ExportOptions) (imaging.WriteResult, error)) {
	if exporter == nil {
		w.exporter = imaging.ExportContext
		return
	}
	w.exporter = exporter
}
