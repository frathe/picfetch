//go:build explorerinstall

package main

import (
	"context"
	"fmt"
	"image/color"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/frathe/picfetch/internal/similarity"
	"github.com/frathe/picfetch/internal/uitest"
)

// This explicit network qualification downloads the actual pinned public files.
// It is separate from both the ordinary suite and the offline model suite.
func TestRealAssetInstall(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	root := t.TempDir()
	client := similarity.Client{Assets: filepath.Join(root, "assets")}
	var received, total int64
	last := time.Time{}
	directory, err := client.InstallAssets(ctx, func(p similarity.DownloadProgress) {
		if p.Received < received || p.Received > p.Total {
			t.Errorf("invalid download progress: %+v", p)
		}
		received, total = p.Received, p.Total
		if time.Since(last) > 10*time.Second || received == total {
			t.Logf("downloaded %.1f of %.1f MB", float64(received)/1e6, float64(total)/1e6)
			last = time.Now()
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	if directory != client.Assets || received != total || received != 413387010 {
		t.Fatalf("installation incomplete: directory=%q, received=%d, total=%d", directory, received, total)
	}
	client.HTTPClient = &http.Client{Transport: assetTransport(func(_ *http.Request) (*http.Response, error) {
		t.Error("verified installation made another request")
		return nil, fmt.Errorf("unexpected network request")
	})}
	if _, err := client.InstallAssets(ctx, nil); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"LICENSE", "ThirdPartyNotices.txt"} {
		data, err := os.ReadFile(filepath.Join(directory, "onnxruntime-osx-arm64-1.29.0", name))
		if err != nil || len(data) == 0 {
			t.Fatalf("runtime notice missing: %s, %v", name, err)
		}
	}
	path := filepath.Join(root, "synthetic.png")
	if err := os.WriteFile(path, uitest.EncodePNG(t, 96, 64, color.White), 0600); err != nil {
		t.Fatal(err)
	}
	var result similarity.Event
	if err := client.Analyze(ctx, []string{path}, nil, func(e similarity.Event) {
		if e.Complete {
			result = e
		}
	}); err != nil {
		t.Fatal(err)
	}
	if !result.OfflineVerified || result.Successful != 1 || result.Failed != 0 || len(result.Items) != 1 {
		t.Fatalf("installed model did not complete offline analysis: %+v", result)
	}
	t.Log("verified installation reused without HTTP; installed runtime analyzed the synthetic image under OS network denial")
}
