package mosaicwin

import (
	"fmt"
	"image"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/storage"

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

	w.workers.Go(func() {
		out, err := filepicker.ChooseSave(suggested)
		if ctx.Err() != nil {
			return
		}
		cancelled := false
		name := ""
		if err == nil {
			picked := filepicker.ParseFileList(out)
			if len(picked) == 0 {
				cancelled = true
			} else {
				destination := mosaicExportDestination(picked[0], format)
				name = destination.Name()
				err = w.exporter(destination, pixels, nil, imaging.ExportOptions{})
			}
		}
		if err != nil {
			fyne.LogError("failed to export mosaic", err)
		}
		w.ui.Do(func() {
			if !w.actionLifecycle.current(revision) || !w.Opened() {
				return
			}
			w.actionBusy = false
			switch {
			case cancelled:
				w.setStatus("")
			case err != nil:
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

func mosaicExportDestination(picked fyne.URI, format ExportFormat) fyne.URI {
	switch strings.ToLower(picked.Extension()) {
	case ".png", ".jpg", ".jpeg":
		return picked
	default:
		return storage.NewFileURI(picked.Path() + string(format))
	}
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

func (w *Window) SetExporter(exporter func(fyne.URI, image.Image, fyne.URI, imaging.ExportOptions) error) {
	if exporter == nil {
		w.exporter = imaging.Export
		return
	}
	w.exporter = exporter
}
