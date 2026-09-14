package display

import (
	"image"
	"testing"

	"fyne.io/fyne/v2"
)

func TestVectorNeedsRender(t *testing.T) {
	for _, tc := range []struct {
		name       string
		have, want image.Point
		expect     bool
	}{
		{"nothing rendered yet", image.Point{}, image.Pt(340, 340), true},
		{"meaningfully sharper wanted", image.Pt(340, 340), image.Pt(680, 680), true},
		{"within the hysteresis band", image.Pt(340, 340), image.Pt(350, 350), false},
		{"slightly smaller wanted", image.Pt(340, 340), image.Pt(300, 300), false},
		{"grossly oversized, release it", image.Pt(2720, 2720), image.Pt(340, 340), true},
		{"degenerate target is never worth producing", image.Pt(100, 100), image.Pt(0, 0), false},
		{"exactly the sharpen boundary holds still", image.Pt(100, 100), image.Pt(105, 105), false},
		{"just past the sharpen boundary re-renders", image.Pt(100, 100), image.Pt(106, 106), true},
		{"exactly the release boundary holds still", image.Pt(100, 100), image.Pt(50, 50), false},
		{"just past the release boundary re-renders", image.Pt(100, 100), image.Pt(49, 49), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := vectorNeedsRender(tc.have, tc.want); got != tc.expect {
				t.Fatalf("vectorNeedsRender(%v, %v) = %v, want %v", tc.have, tc.want, got, tc.expect)
			}
		})
	}
}

func TestVectorRasterTarget(t *testing.T) {
	times2 := func(p fyne.Position) (int, int) {
		return int(p.X*2 + 0.5), int(p.Y*2 + 0.5)
	}

	for _, tc := range []struct {
		name     string
		logical  fyne.Size
		scale    float32
		toPixels func(fyne.Position) (int, int)
		wantW    int
		wantH    int
	}{
		{"1x fit", fyne.NewSize(340, 340), 1, nil, 340, 340},
		{"1x one zoom step", fyne.NewSize(340, 340), 1.25, nil, 425, 425},
		{"2x fit (Retina)", fyne.NewSize(340, 340), 1, times2, 680, 680},
		{"2x one zoom step", fyne.NewSize(340, 340), 1.25, times2, 850, 850},
		{"wide 2x", fyne.NewSize(520, 260), 1, times2, 1040, 520},
		{"non-positive scale is zero", fyne.NewSize(340, 340), 0, times2, 0, 0},
		{"empty logical is zero", fyne.Size{}, 1.25, times2, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, h := vectorRasterTarget(tc.logical, tc.scale, tc.toPixels)
			if w != tc.wantW || h != tc.wantH {
				t.Fatalf("vectorRasterTarget = %dx%d, want %dx%d", w, h, tc.wantW, tc.wantH)
			}
		})
	}
}
