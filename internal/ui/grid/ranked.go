package grid

import (
	"fmt"
	"path/filepath"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
)

// Visit captures file identities independently of the collection's indexes.
type Visit struct {
	Paths, Results, Selected, Subset []string
	Highlight, Query                 string
	Searching, Ranked, Visible       bool
	ScrollOffset                     float32
	subsetBack, analyze              func()
	selectedOccurrences              map[fileOccurrence]bool
	highlightOccurrence              int
}
type fileOccurrence struct {
	path    string
	ordinal int
}

type rankedSourceIndex struct {
	byPath     map[string][]int
	generation uint64
	count      int
}

func (g *Overview) rankedSources() *rankedSourceIndex {
	generation, count := g.host.Generation(), g.host.FileCount()
	if g.rankSources != nil && g.rankSources.generation == generation && g.rankSources.count == count {
		return g.rankSources
	}
	index := &rankedSourceIndex{generation: generation, count: count, byPath: make(map[string][]int, count)}
	for i := range count {
		if source := g.host.FileAt(i); source != nil {
			path := source.Path()
			index.byPath[path] = append(index.byPath[path], i)
		}
	}
	g.rankSources = index
	return index
}

type Progress struct {
	Processed, Total, Failed int
	Complete                 bool
}
type RankedVisit struct {
	ReferencePath    string
	Paths            []string
	Revision         uint64
	Progress         Progress
	Back, Exit, Save func()
}

func (g *Overview) buildRankedBar() *fyne.Container {
	g.rankProgress = widget.NewProgressBar()
	g.rankStatus = widget.NewLabel("")
	g.rankReference = widget.NewLabel("")
	g.rankReference.Truncation = fyne.TextTruncateEllipsis
	g.rankBack = widget.NewButton(lang.L("Back"), func() {
		if g.ranked != nil && g.ranked.Back != nil {
			g.ranked.Back()
		}
	})
	exit := widget.NewButton(lang.L("Exit search"), func() {
		if g.ranked != nil && g.ranked.Exit != nil {
			g.ranked.Exit()
		}
	})
	save := widget.NewButton(lang.L("Save matches to Favorites"), func() {
		if g.ranked != nil && g.ranked.Save != nil {
			g.ranked.Save()
		}
	})
	g.rankedBar = container.NewVBox(g.rankReference, container.NewBorder(nil, nil, container.NewHBox(g.rankBack, exit), save, g.rankStatus), g.rankProgress)
	g.rankedBar.Hide()
	return g.rankedBar
}

// OpenRanked updates an existing result without reopening the Grid or rebinding
// selected identities. A held marquee finishes against its admitted ordering.
func (g *Overview) OpenRanked(visit RankedVisit) {
	if g.ranked != nil && visit.Revision < g.ranked.Revision {
		return
	}
	paths := make([]string, 0, min(len(visit.Paths)+1, 31))
	seen := make(map[string]bool, cap(paths))
	if visit.ReferencePath != "" {
		paths = append(paths, visit.ReferencePath)
		seen[visit.ReferencePath] = true
	}
	otherMatches := 0
	for _, path := range visit.Paths {
		if seen[path] {
			continue
		}
		if otherMatches == 30 {
			break
		}
		paths = append(paths, path)
		seen[path] = true
		otherMatches++
	}
	visit.Paths = paths
	if g.marqueeDragging {
		g.pendingRanked = &visit
		g.SetRankedProgress(visit.Progress)
		return
	}
	previous := g.CaptureVisit()
	updating := g.ranked != nil && g.visible && g.ranked.ReferencePath == visit.ReferencePath
	if !updating {
		g.Close()
		previous = Visit{}
	}
	g.ranked = &visit
	g.rankReference.SetText(fmt.Sprintf(lang.L("Matches for %s"), filepath.Base(visit.ReferencePath)))
	g.subset = make(map[string]bool, len(visit.Paths))
	for _, path := range visit.Paths {
		g.subset[path] = true
	}
	g.applyFilter()
	if !g.visible {
		g.Toggle()
	}
	if updating {
		g.restoreVisitState(previous)
	}
	g.SetRankedProgress(visit.Progress)
	g.rankedBar.Show()
	g.syncTopBar()
}
func (g *Overview) SetRankedProgress(progress Progress) {
	if g.rankProgress == nil {
		return
	}
	g.rankProgress.Max = float64(max(progress.Total, 1))
	g.rankProgress.SetValue(float64(progress.Processed))
	if progress.Complete {
		g.rankProgress.Hide()
	} else {
		g.rankProgress.Show()
	}
	g.rankStatus.SetText(fmt.Sprintf(lang.L("%d of %d processed, %d failed"), progress.Processed, progress.Total, progress.Failed))
}
func (g *Overview) SetOnRankedOpen(open func(Visit)) { g.onRankedOpen = open }

func (g *Overview) CaptureVisit() Visit {
	visit := Visit{Query: g.query, Searching: g.searching, Visible: g.visible, ScrollOffset: g.ScrollOffset(), subsetBack: g.onSubsetBack, analyze: g.onAnalyze}
	if g.ranked != nil {
		visit.Ranked = true
		visit.Paths = slices.Clone(g.ranked.Paths)
	}
	for _, i := range g.ResultIndexes() {
		visit.Results = append(visit.Results, g.host.FileAt(i).Path())
	}
	for _, i := range g.Selection() {
		if i >= 0 && i < g.host.FileCount() {
			visit.Selected = append(visit.Selected, g.host.FileAt(i).Path())
		}
	}
	if i := g.fileIndex(g.highlight); i >= 0 {
		visit.Highlight = g.host.FileAt(i).Path()
	}
	// Keep occurrence ordinals private and immutable so a copied visit retains
	// one selected duplicate even when paths repeat in a merged collection.
	visit.selectedOccurrences = make(map[fileOccurrence]bool, len(visit.Selected))
	if g.ranked != nil {
		index := g.rankedSources()
		for _, i := range g.Selection() {
			if i >= 0 && i < g.host.FileCount() {
				path := g.host.FileAt(i).Path()
				ordinal, found := slices.BinarySearch(index.byPath[path], i)
				if found {
					visit.selectedOccurrences[fileOccurrence{path, ordinal}] = true
				}
			}
		}
		visit.highlightOccurrence, _ = slices.BinarySearch(index.byPath[visit.Highlight], g.fileIndex(g.highlight))
	} else {
		selected := make(map[int]bool, len(visit.Selected))
		counts := make(map[string]int, len(visit.Selected)+1)
		highlight := g.fileIndex(g.highlight)
		last := highlight
		for _, i := range g.Selection() {
			if i >= 0 && i < g.host.FileCount() {
				selected[i] = true
				counts[g.host.FileAt(i).Path()] = 0
				last = max(last, i)
			}
		}
		counts[visit.Highlight] = 0
		for i := 0; i <= last; i++ {
			path := g.host.FileAt(i).Path()
			ordinal, wanted := counts[path]
			if !wanted {
				continue
			}
			if selected[i] {
				visit.selectedOccurrences[fileOccurrence{path, ordinal}] = true
			}
			if i == highlight {
				visit.highlightOccurrence = ordinal
			}
			counts[path]++
		}
	}
	if g.subset != nil {
		visit.Subset = []string{}
	}
	for path := range g.subset {
		visit.Subset = append(visit.Subset, path)
	}
	slices.Sort(visit.Subset)
	return visit
}

// RestoreVisit restores an ordinary/cohort visit; ranked callers first supply
// their current command callbacks through OpenRanked.
func (g *Overview) RestoreVisit(visit Visit) {
	if !visit.Ranked {
		g.Close()
		g.onSubsetBack, g.onAnalyze = visit.subsetBack, visit.analyze
		if visit.Subset != nil {
			g.subset = map[string]bool{}
			for _, path := range visit.Subset {
				g.subset[path] = true
			}
		}
		g.applyFilter()
		if visit.Visible {
			g.Toggle()
		}
	}
	g.restoreVisitState(visit)
}
func (g *Overview) restoreVisitState(visit Visit) {
	g.query, g.searching = visit.Query, visit.Searching
	g.applyFilter()
	var selected []int
	wanted := visit.selectedOccurrences
	if wanted == nil {
		wanted = make(map[fileOccurrence]bool, len(visit.Selected))
		for _, path := range visit.Selected {
			wanted[fileOccurrence{path, 0}] = true
		}
	}
	highlight := -1
	if g.ranked != nil {
		index := g.rankedSources()
		for identity := range wanted {
			indexes := index.byPath[identity.path]
			if identity.ordinal >= 0 && identity.ordinal < len(indexes) && g.subset[identity.path] {
				selected = append(selected, indexes[identity.ordinal])
			}
		}
		indexes := index.byPath[visit.Highlight]
		if visit.highlightOccurrence >= 0 && visit.highlightOccurrence < len(indexes) {
			highlight = indexes[visit.highlightOccurrence]
		}
	} else {
		counts := make(map[string]int, len(visit.Selected)+1)
		for identity := range wanted {
			counts[identity.path] = 0
		}
		counts[visit.Highlight] = 0
		for i := range g.host.FileCount() {
			path := g.host.FileAt(i).Path()
			ordinal, tracked := counts[path]
			if !tracked {
				continue
			}
			counts[path]++
			if wanted[fileOccurrence{path, ordinal}] {
				selected = append(selected, i)
			}
			if path == visit.Highlight && ordinal == visit.highlightOccurrence {
				highlight = i
			}
		}
	}
	g.sel.Replace(selected)
	g.wrap.ScrollToOffset(visit.ScrollOffset)
	if id := g.displayIndexOfHost(highlight); id >= 0 {
		g.setHighlight(id)
	} else if g.count() > 0 {
		g.setHighlight(0)
	}
	g.wrap.Refresh()
	g.syncTopBar()
	g.fireSelectionChanged()
}
func (g *Overview) flushRanked() {
	if pending := g.pendingRanked; pending != nil && !g.marqueeDragging {
		g.pendingRanked = nil
		g.OpenRanked(*pending)
	}
}
