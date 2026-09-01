package compare

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"sort"
	"strconv"

	"fyne.io/fyne/v2"
	xdraw "golang.org/x/image/draw"
)

const (
	detailSamplerCount   = 7
	tileTextureDimension = 1024
	tileGutter           = 1
	tileInterior         = tileTextureDimension - 2*tileGutter
	tileCacheBudgetBytes = int64(64 << 20)
)

type tileKey struct {
	level int
	x     int
	y     int
}

func (k tileKey) cacheKey() string {
	return strconv.Itoa(k.level) + "/" + strconv.Itoa(k.x) + "/" + strconv.Itoa(k.y)
}

type tileRequest struct {
	key     tileKey
	visible bool
}

type sourceRect struct {
	minX float64
	minY float64
	maxX float64
	maxY float64
}

type tilePlan struct {
	level    int
	visible  sourceRect
	requests []tileRequest
}

type renderTile struct {
	key      tileKey
	texture  *image.RGBA
	interior image.Rectangle
	scale    int
}

type edgeClampedImage struct {
	image.Image
	bounds image.Rectangle
}

func (img edgeClampedImage) Bounds() image.Rectangle { return img.bounds }

func (img edgeClampedImage) At(x, y int) color.Color {
	bounds := img.Image.Bounds()
	x = min(max(x, bounds.Min.X), bounds.Max.X-1)
	y = min(max(y, bounds.Min.Y), bounds.Max.Y-1)
	return img.Image.At(x, y)
}

func planTiles(scene paneScene) tilePlan {
	visible, ok := visibleSource(scene)
	if !ok {
		return tilePlan{}
	}
	if overviewCoversDisplay(scene) {
		return tilePlan{visible: visible}
	}
	bounds := scene.source.frame.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	maximumLevel := maxMipLevel(width, height)
	level := desiredMipLevel(width, height, scene.displaySize, maximumLevel)

	var minX, maxX, minY, maxY int
	for {
		minX, maxX, minY, maxY = visibleTileRange(visible, width, height, level)
		count := int64(maxX-minX+1) * int64(maxY-minY+1)
		if count <= detailSamplerCount || level >= maximumLevel {
			break
		}
		level++
	}

	plan := tilePlan{level: level, visible: visible}
	centerX := (visible.minX + visible.maxX) / 2
	centerY := (visible.minY + visible.maxY) / 2
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			plan.requests = append(plan.requests, tileRequest{
				key:     tileKey{level: level, x: x, y: y},
				visible: true,
			})
		}
	}
	sortTileRequests(plan.requests, centerX, centerY)

	columns, rows := tileGrid(width, height, level)
	seen := make(map[tileKey]bool, detailSamplerCount)
	for _, request := range plan.requests {
		seen[request.key] = true
	}
	for radius := 1; len(plan.requests) < detailSamplerCount && radius <= max(columns, rows); radius++ {
		left := max(0, minX-radius)
		right := min(columns-1, maxX+radius)
		top := max(0, minY-radius)
		bottom := min(rows-1, maxY+radius)
		candidates := make([]tileRequest, 0, 2*(right-left+bottom-top+2))
		for y := top; y <= bottom; y++ {
			for x := left; x <= right; x++ {
				if x > left && x < right && y > top && y < bottom {
					continue
				}
				key := tileKey{level: level, x: x, y: y}
				if seen[key] {
					continue
				}
				seen[key] = true
				candidates = append(candidates, tileRequest{key: key})
			}
		}
		sortTileRequests(candidates, centerX, centerY)
		remaining := detailSamplerCount - len(plan.requests)
		if len(candidates) > remaining {
			candidates = candidates[:remaining]
		}
		plan.requests = append(plan.requests, candidates...)
	}
	return plan
}

func overviewCoversDisplay(scene paneScene) bool {
	if scene.source == nil || scene.source.frame == nil || scene.source.overview == nil {
		return false
	}
	frame := scene.source.frame.Bounds().Size()
	overview := scene.source.overview.Bounds().Size()
	if overview.X >= frame.X && overview.Y >= frame.Y {
		return true
	}
	return scene.displaySize.X <= overview.X && scene.displaySize.Y <= overview.Y
}

func visibleSource(scene paneScene) (sourceRect, bool) {
	if scene.source == nil || scene.source.frame == nil ||
		scene.viewport.Width <= 0 || scene.viewport.Height <= 0 ||
		scene.imageSize.Width <= 0 || scene.imageSize.Height <= 0 {
		return sourceRect{}, false
	}
	bounds := scene.source.frame.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 {
		return sourceRect{}, false
	}
	revealPosition := fyne.Position{}
	revealSize := scene.viewport
	if scene.revealSet {
		revealPosition = scene.revealPosition
		revealSize = scene.revealSize
	}
	left := max(float64(revealPosition.X), float64(scene.imagePosition.X))
	top := max(float64(revealPosition.Y), float64(scene.imagePosition.Y))
	right := min(float64(revealPosition.X+revealSize.Width), float64(scene.imagePosition.X+scene.imageSize.Width))
	bottom := min(float64(revealPosition.Y+revealSize.Height), float64(scene.imagePosition.Y+scene.imageSize.Height))
	left = max(left, 0)
	top = max(top, 0)
	right = min(right, float64(scene.viewport.Width))
	bottom = min(bottom, float64(scene.viewport.Height))
	if right <= left || bottom <= top {
		return sourceRect{}, false
	}
	positionX, positionY := float64(scene.imagePosition.X), float64(scene.imagePosition.Y)
	displayWidth, displayHeight := float64(scene.imageSize.Width), float64(scene.imageSize.Height)
	visible := sourceRect{
		minX: (left - positionX) / displayWidth * float64(width),
		minY: (top - positionY) / displayHeight * float64(height),
		maxX: (right - positionX) / displayWidth * float64(width),
		maxY: (bottom - positionY) / displayHeight * float64(height),
	}
	visible.minX = min(max(visible.minX, 0), float64(width))
	visible.minY = min(max(visible.minY, 0), float64(height))
	visible.maxX = min(max(visible.maxX, 0), float64(width))
	visible.maxY = min(max(visible.maxY, 0), float64(height))
	return visible, visible.maxX > visible.minX && visible.maxY > visible.minY
}

func desiredMipLevel(width, height int, display image.Point, maximum int) int {
	if display.X <= 0 || display.Y <= 0 {
		return 0
	}
	sourcePerPixel := max(float64(width)/float64(display.X), float64(height)/float64(display.Y))
	level := 0
	for sourcePerPixel >= 2 && level < maximum {
		sourcePerPixel /= 2
		level++
	}
	return level
}

func visibleTileRange(visible sourceRect, width, height, level int) (minX, maxX, minY, maxY int) {
	columns, rows := tileGrid(width, height, level)
	coverage := float64(tileInterior * mipScale(level))
	minX = min(max(int(math.Floor(visible.minX/coverage)), 0), columns-1)
	minY = min(max(int(math.Floor(visible.minY/coverage)), 0), rows-1)
	maxX = min(max(int(math.Ceil(visible.maxX/coverage))-1, minX), columns-1)
	maxY = min(max(int(math.Ceil(visible.maxY/coverage))-1, minY), rows-1)
	return minX, maxX, minY, maxY
}

func sortTileRequests(requests []tileRequest, centerX, centerY float64) {
	sort.SliceStable(requests, func(i, j int) bool {
		a, b := requests[i].key, requests[j].key
		coverage := float64(tileInterior * mipScale(a.level))
		aX, aY := (float64(a.x)+0.5)*coverage, (float64(a.y)+0.5)*coverage
		bX, bY := (float64(b.x)+0.5)*coverage, (float64(b.y)+0.5)*coverage
		aDistance := (aX-centerX)*(aX-centerX) + (aY-centerY)*(aY-centerY)
		bDistance := (bX-centerX)*(bX-centerX) + (bY-centerY)*(bY-centerY)
		if aDistance != bDistance {
			return aDistance < bDistance
		}
		if a.y != b.y {
			return a.y < b.y
		}
		return a.x < b.x
	})
}

func maxMipLevel(width, height int) int {
	dimension := max(width, height)
	level := 0
	for dimension > 1 {
		dimension = (dimension + 1) / 2
		level++
	}
	return level
}

func mipScale(level int) int {
	if level <= 0 {
		return 1
	}
	return 1 << min(level, 30)
}

func mipDimension(dimension, level int) int {
	scale := mipScale(level)
	return max(1, (dimension+scale-1)/scale)
}

func tileGrid(width, height, level int) (columns, rows int) {
	mipWidth := mipDimension(width, level)
	mipHeight := mipDimension(height, level)
	return (mipWidth + tileInterior - 1) / tileInterior,
		(mipHeight + tileInterior - 1) / tileInterior
}

func generateRenderTile(ctx context.Context, source *renderSource, key tileKey) (*renderTile, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if source == nil || source.frame == nil {
		return nil, errors.New("cannot generate a tile without a render source")
	}
	frameBounds := source.frame.Bounds()
	width, height := frameBounds.Dx(), frameBounds.Dy()
	if key.level < 0 || key.level > maxMipLevel(width, height) {
		return nil, fmt.Errorf("invalid tile level %d", key.level)
	}
	columns, rows := tileGrid(width, height, key.level)
	if key.x < 0 || key.x >= columns || key.y < 0 || key.y >= rows {
		return nil, fmt.Errorf("invalid tile coordinate (%d,%d) for %dx%d grid", key.x, key.y, columns, rows)
	}

	scale := mipScale(key.level)
	mipBounds := image.Rect(0, 0, mipDimension(width, key.level), mipDimension(height, key.level))
	interior := image.Rect(
		key.x*tileInterior,
		key.y*tileInterior,
		min((key.x+1)*tileInterior, mipBounds.Max.X),
		min((key.y+1)*tileInterior, mipBounds.Max.Y),
	)
	texture := image.NewRGBA(image.Rect(0, 0, interior.Dx()+2*tileGutter, interior.Dy()+2*tileGutter))
	sample := interior.Inset(-tileGutter).Intersect(mipBounds)
	destinationMin := sample.Min.Sub(interior.Min.Sub(image.Pt(tileGutter, tileGutter)))
	destination := image.Rectangle{Min: destinationMin, Max: destinationMin.Add(sample.Size())}
	sourceRect := image.Rect(
		frameBounds.Min.X+sample.Min.X*scale,
		frameBounds.Min.Y+sample.Min.Y*scale,
		frameBounds.Min.X+sample.Max.X*scale,
		frameBounds.Min.Y+sample.Max.Y*scale,
	)
	if key.level == 0 {
		draw.Draw(texture, destination, source.frame, sourceRect.Min, draw.Src)
	} else {
		scaleSource := source.frame
		if sourceRect.Intersect(frameBounds) != sourceRect {
			scaleSource = edgeClampedImage{
				Image: source.frame,
				bounds: image.Rect(
					frameBounds.Min.X,
					frameBounds.Min.Y,
					frameBounds.Min.X+mipBounds.Dx()*scale,
					frameBounds.Min.Y+mipBounds.Dy()*scale,
				),
			}
		}
		xdraw.ApproxBiLinear.Scale(texture, destination, scaleSource, sourceRect, draw.Src, nil)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	fillMissingGutters(texture, interior, mipBounds)
	return &renderTile{key: key, texture: texture, interior: interior, scale: scale}, nil
}

func fillMissingGutters(texture *image.RGBA, interior, mipBounds image.Rectangle) {
	width, height := texture.Bounds().Dx(), texture.Bounds().Dy()
	if interior.Min.X == mipBounds.Min.X {
		for y := range height {
			texture.SetRGBA(0, y, texture.RGBAAt(tileGutter, y))
		}
	}
	if interior.Max.X == mipBounds.Max.X {
		for y := range height {
			texture.SetRGBA(width-1, y, texture.RGBAAt(width-1-tileGutter, y))
		}
	}
	if interior.Min.Y == mipBounds.Min.Y {
		for x := range width {
			texture.SetRGBA(x, 0, texture.RGBAAt(x, tileGutter))
		}
	}
	if interior.Max.Y == mipBounds.Max.Y {
		for x := range width {
			texture.SetRGBA(x, height-1, texture.RGBAAt(x, height-1-tileGutter))
		}
	}
}
