package ui

import (
	"github.com/frathe/picfetch/internal/appearance"
	"github.com/frathe/picfetch/internal/filesort"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/preferences"
)

// settings is every standing preference the Settings window's snapshot
// reads and ApplySettings writes, and nothing else: the application
// appearance, memory budget, two geometry caps, fixed-window-size toggle,
// folder-scan cap, favorite-preview-cache toggle, and updates checkbox.
// Grouped so that surface reads as the single concern it is, and so
// run.go's currentPreferences copy is a flat field-for-field one.
//
// The three memory limits (imgCacheMB/thumbCacheMB/maxFileMB) are why this
// file exists at all: they have no single consumer to sit beside - the image
// cache is read in load.go, the thumbnail cache lives in internal/ui/grid,
// and the encoded-input ceiling is process-wide state in internal/imaging -
// while together they are one coherent thing, the app's memory budget. The
// other four fields do have a natural home each (maxScan in drop.go,
// maxWinW/maxWinH in load.go, favPreviewCache in favthumbs.go), and their
// getter/setter pairs stay there exactly as before; only the storage moved
// here so the whole settings-backed state is declared in one struct instead
// of flattened across viewer's ~70 fields.
//
// All the megabyte figures are megabytes rather than bytes because that is
// the unit the user types into the settings window and the unit
// internal/preferences round-trips; the conversion to the byte budgets
// internal/imaging actually enforces happens in the setters, which stay
// where their consumers are.
type settings struct {
	// themeMode is the Settings window's application-wide appearance choice.
	// theme.go applies it through internal/appearance, build.go restores it,
	// and currentPreferences writes it back at shutdown.
	themeMode appearance.Mode

	// maxScan caps how many images a single recursive folder scan will
	// gather - see handleDrop (drop.go). A field rather than the package
	// var it used to be, so tests shrink it per-viewer instead of
	// mutating a global.
	maxScan int

	// maxWinW/maxWinH cap how large the window is ever allowed to
	// auto-grow to fit a loaded image - see resizeToImage (load.go),
	// which never resizes past them. Fields rather than the constants
	// they used to be, so the settings window can change them per-viewer
	// and tests can shrink/grow them without touching a global.
	maxWinW, maxWinH float32

	// imgCacheMB/thumbCacheMB/maxFileMB are the app's memory budget, in
	// the megabytes the settings window shows - see memlimits.go, which
	// holds their getter/setter pairs and converts each to the byte budget
	// its consumer actually enforces.
	imgCacheMB, thumbCacheMB, maxFileMB int

	// favPreviewCache is the settings window's "Cache favorite previews on
	// disk" checkbox - see favthumbs.go for its getter/setter pair. Restored
	// from preferences.State.FavoritePreviewCache in features.go and read
	// back into it by currentPreferences (run.go).
	favPreviewCache bool

	// checkForUpdates is the settings window's "Check for updates" checkbox -
	// see autoupdate.go for its getter/setter pair, which delegates to
	// v.updater for the rest of the opt-in update state (internal/ui/
	// autoupdate.Updater, including the last-check day this struct used to
	// hold). Restored from preferences.State in features.go and read back by
	// currentPreferences.
	checkForUpdates bool

	// staticWindowSize is the settings window's "Keep a fixed window size"
	// checkbox - see load.go for its getter/setter and the autoResizeToImage
	// guard. Restored from preferences.State in features.go and read back by
	// currentPreferences.
	staticWindowSize bool

	// dupeDist is the Hamming threshold hide-duplicates groups at - the
	// saved copy of what internal/dupes holds live, pushed into the model
	// by SetDuplicateDistance below. dupeDistSet distinguishes a saved 0
	// (exact hash) from unset.
	dupeDist    int
	dupeDistSet bool
}

// bytesPerMB converts the megabyte figures above into the byte budgets
// imaging.ByteCache and imaging.SetMaxEncodedBytes take.
const bytesPerMB = 1 << 20

// The shipped defaults, derived from internal/imaging's own so there is
// exactly one place each number is chosen - see DefaultImgCacheBytes,
// DefaultThumbCacheBytes and DefaultMaxEncodedBytes for why each is what it
// is. Used by startup preference normalization when nothing was ever saved,
// the same zero-means-unset fallback maxScan and maxWinW/maxWinH use.
const (
	defaultMaxImageCacheMB = imaging.DefaultImgCacheBytes / bytesPerMB
	defaultMaxThumbCacheMB = imaging.DefaultThumbCacheBytes / bytesPerMB
	defaultMaxFileSizeMB   = imaging.DefaultMaxEncodedBytes / bytesPerMB
)

// MaxImageCacheMB reports the decoded-image cache budget,
// MaxThumbCacheMB the thumbnail cache's, and MaxFileSizeMB the
// per-file encoded ceiling - the settings window's getters.
func (v *viewer) MaxImageCacheMB() int { return v.settings.imgCacheMB }
func (v *viewer) MaxThumbCacheMB() int { return v.settings.thumbCacheMB }
func (v *viewer) MaxFileSizeMB() int   { return v.settings.maxFileMB }

// SetMaxImageCacheMB retunes the decoded-image cache's byte budget, evicting
// down to it immediately - the settings window's binding. Floored at 1 MB
// rather than 0 for the reason SetMaxScan floors at 1: a zero budget isn't a
// "no limit" any of this is written to understand. A budget too small to
// hold even one photo is still perfectly well-defined, because ByteCache
// never evicts its most recently added entry - the image on screen stays
// resident, and only its neighbors stop being kept.
func (v *viewer) SetMaxImageCacheMB(n int) {
	if n < 1 {
		n = 1
	}

	v.settings.imgCacheMB = n
	v.imgCache.SetBudget(int64(n) * bytesPerMB)
	imaging.SetMaxVectorRasterPixels(vectorRasterPixelsFor(n))
}

// vectorRasterPixelsFor derives the SVG re-render ceiling from the image
// cache budget: a quarter of the budget's bytes, at 4 bytes per RGBA
// pixel. The re-render raster is live display state rather than a cache
// entry - deliberately never charged to imgCache - so this derivation is
// how the one memory setting the user sees still bounds it.
// imaging.SetMaxVectorRasterPixels applies the floor and ceiling.
func vectorRasterPixelsFor(cacheMB int) int64 {
	return int64(cacheMB) * bytesPerMB / 4 / 4
}

// SetMaxThumbCacheMB retunes the grid's thumbnail cache the same way, through
// the one setter internal/ui/grid exposes. Lowering it while the grid is open
// is safe: a cell whose thumbnail gets evicted just decodes it again the next
// time it scrolls into view.
func (v *viewer) SetMaxThumbCacheMB(n int) {
	if n < 1 {
		n = 1
	}

	v.settings.thumbCacheMB = n
	v.grid.SetCacheBytes(int64(n) * bytesPerMB)
}

// SetMaxFileSizeMB changes the ceiling on a file's encoded size. Unlike the
// two above, the value it writes is process-wide rather than per-viewer (see
// imaging.SetMaxEncodedBytes for why), so the viewer's own field exists only
// to answer the settings window's getter.
func (v *viewer) SetMaxFileSizeMB(n int) {
	if n < 1 {
		n = 1
	}

	v.settings.maxFileMB = n
	imaging.SetMaxEncodedBytes(int64(n) * bytesPerMB)
}

// DuplicateDistance is the Hamming threshold hide-duplicates uses.
func (v *viewer) DuplicateDistance() int {
	if v.settings.dupeDistSet {
		return v.settings.dupeDist
	}
	return imaging.DuplicateMaxDistance
}

// SetDuplicateDistance clamps n to 0–32, marks it saved, and pushes it to
// the duplicate model. Live: groups rebuild if hide is already on.
//
// The clamp here is not redundant with dupes.Model.SetDistance's own: this
// one is what keeps the *saved preference* in range, since v.settings.dupeDist
// is written before the model ever sees the value and is what
// currentPreferences round-trips.
func (v *viewer) SetDuplicateDistance(n int) {
	if n < 0 {
		n = 0
	}
	if n > 32 {
		n = 32
	}
	v.settings.dupeDist = n
	v.settings.dupeDistSet = true
	v.pushDuplicateDistance(n)
}

// pushDuplicateDistance hands n to the model and, when the stored value
// actually moved, lets the grid re-apply its own view of it - re-checking a
// browsed group, re-filtering while hide is on, or just regrouping - before
// the model's observers run. That ordering is the whole point of the split:
// jumpIfHiddenExtra must see the group snapshot the grid's re-filter
// installed, exactly as it did when the grid owned both halves.
func (v *viewer) pushDuplicateDistance(n int) {
	if !v.dupes.SetDistance(n) {
		return
	}
	v.grid.DuplicateDistanceChanged()
}

// settingsState is the form snapshot Settings Show is seeded from: the
// standing preferences the window can edit, with the same getter
// substitutions the old Host surface used (SlideInterval falls back to the
// picture-frame default; DuplicateDistance falls back to
// imaging.DuplicateMaxDistance). Geometry and last-update-check day stay on
// currentPreferences; they are not form fields and ApplySettings ignores
// them.
func (v *viewer) settingsState() preferences.State {
	return preferences.State{
		SortMode:             v.SortMode().PrefValue(),
		MergeMode:            v.MergeMode(),
		ThemeMode:            v.ThemeMode(),
		SlideInterval:        v.SlideInterval(),
		SlideShuffle:         v.SlideShuffle(),
		MaxScanFiles:         v.MaxScan(),
		MaxWindowWidth:       v.MaxWindowWidth(),
		MaxWindowHeight:      v.MaxWindowHeight(),
		MaxImageCacheMB:      v.MaxImageCacheMB(),
		MaxThumbCacheMB:      v.MaxThumbCacheMB(),
		MaxFileSizeMB:        v.MaxFileSizeMB(),
		FavoritePreviewCache: v.FavoritePreviewCache(),
		CheckForUpdates:      v.CheckForUpdates(),
		StaticWindowSize:     v.StaticWindowSize(),
		DuplicateDistance:    v.DuplicateDistance(),
		DuplicateDistanceSet: v.settings.dupeDistSet,
	}
}

// ApplySettings is the Settings window's write path. next is a full form
// snapshot, not a patch: only fields that differ from the live viewer
// state run their setters, so changing the scan cap does not restart a
// sort or retune caches. Does not persist; shutdown Save still goes
// through currentPreferences.
func (v *viewer) ApplySettings(next preferences.State) {
	if next.ThemeMode != v.settings.themeMode {
		v.SetThemeMode(next.ThemeMode)
	}
	if mode := filesort.FromPref(next.SortMode); mode != v.SortMode() {
		v.SetSortMode(mode)
	}
	if next.MergeMode != v.MergeMode() {
		v.SetMergeMode(next.MergeMode)
	}
	if next.SlideShuffle != v.SlideShuffle() {
		v.SetSlideShuffle(next.SlideShuffle)
	}
	if next.SlideInterval != v.SlideInterval() {
		v.SetSlideInterval(next.SlideInterval)
	}
	if next.MaxScanFiles != v.MaxScan() {
		v.SetMaxScan(next.MaxScanFiles)
	}
	if next.MaxWindowWidth != v.MaxWindowWidth() {
		v.SetMaxWindowWidth(next.MaxWindowWidth)
	}
	if next.MaxWindowHeight != v.MaxWindowHeight() {
		v.SetMaxWindowHeight(next.MaxWindowHeight)
	}
	if next.StaticWindowSize != v.StaticWindowSize() {
		v.SetStaticWindowSize(next.StaticWindowSize)
	}
	if next.MaxImageCacheMB != v.MaxImageCacheMB() {
		v.SetMaxImageCacheMB(next.MaxImageCacheMB)
	}
	if next.MaxThumbCacheMB != v.MaxThumbCacheMB() {
		v.SetMaxThumbCacheMB(next.MaxThumbCacheMB)
	}
	if next.MaxFileSizeMB != v.MaxFileSizeMB() {
		v.SetMaxFileSizeMB(next.MaxFileSizeMB)
	}
	if next.FavoritePreviewCache != v.FavoritePreviewCache() {
		v.SetFavoritePreviewCache(next.FavoritePreviewCache)
	}
	if next.CheckForUpdates != v.CheckForUpdates() {
		v.SetCheckForUpdates(next.CheckForUpdates)
	}
	if next.DuplicateDistance != v.DuplicateDistance() || (next.DuplicateDistanceSet && !v.settings.dupeDistSet) {
		v.SetDuplicateDistance(next.DuplicateDistance)
	}
}
