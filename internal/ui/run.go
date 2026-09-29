// Package ui is the application itself: the viewer, its widgets, and every
// feature wired onto it. The module root's package main is only the entry
// point - it builds the fyne.App, loads translations, and calls Run below.
package ui

import (
	"os"
	"os/signal"
	"syscall"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/launch"
	"github.com/frathe/picfetch/internal/openwith"
	"github.com/frathe/picfetch/internal/preferences"
	"github.com/frathe/picfetch/internal/session"
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
// the caller); empty for a plain launch. opts carries that launch's flags,
// already parsed and validated by internal/launch; the zero value is a
// plain launch that overrides nothing. notices and privacy are this build's
// embedded documents, supplied by main alongside its other resources.
// prepared lends the captured policy and trial recorders; its caller retains
// ownership and closes them after Run joins their producers and returns.
func Run(application fyne.App, initial []fyne.URI, opts launch.Options, prepared *launch.Prepared, notices, privacy string) error {
	policy := prepared.Policy()
	if !policy.Valid() {
		return launch.ErrInvalidPolicy
	}
	trial := prepared.ExplorerTrial()
	view, window, err := buildStartupViewer(application, policy, ordinaryLaunchStorage)
	if err != nil {
		return err
	}
	view.borrowLocationTrial(prepared.LocationMapTrial())
	view.help.SetLicenses(notices)
	view.help.SetPrivacyPolicy(privacy)

	options := view.explorer.Options()
	options.Trial = trial
	view.explorer.Configure(options)

	// After construction, so the flags override what saved preferences just
	// seeded, and before Show, so the window comes up already in the state
	// the command line asked for.
	view.applyLaunchOptions(opts)

	startViewerRuntime(view, window)
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
	view.syncNativeMenuBar()
	registerStartup(application, view, initial)
	stopSignals := func() {}
	if trial != nil || view.locationTrial != nil {
		notices := make(chan os.Signal, 1)
		signal.Notify(notices, os.Interrupt, syscall.SIGTERM)
		stop, done := make(chan struct{}), make(chan struct{})
		go func() {
			defer close(done)
			select {
			case <-notices:
				fyne.Do(func() {
					if !view.stopping {
						view.explorer.RecordAction("stop-requested", 0)
						application.Quit()
					}
				})
			case <-stop:
			}
		}()
		stopSignals = func() { signal.Stop(notices); close(stop); <-done }
	}
	application.Run()
	stopSignals()
	view.waitForShutdown()
	return nil
}

func registerStartup(application fyne.App, view *viewer, initial []fyne.URI) {
	application.Lifecycle().SetOnStarted(func() {
		view.syncNativeMenuBar()
		// The sweep reads failure evidence before any reporter can consume it.
		// A reporter running first could erase a failed-restore warning and let
		// cleanup delete the only working binary.
		if view.launchPolicy.Updates().Allowed() {
			failure := view.sweepUpdateBackup()
			view.maybeShowWhatsNew()
			view.maybeShowUpdateFailure(failure)
		}

		// Install before opening: installing also flushes cold-start Apple
		// Events, combining them with argv in one scan. See openwith.go.
		view.pendingInitial = initial
		view.installOpenWithHandler()
		view.openInitialFiles()
	})
}

func (v *viewer) waitForShutdown() {
	// OnStopped already deactivated the bar. This joins a callback that
	// started too late to submit a repaint and has not exited yet.
	if v.frameChrome != nil {
		v.frameChrome.Wait()
	}
	v.favorites.Wait()
	v.waitHEIC()
	v.help.Wait()
	v.locationMap.Wait()
	// Preview cancellation cannot interrupt a source already blocked in native
	// I/O. Close retires its UI delivery; only the test harness joins those reads
	// through Spiral.Settle after releasing any held source.
	// Shutdown has canceled admission; join the native process after the UI loop
	// retires so the application cannot leave an analysis worker behind.
	v.explorer.Wait()
	v.visualsearch.Wait()
	v.searchView.overlayWorkers.Wait()
	v.analysisCache.Wait()
}

// Runtime side effects start only after feature construction and geometry
// restoration, so polling cannot observe a nil slideshow or replace a saved
// position before it has been applied.
func startViewerRuntime(view *viewer, window fyne.Window) {
	view.startHEICCheck(false)
	view.favorites.SetDir(view.favorites.Dir())
	view.stopWinPosPoll = startWindowPosPolling(view, window)
	view.maybeStartUpdateCheck()
}

// registerShutdown retires UI updates, cancels work, and saves state before
// Fyne performs its guaranteed final preferences flush.
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
		view.stopping = true
		view.favorites.Stop()
		view.stopHEIC()
		view.help.Stop()
		view.closeLocationMap()
		view.locationMap.Stop()
		view.spiral.Close()
		view.closeExplorer()
		view.closeVisualSearch()
		view.visualsearch.Stop()
		view.analysisCache.Stop()
		view.explorer.Stop()
		view.closeFileWork()
		view.closeClipboardWork()
		view.closeOpenChooser()
		view.closeFavoritePreviews()
		view.grid.Stop()
		view.exif.Stop()
		view.deletion.Close()
		// Cancel main and secondary position sampling. Stop discards queued
		// reads without waiting on the retired UI;
		// completion is observed separately by the test harness off UI.
		view.stopWinPosPoll()
		view.settingsWin.StopTracking()
		view.exif.StopTracking()
		view.mosaicWin.StopTracking()
		// Close stops the slideshow worker without telling the menu
		// observer, so an armed dwell or slide would otherwise call
		// fyne.Do after the event loop has stopped.
		if view.frameChrome != nil {
			view.frameChrome.Deactivate()
		}
		view.slides.Close()
		view.scanOp.lifecycle.Invalidate()
		view.sortOp.lifecycle.Invalidate()
		view.display.Stop()
		view.regionCopyLifecycle.Invalidate()
		view.updateOp.Invalidate()
		view.compare.Close()
		view.mosaicWin.Close()

		// Same reasoning as the invalidations above, for the one piece of
		// state that outlives the viewer: openwith's queue is
		// process-global, so a delivery landing mid-shutdown would
		// otherwise reach a viewer whose window is already going away.
		// Native selections still buffered own implicit scope and must retire.
		openwith.Stop()

		session.Save(application, view.state.Observe().Capture(collectionSourceOrder))
		preferences.Save(application, view.currentPreferences())
		if view.launchPolicy.Updates().Allowed() {
			view.updater.ApplyStagedUpdate()
		}
	})
}

// currentPreferences is everything worth remembering about this run, ready
// for preferences.Save. Split out of the SetOnStopped callback above purely
// so it can be read back in a test before Fyne performs its final flush.
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

	state := preferences.State{
		SortMode:                v.state.SortMode().PrefValue(),
		MergeMode:               v.state.MergeMode(),
		ThemeMode:               v.settings.themeMode,
		SlideInterval:           v.slides.Interval(),
		SlideShuffle:            v.slides.Shuffle(),
		MaxScanFiles:            v.settings.maxScan,
		MaxWindowWidth:          v.settings.maxWinW,
		MaxWindowHeight:         v.settings.maxWinH,
		MaxImageCacheMB:         v.settings.imgCacheMB,
		MaxThumbCacheMB:         v.settings.thumbCacheMB,
		MaxFileSizeMB:           v.settings.maxFileMB,
		FavoritePreviewCache:    v.settings.favPreviewCache,
		FavoritePreviewLimit:    v.settings.favPreviewLimit,
		SimilarityFavoriteCache: v.explorer.Settings().CacheFavorites,
		SimilarityLooseCache:    v.settings.looseAnalysisCache,
		AnalysisCacheLimitMiB:   v.settings.analysisCacheMiB,
		SimilarityAutoUpdate:    v.explorer.Settings().Automatic,
		SimilarityAutoFit:       v.explorer.Settings().AutoFit,
		SimilarityMemoryLimitMB: v.explorer.Settings().Limits.MemoryMB,
		SimilarityItemLimit:     v.explorer.Settings().Limits.Items,
		SimilarityIntroSeen:     v.explorer.Settings().IntroSeen,
		CheckForUpdates:         v.settings.checkForUpdates,
		LastUpdateCheckDay:      v.LastUpdateCheckDay(),
		StaticWindowSize:        v.settings.staticWindowSize,
		DuplicateDistance:       v.DuplicateDistance(),
		DuplicateDistanceSet:    v.settings.dupeDistSet,
		WindowSize:              v.windowSize,
		WindowPosX:              posX,
		WindowPosY:              posY,
		WindowPositionSet:       posSet,
		SettingsWindow:          prefGeometry(v.settingsWin.Geometry()),
		ExifWindow:              prefGeometry(v.exif.Geometry()),
		MosaicWindow:            prefGeometry(v.mosaicWin.Geometry()),
		MosaicSettings:          v.mosaicWin.Settings(),
	}

	// Last, so a launch flag never outlives the launch that carried it: the
	// setters above wrote the flag's value into the live viewer, and this
	// puts the value it replaced back for saving. See launchoptions.go.
	v.launchOverride.restore(&state)

	return state
}
