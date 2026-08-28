package ui

import (
	"sync/atomic"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/dupes"
	"github.com/frathe/picfetch/internal/filesort"
)

type appState struct {
	files         []fyne.URI
	unsortedFiles []fyne.URI
	index         int
	sortMode      filesort.Mode
	mergeMode     bool

	// published is the immutable {keys, generation} view of files that
	// readers off the UI goroutine use instead of touching the slice -
	// internal/dupes, whose Model is read from hashing workers while the
	// UI goroutine replaces files underneath them.
	//
	// Every write to files republishes under a bumped generation as its
	// last act, so the two move together. The revision counter this
	// replaces was separate from the list it described, and in finishSort
	// it advanced before the files did: a worker could see the new
	// generation over the old list.
	published atomic.Pointer[dupes.Snapshot]
}

func newAppState(sortMode filesort.Mode, mergeMode bool) appState {
	return appState{sortMode: sortMode, mergeMode: mergeMode}
}

// publish replaces the published snapshot with one built from the
// current files, at the next generation. Mutators call it last; nothing
// else may.
func (s *appState) publish() {
	var gen uint64
	if prev := s.published.Load(); prev != nil {
		gen = prev.Generation()
	}

	keys := make([]string, len(s.files))
	for i, u := range s.files {
		if u != nil {
			keys[i] = u.String()
		}
	}

	snap := dupes.NewSnapshot(keys, gen+1)
	s.published.Store(&snap)
}

// snapshot is the current published view of the file set. Safe from any
// goroutine; it is the only read of the file set that is.
func (s *appState) snapshot() dupes.Snapshot {
	if p := s.published.Load(); p != nil {
		return *p
	}

	return dupes.Snapshot{}
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
	s.setFiles(unsorted, files)
	s.index = 0
}

// reorder replaces files with an already-sorted list of the same members,
// leaving unsortedFiles and index alone. It exists so a reorder goes
// through a mutator like every other write does, rather than assigning
// the field directly and skipping publish.
func (s *appState) reorder(files []fyne.URI) {
	s.files = append([]fyne.URI(nil), files...)
	s.publish()
}

func (s *appState) clearFiles() {
	s.files = nil
	s.unsortedFiles = nil
	s.index = 0
	s.publish()
}

func (s *appState) removeFile(i int) fyne.URI {
	target := s.files[i]
	s.files = append(s.files[:i], s.files[i+1:]...)
	if s.index >= len(s.files) {
		s.index = len(s.files) - 1
	}

	for j, u := range s.unsortedFiles {
		if u.String() == target.String() {
			s.unsortedFiles = append(s.unsortedFiles[:j], s.unsortedFiles[j+1:]...)
			break
		}
	}

	s.publish()

	return target
}
