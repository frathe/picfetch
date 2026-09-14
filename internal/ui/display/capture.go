package display

import (
	"image"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/imaging"
)

// Capture retains read-only pixels and the presentation they came from.
type Capture struct {
	Pixels          image.Image
	Identity        Identity
	Rotation        int
	Vector          *imaging.Vector
	LogicalSize     image.Point
	Rasterize       func(*imaging.Vector, int, int) (image.Image, error)
	owner           *Feature
	requestRevision uint64
}

func (f *Feature) Capture() (Capture, bool) {
	if f.stopped || f.surface.Image == nil {
		return Capture{}, false
	}
	return Capture{Pixels: f.surface.Image, Identity: f.snapshot.Displayed,
		Rotation: f.Rotation(), owner: f, requestRevision: f.revision,
		Vector: f.vector.svg, LogicalSize: image.Pt(int(f.logical.Width+0.5), int(f.logical.Height+0.5)), Rasterize: f.vector.options.Rasterize}, true
}

// ReconcileSaved adopts a committed capture as the saved baseline, preserving
// any subsequent turns and the published image itself.
func (f *Feature) ReconcileSaved(capture Capture) bool {
	if capture.owner != f || f.stopped || f.snapshot.Loading || capture.requestRevision != f.revision ||
		capture.Identity.Revision != f.snapshot.Displayed.Revision || capture.Pixels == nil || f.Count() != 1 {
		return false
	}
	f.frames = []image.Image{capture.Pixels}
	f.rotation = normalizedRotation(f.rotation - capture.Rotation)
	b := capture.Pixels.Bounds()
	f.logical = fyne.NewSize(float32(b.Dx()), float32(b.Dy()))
	return true
}

// CaptureStable retains the source and excludes GIF advancement until release.
func (f *Feature) CaptureStable() (Capture, func(), bool) {
	if !f.snapshot.Animated {
		capture, ok := f.Capture()
		return capture, func() {}, ok
	}
	var capture Capture
	var ok bool
	if !f.pause.pause(func() { capture, ok = f.Capture() }) {
		return Capture{}, nil, false
	}
	release := f.pause.release()
	if !ok {
		release()
		return Capture{}, nil, false
	}
	return capture, release, true
}

// PauseObserved reports when playback has reached the stable capture's gate.
func (f *Feature) PauseObserved() <-chan struct{} {
	f.pause.mu.Lock()
	defer f.pause.mu.Unlock()
	return f.pause.observed
}
