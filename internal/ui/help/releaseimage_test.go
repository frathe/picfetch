package help

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/frathe/picfetch/internal/uitest"
)

const testReleaseImageURL = "https://github.com/frathe/picfetch/release-image"

func releaseImageTestClient(t *testing.T, server *httptest.Server) *http.Client {
	t.Helper()
	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	client := server.Client()
	transport := client.Transport.(*http.Transport).Clone()
	transport.TLSClientConfig.ServerName = serverURL.Hostname()
	transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, serverURL.Host)
	}
	client.Transport = transport
	t.Cleanup(client.CloseIdleConnections)
	return client
}

func TestLoadReleaseImageFormatsAndURLs(t *testing.T) {
	const query = "raw=true&caption=release%20art"
	for _, tc := range []struct {
		name     string
		data     []byte
		redirect bool
	}{
		{name: "PNG with GitHub raw query", data: uitest.EncodePNG(t, 3, 2, color.White)},
		{name: "JPEG after redirect", data: uitest.EncodeJPEG(t, 3, 2, color.White), redirect: true},
		{name: "GIF first frame", data: uitest.EncodeAnimatedGIF(t, 3, 2, []color.Color{color.White, color.Black}, []int{10, 10})},
		{name: "WebP over HTTPS", data: mascotWagsWebP},
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
			server := httptest.NewTLSServer(handler)
			defer server.Close()
			got, err := loadReleaseImage(context.Background(), releaseImageTestClient(t, server), "https://github.com/blob/main/art?"+query)
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

func TestLoadReleaseImageRejectsUntrustedDestinations(t *testing.T) {
	data := releaseTestPNG(t)
	for _, rawURL := range untrustedReleaseImageURLs() {
		t.Run(rawURL, func(t *testing.T) {
			requests := 0
			client := &http.Client{Transport: releaseImageTransport(func(request *http.Request) (*http.Response, error) {
				requests++
				return &http.Response{StatusCode: http.StatusOK, Request: request, Body: io.NopCloser(bytes.NewReader(data))}, nil
			})}
			got, err := loadReleaseImage(context.Background(), client, rawURL)
			if got != nil || err == nil {
				t.Fatalf("loadReleaseImage(%q) = %v, %v; want rejection", rawURL, got, err)
			}
			if requests != 0 {
				t.Fatalf("untrusted initial URL reached transport %d times", requests)
			}
		})
	}
}

// Deliberate downgrade fixture; the transport never opens a network connection.
//
// noinspection HttpUrlsUsage
func untrustedReleaseImageURLs() []string {
	return []string{
		"http://github.com/frathe/picfetch/image.png",
		"https://example.invalid/image.png",
		"https://127.0.0.1/image.png",
		"https://github.com.example.invalid/image.png",
		"https://githubusercontent.com.example.invalid/image.png",
		"https://notgithubusercontent.com/image.png",
		"https://user:password@github.com/image.png",
	}
}

func TestReleaseImageClientRejectsUntrustedRedirect(t *testing.T) {
	data := releaseTestPNG(t)
	for _, target := range untrustedReleaseImageURLs() {
		t.Run(target, func(t *testing.T) {
			for _, trustedHops := range []int{0, 1} {
				requests := 0
				client := &http.Client{Transport: releaseImageTransport(func(request *http.Request) (*http.Response, error) {
					requests++
					if requests <= trustedHops+1 {
						location := target
						if requests <= trustedHops {
							location = "https://raw.githubusercontent.com/frathe/picfetch/main/art.png"
						}
						return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": {location}}, Body: io.NopCloser(strings.NewReader(""))}, nil
					}
					return &http.Response{StatusCode: http.StatusOK, Request: request, Body: io.NopCloser(bytes.NewReader(data))}, nil
				})}
				got, err := loadReleaseImage(context.Background(), client, testReleaseImageURL)
				if got != nil || err == nil || !strings.Contains(err.Error(), "redirect") {
					t.Fatalf("after %d trusted hops: image = %v, error = %v; want redirect rejection", trustedHops, got, err)
				}
				if requests != trustedHops+1 {
					t.Fatalf("after %d trusted hops: transport calls = %d; untrusted target must not be requested", trustedHops, requests)
				}
			}
		})
	}
}

func TestReleaseImageClientAllowsProviderRedirects(t *testing.T) {
	data := releaseTestPNG(t)
	for _, status := range []int{http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect} {
		requests := 0
		target := "https://raw.githubusercontent.com/frathe/picfetch/main/art.png?raw=true"
		client := &http.Client{Transport: releaseImageTransport(func(request *http.Request) (*http.Response, error) {
			requests++
			if requests == 1 {
				return &http.Response{StatusCode: status, Header: http.Header{"Location": {target}}, Body: io.NopCloser(strings.NewReader(""))}, nil
			}
			if request.URL.String() != target || request.Method != http.MethodGet {
				t.Errorf("redirect request = %s %s; want GET %s", request.Method, request.URL, target)
			}
			return &http.Response{StatusCode: http.StatusOK, Request: request, Body: io.NopCloser(bytes.NewReader(data))}, nil
		})}
		got, err := loadReleaseImage(context.Background(), client, testReleaseImageURL)
		if err != nil || got == nil || requests != 2 {
			t.Fatalf("HTTP %d: image = %v, error = %v, requests = %d; want decoded image after two requests", status, got, err, requests)
		}
	}
}

func TestReleaseImageClientBoundsRedirects(t *testing.T) {
	requests := 0
	client := &http.Client{Timeout: time.Second, Transport: releaseImageTransport(func(_ *http.Request) (*http.Response, error) {
		requests++
		return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": {testReleaseImageURL}}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})}
	_, err := loadReleaseImage(context.Background(), client, testReleaseImageURL)
	if err == nil || !strings.Contains(err.Error(), "redirect") || requests != 10 {
		t.Fatalf("redirect loop: error = %v, requests = %d; want rejection after ten requests", err, requests)
	}
}

func TestLoadReleaseImageRejectsUntrustedResponseURL(t *testing.T) {
	data := releaseTestPNG(t)
	client := &http.Client{Transport: releaseImageTransport(func(request *http.Request) (*http.Response, error) {
		finalRequest := request.Clone(request.Context())
		finalRequest.URL.Host = "example.invalid"
		return &http.Response{StatusCode: http.StatusOK, Request: finalRequest, Body: io.NopCloser(bytes.NewReader(data))}, nil
	})}
	got, err := loadReleaseImage(context.Background(), client, testReleaseImageURL)
	if got != nil || err == nil || !strings.Contains(err.Error(), "untrusted URL") {
		t.Fatalf("image = %v, error = %v; want untrusted response URL rejection", got, err)
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
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write(tc.data)
			}))
			defer server.Close()
			got, err := loadReleaseImage(context.Background(), releaseImageTestClient(t, server), testReleaseImageURL)
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
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				// Flush before writing so the response has no Content-Length.
				w.WriteHeader(http.StatusOK)
				w.(http.Flusher).Flush()
				_, _ = w.Write(data)
			}))
			defer server.Close()
			got, err := loadReleaseImage(context.Background(), releaseImageTestClient(t, server), testReleaseImageURL)
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
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			client := releaseImageTestClient(t, server)
			done := make(chan error, 1)
			go func() {
				_, err := loadReleaseImage(ctx, client, testReleaseImageURL)
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
