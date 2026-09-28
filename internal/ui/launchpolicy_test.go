package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	fynetest "fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/launch"
	"github.com/frathe/picfetch/internal/locationtrial"
	"github.com/frathe/picfetch/internal/preferences"
	"github.com/frathe/picfetch/internal/similarity"
	explorerui "github.com/frathe/picfetch/internal/ui/explorer"
	"github.com/frathe/picfetch/internal/ui/settingswin"
	"github.com/frathe/picfetch/internal/uitest"
	"github.com/frathe/picfetch/internal/update"
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
			view, window, err := buildStartupViewer(unopenedLaunchApp{t: t}, launch.Policy{}, func(_ fyne.App) (launch.Storage, error) {
				t.Fatal("absent policy resolved storage")
				return launch.Storage{}, nil
			})
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
		t.Run("denied_storage_before_preferences_and_consumers", func(t *testing.T) {
			refusal := errors.New("ordinary storage unavailable")
			view, window, err := buildStartupViewer(unopenedLaunchApp{t: t}, testLaunchPolicy(t, launch.Options{}, false), func(_ fyne.App) (launch.Storage, error) { return launch.Storage{}, refusal })
			if !errors.Is(err, refusal) || view != nil || window != nil {
				t.Fatalf("storage refusal: view=%v window=%v err=%v", view, window, err)
			}
		})
		t.Run("ordinary_fallbacks_and_identity_derived_cache", func(t *testing.T) {
			configDir, err := os.UserConfigDir()
			if err != nil {
				t.Fatal(err)
			}
			cacheDir, err := os.UserCacheDir()
			if err != nil {
				cacheDir = os.TempDir()
			}
			application := fynetest.NewApp()
			cache := &launchRootCache{Cache: application.Cache(), root: storage.NewFileURI(t.TempDir())}
			identified := launchRootApp{App: application, cache: cache}
			roots, err := ordinaryLaunchStorage(identified)
			if err != nil {
				t.Fatal(err)
			}
			want := launch.Storage{FavoritesDir: filepath.Join(configDir, "picfetch", "favorites"), PresetsDir: filepath.Join(configDir, "picfetch", "presets"), AnalysisDir: filepath.Join(cache.root.Path(), "image-analysis"), UpdatesDir: filepath.Join(cacheDir, "picfetch", "updates")}
			if roots != want || cache.reads != 1 {
				t.Fatalf("ordinary defaults/app cache: %+v, want %+v; app-root reads=%d", roots, want, cache.reads)
			}
		})
		for _, tc := range []struct {
			name    string
			purpose launch.Purpose
			store   bool
		}{
			{"ordinary_portable", launch.Ordinary, false}, {"ordinary_store", launch.Ordinary, true},
			{"explorer_portable", launch.ExplorerTrial, false}, {"explorer_store", launch.ExplorerTrial, true},
			{"location_map_portable", launch.LocationMapTrial, false}, {"location_map_store", launch.LocationMapTrial, true},
		} {
			t.Run(tc.name+"_consumer_construction", func(t *testing.T) {
				dir := filepath.Join(t.TempDir(), "trial")
				opts := launch.Options{}
				if tc.purpose == launch.ExplorerTrial {
					opts.ExplorerTrial = dir
				}
				if tc.purpose == launch.LocationMapTrial {
					opts.LocationMapTrial = dir
				}
				policy := testLaunchPolicy(t, opts, tc.store)
				prepared, err := launch.Prepare(context.Background(), policy, launch.PreparationOptions{VerifyOffline: func(_ context.Context) error { return nil }})
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = prepared.Close() })
				application := fynetest.NewApp()
				fallback, err := testLaunchStorage(t)(application)
				if err != nil {
					t.Fatal(err)
				}
				resolved := 0
				view, window, err := buildStartupViewer(application, prepared.Policy(), func(app fyne.App) (launch.Storage, error) {
					resolved++
					if app != application || tc.purpose != launch.Ordinary {
						t.Fatal("trial resolved ordinary storage or used another app")
					}
					return fallback, nil
				})
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(window.Close)
				t.Cleanup(func() { drain(t, view) })
				want := fallback
				if tc.purpose != launch.Ordinary {
					want = launch.Storage{FavoritesDir: filepath.Join(dir, "favorites"), PresetsDir: filepath.Join(dir, "presets"), AnalysisDir: filepath.Join(dir, "image-analysis"), UpdatesDir: filepath.Join(dir, "updates")}
				}
				if (tc.purpose == launch.Ordinary && resolved != 1) || (tc.purpose != launch.Ordinary && resolved != 0) {
					t.Errorf("ordinary fallback calls: %d", resolved)
				}
				// Observe the completed construction boundary, before flags/runtime can
				// retarget anything, then exercise the cache's captured worker roots.
				if view.favorites.Dir() != want.FavoritesDir {
					t.Errorf("Favorites construction root: %q, want %q", view.favorites.Dir(), want.FavoritesDir)
				}
				if view.explorer.Options().Presets.Dir != want.PresetsDir {
					t.Errorf("Explorer constructor root: %q, want %q", view.explorer.Options().Presets.Dir, want.PresetsDir)
				}
				if view.updater.Dir() != want.UpdatesDir {
					t.Errorf("updater constructor root: %q, want %q", view.updater.Dir(), want.UpdatesDir)
				}
				provider := &launchCacheProbe{roots: make(chan similarity.CacheRoots, 1)}
				cacheOptions := view.analysisCache.Options()
				cacheOptions.Provider, cacheOptions.Queue = provider, &uitest.UIQueue{}
				view.analysisCache.Configure(cacheOptions)
				view.analysisCache.Content(true, 2048)
				view.analysisCache.Settle()
				got := <-provider.roots
				if got.GeneralDir != want.AnalysisDir || got.FavoritesDir != want.FavoritesDir {
					t.Errorf("analysis-cache constructor roots reached worker: %+v, want %+v", got, want)
				}
				view.applyLaunchOptions(launch.Options{LocationMapTrial: filepath.Join(t.TempDir(), "replacement")})
				view.explorer.Close()
				view.locationMap.Close()
				if view.favorites.Dir() != want.FavoritesDir || view.explorer.Options().Presets.Dir != want.PresetsDir || view.updater.Dir() != want.UpdatesDir || view.analysisDir != want.AnalysisDir {
					t.Fatal("later flags/feature close retargeted captured storage")
				}
			})
		}
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
	t.Run("update_entrypoints", func(t *testing.T) {
		for _, tc := range restrictedLaunchCases(t) {
			t.Run(tc.name+"_checks_and_staging", func(t *testing.T) {
				v, _, _ := newTestUIWithPolicy(t, tc.policy)
				v.updater.SetCurrentVersion("0.2.6")
				stage := saveVerifiedUpdateStage(t, v, "v0.2.5", "preserved stage")
				before, err := os.ReadFile(stage.BinaryPath)
				if err != nil {
					t.Fatal(err)
				}
				httpCalls := &launchHTTPProbe{}
				v.updater.SetClient(update.NewClient(update.Config{HTTP: httpCalls, Now: fixedNow("2026-09-28"), Verify: &fakeUpdateVerifier{}, StageDir: v.updater.Dir()}))
				var verifier atomic.Int32
				v.updater.SetVerifierFactory(func() (update.Verifier, error) {
					verifier.Add(1)
					return nil, errors.New("forbidden verifier construction")
				})
				v.settings.checkForUpdates = true
				v.maybeStartUpdateCheck()
				v.SetCheckForUpdates(true)
				if v.CheckForUpdates() || v.currentPreferences().CheckForUpdates {
					t.Error("captured restriction accepted automatic checks")
				}
				for _, state := range []string{"initial", "closed", "replaced"} {
					if state != "initial" {
						v.explorer.Close()
						v.locationMap.Close()
					}
					if state == "replaced" {
						options := v.explorer.Options()
						options.Trial = nil
						v.explorer.Configure(options)
						v.locationTrial = nil
					}
					failures, progress, successes := 0, 0, 0
					v.CheckForUpdatesNow(settingswin.UpdateCallbacks{
						Failed: func(err error) {
							if err == nil {
								t.Error("empty refusal")
							}
							failures++
						},
						Downloading: func(_ string) { progress++ }, Progress: func(_, _ int64) { progress++ },
						Current: func() { successes++ }, Ready: func(_ string) { successes++ },
					})
					if err := v.updater.Settle(context.Background()); err != nil {
						t.Fatal(err)
					}
					if failures != 1 || progress != 0 || successes != 0 {
						t.Errorf("%s callback protocol: failed=%d progress=%d success=%d", state, failures, progress, successes)
					}
				}
				if verifier.Load() != 0 || httpCalls.calls.Load() != 0 || v.updater.Done().Begun() || v.updater.Busy() {
					t.Errorf("restricted checks admitted work: verifier=%d http=%d begun=%t busy=%t", verifier.Load(), httpCalls.calls.Load(), v.updater.Done().Begun(), v.updater.Busy())
				}
				if data, err := os.ReadFile(stage.BinaryPath); err != nil || !bytes.Equal(data, before) {
					t.Errorf("restricted entry point changed stale stage: %q %v", data, err)
				}
			})
		}
	})
	t.Run("update_records", func(t *testing.T) {
		for _, tc := range append(restrictedLaunchCases(t), namedLaunchPolicy{"ordinary", testLaunchPolicy(t, launch.Options{}, false)}) {
			t.Run(tc.name+"_last_check_restore_and_persist", func(t *testing.T) {
				app := fynetest.NewApp()
				const restored = "2026-08-25"
				app.Preferences().SetString("lastUpdateCheckDay", restored)
				prefs := &launchPreferencesProbe{Preferences: app.Preferences()}
				observed := launchPreferencesApp{App: app, prefs: prefs}
				v, win, err := buildStartupViewer(observed, tc.policy, testLaunchStorage(t))
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(win.Close)
				t.Cleanup(func() { drain(t, v) })
				if v.LastUpdateCheckDay() != restored || prefs.writes.Load() != 0 {
					t.Errorf("restoration performed update persistence: day=%q writes=%d", v.LastUpdateCheckDay(), prefs.writes.Load())
				}
				v.SetLastUpdateCheckDay("2026-09-28")
				want, writes := restored, int32(0)
				if tc.name == "ordinary" {
					want, writes = "2026-09-28", 1
				}
				if v.LastUpdateCheckDay() != want || prefs.writes.Load() != writes {
					t.Errorf("check-result persistence: day=%q writes=%d, want %q/%d", v.LastUpdateCheckDay(), prefs.writes.Load(), want, writes)
				}
				preferences.Save(observed, v.currentPreferences())
				if app.Preferences().String("lastUpdateCheckDay") != want || prefs.writes.Load() != writes {
					t.Error("general preferences changed update-specific persistence")
				}
			})
		}
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

type namedLaunchPolicy struct {
	name   string
	policy launch.Policy
}

type launchHTTPProbe struct{ calls atomic.Int32 }

func (p *launchHTTPProbe) Do(_ *http.Request) (*http.Response, error) {
	p.calls.Add(1)
	return nil, errors.New("forbidden update HTTP call")
}

func restrictedLaunchCases(t *testing.T) []namedLaunchPolicy {
	t.Helper()
	var cases []namedLaunchPolicy
	for _, tc := range []struct {
		name    string
		purpose launch.Purpose
		store   bool
	}{
		{"store", launch.Ordinary, true}, {"explorer", launch.ExplorerTrial, false}, {"location_map", launch.LocationMapTrial, false},
		{"store_explorer", launch.ExplorerTrial, true}, {"store_location_map", launch.LocationMapTrial, true},
	} {
		opts := launch.Options{}
		if tc.purpose == launch.ExplorerTrial {
			opts.ExplorerTrial = filepath.Join(t.TempDir(), "trial")
		}
		if tc.purpose == launch.LocationMapTrial {
			opts.LocationMapTrial = filepath.Join(t.TempDir(), "trial")
		}
		cases = append(cases, namedLaunchPolicy{tc.name, testLaunchPolicy(t, opts, tc.store)})
	}
	return cases
}

type launchPreferencesApp struct {
	fyne.App
	prefs fyne.Preferences
}

func (a launchPreferencesApp) Preferences() fyne.Preferences { return a.prefs }

type launchPreferencesProbe struct {
	fyne.Preferences
	writes atomic.Int32
}

func (p *launchPreferencesProbe) SetString(key, value string) {
	if key == "lastUpdateCheckDay" {
		p.writes.Add(1)
	}
	p.Preferences.SetString(key, value)
}

type launchCacheProbe struct {
	similarity.CacheMaintenanceProvider
	roots chan similarity.CacheRoots
}

type launchRootApp struct {
	fyne.App
	cache fyne.Cache
}

func (a launchRootApp) Cache() fyne.Cache { return a.cache }

type launchRootCache struct {
	fyne.Cache
	root  fyne.URI
	reads int
}

func (c *launchRootCache) RootURI() fyne.URI { c.reads++; return c.root }

func (p *launchCacheProbe) Inspect(_ context.Context, roots similarity.CacheRoots, _ func(similarity.CacheProgress)) (similarity.CacheUsage, error) {
	p.roots <- roots
	return similarity.CacheUsage{}, nil
}

func runProductionShutdownHook(v *viewer) {
	lifecycle := v.app.Lifecycle().(interface{ OnStopped() func() })
	previous := lifecycle.OnStopped()
	registerShutdown(v.app, v)
	shutdown := lifecycle.OnStopped()
	v.app.Lifecycle().SetOnStopped(previous)
	shutdown()
}
