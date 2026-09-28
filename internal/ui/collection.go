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

// collectionInput carries prepared orders and a candidate association. Replace
// takes full membership; Merge takes source/retained additions and a full display
// order. Preparation owns source I/O; the model copies and publishes the values.
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

type collectionOrder uint8

const (
	collectionSourceOrder collectionOrder = iota
	collectionDisplayOrder
)

func (s collectionSnapshot) Capture(order collectionOrder) []fyne.URI {
	if order == collectionSourceOrder {
		return s.captureFiles(s.SourceFiles())
	}
	return s.captureFiles(s.DisplayFiles())
}

// Attach unavailable members to their preceding surviving occurrence. Source
// capture restores session order; display capture moves each gap with its anchor.
// URI ordinals keep repeated sources' gaps distinct without resurrecting removals.
func (s collectionSnapshot) captureFiles(files []fyne.URI) []fyne.URI {
	available := make(map[string]int, len(files))
	for _, uri := range files {
		available[uri.String()]++
	}
	type position struct {
		key        string
		occurrence int
	}
	after := make(map[position][]fyne.URI)
	occurrences := make(map[string]int)
	anchor := position{}
	for _, source := range s.Retained() {
		key := source.uri.String()
		if source.unavailable {
			after[anchor] = append(after[anchor], source.uri)
		} else if available[key] > 0 {
			occurrences[key]++
			if occurrences[key] <= available[key] {
				anchor = position{key, occurrences[key]}
			}
		}
	}
	clear(occurrences)
	result := append([]fyne.URI(nil), after[position{}]...)
	for _, uri := range files {
		key := uri.String()
		occurrences[key]++
		anchor = position{key, occurrences[key]}
		result = append(result, uri)
		result = append(result, after[anchor]...)
	}
	return result
}

func (s collectionSnapshot) FileSet() dupes.Snapshot {
	if s.data == nil {
		return dupes.Snapshot{}
	}
	return s.data.fileSet
}

func (s collectionSnapshot) Generation() uint64 { return s.FileSet().Generation() }
func (s collectionSnapshot) Count() int         { return s.FileSet().Count() }

func (s collectionSnapshot) Occurrences() fileidentity.Index {
	if s.data == nil {
		return fileidentity.Index{}
	}
	return s.data.occurrences
}

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

// Reorder publishes an already-prepared stable order of the same membership.
// Selection is captured at commit, not when background sorting was admitted.
func (s *appState) Reorder(files []fyne.URI) collectionChange {
	before := s.Observe()
	bookmark, chosen := before.Bookmark(before.index)
	s.files = slices.Clone(files)
	if chosen {
		identities := fileidentity.NewIndex(len(files), func(i int) string {
			if files[i] != nil {
				return files[i].Path()
			}
			return ""
		})
		s.index = identities.Resolve(bookmark.occurrence)
	}
	s.publish()
	return collectionChange{before: before, after: s.Observe()}
}

// Merge takes additions in source/retained order and an already-prepared full
// display order. Appending committed membership belongs to the model.
func (s *appState) Merge(input collectionInput) collectionChange {
	before := s.Observe()
	if len(input.source) == 0 && len(input.retained) == 0 {
		return collectionChange{before: before, after: before}
	}
	if len(input.source) == 0 {
		input.display = before.DisplayFiles()
		input.index = before.index
	}
	if input.retained == nil {
		for _, uri := range input.source {
			input.retained = append(input.retained, collectionSource{uri: uri})
		}
	}
	input.source = append(before.SourceFiles(), input.source...)
	input.retained = append(before.Retained(), input.retained...)
	return s.Replace(input)
}
