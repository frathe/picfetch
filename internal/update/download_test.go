package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
