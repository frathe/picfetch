package ui

import (
	"errors"
	"fmt"
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/clipboard"
	"github.com/frathe/picfetch/internal/ui/copyselection"
	"github.com/frathe/picfetch/internal/ui/zoom"
)

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

	source, animated, ok := v.captureRegionCopySource()
	if !ok {
		return
	}
	v.regionCopy.Start(v.regionCopyView(v.zoom.Geometry(), source), source)
	if !v.regionCopy.State().Active {
		if animated {
			v.animationPause.unpause()
		}
		return
	}
	v.regionCopyAnimated = animated
	v.regionCopyInfoVisible = v.info.Object().Visible()
	v.info.Object().Hide()
	v.ForceRepaint()
}

func (v *viewer) captureRegionCopySource() (source copyselection.Source, animated bool, ok bool) {
	if v.vector.svg != nil {
		w, h := roundedLogical(v.vector.logical)
		return copyselection.VectorSource(
			v.vector.svg,
			image.Pt(w, h),
			v.display.Rotation(),
			v.vector.rasterize,
		), false, true
	}

	animated = v.display.Count() > 1
	var raster image.Image
	// v.img.Image is the displayed oriented frame redrawRotatedFrame keeps
	// current for every raster path. Capturing it instead of re-running
	// display.Rotated() avoids a second full-size RGBA that the Source
	// would pin for the whole mode. It stays stable while the mode is
	// active: animations are paused right here, and every rotation or
	// navigation path yields the mode before touching the frame.
	capture := func() { raster = v.img.Image }
	if animated {
		if !v.animationPause.pause(capture) {
			return copyselection.Source{}, false, false
		}
	} else {
		capture()
	}
	if raster == nil {
		// Release what this function acquired: the caller cleans up only
		// after a failed Start, not after a failed capture.
		if animated {
			v.animationPause.unpause()
		}
		return copyselection.Source{}, false, false
	}
	return copyselection.RasterSource(raster), animated, true
}

// finishRegionCopy restores viewer-owned state after cancellation or a
// successful copy. The feature invokes it only after it has hidden its own
// overlay and cleared its transient state.
func (v *viewer) finishRegionCopy() {
	v.regionCopyLifecycle.invalidate()
	if v.regionCopyAnimated {
		v.animationPause.unpause()
	}
	v.regionCopyAnimated = false

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

// yieldCopySelection lets another PicFetch command run. A pending copy
// blocks that command — with a toast, so the refusal is visible wherever
// the command came from (drop, menu, favorite, EXIF window); idle mode
// cancels first. Zoom and pan must not call this. Window close and
// shutdown remain available while busy.
func (v *viewer) yieldCopySelection() bool {
	if v.regionCopy != nil && v.regionCopy.State().Busy {
		v.ShowToast(lang.L("finishing the copy - try again in a moment"))
		return false
	}
	v.cancelRegionCopy()
	return true
}

// copySelectionKeepsKey is the viewer-side keep list: keys whose dispatcher
// case touches nothing but v.zoom stay available without cancelling.
// Feature.HandleKey owns Escape, copy, and navigation. Key0 is deliberately
// absent: its case also calls resetRotation, and an orientation change must
// yield the mode exactly as R does — check any key added here against its
// full handleKeyEvent case, not its name.
func copySelectionKeepsKey(key fyne.KeyName) bool {
	switch key {
	case fyne.Key1, fyne.KeyPlus, fyne.KeyEqual, fyne.KeyMinus:
		return true
	}
	return false
}

// regionCopyView is the sole adapter from the zoom presentation to the
// selection feature's oriented image coordinate system. Source.Bounds is
// the captured image-space.
func (v *viewer) regionCopyView(geometry zoom.Geometry, source copyselection.Source) copyselection.View {
	bounds := source.Bounds()
	if bounds.Empty() || v.img.Image == nil {
		return copyselection.View{}
	}
	return copyselection.View{
		ImageBounds: bounds,
		Position:    geometry.Position,
		Size:        geometry.Size,
	}
}

// copyRegionSelection encodes the captured source off the UI thread and
// dispatches PNG bytes to the clipboard. The shared clipboard completion
// signal finishes only after the final UI update, so tests and shutdown can
// wait without sleeping.
func (v *viewer) copyRegionSelection(bounds image.Rectangle) {
	token := v.regionCopyLifecycle.begin()
	done := v.clipboard.Begin()

	go func() {
		defer done()
		defer token.cancelContext()

		if !token.current() {
			return
		}
		data, err := v.regionCopy.Encode(bounds)
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
