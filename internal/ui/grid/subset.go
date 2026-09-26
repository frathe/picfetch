package grid

import "github.com/frathe/picfetch/internal/fileidentity"

// OpenOccurrences opens an exact occurrence cohort without collapsing repeated paths.
func (g *Overview) OpenOccurrences(members []fileidentity.Occurrence, label string, back func()) {
	g.Close()
	// Hidden visits can retain interaction state; it must not filter or target
	// files outside this new cohort. The caller owns restoring the saved visit.
	g.ClearSelection()
	g.clearSearch()
	g.subset = map[string]bool{}
	g.subsetOccurrences = map[fileidentity.Occurrence]bool{}
	g.subsetLabel = label
	g.onSubsetBack = back
	for _, member := range members {
		g.subset[member.Path] = true
		g.subsetOccurrences[member] = true
	}
	g.applyFilter()
	g.Toggle()
}

func (g *Overview) SetOnSubsetOpen(open func(Visit)) { g.onSubsetOpen = open }

// OpenUnassigned adds the selection-only analysis action to this subset visit.
func (g *Overview) OpenUnassigned(paths []string, back, analyze func()) {
	g.OpenSubset(paths, back)
	g.onAnalyze = analyze
	g.syncTopBar()
}

// OpenSubset opens a cohort by stable source identity, retaining root indexes
// for selection, comparison, copy, deletion and normal image loading.
func (g *Overview) OpenSubset(paths []string, back func()) {
	g.Close()
	g.subset = map[string]bool{}
	g.onSubsetBack = back
	for _, path := range paths {
		g.subset[path] = true
	}
	g.applyFilter()
	g.Toggle()
}
