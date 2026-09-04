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
	settings := DefaultSettings()
	target := image.Pt(200, 100)
	plan, err := planLayout(context.Background(), target, settings, 9, cyclingCandidates([]candidate{{id: 1, aspect: 2}, {id: 2, aspect: 0.5}}))
	if err != nil {
		t.Fatal(err)
	}

	base := float64(min(target.X, target.Y)) * settings.MinimumShortEdge
	minimum := base * (1 - settings.SizeVariation)
	maximum := base * (1 + settings.SizeVariation)
	for _, placement := range plan.placements {
		shorter := math.Min(placement.imageRect.width, placement.imageRect.height)
		if shorter < minimum-0.0001 || shorter > maximum+0.0001 {
			t.Errorf("shorter edge %f outside [%f,%f]", shorter, minimum, maximum)
		}
		if math.Abs(placement.angle) > settings.MaximumRotation {
			t.Errorf("angle %f exceeds %f", placement.angle, settings.MaximumRotation)
		}
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
	firstRight := first.centerX + first.bodyWidth/2 + first.shadowSize
	secondLeft := second.centerX - second.bodyWidth/2
	overlap := firstRight - secondLeft
	if overlap < 0 || overlap > first.imageRect.width*0.25 {
		t.Fatalf("first-row footprint overlap = %.2f of %.2f pixels, want light overlap", overlap, first.imageRect.width)
	}
	if first.centerX < first.imageRect.width*0.25 || first.centerY < first.imageRect.height*0.25 {
		t.Fatalf("first placement center = (%.2f,%.2f), want most of the card inside the canvas", first.centerX, first.centerY)
	}
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
