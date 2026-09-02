package compare

import (
	"context"
	"image"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/storage"
	fynetest "fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/imaging"
)

type recordingPaneRenderer struct {
	object *canvas.Rectangle
	scenes []paneScene
	waits  int
}

func newRecordingPaneRenderer() *recordingPaneRenderer {
	return &recordingPaneRenderer{object: canvas.NewRectangle(nil)}
}

func (r *recordingPaneRenderer) Object() fyne.CanvasObject { return r.object }

func (r *recordingPaneRenderer) Present(scene paneScene) {
	r.scenes = append(r.scenes, scene)
}

func (r *recordingPaneRenderer) Wait(_ context.Context) error {
	r.waits++
	return nil
}

func (r *recordingPaneRenderer) latest(t *testing.T) paneScene {
	t.Helper()
	if len(r.scenes) == 0 {
		t.Fatal("renderer received no scene")
	}
	return r.scenes[len(r.scenes)-1]
}

func TestPaneRendererScene_PresentsStableSourceGeometryAndLifecycle(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	renderers := [2]*recordingPaneRenderer{
		newRecordingPaneRenderer(),
		newRecordingPaneRenderer(),
	}
	objects := [2]fyne.CanvasObject{renderers[0].Object(), renderers[1].Object()}
	feature := newFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		size := image.Pt(1200, 600)
		if uri.Name() == "right.png" {
			size = image.Pt(300, 900)
		}
		return &imaging.LoadedImage{Frames: []image.Image{image.NewRGBA(image.Rectangle{Max: size})}}, nil
	}, Callbacks{}, func(index int) paneRenderer { return renderers[index] })
	pixelAnchors := map[fyne.CanvasObject]bool{
		feature.panes[0].input: true,
		feature.panes[1].input: true,
	}
	feature.vectorPixels = func(object fyne.CanvasObject, _ fyne.Size) (int, int) {
		if !pixelAnchors[object] {
			t.Errorf("physical-pixel lookup anchor = %T, want a visible pane input", object)
		}
		return 800, 400
	}
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := feature.Settle(ctx); err != nil {
		t.Fatalf("Settle after Open: %v", err)
	}

	left := renderers[0].latest(t)
	if left.source == nil || left.source.frame.Bounds().Size() != image.Pt(1200, 600) {
		t.Fatalf("left scene source = %#v, want 1200x600", left.source)
	}
	if left.viewport != fyne.NewSize(400, 400) || left.imagePosition != fyne.NewPos(0, 100) || left.imageSize != fyne.NewSize(400, 200) {
		t.Errorf("left scene geometry = viewport %v position %v size %v, want 400x400, (0,100), 400x200",
			left.viewport, left.imagePosition, left.imageSize)
	}
	if left.displaySize != image.Pt(800, 400) {
		t.Errorf("left physical display size = %v, want 800x400", left.displaySize)
	}

	feature.HandleKey(fyne.KeyPlus)
	zoomed := renderers[0].latest(t)
	if zoomed.imageSize != fyne.NewSize(500, 250) {
		t.Errorf("zoomed left scene size = %v, want 500x250", zoomed.imageSize)
	}
	for i, renderer := range renderers {
		if renderer.Object() != objects[i] {
			t.Errorf("pane %d renderer object changed across interaction", i)
		}
	}

	feature.layoutMode = swipe
	feature.dividerAt = 0.25
	feature.layoutSwipe(fyne.NewSize(800, 400))
	leftReveal := renderers[0].latest(t)
	rightReveal := renderers[1].latest(t)
	if !leftReveal.revealSet || leftReveal.revealPosition != (fyne.Position{}) || leftReveal.revealSize != fyne.NewSize(200, 400) {
		t.Errorf("left swipe reveal = set %v position %v size %v, want true (0,0) 200x400",
			leftReveal.revealSet, leftReveal.revealPosition, leftReveal.revealSize)
	}
	if !rightReveal.revealSet || rightReveal.revealPosition != fyne.NewPos(200, 0) || rightReveal.revealSize != fyne.NewSize(600, 400) {
		t.Errorf("right swipe reveal = set %v position %v size %v, want true (200,0) 600x400",
			rightReveal.revealSet, rightReveal.revealPosition, rightReveal.revealSize)
	}

	feature.swapSides()
	left = renderers[0].latest(t)
	if left.source == nil || left.source.frame.Bounds().Size() != image.Pt(300, 900) {
		t.Fatalf("left source after Swap = %#v, want former right 300x900", left.source)
	}

	feature.Close()
	for i, renderer := range renderers {
		if scene := renderer.latest(t); scene.source != nil {
			t.Errorf("pane %d source after Close = %#v, want nil", i, scene.source)
		}
		if renderer.waits == 0 {
			t.Errorf("pane %d renderer Wait was not included in Settle", i)
		}
	}
}

func TestPaneRendererScene_DividerMoveRepublishesRevealWithoutTransform(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	renderers := [2]*recordingPaneRenderer{
		newRecordingPaneRenderer(),
		newRecordingPaneRenderer(),
	}
	feature := newFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return &imaging.LoadedImage{Frames: []image.Image{image.NewRGBA(image.Rect(0, 0, 1600, 800))}}, nil
	}, Callbacks{}, func(index int) paneRenderer { return renderers[index] })
	feature.vectorPixels = func(fyne.CanvasObject, fyne.Size) (int, int) { return 800, 400 }
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := feature.Settle(ctx); err != nil {
		t.Fatalf("Settle after Open: %v", err)
	}
	feature.layoutMode = swipe
	feature.layoutSwipe(fyne.NewSize(800, 400))

	before := [2]paneScene{renderers[0].latest(t), renderers[1].latest(t)}
	beforeCounts := [2]int{len(renderers[0].scenes), len(renderers[1].scenes)}
	feature.HandleKey(fyne.KeyEnd)

	wantReveals := [2]struct {
		position fyne.Position
		size     fyne.Size
	}{
		{size: fyne.NewSize(800, 400)},
		{position: fyne.NewPos(800, 0), size: fyne.NewSize(0, 400)},
	}
	for i, renderer := range renderers {
		if got := len(renderer.scenes); got != beforeCounts[i]+1 {
			t.Errorf("pane %d presentations after divider move = %d, want %d", i, got, beforeCounts[i]+1)
		}
		got := renderer.latest(t)
		want := before[i]
		want.revealPosition = wantReveals[i].position
		want.revealSize = wantReveals[i].size
		want.panePosition = displayPixelPosition(feature.panes[i].root)
		if got != want {
			t.Errorf("pane %d scene after divider move = %+v, want reveal update %+v", i, got, want)
		}
	}
}

func TestNew_UsesTiledShaderRenderersForBothPanes(t *testing.T) {
	feature := New(nil, Callbacks{})
	for i := range feature.panes {
		if _, ok := feature.panes[i].renderer.(*shaderPaneRenderer); !ok {
			t.Errorf("pane %d renderer type = %T, want *shaderPaneRenderer", i, feature.panes[i].renderer)
		}
	}
}
