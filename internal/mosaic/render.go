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
	if err := ctx.Err(); err != nil {
		return err
	}
	bounds := placementPixelBounds(placement).Intersect(destination.Bounds())
	if bounds.Empty() {
		return nil
	}

	halfBodyWidth := placement.bodyWidth / 2
	if placement.shadowSize > 0 {
		shadow := placement.shadowSize
		mask := rotatedRectangleMask(
			bounds,
			placement,
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
			-halfBodyWidth,
			placement.bodyTop,
			halfBodyWidth,
			placement.bodyBottom,
		)
		if err := drawUniformThroughMask(ctx, destination, bounds, mask, backing); err != nil {
			return err
		}
	}

	prepared, transform, err := prepareSourceLayer(source, placement, backing)
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

func prepareSourceLayer(source *loadedSource, placement placement, backing color.NRGBA) (*image.NRGBA, f64.Aff3, error) {
	width := max(1, int(math.Ceil(placement.imageRect.width*sourceRenderScale)))
	height := max(1, int(math.Ceil(placement.imageRect.height*sourceRenderScale)))
	interior := image.Rect(
		sourceEdgePadding,
		sourceEdgePadding,
		sourceEdgePadding+width,
		sourceEdgePadding+height,
	)
	layer := image.NewNRGBA(image.Rect(
		0,
		0,
		width+sourceEdgePadding*2,
		height+sourceEdgePadding*2,
	))
	draw.Draw(layer, layer.Bounds(), image.NewUniform(backing), image.Point{}, draw.Src)

	sourceImage, err := sourceImageAt(source, width, height)
	if err != nil {
		return nil, f64.Aff3{}, err
	}
	xdraw.CatmullRom.Scale(layer, interior, sourceImage, sourceImage.Bounds(), xdraw.Over, nil)
	// The destination-space mask owns the silhouette. Extending the nearest
	// prepared color prevents affine samples just outside the logical photo
	// from introducing transparent black before that mask is applied.
	extendSourceEdges(layer, interior)

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

	return layer, transform, nil
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
	left, top, right, bottom float64,
) *image.Alpha {
	mask := image.NewAlpha(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	rasterizer := vector.NewRasterizer(bounds.Dx(), bounds.Dy())
	radians := placement.angle * math.Pi / 180
	sin, cos := math.Sincos(radians)
	toMask := func(x, y float64) (float32, float32) {
		rotatedX := placement.centerX + x*cos - y*sin - float64(bounds.Min.X)
		rotatedY := placement.centerY + x*sin + y*cos - float64(bounds.Min.Y)
		return float32(rotatedX), float32(rotatedY)
	}

	x, y := toMask(left, top)
	rasterizer.MoveTo(x, y)
	x, y = toMask(right, top)
	rasterizer.LineTo(x, y)
	x, y = toMask(right, bottom)
	rasterizer.LineTo(x, y)
	x, y = toMask(left, bottom)
	rasterizer.LineTo(x, y)
	rasterizer.ClosePath()
	rasterizer.Draw(mask, mask.Bounds(), image.Opaque, image.Point{})
	if placement.angle != 0 {
		softenCoverageMask(mask)
	}

	return mask
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
