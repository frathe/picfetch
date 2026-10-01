package exifwin

import (
	"context"
	"fmt"
	"image"
	"math"
	"net/url"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	xwidget "fyne.io/x/fyne/widget"
	"golang.org/x/image/draw"

	"github.com/frathe/picfetch/internal/ui/mapstyle"
)

type mapView struct {
	centerX, centerY float64
	zoom             int
	width, height    int
}

type displayedTile struct {
	pixels            image.Image
	expires, received time.Time
	noStore           bool
}

type themedMap struct {
	widget.BaseWidget
	tiles     *tileFetcher
	centerX   float64
	centerY   float64
	zoom      int
	markers   []xwidget.MapMarker
	view      mapView
	frame     map[string]displayedTile
	session   context.Context
	ui        UIQueue
	requested map[string]bool
}

func newThemedMap(tiles *tileFetcher) *themedMap {
	result := &themedMap{tiles: tiles, centerX: .5, centerY: .5, zoom: mapZoom}
	result.ExtendBaseWidget(result)
	return result
}

// bindTileDelivery preserves foreground pixels until UI consumes them, independently
// of the encoded-byte LRUs. Each claim owns one decoded tile, never response bytes.
func (viewWidget *themedMap) bindTileDelivery(queue UIQueue) {
	viewWidget.ui = queue
	viewWidget.tiles.setOnTile(func(ctx context.Context, version uint64, address string, tile displayedTile) {
		queue.Do(func() {
			if ctx.Err() != nil || ctx != viewWidget.session || version != viewWidget.tiles.captureView() || !viewWidget.requested[address] {
				return
			}
			delete(viewWidget.requested, address)
			if tile.pixels != nil {
				viewWidget.frame[address] = tile
			}
			viewWidget.Refresh()
		})
	})
}

func displayPixels(entry *cachedTile) displayedTile {
	return displayedTile{pixels: entry.pixels, expires: entry.expires, received: entry.received, noStore: entry.noStore}
}

func projectMapLocation(latitude, longitude float64) (float64, float64) {
	latitude = max(-85.05112878, min(85.05112878, latitude))
	radians := latitude * math.Pi / 180
	return (longitude + 180) / 360, (1 - math.Asinh(math.Tan(radians))/math.Pi) / 2
}

func (viewWidget *themedMap) PanToLatLon(latitude, longitude float64) {
	viewWidget.centerX, viewWidget.centerY = projectMapLocation(latitude, longitude)
	viewWidget.Refresh()
}

func (viewWidget *themedMap) PanEast() { viewWidget.pan(-tileSize, 0) }

func (viewWidget *themedMap) pan(horizontal, vertical float64) {
	scale := float64(tileSize * (int(1) << viewWidget.zoom))
	viewWidget.centerX = math.Mod(viewWidget.centerX-horizontal/scale, 1)
	if viewWidget.centerX < 0 {
		viewWidget.centerX++
	}
	viewWidget.centerY = max(0, min(1, viewWidget.centerY-vertical/scale))
	viewWidget.Refresh()
}

func (viewWidget *themedMap) Zoom(level int) {
	if level >= 0 && level <= 19 {
		viewWidget.zoom = level
		viewWidget.Refresh()
	}
}

func (viewWidget *themedMap) ZoomIn()  { viewWidget.Zoom(viewWidget.zoom + 1) }
func (viewWidget *themedMap) ZoomOut() { viewWidget.Zoom(viewWidget.zoom - 1) }
func (viewWidget *themedMap) Dragged(event *fyne.DragEvent) {
	viewWidget.pan(float64(event.Dragged.DX), float64(event.Dragged.DY))
}
func (viewWidget *themedMap) DragEnd() {}

func (viewWidget *themedMap) SetMarkers(markers []xwidget.MapMarker) {
	viewWidget.markers = append([]xwidget.MapMarker(nil), markers...)
	viewWidget.Refresh()
}

func (viewWidget *themedMap) MinSize() fyne.Size { return fyne.NewSize(64, 64) }

func (viewWidget *themedMap) Hide() {
	viewWidget.tiles.advanceView()
	viewWidget.frame = nil
	viewWidget.requested = nil
	viewWidget.BaseWidget.Hide()
}

func (viewWidget *themedMap) draw(width, height int) image.Image {
	view := mapView{viewWidget.centerX, viewWidget.centerY, viewWidget.zoom, width, height}
	session := viewWidget.tiles.session()
	if session == viewWidget.session && view != viewWidget.view {
		viewWidget.tiles.advanceView()
	}
	if view != viewWidget.view || session != viewWidget.session || session.Err() != nil {
		viewWidget.frame = nil
		viewWidget.requested = nil
		viewWidget.view = view
		viewWidget.session = session
	}
	if viewWidget.requested == nil {
		viewWidget.requested = make(map[string]bool)
	}
	pixels := image.NewNRGBA(image.Rect(0, 0, width, height))
	if width <= 0 || height <= 0 || session.Err() != nil {
		return pixels
	}
	scale := float64(width) / float64(max(1, viewWidget.Size().Width))
	span := float64(tileSize) * scale
	world := 1 << viewWidget.zoom
	left := viewWidget.centerX*float64(world) - float64(width)/span/2
	top := viewWidget.centerY*float64(world) - float64(height)/span/2
	next := make(map[string]displayedTile)
	for tileY := int(math.Floor(top)); float64(tileY) < top+float64(height)/span; tileY++ {
		if tileY < 0 || tileY >= world {
			continue
		}
		for tileX := int(math.Floor(left)); float64(tileX) < left+float64(width)/span; tileX++ {
			wrappedX := (tileX%world + world) % world
			address := fmt.Sprintf(viewWidget.tiles.template, viewWidget.zoom, wrappedX, tileY)
			shown, reused := next[address]
			if !reused {
				shown = viewWidget.frame[address]
			}
			if shown.pixels == nil || (!shown.noStore && shown.expires.After(shown.received) && !viewWidget.tiles.now().Before(shown.expires)) {
				shown = displayedTile{}
				if !viewWidget.requested[address] {
					entry, claimed := viewWidget.tiles.requestDisplayTile(address)
					if entry != nil {
						shown = displayPixels(entry)
					} else if claimed && viewWidget.ui != nil {
						viewWidget.requested[address] = true
					}
				}
			}
			if shown.pixels == nil {
				continue
			}
			next[address] = shown
			target := image.Rect(int(math.Round((float64(tileX)-left)*span)), int(math.Round((float64(tileY)-top)*span)),
				int(math.Round((float64(tileX+1)-left)*span)), int(math.Round((float64(tileY+1)-top)*span)))
			draw.NearestNeighbor.Scale(pixels, target, shown.pixels, shown.pixels.Bounds(), draw.Src, nil)
		}
	}
	viewWidget.frame = next
	return mapstyle.ForTheme(pixels)
}

func (viewWidget *themedMap) CreateRenderer() fyne.WidgetRenderer {
	license := &url.URL{Scheme: "https", Host: "www.openstreetmap.org", Path: "/copyright"}
	renderer := &mapRenderer{owner: viewWidget, raster: canvas.NewRaster(viewWidget.draw),
		credit: widget.NewHyperlink(lang.L("© OpenStreetMap contributors"), license),
		zoom:   container.NewVBox(widget.NewButtonWithIcon("", theme.ZoomInIcon(), viewWidget.ZoomIn), widget.NewButtonWithIcon("", theme.ZoomOutIcon(), viewWidget.ZoomOut)),
		marker: widget.NewButtonWithIcon("", theme.RadioButtonCheckedIcon(), nil)}
	renderer.marker.OnTapped = func() {
		if renderer.marker.Text != "" {
			renderer.marker.SetText("")
		} else if len(viewWidget.markers) > 0 {
			renderer.marker.SetText(viewWidget.markers[0].Title())
		}
		renderer.Layout(viewWidget.Size())
	}
	renderer.root = container.NewWithoutLayout(renderer.raster, renderer.marker, renderer.zoom, renderer.credit)
	return renderer
}

type mapRenderer struct {
	owner  *themedMap
	raster *canvas.Raster
	credit *widget.Hyperlink
	zoom   *fyne.Container
	marker *widget.Button
	root   *fyne.Container
}

func (renderer *mapRenderer) Layout(size fyne.Size) {
	renderer.root.Resize(size)
	renderer.raster.Resize(size)
	renderer.zoom.Resize(renderer.zoom.MinSize())
	renderer.zoom.Move(fyne.NewPos(max(0, size.Width-renderer.zoom.Size().Width), 0))
	renderer.credit.Resize(renderer.credit.MinSize())
	renderer.credit.Move(fyne.NewPos(max(0, size.Width-renderer.credit.Size().Width), max(0, size.Height-renderer.credit.Size().Height)))
	renderer.marker.Hide()
	if len(renderer.owner.markers) == 0 {
		return
	}
	marker := renderer.owner.markers[0]
	positionX, positionY := projectMapLocation(marker.Lat(), marker.Lon())
	deltaX := positionX - renderer.owner.centerX
	deltaX -= math.Round(deltaX)
	scale := float64(tileSize * (int(1) << renderer.owner.zoom))
	position := fyne.NewPos(size.Width/2+float32(deltaX*scale), size.Height/2+float32((positionY-renderer.owner.centerY)*scale))
	if position.X < 0 || position.Y < 0 || position.X > size.Width || position.Y > size.Height {
		return
	}
	renderer.marker.Resize(renderer.marker.MinSize())
	renderer.marker.Move(position.Subtract(fyne.NewPos(renderer.marker.Size().Width/2, renderer.marker.Size().Height)))
	renderer.marker.Show()
}

func (renderer *mapRenderer) MinSize() fyne.Size           { return renderer.owner.MinSize() }
func (renderer *mapRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{renderer.root} }
func (renderer *mapRenderer) Destroy() {
	renderer.owner.tiles.retireView(renderer.owner.session)
	renderer.owner.frame = nil
	renderer.owner.requested = nil
}
func (renderer *mapRenderer) Refresh() {
	renderer.Layout(renderer.owner.Size())
	renderer.raster.Refresh()
	renderer.credit.Refresh()
	renderer.zoom.Refresh()
	renderer.marker.Refresh()
}
