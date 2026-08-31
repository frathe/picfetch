package ui

import (
	"image"

	"fyne.io/fyne/v2"
)

// rotateBy is the R key (clockwise, steps=1) / Shift+R (counter-clockwise,
// steps=-1): rotates the displayed image by one 90-degree step, view-only -
// see imaging.RotateSteps's doc for why this is safe to compose with the
// EXIF orientation already baked into the decoded pixels, and why repeated
// presses never degrade the image. It's a no-op before any image has
// loaded. Like a fresh navigation (finishLoad), a rotation resets zoom back
// to fit and, since a 90/270-degree turn swaps which axis is which, resizes
// the window to match - a manual zoom level or window size chosen for the
// old orientation rarely still makes sense once the axes have swapped.
func (v *viewer) rotateBy(steps int) {
	if !v.cancelRegionCopyBeforeAction() {
		return
	}
	if v.display.Count() == 0 {
		return
	}

	v.display.RotateBy(steps)
	v.redrawRotatedFrame()

	// syncMenus before applyRotationLayout, not after: the layout call can
	// itself spawn a vector re-render (SetLogicalSize/ResetToFit changing
	// the effective scale fires zoom's onScaleChanged), and that
	// goroutine's eventual write to v.img.Image is ordered against this
	// method's own continuation only by fyne.Do - a real, single-goroutine
	// guarantee in production, but the fake test driver runs a fyne.Do
	// callback inline on whichever goroutine calls it rather than
	// marshaling it (see ARCHITECTURE.md's concurrency invariant), so
	// canExport's read of v.img.Image inside syncMenus could otherwise
	// race that write under -race. syncMenus needs nothing
	// applyRotationLayout computes - only the display rotation, already
	// set above - so ordering it first removes the race instead of just
	// hiding it.
	v.syncMenus()
	v.applyRotationLayout()
}

// resetRotation is the other half of the 0 key (see zoom.FitToWindow):
// clears any view-only rotation back to the image's native EXIF
// orientation, the same way 0 resets zoom back to fit.
func (v *viewer) resetRotation() {
	if !v.display.ResetRotation() {
		return
	}

	v.redrawRotatedFrame()

	// Before applyRotationLayout, not after - see rotateBy's identical
	// ordering for why.
	v.syncMenus()
	v.applyRotationLayout()
}

// redrawRotatedFrame recomputes v.img.Image from the current unrotated
// frame and rotation (display.Rotated) and refreshes the canvas. Shared
// by rotateBy/resetRotation (a key press) and finishLoad/animate (loading
// a fresh image or advancing a GIF), so a rotation applied mid-animation
// keeps being applied to every later frame too, not just the one that
// happened to be on screen when R was pressed.
func (v *viewer) redrawRotatedFrame() {
	v.img.Image = v.display.Rotated()
	v.img.Refresh()
	v.animFrame.Add(1)
}

// applyRotationLayout re-fits and, outside picture-frame mode (where the
// window is already full-screen with nothing to resize - see finishLoad's
// matching comment), resizes the window to the image's displayed
// dimensions - the rotation-aware logical size for a vector, whose frame
// on screen may be denser than the size it is laid out at, and exactly the
// frame's bounds for every raster format (see displayedDimensions).
// Mirrors finishLoad's own ordering: re-fit first, for immediate visual
// feedback against whatever viewport size the zoom view currently has
// cached, then the window resize, whose own layout pass will re-lay it out
// against the authoritative new size.
func (v *viewer) applyRotationLayout() {
	// A 90/270-degree turn swaps which axis is which, and the zoom math
	// measures a vector against its logical size rather than its raster -
	// so that size has to turn with it, or fit scale is computed against
	// the wrong axis.
	if v.vector.svg != nil {
		logical := v.vector.logical
		if v.display.Rotation()%2 != 0 {
			logical = fyne.NewSize(logical.Height, logical.Width)
		}

		v.zoom.SetLogicalSize(logical)
	}

	v.zoom.ResetToFit()

	if !v.slides.Active() {
		v.undoGridMaximize()

		// Deliberately displayedDimensions rather than v.img.Image.Bounds():
		// for a vector those bounds are the *current* raster, which
		// rasterizeVector has been making denser the further the user zoomed
		// in - so rotating a zoomed-in SVG would size the window to the
		// zoom level rather than to the image. Same rule, and the same one
		// helper, as the info overlay reports; for every raster format
		// displayedDimensions still returns exactly these bounds.
		w, h := v.displayedDimensions()
		v.autoResizeToImage(image.Rect(0, 0, w, h))
	}

	// Deliberately after the layout work above, unlike the syncMenus
	// calls in rotateBy/resetRotation: this reads zoom.Percent(), which is
	// only correct once the re-fit has run. Under the fake test driver
	// that ordering has the same race shape those
	// call sites fixed - a re-render goroutine the layout spawned may
	// write img.Image while this reads it - and unlike them it cannot be
	// fixed by reordering. Production is safe (fyne.Do serializes onto the
	// UI goroutine, so the goroutine's write cannot overlap this read); a
	// test that opens the info overlay while rotating a vector must first
	// wait out v.vector.pending.
	v.updateInfoOverlay()
}
