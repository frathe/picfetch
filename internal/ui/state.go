package ui

import (
	"sync/atomic"

	"github.com/frathe/picfetch/internal/filesort"
)

type appState struct {
	sortMode  filesort.Mode
	mergeMode bool

	// Readers retain immutable collection data and selection together. Membership
	// writes advance generation; Select shares the data without changing it.
	published atomic.Pointer[collectionSnapshot]
}

func newAppState(sortMode filesort.Mode, mergeMode bool) appState {
	return appState{sortMode: sortMode, mergeMode: mergeMode}
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
