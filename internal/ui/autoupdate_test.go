package ui

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/frathe/picfetch/internal/preferences"
	"github.com/frathe/picfetch/internal/ui/autoupdate"
	"github.com/frathe/picfetch/internal/update"
)

type fakeUpdateVerifier struct {
	err error
}

func (f *fakeUpdateVerifier) Verify(_ context.Context, _, _ []byte, _ update.VerifyPolicy) error {
	return f.err
}

type errorDoer struct {
	t *testing.T
}

func (d errorDoer) Do(*http.Request) (*http.Response, error) {
	d.t.Error("HTTP Do must not be called")
	return nil, errors.New("unexpected HTTP")
}

func fixedNow(day string) func() time.Time {
	t, err := time.ParseInLocation("2006-01-02", day, time.Local)
	if err != nil {
		panic(err)
	}
	return func() time.Time { return t }
}

func updateAssetName(t *testing.T) string {
	t.Helper()
	name, ok := update.AssetName(runtime.GOOS, runtime.GOARCH)
	if !ok {
		t.Skipf("no update asset for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return name
}

func nativeUpdateArchive(t *testing.T, assetName string) []byte {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, assetName)
	var files map[string][]byte
	switch {
	case strings.HasSuffix(assetName, ".tar.gz"):
		bin := strings.TrimSuffix(assetName, ".tar.gz")
		files = map[string][]byte{bin: []byte("elf")}
		if err := writeTarGzFile(path, files); err != nil {
			t.Fatal(err)
		}
	case strings.Contains(assetName, "windows"):
		files = map[string][]byte{"picfetch.exe": []byte("exe")}
		if err := writeZipFile(path, files); err != nil {
			t.Fatal(err)
		}
	case strings.Contains(assetName, "macos"):
		files = map[string][]byte{
			"PicFetch.app/Contents/MacOS/picfetch": []byte("macho"),
			"PicFetch.app/Contents/Info.plist":     []byte("plist"),
		}
		if err := writeZipFile(path, files); err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatalf("unsupported asset %q", assetName)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func writeTarGzFile(path string, files map[string][]byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	for name, body := range files {
		hdr := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(body))}
		if err := tw.WriteHeader(hdr); err != nil {
			_ = f.Close()
			return err
		}
		if _, err := tw.Write(body); err != nil {
			_ = f.Close()
			return err
		}
	}
	if err := tw.Close(); err != nil {
		_ = f.Close()
		return err
	}
	if err := gw.Close(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func writeZipFile(path string, files map[string][]byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(f)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			_ = f.Close()
			return err
		}
		if _, err := w.Write(body); err != nil {
			_ = f.Close()
			return err
		}
	}
	if err := zw.Close(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func serveUpdateAPI(t *testing.T, tag, body, assetName string, archive []byte, digest string) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/repos/frathe/picfetch/releases/latest":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tag_name":   tag,
				"body":       body,
				"draft":      false,
				"prerelease": false,
				"assets": []any{
					map[string]any{
						"name":                 assetName,
						"browser_download_url": srv.URL + "/" + assetName,
						"digest":               "sha256:" + digest,
					},
				},
			})
		case r.URL.Path == "/"+assetName:
			_, _ = w.Write(archive)
		case strings.Contains(r.URL.Path, "/attestations/"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"attestations":[{"bundle":{"mediaType":"test-bundle"}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func attachUpdateClient(t *testing.T, v *viewer, srv *httptest.Server, now func() time.Time, doer update.Doer) *update.Client {
	t.Helper()
	if doer == nil {
		doer = srv.Client()
	}
	c := update.NewClient(update.Config{
		BaseURL:  srv.URL,
		HTTP:     doer,
		Now:      now,
		Verify:   &fakeUpdateVerifier{},
		StageDir: v.updater.Dir(),
		GOOS:     runtime.GOOS,
		GOARCH:   runtime.GOARCH,
	})
	v.updater.SetClient(c)
	return c
}

func TestUpdateCheck_SettingOffNeverCallsHTTP(t *testing.T) {
	v := newTestViewer(t)
	v.updater.SetCurrentVersion("0.2.5")
	v.updater.SetClient(update.NewClient(update.Config{
		HTTP:     errorDoer{t: t},
		Now:      fixedNow("2026-08-26"),
		Verify:   &fakeUpdateVerifier{},
		StageDir: v.updater.Dir(),
		GOOS:     runtime.GOOS,
		GOARCH:   runtime.GOARCH,
	}))

	// Setting stays off (newTestViewer default). maybeStart must not run Check.
	v.maybeStartUpdateCheck()
	waitFor(t, "update check", v.updater.Done())

	if _, err := os.Stat(filepath.Join(v.updater.Dir(), "stage.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("want no stage.json, stat err %v", err)
	}
}

func TestUpdateCheck_VerifierFailurePreservesLifecycle(t *testing.T) {
	v := newTestViewer(t)
	updateAssetName(t)
	v.updater.SetCurrentVersion("0.2.5")
	v.settings.checkForUpdates = true
	wantErr := errors.New("verifier unavailable")
	calls := 0
	v.updater.SetVerifierFactory(func() (update.Verifier, error) {
		calls++
		return nil, wantErr
	})
	prior := v.updateOp.begin()
	wantRevision := v.updateOp.currentRevision()

	v.maybeStartUpdateCheck()

	if calls != 1 {
		t.Errorf("verifier factory calls = %d, want 1", calls)
	}
	if got := v.updateOp.currentRevision(); got != wantRevision {
		t.Errorf("update lifecycle revision = %d, want unchanged %d", got, wantRevision)
	}
	if !prior.current() {
		t.Error("prior update lifecycle token is no longer current")
	}
	select {
	case <-prior.context().Done():
		t.Error("prior update lifecycle context was cancelled")
	default:
	}
	if v.updater.Client() != nil {
		t.Error("Client() after verifier failure must remain nil")
	}
	if v.updater.Done().Begun() {
		t.Error("Done() after verifier failure must not report Begun")
	}
}

func TestUpdateCheck_SameVersionRecordsDayNoStage(t *testing.T) {
	v := newTestViewer(t)
	asset := updateAssetName(t)
	digest := strings.Repeat("a", 64)
	srv := serveUpdateAPI(t, "v0.2.5", "notes", asset, nil, digest)
	attachUpdateClient(t, v, srv, fixedNow("2026-08-26"), nil)
	v.updater.SetCurrentVersion("0.2.5")

	v.SetCheckForUpdates(true)
	waitFor(t, "update check", v.updater.Done())

	if got := v.LastUpdateCheckDay(); got != "2026-08-26" {
		t.Errorf("LastUpdateCheckDay = %q, want 2026-08-26", got)
	}
	if got := preferences.Load(v.app).LastUpdateCheckDay; got != "2026-08-26" {
		t.Errorf("persisted LastUpdateCheckDay = %q, want 2026-08-26", got)
	}
	if _, err := os.Stat(filepath.Join(v.updater.Dir(), "stage.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("want no stage.json when already current, stat err %v", err)
	}
}

func TestUpdateCheck_NewerReleaseStagesBinary(t *testing.T) {
	v := newTestViewer(t)
	asset := updateAssetName(t)
	archive := nativeUpdateArchive(t, asset)
	sum := sha256.Sum256(archive)
	digest := hex.EncodeToString(sum[:])
	srv := serveUpdateAPI(t, "v0.2.6", "## Fixes\n\n- toast", asset, archive, digest)
	attachUpdateClient(t, v, srv, fixedNow("2026-08-26"), nil)
	v.updater.SetCurrentVersion("0.2.5")

	v.SetCheckForUpdates(true)
	waitFor(t, "update check", v.updater.Done())

	if _, err := os.Stat(filepath.Join(v.updater.Dir(), "stage.json")); err != nil {
		t.Fatalf("stage.json: %v", err)
	}
	st, err := update.LoadStage(v.updater.Dir())
	if err != nil {
		t.Fatal(err)
	}
	if st.Version != "v0.2.6" {
		t.Errorf("staged Version = %q, want v0.2.6", st.Version)
	}
	if st.Notes != "## Fixes\n\n- toast" {
		t.Errorf("staged Notes = %q", st.Notes)
	}
}

func TestUpdateCheck_TurningOffInvalidatesInFlight(t *testing.T) {
	v := newTestViewer(t)
	asset := updateAssetName(t)
	started := make(chan struct{})
	block := make(chan struct{})
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/frathe/picfetch/releases/latest" {
			close(started)
			<-block
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tag_name":   "v0.2.6",
				"body":       "notes",
				"draft":      false,
				"prerelease": false,
				"assets": []any{
					map[string]any{
						"name":                 asset,
						"browser_download_url": srv.URL + "/" + asset,
						"digest":               "sha256:" + strings.Repeat("b", 64),
					},
				},
			})
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(block) })

	attachUpdateClient(t, v, srv, fixedNow("2026-08-26"), nil)
	v.updater.SetCurrentVersion("0.2.5")
	v.SetCheckForUpdates(true)

	select {
	case <-started:
	case <-time.After(testTimeout):
		t.Fatal("timed out waiting for check HTTP")
	}

	v.SetCheckForUpdates(false)
	waitFor(t, "update check", v.updater.Done())

	if _, err := os.Stat(filepath.Join(v.updater.Dir(), "stage.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled check must not leave stage.json, stat err %v", err)
	}
}

func TestUpdateCheck_TurningOffKeepsCompletedStage(t *testing.T) {
	v := newTestViewer(t)
	asset := updateAssetName(t)
	archive := nativeUpdateArchive(t, asset)
	sum := sha256.Sum256(archive)
	digest := hex.EncodeToString(sum[:])
	srv := serveUpdateAPI(t, "v0.2.6", "notes", asset, archive, digest)
	attachUpdateClient(t, v, srv, fixedNow("2026-08-26"), nil)
	v.updater.SetCurrentVersion("0.2.5")

	v.SetCheckForUpdates(true)
	waitFor(t, "update check", v.updater.Done())
	if _, err := os.Stat(filepath.Join(v.updater.Dir(), "stage.json")); err != nil {
		t.Fatalf("stage.json before turn-off: %v", err)
	}

	v.SetCheckForUpdates(false)

	if _, err := os.Stat(filepath.Join(v.updater.Dir(), "stage.json")); err != nil {
		t.Fatalf("completed stage must survive SetCheckForUpdates(false): %v", err)
	}
}

func TestUpdateCheck_RestoredTodayDoesNotCheck(t *testing.T) {
	v := newTestViewer(t)
	// Field-only restore (registerFeatures): day before enabling the flag,
	// never SetCheckForUpdates — that would start a check.
	v.SetLastUpdateCheckDay("2026-08-26")
	v.settings.checkForUpdates = true
	v.updater.SetCurrentVersion("0.2.5")
	v.updater.SetClient(update.NewClient(update.Config{
		HTTP:     errorDoer{t: t},
		Now:      fixedNow("2026-08-26"),
		Verify:   &fakeUpdateVerifier{},
		StageDir: v.updater.Dir(),
		GOOS:     runtime.GOOS,
		GOARCH:   runtime.GOARCH,
	}))

	v.maybeStartUpdateCheck()
	waitFor(t, "update check", v.updater.Done())
}

func TestUpdateCheck_RemovesStaleStage(t *testing.T) {
	v := newTestViewer(t)
	v.updater.SetCurrentVersion("0.2.6")
	bin := filepath.Join(v.updater.Dir(), "picfetch-staged")
	if err := os.MkdirAll(v.updater.Dir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := update.SaveStage(v.updater.Dir(), update.Stage{Version: "v0.2.5", Notes: "stale", BinaryPath: bin}); err != nil {
		t.Fatal(err)
	}

	v.SetLastUpdateCheckDay("2026-08-26")
	v.settings.checkForUpdates = true
	v.updater.SetClient(update.NewClient(update.Config{
		HTTP:     errorDoer{t: t},
		Now:      fixedNow("2026-08-26"),
		Verify:   &fakeUpdateVerifier{},
		StageDir: v.updater.Dir(),
		GOOS:     runtime.GOOS,
		GOARCH:   runtime.GOARCH,
	}))

	v.maybeStartUpdateCheck()
	waitFor(t, "update check", v.updater.Done())

	if _, err := os.Stat(filepath.Join(v.updater.Dir(), "stage.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stale stage must be removed, stat err %v", err)
	}
}

func TestDrain_WaitsUpdateDone(t *testing.T) {
	v := newTestViewer(t)
	asset := updateAssetName(t)
	archive := nativeUpdateArchive(t, asset)
	sum := sha256.Sum256(archive)
	digest := hex.EncodeToString(sum[:])
	srv := serveUpdateAPI(t, "v0.2.6", "notes", asset, archive, digest)
	attachUpdateClient(t, v, srv, fixedNow("2026-08-26"), nil)
	v.updater.SetCurrentVersion("0.2.5")
	v.SetCheckForUpdates(true)

	// drain (registered by newTestUI cleanup) must wait v.updater.Done().
	// Waiting here mirrors drain's waitFor on a never-begun-safe Signal.
	waitFor(t, "update check", v.updater.Done())
	if _, err := os.Stat(filepath.Join(v.updater.Dir(), "stage.json")); err != nil {
		t.Fatalf("stage.json after drainable update: %v", err)
	}
}

func TestApplyStagedUpdate_SavesNotesAndCallsApply(t *testing.T) {
	v := newTestViewer(t)
	v.updater.SetCurrentVersion("0.2.5")
	bin := filepath.Join(v.updater.Dir(), "picfetch-staged")
	if err := os.MkdirAll(v.updater.Dir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}
	st := update.Stage{Version: "v0.2.6", Notes: "hello notes", BinaryPath: bin}
	if err := update.SaveStage(v.updater.Dir(), st); err != nil {
		t.Fatal(err)
	}

	var gotStage update.Stage
	var gotDest string
	orig := update.Apply
	update.Apply = func(stage update.Stage, dest string) error {
		gotStage = stage
		gotDest = dest
		return nil
	}
	t.Cleanup(func() { update.Apply = orig })

	v.updater.ApplyStagedUpdate()

	if gotStage.Version != "v0.2.6" || gotStage.Notes != "hello notes" {
		t.Errorf("Apply stage = %+v", gotStage)
	}
	if gotDest == "" {
		t.Error("Apply dest empty")
	}
	wn, err := autoupdate.LoadWhatsNew(v.app)
	if err != nil {
		t.Fatal(err)
	}
	if wn == nil || wn.Version != "v0.2.6" || wn.Body != "hello notes" {
		t.Errorf("whatsNew = %+v", wn)
	}
	if runtime.GOOS != "windows" {
		if _, err := os.Stat(filepath.Join(v.updater.Dir(), "stage.json")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("unix should RemoveStage after Apply, stat err %v", err)
		}
	}
}

func TestApplyStagedUpdate_SameVersionRemovesWithoutApply(t *testing.T) {
	v := newTestViewer(t)
	v.updater.SetCurrentVersion("0.2.6")
	bin := filepath.Join(v.updater.Dir(), "picfetch-staged")
	if err := os.MkdirAll(v.updater.Dir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("same"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := update.SaveStage(v.updater.Dir(), update.Stage{Version: "v0.2.6", Notes: "old", BinaryPath: bin}); err != nil {
		t.Fatal(err)
	}

	orig := update.Apply
	update.Apply = func(update.Stage, string) error {
		t.Error("Apply must not run when staged version is not newer")
		return nil
	}
	t.Cleanup(func() { update.Apply = orig })

	v.updater.ApplyStagedUpdate()

	if _, err := os.Stat(filepath.Join(v.updater.Dir(), "stage.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("want stage removed, stat err %v", err)
	}
	wn, err := autoupdate.LoadWhatsNew(v.app)
	if err != nil {
		t.Fatal(err)
	}
	if wn != nil {
		t.Errorf("whatsNew = %+v, want nil", wn)
	}
}

func TestWhatsNewCache_RoundTripAndClear(t *testing.T) {
	v := newTestViewer(t)
	if err := autoupdate.SaveWhatsNew(v.app, "v0.2.6", "body text"); err != nil {
		t.Fatal(err)
	}
	got, err := autoupdate.LoadWhatsNew(v.app)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.Version != "v0.2.6" || got.Body != "body text" {
		t.Fatalf("loadWhatsNew = %+v", got)
	}
	if err := autoupdate.ClearWhatsNew(v.app); err != nil {
		t.Fatal(err)
	}
	got, err = autoupdate.LoadWhatsNew(v.app)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("after clear, loadWhatsNew = %+v, want nil", got)
	}
}

func TestMaybeShowWhatsNew_ShowsAndClearsCache(t *testing.T) {
	v := newTestViewer(t)
	if err := autoupdate.SaveWhatsNew(v.app, "v0.2.6", "# notes"); err != nil {
		t.Fatal(err)
	}
	v.updater.SetCurrentVersion("0.2.6")

	v.maybeShowWhatsNew()
	if !v.help.WhatsNewOpen() {
		t.Fatal("WhatsNewOpen should be true after matching maybeShowWhatsNew")
	}
	wn, err := autoupdate.LoadWhatsNew(v.app)
	if err != nil {
		t.Fatal(err)
	}
	if wn != nil {
		t.Errorf("cache after show = %+v, want nil", wn)
	}

	// Cleared cache: a second call must not build another window. The first
	// window may still be open; Singleton.Open stays true either way, so
	// assert via cache staying empty and a fresh viewer with no cache.
	v.maybeShowWhatsNew()
	wn, err = autoupdate.LoadWhatsNew(v.app)
	if err != nil {
		t.Fatal(err)
	}
	if wn != nil {
		t.Errorf("second call must not rewrite cache, got %+v", wn)
	}
}

func TestMaybeShowWhatsNew_EmptyCacheDoesNotShow(t *testing.T) {
	v := newTestViewer(t)
	v.updater.SetCurrentVersion("0.2.6")
	v.maybeShowWhatsNew()
	if v.help.WhatsNewOpen() {
		t.Error("maybeShowWhatsNew must not open when cache is empty")
	}
}

func TestMaybeShowWhatsNew_VersionMismatchDoesNotShow(t *testing.T) {
	v := newTestViewer(t)
	if err := autoupdate.SaveWhatsNew(v.app, "v0.2.6", "# notes"); err != nil {
		t.Fatal(err)
	}
	v.updater.SetCurrentVersion("0.2.5")
	v.maybeShowWhatsNew()
	if v.help.WhatsNewOpen() {
		t.Error("version mismatch must not show What's New")
	}
	wn, err := autoupdate.LoadWhatsNew(v.app)
	if err != nil {
		t.Fatal(err)
	}
	if wn == nil || wn.Version != "v0.2.6" {
		t.Errorf("mismatch must leave cache intact, got %+v", wn)
	}
}

func TestStartViewerRuntime_DefaultOffDoesNotAssignClient(t *testing.T) {
	v, win, _ := newTestUI(t)
	if v.updater.Client() != nil {
		t.Fatal("newTestUI must not assign v.updater's client")
	}
	startViewerRuntime(v, win, t.TempDir())
	t.Cleanup(v.stopWinPosPoll)
	if v.updater.Client() != nil {
		t.Fatal("startViewerRuntime with CheckForUpdates=false must not construct a Client")
	}
}

func TestLastUpdateCheckDay_ConcurrentWithCurrentPreferences(t *testing.T) {
	v := newTestViewer(t)
	const n = 200
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for range n {
			v.SetLastUpdateCheckDay("2026-08-26")
		}
	}()
	go func() {
		defer wg.Done()
		for range n {
			_ = v.LastUpdateCheckDay()
			_ = v.currentPreferences()
		}
	}()
	wg.Wait()
}
