package client

import (
	"context"
	"errors"
	"testing"
)

func TestAdmissionPrefersForegroundWithoutStarvingBackground(t *testing.T) {
	var queue admissionQueue
	first := queue.enter(context.Background(), Foreground)
	if err := first.wait(); err != nil {
		t.Fatal(err)
	}
	background := queue.enter(context.Background(), Background)
	foreground := []*admission{
		queue.enter(context.Background(), Foreground),
		queue.enter(context.Background(), Foreground),
		queue.enter(context.Background(), Foreground),
	}
	assertPending := func(request *admission) {
		t.Helper()
		select {
		case <-request.ready:
			t.Fatal("request admitted before its turn")
		default:
		}
	}
	assertPending(background)
	first.release()
	for _, request := range foreground[:2] {
		select {
		case <-request.ready:
		default:
			t.Fatal("foreground was not preferred")
		}
		assertPending(background)
		request.release()
	}
	// At most three consecutive foreground grants, including the first one.
	select {
	case <-background.ready:
	default:
		t.Fatal("background starved")
	}
	assertPending(foreground[2])
	background.release()
	if err := foreground[2].wait(); err != nil {
		t.Fatal(err)
	}
	foreground[2].release()
}

func TestAdmissionCancellationAndIdempotentRelease(t *testing.T) {
	var queue admissionQueue
	first := queue.enter(context.Background(), Background)
	ctx, cancel := context.WithCancel(context.Background())
	cancelled := queue.enter(ctx, Foreground)
	next := queue.enter(context.Background(), Background)
	cancel()
	if err := cancelled.wait(); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled wait: %v", err)
	}
	cancelled.release()
	first.release()
	if err := next.wait(); err != nil {
		t.Fatal(err)
	}
	later := queue.enter(context.Background(), Foreground)
	first.release()
	select {
	case <-later.ready:
		t.Fatal("duplicate release admitted overlapping work")
	default:
	}
	next.release()
	if err := later.wait(); err != nil {
		t.Fatal(err)
	}
	later.release()
}
