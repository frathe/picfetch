package ui

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/storage"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/appearance"
	"github.com/frathe/picfetch/internal/favstore"
	"github.com/frathe/picfetch/internal/filesort"
	"github.com/frathe/picfetch/internal/heic"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/locationtrial"
	"github.com/frathe/picfetch/internal/ui/locationmap"
	"github.com/frathe/picfetch/internal/ui/widgets"
	"github.com/frathe/picfetch/internal/uitest"
)

func locationMenu(t *testing.T, v *viewer) *fyne.MenuItem {
	t.Helper()
	for _, menu := range v.win.MainMenu().Items {
		if menu.Label != lang.L("Window") {
			continue
		}
		for _, item := range menu.Items {
			if item.Label == lang.L("Location Map") {
				return item
			}
		}
	}
	t.Fatal("Window -> Location Map is missing")
	return nil
}

func TestLocationMap(t *testing.T) {
	t.Run("keyboard_navigation", func(t *testing.T) {
		v := newTestViewer(t)
		files := []fyne.URI{
			uitest.PatternedGPSJPEGURI(t, "center.jpg", 1, 64, 48, 0, 0),
			uitest.PatternedGPSJPEGURI(t, "east-a.jpg", 2, 64, 48, 0, 4),
			uitest.PatternedGPSJPEGURI(t, "east-b.jpg", 3, 64, 48, 0, 4),
			uitest.PatternedGPSJPEGURI(t, "west.jpg", 4, 64, 48, 0, -4),
			uitest.PatternedGPSJPEGURI(t, "north.jpg", 5, 64, 48, 4, 0),
			uitest.PatternedGPSJPEGURI(t, "south.jpg", 6, 64, 48, -4, 0),
		}
		dropAndWait(t, v, files...)
		v.win.Resize(fyne.NewSize(1000, 700))
		v.keyModifiers = func() fyne.KeyModifier { return fyne.KeyModifierShift }
		v.win.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyL})
		v.keyModifiers = func() fyne.KeyModifier { return 0 }
		v.locationMap.Settle()
		press := func(key fyne.KeyName) {
			t.Helper()
			if focus := v.win.Canvas().Focused(); focus != nil {
				focus.TypedKey(&fyne.KeyEvent{Name: key})
			} else {
				v.win.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: key})
			}
			v.grid.Settle()
			v.locationMap.Settle()
		}
		selected := func(names ...string) {
			t.Helper()
			marked := false
			explorerWalk(locationPhoto(t, v, names...), func(object fyne.CanvasObject) {
				if frame, ok := object.(*canvas.Rectangle); ok && frame.StrokeWidth == 3 && frame.StrokeColor == theme.Color(theme.ColorNamePrimary) {
					marked = true
				}
			})
			if !marked {
				t.Fatalf("keyboard selection border missing from %v", names)
			}
		}
		center := v.app.Driver().AbsolutePositionForObject(locationPhoto(t, v, "center.jpg"))
		press(fyne.KeyRight)
		selected("center.jpg")
		press(fyne.KeyRight)
		selected("east-a.jpg", "east-b.jpg")
		if got := v.app.Driver().AbsolutePositionForObject(locationPhoto(t, v, "center.jpg")); got != center {
			t.Fatalf("selecting an in-view cluster moved the map: %v -> %v", center, got)
		}
		press(fyne.KeyRight)
		selected("east-a.jpg", "east-b.jpg") // No neighbor in that direction: retain selection.
		press(fyne.KeyReturn)
		if !v.grid.Visible() || v.locationMap.Visible() || len(v.grid.ResultIndexes()) != 2 {
			t.Fatal("Enter did not open the exact selected two-image cluster")
		}
		press(fyne.KeyEscape)
		selected("east-a.jpg", "east-b.jpg")
		press(fyne.KeyLeft)
		selected("center.jpg")
		press(fyne.KeyUp)
		selected("north.jpg")
		press(fyne.KeyDown)
		selected("center.jpg")
		press(fyne.KeyDown)
		selected("south.jpg")
		press(fyne.KeyUp)
		press(fyne.KeyLeft)
		selected("west.jpg")
		press(fyne.KeyEnter)
		waitUntilLoaded(t, v)
		if v.locationMap.Visible() || v.state.files[v.state.index].Name() != "west.jpg" {
			t.Fatal("Enter did not open the selected singleton")
		}
		press(fyne.KeyEscape)
		selected("west.jpg")
		press(fyne.KeyEscape)
		if v.locationMap.Active() {
			t.Fatal("Escape did not leave the map")
		}
	})
	t.Run("keyboard_navigation_offscreen", func(t *testing.T) {
		v := newTestViewer(t)
		dropAndWait(t, v,
			uitest.TempGPSJPEGURI(t, "center.jpg", 24, 16, 0, 0),
			uitest.TempGPSJPEGURI(t, "east.jpg", 24, 16, 0, 4),
			uitest.TempGPSJPEGURI(t, "west.jpg", 24, 16, 0, -4))
		v.win.Resize(fyne.NewSize(1000, 700))
		locationMenu(t, v).Action()
		v.locationMap.Settle()
		press := func(key fyne.KeyName) {
			v.win.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: key})
			v.locationMap.Settle()
		}
		original := locationPhoto(t, v, "east.jpg").Position()
		press(fyne.KeyRight)
		press(fyne.KeyEqual)
		press(fyne.KeyPlus)
		cards := 0
		explorerWalk(locationSurface(t, v), func(object fyne.CanvasObject) {
			if _, ok := object.(*widgets.TappableArea); ok {
				cards++
			}
		})
		if cards != 1 {
			t.Fatalf("zoom fixture should leave only center in view, got %d cards", cards)
		}
		press(fyne.KeyRight)
		photo := locationPhoto(t, v, "east.jpg")
		if photo.Position().X < 0 || photo.Position().X+photo.Size().Width > locationSurface(t, v).Size().Width {
			t.Fatal("keyboard navigation left the selected photo outside the viewport")
		}
		previous := testApp.Settings().Theme()
		t.Cleanup(func() { testApp.Settings().SetTheme(previous) })
		for _, mode := range []appearance.Mode{appearance.Light, appearance.Dark} {
			v.SetThemeMode(mode)
			marked := 0
			explorerWalk(locationSurface(t, v), func(object fyne.CanvasObject) {
				if frame, ok := object.(*canvas.Rectangle); ok && frame.StrokeWidth == 3 {
					marked++
					if frame.StrokeColor != theme.Color(theme.ColorNamePrimary) {
						t.Fatal("keyboard border retained the old theme color")
					}
				}
			})
			if marked != 1 {
				t.Fatalf("expected one keyboard border, got %d", marked)
			}
		}
		press(fyne.KeyMinus)
		press(fyne.Key0)
		if got := locationPhoto(t, v, "east.jpg").Position(); got != original {
			t.Fatalf("keyboard Fit All did not restore framing: %v -> %v", original, got)
		}
		press(fyne.KeyEnter)
		waitUntilLoaded(t, v)
		if v.state.files[v.state.index].Name() != "east.jpg" {
			t.Fatal("zoom/Fit All lost keyboard selection")
		}
	})
	t.Run("keyboard_navigation_recluster", func(t *testing.T) {
		v := newTestViewer(t)
		dropAndWait(t, v,
			uitest.TempGPSJPEGURI(t, "a.jpg", 24, 16, 0, 0),
			uitest.TempGPSJPEGURI(t, "b.jpg", 24, 16, 0, .3),
			uitest.TempGPSJPEGURI(t, "east.jpg", 24, 16, 0, 6),
			uitest.TempGPSJPEGURI(t, "west.jpg", 24, 16, 0, -6))
		v.win.Resize(fyne.NewSize(1000, 700))
		locationMenu(t, v).Action()
		v.locationMap.Settle()
		press := func(key fyne.KeyName) {
			v.win.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: key})
			v.grid.Settle()
			v.locationMap.Settle()
		}
		locationButton(t, v, fmt.Sprintf(lang.L("%d images"), 2))
		press(fyne.KeyRight)
		for range 3 {
			press(fyne.KeyPlus)
		}
		locationPhoto(t, v, "a.jpg")
		locationPhoto(t, v, "b.jpg")
		press(fyne.KeyEnter)
		waitUntilLoaded(t, v)
		if v.locationMap.Visible() || v.grid.Visible() || v.state.files[v.state.index].Name() != "a.jpg" {
			t.Fatal("selection did not follow its source after a cluster split")
		}
		press(fyne.KeyEscape)
		for range 3 {
			press(fyne.KeyMinus)
		}
		press(fyne.KeyEnter)
		if !v.grid.Visible() || len(v.grid.ResultIndexes()) != 2 {
			t.Fatal("Enter retained stale singleton membership after clusters merged")
		}
	})
	t.Run("keyboard_navigation_shift_pan", func(t *testing.T) {
		v := newTestViewer(t)
		dropAndWait(t, v, uitest.TempGPSJPEGURI(t, "selected.jpg", 24, 16, 0, 0))
		v.win.Resize(fyne.NewSize(1000, 700))
		locationMenu(t, v).Action()
		v.locationMap.Settle()
		v.win.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyRight})
		v.keyModifiers = func() fyne.KeyModifier { return fyne.KeyModifierShift }
		for _, move := range []struct {
			key    fyne.KeyName
			dx, dy float32
		}{
			{fyne.KeyRight, -60, 0},
			{fyne.KeyDown, 0, -60},
			{fyne.KeyLeft, 60, 0},
			{fyne.KeyUp, 0, 60},
		} {
			before := locationPhoto(t, v, "selected.jpg").Position()
			v.win.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: move.key})
			v.locationMap.Settle()
			photo := locationPhoto(t, v, "selected.jpg")
			if got := photo.Position().Subtract(before); math.Abs(float64(got.X-move.dx)) > .01 || math.Abs(float64(got.Y-move.dy)) > .01 {
				t.Fatalf("Shift+%s moved the photo by %v, want (%v,%v)", move.key, got, move.dx, move.dy)
			}
			marked := false
			explorerWalk(photo, func(object fyne.CanvasObject) {
				if frame, ok := object.(*canvas.Rectangle); ok && frame.StrokeWidth == 3 {
					marked = true
				}
			})
			if !marked {
				t.Fatal("keyboard panning lost photo selection")
			}
		}
		v.keyModifiers = func() fyne.KeyModifier { return 0 }
		v.win.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyEnter})
		waitUntilLoaded(t, v)
		if v.locationMap.Visible() {
			t.Fatal("Enter no longer opens the selected photo after panning")
		}
	})
	t.Run("dark_filter", func(t *testing.T) {
		previous := testApp.Settings().Theme()
		t.Cleanup(func() { testApp.Settings().SetTheme(previous) })
		testApp.Settings().SetTheme(theme.DefaultTheme())
		v := newTestViewer(t)
		v.SetThemeMode(appearance.Dark)
		tile := image.NewNRGBA(image.Rect(0, 0, 256, 256))
		for y := range 256 {
			for x := range 256 {
				pixel := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
				if x >= 128 {
					pixel = color.NRGBA{A: 255}
				}
				tile.SetNRGBA(x, y, pixel)
			}
		}
		var encoded bytes.Buffer
		if err := png.Encode(&encoded, tile); err != nil {
			t.Fatal(err)
		}
		var calls atomic.Int32
		v.locationMap.ConfigureTiles(locationmap.TileOptions{Client: &http.Client{Transport: locationTileTransport(func(_ *http.Request) (*http.Response, error) {
			calls.Add(1)
			return &http.Response{StatusCode: 200, Header: http.Header{"Cache-Control": []string{"max-age=3600"}}, Body: io.NopCloser(bytes.NewReader(encoded.Bytes()))}, nil
		})}}, noLocationRetry)
		dropAndWait(t, v, uitest.PatternedGPSJPEGURI(t, "unaltered.jpg", 1, 144, 96, 52.52, 13.405))
		locationMenu(t, v).Action()
		v.locationMap.Settle()
		found := 0
		explorerWalk(locationSurface(t, v), func(object fyne.CanvasObject) {
			if img, ok := object.(*canvas.Image); ok && img.Image != nil && img.Image.Bounds().Dx() == 256 {
				found++
				background := color.NRGBAModel.Convert(img.Image.At(0, 0)).(color.NRGBA)
				label := color.NRGBAModel.Convert(img.Image.At(200, 0)).(color.NRGBA)
				if background.R > 64 || background.G > 64 || background.B > 64 || label.R < 180 || label.G < 180 || label.B < 180 {
					t.Fatalf("dark map lacks dark background/light labels: %v / %v", background, label)
				}
			}
		})
		if found == 0 || calls.Load() == 0 {
			t.Fatal("setup did not mount requested map tiles")
		}
		photo := locationPhoto(t, v, "unaltered.jpg")
		var original image.Image
		explorerWalk(photo, func(object fyne.CanvasObject) {
			if img, ok := object.(*canvas.Image); ok {
				original = img.Image
			}
		})
		if original == nil {
			t.Fatal("photo preview missing")
		}
		pixel := original.At(20, 20)
		before := calls.Load()
		locationCapture(t, v, "dark-filter.png")
		for _, mode := range []appearance.Mode{appearance.Light, appearance.Dark, appearance.Light} {
			v.SetThemeMode(mode)
			v.locationMap.Settle()
			want := color.NRGBA{R: 255, G: 255, B: 255, A: 255}
			if mode == appearance.Dark {
				want = color.NRGBA{R: 24, G: 28, B: 34, A: 255}
			}
			locationAssertTileColor(t, v, want)
			capture := v.win.Canvas().Capture()
			for name, y := range map[string]int{"toolbar": 10, "footer": capture.Bounds().Max.Y - 10} {
				background := color.NRGBAModel.Convert(capture.At(capture.Bounds().Max.X-10, y))
				if expected := color.NRGBAModel.Convert(theme.Color(theme.ColorNameBackground)); background != expected {
					t.Errorf("map %s retained the wrong theme background: %v, want %v", name, background, expected)
				}
			}
			explorerWalk(photo, func(object fyne.CanvasObject) {
				if img, ok := object.(*canvas.Image); ok && (img.Image != original || img.Image.At(20, 20) != pixel) {
					t.Fatal("map filter changed the photo preview")
				}
			})
			if calls.Load() != before {
				t.Fatal("appearance change requested new tiles")
			}
		}
		locationCapture(t, v, "light-filter.png")
		t.Run("partial_replacement", func(t *testing.T) {
			v := newTestViewer(t)
			v.SetThemeMode(appearance.Light)
			v.win.Resize(fyne.NewSize(1000, 700))
			oldTile := uitest.EncodePNG(t, 256, 256, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
			newTile := uitest.EncodePNG(t, 256, 256, color.NRGBA{A: 255})
			startReplacement, finishRetry := locationTileRetryFixture(t, v, oldTile, newTile)
			dropAndWait(t, v, uitest.TempGPSJPEGURI(t, "pending.jpg", 24, 16, 52.52, 13.405))
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			startReplacement()
			locationSurface(t, v).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(60, 0)})
			v.locationMap.Settle()
			v.SetThemeMode(appearance.Dark)
			locationAssertTileColor(t, v, color.NRGBA{R: 24, G: 28, B: 34, A: 255})
			finishRetry()
			locationAssertTileColor(t, v, color.NRGBA{R: 215, G: 219, B: 225, A: 255})
		})
	})
	t.Run("photo_presentation", func(t *testing.T) {
		for _, clustered := range []bool{false, true} {
			t.Run(fmt.Sprint(clustered), func(t *testing.T) {
				v := newTestViewer(t)
				a := uitest.PatternedGPSJPEGURI(t, "photo.jpg", 1, 144, 96, 52.52, 13.405)
				sources := []fyne.URI{a}
				if clustered {
					sources = append(sources, uitest.TempGPSJPEGURI(t, "z-neighbor.jpg", 24, 16, 52.52, 13.405))
				}
				dropAndWait(t, v, sources...)
				locationMenu(t, v).Action()
				v.locationMap.Settle()
				photo := locationPhoto(t, v, a.Name())
				var framed, pixels bool
				explorerWalk(photo, func(object fyne.CanvasObject) {
					if r, ok := object.(*canvas.Rectangle); ok && r.StrokeWidth > 0 && r.Shadow.BlurRadius > 0 {
						framed = true
					}
					if img, ok := object.(*canvas.Image); ok && img.Image != nil {
						pixels = true
					}
				})
				if !framed || !pixels {
					t.Fatal("photo lacks decoded preview, frame or shadow")
				}
				explorerWalk(v.locationMap.Overlay(), func(object fyne.CanvasObject) {
					if button, ok := object.(*widget.Button); ok && (button.Text == a.Name() || button.Text == lang.L("Open Image")) {
						t.Fatal("filename caption or sidebar still mounted")
					}
				})
				photo.MouseIn(&desktop.MouseEvent{})
				if !locationFilenameVisible(t, v, a.Name()) {
					t.Fatal("hover did not show filename")
				}
				explorerWalk(locationSurface(t, v), func(object fyne.CanvasObject) {
					if label, ok := object.(*widget.Label); ok && label.Visible() && label.Text == a.Name() && label.Size().Width < fyne.MeasureText(a.Name(), theme.TextSize(), label.TextStyle).Width {
						t.Fatal("tooltip truncates an ordinary filename")
					}
				})
				photo.MouseOut()
				if locationFilenameVisible(t, v, a.Name()) {
					t.Fatal("filename remained after hover")
				}
				if clustered {
					pin := locationButton(t, v, fmt.Sprintf(lang.L("%d images"), 2))
					if v.app.Driver().AbsolutePositionForObject(photo).Y >= v.app.Driver().AbsolutePositionForObject(pin).Y {
						t.Fatal("cluster preview is not above count")
					}
				}
				var before image.Image
				explorerWalk(photo, func(object fyne.CanvasObject) {
					if img, ok := object.(*canvas.Image); ok {
						before = img.Image
					}
				})
				for range 3 {
					locationSurface(t, v).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(20, 10)})
					photo = locationPhoto(t, v, a.Name())
					explorerWalk(photo, func(object fyne.CanvasObject) {
						if img, ok := object.(*canvas.Image); ok && img.Image != before {
							t.Fatal("drag discarded an already-painted photo")
						}
					})
				}
				locationCapture(t, v, fmt.Sprintf("photo-%t.png", clustered))
				photo.MouseIn(&desktop.MouseEvent{})
				locationCapture(t, v, fmt.Sprintf("tooltip-%t.png", clustered))
				fynetest.Tap(photo)
				if clustered {
					v.grid.Settle()
					if v.locationMap.Visible() || !v.grid.Visible() || !slices.Equal(v.grid.ResultIndexes(), []int{0, 1}) {
						t.Fatal("cluster preview did not open its members in Grid")
					}
				} else {
					waitUntilLoaded(t, v)
					if v.locationMap.Visible() || v.grid.Visible() || v.state.index != 0 {
						t.Fatal("singleton preview did not open its image directly")
					}
				}
			})
		}
	})
	t.Run("photo_direct_open", func(t *testing.T) {
		v := newTestViewer(t)
		source := uitest.TempGPSJPEGURI(t, "direct.jpg", 24, 16, 52.52, 13.405)
		dropAndWait(t, v, source)
		locationMenu(t, v).Action()
		v.locationMap.Settle()
		var photo *widgets.TappableArea
		explorerWalk(locationSurface(t, v), func(object fyne.CanvasObject) {
			if area, ok := object.(*widgets.TappableArea); ok {
				photo = area
			}
		})
		if photo == nil {
			t.Fatal("no tappable photo in map")
		}
		fynetest.Tap(photo)
		if v.locationMap.Visible() {
			t.Fatal("photo click opened a sidebar instead of the image")
		}
		waitUntilLoaded(t, v)
		if current, _, _ := v.CurrentFile(); current.String() != source.String() {
			t.Fatal("photo click opened another source")
		}
	})
	t.Run("native_trial_observations", func(t *testing.T) {
		v := newTestViewer(t)
		dir := filepath.Join(t.TempDir(), "native")
		if err := v.configureLocationTrial(dir); err != nil {
			t.Fatal(err)
		}
		dropAndWait(t, v, uitest.TempGPSJPEGURI(t, "trial.jpg", 24, 16, 52.52, 13.405))
		for range 2 {
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		}
		v.stopLocationTrial()
		if err := v.waitLocationTrial(); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(dir, "state.json"))
		if err != nil {
			t.Fatal(err)
		}
		var state locationtrial.State
		if err := json.Unmarshal(data, &state); err != nil {
			t.Fatal(err)
		}
		if state.Images != 1 || state.Formats["jpg"] != 1 || !state.Ready || state.Active || len(state.Stages) != 2 {
			t.Fatalf("incorrect observed admission/session: %+v", state)
		}
		for i, stage := range state.Stages {
			if stage.Kind != []string{"cold", "warm"}[i] || !stage.Complete || stage.EndNS <= stage.StartNS || stage.PreparationNS+stage.ScanNS != stage.EndNS-stage.StartNS {
				t.Fatalf("incoherent stage: %+v", stage)
			}
		}
	})
	t.Run("direct_visit_mode_admission", func(t *testing.T) {
		v := newTestViewer(t)
		source := uitest.TempGPSJPEGURI(t, "mode.jpg", 24, 16, 52.52, 13.405)
		dropAndWait(t, v, source)
		locationMenu(t, v).Action()
		v.locationMap.Settle()
		fynetest.Tap(locationPhoto(t, v, source.Name()))
		waitUntilLoaded(t, v)
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyP})
		if v.slides.Active() {
			t.Fatal("picture frame claimed map-origin Escape")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyG})
		v.grid.Settle()
		if !v.grid.Visible() || v.locationMap.Active() {
			t.Fatal("ordinary Grid retained hidden mapped-image navigation")
		}
	})
	t.Run("metadata_coverage", func(t *testing.T) {
		t.Run("missing_and_incomplete_coordinates", func(t *testing.T) {
			avif, err := os.ReadFile("../imaging/testdata/test_exif.avif")
			if err != nil {
				t.Fatal(err)
			}
			incomplete := uitest.GPSMetadataTIFF(t, 52.52, 13.405, "")
			gps := int(binary.LittleEndian.Uint32(incomplete[18:22]))
			// Keep the actual TIFF sub-IFD but omit longitude entries.
			binary.LittleEndian.PutUint16(incomplete[gps:gps+2], 2)
			for name, data := range map[string][]byte{
				"missing.avif":   avif,
				"malformed.png":  uitest.PNGWithEXIF(t, []byte("not a TIFF payload")),
				"incomplete.png": uitest.PNGWithEXIF(t, incomplete),
			} {
				t.Run(name, func(t *testing.T) {
					v := newTestViewer(t)
					source := storage.NewFileURI(uitest.WriteTempFile(t, name, data))
					dropAndWait(t, v, source)
					locationMenu(t, v).Action()
					v.locationMap.Settle()
					if counts := v.locationMap.Counts(); !counts.Complete || counts.Unlocated != 1 || counts.Failed != 0 || len(v.locationMap.Points()) != 0 {
						t.Fatalf("missing/malformed GPS became location or operational failure: %+v", counts)
					}
				})
			}
		})
		t.Run("system_heic_unavailable", func(t *testing.T) {
			v := newTestViewer(t)
			v.configureHEIC(unavailableHEICBackend{})
			v.startHEICCheck(false)
			v.settleHEIC()
			data, err := os.ReadFile("../imaging/testdata/test_exif.heic")
			if err != nil {
				t.Fatal(err)
			}
			source := storage.NewFileURI(uitest.WriteTempFile(t, "unavailable.heic", data))
			jpeg := uitest.TempGPSJPEGURI(t, "admitted.jpg", 24, 16, 0, 0)
			dropAndWait(t, v, source, jpeg)
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			if counts := v.locationMap.Counts(); v.FileCount() != 1 || counts.Located != 1 || counts.Total != 1 {
				t.Fatalf("map invented admission for unavailable HEIC: %+v", counts)
			}
		})
		t.Run("shared_formats", func(t *testing.T) {
			base, err := os.ReadFile("../../assets/ui-originals/placeholder.webp")
			if err != nil {
				t.Fatal(err)
			}
			metadata := uitest.GPSMetadataTIFF(t, 0, 0, "2024:07:08 09:10:11")
			fixtures := map[string][]byte{
				"jpeg.jpg":  uitest.GPSJPEG(t, 24, 16, 0, 0),
				"png.png":   uitest.PNGWithEXIF(t, metadata),
				"webp.webp": uitest.WebPWithEXIF(t, base, metadata),
				"raw.arw":   uitest.EncodeRAWPreview(t, uitest.RAWPreview{Width: 24, Height: 16, GPS: &[2]float64{0, 0}}),
			}
			for name, data := range fixtures {
				t.Run(name, func(t *testing.T) {
					v := newTestViewer(t)
					source := storage.NewFileURI(uitest.WriteTempFile(t, name, data))
					dropAndWait(t, v, source)
					locationMenu(t, v).Action()
					v.locationMap.Settle()
					points := v.locationMap.Points()
					if len(points) != 1 || !points[0].Metadata.HasGPS || points[0].Metadata.Latitude != 0 || points[0].Metadata.Longitude != 0 {
						t.Fatalf("shared explicit-zero metadata did not reach map: %+v", points)
					}
					read, err := imaging.ReadMetadataURIContext(context.Background(), source)
					if err != nil || read != points[0].Metadata {
						t.Fatalf("map/shared metadata disagree: %v %+v", err, read)
					}
					after, err := os.ReadFile(source.Path())
					if err != nil || !bytes.Equal(data, after) {
						t.Fatal("metadata browsing modified source")
					}
				})
			}
		})
		t.Run("system_heic", func(t *testing.T) {
			v := newTestViewer(t)
			var fail atomic.Bool
			exif := uitest.GPSMetadataTIFF(t, 0, 0, "")
			v.configureHEIC(testHEICBackend{check: func(_ context.Context) error { return nil }, read: func(_ context.Context, _ []byte, request heic.Request) (heic.Result, error) {
				if fail.Load() {
					return heic.Result{}, errors.New("controlled native failure")
				}
				result := heic.Result{Width: 2, Height: 1, EXIF: exif}
				if request.Pixels {
					result.Stride = 8
					result.Pixels = []byte{255, 0, 0, 255, 0, 255, 0, 255}
				}
				return result, nil
			}})
			v.startHEICCheck(false)
			v.settleHEIC()
			data, err := os.ReadFile("../imaging/testdata/test_exif.heic")
			if err != nil {
				t.Fatal(err)
			}
			source := storage.NewFileURI(uitest.WriteTempFile(t, "native.heic", data))
			dropAndWait(t, v, source)
			v.display.Settle()
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			if len(v.locationMap.Points()) != 1 {
				t.Fatal("captured system metadata did not reach map")
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			v.locationMap.InvalidateSources([]fyne.URI{source})
			fail.Store(true)
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			if v.locationMap.Counts().Failed != 1 || v.locationMap.Counts().Unlocated != 0 {
				t.Fatalf("operational HEIC error became no-GPS fact: %+v", v.locationMap.Counts())
			}
		})
	})
	t.Run("gps_cache_lifecycle", func(t *testing.T) {
		t.Run("corrupt_record", func(t *testing.T) {
			v := newTestViewer(t)
			data := uitest.EncodeJPEG(t, 24, 16, color.White)
			base := storage.NewFileURI(uitest.WriteTempFile(t, "corrupt-cache.jpg", data))
			dir := storeFavorite(t, v, "Cached", base)
			dropAndWait(t, v, base)
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			records := locationGPSRecords(t, dir)
			if len(records) != 1 {
				t.Fatal("setup cache record absent")
			}
			if err := os.WriteFile(records[0], []byte("broken JSON"), 0o600); err != nil {
				t.Fatal(err)
			}
			fresh := newTestViewer(t)
			fresh.favorites.SetDir(filepath.Dir(dir))
			var reads atomic.Int32
			source := uitest.ReaderURI(base, func() (io.ReadCloser, error) { reads.Add(1); return io.NopCloser(bytes.NewReader(data)), nil })
			dropAndWait(t, fresh, source)
			fresh.display.Settle()
			reads.Store(0)
			locationMenu(t, fresh).Action()
			fresh.locationMap.Settle()
			if reads.Load() != 1 || fresh.locationMap.Counts().Unlocated != 1 {
				t.Fatalf("corrupt record was not a recoverable miss: reads=%d counts=%+v", reads.Load(), fresh.locationMap.Counts())
			}
		})
		t.Run("write_failure_warns_once", func(t *testing.T) {
			v := newTestViewer(t)
			source := uitest.TempJPEGURI(t, "memory-only.jpg", 24, 16, color.White)
			dir := storeFavorite(t, v, "Unwritable", source)
			if err := os.WriteFile(filepath.Join(dir, ".location-map"), []byte("cache directory obstructed"), 0o600); err != nil {
				t.Fatal(err)
			}
			pixels := uitest.EncodePNG(t, 256, 256, color.Black)
			v.locationMap.ConfigureTiles(locationmap.TileOptions{Client: &http.Client{Transport: locationTileTransport(func(_ *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(pixels))}, nil
			})}}, noLocationRetry)
			dropAndWait(t, v, source)
			before := v.toast.gen.Load()
			for range 2 {
				locationMenu(t, v).Action()
				v.locationMap.Settle()
				if v.locationMap.Counts().Unlocated != 1 || len(v.locationMap.RawFacts()) != 1 {
					t.Fatal("cache failure stopped memory browsing")
				}
				v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			}
			if v.toast.gen.Load() != before+1 || v.toast.text.Text != lang.L("Could not cache Favorite locations. Locations remain available in memory.") {
				t.Fatal("cache failure warning missing or repeated")
			}
		})
		for _, replace := range []bool{false, true} {
			t.Run(map[bool]string{false: "held_removal", true: "held_replacement"}[replace], func(t *testing.T) {
				v := newTestViewer(t)
				data := uitest.EncodeJPEG(t, 24, 16, color.White)
				base := storage.NewFileURI(uitest.WriteTempFile(t, "owned.jpg", data))
				entered, release := make(chan struct{}), make(chan struct{})
				var hold atomic.Bool
				var once sync.Once
				t.Cleanup(func() { once.Do(func() { close(release) }) })
				source := uitest.ReaderURI(base, func() (io.ReadCloser, error) {
					if hold.CompareAndSwap(true, false) {
						close(entered)
						<-release
					}
					return io.NopCloser(bytes.NewReader(data)), nil
				})
				dropAndWait(t, v, source)
				v.display.Settle()
				dir := storeFavorite(t, v, "Owned", base)
				hold.Store(true)
				locationMenu(t, v).Action()
				<-entered
				if replace {
					other := uitest.TempJPEGURI(t, "replacement.jpg", 24, 16, color.Black)
					if err := favstore.Save(filepath.Dir(dir), "Owned", []fyne.URI{other}); err != nil {
						t.Fatal(err)
					}
				} else if err := os.Rename(dir, filepath.Join(t.TempDir(), "retired")); err != nil {
					t.Fatal(err)
				}
				once.Do(func() { close(release) })
				v.locationMap.Settle()
				if len(locationGPSRecords(t, dir)) != 0 {
					t.Fatal("retired producer published into removed/replaced Favorite")
				}
				if !replace {
					if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
						t.Fatal("removed Favorite recreated")
					}
				}
				if v.FileCount() != 1 || len(v.locationMap.RawFacts()) != 1 {
					t.Fatal("retiring Favorite discarded loaded source or memory fact")
				}
			})
		}
	})
	t.Run("automatic_rebuild", func(t *testing.T) {
		t.Run("inactive_write_invalidates_persistence", func(t *testing.T) {
			v := newTestViewer(t)
			source := uitest.TempGPSJPEGURI(t, "saved-write.jpg", 24, 16, 52.52, 13.405)
			dropAndWait(t, v, source)
			dir := storeFavorite(t, v, "Written", source)
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			if len(locationGPSRecords(t, dir)) != 1 {
				t.Fatal("missing initial persisted fact")
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			result, err := imaging.StripJPEGMetadataContext(context.Background(), source)
			if err != nil || !result.Committed {
				t.Fatalf("write did not commit: %v", err)
			}
			v.afterFileWrite(result, false, false, func() {})
			drainFileWork(t, v)
			v.locationMap.Settle()
			if len(locationGPSRecords(t, dir)) != 0 {
				t.Fatal("inactive committed write retained persisted GPS fact")
			}
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			if v.locationMap.Counts().Unlocated != 1 || len(locationGPSRecords(t, dir)) != 1 {
				t.Fatal("fresh raw absence was not republished")
			}
		})
		t.Run("hidden_donor_and_alias_write", func(t *testing.T) {
			v := newTestViewer(t)
			donor := uitest.PatternedGPSJPEGURI(t, "donor.jpg", 1, 64, 48, 52.52, 13.405)
			rep := uitest.PatternedJPEGURISize(t, "representative.jpg", 1, 192, 144)
			aliasPath := filepath.Join(t.TempDir(), "alias.jpg")
			if err := os.Symlink(donor.Path(), aliasPath); err != nil {
				t.Fatal(err)
			}
			alias := storage.NewFileURI(aliasPath)
			dropAndWait(t, v, donor, alias, rep)
			v.menus.Actions().Hide().Action()
			v.grid.Settle()
			locationMenu(t, v).Action()
			v.grid.Settle()
			v.locationMap.Settle()
			if len(v.locationMap.Points()) != 1 || v.locationMap.Points()[0].Donor.URI == nil {
				t.Fatal("setup did not borrow hidden location")
			}
			result, err := imaging.StripJPEGMetadataContext(context.Background(), alias)
			if err != nil || !result.Committed {
				t.Fatalf("metadata removal did not commit: %v", err)
			}
			v.afterFileWrite(result, false, false, func() {})
			drainFileWork(t, v)
			v.grid.Settle()
			v.locationMap.Settle()
			if len(v.locationMap.Points()) != 0 || !v.locationMap.Counts().Complete || v.locationMap.Counts().Unlocated != 1 {
				t.Fatalf("hidden donor or alias retained old GPS: %+v", v.locationMap.Counts())
			}
			for _, fact := range v.locationMap.RawFacts() {
				if fact.Metadata.HasGPS {
					t.Fatal("old raw donor fact survived committed alias write")
				}
			}
		})
		t.Run("frozen_visit_survivors", func(t *testing.T) {
			v := newTestViewer(t)
			v.win.Resize(fyne.NewSize(1000, 700))
			a := uitest.TempGPSJPEGURI(t, "same.jpg", 24, 16, 52.52, 13.405)
			b := uitest.TempGPSJPEGURI(t, "outside.jpg", 24, 16, 51.507, -.128)
			dropAndWait(t, v, a, b)
			v.state.SetMergeMode(true)
			dropAndWait(t, v, a)
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			fynetest.Tap(locationButton(t, v, fmt.Sprintf(lang.L("%d images"), 2)))
			v.grid.Settle()
			members := v.grid.ResultIndexes()
			if len(members) != 2 {
				t.Fatalf("setup cluster: %v", members)
			}
			v.grid.SimulateHover(1)
			v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
			v.RemoveFile(members[0])
			v.grid.Settle()
			v.locationMap.Settle()
			if !v.grid.Visible() || len(v.grid.ResultIndexes()) != 1 || len(v.grid.Selection()) != 1 {
				t.Fatalf("frozen duplicate occurrence lost survivor/selection: results=%v selection=%v", v.grid.ResultIndexes(), v.grid.Selection())
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
			waitUntilLoaded(t, v)
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
			waitUntilLoaded(t, v)
			if v.FileAt(v.state.index).Path() != a.Path() {
				t.Fatal("surviving frozen visit escaped to newly adjacent image")
			}
			v.RemoveFile(v.state.index)
			v.grid.Settle()
			v.locationMap.Settle()
			if !v.locationMap.Visible() || v.grid.Visible() {
				t.Fatal("exhausted frozen visit did not return to map")
			}
		})
		t.Run("committed_source_changes", func(t *testing.T) {
			v := newTestViewer(t)
			a := uitest.TempGPSJPEGURI(t, "a.jpg", 24, 16, 52.52, 13.405)
			b := uitest.TempGPSJPEGURI(t, "b.jpg", 24, 16, 51.507, -.128)
			dropAndWait(t, v, a, b)
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			v.RemoveFile(0)
			v.grid.Settle()
			v.locationMap.Settle()
			points := v.locationMap.Points()
			if len(points) != 1 || points[0].Source.URI.Path() != b.Path() || v.locationMap.Counts().Total != 1 {
				t.Fatalf("deletion left stale map: %+v", points)
			}
		})
		t.Run("sorting", func(t *testing.T) {
			v := newTestViewer(t)
			a := uitest.TempGPSJPEGURI(t, "a.jpg", 24, 16, 52.52, 13.405)
			b := uitest.TempGPSJPEGURI(t, "b.jpg", 24, 16, 51.507, -.128)
			dropAndWait(t, v, b, a)
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			v.SetSortMode(filesort.ByDropOrder)
			waitForSort(t, v)
			v.grid.Settle()
			v.locationMap.Settle()
			if !v.locationMap.Visible() || len(v.locationMap.Points()) != 2 || v.locationMap.Points()[0].Source.URI.Path() != b.Path() {
				t.Fatal("sort failed to preserve and reorder map")
			}
		})
	})
	t.Run("external_changes", func(t *testing.T) {
		t.Run("browsing_routes", func(t *testing.T) {
			for _, route := range []string{"entry", "direct", "cluster"} {
				t.Run(route, func(t *testing.T) {
					v := newTestViewer(t)
					v.win.Resize(fyne.NewSize(1000, 700))
					donor := uitest.PatternedGPSJPEGURI(t, "a-donor.jpg", 1, 64, 48, 52.52, 13.405)
					rep := uitest.PatternedJPEGURISize(t, "b-representative.jpg", 1, 192, 144)
					anchor := uitest.PatternedGPSJPEGURI(t, "z-anchor.jpg", 47, 64, 48, 35.68, 139.69)
					sources := []fyne.URI{donor, rep, anchor}
					if route == "cluster" {
						sources = append(sources, uitest.PatternedGPSJPEGURI(t, "c-colocated.jpg", 93, 64, 48, 52.52, 13.405))
					}
					dropAndWait(t, v, sources...)
					v.menus.Actions().Hide().Action()
					v.grid.Settle()
					locationMenu(t, v).Action()
					v.grid.Settle()
					v.locationMap.Settle()
					locationSurface(t, v).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(35, 20)})
					position := v.app.Driver().AbsolutePositionForObject(locationPhoto(t, v, anchor.Name()))
					if route == "entry" {
						v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
					} else {
						fynetest.Tap(locationPhoto(t, v, rep.Name()))
						if route == "cluster" {
							v.grid.Settle()
							if len(v.grid.ResultIndexes()) != 2 {
								t.Fatal("fixture did not enter a two-member cluster")
							}
						} else {
							waitUntilLoaded(t, v)
						}
					}
					changed := uitest.PatternedGPSJPEGURI(t, "changed.jpg", 1, 64, 48, 40.7, -74)
					data, err := os.ReadFile(changed.Path())
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(donor.Path(), data, 0o600); err != nil {
						t.Fatal(err)
					}
					if route == "entry" {
						locationMenu(t, v).Action()
					} else {
						v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
					}
					// Validation delivers a root rebuild which may start duplicate work.
					v.locationMap.Settle()
					v.grid.Settle()
					v.locationMap.Settle()
					found := false
					for _, point := range v.locationMap.Points() {
						if point.Source.URI.Path() == rep.Path() {
							found = true
							if math.Abs(point.Metadata.Latitude-40.7) > .000001 || point.Donor.URI == nil || point.Donor.URI.Path() != donor.Path() {
								t.Fatalf("hidden donor change not reflected: %+v", point)
							}
						}
					}
					if !found || !v.locationMap.Visible() {
						t.Fatal("return lost the map or borrowed representative")
					}
					if route != "entry" {
						if got := v.app.Driver().AbsolutePositionForObject(locationPhoto(t, v, anchor.Name())); got != position {
							t.Fatalf("external refresh moved the retained camera: %v -> %v", position, got)
						}
					}
				})
			}
		})
		t.Run("stale_validation", func(t *testing.T) {
			v := newTestViewer(t)
			a := uitest.TempGPSJPEGURI(t, "old.jpg", 24, 16, 52.52, 13.405)
			dropAndWait(t, v, a)
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			fynetest.Tap(locationPhoto(t, v, a.Name()))
			waitUntilLoaded(t, v)
			if err := os.WriteFile(a.Path(), uitest.GPSJPEG(t, 24, 16, 40.7, -74), 0o600); err != nil {
				t.Fatal(err)
			}
			oldQueue := &uitest.UIQueue{}
			v.locationMap.SetUIQueue(oldQueue)
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			v.locationMap.Wait() // validation has submitted, but its UI delivery is held
			b := uitest.TempGPSJPEGURI(t, "new.jpg", 24, 16, 0, 0)
			dropAndWait(t, v, b)
			v.locationMap.SetUIQueue(&uitest.UIQueue{})
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			oldQueue.Drain()
			v.grid.Settle()
			v.locationMap.Settle()
			if len(v.locationMap.Points()) != 1 || v.locationMap.Points()[0].Source.URI.Path() != b.Path() {
				t.Fatal("retired validation replaced newer collection")
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			oldQueue.Drain()
			if v.locationMap.Active() {
				t.Fatal("retired validation reopened closed feature")
			}
		})
		t.Run("entry_and_return", func(t *testing.T) {
			v := newTestViewer(t)
			a := uitest.TempGPSJPEGURI(t, "changed.jpg", 24, 16, 52.52, 13.405)
			dropAndWait(t, v, a)
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			fynetest.Tap(locationPhoto(t, v, a.Name()))
			waitUntilLoaded(t, v)
			if err := os.WriteFile(a.Path(), uitest.GPSJPEG(t, 24, 16, 51.507, -.128), 0o600); err != nil {
				t.Fatal(err)
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			v.grid.Settle()
			v.locationMap.Settle()
			points := v.locationMap.Points()
			if len(points) != 1 || math.Abs(points[0].Metadata.Latitude-51.507) > .000001 {
				t.Fatalf("return reused externally stale position: %+v", points)
			}
		})
	})
	t.Run("geographic_clusters", func(t *testing.T) {
		v := newTestViewer(t)
		v.win.Resize(fyne.NewSize(1000, 700))
		east := uitest.TempGPSJPEGURI(t, "east.jpg", 24, 16, 0, 180)
		west := uitest.TempGPSJPEGURI(t, "west.jpg", 24, 16, 0, -180)
		near := uitest.TempGPSJPEGURI(t, "near.jpg", 24, 16, 0, 179)
		invalid := uitest.TempGPSJPEGURI(t, "invalid.jpg", 24, 16, 91, 0)
		dropAndWait(t, v, east, west, near, invalid)
		locationMenu(t, v).Action()
		v.locationMap.Settle()
		if v.locationMap.Counts().Located != 3 || v.locationMap.Counts().Unlocated != 1 {
			t.Fatalf("coordinate validation: %+v", v.locationMap.Counts())
		}
		locationPhoto(t, v, near.Name())
		pin := locationButton(t, v, fmt.Sprintf(lang.L("%d images"), 2))
		if pin.Importance != widget.HighImportance {
			t.Fatal("displayed boundary image did not highlight its containing cluster")
		}
		fynetest.Tap(pin)
		v.grid.Settle()
		var paths []string
		for _, i := range v.grid.ResultIndexes() {
			paths = append(paths, v.FileAt(i).Path())
		}
		if len(paths) != 2 || !slices.Contains(paths, east.Path()) || !slices.Contains(paths, west.Path()) {
			t.Fatalf("dateline cluster included enclosing region rather than exact membership: %v", paths)
		}
	})
	t.Run("gps_cache_policy", func(t *testing.T) {
		t.Run("live_ownership_and_errors", func(t *testing.T) {
			v := newTestViewer(t)
			root := t.TempDir()
			v.favorites.SetDir(root)
			data := uitest.EncodeJPEG(t, 24, 16, color.White)
			var fail atomic.Bool
			var reads [2]atomic.Int32
			sources := make([]fyne.URI, 2)
			for i, name := range []string{"a-valid-absence.jpg", "b-transient-error.jpg"} {
				base := storage.NewFileURI(uitest.WriteTempFile(t, name, data))
				sources[i] = uitest.ReaderURI(base, func() (io.ReadCloser, error) {
					reads[i].Add(1)
					if i == 1 && fail.Load() {
						return nil, errors.New("controlled transient read failure")
					}
					return io.NopCloser(bytes.NewReader(data)), nil
				})
			}
			dropAndWait(t, v, sources...)
			v.display.Settle()
			for i := range reads {
				reads[i].Store(0)
			}
			fail.Store(true)
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			if counts := v.locationMap.Counts(); !counts.Complete || counts.Unlocated != 1 || counts.Failed != 1 {
				t.Fatalf("read failure was confused with successful absence: %+v", counts)
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			fail.Store(false)
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			if counts := v.locationMap.Counts(); counts.Unlocated != 2 || counts.Failed != 0 || reads[0].Load() != 1 || reads[1].Load() != 2 {
				t.Fatalf("retry/reuse mismatch: reads=%d/%d counts=%+v", reads[0].Load(), reads[1].Load(), counts)
			}
			entries, err := os.ReadDir(root)
			if err != nil || len(entries) != 0 {
				t.Fatalf("unsaved map created persistent records: %v %v", entries, err)
			}
		})
		t.Run("save_promotion_and_unsaved_merge", func(t *testing.T) {
			v := newTestViewer(t)
			a := uitest.TempJPEGURI(t, "member.jpg", 24, 16, color.White)
			b := uitest.TempJPEGURI(t, "unsaved.jpg", 24, 16, color.Black)
			dropAndWait(t, v, a)
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			dir := storeFavorite(t, v, "Promoted", a)
			v.favoriteSaved()
			v.locationMap.Settle()
			if len(locationGPSRecords(t, dir)) != 1 {
				t.Fatal("saving did not promote known raw fact")
			}
			v.state.SetMergeMode(true)
			dropAndWait(t, v, b)
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			if len(v.locationMap.RawFacts()) != 2 || len(locationGPSRecords(t, dir)) != 1 {
				t.Fatal("unsaved merge member escaped memory-only ownership")
			}
		})
		t.Run("favorite_reopen", func(t *testing.T) {
			v := newTestViewer(t)
			data := uitest.EncodeJPEG(t, 24, 16, color.White)
			base := storage.NewFileURI(uitest.WriteTempFile(t, "saved.jpg", data))
			var reads atomic.Int32
			source := uitest.ReaderURI(base, func() (io.ReadCloser, error) { reads.Add(1); return io.NopCloser(bytes.NewReader(data)), nil })
			dir := storeFavorite(t, v, "Locations", base)
			dropAndWait(t, v, source)
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			if len(locationGPSRecords(t, dir)) != 1 {
				t.Fatal("Favorite raw GPS record was not persisted")
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			fresh := newTestViewer(t)
			fresh.favorites.SetDir(filepath.Dir(dir))
			fresh.OpenFavorite(dir, []fyne.URI{source})
			waitForScan(t, fresh)
			waitForSort(t, fresh)
			waitUntilLoaded(t, fresh)
			fresh.display.Settle()
			reads.Store(0)
			locationMenu(t, fresh).Action()
			fresh.locationMap.Settle()
			if reads.Load() != 0 || fresh.locationMap.Counts().Unlocated != 1 {
				t.Fatalf("fresh instance did not reuse versioned Favorite fact: reads=%d counts=%+v", reads.Load(), fresh.locationMap.Counts())
			}
		})
		t.Run("live_reuse", func(t *testing.T) {
			t.Run("partial_cancel", func(t *testing.T) {
				v := newTestViewer(t)
				data := uitest.EncodeJPEG(t, 24, 16, color.White)
				base := storage.NewFileURI(uitest.WriteTempFile(t, "a-completed.jpg", data))
				var reads atomic.Int32
				a := uitest.ReaderURI(base, func() (io.ReadCloser, error) {
					reads.Add(1)
					return io.NopCloser(bytes.NewReader(data)), nil
				})
				b, hold, entered, release := locationHeldRead(t, uitest.TempJPEGURI(t, "b-pending.jpg", 24, 16, color.Black))
				dropAndWait(t, v, a, b)
				v.display.Settle()
				reads.Store(0)
				hold.Store(true)
				queue := &uitest.UIQueue{}
				v.locationMap.SetUIQueue(queue)
				locationMenu(t, v).Action()
				<-entered
				queue.Drain()
				if counts := v.locationMap.Counts(); counts.Completed != 1 || counts.Complete {
					t.Fatalf("partial scan fixture did not publish completed member: %+v", counts)
				}
				v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
				release()
				v.locationMap.Settle()
				locationMenu(t, v).Action()
				v.locationMap.Settle()
				if counts := v.locationMap.Counts(); !counts.Complete || counts.Unlocated != 2 || reads.Load() != 1 {
					t.Fatalf("partial cancellation lost reusable fact or unfinished member: reads=%d counts=%+v", reads.Load(), counts)
				}
			})
			v := newTestViewer(t)
			data := uitest.EncodeJPEG(t, 24, 16, color.White)
			base := storage.NewFileURI(uitest.WriteTempFile(t, "no-gps.jpg", data))
			var reads atomic.Int32
			source := uitest.ReaderURI(base, func() (io.ReadCloser, error) { reads.Add(1); return io.NopCloser(bytes.NewReader(data)), nil })
			dropAndWait(t, v, source)
			v.display.Settle()
			reads.Store(0)
			for range 2 {
				locationMenu(t, v).Action()
				v.locationMap.Settle()
				if v.locationMap.Counts().Unlocated != 1 {
					t.Fatal("valid GPS absence lost")
				}
				v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			}
			if got := reads.Load(); got != 1 {
				t.Fatalf("unchanged completed fact reread %d times", got)
			}
		})
	})
	t.Run("cluster_visit", func(t *testing.T) {
		t.Run("preview_opens_exact_members", func(t *testing.T) {
			v := newTestViewer(t)
			v.win.Resize(fyne.NewSize(1000, 700))
			var sources []fyne.URI
			var want []string
			for i := range 22 {
				source := uitest.TempGPSJPEGURI(t, fmt.Sprintf("cluster-%02d.jpg", i), 24, 16, 52.52, 13.405)
				sources = append(sources, source)
				want = append(want, source.String())
			}
			sources = append(sources, uitest.TempGPSJPEGURI(t, "outside.jpg", 24, 16, 40.7, -74))
			dropAndWait(t, v, sources...)
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			for _, preview := range []bool{true, false} {
				if preview {
					fynetest.Tap(locationPhoto(t, v, sources[0].Name()))
				} else {
					fynetest.Tap(locationButton(t, v, fmt.Sprintf(lang.L("%d images"), 22)))
				}
				v.grid.Settle()
				if !v.grid.Visible() || v.locationMap.Visible() {
					t.Fatalf("cluster click did not open Grid (preview=%t)", preview)
				}
				var got []string
				for _, index := range v.grid.ResultIndexes() {
					got = append(got, v.FileAt(index).String())
				}
				if !slices.Equal(got, want) {
					t.Fatalf("cluster click opened %d images instead of the exact 22 members (preview=%t)", len(got), preview)
				}
				v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
				v.locationMap.Settle()
			}
		})
		t.Run("frozen_progressive_members_and_escape_stages", func(t *testing.T) {
			v := newTestViewer(t)
			a := uitest.TempGPSJPEGURI(t, "a.jpg", 24, 16, 52.52, 13.405)
			b := uitest.TempGPSJPEGURI(t, "b.jpg", 24, 16, 52.52, 13.405)
			c := uitest.TempGPSJPEGURI(t, "c.jpg", 24, 16, 52.52, 13.405)
			c, hold, entered, unblock := locationHeldRead(t, c)
			dropAndWait(t, v, a, b, c)
			v.display.Settle()
			hold.Store(true)
			queue := &uitest.UIQueue{}
			v.locationMap.SetUIQueue(queue)
			locationMenu(t, v).Action()
			<-entered
			queue.Drain()
			fynetest.Tap(locationButton(t, v, fmt.Sprintf(lang.L("%d images"), 2)))
			v.grid.Settle()
			unblock()
			v.locationMap.Settle()
			if len(v.locationMap.Points()) != 3 || !slices.Equal(v.grid.ResultIndexes(), []int{0, 1}) {
				t.Fatal("new discoveries changed frozen cluster visit or scan stopped")
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeySpace})
			v.grid.HandleRune('/')
			v.grid.HandleRune('a')
			if !v.grid.Searching() || v.grid.SelectionCount() != 1 {
				t.Fatal("cluster search/selection setup failed")
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			if !v.grid.Visible() || !v.grid.Searching() || v.grid.SelectionCount() != 0 {
				t.Fatal("first Escape did not clear only selection")
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			if !v.grid.Visible() || v.grid.Searching() || !slices.Equal(v.grid.ResultIndexes(), []int{0, 1}) {
				t.Fatal("second Escape did not restore frozen search scope")
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			v.locationMap.Settle()
			if v.grid.Visible() || !v.locationMap.Visible() {
				t.Fatal("final cluster Escape did not return to map")
			}
			locationButton(t, v, fmt.Sprintf(lang.L("%d images"), 3))
		})
		t.Run("navigation", func(t *testing.T) {
			v := newTestViewer(t)
			a := uitest.TempGPSJPEGURI(t, "a.jpg", 24, 16, 52.52, 13.405)
			b := uitest.TempGPSJPEGURI(t, "b.jpg", 24, 16, 52.52, 13.405)
			c := uitest.TempJPEGURI(t, "c.jpg", 24, 16, color.White)
			dropAndWait(t, v, a, b, c)
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			fynetest.Tap(locationButton(t, v, fmt.Sprintf(lang.L("%d images"), 2)))
			v.grid.Settle()
			if !v.grid.Visible() || !slices.Equal(v.grid.ResultIndexes(), []int{0, 1}) {
				t.Fatalf("cluster Grid members: %v", v.grid.ResultIndexes())
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
			waitUntilLoaded(t, v)
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
			waitUntilLoaded(t, v)
			if v.state.index != 1 {
				t.Fatal("cluster image arrows escaped frozen membership")
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
			waitUntilLoaded(t, v)
			if v.state.index != 0 {
				t.Fatal("cluster image wrap included unlocated source")
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			v.grid.Settle()
			if !v.grid.Visible() {
				t.Fatal("image Escape did not retrace to Grid")
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			v.locationMap.Settle()
			if v.grid.Visible() || !v.locationMap.Visible() {
				t.Fatal("Grid Escape did not retrace to map")
			}
		})
	})
	t.Run("duplicate_locations", func(t *testing.T) {
		t.Run("conflict_only", func(t *testing.T) {
			v := newTestViewer(t)
			a := uitest.PatternedGPSJPEGURI(t, "a.jpg", 1, 64, 48, 52.52, 13.405)
			b := uitest.PatternedGPSJPEGURI(t, "b.jpg", 1, 96, 72, 40.7, -74)
			rep := uitest.PatternedJPEGURISize(t, "representative.jpg", 1, 192, 144)
			before := map[string][]byte{}
			for _, source := range []fyne.URI{a, b, rep} {
				data, err := os.ReadFile(source.Path())
				if err != nil {
					t.Fatal(err)
				}
				before[source.Path()] = data
			}
			dropAndWait(t, v, a, b, rep)
			v.menus.Actions().Hide().Action()
			v.grid.Settle()
			locationMenu(t, v).Action()
			v.grid.Settle()
			v.locationMap.Settle()
			counts := v.locationMap.Counts()
			if len(v.locationMap.Points()) != 0 || counts.Total != 1 || counts.Conflicts != 1 || !counts.Complete {
				t.Fatalf("conflicting donors produced a location: %+v", counts)
			}
			found := false
			explorerWalk(v.win.Content(), func(object fyne.CanvasObject) {
				if label, ok := object.(*widget.Label); ok && label.Text == fmt.Sprintf(lang.L("No usable image locations. %d without GPS, %d unreadable, %d conflicts"), 0, 0, 1) {
					found = true
				}
			})
			if !found {
				t.Fatal("conflict-only empty state not mounted")
			}
			for path, data := range before {
				after, err := os.ReadFile(path)
				if err != nil || !bytes.Equal(data, after) {
					t.Fatal("duplicate resolution modified source metadata")
				}
			}
		})
		t.Run("representative_and_fallback", func(t *testing.T) {
			for _, direct := range []bool{false, true} {
				t.Run(map[bool]string{false: "borrowed", true: "direct"}[direct], func(t *testing.T) {
					v := newTestViewer(t)
					donor := uitest.PatternedGPSJPEGURI(t, "a-donor.jpg", 1, 64, 48, 52.52, 13.405)
					rep := uitest.PatternedJPEGURISize(t, "z-representative.jpg", 1, 192, 144)
					if direct {
						rep = uitest.PatternedGPSJPEGURI(t, "z-representative.jpg", 1, 192, 144, 51.507, -.128)
					}
					dropAndWait(t, v, donor, rep)
					v.menus.Actions().Hide().Action()
					v.grid.Settle()
					locationMenu(t, v).Action()
					v.grid.Settle()
					v.locationMap.Settle()
					points := v.locationMap.Points()
					if len(points) != 1 || points[0].Source.URI.Path() != rep.Path() {
						t.Fatalf("representative not located: %+v", points)
					}
					want := 52.52
					if direct {
						want = 51.507
					}
					if math.Abs(points[0].Metadata.Latitude-want) > .000001 {
						t.Fatal("wrong GPS source selected")
					}
					if raw := v.locationMap.RawFacts()[rep.String()]; raw.Metadata.HasGPS != direct {
						t.Fatal("derived donor coordinates were stored as representative raw GPS")
					}
					fynetest.Tap(locationPhoto(t, v, rep.Name()))
					waitUntilLoaded(t, v)
					if current, _, _ := v.CurrentFile(); current.Path() != rep.Path() {
						t.Fatal("fallback opened donor instead of representative")
					}
				})
			}
		})
	})
	t.Run("tile_recovery", func(t *testing.T) {
		previous := testApp.Settings().Theme()
		t.Cleanup(func() { testApp.Settings().SetTheme(previous) })
		testApp.Settings().SetTheme(theme.DefaultTheme())
		t.Run("atomic_retry", func(t *testing.T) {
			v := newTestViewer(t)
			v.SetThemeMode(appearance.Light)
			v.win.Resize(fyne.NewSize(1000, 700))
			oldColor, newColor := color.NRGBA{B: 91, A: 255}, color.NRGBA{G: 91, A: 255}
			oldTile, newTile := uitest.EncodePNG(t, 256, 256, oldColor), uitest.EncodePNG(t, 256, 256, newColor)
			startReplacement, finishRetry := locationTileRetryFixture(t, v, oldTile, newTile)
			dropAndWait(t, v, uitest.TempGPSJPEGURI(t, "partial.jpg", 24, 16, 52.52, 13.405))
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			locationAssertTileColor(t, v, oldColor)
			startReplacement()
			locationSurface(t, v).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(60, 0)})
			v.locationMap.Settle()
			locationAssertTileColor(t, v, oldColor)
			finishRetry()
			locationAssertTileColor(t, v, newColor)
		})
		t.Run("obsolete_viewport", func(t *testing.T) {
			v := newTestViewer(t)
			v.SetThemeMode(appearance.Light)
			v.win.Resize(fyne.NewSize(1000, 700))
			colors := []color.NRGBA{{B: 91, A: 255}, {R: 91, A: 255}, {G: 91, A: 255}}
			pixels := [][]byte{uitest.EncodePNG(t, 256, 256, colors[0]), uitest.EncodePNG(t, 256, 256, colors[1]), uitest.EncodePNG(t, 256, 256, colors[2])}
			var phase atomic.Int32
			entered, release := make(chan struct{}, 128), make(chan struct{})
			unblock := sync.OnceFunc(func() { close(release) })
			t.Cleanup(unblock)
			v.locationMap.ConfigureTiles(locationmap.TileOptions{Client: &http.Client{Transport: locationTileTransport(func(request *http.Request) (*http.Response, error) {
				stage := phase.Load()
				if stage == 2 {
					entered <- struct{}{}
					select {
					case <-request.Context().Done():
						return nil, request.Context().Err()
					case <-release:
					}
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Cache-Control": []string{"no-cache"}}, Body: io.NopCloser(bytes.NewReader(pixels[stage]))}, nil
			})}}, noLocationRetry)
			dropAndWait(t, v, uitest.TempGPSJPEGURI(t, "retired.jpg", 24, 16, 52.52, 13.405))
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			queue := &uitest.UIQueue{}
			v.locationMap.SetUIQueue(queue)
			phase.Store(1)
			locationSurface(t, v).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(60, 0)})
			v.locationMap.Wait()
			phase.Store(2)
			locationSurface(t, v).Scrolled(&fyne.ScrollEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(500, 300)}, Scrolled: fyne.NewDelta(0, 30)})
			<-entered
			queue.Drain()
			locationAssertTileColor(t, v, colors[0])
			unblock()
			v.locationMap.Settle()
			locationAssertTileColor(t, v, colors[2])
		})
		t.Run("stale_during_drag", func(t *testing.T) {
			v := newTestViewer(t)
			v.win.Resize(fyne.NewSize(1000, 700))
			oldTile := uitest.EncodePNG(t, 256, 256, color.NRGBA{R: 31, G: 79, B: 123, A: 255})
			newTile := uitest.EncodePNG(t, 256, 256, color.NRGBA{R: 12, G: 120, B: 31, A: 255})
			var replacing atomic.Bool
			entered, release := make(chan struct{}, 128), make(chan struct{})
			unblock := sync.OnceFunc(func() { close(release) })
			t.Cleanup(unblock)
			v.locationMap.ConfigureTiles(locationmap.TileOptions{Client: &http.Client{Transport: locationTileTransport(func(request *http.Request) (*http.Response, error) {
				pixels := oldTile
				if replacing.Load() {
					entered <- struct{}{}
					select {
					case <-release:
						pixels = newTile
					case <-request.Context().Done():
						return nil, request.Context().Err()
					}
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Cache-Control": []string{"no-cache"}}, Body: io.NopCloser(bytes.NewReader(pixels))}, nil
			})}}, noLocationRetry)
			dropAndWait(t, v, uitest.TempGPSJPEGURI(t, "drag.jpg", 24, 16, 52.52, 13.405))
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			countTiles := func() int {
				count := 0
				explorerWalk(locationSurface(t, v), func(object fyne.CanvasObject) {
					if img, ok := object.(*canvas.Image); ok && img.Image != nil && img.Image.Bounds().Dx() == 256 {
						count++
					}
				})
				return count
			}
			before := countTiles()
			if before == 0 {
				t.Fatal("setup has no painted tiles")
			}
			var retained *canvas.Image
			explorerWalk(locationSurface(t, v), func(object fyne.CanvasObject) {
				if img, ok := object.(*canvas.Image); ok && img.Image != nil && img.Image.Bounds().Dx() == 256 {
					retained = img
				}
			})
			position, size := retained.Position(), retained.Size()
			replacing.Store(true)
			locationSurface(t, v).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(60, 0)})
			<-entered
			if after := countTiles(); after < before {
				t.Fatalf("drag hid loaded map while replacement is pending: %d tiles before, %d after", before, after)
			}
			if got := retained.Position(); math.Abs(float64(got.X-position.X-60)) > .01 || got.Y != position.Y {
				t.Errorf("background did not follow drag: %v -> %v", position, got)
			}
			locationSurface(t, v).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(30, 20)})
			if got := retained.Position(); math.Abs(float64(got.X-position.X-90)) > .01 || math.Abs(float64(got.Y-position.Y-20)) > .01 {
				t.Errorf("background lost repeated drag: %v -> %v", position, got)
			}
			anchor := fyne.NewPos(500, 300)
			locationSurface(t, v).Scrolled(&fyne.ScrollEvent{PointEvent: fyne.PointEvent{Position: anchor}, Scrolled: fyne.NewDelta(0, 10)})
			want := fyne.NewPos((position.X+90-anchor.X)*1.2+anchor.X, (position.Y+20-anchor.Y)*1.2+anchor.Y)
			if got := retained.Position(); math.Abs(float64(got.X-want.X)) > .01 || math.Abs(float64(got.Y-want.Y)) > .01 || math.Abs(float64(retained.Size().Width-size.Width*1.2)) > .01 {
				t.Errorf("background did not follow anchored zoom: %v, want %v", got, want)
			}
			unblock()
			v.locationMap.Settle()
			if countTiles() == 0 {
				t.Fatal("completed replacement has no painted map")
			}
		})
		t.Run("toast_throttle", func(t *testing.T) {
			v := newTestViewer(t)
			var seconds atomic.Int64
			v.locationMap.ConfigureTiles(locationmap.TileOptions{Client: &http.Client{Transport: offlineReleaseImages{}}, Now: func() time.Time { return time.Unix(1000+seconds.Load(), 0) }}, noLocationRetry)
			dropAndWait(t, v, uitest.TempGPSJPEGURI(t, "outage.jpg", 24, 16, 52.52, 13.405))
			before := v.toast.gen.Load()
			for _, elapsed := range []int64{0, 30, 59, 60} {
				seconds.Store(elapsed)
				locationMenu(t, v).Action()
				v.locationMap.Settle()
				want := before + 1
				if elapsed >= 60 {
					want++
				}
				if got := v.toast.gen.Load(); got != want {
					t.Fatalf("toast rate at %ds: %d, want %d", elapsed, got, want)
				}
				if v.toast.text.Text != lang.L("Could not load map tiles. An internet connection is required.") {
					t.Fatal("missing outage explanation")
				}
				locationPhoto(t, v, "outage.jpg")
				v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			}
		})
		t.Run("retry_and_restore", func(t *testing.T) {
			v := newTestViewer(t)
			var recovered atomic.Bool
			var seconds atomic.Int64
			waiting := make(chan time.Duration, 64)
			permit := make(chan struct{})
			pixels := uitest.EncodePNG(t, 256, 256, color.NRGBA{G: 73, A: 255})
			client := &http.Client{Transport: locationTileTransport(func(_ *http.Request) (*http.Response, error) {
				if !recovered.Load() {
					return &http.Response{StatusCode: 503, Header: http.Header{"Retry-After": []string{"2"}}, Body: http.NoBody}, nil
				}
				return &http.Response{StatusCode: 200, Header: http.Header{"Cache-Control": []string{"max-age=3600"}}, Body: io.NopCloser(bytes.NewReader(pixels))}, nil
			})}
			v.locationMap.ConfigureTiles(locationmap.TileOptions{Client: client, Now: func() time.Time { return time.Unix(1000+seconds.Load(), 0) }}, func(ctx context.Context, delay time.Duration) error {
				waiting <- delay
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-permit:
					return nil
				}
			})
			dropAndWait(t, v, uitest.TempGPSJPEGURI(t, "retry.jpg", 24, 16, 52.52, 13.405))
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			if delay := <-waiting; delay != 2*time.Second {
				t.Fatalf("server retry delay = %v", delay)
			}
			locationPhoto(t, v, "retry.jpg")
			recovered.Store(true)
			seconds.Store(2)
			close(permit)
			v.locationMap.Wait()
			v.locationMap.Settle()
			found := false
			explorerWalk(locationSurface(t, v), func(o fyne.CanvasObject) {
				if img, ok := o.(*canvas.Image); ok && img.Image != nil && img.Image.Bounds().Dx() == 256 {
					found = true
				}
			})
			if !found {
				t.Fatal("automatic recovery did not install tile pixels")
			}
		})
		t.Run("hidden_direct_visit", func(t *testing.T) {
			v := newTestViewer(t)
			queue := &locationNoticeQueue{UIQueue: &uitest.UIQueue{}, ready: make(chan struct{}, 1)}
			v.locationMap.SetUIQueue(queue)
			entered := make(chan struct{}, 128)
			cancelled := make(chan struct{}, 128)
			v.locationMap.ConfigureTiles(locationmap.TileOptions{Client: &http.Client{Transport: locationTileTransport(func(request *http.Request) (*http.Response, error) {
				entered <- struct{}{}
				<-request.Context().Done()
				cancelled <- struct{}{}
				return nil, request.Context().Err()
			})}}, noLocationRetry)
			dropAndWait(t, v, uitest.TempGPSJPEGURI(t, "held.jpg", 24, 16, 52.52, 13.405))
			locationMenu(t, v).Action()
			for len(v.locationMap.Points()) == 0 {
				<-queue.ready
				queue.Drain()
			}
			<-entered
			// Metadata delivery is independent of the held tile transport.
			fynetest.Tap(locationPhoto(t, v, "held.jpg"))
			waitUntilLoaded(t, v)
			v.locationMap.Settle()
			<-cancelled
			if v.locationMap.Visible() {
				t.Fatal("held tile kept map visible")
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			<-entered
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			v.locationMap.Settle()
			if v.locationMap.Active() {
				t.Fatal("held tile kept map active after close")
			}
		})
		t.Run("local_results_survive", func(t *testing.T) {
			v := newTestViewer(t)
			var calls atomic.Int32
			pixels := uitest.EncodePNG(t, 256, 256, color.NRGBA{R: 31, G: 79, B: 123, A: 255})
			v.locationMap.ConfigureTiles(locationmap.TileOptions{Client: &http.Client{Transport: locationTileTransport(func(_ *http.Request) (*http.Response, error) {
				calls.Add(1)
				return &http.Response{StatusCode: 200, Header: http.Header{"Cache-Control": []string{"max-age=3600"}}, Body: io.NopCloser(bytes.NewReader(pixels))}, nil
			})}}, nil)
			dropAndWait(t, v, uitest.TempGPSJPEGURI(t, "tile.jpg", 24, 16, 52.52, 13.405))
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			if calls.Load() == 0 {
				t.Fatalf("visible map did not request tiles: surface=%v overlay=%v", locationSurface(t, v).Size(), v.locationMap.Overlay().Size())
			}
			found := false
			explorerWalk(locationSurface(t, v), func(o fyne.CanvasObject) {
				if img, ok := o.(*canvas.Image); ok && img.Image != nil && img.Image.Bounds().Dx() == 256 {
					found = true
				}
			})
			if !found {
				t.Fatal("successful tile pixels are not mounted")
			}
			locationPhoto(t, v, "tile.jpg")
		})
	})
	t.Run("tile_policy_and_bounds", func(t *testing.T) {
		v := newTestViewer(t)
		v.win.Resize(fyne.NewSize(1000, 700))
		pixels := uitest.EncodePNG(t, 256, 256, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
		var badRequest atomic.Bool
		client := &http.Client{Transport: locationTileTransport(func(request *http.Request) (*http.Response, error) {
			if request.URL.Scheme != "https" || request.URL.Host != "tile.openstreetmap.org" || !strings.HasPrefix(request.Header.Get("User-Agent"), "PicFetch (") {
				badRequest.Store(true)
			}
			return &http.Response{StatusCode: 200, Header: http.Header{"Cache-Control": []string{"max-age=86400"}}, Body: io.NopCloser(bytes.NewReader(pixels))}, nil
		})}
		const encodedLimit = 64 * 1024
		const decodedLimit = 256 * 256 * 4
		v.locationMap.ConfigureTiles(locationmap.TileOptions{Client: client, EncodedBytes: encodedLimit, DecodedBytes: decodedLimit}, noLocationRetry)
		dropAndWait(t, v, uitest.TempGPSJPEGURI(t, "local.jpg", 24, 16, 52.52, 13.405))
		locationMenu(t, v).Action()
		v.locationMap.Settle()
		attribution := false
		explorerWalk(v.win.Content(), func(object fyne.CanvasObject) {
			if label, ok := object.(*widget.Label); ok && label.Text == lang.L("© OpenStreetMap contributors") {
				attribution = true
			}
		})
		if !attribution {
			t.Fatal("visible OSM attribution missing")
		}
		for range 20 {
			locationSurface(t, v).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(180, 0)})
			v.locationMap.Settle()
			encoded, decoded := v.locationMap.TileUsage()
			if encoded > encodedLimit || decoded > decodedLimit {
				t.Fatalf("viewport tile cache exceeded byte bounds: %d/%d", encoded, decoded)
			}
		}
		if badRequest.Load() {
			t.Fatal("viewport request lacked compliant endpoint or identification")
		}
	})
	t.Run("preparation_and_counts", func(t *testing.T) {
		t.Run("duplicate_progress", func(t *testing.T) {
			for _, cancel := range []bool{false, true} {
				t.Run(fmt.Sprintf("cancel=%t", cancel), func(t *testing.T) {
					synctest.Test(t, func(t *testing.T) {
						fixture := newGridAnalysisFixture(t, true)
						v := fixture.v
						pixels := uitest.EncodePNG(t, 256, 256, color.Black)
						v.locationMap.ConfigureTiles(locationmap.TileOptions{Client: &http.Client{Transport: locationTileTransport(func(_ *http.Request) (*http.Response, error) {
							return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(pixels))}, nil
						})}}, noLocationRetry)
						locationMenu(t, v).Action()
						fixture.deliver()
						bar, waiting := preparationBars(v.locationMap.Overlay())
						if bar == nil || waiting != nil || bar.Max != 3 || bar.Value != 0 {
							t.Fatalf("initial remaining-check progress: bar=%+v waiting=%v", bar, waiting != nil)
						}
						fixture.releaseNext(4)
						fixture.deliver()
						bar, waiting = preparationBars(v.locationMap.Overlay())
						if bar == nil || waiting != nil || bar.Max != 3 || bar.Value != 1 {
							t.Fatalf("completed check did not advance progress: bar=%+v waiting=%v", bar, waiting != nil)
						}
						if cancel {
							v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
						}
						fixture.releaseNext(5)
						fixture.releaseNext(6)
						fixture.deliver()
						v.grid.Settle()
						v.locationMap.Settle()
						if bar, waiting := preparationBars(v.locationMap.Overlay()); bar != nil || waiting != nil {
							t.Fatal("completed/cancelled preparation retained progress")
						}
						if cancel && v.locationMap.Active() {
							t.Fatal("late progress reopened cancelled Location Map")
						}
					})
				})
			}
		})
		t.Run("fallback_accounting", func(t *testing.T) {
			for _, outcome := range []string{"located", "unlocated", "unreadable", "conflict"} {
				t.Run(outcome, func(t *testing.T) {
					v := newTestViewer(t)
					base := uitest.PatternedGPSJPEGURI(t, "a-donor.jpg", 1, 64, 48, 52.52, 13.405)
					if outcome == "unlocated" {
						base = uitest.PatternedJPEGURISize(t, "a-donor.jpg", 1, 64, 48)
					}
					data, err := os.ReadFile(base.Path())
					if err != nil {
						t.Fatal(err)
					}
					var hold atomic.Bool
					entered, release := make(chan struct{}), make(chan struct{})
					var once sync.Once
					unblock := func() { once.Do(func() { close(release) }) }
					t.Cleanup(unblock)
					donor := uitest.ReaderURI(base, func() (io.ReadCloser, error) {
						if hold.CompareAndSwap(true, false) {
							close(entered)
							<-release
							if outcome == "unreadable" {
								return nil, errors.New("controlled donor failure")
							}
						}
						return io.NopCloser(bytes.NewReader(data)), nil
					})
					rep := uitest.PatternedJPEGURISize(t, "z-representative.jpg", 1, 192, 144)
					sources := []fyne.URI{donor, rep}
					if outcome == "conflict" {
						sources = append(sources, uitest.PatternedGPSJPEGURI(t, "b-conflict.jpg", 1, 96, 72, 40.7, -74))
					}
					dropAndWait(t, v, sources...)
					v.menus.Actions().Hide().Action()
					v.grid.Settle()
					v.display.Settle()
					pixels := uitest.EncodePNG(t, 256, 256, color.Black)
					v.locationMap.ConfigureTiles(locationmap.TileOptions{Client: &http.Client{Transport: locationTileTransport(func(_ *http.Request) (*http.Response, error) {
						return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(pixels))}, nil
					})}}, noLocationRetry)
					queue := &uitest.UIQueue{}
					v.locationMap.SetUIQueue(queue)
					hold.Store(true)
					beforeToast := v.toast.gen.Load()
					locationMenu(t, v).Action()
					<-entered
					queue.Drain()
					if counts := v.locationMap.Counts(); counts.Total != 1 || counts.Completed != 0 || counts.Complete {
						t.Fatalf("group completed before required donor read: %+v", counts)
					}
					unblock()
					v.locationMap.Settle()
					want := locationmap.Counts{Total: 1, Completed: 1, Complete: true}
					switch outcome {
					case "located":
						want.Located = 1
					case "unlocated":
						want.Unlocated = 1
					case "unreadable":
						want.Failed = 1
					case "conflict":
						want.Conflicts = 1
					}
					if got := v.locationMap.Counts(); got != want || v.toast.gen.Load() != beforeToast {
						t.Fatalf("fallback accounting/toast: got %+v want %+v; toast=%q", got, want, v.toast.text.Text)
					}
				})
			}
		})
		t.Run("duplicate_preparation", func(t *testing.T) {
			v := newTestViewer(t)
			a := uitest.PatternedGPSJPEGURI(t, "small.jpg", 1, 64, 48, 52.52, 13.405)
			b := uitest.PatternedGPSJPEGURI(t, "large.jpg", 1, 192, 144, 52.52, 13.405)
			b, hold, entered, unblock := locationHeldRead(t, b)
			dropAndWait(t, v, a, b)
			v.display.Settle()
			hold.Store(true)
			v.menus.Actions().Hide().Action()
			<-entered
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			if len(v.locationMap.Points()) != 0 {
				t.Fatal("map plotted before duplicate preparation completed")
			}
			progress, waiting := preparationBars(v.locationMap.Overlay())
			if progress == nil || waiting != nil || progress.Value >= progress.Max {
				t.Fatal("held duplicate checks have no mounted incomplete progress bar")
			}
			unblock()
			v.grid.Settle()
			v.locationMap.Settle()
			points := v.locationMap.Points()
			counts := v.locationMap.Counts()
			if len(points) != 1 || points[0].Source.URI.Path() != b.Path() || counts.Total != 1 || counts.Completed != 1 || !counts.Complete {
				t.Fatalf("map did not use highest-resolution representative: %v %+v", points, counts)
			}
			if progress, waiting := preparationBars(v.locationMap.Overlay()); progress != nil || waiting != nil {
				t.Fatal("completed preparation retained its progress bar")
			}
		})
		t.Run("progressive_results", func(t *testing.T) {
			v, release, queue := locationProgressFixture(t)
			defer release()
			queue.Drain()
			counts := v.locationMap.Counts()
			if counts.Completed != 1 || counts.Total != 2 || counts.Located != 1 || counts.Complete {
				t.Fatalf("no honest progressive result: %+v", counts)
			}
			locationPhoto(t, v, "a.jpg")
			release()
			v.locationMap.Settle()
			if counts = v.locationMap.Counts(); !counts.Complete || counts.Completed != 2 || counts.Located != 2 {
				t.Fatalf("final counts: %+v", counts)
			}
		})
	})
	t.Run("lifecycle/progressive_direct_visit", func(t *testing.T) {
		v, release, queue := locationProgressFixture(t)
		queue.Drain()
		fynetest.Tap(locationPhoto(t, v, "a.jpg"))
		waitUntilLoaded(t, v)
		if v.locationMap.Visible() {
			t.Fatal("map remains visible during image visit")
		}
		release()
		v.locationMap.Settle()
		if !v.locationMap.Counts().Complete || v.locationMap.Counts().Located != 2 {
			t.Fatal("image visit canceled progressive discovery")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		v.locationMap.Settle()
		locationPhoto(t, v, "b.jpg")
	})
	t.Run("empty_states_and_read_only", func(t *testing.T) {
		t.Run("base_states", func(t *testing.T) {
			for _, kind := range []string{"empty", "unlocated", "single", "unreadable"} {
				t.Run(kind, func(t *testing.T) {
					v := newTestViewer(t)
					var source fyne.URI
					var before []byte
					var fail atomic.Bool
					if kind != "empty" {
						data := uitest.GPSJPEG(t, 24, 16, 0, 0)
						if kind == "unlocated" {
							data = uitest.EncodeJPEG(t, 24, 16, color.White)
						}
						base := storage.NewFileURI(uitest.WriteTempFile(t, "state.jpg", data))
						source = uitest.ReaderURI(base, func() (io.ReadCloser, error) {
							if fail.Load() {
								return nil, errors.New("held source unavailable")
							}
							return io.NopCloser(bytes.NewReader(data)), nil
						})
						dropAndWait(t, v, source)
						v.display.Settle()
						before = data
						if kind == "unreadable" {
							fail.Store(true)
						}
					}
					locationMenu(t, v).Action()
					v.locationMap.Settle()
					counts := v.locationMap.Counts()
					if !counts.Complete {
						t.Fatal("finite base scan did not complete")
					}
					wantLocated, wantMissing, wantFailed := 0, 0, 0
					switch kind {
					case "single":
						wantLocated = 1
					case "unlocated":
						wantMissing = 1
					case "unreadable":
						wantFailed = 1
					}
					if counts.Located != wantLocated || counts.Unlocated != wantMissing || counts.Failed != wantFailed {
						t.Fatalf("wrong outcomes: %+v", counts)
					}
					if source != nil {
						after, err := os.ReadFile(source.Path())
						if err != nil || !bytes.Equal(before, after) {
							t.Fatal("map changed source bytes")
						}
					}
					locationButton(t, v, lang.L("Back to Viewer"))
				})
			}
		})
	})
	t.Run("thumbnail_reuse", func(t *testing.T) {
		t.Run("source_versions", func(t *testing.T) {
			for _, stage := range []string{"metadata", "preview"} {
				t.Run(stage, func(t *testing.T) {
					v := newTestViewer(t)
					base := uitest.TempGPSJPEGURI(t, "changed.jpg", 24, 16, 52.52, 13.405)
					var reads atomic.Int32
					var hold atomic.Bool
					entered, release := make(chan struct{}), make(chan struct{})
					var once sync.Once
					unblock := func() { once.Do(func() { close(release) }) }
					t.Cleanup(unblock)
					heldRead := int32(1)
					if stage == "preview" {
						heldRead = 2
					}
					source := uitest.ReaderURI(base, func() (io.ReadCloser, error) {
						data, err := os.ReadFile(base.Path())
						if err != nil {
							return nil, err
						}
						if reads.Add(1) == heldRead && hold.CompareAndSwap(true, false) {
							close(entered)
							<-release
						}
						return io.NopCloser(bytes.NewReader(data)), nil
					})
					dropAndWait(t, v, source)
					v.display.Settle()
					reads.Store(0)
					hold.Store(true)
					queue := &locationNoticeQueue{UIQueue: &uitest.UIQueue{}, ready: make(chan struct{}, 1)}
					v.locationMap.SetUIQueue(queue)
					locationMenu(t, v).Action()
				waiting:
					for {
						select {
						case <-entered:
							break waiting
						case <-queue.ready:
							queue.Drain()
						}
					}
					if err := os.WriteFile(base.Path(), uitest.GPSJPEG(t, 60, 40, 40.7, -74), 0o600); err != nil {
						t.Fatal(err)
					}
					unblock()
					v.locationMap.Settle()
					if stage == "metadata" {
						if len(v.locationMap.Points()) != 0 || len(v.locationMap.RawFacts()) != 0 {
							t.Fatal("held metadata published facts after its source changed")
						}
					} else {
						explorerWalk(locationPhoto(t, v, source.Name()), func(object fyne.CanvasObject) {
							if img, ok := object.(*canvas.Image); ok && img.Image != nil {
								t.Error("held old preview was mounted after its source changed")
							}
						})
						if _, ok := v.grid.CachedThumb(source); ok {
							t.Fatal("held old preview entered the shared thumbnail cache")
						}
					}
					v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
					locationMenu(t, v).Action()
					v.locationMap.Settle()
					points := v.locationMap.Points()
					if len(points) != 1 || math.Abs(points[0].Metadata.Latitude-40.7) > .000001 {
						t.Fatalf("reopen did not read the current source version: %+v", points)
					}
					fresh := false
					explorerWalk(locationPhoto(t, v, source.Name()), func(object fyne.CanvasObject) {
						if img, ok := object.(*canvas.Image); ok && img.Image != nil && img.Image.Bounds().Dx() == 60 {
							fresh = true
						}
					})
					if !fresh {
						t.Fatal("current source preview was not mounted")
					}
				})
			}
		})
		t.Run("visible_demand", func(t *testing.T) {
			v := newTestViewer(t)
			a := uitest.TempGPSJPEGURI(t, "a.jpg", 24, 16, 52.52, 13.405)
			dropAndWait(t, v, a)
			warmThumbs(t, v)
			cached, _ := v.grid.CachedThumb(a)
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			found := false
			explorerWalk(locationSurface(t, v), func(o fyne.CanvasObject) {
				if img, ok := o.(*canvas.Image); ok && img.Image == cached {
					found = true
				}
			})
			if !found {
				t.Fatal("map did not reuse the shared, versioned Grid preview")
			}
			locationSurface(t, v).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(5000, 0)})
			v.locationMap.Settle()
			explorerWalk(locationSurface(t, v), func(o fyne.CanvasObject) {
				if img, ok := o.(*canvas.Image); ok && img.Image != nil {
					t.Error("distant thumbnail still retains rendered pixels")
				}
			})
		})
	})
	t.Run("lifecycle", func(t *testing.T) {
		t.Run("cluster_visit", func(t *testing.T) {
			v := newTestViewer(t)
			v.win.Resize(fyne.NewSize(1000, 700))
			a := uitest.TempGPSJPEGURI(t, "a.jpg", 24, 16, 52.52, 13.405)
			b := uitest.TempGPSJPEGURI(t, "b.jpg", 24, 16, 52.52, 13.405)
			c, holdC, enteredC, releaseC := locationHeldRead(t, uitest.TempGPSJPEGURI(t, "c.jpg", 24, 16, 52.52, 13.405))
			d, holdD, enteredD, releaseD := locationHeldRead(t, uitest.TempGPSJPEGURI(t, "d.jpg", 24, 16, 52.52, 13.405))
			dropAndWait(t, v, a, b, c, d)
			v.display.Settle()
			var requests atomic.Int32
			var returning atomic.Bool
			enteredTile, cancelledTile := make(chan struct{}, 128), make(chan struct{}, 128)
			pixels := uitest.EncodePNG(t, 256, 256, color.Black)
			v.locationMap.ConfigureTiles(locationmap.TileOptions{Client: &http.Client{Transport: locationTileTransport(func(request *http.Request) (*http.Response, error) {
				requests.Add(1)
				if !returning.Load() {
					enteredTile <- struct{}{}
					<-request.Context().Done()
					cancelledTile <- struct{}{}
					return nil, request.Context().Err()
				}
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(bytes.NewReader(pixels))}, nil
			})}}, noLocationRetry)
			holdC.Store(true)
			holdD.Store(true)
			queue := &uitest.UIQueue{}
			v.locationMap.SetUIQueue(queue)
			locationMenu(t, v).Action()
			<-enteredC
			queue.Drain()
			locationSurface(t, v).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(60, 30)})
			position := v.app.Driver().AbsolutePositionForObject(locationPhoto(t, v, a.Name()))
			<-enteredTile
			fynetest.Tap(locationButton(t, v, fmt.Sprintf(lang.L("%d images"), 2)))
			v.grid.Settle()
			<-cancelledTile
			releaseC()
			<-enteredD
			queue.Drain()
			if len(v.locationMap.Points()) != 3 || !slices.Equal(v.grid.ResultIndexes(), []int{0, 1}) {
				t.Fatal("Grid visit stopped discovery or changed frozen membership")
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
			waitUntilLoaded(t, v)
			releaseD()
			v.locationMap.Settle()
			if counts := v.locationMap.Counts(); !counts.Complete || counts.Located != 4 || v.locationMap.Visible() {
				t.Fatalf("cluster image visit stopped discovery: %+v", counts)
			}
			hiddenRequests := requests.Load()
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			v.grid.Settle()
			v.locationMap.Settle()
			if requests.Load() != hiddenRequests || !v.grid.Visible() {
				t.Fatal("hidden Grid return restarted map tiles")
			}
			returning.Store(true)
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			v.locationMap.Settle()
			if !v.locationMap.Visible() || requests.Load() <= hiddenRequests {
				t.Fatal("map return did not resume visible tile demand")
			}
			if got := v.app.Driver().AbsolutePositionForObject(locationPhoto(t, v, a.Name())); got != position {
				t.Fatalf("cluster discovery/visit moved manual camera: %v -> %v", position, got)
			}
			locationButton(t, v, fmt.Sprintf(lang.L("%d images"), 4))
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			v.locationMap.Settle()
			if v.locationMap.Active() {
				t.Fatal("cluster visit retained active map after final exit")
			}
		})
		t.Run("preparation_and_scan_retirement", func(t *testing.T) {
			v, release, queue := locationProgressFixture(t)
			queue.Drain()
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			if v.locationMap.Active() || v.locationMap.Visible() {
				t.Fatal("final exit waited for held source")
			}
			locationMenu(t, v).Action()
			release()
			v.locationMap.Settle()
			if len(v.locationMap.Points()) != 2 || !v.locationMap.Counts().Complete {
				t.Fatal("retired scan leaked into reopened session")
			}
		})
		t.Run("base_round_trip", func(t *testing.T) {
			v := newTestViewer(t)
			base := uitest.TempGPSJPEGURI(t, "held.jpg", 24, 16, 52.52, 13.405)
			data, err := os.ReadFile(base.Path())
			if err != nil {
				t.Fatal(err)
			}
			var hold atomic.Bool
			entered, release := make(chan struct{}), make(chan struct{})
			var releaseOnce sync.Once
			unblock := func() { releaseOnce.Do(func() { close(release) }) }
			t.Cleanup(unblock)
			u := uitest.ReaderURI(base, func() (io.ReadCloser, error) {
				r := bytes.NewReader(data)
				return uitest.ReadCloser{ReadFunc: func(p []byte) (int, error) {
					if hold.CompareAndSwap(true, false) {
						close(entered)
						<-release
					}
					return r.Read(p)
				}, CloseFunc: func() error { return nil }}, nil
			})
			dropAndWait(t, v, u)
			v.display.Settle()
			hold.Store(true)
			locationMenu(t, v).Action()
			<-entered
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			if v.locationMap.Active() || v.locationMap.Visible() {
				t.Fatal("close waited for source or kept map active")
			}
			fresh := uitest.TempGPSJPEGURI(t, "fresh.jpg", 24, 16, 0, 0)
			dropAndWait(t, v, fresh)
			locationMenu(t, v).Action()
			unblock()
			v.locationMap.Settle()
			points := v.locationMap.Points()
			if len(points) != 1 || points[0].Source.URI.Path() != fresh.Path() {
				t.Fatalf("retired scan replaced new collection: %v", points)
			}
			fynetest.Tap(locationPhoto(t, v, "fresh.jpg"))
			waitUntilLoaded(t, v)
			dropAndWait(t, v, base)
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			if v.FileCount() != 0 {
				t.Fatal("retired map image visit intercepted replacement collection Escape")
			}
			oldQueue := &uitest.UIQueue{}
			v.locationMap.SetUIQueue(oldQueue)
			dropAndWait(t, v, base)
			locationMenu(t, v).Action()
			v.locationMap.Wait()
			v.LeaveLocationMap()
			v.locationMap.SetUIQueue(&uitest.UIQueue{})
			dropAndWait(t, v, fresh)
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			oldQueue.Drain()
			points = v.locationMap.Points()
			if len(points) != 1 || points[0].Source.URI.Path() != fresh.Path() {
				t.Fatal("queued retired scan replaced current points")
			}
			v.locationMap.Stop()
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			if v.locationMap.Active() || v.locationMap.Visible() {
				t.Fatal("Stop admitted a new map session")
			}
		})
	})
	t.Run("scope_and_entry", func(t *testing.T) {
		v := newTestViewer(t)
		a := uitest.TempGPSJPEGURI(t, "berlin.jpg", 24, 16, 52.52, 13.405)
		b := uitest.TempGPSJPEGURI(t, "london.jpg", 24, 16, 51.507, -0.128)
		dropAndWait(t, v, a, b)
		v.grid.Toggle()
		for _, r := range "/berlin" {
			v.handleTypedRune(r)
		}
		v.grid.SelectAll()
		selected := v.grid.Selection()
		item := locationMenu(t, v)
		if item.Disabled {
			t.Fatal("Location Map disabled for loaded GPS photos")
		}
		item.Action()
		v.locationMap.Settle()
		if !item.Checked {
			t.Fatal("Location Map entry did not activate the feature")
		}
		for _, want := range []string{"berlin.jpg", "london.jpg"} {
			locationPhoto(t, v, want)
		}
		if os.Getenv("PICFETCH_LOCATION_MAP_RENDER") != "" {
			v.win.Resize(fyne.NewSize(1000, 700))
			v.locationMap.Settle()
			locationCapture(t, v, "01-map.png")
		}
		if !slices.Equal(v.grid.Selection(), selected) {
			t.Fatal("map entry discarded Grid selection")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		if v.locationMap.Visible() {
			t.Fatal("Escape did not leave the map")
		}
		v.keyModifiers = func() fyne.KeyModifier { return fyne.KeyModifierShift }
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyL})
		v.locationMap.Settle()
		if !v.locationMap.Visible() {
			t.Fatal("Shift+L did not open Location Map")
		}
	})
	t.Run("direct_image_visit", func(t *testing.T) {
		v := newTestViewer(t)
		a := uitest.TempGPSJPEGURI(t, "a.jpg", 24, 16, 52.52, 13.405)
		b := uitest.TempJPEGURI(t, "b.jpg", 24, 16, color.White)
		c := storage.NewFileURI(uitest.WriteTempFile(t, "c.jpg", uitest.GPSDateJPEG(t, 24, 16, 51.507, -0.128, "2026:09:25 12:34:56")))
		dropAndWait(t, v, a, b, c)
		locationMenu(t, v).Action()
		v.locationMap.Settle()
		photo := locationPhoto(t, v, "c.jpg")
		photo.MouseIn(&desktop.MouseEvent{})
		if v.state.index != 0 {
			t.Fatal("hover changed requested image")
		}
		if !locationFilenameVisible(t, v, "c.jpg") {
			t.Fatal("hover filename missing")
		}
		fynetest.Tap(photo)
		waitUntilLoaded(t, v)
		if v.state.index != 2 || v.locationMap.Visible() {
			t.Fatal("photo did not enter requested map image")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
		waitUntilLoaded(t, v)
		if v.state.index != 0 {
			t.Fatal("map image navigation escaped mapped collection order")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyLeft})
		waitUntilLoaded(t, v)
		if v.state.index != 2 {
			t.Fatal("map image reverse navigation escaped mapped collection order")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		v.locationMap.Settle()
		if !v.locationMap.Visible() || v.grid.Visible() {
			t.Fatal("direct image Escape did not return directly to map")
		}
	})
	t.Run("camera_retention", func(t *testing.T) {
		t.Run("automatic_rebuild", func(t *testing.T) {
			v := newTestViewer(t)
			v.win.Resize(fyne.NewSize(1000, 700))
			a := uitest.TempGPSJPEGURI(t, "a.jpg", 24, 16, 52.52, 13.405)
			b := uitest.TempGPSJPEGURI(t, "b.jpg", 24, 16, 40.7, -74)
			dropAndWait(t, v, a, b)
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			locationSurface(t, v).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(-35, 20)})
			before := v.app.Driver().AbsolutePositionForObject(locationPhoto(t, v, a.Name()))
			v.RemoveFile(1)
			v.grid.Settle()
			v.locationMap.Settle()
			after := v.app.Driver().AbsolutePositionForObject(locationPhoto(t, v, a.Name()))
			if before != after {
				t.Fatalf("rebuild moved manual camera: %v -> %v", before, after)
			}
			fynetest.Tap(locationButton(t, v, lang.L("Fit All")))
			v.locationMap.Settle()
			if v.app.Driver().AbsolutePositionForObject(locationPhoto(t, v, a.Name())) == before {
				t.Fatal("Fit All did not explicitly reframe survivors")
			}
		})
		t.Run("progressive_discoveries", func(t *testing.T) {
			v, release, queue := locationProgressFixture(t)
			queue.Drain()
			surface := locationSurface(t, v)
			surface.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(60, 30)})
			position := v.app.Driver().AbsolutePositionForObject(locationPhoto(t, v, "a.jpg"))
			release()
			v.locationMap.Settle()
			if got := v.app.Driver().AbsolutePositionForObject(locationPhoto(t, v, "a.jpg")); got != position {
				t.Fatalf("discovery moved manual camera: %v -> %v", position, got)
			}
			fynetest.Tap(locationButton(t, v, lang.L("Fit All")))
			v.locationMap.Settle()
			locationPhoto(t, v, "b.jpg")
		})
		t.Run("direct_visit", func(t *testing.T) {
			v := newTestViewer(t)
			a := uitest.TempGPSJPEGURI(t, "a.jpg", 24, 16, 52.52, 13.405)
			dropAndWait(t, v, a)
			v.win.Resize(fyne.NewSize(1000, 700))
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			surface := locationSurface(t, v)
			drag, ok := any(surface).(fyne.Draggable)
			if !ok {
				t.Fatal("map does not accept panning")
			}
			position := func() fyne.Position { return v.app.Driver().AbsolutePositionForObject(locationPhoto(t, v, "a.jpg")) }
			before := position()
			drag.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(60, 30)})
			drag.DragEnd()
			v.locationMap.Settle()
			panned := position()
			if panned.X-before.X != 60 || panned.Y-before.Y != 30 {
				t.Fatalf("pan moved %v to %v", before, panned)
			}
			fynetest.Tap(locationPhoto(t, v, "a.jpg"))
			waitUntilLoaded(t, v)
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			v.locationMap.Settle()
			if position() != panned {
				t.Fatal("direct visit moved the camera")
			}
			fynetest.Tap(locationButton(t, v, lang.L("Fit All")))
			v.locationMap.Settle()
			if position() != before {
				t.Fatal("Fit All did not frame locations")
			}
		})
	})
	t.Run("selection_feedback", func(t *testing.T) {
		t.Run("duplicate_provenance", func(t *testing.T) {
			v := newTestViewer(t)
			donor := uitest.PatternedGPSJPEGURI(t, "a-donor.jpg", 1, 64, 48, 52.52, 13.405)
			rep := uitest.PatternedJPEGURISize(t, "z-representative.jpg", 1, 192, 144)
			dropAndWait(t, v, donor, rep)
			v.menus.Actions().Hide().Action()
			v.grid.Settle()
			v.ShowImage(0)
			waitUntilLoaded(t, v)
			if displayed, ok := v.displayedFile(); !ok || displayed.Path() != donor.Path() || !v.dupes.Visibility().HiddenExtra(0) {
				t.Fatal("fixture does not display a hidden duplicate member")
			}
			locationMenu(t, v).Action()
			v.grid.Settle()
			v.locationMap.Settle()
			highlighted := false
			explorerWalk(locationPhoto(t, v, rep.Name()), func(object fyne.CanvasObject) {
				if frame, ok := object.(*canvas.Rectangle); ok && frame.StrokeWidth == 2 && frame.StrokeColor == theme.Color(theme.ColorNamePrimary) {
					highlighted = true
				}
			})
			if !highlighted {
				t.Fatal("displayed hidden member did not highlight its current representative")
			}
		})
		t.Run("direct_images", func(t *testing.T) {
			v := newTestViewer(t)
			a := uitest.TempGPSJPEGURI(t, "a.jpg", 24, 16, 52.52, 13.405)
			b := uitest.TempGPSJPEGURI(t, "b.jpg", 24, 16, 51.507, -0.128)
			dropAndWait(t, v, a, b)
			v.grid.Toggle()
			v.grid.SelectAll()
			selected := v.grid.Selection()
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			highlighted := func(name string) bool {
				found := false
				explorerWalk(locationPhoto(t, v, name), func(object fyne.CanvasObject) {
					if frame, ok := object.(*canvas.Rectangle); ok && frame.StrokeWidth == 2 && frame.StrokeColor == theme.Color(theme.ColorNamePrimary) {
						found = true
					}
				})
				return found
			}
			if !highlighted("a.jpg") {
				t.Fatal("displayed image is not highlighted")
			}
			locationPhoto(t, v, "b.jpg").MouseIn(&desktop.MouseEvent{})
			if !highlighted("a.jpg") || highlighted("b.jpg") {
				t.Fatal("hover replaced displayed-image highlight")
			}
			if !slices.Equal(selected, v.grid.Selection()) {
				t.Fatal("hover discarded Grid selection")
			}
		})
	})
}

type locationTileTransport func(*http.Request) (*http.Response, error)

func preparationBars(root fyne.CanvasObject) (*widget.ProgressBar, *widget.ProgressBarInfinite) {
	var progress *widget.ProgressBar
	var waiting *widget.ProgressBarInfinite
	explorerWalk(root, func(object fyne.CanvasObject) {
		switch bar := object.(type) {
		case *widget.ProgressBar:
			progress = bar
		case *widget.ProgressBarInfinite:
			waiting = bar
		}
	})
	return progress, waiting
}

func locationGPSRecords(t *testing.T, dir string) []string {
	t.Helper()
	var records []string
	err := filepath.WalkDir(filepath.Join(dir, ".location-map"), func(path string, entry os.DirEntry, err error) error {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(path, ".json") {
			records = append(records, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return records
}

type locationNoticeQueue struct {
	*uitest.UIQueue
	ready chan struct{}
}

func (q *locationNoticeQueue) Do(fn func()) {
	q.UIQueue.Do(fn)
	select {
	case q.ready <- struct{}{}:
	default:
	}
}

func (transport locationTileTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func noLocationRetry(_ context.Context, _ time.Duration) error { return context.Canceled }

// locationTileRetryFixture holds one failed replacement tile while the others
// complete, until finishRetry advances time and delivers its successful retry.
func locationTileRetryFixture(t *testing.T, v *viewer, oldTile, newTile []byte) (startReplacement, finishRetry func()) {
	t.Helper()
	var replacing, failed atomic.Bool
	var seconds atomic.Int64
	waiting, permit := make(chan struct{}, 1), make(chan struct{})
	unblock := sync.OnceFunc(func() { close(permit) })
	t.Cleanup(unblock)
	v.locationMap.ConfigureTiles(locationmap.TileOptions{Now: func() time.Time { return time.Unix(1000+seconds.Load(), 0) }, Client: &http.Client{Transport: locationTileTransport(func(_ *http.Request) (*http.Response, error) {
		pixels := oldTile
		if replacing.Load() {
			if failed.CompareAndSwap(false, true) {
				return &http.Response{StatusCode: 503, Header: http.Header{"Retry-After": []string{"2"}}, Body: http.NoBody}, nil
			}
			pixels = newTile
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Cache-Control": []string{"no-cache"}}, Body: io.NopCloser(bytes.NewReader(pixels))}, nil
	})}}, func(ctx context.Context, _ time.Duration) error {
		waiting <- struct{}{}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-permit:
			return nil
		}
	})
	return func() { replacing.Store(true) }, func() {
		<-waiting
		seconds.Store(2)
		unblock()
		v.locationMap.Wait()
		v.locationMap.Settle()
	}
}

// The first mapped source is complete while the second actual source read waits.
func locationProgressFixture(t *testing.T) (*viewer, func(), *uitest.UIQueue) {
	t.Helper()
	v := newTestViewer(t)
	a := uitest.TempGPSJPEGURI(t, "a.jpg", 24, 16, 52.52, 13.405)
	b := uitest.TempGPSJPEGURI(t, "b.jpg", 24, 16, 40.7, -74)
	b, hold, entered, unblock := locationHeldRead(t, b)
	dropAndWait(t, v, a, b)
	v.display.Settle()
	hold.Store(true)
	queue := &uitest.UIQueue{}
	v.locationMap.SetUIQueue(queue)
	v.win.Resize(fyne.NewSize(1000, 700))
	locationMenu(t, v).Action()
	<-entered
	return v, unblock, queue
}

func locationHeldRead(t *testing.T, source fyne.URI) (fyne.URI, *atomic.Bool, <-chan struct{}, func()) {
	t.Helper()
	data, err := os.ReadFile(source.Path())
	if err != nil {
		t.Fatal(err)
	}
	var hold atomic.Bool
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	source = uitest.ReaderURI(source, func() (io.ReadCloser, error) {
		r := bytes.NewReader(data)
		return uitest.ReadCloser{ReadFunc: func(p []byte) (int, error) {
			if hold.CompareAndSwap(true, false) {
				close(entered)
				<-release
			}
			return r.Read(p)
		}, CloseFunc: func() error { return nil }}, nil
	})
	return source, &hold, entered, unblock
}

func locationSurface(t *testing.T, v *viewer) *locationmap.Surface {
	t.Helper()
	var found *locationmap.Surface
	explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
		if s, ok := o.(*locationmap.Surface); ok {
			found = s
		}
	})
	if found == nil {
		t.Fatal("geographic surface not mounted")
	}
	return found
}

func locationButton(t *testing.T, v *viewer, text string) *widget.Button {
	t.Helper()
	var found *widget.Button
	explorerWalk(v.win.Content(), func(o fyne.CanvasObject) {
		if b, ok := o.(*widget.Button); ok && b.Text == text {
			found = b
		}
	})
	if found == nil {
		t.Fatalf("button %q missing from mounted surface", text)
	}
	return found
}

func locationFilenameVisible(t *testing.T, v *viewer, name string) bool {
	t.Helper()
	visible := false
	explorerWalk(locationSurface(t, v), func(object fyne.CanvasObject) {
		if label, ok := object.(*widget.Label); ok && label.Visible() && label.Text == name {
			visible = true
		}
	})
	return visible
}

func locationPhoto(t *testing.T, v *viewer, names ...string) *widgets.TappableArea {
	t.Helper()
	var found *widgets.TappableArea
	explorerWalk(locationSurface(t, v), func(object fyne.CanvasObject) {
		if area, ok := object.(*widgets.TappableArea); ok {
			area.MouseIn(&desktop.MouseEvent{})
			for _, name := range names {
				if locationFilenameVisible(t, v, name) {
					found = area
				}
			}
			area.MouseOut()
		}
	})
	if found == nil {
		t.Fatalf("photos %q missing from mounted surface", names)
	}
	return found
}

func locationAssertTileColor(t *testing.T, v *viewer, want color.NRGBA) {
	t.Helper()
	count := 0
	explorerWalk(locationSurface(t, v), func(object fyne.CanvasObject) {
		if img, ok := object.(*canvas.Image); ok && img.Image != nil && img.Image.Bounds().Dx() == 256 {
			count++
			if got := color.NRGBAModel.Convert(img.Image.At(0, 0)); got != want {
				t.Fatalf("incomplete or obsolete tile scene: pixel %v, want %v", got, want)
			}
		}
	})
	if count == 0 {
		t.Fatal("painted tile scene was cleared")
	}
}

func locationCapture(t *testing.T, v *viewer, name string) {
	t.Helper()
	directory := os.Getenv("PICFETCH_LOCATION_MAP_RENDER")
	if directory == "" {
		return
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, v.win.Canvas().Capture()); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, name), encoded.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
}
