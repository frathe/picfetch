package settingswin

import (
	"errors"
	"os"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/appearance"
	"github.com/frathe/picfetch/internal/filesort"
	"github.com/frathe/picfetch/internal/preferences"
	"github.com/frathe/picfetch/internal/ui/widgets"
)

// testApp is shared across every test below, mirroring internal/ui's own
// testApp (harness_test.go): test.NewApp() resets process-global caches
// (font shaping, theme), so building one per test would pay that cost on
// every single test instead of once for the package.
var testApp fyne.App

type metadataApp struct {
	fyne.App
	metadata fyne.AppMetadata
}

func (a metadataApp) Metadata() fyne.AppMetadata { return a.metadata }

func TestMain(m *testing.M) {
	testApp = test.NewApp()
	os.Exit(m.Run())
}

// fakeHost records what the panel asked the app to do. prefs is the snapshot
// Show seeds from; applyCalls is every ApplySettings push. Mirrors
// internal/ui/deletion's own fakeHost: the panel can be driven, and every
// effect observed, without a real viewer or window.
type fakeHost struct {
	prefs preferences.State

	applyCalls []preferences.State

	updateCallbacks []UpdateCallbacks
	performErr      error
	performCalls    int
}

func (f *fakeHost) ApplySettings(s preferences.State) {
	f.prefs = s
	f.applyCalls = append(f.applyCalls, s)
}
func (f *fakeHost) CheckForUpdatesNow(callbacks UpdateCallbacks) {
	f.updateCallbacks = append(f.updateCallbacks, callbacks)
}
func (f *fakeHost) PerformUpdate() error {
	f.performCalls++
	return f.performErr
}

func showSettings(t *testing.T, host *fakeHost) *Window {
	t.Helper()
	w := New(testApp, host)
	w.Show(host.prefs)
	t.Cleanup(func() {
		if win := w.win.Window(); win != nil {
			win.Close()
		}
	})
	return w
}

func lastApply(t *testing.T, host *fakeHost) preferences.State {
	t.Helper()
	if len(host.applyCalls) == 0 {
		t.Fatal("ApplySettings was not called")
	}
	return host.applyCalls[len(host.applyCalls)-1]
}

// TestShow_SeedsEveryControlFromHostWithoutRoundTripping checks both halves
// of build's own contract: every control reflects the snapshot Show was
// given, and none of that seeding round-trips back as a spurious
// ApplySettings call - which Select.SetSelected/Check.SetChecked/Entry.SetText
// would each risk, since every one of them fires its own OnChanged.
func TestShow_SeedsEveryControlFromHostWithoutRoundTripping(t *testing.T) {
	host := &fakeHost{prefs: preferences.State{
		ThemeMode:         appearance.Dark,
		SortMode:          preferences.SortBySize,
		MergeMode:         true,
		SlideShuffle:      true,
		SlideInterval:     42 * time.Second,
		MaxScanFiles:      777,
		MaxWindowWidth:    1800,
		MaxWindowHeight:   1100,
		MaxImageCacheMB:   384,
		MaxThumbCacheMB:   192,
		MaxFileSizeMB:     256,
		DuplicateDistance: 7,
		StaticWindowSize:  true,
	}}
	w := showSettings(t, host)

	if got, want := w.themeSelect.Selected, appearance.DisplayName(appearance.Dark); got != want {
		t.Errorf("themeSelect.Selected = %q, want %q", got, want)
	}
	if got, want := w.sortSelect.Selected, filesort.DisplayName(filesort.BySize); got != want {
		t.Errorf("sortSelect.Selected = %q, want %q", got, want)
	}
	if !w.mergeCheck.Checked {
		t.Error("mergeCheck should be checked, seeded from MergeMode = true")
	}
	if !w.shuffleCheck.Checked {
		t.Error("shuffleCheck should be checked, seeded from SlideShuffle = true")
	}
	if !w.staticSizeCheck.Checked {
		t.Error("staticSizeCheck should be checked, seeded from StaticWindowSize = true")
	}
	if got, want := w.intervalEntry.Text, "42"; got != want {
		t.Errorf("intervalEntry.Text = %q, want %q", got, want)
	}
	if got, want := w.maxScanEntry.Text, "777"; got != want {
		t.Errorf("maxScanEntry.Text = %q, want %q", got, want)
	}
	if got, want := w.maxWidthEntry.Text, "1800"; got != want {
		t.Errorf("maxWidthEntry.Text = %q, want %q", got, want)
	}
	if got, want := w.maxHeightEntry.Text, "1100"; got != want {
		t.Errorf("maxHeightEntry.Text = %q, want %q", got, want)
	}
	if got, want := w.imgCacheEntry.Text, "384"; got != want {
		t.Errorf("imgCacheEntry.Text = %q, want %q", got, want)
	}
	if got, want := w.thumbCacheEntry.Text, "192"; got != want {
		t.Errorf("thumbCacheEntry.Text = %q, want %q", got, want)
	}
	if got, want := w.maxFileSizeEntry.Text, "256"; got != want {
		t.Errorf("maxFileSizeEntry.Text = %q, want %q", got, want)
	}
	if got, want := w.dupeDistSlider.Value, 7.0; got != want {
		t.Errorf("dupeDistSlider.Value = %v, want %v", got, want)
	}
	if got, want := w.dupeDistValue.Text, "7"; got != want {
		t.Errorf("dupeDistValue.Text = %q, want %q", got, want)
	}

	if len(host.applyCalls) != 0 {
		t.Errorf("seeding the controls should not call ApplySettings, got %d calls", len(host.applyCalls))
	}
}

func TestThemeSelect_ChangeCallsSetThemeMode(t *testing.T) {
	host := &fakeHost{prefs: preferences.State{ThemeMode: appearance.System}}
	w := showSettings(t, host)

	w.themeSelect.SetSelected(appearance.DisplayName(appearance.Light))

	if got := lastApply(t, host); got.ThemeMode != appearance.Light {
		t.Errorf("ApplySettings ThemeMode = %v, want Light", got.ThemeMode)
	}
	if len(host.applyCalls) != 1 {
		t.Errorf("ApplySettings calls = %d, want 1", len(host.applyCalls))
	}
}

func TestShow_RaisesTheSameWindowOnASecondCall(t *testing.T) {
	w := New(testApp, &fakeHost{})

	w.Show(preferences.State{})
	t.Cleanup(func() { w.win.Window().Close() })
	win := w.win.Window()

	w.Show(preferences.State{})

	if w.win.Window() != win {
		t.Error("a second Show should raise the existing window, not open a new one")
	}
}

func TestSortSelect_ChangeCallsSetSortMode(t *testing.T) {
	host := &fakeHost{prefs: preferences.State{SortMode: preferences.SortByName}}
	w := showSettings(t, host)

	w.sortSelect.SetSelected(filesort.DisplayName(filesort.ByCaptureDate))

	if got := lastApply(t, host); got.SortMode != preferences.SortByCaptureDate {
		t.Errorf("ApplySettings SortMode = %q, want %q", got.SortMode, preferences.SortByCaptureDate)
	}
	if len(host.applyCalls) != 1 {
		t.Errorf("ApplySettings calls = %d, want 1", len(host.applyCalls))
	}
}

func TestChecks_ChangeCallTheMatchingSetter(t *testing.T) {
	host := &fakeHost{}
	w := showSettings(t, host)

	w.mergeCheck.SetChecked(true)
	w.shuffleCheck.SetChecked(true)

	if len(host.applyCalls) != 2 {
		t.Fatalf("ApplySettings calls = %d, want 2", len(host.applyCalls))
	}
	if !host.applyCalls[0].MergeMode {
		t.Errorf("first ApplySettings MergeMode = false, want true")
	}
	if !host.applyCalls[1].SlideShuffle || !host.applyCalls[1].MergeMode {
		t.Errorf("second ApplySettings = %+v, want MergeMode and SlideShuffle both true", host.applyCalls[1])
	}
}

// TestFavPreviewCheck_ReflectsHostValue checks both states of the seed, the
// same reason TestShow_SeedsEveryControlFromHostWithoutRoundTripping checks
// every other control against a non-default value - a check seeded from a
// bool that only ever tested one branch (e.g. always false) wouldn't catch
// Checked never actually being assigned.
func TestFavPreviewCheck_ReflectsHostValue(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"checked when host reports true", true},
		{"unchecked when host reports false", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			host := &fakeHost{prefs: preferences.State{FavoritePreviewCache: tc.want}}
			w := showSettings(t, host)

			if w.favPreviewCheck.Checked != tc.want {
				t.Errorf("favPreviewCheck.Checked = %v, want %v (seeded from FavoritePreviewCache)", w.favPreviewCheck.Checked, tc.want)
			}
		})
	}
}

func TestFavPreviewCheck_ChangeCallsSetFavoritePreviewCache(t *testing.T) {
	host := &fakeHost{prefs: preferences.State{FavoritePreviewCache: false}}
	w := showSettings(t, host)

	w.favPreviewCheck.SetChecked(true)

	if got := lastApply(t, host); !got.FavoritePreviewCache {
		t.Errorf("ApplySettings FavoritePreviewCache = false, want true")
	}
}

// TestUpdateCheck_ReflectsHostValue checks both states of the seed, mirroring
// TestFavPreviewCheck_ReflectsHostValue for the updates checkbox.
func TestUpdateCheck_ReflectsHostValue(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		{"checked when host reports true", true},
		{"unchecked when host reports false", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			host := &fakeHost{prefs: preferences.State{CheckForUpdates: tc.want}}
			w := showSettings(t, host)

			if w.updateCheck.Checked != tc.want {
				t.Errorf("updateCheck.Checked = %v, want %v (seeded from CheckForUpdates)", w.updateCheck.Checked, tc.want)
			}
			if len(host.applyCalls) != 0 {
				t.Errorf("seeding updateCheck should not call ApplySettings, got %d calls", len(host.applyCalls))
			}
		})
	}
}

func TestUpdateCheck_ChangeCallsSetCheckForUpdates(t *testing.T) {
	host := &fakeHost{prefs: preferences.State{CheckForUpdates: false}}
	w := showSettings(t, host)

	w.updateCheck.SetChecked(true)

	if got := lastApply(t, host); !got.CheckForUpdates {
		t.Errorf("ApplySettings CheckForUpdates = false, want true")
	}
}

func TestStaticSizeCheck_ReflectsHostValue(t *testing.T) {
	for _, tc := range []struct {
		name string
		want bool
	}{
		{"off", false},
		{"on", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			host := &fakeHost{prefs: preferences.State{StaticWindowSize: tc.want}}
			w := showSettings(t, host)

			if w.staticSizeCheck.Checked != tc.want {
				t.Errorf("staticSizeCheck.Checked = %v, want %v (seeded from StaticWindowSize)", w.staticSizeCheck.Checked, tc.want)
			}
			if len(host.applyCalls) != 0 {
				t.Errorf("seeding staticSizeCheck should not call ApplySettings, got %d calls", len(host.applyCalls))
			}
		})
	}
}

func TestStaticSizeCheck_ChangeCallsSetStaticWindowSize(t *testing.T) {
	host := &fakeHost{prefs: preferences.State{StaticWindowSize: false}}
	w := showSettings(t, host)

	w.staticSizeCheck.SetChecked(true)

	if got := lastApply(t, host); !got.StaticWindowSize {
		t.Errorf("ApplySettings StaticWindowSize = false, want true")
	}
}

func TestIntervalEntry_ValidChangeCallsSetSlideInterval(t *testing.T) {
	host := &fakeHost{}
	w := showSettings(t, host)

	w.intervalEntry.SetText("15")

	if got := lastApply(t, host); got.SlideInterval != 15*time.Second {
		t.Errorf("ApplySettings SlideInterval = %v, want 15s", got.SlideInterval)
	}
}

// TestIntervalEntry_InvalidTextIsIgnored checks that neither an empty field
// (the natural mid-edit state while retyping a value) nor outright garbage
// reaches the host. Values too large to fit in time.Duration are rejected
// too, rather than overflowing to a negative duration.
func TestIntervalEntry_InvalidTextIsIgnored(t *testing.T) {
	host := &fakeHost{}
	w := showSettings(t, host)

	for _, text := range []string{"", "abc", "-5", "0", "9223372037", "999999999999999999999999"} {
		w.intervalEntry.SetText(text)
	}

	if len(host.applyCalls) != 0 {
		t.Errorf("ApplySettings calls = %d, want none for invalid input", len(host.applyCalls))
	}
}

func TestMaxScanEntry_ValidChangeCallsSetMaxScan(t *testing.T) {
	host := &fakeHost{}
	w := showSettings(t, host)

	w.maxScanEntry.SetText("250000")

	if got := lastApply(t, host); got.MaxScanFiles != 250000 {
		t.Errorf("ApplySettings MaxScanFiles = %d, want 250000", got.MaxScanFiles)
	}
}

func TestMaxScanEntry_InvalidTextIsIgnored(t *testing.T) {
	host := &fakeHost{}
	w := showSettings(t, host)

	for _, text := range []string{"", "abc", "-1", "0"} {
		w.maxScanEntry.SetText(text)
	}

	if len(host.applyCalls) != 0 {
		t.Errorf("ApplySettings calls = %d, want none for invalid input", len(host.applyCalls))
	}
}

// TestMaxScanEntry_AcceptsAValueAboveMaxMemoryMB locks the unbounded path:
// scan count is not a memory budget, so a value that the three memory
// entries would reject must still reach ApplySettings. Without this, wiring
// maxScan through newPositiveIntEntry(..., maxMemoryMB, ...) would still
// pass TestMaxScanEntry_ValidChangeCallsSetMaxScan (250000 < maxMemoryMB).
func TestMaxScanEntry_AcceptsAValueAboveMaxMemoryMB(t *testing.T) {
	host := &fakeHost{}
	w := showSettings(t, host)

	w.maxScanEntry.SetText("1048577")

	if got := lastApply(t, host); got.MaxScanFiles != 1048577 {
		t.Errorf("ApplySettings MaxScanFiles = %d, want 1048577 (no memory ceiling on scan count)", got.MaxScanFiles)
	}
}

func TestMaxWidthEntry_ValidChangeCallsSetMaxWindowWidth(t *testing.T) {
	host := &fakeHost{}
	w := showSettings(t, host)

	w.maxWidthEntry.SetText("1600")

	if got := lastApply(t, host); got.MaxWindowWidth != 1600 {
		t.Errorf("ApplySettings MaxWindowWidth = %v, want 1600", got.MaxWindowWidth)
	}
}

func TestMaxWidthEntry_InvalidTextIsIgnored(t *testing.T) {
	host := &fakeHost{}
	w := showSettings(t, host)

	for _, text := range []string{"", "abc", "-1", "0"} {
		w.maxWidthEntry.SetText(text)
	}

	if len(host.applyCalls) != 0 {
		t.Errorf("ApplySettings calls = %d, want none for invalid input", len(host.applyCalls))
	}
}

func TestMaxHeightEntry_ValidChangeCallsSetMaxWindowHeight(t *testing.T) {
	host := &fakeHost{}
	w := showSettings(t, host)

	w.maxHeightEntry.SetText("1000")

	if got := lastApply(t, host); got.MaxWindowHeight != 1000 {
		t.Errorf("ApplySettings MaxWindowHeight = %v, want 1000", got.MaxWindowHeight)
	}
}

func TestMaxHeightEntry_InvalidTextIsIgnored(t *testing.T) {
	host := &fakeHost{}
	w := showSettings(t, host)

	for _, text := range []string{"", "abc", "-1", "0"} {
		w.maxHeightEntry.SetText(text)
	}

	if len(host.applyCalls) != 0 {
		t.Errorf("ApplySettings calls = %d, want none for invalid input", len(host.applyCalls))
	}
}

func TestImgCacheEntry_ValidChangeCallsSetMaxImageCacheMB(t *testing.T) {
	host := &fakeHost{}
	w := showSettings(t, host)

	w.imgCacheEntry.SetText("768")

	if got := lastApply(t, host); got.MaxImageCacheMB != 768 {
		t.Errorf("ApplySettings MaxImageCacheMB = %d, want 768", got.MaxImageCacheMB)
	}
}

// TestImgCacheEntry_AcceptsMaxMemoryMB locks the inclusive ceiling the three
// memory OnChanged blocks already had (`n <= maxMemoryMB`). The invalid-text
// table only checks one-over (1048577).
func TestImgCacheEntry_AcceptsMaxMemoryMB(t *testing.T) {
	host := &fakeHost{}
	w := showSettings(t, host)

	w.imgCacheEntry.SetText("1048576")

	if got := lastApply(t, host); got.MaxImageCacheMB != maxMemoryMB {
		t.Errorf("ApplySettings MaxImageCacheMB = %d, want %d", got.MaxImageCacheMB, maxMemoryMB)
	}
}

func TestThumbCacheEntry_ValidChangeCallsSetMaxThumbCacheMB(t *testing.T) {
	host := &fakeHost{}
	w := showSettings(t, host)

	w.thumbCacheEntry.SetText("128")

	if got := lastApply(t, host); got.MaxThumbCacheMB != 128 {
		t.Errorf("ApplySettings MaxThumbCacheMB = %d, want 128", got.MaxThumbCacheMB)
	}
}

func TestMaxFileSizeEntry_ValidChangeCallsSetMaxFileSizeMB(t *testing.T) {
	host := &fakeHost{}
	w := showSettings(t, host)

	w.maxFileSizeEntry.SetText("64")

	if got := lastApply(t, host); got.MaxFileSizeMB != 64 {
		t.Errorf("ApplySettings MaxFileSizeMB = %d, want 64", got.MaxFileSizeMB)
	}
}

func TestDupeDistSlider_ChangeCallsSetDuplicateDistance(t *testing.T) {
	host := &fakeHost{prefs: preferences.State{DuplicateDistance: 10}}
	w := showSettings(t, host)

	w.dupeDistSlider.SetValue(0)

	got := lastApply(t, host)
	if got.DuplicateDistance != 0 || !got.DuplicateDistanceSet {
		t.Errorf("ApplySettings DuplicateDistance = (%d, set=%v), want (0, set=true)", got.DuplicateDistance, got.DuplicateDistanceSet)
	}
	if got, want := w.dupeDistValue.Text, "0"; got != want {
		t.Errorf("dupeDistValue.Text = %q, want %q", got, want)
	}
}

// The memory entries reject one thing the other numeric entries don't: a
// value past maxMemoryMB, which the host would shift left by 20 into an
// int64 byte budget. "1048577" is one megabyte past the ceiling, and
// "99999999999999999999" is past what Atoi can parse at all - both have to
// be ignored rather than wrapped into a nonsense budget, the same guard
// TestIntervalEntry_InvalidTextIsIgnored makes for time.Duration.
func TestMemoryEntries_InvalidTextIsIgnored(t *testing.T) {
	bad := []string{"", "abc", "-1", "0", "1048577", "99999999999999999999"}

	cases := []struct {
		name  string
		entry func(*Window) *widget.Entry
	}{
		{"image cache", func(w *Window) *widget.Entry { return w.imgCacheEntry }},
		{"thumbnail cache", func(w *Window) *widget.Entry { return w.thumbCacheEntry }},
		{"max file size", func(w *Window) *widget.Entry { return w.maxFileSizeEntry }},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			host := &fakeHost{}
			w := showSettings(t, host)

			for _, text := range bad {
				c.entry(w).SetText(text)
			}

			if len(host.applyCalls) != 0 {
				t.Errorf("ApplySettings calls = %d, want none for invalid input", len(host.applyCalls))
			}
		})
	}
}

func TestOpen_ReflectsWindowLifecycle(t *testing.T) {
	w := New(testApp, &fakeHost{})

	if w.Open() {
		t.Fatal("Open() = true before Show was ever called")
	}

	w.Show(preferences.State{})
	if !w.Open() {
		t.Error("Open() = false, want true once Show has run")
	}

	w.win.Window().Close()
	if w.Open() {
		t.Error("Open() = true, want false once the window is closed")
	}
	if w.sortSelect != nil {
		t.Error("expected the widget fields to be cleared once the window closes")
	}
}

// The saved size is deliberately different from the built-in size in both
// directions, so observing it proves restoration applied it.
func TestRestoreGeometry_OpensAtTheSavedGeometry(t *testing.T) {
	w := New(testApp, &fakeHost{})
	w.RestoreGeometry(widgets.Geometry{X: 210, Y: 220, PositionSet: true, Size: fyne.NewSize(700, 750)})

	w.Show(preferences.State{})
	t.Cleanup(func() { w.win.Window().Close() })

	if got, want := w.win.Window().Canvas().Size(), fyne.NewSize(700, 750); got != want {
		t.Errorf("window size = %v, want the saved %v", got, want)
	}

	got := w.Geometry()
	if !got.PositionSet || got.X != 210 || got.Y != 220 {
		t.Errorf("Geometry() position = (%d, %d, set=%v), want the saved (210, 220, set=true)", got.X, got.Y, got.PositionSet)
	}
}

func TestGeometry_TracksAResizeAndOutlivesTheWindow(t *testing.T) {
	w := New(testApp, &fakeHost{})
	w.RestoreGeometry(widgets.Geometry{})

	w.Show(preferences.State{})
	w.win.Window().Resize(fyne.NewSize(700, 750))
	w.win.Window().Close()

	if got, want := w.Geometry().Size, fyne.NewSize(700, 750); got != want {
		t.Errorf("Geometry().Size after closing = %v, want the last tracked %v", got, want)
	}
}

func TestStopTracking_IsSafeWithNoWindowOpen(t *testing.T) {
	New(testApp, &fakeHost{}).StopTracking()
}

// Width only: windowH is below the laid-out form's own MinSize, so Fyne
// grows the window past it - which is exactly what the window has always
// done, and what makes the height a poor thing to assert on here.
func TestShow_WithoutRestoreGeometryUsesTheBuiltInSize(t *testing.T) {
	w := New(testApp, &fakeHost{})

	w.Show(preferences.State{})
	t.Cleanup(func() { w.win.Window().Close() })

	if got, want := w.win.Window().Canvas().Size().Width, float32(windowW); got != want {
		t.Errorf("window width = %v, want the built-in %v", got, want)
	}
}

// TestNewPositiveIntEntry pins the helper in isolation, before build switches
// onto it. Invalid text is the mid-edit state (empty, garbage, zero, overflow)
// and must not reach set — the same contract
// TestMaxScanEntry_InvalidTextIsIgnored / TestMemoryEntries_InvalidTextIsIgnored
// already make through the widgets.
func TestNewPositiveIntEntry(t *testing.T) {
	validate := func(string) error { return nil }

	t.Run("seeds Text from get without calling set", func(t *testing.T) {
		var calls []int
		e := newPositiveIntEntry(func() int { return 42 }, func(n int) { calls = append(calls, n) }, 0, validate)
		if got, want := e.Text, "42"; got != want {
			t.Errorf("Text = %q, want %q", got, want)
		}
		if len(calls) != 0 {
			t.Errorf("set called on seed: %v", calls)
		}
		if e.Validator == nil {
			t.Error("Validator is nil, want the validate argument")
		}
	})

	t.Run("valid change calls set", func(t *testing.T) {
		var calls []int
		e := newPositiveIntEntry(func() int { return 1 }, func(n int) { calls = append(calls, n) }, 0, validate)
		e.SetText("100")
		if len(calls) != 1 || calls[0] != 100 {
			t.Errorf("set calls = %v, want [100]", calls)
		}
	})

	t.Run("invalid text is ignored", func(t *testing.T) {
		var calls []int
		e := newPositiveIntEntry(func() int { return 1 }, func(n int) { calls = append(calls, n) }, 0, validate)
		for _, text := range []string{"", "abc", "-1", "0", "99999999999999999999"} {
			e.SetText(text)
		}
		if len(calls) != 0 {
			t.Errorf("set calls = %v, want none for invalid input", calls)
		}
	})

	t.Run("max 0 has no ceiling", func(t *testing.T) {
		var calls []int
		e := newPositiveIntEntry(func() int { return 1 }, func(n int) { calls = append(calls, n) }, 0, validate)
		e.SetText("1048577")
		if len(calls) != 1 || calls[0] != 1048577 {
			t.Errorf("set calls = %v, want [1048577] (unbounded)", calls)
		}
	})

	t.Run("maxMemoryMB is accepted, one over is ignored", func(t *testing.T) {
		var calls []int
		e := newPositiveIntEntry(func() int { return 1 }, func(n int) { calls = append(calls, n) }, maxMemoryMB, validate)
		e.SetText("1048576")
		e.SetText("1048577")
		if len(calls) != 1 || calls[0] != maxMemoryMB {
			t.Errorf("set calls = %v, want [%d] (the ceiling, not one over)", calls, maxMemoryMB)
		}
	})
}

func newUpdateTestWindow(t *testing.T, host *fakeHost) *Window {
	t.Helper()
	w := New(testApp, host)
	w.Show(host.prefs)
	t.Cleanup(func() {
		if win := w.win.Window(); win != nil {
			win.Close()
		}
	})
	return w
}

func settingsTabs(t *testing.T, w *Window) *container.AppTabs {
	t.Helper()

	tabs, ok := w.win.Window().Content().(*container.AppTabs)
	if !ok {
		t.Fatalf("settings content = %T, want *container.AppTabs", w.win.Window().Content())
	}
	return tabs
}

func tabVBox(t *testing.T, item *container.TabItem) *fyne.Container {
	t.Helper()

	padded, ok := item.Content.(*fyne.Container)
	if !ok || len(padded.Objects) != 1 {
		t.Fatalf("%q tab content = %T, want one padded scroll", item.Text, item.Content)
	}
	scroll, ok := padded.Objects[0].(*container.Scroll)
	if !ok {
		t.Fatalf("%q tab padded child = %T, want *container.Scroll", item.Text, padded.Objects[0])
	}
	content, ok := scroll.Content.(*fyne.Container)
	if !ok {
		t.Fatalf("%q tab scroll content = %T, want *fyne.Container", item.Text, scroll.Content)
	}
	return content
}

func containsCanvasObject(root, target fyne.CanvasObject) bool {
	if root == target {
		return true
	}

	switch object := root.(type) {
	case *fyne.Container:
		for _, child := range object.Objects {
			if containsCanvasObject(child, target) {
				return true
			}
		}
	case *container.Scroll:
		return containsCanvasObject(object.Content, target)
	case *widget.Form:
		for _, item := range object.Items {
			if containsCanvasObject(item.Widget, target) {
				return true
			}
		}
	}
	return false
}

func TestSettingsTabs_GroupControlsAndOpenOnGeneral(t *testing.T) {
	w := newUpdateTestWindow(t, &fakeHost{})
	tabs := settingsTabs(t, w)

	wantLabels := []string{"General", "Appearance", "Updates", "Limits"}
	if len(tabs.Items) != len(wantLabels) {
		t.Fatalf("tab count = %d, want %d", len(tabs.Items), len(wantLabels))
	}
	for i, want := range wantLabels {
		if got := tabs.Items[i].Text; got != want {
			t.Errorf("tab %d label = %q, want %q", i, got, want)
		}
	}
	if got := tabs.SelectedIndex(); got != 0 {
		t.Errorf("selected tab = %d, want General at index 0", got)
	}

	limitControls := map[string]fyne.CanvasObject{
		"scan cap": w.maxScanEntry, "image cache": w.imgCacheEntry,
		"thumbnail cache": w.thumbCacheEntry, "file-size cap": w.maxFileSizeEntry,
	}

	general := tabs.Items[0].Content
	for name, control := range map[string]fyne.CanvasObject{
		"sort": w.sortSelect, "interval": w.intervalEntry,
		"window width": w.maxWidthEntry, "window height": w.maxHeightEntry,
		"duplicate distance": w.dupeDistSlider,
		"merge":              w.mergeCheck, "shuffle": w.shuffleCheck, "favorite previews": w.favPreviewCheck,
		"fixed window size": w.staticSizeCheck,
	} {
		if !containsCanvasObject(general, control) {
			t.Errorf("General tab does not contain %s control", name)
		}
	}
	for name, control := range limitControls {
		if containsCanvasObject(general, control) {
			t.Errorf("General tab unexpectedly contains %s control", name)
		}
	}
	for name, control := range map[string]fyne.CanvasObject{
		"appearance": w.themeSelect, "automatic updates": w.updateCheck, "manual updates": w.updateNow,
	} {
		if containsCanvasObject(general, control) {
			t.Errorf("General tab unexpectedly contains %s control", name)
		}
	}

	appearanceTab := tabs.Items[1].Content
	if !containsCanvasObject(appearanceTab, w.themeSelect) {
		t.Error("Appearance tab does not contain appearance selector")
	}
	if containsCanvasObject(appearanceTab, w.sortSelect) || containsCanvasObject(appearanceTab, w.updateCheck) {
		t.Error("Appearance tab contains a General or Updates control")
	}

	updates := tabs.Items[2].Content
	if !containsCanvasObject(updates, w.updateCheck) || !containsCanvasObject(updates, w.updateNow) {
		t.Error("Updates tab does not contain both update controls")
	}
	if containsCanvasObject(updates, w.themeSelect) || containsCanvasObject(updates, w.sortSelect) {
		t.Error("Updates tab contains an Appearance or General control")
	}

	limits := tabs.Items[3].Content
	for name, control := range limitControls {
		if !containsCanvasObject(limits, control) {
			t.Errorf("Limits tab does not contain %s control", name)
		}
	}
	for name, control := range map[string]fyne.CanvasObject{
		"sort": w.sortSelect, "interval": w.intervalEntry,
		"window width": w.maxWidthEntry, "window height": w.maxHeightEntry,
		"duplicate distance": w.dupeDistSlider, "appearance": w.themeSelect,
		"automatic updates": w.updateCheck, "manual updates": w.updateNow,
	} {
		if containsCanvasObject(limits, control) {
			t.Errorf("Limits tab unexpectedly contains %s control", name)
		}
	}
}

func TestUpdatesTab_ShowsCurrentVersionAndBuild(t *testing.T) {
	app := metadataApp{
		App:      testApp,
		metadata: fyne.AppMetadata{Version: "2.3.4", Build: 567},
	}
	w := New(app, &fakeHost{})
	w.Show(preferences.State{})
	t.Cleanup(func() { w.win.Window().Close() })

	updates := tabVBox(t, settingsTabs(t, w).Items[2])
	if got, want := w.updateVersion.Text, "Version 2.3.4 (Build 567)"; got != want {
		t.Errorf("version label = %q, want %q", got, want)
	}
	if got := updates.Objects[0]; got != w.updateVersion {
		t.Errorf("first Updates object = %T, want current version label", got)
	}
}

// TestUpdateNow_IsDirectlyBelowTheAutomaticCheckAndStartsOneFlow locks the
// Updates-tab placement and makes the double-activation guard observable
// through the consumer-side Host rather than a real updater worker.
func TestUpdateNow_IsDirectlyBelowTheAutomaticCheckAndStartsOneFlow(t *testing.T) {
	host := &fakeHost{}
	w := newUpdateTestWindow(t, host)

	tabs := settingsTabs(t, w)
	updates := tabVBox(t, tabs.Items[2])
	if len(updates.Objects) != 3 {
		t.Fatalf("Updates tab object count = %d, want 3", len(updates.Objects))
	}
	if got := updates.Objects[1]; got != w.updateCheck {
		t.Errorf("object before Check now = %T, want the automatic update checkbox", got)
	}
	if got := updates.Objects[2]; got != w.updateNow {
		t.Errorf("last settings object = %T, want Check now button", got)
	}

	test.Tap(w.updateNow)
	test.Tap(w.updateNow)

	if got := len(host.updateCallbacks); got != 1 {
		t.Errorf("CheckForUpdatesNow calls = %d, want one despite a duplicate tap", got)
	}
	if !w.updateNow.Disabled() || !w.updateCheck.Disabled() {
		t.Error("update controls should be disabled while checking")
	}
	if got, want := w.updateMessage.Text, "Checking for updates…"; got != want {
		t.Errorf("checking message = %q, want %q", got, want)
	}
	if w.updateInfinite == nil || w.updateInfinite.Hidden || w.updateDialog == nil {
		t.Error("checking should show an indeterminate modal dialog")
	}
}

func TestSettingsContentFitsActualWindow(t *testing.T) {
	w := newUpdateTestWindow(t, &fakeHost{})
	minimum := w.win.Window().Content().MinSize()
	size := w.win.Window().Canvas().Size()
	if minimum.Width > size.Width || minimum.Height > size.Height {
		t.Errorf("settings content minimum = %v, exceeds actual window size %v", minimum, size)
	}
}

func TestUpdateDownloadProgress_SwitchesAndClamps(t *testing.T) {
	host := &fakeHost{}
	w := newUpdateTestWindow(t, host)
	test.Tap(w.updateNow)
	callbacks := host.updateCallbacks[0]

	callbacks.Downloading("v2.3.4")
	if got, want := w.updateMessage.Text, "Downloading version v2.3.4"; got != want {
		t.Errorf("download message = %q, want %q", got, want)
	}
	if !w.updateProgress.Hidden || w.updateInfinite.Hidden {
		t.Error("download should start indeterminate until a positive total arrives")
	}

	callbacks.Progress(9, 0)
	if !w.updateProgress.Hidden || w.updateInfinite.Hidden {
		t.Error("a zero total must remain indeterminate")
	}
	callbacks.Progress(25, 100)
	if w.updateProgress.Hidden || !w.updateInfinite.Hidden {
		t.Error("a positive total must switch to the determinate bar")
	}
	if got, want := w.updateProgress.Value, 0.25; got != want {
		t.Errorf("determinate value = %v, want %v", got, want)
	}
	callbacks.Progress(-1, 100)
	if got := w.updateProgress.Value; got != 0 {
		t.Errorf("negative downloaded value = %v, want clamped 0", got)
	}
	callbacks.Progress(101, 100)
	if got := w.updateProgress.Value; got != 1 {
		t.Errorf("oversized downloaded value = %v, want clamped 1", got)
	}
	callbacks.Progress(1, -1)
	if !w.updateProgress.Hidden || w.updateInfinite.Hidden {
		t.Error("an unknown total after a known one must restore the infinite bar")
	}
}

func TestUpdateCurrentAndFailureRestoreControls(t *testing.T) {
	t.Run("current", func(t *testing.T) {
		host := &fakeHost{}
		w := newUpdateTestWindow(t, host)
		test.Tap(w.updateNow)
		host.updateCallbacks[0].Current()

		if got, want := w.updateMessage.Text, "You are on the current version."; got != want {
			t.Errorf("current message = %q, want %q", got, want)
		}
		if w.updateNow.Disabled() || w.updateCheck.Disabled() || w.updateActive {
			t.Error("current result should restore the controls and finish the request")
		}
		if w.updateChoices == nil || w.updateChoices.Selected() != 0 {
			t.Error("current dialog should have an OK ChoicePanel selected at index zero")
		}
	})

	t.Run("failure", func(t *testing.T) {
		host := &fakeHost{}
		w := newUpdateTestWindow(t, host)
		test.Tap(w.updateNow)
		host.updateCallbacks[0].Failed(errors.New("offline"))

		if got, want := w.updateMessage.Text, "Could not check for updates: offline"; got != want {
			t.Errorf("failure message = %q, want %q", got, want)
		}
		if w.updateNow.Disabled() || w.updateCheck.Disabled() || w.updateActive {
			t.Error("failure result should restore the controls and finish the request")
		}
	})
}

func TestUpdateReady_LaterIsDefaultAndPerformIsSecond(t *testing.T) {
	host := &fakeHost{}
	w := newUpdateTestWindow(t, host)
	test.Tap(w.updateNow)
	host.updateCallbacks[0].Ready("v2.3.4")

	if got, want := w.updateMessage.Text, "Update downloaded successfully."; got != want {
		t.Errorf("ready message = %q, want %q", got, want)
	}
	if got := w.updateChoices.Selected(); got != 0 {
		t.Errorf("default ready choice = %d, want Later at index 0", got)
	}
	if got := w.win.Window().Canvas().Focused(); got != w.updateChoices {
		t.Errorf("focused update control = %T, want the ready ChoicePanel", got)
	}
	w.updateChoices.Confirm()
	if host.performCalls != 0 {
		t.Fatalf("default Later choice performed %d updates, want none", host.performCalls)
	}
	if w.updateNow.Disabled() || w.updateCheck.Disabled() || w.updateActive {
		t.Fatal("Later should restore controls and leave Settings recoverable")
	}

	// A fresh flow proves the second position is the disruptive choice.
	test.Tap(w.updateNow)
	host.updateCallbacks[1].Ready("v2.3.4")
	w.updateChoices.Select(1)
	w.updateChoices.Confirm()
	if got := host.performCalls; got != 1 {
		t.Errorf("second ready choice performed %d updates, want one", got)
	}
	if !w.updateNow.Disabled() || !w.updateCheck.Disabled() {
		t.Error("a successful Perform update request should keep controls disabled for app shutdown")
	}
	// Calling the already-dismissed test seam again must not duplicate the
	// disruptive host call; a real user cannot click a hidden button either.
	w.updateChoices.Confirm()
	if got := host.performCalls; got != 1 {
		t.Errorf("duplicate Perform update calls = %d, want one", got)
	}
}

func TestUpdatePerformFailureShowsRecoverableError(t *testing.T) {
	host := &fakeHost{performErr: errors.New("stage disappeared")}
	w := newUpdateTestWindow(t, host)
	test.Tap(w.updateNow)
	host.updateCallbacks[0].Ready("v2.3.4")
	w.updateChoices.Select(1)
	w.updateChoices.Confirm()

	if got := host.performCalls; got != 1 {
		t.Errorf("PerformUpdate calls = %d, want one", got)
	}
	if got, want := w.updateMessage.Text, "Could not perform update: stage disappeared"; got != want {
		t.Errorf("perform failure message = %q, want %q", got, want)
	}
	if w.updateNow.Disabled() || w.updateCheck.Disabled() || w.updateActive {
		t.Error("a failed Perform update should restore controls for recovery")
	}
}

func TestUpdateCallbacksAfterSettingsCloseAreIgnored(t *testing.T) {
	host := &fakeHost{}
	w := newUpdateTestWindow(t, host)
	test.Tap(w.updateNow)
	callbacks := host.updateCallbacks[0]
	w.win.Window().Close()

	// These model queued worker events after the Settings parent has been
	// destroyed. The only correct result is a no-op with no replacement window.
	callbacks.Downloading("v2.3.4")
	callbacks.Progress(50, 100)
	callbacks.Current()
	callbacks.Ready("v2.3.4")
	callbacks.Failed(errors.New("late"))

	if w.Open() || w.updateDialog != nil || w.updateNow != nil {
		t.Error("late callbacks must not recreate or mutate the closed Settings UI")
	}
}
