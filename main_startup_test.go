package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"path/filepath"
	"slices"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/distribution"
	"github.com/frathe/picfetch/internal/launch"
)

func TestLaunchStartupContract(t *testing.T) {
	t.Run("early_exit", func(t *testing.T) {
		t.Run("help", func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code, err := runStartup([]string{"--help"}, &stdout, &stderr, startupOps{})
			if err != nil || code != 0 || stdout.Len() == 0 || stderr.Len() != 0 {
				t.Fatalf("help: code=%d err=%v stdout=%q stderr=%q", code, err, &stdout, &stderr)
			}
		})
		t.Run("malformed_flags", func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code, err := runStartup([]string{"--not-a-flag"}, &stdout, &stderr, startupOps{})
			if err != nil || code != 2 || stdout.Len() != 0 || stderr.Len() == 0 {
				t.Fatalf("bad flags: code=%d err=%v stdout=%q stderr=%q", code, err, &stdout, &stderr)
			}
		})
		for _, worker := range []string{"heic", "similarity"} {
			t.Run("private_"+worker, func(t *testing.T) {
				var calls []string
				ops := startupOps{
					heicWorker:       func() bool { calls = append(calls, "heic"); return worker == "heic" },
					similarityWorker: func() bool { calls = append(calls, "similarity"); return true },
				}
				code, err := runStartup([]string{"--location-map-trial", "unused-evidence"}, io.Discard, io.Discard, ops)
				want := []string{"heic"}
				if worker == "similarity" {
					want = append(want, "similarity")
				}
				if code != 0 || err != nil || !slices.Equal(calls, want) {
					t.Fatalf("private dispatch: code=%d err=%v calls=%v want=%v", code, err, calls, want)
				}
			})
		}
	})
	t.Run("ordering", func(t *testing.T) {
		t.Run("native_install_predecessor_ordinary_run", func(t *testing.T) {
			var calls []string
			ops := startupObservations(&calls)
			ops.run = func(_ fyne.App, initial []fyne.URI, opts launch.Options, _ launch.Policy) error {
				calls = append(calls, "run")
				if len(initial) != 1 || initial[0].Name() != "photo.jpg" || !opts.PictureFrame {
					t.Fatalf("launch input was not forwarded: %v %+v", initial, opts)
				}
				return nil
			}
			code, err := runStartup([]string{"photo.jpg", "--slideshow"}, io.Discard, io.Discard, ops)
			want := []string{"heic", "similarity", "native", "capture", "prerequisites"}
			// Both compiled distributions exercise this same ordering guard.
			//goland:noinspection GoBoolExpressions
			if !distribution.StoreManaged {
				want = append(want, "predecessor")
			}
			want = append(want, "app", "run")
			if code != 0 || err != nil || !slices.Equal(calls, want) {
				t.Fatalf("startup: code=%d err=%v calls=%v want=%v", code, err, calls, want)
			}
		})
		for _, step := range []string{"capture", "prerequisites", "app", "run"} {
			t.Run(step+"_failure", func(t *testing.T) {
				failure := errors.New("injected " + step + " failure")
				var calls []string
				ops := startupObservations(&calls)
				switch step {
				case "capture":
					ops.capture = func(_ launch.Options) (launch.Policy, error) {
						calls = append(calls, step)
						return launch.Policy{}, failure
					}
				case "prerequisites":
					ops.prerequisites = func(_ context.Context, _ launch.Policy) error { calls = append(calls, step); return failure }
				case "app":
					ops.newApp = func(_ string) (fyne.App, error) { calls = append(calls, step); return nil, failure }
				case "run":
					ops.run = func(_ fyne.App, _ []fyne.URI, _ launch.Options, _ launch.Policy) error {
						calls = append(calls, step)
						return failure
					}
				}
				code, err := runStartup(nil, io.Discard, io.Discard, ops)
				if code != 1 || !errors.Is(err, failure) || len(calls) == 0 || calls[len(calls)-1] != step {
					t.Fatalf("%s failure: code=%d err=%v calls=%v", step, code, err, calls)
				}
			})
		}
		t.Run("compiled_distribution_and_identity", func(t *testing.T) {
			var calls []string
			ops := startupObservations(&calls)
			ops.capture = productionStartup().capture
			var appIdentity string
			ops.newApp = func(identity string) (fyne.App, error) {
				appIdentity = identity
				return nil, nil
			}
			ops.run = func(_ fyne.App, _ []fyne.URI, _ launch.Options, policy launch.Policy) error {
				if !policy.Valid() || policy.StoreManaged() != distribution.StoreManaged || policy.ApplicationID() != appIdentity || appIdentity != appID {
					t.Fatalf("production did not carry its compiled policy and identity: policy=%+v appID=%q", policy, appIdentity)
				}
				return nil
			}
			if code, err := runStartup(nil, io.Discard, io.Discard, ops); code != 0 || err != nil {
				t.Fatalf("compiled launch: %d %v", code, err)
			}
		})
		for _, flag := range []string{"--explorer-trial", "--location-map-trial"} {
			t.Run("captured_"+flag[2:], func(t *testing.T) {
				var calls []string
				ops := startupObservations(&calls)
				ops.capture = productionStartup().capture
				var identity string
				ops.newApp = func(id string) (fyne.App, error) { identity = id; return nil, nil }
				ops.cleanup = func() { t.Fatal("restricted launch invoked predecessor filesystem access") }
				ops.run = func(_ fyne.App, _ []fyne.URI, _ launch.Options, policy launch.Policy) error {
					if !policy.Valid() || policy.Updates().Allowed() || policy.StoreManaged() != distribution.StoreManaged || policy.ApplicationID() != identity || identity == appID {
						t.Fatalf("trial identity/permission lost: %+v identity=%q", policy, identity)
					}
					return nil
				}
				if code, err := runStartup([]string{flag, filepath.Join(t.TempDir(), "trial")}, io.Discard, io.Discard, ops); code != 0 || err != nil {
					t.Fatalf("trial startup: %d %v", code, err)
				}
			})
		}
	})
	t.Run("validation", func(t *testing.T) {
		for _, invalid := range []string{"dual_trials", "absent_capture"} {
			t.Run(invalid, func(t *testing.T) {
				var calls []string
				ops := startupObservations(&calls)
				ops.capture = productionStartup().capture
				ops.prerequisites = func(_ context.Context, _ launch.Policy) error {
					t.Fatal("invalid launch probed prerequisites")
					return nil
				}
				ops.newApp = func(_ string) (fyne.App, error) { t.Fatal("invalid launch constructed app"); return nil, nil }
				ops.cleanup = func() { t.Fatal("invalid launch performed predecessor cleanup") }
				args := []string{"--explorer-trial", "unused", "--location-map-trial", "unused"}
				if invalid == "absent_capture" {
					args = nil
					ops.capture = func(_ launch.Options) (launch.Policy, error) { return launch.Policy{}, nil }
				}
				if code, err := runStartup(args, io.Discard, io.Discard, ops); code != 1 || !errors.Is(err, launch.ErrInvalidPolicy) {
					t.Fatalf("invalid launch: %d %v", code, err)
				}
			})
		}
	})
}

func startupObservations(calls *[]string) startupOps {
	return startupOps{
		heicWorker:       func() bool { *calls = append(*calls, "heic"); return false },
		similarityWorker: func() bool { *calls = append(*calls, "similarity"); return false },
		installOpenWith:  func() { *calls = append(*calls, "native") },
		cleanup:          func() { *calls = append(*calls, "predecessor") },
		capture: func(opts launch.Options) (launch.Policy, error) {
			*calls = append(*calls, "capture")
			return launch.NewPolicy(opts, appID, distribution.StoreManaged)
		},
		prerequisites: func(_ context.Context, _ launch.Policy) error { *calls = append(*calls, "prerequisites"); return nil },
		newApp:        func(_ string) (fyne.App, error) { *calls = append(*calls, "app"); return nil, nil },
		run: func(_ fyne.App, _ []fyne.URI, _ launch.Options, _ launch.Policy) error {
			*calls = append(*calls, "run")
			return nil
		},
	}
}
