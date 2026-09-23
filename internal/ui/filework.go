package ui

import (
	"context"
	"errors"
	"image"
	"os"
	"path/filepath"
	"slices"
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
	exportLifecycle requestLifecycle
	searchLifecycle requestLifecycle
	savePending     bool
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
	v.fileWork.searchLifecycle.invalidate()
	v.cancelSave()
	v.cancelExport()
}

func (v *viewer) cancelExport() {
	v.fileWork.exportLifecycle.invalidate()
	v.fileWork.exportPending = false
}

// A terminal worker error can follow an external source mutation. Revalidate
// the captured collection before restoring it, without parsing worker errors.
// The existing file-work lane owns cancellation, completion and UI delivery.
func (v *viewer) reconcileSearchOrigin() {
	if v.fileWork.closed || !v.searchActive() {
		return
	}
	sessionID, generation := v.visualsearch.State().SessionID, v.Generation()
	files := slices.Clone(v.state.files)
	after := v.visualsearch.Suspend()
	token := v.fileWork.searchLifecycle.begin()
	ctx, queue := token.context(), v.fileWork.ui
	v.fileWork.workers.Go(func() {
		select {
		case <-after:
		case <-ctx.Done():
			return
		}
		var missing []int
		for i, source := range files {
			if ctx.Err() != nil {
				return
			}
			if source == nil || source.Scheme() != "file" {
				continue
			}
			if _, err := os.Stat(source.Path()); errors.Is(err, os.ErrNotExist) {
				missing = append(missing, i)
			}
		}
		queue.Do(func() {
			defer token.cancelContext()
			if !token.current() || v.fileWork.closed || generation != v.Generation() || !v.searchActive() || sessionID != v.visualsearch.State().SessionID {
				return
			}
			v.reconcileSources(sourceChange{kind: sourcesRevalidated, removed: missing})
		})
	})
}

// afterFileWrite runs on UI even for obsolete requests that committed. The
// completion stays with the action until its current-view reconciliation lands.
func (v *viewer) afterFileWrite(result imaging.WriteResult, reload, refreshEXIF bool, done func()) {
	if !result.Committed || v.fileWork.closed {
		done()
		return
	}
	files := v.state.snapshot()
	ctx := v.heicContext(v.fileWork.ctx)
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
			v.reconcileSources(sourceChange{kind: sourceWritten})
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
	revision := v.display.RequestRevision()
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
		metadata, metadataErr := imaging.ReadMetadataContext(ctx, data)
		if infoErr == nil {
			infoErr = metadataErr
		}
		hasEXIF := !metadata.Empty()
		if ctx.Err() != nil {
			done()
			return
		}
		v.fileWork.ui.Do(func() {
			if v.fileWork.closed || revision != v.display.RequestRevision() {
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
