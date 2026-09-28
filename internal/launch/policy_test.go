package launch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/frathe/picfetch/internal/explorertrial"
)

func TestLaunchPolicyContract(t *testing.T) {
	t.Run("matrix", func(t *testing.T) {
		cases := []struct {
			name         string
			options      Options
			storeManaged bool
			purpose      Purpose
			allowed      bool
			reasons      []UpdateReason
		}{
			{"portable ordinary", Options{}, false, Ordinary, true, nil},
			{"store ordinary", Options{}, true, Ordinary, false, []UpdateReason{StoreManagedUpdates}},
			{"portable explorer", Options{ExplorerTrial: "explorer"}, false, ExplorerTrial, false, []UpdateReason{TrialUpdates}},
			{"store explorer", Options{ExplorerTrial: "explorer"}, true, ExplorerTrial, false, []UpdateReason{StoreManagedUpdates, TrialUpdates}},
			{"portable location", Options{LocationMapTrial: "location"}, false, LocationMapTrial, false, []UpdateReason{TrialUpdates}},
			{"store location", Options{LocationMapTrial: "location"}, true, LocationMapTrial, false, []UpdateReason{StoreManagedUpdates, TrialUpdates}},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				policy, err := NewPolicy(tc.options, "normal-app", tc.storeManaged)
				if err != nil {
					t.Fatal(err)
				}
				if !policy.Valid() || policy.Purpose() != tc.purpose || policy.StoreManaged() != tc.storeManaged {
					t.Fatalf("policy=%+v, want purpose %v and store %v", policy, tc.purpose, tc.storeManaged)
				}
				permission := policy.Updates()
				if permission.Allowed() != tc.allowed || !reflect.DeepEqual(permission.Reasons(), tc.reasons) {
					t.Fatalf("updates allowed=%v reasons=%v, want allowed=%v reasons=%v", permission.Allowed(), permission.Reasons(), tc.allowed, tc.reasons)
				}
			})
		}
	})
	t.Run("identity", func(t *testing.T) {
		ordinary, err := NewPolicy(Options{}, "normal-app", false)
		if err != nil || ordinary.ApplicationID() != "normal-app" || ordinary.TrialDir() != "" {
			t.Fatalf("ordinary identity=%q trial dir=%q err=%v", ordinary.ApplicationID(), ordinary.TrialDir(), err)
		}
		workingDir, err := os.Getwd()
		if err != nil {
			t.Fatal(err)
		}
		relative := filepath.Join("testdata", "..", "launch-policy-relative")
		absolute := filepath.Join(workingDir, "launch-policy-relative")
		explorerOpts := Options{ExplorerTrial: relative}
		explorer, err := NewPolicy(explorerOpts, "normal-app", false)
		if err != nil {
			t.Fatalf("passive Explorer policy required a prerequisite: %v", err)
		}
		explorerOpts.ExplorerTrial = "changed-after-construction"
		if explorer.TrialDir() != absolute || explorer.ApplicationID() != explorertrial.Identity(absolute) {
			t.Fatalf("Explorer identity=%q root=%q, want native runner identity=%q root=%q", explorer.ApplicationID(), explorer.TrialDir(), explorertrial.Identity(absolute), absolute)
		}
		locationOpts := Options{LocationMapTrial: relative}
		location, err := NewPolicy(locationOpts, "normal-app", false)
		if err != nil {
			t.Fatal(err)
		}
		locationOpts.LocationMapTrial = "changed-after-construction"
		legacyIdentity, err := (Options{LocationMapTrial: absolute}).ApplicationID(context.Background(), "normal-app")
		if err != nil || location.ApplicationID() != legacyIdentity || location.TrialDir() != absolute {
			t.Fatalf("Location Map identity=%q root=%q, want existing identity=%q root=%q (err=%v)", location.ApplicationID(), location.TrialDir(), legacyIdentity, absolute, err)
		}
	})
	t.Run("storage", func(t *testing.T) {
		ordinaryRoots := Storage{
			FavoritesDir: "existing-favorites",
			PresetsDir:   "",
			AnalysisDir:  "relative/analysis",
			UpdatesDir:   "existing-updates",
		}
		ordinary, err := NewPolicy(Options{}, "normal-app", false)
		if err != nil {
			t.Fatal(err)
		}
		resolved, err := ordinary.ResolveStorage(ordinaryRoots)
		if err != nil || resolved != ordinaryRoots {
			t.Fatalf("ordinary storage=%+v err=%v, want unchanged %+v", resolved, err, ordinaryRoots)
		}
		for _, tc := range []struct {
			name string
			opts Options
		}{
			{"explorer", Options{ExplorerTrial: filepath.Join(t.TempDir(), "new-explorer")}},
			{"location", Options{LocationMapTrial: filepath.Join(t.TempDir(), "new-location")}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				policy, err := NewPolicy(tc.opts, "normal-app", true)
				if err != nil {
					t.Fatal(err)
				}
				got, err := policy.ResolveStorage(ordinaryRoots)
				want := Storage{
					FavoritesDir: filepath.Join(policy.TrialDir(), "favorites"),
					PresetsDir:   filepath.Join(policy.TrialDir(), "presets"),
					AnalysisDir:  filepath.Join(policy.TrialDir(), "image-analysis"),
					UpdatesDir:   filepath.Join(policy.TrialDir(), "updates"),
				}
				if err != nil || got != want {
					t.Fatalf("trial storage=%+v err=%v, want %+v", got, err, want)
				}
			})
		}
		again, err := ordinary.ResolveStorage(ordinaryRoots)
		if err != nil || again != ordinaryRoots {
			t.Fatalf("another policy changed ordinary storage: %+v err=%v", again, err)
		}
	})
	t.Run("validity", func(t *testing.T) {
		var missing Policy
		if missing.Valid() || missing.Updates().Allowed() || !reflect.DeepEqual(missing.Updates().Reasons(), []UpdateReason{MissingPolicy}) {
			t.Fatalf("missing policy became usable: %+v", missing)
		}
		if _, err := missing.ResolveStorage(Storage{FavoritesDir: "ordinary"}); !errors.Is(err, ErrInvalidPolicy) {
			t.Fatalf("missing storage error=%v, want ErrInvalidPolicy", err)
		}

		ordinary, err := NewPolicy(Options{}, "normal-app", false)
		if err != nil || !ordinary.Valid() || !ordinary.Updates().Allowed() {
			t.Fatalf("explicit zero options: policy=%+v err=%v", ordinary, err)
		}
		for _, tc := range []struct {
			name string
			opts Options
			id   string
		}{
			{"empty identity", Options{}, ""},
			{"dual trial", Options{ExplorerTrial: "explorer", LocationMapTrial: "location"}, "normal-app"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				policy, err := NewPolicy(tc.opts, tc.id, false)
				if !errors.Is(err, ErrInvalidPolicy) || policy.Valid() || policy.Updates().Allowed() || policy.ApplicationID() != "" || policy.TrialDir() != "" {
					t.Fatalf("invalid construction returned policy=%+v err=%v", policy, err)
				}
			})
		}

		permission := func() UpdatePermission {
			policy, err := NewPolicy(Options{ExplorerTrial: "explorer"}, "normal-app", true)
			if err != nil {
				t.Fatal(err)
			}
			return policy.Updates()
		}()
		reasons := permission.Reasons()
		reasons[0] = MissingPolicy
		if !reflect.DeepEqual(permission.Reasons(), []UpdateReason{StoreManagedUpdates, TrialUpdates}) {
			t.Fatalf("caller changed captured reasons: %v", permission.Reasons())
		}
	})
}
