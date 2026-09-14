package similarity

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"time"
)

// CacheRoots isolates derived image representations from model assets and user data.
type CacheRoots struct{ GeneralDir, FavoritesDir string }

// CachePolicy is captured when a worker is admitted.
type CachePolicy struct {
	Roots                         CacheRoots
	FavoriteEnabled, LooseEnabled bool
	GeneralLimitBytes             uint64
}

// SearchRequest fixes the comparison universe for the lifetime of one worker.
type SearchRequest struct {
	SessionID uint64
	Paths     []string
	Limit     int
	Cache     CachePolicy
}

// SearchQuery chooses a reference and carries the latest committed Favorite-save
// revision without replacing the worker's prepared scope.
type SearchQuery struct {
	CacheRevision uint64
	ID            uint64
	ReferencePath string
}
type SearchKind string

const (
	SearchProgress     SearchKind = "progress"
	SearchPartial      SearchKind = "partial"
	SearchFinal        SearchKind = "final"
	SearchReady        SearchKind = "ready"
	SearchQueryFailure SearchKind = "query-failure"
	SearchFailure      SearchKind = "failure"
)

// SearchEvent contains immutable source identities and no inference vectors.
type SearchEvent struct {
	CacheRevision                    uint64
	CachePressureBytes               uint64
	SessionID, QueryID, Revision     uint64
	Kind                             SearchKind
	Processed, Total, Failed, Reused int
	Matches                          []Match
	Error, CacheWarning              string
	OfflineVerified                  bool
}
type SearchProvider func(context.Context, SearchRequest, <-chan SearchQuery, func(SearchEvent)) error

const SearchVectorBudget = 256 * 1024 * 1024
const DefaultAnalysisCacheBytes = 2048 * 1024 * 1024

type searchPreparation func(context.Context, string) (Item, bool, error)

func runSearchSession(ctx context.Context, request SearchRequest, queries <-chan SearchQuery, prepare searchPreparation, validate func(context.Context, Item, []Match, bool) error, refresh func(context.Context, []Item), emit func(SearchEvent) error) error {
	paths := make([]string, 0, len(request.Paths))
	members := make(map[string]bool)
	for _, path := range request.Paths {
		if !filepath.IsAbs(path) {
			return fmt.Errorf("search source must be an absolute path")
		}
		path = filepath.Clean(path)
		if !members[path] {
			members[path] = true
			paths = append(paths, path)
		}
	}
	if len(paths) > SearchVectorBudget/(768*4) {
		return fmt.Errorf("search exceeds the vector memory budget")
	}
	limit := request.Limit
	if limit <= 0 || limit > 30 {
		limit = 30
	}
	event := SearchEvent{SessionID: request.SessionID, Total: len(paths), OfflineVerified: EnforcesNetworkIsolation()}
	items := make([]Item, 0, len(paths))
	prepared := make(map[string]Item, len(paths))
	var query SearchQuery
	var queryFailed bool
	cacheChanged := false
	var ranked []Item
	rankedAt := 0
	var lastProgress time.Time
	send := func(kind SearchKind) error {
		event.Kind = kind
		event.QueryID = query.ID
		event.Revision++
		snapshot := event
		snapshot.Matches = slices.Clone(event.Matches)
		return emit(snapshot)
	}
	accept := func(q SearchQuery) bool {
		if q.CacheRevision > event.CacheRevision {
			if refresh != nil {
				refresh(ctx, items)
			}
			event.CacheRevision = q.CacheRevision
			cacheChanged = true
		}
		if q.ID <= query.ID && query.ID != 0 {
			return false
		}
		q.ReferencePath = filepath.Clean(q.ReferencePath)
		query = q
		queryFailed = false
		ranked, rankedAt = nil, 0
		event.Error = ""
		event.Matches = nil
		return true
	}
	publish := func(final bool) error {
		if queryFailed {
			return nil
		}
		ref, ok := prepared[query.ReferencePath]
		if !ok || ref.Error != "" {
			if !ok && members[query.ReferencePath] {
				return nil
			}
			queryFailed = true
			event.Error = "reference image could not be prepared"
			return send(SearchQueryFailure)
		}
		// A fixed reference only needs its previous top-k plus newly prepared
		// candidates. Accepting a new reference resets this to one full pass.
		candidates := items[rankedAt:]
		if len(ranked) > 0 {
			candidates = append(slices.Clone(ranked), candidates...)
		}
		matches, err := RankSimilar(ctx, ref, candidates, limit)
		if err != nil {
			queryFailed = true
			event.Error = err.Error()
			return send(SearchQueryFailure)
		}
		if validate != nil {
			if err := validate(ctx, ref, matches, final); err != nil {
				return err
			}
		}
		ranked = ranked[:0]
		for _, match := range matches {
			ranked = append(ranked, prepared[match.Path])
		}
		rankedAt = len(items)
		event.Matches = matches
		if final {
			return send(SearchFinal)
		}
		return send(SearchPartial)
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case q, ok := <-queries:
		if !ok {
			return nil
		}
		accept(q)
	}
	cursor := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		changed := false
	drain:
		for queries != nil {
			select {
			case q, ok := <-queries:
				if !ok {
					queries = nil
					break drain
				}
				changed = accept(q) || changed
			default:
				break drain
			}
		}
		if cacheChanged {
			cacheChanged = false
			if event.Processed != event.Total {
				if err := send(SearchProgress); err != nil {
					return err
				}
				lastProgress = time.Now()
			}
		}
		if !members[query.ReferencePath] && !queryFailed {
			if err := publish(false); err != nil {
				return err
			}
		}
		if changed {
			if _, ok := prepared[query.ReferencePath]; ok {
				if err := publish(event.Processed == event.Total); err != nil {
					return err
				}
			}
		}
		if event.Processed == event.Total {
			if err := send(SearchReady); err != nil {
				return err
			}
			if queries == nil {
				return nil
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case q, ok := <-queries:
				if !ok {
					return nil
				}
				if accept(q) {
					if err := publish(true); err != nil {
						return err
					}
				}
			}
			continue
		}
		path := query.ReferencePath
		if _, ok := prepared[path]; ok || !members[path] {
			for cursor < len(paths) {
				path = paths[cursor]
				cursor++
				if _, ok := prepared[path]; !ok {
					break
				}
			}
		}
		item, reused, err := prepare(ctx, path)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		item.Path = path
		if err != nil {
			item.Error = err.Error()
		}
		if item.Error == "" {
			if _, err := RankSimilar(ctx, item, nil, 1); err != nil {
				item.Error = err.Error()
			}
		}
		if item.Error != "" {
			item.Embedding = nil
			event.Failed++
		} else if reused {
			event.Reused++
		}
		// Only vectors and source metadata remain resident; previews belong on disk.
		item.Preview = nil
		item.Tags = nil
		prepared[path] = item
		items = append(items, item)
		event.Processed++
		event.Matches = nil
		// Warm caches can prepare thousands of sources between UI frames.
		// Terminal/ranked events keep exact counts; transient counts have a
		// bounded cadence independent of collection size.
		if now := time.Now(); lastProgress.IsZero() || now.Sub(lastProgress) >= 100*time.Millisecond {
			if err := send(SearchProgress); err != nil {
				return err
			}
			lastProgress = now
		}
		if path == query.ReferencePath && item.Error != "" {
			if err := publish(false); err != nil {
				return err
			}
		}
		if event.Processed == event.Total || event.Processed%100 == 0 || changed && path == query.ReferencePath {
			if err := publish(event.Processed == event.Total); err != nil {
				return err
			}
		}
	}
}
