package mosaic

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
)

type qualityScene struct {
	plan          layoutPlan
	result        Result
	colors        []color.NRGBA
	area, visible []float64
	clipped       float64
}

func qualitySettings(rotation, overlap float64) Settings {
	s := DefaultSettings()
	s.MinimumShortEdge, s.SizeVariation = .16, .18
	s.MaximumRotation, s.Overlap = rotation, overlap
	s.Frame = FrameNone
	s.DropShadow = false
	return s
}

func shelfQualitySettings(overlap float64) Settings {
	settings := qualitySettings(0, overlap)
	settings.Layout = LayoutShelf

	return settings
}

// Each uncached load has its own marker, so repeating one source cannot hide
// a concealed occurrence behind the visibility of another copy.
func generateQualityScene(t *testing.T, target image.Point, settings Settings, seed int64, poolSize int, aligned bool) qualityScene {
	t.Helper()
	sources := make([]fyne.URI, poolSize)
	for i := range sources {
		sources[i] = storage.NewFileURI(fmt.Sprintf("/virtual/quality-%d.png", i))
	}
	sizes := []image.Point{{36, 24}, {18, 24}, {43, 24}, {18, 30}, {30, 24}}
	loader := func(markers *[]color.NRGBA) sourceLoader {
		return func(_ context.Context, uri fyne.URI) (*loadedSource, error) {
			var id int
			if _, err := fmt.Sscanf(uri.Name(), "quality-%d.png", &id); err != nil {
				return nil, err
			}
			size := sizes[id%len(sizes)]
			if aligned {
				size = image.Pt(24, 24)
			}
			n := len(*markers)
			c := color.NRGBA{R: uint8(32 + n%7*32), G: uint8(32 + n/7%7*32), B: uint8(32 + n/49%7*32), A: 255}
			*markers = append(*markers, c)
			pixels := image.NewNRGBA(image.Rectangle{Max: size})
			for y := range size.Y {
				for x := range size.X {
					pixels.SetNRGBA(x, y, c)
				}
			}
			return &loadedSource{pixels: pixels, bounds: pixels.Bounds()}, nil
		}
	}
	var planColors []color.NRGBA
	pool := newSourcePool(sources, seed, loader(&planColors))
	pool.cache.SetBudget(1)
	plan, err := planLayout(t.Context(), target, settings, seed, func() (candidate, error) {
		entry, source, err := pool.next(t.Context())
		if err != nil {
			return candidate{}, err
		}
		return candidate{id: entry.id, aspect: float64(source.bounds.Dx()) / float64(source.bounds.Dy())}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var colors []color.NRGBA
	g := New()
	g.load, g.cacheBytes = loader(&colors), 1
	result, err := g.Generate(t.Context(), mustRequest(t, sources, target, settings, seed))
	if err != nil {
		t.Fatal(err)
	}
	if len(colors) < len(plan.placements) {
		t.Fatalf("occurrence mismatch: loads %d, plan loads %d, placements %d", len(colors), len(planColors), len(plan.placements))
	}
	// Shelf's settled plan may reuse a previously measured aspect without
	// reloading its pixels. The final placement loads are the last N markers
	// in either mode because the one-byte cache cannot retain these sources.
	colors = colors[len(colors)-len(plan.placements):]
	scene := qualityScene{plan: plan, result: result, colors: colors}
	if path := os.Getenv("PICFETCH_MOSAIC_EXPERIMENT_IMAGE"); path != "" {
		file, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		err = png.Encode(file, result.Image())
		closeErr := file.Close()
		if err != nil {
			t.Fatal(err)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		t.Logf("experiment render: %s", path)
	}
	scene.measure(t)
	return scene
}

// Independent rectangle subtraction measures photo area and opaque occlusion.
// Supersampled masks select stable interiors for checking actual final pixels.
// No production visibility or coverage helper is used; translucent shadows may
// darken a marker but cannot change its chromaticity.
func (s *qualityScene) measure(t *testing.T) {
	t.Helper()
	const samples = 2
	target := s.plan.target
	width, height := target.X*samples, target.Y*samples
	owners := make([]int, width*height)
	s.area, s.visible = make([]float64, len(s.plan.placements)), make([]float64, len(s.plan.placements))
	order := make([]int, 0, len(s.plan.placements))
	for _, repair := range []bool{true, false} {
		for i, p := range s.plan.placements {
			if p.repair == repair {
				order = append(order, i)
			}
		}
	}
	photos := make([][]testCoveragePoint, len(order))
	bodies := make([][]testCoveragePoint, len(order))
	for _, i := range order {
		p := s.plan.placements[i]
		sin, cos := math.Sincos(p.angle * math.Pi / 180)
		photo := rotatedTestRectangle(p, -p.imageRect.width/2, -p.imageRect.height/2, p.imageRect.width/2, p.imageRect.height/2)
		minX, minY, maxX, maxY := width, height, 0, 0
		body := rotatedTestRectangle(p, -p.bodyWidth/2, p.bodyTop, p.bodyWidth/2, p.bodyBottom)
		bodies[i] = body
		for _, v := range body {
			minX = min(minX, int(math.Floor(v.x*samples)))
			minY = min(minY, int(math.Floor(v.y*samples)))
			maxX = max(maxX, int(math.Ceil(v.x*samples)))
			maxY = max(maxY, int(math.Ceil(v.y*samples)))
		}
		for y := max(0, minY); y < min(height, maxY); y++ {
			for x := max(0, minX); x < min(width, maxX); x++ {
				dx, dy := (float64(x)+.5)/samples-p.centerX, (float64(y)+.5)/samples-p.centerY
				lx, ly := dx*cos+dy*sin, -dx*sin+dy*cos
				inPhoto := math.Abs(lx) < p.imageRect.width/2 && math.Abs(ly) < p.imageRect.height/2
				if math.Abs(lx) < p.bodyWidth/2 && ly > p.bodyTop && ly < p.bodyBottom {
					owners[y*width+x] = -i - 1
					if inPhoto {
						owners[y*width+x] = i + 1
					}
				}
			}
		}
		// Outer clipping is recorded separately from inter-photo coverage.
		polygon := photo
		polygon = clipTestPolygon(polygon, 0, 0, true)
		polygon = clipTestPolygon(polygon, 0, float64(target.X), false)
		polygon = clipTestPolygon(polygon, 1, 0, true)
		polygon = clipTestPolygon(polygon, 1, float64(target.Y), false)
		photos[i] = polygon
		s.area[i] = qualityPolygonArea(polygon)
		s.clipped += p.imageRect.width*p.imageRect.height - s.area[i]
	}
	// Subtract every later opaque rectangle analytically. Supersampling above
	// only selects stable pixels for the independent final-render checks below;
	// it is not the visibility denominator or a source of edge exemptions.
	for position, i := range order {
		pieces := [][]testCoveragePoint{photos[i]}
		for _, later := range order[position+1:] {
			var remaining [][]testCoveragePoint
			for _, piece := range pieces {
				remaining = append(remaining, qualitySubtractRectangle(piece, bodies[later])...)
			}
			pieces = remaining
			if len(pieces) == 0 {
				break
			}
		}
		for _, piece := range pieces {
			s.visible[i] += qualityPolygonArea(piece)
		}
	}
	checked := 0
	pixels := s.result.Image().(*image.NRGBA)
	for y := 2; y < target.Y-2; y++ {
		for x := 2; x < target.X-2; x++ {
			owner := owners[y*samples*width+x*samples]
			if owner <= 0 {
				continue
			}
			stable := true
			for sy := (y - 2) * samples; sy < (y+3)*samples && stable; sy++ {
				for sx := (x - 2) * samples; sx < (x+3)*samples; sx++ {
					if owners[sy*width+sx] != owner {
						stable = false
						break
					}
				}
			}
			if !stable {
				continue
			}
			got, want := pixels.NRGBAAt(x, y), s.colors[owner-1]
			scale := float64(int(got.R)+int(got.G)+int(got.B)) / float64(int(want.R)+int(want.G)+int(want.B))
			for c, value := range []uint8{got.R, got.G, got.B} {
				expected := float64([]uint8{want.R, want.G, want.B}[c]) * scale
				if math.Abs(float64(value)-expected) > 3 || scale < .1 || scale > 1.10 {
					t.Fatalf("final pixel %d,%d does not contain occurrence %d: got %v, marker %v", x, y, owner-1, got, want)
				}
			}
			checked++
		}
	}
	if checked < target.X*target.Y/20 {
		t.Errorf("only %d stable final photo pixels were checked", checked)
	}
}

func qualityPolygonArea(polygon []testCoveragePoint) float64 {
	area := 0.0
	for i, p := range polygon {
		next := polygon[(i+1)%len(polygon)]
		area += p.x*next.y - p.y*next.x
	}
	return math.Abs(area) / 2
}

// A rectangle's outside is the disjoint union of the portion outside its first
// edge, then the portion inside that edge but outside its next edge, and so on.
// Returning those convex pieces keeps overlapping occluders from being counted
// twice. The clipping equations are independent of production layout helpers.
func qualitySubtractRectangle(polygon, rectangle []testCoveragePoint) [][]testCoveragePoint {
	var outside [][]testCoveragePoint
	inside := polygon
	for i, a := range rectangle {
		b := rectangle[(i+1)%len(rectangle)]
		if piece := qualityClipEdge(inside, a, b, false); qualityPolygonArea(piece) > 1e-9 {
			outside = append(outside, piece)
		}
		inside = qualityClipEdge(inside, a, b, true)
		if len(inside) == 0 {
			break
		}
	}
	return outside
}

func qualityClipEdge(polygon []testCoveragePoint, a, b testCoveragePoint, keepInside bool) []testCoveragePoint {
	if len(polygon) == 0 {
		return nil
	}
	side := func(p testCoveragePoint) float64 { return (b.x-a.x)*(p.y-a.y) - (b.y-a.y)*(p.x-a.x) }
	kept := func(value float64) bool {
		if keepInside {
			return value >= 0
		}
		return value <= 0
	}
	var output []testCoveragePoint
	previous := polygon[len(polygon)-1]
	previousSide := side(previous)
	for _, current := range polygon {
		currentSide := side(current)
		if kept(previousSide) != kept(currentSide) {
			fraction := previousSide / (previousSide - currentSide)
			output = append(output, testCoveragePoint{x: previous.x + fraction*(current.x-previous.x), y: previous.y + fraction*(current.y-previous.y)})
		}
		if kept(currentSide) {
			output = append(output, current)
		}
		previous, previousSide = current, currentSide
	}
	return output
}

func (s qualityScene) coveredFraction() float64 {
	area, visible := 0.0, 0.0
	for i := range s.area {
		area += s.area[i]
		visible += s.visible[i]
	}
	return 1 - visible/area
}

func (s qualityScene) checkVisibility(t *testing.T) {
	t.Helper()
	worst, index := 1.0, -1
	concealed := 0
	for i, area := range s.area {
		if area == 0 || s.visible[i] == 0 {
			concealed++
			if index < 0 {
				index = i
			}
			continue
		}
		fraction := s.visible[i] / area
		if fraction < worst {
			worst, index = fraction, i
		}
	}
	t.Logf("occurrences=%d covered=%.4f least-visible=%.4f concealed=%d outer-clipped=%.1f", len(s.area), s.coveredFraction(), worst, concealed, s.clipped)
	// Analytic areas need only floating-point tolerance. No raster tolerance
	// may admit a fully concealed occurrence.
	if concealed > 0 || worst < .45-1e-9 {
		t.Errorf("photo visibility: occurrence %d retains %.2f%%; fully concealed %d", index, worst*100, concealed)
	}
}

func TestGenerate_PhotoVisibility(t *testing.T) {
	t.Run("oracle calibration", func(t *testing.T) {
		for _, frame := range []FrameStyle{FrameNone, FrameThinLight, FrameThinDark, FramePolaroid} {
			for _, angle := range []float64{0, 12, 45, -90, 90} {
				p := newPlacement(0, 15.25, 40.75, 60, 40, angle, frame, true)
				polygon := rotatedTestRectangle(p, -30, -20, 30, 20)
				polygon = clipTestPolygon(polygon, 0, 0, true)
				polygon = clipTestPolygon(polygon, 0, 120, false)
				polygon = clipTestPolygon(polygon, 1, 0, true)
				polygon = clipTestPolygon(polygon, 1, 100, false)
				want := qualityPolygonArea(polygon)
				renders := make([]*image.NRGBA, 0, 2)
				for _, value := range []uint8{0, 255} {
					source := image.NewNRGBA(image.Rect(0, 0, 30, 20))
					fillNRGBA(source, color.NRGBA{R: value, G: value, B: value, A: 255})
					destination := image.NewNRGBA(image.Rect(0, 0, 120, 100))
					if err := renderPlacement(t.Context(), destination, &loadedSource{pixels: source, bounds: source.Bounds()}, p); err != nil {
						t.Fatal(err)
					}
					renders = append(renders, destination)
				}
				got := 0.0
				for y := range 100 {
					for x := range 120 {
						black, _, _, _ := renders[0].At(x, y).RGBA()
						white, _, _, _ := renders[1].At(x, y).RGBA()
						got += float64(int(white)-int(black)) / 65535
					}
				}
				if difference := math.Abs(got-want) / want; difference > .01 {
					t.Errorf("frame=%s angle=%g: final photo mask %.3f, geometric photo area %.3f (%.2f%% difference)", frame, angle, got, want, difference*100)
				}
			}
		}
	})
	// Shelf promises that every retained occurrence remains visible. Random
	// retains its original primary-card guard, whose repair cards may sit below
	// later primary cards as part of its deliberately varied composition.
	for _, overlap := range []float64{0, .08, .20} {
		t.Run(fmt.Sprintf("overlap%g", overlap), func(t *testing.T) {
			for _, frame := range []FrameStyle{FrameNone, FrameThinLight, FrameThinDark, FramePolaroid} {
				for _, shadow := range []bool{false, true} {
					for _, pool := range []int{512, 2} {
						t.Run(fmt.Sprintf("%s/shadow%t/pool%d", frame, shadow, pool), func(t *testing.T) {
							settings := shelfQualitySettings(overlap)
							settings.Frame, settings.DropShadow = frame, shadow
							scene := generateQualityScene(t, image.Pt(320, 180), settings, 7, pool, false)
							scene.checkVisibility(t)
						})
					}
				}
			}
		})
	}
}

func TestGenerate_OverlapResponse(t *testing.T) {
	t.Run("aligned equal-size neighbors", func(t *testing.T) {
		medians := make([]float64, 0, 3)
		for _, overlap := range []float64{0, .08, .20} {
			settings := shelfQualitySettings(overlap)
			settings.MinimumShortEdge, settings.SizeVariation = .2, 0
			scene := generateQualityScene(t, image.Pt(320, 180), settings, 7, 512, true)
			var row []placement
			for _, p := range scene.plan.placements {
				if !p.repair && p.centerY-p.imageRect.height/2 <= .5 && p.centerY+p.imageRect.height/2 >= 18 {
					row = append(row, p)
				}
			}
			slices.SortFunc(row, func(a, b placement) int {
				if a.centerX < b.centerX {
					return -1
				}
				if a.centerX > b.centerX {
					return 1
				}
				return 0
			})
			var insets []float64
			for i := 1; i < len(row); i++ {
				left, right := row[i-1], row[i]
				insets = append(insets, math.Max(0, left.centerX+left.imageRect.width/2-(right.centerX-right.imageRect.width/2))/36)
			}
			if len(insets) < 4 {
				t.Fatalf("only %d measurable neighbors", len(insets))
			}
			slices.Sort(insets)
			medians = append(medians, insets[len(insets)/2])
		}
		t.Logf("aligned neighboring-photo inset medians 0%%/8%%/20%%: %v", medians)
		if !(medians[0] < medians[1] && medians[1] < medians[2] && medians[2]-medians[0] >= .1) {
			t.Errorf("aligned overlap response too weak: %v", medians)
		}
	})
	directory, err := os.MkdirTemp("", "picfetch-mosaic-overlap-")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("retained comparison renders: %s", directory)
	medians := make([]float64, 3)
	for setting, overlap := range []float64{0, .08, .20} {
		values := make([]float64, 0, 3)
		for _, seed := range []int64{7, 31, 987} {
			scene := generateQualityScene(t, image.Pt(320, 180), shelfQualitySettings(overlap), seed, 512, false)
			values = append(values, scene.coveredFraction())
			if seed == 7 {
				path := filepath.Join(directory, fmt.Sprintf("overlap-%.0f.png", overlap*100))
				file, err := os.Create(path)
				if err != nil {
					t.Fatal(err)
				}
				err = png.Encode(file, scene.result.Image())
				closeErr := file.Close()
				if err != nil {
					t.Fatal(err)
				}
				if closeErr != nil {
					t.Fatal(closeErr)
				}
			}
		}
		slices.Sort(values)
		medians[setting] = values[1]
	}
	t.Logf("median covered-photo fractions 0%%/8%%/20%% = %v", medians)
	if !(medians[0] < medians[1] && medians[1] < medians[2] && medians[2]-medians[0] >= .05) {
		t.Errorf("overlap response too weak or unordered: %v", medians)
	}
}

func TestGenerate_RandomLargeRotation(t *testing.T) {
	for _, target := range []image.Point{{320, 180}, {180, 320}} {
		t.Run(fmt.Sprint(target), func(t *testing.T) {
			settings := qualitySettings(90, .08)
			settings.Layout = LayoutRandom
			scene := generateQualityScene(t, target, settings, 31, 512, false)
			if scene.result.Bounds() != (image.Rectangle{Max: target}) {
				t.Fatalf("wrong output bounds: %v", scene.result.Bounds())
			}
			positive, negative := false, false
			for _, p := range scene.plan.placements {
				positive = positive || p.angle > 12
				negative = negative || p.angle < -12
				for _, value := range []float64{p.angle, p.centerX, p.centerY, p.imageRect.width, p.imageRect.height} {
					if math.IsNaN(value) || math.IsInf(value, 0) {
						t.Fatalf("non-finite placement: %+v", p)
					}
				}
				if math.Abs(p.angle) > 90 {
					t.Fatalf("rotation exceeds requested range: %g", p.angle)
				}
				sizes := []image.Point{{36, 24}, {18, 24}, {43, 24}, {18, 30}, {30, 24}}
				size := sizes[p.candidateID%len(sizes)]
				if math.Abs(p.imageRect.width/p.imageRect.height-float64(size.X)/float64(size.Y)) > 1e-12 {
					t.Fatalf("source aspect ratio changed: %+v", p)
				}
			}
			if !positive || !negative {
				t.Fatalf("missing large rotations: positive=%v negative=%v", positive, negative)
			}
			pixels := scene.result.Image().(*image.NRGBA)
			background := color.NRGBA{R: 28, G: 30, B: 34, A: 255}
			for y := range target.Y {
				for x := range target.X {
					if !scene.plan.covered[y*target.X+x] || pixels.NRGBAAt(x, y) == background {
						t.Fatalf("uncovered pixel at %d,%d", x, y)
					}
				}
			}
		})
	}
	t.Run("cancel during large-angle generation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		g := New()
		pixels := image.NewNRGBA(image.Rect(0, 0, 36, 24))
		g.load = func(_ context.Context, _ fyne.URI) (*loadedSource, error) {
			return &loadedSource{pixels: pixels, bounds: pixels.Bounds()}, nil
		}
		prepared := 0
		g.beforePrepare = func(_ preparationPlan) error {
			prepared++
			if prepared == 2 {
				cancel()
			}
			return nil
		}
		settings := qualitySettings(90, .08)
		settings.Layout = LayoutRandom
		result, err := g.Generate(ctx, mustRequest(t, []fyne.URI{storage.NewFileURI("/virtual/photo.png")}, image.Pt(320, 180), settings, 7))
		if err != context.Canceled || !result.Bounds().Empty() || prepared != 2 {
			t.Fatalf("cancelled generation: err=%v result=%v preparations=%d", err, result.Bounds(), prepared)
		}
	})
}
