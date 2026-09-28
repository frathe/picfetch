package launch

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/frathe/picfetch/internal/explorertrial"
	"github.com/frathe/picfetch/internal/locationtrial"
	"github.com/frathe/picfetch/internal/similarity"
)

func preparationPolicy(t *testing.T, purpose Purpose, dir string) Policy {
	t.Helper()
	options := Options{}
	switch purpose {
	case ExplorerTrial:
		options.ExplorerTrial = dir
	case LocationMapTrial:
		options.LocationMapTrial = dir
	default:
		t.Fatalf("unsupported trial purpose %v", purpose)
	}
	policy, err := NewPolicy(options, "normal-app", false)
	if err != nil {
		t.Fatal(err)
	}
	return policy
}

func TestLaunchPreparationContract(t *testing.T) {
	t.Run("reservation", func(t *testing.T) {
		for _, purpose := range []Purpose{ExplorerTrial, LocationMapTrial} {
			name := "explorer"
			if purpose == LocationMapTrial {
				name = "location"
			}
			t.Run(name, func(t *testing.T) {
				options := PreparationOptions{VerifyOffline: func(_ context.Context) error { return nil }}
				t.Run("new", func(t *testing.T) {
					path := filepath.Join(t.TempDir(), "new-trial")
					owner, err := Prepare(context.Background(), preparationPolicy(t, purpose, path), options)
					if err != nil || owner == nil {
						t.Fatalf("new trial was not reserved: owner=%v err=%v", owner, err)
					}
					if _, err := os.Stat(path); err != nil {
						t.Fatalf("reserved directory: %v", err)
					}
					_ = owner.Close() // An empty Explorer trial is deliberately incomplete.
				})
				t.Run("invalid", func(t *testing.T) {
					path := filepath.Join(t.TempDir(), "missing-parent", "trial")
					owner, err := Prepare(context.Background(), preparationPolicy(t, purpose, path), options)
					if err == nil || owner != nil {
						t.Fatalf("invalid path accepted: owner=%v err=%v", owner, err)
					}
				})
				t.Run("used", func(t *testing.T) {
					path := filepath.Join(t.TempDir(), "used-trial")
					if err := os.Mkdir(path, 0o700); err != nil {
						t.Fatal(err)
					}
					marker := filepath.Join(path, "prior-evidence")
					if err := os.WriteFile(marker, []byte("leave intact"), 0o600); err != nil {
						t.Fatal(err)
					}
					owner, err := Prepare(context.Background(), preparationPolicy(t, purpose, path), options)
					if err == nil || owner != nil {
						t.Fatalf("used path accepted: owner=%v err=%v", owner, err)
					}
					data, err := os.ReadFile(marker)
					if err != nil || string(data) != "leave intact" {
						t.Fatalf("prior evidence changed: data=%q err=%v", data, err)
					}
				})
				t.Run("competing", func(t *testing.T) {
					path := filepath.Join(t.TempDir(), "competing-trial")
					policy := preparationPolicy(t, purpose, path)
					start := make(chan struct{})
					type result struct {
						owner *Prepared
						err   error
					}
					results := make(chan result, 2)
					var ready sync.WaitGroup
					ready.Add(2)
					for range 2 {
						go func() {
							ready.Done()
							<-start
							owner, err := Prepare(context.Background(), policy, options)
							results <- result{owner, err}
						}()
					}
					ready.Wait()
					close(start)
					first, second := <-results, <-results
					succeeded, failed := 0, 0
					for _, got := range []result{first, second} {
						if got.err == nil && got.owner != nil {
							succeeded++
							_ = got.owner.Close()
						} else if got.err != nil && got.owner == nil {
							failed++
						}
					}
					if succeeded != 1 || failed != 1 {
						t.Fatalf("competing reservations: first=%+v second=%+v", first, second)
					}
				})
			})
		}
	})
	t.Run("resources", func(t *testing.T) {
		t.Run("missing policy refuses external operations", func(t *testing.T) {
			called := make(chan string, 3)
			owner, err := Prepare(context.Background(), Policy{}, PreparationOptions{
				VerifyOffline: func(_ context.Context) error { called <- "probe"; return nil },
				NewExplorer:   func(_ string) (*explorertrial.Session, error) { called <- "explorer"; return nil, nil },
				NewLocation:   func(_ string) (*locationtrial.Recorder, error) { called <- "location"; return nil, nil },
			})
			if !errors.Is(err, ErrInvalidPolicy) || owner != nil {
				t.Fatalf("missing policy accepted: owner=%v err=%v", owner, err)
			}
			select {
			case operation := <-called:
				t.Fatalf("missing policy called %s", operation)
			default:
			}
		})
		t.Run("ordinary owns no external resource", func(t *testing.T) {
			policy, err := NewPolicy(Options{}, "normal-app", false)
			if err != nil {
				t.Fatal(err)
			}
			called := make(chan string, 3)
			owner, err := Prepare(context.Background(), policy, PreparationOptions{
				VerifyOffline: func(_ context.Context) error { called <- "probe"; return nil },
				NewExplorer:   func(_ string) (*explorertrial.Session, error) { called <- "explorer"; return nil, nil },
				NewLocation:   func(_ string) (*locationtrial.Recorder, error) { called <- "location"; return nil, nil },
			})
			if err != nil || owner == nil || owner.Policy() != policy || owner.ExplorerTrial() != nil || owner.LocationMapTrial() != nil || owner.Close() != nil {
				t.Fatalf("ordinary preparation changed: owner=%v err=%v", owner, err)
			}
			select {
			case operation := <-called:
				t.Fatalf("ordinary launch called %s", operation)
			default:
			}
		})
		t.Run("Explorer prerequisite refuses before reservation", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "unreserved")
			probeErr := errors.New("offline denial unavailable")
			called := make(chan struct{}, 1)
			owner, err := Prepare(context.Background(), preparationPolicy(t, ExplorerTrial, path), PreparationOptions{
				VerifyOffline: func(_ context.Context) error { return probeErr },
				NewExplorer:   func(_ string) (*explorertrial.Session, error) { called <- struct{}{}; return nil, nil },
			})
			if !errors.Is(err, probeErr) || owner != nil {
				t.Fatalf("Explorer prerequisite ignored: owner=%v err=%v", owner, err)
			}
			select {
			case <-called:
				t.Fatal("reservation attempted after prerequisite refusal")
			default:
			}
			if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("refused trial path exists: %v", err)
			}
		})
		t.Run("Location does not probe", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "location")
			owner, err := Prepare(context.Background(), preparationPolicy(t, LocationMapTrial, path), PreparationOptions{
				VerifyOffline: func(_ context.Context) error { t.Fatal("Location probed Explorer prerequisite"); return nil },
			})
			if err != nil || owner == nil || owner.LocationMapTrial() == nil || owner.ExplorerTrial() != nil {
				t.Fatalf("Location preparation: owner=%v err=%v", owner, err)
			}
			if err := owner.Close(); err != nil {
				t.Fatal(err)
			}
		})
		t.Run("nil acquisition cannot claim a reservation", func(t *testing.T) {
			for _, tc := range []struct {
				name    string
				purpose Purpose
				options PreparationOptions
			}{
				{"Explorer", ExplorerTrial, PreparationOptions{
					VerifyOffline: func(_ context.Context) error { return nil },
					NewExplorer:   func(_ string) (*explorertrial.Session, error) { return nil, nil },
				}},
				{"Location", LocationMapTrial, PreparationOptions{
					NewLocation: func(_ string) (*locationtrial.Recorder, error) { return nil, nil },
				}},
			} {
				t.Run(tc.name, func(t *testing.T) {
					path := filepath.Join(t.TempDir(), "unreserved")
					owner, err := Prepare(context.Background(), preparationPolicy(t, tc.purpose, path), tc.options)
					if err == nil || owner != nil {
						t.Fatalf("nil acquisition accepted: owner=%v err=%v", owner, err)
					}
				})
			}
		})
		t.Run("cancellation refuses and closes acquisition", func(t *testing.T) {
			t.Run("before", func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				path := filepath.Join(t.TempDir(), "not-created")
				owner, err := Prepare(ctx, preparationPolicy(t, LocationMapTrial, path), PreparationOptions{})
				if !errors.Is(err, context.Canceled) || owner != nil {
					t.Fatalf("canceled preparation accepted: owner=%v err=%v", owner, err)
				}
				if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("canceled preparation created a trial: %v", err)
				}
			})
			t.Run("after Explorer probe", func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				path := filepath.Join(t.TempDir(), "not-created")
				owner, err := Prepare(ctx, preparationPolicy(t, ExplorerTrial, path), PreparationOptions{
					VerifyOffline: func(_ context.Context) error { cancel(); return nil },
				})
				if !errors.Is(err, context.Canceled) || owner != nil {
					t.Fatalf("post-probe cancellation accepted: owner=%v err=%v", owner, err)
				}
				if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("post-probe cancellation reserved a trial: %v", err)
				}
			})
			t.Run("after Location acquisition", func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				path := filepath.Join(t.TempDir(), "location")
				owner, err := Prepare(ctx, preparationPolicy(t, LocationMapTrial, path), PreparationOptions{
					NewLocation: func(dir string) (*locationtrial.Recorder, error) {
						recorder, err := locationtrial.New(dir)
						if err == nil {
							recorder.Publish(locationtrial.State{Images: 9})
							cancel()
						}
						return recorder, err
					},
				})
				if !errors.Is(err, context.Canceled) || owner != nil {
					t.Fatalf("post-acquisition cancellation accepted: owner=%v err=%v", owner, err)
				}
				if _, err := os.Stat(filepath.Join(path, "state.json")); err != nil {
					t.Fatalf("canceled acquisition was not flushed: %v", err)
				}
			})
		})
	})
	t.Run("evidence", func(t *testing.T) {
		t.Run("Explorer incomplete evidence and repeated close", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "explorer")
			owner, err := Prepare(context.Background(), preparationPolicy(t, ExplorerTrial, path), PreparationOptions{
				VerifyOffline: func(_ context.Context) error { return nil },
			})
			if err != nil || owner == nil || owner.ExplorerTrial() == nil || owner.LocationMapTrial() != nil {
				t.Fatalf("Explorer owner absent: owner=%v err=%v", owner, err)
			}
			first := owner.Close()
			// Identity, not just wrapping, proves finalization was memoized.
			//goland:noinspection GoDirectComparisonOfErrors
			if first == nil || owner.Close() != first {
				t.Fatalf("incomplete Explorer finalization changed on repeat: first=%v", first)
			}
			data, err := os.ReadFile(filepath.Join(path, "session.json"))
			if err != nil {
				t.Fatalf("session summary not finalized: %v", err)
			}
			var summary explorertrial.Summary
			if err := json.Unmarshal(data, &summary); err != nil || summary.Collected {
				t.Fatalf("empty Explorer evidence was marked complete: summary=%+v err=%v", summary, err)
			}
			if _, err := os.Stat(filepath.Join(path, "events.jsonl")); err != nil {
				t.Fatalf("Explorer events missing: %v", err)
			}
		})
		t.Run("partial Explorer acquisition keeps evidence and both errors", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "partial-explorer")
			acquireErr := errors.New("opening failed after reservation")
			options := PreparationOptions{
				VerifyOffline: func(_ context.Context) error { return nil },
				NewExplorer: func(dir string) (*explorertrial.Session, error) {
					session, err := explorertrial.New(dir)
					if err != nil {
						return nil, err
					}
					return session, acquireErr
				},
			}
			owner, err := Prepare(context.Background(), preparationPolicy(t, ExplorerTrial, path), options)
			if owner != nil || !errors.Is(err, acquireErr) || !strings.Contains(err.Error(), "native trial has no completed map") {
				t.Fatalf("partial Explorer errors lost: owner=%v err=%v", owner, err)
			}
			if _, statErr := os.Stat(filepath.Join(path, "session.json")); statErr != nil {
				t.Fatalf("partial Explorer evidence not finalized: %v", statErr)
			}
			owner, retryErr := Prepare(context.Background(), preparationPolicy(t, ExplorerTrial, path), options)
			if retryErr == nil || owner != nil {
				t.Fatalf("used partial evidence was reopened: owner=%v err=%v", owner, retryErr)
			}
			fresh := filepath.Join(t.TempDir(), "fresh-explorer")
			owner, err = Prepare(context.Background(), preparationPolicy(t, ExplorerTrial, fresh), PreparationOptions{VerifyOffline: func(_ context.Context) error { return nil }})
			if err != nil || owner == nil {
				t.Fatalf("fresh Explorer retry refused: owner=%v err=%v", owner, err)
			}
			_ = owner.Close() // Empty collection stays incomplete.
		})
		t.Run("Location final flush and repeated close", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "location")
			owner, err := Prepare(context.Background(), preparationPolicy(t, LocationMapTrial, path), PreparationOptions{})
			if err != nil || owner == nil || owner.LocationMapTrial() == nil {
				t.Fatalf("Location owner absent: owner=%v err=%v", owner, err)
			}
			owner.LocationMapTrial().Publish(locationtrial.State{Sequence: 42, Images: 3, Formats: map[string]int{"jpg": 3}})
			if err := owner.Close(); err != nil {
				t.Fatalf("Location final flush: %v", err)
			}
			if err := owner.Close(); err != nil {
				t.Fatalf("repeated Location close: %v", err)
			}
			owner.LocationMapTrial().Publish(locationtrial.State{Sequence: 99})
			data, err := os.ReadFile(filepath.Join(path, "state.json"))
			if err != nil {
				t.Fatalf("Location snapshot missing: %v", err)
			}
			var state locationtrial.State
			if err := json.Unmarshal(data, &state); err != nil || state.Sequence != 42 || state.Formats["jpg"] != 3 {
				t.Fatalf("final Location snapshot changed: state=%+v err=%v", state, err)
			}
		})
		t.Run("Location flush error remains stable on repeated close", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "location-write-failure")
			owner, err := Prepare(context.Background(), preparationPolicy(t, LocationMapTrial, path), PreparationOptions{})
			if err != nil || owner == nil {
				t.Fatalf("Location owner absent: owner=%v err=%v", owner, err)
			}
			if err := os.Mkdir(filepath.Join(path, "state.json"), 0o700); err != nil {
				t.Fatal(err)
			}
			owner.LocationMapTrial().Publish(locationtrial.State{Images: 1})
			first := owner.Close()
			// Identity, not just wrapping, proves finalization was memoized.
			//goland:noinspection GoDirectComparisonOfErrors
			if first == nil || owner.Close() != first {
				t.Fatalf("Location write failure changed on repeat: first=%v", first)
			}
		})
		t.Run("partial Location acquisition joins flush failure", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "partial-location")
			acquireErr := errors.New("opening failed after reservation")
			owner, err := Prepare(context.Background(), preparationPolicy(t, LocationMapTrial, path), PreparationOptions{
				NewLocation: func(dir string) (*locationtrial.Recorder, error) {
					recorder, err := locationtrial.New(dir)
					if err != nil {
						return nil, err
					}
					if err := os.Mkdir(filepath.Join(dir, "state.json"), 0o700); err != nil {
						recorder.Stop()
						_ = recorder.Wait()
						return nil, err
					}
					recorder.Publish(locationtrial.State{Images: 5})
					return recorder, acquireErr
				},
			})
			if owner != nil || !errors.Is(err, acquireErr) || !strings.Contains(err.Error(), "state.json") {
				t.Fatalf("partial Location errors lost: owner=%v err=%v", owner, err)
			}
			if _, statErr := os.Stat(filepath.Join(path, "state.json")); statErr != nil {
				t.Fatalf("failed Location evidence removed: %v", statErr)
			}
			owner, retryErr := Prepare(context.Background(), preparationPolicy(t, LocationMapTrial, path), PreparationOptions{})
			if retryErr == nil || owner != nil {
				t.Fatalf("used Location evidence was reopened: owner=%v err=%v", owner, retryErr)
			}
			fresh := filepath.Join(t.TempDir(), "fresh-location")
			owner, err = Prepare(context.Background(), preparationPolicy(t, LocationMapTrial, fresh), PreparationOptions{})
			if err != nil || owner == nil || owner.Close() != nil {
				t.Fatalf("fresh Location retry refused: owner=%v err=%v", owner, err)
			}
		})
		t.Run("default Explorer probe reports actual platform result", func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "native-explorer")
			owner, err := Prepare(context.Background(), preparationPolicy(t, ExplorerTrial, path), PreparationOptions{})
			if !similarity.EnforcesNetworkIsolation() && (err == nil || owner != nil) {
				t.Fatalf("platform without OS denial admitted Explorer: owner=%v err=%v", owner, err)
			}
			if err != nil {
				if owner != nil {
					t.Fatalf("probe refusal returned a resource: owner=%v err=%v", owner, err)
				}
				if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
					t.Fatalf("probe refusal reserved evidence: %v", statErr)
				}
				t.Logf("native Explorer prerequisite refused: %v", err)
				return
			}
			if owner == nil || owner.ExplorerTrial() == nil {
				t.Fatal("successful native prerequisite returned no Explorer resource")
			}
			_ = owner.Close() // An empty Explorer trial has no completed analysis.
			t.Log("native Explorer prerequisite accepted")
		})
		t.Run("nil Prepared methods", func(t *testing.T) {
			var owner *Prepared
			if owner.Policy().Valid() || owner.ExplorerTrial() != nil || owner.LocationMapTrial() != nil || owner.Close() != nil {
				t.Fatal("nil preparation owner was not inert")
			}
		})
	})
}
