package visualsearch

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/similarity"
	"github.com/frathe/picfetch/internal/ui/grid"
)

type featureQueue struct{}

func (featureQueue) Do(fn func()) { fyne.Do(fn) }
func (featureQueue) Drain() bool  { return false }

// SessionError identifies a terminal producer error, as distinct from an
// individual reference failure that leaves the prepared scope usable.
type SessionError struct{ Err error }

func (e SessionError) Error() string { return e.Err.Error() }
func (e SessionError) Unwrap() error { return e.Err }

type producer struct {
	id      uint64
	cancel  context.CancelFunc
	queries chan similarity.SearchQuery
	done    chan struct{}
	notice  chan struct{}
}

func (f *Feature) beginProducer(after <-chan struct{}) {
	f.sessionID++
	f.cacheWarned = false
	f.revision = 0
	ctx, cancel := context.WithCancel(context.Background())
	s := &producer{id: f.sessionID, cancel: cancel, queries: make(chan similarity.SearchQuery, 1), done: make(chan struct{}), notice: make(chan struct{}, 1)}
	f.producer = s
	request := similarity.SearchRequest{SessionID: s.id, Paths: slices.Clone(f.scope), Limit: 30, Cache: f.cache}
	provider, queue := f.provider, f.ui
	var barriers []<-chan struct{}
	for _, prior := range f.retired {
		barriers = append(barriers, prior.done)
	}
	go func() {
		defer func() {
			// A canceled successor can still be waiting for an older writer.
			// Its completion includes those predecessors even if it never starts.
			if after != nil {
				<-after
			}
			for _, barrier := range barriers {
				<-barrier
			}
			close(s.done)
		}()
		if after != nil {
			select {
			case <-ctx.Done():
				return
			case <-after:
			}
		}
		for _, barrier := range barriers {
			select {
			case <-ctx.Done():
				return
			case <-barrier:
			}
		}
		if ctx.Err() != nil {
			return
		}
		err := provider(ctx, request, s.queries, func(event similarity.SearchEvent) {
			event.Matches = slices.Clone(event.Matches)
			queue.Do(func() { f.apply(s, event) })
			s.notify()
		})
		queue.Do(func() { f.producerReturned(s, err) })
		s.notify()
	}()
}

func (s *producer) notify() {
	select {
	case s.notice <- struct{}{}:
	default:
	}
}

func (f *Feature) apply(s *producer, event similarity.SearchEvent) {
	if !f.active || f.producer != s || event.SessionID != s.id || event.Revision <= f.revision {
		return
	}
	// Preparation belongs to the retained session even after Back abandons a
	// query. Only its progress/readiness may update the restored frozen visit.
	if event.QueryID != f.queryID && event.Kind != similarity.SearchProgress && event.Kind != similarity.SearchReady {
		return
	}
	if f.queryFailed && (event.Kind == similarity.SearchPartial || event.Kind == similarity.SearchFinal) {
		return
	}
	f.revision = event.Revision
	if event.CacheRevision == f.cacheRevision {
		f.cachePending = false
	}
	if event.CacheWarning != "" && !f.cacheWarned {
		f.cacheWarned = true
		fyne.LogError("visual search analysis cache", errors.New(event.CacheWarning))
	}
	f.progress = grid.Progress{Processed: event.Processed, Total: event.Total, Failed: event.Failed, Complete: event.Kind == similarity.SearchFinal || event.Kind == similarity.SearchReady}
	f.preparing = event.Processed < event.Total
	switch event.Kind {
	case similarity.SearchPartial, similarity.SearchFinal:
		if f.pending {
			f.history = append(f.history, Visit{ReferencePath: f.reference, Grid: grid.Visit{Ranked: true, Visible: true}})
			if len(f.history) > 20 {
				f.history = slices.Clone(f.history[len(f.history)-20:])
			}
			f.pending = false
		}
		last := &f.history[len(f.history)-1]
		last.Paths = make([]string, len(event.Matches))
		for i, match := range event.Matches {
			last.Paths[i] = match.Path
		}
		if event.Kind == similarity.SearchFinal {
			f.awaiting, f.preparing = false, false
		}
		f.host.Present(cloneVisit(*last), f.progress)
	case similarity.SearchQueryFailure:
		f.pending, f.awaiting, f.queryFailed = false, false, true
		f.host.Failed(errors.New(event.Error))
	case similarity.SearchFailure:
		f.retire()
		f.invalidateQuery()
		f.preparing = false
		f.host.Failed(SessionError{Err: errors.New(event.Error)})
	}
	if event.CachePressureBytes > 0 {
		f.Suspend()
		f.host.Failed(similarity.CachePressureError{NeedBytes: event.CachePressureBytes})
	}
	f.host.Changed()
}

func (f *Feature) producerReturned(s *producer, err error) {
	if f.producer != s || !f.active {
		return
	}
	f.producer = nil
	f.retired = append(f.retired, s)
	if err == nil && f.awaiting {
		err = fmt.Errorf("visual search stopped before completing the current query")
	}
	f.pending, f.awaiting, f.preparing, f.cachePending = false, false, false, false
	if err != nil && !errors.Is(err, context.Canceled) {
		f.host.Failed(SessionError{Err: err})
	}
	f.host.Changed()
}

func (f *Feature) retire() <-chan struct{} {
	f.cachePending = false
	remaining := f.retired[:0]
	for _, prior := range f.retired {
		select {
		case <-prior.done:
		default:
			remaining = append(remaining, prior)
		}
	}
	clear(f.retired[len(remaining):])
	f.retired = remaining
	if s := f.producer; s != nil {
		s.cancel()
		f.retired = append(f.retired, s)
		f.producer = nil
		return s.done
	}
	if len(f.retired) > 0 {
		return f.retired[len(f.retired)-1].done
	}
	done := make(chan struct{})
	close(done)
	return done
}

// Suspend cancels producer admission/delivery while retaining frozen browsing.
// Explicit Explore may subsequently admit a new producer after retirement.
func (f *Feature) Suspend() <-chan struct{} {
	f.capture()
	done := f.retire()
	f.invalidateQuery()
	f.preparing = false
	f.host.Changed()
	return done
}

// Close permits later Start and returns immediately without joining workers.
func (f *Feature) Close() { f.clear(); f.host.Changed() }

// Stop permanently ends admission. Wait joins workers off the UI goroutine.
func (f *Feature) Stop() { f.stopped = true; f.Close() }

func (f *Feature) Wait() {
	for _, s := range f.retired {
		<-s.done
	}
	if f.producer != nil {
		<-f.producer.done
	}
}

// Settle waits finite query delivery and retired workers. A live producer stays
// available after a query completes and is never joined just for being READY.
func (f *Feature) Settle() {
	for {
		for _, s := range f.retired {
			<-s.done
		}
		f.retired = nil
		delivered := f.ui.Drain()
		if (f.awaiting || f.cachePending) && f.producer != nil {
			select {
			case <-f.producer.notice:
			case <-f.producer.done:
			}
			continue
		}
		if !delivered {
			return
		}
	}
}
