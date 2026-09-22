package heic

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type checkBackend struct {
	check func(context.Context) error
}

func (b checkBackend) Check(ctx context.Context) error { return b.check(ctx) }
func (b checkBackend) Read(_ context.Context, _ []byte, _ Request) (Result, error) {
	return Result{}, ErrUnavailable
}

func TestHEICCapabilityLifecycle(t *testing.T) {
	identity := Identity{OS: "linux", Architecture: "amd64", OSVersion: "test-os", Revision: Revision}
	t.Run("matching observation avoids native work", func(t *testing.T) {
		for _, available := range []bool{false, true} {
			stored := Observation{Identity: identity, CheckedAt: time.Unix(100, 0), Available: available}
			backend := checkBackend{check: func(_ context.Context) error {
				t.Error("matching startup observation repeated the native probe")
				return nil
			}}
			capability := NewCapability(backend, identity, stored)
			t.Cleanup(func() { capability.Stop(); capability.Wait() })
			<-capability.Ensure(context.Background())
			state := capability.State()
			if !state.Known || state.Checking || state.Observation != stored || state.Available != available {
				t.Fatalf("cached state = %+v", state)
			}
		}
	})
	t.Run("first check coalesces and publishes validated result", func(t *testing.T) {
		entered, release := make(chan struct{}), make(chan struct{})
		var calls atomic.Int32
		backend := checkBackend{check: func(ctx context.Context) error {
			calls.Add(1)
			close(entered)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}}
		capability := NewCapability(backend, identity, Observation{})
		t.Cleanup(func() { capability.Stop(); capability.Wait() })
		done := capability.Ensure(context.Background())
		if state := capability.State(); !state.Checking || state.Known {
			t.Fatalf("running state = %+v", state)
		}
		<-entered
		if duplicate := capability.Check(context.Background()); duplicate != done {
			t.Fatal("duplicate check did not share completion")
		}
		close(release)
		<-done
		state := capability.State()
		if calls.Load() != 1 || !state.Known || !state.Available || state.Checking || state.Err != nil || !state.Observation.Matches(identity) {
			t.Fatalf("completed state = %+v, native checks = %d", state, calls.Load())
		}
		if snapshot := capability.Snapshot(); !snapshot.Available || snapshot.Backend == nil || snapshot.Generation == 0 {
			t.Fatalf("snapshot = %+v", snapshot)
		}
	})
	t.Run("operational failure retains only a valid observation", func(t *testing.T) {
		failure := errors.New("worker exited unexpectedly")
		for _, storedVersion := range []string{identity.OSVersion, "old-os"} {
			stored := Observation{Identity: identity, CheckedAt: time.Unix(100, 0), Available: true}
			stored.Identity.OSVersion = storedVersion
			capability := NewCapability(checkBackend{check: func(_ context.Context) error { return failure }}, identity, stored)
			t.Cleanup(func() { capability.Stop(); capability.Wait() })
			<-capability.Check(context.Background())
			state := capability.State()
			valid := storedVersion == identity.OSVersion
			if state.Available != valid || state.Known != valid || !errors.Is(state.Err, failure) || state.Checking {
				t.Fatalf("failure with stored %q: %+v", storedVersion, state)
			}
		}
	})
	t.Run("backend loss invalidates once and cannot invalidate a later generation", func(t *testing.T) {
		stored := Observation{Identity: identity, CheckedAt: time.Unix(100, 0), Available: true}
		capability := NewCapability(checkBackend{check: func(_ context.Context) error { return nil }}, identity, stored)
		t.Cleanup(func() { capability.Stop(); capability.Wait() })
		old := capability.Snapshot()
		if !capability.Invalidate(old.Generation) || capability.Invalidate(old.Generation) {
			t.Fatal("backend loss was not admitted exactly once")
		}
		if state := capability.State(); state.Known || state.Available {
			t.Fatalf("invalidated state = %+v", state)
		}
		<-capability.Ensure(context.Background())
		if capability.Invalidate(old.Generation) || !capability.Snapshot().Available || !old.Available {
			t.Fatal("old operation changed its captured policy or invalidated recovered support")
		}
	})
	t.Run("failed automatic check does not create a retry loop", func(t *testing.T) {
		var calls atomic.Int32
		capability := NewCapability(checkBackend{check: func(_ context.Context) error {
			calls.Add(1)
			return context.DeadlineExceeded
		}}, identity, Observation{})
		t.Cleanup(func() { capability.Stop(); capability.Wait() })
		<-capability.Ensure(context.Background())
		<-capability.Ensure(context.Background())
		if calls.Load() != 1 {
			t.Fatalf("automatic checks = %d, want 1", calls.Load())
		}
		<-capability.Check(context.Background())
		if calls.Load() != 2 {
			t.Fatal("explicit check must remain available after failure")
		}
	})
	t.Run("native backend disappearance invalidates captured availability", func(t *testing.T) {
		stored := Observation{Identity: identity, CheckedAt: time.Unix(100, 0), Available: true}
		capability := NewCapability(checkBackend{check: func(_ context.Context) error { return nil }}, identity, stored)
		t.Cleanup(func() { capability.Stop(); capability.Wait() })
		snapshot := capability.Snapshot()
		_, err := snapshot.Backend.Read(context.Background(), []byte{1}, Request{})
		if !errors.Is(err, ErrUnavailable) {
			t.Fatalf("read = %v", err)
		}
		if capability.State().Known || capability.State().Available {
			t.Fatal("native backend disappeared but persisted support remained effective")
		}
		if !snapshot.Available {
			t.Fatal("invalidation rewrote an admitted operation's snapshot")
		}
	})
}
