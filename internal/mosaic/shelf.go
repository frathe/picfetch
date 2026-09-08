package mosaic

import (
	"context"
	"fmt"
	"image"
	"math"
	"math/rand/v2"
)

type shelfRow struct {
	start, end    int
	minimumHeight float64
}

// planShelfLayout builds complete horizontal shelves. Every card body begins
// at the shelf edge and the following shelf starts at the shortest body in its
// row. This bounds layering while preserving the source's aspect ratio. Shelf
// deliberately leaves cards axis-aligned, so it does not read MaximumRotation.
func planShelfLayout(
	ctx context.Context,
	target image.Point,
	settings Settings,
	seed int64,
	next candidateFunc,
) (layoutPlan, error) {
	plan := layoutPlan{target: target, covered: make([]bool, target.X*target.Y)}
	minimum := float64(min(target.X, target.Y)) * settings.MinimumShortEdge
	base := minimum
	if settings.SizeVariation < 1 {
		base /= 1 - settings.SizeVariation
	}
	maximum := base * (1 + settings.SizeVariation)
	if settings.SizeVariation >= 1 {
		maximum = minimum
	}
	random := rand.New(rand.NewPCG(uint64(seed), uint64(seed)^0x9e3779b97f4a7c15))

	var shelves []shelfRow
	for top := 0.0; top < float64(target.Y); {
		if err := ctx.Err(); err != nil {
			return layoutPlan{}, err
		}
		row := shelfRow{start: len(plan.placements), minimumHeight: math.Inf(1)}
		left, previousRight := 0.0, 0.0
		for len(plan.placements) == row.start || previousRight < float64(target.X) {
			if err := ctx.Err(); err != nil {
				return layoutPlan{}, err
			}
			item, err := next()
			if err != nil {
				return layoutPlan{}, err
			}
			if err := ctx.Err(); err != nil {
				return layoutPlan{}, err
			}
			if item.aspect <= 0 || math.IsNaN(item.aspect) || math.IsInf(item.aspect, 0) {
				return layoutPlan{}, &LayoutError{Err: fmt.Errorf("candidate %d has invalid aspect ratio", item.id)}
			}

			// A fractional card edge can require many more placements than there
			// are physical pixels on an extremely thin target. Keep every Shelf
			// card at least one pixel wide on its shorter side.
			shorter := math.Max(1, shelfShorter(item.aspect, minimum, maximum, random.Float64()))
			width, height := shorter, shorter
			if item.aspect >= 1 {
				width *= item.aspect
			} else {
				height /= item.aspect
			}
			placed := newPlacement(item.id, 0, 0, width, height, 0, settings.Frame, settings.DropShadow)
			if len(plan.placements) != row.start {
				previous := plan.placements[len(plan.placements)-1]
				left = previousRight - shelfOverlap(settings.Overlap, previous.bodyWidth, placed.bodyWidth)
			}
			placed.centerX = left + placed.bodyWidth/2
			placed.centerY = top - placed.bodyTop
			plan.placements = append(plan.placements, placed)
			previousRight = left + placed.bodyWidth
			row.minimumHeight = math.Min(row.minimumHeight, placed.bodyBottom-placed.bodyTop)
		}
		row.end = len(plan.placements)
		shelves = append(shelves, row)
		if row.minimumHeight <= 0 || math.IsNaN(row.minimumHeight) || math.IsInf(row.minimumHeight, 0) {
			return layoutPlan{}, &LayoutError{Err: fmt.Errorf("invalid shelf height")}
		}
		top += row.minimumHeight
	}

	for _, placed := range plan.placements {
		if err := ctx.Err(); err != nil {
			return layoutPlan{}, err
		}
		markCovered(plan.covered, target, placed)
	}
	for index, covered := range plan.covered {
		if index&1023 == 0 {
			if err := ctx.Err(); err != nil {
				return layoutPlan{}, err
			}
		}
		if !covered {
			return layoutPlan{}, &LayoutError{Err: fmt.Errorf("shelf left pixel %d uncovered", index)}
		}
	}
	if err := ctx.Err(); err != nil {
		return layoutPlan{}, err
	}

	// Move a short interior card to the repair layer. Its neighbours retain a
	// real shared seam, while its shallow body limits later-card occlusion.
	repair, score := -1, math.Inf(1)
	for shelfIndex, row := range shelves {
		if row.end-row.start < 3 || shelfIndex > 0 && repair >= 0 {
			continue
		}
		for index := row.start + 1; index < row.end-1; index++ {
			placed := plan.placements[index]
			if placed.candidateID == plan.placements[index-1].candidateID {
				continue
			}
			candidate := (placed.bodyBottom - placed.bodyTop) / row.minimumHeight
			if candidate < score {
				repair, score = index, candidate
			}
		}
	}

	if repair >= 0 {
		primary := make([]placement, 0, len(plan.placements))
		underneath := make([]placement, 0, 1)
		for index, placed := range plan.placements {
			if index == repair {
				placed.repair = true
				underneath = append(underneath, placed)
				continue
			}
			primary = append(primary, placed)
		}
		plan.placements = append(primary, underneath...)
	}

	return plan, nil
}

func shelfShorter(aspect, minimum, maximum, unit float64) float64 {
	span := maximum - minimum
	if aspect >= 1 {
		return maximum - span*.15*unit
	}

	return minimum + span*.15*unit
}

func shelfOverlap(configured, previousWidth, currentWidth float64) float64 {
	overlap := configured * math.Min(previousWidth, currentWidth)
	if configured > 0 {
		// A fixed five-pixel seam makes regular cards read as layered, but it
		// must never exceed either adjacent body width or a shelf can move
		// backward on a tiny canvas.
		overlap = math.Min(math.Max(overlap, 5), math.Min(previousWidth, currentWidth)*.25)
	}

	return overlap
}
