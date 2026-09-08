package ui

import (
	"context"
	"image"
	"os"
	"path/filepath"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/completion"
	"github.com/frathe/picfetch/internal/dupes"
	"github.com/frathe/picfetch/internal/imaging"
)

// fileMutationWork owns viewer file actions; imaging owns path transactions.
// Admission and pending state belong to UI, while workers capture their inputs.
type fileMutationWork struct {
	saveLifecycle   requestLifecycle
	saveDone        completion.Signal
	savePending     bool
	exportLifecycle requestLifecycle
	exportPending   bool
	closed          bool
	ctx             context.Context
	cancel          context.CancelFunc
	workers         sync.WaitGroup
	ui              fileUIQueue
	save            func(context.Context, fyne.URI, image.Image) (imaging.WriteResult, error)
	export          func(context.Context, fyne.URI, image.Image, fyne.URI, imaging.ExportOptions) (imaging.WriteResult, error)
}

type fileUIQueue interface {
	Do(func())
	Drain() bool
}

type fyneFileQueue struct{}

func (fyneFileQueue) Do(fn func()) { fyne.Do(fn) }
func (fyneFileQueue) Drain() bool  { return false }

func newFileMutationWork() fileMutationWork {
	ctx, cancel := context.WithCancel(context.Background())
	return fileMutationWork{ctx: ctx, cancel: cancel, ui: fyneFileQueue{}, save: imaging.SaveRotatedContext, export: imaging.ExportContext}
}

func (v *viewer) cancelSave() {
	v.fileWork.saveLifecycle.invalidate()
	v.fileWork.savePending = false
}

func (v *viewer) closeFileWork() {
	v.fileWork.closed = true
	v.fileWork.cancel()
	v.cancelSave()
	v.cancelExport()
}

func (v *viewer) cancelExport() {
	v.fileWork.exportLifecycle.invalidate()
	v.fileWork.exportPending = false
}

// afterFileWrite runs on UI even for obsolete requests that committed. The
// completion stays with the action until its current-view reconciliation lands.
func (v *viewer) afterFileWrite(result imaging.WriteResult, reload, refreshEXIF bool, done func()) {
	if !result.Committed || v.fileWork.closed {
		done()
		return
	}
	files := v.state.snapshot()
	ctx := v.fileWork.ctx
	v.fileWork.workers.Go(func() {
		affected := writtenFileLoaded(ctx, result.Path, files)
		if ctx.Err() != nil {
			done()
			return
		}
		v.fileWork.ui.Do(func() {
			if v.fileWork.closed {
				done()
				return
			}
			if files.Generation() != v.Generation() {
				v.afterFileWrite(result, reload, refreshEXIF, done)
				return
			}
			if !affected {
				done()
				return
			}
			v.favThumbLifecycle.invalidate()
			v.grid.InvalidateContent()
			v.compare.Refresh()
			v.refreshWrittenFile(result, reload, refreshEXIF, done)
		})
	})
}

// Exporting a new copy leaves loaded sources unchanged. Resolve aliases on a
// worker before discarding their derived state, including noncurrent sources.
func writtenFileLoaded(ctx context.Context, path string, files dupes.Snapshot) bool {
	if ctx.Err() != nil {
		return false
	}
	if files.IndexOf(storage.NewFileURI(path).String()) >= 0 {
		return true
	}
	written, err := os.Stat(path)
	if err != nil {
		return false
	}
	for i := range files.Count() {
		if ctx.Err() != nil {
			return false
		}
		u, err := storage.ParseURI(files.KeyAt(i))
		if err != nil || u.Scheme() != "file" {
			continue
		}
		if source, err := os.Stat(u.Path()); err == nil && os.SameFile(source, written) {
			return true
		}
	}
	return false
}

// A separate commit may invalidate the cache while this decision is queued.
// Retry its current-file read without repeating global invalidation.
func (v *viewer) refreshWrittenFile(result imaging.WriteResult, reload, refreshEXIF bool, done func()) {
	u, index, ok := v.CurrentFile()
	if !ok {
		done()
		return
	}
	revision := v.loadLifecycle.currentRevision()
	ctx := v.fileWork.ctx
	writer := v.imgCache.Capture()
	v.fileWork.workers.Go(func() {
		if ctx.Err() != nil {
			done()
			return
		}
		path, err := filepath.Abs(u.Path())
		if err == nil {
			path, err = filepath.EvalSymlinks(path)
		}
		if err == nil && path != result.Path {
			current, currentErr := os.Stat(path)
			written, writtenErr := os.Stat(result.Path)
			if currentErr != nil || writtenErr != nil || !os.SameFile(current, written) {
				done()
				return
			}
		}
		if err != nil || ctx.Err() != nil {
			done()
			return
		}
		var data []byte
		var infoErr error
		if !reload {
			data, _, infoErr = imaging.ReadAndProbe(ctx, storage.NewFileURI(path))
		}
		hasEXIF := !imaging.ReadMetadata(data).Empty()
		if ctx.Err() != nil {
			done()
			return
		}
		v.fileWork.ui.Do(func() {
			if v.fileWork.closed || revision != v.loadLifecycle.currentRevision() {
				done()
				return
			}
			if !writer.Current() {
				v.refreshWrittenFile(result, reload, refreshEXIF, done)
				return
			}
			defer done()
			if reload {
				v.ShowImage(index)
				return
			}
			if infoErr == nil {
				v.info.SetFile(int64(len(data)), hasEXIF, false)
				v.syncInfoOverlayVisibility()
				v.updateInfoOverlay()
			}
			if refreshEXIF {
				v.exif.Refresh()
			}
		})
	})
}
