// Package visualsearch owns ranked reference visits and their search lifetime.
package visualsearch

import (
	"slices"

	"github.com/frathe/picfetch/internal/similarity"
	"github.com/frathe/picfetch/internal/ui/grid"
)

// Visit retains source identities and browsing state without decoded images.
type Visit struct {
	ReferencePath string
	Paths         []string
	Grid          grid.Visit
	ImagePath     string
}

// Host owns presentation and cross-feature navigation on the UI goroutine.
type Host interface {
	CaptureVisit() Visit
	Present(Visit, grid.Progress)
	Restore(Visit, bool)
	Changed()
	Failed(error)
}

// UIQueue serializes worker delivery and supports deterministic test settlement.
type UIQueue interface {
	Do(func())
	Drain() bool
}

type Options struct {
	Provider similarity.SearchProvider
	Queue    UIQueue
}

// StartRequest fixes the original scope and waits for prior native retirement.
type StartRequest struct {
	Paths         []string
	ReferencePath string
	Origin        Visit
	Cache         similarity.CachePolicy
	After         <-chan struct{}
}

type State struct {
	Active, Pending, Preparing bool
	Visit                      Visit
	Progress                   grid.Progress
}

// Feature is UI-owned. Workers capture immutable requests and deliver through
// Queue; Wait and Settle are the only blocking lifecycle observations.
type Feature struct {
	host                           Host
	provider                       similarity.SearchProvider
	ui                             UIQueue
	active, pending, preparing     bool
	stopped, awaiting, queryFailed bool
	cacheWarned                    bool
	scope                          []string
	origin                         Visit
	history                        []Visit
	cache                          similarity.CachePolicy
	progress                       grid.Progress
	reference                      string
	sessionID, queryID, revision   uint64
	producer                       *producer
	cacheRevision                  uint64
	cachePending                   bool
	lastQuery                      similarity.SearchQuery
	retired                        []*producer
}

func New(host Host, options Options) *Feature {
	f := &Feature{host: host}
	f.Configure(options)
	return f
}

// Configure replaces adapters before admission. Retired producers must be
// joined and their queued delivery drained before replacing Queue.
func (f *Feature) Configure(options Options) {
	f.provider, f.ui = options.Provider, options.Queue
	if f.provider == nil {
		f.provider = (similarity.Client{}).Search
	}
	if f.ui == nil {
		f.ui = featureQueue{}
	}
}

// SetCachePolicy changes the policy used by the next explicitly started worker.
// A changed persistence preference retires current writes while retaining visits.
func (f *Feature) SetCachePolicy(policy similarity.CachePolicy) {
	if f.cache.FavoriteEnabled != policy.FavoriteEnabled || f.cache.LooseEnabled != policy.LooseEnabled || f.cache.Roots != policy.Roots || policy.GeneralLimitBytes < f.cache.GeneralLimitBytes {
		f.Suspend()
	}
	f.cache = policy
}

func (f *Feature) Start(request StartRequest) bool {
	if f.stopped || request.ReferencePath == "" || !slices.Contains(request.Paths, request.ReferencePath) {
		return false
	}
	f.clear()
	f.active = true
	f.scope = slices.Clone(request.Paths)
	f.origin = cloneVisit(request.Origin)
	f.cache = request.Cache
	f.progress = grid.Progress{Total: len(f.scope)}
	f.preparing = true
	f.beginProducer(request.After)
	f.query(request.ReferencePath)
	return true
}

// Explore changes reference within the captured scope, reusing a live producer.
func (f *Feature) Explore(path string) bool {
	if !f.active || f.stopped || !f.Contains(path) {
		return false
	}
	f.capture()
	if f.producer == nil {
		f.progress = grid.Progress{Total: len(f.scope)}
		f.preparing = true
		f.beginProducer(nil)
	}
	f.query(path)
	return true
}

func (f *Feature) query(path string) {
	f.queryID++
	f.reference = path
	f.pending, f.awaiting, f.queryFailed = true, true, false
	query := similarity.SearchQuery{ID: f.queryID, ReferencePath: path, CacheRevision: f.cacheRevision}
	f.lastQuery = query
	f.enqueueQuery(query)
	f.host.Changed()
}

func (f *Feature) enqueueQuery(query similarity.SearchQuery) {
	// The UI is the only sender. Replace an unconsumed query without blocking.
	select {
	case <-f.producer.queries:
	default:
	}
	f.producer.queries <- query
}

// FavoriteSaved admits persistence after an explicit committed save. The latest
// query carries its cache revision so rapid Explore/save notifications coalesce
// without dropping either intent. Settle observes the worker acknowledgement.
func (f *Feature) FavoriteSaved() {
	if !f.active || f.stopped || f.producer == nil {
		return
	}
	f.cacheRevision++
	f.cachePending = true
	f.lastQuery.CacheRevision = f.cacheRevision
	f.enqueueQuery(f.lastQuery)
}

// Back abandons a pending reference or restores a frozen successful visit. It
// never sends a query or starts a producer.
func (f *Feature) Back() bool {
	if !f.active {
		return false
	}
	wasPending := f.pending
	f.capture()
	f.invalidateQuery()
	if !wasPending && len(f.history) > 0 {
		f.history = f.history[:len(f.history)-1]
	}
	if len(f.history) == 0 {
		f.Exit()
		return true
	}
	last := f.history[len(f.history)-1]
	f.host.Restore(cloneVisit(last), false)
	f.host.Changed()
	return true
}

func (f *Feature) Exit() {
	if !f.active {
		return
	}
	origin := cloneVisit(f.origin)
	f.clear()
	f.host.Restore(origin, true)
	f.host.Changed()
}

// Active observes admission without copying a potentially large saved visit.
func (f *Feature) Active() bool { return f.active }

func (f *Feature) State() State {
	state := State{Active: f.active, Pending: f.pending, Preparing: f.preparing, Progress: f.progress}
	if len(f.history) > 0 {
		state.Visit = cloneVisit(f.history[len(f.history)-1])
	} else if f.active {
		state.Visit = Visit{ReferencePath: f.reference}
	}
	return state
}

// Scope returns the original source universe, independently of ranked results.
func (f *Feature) Scope() []string           { return slices.Clone(f.scope) }
func (f *Feature) Contains(path string) bool { return slices.Contains(f.scope, path) }

// CaptureGrid preserves the ranked browsing anchor before opening an image.
func (f *Feature) CaptureGrid(visit grid.Visit) {
	if len(f.history) > 0 {
		f.history[len(f.history)-1].Grid = cloneGrid(visit)
	}
}

func (f *Feature) capture() {
	if len(f.history) == 0 {
		return
	}
	captured := f.host.CaptureVisit()
	last := &f.history[len(f.history)-1]
	last.Grid = cloneGrid(captured.Grid)
	last.ImagePath = captured.ImagePath
	// Presentation can lag behind publication while an image/modal owns input.
	// Preserve the latest ranked paths even when the captured surface is older.
}

func (f *Feature) invalidateQuery() {
	f.queryID++
	f.reference = ""
	f.pending, f.awaiting, f.queryFailed = false, false, false
}

func (f *Feature) clear() {
	f.retire()
	f.invalidateQuery()
	f.active, f.preparing = false, false
	f.scope, f.history = nil, nil
	f.origin = Visit{}
	f.progress = grid.Progress{}
}

func cloneVisit(visit Visit) Visit {
	visit.Paths = slices.Clone(visit.Paths)
	visit.Grid = cloneGrid(visit.Grid)
	return visit
}

func cloneGrid(visit grid.Visit) grid.Visit {
	visit.Paths = slices.Clone(visit.Paths)
	visit.Results = slices.Clone(visit.Results)
	visit.Selected = slices.Clone(visit.Selected)
	visit.Subset = slices.Clone(visit.Subset)
	return visit
}
