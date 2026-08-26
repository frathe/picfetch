// Package grid is the full-window thumbnail overview (the G key): a
// virtualized grid of every loaded image, for jumping around a large drop
// by sight instead of arrowing through it one file at a time.
//
// It owns the thumbnail cache and the bounded worker pool that fills it,
// and reaches back into the app through Host. It knows nothing about the
// app's other full-window mode (the slideshow): the two don't compose, but
// that guard lives in the app's key dispatcher, not here.
package grid

import (
	"image"
	"image/color"
	"sync"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/decodepool"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/selection"
	"github.com/frathe/picfetch/internal/ui/widgets"
	"github.com/frathe/picfetch/internal/winpos"
)

// cellSize is the fixed width/height, in canvas points, each grid cell is
// laid out at.
const cellSize = 120

// dupBadgeMargin insets the group-size chip from the cell edge so the
// highlight ring (GridRingWidth) cannot run through the digits.
const dupBadgeMargin float32 = 10

const dupBadgeTextSize float32 = 14

// Host is what the overview needs from the application: the file set to
// draw, the generation counter that tells a finished decode whether its
// file set is still current, and the display actions a selection triggers.
type Host interface {
	// FileCount is how many files are loaded.
	FileCount() int

	// FileAt returns the file at index i.
	FileAt(i int) fyne.URI

	// CurrentIndex is the file currently on screen - where the highlight
	// starts when the grid opens.
	CurrentIndex() int

	// Generation is the app's index-to-URI file-set revision. A decode
	// captures it when it starts and discards its result if it no longer
	// matches, so replacement, reorder, or removal cannot paint a stale
	// thumbnail. Navigation alone leaves it unchanged.
	Generation() uint64

	// ShowImage displays the file at index i.
	ShowImage(i int)

	// HighlightChanged reports which file the ring is on while the grid is
	// up, so the app can name it in the window title - the only thing that
	// identifies a thumbnail once the image view is hidden. i is -1
	// whenever no file is under the ring: the grid closing, or a search
	// that matches nothing.
	HighlightChanged(i int)

	// ForceRepaint redraws the window after a visibility change.
	ForceRepaint()

	// Unfocus releases canvas focus - see Close for why that matters.
	Unfocus()

	// Modifiers is which keyboard modifiers are held right now. A Fyne tap
	// carries none of its own, so the multi-select gestures (Cmd/Ctrl+click
	// to toggle a cell, Shift+click to extend a range) have to ask at the
	// moment the tap arrives - the same accessor internal/ui/zoom already
	// uses for its Shift+scroll pan, and stubbable per-viewer for the same
	// reason: Fyne's test driver implements no desktop.Driver to read them
	// from.
	Modifiers() fyne.KeyModifier

	// ShowToast displays a short, non-blocking notification.
	ShowToast(msg string)
}

// Overview is the grid overlay and the state behind it.
type Overview struct {
	host Host
	win  fyne.Window

	visible      bool
	onVisibility func()
	onDupeState  func()
	wrap         *widget.GridWrap
	overlay      *fyne.Container

	// The bar across the top of the overlay, hidden until there is either a
	// search or a selection to report: what was typed on the left, how much
	// of the set still matches and how much of it is picked on the right.
	// empty is the notice drawn over the grid in the one state that has no
	// cells at all to explain itself.
	searchBar   *fyne.Container
	searchLabel *widget.Label
	countLabel  *widget.Label
	selLabel    *widget.Label
	empty       *widget.Label

	// maximized is true from the moment Toggle grows the window via
	// winpos.Maximize until ConsumeMaximized is next called. Left true
	// across a plain Close - see its own doc comment: closing the grid
	// deliberately leaves the window maximized - so a later resize the app
	// makes for a reason of its own (not just picking an image straight out
	// of the grid, but navigating on afterward too) still knows to undo it
	// first.
	maximized bool

	// highlight is which cell's ring is currently drawn, moved by the
	// arrow keys while the grid is up and committed with Return. Reset to
	// the host's current index every time the grid opens, so it starts on
	// whichever image was already on screen.
	highlight int

	// searching is whether the search bar is up, opened by typing '/' and
	// left by Escape. Distinct from "a filter is active" (matches != nil):
	// an open search with nothing typed yet still shows every file.
	searching bool

	// query is the filter text typed so far, matched case-insensitively
	// against each file's base name.
	query string

	// sel is the multi-select: which files are picked, and the anchor a
	// Shift+click extends from. Holds host file indices rather than the
	// display indices actually clicked, so it survives a filter change -
	// see selection.go.
	sel *selection.Set

	// marqueeSaved is the host-index selection frozen at mouse-down, so a
	// Shift/Cmd drag unions against what the user started with rather than
	// against a live set that the previous Dragged already replaced.
	marqueeSaved []int

	// marqueeDragging is true between the first Dragged of a gesture and
	// DragEnd, Escape-cancel, or Close. escape treats it as the first undo
	// stage so a press during a drag restores the snapshot instead of
	// clearing the selection the drag just built.
	marqueeDragging bool

	// marqueeDisarmed suppresses the rest of a gesture after Escape or
	// Close: the driver keeps sending Dragged until the button comes up,
	// and without this the next event would start a fresh marquee from
	// the current point.
	marqueeDisarmed bool

	// marqueeOrigin is the press point in wrap-content coordinates,
	// recovered on the first Dragged by subtracting that event's delta.
	marqueeOrigin fyne.Position

	// catcher is the transparent fyne.Draggable under the padded wrap.
	// marqueeRect is the painted selection rectangle, parented by
	// marqueeBox (WithoutLayout) so Stack cannot stretch it.
	catcher     *marqueeCatcher
	marqueeRect *canvas.Rectangle
	marqueeBox  *fyne.Container

	// matches maps a display index - a cell's position in the grid - to
	// the host's own file index, while a filter is active. nil means no
	// filter, and is what every index below means "identity" by: the grid
	// renumbers its cells from zero when filtered, but ShowImage,
	// FileAt and CurrentIndex all speak the host's numbering, so
	// everything crossing that boundary goes through fileIndex.
	matches []int

	// filterGen counts changes to matches, so a thumbnail decode already
	// in flight can tell that the cell it was started for has been
	// renumbered under it. The host's own generation can't see this: the
	// file set is unchanged by a keystroke, and so is the cell's id - only
	// what that id *means* moved. Atomic because applyFilter writes it on
	// the UI goroutine while a decode worker reads it.
	filterGen atomic.Uint64

	// thumbs holds small, already-downsampled thumbnails keyed by URI
	// string - a separate cache and byte budget from the app's full-size
	// image cache (imaging.NewThumbCache vs NewImgCache), since reusing
	// one for both would evict full-size decodes the normal viewing path
	// still needs, and vice versa. SetCacheBytes retunes the budget while
	// the app runs; see its own comment.
	thumbs *imaging.ByteCache[image.Image]

	// decodes bounds concurrent thumbnail decodes and gates duplicate work
	// per recycled cell - see internal/decodepool. The key is the cell
	// container (the stable per-slot widget, not the image inside it) and the
	// value is the file id that cell's in-flight decode is working toward, so
	// a cell recycled onto a different file supersedes rather than blocks.
	//
	// The duplicate gate earns its keep here specifically: every repaint
	// re-runs GridWrap's update callback for every visible cell, and one
	// multi-megapixel decode easily outlives several repaints - without the
	// gate, each of those passes would queue another goroutine for work
	// already underway. Wait is what Settle waits on.
	decodes *decodepool.Pool[*fyne.Container, int]

	// ui is how a decode worker's completion reaches the UI goroutine -
	// see uiqueue.go for why that is a field and not a direct fyne.Do.
	ui UIQueue

	// cellIDs tracks which file id each recycled cell is currently
	// showing: GridWrap reuses a small, fixed pool of cell widgets as the
	// user scrolls rather than creating one per file, so an async decode
	// kicked off against an earlier id has to check, once it completes,
	// that its cell hasn't since been recycled to show a different file. A
	// *sync.Map, not a plain map: stillWanted reads it from the decode
	// worker goroutine, deliberately before decoding and outside g.ui, so
	// the cheap pre-decode bail (see its call site in requestThumbnail)
	// doesn't have to marshal - while the cell-update callback writes it
	// on the UI goroutine. That read/write pair races under any driver,
	// real or test; it has nothing to do with which one marshals fyne.Do.
	cellIDs sync.Map

	// hashes maps URI string → dHash. Not stored in thumbs: a hash is 8
	// bytes and must survive thumbnail eviction. pixels maps URI string
	// → native Dx*Dy for the same generation; absent means unknown.
	// Thumbnails are capped, so size cannot be recovered from the thumb
	// cache. hashGen is the host Generation those entries belong to; a
	// newer drop wipes hashes, hashFailed, and pixels.
	hashMu sync.Mutex
	hashes map[string]uint64
	pixels map[string]int
	// hashFailed are URIs whose thumbnail decode already failed this
	// generation. hashRemaining must not retry them: mixed-format drops
	// leave unreadable files, and retrying on every Shift+D re-raises
	// the analyzing toast with no CPU work left to do.
	hashFailed map[string]struct{}
	hashGen    uint64

	// hideDupes hides non-representative duplicates; dupeDist is the
	// Hamming threshold DuplicateGroups uses. groupSizes/groupReps are
	// per host index (0 = unhashed). hashing dedups in-flight hashRemaining
	// jobs by URI string. hashJobs counts those pool jobs so the last one
	// can finishBrowse. hideApply stays set until the in-flight UI
	// install returns, so an idle fyne.Do cannot re-arm mid-apply and
	// queue one install per file. hideApplyAt floors mid-window
	// installs so the event loop still sees input while hashing.
	hideDupes   bool
	hideApply   atomic.Bool
	hideApplyAt atomic.Int64
	// browseHost is the host index being browsed, or -1 when browse is
	// off. Zero is a valid file index - New MUST set this to -1.
	browseHost int
	dupeDist   int
	groupSizes []int
	groupReps  []int
	hashing    sync.Map
	hashJobs   atomic.Int32
	// groupComputes counts computeDuplicateGroups calls so tests can
	// tell a hash worker computed off the UI queue rather than inside it.
	groupComputes atomic.Int32
}

// dupBadge is the group-size chip on a grid cell: white digits on a black
// backdrop, pinned top-right and stacked above the highlight ring.
type dupBadge struct {
	chip  *fyne.Container
	bg    *canvas.Rectangle
	label *canvas.Text
}

func newGridCell() *fyne.Container {
	img := canvas.NewImageFromImage(nil)
	img.FillMode = canvas.ImageFillContain
	img.ScaleMode = canvas.ImageScaleFastest
	img.SetMinSize(fyne.NewSize(cellSize, cellSize))

	// The selection tint sits under the highlight ring so the ring's
	// stroke stays crisp over it: the two mark different things and
	// routinely land on the same cell.
	tint := widgets.NewSelectionTint()
	tint.Hide()

	ring := widgets.NewFocusRing(widgets.GridRingWidth, widgets.RingRadius)
	ring.Hide()

	label := canvas.NewText("", color.White)
	label.TextSize = dupBadgeTextSize
	label.TextStyle = fyne.TextStyle{Bold: true}
	label.Alignment = fyne.TextAlignCenter

	bg := canvas.NewRectangle(color.Black)
	bg.CornerRadius = 4

	chip := container.NewStack(bg, container.New(
		layout.NewCustomPaddedLayout(2, 2, 6, 6),
		label,
	))
	chip.Hide()

	// WithoutLayout so the chip can sit in the corner at its own min
	// size; a Border right-slot would stretch it down the cell. Stacked
	// last so the highlight stroke cannot paint through the digits.
	return container.NewStack(img, tint, ring, container.NewWithoutLayout(chip))
}

func unpackGridCell(cell *fyne.Container) (*canvas.Image, *canvas.Rectangle, *canvas.Rectangle, *dupBadge) {
	img := cell.Objects[0].(*canvas.Image)
	tint := cell.Objects[1].(*canvas.Rectangle)
	ring := cell.Objects[2].(*canvas.Rectangle)
	chip := cell.Objects[3].(*fyne.Container).Objects[0].(*fyne.Container)
	return img, tint, ring, &dupBadge{
		chip:  chip,
		bg:    chip.Objects[0].(*canvas.Rectangle),
		label: chip.Objects[1].(*fyne.Container).Objects[0].(*canvas.Text),
	}
}

// New builds the overview (hidden) around host. win is maximized (see
// Toggle) each time the overview opens - a bigger window means bigger, more
// legible thumbnails - the same reason slideshow.Controller is handed win
// directly rather than reaching for it some other way.
//
// Each cell is a small stack of the thumbnail image plus a highlight ring
// (mirroring the delete confirmation's own selection rings), rather than
// relying on GridWrap's own built-in highlight rendering - that ties into
// real Fyne canvas focus, which this app deliberately never hands to
// GridWrap (see Close's comment on why).
func New(host Host, win fyne.Window) *Overview {
	g := &Overview{
		host:       host,
		win:        win,
		sel:        selection.New(),
		thumbs:     imaging.NewThumbCache(imaging.DefaultThumbCacheBytes),
		decodes:    decodepool.New[*fyne.Container, int](thumbConcurrency),
		ui:         fyneQueue{},
		hashes:     make(map[string]uint64),
		hashFailed: make(map[string]struct{}),
		pixels:     make(map[string]int),
		dupeDist:   imaging.DuplicateMaxDistance,
		browseHost: -1,
	}

	g.wrap = widget.NewGridWrap(
		g.count,
		func() fyne.CanvasObject {
			return newGridCell()
		},
		func(id widget.GridWrapItemID, o fyne.CanvasObject) {
			cell := o.(*fyne.Container)
			img, tint, ring, badge := unpackGridCell(cell)

			g.cellIDs.Store(cell, id)
			setCellHighlighted(ring, id == g.highlight)
			setCellSelected(tint, g.isSelected(id))
			g.applyDupBadge(badge, g.fileIndex(id), cell.Size())

			// Refresh reaches this callback whether or not the overlay is
			// actually open: every ForceRepaint refreshes the whole widget
			// tree, hidden GridWrap included. Requesting thumbnails from
			// those refreshes would kick off background decodes for an
			// invisible grid on every navigation step - and, under the
			// fyne test driver (where fyne.Do runs a goroutine's callback
			// inline on the caller instead of marshaling it to the UI
			// goroutine), would let such a decode's completion paint race
			// a later refresh's own cell reset. Skip while hidden: Toggle
			// sets visible before its own refresh, so opening the grid
			// still populates every visible cell.
			if !g.visible {
				img.Image = nil
				img.Refresh()
				return
			}

			// While visible, blanking is requestThumbnail's job, and only
			// on a cache miss - clearing unconditionally here made every
			// scroll tick repaint every already-cached cell twice, with an
			// empty flash in between.
			g.requestThumbnail(cell, img, id, g.host.Generation())
		},
	)

	// A tap on a cell: which of the three things it means depends on what is
	// held down at the time, which the event itself doesn't say - see
	// Host.Modifiers.
	//
	// Both selection gestures have to hand the keyboard back themselves.
	// Fyne's GridWrap grabs canvas focus on every tap, and this app
	// dispatches every key from the *unfocused* canvas handler; Close is
	// what normally undoes that, so a tap that deliberately leaves the grid
	// open would otherwise leave a focused GridWrap swallowing the arrow
	// keys, '/' and everything after it.
	g.wrap.OnSelected = func(id widget.GridWrapItemID) {
		defer g.wrap.UnselectAll()

		switch toggle, extend := pickModifier(g.host.Modifiers()); {
		case toggle:
			g.toggleAt(id)
			g.host.Unfocus()
		case extend:
			g.extendTo(id)
			g.host.Unfocus()
		default:
			// Resolved before Close, not after: closing clears the filter,
			// and an id resolved past that point would map to itself rather
			// than to the file this cell was actually showing.
			i := g.fileIndex(id)

			g.Close()
			if i >= 0 {
				g.host.ShowImage(i)
			}
		}
	}

	// Fired both by keyboard highlight movement (HandleKey forwards the
	// arrow keys to wrap.TypedKey, see nav.go) and by mouse hover (GridWrap
	// wires its own onHovered to the same callback) - either way, move the
	// ring to match.
	//
	// The guard is what stops setHighlight's own re-entry through here from
	// recursing: it re-enters with g.highlight already equal to id.
	g.wrap.OnHighlighted = func(id widget.GridWrapItemID) {
		if id == g.highlight {
			return
		}
		g.setHighlight(id)
	}

	g.searchLabel = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	g.countLabel = widget.NewLabel("")
	g.selLabel = widget.NewLabelWithStyle("", fyne.TextAlignTrailing, fyne.TextStyle{Bold: true})
	g.searchBar = container.NewBorder(nil, nil, nil,
		container.NewHBox(g.selLabel, g.countLabel), g.searchLabel)
	g.searchBar.Hide()

	g.empty = widget.NewLabelWithStyle(lang.L("No file names match"), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	g.empty.Hide()

	// An opaque backdrop, not a translucent scrim like the delete
	// confirmation's: the grid replaces the image view entirely rather
	// than dimming it behind a centered card, so it needs to fully hide
	// whatever's underneath.
	backdrop := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))

	// Body stack order is load-bearing. Walk is back-to-front and the last
	// match wins: the catcher is Draggable but not Tappable, Hoverable, or
	// Scrollable, so cell taps, hover, wheel, and the scrollbar still land
	// on GridWrap, while a drag on a cell or gutter hits the catcher. The
	// rectangle sits on top in a WithoutLayout layer so Stack cannot
	// stretch it to the overlay.
	g.catcher = newMarqueeCatcher(g)
	g.marqueeRect = widgets.NewMarqueeRect()
	g.marqueeBox = container.NewWithoutLayout(g.marqueeRect)

	body := container.NewStack(
		g.catcher,
		container.NewPadded(g.wrap),
		container.NewCenter(g.empty),
		g.marqueeBox,
	)
	g.overlay = container.NewStack(backdrop, container.NewBorder(g.searchBar, nil, nil, nil, body))
	g.overlay.Hide()

	return g
}

// Overlay is the full-window grid, for the app to place in its window
// stack.
func (g *Overview) Overlay() fyne.CanvasObject {
	return g.overlay
}

// Visible reports whether the grid is up - the app's key dispatcher checks
// this before its own handling.
func (g *Overview) Visible() bool {
	return g.visible
}

// SetOnVisibilityChanged registers f to run after the grid opens or
// actually closes. The field is read at fire time. nil is a no-op.
func (g *Overview) SetOnVisibilityChanged(f func()) { g.onVisibility = f }

func (g *Overview) fireVisibility() {
	if g.onVisibility != nil {
		g.onVisibility()
	}
}

// ConsumeMaximized reports whether the window is still sitting maximized
// from an earlier Toggle and hasn't been undone since, clearing the flag
// either way - a one-shot check for whoever is about to resize the window
// for a reason of their own to know whether it first needs to undo the
// grid's maximize (see winpos.Unmaximize), without ever being told twice.
func (g *Overview) ConsumeMaximized() bool {
	m := g.maximized
	g.maximized = false
	return m
}

// Toggle flips the grid on or off. A no-op with nothing loaded. The
// caller is responsible for not opening it while another full-window mode
// owns the screen - see the app's key dispatcher.
func (g *Overview) Toggle() {
	if g.visible {
		g.Close()
		return
	}
	if g.host.FileCount() == 0 {
		return
	}

	g.visible = true

	// Maximize, not full-screen (see winpos.Maximize) - more room for more,
	// bigger thumbnails at once, without picture-frame mode's chrome-free
	// look. Deliberately one-way: closing the grid does not shrink the
	// window back down, the same way clicking a real maximize button
	// doesn't un-maximize when you switch to another app and back. A no-op
	// wherever winpos can't reach a native window (the fyne test driver,
	// Wayland, mobile, wasm), so the grid still opens there, just without
	// the resize.
	winpos.Maximize(g.win)
	g.maximized = true

	// Start the highlight on whichever image is currently on screen, and
	// scroll it into view - setHighlight also refreshes the grid, which is
	// what actually paints the ring. Host index to display index: hide-dupes
	// (and search) renumber the cells.
	g.setHighlight(g.displayIndexOf(g.host.CurrentIndex()))
	g.overlay.Show()
	g.host.ForceRepaint()
	g.fireVisibility()
}

// Close dismisses the grid, restoring the normal image view. A no-op when
// it isn't showing, so the app can call it defensively (on every drop, and
// when entering its other full-window mode) without checking Visible
// first.
//
// Unfocuses the canvas on the way out: tapping a thumbnail is a real Fyne
// widget tap, and Fyne's own GridWrap unconditionally grabs canvas focus
// on tap before calling OnSelected. This app otherwise never uses Fyne's
// widget-focus system - every other key binding is dispatched manually
// from the canvas's default (unfocused) key handler - so a focused
// GridWrap left behind after closing would silently swallow every key
// press afterward (arrow keys included) until something else happened to
// steal focus back.
func (g *Overview) Close() {
	if !g.visible {
		return
	}

	g.visible = false
	// The filter and the selection are both ways of working with the grid,
	// not standing settings: each open starts on the whole set with nothing
	// picked. This also covers the app's defensive Close on every drop,
	// where a query or a selection left over from the previous file set
	// would otherwise be applied to - or, worse, acted on against - the new
	// one.
	g.sel.Clear()
	if g.marqueeDragging {
		g.marqueeDisarmed = true
	}
	g.marqueeDragging = false
	g.marqueeSaved = nil
	g.hideMarqueeRect()
	g.clearSearch()
	// Browse is a way of working with the grid, like search: G must reopen
	// the hide/full set, not the last group. Hide-duplicates stays; Close
	// never clears that standing setting.
	if g.browseHost >= 0 {
		g.SetBrowsingDuplicates(false)
	}
	// Explicitly, because clearSearch returns early with no search open and
	// would otherwise leave the bar showing a selection count that no longer
	// applies the next time the grid opens.
	g.syncTopBar()
	g.overlay.Hide()
	g.host.HighlightChanged(-1)
	g.host.Unfocus()
	g.host.ForceRepaint()
	g.fireVisibility()
}
