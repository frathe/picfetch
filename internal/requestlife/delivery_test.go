package requestlife_test

import (
	"context"
	"errors"
	"testing"

	"github.com/frathe/picfetch/internal/requestlife"
)

func TestFinalDeliveryContract(t *testing.T) {
	t.Run("abandon before handoff releases request", func(t *testing.T) {
		var owner requestlife.Owner
		token := owner.Begin(context.Background())
		delivery := token.FinalDelivery()
		delivery.Abandon()
		if token.Current() || !errors.Is(token.Context().Err(), context.Canceled) {
			t.Fatal("abandoned delivery kept its request live")
		}
	})

	t.Run("early abandon prevents dispatch and finishing", func(t *testing.T) {
		var owner requestlife.Owner
		token := owner.Begin(context.Background())
		delivery := token.FinalDelivery()
		delivery.Abandon()
		delivery.Abandon()
		dispatched, applied, finished := false, false, false
		delivery.Dispatch(func(_ func()) { dispatched = true }, func() { applied = true }, func() { finished = true })
		if dispatched || applied || finished || token.Current() {
			t.Fatal("abandoned delivery dispatched or retained its request")
		}
	})

	t.Run("inline handoff precedes dispatcher reentry", func(t *testing.T) {
		var owner requestlife.Owner
		token := owner.Begin(context.Background())
		delivery := token.FinalDelivery()
		applied, finished := 0, 0
		delivery.Dispatch(func(run func()) {
			delivery.Abandon()
			if !token.Current() {
				t.Fatal("abandon after handoff cancelled the pending result")
			}
			run()
			run()
		}, func() { applied++ }, func() { finished++ })
		if applied != 1 || finished != 1 || !errors.Is(token.Context().Err(), context.Canceled) {
			t.Fatalf("inline delivery: applied %d, finished %d, error %v", applied, finished, token.Context().Err())
		}
	})

	t.Run("deferred handoff survives worker abandon", func(t *testing.T) {
		var owner requestlife.Owner
		token := owner.Begin(context.Background())
		delivery := token.FinalDelivery()
		var queued func()
		applied, finished := 0, 0
		delivery.Dispatch(func(run func()) { queued = run }, func() { applied++ }, func() { finished++ })
		delivery.Abandon()
		if queued == nil || !token.Current() || applied != 0 || finished != 0 {
			t.Fatal("deferred result was released or delivered before queue drain")
		}
		queued()
		if applied != 1 || finished != 1 || token.Current() {
			t.Fatalf("deferred delivery: applied %d, finished %d, current %v", applied, finished, token.Current())
		}
	})

	t.Run("cancelled queued result skips application but finishes on delivery", func(t *testing.T) {
		var owner requestlife.Owner
		token := owner.Begin(context.Background())
		var queued func()
		applied, finished := 0, 0
		token.FinalDelivery().Dispatch(func(run func()) { queued = run }, func() { applied++ }, func() { finished++ })
		owner.Invalidate()
		if token.Current() || !errors.Is(token.Context().Err(), context.Canceled) || finished != 0 {
			t.Fatal("invalidation waited for or finished held UI delivery")
		}
		queued()
		queued()
		if applied != 0 || finished != 1 {
			t.Fatalf("stale delivery: applied %d, finished %d", applied, finished)
		}
	})

	t.Run("application and finish can begin newer requests", func(t *testing.T) {
		var owner requestlife.Owner
		old := owner.Begin(context.Background())
		var fromApply, fromFinish requestlife.Token
		applied, finished := 0, 0
		old.FinalDelivery().Dispatch(func(run func()) { run() }, func() {
			applied++
			fromApply = owner.Begin(context.Background())
		}, func() {
			finished++
			fromFinish = owner.Begin(context.Background())
		})
		if applied != 1 || finished != 1 || old.Current() || fromApply.Current() || !fromFinish.Current() {
			t.Fatal("reentrant delivery lost a new request or failed to finish")
		}
		if !errors.Is(old.Context().Err(), context.Canceled) || fromFinish.Context().Err() != nil || owner.Revision() != 3 {
			t.Fatal("old cleanup cancelled the newest request")
		}
	})

	t.Run("nil finisher is allowed", func(t *testing.T) {
		var owner requestlife.Owner
		token := owner.Begin(context.Background())
		applied := false
		token.FinalDelivery().Dispatch(func(run func()) { run() }, func() { applied = true }, nil)
		if !applied || token.Current() {
			t.Fatal("delivery with nil finisher did not apply and release")
		}
	})
}
