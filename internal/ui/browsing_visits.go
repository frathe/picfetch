package ui

import (
	"slices"

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
	origin  *browsingOrigin
	order   []string
}

type browsingOrigin struct {
	grid  grid.Visit
	image fileidentity.Occurrence
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
	origin      *browsingOrigin
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

func (b *browsingVisits) enterSearch(generation uint64, origin browsingOrigin) (browsingBinding, bool) {
	if b.has(browsingSearch) {
		return browsingBinding{}, false
	}
	origin.grid = origin.grid.Clone()
	b.push(browsingSearch, browsingGrid, generation)
	b.stack[len(b.stack)-1].origin = &origin
	return b.current().binding, true
}

// detachSearch is the committed-source/close path: retirement is independent
// of permission to present the origin. User Exit instead validates a return.
func (b *browsingVisits) detachSearch() (browsingOrigin, bool) {
	visit := b.current()
	if visit.binding.kind != browsingSearch {
		return browsingOrigin{}, false
	}
	origin := *visit.origin
	b.stack = b.stack[:len(b.stack)-1]
	b.revision++
	return origin, true
}

func (b *browsingVisits) openImage(binding browsingBinding, generation uint64, bookmark grid.Visit) bool {
	if !b.matches(binding, generation) || binding.kind != browsingExplorer && binding.kind != browsingSearch {
		return false
	}
	visit := &b.stack[len(b.stack)-1]
	captured := bookmark.Clone()
	visit.surface, visit.grid = browsingImage, &captured
	if binding.kind == browsingSearch {
		visit.order = slices.Clone(bookmark.Results)
	}
	b.revision++
	return true
}

func (b *browsingVisits) planReturn(binding browsingBinding, generation uint64, destination browsingDestination) (browsingReturn, bool) {
	if !b.matches(binding, generation) || binding.kind != browsingExplorer && binding.kind != browsingSearch {
		return browsingReturn{}, false
	}
	if destination != browsingReturnGrid && destination != browsingReturnParent {
		return browsingReturn{}, false
	}
	return browsingReturn{source: binding, revision: b.revision, destination: destination, grid: b.current().grid, origin: b.current().origin}, true
}

func (b *browsingVisits) commitReturn(plan browsingReturn, generation uint64) bool {
	if !b.matches(plan.source, generation) || plan.revision != b.revision {
		return false
	}
	switch plan.destination {
	case browsingReturnGrid:
		b.stack[len(b.stack)-1].surface = browsingGrid
		b.stack[len(b.stack)-1].order = nil
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
