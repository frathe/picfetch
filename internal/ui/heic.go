package ui

import (
	"context"
	"fmt"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/heic"
	"github.com/frathe/picfetch/internal/preferences"
)

type heicWork struct {
	capability *heic.Capability
	backend    heic.Backend
	ui         fileUIQueue
	workers    sync.WaitGroup
	delivery   <-chan struct{}
	stopped    bool
}

func (v *viewer) heicContext(ctx context.Context) context.Context {
	if v.heic == nil {
		return ctx
	}
	return v.heic.capability.CaptureContext(ctx)
}

func (v *viewer) persistedFiles(files []fyne.URI) []fyne.URI {
	result := append([]fyne.URI(nil), files...)
	seen := make(map[string]bool, len(result))
	for _, uri := range result {
		seen[uri.String()] = true
	}
	for _, uri := range v.state.unavailableHEIC {
		if !seen[uri.String()] {
			result = append(result, uri)
			seen[uri.String()] = true
		}
	}
	return result
}

func (v *viewer) explainUnavailableHEIC(skipped []fyne.URI, explicit bool) {
	if len(skipped) == 0 {
		return
	}
	if !explicit {
		v.ShowToast(fmt.Sprintf(lang.L("Skipped %d HEIC files because system support is unavailable. See Settings for the HEIC installation guide."), len(skipped)))
		return
	}
	message := lang.L("HEIC support is unavailable. Check support in Settings or open the installation guide.")
	if !v.heic.capability.State().Known {
		message = lang.L("HEIC support could not be checked. Retry in Settings or open the installation guide.")
	}
	v.ShowToast(message)
	body := widget.NewLabel(message)
	body.Wrapping = fyne.TextWrapWord
	dialog.NewCustomConfirm(lang.L("HEIC support"), lang.L("HEIC installation guide"), lang.L("Close"), body, func(open bool) {
		if open && !v.stopping {
			v.help.ShowHEICGuide()
		}
	}, v.win).Show()
}

// configureHEIC sets the single native boundary before admitting any image or
// check work. The viewer harness uses this same composition with an OS stub.
func (v *viewer) configureHEIC(backend heic.Backend) {
	v.heic = &heicWork{backend: backend, ui: fyneFileQueue{}, capability: heic.NewCapability(backend, heic.SystemIdentity(), preferences.LoadHEICObservation(v.app))}
	work := v.heic
	work.capability.SetOnInvalidated(func() {
		work.ui.Do(func() {
			if work.stopped || v.heic != work {
				return
			}
			if !work.capability.State().Known {
				preferences.ClearHEICObservation(v.app)
				v.startHEICCheck(false)
			}
		})
	})
	v.settingsWin.SetHEICGuideAction(v.help.ShowHEICGuide)
	v.settingsWin.SetHEICCheckAction(func() { v.startHEICCheck(true) })
	v.settingsWin.SetHEICStatus(v.heic.capability.State())
	v.display.SetHEICCapability(v.heic.capability)
	v.grid.SetHEICCapability(v.heic.capability)
	v.exif.SetHEICCapability(v.heic.capability)
	v.spiral.SetHEICCapability(v.heic.capability)
	v.explorer.SetHEICCapability(v.heic.capability)
	v.visualsearch.SetHEICCapability(v.heic.capability)
}

func (v *viewer) startHEICCheck(manual bool) {
	work := v.heic
	if work == nil || work.stopped {
		return
	}
	var done <-chan struct{}
	if manual {
		done = work.capability.Check(context.Background())
	} else {
		done = work.capability.Ensure(context.Background())
	}
	v.settingsWin.SetHEICStatus(work.capability.State())
	if work.delivery == done {
		return
	}
	work.delivery = done
	queue := work.ui
	work.workers.Go(func() {
		<-done
		queue.Do(func() {
			if work.stopped || v.heic != work || work.delivery != done {
				return
			}
			state := work.capability.State()
			if state.Known {
				preferences.SaveHEICObservation(v.app, state.Observation)
			}
			v.settingsWin.SetHEICStatus(state)
		})
	})
}

func (v *viewer) stopHEIC() {
	if v.heic == nil || v.heic.stopped {
		return
	}
	v.heic.stopped = true
	v.heic.capability.Stop()
	if backend, ok := v.heic.backend.(interface{ Stop() }); ok {
		backend.Stop()
	}
}

func (v *viewer) settleHEIC() {
	if v.heic == nil {
		return
	}
	for {
		v.heic.capability.Wait()
		v.heic.workers.Wait()
		if !v.heic.ui.Drain() {
			break
		}
	}
	if backend, ok := v.heic.backend.(interface{ Wait() }); ok {
		backend.Wait()
	}
}
