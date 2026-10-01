// Startup assembly loads persisted inputs once, constructs the complete
// viewer, and only then restores geometry. Runtime side effects remain in
// run.go and start after this sequence returns.

package ui

import (
	"path/filepath"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/explorerpresets"
	"github.com/frathe/picfetch/internal/favstore"
	"github.com/frathe/picfetch/internal/filescan"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/launch"
	"github.com/frathe/picfetch/internal/preferences"
	"github.com/frathe/picfetch/internal/screenshots"
	"github.com/frathe/picfetch/internal/session"
	"github.com/frathe/picfetch/internal/ui/autoupdate"
)

// startupState combines captured launch inputs and persisted state for
// buildViewer and geometry restoration.
type startupState struct {
	policy       launch.Policy
	storage      launch.Storage
	savedSession []fyne.URI
	prefs        preferences.State
}

// loadStartupState reads persistence and fills only preference defaults that
// have no distinct zero-value meaning.
func loadStartupState(application fyne.App) startupState {
	return startupState{
		savedSession: session.Load(application),
		prefs:        normalizePreferenceDefaults(preferences.Load(application)),
	}
}

// buildStartupViewer is the shared load, construct, then restore entry point.
// It leaves noPollerStop installed for startViewerRuntime to replace.
func buildStartupViewer(application fyne.App, policy launch.Policy, resolveOrdinary func(fyne.App) (launch.Storage, error)) (*viewer, fyne.Window, error) {
	return buildStartupViewerForLaunch(application, policy, resolveOrdinary, launch.Options{})
}

func buildStartupViewerForLaunch(application fyne.App, policy launch.Policy, resolveOrdinary func(fyne.App) (launch.Storage, error), opts launch.Options) (*viewer, fyne.Window, error) {
	if !policy.Valid() {
		return nil, nil, launch.ErrInvalidPolicy
	}
	var ordinary launch.Storage
	if policy.Purpose() == launch.Ordinary {
		var err error
		ordinary, err = resolveOrdinary(application)
		if err != nil {
			return nil, nil, err
		}
	}
	selected, err := policy.ResolveStorage(ordinary)
	if err != nil {
		return nil, nil, err
	}
	startup := loadStartupState(application)
	startup.policy = policy
	startup.storage = selected
	if opts.FixedSize != nil {
		application = screenshots.App(application, opts.FixedSize.Width, opts.FixedSize.Height)
	}
	view, window := buildViewer(application, startup)
	if opts.FixedSize != nil {
		view.launchOverride.geometry = &startup.prefs
	}
	restoreStartupGeometry(view, window, startup)
	return view, window, nil
}

// ordinaryLaunchStorage resolves the existing defaults only for ordinary launches.
// The supplied app already owns the captured application identity.
func ordinaryLaunchStorage(application fyne.App) (launch.Storage, error) {
	favorites, err := favstore.DefaultDir()
	if err != nil {
		return launch.Storage{}, err
	}
	roots := launch.Storage{FavoritesDir: favorites, PresetsDir: explorerpresets.DefaultDir(), UpdatesDir: autoupdate.DefaultDir()}
	if root := application.Cache().RootURI(); root != nil && root.Scheme() == "file" {
		roots.AnalysisDir = filepath.Join(root.Path(), "image-analysis")
	}
	return roots, nil
}

// normalizePreferenceDefaults fills only caps. The other zero values remain
// meaningful: an unset slideshow interval is chosen on first use, geometry
// flags distinguish unsaved positions, and zero secondary geometry uses each
// window's built-in placement and size.
func normalizePreferenceDefaults(prefs preferences.State) preferences.State {
	if prefs.FavoritePreviewLimit <= 0 {
		prefs.FavoritePreviewLimit = preferences.DefaultFavoritePreviewLimit
	}
	if prefs.MaxScanFiles <= 0 {
		prefs.MaxScanFiles = filescan.DefaultMax
	}
	if prefs.MaxWindowWidth <= 0 {
		prefs.MaxWindowWidth = defaultMaxWindowWidth
	}
	if prefs.MaxWindowHeight <= 0 {
		prefs.MaxWindowHeight = defaultMaxWindowHeight
	}
	if prefs.MaxImageCacheMB <= 0 {
		prefs.MaxImageCacheMB = defaultMaxImageCacheMB
	}
	if prefs.MaxThumbCacheMB <= 0 {
		prefs.MaxThumbCacheMB = defaultMaxThumbCacheMB
	}
	if prefs.MaxFileSizeMB <= 0 {
		prefs.MaxFileSizeMB = defaultMaxFileSizeMB
	}
	if !prefs.DuplicateDistanceSet {
		prefs.DuplicateDistance = imaging.DuplicateMaxDistance
	}

	return prefs
}

// restoreStartupGeometry runs after feature construction so the settings and
// EXIF windows exist before their remembered geometry is applied.
func restoreStartupGeometry(view *viewer, window fyne.Window, startup startupState) {
	prefs := startup.prefs

	view.settingsWin.RestoreGeometry(widgetGeometry(prefs.SettingsWindow))
	view.exif.RestoreGeometry(widgetGeometry(prefs.ExifWindow))
	view.mosaicWin.RestoreGeometry(widgetGeometry(prefs.MosaicWindow))

	initialSize := fyne.NewSize(startW, startH)
	if prefs.WindowSize.Width > 0 && prefs.WindowSize.Height > 0 {
		initialSize = prefs.WindowSize
	}
	window.Resize(initialSize)

	if prefs.WindowPositionSet {
		view.winPos.Store(prefs.WindowPosX, prefs.WindowPosY)
		view.winPos.Restore(window)
	}
}
