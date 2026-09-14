// SVG presentation policy and asynchronous raster delivery.

package display

import (
	"image"
	"sync"
	"time"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/imaging"
)

const (
	vectorSharpenRatio    = 1.05
	vectorReleaseRatio    = 0.5
	defaultVectorDebounce = 90 * time.Millisecond
)

// VectorOptions configures rasterization and debounce before presentation starts.
type VectorOptions struct {
	Debounce  time.Duration
	Rasterize func(*imaging.Vector, int, int) (image.Image, error)
	After     func(time.Duration) <-chan time.Time
}

type vectorView struct {
	svg       *imaging.Vector
	raster    image.Point
	lifecycle requestLifecycle
	pending   sync.WaitGroup
	options   VectorOptions
}

func (f *Feature) SetVectorOptions(options VectorOptions) {
	if options.Rasterize == nil {
		options.Rasterize = func(vec *imaging.Vector, w, h int) (image.Image, error) { return vec.RasterAt(w, h) }
	}
	if options.After == nil {
		options.After = time.After
	}
	f.vector.options = options
}

func vectorRasterTarget(logical fyne.Size, scale float32, toPixels func(fyne.Position) (int, int)) (w, h int) {
	if scale <= 0 || logical.Width <= 0 || logical.Height <= 0 {
		return 0, 0
	}

	pos := fyne.NewPos(logical.Width*scale, logical.Height*scale)
	if toPixels == nil {
		w, h = int(pos.X+0.5), int(pos.Y+0.5)
	} else {
		w, h = toPixels(pos)
	}
	if w <= 0 || h <= 0 {
		return 0, 0
	}

	return imaging.ClampVectorRaster(w, h)
}

func vectorNeedsRender(have, want image.Point) bool {
	if have.X <= 0 {
		return true // nothing on screen yet - any raster beats none
	}

	// Unreachable from requestVectorRender (ClampVectorRaster floors at
	// 1), but the safe answer for a degenerate target is "no", not a
	// commission to rasterize at zero size.
	if want.X <= 0 {
		return false
	}

	ratio := float64(want.X) / float64(have.X)

	return ratio > vectorSharpenRatio || ratio < vectorReleaseRatio
}

// RequestVectorRender can run inside zoom Layout. It records an intent and
// starts work; publication always crosses the instance UI queue.
func (f *Feature) RequestVectorRender(scale float32, toPixels func(fyne.Position) (int, int)) {
	if f.stopped || f.vector.svg == nil {
		return
	}
	w, h := vectorRasterTarget(f.logical, scale, toPixels)
	if w <= 0 || h <= 0 || !vectorNeedsRender(f.vector.raster, image.Pt(w, h)) {
		return
	}
	token := f.vector.lifecycle.begin()
	vec, identity := f.vector.svg, f.snapshot.Displayed.Revision
	options := f.vector.options
	f.vector.pending.Go(func() {
		handedOff := false
		defer func() {
			if !handedOff {
				token.cancelContext()
			}
		}()
		if options.Debounce > 0 {
			select {
			case <-options.After(options.Debounce):
			case <-token.context().Done():
				return
			}
		}
		if !token.current() {
			return
		}
		frame, err := options.Rasterize(vec, w, h)
		if err != nil {
			return
		} // Retain the valid softer image on sharpening failure.
		f.config.Queue.Do(func() {
			defer token.cancelContext()
			if !token.current() || f.vector.svg != vec || f.snapshot.Displayed.Revision != identity {
				return
			}
			f.frames[f.index] = frame
			f.vector.raster = frame.Bounds().Size()
			f.publish()
			if f.config.Callbacks.Repaint != nil {
				f.config.Callbacks.Repaint()
			}
		})
		handedOff = true
	})
}

func (f *Feature) clearVector() {
	f.vector.lifecycle.invalidate()
	f.vector.svg = nil
	f.vector.raster = image.Point{}
}
