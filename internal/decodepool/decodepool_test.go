package decodepool

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
)

func TestClaim_SecondClaimForSameValueIsRefused(t *testing.T) {
	p := New[string, int](1)
	if !p.Claim("a", 1) {
		t.Fatal("first claim refused")
	}
	if p.Claim("a", 1) {
		t.Fatal("duplicate claim accepted")
	}
}

func TestClaim_DifferentValueOverwrites(t *testing.T) {
	p := New[string, int](1)
	p.Claim("a", 1)
	if !p.Claim("a", 2) {
		t.Fatal("claim for a different value refused")
	}
	// The superseded worker's release must not drop the newer claim.
	p.Release("a", 1)
	if p.Claim("a", 2) {
		t.Fatal("stale release dropped the newer claim")
	}
}

func TestClaim_StructValueBehavesLikeLoadOrStore(t *testing.T) {
	p := New[string, struct{}](1)
	if !p.Claim("a", struct{}{}) {
		t.Fatal("first claim refused")
	}
	if p.Claim("a", struct{}{}) {
		t.Fatal("duplicate claim accepted")
	}
	p.Release("a", struct{}{})
	if !p.Claim("a", struct{}{}) {
		t.Fatal("claim after release refused")
	}
}

func TestGo_LimitBoundsConcurrency(t *testing.T) {
	p := New[int, int](2)
	var inFlight, peak atomic.Int64
	release := make(chan struct{})
	for range 8 {
		p.Go(context.Background(), func(acquired bool) {
			if !acquired {
				t.Error("acquired false with an uncancelled context")
				return
			}
			n := inFlight.Add(1)
			for {
				old := peak.Load()
				if n <= old || peak.CompareAndSwap(old, n) {
					break
				}
			}
			<-release
			inFlight.Add(-1)
		})
	}
	close(release)
	p.Wait()
	if peak.Load() > 2 {
		t.Fatalf("peak concurrency %d, want <= 2", peak.Load())
	}
}

// TestGo_QueueDoesNotCreateWaiterPerJob reproduces the Grid's large-folder
// shape: every decode slot is busy while a whole source list queues behind it.
// A bounded queue keeps those pending jobs as data. The old semaphore design
// instead gave every pending job a goroutine parked on the semaphore.
func TestGo_QueueDoesNotCreateWaiterPerJob(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const limit = 4
		const queuedJobs = 22_000

		p := New[int, int](limit)
		entered := make(chan struct{}, limit)
		release := make(chan struct{})
		unblock := sync.OnceFunc(func() { close(release) })
		t.Cleanup(func() {
			unblock()
			p.Wait()
		})

		for range limit {
			p.Go(context.Background(), func(acquired bool) {
				if !acquired {
					t.Error("uncancelled held job was not admitted")
					return
				}
				entered <- struct{}{}
				<-release
			})
		}
		for range limit {
			<-entered
		}

		baseline := runtime.NumGoroutine()
		for range queuedJobs {
			p.Go(context.Background(), func(bool) {})
		}

		synctest.Wait()
		if excess := runtime.NumGoroutine() - baseline; excess > 8 {
			t.Fatalf("queueing %d jobs added %d goroutines, want <= 8", queuedJobs, excess)
		}
	})
}

func TestWait_JoinsInternalDispatcherLifetime(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := New[int, int](1)
		release := make(chan struct{})
		p.dispatch(func() { <-release })

		waited := make(chan struct{})
		go func() {
			p.Wait()
			close(waited)
		}()
		synctest.Wait()
		select {
		case <-waited:
			t.Fatal("Wait returned while an internal dispatcher was still running")
		default:
		}

		close(release)
		<-waited
	})
}

func TestQueue_CancellationDoesNotCreateCallbackPerJob(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const queuedJobs = 22_000

		p := New[int, int](1)
		held := make(chan struct{})
		holding := make(chan struct{})
		unhold := sync.OnceFunc(func() { close(held) })
		ctx, cancel := context.WithCancel(context.Background())
		q := p.Begin(ctx)
		callbackRelease := make(chan struct{})
		unblockCallbacks := sync.OnceFunc(func() { close(callbackRelease) })
		t.Cleanup(func() {
			unblockCallbacks()
			unhold()
			p.Wait()
		})

		p.Go(context.Background(), func(acquired bool) {
			if !acquired {
				t.Error("uncancelled held job was not admitted")
				return
			}
			close(holding)
			<-held
		})
		<-holding

		callbackStarted := make(chan struct{})
		var first sync.Once
		for range queuedJobs {
			q.Go(func(acquired bool) {
				if acquired {
					t.Error("cancelled queued job was admitted")
				}
				first.Do(func() { close(callbackStarted) })
				<-callbackRelease
			})
		}

		baseline := runtime.NumGoroutine()
		cancel()
		<-callbackStarted
		synctest.Wait()
		if excess := runtime.NumGoroutine() - baseline; excess > 8 {
			t.Fatalf("cancelling %d jobs added %d goroutines, want <= 8", queuedJobs, excess)
		}
	})
}

func TestQueue_InteractiveJobsRunBeforeBackgroundWork(t *testing.T) {
	p := New[int, int](1)
	held := make(chan struct{})
	holding := make(chan struct{})
	unhold := sync.OnceFunc(func() { close(held) })
	t.Cleanup(func() {
		unhold()
		p.Wait()
	})

	p.Go(context.Background(), func(acquired bool) {
		if !acquired {
			t.Error("uncancelled held job was not admitted")
			return
		}
		close(holding)
		<-held
	})
	<-holding

	q := p.Begin(context.Background())
	order := make(chan string, 3)
	q.GoLow(func(bool) { order <- "background" })
	q.Go(func(bool) { order <- "stale cell" })
	q.Go(func(bool) { order <- "current cell" })

	unhold()
	p.Wait()
	for i, want := range []string{"current cell", "stale cell", "background"} {
		if got := <-order; got != want {
			t.Fatalf("run %d = %q, want %q", i, got, want)
		}
	}
}

func TestGo_CancelledWhileQueuedStillCallsFn(t *testing.T) {
	p := New[int, int](1)
	holding := make(chan struct{})
	block := make(chan struct{})
	p.Go(context.Background(), func(bool) {
		close(holding)
		<-block
	})

	// The pool's one worker has to be genuinely occupied before the cancelled
	// call queues behind it. Otherwise the worker could admit the call before
	// cancellation dispatch proves the queued-work path.
	<-holding

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var got atomic.Bool
	var sawAcquired atomic.Bool
	resolved := make(chan struct{})
	p.Go(ctx, func(acquired bool) {
		got.Store(true)
		sawAcquired.Store(acquired)
		close(resolved)
	})

	// It has to resolve while the slot is still held. Releasing first could let
	// ordinary worker admission hide a broken cancellation-dispatch path.
	<-resolved

	close(block)
	p.Wait()
	if !got.Load() {
		t.Fatal("fn was not called for a queue-cancelled request")
	}
	if sawAcquired.Load() {
		t.Fatal("acquired was true for a queue-cancelled request")
	}
}

func TestWait_ReturnsOnlyAfterEveryFnReturns(t *testing.T) {
	p := New[int, int](4)
	var done atomic.Int64
	block := make(chan struct{})
	for range 4 {
		p.Go(context.Background(), func(bool) {
			<-block
			done.Add(1)
		})
	}
	close(block)
	p.Wait()
	if done.Load() != 4 {
		t.Fatalf("Wait returned with %d of 4 done", done.Load())
	}
}
