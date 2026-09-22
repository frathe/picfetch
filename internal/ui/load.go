// Single-image navigation policy and display composition.

package ui

import (
	"errors"
	"fmt"
	"image"
	"math/rand/v2"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/heic"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/ui/display"
)

// ShowImage is the existing admission chokepoint for navigation and image actions.
func (v *viewer) ShowImage(i int) {
	if v.comparisonActive() || len(v.state.files) == 0 || !v.yieldCopySelection() {
		return
	}
	v.explorerImageOpened()
	v.cancelImageClipboard()
	v.cancelSave()
	v.cancelExport()
	v.exif.Invalidate()
	n := len(v.state.files)
	v.state.index = ((i % n) + n) % n
	if v.slides.Active() && v.img.Image != nil {
		v.startFade(0, 1)
	}
	v.display.Load(display.Request{Source: v.state.files[v.state.index], Transition: v.slides.Active()})
}

func (v *viewer) invalidateLoad() uint64 {
	v.cancelImageClipboard()
	v.cancelSave()
	v.cancelExport()
	v.exif.Invalidate()
	v.display.CancelRequest()
	return v.display.RequestRevision()
}

func (v *viewer) imageRequested(_ display.Identity) {
	v.loadingBar.Show()
	v.syncMenus()
	if v.img.Image == nil {
		v.hint.SetText(lang.L("Loading..."))
		v.dropzone.Show()
	}
	v.ForceRepaint()
}

func (v *viewer) imageProbed(bounds image.Rectangle) {
	if !v.slides.Active() && !v.grid.Visible() && !v.explorer.HasCohort() && !v.explorer.Surface().Visible() {
		v.undoGridMaximize()
		v.autoResizeToImage(bounds)
	}
}

// imagePresented runs synchronously after coherent display publication and
// before the owner admits animation or neighbor work and completes the load.
func (v *viewer) imagePresented(snapshot display.Snapshot) []fyne.URI {
	v.syncPresentationLogicalSize()
	v.dropzone.Hide()
	v.welcomeArt.Hide()
	v.emptyStateArt.Hide()
	v.info.SetFile(snapshot.FileSize, snapshot.HasEXIF, snapshot.Preview)
	v.syncInfoOverlayVisibility()
	v.zoom.ResetToFit()
	v.imageProbed(image.Rect(0, 0, int(snapshot.Size.Width), int(snapshot.Size.Height)))
	v.applyLoadedTitle(snapshot)
	v.loadingBar.Hide()
	v.syncMenus()
	v.ForceRepaint()
	v.exif.Refresh()
	if v.explorer.HasCohort() {
		v.recordExplorerView("image-loaded")
	}
	return v.preloadCandidates()
}

func (v *viewer) syncPresentationLogicalSize() {
	snapshot := v.display.Snapshot()
	logical := fyne.Size{}
	if snapshot.Vector {
		logical = snapshot.Size
	}
	v.zoom.SetLogicalSize(logical)
}

func (v *viewer) applyLoadedTitle(snapshot display.Snapshot) {
	title := fmt.Sprintf("%s — %d x %d", snapshot.Displayed.Source.Name(), int(snapshot.Size.Width), int(snapshot.Size.Height))
	if snapshot.Preview {
		title += " " + lang.L("(preview)")
	}
	if snapshot.Animated {
		title += " (animated)"
	}
	v.slides.SetAnimDuration(snapshot.Duration)
	if n := len(v.state.files); n > 1 {
		title = fmt.Sprintf("%s  (%d/%d)", title, v.state.index+1, n)
	}
	v.setTitle(title)
}

func (v *viewer) imageLoadFailed(source fyne.URI, err error) fyne.URI {
	retained := append([]fyne.URI(nil), v.state.unavailableHEIC...)
	if errors.Is(err, heic.ErrUnavailable) {
		retained = append(retained, source)
	}
	msg := fmt.Sprintf(lang.L("could not read %q: %v"), source.Name(), err)
	var dimensions *imaging.InvalidDimensionsError
	var tooLarge *imaging.InputTooLargeError
	switch {
	case errors.As(err, &dimensions):
		msg = fmt.Sprintf(lang.L("invalid image dimensions for %q"), source.Name())
	case errors.As(err, &tooLarge):
		msg = fmt.Sprintf(lang.L("%q is too large to open"), source.Name())
	}
	i := v.state.index
	restoredIndex := v.reconcileSources(sourceChange{kind: sourceLoadFailed, removed: []int{i}})
	if len(v.state.files) == 0 {
		v.ShowEmptyStateError(msg)
		v.state.unavailableHEIC = retained
		if errors.Is(err, heic.ErrUnavailable) {
			v.explainUnavailableHEIC([]fyne.URI{source}, true)
		}
		return nil
	}
	v.state.unavailableHEIC = retained
	v.ShowToast(msg)
	if restoredIndex >= 0 {
		i = restoredIndex
	} else if cohort := v.cohortIndexes(); len(cohort) > 0 {
		next := cohort[0]
		for _, index := range cohort {
			if index >= i {
				next = index
				break
			}
		}
		i = next
	}
	n := len(v.state.files)
	v.state.index = ((i % n) + n) % n
	return v.state.files[v.state.index]
}

func (v *viewer) imageAnimationTruncated(source fyne.URI) {
	v.ShowToast(fmt.Sprintf(lang.L("animation in %q is too large to play"), source.Name()))
}

func (v *viewer) preloadCandidates() []fyne.URI {
	n := len(v.state.files)
	if n < 2 {
		return nil
	}
	next, prev := (v.state.index+1)%n, (v.state.index-1+n)%n
	if order := v.captureSearchOrder(); order.active {
		next = neighborInOrder(order.indexes, v.state.index, 1)
		prev = neighborInOrder(order.indexes, v.state.index, -1)
	}
	var candidates []fyne.URI
	if next != v.state.index {
		candidates = append(candidates, v.state.files[next])
	}
	if prev != next && prev != v.state.index {
		candidates = append(candidates, v.state.files[prev])
	}
	return candidates
}

// defaultMaxWindowWidth and defaultMaxWindowHeight cap how large the window
// is ever allowed to auto-grow to fit a loaded image, until the settings
// window (internal/ui/settingswin) changes them - see the viewer's
// settings.maxWinW/maxWinH fields (memlimits.go) and MaxWindowWidth and
// MaxWindowHeight below.
const (
	defaultMaxWindowWidth  = 1500.0
	defaultMaxWindowHeight = 950.0
)

// MaxWindowWidth and MaxWindowHeight report the current window-size cap -
// the settings window's getters.
func (v *viewer) MaxWindowWidth() float32  { return v.settings.maxWinW }
func (v *viewer) MaxWindowHeight() float32 { return v.settings.maxWinH }

// StaticWindowSize reports whether auto-resize-to-image is off - the settings
// window's "Keep a fixed window size" checkbox.
func (v *viewer) StaticWindowSize() bool { return v.settings.staticWindowSize }

// SetStaticWindowSize turns auto-resize-to-image on or off. When on, load /
// zoom / rotate leave the current window size alone; manual drags still
// update windowSize via windowSizeTracker and persist at shutdown.
func (v *viewer) SetStaticWindowSize(on bool) {
	v.settings.staticWindowSize = on
}

// SetMaxWindowWidth and SetMaxWindowHeight set the window-size cap
// directly - the settings window's binding. Floored at the drop-zone size
// (startW/startH): resizeToImage already never shrinks the window below
// that regardless of the cap, so a lower value would silently have no
// effect - flooring here instead of just letting that happen keeps what the
// settings window shows in sync with what the window actually does.
func (v *viewer) SetMaxWindowWidth(w float32) {
	if w < startW {
		w = startW
	}
	v.settings.maxWinW = w
}

func (v *viewer) SetMaxWindowHeight(h float32) {
	if h < startH {
		h = startH
	}
	v.settings.maxWinH = h
}

// syncWindowToZoom resizes the main window to track the image at the
// current zoom level, clamped between startW/startH and maxWinW/maxWinH.
// A no-op while the slideshow or grid overlay is active, while a fixed
// window size is set, or before any image has been loaded. Called from
// zoom's onChanged (features.go) so the window grows and shrinks with
// every user-driven zoom step. undoGridMaximize is called before each
// resize for the same reason finishLoad and applyRotationLayout call it:
// a plain Resize on an OS-maximized window is silently ignored on some
// platforms.
func (v *viewer) syncWindowToZoom() {
	if v.settings.staticWindowSize {
		return
	}
	if v.slides != nil && v.slides.Active() {
		return
	}
	if v.grid != nil && v.grid.Visible() {
		return
	}
	// Presentation availability and logical dimensions come from its owner.
	if v.display.Count() == 0 {
		return
	}
	w, h := v.displayedDimensions()
	if v.zoom.Fitting() {
		v.undoGridMaximize()
		v.autoResizeToImage(image.Rect(0, 0, w, h))
		return
	}
	s := v.zoom.Scale()
	v.undoGridMaximize()
	v.autoResizeToImage(image.Rect(0, 0, int(float32(w)*s+0.5), int(float32(h)*s+0.5)))
}

// autoResizeToImage resizes the main window to fit b unless the user asked
// for a fixed window size. Shared by load, zoom, and rotate so the static
// toggle has one gate.
func (v *viewer) autoResizeToImage(b image.Rectangle) {
	if v.settings.staticWindowSize {
		return
	}
	resizeToImage(v.win, b, v.settings.maxWinW, v.settings.maxWinH)
}

// resizeToImage resizes w to fit b, scaled down (preserving aspect ratio)
// so neither dimension exceeds maxW/maxH, and never below startW/startH.
func resizeToImage(w fyne.Window, b image.Rectangle, maxW, maxH float32) {
	width := float32(b.Dx())
	height := float32(b.Dy())

	if f := min(maxW/width, maxH/height, float32(1.0)); f < 1 {
		width *= f
		height *= f
	}

	// Never shrink below the drop-zone size — a tiny thumbnail would
	// otherwise produce a window too small to grab or read the title of.
	// ImageFillContain letterboxes the image within the larger frame.
	w.Resize(fyne.NewSize(max(width, startW), max(height, startH)))
}

// slideshowFadeDuration is how long each half of a picture-frame-mode
// transition takes: the outgoing image fades to invisible, then the
// incoming one fades in from invisible - a full transition takes about
// twice this long, overlapping however much of the load it happens to
// take. Ordinary browsing (picture-frame mode off) never calls startFade
// at all, so it stays an instant swap exactly as before.
const slideshowFadeDuration = 400 * time.Millisecond

// startFade starts a fade ticking v.img's Translucency from start to end
// over slideshowFadeDuration, refreshing the canvas on every tick. The
// fade's lifecycle - stopping whichever one is already running first, and
// why that matters - is display.StartFade's; the translucency math stays
// here because v.img is the viewer's.
func (v *viewer) startFade(start, end float64) {
	v.display.FadeTo(float32(start), float32(end), slideshowFadeDuration)
}

// resetFade cancels any fade transition in progress and puts v.img back to
// fully opaque. Called from every place picture-frame mode ends, so
// leaving it mid-transition never strands the image invisible or
// half-faded once it's back in the normal, instant-swap view.
func (v *viewer) resetFade() {
	v.display.ResetFade()
}

// randomOtherIndex picks a uniformly random index in [0,n) other than
// current - Advance's shuffle-mode step. Picking from the n-1 indices that
// aren't current and shifting the ones at or past it up by one, rather
// than rejection-sampling rand.IntN(n) until it misses, keeps this O(1)
// and never repeats the image already on screen, which a plain
// rand.IntN(n) would occasionally do. n<=1 has no "other" index to pick,
// so it returns current unchanged - Advance never calls this in that case
// (the slideshow doesn't run at all with fewer than two files), but a
// direct caller (this function's own tests included) gets a safe answer
// either way instead of a panic or an out-of-range index.
func randomOtherIndex(n, current int) int {
	if n <= 1 {
		return current
	}

	next := rand.IntN(n - 1)
	if next >= current {
		next++
	}

	return next
}
