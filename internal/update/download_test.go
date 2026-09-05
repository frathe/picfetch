package update

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type downloadDoerFunc func(*http.Request) (*http.Response, error)

func (f downloadDoerFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

func serveDownload(t *testing.T, name string, archive []byte, attest http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/"+name {
			_, _ = w.Write(archive)
			return
		}
		if attest != nil && strings.Contains(r.URL.Path, "/attestations/") {
			attest(w, r)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func downloadClient(t *testing.T, srv *httptest.Server, verify Verifier, stageDir string) *Client {
	t.Helper()
	return NewClient(Config{
		BaseURL:  srv.URL,
		HTTP:     srv.Client(),
		Verify:   verify,
		StageDir: stageDir,
		GOOS:     "linux",
		GOARCH:   "amd64",
	})
}

func okAttest(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(attestationsJSON([]byte(`{"mediaType":"test-bundle"}`)))
}

func linuxArchive(t *testing.T) []byte {
	t.Helper()
	path := writeTarGz(t, t.TempDir(), "picfetch-linux-amd64.tar.gz", map[string][]byte{
		"picfetch-linux-amd64": []byte("elf"),
	})
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestDownload(t *testing.T) {
	archive := linuxArchive(t)
	sum := sha256.Sum256(archive)
	digest := hex.EncodeToString(sum[:])
	srv := serveDownload(t, "picfetch-linux-amd64.tar.gz", archive, okAttest)
	stageDir := t.TempDir()
	c := downloadClient(t, srv, &fakeVerifier{}, stageDir)
	rel := Release{
		Version:     "v0.2.6",
		Notes:       "## Fixes\n\n- toast",
		AssetName:   "picfetch-linux-amd64.tar.gz",
		AssetURL:    srv.URL + "/picfetch-linux-amd64.tar.gz",
		AssetDigest: digest,
	}
	st, err := c.Download(context.Background(), rel)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(stageDir, "stage.json")); err != nil {
		t.Fatalf("stage.json: %v", err)
	}
	if _, err := os.Stat(st.BinaryPath); err != nil {
		t.Fatalf("extracted binary: %v", err)
	}
	loaded, err := LoadStage(stageDir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Version != "v0.2.6" {
		t.Errorf("Version = %q", loaded.Version)
	}
	if loaded.Notes != rel.Notes {
		t.Errorf("Notes = %q", loaded.Notes)
	}
	if loaded.BinaryPath != st.BinaryPath {
		t.Errorf("BinaryPath = %q, want %q", loaded.BinaryPath, st.BinaryPath)
	}
	if !filepath.IsAbs(loaded.BinaryPath) {
		t.Errorf("BinaryPath is not absolute: %q", loaded.BinaryPath)
	}
	if err := ValidateStage(loaded); err != nil {
		t.Fatalf("downloaded stage provenance = %v", err)
	}
	if !StageMatchesRelease(loaded, rel) {
		t.Fatal("downloaded stage does not match its verified release")
	}
	if err := ValidateStageForPlatform(loaded, "linux", "amd64"); err != nil {
		t.Fatalf("downloaded stage platform = %v", err)
	}
	if err := ValidateStageForPlatform(loaded, "linux", "arm64"); err == nil {
		t.Fatal("downloaded amd64 stage unexpectedly validated for arm64")
	}
}

func TestValidateStage_RejectsUnverifiedAndTamperedStages(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "picfetch")
	if err := os.WriteFile(bin, []byte("forged"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := SaveStage(dir, Stage{Version: "v0.2.6", BinaryPath: bin}); err != nil {
		t.Fatal(err)
	}
	forged, err := LoadStage(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateStage(forged); err == nil {
		t.Fatal("manually forged stage unexpectedly has verified provenance")
	}

	archive := linuxArchive(t)
	sum := sha256.Sum256(archive)
	digest := hex.EncodeToString(sum[:])
	srv := serveDownload(t, "picfetch-linux-amd64.tar.gz", archive, okAttest)
	rel := Release{
		Version:     "v0.2.6",
		AssetName:   "picfetch-linux-amd64.tar.gz",
		AssetURL:    srv.URL + "/picfetch-linux-amd64.tar.gz",
		AssetDigest: digest,
	}
	verified, err := downloadClient(t, srv, &fakeVerifier{}, dir).Download(context.Background(), rel)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(verified.BinaryPath, []byte("tampered"), 0o755); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadStage(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateStage(loaded); err == nil {
		t.Fatal("tampered staged binary unexpectedly passed verification")
	}
	if StageMatchesRelease(loaded, rel) {
		t.Fatal("tampered staged binary unexpectedly matched release")
	}
}

func TestStageMatchesRelease_RejectsDifferentVerifiedAssetIdentity(t *testing.T) {
	archive := linuxArchive(t)
	sum := sha256.Sum256(archive)
	digest := hex.EncodeToString(sum[:])
	srv := serveDownload(t, "picfetch-linux-amd64.tar.gz", archive, okAttest)
	dir := t.TempDir()
	rel := Release{
		Version:     "v0.2.6",
		AssetName:   "picfetch-linux-amd64.tar.gz",
		AssetURL:    srv.URL + "/picfetch-linux-amd64.tar.gz",
		AssetDigest: digest,
	}
	st, err := downloadClient(t, srv, &fakeVerifier{}, dir).Download(context.Background(), rel)
	if err != nil {
		t.Fatal(err)
	}

	for _, changed := range []Release{
		{Version: "v0.2.7", AssetName: rel.AssetName, AssetDigest: rel.AssetDigest},
		{Version: rel.Version, AssetName: "other-asset.zip", AssetDigest: rel.AssetDigest},
		{Version: rel.Version, AssetName: rel.AssetName, AssetDigest: strings.Repeat("f", 64)},
	} {
		if StageMatchesRelease(st, changed) {
			t.Errorf("stage unexpectedly matched changed release %+v", changed)
		}
	}
}

func TestValidateStage_RejectsTamperedPlistAfterProvenanceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "picfetch")
	plist := filepath.Join(dir, "Info.plist")
	if err := os.WriteFile(bin, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(plist, []byte("plist"), 0o644); err != nil {
		t.Fatal(err)
	}
	binDigest, err := fileSHA256(bin)
	if err != nil {
		t.Fatal(err)
	}
	plistDigest, err := fileSHA256(plist)
	if err != nil {
		t.Fatal(err)
	}
	st := Stage{
		Version:    "v0.2.6",
		BinaryPath: bin,
		PlistPath:  plist,
		verification: stageVerification{
			AssetName:     "picfetch-macos-arm64.zip",
			ArchiveDigest: strings.Repeat("a", 64),
			BinaryDigest:  binDigest,
			PlistDigest:   plistDigest,
			GOOS:          "darwin",
			GOARCH:        "arm64",
		},
	}
	if err := SaveStage(dir, st); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadStage(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateStage(loaded); err != nil {
		t.Fatalf("untampered plist stage = %v", err)
	}
	if err := os.WriteFile(plist, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ValidateStage(loaded); err == nil {
		t.Fatal("tampered staged plist unexpectedly passed verification")
	}
}

func TestDownloadWithProgress_KnownSize(t *testing.T) {
	archive := linuxArchive(t)
	client := downloadProgressClient(t, archive, int64(len(archive)))
	var events []DownloadProgress
	_, err := client.DownloadWithProgress(context.Background(), progressRelease(archive), func(progress DownloadProgress) {
		events = append(events, progress)
	})
	if err != nil {
		t.Fatal(err)
	}
	assertProgressEvents(t, events, int64(len(archive)), int64(len(archive)))
	if len(events) > 101 {
		t.Fatalf("got %d progress events, want at most one per percentage plus initial/final", len(events))
	}
	for i, event := range events[:len(events)-1] {
		if event.Downloaded >= event.Total {
			t.Fatalf("event %d claims completion before terminal event: %+v", i, event)
		}
	}
}

func TestDownloadWithProgress_UnknownSize(t *testing.T) {
	archive := linuxArchive(t)
	client := downloadProgressClient(t, archive, -1)
	var events []DownloadProgress
	_, err := client.DownloadWithProgress(context.Background(), progressRelease(archive), func(progress DownloadProgress) {
		events = append(events, progress)
	})
	if err != nil {
		t.Fatal(err)
	}
	assertProgressEvents(t, events, int64(len(archive)), -1)
	if len(events) != 2 {
		t.Fatalf("events = %+v, want initial and exact terminal for sub-step archive", events)
	}
}

func TestReadArchive_UnknownSizeByteStepCoalescing(t *testing.T) {
	archive := make([]byte, 2*unknownProgressByteStep+123)
	var events []DownloadProgress
	data, err := readArchive(bytes.NewReader(archive), -1, int64(len(archive)), func(progress DownloadProgress) {
		events = append(events, progress)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != len(archive) {
		t.Fatalf("read %d bytes, want %d", len(data), len(archive))
	}
	want := []DownloadProgress{
		{Downloaded: 0, Total: -1},
		{Downloaded: unknownProgressByteStep, Total: -1},
		{Downloaded: 2 * unknownProgressByteStep, Total: -1},
		{Downloaded: int64(len(archive)), Total: -1},
	}
	if len(events) != len(want) {
		t.Fatalf("events = %+v, want %+v", events, want)
	}
	for i := range want {
		if events[i] != want[i] {
			t.Errorf("event %d = %+v, want %+v", i, events[i], want[i])
		}
	}
}

func TestReadArchive_KnownSizePercentageCoalescing(t *testing.T) {
	archive := make([]byte, 1000)
	var events []DownloadProgress
	if _, err := readArchive(bytes.NewReader(archive), int64(len(archive)), int64(len(archive)), func(progress DownloadProgress) {
		events = append(events, progress)
	}); err != nil {
		t.Fatal(err)
	}
	if len(events) != 101 {
		t.Fatalf("got %d events, want initial, 99 intermediate percentages, and terminal", len(events))
	}
	for percent := range 100 {
		want := DownloadProgress{Downloaded: int64(percent * 10), Total: 1000}
		if events[percent] != want {
			t.Errorf("event %d = %+v, want %+v", percent, events[percent], want)
		}
	}
	wantFinal := DownloadProgress{Downloaded: 1000, Total: 1000}
	if events[100] != wantFinal {
		t.Errorf("terminal event = %+v, want %+v", events[100], wantFinal)
	}
}

func TestReadArchive_ShortKnownResponseDoesNotClaimCompletion(t *testing.T) {
	var events []DownloadProgress
	if _, err := readArchive(strings.NewReader("short"), 10, 10, func(progress DownloadProgress) {
		events = append(events, progress)
	}); err != nil {
		t.Fatal(err)
	}
	assertProgressEvents(t, events, 5, 10)
	for i, event := range events {
		if event.Downloaded >= event.Total {
			t.Fatalf("event %d claims a complete declared response: %+v", i, event)
		}
	}
}

func TestReadArchive_EmptyHasInitialAndTerminal(t *testing.T) {
	var events []DownloadProgress
	if _, err := readArchive(bytes.NewReader(nil), 0, 1, func(progress DownloadProgress) {
		events = append(events, progress)
	}); err != nil {
		t.Fatal(err)
	}
	want := DownloadProgress{Downloaded: 0, Total: 0}
	if len(events) != 2 || events[0] != want || events[1] != want {
		t.Fatalf("events = %+v, want initial and terminal %+v", events, want)
	}
}

func TestReadArchive_NilProgress(t *testing.T) {
	got, err := readArchive(strings.NewReader("archive"), 7, 7, nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "archive" {
		t.Fatalf("data = %q", got)
	}
}

func TestDownloadWithProgress_CancellationHasNoTerminal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	body := &cancelDownloadReader{ctx: ctx, cancel: cancel}
	stageDir := t.TempDir()
	client := NewClient(Config{
		BaseURL: "https://api.example.test",
		HTTP: downloadDoerFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode:    http.StatusOK,
				Status:        "200 OK",
				Body:          body,
				ContentLength: 10,
			}, nil
		}),
		Verify:   &fakeVerifier{},
		StageDir: stageDir,
		GOOS:     "linux",
		GOARCH:   "amd64",
	})
	var events []DownloadProgress
	_, err := client.DownloadWithProgress(ctx, Release{
		Version:   "v0.2.6",
		AssetName: "picfetch-linux-amd64.tar.gz",
		AssetURL:  "https://download.example.test/archive",
	}, func(progress DownloadProgress) {
		events = append(events, progress)
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if len(events) < 2 {
		t.Fatalf("events = %+v, want initial and partial progress", events)
	}
	if got := events[len(events)-1].Downloaded; got >= 10 {
		t.Fatalf("last event = %+v, want no terminal event", events[len(events)-1])
	}
	assertNoStage(t, stageDir)
}

func TestReadArchive_OversizeHasNoTerminal(t *testing.T) {
	var events []DownloadProgress
	_, err := readArchive(strings.NewReader("123456"), -1, 5, func(progress DownloadProgress) {
		events = append(events, progress)
	})
	if err == nil || !strings.Contains(err.Error(), "archive exceeds 5 bytes") {
		t.Fatalf("err = %v, want archive limit error", err)
	}
	if len(events) != 1 || events[0] != (DownloadProgress{Downloaded: 0, Total: -1}) {
		t.Fatalf("events = %+v, want only initial event", events)
	}
}

// Release archives never carry links, so Download refuses any archive that
// does. Each fixture below is an escape a link-honouring extractor would
// complete: the tar symlink case has been observed writing "pwned" into the
// sibling directory, and the hard link case pulls a file from outside the
// stage into it.
func TestDownload_RefusesLinkArchiveEntries(t *testing.T) {
	const payloadName = "picfetch-linux-amd64"
	cases := []struct {
		name      string
		assetName string
		archive   func(t *testing.T, root string) []byte
	}{
		{
			name:      "zip symlink",
			assetName: payloadName + ".zip",
			archive: func(t *testing.T, _ string) []byte {
				return zipArchiveBytes(t, []archiveEntry{
					{name: "link", linkTarget: "../outside/pwned"},
					{name: payloadName, body: []byte("elf")},
				})
			},
		},
		{
			name:      "tar symlink",
			assetName: payloadName + ".tar.gz",
			archive: func(t *testing.T, _ string) []byte {
				return tarGzArchiveBytes(t, []archiveEntry{
					{name: "escape", linkTarget: "../outside"},
					{name: "escape/pwned", body: []byte("owned")},
					{name: payloadName, body: []byte("elf")},
				})
			},
		},
		{
			name:      "tar hard link",
			assetName: payloadName + ".tar.gz",
			archive: func(t *testing.T, root string) []byte {
				return tarGzArchiveBytes(t, []archiveEntry{
					{name: "stolen", linkTarget: filepath.Join(root, "outside", "secret"), hard: true},
					{name: payloadName, body: []byte("elf")},
				})
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			outside := filepath.Join(root, "outside")
			if err := os.MkdirAll(outside, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(outside, "secret"), []byte("secret"), 0o600); err != nil {
				t.Fatal(err)
			}
			stageDir := filepath.Join(root, "stage")

			archive := tc.archive(t, root)
			sum := sha256.Sum256(archive)
			srv := serveDownload(t, tc.assetName, archive, okAttest)
			c := downloadClient(t, srv, &fakeVerifier{}, stageDir)

			_, err = c.Download(context.Background(), Release{
				Version:     "v0.2.7",
				AssetName:   tc.assetName,
				AssetURL:    srv.URL + "/" + tc.assetName,
				AssetDigest: hex.EncodeToString(sum[:]),
			})
			if err == nil {
				t.Fatal("Download accepted an archive carrying a link entry")
			}
			if !strings.Contains(err.Error(), "refusing") {
				t.Fatalf("Download error = %v, want the link entry refused", err)
			}
			if _, statErr := os.Stat(stageDir); !errors.Is(statErr, os.ErrNotExist) {
				t.Errorf("stage directory survived a refused archive: %v", statErr)
			}
			if _, loadErr := LoadStage(stageDir); loadErr == nil {
				t.Error("a refused archive left a loadable stage")
			}
			entries, err := os.ReadDir(outside)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 || entries[0].Name() != "secret" {
				var names []string
				for _, e := range entries {
					names = append(names, e.Name())
				}
				t.Errorf("beside the stage directory = %v, want only the pre-existing secret", names)
			}
		})
	}
}

func TestDownload_WrongDigest(t *testing.T) {
	archive := linuxArchive(t)
	srv := serveDownload(t, "picfetch-linux-amd64.tar.gz", archive, func(w http.ResponseWriter, _ *http.Request) {
		t.Error("attestations must not be fetched when AssetDigest is wrong")
		http.Error(w, "no", http.StatusInternalServerError)
	})
	stageDir := t.TempDir()
	c := downloadClient(t, srv, &fakeVerifier{}, stageDir)
	_, err := c.Download(context.Background(), Release{
		Version:     "v0.2.6",
		AssetName:   "picfetch-linux-amd64.tar.gz",
		AssetURL:    srv.URL + "/picfetch-linux-amd64.tar.gz",
		AssetDigest: "0000000000000000000000000000000000000000000000000000000000000000",
	})
	if err == nil {
		t.Fatal("want error for wrong AssetDigest")
	}
	if _, err := os.Stat(filepath.Join(stageDir, "stage.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("want no usable stage.json, stat err %v", err)
	}
}

func TestDownload_EmptyDigest(t *testing.T) {
	archive := linuxArchive(t)
	srv := serveDownload(t, "picfetch-linux-amd64.tar.gz", archive, okAttest)
	stageDir := t.TempDir()
	c := downloadClient(t, srv, &fakeVerifier{}, stageDir)
	st, err := c.Download(context.Background(), Release{
		Version:     "v0.2.6",
		Notes:       "notes",
		AssetName:   "picfetch-linux-amd64.tar.gz",
		AssetURL:    srv.URL + "/picfetch-linux-amd64.tar.gz",
		AssetDigest: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadStage(stageDir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Version != st.Version || loaded.BinaryPath == "" {
		t.Fatalf("LoadStage = %+v", loaded)
	}
}

func TestLoadStage_Missing(t *testing.T) {
	_, err := LoadStage(t.TempDir())
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("err = %v, want os.ErrNotExist", err)
	}
}

func downloadProgressClient(t *testing.T, archive []byte, contentLength int64) *Client {
	t.Helper()
	client := NewClient(Config{
		BaseURL: "https://api.example.test",
		HTTP: downloadDoerFunc(func(req *http.Request) (*http.Response, error) {
			body := archive
			total := contentLength
			if strings.Contains(req.URL.Path, "/attestations/") {
				body = attestationsJSON([]byte(`{"mediaType":"test-bundle"}`))
				total = int64(len(body))
			}
			return &http.Response{
				StatusCode:    http.StatusOK,
				Status:        "200 OK",
				Body:          io.NopCloser(bytes.NewReader(body)),
				ContentLength: total,
			}, nil
		}),
		Verify:   &fakeVerifier{},
		StageDir: t.TempDir(),
		GOOS:     "linux",
		GOARCH:   "amd64",
	})
	return client
}

func progressRelease(archive []byte) Release {
	sum := sha256.Sum256(archive)
	return Release{
		Version:     "v0.2.6",
		AssetName:   "picfetch-linux-amd64.tar.gz",
		AssetURL:    "https://download.example.test/archive",
		AssetDigest: hex.EncodeToString(sum[:]),
	}
}

func assertProgressEvents(t *testing.T, events []DownloadProgress, downloaded, total int64) {
	t.Helper()
	if len(events) < 2 {
		t.Fatalf("events = %+v, want initial and terminal events", events)
	}
	if events[0] != (DownloadProgress{Downloaded: 0, Total: total}) {
		t.Errorf("initial event = %+v, want Downloaded 0 Total %d", events[0], total)
	}
	wantFinal := DownloadProgress{Downloaded: downloaded, Total: total}
	if events[len(events)-1] != wantFinal {
		t.Errorf("terminal event = %+v, want %+v", events[len(events)-1], wantFinal)
	}
	for i := 1; i < len(events); i++ {
		if events[i].Downloaded < events[i-1].Downloaded {
			t.Errorf("events not monotonic at %d: %+v then %+v", i, events[i-1], events[i])
		}
		if events[i].Total != total {
			t.Errorf("event %d Total = %d, want %d", i, events[i].Total, total)
		}
	}
}

type cancelDownloadReader struct {
	ctx    context.Context
	cancel context.CancelFunc
	read   bool
}

func (r *cancelDownloadReader) Read(p []byte) (int, error) {
	if r.read {
		return 0, r.ctx.Err()
	}
	r.read = true
	n := copy(p, "12345")
	r.cancel()
	return n, nil
}

func (r *cancelDownloadReader) Close() error {
	return nil
}
