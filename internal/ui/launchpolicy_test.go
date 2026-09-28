package ui

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/synctest"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/launch"
	"github.com/frathe/picfetch/internal/locationtrial"
	"github.com/frathe/picfetch/internal/similarity"
	explorerui "github.com/frathe/picfetch/internal/ui/explorer"
	"github.com/frathe/picfetch/internal/uitest"
)

func prepareTestLocationTrial(t *testing.T, v *viewer, dir string) *launch.Prepared {
	t.Helper()
	prepared, err := launch.Prepare(context.Background(), testLaunchPolicy(t, launch.Options{LocationMapTrial: dir}, false), launch.PreparationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	v.borrowLocationTrial(prepared.LocationMapTrial())
	t.Cleanup(func() {
		drain(t, v)
		if err := prepared.Close(); err != nil {
			t.Error(err)
		}
	})
	return prepared
}

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
			err := Run(unopenedLaunchApp{t: t}, nil, launch.Options{ExplorerTrial: t.TempDir()}, nil, "", "")
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
	t.Run("shutdown", func(t *testing.T) {
		t.Run("explorer_held_producer_production_hook_postrun", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				v := openGridWith(t, "one.jpg", "two.jpg")
				dir := filepath.Join(t.TempDir(), "trial")
				prepared, err := launch.Prepare(context.Background(), testLaunchPolicy(t, launch.Options{ExplorerTrial: dir}, false), launch.PreparationOptions{VerifyOffline: func(_ context.Context) error { return nil }})
				if err != nil {
					t.Fatal(err)
				}
				entered, release := make(chan struct{}), make(chan struct{})
				unblock := sync.OnceFunc(func() { close(release) })
				defer unblock()
				configureExplorer(v, func(options *explorerui.Options) {
					options.Trial = prepared.ExplorerTrial()
					options.Analyze = func(ctx context.Context, _ []string, _ <-chan similarity.Control, _ func(similarity.Event)) error {
						close(entered)
						<-release
						return ctx.Err()
					}
				})
				explorerMenu(t, v).Action()
				<-entered
				runProductionShutdownHook(v)
				finished := make(chan error, 1)
				go func() { v.waitForShutdown(); finished <- prepared.Close() }()
				synctest.Wait()
				select {
				case err := <-finished:
					t.Fatalf("shutdown passed held Explorer producer: %v", err)
				default:
				}
				if _, err := os.Stat(filepath.Join(dir, "session.json")); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("evidence finalized before producer exit: %v", err)
				}
				unblock()
				if err := <-finished; err == nil || !strings.Contains(err.Error(), "no completed map") {
					t.Fatalf("cancelled trial lost incomplete meaning: %v", err)
				}
				data, err := os.ReadFile(filepath.Join(dir, "events.jsonl"))
				if err != nil {
					t.Fatal(err)
				}
				exit, closed := strings.Index(string(data), "worker-exited"), strings.Index(string(data), "session-closed")
				if exit < 0 || closed <= exit {
					t.Fatalf("evidence closed before observed worker exit: %s", data)
				}
			})
		})
		t.Run("location_map_held_producer_production_hook_postrun", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				v := newTestViewer(t)
				dir := filepath.Join(t.TempDir(), "trial")
				prepared := prepareTestLocationTrial(t, v, dir)
				source, hold, entered, unblock := locationHeldRead(t, uitest.TempGPSJPEGURI(t, "held.jpg", 24, 16, 52.52, 13.405))
				defer unblock()
				dropAndWait(t, v, source)
				v.display.Settle()
				hold.Store(true)
				locationMenu(t, v).Action()
				<-entered
				runProductionShutdownHook(v)
				recorderDone := make(chan error, 1)
				go func() { recorderDone <- prepared.LocationMapTrial().Wait() }()
				finished := make(chan error, 1)
				go func() { v.waitForShutdown(); finished <- prepared.Close() }()
				synctest.Wait()
				select {
				case err := <-recorderDone:
					t.Errorf("production hook stopped recorder before producer join: %v", err)
				default:
				}
				select {
				case err := <-finished:
					t.Errorf("post-run wait passed held metadata read: %v", err)
				default:
				}
				unblock()
				if err := <-finished; err != nil {
					t.Fatal(err)
				}
				data, err := os.ReadFile(filepath.Join(dir, "state.json"))
				if err != nil {
					t.Fatal(err)
				}
				var state locationtrial.State
				if err := json.Unmarshal(data, &state); err != nil {
					t.Fatal(err)
				}
				if state.Active || state.Visible || state.Sequence == 0 {
					t.Fatalf("shutdown snapshot not flushed: %+v", state)
				}
			})
		})
	})
}

func runProductionShutdownHook(v *viewer) {
	lifecycle := v.app.Lifecycle().(interface{ OnStopped() func() })
	previous := lifecycle.OnStopped()
	registerShutdown(v.app, v)
	shutdown := lifecycle.OnStopped()
	v.app.Lifecycle().SetOnStopped(previous)
	shutdown()
}
