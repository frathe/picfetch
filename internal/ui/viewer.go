package ui

import (
	"fmt"
	"os"
	"slices"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/completion"
	"github.com/frathe/picfetch/internal/decodepool"
	"github.com/frathe/picfetch/internal/dupes"
	"github.com/frathe/picfetch/internal/filesort"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/ui/autoupdate"
	"github.com/frathe/picfetch/internal/ui/copyselection"
	"github.com/frathe/picfetch/internal/ui/deletion"
	"github.com/frathe/picfetch/internal/ui/display"
	"github.com/frathe/picfetch/internal/ui/exifwin"
	"github.com/frathe/picfetch/internal/ui/favorites"
	"github.com/frathe/picfetch/internal/ui/grid"
	"github.com/frathe/picfetch/internal/ui/help"
	"github.com/frathe/picfetch/internal/ui/infoview"
	"github.com/frathe/picfetch/internal/ui/menus"
	"github.com/frathe/picfetch/internal/ui/settingswin"
	"github.com/frathe/picfetch/internal/ui/slideshow"
	"github.com/frathe/picfetch/internal/ui/widgets"
	"github.com/frathe/picfetch/internal/ui/zoom"
	"github.com/frathe/picfetch/internal/wingesture"
	"github.com/frathe/picfetch/internal/winpos"
)

// viewer bundles the UI elements and the navigation state so the drop
// handler and the key handler can share them without package-level globals.
type viewer struct {
	app fyne.App
	win fyne.Window
	// quit requests application shutdown after PerformUpdate has successfully
	// recorded apply-and-relaunch intent. buildViewer initializes it from the
	// app instance; tests replace this per-viewer seam so they never stop the
	// shared test app.
	quit       func()
	img        *canvas.Image
	hint       *widget.Label
	dropzone   *fyne.Container
	loadingBar *widget.ProgressBarInfinite

	// windowSize is the window's current content size, kept up to date by
	// windowSizeTracker (windowtrack.go) on every layout of the root
	// content - Window itself exposes no size getter. Read once, at
	// shutdown, by Run's preferences.Save call so the window geometry a
	// user left the app at carries over to the next launch (see
	// internal/preferences).
	windowSize fyne.Size

	// dropzoneArt wraps the whole drop zone - the welcome/placeholder art,
	// the hint text, and restoreLink - in a tappable widget (see
	// openfiles.go) so clicking anywhere in the box opens a file dialog, for
	// users who never drag-and-drop.
	dropzoneArt *widgets.TappableArea

	// welcomeArt greets the user on first launch, in the same box as
	// emptyStateArt. handleDrop hides it on the first drop, so a later
	// error shows emptyStateArt there instead; reset (Escape) brings it
	// back, since that returns the viewer to its just-launched state.
	welcomeArt *canvas.Image

	// emptyStateArt is shown alongside the hint, on the right of the drop
	// zone, only while an error has left it with no images to display.
	emptyStateArt *canvas.Image

	// toast is the self-dismissing notification card behind ShowToast -
	// see the toast type (toast.go) for its widgets and auto-hide
	// lifecycle.
	toast *toast

	// help owns the manual and About windows and the Help menu - see
	// internal/ui/help, which needs nothing from the viewer at all.
	help *help.Help

	// favorites owns the Favorites menu and its add/open/remove dialogs.
	favorites *favorites.Feature

	// exif is the EXIF metadata panel - see internal/ui/exifwin, which
	// reaches back only through the "which file is on screen" accessor
	// registerFeatures hands it. finishLoad calls its Refresh so navigating
	// while it's open keeps it in sync.
	exif *exifwin.Window

	// settingsWin is the Settings window (File menu) - see
	// internal/ui/settingswin, which is seeded from settingsState() and
	// reaches back through ApplySettings plus the two update verbs.
	// Named settingsWin, not settings, because that name belongs to the
	// settings-backed state struct below.
	settingsWin *settingswin.Window

	// menus holds the File, Window and Actions menu items whose
	// Checked/Disabled state moves at runtime - the File three ("Save
	// Changes", "Export image", "Close Files"), the five Window items, and
	// the thirteen Actions ones. They are behind a field, unlike the inert
	// items built alongside them, because that state has to be updated
	// from outside buildMainMenu itself, at every site that can change
	// what is saveable, exportable or open, or which surface is showing:
	// rotate.go, load.go, sort.go, info.go, keys.go, save.go,
	// clearToDropzone, and the feature observers registered in
	// buildMainMenu. internal/ui/menus computes that whole matrix as one
	// pure function of a State value; menuState and syncMenus below are
	// where this viewer fills it in. Set as Wallpaper lives on the Actions
	// menu rather than File.
	menus *menus.Menus

	// restoreLink offers to reload the file set saved when the window last
	// closed (see session.go). Shown only while welcomeArt is - and only
	// when savedSession is non-empty - and hidden together with it, since
	// past the first drop there's nothing left to "restore" into.
	restoreLink *widget.Hyperlink

	// savedSession is the file set loaded from the previous run's session
	// cache (session.go), consumed once by restoreSession. nil when there
	// is nothing to restore.
	savedSession []fyne.URI

	state appState

	// loadLifecycle owns a logical navigation and all of its descendants:
	// probe/decode retries, neighbor preloads, and GIF animation. A newer
	// navigation, drop, clear, or shutdown cancels and supersedes the token.
	loadLifecycle requestLifecycle

	// favThumbLifecycle owns the background favorite-preview pass
	// (favthumbs.go). Independent of every lifecycle above it: the pass
	// belongs to a favorite rather than to whatever is on screen, so it
	// outlives the navigation that started it and is superseded only by
	// the next favorite opened or saved.
	favThumbLifecycle requestLifecycle

	// baseTitle is the window title without the "[merge] " prefix applyTitle
	// adds while merge mode is on, so toggling M can refresh the title
	// immediately without recomputing it.
	baseTitle string

	// gridTitle names the file under the grid overview's highlight while
	// the overview is up, and is empty otherwise. It *overrides* baseTitle
	// in applyTitle rather than replacing it, so whatever the image view
	// last set - or sets while the grid is open, e.g. a drop landing behind
	// it - is still there to fall back to when the grid closes, without
	// anyone having to save and restore a string.
	gridTitle string

	// loading is true while a decode/render is in flight. The key handler
	// checks it to ignore repeat events instead of piling up decodes for
	// images the user has already navigated past.
	loading atomic.Bool

	// winPos is the window's last known on-screen position. Unlike
	// windowSize, which windowSizeTracker captures for free off ordinary
	// layout passes, there is no Fyne hook that fires on a pure window
	// move (see internal/winpos) - startWindowPosPolling (windowtrack.go)
	// keeps the tracker current with a background poller instead, and the
	// slideshow captures and restores it directly around full-screen. A
	// value field, never copied: its state is atomic because the poller
	// reads and writes it off the UI goroutine.
	winPos winpos.Tracker

	// spiralDrag recognises the window being swirled around the desktop in
	// a spiral - the easter egg's second, wordless door (see gesture.go).
	// It is fed from the same poller that keeps winPos current.
	spiralDrag *wingesture.Detector

	// spiralGesture is what a recognised spiral does, taking whether it was
	// drawn clockwise - which selects the pattern the easter egg opens on.
	// registerFeatures points it at help.OpenSpiral; it is
	// a field so tests can watch the gesture fire without opening a real
	// full-screen shader window, and so this file needs to know nothing
	// about what the gesture is for.
	spiralGesture func(clockwise bool)

	// stopWinPosPoll stops startWindowPosPolling's background ticker
	// goroutine; initialized to noPollerStop by buildViewer, replaced by
	// Run after startup geometry is restored, and called from SetOnStopped
	// just before the final preferences save (winPos keeps its last reading,
	// so the save still has a value).
	stopWinPosPoll func()

	// scanOp is the folder-scan progress UI - see asyncop.go's asyncOpUI for
	// the shape it shares with the background reorder (sortOp below).
	// scanOp.lifecycle is independent of navigation, so browsing the
	// existing set during a merge-mode folder scan cannot strand scan UI
	// state. scanOp.active is true while a handleDrop scan (the current
	// generation) is in flight, so Escape can cancel it - see cancelScan.
	// Only ever set on the UI goroutine: true when handleDrop starts, false
	// again once that same generation's completion closure runs, whether it
	// finished, found nothing, or was cancelled. A scan superseded by a
	// newer drop (stale gen) never touches it, since the newer scan already
	// owns the flag by the time the stale one's closure would run.
	// scanOp.done is finished by handleDrop's fyne.Do completion block, once
	// that call's generation has finished applying its result. Tests wait
	// on it directly instead of polling widget state, which otherwise races
	// with the fyne test driver's synchronous fyne.Do under -race. Each
	// call begins a fresh completion generation before starting its async
	// work; a stale request's own generation still gets finished, it just
	// leaves the shared state untouched.
	scanOp asyncOpUI

	// load is begun by ShowImage and finished by whichever step of that
	// call's decode/retry chain ends it - see load.go. The whole chain
	// shares one generation rather than beginning a new one per retry, so
	// a waiter sees the chain as finished only once it truly settles
	// instead of racing whichever retry finishes first.
	// See internal/completion for the contract.
	//
	// A value field, never copied: it holds a mutex.
	load completion.Signal

	// sortOp is the background-reorder progress UI - see asyncop.go's
	// asyncOpUI for the shape it shares with the scan. sortOp.active is
	// true while the current sortOp.lifecycle request is still
	// meaningfully pending, set by startSort and cleared by whichever of
	// invalidateSort (a newer sort, Escape via cancelSort, RemoveFile,
	// clearToDropzone) or finishSort landing that same token notices
	// first - see sort.go's invalidateSort, which every one of those but
	// finishSort itself goes through - so it never gets stuck true once
	// whatever it was tracking has been superseded, cancelled, or
	// discarded. Used for two things: gating cancelSort (nothing to cancel
	// if nothing's in flight) and handleKeyEvent's Escape case (keys.go) -
	// a first-ever drop clears v.scanOp.active before startSort has actually
	// populated v.state.files, so without this Escape would see
	// len(v.state.files) == 0 and quit the window instead of cancelling
	// the still-computing reorder. sortOp.lifecycle owns the cancellable
	// filesort.Order request, staying separate from loadLifecycle so
	// reordering cannot stop an unrelated decode, preload, or playing GIF.
	// sortOp.done is finished by finishSort once that request's reorder has
	// finished applying (or been discarded as stale), mirroring v.scanOp.done
	// and v.load so tests can wait on it deterministically.
	sortOp asyncOpUI

	// animFrame counts every write to v.img.Image - attemptLoad's initial
	// frame plus each one animate cycles to afterwards - and anim is
	// finished by animate once its load token is cancelled or stale and
	// it returns. Both exist so tests can synchronize on frame changes
	// and animation shutdown via an atomic and a completion.Signal
	// instead of reading v.img.Image directly from another goroutine,
	// which would race with attemptLoad's/animate's writes under the fyne
	// test driver: it runs fyne.Do synchronously on the calling goroutine
	// rather than marshaling onto a single UI thread, so even a read
	// sequenced after the load signal finishes has no happens-before edge
	// against a concurrently running animate call - only observing
	// animFrame's new value does. Each animate call gets its own captured
	// finisher (see finishLoad), so a superseded request's completion
	// can't be mistaken for a newer one's.
	//
	// anim is a value field, never copied: it holds a mutex.
	animFrame atomic.Uint64
	anim      completion.Signal

	// frameAfter is time.After behind a per-viewer seam so a test can
	// release GIF frames one at a time instead of racing a live timer.
	// Write-once: set at construction, and by a test only before its first
	// drop (concurrency invariant).
	frameAfter func(time.Duration) <-chan time.Time

	// display owns what is on the canvas right now - the current image's
	// decoded frames, which of them is up, the view-only rotation, and
	// the picture-frame crossfade - see internal/ui/display, whose State
	// doc carries what each piece means. The choreography stays here:
	// installLoadedFrames (load.go) installs a fresh image's frames and
	// resets index and rotation on every navigation, animate advances the
	// index, and rotateBy/resetRotation (rotate.go) turn the rotation and
	// redraw. The fade is only ever running while picture-frame mode is
	// active: ShowImage starts one fading the outgoing image out,
	// finishLoad starts the next fading the incoming one in, and every
	// path that ends picture-frame mode calls resetFade so the image is
	// never left invisible or half-faded once it's back in the normal,
	// instant-swap view. A value field, never copied.
	display display.State

	// imgCache holds recently decoded frames keyed by URI string, so
	// navigating back to an image already seen this session - or one
	// preloadNeighbors decoded speculatively ahead of time - is a cache hit
	// instead of a fresh disk read plus decode. Bounded by a byte budget
	// (imgCacheMB below, the settings window's binding) rather than an
	// entry count, since a decoded image's size varies by four orders of
	// magnitude - see imaging.ByteCache, which is safe for concurrent use
	// on its own, so both attemptLoad's decode goroutine and preloadOne's
	// background goroutines can populate it without going through fyne.Do.
	imgCache *imaging.ByteCache[*imaging.LoadedImage]

	// preloads bounds how many speculative neighbor decodes run at once and
	// stops rapid navigation piling up a second decode of the same
	// not-yet-cached neighbor while the first is still in flight - see
	// internal/decodepool, the same pool *type* internal/ui/grid fills with
	// thumbnails, but its own instance and its own budget of slots, the way
	// imgCache and the grid's thumbnail cache are two caches rather than
	// one. Keyed by URI string; the claim carries no value, since the
	// URI alone says what the work is. Without the bound, rapid navigation
	// could stack an unbounded number of full-size decode goroutines.
	// waitUntilLoaded (harness_test.go) waits it out after every load so a
	// preload goroutine never outlives the test whose navigation spawned it.
	preloads *decodepool.Pool[string, struct{}]

	// zoom is the zoom/pan view of img (0/1/+/- and drag/scroll) - see
	// internal/ui/zoom, whose widget sits in the window's content Stack in
	// place of img itself, so it can override Stack's usual "fill the
	// container" layout. It needs no Host: the app and that package share
	// img on a single-writer-per-field contract (the app owns img.Image,
	// zoom owns img's size and position), and the only reach back is the
	// updateInfoOverlay callback registerFeatures hands it.
	zoom *zoom.Zoom

	// regionCopy is the transient image-region selection surface. The
	// feature owns pointer interaction, image-space geometry, and the
	// captured source's crop/encode path; this viewer owns composition with
	// zoom, information-overlay visibility, animation pause, menus, and the
	// clipboard worker (copyselection.go).
	regionCopy *copyselection.Feature

	// regionCopyDo defers geometry notifications out of zoom's renderer
	// Layout before they mutate the selection overlay. Production uses
	// fyne.Do; newTestUI installs a synchronous per-viewer seam so focused
	// tests stay deterministic under Fyne's inline test driver.
	regionCopyDo func(func())

	// regionCopyInfoVisible remembers whether the information card itself
	// was painted when Copy Selection mode began. Its standing I-key
	// preference remains owned by infoview.Card and is never toggled here.
	regionCopyInfoVisible bool

	// regionCopyAnimated is whether finishRegionCopy must unpause the
	// load-owned animation loop. The captured pixels themselves live on
	// the Feature.
	regionCopyAnimated bool

	// regionCopyLifecycle cancels stale crop/encode work; regionCopyDoAndWait
	// is a per-viewer seam for deterministic UI-hop tests.
	regionCopyLifecycle requestLifecycle
	regionCopyDoAndWait func(func())

	// animationPause keeps an animated frame fixed from source capture until
	// Copy Selection ends, without replacing the load-owned animation loop.
	animationPause animationPause

	// info is the persistent info overlay (I key) - see internal/ui/infoview,
	// which owns its own widgets, its standing show/hide preference, and the
	// current file's raw facts (byte size, EXIF presence, RAW-preview flag).
	// toggleInfoOverlay/syncInfoOverlayVisibility/updateInfoOverlay (info.go)
	// are the thin glue that builds its State snapshot from state.files/
	// zoom/vector, none of which infoview has access to.
	info *infoview.Card

	// deletion is the Shift+Delete confirmation flow - see
	// internal/ui/deletion, which owns its own widgets and selection state
	// and reaches back through the Host interface this viewer satisfies.
	// handleKeyEvent checks its Visible() before anything else so every
	// other key is swallowed while a delete decision is pending.
	deletion *deletion.Confirmer

	// exportPrompt is the export-format choice raised by promptExport
	// (export.go, File menu's "Export image" / Cmd/Ctrl+E): the same
	// widgets.ChoiceCard deletion's own card is built on, but used bare
	// here rather than through a wrapping type - exportAs already re-checks
	// canExport/CurrentFile at call time, so there's no per-request state
	// left for a wrapper to hold the way deletion's targets are.
	// handleKeyEvent checks its Visible() the same way it does deletion's.
	exportPrompt *widgets.ChoiceCard

	// dupes owns which files are duplicates of which - see internal/dupes,
	// which is Fyne-free and reaches this viewer's file set through the
	// dupeFileSet adapter (visibility.go). The viewer constructs it, not
	// the grid: hide-duplicates is a standing setting that outlives the
	// overlay, and plain arrow-key navigation has to answer "is this file
	// visible?" with the grid closed. registerFeatures builds it before
	// grid.New, which is handed the same *dupes.Model to read and feed
	// from its hashing pass.
	dupes *dupes.Model

	// grid is the full-window thumbnail overview (G key) - see
	// internal/ui/grid, which owns the thumbnail cache and its decode
	// worker pool and reaches back through the Host interface this viewer
	// satisfies. handleKeyEvent checks its Visible() before its own
	// dispatch, the same way it does for the delete confirmation.
	grid *grid.Overview

	// slides is picture-frame mode (P key) - see internal/ui/slideshow,
	// which owns the full-screen switch, the auto-advance goroutine and
	// the interval behind it, and reaches back through a two-method Host
	// (FileCount/Advance) this viewer satisfies. The app's other
	// full-window mode is the grid above; neither knows about the other,
	// so keeping them from overlapping is this package's job - see
	// handleKeyEvent's G case and togglePictureFrameMode.
	slides *slideshow.Controller

	// clipboard is begun by copyImageToClipboard (clipboard.go),
	// copyGridSelection (batch.go), and Copy Selection, and finished once
	// that goroutine has
	// fully run, error reporting included. chooser is the same for the
	// native file dialog, shared by openFileDialog (openfiles.go) and
	// exportAs (export.go) - they mean "the native dialog goroutine" and
	// are never in flight at once, since both panels are app-modal.
	// See internal/completion for the contract all of these keep.
	//
	// Value fields, never copied: each holds a mutex.
	clipboard completion.Signal
	chooser   completion.Signal

	// wallpaper is begun by setAsWallpaper (wallpaper.go) and finished
	// once the change has fully landed, toast included. favThumb is the
	// same for SyncFavoritePreviews' pass over a favorite's previews
	// (favthumbs.go); it is begun on every pass, so a test waits on it
	// after triggering one rather than holding it across two.
	// See internal/completion for the contract.
	//
	// Value fields, never copied: each holds a mutex.
	wallpaper completion.Signal
	favThumb  completion.Signal

	// wallpaperDir is where setAsWallpaper (wallpaper.go) writes the PNG it
	// hands to the OS - defaultWallpaperDir in production, a t.TempDir() in
	// tests, which is why it is a field rather than a package-level
	// constant: this is the one action whose side effect outlives the
	// process, on the developer's own desktop.
	wallpaperDir string

	// pendingInitial is the file set the launch itself carried - the
	// command-line paths Run was handed, plus anything macOS delivered as
	// an "Open With" before the viewer existed (see internal/openwith) -
	// waiting for the event loop to come up before it is opened. Run fills
	// it in SetOnStarted and openwith.go drains it; whichever of that
	// file's two paths reaches it first takes the whole batch and clears
	// it, which is what keeps a launch carrying both kinds of file to a
	// single scan. Only ever touched on the UI goroutine.
	pendingInitial []fyne.URI

	// updater owns client preparation, the release-check/download policy, the
	// staged-update lifecycle, the What's-New cache, and the last-check-day
	// storage - see internal/ui/autoupdate. updateOp mirrors
	// scanOp/loadLifecycle: one requestLifecycle for the background
	// check/download, kept here rather than promoted into that package (this
	// refactor's locked decision on cancellation), so
	// maybeStartUpdateCheck (autoupdate.go) prepares the client, then begins
	// the token and hands Updater.Start its context and a staleness func.
	updater  *autoupdate.Updater
	updateOp requestLifecycle

	// settings is the whole settings-backed state - see memlimits.go's
	// settings for what it holds and why it's grouped.
	settings settings

	// keyModifiers reports the keyboard modifiers currently held -
	// defaultKeyModifiers (keys.go) in production, stubbed by tests (the
	// fyne test driver can't synthesize modifier state at all). Read by
	// handleKeyEvent's Shift+R, and by the zoom view's Shift+scroll pan
	// through the closure registerFeatures hands it.
	keyModifiers func() fyne.KeyModifier

	// vector is the whole state of the SVG re-render - see vector.go's
	// vectorView for what it holds and why it's a value field.
	vector vectorView
}

// ForceRepaint refreshes the window's root content object, which has been
// part of the canvas (and so already registered with it) since startup.
// Fyne only registers an object with its canvas the first time it is
// painted while visible, so calling Show()/Refresh() on a widget that has
// spent its whole life hidden - like scanOp.spinner, scanOp.label or
// loadingBar between uses - can't find a canvas to mark dirty and silently
// fails to schedule a repaint; it would otherwise only appear once some
// unrelated event (e.g. a window resize, which marks the canvas dirty
// directly) forces a full repaint. Refreshing an already-registered
// ancestor here triggers that repaint immediately instead.
func (v *viewer) ForceRepaint() {
	v.win.Content().Refresh()
}

// setTitle updates the window title to base, remembering it so a later
// mergeMode toggle can reapply the "[merge] " prefix (or drop it) without
// needing to recompute the title from scratch.
func (v *viewer) setTitle(base string) {
	v.baseTitle = base
	v.applyTitle()
}

// applyTitle (re)applies the current title to the window with a sort-mode
// prefix (see filesort.Label - empty, so invisible, for the default by-name
// sort) and the "[merge]"/"[shuffle]" prefixes when merge mode or the
// slideshow's shuffle order (Shift+P) are on, so the title always makes
// the active drop/sort/slideshow mode visible at a glance. The separating
// space is added here rather than baked into either prefix, so neither
// translation key carries trailing whitespace a translator could silently
// drop. While the grid overview is up gridTitle takes baseTitle's place -
// the image behind it isn't what the user is looking at. Show-variants
// goes further and hides every prefix too: its title is a bare
// `(index/count) [WxH] /absolute/path`, so the bar is only position, size,
// and path, with nothing else competing for room.
func (v *viewer) applyTitle() {
	title := v.baseTitle
	if v.gridTitle != "" {
		title = v.gridTitle
	}
	hidePrefixes := v.grid != nil && v.grid.BrowsingDuplicates() && v.gridTitle != ""
	if !hidePrefixes {
		if v.state.MergeMode() {
			title = lang.L("[merge]") + " " + title
		}
		if v.slides.Shuffle() {
			title = lang.L("[shuffle]") + " " + title
		}
		if p := filesort.Label(v.state.SortMode()); p != "" {
			title = p + " " + title
		}
	}
	v.win.SetTitle(title)
}

// HighlightChanged names the grid overview's highlighted file in the window
// title (internal/ui/grid's Host): with the image view hidden behind the
// overlay, the title is the only place a thumbnail's file name is spelled
// out in full. i is -1 when nothing is highlighted - the grid closing, or a
// search matching no file - which hands the title back to the image view.
// The hide-duplicates and unfiltered grids still omit pixel size: that
// would cost a full decode of a file nobody has picked yet. Show-variants
// reuses the already-probed native size and the full path instead, and
// applyTitle omits [merge]/[shuffle]/sort prefixes for that title.
// Show-variants enablement follows the highlighted host while the grid is
// open, so this also recomputes the menus; syncMenus refreshes the native
// bar only if something in that matrix actually moved.
func (v *viewer) HighlightChanged(i int) {
	if i < 0 || i >= len(v.state.files) {
		v.gridTitle = ""
		v.applyTitle()
	} else {
		v.gridTitle = v.gridHighlightTitle(i)
		v.applyTitle()
	}

	v.syncMenus()
}

// gridHighlightTitle names the file under the grid ring. Hide-duplicates
// and the unfiltered grid keep the basename and a trailing position
// counter. Show-variants compares copies of one shot, so the title is
// `(index/count) [WxH] /absolute/path` from the already-probed native
// size - not a new decode. applyTitle strips mode prefixes for this form.
func (v *viewer) gridHighlightTitle(i int) string {
	u := v.state.files[i]
	if v.grid.BrowsingDuplicates() {
		head := ""
		if n := len(v.state.files); n > 1 {
			head = fmt.Sprintf("(%d/%d) ", i+1, n)
		}
		if w, h, ok := v.dupes.NativeSizeAt(i); ok {
			return fmt.Sprintf("%s[%dx%d] %s", head, w, h, u.Path())
		}
		return head + u.Path()
	}
	title := u.Name()
	if n := len(v.state.files); n > 1 {
		title = fmt.Sprintf("%s  (%d/%d)", title, i+1, n)
	}
	return title
}

// clearToDropzone drops the loaded file list and returns the viewer to an
// empty drop-zone state: no image, no files, and (unless a fixed window
// size is set) the window back to its start size and title. Callers pick
// which art (welcomeArt or emptyStateArt) belongs in the box afterward and
// are responsible for repainting.
func (v *viewer) clearToDropzone() {
	// A full-screen dropzone would look broken, and there's nothing left to
	// frame - safe to call even when picture-frame mode is already off.
	v.slides.Exit()
	v.resetFade()

	v.invalidateLoad()    // invalidate any decode/preload or animation still in flight
	v.invalidateSort()    // cancel a sort still in flight - see sortOp's field comment
	v.scanOp.invalidate() // same shape: supersede the token and finish the overlay if a scan is in flight

	v.state.clearFiles()
	v.dupes.ClearInspect()

	// Purged, not left to age out: with no files open, every decode the
	// cache holds is of something unreachable, so keeping them just spends
	// the byte budget on nothing until the next drop happens to refill it.
	v.imgCache.Purge()

	v.img.Image = nil
	v.img.Hide()
	// Drop leftover frames so rotate/zoom enablement (Count() == 0) and
	// rotateBy's no-op agree with the empty drop zone.
	v.display.Clear()
	v.clearVector() // an in-flight rasterization must not land on whatever loads next

	// The info card's own standing preference is left alone - it's a
	// preference like sortMode/mergeMode, so the card comes back on the
	// next load if it was on. state.files and img.Image are already
	// cleared above, so this call only hides the widget.
	v.syncInfoOverlayVisibility()

	v.loading.Store(false)
	v.loadingBar.Hide()

	v.hint.SetText(lang.L("Drop images here"))
	v.dropzone.Show()

	v.setTitle(appTitle)
	v.undoGridMaximize()
	// With a fixed window size the empty drop zone must not shrink the
	// window back to startW×startH - that is exactly the size the user
	// asked to keep. Dynamic mode still resets to the drop-zone size so
	// Escape returns to the same compact welcome frame as a fresh launch.
	if !v.settings.staticWindowSize {
		v.win.Resize(fyne.NewSize(startW, startH))
	}
	v.syncMenus()
}

// undoGridMaximize restores the window from a grid-triggered maximize (see
// grid.Overview.ConsumeMaximized) before something is about to resize it
// for a reason of its own - a no-op unless the grid actually left it
// maximized. A plain Resize call alone can't shrink a window Maximize
// grew: on Linux and Windows the maximized state is tracked by the OS
// independently of window geometry, so a Resize made while it's still set
// is silently ignored - see winpos.Unmaximize. Restoring the last known
// position afterward matters for the same reason: the OS's own
// un-maximize placement rarely lands back where the window was before the
// grid took over.
func (v *viewer) undoGridMaximize() {
	if !v.grid.ConsumeMaximized() {
		return
	}
	winpos.Unmaximize(v.win)
	v.winPos.Restore(v.win)
}

// toggleMergeMode flips whether the next drop merges into the existing set
// instead of replacing it - see SetMergeMode below, which does the actual
// work.
func (v *viewer) toggleMergeMode() {
	v.SetMergeMode(!v.state.MergeMode())
}

// SetMergeMode sets merge mode directly - the settings window's binding for
// the toggle above - and immediately reflects it in the window title via
// the "[merge] " prefix so it doesn't wait for a drop to become visible.
func (v *viewer) SetMergeMode(on bool) {
	v.state.SetMergeMode(on)
	v.applyTitle()
	v.syncMenus()
}

// MergeMode reports whether merge mode is on - the settings window's
// getter.
func (v *viewer) MergeMode() bool {
	return v.state.MergeMode()
}

// showFileIfPresent looks up target in v.state.files by URI identity and shows it
// if found, reporting whether it was. Used to keep the same file in view
// across an operation - a sort toggle or a merge - that reorders or extends
// v.state.files without changing what's currently on screen.
func (v *viewer) showFileIfPresent(target fyne.URI) bool {
	for i, u := range v.state.files {
		if u.String() == target.String() {
			v.ShowImage(i)
			return true
		}
	}
	return false
}

// reset returns the viewer to the state it was in at launch, so Escape can
// act as "start over" instead of quitting whenever there's something to
// clear.
func (v *viewer) reset() {
	v.clearToDropzone()

	// Also cleared here, not just inside clearToDropzone: every path back to
	// the drop zone must independently abandon the vector, so none of them
	// can regress into leaving an in-flight rasterization able to land on
	// whatever loads next.
	v.clearVector()

	v.showWelcomeState()
	v.ForceRepaint()
}

// closeFiles is the File menu's "Close Files" item: it drops the currently
// loaded set and returns to the welcome drop zone, cancelling a scan still
// in progress first - unlike Escape (handleKeyEvent), it never closes the
// window, since File > Close is a distinct action from quitting the app.
func (v *viewer) closeFiles() {
	if v.scanOp.active {
		v.cancelScan()
	}
	v.reset()
	v.clearVector() // see reset's own comment - each layer clears independently
}

// showWelcomeState restores the launch-time welcome look: welcome art in
// place of the empty-state error art, plus the restore-session link when
// there's a saved session to offer. Shared by reset and cancelScan, which
// differ only in how much else they put back.
func (v *viewer) showWelcomeState() {
	v.emptyStateArt.Hide()
	v.welcomeArt.Show()
	if len(v.savedSession) > 0 {
		v.restoreLink.Show()
	}
}

// ShowEmptyStateError clears back to an empty drop zone - so a previously
// displayed image never lingers behind an error - shows the error
// placeholder art, and raises a toast. Used whenever a drop, scan, or
// decode ends with nothing to display.
func (v *viewer) ShowEmptyStateError(msg string) {
	v.clearToDropzone()

	v.welcomeArt.Hide()
	v.restoreLink.Hide()
	v.emptyStateArt.Show()

	v.ForceRepaint()
	v.ShowToast(msg)
}

// The exported methods on this unexported type - CurrentFile, RemoveFile,
// ShowImage, ShowToast, ShowEmptyStateError, ForceRepaint - are the shared
// vocabulary the feature packages' own Host interfaces are written against
// (see internal/ui/deletion). One method satisfies every such interface, so
// the viewer never grows per-package adapters; and because the type itself
// stays unexported, none of it is reachable from outside internal/ui.

// CurrentFile returns the file currently displayed and its index, or
// ok=false when nothing is loaded.
func (v *viewer) CurrentFile() (u fyne.URI, index int, ok bool) {
	if len(v.state.files) == 0 {
		return nil, 0, false
	}

	return v.state.files[v.state.index], v.state.index, true
}

// displayedFile is CurrentFile narrowed to what the EXIF panel needs: a
// file that is not merely selected but actually decoded and on screen.
// The distinction matters during a failed or in-flight load, when v.state.files
// is non-empty but there is no image to describe.
func (v *viewer) displayedFile() (fyne.URI, bool) {
	if v.img.Image == nil {
		return nil, false
	}

	u, _, ok := v.CurrentFile()

	return u, ok
}

// DisplayedFile is the file decoded and on screen, ok=false when the
// drop zone is showing. Satisfies exifwin.Host; narrower than
// CurrentFile, which still reports a selected index during a failed load.
func (v *viewer) DisplayedFile() (fyne.URI, bool) {
	return v.displayedFile()
}

// AfterMetadataRemoved is exifwin.Host: the JPEG at u just lost its
// identifying tags. Evict that file's decode-cache entry so a later visit
// cannot revive HasEXIF. Overlay size / EXIF-link updates apply only when
// u is still the file on screen, so a navigation while the confirmation
// was up cannot hide a different photo's link.
func (v *viewer) AfterMetadataRemoved(u fyne.URI) {
	if u == nil {
		return
	}
	v.imgCache.Remove(u.String())
	if shown, ok := v.displayedFile(); ok && shown.String() == u.String() {
		info, err := os.Stat(u.Path())
		var size int64
		if err == nil {
			size = info.Size()
		}
		v.info.AfterMetadataRemoved(size, err == nil)
		v.syncInfoOverlayVisibility()
		v.updateInfoOverlay()
	}
	v.exif.Refresh()
}

// RemoveFile drops the file at v.state.files[i] from both v.state.files and
// v.state.unsortedFiles, keeping them in sync so a later sort toggle doesn't
// resurrect a file that failed to load. v.state.files is trimmed by index rather
// than by URI match, since merge mode allows dropping the same file twice
// and a match would risk removing the wrong duplicate; unsortedFiles has
// no equivalent index to use, but any matching duplicate there is an
// equally valid one to drop. Evicting the removed file's decode from
// imgCache is appState's job rather than this method's - see its onRemove
// hook (state.go), which fires for every removal however it is reached.
func (v *viewer) RemoveFile(i int) {
	v.invalidateSort() // cancel a sort still in flight - see sortOp's field comment

	v.state.removeFile(i)
}

// RemoveFiles drops every named index in one pass - what internal/ui/deletion
// calls once a batch of files has actually reached the Trash.
//
// Descending, so an earlier removal can't shift a later index out from under
// the same call, and sorted first because the caller's order is not something
// this should depend on. Duplicates are skipped rather than removing two
// different files for one index named twice.
//
// The grid is reconciled here, at the end, rather than by the caller: every
// index it holds - its selection, its filter's display→host mapping, its
// highlight - is an index into the set that just changed underneath it. It
// is also closed outright once nothing is left, since an open grid over an
// empty file set has no cells to draw and Toggle itself refuses to open in
// that state.
func (v *viewer) RemoveFiles(indices []int) {
	prev := -1
	for _, i := range slices.Backward(slices.Sorted(slices.Values(indices))) {
		if i == prev || i < 0 || i >= len(v.state.files) {
			continue
		}
		prev = i

		v.RemoveFile(i)
	}

	v.grid.FilesChanged()
	if len(v.state.files) == 0 {
		v.grid.Close()
	}
}

// Modifiers is which keyboard modifiers are held right now, for the feature
// packages that need to read a gesture rather than a key press -
// internal/ui/grid's Cmd/Ctrl+click and Shift+click. A method over the
// keyModifiers field rather than the field itself, so it satisfies grid.Host
// alongside the rest of the vocabulary below; internal/ui/zoom takes the same
// value as a bare func, which is why the field exists in the first place.
func (v *viewer) Modifiers() fyne.KeyModifier {
	return v.keyModifiers()
}

// FileCount, FileAt, OpenFiles, CurrentIndex, Generation, Unfocus, Modifiers,
// and Advance complete the exported vocabulary the feature packages' Host
// interfaces bind to (see the note above CurrentFile). internal/ui/grid uses
// the first six: the first three to draw the right cells, Generation to
// discard a decode whose file set has since been replaced, Unfocus to
// hand the keyboard back after a thumbnail tap, and Modifiers to tell its
// three click gestures apart. internal/ui/slideshow needs only FileCount and
// Advance.

// FileCount is how many files are currently loaded.
func (v *viewer) FileCount() int {
	return len(v.state.files)
}

// FileAt returns the file at index i.
func (v *viewer) FileAt(i int) fyne.URI {
	return v.state.files[i]
}

// OpenFiles sends a file list through the same scan, merge, sort, and display
// path as a drag-and-drop or the native file chooser.
func (v *viewer) OpenFiles(files []fyne.URI) {
	v.handleDrop(files)
}

// CurrentIndex is the index of the file on screen.
func (v *viewer) CurrentIndex() int {
	return v.state.index
}

// Generation is the current index-to-URI file-set revision. Navigation does
// not change it; replacement, reorder, removal, and clear operations do.
// Grid thumbnail and deletion work captures it and discards results whose
// generation has moved on. It is deliberately independent of
// loadLifecycle: navigation changes the displayed index but not what any
// index means.
//
// It is read out of the published snapshot rather than a counter of its
// own, so the generation and the keys it describes are one value - see
// appState.publish.
func (v *viewer) Generation() uint64 {
	return v.state.snapshot().Generation()
}

// Unfocus releases Fyne's canvas focus.
func (v *viewer) Unfocus() {
	v.win.Canvas().Unfocus()
}

// Advance displays the next file, wrapping around at the end - attemptLoad
// folds the index back into range, so there is nothing to clamp here. It's
// the slideshow's auto-advance step. With shuffle off it's deliberately
// the same navigation the Right key performs rather than a private one;
// with shuffle on (Shift+P) it picks a random other file instead - see
// randomOtherIndex - still through the same ShowImage every navigation
// goes through, so the crossfade and everything else load.go does on a
// navigation applies here too.
func (v *viewer) Advance() {
	if v.slides.Shuffle() {
		v.ShowImage(v.randomVisibleOther(v.state.index))
		return
	}
	v.ShowImage(v.nextVisibleIndex(v.state.index, 1))
}

// StepImage moves by delta files (typically +1 or -1), wrapping through
// ShowImage. No-op with fewer than two files in the current set, while a
// load is in flight, while a Fyne dialog overlay is up on the main window
// (same reason as handleKeyEvent: the dialog owns the keyboard, and
// StepImage is callable from the always-on-top EXIF window), or while the
// delete / export-format prompt owns the main window. A single-file drop
// may already have been expanded to the parent directory by handleDrop, so
// this guard is the current set size, not whether the user dropped only
// one path.
// Picture-frame shuffle does not apply: this is what the arrow keys do.
func (v *viewer) StepImage(delta int) {
	if v.win.Canvas().Overlays().Top() != nil {
		return
	}
	if v.deletion.Visible() || v.exportPrompt.Visible() {
		return
	}
	if len(v.state.files) < 2 || v.loading.Load() {
		return
	}
	v.ShowImage(v.nextVisibleIndex(v.state.index, delta))
	if v.slides.Active() {
		v.slides.Kick()
	}
}
