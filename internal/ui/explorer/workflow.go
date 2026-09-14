package explorer

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/favstore"
	"github.com/frathe/picfetch/internal/similarity"
)

func (f *Feature) Open(request OpenRequest) bool {
	if f.stopping || len(request.Sources) == 0 {
		return false
	}
	paths := slices.Clone(request.Sources)
	f.surface.Show()
	f.cohort = nil
	f.preparing = false
	f.host.Repaint()
	if slices.Equal(paths, f.sources) && f.complete {
		return true
	}
	f.retireAnalysis()
	f.favoriteDir = request.FavoriteDir
	f.sources = paths
	f.complete = false
	f.hasMap = false
	token := f.lifecycle.begin()
	f.token = token
	f.surface.SetResult(nil, nil)
	f.surface.Status(lang.L("Analyzing images..."))
	f.available, f.mapped = 0, 0
	f.building = false
	f.controls = make(chan similarity.Control, 1)
	f.sendSimilarityControl(false)
	f.surface.UpdateState(false, false)
	controls := f.controls
	f.host.Changed()
	analyze := f.analyze
	if analyze == nil {
		client := f.client
		client.GeneralAnalysisDir = request.GeneralAnalysisDir
		client.FavoritesDir = request.FavoritesDir
		client.DisableFavoriteCache = !f.cacheFavorites
		analyze = client.Analyze
	}
	cacheWarningReported := false
	favoriteDir := f.favoriteDir
	trial := f.trial
	run := trial.Begin(paths)
	f.trialRun = run
	f.trialEvent = 0
	f.cohortStore, f.cohortLoadErr = nil, nil
	f.cohortSaving = false
	previous, before := f.analysisDone, f.analysisBefore
	done := make(chan struct{})
	f.analysisDone = done
	f.workers.Go(func() {
		defer func() {
			for _, barrier := range []<-chan struct{}{previous, before} {
				if barrier != nil {
					<-barrier
				}
			}
			close(done)
		}()
		workerErr := context.Canceled
		defer func() { trial.Exited(run, workerErr) }()
		for _, barrier := range []<-chan struct{}{previous, before} {
			if barrier != nil {
				select {
				case <-barrier:
				case <-token.context().Done():
					return
				}
			}
		}
		if !token.current() {
			return
		}
		if favoriteDir != "" {
			store, groups, err := favstore.OpenCohorts(token.context(), favoriteDir)
			if err == nil && !store.Contains(paths) {
				// Merge mode can combine a favorite with unrelated open files.
				store, groups = nil, favstore.CohortState{}
			}
			if !token.current() {
				return
			}
			f.ui.Do(func() {
				if !token.current() {
					return
				}
				f.cohortStore, f.cohortLoadErr = store, err
				if err != nil {
					fyne.LogError("load favorite cohorts", err)
					f.host.ShowToast(lang.L("Could not load saved cohorts for this favorite."))
					return
				}
				f.surface.RestoreCohorts(groups)
			})
		}
		err := analyze(token.context(), paths, controls, func(event similarity.Event) {
			received := time.Now()
			eventID := trial.Received(run, event)
			if !token.current() {
				return
			}
			f.ui.Do(func() {
				if !token.current() {
					return
				}
				started := time.Now()
				if event.CacheWarning != "" && !cacheWarningReported {
					cacheWarningReported = true
					fyne.LogError("favorite analysis cache", errors.New(event.CacheWarning))
				}
				status := fmt.Sprintf(lang.L("%d ready, %d failed, %d total"), event.Successful, event.Failed, event.Total)
				if event.Reused > 0 {
					status += " | " + fmt.Sprintf(lang.L("%d reused"), event.Reused)
				}
				f.available = event.Successful
				if event.Stage == "layout" {
					f.building = true
					status += " | " + fmt.Sprintf(lang.L("Grouping and arranging %d images..."), event.Successful)
				}
				f.surface.Status(status)
				if event.Complete {
					f.complete = true
				}
				if event.Items != nil {
					f.mapped = event.Successful
					f.building = false
					f.surface.SetResult(event.Items, event.Merges)
					f.host.Repaint()
					if !f.hasMap {
						f.surface.Fit()
					} else if f.autoFit && f.surface.Visible() && !f.host.Presentation().GridVisible {
						f.surface.ExpandToFit()
					}
					f.hasMap = true
					f.trialEvent = eventID
					trial.Applied(run, eventID, event, received, started)
					f.RecordView("view-observed")
				}
				f.surface.UpdateState(!f.complete && f.available > f.mapped, f.building)
			})
		})
		workerErr = err
		if !token.current() {
			return
		}
		f.ui.Do(func() {
			if !token.current() {
				return
			}
			f.controls = nil
			f.surface.UpdateState(false, false)
			displayErr := err
			if displayErr == nil && !f.complete {
				displayErr = errors.New("analysis ended without a completed map")
			}
			if displayErr != nil {
				f.complete = false
				f.assetsReady = false
				fyne.LogError("visual similarity analysis failed", displayErr)
				f.surface.Status(lang.L("Analysis failed. Open the explorer to retry."))
			}
			f.host.Changed()
		})
	})
	return true
}

func (f *Feature) UpdateSimilarityMap() {
	if f.controls == nil || f.building || f.available <= f.mapped {
		return
	}
	f.sendSimilarityControl(true)
	f.building = true
	f.surface.UpdateState(false, true)
}
func (f *Feature) SetSimilarityAutoUpdate(on bool) {
	f.automatic = on
	f.surface.SetAutomatic(on)
	f.sendSimilarityControl(false)
}
func (f *Feature) sendSimilarityControl(update bool) {
	controls := f.controls
	if controls == nil {
		return
	}
	control := similarity.Control{Automatic: f.automatic, Update: update}
	select {
	case pending := <-controls:
		control.Update = control.Update || pending.Update
	default:
	}
	controls <- control
}
