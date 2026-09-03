// Package ui is the application itself: the viewer, its widgets, and every
// feature wired onto it. The module root's package main is only the entry
// point - it builds the fyne.App, loads translations, and calls Run below.
package ui

import (
	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/favstore"
	"github.com/frathe/picfetch/internal/openwith"
	"github.com/frathe/picfetch/internal/preferences"
	"github.com/frathe/picfetch/internal/session"
	"github.com/frathe/picfetch/internal/ui/autoupdate"
)

const (
	appTitle = "PicFetch"

	// size of the empty drop zone
	startW = 520.0
	startH = 340.0

	// height of the "loading next image" bar pinned to the top edge
	loadingBarHeight = 5.0
)

// Run builds the viewer's window and hands control to Fyne's event loop,
// which it does not return from until the window closes. initial is the
// file set to open on startup (command-line arguments, resolved to URIs by
// the caller); empty for a plain launch.
func Run(application fyne.App, initial []fyne.URI) {
	view, window := buildStartupViewer(application)
	startViewerRuntime(view, window, favstore.DefaultDir())
	registerShutdown(application, view)

	// Show() (not ShowAndRun) so we can fold Darwin's Window menus after
	// setupNativeMenu has run. SetMainMenu queues that rebuild until the
	// GLFW view exists; fyne.Do from buildViewer runs inline before Run
	// and would merge while the Fyne Window title is still absent. Show
	// creates the view and drains the queue — two adjacent Window titles
	// until the fold below. OnStarted repeats it (idempotent) and still
	// defers CLI drops until the event loop is running, as handleDrop
	// touches widgets directly.
	window.Show()
	syncNativeMenuBar(view.win.MainMenu())
	application.Lifecycle().SetOnStarted(func() {
		syncNativeMenuBar(view.win.MainMenu())
		// The failure report takes its record from the sweep rather than
		// re-reading the cache, and that data dependency is what keeps the
		// report ordered after the sweep: reporting clears the record, and a
		// cleared record reads as a clean install, so a reporter that ran
		// first would let the sweep take the last working binary.
		if !view.storeManaged {
			failure := view.sweepUpdateBackup()
			view.maybeShowWhatsNew()
			view.maybeShowUpdateFailure(failure)
		}

		// Install before opening, not after: a delivery arriving in the
		// gap between the two would have nobody to take it. Installing
		// also flushes whatever the cold-start Apple Event queued while
		// Fyne was still building this window, and that flush shares
		// pendingInitial with openInitialFiles - so a launch carrying both
		// command-line paths and an "Open With" ends in one scan, not two.
		// See internal/ui/openwith.go.
		view.pendingInitial = initial
		view.installOpenWithHandler()
		view.openInitialFiles()
	})
	application.Run()
}

// Runtime side effects start only after feature construction and geometry
// restoration, so polling cannot observe a nil slideshow or replace a saved
// position before it has been applied.
func startViewerRuntime(view *viewer, window fyne.Window, favoritesDir string) {
	view.favorites.SetDir(favoritesDir)
	view.stopWinPosPoll = startWindowPosPolling(view, window)
	if view.updater.Dir() == "" {
		view.updater.SetDir(autoupdate.DefaultDir())
	}
	view.maybeStartUpdateCheck()
}

// registerShutdown installs the save while the Fyne event loop is still
// available to synchronously flush preferences.
func registerShutdown(application fyne.App, view *viewer) {
	// Wired via SetOnStopped, not run after ShowAndRun returns: Fyne's own
	// app.Preferences() schedules its on-disk flush through a debounced
	// change listener (app.newPreferences in fyne itself) that, once
	// tripped, defers the actual write to a goroutine gated on fyne.DoAndWait
	// - which needs the driver's event loop still alive to ever run. Calling
	// preferences.Save after ShowAndRun returns loses that race every time:
	// the loop has already wound down by then, so the debounced write for
	// everything but the very first preference key never lands, and the
	// process exits before it could anyway. Fyne calls its own equivalent
	// save (SetOnStoppedHookExecuted, app.go) immediately *after* whatever
	// SetOnStopped callback is registered here finishes (see
	// (*Lifecycle).OnStopped) - and Run() blocks on WaitForEvents until both
	// have run - so writing the preferences from here piggybacks on that
	// same guaranteed-synchronous flush instead of racing it.
	application.Lifecycle().SetOnStopped(func() {
		// Stopped first, all three of them: each poller hops through
		// fyne.DoAndWait on every tick, and the event loop they need is
		// about to wind down. The secondary windows only have one running
		// if they're still open right now, and StopTracking says so itself.
		view.stopWinPosPoll()
		view.settingsWin.StopTracking()
		view.exif.StopTracking()
		view.scanOp.lifecycle.invalidate()
		view.loadLifecycle.invalidate()
		view.sortOp.lifecycle.invalidate()
		view.vector.lifecycle.invalidate()
		view.regionCopyLifecycle.invalidate()
		view.updateOp.invalidate()
		view.compare.Close()

		// Same reasoning as the invalidations above, for the one piece of
		// state that outlives the viewer: openwith's queue is
		// process-global, so a delivery landing mid-shutdown would
		// otherwise reach a viewer whose window is already going away.
		// Anything still buffered stays buffered - SetHandler(nil) does
		// not discard it - which costs nothing, as the process is exiting.
		openwith.SetHandler(nil)

		session.Save(application, view.state.unsortedFiles)
		preferences.Save(application, view.currentPreferences())
		if !view.storeManaged {
			view.updater.ApplyStagedUpdate()
		}
	})
}

// currentPreferences is everything worth remembering about this run, ready
// for preferences.Save. Split out of the SetOnStopped callback above purely
// so it can be read back in a test - the callback itself only ever runs
// inside a live Fyne event loop.
//
// view.windowSize is kept current by windowSizeTracker (windowtrack.go) on
// every layout, so it already reflects the window's last size by the time
// the app stops. view.winPos is kept current the same way by
// startWindowPosPolling's background poller, plus the slideshow's own
// capture/restore around full-screen (internal/ui/slideshow). The two
// secondary windows track their own geometry the same way and hand it over
// whole (see widgets.Singleton), including for a window the user closed
// again long ago.
func (v *viewer) currentPreferences() preferences.State {
	posX, posY, posSet := v.winPos.Get()

	return preferences.State{
		SortMode:             v.state.SortMode().PrefValue(),
		MergeMode:            v.state.MergeMode(),
		ThemeMode:            v.settings.themeMode,
		SlideInterval:        v.slides.Interval(),
		SlideShuffle:         v.slides.Shuffle(),
		MaxScanFiles:         v.settings.maxScan,
		MaxWindowWidth:       v.settings.maxWinW,
		MaxWindowHeight:      v.settings.maxWinH,
		MaxImageCacheMB:      v.settings.imgCacheMB,
		MaxThumbCacheMB:      v.settings.thumbCacheMB,
		MaxFileSizeMB:        v.settings.maxFileMB,
		FavoritePreviewCache: v.settings.favPreviewCache,
		CheckForUpdates:      v.settings.checkForUpdates,
		LastUpdateCheckDay:   v.LastUpdateCheckDay(),
		StaticWindowSize:     v.settings.staticWindowSize,
		DuplicateDistance:    v.DuplicateDistance(),
		DuplicateDistanceSet: v.settings.dupeDistSet,
		WindowSize:           v.windowSize,
		WindowPosX:           posX,
		WindowPosY:           posY,
		WindowPositionSet:    posSet,
		SettingsWindow:       prefGeometry(v.settingsWin.Geometry()),
		ExifWindow:           prefGeometry(v.exif.Geometry()),
	}
}
