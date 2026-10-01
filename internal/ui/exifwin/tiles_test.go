package exifwin

import (
	"bytes"
	"errors"
	"fmt"
	"image/color"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/uitest"
)

// tilePNG is a correctly sized map tile accepted by the fetcher's validation.
func tilePNG(t *testing.T) []byte {
	t.Helper()

	return uitest.EncodePNG(t, tileSize, tileSize, color.RGBA{R: 1, G: 2, B: 3, A: 255})
}

func TestTileFetcherHTTPFreshness(t *testing.T) {
	for _, scenario := range []struct {
		name           string
		header         http.Header
		fresh, noStore bool
	}{
		{"max-age=60", http.Header{"Cache-Control": {"max-age=60"}}, true, false},
		{"no-cache", http.Header{"Cache-Control": {"no-cache"}}, false, false},
		{"no-store", http.Header{"Cache-Control": {"no-store"}}, false, true},
		{"vary wildcard", http.Header{"Vary": {"*"}, "Cache-Control": {"max-age=60"}}, false, true},
		{"invalid expires", http.Header{"Expires": {"0"}}, false, false},
		{"invalid expires overridden", http.Header{"Expires": {"0"}, "Cache-Control": {"max-age=60"}}, true, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			body := tilePNG(t)
			var requests atomic.Int32
			var conditional atomic.Bool
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				count := requests.Add(1)
				for name, values := range scenario.header {
					writer.Header()[name] = append([]string(nil), values...)
				}
				writer.Header().Set("ETag", `"tile"`)
				writer.Header().Set("Date", time.Unix(1700000000, 0).UTC().Format(http.TimeFormat))
				if count > 1 && request.Header.Get("If-None-Match") == `"tile"` {
					conditional.Store(true)
					writer.WriteHeader(http.StatusNotModified)
					return
				}
				_, _ = writer.Write(body)
			}))
			defer server.Close()
			fetcher := newTileFetcher(server.URL+"/%d/%d/%d.png", server.Client().Transport)
			defer func() { fetcher.Stop(); fetcher.Wait() }()
			now := time.Unix(1700000000, 0)
			fetcher.now = func() time.Time { return now }
			request, err := http.NewRequest(http.MethodGet, server.URL+"/1/0/0.png", nil)
			if err != nil {
				t.Fatal(err)
			}
			readTile := func() {
				t.Helper()
				response, requestErr := fetcher.RoundTrip(request)
				if errors.Is(requestErr, errTilePending) {
					fetcher.Wait()
					response, requestErr = fetcher.RoundTrip(request)
				}
				if requestErr != nil {
					t.Fatal(requestErr)
				}
				data, readErr := io.ReadAll(response.Body)
				_ = response.Body.Close()
				if readErr != nil || !bytes.Equal(data, body) {
					t.Fatal("tile delivery lost pixels")
				}
			}
			readTile()
			readTile()
			want := int32(2)
			if scenario.fresh {
				want = 1
			}
			if requests.Load() != want {
				t.Fatalf("requests = %d, want %d", requests.Load(), want)
			}
			now = now.Add(2 * time.Minute)
			readTile()
			if requests.Load() != want+1 {
				t.Fatal("expired tile was reused without validation")
			}
			if conditional.Load() == scenario.noStore {
				t.Fatal("validator retention does not match cache policy")
			}
		})
	}
}

func TestTileFetcherResponseDelay(t *testing.T) {
	body := tilePNG(t)
	for _, status := range []int{http.StatusOK, http.StatusNotModified} {
		for _, date := range []string{"", "invalid", time.Unix(1700000000, 0).UTC().Format(http.TimeFormat)} {
			for _, policy := range []struct {
				age, maxAge string
				remaining   time.Duration
			}{
				{"4", "5", 0},
				{"4", "10", 4 * time.Second},
				{"9223372036854775807", "10", 0},
			} {
				t.Run(fmt.Sprintf("%d/date=%s/age=%s/max-age=%s", status, date, policy.age, policy.maxAge), func(t *testing.T) {
					now := time.Unix(1700000000, 0).UTC()
					fetcher := newTileFetcher("https://tiles.invalid/%d/%d/%d.png", tileRoundTripFunc(func(request *http.Request) (*http.Response, error) {
						now = now.Add(2 * time.Second)
						return &http.Response{StatusCode: status, Header: http.Header{"Cache-Control": {"max-age=" + policy.maxAge}, "Age": {policy.age}, "Date": {date}}, Body: io.NopCloser(bytes.NewReader(body)), Request: request}, nil
					}))
					defer func() { fetcher.Stop(); fetcher.Wait() }()
					fetcher.now = func() time.Time { return now }
					url := "https://tiles.invalid/one"
					if status == http.StatusNotModified {
						fetcher.cache.AddIfFits(url, &cachedTile{data: body, header: http.Header{"Etag": {`"tile"`}}})
					}
					entry, err := fetcher.get(fetcher.session(), url)
					if err != nil {
						t.Fatal(err)
					}
					if got := entry.expires.Sub(now); got != policy.remaining {
						t.Fatalf("remaining freshness = %v, want %v after 2s response delay", got, policy.remaining)
					}
				})
			}
		}
	}
}

func TestTileCacheFreshnessRules(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	for _, testCase := range []struct {
		name    string
		header  http.Header
		fresh   time.Duration
		noStore bool
	}{
		{"fallback", http.Header{}, 7 * 24 * time.Hour, false},
		{"escaped quoted max-age", http.Header{"Cache-Control": {`max-age="6\0"`}}, time.Minute, false},
		{"quoted max-age", http.Header{"Cache-Control": {`max-age="60"`}}, time.Minute, false},
		{"max-age with age", http.Header{"Cache-Control": {"max-age=60"}, "Age": {"20"}}, 40 * time.Second, false},
		{"apparent age", http.Header{"Cache-Control": {"max-age=60"}, "Date": {now.Add(-30 * time.Second).Format(http.TimeFormat)}}, 30 * time.Second, false},
		{"expires", http.Header{"Date": {now.Format(http.TimeFormat)}, "Expires": {now.Add(time.Hour).Format(http.TimeFormat)}}, time.Hour, false},
		{"expired", http.Header{"Expires": {now.Add(-time.Hour).Format(http.TimeFormat)}}, 0, false},
		{"repeated expires uses first", http.Header{"Expires": {now.Add(time.Hour).Format(http.TimeFormat), "0"}}, time.Hour, false},
		{"space before equals", http.Header{"Expires": {"0"}, "Cache-Control": {"max-age =86400"}}, 0, true},
		{"tab before equals", http.Header{"Expires": {"0"}, "Cache-Control": {"max-age\t=86400"}}, 0, true},
		{"space after equals", http.Header{"Expires": {"0"}, "Cache-Control": {"max-age= 86400"}}, 0, true},
		{"tab after equals", http.Header{"Expires": {"0"}, "Cache-Control": {"max-age=\t86400"}}, 0, true},
		{"space before quoted argument", http.Header{"Expires": {"0"}, "Cache-Control": {`max-age= "86400"`}}, 0, true},
		{"quoted comma extension", http.Header{"Expires": {"0"}, "Cache-Control": {`ext="a,max-age=86400,b"`}}, 0, false},
		{"escaped quote extension", http.Header{"Expires": {"0"}, "Cache-Control": {`ext="a\",max-age=86400,b"`}}, 0, false},
		{"quoted extension then max-age", http.Header{"Expires": {"0"}, "Cache-Control": {`ext="a,max-age=86400,b", max-age=60`}}, time.Minute, false},
		{"escaped backslash extension", http.Header{"Cache-Control": {`ext="a\\", max-age=60`}}, time.Minute, false},
		{"unclosed extension quote", http.Header{"Cache-Control": {`ext="a,max-age=86400`}}, 0, true},
		{"dangling quoted escape", http.Header{"Cache-Control": {`ext="a\`}}, 0, true},
		{"invalid extension token", http.Header{"Cache-Control": {`ext=a"b", max-age=60`}}, 0, true},
		{"vary wildcard", http.Header{"Vary": {"*"}, "Cache-Control": {"max-age=60"}}, time.Minute, true},
		{"vary wildcard in list", http.Header{"Vary": {"Accept-Encoding, *"}, "Cache-Control": {"max-age=60"}}, time.Minute, true},
		{"zero expires", http.Header{"Expires": {"0"}}, 0, false},
		{"invalid expires", http.Header{"Expires": {"not a date"}}, 0, false},
		{"empty expires", http.Header{"Expires": {""}}, 0, false},
		{"invalid expires overridden", http.Header{"Cache-Control": {"max-age=60"}, "Expires": {"0"}}, time.Minute, false},
		{"repeated max-age", http.Header{"Cache-Control": {"max-age=0, max-age=60"}}, 0, false},
		{"repeated max-age fields", http.Header{"Cache-Control": {"max-age=10", "max-age=60"}}, 0, false},
		{"signed max-age", http.Header{"Cache-Control": {"max-age=+60"}}, 0, false},
		{"malformed quoted max-age", http.Header{"Cache-Control": {`max-age="60""`}}, 0, true},
		{"precedence", http.Header{"Cache-Control": {"max-age=10"}, "Expires": {now.Add(time.Hour).Format(http.TimeFormat)}}, 10 * time.Second, false},
		{"no-cache", http.Header{"Cache-Control": {"max-age=60, no-cache"}}, 0, false},
		{"no-store", http.Header{"Cache-Control": {"no-store, max-age=60"}}, time.Minute, true},
		{"overflow", http.Header{"Cache-Control": {"max-age=9223372036854775807"}}, 0, false},
		{"old age", http.Header{"Cache-Control": {"max-age=60"}, "Age": {"9223372036854775807"}}, 0, false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			entry := cacheTile([]byte("tile"), nil, testCase.header, now, now)
			if entry.expires.Sub(now) != testCase.fresh || entry.noStore != testCase.noStore {
				t.Fatalf("freshness = %v, no-store = %t", entry.expires.Sub(now), entry.noStore)
			}
		})
	}
	previous := http.Header{"Cache-Control": {"max-age=60"}, "Etag": {`"old"`}, "Last-Modified": {now.Add(-time.Hour).Format(http.TimeFormat)}}
	entry := cacheTile([]byte("tile"), previous, http.Header{"Cache-Control": {"max-age=120"}, "Etag": {`"new"`}}, now, now)
	if entry.header.Get("ETag") != `"new"` || entry.header.Get("Last-Modified") != previous.Get("Last-Modified") || entry.expires.Sub(now) != 2*time.Minute {
		t.Fatal("304 did not merge validators and replace freshness")
	}
	oversize := cacheTile([]byte("tile"), nil, http.Header{"Etag": {strings.Repeat("x", 17*1024)}}, now, now)
	if !oversize.noStore || tileWeight(oversize) != 4 {
		t.Fatal("oversized metadata escaped the cache bound")
	}
}

func TestTileFetcherNoStoreDeliveryClearedOnCancel(t *testing.T) {
	body := tilePNG(t)
	fetcher := newTileFetcher("https://tiles.invalid/%d/%d/%d.png", tileRoundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Cache-Control": {"no-store"}}, Body: io.NopCloser(bytes.NewReader(body)), Request: request}, nil
	}))
	defer func() { fetcher.Stop(); fetcher.Wait() }()
	job, _ := fetcher.submit(fetcher.session(), "https://tiles.invalid/1/0/0.png", true)
	<-job.done
	if fetcher.cache.Len() != 0 || fetcher.ready.Len() != 1 {
		t.Fatal("no-store result must only await its pending delivery")
	}
	fetcher.Cancel()
	if fetcher.ready.Len() != 0 {
		t.Fatal("cancel retained no-store delivery")
	}
}

func pngWithDimensions(t *testing.T, width, height int) []byte {
	t.Helper()

	return uitest.EncodePNG(t, width, height, color.Black)
}

// tileServer is a stand-in tile service: it counts what was asked for, and
// can be made to hang until released, so a test can look at the world while
// a download is still in flight.
type tileServer struct {
	*httptest.Server

	mu       sync.Mutex
	requests []string

	block chan struct{}
	fail  bool
}

func newTileServer(t *testing.T) *tileServer {
	t.Helper()

	body := tilePNG(t)
	s := &tileServer{}

	s.Server = httptest.NewServer(http.HandlerFunc(func(wr http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.requests = append(s.requests, r.URL.Path)
		block, fail := s.block, s.fail
		s.mu.Unlock()

		if block != nil {
			<-block
		}

		if fail {
			wr.WriteHeader(http.StatusInternalServerError)
			return
		}

		wr.Header().Set("Content-Type", "image/png")
		_, _ = wr.Write(body)
	}))

	t.Cleanup(s.Close)

	return s
}

func (s *tileServer) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return len(s.requests)
}

func (s *tileServer) paths() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]string(nil), s.requests...)
}

// hold makes every later request hang until the returned func is called.
func (s *tileServer) hold() func() {
	block := make(chan struct{})

	s.mu.Lock()
	s.block = block
	s.mu.Unlock()

	var once sync.Once

	return func() {
		once.Do(func() {
			s.mu.Lock()
			s.block = nil
			s.mu.Unlock()

			close(block)
		})
	}
}

func (s *tileServer) breakIt() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.fail = true
}

// fetcherFor returns a tileFetcher pointed at s instead of the real tile
// service - the swap every test in this package makes, and the reason the
// fetcher is a field on Window rather than package-level state.
func fetcherFor(s *tileServer) *tileFetcher {
	return newTileFetcher(s.URL+"/%d/%d/%d.png", http.DefaultTransport)
}

func TestGet_RejectsInvalidOrUnexpectedTileImages(t *testing.T) {
	tests := []struct {
		name string
		body []byte
	}{
		{name: "not PNG", body: []byte("not a PNG")},
		{name: "wrong dimensions", body: pngWithDimensions(t, 257, 256)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write(test.body)
			}))
			t.Cleanup(server.Close)

			fetcher := newTileFetcher(server.URL+"/%d/%d/%d.png", http.DefaultTransport)
			if _, err := fetcher.get(t.Context(), server.URL+"/15/0/0.png"); err == nil {
				t.Fatal("get accepted an invalid map tile")
			}
		})
	}
}

// waitForPending blocks until the fetcher has nothing outstanding.
func waitForPending(t *testing.T, f *tileFetcher) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if f.Pending() == 0 {
			return
		}

		time.Sleep(5 * time.Millisecond)
	}

	t.Fatalf("timed out with %d tiles still pending", f.Pending())
}

func TestRoundTrip_AnswersAMissImmediatelyAndFetchesInTheBackground(t *testing.T) {
	s := newTileServer(t)
	release := s.hold()
	t.Cleanup(release)

	f := fetcherFor(s)

	req, err := http.NewRequest(http.MethodGet, s.URL+"/15/0/0.png", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	start := time.Now()
	res, err := f.RoundTrip(req)
	elapsed := time.Since(start)

	if res != nil {
		t.Error("RoundTrip returned a response for an uncached tile, want none")
	}

	if !errors.Is(err, errTilePending) {
		t.Fatalf("RoundTrip() err = %v, want errTilePending", err)
	}

	// The map widget calls this from inside its raster draw, on the UI
	// goroutine: whatever the network is doing, it has to come straight
	// back or the app freezes.
	if elapsed > time.Second {
		t.Errorf("RoundTrip blocked for %v on a hanging server, want an immediate answer", elapsed)
	}

	release()
	waitForPending(t, f)

	res, err = f.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip() after the download err = %v, want a cached response", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("cached response status = %d, want 200", res.StatusCode)
	}

	body := make([]byte, 8)
	if _, err := res.Body.Read(body); err != nil {
		t.Fatalf("read cached body: %v", err)
	}

	if !bytes.Equal(body, tilePNG(t)[:8]) {
		t.Error("cached response body is not the tile the server sent")
	}
}

func TestRoundTrip_DownloadsATileOnlyOnce(t *testing.T) {
	s := newTileServer(t)

	// Hold the server so the tile is genuinely still on its way for all the
	// repaints below. Without this the background download can land between
	// two of them - a loopback fetch takes microseconds - and the cache
	// starts answering them for real, which is correct behaviour failing an
	// assertion about a state the test no longer has.
	release := s.hold()
	t.Cleanup(release)

	f := fetcherFor(s)

	req, err := http.NewRequest(http.MethodGet, s.URL+"/15/1/1.png", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	// Every repaint re-asks for the same tile while it is on its way; only
	// the first of those may turn into a request.
	for range 5 {
		if _, err := f.RoundTrip(req); !errors.Is(err, errTilePending) {
			t.Fatalf("RoundTrip() err = %v, want errTilePending", err)
		}
	}

	release()
	waitForPending(t, f)

	if _, err := f.RoundTrip(req); err != nil {
		t.Fatalf("RoundTrip() after the download err = %v, want a cached response", err)
	}

	if got := s.count(); got != 1 {
		t.Errorf("server saw %d requests, want exactly 1", got)
	}
}

func TestWarm_DownloadsTheBlockAroundTheLocationOnce(t *testing.T) {
	s := newTileServer(t)
	f := fetcherFor(s)

	f.Warm(48.858222, 2.2945, mapZoom)

	want := (2*prefetchRadius + 1) * (2*prefetchRadius + 1)
	if got := s.count(); got != want {
		t.Fatalf("server saw %d requests, want the %dx%d block (%d)", got, 2*prefetchRadius+1, 2*prefetchRadius+1, want)
	}

	for _, p := range s.paths() {
		if !strings.HasPrefix(p, "/15/") {
			t.Errorf("prefetched %q, want a tile at zoom %d", p, mapZoom)
		}
	}

	// Re-expanding the section must not re-download what is already cached.
	f.Warm(48.858222, 2.2945, mapZoom)

	if got := s.count(); got != want {
		t.Errorf("server saw %d requests after a second warm, want the cache to serve it (%d)", got, want)
	}

	if f.Pending() != 0 {
		t.Errorf("Pending() = %d after Warm returned, want 0", f.Pending())
	}
}

func TestOnChange_ReportsABackgroundBatchButNotAPrefetch(t *testing.T) {
	s := newTileServer(t)
	f := fetcherFor(s)

	var mu sync.Mutex
	var calls, last int

	// The callback's own completion, not Pending(), is what this test has to
	// wait on: release decrements the counter and only then calls onChange,
	// so a waitForPending here would be free to look between the two and see
	// a callback that has not run yet. Buffered because nothing is waiting on
	// the prefetch's reports, and blocking the fetching goroutine on a report
	// no one reads would deadlock the very thing under test.
	fired := make(chan int, 1+(2*prefetchRadius+1)*(2*prefetchRadius+1))

	f.SetOnChange(func(pending int) {
		mu.Lock()
		calls++
		last = pending
		mu.Unlock()

		fired <- pending
	})

	f.Warm(48.858222, 2.2945, mapZoom)

	mu.Lock()
	during := calls
	mu.Unlock()

	// Warm returning *is* the prefetch's completion report, so letting
	// every tile in the block report as well would only queue redraws of a
	// map the caller is about to redraw anyway.
	if during != 0 {
		t.Errorf("onChange fired %d times during a prefetch, want none", during)
	}

	// A tile the user pans onto is a different matter: nobody is waiting
	// on it, so its arrival is the only thing that can trigger the redraw.
	req, err := http.NewRequest(http.MethodGet, s.URL+"/15/900/900.png", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	if _, err := f.RoundTrip(req); !errors.Is(err, errTilePending) {
		t.Fatalf("RoundTrip() err = %v, want errTilePending", err)
	}

	select {
	case <-fired:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the background tile to report to onChange")
	}

	mu.Lock()
	defer mu.Unlock()

	if calls != 1 {
		t.Errorf("onChange fired %d times for one background tile, want 1", calls)
	}

	if last != 0 {
		t.Errorf("reported pending = %d, want 0 once the tile is in", last)
	}
}

func TestFetch_FailedTileIsRetriedOnlyAfterTheBackoff(t *testing.T) {
	server := newTileServer(t)
	server.breakIt()
	f := fetcherFor(server)
	now := time.Now()
	f.now = func() time.Time { return now }
	req, err := http.NewRequest(http.MethodGet, server.URL+"/15/2/2.png", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.RoundTrip(req)
	f.Wait()
	_, _ = f.RoundTrip(req)
	f.Wait()
	if server.count() != 1 {
		t.Fatal("failed tile retried during backoff")
	}
	now = now.Add(tileRetryAfter + time.Second)
	_, _ = f.RoundTrip(req)
	f.Wait()
	if server.count() != 2 {
		t.Fatal("failed tile did not retry after backoff")
	}
}

func TestNeighborhood_ClampsToTheEdgeOfTheWorld(t *testing.T) {
	f := newTileFetcher("%d/%d/%d", http.DefaultTransport)

	// Zoom 1 is a 2x2 world, so a radius-2 block around any tile in it is
	// almost entirely off the map.
	got := f.neighborhood(0, 0, 1)

	if len(got) != 4 {
		t.Errorf("neighborhood at zoom 1 = %v (%d tiles), want the whole 2x2 world", got, len(got))
	}

	for _, url := range got {
		if strings.Contains(url, "-") {
			t.Errorf("neighborhood produced a negative tile index: %q", url)
		}
	}
}

func TestTileXY(t *testing.T) {
	cases := []struct {
		name     string
		lat, lon float64
		zoom     int
		x, y     int
	}{
		{"whole world at zoom 0", 48.858222, 2.2945, 0, 0, 0},
		{"origin sits at the seam", 0, 0, 1, 1, 1},
		{"north-west corner", 85.05, -180, 1, 0, 0},
		{"the Eiffel Tower", 48.858222, 2.2945, 15, 16592, 11272},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			x, y := tileXY(c.lat, c.lon, c.zoom)

			if x != c.x || y != c.y {
				t.Errorf("tileXY(%v, %v, %d) = (%d, %d), want (%d, %d)", c.lat, c.lon, c.zoom, x, y, c.x, c.y)
			}
		})
	}
}

// logLines drives lines through a filter the way log.Logger does - one
// Write per line, timestamp prefix and all - and returns what got through.
func logLines(t *testing.T, lines ...string) string {
	t.Helper()

	var out bytes.Buffer
	f := &tileLogFilter{out: &out}

	for _, line := range lines {
		p := []byte("2026/08/19 15:39:17 " + line + "\n")

		n, err := f.Write(p)
		if err != nil {
			t.Fatalf("Write(%q) returned %v", line, err)
		}

		// A filtered write still has to claim the whole line, or the
		// standard logger treats the difference as a short write.
		if n != len(p) {
			t.Errorf("Write(%q) = %d, want %d", line, n, len(p))
		}
	}

	return out.String()
}

func TestTileLogFilter(t *testing.T) {
	const (
		header  = "Fyne error:  tile fetch error"
		pending = `  Cause: Get "https://tile.openstreetmap.org/14/8650/5412.png": tile not downloaded yet`
		at      = "  At: /Users/x/go/pkg/mod/fyne.io/x/fyne@v0/widget/map.go:389"
	)

	t.Run("drops a whole pending-tile block", func(t *testing.T) {
		if got := logLines(t, header, pending, at); got != "" {
			t.Errorf("filter passed %q, want nothing", got)
		}
	})

	t.Run("drops every block of a burst", func(t *testing.T) {
		if got := logLines(t, header, pending, at, header, pending, at); got != "" {
			t.Errorf("filter passed %q, want nothing", got)
		}
	})

	t.Run("passes everything else through", func(t *testing.T) {
		got := logLines(t, "Fyne error:  could not read file", "  Cause: no such file", at)

		for _, want := range []string{"could not read file", "no such file", "map.go:389"} {
			if !strings.Contains(got, want) {
				t.Errorf("filter dropped %q from %q", want, got)
			}
		}
	})

	// Only errTilePending is this package's own noise. A "tile fetch error"
	// from anything else is a real fault, and its cause and location have
	// to survive.
	t.Run("keeps a tile error with another cause", func(t *testing.T) {
		got := logLines(t, header, "  Cause: png: invalid format", at)

		if !strings.Contains(got, "png: invalid format") || !strings.Contains(got, "map.go:389") {
			t.Errorf("filter passed %q, want the cause and location kept", got)
		}
	})

	t.Run("does not swallow a line following a partial block", func(t *testing.T) {
		got := logLines(t, header, pending, "Fyne error:  something else")

		if !strings.Contains(got, "something else") {
			t.Errorf("filter passed %q, want the unrelated error kept", got)
		}
	})

	t.Run("is safe to write to from several goroutines", func(t *testing.T) {
		f := &tileLogFilter{out: &bytes.Buffer{}}

		var wg sync.WaitGroup
		for range 8 {

			wg.Go(func() {

				for range 50 {
					_, _ = f.Write([]byte("Fyne error:  tile fetch error\n"))
				}
			})
		}

		wg.Wait()
	})
}

// The filter's whole design rests on the shape fyne.LogError writes, so
// pin it against the real thing rather than a hand-written imitation: a
// future Fyne that logs a pending tile differently has to be noticed here.
func TestTileLogFilter_SwallowsARealLogErrorCall(t *testing.T) {
	var out bytes.Buffer

	restore := log.Writer()
	log.SetOutput(&tileLogFilter{out: &out})
	t.Cleanup(func() { log.SetOutput(restore) })

	fyne.LogError(tileFetchError, errTilePending)

	if got := out.String(); got != "" {
		t.Errorf("a pending tile logged %q, want nothing", got)
	}

	fyne.LogError("something real", errors.New("boom"))

	if got := out.String(); !strings.Contains(got, "boom") {
		t.Errorf("a real error logged %q, want it kept", got)
	}
}

func TestNewTileFetcher_InstallsTheLogFilter(t *testing.T) {
	newTileFetcher(osmTiles, nil)

	if _, ok := log.Writer().(*tileLogFilter); !ok {
		t.Errorf("log.Writer() is %T, want the tile log filter installed", log.Writer())
	}
}

func TestTileFetcher_OverlappingWarmAndForegroundShareBounds(t *testing.T) {
	body := tilePNG(t)
	entered := make(chan struct{}, 200)
	release := make(chan struct{})
	var active, peak atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := active.Add(1)
		defer active.Add(-1)
		for p := peak.Load(); n > p && !peak.CompareAndSwap(p, n); p = peak.Load() {
		}
		entered <- struct{}{}
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(server.Close)
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	f := newTileFetcher(server.URL+"/%d/%d/%d.png", server.Client().Transport)
	warmDone := make(chan struct{})
	go func() { defer close(warmDone); f.Warm(48.858222, 2.2945, mapZoom) }()
	t.Cleanup(func() { unblock(); <-warmDone })
	for range tileWorkers {
		select {
		case <-entered:
		case <-time.After(5 * time.Second):
			t.Fatal("warm workers did not reach server")
		}
	}
	finished := make(chan struct{}, 1)
	f.SetOnChange(func(_ int) {
		select {
		case finished <- struct{}{}:
		default:
		}
	})
	for i := range 100 {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/foreground/%d", server.URL, i), nil)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = f.RoundTrip(req)
		_, _ = f.RoundTrip(req) // a duplicate never consumes another slot
	}
	if n := f.Pending(); n > tileWorkers+64 {
		t.Fatalf("admitted %d requests, want <=68 (4 active + 64 queued)", n)
	}
	f.mu.Lock()
	queued := len(f.queue)
	f.mu.Unlock()
	if queued > tileQueueCapacity {
		t.Errorf("queued requests=%d, want <=%d", queued, tileQueueCapacity)
	}
	unblock()
	select {
	case <-warmDone:
	case <-time.After(5 * time.Second):
		t.Fatal("warm did not finish")
	}
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("foreground callback did not finish")
	}
	f.Wait()
	if n := peak.Load(); n > tileWorkers {
		t.Errorf("aggregate active requests=%d, want <=%d", n, tileWorkers)
	}
	req, _ := http.NewRequest(http.MethodGet, server.URL+"/foreground/99", nil)
	if _, err := f.RoundTrip(req); !errors.Is(err, errTilePending) {
		t.Fatalf("overflowed tile retry=%v", err)
	}
	f.Wait()
	res, err := f.RoundTrip(req)
	if err != nil {
		t.Fatalf("overflowed tile never retried: %v", err)
	}
	_ = res.Body.Close()
}

func TestTileFetcher_CancelReleasesQueueAndWarmCapacityWaiter(t *testing.T) {
	server := newTileServer(t)
	unblock := server.hold()
	t.Cleanup(unblock)
	f := fetcherFor(server)
	defer func() { unblock(); f.Stop(); f.Wait() }()
	for i := range 100 {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/held/%d", server.URL, i), nil)
		_, _ = f.RoundTrip(req)
	}
	ctx := f.session()
	warmDone := make(chan struct{})
	go func() { defer close(warmDone); f.WarmContext(ctx, 48.858222, 2.2945, mapZoom) }()
	f.Cancel()
	select {
	case <-warmDone:
	case <-time.After(5 * time.Second):
		t.Fatal("cancelled warm waiter did not stop")
	}
	f.Wait()
	f.mu.Lock()
	queued, claims, failures, pending := len(f.queue), len(f.inflight), len(f.failed), f.workers+len(f.queue)
	f.mu.Unlock()
	if queued != 0 || claims != 0 || failures != 0 || pending != 0 {
		t.Errorf("cancel left queue=%d claims=%d failures=%d pending=%d", queued, claims, failures, pending)
	}
	if f.cache.Len() != 0 {
		t.Error("cancelled work cached a response")
	}
	f.Stop()
	if ctx := f.Restart(); ctx.Err() == nil {
		t.Error("terminal stop allowed a restart")
	}
}

func TestTileFetcher_FailureCapacityAndExpiry(t *testing.T) {
	server := newTileServer(t)
	server.breakIt()
	f := fetcherFor(server)
	defer func() { f.Stop(); f.Wait() }()
	now := time.Now()
	f.now = func() time.Time { return now }
	for i := range tileFailureCapacity + 32 {
		req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/failure/%d", server.URL, i), nil)
		_, _ = f.RoundTrip(req)
		f.Wait()
	}
	f.mu.Lock()
	failures := len(f.failed)
	f.mu.Unlock()
	if failures != tileFailureCapacity {
		t.Errorf("failure entries=%d, want %d", failures, tileFailureCapacity)
	}
	now = now.Add(tileRetryAfter + time.Second)
	req, _ := http.NewRequest(http.MethodGet, server.URL+"/new-failure", nil)
	_, _ = f.RoundTrip(req)
	f.Wait()
	f.mu.Lock()
	failures = len(f.failed)
	f.mu.Unlock()
	if failures != 1 {
		t.Errorf("expired failure entries retained: %d", failures)
	}
}

type tileRoundTripFunc func(*http.Request) (*http.Response, error)

func (fn tileRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return fn(req) }

func TestTileFetcher_OldCompletionPreservesReplacementClaim(t *testing.T) {
	for _, oldFailure := range []bool{false, true} {
		t.Run(fmt.Sprintf("old-failure=%v", oldFailure), func(t *testing.T) {
			server := newTileServer(t)
			entered := make(chan int32, 2)
			releases := []chan struct{}{make(chan struct{}), make(chan struct{})}
			unblocks := []func(){sync.OnceFunc(func() { close(releases[0]) }), sync.OnceFunc(func() { close(releases[1]) })}
			var calls atomic.Int32
			base := http.DefaultTransport
			f := newTileFetcher(server.URL+"/%d/%d/%d.png", tileRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				res, err := base.RoundTrip(req)
				i := calls.Add(1) - 1
				entered <- i
				<-releases[i]
				if i == 0 && oldFailure {
					if res != nil {
						_ = res.Body.Close()
					}
					return nil, errors.New("obsolete transport failure")
				}
				return res, err
			}))
			defer func() { unblocks[0](); unblocks[1](); f.Stop(); f.Wait() }()
			url := server.URL + "/same-source"
			old, _ := f.submit(f.session(), url, true)
			select {
			case <-entered:
			case <-time.After(5 * time.Second):
				t.Fatal("old request not entered")
			}
			ctx := f.Restart()
			current, _ := f.submit(ctx, url, true)
			select {
			case <-entered:
			case <-time.After(5 * time.Second):
				t.Fatal("replacement request not entered")
			}
			unblocks[0]()
			select {
			case <-old.done:
			case <-time.After(5 * time.Second):
				t.Fatal("old request did not complete")
			}
			f.mu.Lock()
			claim, failed := f.inflight[url], len(f.failed)
			f.mu.Unlock()
			if claim != current || failed != 0 || f.cache.Contains(url) {
				t.Error("old completion changed replacement claim/facts")
			}
			unblocks[1]()
			select {
			case <-current.done:
			case <-time.After(5 * time.Second):
				t.Fatal("replacement did not complete")
			}
			if !f.cache.Contains(url) {
				t.Error("replacement did not cache its response")
			}
		})
	}
}

func TestTileFetcher_WaitIncludesActiveNotice(t *testing.T) {
	body := tilePNG(t)
	synctest.Test(t, func(t *testing.T) {
		f := newTileFetcher("http://tiles.invalid/%d/%d/%d", tileRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			return tileResponse(req, body), nil
		}))
		entered, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
		f.SetOnChange(func(_ int) { close(entered); <-release })
		job, _ := f.submit(f.session(), "http://tiles.invalid/one", true)
		<-entered
		if f.Pending() != 0 {
			t.Fatal("fixture did not reach the notice after decrement")
		}
		go func() { f.Wait(); close(finished) }()
		synctest.Wait()
		select {
		case <-finished:
			t.Error("Wait ignored an active onChange callback")
		default:
		}
		f.Cancel() // cancellation must return without waiting on the notice
		select {
		case <-job.done:
			t.Error("job completed before its notice")
		default:
		}
		close(release)
		<-finished
		<-job.done
	})
}

func TestTileFetcher_CancelStopsBlockedWarmSubmission(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		entered := make(chan struct{}, tileWorkers)
		f := newTileFetcher("http://tiles.invalid/%d/%d/%d", tileRoundTripFunc(func(req *http.Request) (*http.Response, error) {
			entered <- struct{}{}
			<-req.Context().Done()
			return nil, req.Context().Err()
		}))
		ctx := f.session()
		for i := range 100 {
			_, _ = f.submit(ctx, fmt.Sprintf("http://tiles.invalid/foreground/%d", i), true)
		}
		for range tileWorkers {
			<-entered
		}
		finished := make(chan struct{})
		go func() { defer close(finished); f.WarmContext(ctx, 48.858222, 2.2945, mapZoom) }()
		synctest.Wait()
		select {
		case <-finished:
			t.Fatal("warm did not wait for the full queue")
		default:
		}
		f.Cancel()
		<-finished
		f.Wait()
		if f.Pending() != 0 {
			t.Error("cancelled capacity waiter left admitted work")
		}
	})
}

func TestTileFetcher_ForegroundJoinsWarmJob(t *testing.T) {
	server := newTileServer(t)
	unblock := server.hold()
	t.Cleanup(unblock)
	f := fetcherFor(server)
	defer func() { unblock(); f.Stop(); f.Wait() }()
	url := server.URL + "/shared"
	job, _ := f.submit(f.session(), url, false)
	calls := 0
	f.SetOnChange(func(_ int) { calls++ })
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	_, _ = f.RoundTrip(req)
	_, _ = f.RoundTrip(req)
	unblock()
	<-job.done
	f.Wait()
	if calls != 1 || server.count() != 1 {
		t.Errorf("joined warm job: notices=%d requests=%d, want 1 each", calls, server.count())
	}
}

func TestTileFetcher_ObsoleteActiveRequestDoesNotKeepCurrentLoading(t *testing.T) {
	server := newTileServer(t)
	oldEntered, release := make(chan struct{}), make(chan struct{})
	unblock := sync.OnceFunc(func() { close(release) })
	base := http.DefaultTransport
	f := newTileFetcher(server.URL+"/%d/%d/%d.png", tileRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		res, err := base.RoundTrip(req)
		if req.URL.Path == "/old" {
			close(oldEntered)
			<-release
		}
		return res, err
	}))
	defer func() { unblock(); f.Stop(); f.Wait() }()
	old, _ := f.submit(f.session(), server.URL+"/old", true)
	<-oldEntered
	ctx := f.Restart()
	pending := -1
	f.SetOnChange(func(n int) { pending = n })
	current, _ := f.submit(ctx, server.URL+"/current", true)
	<-current.done
	if pending != 0 || f.Pending() != 0 {
		t.Errorf("obsolete work kept current loading: notice=%d pending=%d", pending, f.Pending())
	}
	select {
	case <-old.done:
		t.Fatal("fixture did not retain the old active worker")
	default:
	}
}
