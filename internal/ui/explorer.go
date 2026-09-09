package ui

import (
	"errors"
	"fmt"
	"slices"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/similarity"
	explorerui "github.com/frathe/picfetch/internal/ui/explorer"
	"github.com/frathe/picfetch/internal/ui/grid"
	"github.com/frathe/picfetch/internal/winpos"
)

type explorerWork struct {
	cacheFavorites, autoFit bool
	prepare                 func()
	lifecycle               requestLifecycle
	workers                 sync.WaitGroup
	ui                      grid.UIQueue
	surface                 *explorerui.Map
	sources                 []string
	cohort                  []string
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
			v.explorer.surface.SetResult(nil)
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
	v.explorer.surface.SetResult(nil)
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
	v.explorer.workers.Go(func() {
		err := analyze(token.context(), paths, controls, func(event similarity.Event) {
			if !token.current() {
				return
			}
			v.explorer.ui.Do(func() {
				if !token.current() {
					return
				}
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
					v.explorer.surface.SetResult(event.Items)
					v.ForceRepaint()
					if !v.explorer.hasMap {
						v.explorer.surface.Fit()
					} else if v.explorer.autoFit {
						v.explorer.surface.ExpandToFit()
					}
					v.explorer.hasMap = true
				}
				v.explorer.surface.UpdateState(!v.explorer.complete && v.explorer.available > v.explorer.mapped, v.explorer.building)
			})
		})
		if !token.current() {
			return
		}
		v.explorer.ui.Do(func() {
			if !token.current() {
				return
			}
			v.explorer.controls = nil
			v.explorer.surface.UpdateState(false, false)
			if err != nil {
				fyne.LogError("visual similarity analysis failed", err)
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
	if len(paths) == 0 {
		return
	}
	v.explorer.cohort = append([]string(nil), paths...)
	v.grid.OpenSubset(paths, v.backToSimilarityMap)
	v.syncMenus()
}
func (v *viewer) LeaveSimilarityMap() {
	v.closeExplorer()
	v.grid.Close()
	v.syncMenus()
}
func (v *viewer) closeExplorer() {
	if v.explorer.prepare != nil {
		v.explorer.prepare = nil
		v.grid.Close()
	}
	v.explorer.controls = nil
	v.explorer.lifecycle.invalidate()
	v.explorer.surface.SetResult(nil)
	v.explorer.surface.UpdateState(false, false)
	v.explorer.surface.Hide()
	v.explorer.sources = nil
	v.explorer.cohort = nil
	v.explorer.complete = false
	v.explorer.hasMap = false
}
func (v *viewer) settleExplorer() {
	for {
		v.grid.Settle()
		v.explorer.workers.Wait()
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
		}
		return true
	}
	if len(v.explorer.cohort) > 0 && (key == fyne.KeyEscape || key == fyne.KeyG) {
		v.grid.OpenSubset(v.explorer.cohort, v.backToSimilarityMap)
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

func (v *viewer) backToSimilarityMap() { v.explorer.surface.Show(); v.ForceRepaint(); v.syncMenus() }
