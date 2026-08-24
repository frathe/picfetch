package uitest

import (
	"sync"
	"testing"
)

func TestUIQueue_DoDefersInsteadOfRunning(t *testing.T) {
	var q UIQueue
	ran := false

	q.Do(func() { ran = true })

	if ran {
		t.Fatal("Do must not run the callback on the calling goroutine - that is the whole point")
	}
	if q.Len() != 1 {
		t.Fatalf("Len() = %d, want 1", q.Len())
	}
	if !q.Drain() {
		t.Fatal("Drain() = false with one callback queued, want true")
	}
	if !ran {
		t.Error("Drain must run the queued callback")
	}
}

func TestUIQueue_DrainRunsInOrderThenReportsEmpty(t *testing.T) {
	var q UIQueue
	var order []int
	for i := range 3 {
		q.Do(func() { order = append(order, i) })
	}

	q.Drain()

	if len(order) != 3 || order[0] != 0 || order[1] != 1 || order[2] != 2 {
		t.Fatalf("order = %v, want [0 1 2]", order)
	}
	if q.Drain() {
		t.Error("Drain() = true on an empty queue, want false")
	}
}

func TestUIQueue_WorkQueuedDuringADrainWaitsForTheNext(t *testing.T) {
	var q UIQueue
	inner := false
	q.Do(func() { q.Do(func() { inner = true }) })

	if !q.Drain() {
		t.Fatal("first Drain() = false, want true - the outer callback was queued")
	}
	if inner {
		t.Fatal("a callback queued during a Drain must not run in that same Drain")
	}
	if !q.Drain() {
		t.Fatal("second Drain() = false, want true - the inner callback was queued")
	}
	if !inner {
		t.Error("the inner callback never ran")
	}
}

func TestUIQueue_ConcurrentDo(t *testing.T) {
	var q UIQueue
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() { q.Do(func() {}) })
	}
	wg.Wait()

	if q.Len() != 50 {
		t.Fatalf("Len() = %d after 50 concurrent Do calls, want 50", q.Len())
	}
}
