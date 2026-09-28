package autoupdate

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/launch"
	"github.com/frathe/picfetch/internal/update"
)

func TestUpdaterLaunchPolicy(t *testing.T) {
	t.Run("admission", func(t *testing.T) {
		for _, tc := range []struct {
			name         string
			trial        launch.Purpose
			storeManaged bool
			missing      bool
		}{
			{name: "store-managed ordinary", storeManaged: true},
			{name: "Explorer trial", trial: launch.ExplorerTrial},
			{name: "Location Map trial", trial: launch.LocationMapTrial},
			{name: "store-managed Explorer trial", trial: launch.ExplorerTrial, storeManaged: true},
			{name: "store-managed Location Map trial", trial: launch.LocationMapTrial, storeManaged: true},
			{name: "missing policy", missing: true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				policy := updaterTestPolicy(t, tc.trial, tc.storeManaged, tc.missing)
				dir := t.TempDir()
				if err := update.SaveStage(dir, update.Stage{Version: "v0.2.5", Notes: "seeded"}); err != nil {
					t.Fatal(err)
				}
				var persisted, verifierCalls, reads, removes, requests atomic.Int32
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					requests.Add(1)
					w.WriteHeader(http.StatusInternalServerError)
				}))
				defer srv.Close()
				u := New(test.NewApp(), dir, policy, func(_ string) { persisted.Add(1) })
				u.SetCurrentVersion("v0.2.6")
				u.SetVerifierFactory(func() (update.Verifier, error) {
					verifierCalls.Add(1)
					return fakeVerifier{}, nil
				})
				u.loadStage = func(dir string) (update.Stage, error) {
					reads.Add(1)
					return update.LoadStage(dir)
				}
				u.removeStage = func(dir string) error {
					removes.Add(1)
					return update.RemoveStage(dir)
				}

				if err := u.EnsureClient(); !errors.Is(err, ErrUpdatesUnavailable) {
					t.Fatalf("EnsureClient error = %v, want unavailable", err)
				}
				u.SetClient(updaterClient(u, srv, fakeVerifier{}, time.Now()))
				if err := u.EnsureClient(); !errors.Is(err, ErrUpdatesUnavailable) {
					t.Errorf("preconfigured EnsureClient error = %v, want unavailable", err)
				}
				if err := u.Start(context.Background(), func() bool { return false }, "v0.2.6"); !errors.Is(err, ErrUpdatesUnavailable) {
					t.Errorf("Start error = %v, want unavailable", err)
				}
				failures := 0
				u.StartManual(context.Background(), func() bool { return false }, "v0.2.6", Events{Failed: func(err error) {
					if !errors.Is(err, ErrUpdatesUnavailable) {
						t.Errorf("manual refusal = %v", err)
					}
					failures++
				}})
				if failures != 1 {
					t.Errorf("manual refusal callbacks = %d, want one", failures)
				}
				u.RemoveStaleStage()
				u.SetLastCheckDay("2026-09-28")
				if got := u.LastCheckDay(); got != "" {
					t.Errorf("restricted SetLastCheckDay changed day to %q", got)
				}
				if u.Done().Begun() || u.Busy() {
					t.Error("restricted calls admitted a completion or worker")
				}
				if got := persisted.Load(); got != 0 {
					t.Errorf("persistence calls = %d, want zero", got)
				}
				if got := verifierCalls.Load(); got != 0 {
					t.Errorf("verifier calls = %d, want zero", got)
				}
				if got := reads.Load(); got != 0 {
					t.Errorf("stage reads = %d, want zero", got)
				}
				if got := removes.Load(); got != 0 {
					t.Errorf("stage removals = %d, want zero", got)
				}
				if got := requests.Load(); got != 0 {
					t.Errorf("HTTP requests = %d, want zero", got)
				}
				if _, err := update.LoadStage(dir); err != nil {
					t.Errorf("seeded stage changed: %v", err)
				}
			})
		}
	})

	t.Run("preconfigured", func(t *testing.T) {
		policy := updaterTestPolicy(t, launch.ExplorerTrial, false, false)
		var requests, reads, removes, persisted atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requests.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()
		dir := t.TempDir()
		if err := update.SaveStage(dir, update.Stage{Version: "v0.2.5"}); err != nil {
			t.Fatal(err)
		}
		u := New(test.NewApp(), dir, policy, func(_ string) { persisted.Add(1) })
		u.SetClient(updaterClient(u, srv, fakeVerifier{}, time.Now()))
		u.loadStage = func(dir string) (update.Stage, error) {
			reads.Add(1)
			return update.LoadStage(dir)
		}
		u.removeStage = func(dir string) error {
			removes.Add(1)
			return update.RemoveStage(dir)
		}
		if err := u.EnsureClient(); !errors.Is(err, ErrUpdatesUnavailable) {
			t.Errorf("preconfigured EnsureClient error = %v, want unavailable", err)
		}
		if err := u.Start(context.Background(), func() bool { return false }, "v0.2.6"); !errors.Is(err, ErrUpdatesUnavailable) {
			t.Errorf("preconfigured Start error = %v, want unavailable", err)
		}
		var failed error
		var otherEvents int
		u.StartManual(context.Background(), func() bool { return false }, "v0.2.6", Events{
			Failed:      func(err error) { failed = err },
			Current:     func() { otherEvents++ },
			Downloading: func(_ string) { otherEvents++ },
			Progress:    func(_ update.DownloadProgress) { otherEvents++ },
			Ready:       func(_ update.Stage) { otherEvents++ },
		})
		if !errors.Is(failed, ErrUpdatesUnavailable) || failed.Error() != lang.L("Updates are unavailable in this session") {
			t.Errorf("synchronous manual failure = %v, want localized unavailable", failed)
		}
		if otherEvents != 0 {
			t.Errorf("other callbacks = %d, want zero", otherEvents)
		}
		cancelled, cancel := context.WithCancel(context.Background())
		cancel()
		failed = nil
		u.StartManual(cancelled, func() bool { return false }, "v0.2.6", Events{Failed: func(err error) { failed = err }})
		if failed != nil {
			t.Errorf("cancelled manual call delivered %v", failed)
		}
		u.StartManual(context.Background(), func() bool { return true }, "v0.2.6", Events{Failed: func(err error) { failed = err }})
		if failed != nil {
			t.Errorf("stale manual call delivered %v", failed)
		}
		u.RemoveStaleStage()
		waitUpdater(t, u)
		if requests.Load() != 0 || reads.Load() != 0 || removes.Load() != 0 || persisted.Load() != 0 || u.Done().Begun() || u.Busy() {
			t.Errorf("preconfigured refusal leaked effects: requests=%d reads=%d removes=%d persist=%d begun=%v busy=%v", requests.Load(), reads.Load(), removes.Load(), persisted.Load(), u.Done().Begun(), u.Busy())
		}
		if _, err := update.LoadStage(dir); err != nil {
			t.Errorf("preconfigured refusal changed seeded stage: %v", err)
		}
	})

	t.Run("persistence", func(t *testing.T) {
		var restrictedWrites atomic.Int32
		restricted := New(test.NewApp(), t.TempDir(), launch.Policy{}, func(_ string) { restrictedWrites.Add(1) })
		restricted.RestoreLastCheckDay("2026-09-27")
		if restricted.LastCheckDay() != "2026-09-27" || restrictedWrites.Load() != 0 {
			t.Fatal("restricted restore must seed memory without persistence")
		}
		restricted.SetLastCheckDay("2026-09-28")
		if restricted.LastCheckDay() != "2026-09-27" || restrictedWrites.Load() != 0 {
			t.Fatal("restricted set changed restored day or persisted it")
		}

		asset := updaterAssetName(t)
		srv := updaterReleaseServer(t, "v0.2.5", asset, nil, "", nil)
		var saved []string
		ordinary := ordinaryUpdater(t, test.NewApp(), t.TempDir(), func(day string) { saved = append(saved, day) })
		ordinary.RestoreLastCheckDay("2026-09-27")
		if len(saved) != 0 {
			t.Fatal("ordinary restore persisted")
		}
		ordinary.SetClient(updaterClient(ordinary, srv, fakeVerifier{}, time.Date(2026, 9, 28, 10, 0, 0, 0, time.Local)))
		ordinary.StartManual(context.Background(), func() bool { return false }, "v0.2.5", Events{})
		waitUpdater(t, ordinary)
		if ordinary.LastCheckDay() != "2026-09-28" || len(saved) != 1 || saved[0] != "2026-09-28" {
			t.Errorf("successful ordinary check: day=%q saved=%v", ordinary.LastCheckDay(), saved)
		}
	})
}

func updaterTestPolicy(t *testing.T, trial launch.Purpose, storeManaged, missing bool) launch.Policy {
	t.Helper()
	if missing {
		return launch.Policy{}
	}
	opts := launch.Options{}
	switch trial {
	case launch.Ordinary:
	case launch.ExplorerTrial:
		opts.ExplorerTrial = t.TempDir()
	case launch.LocationMapTrial:
		opts.LocationMapTrial = t.TempDir()
	default:
		t.Fatalf("unknown test launch purpose: %d", trial)
	}
	policy, err := launch.NewPolicy(opts, "io.picfetch", storeManaged)
	if err != nil {
		t.Fatal(err)
	}
	return policy
}
