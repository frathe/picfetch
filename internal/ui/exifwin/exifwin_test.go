package exifwin

import (
	"bytes"
	"image/color"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/ui/widgets"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestFormatExifMetadata(t *testing.T) {
	t.Run("every field set", func(t *testing.T) {
		m := imaging.Metadata{
			Make: "Canon", Model: "EOS 90D",
			LensModel:    "EF50mm f/1.8",
			ExposureTime: "1/200 s",
			FNumber:      "f/2.8",
			ISO:          "ISO 400",
			FocalLength:  "50 mm",
			DateTaken:    "2024-08-12 14:33:02",
		}

		want := "Camera: Canon EOS 90D\n" +
			"Lens: EF50mm f/1.8\n" +
			"Exposure: 1/200 s\n" +
			"Aperture: f/2.8\n" +
			"ISO: ISO 400\n" +
			"Focal length: 50 mm\n" +
			"Date taken: 2024-08-12 14:33:02"

		if got := formatExifMetadata(m); got != want {
			t.Errorf("formatExifMetadata() = %q, want %q", got, want)
		}
	})

	t.Run("only some fields set", func(t *testing.T) {
		m := imaging.Metadata{Make: "Canon", ISO: "ISO 400"}

		want := "Camera: Canon\nISO: ISO 400"
		if got := formatExifMetadata(m); got != want {
			t.Errorf("formatExifMetadata() = %q, want %q", got, want)
		}
	})

	t.Run("nothing set", func(t *testing.T) {
		want := "No EXIF metadata found in this file."
		if got := formatExifMetadata(imaging.Metadata{}); got != want {
			t.Errorf("formatExifMetadata() = %q, want %q", got, want)
		}
	})

	t.Run("position set", func(t *testing.T) {
		m := imaging.Metadata{Make: "Canon", Latitude: 48.858222, Longitude: 2.2945, HasGPS: true}

		want := "Camera: Canon\nLatitude: 48.858222°\nLongitude: 2.294500°"
		if got := formatExifMetadata(m); got != want {
			t.Errorf("formatExifMetadata() = %q, want %q", got, want)
		}
	})

	t.Run("southern and western hemispheres keep their sign", func(t *testing.T) {
		m := imaging.Metadata{Latitude: -33.856784, Longitude: -70.664247, HasGPS: true}

		want := "Latitude: -33.856784°\nLongitude: -70.664247°"
		if got := formatExifMetadata(m); got != want {
			t.Errorf("formatExifMetadata() = %q, want %q", got, want)
		}
	})

	// Null Island is a real position, and the only one a zero-valued
	// Metadata could be mistaken for - HasGPS is what tells them apart.
	t.Run("a zero position is still shown when it is a position", func(t *testing.T) {
		want := "Latitude: 0.000000°\nLongitude: 0.000000°"
		if got := formatExifMetadata(imaging.Metadata{HasGPS: true}); got != want {
			t.Errorf("formatExifMetadata() = %q, want %q", got, want)
		}
	})

	t.Run("coordinates are left out without GPS", func(t *testing.T) {
		m := imaging.Metadata{Make: "Canon", Latitude: 48.858222, Longitude: 2.2945}

		want := "Camera: Canon"
		if got := formatExifMetadata(m); got != want {
			t.Errorf("formatExifMetadata() = %q, want %q", got, want)
		}
	})
}

// stubHost is exifwin.Host for tests: current supplies DisplayedFile, and
// toasts/after record what the window did to the rest of the app.
type stubHost struct {
	current func() (fyne.URI, bool)
	toasts  []string
	after   int
	afterU  fyne.URI
}

func (s *stubHost) DisplayedFile() (fyne.URI, bool) {
	if s.current == nil {
		return nil, false
	}
	return s.current()
}
func (s *stubHost) AfterMetadataRemoved(u fyne.URI) {
	s.after++
	s.afterU = u
}
func (s *stubHost) ShowToast(msg string) {
	s.toasts = append(s.toasts, msg)
}

// The panel needs a file to read before it will open at all (Show is a
// no-op with nothing displayed), so every geometry test below hands it one.
func testApp(t *testing.T) (fyne.App, *stubHost) {
	t.Helper()
	app := test.NewApp()
	u := uitest.TempJPEGURI(t, "exif.jpg", 8, 8, color.White)

	return app, &stubHost{current: func() (fyne.URI, bool) { return u, true }}
}

func TestRestoreGeometry_OpensAtTheSavedGeometry(t *testing.T) {
	app, host := testApp(t)
	w := New(app, host)
	w.RestoreGeometry(widgets.Geometry{X: 310, Y: 320, PositionSet: true, Size: fyne.NewSize(520, 480)})

	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	if got, want := w.Window().Canvas().Size(), fyne.NewSize(520, 480); got != want {
		t.Errorf("window size = %v, want the saved %v", got, want)
	}

	got := w.Geometry()
	if !got.PositionSet || got.X != 310 || got.Y != 320 {
		t.Errorf("Geometry() position = (%d, %d, set=%v), want the saved (310, 320, set=true)", got.X, got.Y, got.PositionSet)
	}
}

func TestGeometry_TracksAResizeAndOutlivesTheWindow(t *testing.T) {
	app, host := testApp(t)
	w := New(app, host)
	w.RestoreGeometry(widgets.Geometry{})

	w.Show()
	w.Window().Resize(fyne.NewSize(560, 500))
	w.Window().Close()

	if got, want := w.Geometry().Size, fyne.NewSize(560, 500); got != want {
		t.Errorf("Geometry().Size after closing = %v, want the last tracked %v", got, want)
	}
}

func TestStopTracking_IsSafeWithNoWindowOpen(t *testing.T) {
	app, host := testApp(t)

	New(app, host).StopTracking()
}

func TestShow_WithoutRestoreGeometryUsesTheBuiltInSize(t *testing.T) {
	app, host := testApp(t)
	w := New(app, host)

	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	if got, want := w.Window().Canvas().Size(), fyne.NewSize(exifW, exifH); got != want {
		t.Errorf("window size = %v, want the built-in %v", got, want)
	}
}

// gpsApp is testApp with a photo that carries GPS tags, for the map
// section's tests. The coordinates are the Eiffel Tower's.
func gpsApp(t *testing.T) (fyne.App, *stubHost) {
	t.Helper()
	app := test.NewApp()
	u := uitest.TempGPSJPEGURI(t, "gps.jpg", 8, 8, 48.858222, 2.2945)

	return app, &stubHost{current: func() (fyne.URI, bool) { return u, true }}
}

func TestShow_LocationSectionIsShownCollapsedForAPhotoWithGPS(t *testing.T) {
	app, host := gpsApp(t)
	w := New(app, host)

	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	loc := w.Location()
	if loc == nil {
		t.Fatal("Location() = nil while the window is open")
	}

	if !loc.Visible() {
		t.Error("location section is hidden for a photo that has GPS tags, want shown")
	}

	if w.LocationExpanded() {
		t.Error("location section starts expanded, want collapsed until the user opens it")
	}

	if w.body.Visible() {
		t.Error("map is visible while the section is collapsed, want hidden")
	}
}

func TestShow_LocationSectionIsHiddenWithoutGPS(t *testing.T) {
	app, host := testApp(t) // a plain JPEG, no Exif at all
	w := New(app, host)

	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	if w.Location().Visible() {
		t.Error("location section is shown for a photo with no GPS tags, want hidden")
	}
}

func TestRefresh_LocationSectionFollowsTheCurrentImage(t *testing.T) {
	app := test.NewApp()

	withGPS := uitest.TempGPSJPEGURI(t, "gps.jpg", 8, 8, 48.858222, 2.2945)
	without := uitest.TempJPEGURI(t, "plain.jpg", 8, 8, color.White)

	shown := withGPS
	w := New(app, &stubHost{current: func() (fyne.URI, bool) { return shown, true }})

	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	if !w.Location().Visible() {
		t.Fatal("location section is hidden for the GPS photo, want shown")
	}

	shown = without
	w.Refresh()

	if w.Location().Visible() {
		t.Error("location section stayed visible after navigating to a photo with no GPS, want hidden")
	}

	shown = withGPS
	w.Refresh()

	if !w.Location().Visible() {
		t.Error("location section stayed hidden after navigating back to the GPS photo, want shown")
	}
}

func TestRefresh_LocationSectionIsHiddenForAnUnreadableFile(t *testing.T) {
	app := test.NewApp()

	missing := storage.NewFileURI(filepath.Join(t.TempDir(), "gone.jpg"))
	shown := uitest.TempGPSJPEGURI(t, "gps.jpg", 8, 8, 48.858222, 2.2945)

	w := New(app, &stubHost{current: func() (fyne.URI, bool) { return shown, true }})

	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	shown = missing
	w.Refresh()

	if w.Location().Visible() {
		t.Error("location section stayed visible for an unreadable file, want hidden")
	}

	if got, want := w.Text().Text, "Could not read this file's metadata."; got != want {
		t.Errorf("text = %q, want %q", got, want)
	}
}

// waitForWarm blocks until the prefetch the last expand started has
// finished, so a test can assert on the loading indicator without racing
// it. Deliberately a channel wait rather than polling widget state: the
// Fyne test driver runs fyne.Do inline, so widget state is written from the
// fetching goroutine.
func waitForWarm(t *testing.T, w *Window) {
	t.Helper()

	if w.warmDone == nil {
		t.Fatal("no prefetch has been started")
	}

	select {
	case <-w.warmDone:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the map prefetch")
	}
}

// The tile server is held for every stretch in which the test's own
// goroutine touches widgets: the Fyne test driver runs fyne.Do inline on
// the calling goroutine, so a tile landing mid-assertion would have a
// background goroutine repainting the map while this one reads it.
func TestToggleLocation_ShowsAndHidesTheMap(t *testing.T) {
	app, host := gpsApp(t)

	server := newTileServer(t)
	release := server.hold()

	w := New(app, host)
	w.tiles = fetcherFor(server)

	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	w.ToggleLocation()

	if !w.LocationExpanded() || !w.body.Visible() {
		t.Fatal("map is still hidden after expanding the section, want shown")
	}

	release()
	waitForWarm(t, w)

	w.ToggleLocation()

	if w.LocationExpanded() || w.body.Visible() {
		t.Error("map is still shown after collapsing the section, want hidden")
	}
}

func TestRefresh_IsANoOpWhileTheWindowIsClosed(t *testing.T) {
	app, host := gpsApp(t)
	w := New(app, host)

	w.Refresh() // must not panic on the nil label and nil map

	if w.Location() != nil {
		t.Error("Location() is non-nil with no window open")
	}
}

func TestToggleLocation_ShowsTheLoadingIndicatorUntilTheTilesAreIn(t *testing.T) {
	app, host := gpsApp(t)

	server := newTileServer(t)
	release := server.hold()
	t.Cleanup(release)

	w := New(app, host)
	w.tiles = fetcherFor(server)

	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	if w.loading.Visible() {
		t.Error("loading indicator is up before the section was ever expanded, want hidden")
	}

	w.ToggleLocation()

	if !w.loading.Visible() {
		t.Error("loading indicator is hidden while tiles are still downloading, want shown")
	}

	if w.locationMap.Visible() {
		t.Error("map is drawn while its tiles are still downloading, want it held back until they are in")
	}

	release()
	waitForWarm(t, w)

	if w.loading.Visible() {
		t.Error("loading indicator stayed up after the tiles arrived, want hidden")
	}

	if !w.locationMap.Visible() {
		t.Error("map is still hidden after its tiles arrived, want shown")
	}
}

func TestToggleLocation_FetchesNothingUntilTheSectionIsExpanded(t *testing.T) {
	app, host := gpsApp(t)

	server := newTileServer(t)
	release := server.hold()

	w := New(app, host)
	w.tiles = fetcherFor(server)

	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	w.Refresh()

	if got := server.count(); got != 0 {
		t.Fatalf("server saw %d requests with the section collapsed, want none", got)
	}

	w.ToggleLocation()
	release()
	waitForWarm(t, w)

	if server.count() == 0 {
		t.Error("server saw no requests after expanding the section, want the prefetch")
	}
}

func TestRefresh_ExpandedSectionRefetchesForANewPosition(t *testing.T) {
	app := test.NewApp()

	server := newTileServer(t)

	paris := uitest.TempGPSJPEGURI(t, "paris.jpg", 8, 8, 48.858222, 2.2945)
	sydney := uitest.TempGPSJPEGURI(t, "sydney.jpg", 8, 8, -33.856785, 151.215194)

	release := server.hold()

	shown := paris
	w := New(app, &stubHost{current: func() (fyne.URI, bool) { return shown, true }})
	w.tiles = fetcherFor(server)

	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	w.ToggleLocation()
	release()
	waitForWarm(t, w)

	first := server.count()

	release = server.hold()

	shown = sydney
	w.Refresh()
	release()
	waitForWarm(t, w)

	if server.count() <= first {
		t.Error("navigating to a photo on the other side of the world fetched no new tiles")
	}

	if w.loading.Visible() {
		t.Error("loading indicator stayed up after the second prefetch, want hidden")
	}
}

func TestClose_StopsTheFetcherFromTouchingDeadWidgets(t *testing.T) {
	app, host := gpsApp(t)

	server := newTileServer(t)
	release := server.hold()

	w := New(app, host)
	w.tiles = fetcherFor(server)

	w.Show()
	w.ToggleLocation()
	w.Window().Close()

	// The tiles land after the window is gone: the fetcher must find no
	// callback and the prefetch must find no map, rather than panicking on
	// either.
	release()
	waitForWarm(t, w)

	if w.Location() != nil {
		t.Error("Location() is non-nil after the window closed")
	}
}

func TestPaint_DoesNotBlockOnSlowTiles(t *testing.T) {
	app, host := gpsApp(t)

	server := newTileServer(t)

	w := New(app, host)
	w.tiles = fetcherFor(server)

	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	w.Window().Resize(fyne.NewSize(exifW, exifH))
	w.ToggleLocation()
	waitForWarm(t, w)

	if w.locationMap.Size().IsZero() {
		t.Fatal("expanded map has no size, so this test would not be painting it at all")
	}

	// Panning off the prefetched block is the case that still reaches the
	// network from inside the widget's raster draw - which runs on the UI
	// goroutine, so before this package's tile plumbing existed a frame
	// like this froze the whole app for as long as the server took.
	release := server.hold()
	t.Cleanup(release)

	w.locationMap.PanEast()
	w.locationMap.PanEast()
	w.locationMap.PanEast()

	start := time.Now()
	w.Window().Canvas().Capture()
	elapsed := time.Since(start)

	if elapsed > 2*time.Second {
		t.Errorf("painting the map took %v while the tile server was hanging, want a prompt frame", elapsed)
	}
}

func TestToggleLocation_ExpandedMapGetsRealSpace(t *testing.T) {
	app, host := gpsApp(t)

	server := newTileServer(t)

	w := New(app, host)
	w.tiles = fetcherFor(server)

	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	w.Window().Resize(fyne.NewSize(exifW, exifH))
	w.ToggleLocation()
	waitForWarm(t, w)

	// Revealing a child does not re-run its parent's layout by itself, and
	// a hidden child is given no space: without an explicit refresh the
	// map is "visible" at zero height and never drawn at all.
	if got := w.locationMap.Size(); got.Height < mapH {
		t.Errorf("expanded map size = %v, want at least %v tall", got, mapH)
	}
}

func TestToggleLocation_MapGrowsWithTheWindow(t *testing.T) {
	app, host := gpsApp(t)

	server := newTileServer(t)

	w := New(app, host)
	w.tiles = fetcherFor(server)

	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	w.Window().Resize(fyne.NewSize(exifW, exifH))
	w.ToggleLocation()
	waitForWarm(t, w)

	before := w.locationMap.Size().Height

	w.Window().Resize(fyne.NewSize(exifW, exifH+300))

	// The map fills what the metadata above it leaves, so a taller window
	// is a taller map - the whole extra height, since nothing else in the
	// panel grows.
	if got := w.locationMap.Size().Height; got < before+300 {
		t.Errorf("map height after growing the window by 300 = %v, want at least %v", got, before+300)
	}
}

func TestStripButton_HiddenForAJPEGWithNoMetadata(t *testing.T) {
	app, host := testApp(t) // plain TempJPEGURI
	w := New(app, host)
	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	if w.StripButton() == nil || w.StripButton().Visible() {
		t.Fatal("want the button hidden when nothing is removable")
	}
}

func TestStripButton_ShownForAGPSJPEG(t *testing.T) {
	app, host := gpsApp(t)
	w := New(app, host)
	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	if !w.StripButton().Visible() {
		t.Fatal("want the button shown for GPS JPEG")
	}
}

func TestStripButton_DoesNotSpanTheWindow(t *testing.T) {
	app, host := gpsApp(t)
	w := New(app, host)
	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	w.Window().Resize(fyne.NewSize(exifW, exifH))

	btn := w.StripButton()
	if btn == nil || !btn.Visible() {
		t.Fatal("setup: GPS JPEG should show the button")
	}
	if btn.Importance != widget.DangerImportance {
		t.Errorf("Importance = %v, want widget.DangerImportance", btn.Importance)
	}
	if btn.Size().Width >= exifW*0.8 {
		t.Fatalf("button width %v fills the %v-wide panel; want shrink-wrapped to the label", btn.Size().Width, exifW)
	}
}

func TestStripButton_HiddenForAPNG(t *testing.T) {
	app := test.NewApp()
	u := storage.NewFileURI(uitest.WriteTempFile(t, "plain.png", uitest.EncodePNG(t, 8, 8, color.White)))
	host := &stubHost{current: func() (fyne.URI, bool) { return u, true }}
	w := New(app, host)
	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	if w.StripButton() == nil || w.StripButton().Visible() {
		t.Fatal("want the button hidden for PNG")
	}
}

func TestStripButton_ShownForATrailerOnlyJPEG(t *testing.T) {
	app := test.NewApp()
	data := append(uitest.EncodeJPEG(t, 8, 8, color.White), []byte("ftypmp42fake-video")...)
	u := storage.NewFileURI(uitest.WriteTempFile(t, "trailer.jpg", data))
	host := &stubHost{current: func() (fyne.URI, bool) { return u, true }}
	w := New(app, host)
	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	if w.StripButton() == nil || !w.StripButton().Visible() {
		t.Fatal("want the button shown: bytes after EOI are removable even when the tag list is empty")
	}
	if got := w.Text().Text; got != lang.L("No EXIF metadata found in this file.") && got != "No EXIF metadata found in this file." {
		t.Fatalf("text = %q, want the empty-panel message (ReadMetadata sees no camera tags)", got)
	}
}

func TestStripButton_HiddenBarTakesNoHeightAfterNavigate(t *testing.T) {
	app := test.NewApp()
	gps := uitest.TempGPSJPEGURI(t, "gps.jpg", 8, 8, 48.858222, 2.2945)
	plain := uitest.TempJPEGURI(t, "plain.jpg", 8, 8, color.White)
	shown := gps
	host := &stubHost{current: func() (fyne.URI, bool) { return shown, true }}
	w := New(app, host)
	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	w.Window().Resize(fyne.NewSize(exifW, exifH))

	if w.StripButton() == nil || !w.StripButton().Visible() {
		t.Fatal("setup: GPS JPEG should show the button")
	}
	if w.stripBar == nil || w.stripBar.Size().Height <= 0 {
		t.Fatal("setup: visible stripBar should have height")
	}

	shown = plain
	w.Refresh()

	if w.StripButton().Visible() {
		t.Fatal("want the button hidden after navigating to a JPEG with nothing removable")
	}
	if got := w.stripBar.Size().Height; got != 0 {
		t.Fatalf("hidden stripBar height = %v, want 0 (parent layout must run after Hide)", got)
	}
}

func TestStripButton_GainsHeightAfterNavigateToGPS(t *testing.T) {
	app := test.NewApp()
	gps := uitest.TempGPSJPEGURI(t, "gps.jpg", 8, 8, 48.858222, 2.2945)
	plain := uitest.TempJPEGURI(t, "plain.jpg", 8, 8, color.White)
	shown := plain
	host := &stubHost{current: func() (fyne.URI, bool) { return shown, true }}
	w := New(app, host)
	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	w.Window().Resize(fyne.NewSize(exifW, exifH))

	if w.StripButton() == nil || w.StripButton().Visible() {
		t.Fatal("setup: plain JPEG should hide the button")
	}

	shown = gps
	w.Refresh()

	if !w.StripButton().Visible() {
		t.Fatal("want the button shown after navigating to a GPS JPEG")
	}
	if got := w.stripBar.Size().Height; got <= 0 {
		t.Fatalf("shown stripBar height = %v, want > 0 (parent layout must run after Show)", got)
	}
}

func TestStripButton_SitsAboveTheMap(t *testing.T) {
	app, host := gpsApp(t)
	w := New(app, host)
	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	w.Window().Resize(fyne.NewSize(exifW, exifH))
	w.ToggleLocation()

	root := w.Window().Content()
	btn := w.StripButton()
	loc := w.Location()
	btnPos, ok := absolutePos(root, btn)
	if !ok {
		t.Fatal("Remove Metadata button is not in the window content tree")
	}
	locPos, ok := absolutePos(root, loc)
	if !ok {
		t.Fatal("location section is not in the window content tree")
	}
	btnBottom := btnPos.Y + btn.Size().Height
	if locPos.Y+1 < btnBottom {
		t.Fatalf("location Y=%v sits above button bottom %v: the map would cover Remove Metadata", locPos.Y, btnBottom)
	}
	if btn.Size().Height <= 0 {
		t.Fatal("Remove Metadata button has no laid-out height")
	}
}

// absolutePos is the position of target in root's coordinate space, walking
// nested *fyne.Container parents. Positions on CanvasObject are relative to
// the parent, so comparing StripButton().Position() to Location().Position()
// directly is meaningless - they do not share a parent.
func absolutePos(root, target fyne.CanvasObject) (fyne.Position, bool) {
	var walk func(n fyne.CanvasObject, acc fyne.Position) (fyne.Position, bool)
	walk = func(n fyne.CanvasObject, acc fyne.Position) (fyne.Position, bool) {
		if n == nil {
			return fyne.Position{}, false
		}
		here := acc.Add(n.Position())
		if n == target {
			return here, true
		}
		c, ok := n.(*fyne.Container)
		if !ok {
			return fyne.Position{}, false
		}
		for _, ch := range c.Objects {
			if p, found := walk(ch, here); found {
				return p, true
			}
		}
		return fyne.Position{}, false
	}
	return walk(root, fyne.NewPos(0, 0).Subtract(root.Position()))
}

func TestRequestStrip_CancelLeavesTheFileUnchanged(t *testing.T) {
	app, host := gpsApp(t)
	w := New(app, host)
	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	u, ok := host.DisplayedFile()
	if !ok {
		t.Fatal("setup: DisplayedFile")
	}
	before, err := os.ReadFile(u.Path())
	if err != nil {
		t.Fatalf("read before: %v", err)
	}

	w.StripButton().OnTapped()
	// Escape on the focused panel
	w.Window().Canvas().Focused().(*widgets.ChoicePanel).TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})

	after, err := os.ReadFile(u.Path())
	if err != nil {
		t.Fatalf("read after: %v", err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("Cancel must not write")
	}
	if host.after != 0 {
		t.Fatal("AfterMetadataRemoved must not run")
	}
}

func TestRequestStrip_ConfirmRemovesGPSAndCallsHost(t *testing.T) {
	app, host := gpsApp(t)
	w := New(app, host)
	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	w.StripButton().OnTapped()
	panel := w.Window().Canvas().Focused().(*widgets.ChoicePanel)
	panel.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	panel.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})

	u, ok := host.DisplayedFile()
	if !ok {
		t.Fatal("setup: DisplayedFile")
	}
	data, err := os.ReadFile(u.Path())
	if err != nil {
		t.Fatalf("read stripped file: %v", err)
	}
	if !imaging.ReadMetadata(data).Empty() {
		t.Fatal("want metadata gone")
	}
	if host.after != 1 {
		t.Fatalf("AfterMetadataRemoved calls = %d, want 1", host.after)
	}
	if host.afterU == nil || host.afterU.String() != u.String() {
		t.Fatalf("AfterMetadataRemoved URI = %v, want %v", host.afterU, u)
	}
	if w.Location().Visible() {
		t.Fatal("map section should hide after strip+Refresh")
	}
	if got := w.Text().Text; got != lang.L("No EXIF metadata found in this file.") && got != "No EXIF metadata found in this file." {
		t.Fatalf("text = %q", got)
	}
	if w.StripButton().Visible() {
		t.Fatal("button should hide after a successful strip")
	}
	if w.stripBar != nil && w.stripBar.Size().Height != 0 {
		t.Fatalf("stripBar height after strip = %v, want 0", w.stripBar.Size().Height)
	}
}

func TestRequestStrip_ErrorToastsAndLeavesTheFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory write bits are not Unix-like")
	}
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory write bits")
	}

	dir := t.TempDir()
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })

	path := filepath.Join(locked, "gps.jpg")
	if err := os.WriteFile(path, uitest.GPSJPEG(t, 8, 8, 48.858222, 2.2945), 0o600); err != nil {
		t.Fatal(err)
	}
	u := storage.NewFileURI(path)

	app := test.NewApp()
	host := &stubHost{current: func() (fyne.URI, bool) { return u, true }}
	w := New(app, host)
	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	if err := os.Chmod(locked, 0o500); err != nil {
		t.Fatalf("chmod locked dir: %v", err)
	}

	w.StripButton().OnTapped()
	panel := w.Window().Canvas().Focused().(*widgets.ChoicePanel)
	panel.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	panel.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})

	if host.after != 0 {
		t.Fatal("AfterMetadataRemoved must not run on a failed strip")
	}
	if len(host.toasts) != 1 {
		t.Fatalf("toasts = %v, want one error toast", host.toasts)
	}
	if !strings.Contains(host.toasts[0], "could not remove metadata") {
		t.Fatalf("toast = %q, want the strip-failure message", host.toasts[0])
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("re-read file: %v", err)
	}
	if !imaging.ReadMetadata(data).HasGPS {
		t.Fatal("failed strip must leave GPS in the file")
	}
}

func TestRefresh_DismissesConfirmWhenTheFileChanges(t *testing.T) {
	app := test.NewApp()
	gps := uitest.TempGPSJPEGURI(t, "gps.jpg", 8, 8, 48.858222, 2.2945)
	plain := uitest.TempJPEGURI(t, "plain.jpg", 8, 8, color.White)
	shown := gps
	host := &stubHost{current: func() (fyne.URI, bool) { return shown, true }}

	w := New(app, host)
	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	w.StripButton().OnTapped()
	if n := len(w.Window().Canvas().Overlays().List()); n != 1 {
		t.Fatalf("overlay count = %d after tapping Remove Metadata, want 1", n)
	}

	shown = plain
	w.Refresh()

	if n := len(w.Window().Canvas().Overlays().List()); n != 0 {
		t.Fatalf("overlay count = %d after navigating away, want the confirmation dismissed", n)
	}
}
