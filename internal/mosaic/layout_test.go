package mosaic

import (
	"context"
	"errors"
	"image"
	"math"
	"reflect"
	"testing"
)

func TestLayout_Deterministic(t *testing.T) {
	settings := DefaultSettings()
	candidates := []candidate{{id: 1, aspect: 1.5}, {id: 2, aspect: 0.75}, {id: 3, aspect: 2}}

	first, err := planLayout(context.Background(), image.Pt(160, 90), settings, 1234, cyclingCandidates(candidates))
	if err != nil {
		t.Fatal(err)
	}
	second, err := planLayout(context.Background(), image.Pt(160, 90), settings, 1234, cyclingCandidates(candidates))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first.placements, second.placements) {
		t.Fatal("fixed layout inputs produced different placements")
	}
}

func TestLayout_Bounds(t *testing.T) {
	tests := []struct {
		name      string
		minimum   float64
		variation float64
	}{
		{name: "no variation at minimum setting", minimum: 0.10, variation: 0},
		{name: "defaults", minimum: 0.18, variation: 0.12},
		{name: "maximum variation at maximum setting", minimum: 0.30, variation: 0.25},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings := DefaultSettings()
			settings.MinimumShortEdge = tt.minimum
			settings.SizeVariation = tt.variation
			target := image.Pt(200, 100)
			plan, err := planLayout(context.Background(), target, settings, 9, cyclingCandidates([]candidate{{id: 1, aspect: 2}, {id: 2, aspect: 0.5}}))
			if err != nil {
				t.Fatal(err)
			}

			minimum := float64(min(target.X, target.Y)) * settings.MinimumShortEdge
			base := minimum / (1 - settings.SizeVariation)
			maximum := base * (1 + settings.SizeVariation)
			var firstSize float64
			var varied bool
			for _, placement := range plan.placements {
				shorter := math.Min(placement.imageRect.width, placement.imageRect.height)
				if shorter < minimum-0.0001 || shorter > maximum+0.0001 {
					t.Errorf("shorter edge %f outside [%f,%f]", shorter, minimum, maximum)
				}
				if math.Abs(placement.angle) > settings.MaximumRotation {
					t.Errorf("angle %f exceeds %f", placement.angle, settings.MaximumRotation)
				}
				if firstSize == 0 {
					firstSize = shorter
				} else if math.Abs(shorter-firstSize) > 0.0001 {
					varied = true
				}
			}
			if settings.SizeVariation > 0 && !varied {
				t.Fatal("configured card-size variation produced one shorter-edge size")
			}
		})
	}
}

func TestLayout_Coverage(t *testing.T) {
	sizes := []image.Point{
		image.Pt(192, 108),
		image.Pt(160, 100),
		image.Pt(210, 90),
		image.Pt(120, 90),
		image.Pt(90, 160),
		image.Pt(7, 5),
		image.Pt(101, 67),
	}
	for _, size := range sizes {
		t.Run(size.String(), func(t *testing.T) {
			plan, err := planLayout(context.Background(), size, DefaultSettings(), 77,
				cyclingCandidates([]candidate{{id: 1, aspect: 1.6}, {id: 2, aspect: 0.7}}))
			if err != nil {
				t.Fatal(err)
			}
			for y := range size.Y {
				for x := range size.X {
					if !plan.covered[y*size.X+x] {
						t.Fatalf("target pixel (%d,%d) is uncovered", x, y)
					}
				}
			}
		})
	}
}

func TestLayout_VariationOneAvoidsNonFiniteCardSizes(t *testing.T) {
	settings := DefaultSettings()
	settings.SizeVariation = 1
	settings.Overlap = 0
	settings.MaximumRotation = 0
	settings.Frame = FrameNone
	settings.DropShadow = false
	plan, err := planLayout(context.Background(), image.Pt(20, 10), settings, 17,
		cyclingCandidates([]candidate{{id: 1, aspect: 1}}))
	if err != nil {
		t.Fatal(err)
	}
	for _, placement := range plan.placements {
		shorter := math.Min(placement.imageRect.width, placement.imageRect.height)
		if shorter <= 0 || math.IsNaN(shorter) || math.IsInf(shorter, 0) {
			t.Fatalf("variation 1 produced invalid shorter edge %v", shorter)
		}
	}
}

func TestLayout_OverlapStaysLightOnRegularRow(t *testing.T) {
	settings := DefaultSettings()
	settings.MinimumShortEdge = 0.20
	settings.SizeVariation = 0
	settings.MaximumRotation = 0
	settings.Overlap = 0.08
	plan, err := planLayout(context.Background(), image.Pt(120, 60), settings, 31,
		cyclingCandidates([]candidate{{id: 1, aspect: 1}}))
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.placements) < 2 {
		t.Fatalf("placements = %d, want at least 2", len(plan.placements))
	}
	first, second := plan.placements[0], plan.placements[1]
	firstRight := first.centerX + first.bodyWidth/2
	secondLeft := second.centerX - second.bodyWidth/2
	overlap := firstRight - secondLeft
	if overlap < 0 || overlap > first.imageRect.width*0.25 {
		t.Fatalf("first-row footprint overlap = %.2f of %.2f pixels, want light overlap", overlap, first.imageRect.width)
	}
	if first.centerX < first.imageRect.width*0.25 || first.centerY < first.imageRect.height*0.25 {
		t.Fatalf("first placement center = (%.2f,%.2f), want most of the card inside the canvas", first.centerX, first.centerY)
	}
}

func TestLayout_ShadowDoesNotCountAsOpaqueCoverage(t *testing.T) {
	target := image.Pt(30, 30)
	placed := newPlacement(1, 15, 15, 10, 10, 0, FrameNone, true)
	if placed.shadowSize == 0 {
		t.Fatal("shadow-enabled placement has no shadow geometry")
	}
	covered := make([]bool, target.X*target.Y)
	markCovered(covered, target, placed)
	shadowOnly := image.Pt(20, 15)
	if covered[shadowOnly.Y*target.X+shadowOnly.X] {
		t.Fatalf("shadow-only pixel %v counted as opaque layout coverage", shadowOnly)
	}
	withoutShadow := newPlacement(1, 15, 15, 10, 10, 0, FrameNone, false)
	if withoutShadow.shadowSize != 0 {
		t.Fatalf("shadow-disabled placement footprint = %.2f, want zero", withoutShadow.shadowSize)
	}
}

func TestLayout_ConfiguredOverlapDoesNotBuryInteriorCards(t *testing.T) {
	settings := DefaultSettings()
	settings.MinimumShortEdge = 0.16
	settings.SizeVariation = 0.18
	settings.Overlap = 0.07
	settings.MaximumRotation = 12
	settings.Frame = FrameThinLight
	target := image.Pt(320, 180)
	plan, err := planLayout(context.Background(), target, settings, 7, cyclingCandidates([]candidate{
		{id: 1, aspect: 1.5},
		{id: 2, aspect: 0.75},
		{id: 3, aspect: 1.8},
		{id: 4, aspect: 0.6},
		{id: 5, aspect: 1.25},
	}))
	if err != nil {
		t.Fatal(err)
	}

	minimumVisible := 1.0
	checked := 0
	repairs := 0
	for index, placed := range plan.placements {
		if placed.repair {
			repairs++
			continue
		}
		if placed.centerX < 0 || placed.centerX >= float64(target.X) || placed.centerY < 0 || placed.centerY >= float64(target.Y) {
			continue
		}
		area, visible := visibleCardPixels(plan.placements, index, target)
		nominalArea := placed.bodyWidth * (placed.bodyBottom - placed.bodyTop)
		if area < int(nominalArea*0.75) {
			continue
		}
		checked++
		minimumVisible = math.Min(minimumVisible, float64(visible)/float64(area))
	}
	if checked < 4 {
		t.Fatalf("layout had %d substantially in-canvas primary cards, want at least 4", checked)
	}
	if repairs == 0 {
		t.Fatal("layout had no underneath coverage repairs")
	}
	if minimumVisible < 0.45 {
		t.Fatalf("least-visible interior card retains %.1f%% of its pixels, want at least 45%%", minimumVisible*100)
	}
}

func visibleCardPixels(placements []placement, index int, target image.Point) (area, visible int) {
	placed := placements[index]
	bounds := placementPixelBounds(placed).Intersect(image.Rectangle{Max: target})
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			centerX, centerY := float64(x)+0.5, float64(y)+0.5
			if !cardBodyCovers(placed, centerX, centerY) {
				continue
			}
			area++
			coveredLater := false
			for later := index + 1; later < len(placements); later++ {
				if !placements[later].repair && cardBodyCovers(placements[later], centerX, centerY) {
					coveredLater = true
					break
				}
			}
			if !coveredLater {
				visible++
			}
		}
	}

	return area, visible
}

func cardBodyCovers(placed placement, x, y float64) bool {
	localX, localY := inverseRotate(placed, x, y)
	return localX >= -placed.bodyWidth/2 && localX <= placed.bodyWidth/2 &&
		localY >= placed.bodyTop && localY <= placed.bodyBottom
}

func TestLayout_PoolStopsAtCoverageAndRepeatsOneSource(t *testing.T) {
	var calls int
	next := func() (candidate, error) {
		calls++
		return candidate{id: calls, aspect: 1}, nil
	}
	plan, err := planLayout(context.Background(), image.Pt(80, 50), DefaultSettings(), 5, next)
	if err != nil {
		t.Fatal(err)
	}
	if calls != len(plan.placements) {
		t.Fatalf("candidate calls = %d, placements = %d; pool was read beyond coverage", calls, len(plan.placements))
	}

	one, err := planLayout(context.Background(), image.Pt(80, 50), DefaultSettings(), 5,
		cyclingCandidates([]candidate{{id: 42, aspect: 1.25}}))
	if err != nil {
		t.Fatal(err)
	}
	if len(one.placements) < 2 {
		t.Fatalf("one-source layout used %d placement, want repetition", len(one.placements))
	}
	for _, placement := range one.placements {
		if placement.candidateID != 42 {
			t.Fatalf("placement source = %d, want repeated source 42", placement.candidateID)
		}
	}
}

func TestLayout_Cancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	_, err := planLayout(ctx, image.Pt(120, 80), DefaultSettings(), 1, func() (candidate, error) {
		calls++
		cancel()
		return candidate{id: 1, aspect: 1}, nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("planLayout() = %v, want context.Canceled", err)
	}
	if calls != 1 {
		t.Fatalf("candidate calls = %d, want 1 before cancellation", calls)
	}

	want, err := planLayout(context.Background(), image.Pt(120, 80), DefaultSettings(), 1,
		cyclingCandidates([]candidate{{id: 1, aspect: 1}}))
	if err != nil {
		t.Fatal(err)
	}
	got, err := planLayout(context.Background(), image.Pt(120, 80), DefaultSettings(), 1,
		cyclingCandidates([]candidate{{id: 1, aspect: 1}}))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.placements, want.placements) {
		t.Fatal("a cancelled run changed later deterministic output")
	}
}

func cyclingCandidates(candidates []candidate) candidateFunc {
	index := 0
	return func() (candidate, error) {
		candidate := candidates[index%len(candidates)]
		index++
		return candidate, nil
	}
}
