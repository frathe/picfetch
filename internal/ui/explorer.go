package ui

import (
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
	lifecycle requestLifecycle
	workers   sync.WaitGroup
	ui        grid.UIQueue
	surface   *explorerui.Map
	sources   []string
	cohort    []string
	complete  bool
	maximized bool
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
	paths := make([]string, v.FileCount())
	for i := range paths {
		paths[i] = v.FileAt(i).Path()
	}
	winpos.Maximize(v.win)
	v.explorer.maximized = true
	v.explorer.surface.Show()
	v.ForceRepaint()
	v.syncMenus()
	if slices.Equal(paths, v.explorer.sources) && v.explorer.complete {
		return
	}
	v.explorer.sources = paths
	v.explorer.complete = false
	token := v.explorer.lifecycle.begin()
	v.explorer.surface.SetResult(nil)
	v.explorer.surface.Status(lang.L("Analyzing images..."))
	analyze := v.explorerAnalyze
	v.explorer.workers.Go(func() {
		err := analyze(token.context(), paths, func(event similarity.Event) {
			if !token.current() {
				return
			}
			v.explorer.ui.Do(func() {
				if !token.current() {
					return
				}
				v.explorer.surface.Status(fmt.Sprintf(lang.L("%d ready, %d failed, %d total"), event.Successful, event.Failed, event.Total))
				if event.Stage == "layout" {
					v.explorer.surface.Status(fmt.Sprintf(lang.L("Grouping and arranging %d images..."), event.Successful))
				}
				if event.Complete {
					v.explorer.complete = true
					v.explorer.surface.SetResult(event.Items)
					v.ForceRepaint()
					v.explorer.surface.Fit()
				}
			})
		})
		if !token.current() {
			return
		}
		v.explorer.ui.Do(func() {
			if !token.current() {
				return
			}
			if err != nil {
				fyne.LogError("visual similarity analysis failed", err)
				v.explorer.surface.Status(lang.L("Analysis failed. Open the explorer to retry."))
			}
		})
	})
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
	if !v.explorer.complete {
		v.explorer.lifecycle.invalidate()
		v.explorer.sources = nil
	}
	v.explorer.surface.Hide()
	v.explorer.cohort = nil
	v.grid.Close()
	v.syncMenus()
}
func (v *viewer) closeExplorer() {
	v.explorer.lifecycle.invalidate()
	v.explorer.surface.Hide()
	v.explorer.sources = nil
	v.explorer.cohort = nil
	v.explorer.complete = false
}
func (v *viewer) settleExplorer() {
	for {
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
