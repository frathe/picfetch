package visualsearch_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"testing/synctest"

	"github.com/frathe/picfetch/internal/fileidentity"
	"github.com/frathe/picfetch/internal/similarity"
	"github.com/frathe/picfetch/internal/ui/grid"
	"github.com/frathe/picfetch/internal/ui/visualsearch"
	"github.com/frathe/picfetch/internal/uitest"
)

type visitHost struct {
	current   visualsearch.Visit
	hold      bool
	presented []visualsearch.Visit
	restored  []visualsearch.Visit
	origins   []bool
	errors    []error
}

func (h *visitHost) CaptureVisit() visualsearch.Visit { return h.current }
func (h *visitHost) Present(v visualsearch.Visit, _ grid.Progress) {
	if !h.hold {
		h.current = v
	}
	h.presented = append(h.presented, v)
}
func (h *visitHost) Restore(v visualsearch.Visit, origin bool) {
	h.current = v
	h.restored = append(h.restored, v)
	h.origins = append(h.origins, origin)
}
func (*visitHost) Changed()           {}
func (h *visitHost) Failed(err error) { h.errors = append(h.errors, err) }

type providerCall struct {
	request similarity.SearchRequest
	queries <-chan similarity.SearchQuery
	emit    func(similarity.SearchEvent)
	exit    chan error
}

func heldProvider(calls chan<- providerCall) similarity.SearchProvider {
	return func(ctx context.Context, r similarity.SearchRequest, queries <-chan similarity.SearchQuery, emit func(similarity.SearchEvent)) error {
		call := providerCall{request: r, queries: queries, emit: emit, exit: make(chan error)}
		select {
		case calls <- call:
		case <-ctx.Done():
			return ctx.Err()
		}
		select {
		case err := <-call.exit:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func newSearch(t *testing.T) (*visualsearch.Feature, *visitHost, *uitest.UIQueue, chan providerCall) {
	t.Helper()
	h := &visitHost{}
	q := &uitest.UIQueue{}
	calls := make(chan providerCall, 8)
	f := visualsearch.New(h, visualsearch.Options{Provider: heldProvider(calls), Queue: q})
	t.Cleanup(func() { f.Stop(); f.Settle() })
	return f, h, q, calls
}

func publish(call providerCall, query similarity.SearchQuery, revision uint64, kind similarity.SearchKind, paths ...string) {
	event := similarity.SearchEvent{SessionID: call.request.SessionID, QueryID: query.ID, Revision: revision, Kind: kind, Processed: 2, Total: 3}
	for _, path := range paths {
		event.Matches = append(event.Matches, similarity.Match{Path: path})
	}
	call.emit(event)
}

func TestVisualSearchCachePressureRetainsOnlyCompletedProducer(t *testing.T) {
	for _, final := range []bool{false, true} {
		t.Run(fmt.Sprintf("final_%t", final), func(t *testing.T) {
			f, h, q, calls := newSearch(t)
			f.Start(visualsearch.StartRequest{Paths: []string{"/a", "/b", "/c"}, ReferencePath: "/a"})
			call := <-calls
			query := <-call.queries
			kind, processed := similarity.SearchPartial, 2
			if final {
				kind, processed = similarity.SearchFinal, 3
			}
			call.emit(similarity.SearchEvent{SessionID: call.request.SessionID, QueryID: query.ID, Revision: 1, Kind: kind, Processed: processed, Total: 3, CachePressureBytes: 100, Matches: []similarity.Match{{Path: "/b"}}})
			q.Drain()
			f.Settle()
			var pressure similarity.CachePressureError
			if len(h.errors) != 1 || !errors.As(h.errors[0], &pressure) || pressure.NeedBytes != 100 || !f.Active() {
				t.Fatalf("cache pressure lost its request or browsing: %v", h.errors)
			}
			if !f.Explore("/b") {
				t.Fatal("next reference was rejected")
			}
			if f.State().Preparing == final || (f.State().SessionID == call.request.SessionID) != final {
				t.Fatalf("completed=%t, next query repeated the wrong preparation lifetime: %+v", final, f.State())
			}
			if final {
				query = <-call.queries
				call.emit(similarity.SearchEvent{SessionID: call.request.SessionID, QueryID: query.ID, Revision: 2, Kind: similarity.SearchFinal, Processed: 3, Total: 3, CachePressureBytes: 100, Matches: []similarity.Match{{Path: "/a"}}})
				f.Settle()
				if len(h.errors) != 1 || len(calls) != 0 {
					t.Fatal("retained query repeated handled pressure or restarted the provider")
				}
			}
		})
	}
}

func TestVisualSearchCachePressureWriterQuiescence(t *testing.T) {
	for _, phase := range []string{"preparing", "ready", "favorite-pending"} {
		t.Run(phase, func(t *testing.T) {
			f, _, q, calls := newSearch(t)
			f.Start(visualsearch.StartRequest{Paths: []string{"/a", "/b", "/c"}, ReferencePath: "/a"})
			call := <-calls
			query := <-call.queries
			kind := similarity.SearchFinal
			if phase == "preparing" {
				kind = similarity.SearchPartial
			}
			publish(call, query, 1, kind, "/b")
			q.Drain()
			if phase == "favorite-pending" {
				f.FavoriteSaved()
			}
			<-f.CacheWritesRevoked()
			if !f.Explore("/b") {
				t.Fatal("explicit next reference rejected")
			}
			retained := f.State().SessionID == call.request.SessionID
			if retained != (phase == "ready") {
				t.Fatalf("writer quiescence: phase=%s retained=%t", phase, retained)
			}
		})
	}
}

func TestVisualSearchProgressiveVisitUsesRetainedProvider(t *testing.T) {
	f, h, q, calls := newSearch(t)
	scope := []string{"/a", "/b", "/c"}
	origin := visualsearch.Visit{Paths: scope, Image: fileidentity.Occurrence{Path: "/a"}}
	if !f.Start(visualsearch.StartRequest{Paths: scope, ReferencePath: "/a", Origin: origin}) {
		t.Fatal("valid search rejected")
	}
	scope[0] = "/changed"
	call := <-calls
	if !reflect.DeepEqual(call.request.Paths, []string{"/a", "/b", "/c"}) {
		t.Fatalf("scope aliased: %v", call.request.Paths)
	}
	query := <-call.queries
	publish(call, query, 1, similarity.SearchPartial, "/b")
	if len(h.presented) != 0 {
		t.Fatal("worker presented outside the UI queue")
	}
	q.Drain()
	if state := f.State(); state.Pending || !state.Preparing || !reflect.DeepEqual(state.Visit.Paths, []string{"/b"}) {
		t.Fatalf("partial state: %+v", state)
	}
	publish(call, query, 2, similarity.SearchFinal, "/c", "/b")
	f.Settle()
	if f.State().Preparing {
		t.Fatal("final query still preparing")
	}
	if !f.Explore("/b") {
		t.Fatal("second reference rejected")
	}
	if f.State().Preparing || !f.State().Pending {
		t.Fatalf("warm query lost prepared-collection status: %+v", f.State())
	}
	query = <-call.queries
	publish(call, query, 3, similarity.SearchFinal, "/a")
	f.Settle()
	if len(calls) != 0 {
		t.Fatal("warm query started another provider")
	}
	if !f.Back() || !reflect.DeepEqual(h.current.Paths, []string{"/c", "/b"}) {
		t.Fatalf("Back did not restore revised first visit: %+v", h.current)
	}
	if !f.Back() || f.State().Active || !reflect.DeepEqual(h.current.Paths, []string{"/a", "/b", "/c"}) {
		t.Fatalf("origin was not retained independently: %+v", h.current)
	}
	if !reflect.DeepEqual(h.origins, []bool{false, true}) {
		t.Fatalf("restoration kinds: %v", h.origins)
	}
}

func TestVisualSearchLifecyclePendingBackRejectsQueuedReferences(t *testing.T) {
	f, h, q, calls := newSearch(t)
	f.Start(visualsearch.StartRequest{Paths: []string{"/a", "/b", "/c"}, ReferencePath: "/a", Origin: visualsearch.Visit{Image: fileidentity.Occurrence{Path: "/origin"}}})
	call := <-calls
	first := <-call.queries
	publish(call, first, 1, similarity.SearchFinal, "/b", "/c")
	f.Settle()
	h.current.Grid = grid.Visit{Paths: []string{"/b", "/c"}, Results: []string{"/c"}, Selected: []string{"/c"}, Subset: []string{"/b", "/c"}, Query: "cat", Highlight: "/c", ScrollOffset: 75, Ranked: true, Visible: true}
	wantGrid := h.current.Grid
	f.Explore("/b")
	second := <-call.queries
	publish(call, first, 2, similarity.SearchPartial, "/old")
	publish(call, second, 3, similarity.SearchPartial, "/abandoned")
	if !f.Back() {
		t.Fatal("pending Back rejected")
	}
	q.Drain()
	if got := f.State().Visit; got.ReferencePath != "/a" || !reflect.DeepEqual(got.Paths, []string{"/b", "/c"}) {
		t.Fatalf("queued abandoned references replaced frozen visit: %+v", got)
	}
	if !reflect.DeepEqual(h.current.Grid, wantGrid) {
		t.Fatalf("browsing anchor was lost: %+v", h.current.Grid)
	}
	if len(h.presented) != 1 {
		t.Fatalf("stale callbacks reached presentation: %d", len(h.presented))
	}
	select {
	case query := <-call.queries:
		t.Fatalf("Back triggered inference: %+v", query)
	default:
	}
	f.Back()
	if h.current.Image.Path != "/origin" || !h.origins[len(h.origins)-1] {
		t.Fatal("pending query consumed a successful-history entry")
	}
}

func TestVisualSearchBackRetainsSessionCompletion(t *testing.T) {
	f, h, q, calls := newSearch(t)
	f.Start(visualsearch.StartRequest{Paths: []string{"/a", "/b", "/c"}, ReferencePath: "/a"})
	call := <-calls
	first := <-call.queries
	publish(call, first, 1, similarity.SearchPartial, "/b")
	q.Drain()
	h.current.Grid = grid.Visit{Query: "cat", ScrollOffset: 75, Ranked: true, Visible: true}
	wantGrid := h.current.Grid
	f.Explore("/b")
	second := <-call.queries
	call.emit(similarity.SearchEvent{SessionID: call.request.SessionID, QueryID: second.ID, Revision: 2, Kind: similarity.SearchFinal, Processed: 3, Total: 3, Matches: []similarity.Match{{Path: "/c"}}})
	f.Settle()
	completed := f.State().Progress
	if !f.Back() || f.State().Preparing || !f.State().Progress.Complete || f.State().Progress != completed {
		t.Fatalf("Back restored obsolete preparation status: %+v", f.State())
	}
	if h.current.ReferencePath != "/a" || !reflect.DeepEqual(h.current.Paths, []string{"/b"}) || !reflect.DeepEqual(h.current.Grid, wantGrid) {
		t.Fatalf("Back lost the frozen ranking or browsing state: %+v", h.current)
	}
}

func TestVisualSearchBackStillObservesPreparation(t *testing.T) {
	f, h, q, calls := newSearch(t)
	f.Start(visualsearch.StartRequest{Paths: []string{"/a", "/b", "/c"}, ReferencePath: "/a"})
	call := <-calls
	first := <-call.queries
	publish(call, first, 1, similarity.SearchPartial, "/b")
	q.Drain()
	f.Explore("/b")
	second := <-call.queries
	if !f.Back() {
		t.Fatal("pending Back rejected")
	}
	call.emit(similarity.SearchEvent{SessionID: call.request.SessionID, QueryID: second.ID, Revision: 2, Kind: similarity.SearchProgress, Processed: 3, Total: 3})
	publish(call, second, 3, similarity.SearchFinal, "/c")
	call.emit(similarity.SearchEvent{SessionID: call.request.SessionID, QueryID: second.ID, Revision: 4, Kind: similarity.SearchReady, Processed: 3, Total: 3})
	q.Drain()
	if state := f.State(); state.Preparing || !state.Progress.Complete || state.Progress.Processed != 3 {
		t.Fatalf("Back stopped observing its retained preparation: %+v", state)
	}
	if h.current.ReferencePath != "/a" || !reflect.DeepEqual(h.current.Paths, []string{"/b"}) || len(h.presented) != 1 {
		t.Fatalf("abandoned query changed the frozen visit: %+v", h.current)
	}
}

func TestVisualSearchLatestQueryRevisionAndImmutableDelivery(t *testing.T) {
	f, h, q, calls := newSearch(t)
	f.Start(visualsearch.StartRequest{Paths: []string{"/a", "/b", "/c"}, ReferencePath: "/a"})
	call := <-calls
	first := <-call.queries
	f.Explore("/b")
	second := <-call.queries
	paths := []similarity.Match{{Path: "/c"}, {Path: "/a"}}
	call.emit(similarity.SearchEvent{SessionID: call.request.SessionID, QueryID: second.ID, Revision: 3, Kind: similarity.SearchFinal, Matches: paths})
	paths[0].Path = "/mutated"
	publish(call, first, 4, similarity.SearchFinal, "/wrong-query")
	publish(call, second, 2, similarity.SearchPartial, "/old-revision")
	q.Drain()
	want := []string{"/c", "/a"}
	if got := f.State().Visit; got.ReferencePath != "/b" || !reflect.DeepEqual(got.Paths, want) {
		t.Fatalf("latest eligible publication lost: %+v", got)
	}
	h.current.Paths[0] = "/host-mutated"
	state := f.State()
	state.Visit.Paths[0] = "/state-mutated"
	scope := f.Scope()
	scope[0] = "/scope-mutated"
	if !reflect.DeepEqual(f.State().Visit.Paths, want) || !f.Contains("/a") || f.Contains("/scope-mutated") {
		t.Fatal("exported snapshots alias feature-owned slices")
	}
	f.Back()
	if f.State().Active {
		t.Fatal("superseded reference created a history entry")
	}
}

func TestVisualSearchHistoryBranchAndTwentyVisitLimit(t *testing.T) {
	f, h, _, calls := newSearch(t)
	paths := make([]string, 23)
	for i := range paths {
		paths[i] = fmt.Sprintf("/%02d", i)
	}
	f.Start(visualsearch.StartRequest{Paths: paths, ReferencePath: paths[0], Origin: visualsearch.Visit{Image: fileidentity.Occurrence{Path: "/independent-origin"}}})
	call := <-calls
	for i := 0; i < 22; i++ {
		if i > 0 {
			f.Explore(paths[i])
		}
		query := <-call.queries
		publish(call, query, uint64(i+1), similarity.SearchFinal, paths[(i+1)%len(paths)])
		f.Settle()
	}
	f.Back()
	if f.State().Visit.ReferencePath != paths[20] {
		t.Fatal("last successful reference did not restore")
	}
	f.Explore(paths[22])
	query := <-call.queries
	publish(call, query, 23, similarity.SearchFinal, paths[0])
	f.Settle()
	for i := 20; i >= 2; i-- {
		f.Back()
		if got := f.State().Visit.ReferencePath; got != paths[i] {
			t.Fatalf("history Back wanted %s, got %s", paths[i], got)
		}
	}
	f.Back()
	if f.State().Active || h.current.Image.Path != "/independent-origin" {
		t.Fatalf("eviction lost independent origin: %+v", h.current)
	}
}

func TestVisualSearchQueryFailurePreservesLastUsableVisit(t *testing.T) {
	f, h, q, calls := newSearch(t)
	f.Start(visualsearch.StartRequest{Paths: []string{"/a", "/b", "/c"}, ReferencePath: "/a", Origin: visualsearch.Visit{Image: fileidentity.Occurrence{Path: "/origin"}}})
	call := <-calls
	first := <-call.queries
	publish(call, first, 1, similarity.SearchPartial, "/b")
	q.Drain()
	call.emit(similarity.SearchEvent{SessionID: call.request.SessionID, QueryID: first.ID, Revision: 2, Kind: similarity.SearchQueryFailure, Processed: 2, Total: 3, Error: "reference changed"})
	f.Settle()
	if !f.State().Preparing || f.State().Pending || !reflect.DeepEqual(f.State().Visit.Paths, []string{"/b"}) {
		t.Fatalf("late failure discarded usable result or hid preparation: %+v", f.State())
	}
	f.Explore("/c")
	second := <-call.queries
	call.emit(similarity.SearchEvent{SessionID: call.request.SessionID, QueryID: second.ID, Revision: 3, Kind: similarity.SearchQueryFailure, Processed: 2, Total: 3, Error: "invalid reference"})
	publish(call, second, 4, similarity.SearchFinal, "/must-not-commit")
	f.Settle()
	if len(h.errors) != 2 || !reflect.DeepEqual(f.State().Visit.Paths, []string{"/b"}) {
		t.Fatalf("pending failure changed successful history: %+v", f.State())
	}
	var terminal visualsearch.SessionError
	if errors.As(h.errors[0], &terminal) {
		t.Fatal("recoverable query error classified as terminal")
	}
	f.Back()
	if f.State().Active || h.current.Image.Path != "/origin" {
		t.Fatal("failed query added a history entry")
	}
}

func TestVisualSearchLifecycleSuspendWaitsForWriterAndRetainsBrowsing(t *testing.T) {
	h := &visitHost{}
	q := &uitest.UIQueue{}
	started, canceled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	provider := func(ctx context.Context, request similarity.SearchRequest, queries <-chan similarity.SearchQuery, emit func(similarity.SearchEvent)) error {
		query := <-queries
		emit(similarity.SearchEvent{SessionID: request.SessionID, QueryID: query.ID, Revision: 1, Kind: similarity.SearchFinal, Matches: []similarity.Match{{Path: "/b"}}})
		close(started)
		<-ctx.Done()
		emit(similarity.SearchEvent{SessionID: request.SessionID, QueryID: query.ID, Revision: 2, Kind: similarity.SearchFinal, Matches: []similarity.Match{{Path: "/stale"}}})
		close(canceled)
		<-release
		return ctx.Err()
	}
	f := visualsearch.New(h, visualsearch.Options{Provider: provider, Queue: q})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
		f.Stop()
		f.Settle()
	})
	f.Start(visualsearch.StartRequest{Paths: []string{"/a", "/b"}, ReferencePath: "/a", Origin: visualsearch.Visit{Image: fileidentity.Occurrence{Path: "/origin"}}})
	<-started
	f.Settle()
	done := f.Suspend()
	<-canceled
	select {
	case <-done:
		t.Fatal("suspension completed while the retired provider could still write")
	default:
	}
	q.Drain()
	if !f.State().Active || !reflect.DeepEqual(f.State().Visit.Paths, []string{"/b"}) || len(h.presented) != 1 {
		t.Fatalf("suspension or stale publication changed browsing: %+v", f.State())
	}
	close(release)
	<-done
	f.Settle()
	if len(h.errors) != 0 {
		t.Fatalf("expected cancellation reported as failure: %v", h.errors)
	}
	f.Back()
	if h.current.Image.Path != "/origin" {
		t.Fatal("suspension lost origin")
	}
}

func TestVisualSearchLifecycleDetachOrigin(t *testing.T) {
	f, h, q, calls := newSearch(t)
	origin := visualsearch.Visit{
		Image: fileidentity.Occurrence{Path: "/a", Ordinal: 1},
		Grid:  grid.Visit{Selected: []string{"/a"}, Query: "saved", Visible: true},
	}
	f.Start(visualsearch.StartRequest{Paths: []string{"/a", "/b"}, ReferencePath: "/a", Origin: origin})
	call := <-calls
	query := <-call.queries
	publish(call, query, 1, similarity.SearchPartial, "/b")
	q.Drain()
	presentations := len(h.presented)
	publish(call, query, 2, similarity.SearchFinal, "/late")
	detached, active := f.DetachOrigin()
	if !active || !reflect.DeepEqual(detached, origin) || f.Active() {
		t.Fatal("detachment lost the origin or retained the search session")
	}
	f.Settle()
	if len(h.restored) != 0 || len(h.presented) != presentations {
		t.Fatal("detachment or retired delivery changed the surface before root reconciliation")
	}
	detached.Grid.Selected[0] = "/changed"
	if origin.Grid.Selected[0] != "/a" {
		t.Fatal("detached origin aliases its caller's bookmark")
	}
	if _, active := f.DetachOrigin(); active || f.Back() {
		t.Fatal("retired origin can be restored twice")
	}
	f.Start(visualsearch.StartRequest{Paths: []string{"/new"}, ReferencePath: "/new"})
	if !f.Active() {
		t.Fatal("detachment permanently stopped search admission")
	}
}

func TestVisualSearchCacheLimitChangesRetireCapturedPolicy(t *testing.T) {
	for _, limit := range []uint64{512, 1024, 2048} {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			f, _, queue, calls := newSearch(t)
			policy := similarity.CachePolicy{LooseEnabled: true, GeneralLimitBytes: 1024}
			f.Start(visualsearch.StartRequest{Paths: []string{"/a", "/b"}, ReferencePath: "/a", Cache: policy})
			first := <-calls
			query := <-first.queries
			publish(first, query, 1, similarity.SearchPartial, "/b")
			queue.Drain()
			policy.GeneralLimitBytes = limit
			f.SetCachePolicy(policy)
			changed := limit != first.request.Cache.GeneralLimitBytes
			state := f.State()
			if state.Preparing == changed {
				f.Stop()
				t.Fatalf("policy change retirement: changed=%v preparing=%v", changed, state.Preparing)
			}
			if !state.Active || !reflect.DeepEqual(state.Visit.Paths, []string{"/b"}) {
				t.Fatalf("policy update lost the current visit: %+v", state)
			}
			if changed {
				f.Settle()
				publish(first, query, 2, similarity.SearchFinal, "/stale")
				queue.Drain()
				if !reflect.DeepEqual(f.State().Visit.Paths, []string{"/b"}) {
					t.Fatal("retired policy applied stale results")
				}
			}
			if len(calls) != 0 || !f.Explore("/b") {
				t.Fatal("policy update did not preserve explicit restart admission")
			}
			if changed {
				second := <-calls
				if second.request.SessionID == first.request.SessionID || second.request.Cache != policy {
					t.Fatalf("restart retained the old cache policy: %+v", second.request)
				}
			} else {
				select {
				case <-first.queries:
				default:
					t.Fatal("unchanged policy lost its retained query lane")
				}
			}
		})
	}
}

func TestVisualSearchLifecycleSuspendRestartsOnlyForExplicitQuery(t *testing.T) {
	f, h, q, calls := newSearch(t)
	cache := similarity.CachePolicy{LooseEnabled: true, Roots: similarity.CacheRoots{GeneralDir: "/cache"}}
	f.Start(visualsearch.StartRequest{Paths: []string{"/a", "/b", "/c"}, ReferencePath: "/a", Cache: cache})
	first := <-calls
	query := <-first.queries
	publish(first, query, 1, similarity.SearchFinal, "/b", "/c")
	f.Settle()
	f.Explore("/b")
	query = <-first.queries
	publish(first, query, 2, similarity.SearchFinal, "/c")
	f.Settle()
	publish(first, query, 3, similarity.SearchPartial, "/stale")
	<-f.Suspend()
	f.Settle()
	f.Back()
	if len(calls) != 0 || f.State().Visit.ReferencePath != "/a" {
		t.Fatal("Back after suspension triggered preparation or lost history")
	}
	if !f.Explore("/c") {
		t.Fatal("explicit reference could not restart suspended session")
	}
	second := <-calls
	if first.request.SessionID == second.request.SessionID || !reflect.DeepEqual(second.request.Paths, []string{"/a", "/b", "/c"}) || second.request.Cache != cache {
		t.Fatalf("restart lost original worker contract: %+v", second.request)
	}
	newQuery := <-second.queries
	publish(second, newQuery, 1, similarity.SearchFinal, "/a")
	publish(first, query, 99, similarity.SearchFinal, "/retired-session")
	f.Settle()
	if !reflect.DeepEqual(h.current.Paths, []string{"/a"}) {
		t.Fatalf("retired session replaced restarted session: %+v", h.current)
	}
	publish(second, newQuery, 2, similarity.SearchPartial, "/after-stop")
	f.Stop()
	f.Wait()
	q.Drain()
	if f.Start(visualsearch.StartRequest{Paths: []string{"/a"}, ReferencePath: "/a"}) || f.Explore("/a") || f.Back() || f.State().Active {
		t.Fatal("terminal shutdown admitted new work")
	}
	if len(h.presented) != 3 {
		t.Fatalf("shutdown delivered stale pixels: %d", len(h.presented))
	}
}

func TestVisualSearchLifecycleCancellationBeforeAdmissionAndLatestBoundedQuery(t *testing.T) {
	f, _, _, calls := newSearch(t)
	after := make(chan struct{})
	request := visualsearch.StartRequest{Paths: []string{"/a", "/b", "/c"}, ReferencePath: "/a", After: after, Cache: similarity.CachePolicy{FavoriteEnabled: true}}
	f.Start(request)
	f.Close()
	close(after)
	f.Settle()
	if len(calls) != 0 {
		t.Fatal("native producer entered before predecessor retirement")
	}
	after = make(chan struct{})
	request.After = after
	f.Start(request)
	for range 50 {
		f.Explore("/b")
		f.FavoriteSaved()
		f.Explore("/c")
	}
	close(after)
	call := <-calls
	query := <-call.queries
	if query.ReferencePath != "/c" || query.CacheRevision != 50 {
		t.Fatalf("pending queries were not coalesced: %+v", query)
	}
	select {
	case extra := <-call.queries:
		t.Fatalf("obsolete query retained in bounded lane: %+v", extra)
	default:
	}
	call.emit(similarity.SearchEvent{SessionID: call.request.SessionID, QueryID: query.ID, CacheRevision: query.CacheRevision, Revision: 1, Kind: similarity.SearchFinal, Total: 3, Processed: 3})
	f.Settle()
}

func TestVisualSearchLifecycleUnexpectedExitReportsTerminalFailure(t *testing.T) {
	for _, failure := range []error{nil, errors.New("source version changed")} {
		t.Run(fmt.Sprint(failure), func(t *testing.T) {
			f, h, _, calls := newSearch(t)
			f.Start(visualsearch.StartRequest{Paths: []string{"/a"}, ReferencePath: "/a"})
			call := <-calls
			<-call.queries
			call.exit <- failure
			f.Settle()
			if len(h.errors) != 1 {
				t.Fatalf("uncompleted query did not report producer exit: %v", h.errors)
			}
			var terminal visualsearch.SessionError
			if !errors.As(h.errors[0], &terminal) {
				t.Fatalf("producer exit is not distinguished from query failure: %v", h.errors[0])
			}
			if failure != nil && !errors.Is(terminal, failure) {
				t.Fatal("terminal wrapper lost the provider error identity")
			}
			if f.State().Pending || f.State().Preparing || len(f.State().Visit.Paths) != 0 {
				t.Fatalf("exit invented a successful visit: %+v", f.State())
			}
		})
	}
}

func TestVisualSearchCaptureGridBeforePublication(t *testing.T) {
	f, h, q, calls := newSearch(t)
	f.Start(visualsearch.StartRequest{Paths: []string{"/a", "/b", "/c"}, ReferencePath: "/a"})
	call := <-calls
	query := <-call.queries
	anchor := grid.Visit{Ranked: true, Visible: true, Query: "a", Searching: true, Selected: []string{"/a"}, Highlight: "/a", ScrollOffset: 30}
	f.CaptureGrid(anchor)
	anchor.Selected[0] = "/changed"
	if got := f.State().Visit.Grid; got.Query != "a" || !reflect.DeepEqual(got.Selected, []string{"/a"}) {
		t.Fatalf("pending visit lost its Grid anchor: %+v", got)
	}
	h.hold = true
	publish(call, query, 1, similarity.SearchFinal, "/c", "/b")
	q.Drain()
	visit := f.State().Visit
	if got := visit.Grid; got.Query != "a" || !got.Searching || got.Highlight != "/a" || got.ScrollOffset != 30 || !reflect.DeepEqual(got.Selected, []string{"/a"}) {
		t.Fatalf("first publication lost its Grid anchor: %+v", got)
	}
	if !reflect.DeepEqual(visit.Paths, []string{"/c", "/b"}) {
		t.Fatalf("first publication lost new results: %v", visit.Paths)
	}
	f.Close()
	f.Start(visualsearch.StartRequest{Paths: []string{"/a", "/b"}, ReferencePath: "/b"})
	if f.State().Visit.Grid.Query != "" {
		t.Fatal("new session retained the prior Grid anchor")
	}
}

func TestVisualSearchCaptureGridKeepsImageAnchorWithNewestResultPaths(t *testing.T) {
	f, h, q, calls := newSearch(t)
	f.Start(visualsearch.StartRequest{Paths: []string{"/a", "/b", "/c"}, ReferencePath: "/a"})
	call := <-calls
	query := <-call.queries
	publish(call, query, 1, similarity.SearchPartial, "/b")
	q.Drain()
	queueVisit := grid.Visit{Paths: []string{"/b"}, Results: []string{"/b"}, Selected: []string{"/b"}, Subset: []string{"/b"}, Highlight: "/b", Query: "b", ScrollOffset: 30}
	f.CaptureGrid(queueVisit)
	queueVisit.Paths[0], queueVisit.Results[0], queueVisit.Selected[0], queueVisit.Subset[0] = "/mutated", "/mutated", "/mutated", "/mutated"
	h.current = f.State().Visit
	h.current.Image.Path = "/b"
	h.hold = true
	publish(call, query, 2, similarity.SearchFinal, "/c", "/b")
	f.Settle()
	f.Explore("/c")
	query = <-call.queries
	h.hold = false
	publish(call, query, 3, similarity.SearchFinal, "/a")
	f.Settle()
	f.Back()
	if !reflect.DeepEqual(h.current.Paths, []string{"/c", "/b"}) || h.current.Image.Path != "/b" {
		t.Fatalf("capturing the older image surface lost newest ranking: %+v", h.current)
	}
	got := h.current.Grid
	if !reflect.DeepEqual(got.Paths, []string{"/b"}) || !reflect.DeepEqual(got.Results, []string{"/b"}) || !reflect.DeepEqual(got.Selected, []string{"/b"}) || !reflect.DeepEqual(got.Subset, []string{"/b"}) || got.Query != "b" || got.ScrollOffset != 30 {
		t.Fatalf("image admission anchor was aliased or replaced: %+v", got)
	}
}

func TestVisualSearchCanceledSuccessorRetainsExternalRetirementBarrier(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f, _, _, calls := newSearch(t)
		after := make(chan struct{})
		defer close(after)
		f.Start(visualsearch.StartRequest{Paths: []string{"/a"}, ReferencePath: "/a", After: after})
		done := f.Suspend()
		synctest.Wait()
		select {
		case <-done:
			t.Fatal("canceled successor completed before the external writer retired")
		default:
		}
		if len(calls) != 0 {
			t.Fatal("canceled successor admitted native work")
		}
	})
}
