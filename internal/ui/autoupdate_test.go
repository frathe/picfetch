package ui

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"

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

func TestUpdateCheck_StoreManagedBuildNeverTouchesGitHubStage(t *testing.T) {
	v := newTestViewer(t)
	v.storeManaged = true
	v.settings.checkForUpdates = true
	v.updater.SetCurrentVersion("0.2.6")
	calls := 0
	v.updater.SetVerifierFactory(func() (update.Verifier, error) {
		calls++
		return &fakeUpdateVerifier{}, nil
	})

	bin := filepath.Join(v.updater.Dir(), "picfetch-staged")
	if err := os.WriteFile(bin, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := update.SaveStage(v.updater.Dir(), update.Stage{Version: "v0.2.5", BinaryPath: bin}); err != nil {
		t.Fatal(err)
	}

	v.maybeStartUpdateCheck()

	if calls != 0 {
		t.Errorf("verifier factory calls = %d, want 0", calls)
	}
	if v.updater.Done().Begun() || v.updateOp.currentRevision() != 0 {
		t.Error("Store-managed automatic update began background work")
	}
	if _, err := os.Stat(filepath.Join(v.updater.Dir(), "stage.json")); err != nil {
		t.Fatalf("Store-managed check touched an existing GitHub stage: %v", err)
	}
}

func TestMicrosoftStoreUpdateActionsAreRefused(t *testing.T) {
	v := newTestViewer(t)
	v.storeManaged = true
	v.settings.checkForUpdates = true

	v.SetCheckForUpdates(true)
	if v.CheckForUpdates() || v.currentPreferences().CheckForUpdates {
		t.Error("Store-managed build retained the GitHub update preference")
	}

	var manualErr error
	v.CheckForUpdatesNow(settingswin.UpdateCallbacks{Failed: func(err error) { manualErr = err }})
	if manualErr == nil || manualErr.Error() != "updates are managed by Microsoft Store" {
		t.Fatalf("manual update error = %v", manualErr)
	}
	if err := v.PerformUpdate(); err == nil || err.Error() != "updates are managed by Microsoft Store" {
		t.Fatalf("PerformUpdate error = %v", err)
	}
	if v.updater.Done().Begun() {
		t.Error("Store-managed manual update began background work")
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
	// Windows is no longer excepted: the in-process swap completes before
	// Apply returns, so nothing outlives it that still needs the staged file.
	if _, err := os.Stat(filepath.Join(v.updater.Dir(), "stage.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("RemoveStage after Apply, stat err %v", err)
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

// updateBackupFixture writes an executable and the backup beside it, and
// returns the executable's path.
func updateBackupFixture(t *testing.T) string {
	t.Helper()
	dest := filepath.Join(t.TempDir(), "picfetch")
	for _, suffix := range []string{"", ".old"} {
		if err := os.WriteFile(dest+suffix, []byte("x"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return dest
}

func backupExists(t *testing.T, dest string) bool {
	t.Helper()
	_, err := os.Stat(dest + ".old")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	return err == nil
}

func TestSweepUpdateBackup_KeepsTheBackupAfterAFailedRestore(t *testing.T) {
	// Op "restore" means the swap could neither install the new binary nor
	// put the old one back, so what sits at dest may be a half-written file
	// that stats fine. The backup is the only executable known to work.
	v := newTestViewer(t)
	dest := updateBackupFixture(t)
	if err := autoupdate.SaveApplyFailure(v.app, autoupdate.ApplyFailure{
		Version: "v0.2.6",
		Reason:  string(update.ReasonAccessDenied),
		Op:      "restore",
		Path:    dest,
	}); err != nil {
		t.Fatal(err)
	}

	got := v.sweepUpdateBackupAt(dest)

	if !backupExists(t, dest) {
		t.Error("swept the backup a failed restore left as the only working executable")
	}
	if got == nil || got.Op != "restore" {
		t.Errorf("returned record = %+v, want the restore failure the report is built from", got)
	}
}

func TestSweepUpdateBackup_SweepsWithNoFailureRecord(t *testing.T) {
	v := newTestViewer(t)
	dest := updateBackupFixture(t)

	got := v.sweepUpdateBackupAt(dest)

	if backupExists(t, dest) {
		t.Error("an applied update left its predecessor on disk forever")
	}
	if got != nil {
		t.Errorf("returned record = %+v, want nil so nothing is reported", got)
	}
}

func TestSweepUpdateBackup_SweepsAfterAFailureThatLeftTheExecutableAlone(t *testing.T) {
	// Every step but "restore" either never touched dest or already rolled
	// it back, so the running executable is intact and the backup is dead
	// weight.
	for _, op := range []string{"", "rename", "copy", "verify", "relaunch"} {
		t.Run(op, func(t *testing.T) {
			v := newTestViewer(t)
			dest := updateBackupFixture(t)
			if err := autoupdate.SaveApplyFailure(v.app, autoupdate.ApplyFailure{
				Version: "v0.2.6",
				Reason:  string(update.ReasonAccessDenied),
				Op:      op,
				Path:    dest,
			}); err != nil {
				t.Fatal(err)
			}

			got := v.sweepUpdateBackupAt(dest)

			if backupExists(t, dest) {
				t.Errorf("Op %q kept a backup nothing depends on", op)
			}
			if got == nil || got.Op != op {
				t.Errorf("returned record = %+v, want the failure with Op %q", got, op)
			}
		})
	}
}

func TestSweepUpdateBackup_KeepsTheBackupWhenTheRecordCannotBeRead(t *testing.T) {
	// An unreadable record is not the same as no record: it might have said
	// "restore", and losing the last working executable is the worse of the
	// two mistakes.
	v := newTestViewer(t)
	dest := updateBackupFixture(t)
	w, err := v.app.Cache().Write(autoupdate.ApplyFailureCacheKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("{not json")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	got := v.sweepUpdateBackupAt(dest)

	if !backupExists(t, dest) {
		t.Error("swept the backup on the strength of a record it could not read")
	}
	if got != nil {
		t.Errorf("returned record = %+v, want nil: an unreadable record says nothing worth reporting", got)
	}
}

// --- next-launch update failure report --------------------------------------

// updateFailurePath is a path no message template can produce on its own, so
// a test that looks for it is really asking whether the record's Path was
// substituted in.
const updateFailurePath = "/Applications/PicFetch.app/Contents/MacOS/picfetch"

// updateFailureDetail stands in for the raw OS error text. It must never
// reach the user: an errno-flavoured sentence explains nothing and cannot be
// translated.
const updateFailureDetail = "update apply: copy " + updateFailurePath + ": CreateFile: Access is denied."

func updateFailureRecord(reason update.FailureReason) autoupdate.ApplyFailure {
	return autoupdate.ApplyFailure{
		Version: "v0.2.6",
		Reason:  string(reason),
		Op:      "copy",
		Path:    updateFailurePath,
		Detail:  updateFailureDetail,
	}
}

// updateFailureReasons is every value the switch can see: the four declared
// reasons plus the shapes a hand-edited or future-written cache file could
// carry. The empty string is what a record saved before Reason existed would
// decode to.
var updateFailureReasons = []update.FailureReason{
	update.ReasonAccessDenied,
	update.ReasonVirusBlocked,
	update.ReasonSharingViolation,
	update.ReasonUnknown,
	"",
	"ACCESS-DENIED",
	"reason-from-a-newer-picfetch",
}

func TestMaybeShowUpdateFailure_NoRecordShowsNothing(t *testing.T) {
	v := newTestViewer(t)

	v.maybeShowUpdateFailure(nil)

	if n := len(v.win.Canvas().Overlays().List()); n != 0 {
		t.Errorf("overlay count = %d, want 0 - a launch after a clean apply must stay quiet", n)
	}
}

func TestMaybeShowUpdateFailure_ClearsRecord(t *testing.T) {
	v := newTestViewer(t)
	rec := updateFailureRecord(update.ReasonAccessDenied)
	if err := autoupdate.SaveApplyFailure(v.app, rec); err != nil {
		t.Fatal(err)
	}

	v.maybeShowUpdateFailure(&rec)

	if n := len(v.win.Canvas().Overlays().List()); n != 1 {
		t.Errorf("overlay count = %d, want 1 (the update failure dialog)", n)
	}
	got, err := autoupdate.LoadApplyFailure(v.app)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("record after the report = %+v, want nil so the next launch stays quiet", got)
	}
}

// TestMaybeShowUpdateFailure_ReportsWhateverVersionFailed pins the deliberate
// difference from maybeShowWhatsNew: that one is gated on the cached version
// matching this build, and the same gate here would suppress the dialog
// without clearing the record - stranding an Op "restore" that vetoes the
// backup sweep on every later launch.
func TestMaybeShowUpdateFailure_ReportsWhateverVersionFailed(t *testing.T) {
	v := newTestViewer(t)
	v.updater.SetCurrentVersion("0.2.5")
	rec := updateFailureRecord(update.ReasonAccessDenied)
	rec.Version = "v9.9.9"
	if err := autoupdate.SaveApplyFailure(v.app, rec); err != nil {
		t.Fatal(err)
	}

	v.maybeShowUpdateFailure(&rec)

	if n := len(v.win.Canvas().Overlays().List()); n != 1 {
		t.Errorf("overlay count = %d, want 1 - the report must not be version-gated", n)
	}
	got, err := autoupdate.LoadApplyFailure(v.app)
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("record after the report = %+v, want nil", got)
	}
}

func TestUpdateFailureMessage_AccessDeniedNamesFolderAccessAndPath(t *testing.T) {
	got := updateFailureMessage(updateFailureRecord(update.ReasonAccessDenied), "windows")

	if !strings.Contains(got, updateFailurePath) {
		t.Errorf("message = %q, want it to name the path that could not be replaced", got)
	}
	if !strings.Contains(got, "Controlled Folder Access") {
		t.Errorf("message = %q, want it to name Controlled Folder Access, the usual cause", got)
	}
	if generic := updateFailureMessage(updateFailureRecord(update.ReasonUnknown), "windows"); got == generic {
		t.Errorf("access-denied message is identical to the generic one (%q) - the classification buys the user nothing", got)
	}
}

// TestUpdateFailureMessage_ControlledFolderAccessIsWindowsOnly is the pin on
// the one sentence in this app that is true on one platform and false on the
// others. A read-only install directory on macOS or Linux makes applyUnix
// return "permission denied", which ClassifyApplyError maps to the same
// ReasonAccessDenied - and before this gate the user was told Windows had
// blocked PicFetch.
func TestUpdateFailureMessage_ControlledFolderAccessIsWindowsOnly(t *testing.T) {
	rec := updateFailureRecord(update.ReasonAccessDenied)
	generic := updateFailureMessage(updateFailureRecord(update.ReasonUnknown), "windows")

	for _, goos := range []string{"darwin", "linux", "freebsd", ""} {
		got := updateFailureMessage(rec, goos)

		for _, forbidden := range []string{"Windows", "Controlled Folder Access", "Ordnerzugriff"} {
			if strings.Contains(got, forbidden) {
				t.Errorf("message on %q = %q, want no mention of %q", goos, got, forbidden)
			}
		}
		if got != generic {
			t.Errorf("message on %q = %q, want the generic wording %q", goos, got, generic)
		}
	}

	if got := updateFailureMessage(rec, "windows"); !strings.Contains(got, "Controlled Folder Access") {
		t.Errorf("message on windows = %q, want it to name Controlled Folder Access", got)
	}
}

func TestUpdateFailureMessage_UnknownReasonFallsBackToGeneric(t *testing.T) {
	want := updateFailureMessage(updateFailureRecord(update.ReasonUnknown), "windows")

	for _, reason := range []update.FailureReason{"", "ACCESS-DENIED", "reason-from-a-newer-picfetch"} {
		if got := updateFailureMessage(updateFailureRecord(reason), "windows"); got != want {
			t.Errorf("message for reason %q = %q, want the generic %q", reason, got, want)
		}
	}
	if !strings.Contains(want, updateFailurePath) {
		t.Errorf("generic message = %q, want it to name the path", want)
	}
}

func TestUpdateFailureMessage_VirusBlockedNamesTheScannerWithoutThePath(t *testing.T) {
	// The antivirus string carries no %s: the file the scanner ate is the
	// download, not the executable the record's Path names.
	got := updateFailureMessage(updateFailureRecord(update.ReasonVirusBlocked), "windows")

	if !strings.Contains(got, "antivirus") {
		t.Errorf("message = %q, want it to name the antivirus", got)
	}
	if strings.Contains(got, updateFailurePath) {
		t.Errorf("message = %q, want no path - the template has no verb to hold one", got)
	}
}

func TestUpdateFailureMessage_SharingViolationSaysTheFileWasInUse(t *testing.T) {
	got := updateFailureMessage(updateFailureRecord(update.ReasonSharingViolation), "windows")

	if !strings.Contains(got, updateFailurePath) {
		t.Errorf("message = %q, want it to name the locked path", got)
	}
	if !strings.Contains(got, "in use") {
		t.Errorf("message = %q, want it to say the file was in use", got)
	}
}

// TestUpdateFailureMessage_EveryReasonReassuresAndStaysWellFormed is the
// invariant sweep: whatever the record says, the user is told the old
// PicFetch still works, no raw error text leaks through, and no template was
// handed the wrong number of arguments.
func TestUpdateFailureMessage_EveryReasonReassuresAndStaysWellFormed(t *testing.T) {
	reassurance := lang.L("The previous version is still installed and running.")

	for _, goos := range []string{"windows", "darwin", "linux"} {
		for _, reason := range updateFailureReasons {
			t.Run(goos+"/"+string(reason), func(t *testing.T) {
				got := updateFailureMessage(updateFailureRecord(reason), goos)

				if !strings.Contains(got, reassurance) {
					t.Errorf("message = %q, want it to end with %q", got, reassurance)
				}
				if strings.Contains(got, updateFailureDetail) {
					t.Errorf("message = %q, want the raw OS error kept out of it", got)
				}
				if strings.Contains(got, "%!") {
					t.Errorf("message = %q, want no Sprintf argument mismatch", got)
				}
				if strings.Contains(got, "%s") {
					t.Errorf("message = %q, want the path substituted, not the verb left standing", got)
				}
			})
		}
	}
}

// TestUpdateFailureMessage_EachClassifiedReasonHasItsOwnWording proves the
// switch actually branches: collapsing any arm into the generic one makes two
// of these equal.
func TestUpdateFailureMessage_EachClassifiedReasonHasItsOwnWording(t *testing.T) {
	seen := map[string]update.FailureReason{}
	for _, reason := range []update.FailureReason{
		update.ReasonAccessDenied,
		update.ReasonVirusBlocked,
		update.ReasonSharingViolation,
		update.ReasonUnknown,
	} {
		got := updateFailureMessage(updateFailureRecord(reason), "windows")
		if other, dup := seen[got]; dup {
			t.Errorf("reasons %q and %q produce the same message %q", other, reason, got)
		}
		seen[got] = reason
	}
}

// TestUpdateFailureMessage_EmptyRecordStillSaysSomethingUseful covers the
// zero value: a record whose fields never got filled in must still yield a
// sentence, not a bare template.
func TestUpdateFailureMessage_EmptyRecordStillSaysSomethingUseful(t *testing.T) {
	got := updateFailureMessage(autoupdate.ApplyFailure{}, "windows")

	if !strings.Contains(got, lang.L("The previous version is still installed and running.")) {
		t.Errorf("message = %q, want the reassurance even with nothing recorded", got)
	}
	if strings.Contains(got, "%!") || strings.Contains(got, "%s") {
		t.Errorf("message = %q, want a substituted template", got)
	}
}

// germanUpdateFailureMessage builds the message a German user reads, from
// the shipped catalogue rather than a copy: lang.L has no bundle loaded in
// this package's tests (main.go is what calls AddTranslationsFS), so the only
// way to measure the translated wording is to read it off disk.
func germanUpdateFailureMessage(t *testing.T, path string) string {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("..", "..", "translations", "de.json"))
	if err != nil {
		t.Fatal(err)
	}
	var catalogue map[string]string
	if err := json.Unmarshal(raw, &catalogue); err != nil {
		t.Fatal(err)
	}
	lookup := func(key string) string {
		value, ok := catalogue[key]
		if !ok {
			t.Fatalf("de.json has no entry for %q", key)
		}
		return value
	}
	var cfaKey string
	for key := range catalogue {
		if strings.HasPrefix(key, "Windows blocked PicFetch from replacing itself at %s.") {
			cfaKey = key
		}
	}
	if cfaKey == "" {
		t.Fatal("de.json has no Controlled Folder Access entry")
	}
	return fmt.Sprintf(lookup(cfaKey), path) + "\n\n" +
		lookup("The previous version is still installed and running.")
}

// updateFailureMessages is the set of wordings the failure dialog has to
// lay out: the German Controlled Folder Access text is the longest, and a
// deep install path pushes even the English one over a fixed box.
func updateFailureMessages(t *testing.T) map[string]string {
	t.Helper()

	const deepPath = `C:\Users\Ein sehr langer Benutzername\Documents\Programme\PicFetch portable\bin\picfetch.exe`
	deep := updateFailureRecord(update.ReasonAccessDenied)
	deep.Path = deepPath

	return map[string]string{
		"english":     updateFailureMessage(updateFailureRecord(update.ReasonAccessDenied), "windows"),
		"german":      germanUpdateFailureMessage(t, updateFailurePath),
		"deep path":   updateFailureMessage(deep, "windows"),
		"german deep": germanUpdateFailureMessage(t, deepPath),
	}
}

// showUpdateFailureDialog opens the real dialog on win, sized the way
// maybeShowUpdateFailure sizes it, and hands back the scrolled body.
func showUpdateFailureDialog(t *testing.T, win fyne.Window, message string) *container.Scroll {
	t.Helper()

	body := updateFailureContent(message, updateFailureBodySize(win.Canvas().Size()))
	d := dialog.NewCustomConfirm(
		lang.L("Update could not be installed"),
		lang.L("Open download page"),
		lang.L("Close"),
		body,
		func(bool) {},
		win,
	)
	d.Show()
	t.Cleanup(d.Hide)

	if needed, canvas := d.MinSize(), win.Canvas().Size(); needed.Width > canvas.Width || needed.Height > canvas.Height {
		t.Errorf("dialog MinSize = %v, want no larger than the %v canvas - the popup gets clamped and the overflow is simply cut", needed, canvas)
	}
	if _, ok := body.Content.(*widget.Label); !ok {
		t.Fatalf("scrolled content = %T, want the message label", body.Content)
	}
	if got, want := body.Content.Size().Width, body.Size().Width; got > want+0.5 {
		t.Errorf("content width %.1f exceeds the %.1f viewport - the message would need a horizontal scrollbar", got, want)
	}
	return body
}

// TestUpdateFailureContent_KeepsTheWholeMessageReachableAtTheSmallestWindow
// is calibrated to startW x startH, the size PicFetch opens at with no
// remembered geometry. A modal popup is clamped to its canvas and whatever
// the dialog cannot fit is cut off, not made reachable - the German
// Controlled Folder Access text overflows a window that small, so at this
// size the scroll is what carries the tail.
func TestUpdateFailureContent_KeepsTheWholeMessageReachableAtTheSmallestWindow(t *testing.T) {
	_, win, _ := newTestUI(t)

	canvas := win.Canvas().Size()
	if canvas.Width != startW || canvas.Height != startH {
		t.Fatalf("canvas = %v, want %vx%v - this test's numbers assume the default window", canvas, startW, startH)
	}

	for name, message := range updateFailureMessages(t) {
		t.Run(name, func(t *testing.T) {
			body := showUpdateFailureDialog(t, win, message)

			label := body.Content.(*widget.Label)
			if got, want := label.Size().Height, label.MinSize().Height; got+0.5 < want {
				t.Errorf("label laid out at %.1fpt but needs %.1fpt - the tail of the message is cut off, not scrollable", got, want)
			}
			if got := body.Size().Height; got < updateFailureBodyMinH {
				t.Errorf("visible body = %.1fpt, want at least %dpt - the message is a sliver", got, updateFailureBodyMinH)
			}

			body.ScrollToBottom()
			if reached, need := body.Offset.Y+body.Size().Height, body.Content.Size().Height; reached+0.5 < need {
				t.Errorf("scrolled to %.1fpt of %.1fpt - the end of the message cannot be reached", reached, need)
			}
		})
	}
}

// TestUpdateFailureContent_NeedsNoScrollingOnARoomyWindow is the reason the
// body is sized from the canvas at all. The dialog is the one place PicFetch
// asks the user to go and change a Windows security setting, and a message
// that arrives two visible lines at a time reads as a warning to dismiss
// rather than an instruction to follow. Any window with room for the whole
// text has to show the whole text.
//
// 1280x800 stands in for a normal desktop window - well under the size the
// screenshot that prompted this came from, so the maxima are what bound the
// body here, not the canvas.
func TestUpdateFailureContent_NeedsNoScrollingOnARoomyWindow(t *testing.T) {
	_, win, _ := newTestUI(t)
	win.Resize(fyne.NewSize(1280, 800))

	for name, message := range updateFailureMessages(t) {
		t.Run(name, func(t *testing.T) {
			body := showUpdateFailureDialog(t, win, message)

			if got, want := body.Content.Size().Height, body.Size().Height; got > want+0.5 {
				t.Errorf("message needs %.1fpt in a %.1fpt viewport - it still has to be scrolled on a window with room to spare", got, want)
			}
			if body.Offset.Y != 0 {
				t.Errorf("body opened scrolled to %.1f, want the top", body.Offset.Y)
			}
		})
	}
}

// TestUpdateFailureBodySize_TracksTheCanvasBetweenItsBounds covers the two
// clamps without opening a dialog: a window too small for the floor, and one
// large enough that the maxima stop the paragraph from spanning the display.
func TestUpdateFailureBodySize_TracksTheCanvasBetweenItsBounds(t *testing.T) {
	tests := []struct {
		name   string
		canvas fyne.Size
		want   fyne.Size
	}{
		{"tiny window takes the floor", fyne.NewSize(120, 90), fyne.NewSize(updateFailureBodyMinW, updateFailureBodyMinH)},
		{"default window takes the canvas", fyne.NewSize(startW, startH), fyne.NewSize(startW-updateFailureChromeW, startH-updateFailureChromeH)},
		{"full screen takes the maxima", fyne.NewSize(3840, 2160), fyne.NewSize(updateFailureBodyMaxW, updateFailureBodyMaxH)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := updateFailureBodySize(tc.canvas); got != tc.want {
				t.Errorf("updateFailureBodySize(%v) = %v, want %v", tc.canvas, got, tc.want)
			}
		})
	}
}

// urlRecorder is the shared test app with OpenURL replaced. Embedding the
// interface keeps the substitution local to one viewer: nothing calls
// fyne.SetCurrentApp, so the process-wide current app newTestUI depends on
// stays the one every widget was built against.
type urlRecorder struct {
	fyne.App

	got *url.URL
	err error
}

func (r *urlRecorder) OpenURL(u *url.URL) error {
	r.got = u
	return r.err
}

func TestOpenReleasesPage_HandsTheReleasesURLToTheApp(t *testing.T) {
	v := newTestViewer(t)
	recorder := &urlRecorder{App: v.app}
	v.app = recorder

	v.openReleasesPage()

	if recorder.got == nil {
		t.Fatal("OpenURL was never called - the confirm button would do nothing")
	}
	if got := recorder.got.String(); got != update.DownloadPageURL {
		t.Errorf("OpenURL got %q, want %q", got, update.DownloadPageURL)
	}
}

// TestOpenReleasesPage_LogsAnAppThatCannotOpenTheURL covers the branch a
// desktop with no registered https handler takes. Swallowing it would leave
// a button that silently does nothing.
func TestOpenReleasesPage_LogsAnAppThatCannotOpenTheURL(t *testing.T) {
	v := newTestViewer(t)
	v.app = &urlRecorder{App: v.app, err: errors.New("no handler for https")}

	logged := captureLog(t, v.openReleasesPage)

	for _, want := range []string{"failed to open the download page", "no handler for https"} {
		if !strings.Contains(logged, want) {
			t.Errorf("log = %q, want it to mention %q", logged, want)
		}
	}
}

// captureLog collects what fn writes through the standard logger, which is
// where fyne.LogError ends up. Unlike the same pattern in internal/ui/help
// the buffer is mutex-guarded: this package keeps background goroutines
// alive across tests, and one of them logging mid-capture would otherwise be
// a data race rather than just noise.
func captureLog(t *testing.T, fn func()) string {
	t.Helper()

	var buf lockedBuffer
	restore := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(restore) })

	fn()

	log.SetOutput(restore)
	return buf.String()
}

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
