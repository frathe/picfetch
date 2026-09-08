package mosaic

import (
	"fmt"
	"image"
	"math"

	"golang.org/x/image/math/f64"

	"github.com/frathe/picfetch/internal/imaging"
)

// This bounds live rendering scratch, separately from the two output canvases,
// layout coverage, decoded sources and the byte-budgeted repeat cache.
const (
	maxPreparationBytes = 64 << 20
	preparationTileSize = 256
)

type preparationPlan struct {
	width, height int
	layer         image.Rectangle
	vectorSize    image.Point
	bytes         uint64
	tiled         bool
	separable     bool
}

func planPreparation(source *loadedSource, placed placement, bounds image.Rectangle) (preparationPlan, error) {
	w := math.Ceil(placed.imageRect.width * sourceRenderScale)
	h := math.Ceil(placed.imageRect.height * sourceRenderScale)
	// x/image's kernels use int32 coordinates internally. Reject dimensions
	// outside that representation before converting or allocating anything.
	if !(w > 0 && h > 0 && w < math.MaxInt32/2 && h < math.MaxInt32/2) {
		return preparationPlan{}, fmt.Errorf("mosaic preparation dimensions are out of range")
	}
	p := preparationPlan{width: int(w), height: int(h)}
	p.layer = image.Rect(0, 0, p.width+2*sourceEdgePadding, p.height+2*sourceEdgePadding)
	size := source.bounds.Size()
	if source.vector != nil {
		p.vectorSize.X, p.vectorSize.Y = imaging.ClampVectorRaster(p.width, p.height)
		size = p.vectorSize
	} else if source.pixels != nil {
		size = source.pixels.Bounds().Size()
	}
	// CatmullRom.Scale retains a [4]float64 intermediate for every prepared
	// column and source row, plus 24-byte sources and 16-byte contributions.
	// Six contributions per larger-axis pixel conservatively bound support 2.
	p.bytes = preparationBytes(4*float64(p.layer.Dx())*float64(p.layer.Dy()) +
		32*float64(p.width)*float64(size.Y) +
		24*(float64(p.width)+float64(p.height)) +
		96*(float64(max(p.width, size.X))+float64(max(p.height, size.Y))) +
		preparationOverhead(bounds, p.vectorSize, false))
	return p, nil
}

func preparationOverhead(bounds image.Rectangle, vectorSize image.Point, tiled bool) float64 {
	// Reserve all three masks, their rasterizer buffers and filter margins,
	// plus the vector RGBA and rasterizer buffers. Small object overhead gets
	// its own allowance; none of these terms depends on unbounded byte math.
	rasterHeight := float64(bounds.Dy()) + 2
	if tiled {
		rasterHeight = max(rasterHeight, 513)
	}
	return 30*(float64(bounds.Dx())+2)*rasterHeight +
		12*float64(vectorSize.X)*float64(vectorSize.Y) + 16<<10
}

func preparationBytes(bytes float64) uint64 {
	// Floating-point estimates are exact at the budget's scale and avoid
	// overflowing intermediate integer products for rejected plans.
	if !(bytes >= 0 && bytes < math.MaxUint64) {
		return math.MaxUint64
	}
	return uint64(math.Ceil(bytes))
}

func (p preparationPlan) forTile(bounds image.Rectangle, transform f64.Aff3, sourceSize image.Point, budget uint64) preparationPlan {
	p.tiled = true
	p.vectorSize = boundedVectorSize(p.vectorSize)
	if p.vectorSize.X > 0 {
		sourceSize = p.vectorSize
	}
	determinant := transform[0]*transform[4] - transform[1]*transform[3]
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, point := range []image.Point{bounds.Min, bounds.Max, {bounds.Min.X, bounds.Max.Y}, {bounds.Max.X, bounds.Min.Y}} {
		x, y := float64(point.X)-transform[2], float64(point.Y)-transform[5]
		u := (transform[4]*x - transform[1]*y) / determinant
		v := (-transform[3]*x + transform[0]*y) / determinant
		minX, minY = min(minX, u), min(minY, v)
		maxX, maxY = max(maxX, u), max(maxY, v)
	}
	// ApproxBiLinear needs neighboring prepared pixels. Keep the original
	// virtual coordinates and include enough samples for its filter support.
	p.layer = image.Rect(int(math.Floor(minX))-3, int(math.Floor(minY))-3,
		int(math.Ceil(maxX))+3, int(math.Ceil(maxY))+3).Intersect(p.layer)
	if !p.layer.Empty() {
		// A tile touching only padding still needs its nearest interior color.
		interior := p.interior()
		p.layer = p.layer.Union(image.Rect(
			max(interior.Min.X, min(p.layer.Min.X, interior.Max.X-1)),
			max(interior.Min.Y, min(p.layer.Min.Y, interior.Max.Y-1)),
			max(interior.Min.X+1, min(p.layer.Max.X, interior.Max.X)),
			max(interior.Min.Y+1, min(p.layer.Max.Y, interior.Max.Y))))
	}
	// Kernel.Transform only retains its two kernel-weight arrays; unlike
	// Scale it has no full-width/source-height intermediate or distributions.
	weights := 8 * (2 + 2*math.Ceil(2*max(1, float64(sourceSize.X)/float64(p.width))) +
		2*math.Ceil(2*max(1, float64(sourceSize.Y)/float64(p.height))))
	p.bytes = preparationBytes(4*float64(p.layer.Dx())*float64(p.layer.Dy()) + weights + preparationOverhead(bounds, p.vectorSize, true))
	separableBytes := preparationBytes(float64(p.bytes) + separableScratch(p.layer.Intersect(p.interior()).Size(), sourceSize, p.width))
	if separableBytes <= budget {
		p.separable = true
		p.bytes = separableBytes
	}
	return p
}

func (p preparationPlan) interior() image.Rectangle {
	return image.Rect(sourceEdgePadding, sourceEdgePadding, sourceEdgePadding+p.width, sourceEdgePadding+p.height)
}

func boundedVectorSize(size image.Point) image.Point {
	const pixels = maxPreparationBytes / 32
	area := float64(size.X) * float64(size.Y)
	if area <= pixels {
		return size
	}
	scale := math.Sqrt(pixels / area)
	size.X, size.Y = max(1, int(float64(size.X)*scale)), max(1, int(float64(size.Y)*scale))
	if size.X > size.Y {
		size.X = min(size.X, pixels/size.Y)
	} else {
		size.Y = min(size.Y, pixels/size.X)
	}
	return size
}
