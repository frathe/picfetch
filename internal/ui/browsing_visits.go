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
	browsingReturnMap
)

type browsingVisit struct {
	binding     browsingBinding
	surface     browsingSurface
	grid        *grid.Visit
	origin      *browsingOrigin
	order       []string
	occurrences []fileidentity.Occurrence
}

type browsingOrigin struct {
	grid  grid.Visit
	image fileidentity.Occurrence
}

func (o browsingOrigin) remap(survivors map[fileidentity.Occurrence]fileidentity.Occurrence) browsingOrigin {
	if survivors == nil {
		return o
	}
	o.grid = o.grid.RemapOccurrences(survivors)
	if next, ok := survivors[o.image]; ok {
		o.image = next
	} else {
		// Keep the source as a fallback, but do not mistake a shifted ordinal
		// for the removed exact occurrence.
		o.image.Ordinal = -1
	}
	return o
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

func (b *browsingVisits) enterLocation(generation uint64) {
	b.stack = nil
	b.push(browsingLocationMap, browsingMap, generation)
}

func (b *browsingVisits) openLocationImage(binding browsingBinding, generation uint64, selected fileidentity.Occurrence, discovered []fileidentity.Occurrence) bool {
	if !b.matches(binding, generation) || binding.kind != browsingLocationMap || !slices.Contains(discovered, selected) {
		return false
	}
	b.push(browsingLocation, browsingImage, generation)
	b.stack[len(b.stack)-1].occurrences = slices.Clone(discovered)
	return true
}

func (b *browsingVisits) leaveLocation() {
	if b.has(browsingLocationMap) {
		b.stack = nil
		b.revision++
	}
}

func (b *browsingVisits) openLocationCluster(binding browsingBinding, generation uint64, members []fileidentity.Occurrence, origin grid.Visit) bool {
	if !b.matches(binding, generation) || binding.kind != browsingLocationMap || len(members) == 0 {
		return false
	}
	b.push(browsingCluster, browsingGrid, generation)
	visit := &b.stack[len(b.stack)-1]
	visit.occurrences = slices.Clone(members)
	visit.origin = &browsingOrigin{grid: origin.Clone()}
	return true
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
	if !b.matches(binding, generation) || binding.kind != browsingExplorer && binding.kind != browsingSearch && binding.kind != browsingCluster {
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
	if !b.matches(binding, generation) {
		return browsingReturn{}, false
	}
	allowed := false
	switch binding.kind {
	case browsingExplorer, browsingSearch, browsingCluster:
		allowed = destination == browsingReturnGrid || destination == browsingReturnParent
	case browsingLocation:
		allowed = destination == browsingReturnParent
	case browsingLocationMap:
		allowed = destination == browsingReturnMap
	default:
		return browsingReturn{}, false
	}
	if !allowed {
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
	case browsingReturnMap:
		b.stack[len(b.stack)-1].surface = browsingMap
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
		if origin := b.stack[i].origin; origin != nil && survivors != nil {
			remapped := origin.remap(survivors)
			b.stack[i].origin = &remapped
		}
		if survivors != nil && b.stack[i].occurrences != nil {
			var remapped []fileidentity.Occurrence
			for _, identity := range b.stack[i].occurrences {
				if next, ok := survivors[identity]; ok {
					remapped = append(remapped, next)
				}
			}
			b.stack[i].occurrences = remapped
		}
	}
	b.revision++
}
