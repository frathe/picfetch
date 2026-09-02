package compare

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	fynetest "fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/imaging"
)

func TestPrepareRenderSource_BuildsBoundedOverviewWithoutMutatingFrame(t *testing.T) {
	frame := image.NewNRGBA(image.Rect(11, 17, 4107, 1041))
	for y := frame.Bounds().Min.Y; y < frame.Bounds().Max.Y; y++ {
		for x := frame.Bounds().Min.X; x < frame.Bounds().Max.X; x++ {
			frame.SetNRGBA(x, y, color.NRGBA{
				R: uint8(x % 251), G: uint8(y % 241), B: uint8((x + y) % 239), A: uint8(64 + (x+y)%192),
			})
		}
	}
	before := append([]byte(nil), frame.Pix...)

	source, err := prepareRenderSource(context.Background(), frame)
	if err != nil {
		t.Fatalf("prepareRenderSource: %v", err)
	}
	if source.frame != frame {
		t.Fatal("prepared source replaced the canonical decoded frame")
	}
	if got := source.overview.Bounds().Size(); got != image.Pt(1024, 256) {
		t.Errorf("overview size = %v, want 1024x256", got)
	}
	if !bytes.Equal(frame.Pix, before) {
		t.Fatal("overview generation mutated the canonical decoded pixels")
	}
	_, _, _, alpha := source.overview.At(511, 127).RGBA()
	if alpha == 0 || alpha == 0xffff {
		t.Errorf("overview alpha = %#x, want preserved partial transparency", alpha)
	}
}

func TestPrepareRenderSource_ObservesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := prepareRenderSource(ctx, image.NewRGBA(image.Rect(0, 0, 2000, 1000)))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("prepareRenderSource error = %v, want context.Canceled", err)
	}
}

func TestCompareReady_WaitsForBothPreparedOverviews(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	started := make(chan struct{}, 2)
	release := make(chan struct{})
	feature := newReferenceFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return &imaging.LoadedImage{Frames: []image.Image{image.NewRGBA(image.Rect(0, 0, 1600, 800))}}, nil
	}, Callbacks{})
	feature.prepareSource = func(ctx context.Context, frame image.Image) (*renderSource, error) {
		started <- struct{}{}
		select {
		case <-release:
			return prepareRenderSource(ctx, frame)
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	for range 2 {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for overview preparation")
		}
	}
	if feature.Ready() {
		t.Fatal("comparison became ready before overview preparation completed")
	}
	for i := range feature.panes {
		if !feature.panes[i].spinner.Visible() {
			t.Errorf("pane %d spinner hidden while overview preparation is pending", i)
		}
	}

	close(release)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := feature.Settle(ctx); err != nil {
		t.Fatalf("Settle: %v", err)
	}
	if !feature.Ready() {
		t.Fatal("comparison did not become ready after both overviews completed")
	}
	for i, source := range feature.renderSources {
		if source == nil || source.overview == nil {
			t.Errorf("pane %d has no display-ready overview", i)
		}
		if feature.panes[i].spinner.Visible() {
			t.Errorf("pane %d spinner remained visible after overview preparation", i)
		}
	}
}
