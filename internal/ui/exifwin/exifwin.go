// Package exifwin is the EXIF metadata window: a small panel listing the
// current image's camera settings, opened with the E key or the info
// overlay's "Show EXIF data" link. Below the list (and, for a JPEG with
// tags to strip, a Remove Metadata button) sits a collapsible
// OpenStreetMap view, shown only for a photo that carries GPS tags and
// collapsed until the user expands it - which is also what keeps the widget
// from fetching any map tiles unasked.
package exifwin

import (
	"context"
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	xwidget "fyne.io/x/fyne/widget"

	"github.com/frathe/picfetch/internal/completion"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/ui/widgets"
)

const (
	exifW = 420.0
	// exifH leaves room for the Remove Metadata button above the map
	// (it sits in the north stack with the tag list, not under the map
	// in the south: a 240px map MinSize in the Border center overflows
	// and paints over a south slot). Without the extra height the map
	// itself would open shorter than mapH.
	exifH = 420.0

	// mapH is the least the map opens at, and mapZoom how far in it
	// starts: close enough to read the streets around the pin, far enough
	// to place it in its town. Beyond mapH the map follows the window.
	mapH    = 240.0
	mapZoom = 15
)

// Host is what the EXIF window needs from the app once it can mutate the
// file on disk. DisplayedFile is the file actually on screen (not merely
// selected during a failed load) — the same fact New used to take as a
// func. It is not named CurrentFile: viewer.CurrentFile already returns
// an index for deletion.Host. AfterMetadataRemoved is given the URI that
// was just rewritten, not whatever is on screen now.
type Host interface {
	DisplayedFile() (fyne.URI, bool)
	AfterMetadataRemoved(u fyne.URI)
	ShowToast(msg string)
	StepImage(delta int)
}

// Window is the EXIF panel. At most one is open at a time (widgets.
// Singleton): a second request raises the existing window rather than
// stacking up duplicates.
type Window struct {
	app  fyne.App
	host Host

	win widgets.Singleton

	// text is the panel's content label, live only while the window is
	// open (nil otherwise, which is what makes Refresh a no-op then).
	text *widget.Label

	// strip is the Remove Metadata button, live only while the window is
	// open. stripBar is the centered wrapper around it in the north stack,
	// hidden as a unit so a hidden button does not leave its row.
	// canStrip is whether the current file has anything for it to
	// remove (Task 2's imaging.CanStripJPEGMetadata) - the button is
	// hidden rather than disabled while it's false, per Florian: no
	// greyed-out button sitting there for a file with nothing to strip.
	strip    *widget.Button
	stripBar *fyne.Container
	canStrip bool

	// pending is the file the open confirmation is about, nil when none
	// is showing. confirm is that dialog itself, so a second request can
	// hide the previous one first (showConfirm's own superseded-dialog
	// guard).
	pending fyne.URI
	confirm dialog.Dialog

	// north is the VBox holding the tag list and stripBar, live only while
	// the window is open. syncStripVisible Refreshes it after Show/Hide:
	// Fyne does not re-run a parent's layout when a child is hidden, so
	// without this a hidden stripBar keeps the height of the last layout
	// (the same trap toggleLocation documents for the map body).
	north *fyne.Container

	// locationMap and location are the OpenStreetMap view and the
	// collapsible section holding it, both live only while the window is
	// open. The section is hidden entirely for a photo with no GPS tags,
	// and starts collapsed otherwise: no tiles are fetched until the user
	// asks to see them.
	locationMap *xwidget.Map
	location    *fyne.Container
	toggle      *widget.Button
	body        *fyne.Container
	loading     *fyne.Container

	// expanded is whether the user has opened the section; lat/lon/hasPos
	// are the position the current image carries, kept so expanding later
	// knows where to point without re-reading the file.
	expanded bool
	lat, lon float64
	hasPos   bool

	// tiles downloads and caches the map's tiles off the UI goroutine -
	// see tiles.go for why the widget's own fetching can't be left to it.
	// warming and warmGen track the prefetch that fills the first view,
	// warm is the completion.Signal tests wait on - see internal/completion.
	tiles   *tileFetcher
	warming bool
	warmGen int
	warm    completion.Signal
}

// New returns the EXIF window for application. host.DisplayedFile is called
// on every open and refresh to find the file to read.
func New(application fyne.App, host Host) *Window {
	w := &Window{
		app:   application,
		host:  host,
		tiles: newTileFetcher(osmTiles, nil),
	}

	// The panel is read against the photo it describes, so it floats above
	// the image window instead of disappearing behind it the moment the
	// user clicks back to navigate.
	w.win.KeepOnTop()
	w.win.SetExtraKeys(w.handleKey)

	return w
}

// handleKey is Left/Right on the EXIF window itself: it steps the displayed
// image the same way the main window's arrows do. A no-op while the Remove
// Metadata confirmation is up - see releaseKeyboard for why the panel stays
// focused then, and stepping the image out from under an open prompt would
// be confusing regardless. This package claims only Left/Right;
// Up/Down/Home/End stay with the main window, where Up/Down also tune the
// slideshow interval.
func (w *Window) handleKey(ev *fyne.KeyEvent) {
	if w.confirm != nil {
		return
	}
	switch ev.Name {
	case fyne.KeyRight:
		w.host.StepImage(1)
	case fyne.KeyLeft:
		w.host.StepImage(-1)
	}
}

// releaseKeyboard returns Left/Right to Singleton's unfocused OnTypedKey
// handler (handleKey above). Fyne delivers TypedKey only to
// Canvas.Focused() when it is non-nil, and a Button click would otherwise
// swallow arrows by keeping itself focused. The Remove Metadata
// confirmation is the exception: its ChoicePanel must stay focused so
// Left/Right move its selection ring rather than reaching the unfocused
// handler and calling StepImage instead.
func (w *Window) releaseKeyboard() {
	if w.confirm != nil {
		return
	}
	if win := w.win.Window(); win != nil {
		win.Canvas().Unfocus()
	}
}

// Show opens the panel, or raises it and syncs it to the current image if
// it's already open. A no-op when nothing is displayed, since there's no
// file to read metadata from.
func (w *Window) Show() {
	if _, ok := w.host.DisplayedFile(); !ok {
		return
	}

	// Raising an already-open window must first sync it to whatever image
	// is now current; Refresh no-ops while the window isn't open yet (text
	// nil), so the fresh-window path below isn't affected.
	w.Refresh()

	w.win.Show(w.app, lang.L("EXIF Data"), fyne.NewSize(exifW, exifH), func() fyne.CanvasObject {
		w.text = widget.NewLabel("")
		w.text.Wrapping = fyne.TextWrapWord

		w.buildLocation()
		w.Refresh()

		w.strip = widget.NewButton(lang.L("Remove Metadata"), w.requestStrip)
		w.strip.Importance = widget.DangerImportance
		w.stripBar = container.NewCenter(w.strip)

		w.north = container.NewVBox(
			container.NewPadded(w.text),
			w.stripBar,
		)
		w.syncStripVisible()

		// Border, not a scrolled box: the metadata and the strip button
		// take the height they need at the top and the map section gets
		// everything below them, so dragging the window taller makes the
		// map taller with it. The button sits above the map, not in the
		// south: a map MinSize of mapH in the center overflows a 420px
		// window and would paint over a south-slot control. Nothing here
		// needs to scroll - the panel's minimum size already covers the
		// longest the metadata gets.
		return container.NewBorder(
			w.north,
			nil, nil, nil,
			w.location,
		)
	}, func() {
		w.tiles.SetOnChange(nil)

		// Anything a prefetch still has in flight belongs to a window that
		// no longer exists.
		w.warmGen++

		w.hideConfirm()
		w.pending = nil

		w.text = nil
		w.strip = nil
		w.stripBar = nil
		w.north = nil
		w.canStrip = false
		w.locationMap = nil
		w.location = nil
		w.toggle = nil
		w.body = nil
		w.loading = nil
		w.expanded = false
		w.warming = false
	})

	w.releaseKeyboard()
}

// Refresh re-reads the current file's raw bytes and updates the panel from
// them. A no-op while the window isn't open. Called both from Show (opening,
// or raising an already-open window onto whatever image is now current) and
// by the app's finishLoad, so navigating to a different image while the
// window is up keeps it in sync instead of showing a stale file's metadata.
//
// Re-reading from disk here rather than keeping the raw bytes from the
// original decode around is a deliberate trade: the image cache only ever
// holds decoded pixels (see its own size comment), and the EXIF window is an
// on-demand, comparatively rare action - not worth doubling every cached
// entry's memory with raw file bytes it usually never needs.
func (w *Window) Refresh() {
	u, ok := w.host.DisplayedFile()
	if w.text == nil || !ok {
		return
	}

	// context.Background(): this is a quick, on-demand, synchronous re-read
	// for the EXIF panel, not part of the cancellable load/preload chain
	// internal/ui's ShowImage/attemptLoad/preloadOne share a generation's
	// context for.
	data, _, err := imaging.ReadAndProbe(context.Background(), u)
	if err != nil {
		w.text.SetText(lang.L("Could not read this file's metadata."))
		w.showLocation(imaging.Metadata{})
		w.canStrip = false
		w.syncStripVisible()
		w.dismissStalePending()
		return
	}

	m := imaging.ReadMetadata(data)

	w.text.SetText(formatExifMetadata(m))
	w.showLocation(m)

	w.canStrip = imaging.CanStripJPEGMetadata(data)
	w.syncStripVisible()
	w.dismissStalePending()
}

// dismissStalePending hides the open confirmation if it no longer matches
// the file now displayed - navigating away from the photo a "Remove
// Metadata?" prompt was about must not leave that prompt answering for
// whatever is on screen now.
func (w *Window) dismissStalePending() {
	if w.pending == nil {
		return
	}

	u, ok := w.host.DisplayedFile()
	if !ok || u == nil || u.String() != w.pending.String() {
		w.hideConfirm()
	}
}

// syncStripVisible shows the Remove Metadata button when the current file
// has anything removable and hides it otherwise. Hidden, not disabled: per
// Florian, no greyed-out button sitting there for a file with nothing to
// strip.
func (w *Window) syncStripVisible() {
	if w.strip == nil {
		return
	}

	if w.canStrip {
		w.strip.Show()
		if w.stripBar != nil {
			w.stripBar.Show()
		}
	} else {
		w.strip.Hide()
		if w.stripBar != nil {
			w.stripBar.Hide()
			// layout.vBoxLayout skips an invisible child entirely - it
			// never calls Resize on one - so a hidden stripBar otherwise
			// keeps reporting whatever height it last had while visible,
			// even after north/content below are told to Refresh.
			w.stripBar.Resize(fyne.NewSize(0, 0))
		}
	}

	// Showing/hiding a child does not re-run its parent's layout, and a
	// hidden child is given no space only on the next layout - without
	// this a hidden stripBar keeps a full-width hole above the map (the
	// same trap toggleLocation documents for the map body).
	if w.north != nil {
		w.north.Refresh()
	}
	if win := w.Window(); win != nil {
		if c := win.Content(); c != nil {
			c.Refresh()
		}
	}
}

// requestStrip opens the "Remove Metadata?" confirmation for the currently
// displayed file. A no-op with nothing removable or nothing displayed - the
// button is hidden in the first case, but this guards the same ground for
// any other caller (a future menu item or shortcut).
func (w *Window) requestStrip() {
	if !w.canStrip {
		return
	}

	u, ok := w.host.DisplayedFile()
	if !ok {
		return
	}

	if w.showConfirm(confirmation{
		title:      lang.L("Remove Metadata?"),
		message:    fmt.Sprintf(lang.L("Remove camera, date, GPS, and other tags from %q? This cannot be undone."), u.Name()),
		action:     lang.L("Remove Metadata"),
		importance: widget.DangerImportance,
		onConfirm:  func() { w.performStrip(u) },
		onCancel:   func() { w.pending = nil },
		onClosed:   func() { w.pending = nil },
	}) == nil {
		return
	}
	// After showConfirm: it hides any previous prompt first, and that Hide
	// fires the old onClosed which nils pending. Assign the new URI last.
	w.pending = u
}

// performStrip rewrites u in place with imaging.StripJPEGMetadata and
// reports the outcome by toast - the only place in this package that
// toasts, so a successful strip never shows twice (once here, once from
// Host.AfterMetadataRemoved, which only re-reads the panel and does not
// toast on its own).
//
// The Refresh here duplicates what the production Host.AfterMetadataRemoved
// (viewer.AfterMetadataRemoved) already does through its own call to
// exif.Refresh() - harmless since Refresh just re-reads the file, and it is
// what keeps this window in sync for any Host whose AfterMetadataRemoved
// does not refresh it itself.
func (w *Window) performStrip(u fyne.URI) {
	w.pending = nil

	if err := imaging.StripJPEGMetadata(u); err != nil {
		fyne.LogError("failed to remove metadata", err)
		w.host.ShowToast(fmt.Sprintf(lang.L("could not remove metadata from %q: %v"), u.Name(), err))
		return
	}

	w.host.AfterMetadataRemoved(u)
	w.Refresh()
	w.host.ShowToast(lang.L("Metadata removed"))
}

// buildLocation assembles the collapsible location section: a disclosure
// button, and under it the map with a loading indicator stacked over it.
//
// It is a hand-rolled disclosure rather than a widget.Accordion because
// expanding is the moment the first tiles may be downloaded, and Accordion
// offers no way to be told when that happens - the whole point of this
// section is that nothing is fetched until the user asks for it.
func (w *Window) buildLocation() {
	w.locationMap = xwidget.NewMapWithOptions(
		xwidget.WithOsmTiles(),
		xwidget.WithTileSource(w.tiles.template),
		xwidget.WithHTTPClient(w.tiles.client()),
		xwidget.WithZoomButtons(true),
		xwidget.WithScrollButtons(false),
		xwidget.AtZoomLevel(mapZoom),
	)

	// A tile that arrives after the frame that asked for it only reaches
	// the screen if the map is told to redraw - see tiles.go. Redrawing
	// once the batch is in, rather than per tile, is what keeps a pan
	// across a dozen new tiles from queueing a dozen repaints of a map
	// that is still mostly holes.
	w.tiles.SetOnChange(func(pending int) {
		if pending > 0 {
			return
		}

		fyne.Do(func() {
			if w.locationMap == nil {
				return
			}

			w.syncLoading()
			w.locationMap.Refresh()
		})
	})

	spinner := widget.NewProgressBarInfinite()
	w.loading = container.NewCenter(container.NewVBox(widget.NewLabel(lang.L("Loading map…")), spinner))
	w.loading.Hide()

	// The map's MinSize is a single tile, so a transparent rectangle
	// stacked behind it gives the section a floor to open at; above that
	// the map grows with the window - see the panel's content layout.
	spacer := canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(0, mapH))

	w.body = container.NewStack(spacer, w.locationMap, w.loading)
	w.body.Hide()

	w.toggle = widget.NewButtonWithIcon(lang.L("Location"), theme.MenuExpandIcon(), w.toggleLocation)
	w.toggle.Alignment = widget.ButtonAlignLeading
	w.toggle.Importance = widget.LowImportance

	w.location = container.NewBorder(w.toggle, nil, nil, nil, w.body)
}

// toggleLocation opens or closes the section. Opening is what starts the
// download of the tiles around the capture position, and what puts the
// loading indicator up until they are all in.
func (w *Window) toggleLocation() {
	w.expanded = !w.expanded

	if w.expanded {
		w.toggle.SetIcon(theme.MenuDropDownIcon())
		w.body.Show()

		// Showing a child doesn't re-run its parent's layout, and a hidden
		// child is given no space at all - without this the map would be
		// revealed at zero height, and so never drawn.
		w.location.Refresh()
		w.startWarm()
		w.releaseKeyboard()

		return
	}

	w.toggle.SetIcon(theme.MenuExpandIcon())
	w.body.Hide()
	w.location.Refresh()
	w.releaseKeyboard()
}

// startWarm downloads the block of tiles around the current position in
// the background, showing the loading indicator until they land. Its own
// generation counter is what keeps a prefetch for an image the user has
// already navigated away from - or for a window they have since closed -
// from touching anything when it finishes.
func (w *Window) startWarm() {
	if !w.hasPos || w.locationMap == nil {
		return
	}

	w.warmGen++
	gen := w.warmGen
	lat, lon := w.lat, w.lon

	done := w.warm.Begin()

	w.warming = true

	// Drawing the map before its tiles are cached would have it ask for
	// every one of them and get nothing (see tiles.go), logging a failure
	// per tile per frame and showing a grid of holes. Keeping it hidden
	// until the block is in trades that for a spinner and one clean frame.
	w.locationMap.Hide()
	w.syncLoading()

	tiles := w.tiles

	go func() {
		tiles.Warm(lat, lon, mapZoom)

		fyne.Do(func() {
			defer done()

			if gen != w.warmGen || w.locationMap == nil {
				return
			}

			w.warming = false
			w.syncLoading()
			w.locationMap.Show()

			// Revealing the map has to re-run the stack's layout for the
			// same reason expanding the section does.
			w.body.Refresh()
		})
	}()
}

// syncLoading shows the indicator while the first view is still being
// prefetched or any tile a pan or zoom asked for is still on its way, and
// hides it once nothing is outstanding.
func (w *Window) syncLoading() {
	if w.loading == nil {
		return
	}

	if w.warming || w.tiles.Pending() > 0 {
		w.loading.Show()
		return
	}

	w.loading.Hide()
}

// showLocation points the map at m's capture position and reveals the
// section holding it, or hides the section entirely when m carries no GPS
// tags - most photos don't, and an empty map of the Atlantic is worse than
// no map at all. The section is left however the user set it: a photo that
// still has a position doesn't re-collapse an expanded map out from under
// them, only a fresh window starts collapsed.
func (w *Window) showLocation(m imaging.Metadata) {
	if w.location == nil {
		return
	}

	w.lat, w.lon, w.hasPos = m.Latitude, m.Longitude, m.HasGPS

	if !m.HasGPS {
		w.location.Hide()
		return
	}

	w.locationMap.SetMarkers([]xwidget.MapMarker{
		xwidget.NewMapMarker(m.Latitude, m.Longitude, lang.L("Photo location")),
	})
	w.locationMap.PanToLatLon(m.Latitude, m.Longitude)
	w.location.Show()

	// An expanded section following the user from image to image needs the
	// new position's tiles, which are usually nowhere near the old ones.
	if w.expanded {
		w.startWarm()
	}
}

// Open reports whether the panel is currently showing.
func (w *Window) Open() bool {
	return w.win.Open()
}

// RestoreGeometry makes the panel remember where and how large it was,
// seeded with what the last run left it at. Called once during internal/ui's
// startup restoration; the app reads the current values back out of
// Geometry at shutdown. Without it the panel opens at exifW x exifH
// wherever the OS puts it, which is what it always did.
func (w *Window) RestoreGeometry(g widgets.Geometry) {
	w.win.Remember(g)
}

// Geometry is where the panel currently is and how large - or where it was
// last, since it outlives the window being closed. What internal/ui hands
// preferences.Save at shutdown.
func (w *Window) Geometry() widgets.Geometry {
	return w.win.Geometry()
}

// StopTracking stops following the panel's position, for a shutdown that
// finds it still open - see widgets.Singleton.StopTracking.
func (w *Window) StopTracking() {
	w.win.StopTracking()
}

// Window returns the open window, or nil when it's closed - the identity
// callers and tests use to tell "raised the same window" from "opened a
// second one".
func (w *Window) Window() fyne.Window {
	return w.win.Window()
}

// Text returns the panel's content label while it's open, or nil - the
// rendered metadata, for callers and tests that need to read it back.
func (w *Window) Text() *widget.Label {
	return w.text
}

// Location returns the collapsible map section while the panel is open, or
// nil - for tests that need to check whether the current image has a
// position to show at all.
func (w *Window) Location() *fyne.Container {
	return w.location
}

// StripButton returns the Remove Metadata button while the panel is open,
// or nil - for tests that need to check whether it is shown and to drive it.
func (w *Window) StripButton() *widget.Button {
	return w.strip
}

// LocationExpanded reports whether the map section is open. False for a
// freshly-opened window, which is what keeps a photo's coordinates off the
// network until the user asks to see them.
func (w *Window) LocationExpanded() bool {
	return w.expanded
}

// ToggleLocation opens or closes the map section, as tapping its header
// does - the entry point for tests, and for any future menu item or key
// that wants to drive it.
func (w *Window) ToggleLocation() {
	if w.toggle == nil {
		return
	}

	w.toggle.OnTapped()
}

// formatExifMetadata renders m as one display line per field that's
// actually set - a file with only some tags (or, for non-JPEG formats,
// none at all) just shows fewer lines rather than a wall of blanks.
func formatExifMetadata(m imaging.Metadata) string {
	// Six decimals is about a tenth of a metre - past anything a camera's
	// GPS resolves, and short enough to read.
	var lat, lon string
	if m.HasGPS {
		lat = fmt.Sprintf("%.6f°", m.Latitude)
		lon = fmt.Sprintf("%.6f°", m.Longitude)
	}

	fields := []struct {
		label, value string
	}{
		{lang.L("Camera"), strings.TrimSpace(m.Make + " " + m.Model)},
		{lang.L("Lens"), m.LensModel},
		{lang.L("Exposure"), m.ExposureTime},
		{lang.L("Aperture"), m.FNumber},
		{lang.L("ISO"), m.ISO},
		{lang.L("Focal length"), m.FocalLength},
		{lang.L("Date taken"), m.DateTaken},
		{lang.L("Latitude"), lat},
		{lang.L("Longitude"), lon},
	}

	var lines []string
	for _, f := range fields {
		if f.value == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: %s", f.label, f.value))
	}

	if len(lines) == 0 {
		return lang.L("No EXIF metadata found in this file.")
	}

	return strings.Join(lines, "\n")
}
