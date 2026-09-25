package locationmap

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type tileTransport func(*http.Request) (*http.Response, error)

func (t tileTransport) RoundTrip(req *http.Request) (*http.Response, error) { return t(req) }

func tileClient(handler http.HandlerFunc) *http.Client {
	return &http.Client{Transport: tileTransport(func(req *http.Request) (*http.Response, error) {
		recorder := httptest.NewRecorder()
		handler(recorder, req)
		return recorder.Result(), nil
	})}
}

func tilePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 256, 256))
	img.Set(0, 0, color.NRGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestTileStoreRejectsRedirects(t *testing.T) {
	for _, destination := range []string{
		"https://other.example/1/0/0.png",
		"https://127.0.0.1/private",
		"http://192.168.1.1/private",
		"http://tile.openstreetmap.org/1/0/0.png",
		"https://tile.openstreetmap.org/redirected.png",
	} {
		t.Run(destination, func(t *testing.T) {
			hits := 0
			client := tileClient(func(w http.ResponseWriter, _ *http.Request) {
				hits++
				if hits == 1 {
					w.Header().Set("Location", destination)
					w.WriteHeader(http.StatusFound)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			})
			store := NewTileStore(TileOptions{Client: client})
			if _, _, err := store.Fetch(context.Background(), TileKey{Z: 1}); err == nil {
				t.Fatal("redirect response accepted as a tile")
			}
			if hits != 1 {
				t.Fatalf("redirect sent %d requests; want only the original OSM request", hits)
			}
			if client.CheckRedirect != nil {
				t.Fatal("tile store changed the caller's shared HTTP client")
			}
		})
	}
}

func TestTileStoreFreshAndRevalidate(t *testing.T) {
	pixels := tilePNG(t)
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	var hits int
	client := tileClient(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if got := r.Header.Get("User-Agent"); got != "PicFetch (+https://github.com/frathe/picfetch)" {
			t.Errorf("User-Agent = %q", got)
		}
		if r.URL.Path != "/3/4/5.png" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if hits == 1 {
			w.Header().Set("Cache-Control", "max-age=10")
			w.Header().Set("Age", "4")
			w.Header().Set("ETag", `"first"`)
			_, _ = w.Write(pixels)
			return
		}
		if got := r.Header.Get("If-None-Match"); got != `"first"` {
			t.Errorf("If-None-Match = %q", got)
		}
		w.Header().Set("Cache-Control", "max-age=20")
		w.WriteHeader(http.StatusNotModified)
	})
	store := NewTileStore(TileOptions{Client: client, Now: func() time.Time { return now }, URL: "https://tiles.test/%d/%d/%d.png"})
	key := TileKey{Z: 3, X: 4, Y: 5}
	for _, advance := range []time.Duration{0, 5 * time.Second, 2 * time.Second, 15 * time.Second} {
		now = now.Add(advance)
		img, retry, err := store.Fetch(context.Background(), key)
		if err != nil || img == nil || !retry.IsZero() {
			t.Fatalf("Fetch = %v, %v, %v", img, retry, err)
		}
	}
	if hits != 2 {
		t.Fatalf("HTTP hits = %d, want 2", hits)
	}
	encoded, decoded := store.Usage()
	const metadataBytes = len("Cache-Control") + len("max-age=20") + 16 + len("ETag") + len(`"first"`) + 16
	if encoded != int64(len(pixels)+metadataBytes) || decoded != 256*256*4 {
		t.Fatalf("Usage = %d, %d", encoded, decoded)
	}
}

func TestTileStoreResponseMetadata(t *testing.T) {
	pixels := tilePNG(t)
	t.Run("oversized", func(t *testing.T) {
		for _, status := range []int{http.StatusOK, http.StatusNotModified} {
			for _, field := range []string{"ETag", "Last-Modified", "Cache-Control"} {
				t.Run(http.StatusText(status)+"/"+field, func(t *testing.T) {
					hits := 0
					client := tileClient(func(w http.ResponseWriter, _ *http.Request) {
						hits++
						if status == http.StatusNotModified && hits == 1 {
							w.Header().Set("Cache-Control", "no-cache")
							w.Header().Set("ETag", "initial")
							_, _ = w.Write(pixels)
							return
						}
						w.Header().Set("Cache-Control", "max-age=3600")
						w.Header().Set(field, strings.Repeat("x", 4*1024+1))
						if status == http.StatusNotModified {
							w.WriteHeader(status)
						} else {
							_, _ = w.Write(pixels)
						}
					})
					store := NewTileStore(TileOptions{Client: client})
					if status == http.StatusNotModified {
						if _, _, err := store.Fetch(context.Background(), TileKey{}); err != nil {
							t.Fatal(err)
						}
					}
					if img, _, err := store.Fetch(context.Background(), TileKey{}); err != nil || img == nil {
						t.Fatalf("oversized metadata prevented displaying valid pixels: %v", err)
					}
					if encoded, decoded := store.Usage(); encoded != 0 || decoded != 0 {
						t.Fatalf("oversized metadata response retained cache state: %d, %d", encoded, decoded)
					}
				})
			}
		}
	})
	t.Run("ignore_unrelated", func(t *testing.T) {
		hits := 0
		store := NewTileStore(TileOptions{Client: tileClient(func(w http.ResponseWriter, _ *http.Request) {
			hits++
			w.Header().Set("X-Unused", strings.Repeat("x", 64*1024))
			w.Header().Set("Cache-Control", "max-age=3600")
			_, _ = w.Write(pixels)
		})})
		for range 2 {
			if _, _, err := store.Fetch(context.Background(), TileKey{}); err != nil {
				t.Fatal(err)
			}
		}
		encoded, _ := store.Usage()
		if hits != 1 || encoded <= int64(len(pixels)) || encoded > int64(len(pixels))+128 {
			t.Fatalf("retained metadata must be small, charged, and reusable: hits=%d encoded=%d pixels=%d", hits, encoded, len(pixels))
		}
	})
	t.Run("budget_eviction", func(t *testing.T) {
		hits := 0
		limit := int64(2 * len(pixels))
		store := NewTileStore(TileOptions{EncodedBytes: limit, DecodedBytes: 1, Client: tileClient(func(w http.ResponseWriter, _ *http.Request) {
			hits++
			w.Header().Set("ETag", strings.Repeat("e", len(pixels)/2))
			w.Header().Set("Cache-Control", "max-age=3600")
			_, _ = w.Write(pixels)
		})})
		for _, key := range []TileKey{{X: 1}, {X: 2}, {X: 1}} {
			if _, _, err := store.Fetch(context.Background(), key); err != nil {
				t.Fatal(err)
			}
			if encoded, decoded := store.Usage(); encoded > limit || decoded != 0 {
				t.Fatalf("metadata escaped the configured cache budgets: %d, %d", encoded, decoded)
			}
		}
		if hits != 3 {
			t.Fatalf("metadata did not consume encoded budget: hits=%d", hits)
		}
	})
	t.Run("revalidation_growth", func(t *testing.T) {
		hits := 0
		store := NewTileStore(TileOptions{EncodedBytes: int64(len(pixels)) + 128, DecodedBytes: 1, Client: tileClient(func(w http.ResponseWriter, r *http.Request) {
			hits++
			if hits == 2 {
				w.Header().Set("ETag", strings.Repeat("e", 256))
				w.Header().Set("Cache-Control", "max-age=3600")
				w.WriteHeader(http.StatusNotModified)
				return
			}
			if r.Header.Get("If-None-Match") != "" {
				t.Error("discarded metadata still supplied a validator")
			}
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("ETag", "initial")
			_, _ = w.Write(pixels)
		})})
		for range 2 {
			if img, _, err := store.Fetch(context.Background(), TileKey{}); err != nil || img == nil {
				t.Fatalf("fetch failed: %v", err)
			}
		}
		if encoded, decoded := store.Usage(); encoded != 0 || decoded != 0 {
			t.Fatalf("304 growth escaped encoded budget: %d, %d", encoded, decoded)
		}
		if _, _, err := store.Fetch(context.Background(), TileKey{}); err != nil || hits != 3 {
			t.Fatalf("discarded entry was reused: hits=%d error=%v", hits, err)
		}
	})
	t.Run("decoded_only_budget", func(t *testing.T) {
		hits := 0
		store := NewTileStore(TileOptions{EncodedBytes: 128, DecodedBytes: 4 * 256 * 256 * 4, Client: tileClient(func(w http.ResponseWriter, _ *http.Request) {
			hits++
			w.Header().Set("ETag", strings.Repeat("e", 32))
			w.Header().Set("Cache-Control", "max-age=3600")
			_, _ = w.Write(pixels)
		})})
		for _, key := range []TileKey{{X: 1}, {X: 2}, {X: 1}} {
			if _, _, err := store.Fetch(context.Background(), key); err != nil {
				t.Fatal(err)
			}
			if encoded, _ := store.Usage(); encoded <= 0 || encoded > 128 {
				t.Fatalf("decoded-only metadata escaped encoded budget: %d", encoded)
			}
		}
		if hits != 3 {
			t.Fatalf("decoded-only metadata was not evicted: hits=%d", hits)
		}
	})
	t.Run("merged_metadata_limit", func(t *testing.T) {
		hits := 0
		store := NewTileStore(TileOptions{Client: tileClient(func(w http.ResponseWriter, _ *http.Request) {
			hits++
			if hits == 1 {
				w.Header().Set("ETag", strings.Repeat("e", 3*1024))
				w.Header().Set("Cache-Control", "no-cache")
				_, _ = w.Write(pixels)
				return
			}
			w.Header().Set("Last-Modified", strings.Repeat("m", 2*1024))
			w.WriteHeader(http.StatusNotModified)
		})})
		for range 2 {
			if img, _, err := store.Fetch(context.Background(), TileKey{}); err != nil || img == nil {
				t.Fatalf("valid pixels were lost: %v", err)
			}
		}
		if encoded, decoded := store.Usage(); encoded != 0 || decoded != 0 {
			t.Fatalf("304 merged metadata exceeded the per-entry limit: %d, %d", encoded, decoded)
		}
	})
}

func TestTileStoreNoStoreAndExpiry(t *testing.T) {
	pixels := tilePNG(t)
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	var hits int
	client := tileClient(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		if hits <= 2 {
			w.Header().Set("Cache-Control", "no-store")
		} else {
			w.Header().Set("Expires", now.Add(3*time.Second).Format(http.TimeFormat))
			w.Header().Set("Age", "1")
		}
		_, _ = w.Write(pixels)
	})
	store := NewTileStore(TileOptions{Client: client, Now: func() time.Time { return now }, URL: "https://tiles.test/%d/%d/%d.png"})
	key := TileKey{Z: 1}
	for i := 0; i < 2; i++ {
		if _, _, err := store.Fetch(context.Background(), key); err != nil {
			t.Fatal(err)
		}
	}
	if encoded, decoded := store.Usage(); encoded != 0 || decoded != 0 {
		t.Fatalf("no-store retained %d, %d", encoded, decoded)
	}
	for i := 0; i < 2; i++ {
		if _, _, err := store.Fetch(context.Background(), key); err != nil {
			t.Fatal(err)
		}
	}
	if hits != 3 {
		t.Fatalf("HTTP hits = %d, want 3", hits)
	}
	const metadataBytes = len("Expires") + len(http.TimeFormat) + 16 + len("Age") + len("1") + 16
	if encoded, decoded := store.Usage(); encoded != int64(len(pixels)+metadataBytes) || decoded != 256*256*4 {
		t.Fatalf("retained fresh entry = %d, %d", encoded, decoded)
	}
	now = now.Add(3 * time.Second)
	if _, _, err := store.Fetch(context.Background(), key); err != nil {
		t.Fatal(err)
	}
	if hits != 4 {
		t.Fatalf("expired HTTP hits = %d, want 4", hits)
	}
}

func TestTileStoreRetryAndCancellation(t *testing.T) {
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	var hits atomic.Int32
	client := tileClient(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusServiceUnavailable)
	})
	store := NewTileStore(TileOptions{Client: client, Now: func() time.Time { return now }, URL: "https://tiles.test/%d/%d/%d.png"})
	key := TileKey{}
	for i := 0; i < 2; i++ {
		img, retry, err := store.Fetch(context.Background(), key)
		// Fetch intentionally returns a retry deadline together with an error.
		//goland:noinspection GoDfaErrorMayBeNotNil
		if img != nil || err == nil || !retry.Equal(now.Add(5*time.Second)) {
			t.Fatalf("Fetch = %v, %v, %v", img, retry, err)
		}
	}
	if hits.Load() != 1 {
		t.Fatalf("retry cache hits = %d", hits.Load())
	}
	now = now.Add(5 * time.Second)
	_, retry, err := store.Fetch(context.Background(), key)
	// Fetch intentionally returns a retry deadline together with an error.
	//goland:noinspection GoDfaErrorMayBeNotNil
	if err == nil || !retry.Equal(now.Add(5*time.Second)) || hits.Load() != 2 {
		t.Fatalf("second failure = %v, %v, hits %d", retry, err, hits.Load())
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	_, retry, err = store.Fetch(cancelled, TileKey{X: 1})
	if !errors.Is(err, context.Canceled) || !retry.IsZero() {
		t.Fatalf("cancelled = %v, %v", retry, err)
	}
	_, _, _ = store.Fetch(context.Background(), TileKey{X: 1})
	if hits.Load() != 3 {
		t.Fatalf("cancelled request installed state: hits %d", hits.Load())
	}
}

func TestTileStoreBoundsAndInvalidPNG(t *testing.T) {
	pixels := tilePNG(t)
	var hits int
	client := tileClient(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Cache-Control", "max-age=3600")
		if r.URL.Path == "/0/9/0.png" {
			_, _ = w.Write([]byte("not a PNG"))
			return
		}
		_, _ = w.Write(pixels)
	})
	store := NewTileStore(TileOptions{Client: client, URL: "https://tiles.test/%d/%d/%d.png", EncodedBytes: int64(len(pixels)) + 1, DecodedBytes: 256*256*4 + 1})
	for _, key := range []TileKey{{X: 1}, {X: 2}, {X: 1}} {
		if _, _, err := store.Fetch(context.Background(), key); err != nil {
			t.Fatal(err)
		}
		encoded, decoded := store.Usage()
		if encoded > int64(len(pixels))+1 || decoded > 256*256*4+1 {
			t.Fatalf("unbounded usage %d, %d", encoded, decoded)
		}
	}
	if hits != 3 {
		t.Fatalf("eviction hits = %d", hits)
	}
	if img, _, err := store.Fetch(context.Background(), TileKey{X: 9}); img != nil || err == nil {
		t.Fatalf("invalid PNG = %v, %v", img, err)
	}
}

func TestTileStoreOversizeAndCanceledDelivery(t *testing.T) {
	pixels := tilePNG(t)
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	ctx, cancel := context.WithCancel(context.Background())
	var hits int
	client := tileClient(func(w http.ResponseWriter, r *http.Request) {
		hits++
		switch r.URL.Path {
		case "/0/1/0.png":
			_, _ = w.Write(bytes.Repeat([]byte{'x'}, maxTileResponse+1))
		case "/0/2/0.png":
			if hits == 2 {
				cancel()
			}
			_, _ = w.Write(pixels)
		}
	})
	store := NewTileStore(TileOptions{Client: client, Now: func() time.Time { return now }, URL: "https://tiles.test/%d/%d/%d.png"})
	// Fetch intentionally returns a retry deadline together with an error.
	//goland:noinspection GoDfaErrorMayBeNotNil
	if _, retry, err := store.Fetch(context.Background(), TileKey{X: 1}); err == nil || !retry.Equal(now.Add(time.Second)) {
		t.Fatalf("oversize retry = %v, %v", retry, err)
	}
	// Cancellation deliberately has a zero deadline, unlike a retriable error.
	//goland:noinspection GoDfaErrorMayBeNotNil
	if img, retry, err := store.Fetch(ctx, TileKey{X: 2}); img != nil || !retry.IsZero() || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled delivery = %v, %v, %v", img, retry, err)
	}
	if encoded, decoded := store.Usage(); encoded != 0 || decoded != 0 {
		t.Fatalf("canceled/oversize usage = %d, %d", encoded, decoded)
	}
	if _, _, err := store.Fetch(context.Background(), TileKey{X: 2}); err != nil || hits != 3 {
		t.Fatalf("retry canceled key: hits %d, error %v", hits, err)
	}
}

func TestTileStoreRetryAfterDateAndFailureBound(t *testing.T) {
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	var hits int
	client := tileClient(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		w.Header().Set("Retry-After", now.Add(15*time.Second).Format(http.TimeFormat))
		w.WriteHeader(http.StatusTooManyRequests)
	})
	store := NewTileStore(TileOptions{Client: client, Now: func() time.Time { return now }, URL: "https://tiles.test/%d/%d/%d.png"})
	for x := 0; x <= maxTileFailures; x++ {
		_, retry, err := store.Fetch(context.Background(), TileKey{X: x})
		// Fetch intentionally returns a retry deadline together with an error.
		//goland:noinspection GoDfaErrorMayBeNotNil
		if err == nil || !retry.Equal(now.Add(15*time.Second)) {
			t.Fatalf("key %d retry = %v, %v", x, retry, err)
		}
	}
	if hits != maxTileFailures+1 {
		t.Fatalf("initial HTTP hits = %d", hits)
	}
	_, _, _ = store.Fetch(context.Background(), TileKey{X: 0})
	if hits != maxTileFailures+2 {
		t.Fatalf("oldest failure was retained: hits %d", hits)
	}
	_, _, _ = store.Fetch(context.Background(), TileKey{X: maxTileFailures})
	if hits != maxTileFailures+2 {
		t.Fatalf("recent failure was discarded: hits %d", hits)
	}
}

func TestTileStoreNoCacheAndFallbackTTL(t *testing.T) {
	pixels := tilePNG(t)
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, time.UTC)
	var hits int
	client := tileClient(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if r.URL.Path == "/0/1/0.png" {
			if hits > 1 && r.Header.Get("If-Modified-Since") != "Wed, 21 Oct 2015 07:28:00 GMT" {
				t.Errorf("If-Modified-Since = %q", r.Header.Get("If-Modified-Since"))
			}
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
		}
		_, _ = w.Write(pixels)
	})
	store := NewTileStore(TileOptions{Client: client, Now: func() time.Time { return now }, URL: "https://tiles.test/%d/%d/%d.png"})
	for i := 0; i < 2; i++ {
		if _, _, err := store.Fetch(context.Background(), TileKey{X: 1}); err != nil {
			t.Fatal(err)
		}
	}
	if hits != 2 {
		t.Fatalf("no-cache HTTP hits = %d", hits)
	}
	if _, _, err := store.Fetch(context.Background(), TileKey{X: 2}); err != nil {
		t.Fatal(err)
	}
	now = now.Add(7*24*time.Hour - time.Second)
	if _, _, err := store.Fetch(context.Background(), TileKey{X: 2}); err != nil || hits != 3 {
		t.Fatalf("fallback before expiry: hits %d, error %v", hits, err)
	}
	now = now.Add(2 * time.Second)
	if _, _, err := store.Fetch(context.Background(), TileKey{X: 2}); err != nil || hits != 4 {
		t.Fatalf("fallback after expiry: hits %d, error %v", hits, err)
	}
}

func TestTileStoreRevalidationRetainsNoCache(t *testing.T) {
	pixels := tilePNG(t)
	hits := 0
	client := tileClient(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		if hits == 1 {
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("ETag", "v1")
			_, _ = w.Write(pixels)
			return
		}
		w.WriteHeader(http.StatusNotModified)
	})
	store := NewTileStore(TileOptions{Client: client})
	for range 3 {
		if _, _, err := store.Fetch(context.Background(), TileKey{}); err != nil {
			t.Fatal(err)
		}
	}
	if hits != 3 {
		t.Fatalf("304 discarded stored no-cache directive: %d requests", hits)
	}
}

func TestTileStoreSixteenBitPNGResidency(t *testing.T) {
	img := image.NewNRGBA64(image.Rect(0, 0, 256, 256))
	img.Set(0, 0, color.NRGBA64{R: 65535, A: 65535})
	var data bytes.Buffer
	if err := png.Encode(&data, img); err != nil {
		t.Fatal(err)
	}
	store := NewTileStore(TileOptions{Client: tileClient(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(data.Bytes()) })})
	got, _, err := store.Fetch(context.Background(), TileKey{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.(*image.NRGBA64); ok {
		t.Fatal("16-bit tile retained twice the charged decoded bytes")
	}
}
