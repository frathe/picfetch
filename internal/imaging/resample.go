package imaging

import (
	"image"
	"image/color"
	"math"

	"golang.org/x/image/draw"
)

type jpegContribution struct {
	coordinate int
	weight     float64
}

type jpegSpan struct {
	contributions []jpegContribution
	inverse       float64
}

// jpegSpans preserves CatmullRom.Scale's support, normalization and contribution
// order. Explicit float64 products prevent fused multiply-add from changing its
// rounding. Color conversion retains all sixteen bits until final output.
func jpegSpans(destination, source int) []jpegSpan {
	scale := float64(source) / float64(destination)
	support, argumentScale := 2*max(1, scale), min(1, 1/scale)
	spans := make([]jpegSpan, destination)
	weights := make([]jpegContribution, 0, destination*min(source, int(math.Ceil(2*support))+1))
	for x := range spans {
		center := float64((float64(x)+0.5)*scale) - 0.5
		first := max(0, int(math.Floor(center-support)))
		last := min(source, int(math.Ceil(center+support)))
		start, total := len(weights), 0.0
		for coordinate := first; coordinate < last; coordinate++ {
			distance := math.Abs((center - float64(coordinate)) * argumentScale)
			if distance >= 2 {
				continue
			}
			weight := draw.CatmullRom.At(distance)
			if weight == 0 {
				continue
			}
			weights = append(weights, jpegContribution{coordinate, weight})
			total += weight
		}
		spans[x] = jpegSpan{contributions: weights[start:len(weights)], inverse: 1 / total}
	}
	return spans
}

// scaleJPEG filters each source row once, converting each YCbCr sample once
// instead of once per overlapping filter tap. A rolling buffer retains only
// the horizontal rows needed by one vertical support, rather than an entire
// source-height floating-point image. The caller supplies a fitted downscale.
func scaleJPEG(destination *image.RGBA, source *image.YCbCr) {
	bounds := source.Bounds()
	width, height := destination.Bounds().Dx(), destination.Bounds().Dy()
	columns := jpegSpans(width, bounds.Dx())
	rows := jpegSpans(height, bounds.Dy())
	retained := 1
	for _, row := range rows {
		c := row.contributions
		retained = max(retained, c[len(c)-1].coordinate-c[0].coordinate+1)
	}
	converted := make([]color.RGBA64, bounds.Dx())
	filtered := make([][3]float64, retained*width)
	identities := make([]int, retained)
	for i := range identities {
		identities[i] = -1
	}
	sums := make([][3]float64, width)
	for y, row := range rows {
		clear(sums)
		alpha := 0.0
		for _, vertical := range row.contributions {
			sy := vertical.coordinate
			horizontal := filtered[(sy%retained)*width : (sy%retained+1)*width]
			if identities[sy%retained] != sy {
				for x := range converted {
					converted[x] = source.RGBA64At(bounds.Min.X+x, bounds.Min.Y+sy)
				}
				filterJPEGRow(horizontal, converted, columns)
				identities[sy%retained] = sy
			}
			for x, sample := range horizontal {
				sums[x][0] += float64(sample[0] * vertical.weight)
				sums[x][1] += float64(sample[1] * vertical.weight)
				sums[x][2] += float64(sample[2] * vertical.weight)
			}
			alpha += vertical.weight
		}
		for x, sum := range sums {
			d := y*destination.Stride + x*4
			for channel := range 3 {
				value := min(sum[channel], alpha) * row.inverse
				destination.Pix[d+channel] = uint8(max(0, min(0xffff, int32(float64(0xffff*value)+0.5))) >> 8)
			}
			destination.Pix[d+3] = 255
		}
	}
}

func filterJPEGRow(destination [][3]float64, source []color.RGBA64, columns []jpegSpan) {
	for x, column := range columns {
		var red, green, blue float64
		for _, contribution := range column.contributions {
			pixel := source[contribution.coordinate]
			red += float64(float64(pixel.R) * contribution.weight)
			green += float64(float64(pixel.G) * contribution.weight)
			blue += float64(float64(pixel.B) * contribution.weight)
		}
		inverse := column.inverse / 0xffff
		destination[x] = [3]float64{red * inverse, green * inverse, blue * inverse}
	}
}
