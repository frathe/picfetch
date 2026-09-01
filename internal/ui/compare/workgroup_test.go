package compare

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestWorkTracker_ContextTimeoutDoesNotPoisonReuse(t *testing.T) {
	var work workTracker
	work.Add(1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := work.WaitContext(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("WaitContext error = %v, want context cancellation", err)
	}

	work.Done()
	work.Add(1)
	go work.Done()
	ctx, cancel = context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := work.WaitContext(ctx); err != nil {
		t.Fatalf("reused work tracker did not settle: %v", err)
	}
}

func TestWorkTracker_WaitReturnsImmediatelyBeforeFirstWork(t *testing.T) {
	var work workTracker
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := work.WaitContext(ctx); err != nil {
		t.Fatalf("empty work tracker wait: %v", err)
	}
}
