package favorites

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/favstore"
)

// enqueueMutation preserves admission order even when a queued request is
// cancelled. Native work may outlive cancellation; its successor still waits.
func (f *Feature) enqueueMutation(ctx context.Context, work func() func()) {
	if f.stopped {
		return
	}
	previous, done, queue := f.mutationTail, make(chan struct{}), f.ui
	f.mutationTail = done
	f.workers.Go(func() {
		defer close(done)
		if previous != nil {
			<-previous
		}
		if ctx.Err() != nil {
			return
		}
		if deliver := work(); deliver != nil {
			queue.Do(deliver)
		}
	})
}

type saveRequest struct {
	ctx       context.Context
	dir, name string
	files     []fyne.URI
	revision  uint64
	storage   Storage
}

func (f *Feature) saveCurrent(request saveRequest) bool {
	return !f.stopped && request.ctx.Err() == nil && request.revision == f.saveRevision
}

func (f *Feature) writeFavorite(name string) { f.saveFavorite(name) }

func (f *Feature) saveFavorite(name string) {
	if f.stopped {
		return
	}
	name = strings.TrimSpace(name)
	if !favstore.ValidName(name) {
		f.host.ShowToast(lang.L(`enter a name without / \ : * ? " < > |`))
		return
	}
	files := f.addFiles
	if files == nil {
		files = f.host.CurrentFiles()
	}
	if len(files) == 0 {
		f.host.ShowToast(lang.L("there are no open files to add to favorites"))
		return
	}
	f.saveRevision++
	request := saveRequest{ctx: f.viewContext(), dir: f.dir, name: name, files: slices.Clone(files), revision: f.saveRevision, storage: f.storage}
	f.prepareSave(request, false)
}

func (f *Feature) prepareSave(request saveRequest, conflict bool) {
	f.enqueueMutation(request.ctx, func() func() {
		target, err := request.storage.Capture(request.ctx, request.dir, request.name)
		if err != nil {
			return f.saveDelivery(request, favstore.SaveResult{}, err)
		}
		if !target.Occupied() && !conflict {
			result, err := request.storage.Save(request.ctx, target, request.files)
			return f.saveDelivery(request, result, err)
		}
		return func() {
			if !f.saveCurrent(request) || !f.host.AdmitFavorite(AddCommand) {
				return
			}
			reopen := func() {
				if !f.saveCurrent(request) {
					return
				}
				f.addFiles = slices.Clone(request.files)
				f.showAdd(request.name)
			}
			if !target.Occupied() {
				reopen()
				return
			}
			f.showConfirm(confirmation{
				title:   lang.L("Replace Favorite"),
				message: fmt.Sprintf(lang.L("A favorite named %q already exists. Replace it?"), request.name),
				action:  lang.L("Replace"),
				onConfirm: func() {
					if !f.saveCurrent(request) {
						return
					}
					f.enqueueMutation(request.ctx, func() func() {
						result, err := request.storage.Save(request.ctx, target, request.files)
						return f.saveDelivery(request, result, err)
					})
				},
				onCancel: reopen,
				onClosed: func() { f.win.Canvas().Unfocus() },
			})
		}
	})
}

// saveDelivery separates authoritative disk effects from view-bound presentation.
func (f *Feature) saveDelivery(request saveRequest, result favstore.SaveResult, err error) func() {
	return func() {
		if f.stopped {
			return
		}
		if result.Committed {
			if f.onSaved != nil {
				f.onSaved()
			}
			f.host.SyncFavoritePreviews(result.Definition.Owner, request.files)
			f.refreshMenu()
			if f.saveCurrent(request) {
				f.addFiles = nil
				f.host.ShowToast(fmt.Sprintf(lang.L("saved favorite %q"), request.name))
			}
			return
		}
		if !f.saveCurrent(request) {
			return
		}
		if errors.Is(err, favstore.ErrConflict) {
			f.prepareSave(request, true)
			return
		}
		if err != nil {
			f.reportError(lang.L("could not save favorite %q: %v"), request.name, err)
		}
	}
}
