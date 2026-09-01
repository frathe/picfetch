package compare

import (
	"image"
	"sync"
	"time"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/imaging"
)

const defaultCompareVectorDebounce = 90 * time.Millisecond

type vectorRasterState struct {
	lifecycle requestLifecycle
	pending   sync.WaitGroup
	raster    image.Point
	requested image.Point
}

func (s *vectorRasterState) setRaster(frame image.Image) {
	s.lifecycle.invalidate()
	s.requested = image.Point{}
	if frame == nil {
		s.raster = image.Point{}
		return
	}
	bounds := frame.Bounds()
	s.raster = image.Pt(bounds.Dx(), bounds.Dy())
}

func (f *Feature) clearVectorRequests() {
	for i := range f.vectors {
		f.vectors[i].lifecycle.invalidate()
		f.vectors[i].requested = image.Point{}
	}
}

func (f *Feature) clearVectorRasters() {
	f.clearVectorRequests()
	for i := range f.vectors {
		f.vectors[i].raster = image.Point{}
	}
}

func displayPixelSize(object fyne.CanvasObject, display fyne.Size) (int, int) {
	position := fyne.NewPos(display.Width, display.Height)
	if app := fyne.CurrentApp(); app != nil && app.Driver() != nil {
		if canvas := app.Driver().CanvasForObject(object); canvas != nil {
			return canvas.PixelCoordinateForPosition(position)
		}
	}
	return int(position.X + 0.5), int(position.Y + 0.5)
}

func (f *Feature) requestVectorRender(index int, display fyne.Size) {
	if !f.active || !f.ready || index < 0 || index >= len(f.loaded) {
		return
	}
	loaded := f.loaded[index]
	if loaded == nil || loaded.Vector == nil || !validViewport(display) {
		return
	}

	width, height := f.vectorPixels(f.panes[index].image, display)
	width, height = imaging.ClampVectorRaster(width, height)
	target := image.Pt(width, height)
	state := &f.vectors[index]
	if state.raster == target {
		if state.requested != (image.Point{}) {
			state.lifecycle.invalidate()
			state.requested = image.Point{}
		}
		return
	}
	if state.requested == target {
		return
	}

	token := state.lifecycle.begin()
	state.requested = target
	state.pending.Add(1)
	go f.rasterizeVector(index, loaded.Vector, target, token)
}

func (f *Feature) rasterizeVector(index int, vector *imaging.Vector, target image.Point, token requestToken) {
	defer f.vectors[index].pending.Done()

	if f.vectorDebounce > 0 {
		select {
		case <-f.vectorAfter(f.vectorDebounce):
		case <-token.context().Done():
			return
		}
	}
	if !token.latest() {
		return
	}

	frame, err := f.vectorRasterize(vector, target.X, target.Y)
	if err != nil || !token.latest() {
		return
	}

	f.queueUI(func() {
		state := &f.vectors[index]
		if !token.latest() || !f.active || !f.ready || f.loaded[index] == nil ||
			f.loaded[index].Vector != vector || state.requested != target {
			return
		}
		f.rendered[index] = frame
		state.setRaster(frame)
		f.panes[index].image.Image = frame
		f.panes[index].image.Refresh()
		f.repaint()
	})
}
