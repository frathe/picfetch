package ui

import (
	"errors"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/launch"
)

// No inherited app operation is usable: refusal must precede construction.
type unopenedLaunchApp struct {
	fyne.App
	t *testing.T
}

func (a unopenedLaunchApp) Cache() fyne.Cache {
	a.t.Fatal("missing policy opened app cache before refusing composition")
	return nil
}

func (a unopenedLaunchApp) Preferences() fyne.Preferences {
	a.t.Fatal("missing policy opened preferences before refusing composition")
	return nil
}

func TestLaunchPolicyIntegration(t *testing.T) {
	t.Run("construction", func(t *testing.T) {
		t.Run("absent_policy_before_storage", func(t *testing.T) {
			view, window, err := buildStartupViewer(unopenedLaunchApp{t: t}, launch.Policy{})
			if !errors.Is(err, launch.ErrInvalidPolicy) || view != nil || window != nil {
				t.Fatalf("absent policy: view=%v window=%v err=%v", view, window, err)
			}
		})
		t.Run("run_absent_policy_before_trial", func(t *testing.T) {
			err := Run(unopenedLaunchApp{t: t}, nil, launch.Options{ExplorerTrial: t.TempDir()}, launch.Policy{}, "", "")
			if !errors.Is(err, launch.ErrInvalidPolicy) {
				t.Fatalf("Run must reject absent policy before trial acquisition: %v", err)
			}
		})
		t.Run("explicit_ordinary", func(t *testing.T) {
			v, _, _ := newTestUIWithPolicy(t, testLaunchPolicy(t, launch.Options{}, false))
			if !v.launchPolicy.Valid() || !v.launchPolicy.Updates().Allowed() || v.launchPolicy.ApplicationID() != "io.github.frathe.picfetch" {
				t.Fatalf("explicit ordinary composition: %+v", v.launchPolicy)
			}
		})
	})
	t.Run("feature_lifetime", func(t *testing.T) {
		for _, purpose := range []string{"explorer", "location_map"} {
			t.Run("immutable_"+purpose, func(t *testing.T) {
				dir := filepath.Join(t.TempDir(), "trial")
				opts := launch.Options{ExplorerTrial: dir}
				if purpose == "location_map" {
					opts = launch.Options{LocationMapTrial: dir}
				}
				policy := testLaunchPolicy(t, opts, false)
				v, _, _ := newTestUIWithPolicy(t, policy)
				opts.ExplorerTrial, opts.LocationMapTrial = "", ""
				v.explorer.Close()
				v.locationMap.Close()
				if v.launchPolicy.Updates().Allowed() || v.launchPolicy.TrialDir() != dir || v.launchPolicy.ApplicationID() != policy.ApplicationID() {
					t.Fatalf("capture changed after input/feature changes: %+v", v.launchPolicy)
				}
			})
		}
		t.Run("independent_viewers", func(t *testing.T) {
			ordinary, _, _ := newTestUIWithPolicy(t, testLaunchPolicy(t, launch.Options{}, false))
			store, _, _ := newTestUIWithPolicy(t, testLaunchPolicy(t, launch.Options{}, true))
			if !ordinary.launchPolicy.Updates().Allowed() || store.launchPolicy.Updates().Allowed() || ordinary.storeManaged || !store.storeManaged {
				t.Fatal("one viewer's launch changed another viewer's captured permission")
			}
		})
	})
}
