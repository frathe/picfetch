// Root forwards zoom density; display owns SVG policy and workers.

package ui

import "fyne.io/fyne/v2"

func (v *viewer) requestVectorRender(scale float32) {
	var toPixels func(fyne.Position) (int, int)
	if v.win != nil && v.win.Canvas() != nil {
		toPixels = v.win.Canvas().PixelCoordinateForPosition
	}
	v.display.RequestVectorRender(scale, toPixels)
}
