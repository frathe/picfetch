// The vector re-render: how an SVG stays sharp when the display scale
// moves. internal/imaging owns the parsing and rasterizing; what lives here
// is the policy (is a new raster worth making?), the coalescing, and the
// hand-off back onto the UI goroutine.

package ui

import (
	"image"
	"sync"
	"time"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/imaging"
)

const (
	// vectorSharpenRatio is how much denser a wanted raster must be before
	// it is worth producing. A little above 1 so a slow scroll, which
	// changes the scale a fraction of a percent at a time, doesn't
	// re-render on every frame.
	vectorSharpenRatio = 1.05

	// vectorReleaseRatio is the other side of the band: zooming out does
	// not hurt sharpness at all (a dense raster downscales cleanly), so a
	// re-render happens on the way down only to release memory the zoom
	// level no longer justifies.
	vectorReleaseRatio = 0.5

	// defaultVectorDebounce is long enough to swallow a scroll gesture's
	// burst of events and short enough not to feel laggy. Zero in tests -
	// see viewer.vector.debounce.
	defaultVectorDebounce = 90 * time.Millisecond
)

// vectorView is the whole state of the SVG re-render: the parsed document,
// the two sizes the policy compares, the lifecycle its rasterization runs
// under, and the four write-once seams tests replace. A value field on
// viewer, never copied - it holds a WaitGroup and a lifecycle mutex.
type vectorView struct {
	// svg is the parsed SVG behind the image on screen, nil for every
	// raster format. Non-nil is what makes a scale change mean anything.
	svg *imaging.Vector

	// logical is the size the app treats that vector as being - what
	// the window, the title and the info overlay are built on. Fixed for
	// the lifetime of the loaded image; the raster behind it is not.
	logical fyne.Size

	// raster is the pixel size of the raster currently on screen,
	// which requestVectorRender compares a new target against.
	raster image.Point

	// lifecycle owns debounce and rasterization for the latest SVG
	// render request. A newer scale, image change, clear, or shutdown cancels
	// the previous token and wakes it out of the debounce immediately.
	lifecycle requestLifecycle

	// pending is waited on by the test suite's drain, per the
	// module's concurrency invariant.
	pending sync.WaitGroup

	// debounce coalesces a burst of scroll-driven scale changes into
	// one rasterization. A per-viewer field rather than a package var
	// (concurrency invariant: the viewer has no mutable package state),
	// which is also the seam that lets tests set it to zero.
	debounce time.Duration

	// rasterize, after, and do are RasterAt, time.After, and fyne.Do
	// behind per-viewer seams (the concurrency invariant forbids mutable
	// package state). The coalescing test counts rasterizations and
	// releases a parked burst through the first two; the async-hop test
	// queues the UI callback through do, matching production fyne.Do
	// (async) rather than the test driver (inline). Production never
	// overrides them. Write-once: set at construction, and by a test only
	// before its first drop.
	rasterize func(vec *imaging.Vector, w, h int) (image.Image, error)
	after     func(time.Duration) <-chan time.Time
	do        func(func())
}

// vectorRasterTarget is the pixel size requestVectorRender asks RasterAt
// for: logical size × zoom scale, converted from canvas points to device
// pixels. On macOS fyne.Canvas.Scale() is always 1 and Retina lives in a
// private texScale; PixelCoordinateForPosition is the conversion
// Fyne's own SVG rasterizer and internal/ui/spiral already use. A nil
// toPixels (no canvas yet) rounds the point size, matching the old
// logical*scale arithmetic.
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

// requestVectorRender is zoom's onScaleChanged handler: the effective
// display scale just moved, from a key, a scroll, or a window resize while
// fitting.
//
// It runs inside the zoom renderer's Layout, so it must not touch a widget
// synchronously - see zoom's package doc. It only reads viewer state and
// spawns; the pixels land later, through fyne.Do.
func (v *viewer) requestVectorRender(scale float32) {
	if v.vector.svg == nil || scale <= 0 || v.vector.logical.Width <= 0 {
		return
	}

	var toPixels func(fyne.Position) (int, int)
	if v.win != nil {
		if c := v.win.Canvas(); c != nil {
			toPixels = c.PixelCoordinateForPosition
		}
	}
	w, h := vectorRasterTarget(v.vector.logical, scale, toPixels)
	if w <= 0 || h <= 0 {
		return
	}

	// No rotation adjustment here on purpose. The raster is produced in
	// unrotated space and redrawRotatedFrame turns it afterwards, and a
	// quarter turn preserves pixel count - so a raster of the unrotated
	// logical size times the scale rotates to exactly the size zoom lays
	// out. Swapping the axes would instead stretch the drawing, since
	// oksvg's SetTarget scales each axis independently.

	if !vectorNeedsRender(v.vector.raster, image.Pt(w, h)) {
		return
	}

	token := v.vector.lifecycle.begin()
	v.vector.pending.Add(1)

	go v.rasterizeVector(v.vector.svg, w, h, token)
}

// vectorNeedsRender is the hysteresis band described on the two ratio
// constants above. Comparing one axis is enough: ClampVectorRaster and the
// logical size both preserve aspect, so the two move together.
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

// rasterizeVector waits out the debounce, checks it has not been
// superseded, rasterizes, and hands the result back to the UI goroutine.
// Every early return costs nothing: a burst of twenty scale changes spawns
// twenty of these and rasterizes once, because the other nineteen find the
// generation moved on before allocating anything.
func (v *viewer) rasterizeVector(vec *imaging.Vector, w, h int, token requestToken) {
	defer v.vector.pending.Done()

	// Production fyne.Do queues this callback and returns; the test driver
	// runs it inline. Cancelling here on the way out (the pattern that
	// works for finishSort/applyScanResult, which cancel *inside* their
	// fyne.Do) makes token.current() false before the real driver ever
	// applies the frame - zoom grows the window and the Go image is
	// replaced in tests, but the screen never updates. Early returns
	// still release the context; the hop takes that over once queued.
	// TestVectorRasterLandsWhenUIHopIsAsync is the lock.
	handedOff := false
	defer func() {
		if !handedOff {
			token.cancelContext()
		}
	}()

	if v.vector.debounce > 0 {
		select {
		case <-v.vector.after(v.vector.debounce):
		case <-token.context().Done():
			return
		}
	}

	if !token.current() {
		return
	}

	frame, err := v.vector.rasterize(vec, w, h)
	if err != nil {
		// Deliberately silent. There is always a valid, if softer, raster
		// already on screen, and toasting on a zoom step would be noise; a
		// failure during the initial decode is reported by the load path
		// instead.
		return
	}

	do := v.vector.do
	if do == nil {
		do = fyne.Do
	}
	do(func() {
		defer token.cancelContext()

		// Re-checked on this side too: the generation can move between the
		// check above and this callback running.
		if !token.current() || v.vector.svg != vec || v.display.Count() == 0 {
			return
		}

		b := frame.Bounds()

		// Safe only because finishLoad gave a vector its own one-element
		// slice - see the comment there. Writing loaded.Frames would mutate
		// the cached LoadedImage and invalidate its ByteCache weight. The
		// current frame is that slice's only element: a vector never
		// animates, so the display index stays at finishLoad's 0.
		v.display.ReplaceCurrent(frame)
		v.vector.raster = image.Pt(b.Dx(), b.Dy())

		// The one place that writes v.img.Image, which is what makes the
		// re-render compose with a pending rotation for free.
		v.redrawRotatedFrame()

		// finishLoad ForceRepaints after putting pixels on screen for the
		// same reason: Refresh on a nested canvas.Image can miss the
		// registered content tree. Zoom already sized the image; once
		// syncWindowToZoom grows the window to match, Resize is a no-op
		// and will not rebuild the GL texture for the denser raster.
		//
		// Skipped when debounce is zero: that is the test harness
		// (newTestUI), and under the test driver fyne.Do runs this
		// callback on the raster goroutine. Layouting the content tree
		// here would race the test's own zoom.In → syncWindowToZoom
		// Resize. Production debounce is 90ms and fyne.Do is serialized
		// onto the UI goroutine, so the refresh cannot overlap a key
		// handler. img.Refresh inside redrawRotatedFrame still runs.
		if v.vector.debounce > 0 {
			v.ForceRepaint()
		}
	})
	handedOff = true
}

// clear drops the vector state and abandons any re-render in flight, so a
// rasterization started for the previous image can never land on the next.
func (vv *vectorView) clear() {
	vv.svg = nil
	vv.logical = fyne.Size{}
	vv.raster = image.Point{}
	vv.lifecycle.invalidate()
}

// clearVector drops the vector state and abandons any re-render in flight,
// so a rasterization started for the previous image can never land on the
// next one.
func (v *viewer) clearVector() {
	v.vector.clear()
	v.zoom.SetLogicalSize(fyne.Size{})
}
