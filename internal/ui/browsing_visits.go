package ui

import (
	"github.com/frathe/picfetch/internal/fileidentity"
	"github.com/frathe/picfetch/internal/ui/grid"
)

type browsingSurface uint8

const (
	browsingImage browsingSurface = iota
	browsingGrid
	browsingMap
)

type browsingDestination uint8

const (
	browsingReturnGrid browsingDestination = iota
	browsingReturnParent
)

type browsingVisit struct {
	binding browsingBinding
	surface browsingSurface
	grid    *grid.Visit
}

// browsingVisits owns navigation authority, not feature data or visibility.
// Retained parents survive image and covering-interaction transitions. Each
// captured return belongs to both its collection and its transition revision.
type browsingVisits struct {
	serial   uint64
	revision uint64
	stack    []browsingVisit
}

type browsingReturn struct {
	source      browsingBinding
	revision    uint64
	destination browsingDestination
	grid        *grid.Visit
}

func (b *browsingVisits) current() browsingVisit {
	if len(b.stack) == 0 {
		return browsingVisit{}
	}
	return b.stack[len(b.stack)-1]
}

func (b *browsingVisits) has(kind browsingKind) bool {
	for _, visit := range b.stack {
		if visit.binding.kind == kind {
			return true
		}
	}
	return false
}

func (b *browsingVisits) push(kind browsingKind, surface browsingSurface, generation uint64) {
	b.serial++
	b.revision++
	b.stack = append(b.stack, browsingVisit{
		binding: browsingBinding{kind: kind, collection: generation, visit: b.serial}, surface: surface,
	})
}

func (b *browsingVisits) enterExplorer(generation uint64) {
	b.stack = nil
	b.push(browsingExplorerMap, browsingMap, generation)
}

func (b *browsingVisits) openExplorerCohort(generation uint64) browsingBinding {
	if !b.has(browsingExplorerMap) {
		b.enterExplorer(generation)
	} else {
		b.stack = b.stack[:1]
	}
	b.push(browsingExplorer, browsingGrid, generation)
	return b.current().binding
}

func (b *browsingVisits) matches(binding browsingBinding, generation uint64) bool {
	return binding.visit != 0 && binding.collection == generation && b.current().binding == binding
}

func (b *browsingVisits) openImage(binding browsingBinding, generation uint64, bookmark grid.Visit) bool {
	if !b.matches(binding, generation) || b.current().binding.kind != browsingExplorer {
		return false
	}
	visit := &b.stack[len(b.stack)-1]
	visit.surface, visit.grid = browsingImage, &bookmark
	b.revision++
	return true
}

func (b *browsingVisits) planReturn(binding browsingBinding, generation uint64, destination browsingDestination) (browsingReturn, bool) {
	if !b.matches(binding, generation) || binding.kind != browsingExplorer {
		return browsingReturn{}, false
	}
	if destination != browsingReturnGrid && destination != browsingReturnParent {
		return browsingReturn{}, false
	}
	return browsingReturn{source: binding, revision: b.revision, destination: destination, grid: b.current().grid}, true
}

func (b *browsingVisits) commitReturn(plan browsingReturn, generation uint64) bool {
	if !b.matches(plan.source, generation) || plan.revision != b.revision {
		return false
	}
	switch plan.destination {
	case browsingReturnGrid:
		b.stack[len(b.stack)-1].surface = browsingGrid
	case browsingReturnParent:
		b.stack = b.stack[:len(b.stack)-1]
	default:
		return false
	}
	b.revision++
	return true
}

func (b *browsingVisits) leaveExplorer() {
	if b.has(browsingExplorerMap) {
		b.stack = nil
		b.revision++
	}
}

// Rebinding is a committed collection effect, not permission to reveal a view.
func (b *browsingVisits) rebind(generation uint64) {
	b.reconcile(generation, nil)
}

func (b *browsingVisits) reconcile(generation uint64, survivors map[fileidentity.Occurrence]fileidentity.Occurrence) {
	for i := range b.stack {
		b.stack[i].binding.collection = generation
		if bookmark := b.stack[i].grid; bookmark != nil && survivors != nil {
			remapped := bookmark.RemapOccurrences(survivors)
			b.stack[i].grid = &remapped
		}
	}
	b.revision++
}
