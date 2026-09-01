package ui

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/dupes"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/preferences"
	"github.com/frathe/picfetch/internal/ui/assets"
	compareui "github.com/frathe/picfetch/internal/ui/compare"
	"github.com/frathe/picfetch/internal/ui/copyselection"
	"github.com/frathe/picfetch/internal/ui/deletion"
	"github.com/frathe/picfetch/internal/ui/exifwin"
	"github.com/frathe/picfetch/internal/ui/favorites"
	"github.com/frathe/picfetch/internal/ui/grid"
	"github.com/frathe/picfetch/internal/ui/help"
	"github.com/frathe/picfetch/internal/ui/settingswin"
	"github.com/frathe/picfetch/internal/ui/slideshow"
	"github.com/frathe/picfetch/internal/ui/widgets"
	"github.com/frathe/picfetch/internal/ui/zoom"
	"github.com/frathe/picfetch/internal/wingesture"
)

var _ settingswin.Host = (*viewer)(nil)

// registerFeatures constructs every feature in dependency order. It only
// assigns the viewer's feature fields; build.go still decides how their
// widgets compose, and menu.go still decides how their menus compose.
func registerFeatures(view *viewer, application fyne.App, window fyne.Window, prefs preferences.State) {
	view.help = help.New(application, appTitle, assets.ComparingWebP)

	// The window-drag easter egg (gesture.go): the detector is fed by the
	// position poller, and a recognised spiral goes to the same Help that
	// owns the manual's secret-phrase door, so both raise one window.
	view.spiralDrag = wingesture.New(wingesture.Config{})
	view.spiralGesture = view.help.OpenSpiral
	view.exif = exifwin.New(application, view)

	// Resolve these callbacks against the viewer at call time so tests can
	// replace keyModifiers after construction.
	view.zoom = zoom.New(
		view.img,
		func() {
			view.syncWindowToZoom()
			view.updateInfoOverlay()
		},
		func() fyne.KeyModifier { return view.keyModifiers() },
		view.requestVectorRender,
	)
	view.regionCopy = copyselection.New(copyselection.Callbacks{
		Copy:   view.copyRegionSelection,
		Ended:  view.finishRegionCopy,
		Scroll: view.zoom.HandleScroll,
	})
	view.zoom.SetOnGeometryChanged(func(geometry zoom.Geometry) {
		// Delivered synchronously, possibly inside imageRenderer.Layout —
		// see zoom.SetOnGeometryChanged. Reading mode state here is safe;
		// mutating widgets is not, so the update still crosses
		// regionCopyDo. The Active read is hoisted so the common inactive
		// frame pays no closure or queue hop, and re-checked inside because
		// the mode can end between queueing and running. No ForceRepaint:
		// ViewChanged's own chrome sync refreshes the selection overlay,
		// and zoom's apply already painted the image.
		if !view.regionCopy.State().Active {
			return
		}
		do := view.regionCopyDo
		if do == nil {
			do = fyne.Do
		}
		do(func() {
			if !view.regionCopy.State().Active {
				return
			}
			view.regionCopy.ViewChanged(copyselection.View{
				Position: geometry.Position,
				Size:     geometry.Size,
			})
		})
	})

	// The duplicate model is the viewer's, not the grid's: hide-duplicates
	// survives the overlay closing and plain arrow-key navigation asks it
	// which files are visible with the grid down. It must exist before
	// grid.New, which is handed the same model to read and to feed from
	// its hashing pass, and before the saved threshold is pushed into it
	// below.
	view.dupes = dupes.New(dupeFileSet{v: view})

	// Registration order is fire order (dupes.Model.OnChange): this is the
	// only observer registered today, and it must stay behind whatever
	// re-filters the grid, because jumpIfHiddenExtra reads the group
	// snapshot the filter pass installs. The grid keeps that re-filter as
	// a direct call at each of its own transitions - see
	// internal/ui/grid/dupes.go - rather than as an observer, because one
	// of them needs a keepHost argument no parameterless observer can
	// carry.
	view.dupes.OnChange(view.jumpIfHiddenExtra)

	// The thumbnail-cache setter reaches into the grid, so the grid must be
	// registered before saved cache limits are applied.
	view.grid = grid.New(view, window, view.dupes)
	view.compare = compareui.New(
		func(ctx context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
			return view.compareLoad(ctx, uri)
		},
		compareui.Callbacks{
			Repaint:      view.ForceRepaint,
			Opened:       view.syncMenus,
			Closed:       view.comparisonClosed,
			Failed:       view.compareFailed,
			OrderChanged: view.compareOrderChanged,
			Modifiers:    func() fyne.KeyModifier { return view.keyModifiers() },
		},
	)
	view.SetMaxThumbCacheMB(prefs.MaxThumbCacheMB)
	view.SetMaxFileSizeMB(prefs.MaxFileSizeMB)

	// Deliberately not SetDuplicateDistance: that setter marks the value
	// as saved, and restoring a preference that was never set must not
	// turn dupeDistSet on behind the user's back.
	view.settings.dupeDist = prefs.DuplicateDistance
	view.settings.dupeDistSet = prefs.DuplicateDistanceSet
	view.pushDuplicateDistance(view.DuplicateDistance())

	view.SetFavoritePreviewCache(prefs.FavoritePreviewCache)
	// Restore update prefs without SetCheckForUpdates: that setter starts a
	// network check. Day must be in place before startViewerRuntime's
	// maybeStartUpdateCheck so Due sees the saved calendar day.
	view.SetLastUpdateCheckDay(prefs.LastUpdateCheckDay)
	view.settings.checkForUpdates = prefs.CheckForUpdates
	view.settings.staticWindowSize = prefs.StaticWindowSize

	view.deletion = deletion.New(view)

	// The export-format prompt (promptExport, export.go) is a bare
	// widgets.ChoiceCard, unlike deletion's own wrapping Confirmer: each
	// choice's OnChosen already re-checks canExport/CurrentFile through
	// exportAs at the moment it runs, so there's no per-request state (the
	// way deletion's targets are) for a wrapper type to hold.
	//
	// Filled by index rather than listed positionally so that export.go's
	// pngChoice/jpegChoice constants are what actually place the buttons -
	// the same constants the prompt is selected and asserted through
	// elsewhere. Written as two positional literals, the constants would
	// only describe the order, and a swap of the two lines would leave the
	// PNG button quietly writing JPEGs.
	choices := make([]widgets.Choice, 2)
	choices[pngChoice] = widgets.Choice{Label: lang.L("PNG"), OnChosen: func() { view.exportAs(exportPNGExt) }}
	choices[jpegChoice] = widgets.Choice{Label: lang.L("JPEG"), OnChosen: func() { view.exportAs(exportJPEGExt) }}
	view.exportPrompt = widgets.NewChoiceCard(view.ForceRepaint, choices...)

	// Run starts the position poller only after buildViewer returns. Register
	// the slideshow first because the poller's skip callback reads Active.
	view.slides = slideshow.New(view, window, &view.winPos)
	if prefs.SlideInterval > 0 {
		view.slides.SetInterval(prefs.SlideInterval)
	}
	view.slides.SetShuffle(prefs.SlideShuffle)

	view.settingsWin = settingswin.New(application, view)
	view.favorites = favorites.New(view, window)
}
