package ui

import (
	"slices"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/dupes"
	"github.com/frathe/picfetch/internal/fileidentity"
)

// collectionData is immutable after publication. Selection observations share
// it, so navigation never rebuilds membership or its occurrence indexes.
type collectionData struct {
	files, source []fyne.URI
	retained      []collectionSource
	fileSet       dupes.Snapshot
	occurrences   fileidentity.Index
	favorite      string
}

// collectionInput is a fully prepared collection. Preparation owns source I/O
// and ordering; the model copies these values and publishes them together.
type collectionInput struct {
	source, display []fyne.URI
	retained        []collectionSource
	index           int
	favorite        string
}

type collectionChange struct {
	before, after collectionSnapshot
}

type collectionSnapshot struct {
	data  *collectionData
	index int
}

// A bookmark uses paths, whereas duplicate/cache keys use complete URI strings.
// Its generation is required: the same path and ordinal in a replacement is a
// different collection occurrence. Changes explicitly remap surviving bookmarks.
type collectionBookmark struct {
	generation uint64
	occurrence fileidentity.Occurrence
}

func (s collectionSnapshot) FileSet() dupes.Snapshot {
	if s.data == nil {
		return dupes.Snapshot{}
	}
	return s.data.fileSet
}

func (s collectionSnapshot) Generation() uint64 { return s.FileSet().Generation() }
func (s collectionSnapshot) Count() int         { return s.FileSet().Count() }

func (s collectionSnapshot) FileAt(i int) fyne.URI { return s.data.files[i] }

func (s collectionSnapshot) SourceFiles() []fyne.URI {
	if s.data == nil {
		return nil
	}
	return slices.Clone(s.data.source)
}

func (s collectionSnapshot) Retained() []collectionSource {
	if s.data == nil {
		return nil
	}
	return slices.Clone(s.data.retained)
}

func (s collectionSnapshot) Favorite() string {
	if s.data == nil {
		return ""
	}
	return s.data.favorite
}

func (s collectionSnapshot) DisplayFiles() []fyne.URI {
	if s.data == nil {
		return nil
	}
	return slices.Clone(s.data.files)
}

func (s collectionSnapshot) Current() (fyne.URI, int, bool) {
	if s.index < 0 || s.index >= s.Count() {
		return nil, 0, false
	}
	return s.FileAt(s.index), s.index, true
}

func (s collectionSnapshot) Bookmark(i int) (collectionBookmark, bool) {
	if i < 0 || i >= s.Count() || s.FileAt(i) == nil {
		return collectionBookmark{}, false
	}
	occurrence, ok := s.data.occurrences.Capture(s.FileAt(i).Path(), i)
	return collectionBookmark{s.Generation(), occurrence}, ok
}

func (s collectionSnapshot) Resolve(bookmark collectionBookmark) int {
	if s.data == nil || bookmark.generation != s.Generation() {
		return -1
	}
	return s.data.occurrences.Resolve(bookmark.occurrence)
}

// Observe is safe from any goroutine. Its order, requested occurrence and URI
// and path indexes always describe one publication, including after later writes.
func (s *appState) Observe() collectionSnapshot {
	if p := s.published.Load(); p != nil {
		return *p
	}
	return collectionSnapshot{}
}

// Select wraps ordinary navigation within the browsable order. It changes only
// the small observation; collection identity and worker admission stay intact.
func (s *appState) Select(i int) bool {
	observation := s.Observe()
	count := observation.Count()
	if count == 0 {
		return false
	}
	s.index = ((i % count) + count) % count
	observation.index = s.index
	s.published.Store(&observation)
	return true
}

func (s *appState) Replace(input collectionInput) collectionChange {
	before := s.Observe()
	s.unsortedFiles = slices.Clone(input.source)
	s.files = slices.Clone(input.display)
	s.unavailableOrder = slices.Clone(input.retained)
	s.favoriteDir = input.favorite
	if len(s.files) == 0 && len(s.unavailableOrder) == 0 {
		s.favoriteDir = ""
	}
	s.index = 0
	if count := len(s.files); count > 0 {
		s.index = ((input.index % count) + count) % count
	}
	s.publish()
	return collectionChange{before: before, after: s.Observe()}
}

func (s *appState) Clear() collectionChange { return s.Replace(collectionInput{}) }
