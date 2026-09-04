package mosaic

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"math"

	xdraw "golang.org/x/image/draw"
)

func renderPlacement(ctx context.Context, destination *image.NRGBA, source *loadedSource, placement placement) error {
	scaled, err := scaleSource(source, placement)
	if err != nil {
		return err
	}
	bounds := placementPixelBounds(placement).Intersect(destination.Bounds())
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			localX, localY := inverseRotate(placement, float64(x)+0.5, float64(y)+0.5)
			paintCardPixel(destination, scaled, placement, x, y, localX, localY)
		}
	}

	return nil
}

func scaleSource(source *loadedSource, placement placement) (image.Image, error) {
	width := max(1, int(math.Ceil(placement.imageRect.width)))
	height := max(1, int(math.Ceil(placement.imageRect.height)))
	if source.vector != nil {
		return source.vector.RasterAt(width, height)
	}
	if source.pixels == nil {
		return nil, fmt.Errorf("decoded image has no pixels")
	}

	scaled := image.NewNRGBA(image.Rect(0, 0, width, height))
	xdraw.CatmullRom.Scale(scaled, scaled.Bounds(), source.pixels, source.pixels.Bounds(), xdraw.Src, nil)

	return scaled, nil
}

func paintCardPixel(
	destination *image.NRGBA,
	source image.Image,
	placement placement,
	x, y int,
	localX, localY float64,
) {
	halfBodyWidth := placement.bodyWidth / 2
	shadow := placement.shadowSize
	inShadow := localX >= -halfBodyWidth+shadow && localX <= halfBodyWidth+shadow &&
		localY >= placement.bodyTop+shadow && localY <= placement.bodyBottom+shadow
	if inShadow {
		composite(destination, x, y, color.NRGBA{A: 105})
	}

	inBody := localX >= -halfBodyWidth && localX <= halfBodyWidth &&
		localY >= placement.bodyTop && localY <= placement.bodyBottom
	if !inBody {
		return
	}

	backing := frameColor(placement.frame)
	destination.SetNRGBA(x, y, backing)
	halfImageWidth := placement.imageRect.width / 2
	halfImageHeight := placement.imageRect.height / 2
	if localX < -halfImageWidth || localX > halfImageWidth ||
		localY < -halfImageHeight || localY > halfImageHeight {
		return
	}

	u := (localX + halfImageWidth) / placement.imageRect.width
	v := (localY + halfImageHeight) / placement.imageRect.height
	sx := min(source.Bounds().Max.X-1, source.Bounds().Min.X+int(u*float64(source.Bounds().Dx())))
	sy := min(source.Bounds().Max.Y-1, source.Bounds().Min.Y+int(v*float64(source.Bounds().Dy())))
	pixel := color.NRGBAModel.Convert(source.At(sx, sy)).(color.NRGBA)
	composite(destination, x, y, pixel)
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

func composite(destination *image.NRGBA, x, y int, source color.NRGBA) {
	if source.A == 255 {
		destination.SetNRGBA(x, y, source)
		return
	}
	if source.A == 0 {
		return
	}
	dest := destination.NRGBAAt(x, y)
	alpha := uint32(source.A)
	inverse := uint32(255 - source.A)
	destination.SetNRGBA(x, y, color.NRGBA{
		R: uint8((uint32(source.R)*alpha + uint32(dest.R)*inverse + 127) / 255),
		G: uint8((uint32(source.G)*alpha + uint32(dest.G)*inverse + 127) / 255),
		B: uint8((uint32(source.B)*alpha + uint32(dest.B)*inverse + 127) / 255),
		A: 255,
	})
}
