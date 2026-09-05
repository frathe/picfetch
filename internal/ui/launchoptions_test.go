// Launch flags as the app applies them: the setters they drive, the
// session-only save behaviour that keeps a scripted launch from rewriting
// saved defaults, and the one-shot picture-frame request. The parsing
// itself is tested in internal/launch.

package ui

import (
	"image/color"
	"testing"
	"time"

	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/filesort"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/launch"
	"github.com/frathe/picfetch/internal/preferences"
	"github.com/frathe/picfetch/internal/uitest"
)

// savedPreferences is the state every persistence case below starts from, so
// each one can tell a pre-flag value apart from whatever the flag set.
func savedPreferences() preferences.State {
	return preferences.State{
		SortMode:      preferences.SortBySize,
		MergeMode:     true,
		SlideInterval: 7 * time.Second,
		SlideShuffle:  true,
		MaxScanFiles:  5000,
	}
}

// everyOption overrides all five standing settings savedPreferences sets, to
// the other value in each case.
func everyOption() launch.Options {
	sort := preferences.SortByCaptureDate
	merge, shuffle := false, false
	interval := 20 * time.Second
	maxFiles := 99

	return launch.Options{
		Sort:     &sort,
		Merge:    &merge,
		Shuffle:  &shuffle,
		Interval: &interval,
		MaxFiles: &maxFiles,
	}
}

// viewerWithSavedPreferences builds a viewer through the production startup
// path with savedPreferences already on disk, the way preferences_wiring_test
// does - its own app, because Save/Load is what is under test here.
func viewerWithSavedPreferences(t *testing.T) *viewer {
	t.Helper()

	application := test.NewApp()
	preferences.Save(application, savedPreferences())

	v, win := buildStartupViewer(application)
	t.Cleanup(func() { win.Close() })
	t.Cleanup(func() { imaging.SetMaxEncodedBytes(0) }) // process-wide - see memlimits.go

	return v
}

func TestLaunchOptions_AppliesEverySetting(t *testing.T) {
	v := viewerWithSavedPreferences(t)

	v.applyLaunchOptions(everyOption())

	if got, want := v.state.SortMode(), filesort.ByCaptureDate; got != want {
		t.Errorf("SortMode() = %v, want %v", got, want)
	}
	if v.MergeMode() {
		t.Error("MergeMode() = true, want false from --merge=false")
	}
	if v.SlideShuffle() {
		t.Error("SlideShuffle() = true, want false from --shuffle=false")
	}
	if got, want := v.slides.Interval(), 20*time.Second; got != want {
		t.Errorf("slides.Interval() = %v, want %v", got, want)
	}
	if got, want := v.MaxScan(), 99; got != want {
		t.Errorf("MaxScan() = %d, want %d", got, want)
	}
}

// TestLaunchOptions_ZeroOptionsChangeNothing is the plain launch: no flags
// means every saved preference survives untouched.
func TestLaunchOptions_ZeroOptionsChangeNothing(t *testing.T) {
	v := viewerWithSavedPreferences(t)

	v.applyLaunchOptions(launch.Options{})

	if got, want := v.state.SortMode(), filesort.BySize; got != want {
		t.Errorf("SortMode() = %v, want %v from the saved preference", got, want)
	}
	if !v.MergeMode() {
		t.Error("MergeMode() = false, want the saved true")
	}
	if !v.SlideShuffle() {
		t.Error("SlideShuffle() = false, want the saved true")
	}
	if got, want := v.slides.Interval(), 7*time.Second; got != want {
		t.Errorf("slides.Interval() = %v, want the saved %v", got, want)
	}
	if got, want := v.MaxScan(), 5000; got != want {
		t.Errorf("MaxScan() = %d, want the saved %d", got, want)
	}
	if v.pendingPictureFrame {
		t.Error("pendingPictureFrame = true, want false without --slideshow")
	}
}

// TestLaunchOptions_SavedPreferencesKeepPreFlagValues is the decision this
// feature turns on: a flag applies to the run, not to the saved settings, so
// "picfetch --shuffle" cannot leave shuffle on forever.
func TestLaunchOptions_SavedPreferencesKeepPreFlagValues(t *testing.T) {
	v := viewerWithSavedPreferences(t)

	v.applyLaunchOptions(everyOption())
	saved := v.currentPreferences()

	if got, want := saved.SortMode, preferences.SortBySize; got != want {
		t.Errorf("saved SortMode = %q, want the pre-flag %q", got, want)
	}
	if !saved.MergeMode {
		t.Error("saved MergeMode = false, want the pre-flag true")
	}
	if !saved.SlideShuffle {
		t.Error("saved SlideShuffle = false, want the pre-flag true")
	}
	if got, want := saved.SlideInterval, 7*time.Second; got != want {
		t.Errorf("saved SlideInterval = %v, want the pre-flag %v", got, want)
	}
	if got, want := saved.MaxScanFiles, 5000; got != want {
		t.Errorf("saved MaxScanFiles = %d, want the pre-flag %d", got, want)
	}
}

// TestLaunchOptions_UntouchedSettingsStillSaveLiveValues is the other half:
// only the fields a flag actually overrode are held back, so a setting the
// user changes during a flagged run still persists.
func TestLaunchOptions_UntouchedSettingsStillSaveLiveValues(t *testing.T) {
	v := viewerWithSavedPreferences(t)

	maxFiles := 99
	v.applyLaunchOptions(launch.Options{MaxFiles: &maxFiles})
	v.SetSortMode(filesort.ByModTime)

	saved := v.currentPreferences()
	if got, want := saved.SortMode, preferences.SortByModTime; got != want {
		t.Errorf("saved SortMode = %q, want the live %q - no flag touched it", got, want)
	}
	if got, want := saved.MaxScanFiles, 5000; got != want {
		t.Errorf("saved MaxScanFiles = %d, want the pre-flag %d", got, want)
	}
}

// TestLaunchOptions_PreFlagValueWinsOverALiveChange pins the documented
// limit of the session-only rule (see plans/2026-09-05-launch-flags.md): the
// restore is per field, not per change, so changing an overridden setting in
// the Settings window during a flagged run does not persist either. Pinned
// rather than left to chance so the behaviour is a decision, not a surprise.
func TestLaunchOptions_PreFlagValueWinsOverALiveChange(t *testing.T) {
	v := viewerWithSavedPreferences(t)

	sort := preferences.SortByCaptureDate
	v.applyLaunchOptions(launch.Options{Sort: &sort})
	v.SetSortMode(filesort.ByDropOrder)

	if got, want := v.currentPreferences().SortMode, preferences.SortBySize; got != want {
		t.Errorf("saved SortMode = %q, want the pre-flag %q", got, want)
	}
}

// TestLaunchOptions_PictureFrameStartsAfterTheLaunchFilesLoad is why
// --slideshow is not applied at startup: slideshow.Toggle no-ops at zero
// files, and at the point options are applied nothing has been scanned yet.
func TestLaunchOptions_PictureFrameStartsAfterTheLaunchFilesLoad(t *testing.T) {
	v := newTestViewer(t)

	v.applyLaunchOptions(launch.Options{PictureFrame: true})
	if v.slides.Active() {
		t.Fatal("picture-frame mode is on before any file loaded")
	}

	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White), uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White))
	t.Cleanup(func() { settleSlideshow(t, v) })

	if !v.slides.Active() {
		t.Error("picture-frame mode should be on once the launch files have loaded")
	}
	if !v.win.FullScreen() {
		t.Error("window should be full-screen in picture-frame mode")
	}
}

// TestLaunchOptions_PictureFrameStartsOnlyOnce keeps the request a launch
// action rather than a standing mode: after the user leaves it, the next
// drop is an ordinary drop.
func TestLaunchOptions_PictureFrameStartsOnlyOnce(t *testing.T) {
	v := newTestViewer(t)

	v.applyLaunchOptions(launch.Options{PictureFrame: true})
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	settleSlideshow(t, v)

	if v.slides.Active() {
		t.Fatal("settleSlideshow should have left picture-frame mode")
	}

	dropAndWait(t, v, uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White))
	t.Cleanup(func() { settleSlideshow(t, v) })

	if v.slides.Active() {
		t.Error("a later drop re-entered picture-frame mode; the request should be spent")
	}
}

// TestLaunchOptions_PictureFrameSpentWhenTheLaunchLoadsNothing covers
// "--slideshow ~/notes.txt": the launch fails to load an image, and the
// request must not lie in wait for whatever the user drops next.
func TestLaunchOptions_PictureFrameSpentWhenTheLaunchLoadsNothing(t *testing.T) {
	v := newTestViewer(t)

	v.applyLaunchOptions(launch.Options{PictureFrame: true})
	dropAndWaitScan(t, v, storage.NewFileURI(uitest.WriteTempFile(t, "notes.txt", []byte("not an image"))))

	if v.slides.Active() {
		t.Fatal("picture-frame mode is on with no image loaded")
	}

	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	t.Cleanup(func() { settleSlideshow(t, v) })

	if v.slides.Active() {
		t.Error("a later drop entered picture-frame mode; the failed launch should have spent the request")
	}
}
