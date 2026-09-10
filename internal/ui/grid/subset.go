package grid

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
