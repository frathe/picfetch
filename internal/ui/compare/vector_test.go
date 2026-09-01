package compare

import (
	"context"
	"image"
	"image/color"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/storage"
	fynetest "fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/uitest"
)

type boundsOnlyImage struct{ bounds image.Rectangle }

func (i boundsOnlyImage) ColorModel() color.Model { return color.RGBAModel }
func (i boundsOnlyImage) Bounds() image.Rectangle { return i.bounds }
func (boundsOnlyImage) At(int, int) color.Color   { return color.Transparent }

func visibleComparisonImages(root fyne.CanvasObject) []*canvas.Image {
	var images []*canvas.Image
	var walk func(fyne.CanvasObject)
	walk = func(object fyne.CanvasObject) {
		if raster, ok := object.(*canvas.Image); ok && raster.Image != nil {
			images = append(images, raster)
		}
		switch object := object.(type) {
		case *fyne.Container:
			for _, child := range object.Objects {
				walk(child)
			}
		case *container.Clip:
			walk(object.Content)
		}
	}
	walk(root)
	return images
}

func TestCompareSettle_WaitsForQueuedVectorCompletion(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	vector, err := imaging.DecodeLoaded(context.Background(), uitest.SVGBytes(40, 20), imaging.DefaultImgCacheBytes)
	if err != nil {
		t.Fatalf("decode SVG fixture: %v", err)
	}
	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "left.svg" {
			return vector, nil
		}
		return &imaging.LoadedImage{Frames: []image.Image{image.NewRGBA(image.Rect(0, 0, 40, 20))}}, nil
	}, Callbacks{})
	feature.vectorDebounce = 0
	queue := &uitest.UIQueue{}
	feature.SetUIQueue(queue)
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.svg"),
		storage.NewFileURI("right.png"),
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := feature.Settle(ctx); err != nil {
		t.Fatalf("Settle: %v", err)
	}
	if queue.Len() != 0 {
		t.Fatalf("queued UI completions after Settle = %d, want 0", queue.Len())
	}
	images := visibleComparisonImages(feature.Overlay())
	if len(images) != 2 || images[0].Image.Bounds().Size() != image.Pt(400, 200) {
		t.Fatal("Settle returned before applying the fitted SVG raster")
	}
}

func TestCompareSettle_DrainsVectorReplacementBeforeWaitingForObsoleteTiles(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	vector, err := imaging.DecodeLoaded(context.Background(), uitest.SVGBytes(4096, 2048), imaging.DefaultImgCacheBytes)
	if err != nil {
		t.Fatalf("decode SVG fixture: %v", err)
	}
	feature := newFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "left.svg" {
			return vector, nil
		}
		return &imaging.LoadedImage{Frames: []image.Image{image.NewRGBA(image.Rect(0, 0, 40, 20))}}, nil
	}, Callbacks{}, newShaderPaneRenderer)
	feature.vectorDebounce = 0
	feature.vectorPixels = func(fyne.CanvasObject, fyne.Size) (int, int) { return 2048, 1024 }
	queue := &uitest.UIQueue{}
	feature.SetUIQueue(queue)

	left := feature.panes[0].renderer.(*shaderPaneRenderer)
	oldTileStarted := make(chan struct{})
	var started sync.Once
	left.generateTile = func(ctx context.Context, source *renderSource, key tileKey) (*renderTile, error) {
		if source.frame.Bounds().Dx() == 4096 {
			started.Do(func() { close(oldTileStarted) })
			<-ctx.Done()
			return nil, ctx.Err()
		}
		return generateRenderTile(ctx, source, key)
	}
	feature.vectorRasterize = func(_ *imaging.Vector, width, height int) (image.Image, error) {
		select {
		case <-oldTileStarted:
		case <-time.After(time.Second):
			t.Fatal("obsolete tile worker did not start before vector replacement")
		}
		return image.NewRGBA(image.Rect(0, 0, width, height)), nil
	}

	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.svg"),
		storage.NewFileURI("right.png"),
	})
	t.Cleanup(func() {
		feature.Close()
		cleanupCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = feature.Settle(cleanupCtx)
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := feature.Settle(ctx); err != nil {
		t.Fatalf("Settle waited for an obsolete tile before applying its queued vector replacement: %v", err)
	}
	if got := feature.rendered[0].Bounds().Size(); got != image.Pt(2048, 1024) {
		t.Fatalf("settled vector raster = %v, want 2048x1024 replacement", got)
	}
}

func TestCompareStale_VectorRenderCannotPaintSupersededTarget(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	vector, err := imaging.DecodeLoaded(context.Background(), uitest.SVGBytes(40, 20), imaging.DefaultImgCacheBytes)
	if err != nil {
		t.Fatalf("decode SVG fixture: %v", err)
	}
	oldStarted := make(chan struct{})
	newFinished := make(chan struct{})
	releaseOld := make(chan struct{})

	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "left.svg" {
			return vector, nil
		}
		return &imaging.LoadedImage{Frames: []image.Image{image.NewRGBA(image.Rect(0, 0, 40, 20))}}, nil
	}, Callbacks{})
	feature.vectorDebounce = 0
	queue := &uitest.UIQueue{}
	feature.SetUIQueue(queue)
	feature.vectorRasterize = func(_ *imaging.Vector, width, height int) (image.Image, error) {
		switch image.Pt(width, height) {
		case image.Pt(400, 200):
			close(oldStarted)
			<-releaseOld
		case image.Pt(500, 250):
			defer close(newFinished)
		default:
			t.Errorf("unexpected vector target %dx%d", width, height)
		}
		return image.NewRGBA(image.Rect(0, 0, width, height)), nil
	}
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.svg"),
		storage.NewFileURI("right.png"),
	})
	feature.workers.Wait()
	if !queue.Drain() {
		t.Fatal("comparison load did not queue a UI completion")
	}
	select {
	case <-oldStarted:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for old vector target")
	}

	feature.HandleKey(fyne.KeyPlus)
	select {
	case <-newFinished:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for new vector target")
	}
	close(releaseOld)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := feature.Settle(ctx); err != nil {
		t.Fatalf("Settle: %v", err)
	}

	images := visibleComparisonImages(feature.Overlay())
	if len(images) != 2 {
		t.Fatalf("comparison images = %d, want 2", len(images))
	}
	if got := images[0].Image.Bounds().Size(); got != image.Pt(500, 250) {
		t.Errorf("final SVG raster = %v, want newest 500x250 target", got)
	}
}

func TestCompareCancel_VectorCompletionCannotPaintAfterClose(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	vector, err := imaging.DecodeLoaded(context.Background(), uitest.SVGBytes(40, 20), imaging.DefaultImgCacheBytes)
	if err != nil {
		t.Fatalf("decode SVG fixture: %v", err)
	}
	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "left.svg" {
			return vector, nil
		}
		return &imaging.LoadedImage{Frames: []image.Image{image.NewRGBA(image.Rect(0, 0, 40, 20))}}, nil
	}, Callbacks{})
	feature.vectorDebounce = 0
	queue := &uitest.UIQueue{}
	feature.SetUIQueue(queue)
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.svg"),
		storage.NewFileURI("right.png"),
	})
	feature.workers.Wait()
	if !queue.Drain() {
		t.Fatal("comparison load did not queue a UI completion")
	}
	feature.vectors[0].pending.Wait()
	if queue.Len() == 0 {
		t.Fatal("vector raster did not queue a UI completion")
	}
	feature.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := feature.Settle(ctx); err != nil {
		t.Fatalf("Settle: %v", err)
	}

	if feature.Visible() {
		t.Fatal("comparison became visible after a cancelled vector completion")
	}
	if images := visibleComparisonImages(feature.Overlay()); len(images) != 0 {
		t.Errorf("cancelled vector completion repainted %d comparison images", len(images))
	}
}

func TestCompareVector_RasterTargetHonorsExistingPixelLimit(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	oldLimit := imaging.MaxVectorRasterPixels()
	imaging.SetMaxVectorRasterPixels(8_000_000)
	t.Cleanup(func() { imaging.SetMaxVectorRasterPixels(oldLimit) })
	vector, err := imaging.DecodeLoaded(context.Background(), uitest.SVGBytes(40, 20), imaging.DefaultImgCacheBytes)
	if err != nil {
		t.Fatalf("decode SVG fixture: %v", err)
	}
	requested := image.Point{}
	feature := newReferenceFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "left.svg" {
			return vector, nil
		}
		return &imaging.LoadedImage{Frames: []image.Image{image.NewRGBA(image.Rect(0, 0, 40, 20))}}, nil
	}, Callbacks{})
	feature.vectorDebounce = 0
	feature.vectorPixels = func(fyne.CanvasObject, fyne.Size) (int, int) { return 5000, 2500 }
	feature.vectorRasterize = func(_ *imaging.Vector, width, height int) (image.Image, error) {
		requested = image.Pt(width, height)
		return boundsOnlyImage{bounds: image.Rect(0, 0, width, height)}, nil
	}
	queue := &uitest.UIQueue{}
	feature.SetUIQueue(queue)
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.svg"),
		storage.NewFileURI("right.png"),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := feature.Settle(ctx); err != nil {
		t.Fatalf("Settle: %v", err)
	}

	if requested != image.Pt(4000, 2000) {
		t.Errorf("RasterAt target = %v, want aspect-preserving 4000x2000 clamp", requested)
	}
	images := visibleComparisonImages(feature.Overlay())
	if len(images) != 2 || images[0].Image.Bounds().Size() != image.Pt(4000, 2000) {
		t.Fatal("comparison did not display the vector raster clamped to the existing pixel limit")
	}
}
