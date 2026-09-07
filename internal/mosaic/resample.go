package mosaic

import (
	"context"
	"image"
	"image/color"
	"math"

	xdraw "golang.org/x/image/draw"
)

// A band retains premultiplied floating-point values between the two filter
// passes. Quantizing or clamping that intermediate would change ringing and
// translucent pixels. Coordinates and normalization match CatmullRom.Scale.
// Explicit float64 products below preserve its rounding by preventing fused
// multiply-add, even though their operands already produce float64 values.
type sampleSpan struct {
	first, last int
	center      float64
	inverse     float64
}

func catmullSpan(position, destinationSize, sourceSize int) sampleSpan {
	scale := float64(sourceSize) / float64(destinationSize)
	center := (float64(position)+0.5)*scale - 0.5
	support := 2 * max(1, scale)
	s := sampleSpan{
		first:  max(0, int(math.Floor(center-support))),
		last:   min(sourceSize, int(math.Ceil(center+support))),
		center: center,
	}
	total := 0.0
	for i := s.first; i < s.last; i++ {
		total += catmullWeight(center, i, scale)
	}
	s.inverse = 1 / total
	return s
}

func catmullWeight(center float64, source int, scale float64) float64 {
	distance := math.Abs((center - float64(source)) * min(1, 1/scale))
	if distance >= 2 {
		return 0
	}
	return xdraw.CatmullRom.At(distance)
}

func separableScratch(size, source image.Point, width int) float64 {
	columns := float64(size.X)
	rows := float64(min(size.Y, transformBandRows))
	contributions := min(float64(source.X), math.Ceil(4*max(1, float64(source.X)/float64(width)))+2)
	// Band accumulators, one horizontal row, spans and horizontal weights.
	return 32*columns*(rows+1) + 32*(columns+rows) + 8*columns*contributions + 8*float64(source.X)
}

// scaleSourceRegion filters only the visible part of a virtual prepared image.
// Each source row is horizontally filtered once per band, then reused for all
// destination rows it contributes to. This avoids a full source-height scratch
// image and the repeated two-dimensional sampling of Kernel.Transform.
func scaleSourceRegion(ctx context.Context, destination *image.NRGBA, interior image.Rectangle, source image.Image) error {
	visible := interior.Intersect(destination.Bounds())
	if visible.Empty() {
		return nil
	}
	sourceBounds := source.Bounds()
	width := visible.Dx()
	xScale := float64(sourceBounds.Dx()) / float64(interior.Dx())
	yScale := float64(sourceBounds.Dy()) / float64(interior.Dy())
	columns := make([]sampleSpan, width)
	weightCount := 0
	for x := range columns {
		columns[x] = catmullSpan(visible.Min.X-interior.Min.X+x, interior.Dx(), sourceBounds.Dx())
		weightCount += columns[x].last - columns[x].first
	}
	weights := make([]float64, 0, weightCount)
	for _, column := range columns {
		for sx := column.first; sx < column.last; sx++ {
			weights = append(weights, catmullWeight(column.center, sx, xScale))
		}
	}
	read := sourceRGBA64(source)
	firstColumn := columns[0].first
	sourceRow := make([]color.RGBA64, columns[width-1].last-firstColumn)
	row := make([][4]float64, width)
	accumulated := make([][4]float64, width*min(visible.Dy(), transformBandRows))
	rows := make([]sampleSpan, min(visible.Dy(), transformBandRows))
	for top := visible.Min.Y; top < visible.Max.Y; top += transformBandRows {
		height := min(transformBandRows, visible.Max.Y-top)
		clear(accumulated)
		for y := range height {
			rows[y] = catmullSpan(top-interior.Min.Y+y, interior.Dy(), sourceBounds.Dy())
		}
		firstRow := 0
		for sy := rows[0].first; sy < rows[height-1].last; sy++ {
			if err := ctx.Err(); err != nil {
				return err
			}
			for x := range sourceRow {
				sourceRow[x] = read(sourceBounds.Min.X+firstColumn+x, sourceBounds.Min.Y+sy)
			}
			filterSourceRow(row, columns, weights, sourceRow, firstColumn)
			for firstRow < height && rows[firstRow].last <= sy {
				firstRow++
			}
			for y := firstRow; y < height && rows[y].first <= sy; y++ {
				weight := catmullWeight(rows[y].center, sy, yScale)
				if weight == 0 {
					continue
				}
				output := accumulated[y*width : (y+1)*width]
				for x, sample := range row {
					for channel := range 4 {
						output[x][channel] += float64(sample[channel] * weight)
					}
				}
			}
		}
		for y := range height {
			for x, sample := range accumulated[y*width : (y+1)*width] {
				blendFiltered(destination, visible.Min.X+x, top+y, sample, rows[y].inverse)
			}
		}
	}
	return ctx.Err()
}

func sourceRGBA64(source image.Image) func(int, int) color.RGBA64 {
	if source, ok := source.(image.RGBA64Image); ok {
		return source.RGBA64At
	}
	return func(x, y int) color.RGBA64 {
		r, g, b, a := source.At(x, y).RGBA()
		return color.RGBA64{R: uint16(r), G: uint16(g), B: uint16(b), A: uint16(a)}
	}
}

func filterSourceRow(row [][4]float64, columns []sampleSpan, weights []float64, source []color.RGBA64, first int) {
	offset := 0
	for x, column := range columns {
		var sum [4]float64
		for _, sample := range source[column.first-first : column.last-first] {
			weight := weights[offset]
			offset++
			if weight == 0 {
				continue
			}
			sum[0] += float64(float64(sample.R) * weight)
			sum[1] += float64(float64(sample.G) * weight)
			sum[2] += float64(float64(sample.B) * weight)
			sum[3] += float64(float64(sample.A) * weight)
		}
		inverse := column.inverse / 0xffff
		for channel := range 4 {
			row[x][channel] = sum[channel] * inverse
		}
	}
}

func blendFiltered(destination *image.NRGBA, x, y int, sample [4]float64, inverse float64) {
	alpha := filterUint16(sample[3] * inverse)
	red := filterUint16(min(sample[0], sample[3]) * inverse)
	green := filterUint16(min(sample[1], sample[3]) * inverse)
	blue := filterUint16(min(sample[2], sample[3]) * inverse)
	previous := destination.RGBA64At(x, y)
	remainder := uint32(0xffff - alpha)
	destination.SetRGBA64(x, y, color.RGBA64{
		R: uint16(uint32(previous.R)*remainder/0xffff + uint32(red)),
		G: uint16(uint32(previous.G)*remainder/0xffff + uint32(green)),
		B: uint16(uint32(previous.B)*remainder/0xffff + uint32(blue)),
		A: uint16(uint32(previous.A)*remainder/0xffff + uint32(alpha)),
	})
}

func filterUint16(value float64) uint16 {
	return uint16(max(0, min(0xffff, int32(float64(0xffff*value)+0.5))))
}
