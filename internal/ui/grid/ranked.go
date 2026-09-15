package grid

import (
	"fmt"
	"path/filepath"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/fileidentity"
)

// Visit captures file identities independently of the collection's indexes.
type Visit struct {
	Paths, Results, Selected, Subset []string
	Highlight, Query                 string
	Searching, Ranked, Visible       bool
	ScrollOffset                     float32
	subsetBack, analyze              func()
	selectedOccurrences              map[fileidentity.Occurrence]bool
	highlightIdentity                fileidentity.Occurrence
}
type visitSourceIndex struct {
	identities fileidentity.Index
	generation uint64
	count      int
}

func (g *Overview) visitSources() *visitSourceIndex {
	generation, count := g.host.Generation(), g.host.FileCount()
	if g.visitIndex != nil && g.visitIndex.generation == generation && g.visitIndex.count == count {
		return g.visitIndex
	}
	index := &visitSourceIndex{generation: generation, count: count}
	index.identities = fileidentity.NewIndex(count, func(i int) string {
		if source := g.host.FileAt(i); source != nil {
			return source.Path()
		}
		return ""
	})
	g.visitIndex = index
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
	// Occurrence bookmarks are immutable after capture, including when paths
	// repeat in a merged collection. Reuse the generation-bound source index.
	visit.selectedOccurrences = make(map[fileidentity.Occurrence]bool, len(visit.Selected))
	index := g.visitSources().identities
	for _, i := range g.Selection() {
		if i >= 0 && i < g.host.FileCount() {
			if identity, ok := index.Capture(g.host.FileAt(i).Path(), i); ok {
				visit.selectedOccurrences[identity] = true
			}
		}
	}
	visit.highlightIdentity, _ = index.Capture(visit.Highlight, g.fileIndex(g.highlight))
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
		wanted = make(map[fileidentity.Occurrence]bool, len(visit.Selected))
		for _, path := range visit.Selected {
			wanted[fileidentity.Occurrence{Path: path}] = true
		}
	}
	index := g.visitSources().identities
	for identity := range wanted {
		if i := index.Resolve(identity); i >= 0 && (g.ranked == nil || g.subset[identity.Path]) {
			selected = append(selected, i)
		}
	}
	highlightIdentity := visit.highlightIdentity
	if highlightIdentity.Path == "" {
		highlightIdentity.Path = visit.Highlight
	}
	highlight := index.Resolve(highlightIdentity)
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
