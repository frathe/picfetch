// Package preferences persists and restores standing UI preferences - sort
// order, merge mode, application appearance, the picture-frame slideshow's
// interval and shuffle order, window size and position, whether the window
// stays a fixed size, and whether favorite previews are cached to disk -
// across launches, via Fyne's app-scoped Preferences store. Unlike
// internal/session (which persists the transient dropped file set),
// everything here is a setting the user deliberately chose and expects to
// stick, so it belongs in fyne.Preferences: unlike the app cache, it's meant
// for this and survives cache clearing.
package preferences

import (
	"sync"
	"time"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/appearance"
	"github.com/frathe/picfetch/internal/mosaic"
)

const (
	keySortMode        = "sortMode"
	keyMergeMode       = "mergeMode"
	keyThemeMode       = "themeMode"
	keySlideIntervalS  = "slideIntervalSeconds"
	keySlideShuffle    = "slideShuffle"
	keyMaxScanFiles    = "maxScanFiles"
	keyMaxWindowWidth  = "maxWindowWidth"
	keyMaxWindowHeight = "maxWindowHeight"
	keyMaxImageCacheMB = "maxImageCacheMB"
	keyMaxThumbCacheMB = "maxThumbCacheMB"
	keyMaxFileSizeMB   = "maxFileSizeMB"
	keyWindowWidth     = "windowWidth"
	keyWindowHeight    = "windowHeight"
	keyWindowPosX      = "windowPosX"
	keyWindowPosY      = "windowPosY"
	keyWindowPosSet    = "windowPosSet"

	keyFavoritePreviewCache = "favoritePreviewCache"

	keyCheckForUpdates    = "checkForUpdates"
	keyLastUpdateCheckDay = "lastUpdateCheckDay"

	keyStaticWindowSize = "staticWindowSize"

	keyDuplicateDistance    = "duplicateDistance"
	keyDuplicateDistanceSet = "duplicateDistanceSet"

	keyMosaicMinimumShortEdge = "mosaicMinimumShortEdge"
	keyMosaicSizeVariation    = "mosaicSizeVariation"
	keyMosaicOverlap          = "mosaicOverlap"
	keyMosaicMaximumRotation  = "mosaicMaximumRotation"
	keyMosaicFrame            = "mosaicFrame"
)

// geometryKeys names the five preference keys one secondary window's
// geometry occupies. The main window's own keys above are spelled out as
// plain constants instead: there is exactly one of it, while every
// widgets.Singleton window that remembers where it was needs the same five,
// and naming them per window is what keeps Save/Load from growing a
// copy of the same five statements per window.
type geometryKeys struct {
	posX, posY, posSet, width, height string
}

var (
	settingsWinKeys = geometryKeys{"settingsWinPosX", "settingsWinPosY", "settingsWinPosSet", "settingsWinWidth", "settingsWinHeight"}
	exifWinKeys     = geometryKeys{"exifWinPosX", "exifWinPosY", "exifWinPosSet", "exifWinWidth", "exifWinHeight"}
	mosaicWinKeys   = geometryKeys{"mosaicWinPosX", "mosaicWinPosY", "mosaicWinPosSet", "mosaicWinWidth", "mosaicWinHeight"}
)

// Valid values for State.SortMode, persisted under keySortMode. Defined as
// strings here - rather than reusing the root package's own sortMode enum,
// which this package can't import without an import cycle - so the on-disk
// representation stays stable and human-readable even if that enum's
// members are ever reordered or renamed.
const (
	SortByName        = "name"
	SortByCaptureDate = "date"
	SortByModTime     = "modified"
	SortBySize        = "size"
	SortByDropOrder   = "drop"
)

// State is the set of standing preferences Save/Load round-trip.
type State struct {
	// SortMode is one of the SortBy* constants above. See Load's comment
	// for how an empty or unrecognized value is handled.
	SortMode      string
	MergeMode     bool
	ThemeMode     appearance.Mode
	SlideInterval time.Duration
	SlideShuffle  bool

	// MaxScanFiles caps how many images a single recursive folder scan
	// gathers - see internal/ui's maxScan field. Zero means "nothing
	// saved yet", the same sentinel WindowSize below uses, since the
	// viewer never accepts a zero cap itself - internal/ui substitutes
	// its own built-in default at that point.
	MaxScanFiles int

	// MaxWindowWidth and MaxWindowHeight cap how large the window is ever
	// allowed to auto-grow to fit a loaded image - see internal/ui's
	// maxWinW/maxWinH fields and resizeToImage (load.go). Zero means "nothing
	// saved yet", the same sentinel MaxScanFiles above uses, since the viewer
	// never accepts a zero cap itself.
	MaxWindowWidth  float32
	MaxWindowHeight float32

	// MaxImageCacheMB and MaxThumbCacheMB are the byte budgets, in megabytes,
	// for the decoded-image and thumbnail caches (internal/imaging's
	// ByteCache). MaxFileSizeMB is the ceiling on a file's encoded size,
	// before any of it is decoded - see imaging.MaxEncodedBytes. All three use
	// the same zero-means-unset sentinel MaxScanFiles above does; the megabyte
	// unit rather than raw bytes is what the settings window shows and what
	// the user typed, so it's what gets stored.
	MaxImageCacheMB int
	MaxThumbCacheMB int
	MaxFileSizeMB   int

	WindowSize fyne.Size // zero Size means "nothing saved yet"

	// WindowPosX and WindowPosY are the on-screen position (see
	// internal/winpos, the only way to read one back at all) a manual move
	// last left the window at; WindowPositionSet distinguishes "saved at
	// (0,0)" from "never saved", which a zero-value check like WindowSize's
	// can't - (0,0) is a perfectly valid on-screen position, unlike a 0x0
	// size.
	WindowPosX, WindowPosY int
	WindowPositionSet      bool

	// SettingsWindow and ExifWindow are where those two secondary windows
	// (internal/ui/settingswin, internal/ui/exifwin) were last left. Grouped
	// into a struct where the main window's own geometry above is flat,
	// because there are two of them and every further widgets.Singleton window
	// that wants to be remembered adds another - see WindowGeometry.
	SettingsWindow WindowGeometry
	ExifWindow     WindowGeometry
	MosaicWindow   WindowGeometry

	// MosaicSettings are the last valid visual controls from the mosaic
	// window. Sources, seed, display target, pixels, and lifecycle state are
	// deliberately transient and never enter standing preferences.
	MosaicSettings mosaic.Settings

	// FavoritePreviewCache is the one preference in this struct whose
	// default is true rather than the zero value: favorites/disk thumbnail
	// caching ships on, so a fresh install with nothing saved yet must still
	// read true. Every other bool field above defaults to false and reads
	// back with plain p.Bool, but that would make a fresh install read this
	// one false too - Load instead uses p.BoolWithFallback(key, true), and
	// Save writes it unconditionally (never gated behind an "only if set"
	// check) so that a user who explicitly turns it off can have that
	// choice persist.
	FavoritePreviewCache bool

	// CheckForUpdates is the settings window's opt-in for looking for a newer
	// release on startup. Defaults to false (plain p.Bool) so a fresh install
	// never phones home until the user turns it on. LastUpdateCheckDay is the
	// local calendar day (YYYY-MM-DD) of the last successful check attempt, or
	// empty when none has run yet. It is persisted only by
	// SaveLastUpdateCheckDay, not by Save, so a quit-time snapshot cannot
	// clobber a day the background check already wrote.
	CheckForUpdates    bool
	LastUpdateCheckDay string

	// StaticWindowSize is the settings window's "Keep a fixed window size"
	// checkbox. When true, the main window no longer auto-resizes to fit
	// loaded images, zoom steps, or rotations; the user-chosen size from
	// WindowSize is kept instead. Defaults to false (plain p.Bool).
	StaticWindowSize bool

	// DuplicateDistance is the Hamming threshold hide-duplicates uses.
	// DuplicateDistanceSet distinguishes a saved 0 (exact thumbnail hash)
	// from "never saved", the same idea WindowPositionSet uses for (0,0).
	DuplicateDistance    int
	DuplicateDistanceSet bool
}

// WindowGeometry is one secondary window's remembered position and size -
// what widgets.Singleton hands over at shutdown and is seeded back with at
// the next launch. Size's zero value means "nothing saved yet" (the window
// then opens at its own built-in size), and PositionSet distinguishes
// "saved at (0,0)" from "never saved", both for the reasons State's
// WindowSize and WindowPositionSet spell out above.
type WindowGeometry struct {
	X, Y        int
	PositionSet bool
	Size        fyne.Size
}

// prefsWriteMu serializes Save and SaveLastUpdateCheckDay. The check
// goroutine writes the day while OnStopped may still be inside Save, and
// Fyne's preference store is not documented as concurrent-safe.
var prefsWriteMu sync.Mutex

// SaveLastUpdateCheckDay persists the local calendar day (YYYY-MM-DD)
// of the last successful update check without a full Save. Used by the
// auto-update glue so a crash after Check still skips a second check today,
// and so OnStopped's Save cannot overwrite a newer day.
func SaveLastUpdateCheckDay(app fyne.App, day string) {
	prefsWriteMu.Lock()
	defer prefsWriteMu.Unlock()
	app.Preferences().SetString(keyLastUpdateCheckDay, day)
}

// Save persists s via app.Preferences(). SlideInterval and WindowSize are
// only written when non-zero, and WindowPosX/WindowPosY only when
// WindowPositionSet, so a run that never touched picture-frame mode or
// never got a window-size/position reading (see windowSizeTracker and
// startWindowPosPolling in internal/ui/windowtrack.go) doesn't clobber a
// good value saved by an earlier run. LastUpdateCheckDay is omitted for
// the same reason: SaveLastUpdateCheckDay is the only writer of that key.
func Save(app fyne.App, s State) {
	prefsWriteMu.Lock()
	defer prefsWriteMu.Unlock()
	p := app.Preferences()
	p.SetString(keySortMode, s.SortMode)
	p.SetBool(keyMergeMode, s.MergeMode)
	p.SetString(keyThemeMode, s.ThemeMode.PrefValue())
	p.SetBool(keySlideShuffle, s.SlideShuffle)
	p.SetBool(keyFavoritePreviewCache, s.FavoritePreviewCache)
	p.SetBool(keyCheckForUpdates, s.CheckForUpdates)
	p.SetBool(keyStaticWindowSize, s.StaticWindowSize)

	if s.SlideInterval > 0 {
		p.SetFloat(keySlideIntervalS, s.SlideInterval.Seconds())
	}
	if s.MaxScanFiles > 0 {
		p.SetInt(keyMaxScanFiles, s.MaxScanFiles)
	}
	if s.MaxWindowWidth > 0 {
		p.SetFloat(keyMaxWindowWidth, float64(s.MaxWindowWidth))
	}
	if s.MaxWindowHeight > 0 {
		p.SetFloat(keyMaxWindowHeight, float64(s.MaxWindowHeight))
	}
	if s.MaxImageCacheMB > 0 {
		p.SetInt(keyMaxImageCacheMB, s.MaxImageCacheMB)
	}
	if s.MaxThumbCacheMB > 0 {
		p.SetInt(keyMaxThumbCacheMB, s.MaxThumbCacheMB)
	}
	if s.MaxFileSizeMB > 0 {
		p.SetInt(keyMaxFileSizeMB, s.MaxFileSizeMB)
	}
	if s.WindowSize.Width > 0 && s.WindowSize.Height > 0 {
		p.SetFloat(keyWindowWidth, float64(s.WindowSize.Width))
		p.SetFloat(keyWindowHeight, float64(s.WindowSize.Height))
	}
	if s.WindowPositionSet {
		p.SetInt(keyWindowPosX, s.WindowPosX)
		p.SetInt(keyWindowPosY, s.WindowPosY)
		p.SetBool(keyWindowPosSet, true)
	}
	if s.DuplicateDistanceSet {
		p.SetInt(keyDuplicateDistance, s.DuplicateDistance)
		p.SetBool(keyDuplicateDistanceSet, true)
	}
	// MinimumShortEdge cannot validly be zero, so it is the one safe marker
	// for an old caller that did not seed mosaic settings at all. Once seeded,
	// the other three numeric values are written unconditionally because zero
	// is an explicit, valid user choice for each.
	if s.MosaicSettings.MinimumShortEdge != 0 {
		settings := s.MosaicSettings.Normalized()
		p.SetFloat(keyMosaicMinimumShortEdge, settings.MinimumShortEdge)
		p.SetFloat(keyMosaicSizeVariation, settings.SizeVariation)
		p.SetFloat(keyMosaicOverlap, settings.Overlap)
		p.SetFloat(keyMosaicMaximumRotation, settings.MaximumRotation)
		p.SetString(keyMosaicFrame, string(settings.Frame))
	}

	saveGeometry(p, settingsWinKeys, s.SettingsWindow)
	saveGeometry(p, exifWinKeys, s.ExifWindow)
	saveGeometry(p, mosaicWinKeys, s.MosaicWindow)
}

// saveGeometry writes one secondary window's geometry under k, position and
// size each guarded by their own "only when set" check - a window the user
// never opened this run reports a zero geometry, which must not clobber
// where they left it last time.
func saveGeometry(p fyne.Preferences, k geometryKeys, g WindowGeometry) {
	if g.PositionSet {
		p.SetInt(k.posX, g.X)
		p.SetInt(k.posY, g.Y)
		p.SetBool(k.posSet, true)
	}
	if g.Size.Width > 0 && g.Size.Height > 0 {
		p.SetFloat(k.width, float64(g.Size.Width))
		p.SetFloat(k.height, float64(g.Size.Height))
	}
}

// loadGeometry reads back what saveGeometry wrote under k.
func loadGeometry(p fyne.Preferences, k geometryKeys) WindowGeometry {
	return WindowGeometry{
		X:           p.Int(k.posX),
		Y:           p.Int(k.posY),
		PositionSet: p.Bool(k.posSet),
		Size: fyne.NewSize(
			float32(p.Float(k.width)),
			float32(p.Float(k.height)),
		),
	}
}

// Load returns the previously saved State. SortMode defaults to
// SortByName, matching the app's shipped default, when nothing has been
// saved yet - and internal/filesort's FromPref falls back to the same
// default for any value it doesn't recognize, so a preferences file
// written by a newer build with a since-removed mode still loads cleanly.
// FavoritePreviewCache defaults to true - see its field comment on State.
// Every other field defaults to its zero value, which callers already treat
// as "use the built-in default" (a zero SlideInterval falls back to
// slideshow.DefaultInterval, a zero WindowSize to internal/ui's
// startW/startH).
func Load(app fyne.App) State {
	p := app.Preferences()
	defaults := mosaic.DefaultSettings()
	mosaicSettings := mosaic.Settings{
		MinimumShortEdge: p.FloatWithFallback(keyMosaicMinimumShortEdge, defaults.MinimumShortEdge),
		SizeVariation:    p.FloatWithFallback(keyMosaicSizeVariation, defaults.SizeVariation),
		Overlap:          p.FloatWithFallback(keyMosaicOverlap, defaults.Overlap),
		MaximumRotation:  p.FloatWithFallback(keyMosaicMaximumRotation, defaults.MaximumRotation),
		Frame:            mosaic.FrameStyleFromPreference(p.StringWithFallback(keyMosaicFrame, string(defaults.Frame))),
	}.Normalized()

	// float64(time.Second) is reported as a redundant conversion but is
	// load-bearing: Preferences.Float returns float64 and time.Second is a
	// typed time.Duration constant, so without it the multiplication does
	// not compile.
	//goland:noinspection GoRedundantConversion
	return State{
		SortMode:        p.StringWithFallback(keySortMode, SortByName),
		MergeMode:       p.Bool(keyMergeMode),
		ThemeMode:       appearance.FromPref(p.String(keyThemeMode)),
		SlideInterval:   time.Duration(p.Float(keySlideIntervalS) * float64(time.Second)),
		SlideShuffle:    p.Bool(keySlideShuffle),
		MaxScanFiles:    p.Int(keyMaxScanFiles),
		MaxWindowWidth:  float32(p.Float(keyMaxWindowWidth)),
		MaxWindowHeight: float32(p.Float(keyMaxWindowHeight)),
		MaxImageCacheMB: p.Int(keyMaxImageCacheMB),
		MaxThumbCacheMB: p.Int(keyMaxThumbCacheMB),
		MaxFileSizeMB:   p.Int(keyMaxFileSizeMB),
		WindowSize: fyne.NewSize(
			float32(p.Float(keyWindowWidth)),
			float32(p.Float(keyWindowHeight)),
		),
		WindowPosX:           p.Int(keyWindowPosX),
		WindowPosY:           p.Int(keyWindowPosY),
		WindowPositionSet:    p.Bool(keyWindowPosSet),
		SettingsWindow:       loadGeometry(p, settingsWinKeys),
		ExifWindow:           loadGeometry(p, exifWinKeys),
		MosaicWindow:         loadGeometry(p, mosaicWinKeys),
		MosaicSettings:       mosaicSettings,
		FavoritePreviewCache: p.BoolWithFallback(keyFavoritePreviewCache, true),
		CheckForUpdates:      p.Bool(keyCheckForUpdates),
		LastUpdateCheckDay:   p.String(keyLastUpdateCheckDay),
		StaticWindowSize:     p.Bool(keyStaticWindowSize),
		DuplicateDistance:    p.Int(keyDuplicateDistance),
		DuplicateDistanceSet: p.Bool(keyDuplicateDistanceSet),
	}
}
