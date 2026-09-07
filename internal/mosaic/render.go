package mosaic

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/math/f64"
	"golang.org/x/image/vector"
)

const (
	sourceRenderScale = 2.0
	sourceEdgePadding = 4
	transformBandRows = 32
)

func renderPlacement(ctx context.Context, destination *image.NRGBA, source *loadedSource, placement placement) error {
	return renderPlacementWithBudget(ctx, destination, source, placement, maxPreparationBytes, nil)
}

func renderPlacementWithBudget(ctx context.Context, destination *image.NRGBA, source *loadedSource, placement placement, budget uint64, beforePrepare func(preparationPlan) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	bounds := placementPixelBounds(placement).Intersect(destination.Bounds())
	if bounds.Empty() {
		return nil
	}
	plan, err := planPreparation(source, placement, bounds)
	if err != nil {
		return err
	}
	var sourceImage image.Image
	render := func(tile image.Rectangle, plan preparationPlan) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if plan.bytes > budget {
			return fmt.Errorf("mosaic preparation needs %d bytes, budget %d", plan.bytes, budget)
		}
		if beforePrepare != nil {
			if err := beforePrepare(plan); err != nil {
				return err
			}
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if sourceImage == nil {
			width, height := plan.width, plan.height
			if plan.vectorSize.X > 0 {
				width, height = plan.vectorSize.X, plan.vectorSize.Y
			}
			sourceImage, err = sourceImageAt(source, width, height)
			if err != nil {
				return err
			}
		}
		return renderPlacementTile(ctx, destination, sourceImage, placement, tile, plan)
	}
	if plan.bytes <= budget {
		return render(bounds, plan)
	}
	transform := preparationTransform(placement, plan)
	sourceSize := source.bounds.Size()
	if source.pixels != nil {
		sourceSize = source.pixels.Bounds().Size()
	}
	for top := bounds.Min.Y; top < bounds.Max.Y; top += preparationTileSize {
		for left := bounds.Min.X; left < bounds.Max.X; left += preparationTileSize {
			tile := image.Rect(left, top, min(left+preparationTileSize, bounds.Max.X), min(top+preparationTileSize, bounds.Max.Y))
			if err := render(tile, plan.forTile(tile, transform, sourceSize, budget)); err != nil {
				return err
			}
		}
	}
	return nil
}

func renderPlacementTile(ctx context.Context, destination *image.NRGBA, source image.Image, placement placement, bounds image.Rectangle, plan preparationPlan) error {
	halfBodyWidth := placement.bodyWidth / 2
	if placement.shadowSize > 0 {
		shadow := placement.shadowSize
		mask := rotatedRectangleMask(
			bounds,
			placement,
			plan.tiled,
			-halfBodyWidth+shadow,
			placement.bodyTop+shadow,
			halfBodyWidth+shadow,
			placement.bodyBottom+shadow,
		)
		if err := drawUniformThroughMask(ctx, destination, bounds, mask, color.NRGBA{A: 105}); err != nil {
			return err
		}
	}

	backing := frameColor(placement.frame)
	if placement.frame != FrameNone {
		mask := rotatedRectangleMask(
			bounds,
			placement,
			plan.tiled,
			-halfBodyWidth,
			placement.bodyTop,
			halfBodyWidth,
			placement.bodyBottom,
		)
		if err := drawUniformThroughMask(ctx, destination, bounds, mask, backing); err != nil {
			return err
		}
	}

	prepared, transform, err := prepareSourceLayer(ctx, source, placement, backing, plan)
	if err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	halfImageWidth := placement.imageRect.width / 2
	halfImageHeight := placement.imageRect.height / 2
	mask := rotatedRectangleMask(
		bounds,
		placement,
		plan.tiled,
		-halfImageWidth,
		-halfImageHeight,
		halfImageWidth,
		halfImageHeight,
	)
	options := &xdraw.Options{
		DstMask:  mask,
		DstMaskP: image.Pt(-bounds.Min.X, -bounds.Min.Y),
	}
	for top := bounds.Min.Y; top < bounds.Max.Y; top += transformBandRows {
		bottom := min(top+transformBandRows, bounds.Max.Y)
		band := destination.SubImage(image.Rect(bounds.Min.X, top, bounds.Max.X, bottom)).(*image.NRGBA)
		xdraw.ApproxBiLinear.Transform(band, transform, prepared, prepared.Bounds(), xdraw.Over, options)
		if err := ctx.Err(); err != nil {
			return err
		}
	}

	return nil
}

func prepareSourceLayer(ctx context.Context, source image.Image, placement placement, backing color.NRGBA, plan preparationPlan) (*image.NRGBA, f64.Aff3, error) {
	if err := ctx.Err(); err != nil {
		return nil, f64.Aff3{}, err
	}
	interior := plan.interior()
	layer := image.NewNRGBA(plan.layer)
	draw.Draw(layer, layer.Bounds(), image.NewUniform(backing), image.Point{}, draw.Src)
	if !plan.tiled {
		xdraw.CatmullRom.Scale(layer, interior, source, source.Bounds(), xdraw.Over, nil)
	} else if plan.separable {
		if err := scaleSourceRegion(ctx, layer, interior, source); err != nil {
			return nil, f64.Aff3{}, err
		}
	} else {
		sourceBounds := source.Bounds()
		xScale := float64(plan.width) / float64(sourceBounds.Dx())
		yScale := float64(plan.height) / float64(sourceBounds.Dy())
		transform := f64.Aff3{xScale, 0, float64(interior.Min.X) - xScale*float64(sourceBounds.Min.X),
			0, yScale, float64(interior.Min.Y) - yScale*float64(sourceBounds.Min.Y)}
		visible := interior.Intersect(layer.Bounds())
		for top := visible.Min.Y; top < visible.Max.Y; top += transformBandRows {
			if err := ctx.Err(); err != nil {
				return nil, f64.Aff3{}, err
			}
			band := layer.SubImage(image.Rect(visible.Min.X, top, visible.Max.X, min(top+transformBandRows, visible.Max.Y))).(*image.NRGBA)
			xdraw.CatmullRom.Transform(band, transform, source, sourceBounds, xdraw.Over, nil)
		}
	}
	// The destination mask owns the silhouette. Padding samples the nearest
	// interior color, so affine filtering never blends transparent black.
	extendSourceEdges(layer, interior.Intersect(layer.Bounds()))
	return layer, preparationTransform(placement, plan), nil
}

func preparationTransform(placement placement, plan preparationPlan) f64.Aff3 {
	width, height := plan.width, plan.height
	scaleX := placement.imageRect.width / float64(width)
	scaleY := placement.imageRect.height / float64(height)
	halfImageWidth := placement.imageRect.width / 2
	halfImageHeight := placement.imageRect.height / 2
	localMinX := -halfImageWidth - float64(sourceEdgePadding)*scaleX
	localMinY := -halfImageHeight - float64(sourceEdgePadding)*scaleY
	radians := placement.angle * math.Pi / 180
	sin, cos := math.Sincos(radians)
	transform := f64.Aff3{
		cos * scaleX,
		-sin * scaleY,
		placement.centerX + localMinX*cos - localMinY*sin,
		sin * scaleX,
		cos * scaleY,
		placement.centerY + localMinX*sin + localMinY*cos,
	}

	return transform
}

func sourceImageAt(source *loadedSource, width, height int) (image.Image, error) {
	width = max(1, width)
	height = max(1, height)
	if source.vector != nil {
		return source.vector.RasterAt(width, height)
	}
	if source.pixels == nil {
		return nil, fmt.Errorf("decoded image has no pixels")
	}

	return source.pixels, nil
}

func extendSourceEdges(layer *image.NRGBA, interior image.Rectangle) {
	if interior.Empty() {
		return
	}
	for y := interior.Min.Y; y < interior.Max.Y; y++ {
		left := layer.NRGBAAt(interior.Min.X, y)
		right := layer.NRGBAAt(interior.Max.X-1, y)
		for x := layer.Bounds().Min.X; x < interior.Min.X; x++ {
			layer.SetNRGBA(x, y, left)
		}
		for x := interior.Max.X; x < layer.Bounds().Max.X; x++ {
			layer.SetNRGBA(x, y, right)
		}
	}
	for x := layer.Bounds().Min.X; x < layer.Bounds().Max.X; x++ {
		top := layer.NRGBAAt(x, interior.Min.Y)
		bottom := layer.NRGBAAt(x, interior.Max.Y-1)
		for y := layer.Bounds().Min.Y; y < interior.Min.Y; y++ {
			layer.SetNRGBA(x, y, top)
		}
		for y := interior.Max.Y; y < layer.Bounds().Max.Y; y++ {
			layer.SetNRGBA(x, y, bottom)
		}
	}
}

func rotatedRectangleMask(
	bounds image.Rectangle,
	placement placement,
	clip bool,
	left, top, right, bottom float64,
) *image.Alpha {
	rasterBounds := bounds
	if placement.angle != 0 {
		// The filter needs actual off-canvas coverage on each side. Clipping
		// before filtering would turn the canvas boundary into a photo edge.
		rasterBounds = bounds.Inset(-1)
	}
	mask := image.NewAlpha(image.Rect(0, 0, rasterBounds.Dx(), rasterBounds.Dy()))
	rasterHeight := rasterBounds.Dy()
	if clip {
		// x/image uses fixed-point edges for surfaces <= 512 on both axes.
		// Their accumulated rounding changes when an edge is split into tiles.
		// A padded raster selects the floating-point path used by large full
		// surfaces; the extra rows are scratch only and are never drawn.
		rasterHeight = max(rasterHeight, 513)
	}
	rasterizer := vector.NewRasterizer(rasterBounds.Dx(), rasterHeight)
	radians := placement.angle * math.Pi / 180
	sin, cos := math.Sincos(radians)
	toMask := func(x, y float64) f64.Vec2 {
		return f64.Vec2{placement.centerX + x*cos - y*sin - float64(rasterBounds.Min.X),
			placement.centerY + x*sin + y*cos - float64(rasterBounds.Min.Y)}
	}
	points := []f64.Vec2{toMask(left, top), toMask(right, top), toMask(right, bottom), toMask(left, bottom)}
	if clip {
		// Clip before float32/fixed-point rasterization. Very long off-canvas
		// edges otherwise overflow or accumulate error while walking to a tile.
		points = clipMaskPolygon(points, rasterBounds.Size())
	}
	if len(points) == 0 {
		return image.NewAlpha(image.Rectangle{Max: bounds.Size()})
	}
	rasterizer.MoveTo(float32(points[0][0]), float32(points[0][1]))
	for _, point := range points[1:] {
		rasterizer.LineTo(float32(point[0]), float32(point[1]))
	}
	rasterizer.ClosePath()
	rasterizer.Draw(mask, mask.Bounds(), image.Opaque, image.Point{})
	if placement.angle != 0 {
		softenCoverageMask(mask)
		mask = mask.SubImage(mask.Bounds().Inset(1)).(*image.Alpha)
		// Callers address the returned mask relative to bounds.Min. Rebase
		// the cropped view without copying its pixels or changing its stride.
		mask.Rect = mask.Rect.Sub(mask.Rect.Min)
	}

	return mask
}

func clipMaskPolygon(points []f64.Vec2, size image.Point) []f64.Vec2 {
	for axis, limit := range []float64{float64(size.X), float64(size.Y)} {
		for _, upper := range []bool{false, true} {
			boundary := 0.0
			if upper {
				boundary = limit
			}
			inside := func(p f64.Vec2) bool {
				if upper {
					return p[axis] <= boundary
				}
				return p[axis] >= boundary
			}
			clipped := make([]f64.Vec2, 0, len(points)+1)
			for i, a := range points {
				b := points[(i+1)%len(points)]
				if inside(a) {
					clipped = append(clipped, a)
				}
				if inside(a) != inside(b) {
					fraction := (boundary - a[axis]) / (b[axis] - a[axis])
					crossing := f64.Vec2{a[0] + fraction*(b[0]-a[0]), a[1] + fraction*(b[1]-a[1])}
					crossing[axis] = boundary
					clipped = append(clipped, crossing)
				}
			}
			points = clipped
		}
	}
	return points
}

func softenCoverageMask(mask *image.Alpha) {
	// A separable 1:14:1 kernel is deliberately much narrower than a normal
	// image blur. It removes the last mask-quantization steps without touching
	// any source-photo pixels.
	horizontal := image.NewAlpha(mask.Bounds())
	for y := mask.Bounds().Min.Y; y < mask.Bounds().Max.Y; y++ {
		for x := mask.Bounds().Min.X; x < mask.Bounds().Max.X; x++ {
			offset := mask.PixOffset(x, y)
			left, right := uint32(0), uint32(0)
			if x > mask.Bounds().Min.X {
				left = uint32(mask.Pix[offset-1])
			}
			if x+1 < mask.Bounds().Max.X {
				right = uint32(mask.Pix[offset+1])
			}
			horizontal.Pix[horizontal.PixOffset(x, y)] = uint8((left + 14*uint32(mask.Pix[offset]) + right + 8) / 16)
		}
	}
	for y := mask.Bounds().Min.Y; y < mask.Bounds().Max.Y; y++ {
		for x := mask.Bounds().Min.X; x < mask.Bounds().Max.X; x++ {
			offset := horizontal.PixOffset(x, y)
			top, bottom := uint32(0), uint32(0)
			if y > mask.Bounds().Min.Y {
				top = uint32(horizontal.Pix[offset-horizontal.Stride])
			}
			if y+1 < mask.Bounds().Max.Y {
				bottom = uint32(horizontal.Pix[offset+horizontal.Stride])
			}
			mask.Pix[mask.PixOffset(x, y)] = uint8((top + 14*uint32(horizontal.Pix[offset]) + bottom + 8) / 16)
		}
	}
}

func drawUniformThroughMask(
	ctx context.Context,
	destination *image.NRGBA,
	bounds image.Rectangle,
	mask *image.Alpha,
	value color.NRGBA,
) error {
	uniform := image.NewUniform(value)
	for top := bounds.Min.Y; top < bounds.Max.Y; top += transformBandRows {
		bottom := min(top+transformBandRows, bounds.Max.Y)
		band := image.Rect(bounds.Min.X, top, bounds.Max.X, bottom)
		maskPoint := band.Min.Sub(bounds.Min)
		draw.DrawMask(destination, band, uniform, image.Point{}, mask, maskPoint, draw.Over)
		if err := ctx.Err(); err != nil {
			return err
		}
	}

	return nil
}

func frameColor(frame FrameStyle) color.NRGBA {
	switch frame {
	case FrameThinDark:
		return color.NRGBA{R: 35, G: 35, B: 38, A: 255}
	case FrameThinLight, FramePolaroid:
		return color.NRGBA{R: 245, G: 243, B: 236, A: 255}
	default:
		// A backing makes transparent source pixels composable into an opaque
		// wallpaper without cropping or changing the source itself.
		return color.NRGBA{R: 245, G: 245, B: 245, A: 255}
	}
}
