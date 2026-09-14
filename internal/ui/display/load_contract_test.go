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

func loadContract(t *testing.T) {
	t.Run("cache generation", loadCacheGenerationContract)
	t.Run("retired worker", loadRetiredWorkerContract)
	t.Run("failure reentry", loadFailureReentryContract)
	t.Run("success reentry keeps playback", loadSuccessReentryContract)
	cache := imaging.NewImgCache(1024 * 1024)
	first, second := storage.NewFileURI("/first.png"), storage.NewFileURI("/second.png")
	one, two := image.NewNRGBA(image.Rect(0, 0, 2, 3)), image.NewNRGBA(image.Rect(0, 0, 4, 5))
	cache.Add(first.String(), &imaging.LoadedImage{Frames: []image.Image{one}})
	cache.Add(second.String(), &imaging.LoadedImage{Frames: []image.Image{two}})
	queue := &uitest.UIQueue{}
	var f *display.Feature
	var presented []string
	reenter := false
	f = display.New(display.Config{Cache: cache, Queue: queue, Callbacks: display.Callbacks{
		Presented: func(snapshot display.Snapshot) []fyne.URI {
			if f.Surface().Image == nil || snapshot.Displayed.Source == nil {
				t.Fatal("handoff preceded coherent pixels and identity")
			}
			presented = append(presented, snapshot.Displayed.Source.String())
			if reenter && snapshot.Displayed.Source.String() == first.String() {
				f.Load(display.Request{Source: second})
			}
			return nil
		},
		Failed: func(_ fyne.URI, _ error) fyne.URI { return second },
	}})
	defer func() { f.Stop(); f.Wait(); f.Settle() }()
	f.Load(display.Request{Source: first})
	if f.Surface().Image != one || len(presented) != 1 || !f.LoadBegun() {
		t.Fatal("cache hit did not finish synchronously through the handoff")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := f.LoadDone().Wait(ctx); err != nil {
		t.Fatal(err)
	}
	id := f.Snapshot().Displayed
	f.Load(display.Request{Source: storage.NewFileURI(t.TempDir() + "/broken.png")})
	if !f.Snapshot().Loading || f.Snapshot().Displayed != id || f.Surface().Image != one {
		t.Fatal("request discarded or relabeled outgoing content")
	}
	chain := f.LoadDone()
	f.Settle()
	if err := chain.Wait(ctx); err != nil {
		t.Fatal("broken-source chain never completed", err)
	}
	if f.Surface().Image != two || f.Snapshot().Loading {
		t.Fatal("broken-source retry did not complete replacement")
	}
	reenter = true
	f.Load(display.Request{Source: first})
	if f.Surface().Image != two || f.Snapshot().Displayed.Source.String() != second.String() {
		t.Fatal("older handoff overwrote reentrant navigation")
	}
}

func loadSuccessReentryContract(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		cache := imaging.NewImgCache(1024 * 1024)
		first, second := storage.NewFileURI("/still.png"), storage.NewFileURI("/moving.gif")
		pixels := image.NewNRGBA(image.Rect(0, 0, 2, 3))
		cache.Add(first.String(), &imaging.LoadedImage{Frames: []image.Image{pixels}})
		cache.Add(second.String(), &imaging.LoadedImage{Frames: []image.Image{pixels, pixels}, Delays: []time.Duration{time.Second, time.Second}})
		stopped := make(chan struct{})
		ticks := make(chan time.Time)
		var f *display.Feature
		f = display.New(display.Config{Cache: cache, Queue: &uitest.UIQueue{}, AnimationAfter: func(_ time.Duration) <-chan time.Time { return ticks }, Callbacks: display.Callbacks{Presented: func(snapshot display.Snapshot) []fyne.URI {
			if snapshot.Displayed.Source.String() == first.String() {
				f.Load(display.Request{Source: second})
				playback := f.AnimationDone()
				go func() { _ = playback.Wait(context.Background()); close(stopped) }()
			}
			return nil
		}}})
		defer func() { f.Stop(); f.Wait(); f.Settle() }()
		f.Load(display.Request{Source: first})
		synctest.Wait()
		select {
		case <-stopped:
			t.Fatal("obsolete handoff restarted the reentrant presentation's playback")
		default:
		}
	})
}

func loadCacheGenerationContract(t *testing.T) {
	cache := imaging.NewImgCache(1024 * 1024)
	oldBytes, newBytes := uitest.EncodePNG(t, 2, 3, color.Black), uitest.EncodePNG(t, 4, 5, color.White)
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	var reads atomic.Int32
	source := uitest.ReaderURI(storage.NewFileURI("/changing.png"), func() (io.ReadCloser, error) {
		if reads.Add(1) == 1 {
			close(entered)
			<-release
			return io.NopCloser(bytes.NewReader(oldBytes)), nil
		}
		return io.NopCloser(bytes.NewReader(newBytes)), nil
	})
	var probes []image.Point
	f := display.New(display.Config{Cache: cache, Queue: &uitest.UIQueue{}, Callbacks: display.Callbacks{Probed: func(bounds image.Rectangle) { probes = append(probes, bounds.Size()) }}})
	defer func() { once.Do(func() { close(release) }); f.Stop(); f.Wait(); f.Settle() }()
	f.Load(display.Request{Source: source})
	<-entered
	request := f.Snapshot().Requested.Revision
	cache.Purge()
	once.Do(func() { close(release) })
	f.Settle()
	if reads.Load() != 2 || f.Surface().Image.Bounds().Size() != image.Pt(4, 5) || f.Snapshot().Requested.Revision != request {
		t.Fatal("cache invalidation did not retry inside the same navigation")
	}
	if len(probes) != 1 || probes[0] != image.Pt(4, 5) {
		t.Fatalf("stale probe reached root: %v", probes)
	}
}

func loadRetiredWorkerContract(t *testing.T) {
	cache := imaging.NewImgCache(1024 * 1024)
	data := uitest.EncodePNG(t, 2, 3, color.Black)
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	slow := uitest.ReaderURI(storage.NewFileURI("/slow.png"), func() (io.ReadCloser, error) {
		close(entered)
		<-release
		return io.NopCloser(bytes.NewReader(data)), nil
	})
	latest := storage.NewFileURI("/latest.png")
	pixels := image.NewNRGBA(image.Rect(0, 0, 4, 5))
	cache.Add(latest.String(), &imaging.LoadedImage{Frames: []image.Image{pixels}})
	f := display.New(display.Config{Cache: cache, Queue: &uitest.UIQueue{}})
	defer func() { once.Do(func() { close(release) }); f.Stop(); f.Wait(); f.Settle() }()
	f.Load(display.Request{Source: slow})
	<-entered
	retired := f.LoadDone()
	f.Load(display.Request{Source: latest})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := retired.Wait(ctx); err != nil {
		t.Fatal("cancellation did not complete the retired request", err)
	}
	joined := make(chan struct{})
	go func() { f.Wait(); close(joined) }()
	select {
	case <-joined:
		t.Fatal("Wait forgot an older blocked worker")
	default:
	}
	once.Do(func() { close(release) })
	select {
	case <-joined:
	case <-ctx.Done():
		t.Fatal("retired worker did not join")
	}
	f.Settle()
	if f.Surface().Image != pixels || cache.Contains(slow.String()) {
		t.Fatal("retired decode replaced or cached obsolete content")
	}
}

func loadFailureReentryContract(t *testing.T) {
	cache := imaging.NewImgCache(1024 * 1024)
	latest := storage.NewFileURI("/reentered.png")
	pixels := image.NewNRGBA(image.Rect(0, 0, 4, 5))
	cache.Add(latest.String(), &imaging.LoadedImage{Frames: []image.Image{pixels}})
	var f *display.Feature
	f = display.New(display.Config{Cache: cache, Queue: &uitest.UIQueue{}, Callbacks: display.Callbacks{Failed: func(_ fyne.URI, _ error) fyne.URI {
		f.Load(display.Request{Source: latest})
		return storage.NewFileURI("/must-not-retry.png")
	}}})
	defer func() { f.Stop(); f.Wait(); f.Settle() }()
	f.Load(display.Request{Source: storage.NewFileURI(t.TempDir() + "/missing.png")})
	f.Settle()
	if f.Surface().Image != pixels || f.Snapshot().Requested.Source.String() != latest.String() {
		t.Fatal("old failure decision resumed after reentrant navigation")
	}
}
