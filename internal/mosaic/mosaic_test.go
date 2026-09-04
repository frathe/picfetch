package mosaic

import (
	"errors"
	"image"
	"image/color"
	"math"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
)

func TestSettingsDefaultsAndRanges(t *testing.T) {
	want := Settings{
		MinimumShortEdge: 0.18,
		SizeVariation:    0.12,
		Overlap:          0.08,
		MaximumRotation:  7,
		Frame:            FrameNone,
		DropShadow:       true,
	}
	if got := DefaultSettings(); got != want {
		t.Fatalf("DefaultSettings() = %+v, want %+v", got, want)
	}

	valid := []Settings{
		{MinimumShortEdge: 0.10, SizeVariation: 0, Overlap: 0, MaximumRotation: 0, Frame: FrameNone, DropShadow: false},
		{MinimumShortEdge: 0.30, SizeVariation: 0.25, Overlap: 0.20, MaximumRotation: 12, Frame: FramePolaroid, DropShadow: true},
	}
	for _, settings := range valid {
		if err := settings.Validate(); err != nil {
			t.Errorf("Settings.Validate(%+v) = %v, want nil", settings, err)
		}
	}
}

func TestSettingsNormalizationPreservesDropShadowChoice(t *testing.T) {
	settings := DefaultSettings()
	settings.DropShadow = false
	if got := settings.Normalized(); got.DropShadow {
		t.Fatal("Settings.Normalized() enabled an explicitly disabled drop shadow")
	}
}

func TestSettingsValidationIdentifiesField(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Settings)
		field  string
	}{
		{name: "minimum below range", mutate: func(s *Settings) { s.MinimumShortEdge = 0.09 }, field: "minimum_short_edge"},
		{name: "variation above range", mutate: func(s *Settings) { s.SizeVariation = 0.26 }, field: "size_variation"},
		{name: "overlap NaN", mutate: func(s *Settings) { s.Overlap = math.NaN() }, field: "overlap"},
		{name: "rotation infinite", mutate: func(s *Settings) { s.MaximumRotation = math.Inf(1) }, field: "maximum_rotation"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := DefaultSettings()
			tt.mutate(&settings)
			var validation *ValidationError
			if err := settings.Validate(); !errors.As(err, &validation) || validation.Field != tt.field {
				t.Fatalf("Settings.Validate() = %v, want ValidationError for %q", err, tt.field)
			}
		})
	}
}

func TestFrameStylePreferenceValuesAndNormalization(t *testing.T) {
	tests := []struct {
		value string
		want  FrameStyle
	}{
		{value: "none", want: FrameNone},
		{value: "thin-light", want: FrameThinLight},
		{value: "thin-dark", want: FrameThinDark},
		{value: "polaroid", want: FramePolaroid},
		{value: "future-style", want: FrameNone},
	}
	for _, tt := range tests {
		if got := FrameStyleFromPreference(tt.value); got != tt.want {
			t.Errorf("FrameStyleFromPreference(%q) = %q, want %q", tt.value, got, tt.want)
		}
	}
}

func TestRequestValidatesAndOwnsSources(t *testing.T) {
	sources := []fyne.URI{storage.NewFileURI("/one.png"), storage.NewFileURI("/two.jpg")}
	request, err := NewRequest(sources, image.Pt(1920, 1080), DefaultSettings(), 42)
	if err != nil {
		t.Fatalf("NewRequest() = %v", err)
	}

	sources[0] = storage.NewFileURI("/changed.png")
	got := request.Sources()
	if got[0].Path() != "/one.png" {
		t.Fatalf("request source changed through caller slice: %q", got[0].Path())
	}
	got[0] = storage.NewFileURI("/also-changed.png")
	if request.Sources()[0].Path() != "/one.png" {
		t.Fatal("request source changed through accessor slice")
	}
	if request.TargetSize() != image.Pt(1920, 1080) || request.Settings() != DefaultSettings() || request.Seed() != 42 {
		t.Fatalf("request accessors returned the wrong values")
	}
}

func TestRequestValidationIdentifiesField(t *testing.T) {
	good := []fyne.URI{storage.NewFileURI("/one.png")}
	tests := []struct {
		name    string
		sources []fyne.URI
		target  image.Point
		field   string
	}{
		{name: "empty sources", target: image.Pt(1, 1), field: "sources"},
		{name: "nil source", sources: []fyne.URI{nil}, target: image.Pt(1, 1), field: "sources[0]"},
		{name: "zero width", sources: good, target: image.Pt(0, 1), field: "target_width"},
		{name: "negative height", sources: good, target: image.Pt(1, -1), field: "target_height"},
		{name: "overflowing area", sources: good, target: image.Pt(math.MaxInt, 2), field: "target_area"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewRequest(tt.sources, tt.target, DefaultSettings(), 1)
			var validation *ValidationError
			if !errors.As(err, &validation) || validation.Field != tt.field {
				t.Fatalf("NewRequest() = %v, want ValidationError for %q", err, tt.field)
			}
		})
	}
}

func TestResultOwnsPixels(t *testing.T) {
	pixels := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	pixels.Set(0, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	result := newResult(pixels)

	pixels.Set(0, 0, color.NRGBA{R: 200, A: 255})
	first := result.Image().(*image.NRGBA)
	if got := first.NRGBAAt(0, 0); got != (color.NRGBA{R: 10, G: 20, B: 30, A: 255}) {
		t.Fatalf("Result pixel = %v after source mutation", got)
	}
	first.Set(0, 0, color.NRGBA{G: 200, A: 255})
	if got := result.Image().(*image.NRGBA).NRGBAAt(0, 0); got != (color.NRGBA{R: 10, G: 20, B: 30, A: 255}) {
		t.Fatalf("Result pixel = %v after accessor mutation", got)
	}
	if result.Bounds() != image.Rect(0, 0, 2, 1) {
		t.Fatalf("Result.Bounds() = %v", result.Bounds())
	}
}
