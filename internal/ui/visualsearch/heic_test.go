package visualsearch_test

import (
	"context"
	"testing"

	"github.com/frathe/picfetch/internal/heic"
	"github.com/frathe/picfetch/internal/similarity"
	"github.com/frathe/picfetch/internal/ui/visualsearch"
	"github.com/frathe/picfetch/internal/uitest"
)

type heicBackend struct{}

func (heicBackend) Check(_ context.Context) error { return nil }
func (heicBackend) Read(_ context.Context, _ []byte, _ heic.Request) (heic.Result, error) {
	return heic.Result{}, heic.ErrUnsupported
}

func TestHEICVisualSearchRetainsCapturedCapability(t *testing.T) {
	capability := heic.NewCapability(heicBackend{}, heic.Identity{}, heic.Observation{})
	t.Cleanup(func() { capability.Stop(); capability.Wait() })
	<-capability.Check(context.Background())
	type capturedQuery struct {
		ctx     context.Context
		session uint64
	}
	queries := make(chan capturedQuery, 3)
	provider := func(ctx context.Context, req similarity.SearchRequest, input <-chan similarity.SearchQuery, emit func(similarity.SearchEvent)) error {
		var revision uint64
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case query, ok := <-input:
				if !ok {
					return nil
				}
				queries <- capturedQuery{ctx, req.SessionID}
				revision++
				emit(similarity.SearchEvent{SessionID: req.SessionID, QueryID: query.ID, Revision: revision, Kind: similarity.SearchFinal, Total: 2, Processed: 2, Matches: []similarity.Match{{Path: "/other.heic"}}})
				revision++
				emit(similarity.SearchEvent{SessionID: req.SessionID, QueryID: query.ID, Revision: revision, Kind: similarity.SearchReady, Total: 2, Processed: 2})
			}
		}
	}
	f := visualsearch.New(&visitHost{}, visualsearch.Options{Provider: provider, Queue: &uitest.UIQueue{}})
	f.SetHEICCapability(capability)
	t.Cleanup(func() { f.Stop(); f.Settle() })
	before := make(chan struct{})
	if !f.Start(visualsearch.StartRequest{Paths: []string{"/photo.heic", "/other.heic"}, ReferencePath: "/photo.heic", After: before}) {
		t.Fatal("search rejected")
	}
	<-capability.Check(context.Background())
	close(before)
	first := <-queries
	f.Settle()
	if got := heic.FromContext(first.ctx); !got.Available || got.Backend == nil || got.Generation != 1 {
		t.Fatalf("search did not capture before waiting for its predecessor: %+v", got)
	}
	<-capability.Check(context.Background())
	if !f.Explore("/other.heic") {
		t.Fatal("retained reference rejected")
	}
	retained := <-queries
	f.Settle()
	if got := heic.FromContext(retained.ctx); got.Generation != 1 || retained.session != first.session {
		t.Fatalf("support recheck replaced retained producer policy: %+v session=%d", got, retained.session)
	}
	<-f.Suspend()
	f.Settle()
	if first.ctx.Err() == nil {
		t.Fatal("suspend did not cancel captured producer")
	}
	if !f.Explore("/photo.heic") {
		t.Fatal("explicit restart rejected")
	}
	next := <-queries
	f.Settle()
	if got := heic.FromContext(next.ctx); !got.Available || got.Generation != 3 || next.session == first.session {
		t.Fatalf("new producer did not capture new support: %+v session=%d", got, next.session)
	}
}
