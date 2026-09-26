package ui

import (
	"errors"
	"fmt"
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/clipboard"
	"github.com/frathe/picfetch/internal/ui/copyselection"
	"github.com/frathe/picfetch/internal/ui/zoom"
)

// regionCopyAvailable is the one viewer-side availability rule shared by
// direct activation, the menu snapshot, and the shortcut action. A decoded
// image must be settled in the normal viewer with no modal surface.
func (v *viewer) regionCopyAvailable() bool {
	if v.explorerMapActive() || v.locationMapVisible() {
		return false
	}
	if v.display.Count() == 0 || v.img.Image == nil || v.display.Snapshot().Loading {
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

	source, release, ok := v.captureRegionCopySource()
	if !ok {
		return
	}
	v.regionCopy.Start(v.regionCopyView(v.zoom.Geometry(), source), source)
	if !v.regionCopy.State().Active {
		release()
		return
	}
	v.regionCopyRelease = release
	v.regionCopyInfoVisible = v.info.Object().Visible()
	v.info.Object().Hide()
	v.ForceRepaint()
}

func (v *viewer) captureRegionCopySource() (copyselection.Source, func(), bool) {
	capture, release, ok := v.display.CaptureStable()
	if !ok {
		return copyselection.Source{}, nil, false
	}
	if capture.Vector != nil {
		return copyselection.VectorSource(capture.Vector, capture.LogicalSize, capture.Rotation, capture.Rasterize), release, true
	}
	return copyselection.RasterSource(capture.Pixels), release, true
}

// finishRegionCopy restores viewer-owned state after cancellation or a
// successful copy. The feature invokes it only after it has hidden its own
// overlay and cleared its transient state.
func (v *viewer) finishRegionCopy() {
	v.regionCopyLifecycle.invalidate()
	if v.regionCopyRelease != nil {
		v.regionCopyRelease()
		v.regionCopyRelease = nil
	}

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

// copySelectionKeepsKey is the viewer-side keep list: modifier-only events
// touch no command, and keys whose dispatcher case touches nothing but v.zoom
// stay available without cancelling. Feature.HandleKey owns Escape, copy, and
// navigation. Key0 is deliberately absent: its case also calls resetRotation,
// and an orientation change must yield the mode exactly as R does — check any
// key added here against its full handleKeyEvent case, not its name.
func copySelectionKeepsKey(key fyne.KeyName) bool {
	switch key {
	case fyne.Key1, fyne.KeyPlus, fyne.KeyEqual, fyne.KeyMinus:
		return true
	case desktop.KeyShiftLeft, desktop.KeyShiftRight,
		desktop.KeyControlLeft, desktop.KeyControlRight,
		desktop.KeyAltLeft, desktop.KeyAltRight,
		desktop.KeySuperLeft, desktop.KeySuperRight:
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
	_, done, ok := v.beginClipboardCopy(false)
	if !ok {
		v.regionCopy.Complete(errors.New("clipboard copy already pending"))
		return
	}
	token := v.regionCopyLifecycle.begin()

	v.clipboardWork.workers.Go(func() {
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
	})
}

func (v *viewer) reportRegionCopyError(err error) {
	detail := chooserErrorDetail(err)
	fyne.LogError("copy selected image failed", errors.New(detail))
	v.ShowToast(fmt.Sprintf(lang.L("could not copy the image: %v"), detail))
}
