package spiral

import (
	"context"
	"encoding/hex"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/heic"
)

type spiralHEICBackend struct{ exif []byte }

func (spiralHEICBackend) Check(_ context.Context) error { return nil }
func (b spiralHEICBackend) Read(_ context.Context, _ []byte, request heic.Request) (heic.Result, error) {
	result := heic.Result{Width: 2, Height: 1, EXIF: b.exif}
	if request.Pixels {
		result.Stride = 8
		result.Pixels = []byte{255, 0, 0, 255, 0, 255, 0, 255}
	}
	return result, nil
}

func TestHEICSpiralPreview(t *testing.T) {
	s := newTestSpiral(t)
	exif, err := hex.DecodeString("49492a0008000000010012010300010000000600000000000000")
	if err != nil {
		t.Fatal(err)
	}
	capability := heic.NewCapability(spiralHEICBackend{exif: exif}, heic.SystemIdentity(), heic.Observation{})
	t.Cleanup(func() { capability.Stop(); capability.Wait() })
	<-capability.Check(t.Context())
	s.SetHEICCapability(capability)
	data, err := os.ReadFile("../../heic/testdata/probe8.heic")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "preview.heic")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	sources := []fyne.URI{storage.NewFileURI(path)}
	s.Show(sources)
	s.win.Resize(fyne.NewSize(800, 600))
	capability.Invalidate(capability.State().Generation)
	settleTunnelPreviews(s)
	pixels := s.shader.Textures["traveller0"]
	if pixels.Bounds() != image.Rect(0, 0, 2, 1) {
		t.Fatalf("captured native preview did not reach the shader: %v", pixels.Bounds())
	}
	for x, want := range []color.NRGBA{{R: 255, A: 255}, {G: 255, A: 255}} {
		if got := color.NRGBAModel.Convert(pixels.At(x, 0)); got != want {
			t.Fatalf("canonical pixel %d = %v, want %v", x, got, want)
		}
	}
	s.Close()
	s.Settle()
	s.Show(sources)
	s.win.Resize(fyne.NewSize(800, 600))
	settleTunnelPreviews(s)
	if s.shader.Textures["traveller0"].Bounds() != image.Rect(0, 0, 1, 1) || s.shader.Uniforms["traveller0Active"] != float32(0) {
		t.Fatal("new session reused the retired available capability")
	}
}
