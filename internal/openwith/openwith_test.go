package openwith

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/uitest"
)

// reset restores the default queue to its zero state so package-level
// Deliver/SetHandler tests don't leak state into one another. Unexported on
// purpose - see defaultQueue's doc comment in openwith.go for why this
// isn't an exported test seam.
func reset() {
	defaultQueue.mu.Lock()
	defaultQueue.pending = nil
	defaultQueue.handler = nil
	defaultQueue.mu.Unlock()
}

func TestQueue_DeliverBeforeSetHandler_BuffersThenFlushesInArrivalOrder(t *testing.T) {
	t.Parallel()

	q := &queue{}
	a := uitest.FakeURI{FileName: "a.jpg"}
	b := uitest.FakeURI{FileName: "b.jpg"}

	q.Deliver([]fyne.URI{a})
	q.Deliver([]fyne.URI{b})

	var got []fyne.URI
	q.SetHandler(func(uris []fyne.URI) {
		got = append(got, uris...)
	})

	if len(got) != 2 || got[0] != fyne.URI(a) || got[1] != fyne.URI(b) {
		t.Fatalf("got %v, want [a b]", got)
	}
	if len(q.pending) != 0 {
		t.Fatalf("pending = %v, want empty after SetHandler flush", q.pending)
	}
}

func TestQueue_DeliverAfterSetHandler_CallsThroughImmediatelyAndLeavesPendingEmpty(t *testing.T) {
	t.Parallel()

	q := &queue{}

	var got []fyne.URI
	q.SetHandler(func(uris []fyne.URI) {
		got = append(got, uris...)
	})

	a := uitest.FakeURI{FileName: "a.jpg"}
	q.Deliver([]fyne.URI{a})

	if len(got) != 1 || got[0] != fyne.URI(a) {
		t.Fatalf("got %v, want [a]", got)
	}
	if len(q.pending) != 0 {
		t.Fatalf("pending = %v, want empty", q.pending)
	}
}

func TestQueue_SetHandlerNil_ClearsHandlerAndBuffersAgain(t *testing.T) {
	t.Parallel()

	q := &queue{}

	var calls int
	q.SetHandler(func([]fyne.URI) { calls++ })
	q.SetHandler(nil)

	a := uitest.FakeURI{FileName: "a.jpg"}
	q.Deliver([]fyne.URI{a})

	if calls != 0 {
		t.Fatalf("handler invoked %d times after SetHandler(nil), want 0", calls)
	}
	if len(q.pending) != 1 || q.pending[0] != fyne.URI(a) {
		t.Fatalf("pending = %v, want [a]", q.pending)
	}
}

func TestQueue_SetHandlerNil_DoesNotDiscardPreviouslyBuffered(t *testing.T) {
	t.Parallel()

	q := &queue{}
	a := uitest.FakeURI{FileName: "a.jpg"}
	q.Deliver([]fyne.URI{a})

	// SetHandler(nil) must not silently drop what Deliver already buffered
	// before any handler was ever installed.
	q.SetHandler(nil)

	if len(q.pending) != 1 || q.pending[0] != fyne.URI(a) {
		t.Fatalf("pending = %v, want [a] preserved across SetHandler(nil)", q.pending)
	}
}

func TestQueue_Deliver_EmptyOrNilIsNoop(t *testing.T) {
	t.Parallel()

	q := &queue{}

	q.Deliver(nil)
	q.Deliver([]fyne.URI{})
	if len(q.pending) != 0 {
		t.Fatalf("pending = %v, want empty after nil/empty Deliver", q.pending)
	}

	var called bool
	q.SetHandler(func([]fyne.URI) { called = true })

	q.Deliver(nil)
	q.Deliver([]fyne.URI{})
	if called {
		t.Fatal("handler invoked for a nil/empty Deliver")
	}
}

// TestQueue_HandlerNeverInvokedWhileMutexHeld proves the handler runs
// outside the lock by having it call Deliver reentrantly. If the handler
// were ever invoked while q.mu was held, that reentrant Deliver would
// deadlock; the timeout turns a regression into a failure instead of a
// hung test suite.
func TestQueue_HandlerNeverInvokedWhileMutexHeld(t *testing.T) {
	t.Parallel()

	q := &queue{}
	a := uitest.FakeURI{FileName: "a.jpg"}
	b := uitest.FakeURI{FileName: "b.jpg"}

	var mu sync.Mutex
	var got []fyne.URI
	done := make(chan struct{})

	q.SetHandler(func(uris []fyne.URI) {
		mu.Lock()
		got = append(got, uris...)
		reentrant := len(got) == 1
		mu.Unlock()

		if reentrant {
			q.Deliver([]fyne.URI{b})
			return
		}
		close(done)
	})

	q.Deliver([]fyne.URI{a})

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("reentrant Deliver from inside the handler deadlocked")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(got) != 2 || got[0] != fyne.URI(a) || got[1] != fyne.URI(b) {
		t.Fatalf("got %v, want [a b]", got)
	}
}

// TestQueue_ConcurrentDeliverAndSetHandler drives Deliver and SetHandler
// from many goroutines at once. handled + whatever is still buffered at
// the end must equal exactly n: every Deliver either lands directly on the
// handler or is buffered until the first SetHandler call flips the handler
// from nil to non-nil and flushes it - each item is counted exactly once
// regardless of goroutine scheduling, which is what -race plus this
// invariant check is here to prove.
func TestQueue_ConcurrentDeliverAndSetHandler(t *testing.T) {
	q := &queue{}

	var handled atomic.Int64
	handler := func(uris []fyne.URI) {
		handled.Add(int64(len(uris)))
	}

	const n = 200
	var wg sync.WaitGroup
	wg.Add(2 * n)

	for range n {
		go func() {
			defer wg.Done()
			q.Deliver([]fyne.URI{uitest.FakeURI{FileName: "x.jpg"}})
		}()
		go func() {
			defer wg.Done()
			q.SetHandler(handler)
		}()
	}
	wg.Wait()

	if got := handled.Load(); got != n {
		t.Fatalf("handled = %d, want %d (n Delivers, exactly-once accounting)", got, n)
	}
}

func TestDeliver_PackageLevel_BuffersUntilSetHandlerInstalled(t *testing.T) {
	reset()
	t.Cleanup(reset)

	a := uitest.FakeURI{FileName: "a.jpg"}
	Deliver([]fyne.URI{a})

	var got []fyne.URI
	SetHandler(func(uris []fyne.URI) {
		got = append(got, uris...)
	})

	if len(got) != 1 || got[0] != fyne.URI(a) {
		t.Fatalf("got %v, want [a]", got)
	}
}

func TestSetHandler_PackageLevel_CallsThroughAfterInstalled(t *testing.T) {
	reset()
	t.Cleanup(reset)

	var got []fyne.URI
	SetHandler(func(uris []fyne.URI) {
		got = append(got, uris...)
	})

	a := uitest.FakeURI{FileName: "a.jpg"}
	Deliver([]fyne.URI{a})

	if len(got) != 1 || got[0] != fyne.URI(a) {
		t.Fatalf("got %v, want [a]", got)
	}
}

func TestURIsFromFileURLs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   []string
		want []string // expected Path() of each surviving URI, in order
	}{
		{
			name: "plain file path",
			in:   []string{"file:///a/b.jpg"},
			want: []string{"/a/b.jpg"},
		},
		{
			name: "percent-encoded space decodes",
			in:   []string{"file:///a/with%20space.jpg"},
			want: []string{"/a/with space.jpg"},
		},
		{
			name: "percent-encoded unicode decodes",
			in:   []string{"file:///a/%C3%BCber.png"},
			want: []string{"/a/über.png"},
		},
		{
			name: "non-file scheme is skipped",
			in:   []string{"http://x/y.jpg"},
			want: nil,
		},
		{
			name: "empty string is skipped",
			in:   []string{""},
			want: nil,
		},
		{
			name: "empty path is skipped",
			in:   []string{"file://"},
			want: nil,
		},
		{
			name: "malformed percent-escape is skipped",
			in:   []string{"file:///a/%zz"},
			want: nil,
		},
		{
			name: "explicit localhost authority is accepted",
			in:   []string{"file://localhost/a/b.jpg"},
			want: []string{"/a/b.jpg"},
		},
		{
			name: "remote host authority is skipped",
			in:   []string{"file://elsewhere/a/b.jpg"},
			want: nil,
		},
		{
			name: "mixed slice keeps only the good entries in order",
			in: []string{
				"file:///a/b.jpg",
				"http://x/y.jpg",
				"",
				"file:///a/%zz",
				"file:///a/c.jpg",
			},
			want: []string{"/a/b.jpg", "/a/c.jpg"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := URIsFromFileURLs(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("URIsFromFileURLs(%v) = %v, want paths %v", tt.in, got, tt.want)
			}
			for i, u := range got {
				if u.Path() != tt.want[i] {
					t.Fatalf("URIsFromFileURLs(%v)[%d].Path() = %q, want %q", tt.in, i, u.Path(), tt.want[i])
				}
			}
		})
	}
}
