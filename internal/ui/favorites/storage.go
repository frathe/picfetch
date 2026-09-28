package favorites

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/favstore"
)

// Storage is the read boundary owned by Favorite storage workers.
type Storage interface {
	List(context.Context, string) ([]favstore.Entry, error)
	Open(context.Context, string) (favstore.Definition, error)
}

// UIQueue marshals storage completions onto UI. Tests use a drainable queue.
type UIQueue interface {
	Do(func())
	Drain() bool
}

type fyneQueue struct{}

func (fyneQueue) Do(f func()) { fyne.Do(f) }
func (fyneQueue) Drain() bool { return false }

// SetUIQueue configures delivery before any storage operation begins.
func (f *Feature) SetUIQueue(queue UIQueue) {
	if queue == nil {
		queue = fyneQueue{}
	}
	f.ui = queue
}

// SetStorage configures a feature-local storage provider before starting work.
func (f *Feature) SetStorage(storage Storage) {
	if storage == nil {
		storage = &favstore.Store{}
	}
	f.storage = storage
}

// Wait joins current and retired storage workers off UI.
func (f *Feature) Wait() { f.workers.Wait() }

// Settle joins workers and drains test delivery until no completion starts work.
func (f *Feature) Settle() {
	for {
		f.Wait()
		if !f.ui.Drain() {
			return
		}
	}
}

func (f *Feature) viewContext() context.Context {
	if f.viewCtx == nil {
		f.viewCtx, f.viewCancel = context.WithCancel(context.Background())
	}
	return f.viewCtx
}

// CancelOpen retires a pending open when the host changes its source scope.
func (f *Feature) CancelOpen() {
	if f.openCancel != nil {
		f.openCancel()
		f.openCancel = nil
	}
}

// Close cancels view-bound storage and dialogs without waiting on UI. A later
// action may start a fresh view; committed disk effects remain independently live.
func (f *Feature) Close() {
	f.CancelOpen()
	if f.viewCancel != nil {
		f.viewCancel()
		f.viewCtx, f.viewCancel = nil, nil
	}
	if f.refreshCancel != nil {
		f.refreshCancel()
	}
	f.refreshRevision++
	f.refreshPending, f.manageRequested = false, false
	f.hideManage()
	if f.addDialog != nil {
		f.addDialog.Hide()
	}
	if f.confirmDialog != nil {
		f.confirmDialog.Hide()
	}
	f.addFiles = nil
}

// Stop ends admission permanently; Wait/Settle join active native calls off UI.
func (f *Feature) Stop() {
	f.stopped = true
	f.Close()
}

func (f *Feature) refreshMenu() bool {
	if f.stopped {
		return false
	}
	f.refreshRevision++
	f.refreshPending = true
	if f.refreshCancel != nil {
		f.refreshCancel()
	}
	f.startRefresh()
	return true
}

func (f *Feature) startRefresh() {
	if f.stopped || f.refreshRunning || !f.refreshPending {
		return
	}
	f.refreshRunning, f.refreshPending = true, false
	ctx, cancel := context.WithCancel(f.viewContext())
	f.refreshCancel = cancel
	revision, dir, storage, queue := f.refreshRevision, f.dir, f.storage, f.ui
	f.workers.Go(func() {
		entries, err := storage.List(ctx, dir)
		queue.Do(func() {
			f.refreshRunning = false
			f.refreshCancel = nil
			if !f.stopped && ctx.Err() == nil && revision == f.refreshRevision {
				if err != nil {
					f.manageRequested = false
					f.reportError(lang.L("could not list favorites: %v"), err)
				} else {
					f.installMenu(entries)
					if f.managePanel != nil {
						f.rebuildManage()
					} else if f.manageRequested && f.host.AdmitFavorite(ManageCommand) {
						f.buildManage()
					}
					f.manageRequested = false
				}
			}
			cancel()
			f.startRefresh()
		})
	})
}

func (f *Feature) openFavorite(name string) {
	if f.stopped || !favstore.ValidName(name) || !f.host.AdmitFavorite(OpenCommand) {
		return
	}
	f.CancelOpen()
	ctx, cancel := context.WithCancel(f.viewContext())
	f.openCancel = cancel
	dir, storage, queue := favstore.Dir(f.dir, name), f.storage, f.ui
	f.workers.Go(func() {
		definition, err := storage.Open(ctx, dir)
		var files []fyne.URI
		if err == nil {
			// The provider may have returned a held result. Validate again at
			// publication, without putting filesystem calls on the UI queue.
			var access *favstore.Access
			access, err = definition.Owner.Acquire(ctx)
			if err == nil {
				files = definition.Files()
				err = access.Current(ctx)
				_ = access.Close()
			}
		}
		queue.Do(func() {
			defer cancel()
			if f.stopped || ctx.Err() != nil || !f.host.AdmitFavorite(OpenCommand) {
				return
			}
			if err != nil {
				f.reportError(lang.L("could not open favorite %q: %v"), name, err)
				return
			}
			f.host.OpenFavorite(definition.Owner, files)
		})
	})
}
