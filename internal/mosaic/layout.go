package mosaic

import (
	"context"
	"fmt"
	"image"
	"math"
	"math/rand/v2"
)

type candidate struct {
	id     int
	aspect float64
}

type candidateFunc func() (candidate, error)

const (
	minimumPrimaryNovelty    = 0.45
	minimumPrimaryVisibility = 0.45
)

type floatRect struct {
	width  float64
	height float64
}

type placement struct {
	candidateID int
	centerX     float64
	centerY     float64
	imageRect   floatRect
	bodyWidth   float64
	bodyTop     float64
	bodyBottom  float64
	shadowSize  float64
	angle       float64
	frame       FrameStyle
	repair      bool
}

type layoutPlan struct {
	target     image.Point
	placements []placement
	covered    []bool
}

// LayoutError reports that valid candidates could not complete a covering
// layout within its progress bound.
type LayoutError struct {
	Err error
}

func (e *LayoutError) Error() string { return fmt.Sprintf("mosaic layout: %v", e.Err) }
func (e *LayoutError) Unwrap() error { return e.Err }

func planLayout(
	ctx context.Context,
	target image.Point,
	settings Settings,
	seed int64,
	next candidateFunc,
) (layoutPlan, error) {
	return walkLayout(ctx, target, settings, seed, next, nil)
}

func walkLayout(
	ctx context.Context,
	target image.Point,
	settings Settings,
	seed int64,
	next candidateFunc,
	onPlacement func(placement) error,
) (layoutPlan, error) {
	plan := layoutPlan{
		target:  target,
		covered: make([]bool, target.X*target.Y),
	}
	// Two explicit 64-bit words avoid package-global random state. Mixing the
	// signed seed into both words also keeps negative seeds deterministic.
	random := rand.New(rand.NewPCG(uint64(seed), uint64(seed)^0x9e3779b97f4a7c15))
	baseShort := float64(min(target.X, target.Y)) * settings.MinimumShortEdge
	if settings.SizeVariation < 1 {
		baseShort /= 1 - settings.SizeVariation
	}
	nextUncovered := 0
	primaryOwner := make([]uint32, len(plan.covered))
	primaryVisible := make([]int, 0)
	primaryArea := make([]int, 0)

	// Every placement is centered on an uncovered pixel (or reset there after
	// jitter), so it covers at least that pixel. Target area is therefore a
	// strict progress bound, not an arbitrary retry count.
	for attempt := 0; attempt < len(plan.covered); attempt++ {
		if err := ctx.Err(); err != nil {
			return layoutPlan{}, err
		}
		for nextUncovered < len(plan.covered) && plan.covered[nextUncovered] {
			nextUncovered++
		}
		if nextUncovered == len(plan.covered) {
			return plan, nil
		}
		x, y := nextUncovered%target.X, nextUncovered/target.X

		item, err := next()
		if err != nil {
			return layoutPlan{}, err
		}
		if item.aspect <= 0 || math.IsNaN(item.aspect) || math.IsInf(item.aspect, 0) {
			return layoutPlan{}, &LayoutError{Err: fmt.Errorf("candidate %d has invalid aspect ratio", item.id)}
		}

		variation := symmetric(random.Float64(), settings.SizeVariation)
		shorter := baseShort * (1 + variation)
		width, height := shorter, shorter
		if item.aspect >= 1 {
			width *= item.aspect
		} else {
			height /= item.aspect
		}
		angle := symmetric(random.Float64(), settings.MaximumRotation)
		anchorX, anchorY := float64(x)+0.5, float64(y)+0.5
		candidatePlacement := newPlacement(item.id, 0, 0, width, height, angle, settings.Frame, settings.DropShadow)
		// Put the first uncovered pixel just inside the card's leading edges.
		// This advances most of each card into uncovered space instead of
		// centering half of it over already covered pixels. Randomizing each
		// inset around the requested overlap supplies bounded organic jitter.
		overlap := settings.Overlap * shorter
		localX := -candidatePlacement.bodyWidth/2 + overlap*(0.5+random.Float64())
		localY := candidatePlacement.bodyTop + overlap*(0.5+random.Float64())
		radians := candidatePlacement.angle * math.Pi / 180
		sin, cos := math.Sincos(radians)
		candidatePlacement.centerX = anchorX - (localX*cos - localY*sin)
		candidatePlacement.centerY = anchorY - (localX*sin + localY*cos)
		if !placementCovers(candidatePlacement, anchorX, anchorY) {
			// Floating-point roundoff can put a zero-overlap anchor a fraction
			// outside the inclusive edge. Centering that one card still makes
			// strict progress without weakening the global safety bound.
			candidatePlacement.centerX = anchorX
			candidatePlacement.centerY = anchorY
		}
		candidatePlacement.repair = placementUncoveredFraction(plan.covered, target, candidatePlacement) < minimumPrimaryNovelty ||
			!preservesPrimaryVisibility(primaryOwner, primaryVisible, primaryArea, target, candidatePlacement)
		if onPlacement != nil {
			if err := onPlacement(candidatePlacement); err != nil {
				return layoutPlan{}, err
			}
		}

		plan.placements = append(plan.placements, candidatePlacement)
		primaryVisible = append(primaryVisible, 0)
		primaryArea = append(primaryArea, 0)
		if !candidatePlacement.repair {
			applyPrimaryPlacement(primaryOwner, primaryVisible, primaryArea, target, len(plan.placements)-1, candidatePlacement)
		}
		markCovered(plan.covered, target, candidatePlacement)
	}

	for nextUncovered < len(plan.covered) && plan.covered[nextUncovered] {
		nextUncovered++
	}
	if nextUncovered < len(plan.covered) {
		return layoutPlan{}, &LayoutError{Err: fmt.Errorf("coverage progress bound exhausted")}
	}

	return plan, nil
}

func symmetric(unit, maximum float64) float64 {
	return (unit*2 - 1) * maximum
}

func newPlacement(id int, centerX, centerY, width, height, angle float64, frame FrameStyle, dropShadow bool) placement {
	shorter := math.Min(width, height)
	border, footer := frameInsets(frame, shorter)
	shadow := 0.0
	if dropShadow {
		shadow = math.Max(0.5, shorter*0.035)
	}

	return placement{
		candidateID: id,
		centerX:     centerX,
		centerY:     centerY,
		imageRect:   floatRect{width: width, height: height},
		bodyWidth:   width + border*2,
		bodyTop:     -height/2 - border,
		bodyBottom:  height/2 + border + footer,
		shadowSize:  shadow,
		angle:       angle,
		frame:       frame,
	}
}

func frameInsets(frame FrameStyle, shorter float64) (border, footer float64) {
	switch frame {
	case FrameThinLight, FrameThinDark:
		return math.Max(1, shorter*0.025), 0
	case FramePolaroid:
		return math.Max(1, shorter*0.055), math.Max(2, shorter*0.20)
	default:
		return 0, 0
	}
}

func markCovered(covered []bool, target image.Point, placement placement) {
	bounds := placementPixelBounds(placement).Intersect(image.Rectangle{Max: target})
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			index := y*target.X + x
			if !covered[index] && placementCovers(placement, float64(x)+0.5, float64(y)+0.5) {
				covered[index] = true
			}
		}
	}
}

func placementUncoveredFraction(covered []bool, target image.Point, placement placement) float64 {
	bounds := placementPixelBounds(placement).Intersect(image.Rectangle{Max: target})
	total, uncovered := 0, 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if !placementCovers(placement, float64(x)+0.5, float64(y)+0.5) {
				continue
			}
			total++
			if !covered[y*target.X+x] {
				uncovered++
			}
		}
	}
	if total == 0 {
		return 0
	}

	return float64(uncovered) / float64(total)
}

func preservesPrimaryVisibility(
	owners []uint32,
	visible []int,
	areas []int,
	target image.Point,
	placement placement,
) bool {
	occluded := make(map[int]int)
	bounds := placementPixelBounds(placement).Intersect(image.Rectangle{Max: target})
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if !placementCovers(placement, float64(x)+0.5, float64(y)+0.5) {
				continue
			}
			owner := owners[y*target.X+x]
			if owner != 0 {
				occluded[int(owner)-1]++
			}
		}
	}
	for index, pixels := range occluded {
		if index >= len(areas) || areas[index] == 0 {
			continue
		}
		if float64(visible[index]-pixels)/float64(areas[index]) < minimumPrimaryVisibility {
			return false
		}
	}

	return true
}

func applyPrimaryPlacement(
	owners []uint32,
	visible []int,
	areas []int,
	target image.Point,
	index int,
	placement placement,
) {
	bounds := placementPixelBounds(placement).Intersect(image.Rectangle{Max: target})
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if !placementCovers(placement, float64(x)+0.5, float64(y)+0.5) {
				continue
			}
			pixel := y*target.X + x
			if previous := owners[pixel]; previous != 0 {
				visible[int(previous)-1]--
			}
			owners[pixel] = uint32(index + 1)
			visible[index]++
			areas[index]++
		}
	}
}

func placementPixelBounds(placement placement) image.Rectangle {
	halfWidth := placement.bodyWidth / 2
	shadow := placement.shadowSize
	corners := [][2]float64{
		{-halfWidth, placement.bodyTop},
		{halfWidth + shadow, placement.bodyTop},
		{halfWidth + shadow, placement.bodyBottom + shadow},
		{-halfWidth, placement.bodyBottom + shadow},
	}
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	radians := placement.angle * math.Pi / 180
	sin, cos := math.Sincos(radians)
	for _, corner := range corners {
		x := placement.centerX + corner[0]*cos - corner[1]*sin
		y := placement.centerY + corner[0]*sin + corner[1]*cos
		minX, minY = math.Min(minX, x), math.Min(minY, y)
		maxX, maxY = math.Max(maxX, x), math.Max(maxY, y)
	}

	return image.Rect(int(math.Floor(minX))-1, int(math.Floor(minY))-1,
		int(math.Ceil(maxX))+1, int(math.Ceil(maxY))+1)
}

func placementCovers(placement placement, x, y float64) bool {
	localX, localY := inverseRotate(placement, x, y)
	halfWidth := placement.bodyWidth / 2
	return localX >= -halfWidth && localX <= halfWidth &&
		localY >= placement.bodyTop && localY <= placement.bodyBottom
}

func inverseRotate(placement placement, x, y float64) (float64, float64) {
	radians := -placement.angle * math.Pi / 180
	sin, cos := math.Sincos(radians)
	dx, dy := x-placement.centerX, y-placement.centerY

	return dx*cos - dy*sin, dx*sin + dy*cos
}
