package ui

import (
	"errors"
	"fmt"
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/clipboard"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/ui/copyselection"
	"github.com/frathe/picfetch/internal/ui/zoom"
)

// regionCopySource is the source presentation captured when the mode begins.
// Raster and RAW inputs retain their oriented displayed frame. SVG retains
// the parsed vector plus logical size and view rotation so the worker can
// rasterize at source resolution instead of the zoom-dependent canvas size.
type regionCopySource struct {
	raster    image.Image
	vector    *imaging.Vector
	logical   image.Point
	rotation  int
	rasterize func(*imaging.Vector, int, int) (image.Image, error)
	animated  bool
}

func (s regionCopySource) pixels() (image.Image, error) {
	if s.vector == nil {
		if s.raster == nil {
			return nil, errors.New("copy selection source is unavailable")
		}
		return s.raster, nil
	}
	if s.logical.X <= 0 || s.logical.Y <= 0 || s.rasterize == nil {
		return nil, errors.New("copy selection vector source is unavailable")
	}

	frame, err := s.rasterize(s.vector, s.logical.X, s.logical.Y)
	if err != nil {
		return nil, err
	}
	return imaging.RotateSteps(frame, s.rotation), nil
}

// cropBounds maps the feature's oriented logical SVG coordinates onto the
// actual raster returned by RasterAt. They are identical below the safety
// ceiling; above it, RasterAt scales both axes down and the crop must follow
// that same scale. Raster sources already use literal pixel coordinates.
func (s regionCopySource) cropBounds(bounds, pixels image.Rectangle) (image.Rectangle, error) {
	if s.vector == nil {
		return bounds, nil
	}

	logical := image.Rect(0, 0, s.logical.X, s.logical.Y)
	if s.rotation%2 != 0 {
		logical = image.Rect(0, 0, s.logical.Y, s.logical.X)
	}
	if logical.Empty() || bounds.Empty() || bounds.Intersect(logical) != bounds || pixels.Empty() {
		return image.Rectangle{}, fmt.Errorf("copy selection bounds %v outside SVG source %v", bounds, logical)
	}

	return image.Rect(
		scaleFloor(bounds.Min.X, logical.Dx(), pixels.Dx()),
		scaleFloor(bounds.Min.Y, logical.Dy(), pixels.Dy()),
		scaleCeil(bounds.Max.X, logical.Dx(), pixels.Dx()),
		scaleCeil(bounds.Max.Y, logical.Dy(), pixels.Dy()),
	), nil
}

func scaleFloor(value, from, to int) int {
	return int(int64(value) * int64(to) / int64(from))
}

func scaleCeil(value, from, to int) int {
	numerator := int64(value) * int64(to)
	return int((numerator + int64(from) - 1) / int64(from))
}

// regionCopyAvailable is the one viewer-side availability rule shared by
// direct activation, the menu snapshot, and the shortcut action. A decoded
// image must be settled in the normal viewer with no modal surface.
func (v *viewer) regionCopyAvailable() bool {
	if v.display.Count() == 0 || v.img.Image == nil || v.loading.Load() {
		return false
	}
	if v.grid.Visible() || v.slides.Active() || v.deletion.Visible() || v.exportPrompt.Visible() {
		return false
	}
	return v.win == nil || v.win.Canvas().Overlays().Top() == nil
}

// startRegionCopy begins a fresh Copy Selection mode. Repeated activation is
// deliberately a no-op; Feature.Start owns the same guard, but keeping it here
// avoids disturbing viewer-owned temporary state before that call.
func (v *viewer) startRegionCopy() {
	if !v.regionCopyAvailable() || v.regionCopy.State().Active {
		return
	}

	source, ok := v.captureRegionCopySource()
	if !ok {
		return
	}
	v.regionCopy.Start(v.regionCopyView(v.zoom.Geometry()))
	if !v.regionCopy.State().Active {
		if source.animated {
			v.animationPause.unpause()
		}
		return
	}
	v.regionCopySource = source
	v.regionCopyInfoVisible = v.info.Object().Visible()
	v.info.Object().Hide()
	v.ForceRepaint()
}

func (v *viewer) captureRegionCopySource() (regionCopySource, bool) {
	if v.vector.svg != nil {
		return regionCopySource{
			vector:    v.vector.svg,
			logical:   image.Pt(int(v.vector.logical.Width+0.5), int(v.vector.logical.Height+0.5)),
			rotation:  v.display.Rotation(),
			rasterize: v.vector.rasterize,
		}, true
	}

	source := regionCopySource{animated: v.display.Count() > 1}
	capture := func() { source.raster = v.display.Rotated() }
	if source.animated {
		if !v.animationPause.pause(capture) {
			return regionCopySource{}, false
		}
	} else {
		capture()
	}
	return source, source.raster != nil
}

// finishRegionCopy restores viewer-owned state after cancellation or a
// successful copy. The feature invokes it only after it has hidden its own
// overlay and cleared its transient state.
func (v *viewer) finishRegionCopy() {
	v.regionCopyLifecycle.invalidate()
	if v.regionCopySource.animated {
		v.animationPause.unpause()
	}
	v.regionCopySource = regionCopySource{}

	if v.regionCopyInfoVisible {
		v.syncInfoOverlayVisibility()
	} else {
		v.info.Object().Hide()
	}
	v.regionCopyInfoVisible = false
	v.ForceRepaint()
}

// cancelRegionCopy is the viewer-owned exit door used by direct lifecycle
// coordination. Busy mode intentionally ignores cancellation inside Feature.
func (v *viewer) cancelRegionCopy() {
	v.regionCopy.Cancel()
}

// cancelRegionCopyBeforeAction cancels Copy Selection so another PicFetch
// command can run. It returns false while a copy worker is pending, and
// that command must not proceed. Zoom and pan must not call this. Window
// close and shutdown remain available while busy.
func (v *viewer) cancelRegionCopyBeforeAction() bool {
	if v.regionCopy != nil && v.regionCopy.State().Busy {
		return false
	}
	v.cancelRegionCopy()
	return true
}

// regionCopyView is the sole adapter from the zoom presentation to the
// selection feature's oriented image coordinate system. displayedDimensions
// supplies logical SVG dimensions and accounts for view-only quarter turns.
func (v *viewer) regionCopyView(geometry zoom.Geometry) copyselection.View {
	if v.display.Count() == 0 || v.img.Image == nil {
		return copyselection.View{}
	}
	w, h := v.displayedDimensions()
	return copyselection.View{
		ImageBounds: image.Rect(0, 0, w, h),
		Position:    geometry.Position,
		Size:        geometry.Size,
	}
}

// copyRegionSelection crops, encodes, and dispatches the captured source off
// the UI thread. The shared clipboard completion signal finishes only after
// the final UI update, so tests and shutdown can wait without sleeping.
func (v *viewer) copyRegionSelection(bounds image.Rectangle) {
	source := v.regionCopySource
	token := v.regionCopyLifecycle.begin()
	done := v.clipboard.Begin()

	go func() {
		defer done()
		defer token.cancelContext()

		if !token.current() {
			return
		}
		var data []byte
		cropBounds := bounds
		pixels, err := source.pixels()
		if err == nil && token.current() {
			cropBounds, err = source.cropBounds(bounds, pixels.Bounds())
		}
		if err == nil && token.current() {
			encode := v.regionCopyEncode
			if encode == nil {
				encode = copyselection.PNG
			}
			data, err = encode(pixels, cropBounds)
		}
		if err == nil && token.current() {
			err = clipboard.CopyImage(data)
		}
		if !token.current() {
			return
		}

		do := v.regionCopyDoAndWait
		if do == nil {
			do = fyne.DoAndWait
		}
		do(func() {
			if !token.current() {
				return
			}
			if err != nil {
				v.reportRegionCopyError(err)
				v.regionCopy.Complete(err)
				v.ForceRepaint()
				return
			}
			v.regionCopy.Complete(nil)
		})
	}()
}

func (v *viewer) reportRegionCopyError(err error) {
	detail := chooserErrorDetail(err)
	fyne.LogError("copy selected image failed", errors.New(detail))
	v.ShowToast(fmt.Sprintf(lang.L("could not copy the image: %v"), detail))
}
