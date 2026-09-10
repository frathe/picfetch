package ui

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/explorerpresets"
	"github.com/frathe/picfetch/internal/explorertrial"
	"github.com/frathe/picfetch/internal/favstore"
	"github.com/frathe/picfetch/internal/similarity"
	explorerui "github.com/frathe/picfetch/internal/ui/explorer"
	"github.com/frathe/picfetch/internal/ui/grid"
	"github.com/frathe/picfetch/internal/winpos"
)

type explorerWork struct {
	trial                   *explorertrial.Session
	trialRun                int
	pendingLaunch           bool
	cacheFavorites, autoFit bool
	prepare                 func()
	lifecycle               requestLifecycle
	token                   requestToken
	workers                 sync.WaitGroup
	presetWorkers           sync.WaitGroup
	ui                      grid.UIQueue
	surface                 *explorerui.Map
	sources                 []string
	cohort                  []string
	unassignedCohort        bool
	cohortDialog            dialog.Dialog
	favoriteDir             string
	cohortStore             *favstore.CohortStore
	cohortLoadErr           error
	cohortSaving            bool
	presets                 *explorerpresets.Store
	presetOp                requestLifecycle
	complete                bool
	hasMap                  bool
	maximized               bool
	controls                chan similarity.Control
	available, mapped       int
	building, automatic     bool
}
type explorerQueue struct{}

func (explorerQueue) Do(f func()) { fyne.Do(f) }
func (explorerQueue) Drain() bool { return false }

func (v *viewer) showExplorer() {
	if v.stopping || v.comparisonActive() || v.FileCount() == 0 || !v.yieldCopySelection() {
		return
	}
	if v.slides.Active() {
		v.slides.Exit()
		v.resetFade()
	}
	v.grid.Close()
	v.explorer.cohort = nil
	winpos.Maximize(v.win)
	v.explorer.maximized = true
	v.explorer.surface.Show()
	v.ForceRepaint()
	v.syncMenus()
	if v.dupes.HideDuplicates() {
		v.explorer.controls = nil
		token := v.explorer.lifecycle.begin()
		v.explorer.prepare = func() {
			if token.current() {
				v.beginExplorerAnalysis()
			}
		}
		if !v.grid.PrepareDuplicateGroups() {
			v.explorer.complete = false
			v.explorer.hasMap = false
			v.explorer.surface.SetResult(nil, nil)
			v.explorer.surface.Status(lang.L("Checking duplicate groups..."))
			v.explorer.surface.UpdateState(false, false)
			return
		}
		v.explorer.prepare = nil
	}
	v.beginExplorerAnalysis()
}

func (v *viewer) beginExplorerAnalysis() {
	paths := make([]string, 0, v.FileCount())
	visibility := v.dupes.Visibility()
	for i := range v.FileCount() {
		if visibility.Visible(i) {
			paths = append(paths, v.FileAt(i).Path())
		}
	}
	if slices.Equal(paths, v.explorer.sources) && v.explorer.complete {
		return
	}
	v.explorer.sources = paths
	v.explorer.complete = false
	v.explorer.hasMap = false
	token := v.explorer.lifecycle.begin()
	v.explorer.token = token
	v.explorer.surface.SetResult(nil, nil)
	v.explorer.surface.Status(lang.L("Analyzing images..."))
	v.explorer.available, v.explorer.mapped = 0, 0
	v.explorer.building = false
	v.explorer.controls = make(chan similarity.Control, 1)
	v.sendSimilarityControl(false)
	v.explorer.surface.UpdateState(false, false)
	controls := v.explorer.controls
	analyze := v.explorerAnalyze
	if analyze == nil {
		client := similarity.Client{}
		if v.explorer.cacheFavorites {
			client.FavoritesDir = v.favorites.Dir()
		}
		analyze = client.Analyze
	}
	cacheWarningReported := false
	favoriteDir := v.explorer.favoriteDir
	trial := v.explorer.trial
	run := trial.Begin(paths)
	v.explorer.trialRun = run
	v.explorer.cohortStore, v.explorer.cohortLoadErr = nil, nil
	v.explorer.cohortSaving = false
	v.explorer.workers.Go(func() {
		workerErr := context.Canceled
		defer func() { trial.Exited(run, workerErr) }()
		if favoriteDir != "" {
			store, groups, err := favstore.OpenCohorts(token.context(), favoriteDir)
			if err == nil && !store.Contains(paths) {
				// Merge mode can combine a favorite with unrelated open files.
				store, groups = nil, favstore.CohortState{}
			}
			if !token.current() {
				return
			}
			v.explorer.ui.Do(func() {
				if !token.current() {
					return
				}
				v.explorer.cohortStore, v.explorer.cohortLoadErr = store, err
				if err != nil {
					fyne.LogError("load favorite cohorts", err)
					v.ShowToast(lang.L("Could not load saved cohorts for this favorite."))
					return
				}
				v.explorer.surface.RestoreCohorts(groups)
			})
		}
		err := analyze(token.context(), paths, controls, func(event similarity.Event) {
			received := time.Now()
			trial.Received(run, event)
			if !token.current() {
				return
			}
			v.explorer.ui.Do(func() {
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
				v.explorer.available = event.Successful
				if event.Stage == "layout" {
					v.explorer.building = true
					status += " | " + fmt.Sprintf(lang.L("Grouping and arranging %d images..."), event.Successful)
				}
				v.explorer.surface.Status(status)
				if event.Complete {
					v.explorer.complete = true
				}
				if event.Items != nil {
					v.explorer.mapped = event.Successful
					v.explorer.building = false
					v.explorer.surface.SetResult(event.Items, event.Merges)
					v.ForceRepaint()
					if !v.explorer.hasMap {
						v.explorer.surface.Fit()
					} else if v.explorer.autoFit {
						v.explorer.surface.ExpandToFit()
					}
					v.explorer.hasMap = true
					trial.Applied(run, event, received, started)
				}
				v.explorer.surface.UpdateState(!v.explorer.complete && v.explorer.available > v.explorer.mapped, v.explorer.building)
			})
		})
		workerErr = err
		if !token.current() {
			return
		}
		v.explorer.ui.Do(func() {
			if !token.current() {
				return
			}
			v.explorer.controls = nil
			v.explorer.surface.UpdateState(false, false)
			displayErr := err
			if displayErr == nil && !v.explorer.complete {
				displayErr = errors.New("analysis ended without a completed map")
			}
			if displayErr != nil {
				v.explorer.complete = false
				fyne.LogError("visual similarity analysis failed", displayErr)
				v.explorer.surface.Status(lang.L("Analysis failed. Open the explorer to retry."))
			}
		})
	})
}

func (v *viewer) UpdateSimilarityMap() {
	if v.explorer.controls == nil || v.explorer.building || v.explorer.available <= v.explorer.mapped {
		return
	}
	v.sendSimilarityControl(true)
	v.explorer.building = true
	v.explorer.surface.UpdateState(false, true)
}
func (v *viewer) SetSimilarityAutoUpdate(on bool) {
	v.explorer.automatic = on
	v.explorer.surface.SetAutomatic(on)
	v.sendSimilarityControl(false)
}
func (v *viewer) sendSimilarityControl(update bool) {
	controls := v.explorer.controls
	if controls == nil {
		return
	}
	control := similarity.Control{Automatic: v.explorer.automatic, Update: update}
	select {
	case pending := <-controls:
		control.Update = control.Update || pending.Update
	default:
	}
	controls <- control
}

// OpenSimilarityCohort captures membership at activation, so returning from an
// image reopens the same cohort even if a later map revision has arrived.
func (v *viewer) OpenSimilarityCohort(paths []string) {
	v.openSimilarityCohort(paths, false)
}

func (v *viewer) OpenSimilarityUnassigned(paths []string) {
	v.openSimilarityCohort(paths, true)
}

func (v *viewer) openSimilarityCohort(paths []string, unassigned bool) {
	if len(paths) == 0 {
		return
	}
	v.explorer.cohort = append([]string(nil), paths...)
	v.explorer.unassignedCohort = unassigned
	v.openExplorerGrid()
	v.syncMenus()
	v.explorer.trial.Action(v.explorer.trialRun, "cohort-open", len(paths))
}

func (v *viewer) openExplorerGrid() {
	if v.explorer.unassignedCohort {
		v.grid.OpenUnassigned(v.explorer.cohort, v.backToSimilarityMap, v.analyzeSimilaritySelection)
	} else {
		v.grid.OpenSubset(v.explorer.cohort, v.backToSimilarityMap)
	}
}

func (v *viewer) LeaveSimilarityMap() {
	v.closeExplorer()
	v.grid.Close()
	v.syncMenus()
}
func (v *viewer) closeExplorer() {
	v.retireExplorerAnalysis()
	v.explorer.surface.Hide()
	v.explorer.cohort = nil
}

// retireExplorerAnalysis drops source-derived state without disturbing a cohort
// currently being browsed. Committed file effects may arrive after navigation.
func (v *viewer) retireExplorerAnalysis() {
	if len(v.explorer.sources) > 0 {
		v.explorer.trial.Action(v.explorer.trialRun, "explorer-exit", len(v.explorer.sources))
	}
	v.explorer.presetOp.invalidate()
	if v.explorer.cohortDialog != nil {
		v.explorer.cohortDialog.Hide()
	}
	if v.explorer.prepare != nil {
		v.explorer.prepare = nil
		v.grid.Close()
	}
	v.explorer.controls = nil
	v.explorer.lifecycle.invalidate()
	v.explorer.cohortStore, v.explorer.cohortLoadErr = nil, nil
	v.explorer.cohortSaving = false
	v.explorer.surface.SetResult(nil, nil)
	v.explorer.surface.UpdateState(false, false)
	v.explorer.sources = nil
	v.explorer.complete = false
	v.explorer.hasMap = false
}

func (v *viewer) explorerSourcesChanged() {
	if len(v.explorer.sources) == 0 && v.explorer.prepare == nil {
		return
	}
	v.retireExplorerAnalysis()
	v.explorer.surface.Status(lang.L("Source files changed. Open the explorer to analyze again."))
}
func (v *viewer) settleExplorer() {
	for {
		v.grid.Settle()
		v.explorer.workers.Wait()
		v.explorer.presetWorkers.Wait()
		if !v.explorer.ui.Drain() {
			return
		}
	}
}
func (v *viewer) explorerGridChanged() { v.syncMenus() }
func (v *viewer) explorerMapActive() bool {
	return v.explorer.surface != nil && v.explorer.surface.Visible() && !v.grid.Visible()
}
func (v *viewer) explorerImageOpened() {
	if len(v.explorer.cohort) > 0 {
		v.explorer.surface.Hide()
	}
}
func (v *viewer) explorerKey(key fyne.KeyName) bool {
	if v.explorer.surface.Visible() {
		switch key {
		case fyne.KeyEscape, fyne.KeyV:
			v.LeaveSimilarityMap()
		case fyne.KeyF1:
			v.help.ShowManual()
		default:
			v.explorer.surface.HandleKey(key)
		}
		return true
	}
	if len(v.explorer.cohort) > 0 && (key == fyne.KeyEscape || key == fyne.KeyG) {
		v.openExplorerGrid()
		v.explorer.surface.Show()
		v.ForceRepaint()
		return true
	}
	return false
}

func (v *viewer) cohortIndexes() []int {
	if len(v.explorer.cohort) == 0 {
		return nil
	}
	members := make(map[string]bool, len(v.explorer.cohort))
	for _, path := range v.explorer.cohort {
		members[path] = true
	}
	var indexes []int
	for i := range v.FileCount() {
		if members[v.FileAt(i).Path()] {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func (v *viewer) backToSimilarityMap() {
	v.explorer.surface.Show()
	v.ForceRepaint()
	v.syncMenus()
	v.explorer.trial.Action(v.explorer.trialRun, "map-return", len(v.explorer.cohort))
}
