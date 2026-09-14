package display_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/ui/display"
	"github.com/frathe/picfetch/internal/uitest"
)

func preloadContract(t *testing.T) {
	t.Run("join retired workers", preloadJoinContract)
	t.Run("cache admission", preloadCacheContract)
	data := uitest.EncodePNG(t, 2, 2, color.Black)
	entered, release := make(chan struct{}, 4), make(chan struct{})
	var once sync.Once
	var active, peak, reads atomic.Int32
	var neighbors []fyne.URI
	for _, name := range []string{"a.png", "b.png", "c.png"} {
		neighbors = append(neighbors, uitest.ReaderURI(storage.NewFileURI("/"+name), func() (io.ReadCloser, error) {
			reads.Add(1)
			n := active.Add(1)
			for {
				old := peak.Load()
				if old >= n || peak.CompareAndSwap(old, n) {
					break
				}
			}
			entered <- struct{}{}
			<-release
			active.Add(-1)
			return io.NopCloser(bytes.NewReader(data)), nil
		}))
	}
	neighbors = append(neighbors, neighbors[0])
	cache := imaging.NewImgCache(1024 * 1024)
	source := storage.NewFileURI("/main.png")
	cache.Add(source.String(), &imaging.LoadedImage{Frames: []image.Image{image.NewNRGBA(image.Rect(0, 0, 2, 2))}})
	f := display.New(display.Config{Cache: cache, Queue: &uitest.UIQueue{}, Callbacks: display.Callbacks{Presented: func(_ display.Snapshot) []fyne.URI { return neighbors }}})
	defer func() { once.Do(func() { close(release) }); f.Stop(); f.Wait(); f.Settle() }()
	f.Load(display.Request{Source: source})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := f.LoadDone().Wait(ctx); err != nil {
		t.Fatal("load completion waited for speculative reads")
	}
	for range 2 {
		select {
		case <-entered:
		case <-ctx.Done():
			t.Fatal("neighbor preparation was not admitted")
		}
	}
	once.Do(func() { close(release) })
	f.Settle()
	if reads.Load() != 3 || peak.Load() != 2 {
		t.Fatalf("preloads read %d sources with peak %d, want unique sources and two workers", reads.Load(), peak.Load())
	}
	for _, neighbor := range neighbors {
		if !cache.Contains(neighbor.String()) {
			t.Fatal("completed neighbor was not cached")
		}
	}
}

func preloadJoinContract(t *testing.T) {
	data := uitest.EncodePNG(t, 2, 2, color.Black)
	synctest.Test(t, func(t *testing.T) {
		entered, release := make(chan struct{}), make(chan struct{})
		var once sync.Once
		defer once.Do(func() { close(release) })
		neighbor := uitest.ReaderURI(storage.NewFileURI("/held-neighbor.png"), func() (io.ReadCloser, error) {
			close(entered)
			<-release
			return io.NopCloser(bytes.NewReader(data)), nil
		})
		f := newPresentation(t, display.Config{Callbacks: display.Callbacks{Presented: func(_ display.Snapshot) []fyne.URI { return []fyne.URI{neighbor} }}})
		f.Present(storage.NewFileURI("/main.png"), &imaging.LoadedImage{Frames: []image.Image{image.NewNRGBA(image.Rect(0, 0, 2, 2))}}, false)
		<-entered
		f.Stop()
		joined := make(chan struct{})
		go func() { f.Wait(); close(joined) }()
		synctest.Wait()
		select {
		case <-joined:
			t.Error("Wait forgot a retired speculative worker")
		default:
		}
		once.Do(func() { close(release) })
		<-joined
		f.Settle()
		if f.cache.Contains(neighbor.String()) {
			t.Fatal("cancelled preload published a cache record")
		}
	})
}

func preloadCacheContract(t *testing.T) {
	t.Run("presence does not promote", func(t *testing.T) {
		cache := imaging.NewImgCache(32)
		main, neighbor, third := storage.NewFileURI("/main.png"), storage.NewFileURI("/neighbor.png"), storage.NewFileURI("/third.png")
		record := &imaging.LoadedImage{Frames: []image.Image{image.NewNRGBA(image.Rect(0, 0, 2, 2))}}
		cache.Add(neighbor.String(), record)
		cache.Add(main.String(), record)
		f := newPresentation(t, display.Config{Cache: cache, Callbacks: display.Callbacks{Presented: func(_ display.Snapshot) []fyne.URI { return []fyne.URI{neighbor} }}})
		f.Load(display.Request{Source: main})
		f.Settle()
		cache.Add(third.String(), record)
		if !cache.Contains(main.String()) || cache.Contains(neighbor.String()) {
			t.Fatal("speculative presence check promoted a neighbor")
		}
	})
	t.Run("speculation does not evict", func(t *testing.T) {
		data := uitest.EncodePNG(t, 2, 2, color.Black)
		neighbor := uitest.ReaderURI(storage.NewFileURI("/neighbor.png"), func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(data)), nil })
		main := storage.NewFileURI("/main.png")
		f := newPresentation(t, display.Config{Cache: imaging.NewImgCache(64), Callbacks: display.Callbacks{Presented: func(_ display.Snapshot) []fyne.URI { return []fyne.URI{neighbor} }}})
		f.Present(main, &imaging.LoadedImage{Frames: []image.Image{image.NewNRGBA(image.Rect(0, 0, 4, 4))}}, false)
		f.Settle()
		if !f.cache.Contains(main.String()) || f.cache.Contains(neighbor.String()) {
			t.Fatal("speculative admission evicted the displayed image")
		}
	})
}
