// Package mosaic generates immutable, display-sized image mosaics.
package mosaic

import (
	"fmt"
	"image"
	"image/draw"
	"math"
	"slices"

	"fyne.io/fyne/v2"
)

// FrameStyle is the locale-independent frame value used by generation and
// standing preferences.
type FrameStyle string

const (
	FrameNone      FrameStyle = "none"
	FrameThinLight FrameStyle = "thin-light"
	FrameThinDark  FrameStyle = "thin-dark"
	FramePolaroid  FrameStyle = "polaroid"
)

// FrameStyleFromPreference restores a persisted frame value. Unknown values
// deliberately fall back to no frame so a future or corrupt value stays safe.
func FrameStyleFromPreference(value string) FrameStyle {
	style := FrameStyle(value)
	switch style {
	case FrameNone, FrameThinLight, FrameThinDark, FramePolaroid:
		return style
	default:
		return FrameNone
	}
}

// Settings contains the visual controls for one mosaic generation. Ratios are
// expressed as fractions and rotation is expressed in degrees.
type Settings struct {
	MinimumShortEdge float64
	SizeVariation    float64
	Overlap          float64
	MaximumRotation  float64
	Frame            FrameStyle
	DropShadow       bool
}

// DefaultSettings returns the product defaults.
func DefaultSettings() Settings {
	return Settings{
		MinimumShortEdge: 0.18,
		SizeVariation:    0.12,
		Overlap:          0.08,
		MaximumRotation:  7,
		Frame:            FrameNone,
		DropShadow:       true,
	}
}

// ValidationError identifies the request field whose value is invalid.
type ValidationError struct {
	Field string
	Err   error
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("invalid mosaic %s: %v", e.Field, e.Err)
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

// Validate checks every configured numeric range. Zero remains a valid value
// for variation, overlap, and rotation.
func (s Settings) Validate() error {
	checks := []struct {
		field   string
		value   float64
		minimum float64
		maximum float64
	}{
		{field: "minimum_short_edge", value: s.MinimumShortEdge, minimum: 0.10, maximum: 0.30},
		{field: "size_variation", value: s.SizeVariation, minimum: 0, maximum: 0.25},
		{field: "overlap", value: s.Overlap, minimum: 0, maximum: 0.20},
		{field: "maximum_rotation", value: s.MaximumRotation, minimum: 0, maximum: 12},
	}
	for _, check := range checks {
		if math.IsNaN(check.value) || math.IsInf(check.value, 0) || check.value < check.minimum || check.value > check.maximum {
			return &ValidationError{
				Field: check.field,
				Err:   fmt.Errorf("%g is outside [%g,%g]", check.value, check.minimum, check.maximum),
			}
		}
	}

	return nil
}

// Normalized returns settings suitable for restoring untrusted preferences.
// Each invalid field falls back independently so one corrupt value does not
// discard the remaining valid settings.
func (s Settings) Normalized() Settings {
	defaults := DefaultSettings()
	if invalidRatio(s.MinimumShortEdge, 0.10, 0.30) {
		s.MinimumShortEdge = defaults.MinimumShortEdge
	}
	if invalidRatio(s.SizeVariation, 0, 0.25) {
		s.SizeVariation = defaults.SizeVariation
	}
	if invalidRatio(s.Overlap, 0, 0.20) {
		s.Overlap = defaults.Overlap
	}
	if invalidRatio(s.MaximumRotation, 0, 12) {
		s.MaximumRotation = defaults.MaximumRotation
	}
	s.Frame = FrameStyleFromPreference(string(s.Frame))

	return s
}

func invalidRatio(value, minimum, maximum float64) bool {
	return math.IsNaN(value) || math.IsInf(value, 0) || value < minimum || value > maximum
}

// Request is a validated, immutable description of one generation.
type Request struct {
	sources  []fyne.URI
	target   image.Point
	settings Settings
	seed     int64
}

// NewRequest validates and snapshots all inputs for a generation.
func NewRequest(sources []fyne.URI, target image.Point, settings Settings, seed int64) (Request, error) {
	if len(sources) == 0 {
		return Request{}, &ValidationError{Field: "sources", Err: fmt.Errorf("must not be empty")}
	}
	for i, source := range sources {
		if source == nil {
			return Request{}, &ValidationError{Field: fmt.Sprintf("sources[%d]", i), Err: fmt.Errorf("must not be nil")}
		}
	}
	if target.X <= 0 {
		return Request{}, &ValidationError{Field: "target_width", Err: fmt.Errorf("must be positive")}
	}
	if target.Y <= 0 {
		return Request{}, &ValidationError{Field: "target_height", Err: fmt.Errorf("must be positive")}
	}
	if target.X > math.MaxInt/target.Y {
		return Request{}, &ValidationError{Field: "target_area", Err: fmt.Errorf("overflows int")}
	}
	if err := settings.Validate(); err != nil {
		return Request{}, err
	}
	settings.Frame = FrameStyleFromPreference(string(settings.Frame))

	return Request{
		sources:  slices.Clone(sources),
		target:   target,
		settings: settings,
		seed:     seed,
	}, nil
}

// Sources returns a defensive copy of the generation's source snapshot.
func (r Request) Sources() []fyne.URI {
	return slices.Clone(r.sources)
}

// TargetSize returns the requested output dimensions in native pixels.
func (r Request) TargetSize() image.Point {
	return r.target
}

// Settings returns the validated visual settings.
func (r Request) Settings() Settings {
	return r.settings
}

// Seed returns the deterministic generation seed.
func (r Request) Seed() int64 {
	return r.seed
}

// Result is an immutable rendered mosaic.
type Result struct {
	pixels *image.NRGBA
}

func newResult(source image.Image) Result {
	pixels := image.NewNRGBA(source.Bounds())
	draw.Draw(pixels, pixels.Bounds(), source, source.Bounds().Min, draw.Src)

	return Result{pixels: pixels}
}

// Bounds returns the exact output bounds.
func (r Result) Bounds() image.Rectangle {
	if r.pixels == nil {
		return image.Rectangle{}
	}

	return r.pixels.Bounds()
}

// Image returns a mutable copy while retaining the result's pixels unchanged.
func (r Result) Image() image.Image {
	if r.pixels == nil {
		return nil
	}

	pixels := image.NewNRGBA(r.pixels.Bounds())
	copy(pixels.Pix, r.pixels.Pix)

	return pixels
}
