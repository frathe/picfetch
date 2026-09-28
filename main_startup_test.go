package main

import (
	"bytes"
	"context"
	"errors"
	"io"
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
			ops.run = func(_ fyne.App, initial []fyne.URI, opts launch.Options) error {
				calls = append(calls, "run")
				if len(initial) != 1 || initial[0].Name() != "photo.jpg" || !opts.PictureFrame {
					t.Fatalf("launch input was not forwarded: %v %+v", initial, opts)
				}
				return nil
			}
			code, err := runStartup([]string{"photo.jpg", "--slideshow"}, io.Discard, io.Discard, ops)
			want := []string{"heic", "similarity", "native"}
			// Both compiled distributions exercise this same ordering guard.
			//goland:noinspection GoBoolExpressions
			if !distribution.StoreManaged {
				want = append(want, "predecessor")
			}
			want = append(want, "identity", "app", "run")
			if code != 0 || err != nil || !slices.Equal(calls, want) {
				t.Fatalf("startup: code=%d err=%v calls=%v want=%v", code, err, calls, want)
			}
		})
		for _, step := range []string{"identity", "app", "run"} {
			t.Run(step+"_failure", func(t *testing.T) {
				failure := errors.New("injected " + step + " failure")
				var calls []string
				ops := startupObservations(&calls)
				switch step {
				case "identity":
					ops.identity = func(_ context.Context, _ launch.Options) (string, error) {
						calls = append(calls, step)
						return "", failure
					}
				case "app":
					ops.newApp = func(_ string) (fyne.App, error) { calls = append(calls, step); return nil, failure }
				case "run":
					ops.run = func(_ fyne.App, _ []fyne.URI, _ launch.Options) error { calls = append(calls, step); return failure }
				}
				code, err := runStartup(nil, io.Discard, io.Discard, ops)
				if code != 1 || !errors.Is(err, failure) || len(calls) == 0 || calls[len(calls)-1] != step {
					t.Fatalf("%s failure: code=%d err=%v calls=%v", step, code, err, calls)
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
		identity: func(_ context.Context, _ launch.Options) (string, error) {
			*calls = append(*calls, "identity")
			return appID, nil
		},
		newApp: func(_ string) (fyne.App, error) { *calls = append(*calls, "app"); return nil, nil },
		run:    func(_ fyne.App, _ []fyne.URI, _ launch.Options) error { *calls = append(*calls, "run"); return nil },
	}
}
