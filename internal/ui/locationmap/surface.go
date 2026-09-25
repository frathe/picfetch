package locationmap

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/fileidentity"
	"github.com/frathe/picfetch/internal/ui/widgets"
)

// Surface is the clipped geographic canvas. Its children exist only in view.
type Surface struct {
	widget.BaseWidget
	feature                 *Feature
	layer                   *fyne.Container
	tileLayer               *fyne.Container
	background              *canvas.Raster
	tooltip                 *fyne.Container
	filename                *widget.Label
	centerX, centerY, scale float64
	images                  []*canvas.Image
	previewKeys             []previewKey
	revision                uint64
	manual                  bool
}

type previewKey struct{ uri, version string }

func newSurface(feature *Feature) *Surface {
	s := &Surface{feature: feature, layer: container.NewWithoutLayout(), tileLayer: container.NewWithoutLayout(), centerX: .5, centerY: .5, scale: 256}
	s.background = canvas.NewRasterWithPixels(func(x, y, _, _ int) color.Color {
		if (x/16+y/16)%2 == 0 {
			return color.NRGBA{R: 42, G: 44, B: 47, A: 255}
		}
		return color.NRGBA{R: 52, G: 54, B: 57, A: 255}
	})
	s.filename = widget.NewLabel("")
	s.filename.Wrapping = fyne.TextWrapBreak
	s.tooltip = container.NewStack(canvas.NewRectangle(theme.Color(theme.ColorNameOverlayBackground)), s.filename)
	s.hideFilename()
	s.ExtendBaseWidget(s)
	return s
}

func (s *Surface) positions() []WorldPoint {
	points := make([]WorldPoint, len(s.feature.points))
	for i, point := range s.feature.points {
		p, ok := Project(point.Metadata.Latitude, point.Metadata.Longitude)
		if !ok {
			p = WorldPoint{X: math.NaN(), Y: math.NaN()}
		}
		points[i] = p
	}
	return points
}

func (s *Surface) fit() {
	camera := FitCamera(s.positions(), float64(s.Size().Width), float64(s.Size().Height), 60)
	s.centerX, s.centerY, s.scale = camera.X, camera.Y, camera.Scale
	s.arrange()
}

func (s *Surface) clear() {
	s.retirePreviewWork()
	for _, img := range s.images {
		img.Image = nil
		img.Refresh()
	}
	s.images = nil
	s.previewKeys = nil
	s.layer.RemoveAll()
}

func (s *Surface) retirePreviewWork() {
	s.hideFilename()
	if s.feature.previewCancel != nil {
		s.feature.previewCancel()
		s.feature.previewCancel = nil
	}
	s.revision++
}

func (s *Surface) arrange() {
	if !s.feature.active || !s.feature.Visible() {
		return
	}
	if s.feature.ctx == nil {
		return
	}
	s.retirePreviewWork()
	// Reuse only pixels already painted for this exact source version. Keep the
	// old cards mounted until their fully populated replacements are ready.
	pixels := make(map[previewKey]image.Image, len(s.images))
	for i, img := range s.images {
		pixels[s.previewKeys[i]] = img.Image
	}
	previousImages := s.images
	s.images = nil
	s.previewKeys = nil
	var objects []fyne.CanvasObject
	s.feature.updateTiles()
	var visible []Point
	for _, cluster := range Clusters(s.positions(), Camera{s.centerX, s.centerY, s.scale}, float64(s.Size().Width), float64(s.Size().Height), 108) {
		px, py := float32(cluster.X), float32(cluster.Y)
		point := s.feature.points[cluster.Members[0]]
		open := func() { s.feature.host.OpenLocationImage(point.Source.Identity) }
		photoY := py - 36
		if len(cluster.Members) > 1 {
			members := make([]fileidentity.Occurrence, len(cluster.Members))
			highlight := false
			for i, index := range cluster.Members {
				members[i] = s.feature.points[index].Source.Identity
				highlight = highlight || members[i] == s.feature.displayed
			}
			open = func() { s.feature.host.OpenLocationCluster(members) }
			pin := widget.NewButton(fmt.Sprintf(lang.L("%d images"), len(members)), open)
			if highlight {
				pin.Importance = widget.HighImportance
			}
			pin.Resize(fyne.NewSize(96, 32))
			pin.Move(fyne.NewPos(px-48, py+20))
			objects = append(objects, pin)
			photoY = py - 54
		}
		key := previewKey{point.Source.URI.String(), point.Version}
		img := canvas.NewImageFromImage(pixels[key])
		img.FillMode = canvas.ImageFillContain
		frame := canvas.NewRectangle(color.NRGBA{R: 245, G: 245, B: 245, A: 255})
		frame.StrokeColor = color.NRGBA{R: 190, G: 190, B: 190, A: 255}
		frame.StrokeWidth = 1
		frame.Shadow = canvas.Shadow{Color: color.NRGBA{A: 100}, BlurRadius: 4, Offset: fyne.NewPos(1, 2)}
		if point.Source.Identity == s.feature.displayed {
			frame.StrokeColor = theme.Color(theme.ColorNamePrimary)
			frame.StrokeWidth = 2
		}
		card := widgets.NewTappableArea(container.NewStack(frame, container.NewPadded(img)), open)
		card.Resize(fyne.NewSize(96, 72))
		card.Move(fyne.NewPos(px-48, photoY))
		card.OnHover = func(hovering bool) {
			if !hovering {
				s.hideFilename()
				return
			}
			s.filename.SetText(point.Source.URI.Name())
			s.filename.Show()
			size := fyne.MeasureText(s.filename.Text, theme.TextSize(), s.filename.TextStyle).Add(fyne.NewSquareSize(2 * theme.InnerPadding()))
			size.Width = min(size.Width, max(0, s.Size().Width-16))
			s.filename.Resize(size)
			size.Height = s.filename.MinSize().Height
			s.tooltip.Resize(size)
			y := photoY - size.Height - 4
			if y < 0 {
				y = photoY + 76
			}
			s.tooltip.Move(fyne.NewPos(max(0, min(px-size.Width/2, s.Size().Width-size.Width)), max(0, min(y, s.Size().Height-size.Height))))
			s.tooltip.Show()
		}
		objects = append(objects, card)
		s.images = append(s.images, img)
		s.previewKeys = append(s.previewKeys, key)
		visible = append(visible, point)
	}
	if len(visible) > 0 {
		s.feature.previews(visible, s.revision)
	}
	s.layer.Objects = objects
	s.layer.Refresh()
	for _, img := range previousImages {
		img.Image = nil
	}
}

func (s *Surface) hideFilename() {
	s.filename.Hide()
	s.tooltip.Hide()
}

func (s *Surface) setPreviews(images []image.Image) {
	for i, img := range images {
		s.images[i].Image = img
		s.images[i].Refresh()
	}
}

func (s *Surface) CreateRenderer() fyne.WidgetRenderer { return &surfaceRenderer{s: s} }

type surfaceRenderer struct {
	s    *Surface
	size fyne.Size
}

func (r *surfaceRenderer) Layout(size fyne.Size) {
	if r.size == size {
		return
	}
	r.size = size
	r.s.background.Resize(size)
	r.s.layer.Resize(size)
	r.s.tileLayer.Resize(size)
	r.s.arrange()
}
func (*surfaceRenderer) MinSize() fyne.Size { return fyne.NewSize(200, 140) }
func (r *surfaceRenderer) Refresh() {
	r.s.background.Refresh()
	r.s.tileLayer.Refresh()
	r.s.layer.Refresh()
}
func (r *surfaceRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.s.background, r.s.tileLayer, r.s.layer, r.s.tooltip}
}
func (*surfaceRenderer) Destroy() {}

func (s *Surface) Dragged(event *fyne.DragEvent) {
	s.manual = true
	s.centerX -= float64(event.Dragged.DX) / s.scale
	s.centerY -= float64(event.Dragged.DY) / s.scale
	s.arrange()
}
func (*Surface) DragEnd() {}
func (s *Surface) Scrolled(event *fyne.ScrollEvent) {
	s.zoom(math.Pow(1.2, float64(event.Scrolled.DY)/10), event.Position)
}
func (s *Surface) zoom(factor float64, anchor fyne.Position) {
	s.manual = true
	old := s.scale
	s.scale = math.Max(256, math.Min(256*math.Pow(2, 19), s.scale*factor))
	s.centerX += float64(anchor.X-s.Size().Width/2) * (1/old - 1/s.scale)
	s.centerY += float64(anchor.Y-s.Size().Height/2) * (1/old - 1/s.scale)
	s.arrange()
}

func (f *Feature) HandleKey(key fyne.KeyName) {
	s := f.surface
	switch key {
	case fyne.Key0:
		s.fit()
	case fyne.KeyPlus, fyne.KeyEqual:
		s.zoom(2, fyne.NewPos(s.Size().Width/2, s.Size().Height/2))
	case fyne.KeyMinus:
		s.zoom(.5, fyne.NewPos(s.Size().Width/2, s.Size().Height/2))
	case fyne.KeyLeft:
		s.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(60, 0)})
	case fyne.KeyRight:
		s.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(-60, 0)})
	case fyne.KeyUp:
		s.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(0, 60)})
	case fyne.KeyDown:
		s.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(0, -60)})
	}
}
