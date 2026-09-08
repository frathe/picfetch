package favthumbs

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/uitest"
)

// BenchmarkPreviewForegroundContention measures the independent Sync workers
// competing with foreground thumbnail reads/decodes on the Go scheduler. It
// excludes fixture creation, cache priming and verification from timings. Cold
// means no application previews; source bytes may be in the OS page cache.
// B/op and allocs/op include BOTH workloads, not just the foreground request.
func BenchmarkPreviewForegroundContention(b *testing.B) {
	const sources, foreground = 64, 8
	dir := b.TempDir()
	pixels := image.NewRGBA(image.Rect(0, 0, 3072, 2048))
	var seed uint32 = 1
	for y := range 2048 {
		for x := range 3072 {
			seed = 1664525*seed + 1013904223
			pixels.SetRGBA(x, y, color.RGBA{R: uint8(x / 12), G: uint8(y / 8), B: uint8(seed >> 24), A: 255})
		}
	}
	var encoded bytes.Buffer
	if err := jpeg.Encode(&encoded, pixels, &jpeg.Options{Quality: 85}); err != nil {
		b.Fatal(err)
	}
	files := make([]fyne.URI, sources+foreground)
	for i := range files {
		path := filepath.Join(dir, fmt.Sprintf("%03d.jpg", i))
		if err := os.WriteFile(path, encoded.Bytes(), 0600); err != nil {
			b.Fatal(err)
		}
		files[i] = storage.NewFileURI(path)
	}
	b.Logf("%s/%s %s: %d preview + %d foreground sources, JPEG 3072x2048 quality 85, %d encoded bytes each", runtime.GOOS, runtime.GOARCH, runtime.Version(), sources, foreground, encoded.Len())
	for _, procs := range slices.Compact([]int{2, runtime.GOMAXPROCS(0)}) {
		b.Run(fmt.Sprintf("P%d", procs), func(b *testing.B) {
			for _, condition := range []string{"alone", "cold", "disk-warm"} {
				b.Run(condition, func(b *testing.B) {
					previous := runtime.GOMAXPROCS(procs)
					defer runtime.GOMAXPROCS(previous)
					benchmarkPreviewContention(b, files[:sources], files[sources:], condition)
					b.ReportMetric(float64(runtime.GOMAXPROCS(0)), "processors")
				})
			}
		})
	}
}

// The sink parks only the first admitted worker, before cache lookup, so the
// foreground starts while a real pass is alive. It does not invent a shared
// semaphore or constrain subsequent worker scheduling. The bounded ByteCache
// models the grid sink's stop-offering-when-full policy.
type contentionSink struct {
	cache   *imaging.ByteCache[image.Image]
	entered chan struct{}
	release chan struct{}
	once    sync.Once
	stores  atomic.Int64
}

func (s *contentionSink) Cached(u fyne.URI) (image.Image, bool) {
	s.once.Do(func() {
		close(s.entered)
		<-s.release
	})
	return s.cache.Get(u.String())
}

func (s *contentionSink) Store(u fyne.URI, img image.Image) {
	s.stores.Add(1)
	if s.cache.Bytes() < s.cache.Budget() {
		_ = s.cache.AddIfFits(u.String(), img)
	}
}

func benchmarkPreviewContention(b *testing.B, sources, foreground []fyne.URI, condition string) {
	b.StopTimer()
	favDir := b.TempDir()
	var opens atomic.Int64
	files := make([]fyne.URI, len(sources))
	for i, u := range sources {
		files[i] = uitest.ReaderURI(u, func() (io.ReadCloser, error) {
			opens.Add(1)
			return os.Open(u.Path())
		})
	}
	if condition == "disk-warm" {
		if err := Sync(context.Background(), favDir, files, nil); err != nil {
			b.Fatal(err)
		}
	}
	var latencies []float64
	var foregroundTime, previewTime time.Duration
	var overlaps, sourceOpens int64
	b.ResetTimer()
	for range b.N {
		if condition == "cold" {
			if err := os.RemoveAll(Dir(favDir)); err != nil {
				b.Fatal(err)
			}
		}
		opens.Store(0)
		sink := &contentionSink{
			cache:   imaging.NewThumbCache(1 << 20),
			entered: make(chan struct{}), release: make(chan struct{}),
		}
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		started := time.Now()
		var finished time.Time
		b.StartTimer()
		if condition != "alone" {
			go func() {
				err := Sync(ctx, favDir, files, sink)
				finished = time.Now()
				done <- err
			}()
			select {
			case <-sink.entered:
			case err := <-done:
				cancel()
				b.Fatalf("preview pass never admitted a source: %v", err)
			}
		}
		close(sink.release)
		var foregroundErr error
		for _, u := range foreground {
			if condition != "alone" && len(done) == 0 {
				overlaps++
			}
			start := time.Now()
			thumb, err := imaging.LoadThumbnailContext(ctx, u)
			elapsed := time.Since(start)
			foregroundTime += elapsed
			latencies = append(latencies, float64(elapsed.Nanoseconds()))
			if err != nil || thumb.Bounds().Dx() != imaging.ThumbnailSize {
				foregroundErr = fmt.Errorf("foreground thumbnail: %v", err)
				cancel()
				break
			}
		}
		var previewErr error
		if condition != "alone" {
			previewErr = <-done
			previewTime += finished.Sub(started)
		}
		cancel()
		b.StopTimer()
		if foregroundErr != nil || previewErr != nil {
			b.Fatalf("foreground=%v preview=%v", foregroundErr, previewErr)
		}
		sourceOpens += opens.Load()
		if condition != "alone" {
			if sink.stores.Load() != int64(len(files)) || sink.cache.Bytes() > sink.cache.Budget() {
				b.Fatal("preview pass did not converge within the memory budget")
			}
			for _, u := range files {
				if _, ok, err := ReadContext(context.Background(), favDir, u); err != nil || !ok {
					b.Fatalf("missing readable preview for %s: %v", u, err)
				}
			}
			wantOpens := int64(0)
			if condition == "cold" {
				wantOpens = int64(len(files))
			}
			if opens.Load() != wantOpens {
				b.Fatalf("%s source opens = %d, want %d", condition, opens.Load(), wantOpens)
			}
		}
	}
	slices.Sort(latencies)
	b.ReportMetric(latencies[len(latencies)/2]/1e6, "fg-p50-ms")
	b.ReportMetric(latencies[(len(latencies)*95+99)/100-1]/1e6, "fg-p95-ms")
	b.ReportMetric(float64(len(latencies))/foregroundTime.Seconds(), "fg/s")
	b.ReportMetric(float64(overlaps)/float64(b.N), "overlapping-fg/op")
	b.ReportMetric(float64(sourceOpens)/float64(b.N), "preview-source-reads/op")
	if condition != "alone" {
		b.ReportMetric(float64(previewTime.Milliseconds())/float64(b.N), "converged-ms/op")
		b.ReportMetric(float64(len(files)*b.N)/previewTime.Seconds(), "previews/s")
	}
}
