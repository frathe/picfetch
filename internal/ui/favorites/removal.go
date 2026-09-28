package favorites

import (
	"context"
	"errors"
	"fmt"
	"os"

	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/favstore"
)

type removalRequest struct {
	ctx       context.Context
	dir, name string
	storage   Storage
}

func (f *Feature) cancelRemove() {
	if f.removeCancel != nil {
		f.removeCancel()
		f.removeCancel = nil
	}
	if f.removeDialog != nil {
		f.removeDialog.Hide()
		f.removeDialog = nil
	}
}

func (f *Feature) beginRemove(name string, confirm bool) {
	if f.stopped {
		return
	}
	f.cancelRemove()
	ctx, cancel := context.WithCancel(f.viewContext())
	f.removeCancel = cancel
	f.prepareRemove(removalRequest{ctx: ctx, dir: f.dir, name: name, storage: f.storage}, confirm)
}

func (f *Feature) prepareRemove(request removalRequest, confirm bool) {
	f.enqueueMutation(request.ctx, func() func() {
		target, err := request.storage.Capture(request.ctx, request.dir, request.name)
		if err != nil {
			return f.removeDelivery(request, favstore.RemovalResult{}, err)
		}
		if !target.Occupied() {
			return f.removeDelivery(request, favstore.RemovalResult{}, os.ErrNotExist)
		}
		if !confirm {
			result, err := request.storage.Remove(request.ctx, target)
			return f.removeDelivery(request, result, err)
		}
		return func() {
			if f.stopped || request.ctx.Err() != nil {
				return
			}
			f.removeDialog = f.showConfirm(confirmation{
				title:   lang.L("Remove Favorite"),
				message: fmt.Sprintf(lang.L("Remove %q from favorites?"), request.name),
				action:  lang.L("Remove"), importance: widget.DangerImportance,
				onConfirm: func() {
					if request.ctx.Err() != nil {
						return
					}
					f.enqueueMutation(request.ctx, func() func() {
						result, err := request.storage.Remove(request.ctx, target)
						return f.removeDelivery(request, result, err)
					})
				},
				onCancel: f.cancelRemove,
				onClosed: func() { f.removeDialog = nil; f.focusManage() },
			})
		}
	})
}

func (f *Feature) removeDelivery(request removalRequest, result favstore.RemovalResult, err error) func() {
	return func() {
		if f.stopped {
			return
		}
		if result.Committed {
			f.refreshMenu()
			if request.ctx.Err() == nil {
				f.host.ShowToast(fmt.Sprintf(lang.L("removed favorite %q"), request.name))
			}
			return
		}
		if request.ctx.Err() != nil {
			return
		}
		if errors.Is(err, favstore.ErrConflict) {
			f.prepareRemove(request, true)
			return
		}
		if err != nil {
			f.reportError(lang.L("could not remove favorite %q: %v"), request.name, err)
		}
	}
}
