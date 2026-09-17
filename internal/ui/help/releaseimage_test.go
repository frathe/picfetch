package help

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/frathe/picfetch/internal/uitest"
)

func TestLoadReleaseImageFormatsAndURLs(t *testing.T) {
	const query = "raw=true&caption=release%20art"
	for _, tc := range []struct {
		name     string
		data     []byte
		tls      bool
		redirect bool
	}{
		{name: "PNG with GitHub raw query", data: uitest.EncodePNG(t, 3, 2, color.White)},
		{name: "JPEG after redirect", data: uitest.EncodeJPEG(t, 3, 2, color.White), redirect: true},
		{name: "GIF first frame", data: uitest.EncodeAnimatedGIF(t, 3, 2, []color.Color{color.White, color.Black}, []int{10, 10})},
		{name: "WebP over HTTPS", data: mascotWagsWebP, tls: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.RawQuery != query {
					http.Error(w, "request method or query changed", http.StatusBadRequest)
					return
				}
				if tc.redirect && r.URL.Path == "/blob/main/art" {
					http.Redirect(w, r, "/raw/art?"+query, http.StatusFound)
					return
				}
				_, _ = w.Write(tc.data)
			})
			var server *httptest.Server
			if tc.tls {
				server = httptest.NewTLSServer(handler)
			} else {
				server = httptest.NewServer(handler)
			}
			defer server.Close()
			got, err := loadReleaseImage(context.Background(), server.Client(), server.URL+"/blob/main/art?"+query)
			if err != nil {
				t.Fatal(err)
			}
			config, _, err := image.DecodeConfig(bytes.NewReader(tc.data))
			if err != nil {
				t.Fatal(err)
			}
			if want := image.Rect(0, 0, config.Width, config.Height); got.Bounds() != want {
				t.Fatalf("bounds = %v, want %v", got.Bounds(), want)
			}
		})
	}
}

func TestLoadReleaseImageRejectsInvalidResponses(t *testing.T) {
	for _, tc := range []struct {
		name    string
		status  int
		data    []byte
		wantErr string
	}{
		{name: "missing image", status: http.StatusNotFound, data: []byte("missing"), wantErr: "404"},
		{name: "redirect without location", status: http.StatusFound, wantErr: "302"},
		{name: "empty image", status: http.StatusNoContent, wantErr: "image"},
		{name: "not an image", status: http.StatusOK, data: []byte("<html>not an image</html>"), wantErr: "image"},
		{name: "incomplete image", status: http.StatusOK, data: uitest.TruncatedPNGHeader(t, 3, 2), wantErr: "decode"},
		{name: "pixel limit", status: http.StatusOK, data: uitest.TruncatedPNGHeader(t, 4097, 4096), wantErr: "dimensions"},
		{name: "large dimension product", status: http.StatusOK, data: uitest.TruncatedPNGHeader(t, 1<<30, 1<<30), wantErr: "dimension"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write(tc.data)
			}))
			defer server.Close()
			got, err := loadReleaseImage(context.Background(), server.Client(), server.URL)
			if got != nil || err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("loadReleaseImage = %v, %v; want nil image and error containing %q", got, err, tc.wantErr)
			}
		})
	}
}

func TestLoadReleaseImageEncodedSizeLimit(t *testing.T) {
	const limit = 8 * 1024 * 1024
	for _, tc := range []struct {
		name string
		size int
	}{
		{name: "at limit", size: limit},
		{name: "over limit", size: limit + 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := make([]byte, tc.size)
			copy(data, uitest.EncodePNG(t, 3, 2, color.White))
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				// Flush before writing so the response has no Content-Length.
				w.WriteHeader(http.StatusOK)
				w.(http.Flusher).Flush()
				_, _ = w.Write(data)
			}))
			defer server.Close()
			got, err := loadReleaseImage(context.Background(), server.Client(), server.URL)
			if tc.size > limit {
				if got != nil || err == nil || !strings.Contains(err.Error(), "size limit") {
					t.Fatalf("over-limit response = %v, %v; want size limit error", got, err)
				}
			} else if err != nil || got == nil {
				t.Fatalf("at-limit response = %v, %v; want decoded image", got, err)
			}
		})
	}
}

func TestLoadReleaseImageRejectsNonHTTPURLs(t *testing.T) {
	for _, rawURL := range []string{"file:///tmp/image.png", "data:image/png;base64,AAAA", "ftp://example.invalid/image.png", "relative.png", "://invalid"} {
		t.Run(rawURL, func(t *testing.T) {
			got, err := loadReleaseImage(context.Background(), &http.Client{}, rawURL)
			if got != nil || err == nil {
				t.Fatalf("loadReleaseImage(%q) = %v, %v; want rejection", rawURL, got, err)
			}
		})
	}
}

func TestLoadReleaseImageCancellation(t *testing.T) {
	for _, stage := range []string{"before request", "waiting for headers", "reading body"} {
		t.Run(stage, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			started := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if stage == "reading body" {
					w.WriteHeader(http.StatusOK)
					w.(http.Flusher).Flush()
				}
				close(started)
				<-r.Context().Done()
			}))
			defer server.Close()
			if stage == "before request" {
				cancel()
			}
			done := make(chan error, 1)
			go func() {
				_, err := loadReleaseImage(ctx, server.Client(), server.URL)
				done <- err
			}()
			if stage != "before request" {
				select {
				case <-started:
				case <-time.After(10 * time.Second):
					cancel()
					t.Fatal("image request did not reach server")
				}
				cancel()
			}
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("error = %v, want context.Canceled", err)
				}
			case <-time.After(10 * time.Second):
				t.Fatal("image request did not finish after cancellation")
			}
		})
	}
}
