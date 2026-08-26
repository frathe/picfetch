package update

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

const checkDigestHex = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func serveLatest(t *testing.T, mutate func(map[string]any), onReq func(*http.Request)) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if onReq != nil {
			onReq(r)
		}
		if r.URL.Path != "/repos/frathe/picfetch/releases/latest" {
			http.NotFound(w, r)
			return
		}
		body := map[string]any{
			"tag_name":   "v0.2.6",
			"body":       "## Fixes\n\n- toast",
			"draft":      false,
			"prerelease": false,
			"assets": []any{
				map[string]any{
					"name":                 "picfetch-linux-amd64.tar.gz",
					"browser_download_url": srv.URL + "/files/picfetch-linux-amd64.tar.gz",
					"digest":               "sha256:" + checkDigestHex,
				},
			},
		}
		if mutate != nil {
			mutate(body)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func checkClient(t *testing.T, srv *httptest.Server, goos, goarch, current string) (*Release, error) {
	t.Helper()
	c := NewClient(Config{
		BaseURL: srv.URL,
		HTTP:    srv.Client(),
		GOOS:    goos,
		GOARCH:  goarch,
	})
	return c.Check(context.Background(), current)
}

func TestCheck(t *testing.T) {
	var archiveGets atomic.Int32
	var gotReq *http.Request
	srv := serveLatest(t, nil, func(r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".tar.gz") {
			archiveGets.Add(1)
		}
		if r.URL.Path == "/repos/frathe/picfetch/releases/latest" {
			gotReq = r.Clone(r.Context())
		}
	})

	rel, err := checkClient(t, srv, "linux", "amd64", "0.2.5")
	if err != nil {
		t.Fatal(err)
	}
	if rel == nil {
		t.Fatal("want a Release")
	}
	if rel.Version != "v0.2.6" {
		t.Errorf("Version = %q, want v0.2.6", rel.Version)
	}
	if !strings.Contains(rel.Notes, "toast") {
		t.Errorf("Notes = %q, want to contain toast", rel.Notes)
	}
	if rel.AssetName != "picfetch-linux-amd64.tar.gz" {
		t.Errorf("AssetName = %q, want picfetch-linux-amd64.tar.gz", rel.AssetName)
	}
	if rel.AssetDigest != checkDigestHex {
		t.Errorf("AssetDigest = %q, want hex without sha256: prefix", rel.AssetDigest)
	}
	if !strings.HasSuffix(rel.AssetURL, "/files/picfetch-linux-amd64.tar.gz") {
		t.Errorf("AssetURL = %q, want download URL on the test server", rel.AssetURL)
	}
	if archiveGets.Load() != 0 {
		t.Fatal("Check must not download the archive")
	}
	if gotReq == nil {
		t.Fatal("no request to /releases/latest")
	}
	if got := gotReq.Header.Get("Accept"); got != "application/vnd.github+json" {
		t.Errorf("Accept = %q", got)
	}
	if got := gotReq.Header.Get("User-Agent"); got != "picfetch/0.2.5" {
		t.Errorf("User-Agent = %q, want picfetch/0.2.5", got)
	}
	if got := gotReq.Header.Get("X-GitHub-Api-Version"); got != "2022-11-28" {
		t.Errorf("X-GitHub-Api-Version = %q", got)
	}
	if got := gotReq.Header.Get("Authorization"); got != "" {
		t.Errorf("Authorization = %q, want empty", got)
	}
}

func TestCheck_SameTag(t *testing.T) {
	srv := serveLatest(t, nil, nil)
	rel, err := checkClient(t, srv, "linux", "amd64", "0.2.6")
	if err != nil {
		t.Fatal(err)
	}
	if rel != nil {
		t.Fatalf("want nil Release for same tag, got %+v", rel)
	}
}

func TestCheck_UnsupportedGOOS(t *testing.T) {
	srv := serveLatest(t, nil, nil)
	rel, err := checkClient(t, srv, "freebsd", "amd64", "0.2.5")
	if err != nil {
		t.Fatal(err)
	}
	if rel != nil {
		t.Fatalf("want nil Release for unsupported GOOS, got %+v", rel)
	}
}

func TestCheck_Draft(t *testing.T) {
	srv := serveLatest(t, func(body map[string]any) {
		body["draft"] = true
	}, nil)
	rel, err := checkClient(t, srv, "linux", "amd64", "0.2.5")
	if err != nil {
		t.Fatal(err)
	}
	if rel != nil {
		t.Fatalf("want nil Release for draft, got %+v", rel)
	}
}

func TestCheck_Prerelease(t *testing.T) {
	srv := serveLatest(t, func(body map[string]any) {
		body["prerelease"] = true
	}, nil)
	rel, err := checkClient(t, srv, "linux", "amd64", "0.2.5")
	if err != nil {
		t.Fatal(err)
	}
	if rel != nil {
		t.Fatalf("want nil Release for prerelease, got %+v", rel)
	}
}

func TestCheck_MissingAsset(t *testing.T) {
	srv := serveLatest(t, func(body map[string]any) {
		body["assets"] = []any{
			map[string]any{
				"name":                 "picfetch-windows-amd64.zip",
				"browser_download_url": "http://example.invalid/windows.zip",
				"digest":               "sha256:" + checkDigestHex,
			},
		}
	}, nil)
	rel, err := checkClient(t, srv, "linux", "amd64", "0.2.5")
	if err == nil {
		t.Fatalf("want error for missing platform asset, got %+v", rel)
	}
}

func TestCheck_EmptyDigest(t *testing.T) {
	srv := serveLatest(t, func(body map[string]any) {
		assets := body["assets"].([]any)
		a := assets[0].(map[string]any)
		a["digest"] = ""
	}, nil)
	rel, err := checkClient(t, srv, "linux", "amd64", "0.2.5")
	if err != nil {
		t.Fatal(err)
	}
	if rel == nil {
		t.Fatal("want a Release when digest is empty")
	}
	if rel.AssetDigest != "" {
		t.Errorf("AssetDigest = %q, want empty", rel.AssetDigest)
	}
}

func TestCheck_MissingDigest(t *testing.T) {
	srv := serveLatest(t, func(body map[string]any) {
		assets := body["assets"].([]any)
		a := assets[0].(map[string]any)
		delete(a, "digest")
	}, nil)
	rel, err := checkClient(t, srv, "linux", "amd64", "0.2.5")
	if err != nil {
		t.Fatal(err)
	}
	if rel == nil {
		t.Fatal("want a Release when digest is omitted")
	}
	if rel.AssetDigest != "" {
		t.Errorf("AssetDigest = %q, want empty", rel.AssetDigest)
	}
}

func TestCheck_InvalidDigest(t *testing.T) {
	for _, digest := range []string{
		"sha256:deadbeef",
		"not-a-digest",
		checkDigestHex,
		"md5:" + checkDigestHex,
		"sha256:" + checkDigestHex + "ff",
	} {
		t.Run(digest, func(t *testing.T) {
			srv := serveLatest(t, func(body map[string]any) {
				assets := body["assets"].([]any)
				a := assets[0].(map[string]any)
				a["digest"] = digest
			}, nil)
			rel, err := checkClient(t, srv, "linux", "amd64", "0.2.5")
			if err == nil {
				t.Fatalf("want error for digest %q, got %+v", digest, rel)
			}
		})
	}
}
