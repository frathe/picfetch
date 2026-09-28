package ui

import (
	"slices"
	"sync/atomic"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/dupes"
	"github.com/frathe/picfetch/internal/fileidentity"
	"github.com/frathe/picfetch/internal/filesort"
)

type collectionSource struct {
	uri         fyne.URI
	unavailable bool
}

type appState struct {
	files         []fyne.URI
	unsortedFiles []fyne.URI
	// Retained for saved collections while a system decoder is unavailable.
	unavailableOrder []collectionSource
	index            int
	favoriteDir      string
	sortMode         filesort.Mode
	mergeMode        bool

	// Readers retain immutable collection data and selection together. Membership
	// writes advance generation; Select shares the data without changing it.
	published atomic.Pointer[collectionSnapshot]
}

func newAppState(sortMode filesort.Mode, mergeMode bool) appState {
	return appState{sortMode: sortMode, mergeMode: mergeMode}
}

// publish replaces the published snapshot with one built from the
// current files, at the next generation. Mutators call it last; nothing
// else may.
func (s *appState) publish() {
	s.publishGeneration(s.Observe().Generation() + 1)
}

func (s *appState) publishGeneration(generation uint64) {
	keys := make([]string, len(s.files))
	for i, u := range s.files {
		if u != nil {
			keys[i] = u.String()
		}
	}

	files := slices.Clone(s.files)
	retained := slices.Clone(s.unavailableOrder)
	if retained == nil {
		for _, uri := range s.unsortedFiles {
			retained = append(retained, collectionSource{uri: uri})
		}
	}
	snap := collectionSnapshot{data: &collectionData{
		files: files, source: slices.Clone(s.unsortedFiles), retained: retained, favorite: s.favoriteDir,
		fileSet: dupes.NewSnapshot(keys, generation),
		occurrences: fileidentity.NewIndex(len(files), func(i int) string {
			if files[i] != nil {
				return files[i].Path()
			}
			return ""
		}),
	}, index: s.index}
	s.published.Store(&snap)
}

// snapshot is the current published view of the file set. Safe from any
// goroutine; it is the only read of the file set that is.
func (s *appState) snapshot() dupes.Snapshot {
	return s.Observe().FileSet()
}

func (s *appState) SortMode() filesort.Mode {
	return s.sortMode
}

func (s *appState) SetSortMode(mode filesort.Mode) {
	s.sortMode = mode
}

func (s *appState) MergeMode() bool {
	return s.mergeMode
}

func (s *appState) SetMergeMode(on bool) {
	s.mergeMode = on
}

func (s *appState) setFiles(unsorted, files []fyne.URI) {
	s.unsortedFiles = append([]fyne.URI(nil), unsorted...)
	s.files = append([]fyne.URI(nil), files...)
	s.publish()
}

func (s *appState) replaceFiles(unsorted, files []fyne.URI) {
	s.index = 0
	s.setFiles(unsorted, files)
}

func (s *appState) clearFiles() {
	s.Clear()
}

// Stable sorting preserves each repeated URI's occurrence ordinal across displayed,
// unsorted and retained orders even when other sources move around it.
func (s *appState) fileOccurrence(i int) int {
	key := s.files[i].String()
	occurrence := 0
	for _, uri := range s.files[:i+1] {
		if uri.String() == key {
			occurrence++
		}
	}
	return occurrence
}

func (s *appState) retainOrder(order []collectionSource) {
	s.unavailableOrder = nil
	for _, source := range order {
		if source.unavailable {
			s.unavailableOrder = slices.Clone(order)
			break
		}
	}
	// Legacy retained-order adapter. File-set generation changes with the
	// accompanying membership operation, not this value-only observation refresh.
	s.publishGeneration(s.Observe().Generation())
}
