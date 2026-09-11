package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/frathe/picfetch/internal/distribution"
	"github.com/frathe/picfetch/internal/similarity"
)

type assetTransport func(*http.Request) (*http.Response, error)

func (f assetTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestAssetInstall(t *testing.T) {
	t.Run("windows_policy", func(t *testing.T) {
		if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
			t.Skip("Windows x64/ARM64 policy")
		}
		if !similarity.SupportedPlatform() || similarity.EnforcesNetworkIsolation() {
			t.Fatal("Windows must support local analysis without claiming OS network isolation")
		}
		if err := similarity.VerifyOffline(context.Background()); err == nil || !strings.Contains(err.Error(), "does not enforce") {
			t.Fatalf("Windows must explicitly reject an OS-isolation qualification: %v", err)
		}
	})
	t.Run("ubuntu_supported", func(t *testing.T) {
		if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
			t.Skip("Linux amd64/arm64 support")
		}
		if !similarity.SupportedPlatform() {
			t.Fatal("Linux amd64/arm64 cannot install or run Explorer")
		}
	})
	t.Run("worker_reaches_asset_check_and_exits", func(t *testing.T) {
		if !similarity.SupportedPlatform() {
			t.Skip("requires a supported analysis worker")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := (similarity.Client{Assets: t.TempDir()}).Analyze(ctx, nil, nil, func(_ similarity.Event) {})
		if err == nil || !strings.Contains(err.Error(), "assets:") {
			t.Fatalf("worker must reject missing assets and exit with open controls: %v", err)
		}
	})
	t.Run("cancelled_before_start", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		client := similarity.Client{Assets: t.TempDir(), HTTPClient: &http.Client{Transport: assetTransport(func(_ *http.Request) (*http.Response, error) {
			t.Error("cancelled setup made a request")
			return nil, errors.New("unexpected request")
		})}}
		if _, err := client.InstallAssets(ctx, nil); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled setup: %v", err)
		}
	})
	if !similarity.AssetPlatformSupported() {
		t.Skip("downloads require a supported native runtime")
	}
	if distribution.StoreManaged {
		executable, err := os.Executable()
		if err != nil {
			t.Fatal(err)
		}
		libraries, err := filepath.Glob(filepath.Join(filepath.Dir(executable), "onnxruntime-*", "lib", "onnxruntime.dll"))
		if err != nil {
			t.Fatal(err)
		}
		if len(libraries) == 0 {
			t.Skip("Store HTTP fixtures require a test EXE staged with scripts/msixstage; missing-runtime rejection is covered by similarity.TestStoreDownloadPolicy")
		}
	}
	t.Run("rejects_untrusted_redirect", func(t *testing.T) {
		requests := 0
		directory := filepath.Join(t.TempDir(), "assets")
		client := similarity.Client{Assets: directory, HTTPClient: &http.Client{Transport: assetTransport(func(_ *http.Request) (*http.Response, error) {
			requests++
			return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": {"https://untrusted.example/model"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
		})}}
		if _, err := client.InstallAssets(context.Background(), nil); err == nil || requests != 1 {
			t.Fatalf("untrusted redirect was followed: requests=%d, err=%v", requests, err)
		}
	})
	t.Run("failure_preserves_existing_files", func(t *testing.T) {
		directory := t.TempDir()
		original := filepath.Join(directory, "vision_model.onnx")
		if err := os.WriteFile(original, []byte("existing model"), 0600); err != nil {
			t.Fatal(err)
		}
		client := similarity.Client{Assets: directory, HTTPClient: &http.Client{Transport: assetTransport(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusServiceUnavailable, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(""))}, nil
		})}}
		if _, err := client.InstallAssets(context.Background(), nil); err == nil {
			t.Fatal("unavailable service was accepted")
		}
		got, err := os.ReadFile(original)
		if err != nil || string(got) != "existing model" {
			t.Fatalf("failed repair damaged existing file: %q, %v", got, err)
		}
	})
	t.Run("rejects_modified_download", func(t *testing.T) {
		directory := filepath.Join(t.TempDir(), "model")
		requests := 0
		client := similarity.Client{Assets: directory, HTTPClient: &http.Client{Transport: assetTransport(func(request *http.Request) (*http.Response, error) {
			requests++
			if request.Method != http.MethodGet || request.URL.Scheme != "https" || request.Body != nil {
				t.Error("asset request must be a public HTTPS GET without an upload")
			}
			if request.Header.Get("Cookie") != "" || request.Header.Get("Authorization") != "" || request.URL.RawQuery != "" {
				t.Error("asset request carries unexpected account or query data")
			}
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("modified model")), ContentLength: -1, Header: make(http.Header)}, nil
		})}}
		if _, err := client.InstallAssets(context.Background(), nil); err == nil {
			t.Fatal("modified download was accepted as an installed model")
		}
		if requests != 1 {
			t.Fatalf("received %d requests, want one rejected download", requests)
		}
		if _, err := os.Stat(filepath.Join(directory, "vision_model.onnx")); !os.IsNotExist(err) {
			t.Fatalf("rejected model became visible at its installed path: %v", err)
		}
		entries, err := os.ReadDir(filepath.Dir(directory))
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 0 {
			t.Fatal("rejected download left installation or staging files behind")
		}
	})
}
