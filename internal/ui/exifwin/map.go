package exifwin

import (
	"image"
	"net/http"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	xwidget "fyne.io/x/fyne/widget"

	"github.com/frathe/picfetch/internal/ui/mapstyle"
)

// themedMap retains Fyne-X's map controls, markers and tile cache. Only its
// raster generator receives the shared local color filter.
type themedMap struct{ xwidget.Map }

func newThemedMap(source string, client *http.Client) *themedMap {
	m := &themedMap{}
	for _, option := range []xwidget.MapOption{
		xwidget.WithOsmTiles(), xwidget.WithTileSource(source),
		xwidget.WithHTTPClient(client), xwidget.WithZoomButtons(true),
		xwidget.WithScrollButtons(false), xwidget.AtZoomLevel(mapZoom),
	} {
		option(&m.Map)
	}
	m.ExtendBaseWidget(m)
	return m
}

func (m *themedMap) CreateRenderer() fyne.WidgetRenderer {
	renderer := m.Map.CreateRenderer()
	// The pinned Fyne-X renderer exposes its map raster as a direct child of
	// its root stack, separately from controls and markers. The capture test
	// guards this integration when the upstream dependency changes.
	for _, object := range renderer.Objects() {
		stack, ok := object.(*fyne.Container)
		if !ok {
			continue
		}
		for _, child := range stack.Objects {
			if raster, ok := child.(*canvas.Raster); ok {
				generate := raster.Generator
				raster.Generator = func(width, height int) image.Image {
					return mapstyle.ForTheme(generate(width, height))
				}
				return renderer
			}
		}
	}
	fyne.LogError("Could not apply the map color theme: map raster not found", nil)
	return renderer
}
