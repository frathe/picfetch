package ui

import (
	"context"
	"os"
	"slices"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/completion"
	"github.com/frathe/picfetch/internal/explorerpresets"
	"github.com/frathe/picfetch/internal/openwith"
	"github.com/frathe/picfetch/internal/similarity"
	"github.com/frathe/picfetch/internal/ui/analysiscache"
	"github.com/frathe/picfetch/internal/ui/autoupdate"
	"github.com/frathe/picfetch/internal/ui/display"
	explorerui "github.com/frathe/picfetch/internal/ui/explorer"
	searchui "github.com/frathe/picfetch/internal/ui/visualsearch"
	"github.com/frathe/picfetch/internal/uitest"
)

// This file is the shared harness every other test file in this package
// builds on: newTestUI/newTestViewer construct a viewer through the same
// startup path production uses, and the wait/settle/assert helpers below let
// tests synchronize with its background work instead of guessing at timing.
// newTestViewer alone is used by 17 of the package's other test files,
// dropAndWait by 16, waitUntilLoaded by 15 - a new helper shared across more
// than one feature belongs here, not bolted onto whichever feature file
// happens to need it first.
//
// Display's load/GIF/SVG delivery uses its instance UIQueue. Settle joins finite
// workers and drains their callbacks; LoadDone then observes the complete retry
// chain and root handoff. Frame application and animation shutdown have separate
// observations. Scan still uses its completion.Signal. Never sleep to guess
// completion or infer it from a widget's visibility.
//
// AGENTS.md states the rule this file exists to enforce: every goroutine
// needs cancellation/staleness handling plus an observable stop/done signal,
// and any new background work must be added to newTestUI's drain cleanup,
// below.
var testApp fyne.App

// testTextEntry lets surface tests use Entry behavior without depending on
// whether a feature extends the widget to handle additional keys.
type testTextEntry interface {
	fyne.Focusable
	fyne.Shortcutable
	SetText(string)
	SelectedText() string
}

func TestMain(m *testing.M) {
	if similarity.WorkerMain() {
		return
	}
	testApp = test.NewApp()

	// No global tweaks needed here anymore: the toast auto-hide duration,
	// the folder-scan cap, and the key-modifier reader - all package vars
	// once, all mutated from tests - are per-viewer state now, overridden
	// where each viewer is built (newTestUI, or the individual test).
	os.Exit(m.Run())
}

// --- test viewer construction ----------------------------------------------

// newTestUI builds a fresh app and window through the same startup load,
// assembly, and geometry restoration Run uses, without starting runtime
// polling or touching production favorites storage.
//
// It tracks whether the window has already been closed (e.g. by Escape,
// mid-test) so cleanup never closes it a second time: Fyne's test driver's
// removeWindow (test/driver.go) unlocks its windows-list mutex without a
// defer, so a second Close panics partway through and leaves that mutex
// permanently locked - wedging every later test in the package that touches
// a window, not just this one.
func newTestUI(t *testing.T) (*viewer, fyne.Window, func() bool) {
	t.Helper()
	return newTestUIWithStartupImages(t, nil)
}

func newTestUIWithImages(t *testing.T, images imageServices) (*viewer, fyne.Window, func() bool) {
	t.Helper()
	return newTestUIWithStartupImages(t, &images)
}

func newTestUIWithStartupImages(t *testing.T, override *imageServices) (v *viewer, win fyne.Window, closed func() bool) {
	t.Helper()

	// Reassert the shared app as the current one before building: the
	// persistence tests construct their own app, and Fyne makes whichever
	// was built last the process-wide current app - which widget internals
	// (theme, driver) read directly. Without this, a test running after one
	// of those would build its widgets against a different app than the one
	// buildViewer was handed. Cheap, unlike test.NewApp - see testApp.
	fyne.SetCurrentApp(testApp)

	// Update glue persists lastUpdateCheckDay, whatsnew.json and
	// updatefailure.json onto the shared test app; clear all three so one
	// test's check cannot make Due false, leave notes, or leave a recorded
	// apply failure for the next.
	testApp.Preferences().SetString("lastUpdateCheckDay", "")
	testApp.Preferences().SetBool("checkForUpdates", false)
	for _, key := range []string{autoupdate.WhatsNewCacheKey, autoupdate.ApplyFailureCacheKey} {
		if testApp.Cache().Exists(key) {
			_ = testApp.Cache().Remove(key)
		}
	}

	startup := loadStartupState(testApp)
	if override != nil {
		startup.images.Stop()
		startup.images.Wait()
		startup.images = *override
	}
	images := startup.images
	v, win = buildConfiguredViewer(testApp, startup)
	v.display.SetUIQueue(&uitest.UIQueue{})
	v.grid.SetUIQueue(&uitest.UIQueue{})
	v.visualsearch.Configure(searchui.Options{Provider: (similarity.Client{HEIC: images.owner}).Search, Queue: &uitest.UIQueue{}})
	v.searchView.overlayUI = &uitest.UIQueue{}
	v.analysisDir = t.TempDir()
	v.analysisCache.Configure(analysiscache.Options{Roots: v.analysisRoots(), Queue: &uitest.UIQueue{}, ConfirmClear: v.settingsWin.ConfirmClearAnalysis, Changed: v.syncMenus})
	v.spiral.SetUIQueue(&uitest.UIQueue{})
	// Ordinary Explorer fixtures begin after first-use setup; setup cases reset these.
	configureExplorer(v, func(options *explorerui.Options) {
		options.Queue = &uitest.UIQueue{}
		options.Settings.IntroSeen, options.AssetsReady = true, true
		options.Presets = &explorerpresets.Store{Dir: t.TempDir()}
	})
	v.compare.SetUIQueue(&uitest.UIQueue{})
	v.mosaicWin.SetUIQueue(&uitest.UIQueue{})
	v.deletion.SetUIQueue(&uitest.UIQueue{})
	v.slides.SetUIQueue(&uitest.UIQueue{})
	v.exif.SetUIQueue(&uitest.UIQueue{})
	v.fileWork.ui = &uitest.UIQueue{}
	v.chooserUI = &uitest.UIQueue{}
	v.clipboardWork.ui = &uitest.UIQueue{}

	// The auto-hide timer must never fire on its own mid-suite: its inline
	// fyne.Do (under the test driver) would write widgets concurrently with
	// whatever the test goroutine is doing by then. Tests drive the hide
	// synchronously via settleToast instead, so the production duration is
	// irrelevant here - an hour just guarantees a leaked timer sleeps
	// harmlessly until the process exits.
	v.toast.duration = time.Hour

	// Vector re-renders fire from every effective-scale change (a key, a
	// scroll, or a window resize), and the production debounce would leave
	// work pending until the timer fires. Use zero delay and explicit settlement
	// so assertions inspect applied results.
	v.display.SetVectorOptions(display.VectorOptions{})

	// Zoom reports presentation geometry from inside renderer Layout.
	// Production defers the Copy Selection update through fyne.Do; the test
	// driver runs that inline anyway, so make the per-viewer seam explicit
	// and deterministic here.
	v.regionCopyDo = func(f func()) { f() }
	v.regionCopyDoAndWait = func(f func()) { f() }

	// setAsWallpaper writes a PNG it then hands to the OS, and unlike every
	// other file this suite produces that one is meant to outlive the
	// process - so it is redirected out of the user's real cache directory
	// here, the same way the toast's duration is neutralized above.
	// wallpaper.Set itself is stubbed per-test (uitest.StubWallpaperSet), so
	// nothing here ever reaches the desktop.
	v.wallpaperDir = t.TempDir()

	// Update staging must never touch the real cache directory or construct
	// a live GitHub client. Tests that exercise Check/Download assign
	// v.updater's client themselves (httptest + fake Verifier) before
	// enabling the setting; newTestUI only redirects the stage dir.
	v.updater.SetDir(t.TempDir())

	var isClosed bool
	win.SetOnClosed(func() { isClosed = true })

	t.Cleanup(func() {
		if !isClosed {
			win.Close()
		}
	})

	// Cleanup runs before window close (LIFO). Join workers and drain stale
	// delivery while this window is alive so no work reaches the next test.
	t.Cleanup(func() { drain(t, v) })

	return v, win, func() bool { return isClosed }
}

// drain waits out every background operation this viewer may still have in
// flight. Each wait is individually optional - Wait on a completion.Signal
// that never began returns immediately - but the set is exhaustive on
// purpose: it is the backstop that keeps one test's goroutines out of the
// next one, whatever that test happened to exercise. toast.hidden is the
// one Signal deliberately left out of the table below - see newTestUI's
// v.toast.duration comment for why waiting it out here would block for an
// hour.
func drain(t *testing.T, v *viewer) {
	t.Helper()

	// Before anything else, and unlike every other row here: the open-with
	// handler is the one piece of state this viewer installs *outside*
	// itself. internal/openwith wraps a single package-level queue - one
	// NSApp callback per process - so a viewer left installed there would
	// go on receiving a later test's delivery and drop files into a window
	// this test has already closed. Clearing it first also means nothing
	// can start a fresh scan behind the waits below.
	openwith.SetHandler(nil)
	v.images.Stop()
	v.stopSearchOverlayWait()
	v.searchView.overlayWorkers.Wait()
	if v.searchView.overlayUI != nil {
		v.searchView.overlayUI.Drain()
	}
	v.visualsearch.Stop()
	v.analysisCache.Stop()
	v.analysisCache.Settle()
	v.visualsearch.Settle()
	v.closeExplorer()
	v.settleExplorer()
	v.closeFileWork()
	v.closeClipboardWork()
	v.closeOpenChooser()
	v.stopWinPosPoll()
	v.settingsWin.StopTracking()
	v.exif.StopTracking()
	v.mosaicWin.StopTracking()

	// Supersede any in-flight decode/retry chain first, so a load that was
	// deliberately abandoned mid-test (a broken-file retry loop, say) stops
	// re-entering rather than being waited out step by step - invalidateLoad
	// also cancels its context, so an abandoned decode/preload actually
	// stops doing I/O instead of just being ignored once it finishes. The
	// slideshow is asked to stop for the same reason, on this goroutine,
	// since leaving picture-frame mode touches the window.
	v.invalidateLoad()
	v.scanOp.lifecycle.invalidate()
	v.sortOp.lifecycle.invalidate()
	v.regionCopyLifecycle.invalidate()
	v.display.Stop()
	v.closeFavoritePreviews()
	v.grid.Stop()
	v.exif.Stop()
	v.updateOp.invalidate()
	v.slides.Exit()
	v.compare.Close()
	v.spiral.Close()
	v.mosaicWin.Close()
	v.deletion.Close()
	v.deletion.Settle()

	// Stop closed display admission before settlement; late root callbacks
	// cannot begin another load or raster pass during teardown.
	v.display.Settle()
	drainClipboard(t, v)
	drainFileWork(t, v)
	waitFor(t, "the file chooser at cleanup", &v.chooser)
	drainOpenChooser(t, v)

	// Native chooser delivery can begin scan, which can begin sort and load.
	// Drain that delivery first, then observe the current root generations.
	// Display was stopped above and joins all its retired workers afterward.
	for _, c := range []struct {
		name string
		sig  *completion.Signal
	}{
		{"the clipboard copy at cleanup", &v.clipboard},
		{"the file-manager reveal at cleanup", &v.reveal},
		{"the wallpaper at cleanup", &v.wallpaper},
		{"the update check at cleanup", v.updater.Done()},
		{"the favorite previews at cleanup", &v.favThumb},
		{"the scan at cleanup", &v.scanOp.done},
		{"the sort at cleanup", &v.sortOp.done},
		{"the comparison at cleanup", v.compare.Done()},
	} {
		waitFor(t, c.name, c.sig)
	}

	v.display.Wait()
	v.display.Settle()

	// Done names only the latest generation. A manual request can supersede
	// an automatic worker while the older one is still unwinding before it
	// reaches the serialized stage transaction, so drain the updater's full
	// worker set as a second barrier.
	updateCtx, cancelUpdate := context.WithTimeout(context.Background(), testTimeout)
	defer cancelUpdate()
	if err := v.updater.Settle(updateCtx); err != nil {
		t.Fatal("timed out draining all update workers at cleanup")
	}

	compareCtx, cancelCompare := context.WithTimeout(context.Background(), testTimeout)
	defer cancelCompare()
	if err := v.compare.Settle(compareCtx); err != nil {
		t.Fatal("timed out draining all comparison workers at cleanup")
	}

	mosaicCtx, cancelMosaic := context.WithTimeout(context.Background(), testTimeout)
	defer cancelMosaic()
	if err := v.mosaicWin.Settle(mosaicCtx); err != nil {
		t.Fatal("timed out draining all mosaic workers at cleanup")
	}

	settled := make(chan struct{})
	go func() {
		v.waitWinPosPoll()
		v.settingsWin.WaitForTracking()
		v.exif.WaitForTracking()
		v.mosaicWin.WaitForTracking()
		v.favThumbWorkers.Wait()
		v.display.WaitPreloads()
		v.grid.Settle()
		v.slides.Settle()
		v.spiral.Settle()
		v.exif.Settle()
		v.images.Wait()
		close(settled)
	}()

	select {
	case <-settled:
	case <-time.After(testTimeout):
		t.Fatal("timed out draining favorite/preload/thumbnail/slideshow goroutines at cleanup")
	}
}

// newTestViewer is newTestUI for the majority of tests, which drive the
// viewer directly and never need the window handle or the closed-reporter.
func newTestViewer(t *testing.T) *viewer {
	t.Helper()

	v, _, _ := newTestUI(t)

	return v
}

// --- waiting for async work -------------------------------------------------

// testTimeout is the deadline every wait helper below gives its operation.
// One value for all of them, rather than a per-call argument: a timeout
// here is a failure deadline, not a delay - a passing test returns as soon
// as its channel closes and never waits this long - so a single generous
// value costs nothing and keeps the call sites free of a number that
// suggested a tuning knob nobody was actually turning.
// Concurrent Linux/amd64 race shards under emulation can exceed five seconds
// while painting a valid completion; allow that work to settle before teardown.
const testTimeout = 30 * time.Second

// waitFor blocks until s's current operation finishes, failing the test on
// timeout. One helper for every completion.Signal on the viewer, so the
// testTimeout deadline lives in exactly one place instead of being
// restated by a hand-rolled select per operation. It deliberately does not
// check Begun(): drain uses it, and a never-begun Signal must still return
// immediately.
func waitFor(t *testing.T, name string, s *completion.Signal) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	if err := s.Wait(ctx); err != nil {
		t.Fatalf("timed out waiting for %s", name)
	}
}

// waitHandle is waitFor for a generation captured before a newer request
// superseded it - see completion.Signal.Current.
func waitHandle(t *testing.T, name string, h completion.Handle) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	if err := h.Wait(ctx); err != nil {
		t.Fatalf("timed out waiting for %s", name)
	}
}

// dropAndWait drops uris and waits for the resulting scan, reorder and load
// to finish - the opening lines of nearly every test in this suite. The sort
// step is part of the chain because applyScanResult hands the scanned files
// to startSort, which only shows the first image once the reorder lands.
// Use dropAndWaitScan instead when the drop is expected to load nothing
// (no supported images), since neither sortOp.done nor display.LoadDone is touched in
// that case.
func dropAndWait(t *testing.T, v *viewer, uris ...fyne.URI) {
	t.Helper()

	v.handleDrop(uris)
	waitForScan(t, v)
	waitForSort(t, v)
	waitUntilLoaded(t, v)
}

// dropAndWaitScan drops uris and waits only for the scan, for drops that
// end with nothing displayable - an unsupported file, an empty folder, a
// merge that adds nothing. Deliberately no waitForSort: applyScanResult
// returns before ever reaching startSort in that case, so v.sortOp.done is
// left untouched at whatever generation some earlier call begun.
func dropAndWaitScan(t *testing.T, v *viewer, uris ...fyne.URI) {
	t.Helper()

	v.handleDrop(uris)
	waitForScan(t, v)
}

func waitUntilLoaded(t *testing.T, v *viewer) {
	t.Helper()

	if !v.display.LoadBegun() {
		t.Fatal("the image load never started")
	}

	v.display.Settle()
	waitHandle(t, "the image to finish loading", v.display.LoadDone())

}

func waitForScan(t *testing.T, v *viewer) {
	t.Helper()

	if !v.scanOp.done.Begun() {
		t.Fatal("the scan never started")
	}

	waitFor(t, "the scan", &v.scanOp.done)
}

func waitForSort(t *testing.T, v *viewer) {
	t.Helper()

	if !v.sortOp.done.Begun() {
		t.Fatal("the sort never started")
	}

	waitFor(t, "the sort", &v.sortOp.done)
}

// waitForAnimFrame drains display delivery until at least n frames have been
// applied. Worker submission alone cannot satisfy this observation.
func waitForAnimFrame(t *testing.T, v *viewer, n uint64) {
	t.Helper()

	deadline := time.Now().Add(testTimeout)
	for v.display.AppliedFrames() < n {
		v.display.Settle()
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for animFrame to reach %d", n)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// parkAnimate installs a clock that never ticks before the first load.
// Cancellation still releases playback; frameClock permits individual ticks.
func parkAnimate(v *viewer) {
	v.display.SetAnimationClock(func(_ time.Duration) <-chan time.Time { return make(chan time.Time) })
}

// waitForAnimStopped observes the current playback worker's return.
func waitForAnimStopped(t *testing.T, v *viewer) {
	t.Helper()

	if !v.display.AnimationBegun() {
		t.Fatal("the animation never started")
	}

	waitHandle(t, "the animation to stop", v.display.AnimationDone())
}

// waitForClipboard waits encoding/dispatch, drains queued results, then observes
// the operation's completion. A dispatcher-entry channel precedes UI effects.
func waitForClipboard(t *testing.T, v *viewer) {
	t.Helper()

	if !v.clipboard.Begun() {
		t.Fatal("the clipboard copy never started")
	}

	drainClipboard(t, v)
	waitFor(t, "the clipboard copy", &v.clipboard)
}

func drainClipboard(t *testing.T, v *viewer) {
	t.Helper()
	finished := make(chan struct{})
	go func() { v.clipboardWork.workers.Wait(); close(finished) }()
	select {
	case <-finished:
	case <-time.After(testTimeout):
		t.Fatal("timed out waiting for clipboard encoding/dispatch")
	}
	for v.clipboardWork.ui.Drain() {
	}
}

// waitForReveal waits out the goroutine a file-manager reveal runs on -
// v.reveal is finished once that goroutine has fully run, error toast
// included, exactly as waitForClipboard's is.
func waitForReveal(t *testing.T, v *viewer) {
	t.Helper()

	if !v.reveal.Begun() {
		t.Fatal("the file-manager reveal never started")
	}

	waitFor(t, "the file-manager reveal", &v.reveal)
}

// waitForCached polls imgCache - populated from preloadOne's background
// goroutines, which run independently of display/scan completion - until it holds
// an entry for u, the same polling-with-timeout style waitForAnimFrame uses
// for animate's background writes.
func waitForCached(t *testing.T, v *viewer, u fyne.URI) {
	t.Helper()

	deadline := time.Now().Add(testTimeout)
	for !v.imgCache.Contains(u.String()) {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %q to be preloaded into imgCache", u.Name())
		}
		time.Sleep(2 * time.Millisecond)
	}
}

// --- priming and settling background work -----------------------------------

// settleToast finishes the current toast deterministically: it cancels the
// pending auto-hide timer, waits for that goroutine to exit, then runs the
// hide synchronously through the same autoHide path the timer would have
// taken. Any test that triggers a toast should call this before returning.
// It replaces the old real-time wait-for-auto-hide design, which both kept
// ~2s of wall-clock per toast test (a shortened global duration) and let
// the timer's inline fyne.Do perform widget writes concurrently with the
// test goroutine's own - the suite's dominant source of -race failures
// before stage 2 (concurrent access to Fyne/harfbuzz's shared text-shaping
// state included).
func settleToast(t *testing.T, v *viewer) {
	t.Helper()

	if v.toast.stop == nil {
		t.Fatal("no toast auto-hide pending to settle - was a toast actually shown?")
	}

	v.toast.cancelAutoHide()

	waitFor(t, "the toast's auto-hide goroutine", &v.toast.hidden)

	v.toast.autoHide(v.toast.gen.Load())
}

// settleChooser waits for native work and drains open results on this test's
// UI goroutine. Only then can callers wait for any child scan/sort/load work.
func settleChooser(t *testing.T, v *viewer) {
	t.Helper()
	if !v.chooser.Begun() {
		t.Fatal("no file-chooser goroutine pending to settle")
	}
	drainFileWork(t, v)
	waitFor(t, "the file-chooser goroutine", &v.chooser)
	drainOpenChooser(t, v)
}

func drainOpenChooser(t *testing.T, v *viewer) {
	t.Helper()
	stopped := make(chan struct{})
	go func() { v.openChooserWorkers.Wait(); close(stopped) }()
	select {
	case <-stopped:
	case <-time.After(testTimeout):
		t.Fatal("timed out waiting for all native open chooser workers")
	}
	for v.chooserUI.Drain() {
	}
}

// settleSlideshow leaves picture-frame mode (a no-op when it's already
// off) and waits for the session's auto-advance goroutine to exit. Every
// test that enters picture-frame mode registers this as a cleanup:
// without it the goroutine outlives the test, sleeps out its interval
// (10s default), and then wakes to advance a slide - full inline-fyne.Do
// UI work - in the middle of whatever test is running by then.
//
// Exit runs on this goroutine, since it un-full-screens the window; only
// the wait is handed off, so a goroutine that never notices fails the test
// instead of hanging it.
func settleSlideshow(t *testing.T, v *viewer) {
	t.Helper()

	v.slides.Exit()

	settled := make(chan struct{})
	go func() {
		v.slides.Settle()
		v.exif.Settle()
		close(settled)
	}()

	select {
	case <-settled:
	case <-time.After(testTimeout):
		t.Fatal("timed out waiting for the slideshow goroutine to exit")
	}
}

// warmThumbs decodes every current file's thumbnail synchronously into the
// grid's cache, so opening the grid afterwards populates each cell from
// the cache without spawning decode goroutines. That matters under the
// fyne test driver: a spawned decode's completion paint runs inline on the
// decode goroutine and can interleave with the very cell-refresh walk that
// spawned it - a race that is already over before any post-hoc wait could
// begin, so it can only be prevented, not waited out. The async decode path
// itself is still covered by TestRequestThumbnail_DecodesInBackgroundAndCaches,
// which drives requestThumbnail directly while the main goroutine stays
// quiescent.
func warmThumbs(t *testing.T, v *viewer) {
	t.Helper()

	if err := v.grid.Warm(); err != nil {
		t.Fatalf("warming thumbnails: %v", err)
	}
}

// --- file-set assertions ----------------------------------------------------

func assertEquivalentFileSlices(t *testing.T, v *viewer) {
	t.Helper()

	files := namesOfURIs(v.state.files)
	unsorted := namesOfURIs(v.state.unsortedFiles)
	slices.Sort(files)
	slices.Sort(unsorted)
	if !slices.Equal(files, unsorted) {
		t.Errorf("files = %v and unsortedFiles = %v do not contain the same URIs", v.state.files, v.state.unsortedFiles)
	}
}

func assertValidFileIndex(t *testing.T, v *viewer) {
	t.Helper()

	if len(v.state.files) == 0 {
		if v.state.index != 0 {
			t.Errorf("index = %d, want 0 with no files", v.state.index)
		}
		return
	}
	if v.state.index < 0 || v.state.index >= len(v.state.files) {
		t.Errorf("index = %d, want a value in [0, %d)", v.state.index, len(v.state.files))
	}
}

func namesOfURIs(files []fyne.URI) []string {
	names := make([]string, len(files))
	for i, u := range files {
		names[i] = u.Name()
	}
	return names
}

// Leave a real uncached request queued; callers cancel it before test cleanup.
func beginPendingImageLoad(v *viewer) {
	v.display.Load(display.Request{Source: storage.NewFileURI("/picfetch-pending-test.png")})
}
