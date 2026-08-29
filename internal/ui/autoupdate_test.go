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
	"fmt"
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
	"github.com/frathe/picfetch/internal/ui/settingswin"
	"github.com/frathe/picfetch/internal/update"
)

type fakeUpdateVerifier struct {
	err   error
	calls int
}

func (f *fakeUpdateVerifier) Verify(_ context.Context, _, _ []byte, _ update.VerifyPolicy) error {
	f.calls++
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

func saveVerifiedUpdateStage(t *testing.T, v *viewer, version, notes string) update.Stage {
	t.Helper()
	asset := updateAssetName(t)
	archive := nativeUpdateArchive(t, asset)
	sum := sha256.Sum256(archive)
	srv := serveUpdateAPI(t, version, notes, asset, archive, hex.EncodeToString(sum[:]))
	client := attachUpdateClient(t, v, srv, fixedNow("2026-08-30"), nil)
	st, err := client.Download(context.Background(), update.Release{
		Version:     version,
		Notes:       notes,
		AssetName:   asset,
		AssetURL:    srv.URL + "/" + asset,
		AssetDigest: hex.EncodeToString(sum[:]),
	})
	if err != nil {
		t.Fatal(err)
	}
	return st
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

func TestManualUpdateCheck_BypassesSettingAndDailyGate(t *testing.T) {
	v := newTestViewer(t)
	asset := updateAssetName(t)
	srv := serveUpdateAPI(t, "v0.2.5", "notes", asset, nil, strings.Repeat("0", 64))
	attachUpdateClient(t, v, srv, fixedNow("2026-08-30"), nil)
	v.updater.SetCurrentVersion("0.2.5")
	v.SetLastUpdateCheckDay("2026-08-30")
	if v.CheckForUpdates() {
		t.Fatal("test requires automatic updates to remain off")
	}

	var current, failed int
	v.CheckForUpdatesNow(settingswin.UpdateCallbacks{
		Current: func() { current++ },
		Failed:  func(error) { failed++ },
	})
	waitFor(t, "manual update check", v.updater.Done())

	if current != 1 || failed != 0 {
		t.Errorf("current=%d failed=%d, want current=1 failed=0", current, failed)
	}
	if got := v.LastUpdateCheckDay(); got != "2026-08-30" {
		t.Errorf("LastUpdateCheckDay() = %q, want successful manual check day", got)
	}
}

func TestManualUpdateCheck_PreparationFailureArrivesThroughCallback(t *testing.T) {
	v := newTestViewer(t)
	updateAssetName(t)
	v.updater.SetCurrentVersion("0.2.5")
	wantErr := errors.New("verifier unavailable")
	v.updater.SetVerifierFactory(func() (update.Verifier, error) { return nil, wantErr })

	var got error
	v.CheckForUpdatesNow(settingswin.UpdateCallbacks{Failed: func(err error) { got = err }})
	waitFor(t, "manual update preparation", v.updater.Done())

	if !errors.Is(got, wantErr) {
		t.Errorf("Failed error = %v, want %v", got, wantErr)
	}
	if v.updater.Client() != nil {
		t.Error("failed manual preparation must leave Client nil")
	}
}

func TestAutomaticCheckDoesNotBlockOrSupersedeManualPreparation(t *testing.T) {
	v := newTestViewer(t)
	updateAssetName(t)
	v.updater.SetCurrentVersion("0.2.5")
	v.settings.checkForUpdates = true
	started := make(chan struct{})
	release := make(chan struct{})
	wantErr := errors.New("manual verifier unavailable")
	v.updater.SetVerifierFactory(func() (update.Verifier, error) {
		close(started)
		<-release
		return nil, wantErr
	})

	var got error
	v.CheckForUpdatesNow(settingswin.UpdateCallbacks{Failed: func(err error) { got = err }})
	select {
	case <-started:
	case <-time.After(testTimeout):
		t.Fatal("manual verifier preparation did not start")
	}
	wantRevision := v.updateOp.currentRevision()

	returned := make(chan struct{})
	go func() {
		v.maybeStartUpdateCheck()
		close(returned)
	}()
	select {
	case <-returned:
	case <-time.After(testTimeout):
		t.Fatal("automatic check blocked behind manual verifier preparation")
	}
	if gotRevision := v.updateOp.currentRevision(); gotRevision != wantRevision {
		t.Errorf("automatic check superseded manual revision: got %d, want %d", gotRevision, wantRevision)
	}

	close(release)
	waitFor(t, "manual verifier preparation", v.updater.Done())
	if !errors.Is(got, wantErr) {
		t.Errorf("manual Failed error = %v, want %v", got, wantErr)
	}
}

func TestManualUpdateCheck_ReadyCallbackUsesVersionString(t *testing.T) {
	v := newTestViewer(t)
	asset := updateAssetName(t)
	archive := nativeUpdateArchive(t, asset)
	sum := sha256.Sum256(archive)
	srv := serveUpdateAPI(t, "v0.2.6", "notes", asset, archive, hex.EncodeToString(sum[:]))
	attachUpdateClient(t, v, srv, fixedNow("2026-08-30"), nil)
	v.updater.SetCurrentVersion("0.2.5")

	var readyVersion string
	var progressCalls int
	v.CheckForUpdatesNow(settingswin.UpdateCallbacks{
		Progress: func(downloaded, total int64) {
			progressCalls++
			if downloaded < 0 {
				t.Errorf("negative download progress %d/%d", downloaded, total)
			}
		},
		Ready:  func(version string) { readyVersion = version },
		Failed: func(err error) { t.Errorf("unexpected update failure: %v", err) },
	})
	waitFor(t, "manual update download", v.updater.Done())

	if readyVersion != "v0.2.6" {
		t.Errorf("Ready version = %q, want v0.2.6", readyVersion)
	}
	if progressCalls == 0 {
		t.Error("manual update emitted no progress callbacks")
	}
}

// TestManualUpdateFlow_CallbacksReadyThenPerformRequestsRelaunch covers the
// viewer integration boundary: a verified fake GitHub release drives the
// Settings callback vocabulary in order, then the ready stage can request the
// existing shutdown-time apply path with explicit relaunch intent. The
// per-viewer quit and update.Apply seams keep the test from quitting,
// replacing the test executable, or starting a process.
func TestManualUpdateFlow_CallbacksReadyThenPerformRequestsRelaunch(t *testing.T) {
	v := newTestViewer(t)
	asset := updateAssetName(t)
	archive := nativeUpdateArchive(t, asset)
	sum := sha256.Sum256(archive)
	srv := serveUpdateAPI(t, "v0.2.6", "manual-flow notes", asset, archive, hex.EncodeToString(sum[:]))
	verifier := &fakeUpdateVerifier{}
	v.updater.SetClient(update.NewClient(update.Config{
		BaseURL:  srv.URL,
		HTTP:     srv.Client(),
		Now:      fixedNow("2026-08-30"),
		Verify:   verifier,
		StageDir: v.updater.Dir(),
		GOOS:     runtime.GOOS,
		GOARCH:   runtime.GOARCH,
	}))
	v.updater.SetCurrentVersion("0.2.5")

	var events []string
	var callbackErr error
	v.CheckForUpdatesNow(settingswin.UpdateCallbacks{
		Downloading: func(version string) { events = append(events, "downloading:"+version) },
		Progress: func(downloaded, total int64) {
			events = append(events, "progress")
			if downloaded < 0 || total <= 0 || downloaded > total {
				callbackErr = fmt.Errorf("invalid progress %d/%d", downloaded, total)
			}
		},
		Ready:  func(version string) { events = append(events, "ready:"+version) },
		Failed: func(err error) { callbackErr = err },
	})
	waitFor(t, "manual update flow", v.updater.Done())

	if callbackErr != nil {
		t.Fatalf("manual update callback failure: %v", callbackErr)
	}
	if verifier.calls != 1 {
		t.Fatalf("release-attestation verifier calls = %d, want 1", verifier.calls)
	}
	if len(events) < 3 || events[0] != "downloading:v0.2.6" || events[len(events)-1] != "ready:v0.2.6" {
		t.Fatalf("manual callback sequence = %v, want Downloading -> Progress -> Ready", events)
	}
	for _, event := range events[1 : len(events)-1] {
		if event != "progress" {
			t.Fatalf("manual callback sequence = %v, want only progress between Downloading and Ready", events)
		}
	}

	quitCalls := 0
	v.quit = func() { quitCalls++ }
	if err := v.PerformUpdate(); err != nil {
		t.Fatal(err)
	}
	if quitCalls != 1 {
		t.Fatalf("quit calls = %d, want 1", quitCalls)
	}

	var applied update.ApplyOptions
	originalApply := update.Apply
	update.Apply = func(_ update.Stage, _ string, options update.ApplyOptions) error {
		applied = options
		return nil
	}
	t.Cleanup(func() { update.Apply = originalApply })
	v.updater.ApplyStagedUpdate()
	if !applied.Relaunch {
		t.Error("PerformUpdate did not record relaunch intent for shutdown apply")
	}
}

func TestManualUpdateCheck_CancelledRequestEmitsNoTerminalCallback(t *testing.T) {
	v := newTestViewer(t)
	asset := updateAssetName(t)
	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/frathe/picfetch/releases/latest" {
			http.NotFound(w, r)
			return
		}
		close(started)
		<-release
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name": "v0.2.5",
			"assets":   []any{map[string]any{"name": asset, "browser_download_url": srv.URL + "/" + asset}},
		})
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(unblock)
	attachUpdateClient(t, v, srv, fixedNow("2026-08-30"), nil)
	v.updater.SetCurrentVersion("0.2.5")

	var terminal int
	v.CheckForUpdatesNow(settingswin.UpdateCallbacks{
		Current: func() { terminal++ },
		Ready:   func(string) { terminal++ },
		Failed:  func(error) { terminal++ },
	})
	select {
	case <-started:
	case <-time.After(testTimeout):
		t.Fatal("manual update check did not reach HTTP")
	}
	v.updateOp.invalidate()
	unblock()
	waitFor(t, "cancelled manual update check", v.updater.Done())

	if terminal != 0 {
		t.Errorf("cancelled request terminal callbacks = %d, want 0", terminal)
	}
}

func TestCurrentUpdateCallback_DropsEventSupersededWhileQueued(t *testing.T) {
	v := newTestViewer(t)
	token := v.updateOp.begin()
	called := false
	queued := currentUpdateCallback(token, func() { called = true })

	v.updateOp.invalidate()
	queued()

	if called {
		t.Error("queued update callback ran after its token was superseded")
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
	saveVerifiedUpdateStage(t, v, "v0.2.6", "hello notes")

	var gotStage update.Stage
	var gotDest string
	var gotOptions update.ApplyOptions
	orig := update.Apply
	update.Apply = func(stage update.Stage, dest string, options update.ApplyOptions) error {
		gotStage = stage
		gotDest = dest
		gotOptions = options
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
	if gotOptions.Relaunch {
		t.Error("normal shutdown apply unexpectedly requested relaunch")
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
	update.Apply = func(update.Stage, string, update.ApplyOptions) error {
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

func TestApplyStagedUpdate_UnverifiedStageNeverApplies(t *testing.T) {
	v := newTestViewer(t)
	v.updater.SetCurrentVersion("0.2.5")
	bin := filepath.Join(v.updater.Dir(), "forged-stage")
	if err := os.MkdirAll(v.updater.Dir(), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("forged"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := update.SaveStage(v.updater.Dir(), update.Stage{Version: "v0.2.6", BinaryPath: bin}); err != nil {
		t.Fatal(err)
	}

	orig := update.Apply
	update.Apply = func(update.Stage, string, update.ApplyOptions) error {
		t.Error("Apply must not run for a stage without verified provenance")
		return nil
	}
	t.Cleanup(func() { update.Apply = orig })

	v.updater.ApplyStagedUpdate()
	if _, err := update.LoadStage(v.updater.Dir()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("unverified stage should be removed, got %v", err)
	}
	if notes, err := autoupdate.LoadWhatsNew(v.app); err != nil || notes != nil {
		t.Fatalf("unverified stage wrote What's New: notes=%+v err=%v", notes, err)
	}
}

func TestPerformUpdate_RecordsRelaunchThenQuits(t *testing.T) {
	v := newTestViewer(t)
	v.updater.SetCurrentVersion("0.2.5")
	saveVerifiedUpdateStage(t, v, "v0.2.6", "restart notes")

	quitCalls := 0
	v.quit = func() { quitCalls++ }
	if err := v.PerformUpdate(); err != nil {
		t.Fatal(err)
	}
	if quitCalls != 1 {
		t.Fatalf("quit calls = %d, want 1", quitCalls)
	}

	var gotOptions update.ApplyOptions
	orig := update.Apply
	update.Apply = func(_ update.Stage, _ string, options update.ApplyOptions) error {
		gotOptions = options
		return nil
	}
	t.Cleanup(func() { update.Apply = orig })
	v.updater.ApplyStagedUpdate()
	if !gotOptions.Relaunch {
		t.Error("PerformUpdate intent did not propagate to apply")
	}
}

func TestPerformUpdate_InvalidMissingOrSameStageDoesNotQuit(t *testing.T) {
	tests := []struct {
		name  string
		stage *update.Stage
	}{
		{name: "missing metadata"},
		{name: "missing binary", stage: &update.Stage{Version: "v0.2.6", BinaryPath: "/missing/picfetch-staged"}},
		{name: "unverified binary", stage: &update.Stage{Version: "v0.2.6"}},
		{name: "invalid version", stage: &update.Stage{Version: "not-a-version"}},
		{name: "same version", stage: &update.Stage{Version: "v0.2.5"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := newTestViewer(t)
			v.updater.SetCurrentVersion("0.2.5")
			if tc.stage != nil {
				stage := *tc.stage
				if stage.BinaryPath == "" {
					stage.BinaryPath = filepath.Join(v.updater.Dir(), "picfetch-staged")
					if err := os.MkdirAll(v.updater.Dir(), 0o700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(stage.BinaryPath, []byte("staged"), 0o755); err != nil {
						t.Fatal(err)
					}
				}
				if err := update.SaveStage(v.updater.Dir(), stage); err != nil {
					t.Fatal(err)
				}
			}

			quitCalls := 0
			v.quit = func() { quitCalls++ }
			if err := v.PerformUpdate(); err == nil {
				t.Fatal("PerformUpdate succeeded for an unusable stage")
			}
			if quitCalls != 0 {
				t.Errorf("quit calls = %d, want 0", quitCalls)
			}
		})
	}
}

func TestApplyStagedUpdate_RelaunchFailureRetainsNotesAndStage(t *testing.T) {
	v := newTestViewer(t)
	v.updater.SetCurrentVersion("0.2.5")
	saveVerifiedUpdateStage(t, v, "v0.2.6", "recoverable notes")
	v.quit = func() {}
	if err := v.PerformUpdate(); err != nil {
		t.Fatal(err)
	}

	wantErr := errors.New("relaunch failed")
	orig := update.Apply
	update.Apply = func(_ update.Stage, _ string, options update.ApplyOptions) error {
		if !options.Relaunch {
			t.Error("apply did not receive relaunch intent")
		}
		return wantErr
	}
	t.Cleanup(func() { update.Apply = orig })
	v.updater.ApplyStagedUpdate()

	wn, err := autoupdate.LoadWhatsNew(v.app)
	if err != nil {
		t.Fatal(err)
	}
	if wn == nil || wn.Version != "v0.2.6" || wn.Body != "recoverable notes" {
		t.Errorf("whatsNew after relaunch failure = %+v", wn)
	}
	if _, err := update.LoadStage(v.updater.Dir()); err != nil {
		t.Fatalf("stage must remain recoverable after relaunch failure: %v", err)
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
