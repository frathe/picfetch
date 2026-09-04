package mosaic

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestMain(m *testing.M) {
	test.NewApp()
	os.Exit(m.Run())
}

func TestGenerate_DeterministicAndCoverage(t *testing.T) {
	source := mosaicPNG(t, "source.png", 12, 8, func(x, y int) color.NRGBA {
		return color.NRGBA{R: uint8(x * 15), G: uint8(y * 20), B: 80, A: 255}
	})
	request := mustRequest(t, []fyne.URI{source}, image.Pt(83, 47), DefaultSettings(), 987)

	first, err := Generate(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Generate(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if first.Bounds() != image.Rect(0, 0, 83, 47) {
		t.Fatalf("result bounds = %v", first.Bounds())
	}
	a := first.Image().(*image.NRGBA)
	b := second.Image().(*image.NRGBA)
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("fixed generation inputs produced different pixels")
	}
	for i := 3; i < len(a.Pix); i += 4 {
		if a.Pix[i] != 255 {
			t.Fatalf("output pixel %d has alpha %d, want fully covered", i/4, a.Pix[i])
		}
	}
}

func TestGenerate_CoverageHasNoUntouchedCanvasPixel(t *testing.T) {
	source := mosaicPNG(t, "white.png", 10, 10, func(_, _ int) color.NRGBA {
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	})
	result, err := Generate(context.Background(), mustRequest(
		t, []fyne.URI{source}, image.Pt(83, 47), DefaultSettings(), 987,
	))
	if err != nil {
		t.Fatal(err)
	}
	pixels := result.Image().(*image.NRGBA)
	for y := pixels.Bounds().Min.Y; y < pixels.Bounds().Max.Y; y++ {
		for x := pixels.Bounds().Min.X; x < pixels.Bounds().Max.X; x++ {
			pixel := pixels.NRGBAAt(x, y)
			if pixel == (color.NRGBA{R: 28, G: 30, B: 34, A: 255}) {
				t.Fatalf("canvas pixel at (%d,%d) was never composited", x, y)
			}
		}
	}
}

func TestGenerate_RepairCardsRenderBehindPrimaryCards(t *testing.T) {
	colors := []color.NRGBA{
		{R: 240, G: 20, B: 20, A: 255},
		{R: 20, G: 220, B: 20, A: 255},
		{R: 20, G: 20, B: 240, A: 255},
		{R: 230, G: 180, B: 20, A: 255},
		{R: 180, G: 20, B: 220, A: 255},
	}
	sources := make([]fyne.URI, len(colors))
	for index := range sources {
		sources[index] = storage.NewFileURI(filepath.Join("/virtual", fmt.Sprintf("source-%d.png", index)))
	}
	load := func(_ context.Context, uri fyne.URI) (*loadedSource, error) {
		var index int
		if _, err := fmt.Sscanf(uri.Name(), "source-%d.png", &index); err != nil {
			return nil, err
		}
		pixels := image.NewNRGBA(image.Rect(0, 0, 12, 12))
		fillNRGBA(pixels, colors[index])
		return &loadedSource{pixels: pixels, bounds: pixels.Bounds()}, nil
	}

	settings := DefaultSettings()
	settings.MinimumShortEdge = 0.16
	settings.SizeVariation = 0.18
	settings.Overlap = 0.07
	settings.MaximumRotation = 12
	settings.DropShadow = false
	target := image.Pt(320, 180)
	seed := int64(7)
	request := mustRequest(t, sources, target, settings, seed)

	pool := newSourcePool(sources, seed, load)
	plan, err := planLayout(context.Background(), target, settings, seed, func() (candidate, error) {
		entry, source, err := pool.next(context.Background())
		if err != nil {
			return candidate{}, err
		}
		return candidate{id: entry.id, aspect: float64(source.bounds.Dx()) / float64(source.bounds.Dy())}, nil
	})
	if err != nil {
		t.Fatal(err)
	}

	point, primary := repairOverPrimaryPixel(plan)
	if primary < 0 {
		t.Fatal("test layout had no later repair over a differently colored primary")
	}
	generator := New()
	generator.load = load
	result, err := generator.Generate(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := result.Image().(*image.NRGBA).NRGBAAt(point.X, point.Y), colors[plan.placements[primary].candidateID]; got != want {
		t.Fatalf("pixel %v = %v, want foreground primary %v", point, got, want)
	}
}

func TestRenderPlacement_AntialiasesRotatedFrameEdges(t *testing.T) {
	sourcePixels := image.NewNRGBA(image.Rect(0, 0, 20, 12))
	fillNRGBA(sourcePixels, color.NRGBA{R: 25, G: 80, B: 180, A: 255})
	source := &loadedSource{pixels: sourcePixels, bounds: sourcePixels.Bounds()}
	destination := image.NewNRGBA(image.Rect(0, 0, 80, 60))
	placed := newPlacement(1, 40, 30, 42, 25, 12, FrameThinLight, false)

	if err := renderPlacement(context.Background(), destination, source, placed); err != nil {
		t.Fatal(err)
	}
	transitionPixels := 0
	for index := 3; index < len(destination.Pix); index += 4 {
		if destination.Pix[index] > 0 && destination.Pix[index] < 255 {
			transitionPixels++
		}
	}
	if transitionPixels == 0 {
		t.Fatal("rotated frame edge has no partially covered anti-aliased pixels")
	}
}

func TestRenderPlacement_AreaCoverageMatchesRotatedCardGeometry(t *testing.T) {
	sourcePixels := image.NewNRGBA(image.Rect(0, 0, 24, 16))
	fillNRGBA(sourcePixels, color.NRGBA{R: 25, G: 80, B: 180, A: 255})
	source := &loadedSource{pixels: sourcePixels, bounds: sourcePixels.Bounds()}
	destination := image.NewNRGBA(image.Rect(0, 0, 220, 180))
	placed := newPlacement(1, 110.2, 90.35, 120, 80, 12, FrameThinLight, false)

	if err := renderPlacement(context.Background(), destination, source, placed); err != nil {
		t.Fatal(err)
	}

	polygon := rotatedTestRectangle(
		placed,
		-placed.bodyWidth/2,
		placed.bodyTop,
		placed.bodyWidth/2,
		placed.bodyBottom,
	)
	partialPixels := 0
	binaryPixels := 0
	maximumError := 0
	totalError := 0
	worstPoint := image.Point{}
	worstGot := 0
	worstExpected := 0
	for y := destination.Bounds().Min.Y; y < destination.Bounds().Max.Y; y++ {
		for x := destination.Bounds().Min.X; x < destination.Bounds().Max.X; x++ {
			coverage := clippedPixelCoverage(polygon, x, y)
			expected := int(math.Round(coverage * 255))
			if expected == 0 || expected == 255 {
				continue
			}
			partialPixels++
			got := int(destination.NRGBAAt(x, y).A)
			if (got == 0 || got == 255) && expected > 2 && expected < 253 {
				binaryPixels++
			}
			alphaError := absInt(got - expected)
			totalError += alphaError
			if alphaError > maximumError {
				maximumError = alphaError
				worstPoint = image.Pt(x, y)
				worstGot = got
				worstExpected = expected
			}
		}
	}
	if partialPixels < 100 {
		t.Fatalf("oracle found %d partially covered pixels, want a representative rotated edge", partialPixels)
	}
	meanError := float64(totalError) / float64(partialPixels)
	t.Logf("%d partial edge pixels: mean alpha error %.1f, maximum %d", partialPixels, meanError, maximumError)
	if binaryPixels != 0 || maximumError > 48 || meanError > 16 {
		t.Fatalf("rotated edge had %d/%d binary partial pixels, mean alpha error %.1f, and maximum error %d at %v (got %d, want %d); want 0, at most 16, and at most 48", binaryPixels, partialPixels, meanError, maximumError, worstPoint, worstGot, worstExpected)
	}
}

type testCoveragePoint struct {
	x float64
	y float64
}

func rotatedTestRectangle(placed placement, left, top, right, bottom float64) []testCoveragePoint {
	radians := placed.angle * math.Pi / 180
	sin, cos := math.Sincos(radians)
	points := []testCoveragePoint{
		{x: left, y: top},
		{x: right, y: top},
		{x: right, y: bottom},
		{x: left, y: bottom},
	}
	for index := range points {
		local := points[index]
		points[index] = testCoveragePoint{
			x: placed.centerX + local.x*cos - local.y*sin,
			y: placed.centerY + local.x*sin + local.y*cos,
		}
	}

	return points
}

func clippedPixelCoverage(polygon []testCoveragePoint, x, y int) float64 {
	clipped := append([]testCoveragePoint(nil), polygon...)
	clipped = clipTestPolygon(clipped, 0, float64(x), true)
	clipped = clipTestPolygon(clipped, 0, float64(x+1), false)
	clipped = clipTestPolygon(clipped, 1, float64(y), true)
	clipped = clipTestPolygon(clipped, 1, float64(y+1), false)
	if len(clipped) < 3 {
		return 0
	}

	area := 0.0
	for index, point := range clipped {
		next := clipped[(index+1)%len(clipped)]
		area += point.x*next.y - next.x*point.y
	}
	return math.Min(1, math.Abs(area)/2)
}

func clipTestPolygon(polygon []testCoveragePoint, axis int, boundary float64, keepGreater bool) []testCoveragePoint {
	if len(polygon) == 0 {
		return nil
	}
	coordinate := func(point testCoveragePoint) float64 {
		if axis == 0 {
			return point.x
		}
		return point.y
	}
	inside := func(point testCoveragePoint) bool {
		if keepGreater {
			return coordinate(point) >= boundary
		}
		return coordinate(point) <= boundary
	}
	intersection := func(from, to testCoveragePoint) testCoveragePoint {
		ratio := (boundary - coordinate(from)) / (coordinate(to) - coordinate(from))
		return testCoveragePoint{
			x: from.x + ratio*(to.x-from.x),
			y: from.y + ratio*(to.y-from.y),
		}
	}

	output := make([]testCoveragePoint, 0, len(polygon)+1)
	previous := polygon[len(polygon)-1]
	previousInside := inside(previous)
	for _, current := range polygon {
		currentInside := inside(current)
		if currentInside != previousInside {
			output = append(output, intersection(previous, current))
		}
		if currentInside {
			output = append(output, current)
		}
		previous = current
		previousInside = currentInside
	}

	return output
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func TestRenderPlacement_InterpolatesFractionalSourceCoordinates(t *testing.T) {
	sourcePixels := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	sourcePixels.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	sourcePixels.SetNRGBA(1, 0, color.NRGBA{B: 255, A: 255})
	source := &loadedSource{pixels: sourcePixels, bounds: sourcePixels.Bounds()}
	destination := image.NewNRGBA(image.Rect(0, 0, 6, 6))
	placed := newPlacement(1, 2.25, 2.5, 2, 2, 0, FrameNone, false)

	if err := renderPlacement(context.Background(), destination, source, placed); err != nil {
		t.Fatal(err)
	}
	pixel := destination.NRGBAAt(2, 2)
	if pixel.R == 0 || pixel.B == 0 || pixel.G != 0 {
		t.Fatalf("fractional source sample = %v, want an interpolated red/blue pixel", pixel)
	}
}

func TestRenderPlacement_StopsBetweenTransformBands(t *testing.T) {
	sourcePixels := image.NewNRGBA(image.Rect(0, 0, 20, 20))
	fillNRGBA(sourcePixels, color.NRGBA{R: 80, G: 150, B: 230, A: 255})
	source := &loadedSource{pixels: sourcePixels, bounds: sourcePixels.Bounds()}
	destination := image.NewNRGBA(image.Rect(0, 0, 220, 220))
	placed := newPlacement(1, 110, 110, 180, 180, 0, FrameNone, false)
	ctx := &cancelAfterChecksContext{cancelAt: 3}

	if err := renderPlacement(ctx, destination, source, placed); !errors.Is(err, context.Canceled) {
		t.Fatalf("renderPlacement() = %v, want context.Canceled", err)
	}
	if got := destination.NRGBAAt(110, 180).A; got != 0 {
		t.Fatalf("pixel below first transform band has alpha %d after cancellation, want zero", got)
	}
}

type cancelAfterChecksContext struct {
	checks   int
	cancelAt int
}

func (c *cancelAfterChecksContext) Deadline() (time.Time, bool) { return time.Time{}, false }
func (*cancelAfterChecksContext) Done() <-chan struct{}         { return nil }
func (c *cancelAfterChecksContext) Err() error {
	c.checks++
	if c.checks >= c.cancelAt {
		return context.Canceled
	}

	return nil
}
func (*cancelAfterChecksContext) Value(_ any) any { return nil }

func TestGenerate_DropShadowCanBeDisabled(t *testing.T) {
	source := mosaicPNG(t, "shadow-source.png", 12, 8, func(_, _ int) color.NRGBA {
		return color.NRGBA{R: 70, G: 130, B: 220, A: 255}
	})
	settings := DefaultSettings()
	settings.SizeVariation = 0
	settings.MaximumRotation = 5
	withShadow, err := Generate(context.Background(), mustRequest(t, []fyne.URI{source}, image.Pt(96, 60), settings, 19))
	if err != nil {
		t.Fatal(err)
	}
	settings.DropShadow = false
	withoutShadow, err := Generate(context.Background(), mustRequest(t, []fyne.URI{source}, image.Pt(96, 60), settings, 19))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(withShadow.Image().(*image.NRGBA).Pix, withoutShadow.Image().(*image.NRGBA).Pix) {
		t.Fatal("enabled and disabled drop shadows rendered identically")
	}
}

func repairOverPrimaryPixel(plan layoutPlan) (image.Point, int) {
	for y := 1; y < plan.target.Y-1; y++ {
		for x := 1; x < plan.target.X-1; x++ {
			primary, top := -1, -1
			for index, placed := range plan.placements {
				if !placementCovers(placed, float64(x)+0.5, float64(y)+0.5) {
					continue
				}
				top = index
				if !placed.repair {
					primary = index
				}
			}
			if primary < 0 || top <= primary || !plan.placements[top].repair ||
				plan.placements[top].candidateID == plan.placements[primary].candidateID {
				continue
			}
			if bodyEdgeDistance(plan.placements[primary], x, y) >= 2 && bodyEdgeDistance(plan.placements[top], x, y) >= 2 {
				return image.Pt(x, y), primary
			}
		}
	}

	return image.Point{}, -1
}

func bodyEdgeDistance(placed placement, x, y int) float64 {
	localX, localY := inverseRotate(placed, float64(x)+0.5, float64(y)+0.5)
	return min(
		localX+placed.bodyWidth/2,
		placed.bodyWidth/2-localX,
		localY-placed.bodyTop,
		placed.bodyBottom-localY,
	)
}

func TestGenerate_SourceFidelity(t *testing.T) {
	source := mosaicPNG(t, "two-colors.png", 20, 10, func(x, _ int) color.NRGBA {
		if x < 10 {
			return color.NRGBA{R: 255, A: 255}
		}
		return color.NRGBA{B: 255, A: 255}
	})
	settings := DefaultSettings()
	settings.SizeVariation = 0
	settings.Overlap = 0
	settings.MaximumRotation = 0
	request := mustRequest(t, []fyne.URI{source}, image.Pt(80, 50), settings, 2)

	result, err := Generate(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	var red, blue bool
	image := result.Image().(*image.NRGBA)
	for y := range image.Bounds().Dy() {
		for x := range image.Bounds().Dx() {
			pixel := image.NRGBAAt(x, y)
			red = red || pixel.R > 230 && pixel.B < 25
			blue = blue || pixel.B > 230 && pixel.R < 25
		}
	}
	if !red || !blue {
		t.Fatalf("source colors missing after generation: red=%v blue=%v", red, blue)
	}
}

func TestGenerate_RespectsDecodedOrientation(t *testing.T) {
	settings := DefaultSettings()
	settings.SizeVariation = 0
	settings.Overlap = 0
	settings.MaximumRotation = 0
	settings.Frame = FrameNone
	settings.DropShadow = false
	pngSource, pngPixels := mosaicOrientedPNG(t, "oriented.png", 12, 6, 6)
	webpSource, webpPixels := mosaicOrientedWebP(t, "oriented.webp", 6)
	tests := []struct {
		name   string
		source fyne.URI
		pixels image.Image
	}{
		{name: "PNG eXIf", source: pngSource, pixels: pngPixels},
		{name: "WebP EXIF", source: webpSource, pixels: webpPixels},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mustRequest(t, []fyne.URI{tt.source}, image.Pt(96, 60), settings, 41)
			got, err := Generate(context.Background(), request)
			if err != nil {
				t.Fatal(err)
			}
			reference := New()
			reference.load = func(context.Context, fyne.URI) (*loadedSource, error) {
				return &loadedSource{pixels: tt.pixels, bounds: tt.pixels.Bounds()}, nil
			}
			want, err := reference.Generate(context.Background(), request)
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(got.Image().(*image.NRGBA).Pix, want.Image().(*image.NRGBA).Pix) {
				t.Fatal("metadata-oriented source rendered with pre-orientation geometry")
			}
		})
	}
}

func TestGenerate_SourceFidelityCanonicalFormats(t *testing.T) {
	oriented := storage.NewFileURI(uitest.WriteTempFile(
		t, "oriented.jpg", uitest.EncodeOrientedJPEG(t, 12, 6, 6),
	))
	animated := storage.NewFileURI(uitest.WriteTempFile(
		t, "animated.gif", uitest.EncodeAnimatedGIF(t, 9, 7,
			[]color.Color{color.RGBA{R: 255, A: 255}, color.RGBA{B: 255, A: 255}},
			[]int{5, 5}),
	))
	transparent := mosaicPNG(t, "transparent.png", 8, 8, func(_, _ int) color.NRGBA {
		return color.NRGBA{G: 220, A: 80}
	})

	tests := []struct {
		name       string
		source     fyne.URI
		wantBounds image.Rectangle
		wantVector bool
		check      func(*testing.T, image.Image)
	}{
		{
			name:       "EXIF orientation",
			source:     oriented,
			wantBounds: image.Rect(0, 0, 6, 12),
			check: func(t *testing.T, pixels image.Image) {
				t.Helper()
				top := color.NRGBAModel.Convert(pixels.At(3, 2)).(color.NRGBA)
				bottom := color.NRGBAModel.Convert(pixels.At(3, 9)).(color.NRGBA)
				if top.R < 220 || bottom.B < 220 {
					t.Fatalf("orientation pixels top=%v bottom=%v", top, bottom)
				}
			},
		},
		{
			name:       "first animated GIF frame",
			source:     animated,
			wantBounds: image.Rect(0, 0, 9, 7),
			check: func(t *testing.T, pixels image.Image) {
				t.Helper()
				pixel := color.NRGBAModel.Convert(pixels.At(4, 3)).(color.NRGBA)
				if pixel.R < 240 || pixel.B > 15 {
					t.Fatalf("first GIF frame pixel = %v, want red", pixel)
				}
			},
		},
		{name: "SVG vector", source: uitest.TempSVGURI(t, "wide.svg", 20, 10), wantBounds: image.Rect(0, 0, 520, 260), wantVector: true},
		{name: "RAW preview", source: uitest.TempRAWURI(t, "photo.cr2", 13, 7, color.RGBA{R: 210, A: 255}), wantBounds: image.Rect(0, 0, 13, 7)},
		{
			name:       "transparent raster",
			source:     transparent,
			wantBounds: image.Rect(0, 0, 8, 8),
			check: func(t *testing.T, pixels image.Image) {
				t.Helper()
				if alpha := color.NRGBAModel.Convert(pixels.At(4, 4)).(color.NRGBA).A; alpha != 80 {
					t.Fatalf("decoded alpha = %d, want 80", alpha)
				}
			},
		},
		{name: "portrait raster", source: mosaicPNG(t, "portrait.png", 7, 13, func(_, _ int) color.NRGBA { return color.NRGBA{R: 255, A: 255} }), wantBounds: image.Rect(0, 0, 7, 13)},
		{name: "landscape raster", source: mosaicPNG(t, "landscape.png", 13, 7, func(_, _ int) color.NRGBA { return color.NRGBA{B: 255, A: 255} }), wantBounds: image.Rect(0, 0, 13, 7)},
	}

	settings := DefaultSettings()
	settings.SizeVariation = 0
	settings.Overlap = 0
	settings.MaximumRotation = 0
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loaded, err := loadCanonicalSource(context.Background(), tt.source)
			if err != nil {
				t.Fatal(err)
			}
			if loaded.bounds != tt.wantBounds {
				t.Fatalf("canonical bounds = %v, want %v", loaded.bounds, tt.wantBounds)
			}
			if (loaded.vector != nil) != tt.wantVector {
				t.Fatalf("vector present = %v, want %v", loaded.vector != nil, tt.wantVector)
			}
			if tt.check != nil {
				tt.check(t, loaded.pixels)
			}

			result, err := Generate(context.Background(), mustRequest(t, []fyne.URI{tt.source}, image.Pt(61, 37), settings, 8))
			if err != nil {
				t.Fatal(err)
			}
			if result.Bounds() != image.Rect(0, 0, 61, 37) {
				t.Fatalf("generated bounds = %v", result.Bounds())
			}
		})
	}
}

func TestGenerate_FrameStyles(t *testing.T) {
	source := mosaicPNG(t, "black.png", 10, 10, func(_, _ int) color.NRGBA {
		return color.NRGBA{A: 255}
	})
	want := map[FrameStyle]string{
		FrameNone:      "d05e0124f64ef52eff7bd6e5222ec8e59e5668be484d067bf63bdc8330d52854",
		FrameThinLight: "4f83f9d4116a8460d00f9bc4a1ff81849140940252e53a43ba0d42c8ee31269e",
		FrameThinDark:  "3861291343c1efc843119adcd12108cf0f32a50d2367cfa06ada9b8f27efab04",
		FramePolaroid:  "6c92aa0aba579e6fdb93193ce2e4075420fa13a3244e4e6bee71f1a4d90bdf26",
	}
	seen := make(map[[32]byte]FrameStyle)
	for _, frame := range []FrameStyle{FrameNone, FrameThinLight, FrameThinDark, FramePolaroid} {
		settings := DefaultSettings()
		settings.Frame = frame
		request := mustRequest(t, []fyne.URI{source}, image.Pt(64, 40), settings, 4)
		result, err := Generate(context.Background(), request)
		if err != nil {
			t.Fatalf("Generate(frame=%s) = %v", frame, err)
		}
		hash := sha256.Sum256(result.Image().(*image.NRGBA).Pix)
		if fmt.Sprintf("%x", hash) != want[frame] {
			t.Errorf("frame %s hash = %x", frame, hash)
		}
		if previous, ok := seen[hash]; ok {
			t.Fatalf("frames %s and %s rendered identically", previous, frame)
		}
		seen[hash] = frame
	}
}

func TestGenerate_UnreadableSources(t *testing.T) {
	missing := storage.NewFileURI(filepath.Join(t.TempDir(), "missing.png"))
	valid := mosaicPNG(t, "valid.png", 4, 4, func(_, _ int) color.NRGBA {
		return color.NRGBA{G: 255, A: 255}
	})

	if _, err := Generate(context.Background(), mustRequest(t, []fyne.URI{missing, valid}, image.Pt(30, 20), DefaultSettings(), 1)); err != nil {
		t.Fatalf("broken source followed by readable source failed: %v", err)
	}

	_, err := Generate(context.Background(), mustRequest(t, []fyne.URI{missing}, image.Pt(30, 20), DefaultSettings(), 1))
	var unreadable *NoReadableSourcesError
	if !errors.As(err, &unreadable) || len(unreadable.Attempts) != 1 {
		t.Fatalf("all-broken Generate() = %v, want one-attempt NoReadableSourcesError", err)
	}
}

func TestGenerate_LazyPool(t *testing.T) {
	sources := make([]fyne.URI, 10_000)
	for i := range sources {
		sources[i] = storage.NewFileURI(filepath.Join("/virtual", string(rune(i+1))+".png"))
	}
	request := mustRequest(t, sources, image.Pt(120, 80), DefaultSettings(), 7)
	loads := 0
	generator := New()
	generator.load = func(context.Context, fyne.URI) (*loadedSource, error) {
		loads++
		pixels := image.NewNRGBA(image.Rect(0, 0, 8, 6))
		for i := 3; i < len(pixels.Pix); i += 4 {
			pixels.Pix[i] = 255
		}
		return &loadedSource{pixels: pixels, bounds: pixels.Bounds()}, nil
	}

	if _, err := generator.Generate(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if loads == 0 || loads >= len(sources) {
		t.Fatalf("loader calls = %d for %d sources, want a lazy prefix", loads, len(sources))
	}
}

func TestGenerate_UsesDistinctSourceURIsBeforeReuse(t *testing.T) {
	const distinctSources = 128
	unique := make([]fyne.URI, distinctSources)
	for index := range unique {
		unique[index] = storage.NewFileURI(filepath.Join("/virtual", fmt.Sprintf("source-%02d.png", index)))
	}
	sources := make([]fyne.URI, 0, distinctSources*3)
	for range 3 {
		sources = append(sources, unique...)
	}

	settings := DefaultSettings()
	settings.MinimumShortEdge = 0.30
	settings.SizeVariation = 0
	settings.Overlap = 0
	settings.MaximumRotation = 0
	settings.Frame = FrameNone
	settings.DropShadow = false
	request := mustRequest(t, sources, image.Pt(48, 32), settings, 7)

	loadedURIs := make([]string, 0, distinctSources)
	generator := New()
	generator.load = func(_ context.Context, uri fyne.URI) (*loadedSource, error) {
		loadedURIs = append(loadedURIs, uri.String())
		pixels := image.NewNRGBA(image.Rect(0, 0, 8, 6))
		fillNRGBA(pixels, color.NRGBA{R: 90, G: 140, B: 210, A: 255})
		return &loadedSource{pixels: pixels, bounds: pixels.Bounds()}, nil
	}

	if _, err := generator.Generate(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if len(loadedURIs) >= distinctSources {
		t.Fatalf("test used %d sources from a %d-source distinct pool; want unused choices remaining", len(loadedURIs), distinctSources)
	}
	seen := make(map[string]struct{}, len(loadedURIs))
	for _, uri := range loadedURIs {
		if _, exists := seen[uri]; exists {
			t.Fatalf("source URI %q was loaded twice while %d distinct sources remained unused", uri, distinctSources-len(seen))
		}
		seen[uri] = struct{}{}
	}
}

func TestGenerate_SourcesUnchanged(t *testing.T) {
	source := mosaicPNG(t, "immutable.png", 8, 5, func(x, y int) color.NRGBA {
		return color.NRGBA{R: uint8(x * 20), B: uint8(y * 30), A: 255}
	})
	before, err := os.ReadFile(source.Path())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(context.Background(), mustRequest(t, []fyne.URI{source}, image.Pt(50, 30), DefaultSettings(), 3)); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(source.Path())
	if err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(before) != sha256.Sum256(after) {
		t.Fatal("source bytes changed during generation")
	}
}

func mosaicPNG(t *testing.T, name string, width, height int, pixel func(int, int) color.NRGBA) fyne.URI {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	image := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			image.SetNRGBA(x, y, pixel(x, y))
		}
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, image); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	return storage.NewFileURI(path)
}

func mosaicOrientedPNG(t *testing.T, name string, width, height int, orientation uint16) (fyne.URI, image.Image) {
	t.Helper()
	pixels := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			if x < width/2 {
				pixels.SetNRGBA(x, y, color.NRGBA{R: 255, A: 255})
			} else {
				pixels.SetNRGBA(x, y, color.NRGBA{B: 255, A: 255})
			}
		}
	}

	var encoded bytes.Buffer
	if err := png.Encode(&encoded, pixels); err != nil {
		t.Fatal(err)
	}
	tiff := mosaicOrientationTIFF(orientation)

	chunk := make([]byte, 12+len(tiff))
	binary.BigEndian.PutUint32(chunk[:4], uint32(len(tiff)))
	copy(chunk[4:8], "eXIf")
	copy(chunk[8:8+len(tiff)], tiff)
	binary.BigEndian.PutUint32(chunk[8+len(tiff):], crc32.ChecksumIEEE(chunk[4:8+len(tiff)]))
	const afterIHDR = 8 + 4 + 4 + 13 + 4
	data := make([]byte, 0, encoded.Len()+len(chunk))
	data = append(data, encoded.Bytes()[:afterIHDR]...)
	data = append(data, chunk...)
	data = append(data, encoded.Bytes()[afterIHDR:]...)

	return storage.NewFileURI(uitest.WriteTempFile(t, name, data)), imaging.ApplyOrientation(pixels, int(orientation))
}

func mosaicOrientedWebP(t *testing.T, name string, orientation uint16) (fyne.URI, image.Image) {
	t.Helper()
	// This compact lossless fixture comes from golang.org/x/image/webp's
	// BSD-licensed testdata. Wrapping it as extended WebP lets the test add an
	// EXIF chunk without depending on an external encoder.
	const encodedFixture = "UklGRrIBAABXRUJQVlA4TKUBAAAvSsAYAA8w//M///MfeJAkbXvaSG7m8Q3GfYSBJekwQztm/IcZlgwnmWImn2BK7aFmBtnVir6q//8VOkFE/xm4baTIu8c48ArEo6+B3zFKYln3pqClSCKX0begFTAXFOLXHSyF8cCNcZEG4OywuA4KVVfJCiArU7GAgJI8+lJP/OKMT/fBAjevg1cYB7YVkFuWga2lyPi5I0HFy5YTpWIHg0RZpkniRVW9odHAKOwosWuOGdxIyn2OvaCDvhg/we6TwadPBPbqBV58MsLmMJ8yZnOWk8SRz4N+QoyPL+MnamzMvcE1rHNEr91F9GKZPVUcS9w7PhhH36suB9qPeYb/oLk6cuTiJ0wOK3m5h1cKjW6EVZCYMK7dxcKCBdgP9HkKr9gkAO2P8GKZGWVdIAatQa+1IDpt6qyorVwdy01xdW8Jkfk6xjEXmVQQ+HQdFr6OKhIN34dXWq0+0qr6EJSCeeVLH9+gvGTLyqM65PQ44ihzlTXxQKjKbAvshXgir7Lil9w4L2bvMycmjQcqXaMCO6BlY28i+FOLzbfI1vEqxAhotocAAA=="
	simple, err := base64.StdEncoding.DecodeString(encodedFixture)
	if err != nil {
		t.Fatal(err)
	}
	pixels, _, err := image.Decode(bytes.NewReader(simple))
	if err != nil {
		t.Fatal(err)
	}

	width, height := pixels.Bounds().Dx(), pixels.Bounds().Dy()
	vp8x := make([]byte, 18)
	copy(vp8x[:4], "VP8X")
	binary.LittleEndian.PutUint32(vp8x[4:8], 10)
	vp8x[8] = 0x08
	putLittleEndian24(vp8x[12:15], width-1)
	putLittleEndian24(vp8x[15:18], height-1)
	exif := webPChunk("EXIF", mosaicOrientationTIFF(orientation))
	data := make([]byte, 12, 12+len(vp8x)+len(simple)-12+len(exif))
	copy(data[:4], "RIFF")
	copy(data[8:12], "WEBP")
	data = append(data, vp8x...)
	data = append(data, simple[12:]...)
	data = append(data, exif...)
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(data)-8))

	return storage.NewFileURI(uitest.WriteTempFile(t, name, data)), imaging.ApplyOrientation(pixels, int(orientation))
}

func mosaicOrientationTIFF(orientation uint16) []byte {
	tiff := make([]byte, 26)
	copy(tiff, "II")
	binary.LittleEndian.PutUint16(tiff[2:4], 0x002A)
	binary.LittleEndian.PutUint32(tiff[4:8], 8)
	binary.LittleEndian.PutUint16(tiff[8:10], 1)
	binary.LittleEndian.PutUint16(tiff[10:12], 0x0112)
	binary.LittleEndian.PutUint16(tiff[12:14], 3)
	binary.LittleEndian.PutUint32(tiff[14:18], 1)
	binary.LittleEndian.PutUint16(tiff[18:20], orientation)
	return tiff
}

func webPChunk(name string, payload []byte) []byte {
	chunk := make([]byte, 8, 8+len(payload)+len(payload)%2)
	copy(chunk[:4], name)
	binary.LittleEndian.PutUint32(chunk[4:8], uint32(len(payload)))
	chunk = append(chunk, payload...)
	if len(payload)%2 != 0 {
		chunk = append(chunk, 0)
	}
	return chunk
}

func putLittleEndian24(destination []byte, value int) {
	destination[0] = byte(value)
	destination[1] = byte(value >> 8)
	destination[2] = byte(value >> 16)
}

func mustRequest(t *testing.T, sources []fyne.URI, target image.Point, settings Settings, seed int64) Request {
	t.Helper()
	request, err := NewRequest(sources, target, settings, seed)
	if err != nil {
		t.Fatal(err)
	}

	return request
}
