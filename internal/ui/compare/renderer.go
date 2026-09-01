package compare

import (
	"context"
	"errors"
	"image"
	"image/draw"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"

	xdraw "golang.org/x/image/draw"

	"github.com/frathe/picfetch/internal/imaging"
)

const overviewMaxDimension = 1024

// renderSource is the immutable image identity handed to a pane renderer. It
// retains the canonical decoded frame, a bounded overview, and the source's
// byte-budgeted detail-tile cache.
type renderSource struct {
	frame    image.Image
	overview image.Image
	tiles    *imaging.ByteCache[*renderTile]
}

func prepareRenderSource(ctx context.Context, frame image.Image) (*renderSource, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if frame == nil {
		return nil, errors.New("decoded image has no frame")
	}
	bounds := frame.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return nil, errors.New("decoded image has invalid dimensions")
	}
	if width <= overviewMaxDimension && height <= overviewMaxDimension {
		return newPreparedRenderSource(frame, frame), nil
	}

	scale := float64(overviewMaxDimension) / float64(max(width, height))
	overviewWidth := max(1, int(math.Round(float64(width)*scale)))
	overviewHeight := max(1, int(math.Round(float64(height)*scale)))
	overview := image.NewRGBA(image.Rect(0, 0, overviewWidth, overviewHeight))
	xdraw.ApproxBiLinear.Scale(overview, overview.Bounds(), frame, bounds, draw.Src, nil)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return newPreparedRenderSource(frame, overview), nil
}

func newPreparedRenderSource(frame, overview image.Image) *renderSource {
	return &renderSource{
		frame:    frame,
		overview: overview,
		tiles: imaging.NewByteCache(tileCacheBudgetBytes, func(tile *renderTile) int64 {
			if tile == nil || tile.texture == nil {
				return 0
			}
			return int64(len(tile.texture.Pix))
		}),
	}
}

// paneScene is one complete presentation snapshot. Logical geometry stays in
// Fyne points while displaySize records the corresponding physical pixels for
// mip selection and vector raster targets.
type paneScene struct {
	source         *renderSource
	viewport       fyne.Size
	revealSet      bool
	revealPosition fyne.Position
	revealSize     fyne.Size
	imagePosition  fyne.Position
	imageSize      fyne.Size
	displaySize    image.Point
}

// paneRenderer is the private boundary between comparison transform policy and
// a concrete canvas implementation. Present runs on the UI path; Wait makes
// renderer-owned asynchronous work observable to Feature.Settle.
type paneRenderer interface {
	Object() fyne.CanvasObject
	Present(paneScene)
	Wait(context.Context) error
}

type paneRendererFactory func(index int) paneRenderer

type paneRendererQueue interface {
	setQueueUI(func(func()))
}

// canvasPaneRenderer is the deterministic reference adapter used by unit tests
// that exercise comparison geometry and source identity. Production uses the
// tiled shader adapter.
type canvasPaneRenderer struct {
	image *canvas.Image
}

func newCanvasPaneRenderer(int) paneRenderer {
	img := canvas.NewImageFromImage(nil)
	img.FillMode = canvas.ImageFillContain
	img.ScaleMode = canvas.ImageScaleSmooth
	img.Hide()
	return &canvasPaneRenderer{image: img}
}

func (r *canvasPaneRenderer) Object() fyne.CanvasObject { return r.image }

func (r *canvasPaneRenderer) Present(scene paneScene) {
	if scene.source == nil || scene.source.frame == nil {
		r.image.Image = nil
		r.image.Hide()
		return
	}

	changed := r.image.Image != scene.source.frame
	r.image.Image = scene.source.frame
	r.image.Resize(scene.imageSize)
	r.image.Move(scene.imagePosition)
	r.image.Show()
	if changed {
		r.image.Refresh()
	}
}

func (*canvasPaneRenderer) Wait(context.Context) error { return nil }
