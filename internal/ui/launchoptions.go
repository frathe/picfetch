// Applying the launch flags internal/launch parsed: the setters they drive,
// the pre-flag values shutdown puts back, and the one-shot picture-frame
// request. Parsing and validation happen before the app exists and stay in
// internal/launch; nothing here reads argv.

package ui

import (
	"time"

	"github.com/frathe/picfetch/internal/filesort"
	"github.com/frathe/picfetch/internal/launch"
	"github.com/frathe/picfetch/internal/preferences"
)

// launchOverride holds the value each launch flag replaced. A nil field was
// not overridden and saves whatever the viewer holds at shutdown, as usual.
//
// This is what makes a flag session-only. The flags drive the same setters
// the Settings window does, and currentPreferences saves the viewer's live
// state, so without this a scripted "picfetch --shuffle" would leave shuffle
// switched on for every later launch.
//
// The restore is per field rather than per change: an overridden setting
// that the user then changes in the Settings window still saves its pre-flag
// value for that run. Tracking the difference would mean a dirty bit on five
// settings to save one relaunch - see
// plans/2026-09-05-launch-flags.md's "The honest limit".
type launchOverride struct {
	sortMode *string
	merge    *bool
	shuffle  *bool
	interval *time.Duration
	maxScan  *int
}

// restore rewrites every overridden field of s back to its pre-flag value.
// Called by currentPreferences, at the end, so it is the last word on what
// shutdown writes.
func (o launchOverride) restore(s *preferences.State) {
	if o.sortMode != nil {
		s.SortMode = *o.sortMode
	}
	if o.merge != nil {
		s.MergeMode = *o.merge
	}
	if o.shuffle != nil {
		s.SlideShuffle = *o.shuffle
	}
	if o.interval != nil {
		s.SlideInterval = *o.interval
	}
	if o.maxScan != nil {
		s.MaxScanFiles = *o.maxScan
	}
}

// applyLaunchOptions applies one launch's flags to a viewer that has already
// been built from saved preferences, recording what each one replaced.
//
// Called from Run between construction and Show: every setter here is the
// same one the Settings window binds to, so the flags reach exactly the
// state a user could have set by hand. The one exception is --slideshow,
// which is armed rather than applied - see startPendingPictureFrame.
func (v *viewer) applyLaunchOptions(opts launch.Options) {
	if opts.Sort != nil {
		prev := v.state.SortMode().PrefValue()
		v.launchOverride.sortMode = &prev
		v.SetSortMode(filesort.FromPref(*opts.Sort))
	}
	if opts.Merge != nil {
		prev := v.MergeMode()
		v.launchOverride.merge = &prev
		v.SetMergeMode(*opts.Merge)
	}
	if opts.Shuffle != nil {
		prev := v.SlideShuffle()
		v.launchOverride.shuffle = &prev
		v.SetSlideShuffle(*opts.Shuffle)
	}
	if opts.Interval != nil {
		// v.slides.Interval(), not v.SlideInterval(): the latter substitutes
		// DefaultInterval for a controller that has never been given one,
		// and saving that substitution would turn "no interval chosen yet"
		// into a chosen interval on the way out.
		prev := v.slides.Interval()
		v.launchOverride.interval = &prev
		v.SetSlideInterval(*opts.Interval)
	}
	if opts.MaxFiles != nil {
		prev := v.MaxScan()
		v.launchOverride.maxScan = &prev
		v.SetMaxScan(*opts.MaxFiles)
	}

	v.pendingPictureFrame = opts.PictureFrame
}

// startPendingPictureFrame honors --slideshow once the launch scan has
// finished, and spends the request either way - a launch that loaded nothing
// must not leave picture-frame mode armed for whatever the user drops next.
//
// It cannot run when the options are applied: picture-frame mode has nothing
// to frame with zero files loaded, so slideshow.Toggle no-ops there (see
// internal/ui/slideshow). Both of applyScanResult's endings call this, which
// is why it checks Active first - togglePictureFrameMode is a toggle, and
// leaving picture-frame mode would be the opposite of what the flag asked
// for.
func (v *viewer) startPendingPictureFrame() {
	if !v.pendingPictureFrame {
		return
	}
	v.pendingPictureFrame = false

	if v.slides.Active() {
		return
	}
	v.togglePictureFrameMode()
}
