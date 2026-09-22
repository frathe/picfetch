package ui

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"

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
	stopped    atomic.Bool
}

func (v *viewer) heicContext(ctx context.Context) context.Context {
	if v.heic == nil {
		return ctx
	}
	return v.heic.capability.CaptureContext(ctx)
}

func (v *viewer) persistedFiles(files []fyne.URI) []fyne.URI {
	available := make(map[string]int, len(files))
	for _, uri := range files {
		available[uri.String()]++
	}
	// Attach unavailable members to the preceding surviving source. This
	// reconstructs unsorted session order while preserving a Favorite's chosen
	// order for its visible members, and never resurrects removed visible files.
	// Merge mode permits repeated visible sources. Each occurrence owns its
	// own following gaps, even when another occurrence has the same URI.
	type position struct {
		key        string
		occurrence int
	}
	after := make(map[position][]fyne.URI)
	occurrences := make(map[string]int)
	anchor := position{}
	for _, source := range v.state.unavailableOrder {
		key := source.uri.String()
		if source.unavailable {
			after[anchor] = append(after[anchor], source.uri)
		} else if available[key] > 0 {
			occurrences[key]++
			if occurrences[key] <= available[key] {
				anchor = position{key, occurrences[key]}
			}
		}
	}
	clear(occurrences)
	result := append([]fyne.URI(nil), after[position{}]...)
	for _, uri := range files {
		key := uri.String()
		occurrences[key]++
		anchor = position{key, occurrences[key]}
		result = append(result, uri)
		result = append(result, after[anchor]...)
	}
	return result
}

func (v *viewer) retainedOrder() []collectionSource {
	if v.state.unavailableOrder != nil {
		return slices.Clone(v.state.unavailableOrder)
	}
	order := make([]collectionSource, len(v.state.unsortedFiles))
	for i, uri := range v.state.unsortedFiles {
		order[i].uri = uri
	}
	return order
}

func (v *viewer) retainUnavailableHEIC(merging bool, skipped, order []fyne.URI) {
	var retained []collectionSource
	if merging {
		retained = v.retainedOrder()
	}
	missing := make(map[string]bool, len(skipped))
	for _, uri := range skipped {
		missing[uri.String()] = true
	}
	for _, uri := range order {
		retained = append(retained, collectionSource{uri, missing[uri.String()]})
	}
	v.state.retainOrder(retained)
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
		if work.stopped.Load() {
			return
		}
		work.ui.Do(func() {
			if work.stopped.Load() || v.heic != work {
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
	if work == nil || work.stopped.Load() {
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
		if work.stopped.Load() {
			return
		}
		queue.Do(func() {
			if work.stopped.Load() || v.heic != work || work.delivery != done {
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
	if v.heic == nil || v.heic.stopped.Swap(true) {
		return
	}
	v.heic.capability.Stop()
	if !v.heic.capability.State().Known {
		// Shutdown suppresses queued UI delivery, including invalidation
		// persistence. Reconcile before Fyne flushes preferences at OnStopped.
		preferences.ClearHEICObservation(v.app)
	}
	if backend, ok := v.heic.backend.(interface{ Stop() }); ok {
		backend.Stop()
	}
}

// waitHEIC joins canceled workers after the event loop exits. It never drains
// UI callbacks; those have already been invalidated by stopHEIC.
func (v *viewer) waitHEIC() {
	if v.heic == nil {
		return
	}
	v.heic.capability.Wait()
	v.heic.workers.Wait()
	if backend, ok := v.heic.backend.(interface{ Wait() }); ok {
		backend.Wait()
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
