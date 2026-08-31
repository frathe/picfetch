package ui

import (
	"image/color"
	"testing"

	"github.com/frathe/picfetch/internal/appearance"
	"github.com/frathe/picfetch/internal/filesort"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestSettingsState_SubstitutesSlideIntervalDefault(t *testing.T) {
	v := newTestViewer(t)
	if v.slides.Interval() != 0 {
		t.Fatalf("raw interval = %v, want 0 (unset)", v.slides.Interval())
	}
	if got, want := v.settingsState().SlideInterval, v.SlideInterval(); got != want {
		t.Errorf("settingsState SlideInterval = %v, want the getter default %v", got, want)
	}
	if got, want := v.settingsState().DuplicateDistance, imaging.DuplicateMaxDistance; got != want {
		t.Errorf("settingsState DuplicateDistance = %d, want the unset default %d", got, want)
	}
}

func TestApplySettings_AppliesOnlyChangedFields(t *testing.T) {
	v := newTestViewer(t)
	t.Cleanup(func() { imaging.SetMaxEncodedBytes(0) })

	a := uitest.TempJPEGURI(t, "a.jpg", 8, 8, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 8, 8, color.White)
	dropAndWait(t, v, a, b)

	startSort := v.SortMode()
	startRev := v.sortOp.lifecycle.currentRevision()
	startBudget := v.imgCache.Budget()
	startTheme := v.ThemeMode()
	if startTheme == appearance.Dark {
		t.Fatal("precondition: ThemeMode is already Dark")
	}

	prev := v.settingsState()
	next := prev
	next.ThemeMode = appearance.Dark
	v.ApplySettings(prev, next)

	if v.ThemeMode() != appearance.Dark {
		t.Errorf("ThemeMode = %v, want Dark", v.ThemeMode())
	}
	if v.SortMode() != startSort {
		t.Errorf("SortMode = %v, want unchanged %v", v.SortMode(), startSort)
	}
	if got := v.sortOp.lifecycle.currentRevision(); got != startRev {
		t.Errorf("sort revision = %d, want %d (theme-only apply must not restart sort)", got, startRev)
	}
	if got := v.imgCache.Budget(); got != startBudget {
		t.Errorf("imgCache.Budget() = %d, want unchanged %d", got, startBudget)
	}
}

func TestApplySettings_RetunesImageCacheWithoutSorting(t *testing.T) {
	v := newTestViewer(t)
	t.Cleanup(func() { imaging.SetMaxEncodedBytes(0) })

	a := uitest.TempJPEGURI(t, "a.jpg", 8, 8, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 8, 8, color.White)
	dropAndWait(t, v, a, b)

	startRev := v.sortOp.lifecycle.currentRevision()
	startSort := v.SortMode()
	startTheme := v.ThemeMode()

	prev := v.settingsState()
	next := prev
	next.MaxImageCacheMB = v.MaxImageCacheMB() + 64
	v.ApplySettings(prev, next)

	if got, want := v.imgCache.Budget(), int64(next.MaxImageCacheMB)*bytesPerMB; got != want {
		t.Errorf("imgCache.Budget() = %d, want %d", got, want)
	}
	if v.SortMode() != startSort {
		t.Errorf("SortMode = %v, want unchanged %v", v.SortMode(), startSort)
	}
	if got := v.sortOp.lifecycle.currentRevision(); got != startRev {
		t.Errorf("sort revision = %d, want %d", got, startRev)
	}
	if v.ThemeMode() != startTheme {
		t.Errorf("ThemeMode = %v, want unchanged %v", v.ThemeMode(), startTheme)
	}
}

func TestApplySettings_ChangesSortWhenAsked(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 8, 8, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 8, 8, color.White)
	dropAndWait(t, v, a, b)

	if v.SortMode() == filesort.BySize {
		t.Fatal("precondition: SortMode is already BySize")
	}

	prev := v.settingsState()
	next := prev
	next.SortMode = filesort.BySize.PrefValue()
	v.ApplySettings(prev, next)
	waitForSort(t, v)

	if v.SortMode() != filesort.BySize {
		t.Errorf("SortMode = %v, want BySize", v.SortMode())
	}
}

// The main window stays live while Settings is open: M, S, Shift+P and
// Up/Down all change standing preferences the Settings form snapshotted at
// Show. Applying an unrelated Settings edit must not push those stale
// snapshot values back over the live ones.
func TestApplySettings_DoesNotRevertLiveShortcutChanges(t *testing.T) {
	v := newTestViewer(t)

	prev := v.settingsState() // the snapshot Show would seed the form with

	v.toggleMergeMode() // the user presses M in the main window
	if !v.MergeMode() {
		t.Fatal("precondition: merge mode is off after toggling")
	}

	next := prev
	next.StaticWindowSize = !next.StaticWindowSize // an unrelated control edit
	v.ApplySettings(prev, next)

	if !v.MergeMode() {
		t.Error("applying an unrelated setting reverted the live merge-mode change")
	}
	if v.StaticWindowSize() != next.StaticWindowSize {
		t.Errorf("StaticWindowSize = %v, want %v", v.StaticWindowSize(), next.StaticWindowSize)
	}
}
