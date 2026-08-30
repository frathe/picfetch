package autoupdate

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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/update"
)

type fakeVerifier struct{}

func (fakeVerifier) Verify(context.Context, []byte, []byte, update.VerifyPolicy) error {
	return nil
}

type failingVerifier struct{ err error }

func (v failingVerifier) Verify(context.Context, []byte, []byte, update.VerifyPolicy) error {
	return v.err
}

type blockingVerifier struct {
	once    sync.Once
	entered chan struct{}
	release chan struct{}
}

func (v *blockingVerifier) Verify(context.Context, []byte, []byte, update.VerifyPolicy) error {
	v.once.Do(func() { close(v.entered) })
	<-v.release
	return nil
}

func updaterAssetName(t *testing.T) string {
	t.Helper()
	name, ok := update.AssetName(runtime.GOOS, runtime.GOARCH)
	if !ok {
		t.Skipf("no update asset for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return name
}

func updaterNativeArchive(t *testing.T, assetName string) []byte {
	t.Helper()
	var buf bytes.Buffer
	switch {
	case strings.HasSuffix(assetName, ".tar.gz"):
		gw := gzip.NewWriter(&buf)
		tw := tar.NewWriter(gw)
		name := strings.TrimSuffix(assetName, ".tar.gz")
		body := []byte("elf")
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(body); err != nil {
			t.Fatal(err)
		}
		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}
		if err := gw.Close(); err != nil {
			t.Fatal(err)
		}
	case strings.Contains(assetName, "windows"):
		zw := zip.NewWriter(&buf)
		w, err := zw.Create("picfetch.exe")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte("exe")); err != nil {
			t.Fatal(err)
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
	case strings.Contains(assetName, "macos"):
		zw := zip.NewWriter(&buf)
		for name, body := range map[string]string{
			"PicFetch.app/Contents/MacOS/picfetch": "macho",
			"PicFetch.app/Contents/Info.plist":     "plist",
		} {
			w, err := zw.Create(name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write([]byte(body)); err != nil {
				t.Fatal(err)
			}
		}
		if err := zw.Close(); err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatalf("unsupported update asset %q", assetName)
	}
	return buf.Bytes()
}

func updaterReleaseServer(t *testing.T, version, assetName string, archive []byte, digest string, archiveCalls *int) *httptest.Server {
	t.Helper()
	var callsMu sync.Mutex
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/repos/frathe/picfetch/releases/latest":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tag_name": version,
				"body":     "release notes",
				"assets": []any{map[string]any{
					"name":                 assetName,
					"browser_download_url": srv.URL + "/" + assetName,
					"digest":               "sha256:" + digest,
				}},
			})
		case r.URL.Path == "/"+assetName:
			if archiveCalls != nil {
				callsMu.Lock()
				*archiveCalls++
				callsMu.Unlock()
			}
			_, _ = w.Write(archive)
		case strings.Contains(r.URL.Path, "/attestations/"):
			_, _ = w.Write([]byte(`{"attestations":[{"bundle":{"mediaType":"test"}}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func updaterClient(u *Updater, srv *httptest.Server, verifier update.Verifier, now time.Time) *update.Client {
	return update.NewClient(update.Config{
		BaseURL:  srv.URL,
		HTTP:     srv.Client(),
		Now:      func() time.Time { return now },
		Verify:   verifier,
		StageDir: u.Dir(),
		GOOS:     runtime.GOOS,
		GOARCH:   runtime.GOARCH,
	})
}

func waitUpdater(t *testing.T, u *Updater) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := u.Settle(ctx); err != nil {
		t.Fatal("timed out waiting for updater workers")
	}
}

// --- Due / not-due -----------------------------------------------------

func TestUpdater_Due(t *testing.T) {
	now := time.Date(2026, 8, 26, 9, 0, 0, 0, time.Local)

	tests := []struct {
		name    string
		lastDay string
		want    bool
	}{
		{"never checked is due", "", true},
		{"same local calendar day is not due", "2026-08-26", false},
		{"previous local calendar day is due", "2026-08-25", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u := New(test.NewApp(), t.TempDir(), nil)
			if tc.lastDay != "" {
				u.SetLastCheckDay(tc.lastDay)
			}
			if got := u.Due(now); got != tc.want {
				t.Errorf("Due(%v) with lastCheckDay %q = %v, want %v", now, tc.lastDay, got, tc.want)
			}
		})
	}
}

func TestUpdater_LastCheckDayZeroValueIsEmpty(t *testing.T) {
	u := New(test.NewApp(), t.TempDir(), nil)
	if got := u.LastCheckDay(); got != "" {
		t.Errorf("LastCheckDay() on a fresh Updater = %q, want empty", got)
	}
}

// --- LastCheckDay persistence seam --------------------------------------

func TestUpdater_SetLastCheckDayCallsPersist(t *testing.T) {
	var got []string
	var mu sync.Mutex
	u := New(test.NewApp(), t.TempDir(), func(day string) {
		mu.Lock()
		defer mu.Unlock()
		got = append(got, day)
	})

	u.SetLastCheckDay("2026-08-26")
	u.SetLastCheckDay("2026-08-27")

	mu.Lock()
	defer mu.Unlock()
	want := []string{"2026-08-26", "2026-08-27"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("persisted days = %v, want %v", got, want)
	}
	if u.LastCheckDay() != "2026-08-27" {
		t.Errorf("LastCheckDay() = %q, want 2026-08-27", u.LastCheckDay())
	}
}

// TestUpdater_NilPersistIsSafe pins that persist is an optional seam: a
// nil func (the zero value, and what most package-level unit tests here
// pass) must not panic SetLastCheckDay.
func TestUpdater_NilPersistIsSafe(t *testing.T) {
	u := New(test.NewApp(), t.TempDir(), nil)
	u.SetLastCheckDay("2026-08-26")
	if got := u.LastCheckDay(); got != "2026-08-26" {
		t.Errorf("LastCheckDay() = %q, want 2026-08-26", got)
	}
}

// TestUpdater_LastCheckDayConcurrent proves dayMu actually guards
// lastCheckDay end to end through the exported pair, the same invariant
// internal/ui's TestLastUpdateCheckDay_ConcurrentWithCurrentPreferences
// pins at the viewer layer.
func TestUpdater_LastCheckDayConcurrent(t *testing.T) {
	u := New(test.NewApp(), t.TempDir(), nil)
	const n = 200
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for range n {
			u.SetLastCheckDay("2026-08-26")
		}
	}()
	go func() {
		defer wg.Done()
		for range n {
			_ = u.LastCheckDay()
		}
	}()
	wg.Wait()
}

// --- stale-stage removal -------------------------------------------------

func TestUpdater_RemoveStaleStage_NoStagePresentIsNoop(t *testing.T) {
	u := New(test.NewApp(), t.TempDir(), nil)
	u.SetCurrentVersion("0.2.6")

	u.RemoveStaleStage() // must not panic

	if _, err := update.LoadStage(u.Dir()); err == nil {
		t.Fatal("expected no stage to load")
	}
}

func TestUpdater_RemoveStaleStage_OlderStagedVersionIsRemoved(t *testing.T) {
	dir := t.TempDir()
	if err := update.SaveStage(dir, update.Stage{Version: "v0.2.5", Notes: "stale"}); err != nil {
		t.Fatal(err)
	}
	u := New(test.NewApp(), dir, nil)
	u.SetCurrentVersion("0.2.6")

	u.RemoveStaleStage()

	if _, err := update.LoadStage(dir); err == nil {
		t.Error("stale stage must be removed")
	}
}

func TestUpdater_RemoveStaleStage_SameStagedVersionIsRemoved(t *testing.T) {
	dir := t.TempDir()
	if err := update.SaveStage(dir, update.Stage{Version: "v0.2.6", Notes: "same"}); err != nil {
		t.Fatal(err)
	}
	u := New(test.NewApp(), dir, nil)
	u.SetCurrentVersion("0.2.6")

	u.RemoveStaleStage()

	if _, err := update.LoadStage(dir); err == nil {
		t.Error("a stage matching the current version must be removed too - Newer is strict")
	}
}

func TestUpdater_RemoveStaleStage_DoesNotBlockActiveTransaction(t *testing.T) {
	dir := t.TempDir()
	if err := update.SaveStage(dir, update.Stage{Version: "v0.2.5", Notes: "stale"}); err != nil {
		t.Fatal(err)
	}
	u := New(test.NewApp(), dir, nil)
	u.SetCurrentVersion("0.2.6")

	<-u.transaction
	u.RemoveStaleStage()
	if _, err := update.LoadStage(dir); err != nil {
		t.Fatalf("busy cleanup should leave the worker-owned stage alone: %v", err)
	}
	u.transaction <- struct{}{}

	u.RemoveStaleStage()
	if _, err := update.LoadStage(dir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("idle cleanup should remove the stale stage, got: %v", err)
	}
}

func TestUpdater_RemoveStaleStage_NewerStagedVersionIsKept(t *testing.T) {
	dir := t.TempDir()
	if err := update.SaveStage(dir, update.Stage{Version: "v0.2.7", Notes: "fresh"}); err != nil {
		t.Fatal(err)
	}
	u := New(test.NewApp(), dir, nil)
	u.SetCurrentVersion("0.2.6")

	u.RemoveStaleStage()

	st, err := update.LoadStage(dir)
	if err != nil {
		t.Fatalf("expected the still-relevant stage to survive, got: %v", err)
	}
	if st.Version != "v0.2.7" {
		t.Errorf("LoadStage().Version = %q, want v0.2.7", st.Version)
	}
}

// --- version-normalisation gates -----------------------------------------

func TestUpdater_CurrentVersion_OverrideWins(t *testing.T) {
	u := New(test.NewApp(), t.TempDir(), nil)
	u.SetCurrentVersion("0.2.5")

	if got := u.CurrentVersion(); got != "0.2.5" {
		t.Errorf("CurrentVersion() = %q, want 0.2.5", got)
	}
	if got := update.NormalizeVersion(u.CurrentVersion()); got != "v0.2.5" {
		t.Errorf("NormalizeVersion(CurrentVersion()) = %q, want v0.2.5", got)
	}
}

// TestUpdater_CurrentVersion_FallsBackToAppMetadata pins the gate
// maybeStartUpdateCheck relies on: Fyne's test app ships an empty
// Metadata().Version, so an Updater with no override normalises to "" and
// every caller must treat that as "no usable version, do not check".
func TestUpdater_CurrentVersion_FallsBackToAppMetadata(t *testing.T) {
	app := test.NewApp()
	u := New(app, t.TempDir(), nil)

	if got := u.CurrentVersion(); got != app.Metadata().Version {
		t.Errorf("CurrentVersion() = %q, want app.Metadata().Version %q", got, app.Metadata().Version)
	}
	if got := update.NormalizeVersion(u.CurrentVersion()); got != "" {
		t.Errorf("NormalizeVersion(CurrentVersion()) = %q, want empty (test app metadata is unset)", got)
	}
}

func TestUpdater_CurrentVersion_EmptyOverrideDoesNotStick(t *testing.T) {
	u := New(test.NewApp(), t.TempDir(), nil)
	u.SetCurrentVersion("0.2.5")
	u.SetCurrentVersion("") // an empty override falls back, same as never set

	if got := u.CurrentVersion(); got != "" {
		t.Errorf("CurrentVersion() = %q, want empty (Fyne test app metadata)", got)
	}
}

// --- Dir / Client / Done round trips --------------------------------------

func TestUpdater_DirRoundTrip(t *testing.T) {
	u := New(test.NewApp(), "", nil)
	if got := u.Dir(); got != "" {
		t.Fatalf("Dir() = %q, want empty before SetDir", got)
	}

	dir := filepath.Join(t.TempDir(), "updates")
	u.SetDir(dir)
	if got := u.Dir(); got != dir {
		t.Errorf("Dir() = %q, want %q", got, dir)
	}
}

func TestUpdater_ClientRoundTrip(t *testing.T) {
	u := New(test.NewApp(), t.TempDir(), nil)
	if u.Client() != nil {
		t.Fatal("Client() on a fresh Updater must be nil")
	}

	c := update.NewClient(update.Config{StageDir: u.Dir()})
	u.SetClient(c)
	if u.Client() != c {
		t.Error("Client() must return exactly what SetClient assigned")
	}
}

func TestUpdater_EnsureClient_PreSetClientBypassesVerifierFactory(t *testing.T) {
	u := New(test.NewApp(), t.TempDir(), nil)
	want := update.NewClient(update.Config{StageDir: u.Dir()})
	u.SetClient(want)
	calls := 0
	u.SetVerifierFactory(func() (update.Verifier, error) {
		calls++
		return fakeVerifier{}, nil
	})

	if err := u.EnsureClient(); err != nil {
		t.Fatalf("EnsureClient() error = %v, want nil", err)
	}
	if calls != 0 {
		t.Errorf("verifier factory calls = %d, want 0", calls)
	}
	if got := u.Client(); got != want {
		t.Error("EnsureClient() replaced the pre-set client")
	}
}

func TestUpdater_EnsureClient_SuccessIsIdempotent(t *testing.T) {
	u := New(test.NewApp(), t.TempDir(), nil)
	calls := 0
	u.SetVerifierFactory(func() (update.Verifier, error) {
		calls++
		return fakeVerifier{}, nil
	})

	if err := u.EnsureClient(); err != nil {
		t.Fatalf("first EnsureClient() error = %v, want nil", err)
	}
	want := u.Client()
	if want == nil {
		t.Fatal("Client() after successful EnsureClient() must not be nil")
	}
	if err := u.EnsureClient(); err != nil {
		t.Fatalf("second EnsureClient() error = %v, want nil", err)
	}
	if calls != 1 {
		t.Errorf("verifier factory calls = %d, want 1", calls)
	}
	if got := u.Client(); got != want {
		t.Error("repeated EnsureClient() replaced the prepared client")
	}
}

func TestUpdater_EnsureClient_FailureIsRetryable(t *testing.T) {
	u := New(test.NewApp(), t.TempDir(), nil)
	wantErr := errors.New("verifier unavailable")
	calls := 0
	u.SetVerifierFactory(func() (update.Verifier, error) {
		calls++
		if calls == 1 {
			return nil, wantErr
		}
		return fakeVerifier{}, nil
	})

	if err := u.EnsureClient(); !errors.Is(err, wantErr) {
		t.Fatalf("first EnsureClient() error = %v, want errors.Is(_, %v)", err, wantErr)
	}
	if u.Client() != nil {
		t.Fatal("Client() after failed EnsureClient() must remain nil")
	}
	if err := u.EnsureClient(); err != nil {
		t.Fatalf("second EnsureClient() error = %v, want nil", err)
	}
	if calls != 2 {
		t.Errorf("verifier factory calls = %d, want 2", calls)
	}
	if u.Client() == nil {
		t.Fatal("Client() after successful retry must not be nil")
	}
}

func TestUpdater_EnsureClient_NilVerifierFactoryRestoresDefault(t *testing.T) {
	u := New(test.NewApp(), t.TempDir(), nil)
	u.SetVerifierFactory(func() (update.Verifier, error) { return fakeVerifier{}, nil })
	u.SetVerifierFactory(nil)

	got := reflect.ValueOf(u.verifierFactory).Pointer()
	want := reflect.ValueOf(update.NewSigstoreVerifier).Pointer()
	if got != want {
		t.Error("SetVerifierFactory(nil) did not restore update.NewSigstoreVerifier")
	}
}

func TestUpdater_Start_RequiresPreparedClient(t *testing.T) {
	u := New(test.NewApp(), t.TempDir(), nil)

	err := u.Start(context.Background(), func() bool { return false }, "v0.2.6")

	if !errors.Is(err, errClientNotPrepared) {
		t.Fatalf("Start() error = %v, want errors.Is(_, %v)", err, errClientNotPrepared)
	}
	if u.Done().Begun() {
		t.Error("Done() after invalid Start must not report Begun")
	}
}

func TestUpdater_Now_FallsBackToRealClockWithoutClient(t *testing.T) {
	u := New(test.NewApp(), t.TempDir(), nil)

	before := time.Now()
	got := u.Now()
	after := time.Now()

	if got.Before(before) || got.After(after) {
		t.Errorf("Now() = %v, want between %v and %v", got, before, after)
	}
}

func TestUpdater_Now_UsesClientClockOnceOneExists(t *testing.T) {
	u := New(test.NewApp(), t.TempDir(), nil)
	fixed := time.Date(2026, 8, 26, 9, 0, 0, 0, time.UTC)
	u.SetClient(update.NewClient(update.Config{
		StageDir: u.Dir(),
		Now:      func() time.Time { return fixed },
	}))

	if got := u.Now(); !got.Equal(fixed) {
		t.Errorf("Now() = %v, want %v", got, fixed)
	}
}

func TestUpdater_Done_StartsNotBegun(t *testing.T) {
	u := New(test.NewApp(), t.TempDir(), nil)
	if u.Done().Begun() {
		t.Error("Done() on a fresh Updater must not report Begun")
	}
}

func TestDefaultDir_EndsInPicfetchUpdates(t *testing.T) {
	got := DefaultDir()
	want := filepath.Join("picfetch", "updates")
	if filepath.Base(filepath.Dir(got)) != "picfetch" || filepath.Base(got) != "updates" {
		t.Errorf("DefaultDir() = %q, want it to end in %q", got, want)
	}
}

// --- manual events / shared worker -----------------------------------------

func TestUpdater_StartManual_CurrentEventAndSuccessfulCheckDay(t *testing.T) {
	asset := updaterAssetName(t)
	now := time.Date(2026, 8, 30, 10, 0, 0, 0, time.Local)
	srv := updaterReleaseServer(t, "v0.2.5", asset, nil, strings.Repeat("0", 64), nil)
	u := New(test.NewApp(), t.TempDir(), nil)
	u.SetClient(updaterClient(u, srv, fakeVerifier{}, now))

	var events []string
	u.StartManual(context.Background(), func() bool { return false }, "v0.2.5", Events{
		Current:     func() { events = append(events, "current") },
		Downloading: func(string) { events = append(events, "downloading") },
		Ready:       func(update.Stage) { events = append(events, "ready") },
		Failed:      func(error) { events = append(events, "failed") },
	})
	waitUpdater(t, u)

	if want := []string{"current"}; !reflect.DeepEqual(events, want) {
		t.Errorf("events = %v, want %v", events, want)
	}
	if got := u.LastCheckDay(); got != "2026-08-30" {
		t.Errorf("LastCheckDay() = %q, want 2026-08-30", got)
	}
}

func TestUpdater_StartManual_DownloadEventOrder(t *testing.T) {
	asset := updaterAssetName(t)
	archive := updaterNativeArchive(t, asset)
	sum := sha256.Sum256(archive)
	srv := updaterReleaseServer(t, "v0.2.6", asset, archive, hex.EncodeToString(sum[:]), nil)
	u := New(test.NewApp(), t.TempDir(), nil)
	u.SetClient(updaterClient(u, srv, fakeVerifier{}, time.Date(2026, 8, 30, 10, 0, 0, 0, time.Local)))

	var events []string
	u.StartManual(context.Background(), func() bool { return false }, "v0.2.5", Events{
		Downloading: func(version string) { events = append(events, "downloading:"+version) },
		Progress:    func(update.DownloadProgress) { events = append(events, "progress") },
		Current:     func() { events = append(events, "current") },
		Ready:       func(st update.Stage) { events = append(events, "ready:"+st.Version) },
		Failed:      func(error) { events = append(events, "failed") },
	})
	waitUpdater(t, u)

	if len(events) < 3 {
		t.Fatalf("events = %v, want downloading, progress, ready", events)
	}
	if events[0] != "downloading:v0.2.6" || events[len(events)-1] != "ready:v0.2.6" {
		t.Errorf("events = %v, want downloading first and ready last", events)
	}
	for _, event := range events[1 : len(events)-1] {
		if event != "progress" {
			t.Errorf("middle event = %q, want progress; all events %v", event, events)
		}
	}
}

func TestUpdater_StartManual_ReusesMatchingUsableStageAfterCheck(t *testing.T) {
	asset := updaterAssetName(t)
	archive := updaterNativeArchive(t, asset)
	sum := sha256.Sum256(archive)
	archiveCalls := 0
	srv := updaterReleaseServer(t, "v0.2.6", asset, archive, hex.EncodeToString(sum[:]), &archiveCalls)
	dir := t.TempDir()
	u := New(test.NewApp(), dir, nil)
	client := updaterClient(u, srv, fakeVerifier{}, time.Date(2026, 8, 30, 10, 0, 0, 0, time.Local))
	rel, err := client.Check(context.Background(), "v0.2.5")
	if err != nil {
		t.Fatal(err)
	}
	if rel == nil {
		t.Fatal("Check found no release to download")
	}
	want, err := client.Download(context.Background(), *rel)
	if err != nil {
		t.Fatal(err)
	}
	downloadsBeforeManual := archiveCalls
	u.SetClient(client)

	var got update.Stage
	var downloading, failed bool
	u.StartManual(context.Background(), func() bool { return false }, "v0.2.5", Events{
		Downloading: func(string) { downloading = true },
		Ready:       func(st update.Stage) { got = st },
		Failed:      func(error) { failed = true },
	})
	waitUpdater(t, u)

	if got.Version != want.Version || got.BinaryPath != want.BinaryPath {
		t.Errorf("Ready stage = %+v, want cached %+v", got, want)
	}
	if archiveCalls != downloadsBeforeManual || downloading || failed {
		t.Errorf("archiveCalls=%d before=%d downloading=%v failed=%v, want reuse without download/failure", archiveCalls, downloadsBeforeManual, downloading, failed)
	}
}

func TestUpdater_StartManual_MissingVerifierNeverReusesVerifiedStage(t *testing.T) {
	asset := updaterAssetName(t)
	archive := updaterNativeArchive(t, asset)
	sum := sha256.Sum256(archive)
	srv := updaterReleaseServer(t, "v0.2.6", asset, archive, hex.EncodeToString(sum[:]), nil)
	u := New(test.NewApp(), t.TempDir(), nil)
	client := updaterClient(u, srv, fakeVerifier{}, time.Date(2026, 8, 30, 10, 0, 0, 0, time.Local))
	rel, err := client.Check(context.Background(), "v0.2.5")
	if err != nil {
		t.Fatal(err)
	}
	if rel == nil {
		t.Fatal("Check found no release to download")
	}
	if _, err := client.Download(context.Background(), *rel); err != nil {
		t.Fatal(err)
	}
	u.SetClient(nil)
	wantErr := errors.New("verifier unavailable")
	u.SetVerifierFactory(func() (update.Verifier, error) { return nil, wantErr })

	var ready bool
	var gotErr error
	u.StartManual(context.Background(), func() bool { return false }, "v0.2.5", Events{
		Ready:  func(update.Stage) { ready = true },
		Failed: func(err error) { gotErr = err },
	})
	waitUpdater(t, u)

	if ready || !errors.Is(gotErr, wantErr) {
		t.Errorf("ready=%v error=%v, want verifier failure and no Ready", ready, gotErr)
	}
}

func TestUpdater_StartManual_UnverifiedMatchingStageNeverReady(t *testing.T) {
	asset := updaterAssetName(t)
	archive := updaterNativeArchive(t, asset)
	srv := updaterReleaseServer(t, "v0.2.6", asset, archive, strings.Repeat("f", 64), nil)
	dir := t.TempDir()
	bin := filepath.Join(dir, "forged")
	if err := os.WriteFile(bin, []byte("forged"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := update.SaveStage(dir, update.Stage{Version: "v0.2.6", BinaryPath: bin}); err != nil {
		t.Fatal(err)
	}
	u := New(test.NewApp(), dir, nil)
	u.SetClient(updaterClient(u, srv, fakeVerifier{}, time.Date(2026, 8, 30, 10, 0, 0, 0, time.Local)))

	var ready, failed bool
	u.StartManual(context.Background(), func() bool { return false }, "v0.2.5", Events{
		Ready:  func(update.Stage) { ready = true },
		Failed: func(error) { failed = true },
	})
	waitUpdater(t, u)
	if ready || !failed {
		t.Errorf("ready=%v failed=%v, want forged stage rejection followed by download failure", ready, failed)
	}
}

func TestUpdater_StartManual_TamperedVerifiedStageIsRedownloaded(t *testing.T) {
	asset := updaterAssetName(t)
	archive := updaterNativeArchive(t, asset)
	sum := sha256.Sum256(archive)
	archiveCalls := 0
	srv := updaterReleaseServer(t, "v0.2.6", asset, archive, hex.EncodeToString(sum[:]), &archiveCalls)
	u := New(test.NewApp(), t.TempDir(), nil)
	client := updaterClient(u, srv, fakeVerifier{}, time.Date(2026, 8, 30, 10, 0, 0, 0, time.Local))
	rel, err := client.Check(context.Background(), "v0.2.5")
	if err != nil {
		t.Fatal(err)
	}
	if rel == nil {
		t.Fatal("Check found no release to download")
	}
	st, err := client.Download(context.Background(), *rel)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(st.BinaryPath, []byte("tampered"), 0o755); err != nil {
		t.Fatal(err)
	}
	downloadsBeforeManual := archiveCalls
	u.SetClient(client)

	var ready, failed bool
	u.StartManual(context.Background(), func() bool { return false }, "v0.2.5", Events{
		Ready:  func(update.Stage) { ready = true },
		Failed: func(error) { failed = true },
	})
	waitUpdater(t, u)
	if !ready || failed || archiveCalls != downloadsBeforeManual+1 {
		t.Errorf("ready=%v failed=%v archiveCalls=%d before=%d, want verified redownload", ready, failed, archiveCalls, downloadsBeforeManual)
	}
	loaded, err := update.LoadStage(u.Dir())
	if err != nil {
		t.Fatal(err)
	}
	if err := update.ValidateStageForPlatform(loaded, runtime.GOOS, runtime.GOARCH); err != nil {
		t.Fatalf("redownloaded stage = %v", err)
	}
}

func TestUpdater_StartManual_CheckFailureDoesNotRecordDay(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	t.Cleanup(srv.Close)
	u := New(test.NewApp(), t.TempDir(), nil)
	u.SetClient(updaterClient(u, srv, fakeVerifier{}, time.Date(2026, 8, 30, 10, 0, 0, 0, time.Local)))

	var failures int
	u.StartManual(context.Background(), func() bool { return false }, "v0.2.5", Events{
		Failed: func(error) { failures++ },
	})
	waitUpdater(t, u)

	if failures != 1 {
		t.Errorf("Failed calls = %d, want 1", failures)
	}
	if got := u.LastCheckDay(); got != "" {
		t.Errorf("LastCheckDay() = %q after failed check, want empty", got)
	}
}

func TestUpdater_StartManual_WrongDigestNeverReadyAndLeavesNoStage(t *testing.T) {
	testManualVerificationFailure(t, fakeVerifier{}, strings.Repeat("f", 64))
}

func TestUpdater_StartManual_AttestationFailureNeverReadyAndLeavesNoStage(t *testing.T) {
	wantErr := errors.New("attestation rejected")
	testManualVerificationFailure(t, failingVerifier{err: wantErr}, "")
}

func testManualVerificationFailure(t *testing.T, verifier update.Verifier, digestOverride string) {
	t.Helper()
	asset := updaterAssetName(t)
	archive := updaterNativeArchive(t, asset)
	sum := sha256.Sum256(archive)
	digest := hex.EncodeToString(sum[:])
	if digestOverride != "" {
		digest = digestOverride
	}
	srv := updaterReleaseServer(t, "v0.2.6", asset, archive, digest, nil)
	u := New(test.NewApp(), t.TempDir(), nil)
	u.SetClient(updaterClient(u, srv, verifier, time.Date(2026, 8, 30, 10, 0, 0, 0, time.Local)))

	var ready, failed bool
	u.StartManual(context.Background(), func() bool { return false }, "v0.2.5", Events{
		Ready:  func(update.Stage) { ready = true },
		Failed: func(error) { failed = true },
	})
	waitUpdater(t, u)

	if ready || !failed {
		t.Errorf("ready=%v failed=%v, want no Ready and one failure path", ready, failed)
	}
	if _, err := update.LoadStage(u.Dir()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("verification failure left a usable stage: %v", err)
	}
}

func TestUpdater_StartManual_PreparesVerifierOnTrackedWorker(t *testing.T) {
	u := New(test.NewApp(), t.TempDir(), nil)
	started := make(chan struct{})
	release := make(chan struct{})
	wantErr := errors.New("preparation failed")
	u.SetVerifierFactory(func() (update.Verifier, error) {
		close(started)
		<-release
		return nil, wantErr
	})

	var got error
	u.StartManual(context.Background(), func() bool { return false }, "v0.2.5", Events{
		Failed: func(err error) { got = err },
	})
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("verifier preparation never started")
	}
	if !u.Done().Begun() {
		t.Fatal("manual verifier preparation is not tracked by Done")
	}
	close(release)
	waitUpdater(t, u)
	if !errors.Is(got, wantErr) {
		t.Errorf("Failed error = %v, want %v", got, wantErr)
	}
}

func TestUpdater_SettleWaitsForSupersededWorker(t *testing.T) {
	asset := updaterAssetName(t)
	srv := updaterReleaseServer(t, "v0.2.5", asset, nil, strings.Repeat("0", 64), nil)
	u := New(test.NewApp(), t.TempDir(), nil)
	u.SetClient(updaterClient(u, srv, fakeVerifier{}, time.Date(2026, 8, 30, 10, 0, 0, 0, time.Local)))

	staleEntered := make(chan struct{})
	releaseStale := make(chan struct{})
	var once sync.Once
	u.StartManual(context.Background(), func() bool {
		once.Do(func() { close(staleEntered) })
		<-releaseStale
		return true
	}, "v0.2.5", Events{})
	first := u.Done().Current()
	select {
	case <-staleEntered:
	case <-time.After(5 * time.Second):
		t.Fatal("first worker did not reach staleness check")
	}

	if err := u.Start(context.Background(), func() bool { return false }, "v0.2.5"); err != nil {
		t.Fatal(err)
	}
	latest := u.Done().Current()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := latest.Wait(ctx); err != nil {
		t.Fatal("latest worker did not finish")
	}

	settled := make(chan struct{})
	go func() {
		_ = u.Settle(context.Background())
		close(settled)
	}()
	select {
	case <-settled:
		t.Fatal("Settle returned while superseded worker was still running")
	default:
	}
	close(releaseStale)
	if err := first.Wait(ctx); err != nil {
		t.Fatal("superseded worker did not finish")
	}
	select {
	case <-settled:
	case <-ctx.Done():
		t.Fatal("Settle did not return after all workers finished")
	}
}

func TestUpdater_AutomaticAndManualShareCompleteTransaction(t *testing.T) {
	asset := updaterAssetName(t)
	archive := updaterNativeArchive(t, asset)
	sum := sha256.Sum256(archive)
	archiveCalls := 0
	srv := updaterReleaseServer(t, "v0.2.6", asset, archive, hex.EncodeToString(sum[:]), &archiveCalls)
	verifier := &blockingVerifier{entered: make(chan struct{}), release: make(chan struct{})}
	u := New(test.NewApp(), t.TempDir(), nil)
	u.SetClient(updaterClient(u, srv, verifier, time.Date(2026, 8, 30, 10, 0, 0, 0, time.Local)))

	if err := u.Start(context.Background(), func() bool { return false }, "v0.2.5"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-verifier.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("automatic request did not reach attestation verification")
	}
	if got := len(u.transaction); got != 0 {
		t.Fatalf("transaction token available during verification (len=%d), want held", got)
	}

	var manualReady bool
	u.StartManual(context.Background(), func() bool { return false }, "v0.2.5", Events{
		Ready: func(update.Stage) { manualReady = true },
	})
	close(verifier.release)
	waitUpdater(t, u)

	if archiveCalls != 1 {
		t.Errorf("archive downloads = %d, want one download followed by matching-stage reuse", archiveCalls)
	}
	if !manualReady {
		t.Error("manual request did not report the stage written by the serialized automatic request")
	}
	if got := len(u.transaction); got != 1 {
		t.Errorf("transaction token count after settle = %d, want 1", got)
	}
}
