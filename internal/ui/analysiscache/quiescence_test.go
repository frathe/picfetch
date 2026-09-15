package analysiscache

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"

	"github.com/frathe/picfetch/internal/uitest"
)

func TestJoinRevokedWriters(t *testing.T) {
	t.Run("cancel-before-claim", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			q := &uitest.UIQueue{}
			defer func() { cancel(); q.Drain() }()
			called := false
			done := make(chan error, 1)
			go func() {
				done <- joinRevokedWriters(ctx, q, func() {}, func() []<-chan struct{} {
					called = true
					return nil
				})
			}()
			synctest.Wait()
			if q.Len() != 1 {
				t.Fatal("retirement callback was not queued")
			}
			cancel()
			synctest.Wait()
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("cancellation returned %v", err)
				}
			default:
				t.Fatal("canceling an unclaimed callback required a UI drain")
			}
			q.Drain()
			if called {
				t.Fatal("an abandoned callback retired producers after completion")
			}
		})
	})
	for _, phase := range []string{"claimed", "joining", "uncanceled"} {
		t.Run(phase, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				q := &uitest.UIQueue{}
				first, second := make(chan struct{}), make(chan struct{})
				defer func() {
					cancel()
					for _, barrier := range []chan struct{}{first, second} {
						select {
						case <-barrier:
						default:
							close(barrier)
						}
					}
					q.Drain()
				}()
				done := make(chan error, 1)
				early := false
				calls := 0
				go func() {
					done <- joinRevokedWriters(ctx, q, func() {}, func() []<-chan struct{} {
						calls++
						if phase == "claimed" {
							cancel()
							synctest.Wait()
							select {
							case <-done:
								early = true
							default:
							}
						}
						return []<-chan struct{}{first, nil, second}
					})
				}()
				synctest.Wait()
				q.Drain()
				if early {
					t.Fatal("maintenance completed while a claimed callback still owned producer retirement")
				}
				if phase == "joining" {
					cancel()
				}
				for _, barrier := range []chan struct{}{first, second} {
					synctest.Wait()
					select {
					case <-done:
						t.Fatal("maintenance completed before both retired producers stopped")
					default:
					}
					close(barrier)
				}
				if err := <-done; !errors.Is(err, ctx.Err()) || calls != 1 {
					t.Fatalf("joined retirement: error=%v calls=%d", err, calls)
				}
			})
		})
	}
}
