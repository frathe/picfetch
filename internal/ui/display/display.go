// Package display owns what internal/ui's viewer currently has on the
// canvas: the current image's decoded frames, which of them is up, the
// view-only rotation, and the picture-frame crossfade. It composes
// imaging.RotateSteps itself (Rotated), so "the current frame at the
// current rotation" is decided in one place; actually putting that frame
// on screen - writing the canvas image, refreshing it, counting the write
// for tests - stays internal/ui's (redrawRotatedFrame's), which owns the
// canvas image.
package display

import (
	"image"
	"time"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/imaging"
)

// State is the viewer's display state - a value field on internal/ui's
// viewer, never copied.
//
// frames is the current image's decoded, EXIF-corrected frames
// (imaging.LoadedImage.Frames - unrotated), and idx which one of them is
// currently on screen: index 0 for a static image, or whichever one the
// viewer's animate goroutine most recently cycled to for an animated GIF.
// rotateBy (internal/ui/rotate.go) needs both to redraw at a new rotation
// without waiting for animate's next tick, since it can't otherwise tell
// which frame of an in-progress animation is currently up.
//
// rotation is the view-only clockwise quarter-turn count (0-3) composed
// with the EXIF orientation already baked into those frames at render
// time - see imaging.RotateSteps - and is never written back to disk.
// Reset to 0 by every fresh navigation (internal/ui's finishLoad) and the
// 0 key, mirroring the way the zoom view resets to fit.
//
// fade is the crossfade in progress, if any, between the last image on
// screen and the one replacing it - see internal/ui/load.go's
// startFade/resetFade for who starts and resets one.
type State struct {
	frames   []image.Image
	idx      int
	rotation int
	fade     *fyne.Animation
}

// SetFrames installs a fresh image's decoded frames. The index and
// rotation are deliberately left alone: installLoadedFrames
// (internal/ui/load.go) resets both explicitly, once per navigation,
// after it has re-sliced a vector's frames.
func (s *State) SetFrames(frames []image.Image) { s.frames = frames }

// Count is how many frames the current image decoded to: 0 before any
// image has loaded (what rotate/zoom enablement and rotateBy's no-op key
// off), 1 for a static image, more for an animated GIF.
func (s *State) Count() int { return len(s.frames) }

// Index is which frame is currently on screen.
func (s *State) Index() int { return s.idx }

// SetIndex records which frame is currently on screen - animate's
// per-tick write, and finishLoad's reset to 0.
func (s *State) SetIndex(i int) { s.idx = i }

// Rotation is the view-only clockwise quarter-turn count, always in 0-3.
func (s *State) Rotation() int { return s.rotation }

// RotateBy adds steps clockwise quarter-turns, normalizing back into 0-3
// so turns in either direction wrap instead of accumulating (-1 and 3
// both mean one turn counter-clockwise).
func (s *State) RotateBy(steps int) { s.rotation = ((s.rotation+steps)%4 + 4) % 4 }

// ResetRotation clears the rotation back to 0 and reports whether that
// changed anything - false lets the 0 key (internal/ui's resetRotation)
// skip the redraw and re-layout entirely.
func (s *State) ResetRotation() (changed bool) {
	if s.rotation == 0 {
		return false
	}

	s.rotation = 0
	return true
}

// Current is the unrotated frame currently on screen. Like Rotated and
// ReplaceCurrent it indexes into the frames, so it requires Count() > 0 -
// every caller already guards on that, the same way they guarded the
// direct indexing this replaces.
func (s *State) Current() image.Image { return s.frames[s.idx] }

// Rotated is the current frame at the current rotation - what
// redrawRotatedFrame (internal/ui/rotate.go) puts on the canvas. See
// imaging.RotateSteps's doc for why composing on top of the EXIF
// orientation already baked into the decoded pixels is safe and why
// repeated turns never degrade the image.
func (s *State) Rotated() image.Image { return imaging.RotateSteps(s.frames[s.idx], s.rotation) }

// ReplaceCurrent swaps the current frame's unrotated pixels in place: a
// vector re-render landing (internal/ui/vector.go) or a saved rotation
// being folded back into the frame it came from (internal/ui/save.go).
func (s *State) ReplaceCurrent(img image.Image) { s.frames[s.idx] = img }

// Clear drops everything back to the zero state: no frames, index and
// rotation 0, no fade - the viewer's empty drop-zone state. Restoring the
// canvas image itself (translucency, hiding) stays the caller's job.
func (s *State) Clear() {
	s.ResetFade()
	s.frames = nil
	s.idx = 0
	s.rotation = 0
}

// Fade is the crossfade in progress, if any - nil once ResetFade or Clear
// has run. Exposed so callers can assert on the fade's lifecycle
// directly, the same "reach through an exported accessor" shape
// internal/ui/menus uses for its menu items.
func (s *State) Fade() *fyne.Animation { return s.fade }

// StartFade stops whatever fade is already running - a no-op if none is -
// and starts a fresh one calling tick over d. Stopping the previous
// animation first matters when a fade-in begins before the fade-out
// before it has finished (a fast, likely cache-hit load - see
// internal/ui/load.go's attemptLoad): without it, the outgoing
// animation's next tick could overwrite a value the new one already set.
// Under the fyne test driver, Start ticks straight to the end state
// synchronously (see fyne/test's driver.StartAnimation), so a test never
// observes an in-between value.
func (s *State) StartFade(d time.Duration, tick func(t float32)) {
	if s.fade != nil {
		s.fade.Stop()
	}

	s.fade = fyne.NewAnimation(d, tick)
	s.fade.Start()
}

// ResetFade stops and forgets the fade in progress - safe to call when
// none is running. Putting the canvas image back to fully opaque is the
// caller's job (internal/ui's resetFade), which owns that image.
func (s *State) ResetFade() {
	if s.fade != nil {
		s.fade.Stop()
		s.fade = nil
	}
}
