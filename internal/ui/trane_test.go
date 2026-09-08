package ui

import (
	"bytes"
	"image"
	"image/draw"
	"math"
	"os"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"golang.org/x/image/webp"
)

// Walk the actual surface, including widget renderers, so a detached portrait
// cannot satisfy the welcome-screen assertion.
func traneSurfaceImage(object, target fyne.CanvasObject, inside bool) *canvas.Image {
	inside = inside || object == target
	if picture, ok := object.(*canvas.Image); ok && inside {
		return picture
	}
	var children []fyne.CanvasObject
	switch object := object.(type) {
	case *fyne.Container:
		children = object.Objects
	case fyne.Widget:
		children = test.WidgetRenderer(object).Objects()
	}
	for _, child := range children {
		if picture := traneSurfaceImage(child, target, inside); picture != nil {
			return picture
		}
	}
	return nil
}

var canonicalTraneAtlas = sync.OnceValues(func() (image.Image, error) {
	data, err := os.ReadFile("../../assets/trane/codex-pet/spritesheet.webp")
	if err != nil {
		return nil, err
	}
	return webp.Decode(bytes.NewReader(data))
})

func assertTraneFrame(t *testing.T, picture *canvas.Image, column, row int) {
	t.Helper()
	if picture == nil || picture.Image == nil {
		t.Fatal("welcome surface does not contain a decoded Trane pet frame")
	}
	atlas, err := canonicalTraneAtlas()
	if err != nil {
		t.Fatal(err)
	}
	bounds := image.Rect(0, 0, 192, 208)
	want, got := image.NewNRGBA(bounds), image.NewNRGBA(bounds)
	draw.Draw(want, bounds, atlas, image.Pt(column*192, row*208), draw.Src)
	if picture.Image.Bounds() != bounds {
		t.Fatalf("Trane frame bounds = %v, want %v", picture.Image.Bounds(), bounds)
	}
	draw.Draw(got, bounds, picture.Image, image.Point{}, draw.Src)
	for offset := 0; offset < len(got.Pix); offset += 4 {
		before, after := want.Pix[offset:offset+4], got.Pix[offset:offset+4]
		pink := int(before[0]) > int(before[1])+35 && int(before[2]) > int(before[1])+20
		if after[3] != before[3] || (!pink && !bytes.Equal(before, after)) {
			t.Fatalf("Trane changed the pose or non-fringe pixels in column %d, row %d", column, row)
		}
	}
}

func TestTraneWelcomeUsesPet(t *testing.T) {
	v, win, _ := newTestUI(t)
	assertTraneFrame(t, traneSurfaceImage(win.Content(), v.welcomeArt, false), 6, 0)
}

func TestTraneFollowsCursor(t *testing.T) {
	v, win, _ := newTestUI(t)
	picture := traneSurfaceImage(win.Content(), v.welcomeArt, false)
	for _, size := range []fyne.Size{fyne.NewSize(520, 340), fyne.NewSize(780, 560)} {
		win.Resize(size)
		scale := min(picture.Size().Width/192, picture.Size().Height/208)
		origin := v.app.Driver().AbsolutePositionForObject(picture).Add(fyne.NewPos(
			picture.Size().Width/2, (picture.Size().Height-208*scale)/2+64*scale))
		for sector := range 16 {
			angle := float64(sector) * math.Pi / 8
			pointer := origin.Add(fyne.NewPos(float32(math.Sin(angle)*60), float32(-math.Cos(angle)*60)))
			v.dropzoneArt.MouseMoved(&desktop.MouseEvent{AbsolutePosition: pointer})
			assertTraneFrame(t, picture, sector%8, 9+sector/8)
		}
		// The face itself has a neutral dead zone instead of flickering
		// between opposing sectors for tiny pointer movements.
		v.dropzoneArt.MouseMoved(&desktop.MouseEvent{AbsolutePosition: origin})
		assertTraneFrame(t, picture, 6, 0)
		v.dropzoneArt.MouseMoved(&desktop.MouseEvent{AbsolutePosition: origin.Add(fyne.NewPos(20*scale, 0))})
		assertTraneFrame(t, picture, 4, 9)
		v.dropzoneArt.MouseIn(&desktop.MouseEvent{AbsolutePosition: origin.Add(fyne.NewPos(60, 0))})
		assertTraneFrame(t, picture, 4, 9)
		if size.Width == 520 {
			// A stationary pointer still guides the pet when layout changes.
			win.Resize(fyne.NewSize(780, 560))
			assertTraneFrame(t, picture, 5, 10)
		}
		v.dropzoneArt.MouseOut()
		assertTraneFrame(t, picture, 6, 0)
	}
}

func TestTraneOnlyMovesForPointer(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		v, win, _ := newTestUI(t)
		startViewerRuntime(v, win, t.TempDir())
		picture := traneSurfaceImage(win.Content(), v.welcomeArt, false)
		for _, duration := range []time.Duration{3 * time.Second, 7 * time.Second, time.Minute} {
			time.Sleep(duration)
			synctest.Wait()
			assertTraneFrame(t, picture, 6, 0)
		}
		v.dropzoneArt.MouseIn(&desktop.MouseEvent{})
		looking := picture.Image
		time.Sleep(time.Minute)
		synctest.Wait()
		if picture.Image != looking {
			t.Fatal("Trane changed pose without pointer movement")
		}
		v.welcomeArt.Hide()
		v.showWelcomeState()
		assertTraneFrame(t, picture, 6, 0)
	})
}

func TestTraneCleansPinkFringe(t *testing.T) {
	v, win, _ := newTestUI(t)
	picture := traneSurfaceImage(win.Content(), v.welcomeArt, false)
	origin := v.app.Driver().AbsolutePositionForObject(picture).Add(fyne.NewPos(
		picture.Size().Width/2, picture.Size().Height/2-(104-64)*picture.Size().Width/192))
	v.dropzoneArt.MouseMoved(&desktop.MouseEvent{AbsolutePosition: origin.Add(fyne.NewPos(-60, 0))})
	assertTraneFrame(t, picture, 4, 10)
	// This opaque magenta contamination is in the supplied left-facing
	// frame's paw edge. The tongue and collar are not at this location.
	r, g, b, a := picture.Image.At(91, 197).RGBA()
	if r > g+35*257 && b > g+20*257 {
		t.Fatal("pink background fringe remains on Trane's paw")
	}
	// A deeper pink pixel in the same paw must remain untouched: the
	// filter is spatially bounded, not a global color replacement.
	atlas, err := canonicalTraneAtlas()
	if err != nil {
		t.Fatal(err)
	}
	if picture.Image.At(91, 190) != atlas.At(4*192+91, 10*208+190) {
		t.Fatal("fringe repair changed an interior color")
	}
	if a != 65535 {
		t.Fatal("fringe repair erased the paw instead of preserving its alpha")
	}
}
