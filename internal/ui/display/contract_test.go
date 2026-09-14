package display_test

import (
	"image"
	"image/color"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/ui/display"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestPresentationContract(t *testing.T) {
	t.Run("animation", animationContract)
	t.Run("vector", vectorContract)
	t.Run("load", loadContract)
	t.Run("preloads", preloadContract)
	t.Run("captures", func(t *testing.T) {
		for _, reset := range []bool{false, true} {
			t.Run(map[bool]string{false: "later turn", true: "later reset"}[reset], func(t *testing.T) {
				f := newPresentation(t, display.Config{})
				t.Cleanup(f.Stop)
				source := storage.NewFileURI("/saved.png")
				pixels := image.NewNRGBA(image.Rect(0, 0, 2, 3))
				loaded := &imaging.LoadedImage{Frames: []image.Image{pixels}}
				f.Present(source, loaded, false)
				f.RotateBy(1)
				capture, ok := f.Capture()
				if !ok || capture.Pixels != f.Surface().Image || capture.Identity.Source.String() != source.String() {
					t.Fatal("capture did not retain the published content and identity")
				}
				if reset {
					f.ResetRotation()
				} else {
					f.RotateBy(1)
				}
				visible := f.Surface().Image
				if !f.ReconcileSaved(capture) {
					t.Fatal("current committed capture was rejected")
				}
				wantRotation := 1
				if reset {
					wantRotation = 3
				}
				if f.Snapshot().Rotation != wantRotation || f.Surface().Image != visible {
					t.Fatal("save reconciliation discarded a later view adjustment or republished pixels")
				}
				if capture.Rotation != 1 || capture.Pixels.Bounds().Size() != image.Pt(3, 2) {
					t.Fatal("capture changed after the view changed")
				}
				f.ResetRotation()
				if f.Surface().Image != capture.Pixels || f.Snapshot().Size != fyne.NewSize(3, 2) {
					t.Fatal("reset did not use the saved baseline")
				}
				f.Load(display.Request{Source: storage.NewFileURI("/pending-other.png")})
				if f.ReconcileSaved(capture) {
					t.Fatal("new navigation accepted outgoing save reconciliation")
				}
				outgoing, ok := f.Capture()
				if !ok || outgoing.Identity != capture.Identity {
					t.Fatal("outgoing capture was labeled with requested content")
				}
				f.Present(source, loaded, false)
				if f.ReconcileSaved(outgoing) {
					t.Fatal("same-source reopen accepted retired capture")
				}
			})
		}
	})
	t.Run("surface", func(t *testing.T) {
		f := newPresentation(t, display.Config{})
		t.Cleanup(f.Stop)
		root := container.NewStack(f.Surface())
		if !contains(root, f.Surface()) {
			t.Fatal("presentation surface is missing from composition")
		}
		if f.Surface().Visible() {
			t.Fatal("empty surface is visible")
		}
		pixels := image.NewNRGBA(image.Rect(0, 0, 2, 3))
		pixels.Set(0, 0, color.NRGBA{R: 255, A: 255})
		source := storage.NewFileURI("/presentation.png")
		loaded := &imaging.LoadedImage{Frames: []image.Image{pixels}}
		f.Present(source, loaded, false)
		first := f.Snapshot()
		if first.Displayed.Source == nil || first.Displayed.Source.String() != source.String() {
			t.Fatal("presented content has no source identity")
		}
		if !f.Surface().Visible() || f.Surface().Image != pixels {
			t.Fatal("initial pixels were not published")
		}
		f.RotateBy(1)
		if f.Surface().Image.Bounds().Size() != image.Pt(3, 2) {
			t.Fatal("rotation did not publish turned pixels")
		}
		if got := color.NRGBAModel.Convert(f.Surface().Image.At(2, 0)); got != (color.NRGBA{R: 255, A: 255}) {
			t.Fatalf("clockwise rotation lost the red corner: %v", got)
		}
		if f.Snapshot().Size != fyne.NewSize(3, 2) {
			t.Fatal("logical size did not turn")
		}
		if !f.ResetRotation() || f.Surface().Image != pixels {
			t.Fatal("reset did not restore original pixels")
		}
		f.FadeTo(0, 1, time.Second)
		if f.Surface().Translucency != 1 {
			t.Fatal("fade did not reach its endpoint")
		}
		f.ResetFade()
		if f.Surface().Translucency != 0 {
			t.Fatal("reset left the image faded")
		}
		f.Present(source, loaded, false)
		if f.Snapshot().Displayed.Revision == first.Displayed.Revision {
			t.Fatal("reopening retained the old presentation identity")
		}
		if first.Rotation != 0 || first.Size != fyne.NewSize(2, 3) {
			t.Fatal("observation changed after later presentation")
		}
		other := storage.NewFileURI("/next.png")
		f.Load(display.Request{Source: other})
		pending := f.Snapshot()
		if !pending.Loading || pending.Requested.Source.String() != other.String() || pending.Displayed.Source.String() != source.String() || f.Surface().Image != pixels {
			t.Fatal("pending navigation lost the distinction between requested and displayed content")
		}
		f.Clear()
		if f.Surface().Visible() || f.Surface().Image != nil || f.Snapshot().Displayed.Source != nil {
			t.Fatal("clear retained displayed content")
		}
		f.Present(source, loaded, false)
		if !f.Surface().Visible() {
			t.Fatal("ordinary clear prevented reopening")
		}
		f.Stop()
		f.Present(source, loaded, false)
		if f.Surface().Visible() {
			t.Fatal("terminal stop admitted new content")
		}
		f.FadeTo(0, 1, time.Second)
		if f.Surface().Translucency != 0 {
			t.Fatal("terminal stop admitted a new fade")
		}
	})
}

func contains(root fyne.CanvasObject, wanted fyne.CanvasObject) bool {
	if root == wanted {
		return true
	}
	if c, ok := root.(*fyne.Container); ok {
		for _, child := range c.Objects {
			if contains(child, wanted) {
				return true
			}
		}
	}
	return false
}

// Present exercises the final loading interface through a real synchronous cache hit.
type presentationFixture struct {
	*display.Feature
	cache *imaging.ByteCache[*imaging.LoadedImage]
}

func newPresentation(t *testing.T, config display.Config) *presentationFixture {
	t.Helper()
	if config.Cache == nil {
		config.Cache = imaging.NewImgCache(256 * 1024 * 1024)
	}
	if config.Queue == nil {
		config.Queue = &uitest.UIQueue{}
	}
	f := &presentationFixture{Feature: display.New(config), cache: config.Cache}
	t.Cleanup(func() { f.Stop(); f.Wait(); f.Settle() })
	return f
}
func (f *presentationFixture) Present(source fyne.URI, loaded *imaging.LoadedImage, transition bool) {
	f.cache.Add(source.String(), loaded)
	f.Load(display.Request{Source: source, Transition: transition})
}
