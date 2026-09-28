package requestlife_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/frathe/picfetch/internal/requestlife"
)

func TestRequestLifetimeContract(t *testing.T) {
	t.Run("zero owner begins and invalidates", func(t *testing.T) {
		var owner requestlife.Owner
		if got := owner.Revision(); got != 0 {
			t.Fatalf("initial revision = %d, want 0", got)
		}
		token := owner.Begin(context.Background())
		if got := token.Revision(); got != 1 {
			t.Fatalf("first token revision = %d, want 1", got)
		}
		if !token.Current() {
			t.Fatal("first token is not current")
		}
		if got := owner.Invalidate(); got != 2 {
			t.Fatalf("invalidated revision = %d, want 2", got)
		}
		if token.Current() {
			t.Fatal("invalidated token is current")
		}
		if err := token.Context().Err(); !errors.Is(err, context.Canceled) {
			t.Fatalf("invalidated context error = %v, want canceled", err)
		}
	})

	t.Run("zero token and independent owners", func(t *testing.T) {
		var zero requestlife.Token
		zero.Release()
		if zero.Current() || zero.Revision() != 0 {
			t.Fatal("zero token has a live identity")
		}

		var first, second requestlife.Owner
		firstToken := first.Begin(context.Background())
		secondToken := second.Begin(context.Background())
		first.Invalidate()
		if !secondToken.Current() || secondToken.Context().Err() != nil || second.Revision() != 1 {
			t.Fatal("invalidating one owner affected another")
		}
		if firstToken.Current() {
			t.Fatal("invalidated token remained current")
		}
	})

	t.Run("begin and repeated invalidation preserve token identities", func(t *testing.T) {
		var owner requestlife.Owner
		if got := owner.Invalidate(); got != 1 {
			t.Fatalf("pre-use invalidation = %d, want 1", got)
		}
		old := owner.Begin(context.Background())
		newer := owner.Begin(context.Background())
		if old.Revision() != 2 || newer.Revision() != 3 || owner.Revision() != 3 {
			t.Fatalf("revisions = %d, %d, %d; want 2, 3, 3", old.Revision(), newer.Revision(), owner.Revision())
		}
		if old.Current() || !errors.Is(old.Context().Err(), context.Canceled) || !newer.Current() {
			t.Fatal("replacement did not retire predecessor")
		}
		if got := owner.Invalidate(); got != 4 {
			t.Fatalf("first invalidation = %d, want 4", got)
		}
		if got := owner.Invalidate(); got != 5 {
			t.Fatalf("repeated invalidation = %d, want 5", got)
		}
		if newer.Revision() != 3 || newer.Current() || !errors.Is(newer.Context().Err(), context.Canceled) {
			t.Fatal("invalidation changed the retained token identity or left it live")
		}
	})

	t.Run("parent context is captured and release is isolated", func(t *testing.T) {
		type contextKey struct{}
		deadline := time.Now().Add(time.Hour)
		parent, cancelParent := context.WithDeadline(context.WithValue(context.Background(), contextKey{}, "source"), deadline)
		defer cancelParent()
		var owner requestlife.Owner
		first := owner.Begin(parent)
		if got := first.Context().Value(contextKey{}); got != "source" {
			t.Fatalf("captured value = %v, want source", got)
		}
		if got, ok := first.Context().Deadline(); !ok || !got.Equal(deadline) {
			t.Fatalf("captured deadline = %v, %v; want %v", got, ok, deadline)
		}
		second := owner.Begin(parent)
		first.Release()
		first.Release()
		if owner.Revision() != 2 || !second.Current() || parent.Err() != nil {
			t.Fatal("old token release changed the owner, newer request, or parent")
		}
		second.Release()
		second.Release()
		if second.Current() || !errors.Is(second.Context().Err(), context.Canceled) || owner.Revision() != 2 || parent.Err() != nil {
			t.Fatal("idempotent release did not isolate its child context")
		}
		third := owner.Begin(parent)
		cancelParent()
		if third.Current() || !errors.Is(third.Context().Err(), context.Canceled) {
			t.Fatal("parent cancellation did not propagate")
		}
	})

	t.Run("cancelled parent still supersedes", func(t *testing.T) {
		var owner requestlife.Owner
		old := owner.Begin(context.Background())
		parent, cancel := context.WithCancel(context.Background())
		cancel()
		newer := owner.Begin(parent)
		if owner.Revision() != 2 || newer.Revision() != 2 || newer.Current() || !errors.Is(newer.Context().Err(), context.Canceled) {
			t.Fatal("cancelled parent did not consume a non-current new revision")
		}
		if old.Current() || !errors.Is(old.Context().Err(), context.Canceled) {
			t.Fatal("cancelled-parent begin revived predecessor")
		}
	})

	t.Run("concurrent begins leave one current request", func(t *testing.T) {
		const count = 64
		var owner requestlife.Owner
		var workers sync.WaitGroup
		workers.Add(count)
		start := make(chan struct{})
		tokens := make(chan requestlife.Token, count)
		for range count {
			go func() {
				defer workers.Done()
				<-start
				tokens <- owner.Begin(context.Background())
			}()
		}
		close(start)
		workers.Wait()
		close(tokens)
		seen := make(map[uint64]bool, count)
		current := 0
		for token := range tokens {
			if seen[token.Revision()] {
				t.Fatalf("duplicate revision %d", token.Revision())
			}
			seen[token.Revision()] = true
			if token.Current() {
				current++
				if token.Revision() != count {
					t.Fatalf("current revision = %d, want %d", token.Revision(), count)
				}
			} else if !errors.Is(token.Context().Err(), context.Canceled) {
				t.Fatalf("superseded revision %d was not cancelled", token.Revision())
			}
		}
		if owner.Revision() != count || len(seen) != count || current != 1 {
			t.Fatalf("revision = %d, distinct = %d, current = %d; want %d, %d, 1", owner.Revision(), len(seen), current, count, count)
		}
	})
}
