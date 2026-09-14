package ui

import "image"

// rotateBy is the R key (clockwise, steps=1) / Shift+R (counter-clockwise,
// steps=-1): rotates the displayed image by one 90-degree step, view-only -
// see imaging.RotateSteps's doc for why this is safe to compose with the
// EXIF orientation already baked into the decoded pixels, and why repeated
// presses never degrade the image. It's a no-op before any image has
// loaded. Like a fresh navigation (imagePresented), a rotation resets zoom back
// to fit and, since a 90/270-degree turn swaps which axis is which, resizes
// the window to match - a manual zoom level or window size chosen for the
// old orientation rarely still makes sense once the axes have swapped.
func (v *viewer) rotateBy(steps int) {
	if v.display.Count() == 0 {
		return
	}

	v.display.RotateBy(steps)

	// Rotation is already committed; synchronize action availability before layout.
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

	v.syncMenus()
	v.applyRotationLayout()
}

// applyRotationLayout re-fits and, outside picture-frame mode (where the
// window is already full-screen with nothing to resize - see imagePresented's
// matching comment), resizes the window to the image's displayed
// dimensions - the rotation-aware logical size for a vector, whose frame
// on screen may be denser than the size it is laid out at, and exactly the
// frame's bounds for every raster format (see displayedDimensions).
// Mirrors imagePresented's own ordering: re-fit first, for immediate visual
// feedback against whatever viewport size the zoom view currently has
// cached, then the window resize, whose own layout pass will re-lay it out
// against the authoritative new size.
func (v *viewer) applyRotationLayout() {
	v.syncPresentationLogicalSize()

	v.zoom.ResetToFit()

	if !v.slides.Active() {
		v.undoGridMaximize()

		// Deliberately displayedDimensions rather than v.img.Image.Bounds():
		// for a vector those bounds are the *current* raster, which
		// display sharpening has been making denser the further the user zoomed
		// in - so rotating a zoomed-in SVG would size the window to the
		// zoom level rather than to the image. Same rule, and the same one
		// helper, as the info overlay reports; for every raster format
		// displayedDimensions still returns exactly these bounds.
		w, h := v.displayedDimensions()
		v.autoResizeToImage(image.Rect(0, 0, w, h))
	}

	// The overlay reads zoom.Percent(), which is current only after the re-fit.
	v.updateInfoOverlay()
}

// displayedDimensions reports oriented logical dimensions, independent of raster density.
func (v *viewer) displayedDimensions() (w, h int) {
	size := v.display.Snapshot().Size
	return int(size.Width + 0.5), int(size.Height + 0.5)
}
