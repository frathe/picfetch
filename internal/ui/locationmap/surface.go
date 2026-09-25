package locationmap

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/fileidentity"
	"github.com/frathe/picfetch/internal/ui/mapstyle"
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
	selected                fileidentity.Occurrence
	cards                   []mapCard
}

type previewKey struct{ uri, version string }

type mapCard struct {
	members []fileidentity.Occurrence
	frame   *canvas.Rectangle
	pin     *widget.Button
	open    func()
}

func newSurface(feature *Feature) *Surface {
	s := &Surface{feature: feature, layer: container.NewWithoutLayout(), tileLayer: container.NewWithoutLayout(), centerX: .5, centerY: .5, scale: 256}
	s.background = canvas.NewRaster(func(width, height int) image.Image {
		colors := [2]color.NRGBA{
			color.NRGBAModel.Convert(theme.Color(theme.ColorNameBackground)).(color.NRGBA),
			color.NRGBAModel.Convert(theme.Color(theme.ColorNameInputBackground)).(color.NRGBA),
		}
		pixels := image.NewNRGBA(image.Rect(0, 0, width, height))
		for y := range height {
			for x := range width {
				pixels.SetNRGBA(x, y, colors[(x/16+y/16)%2])
			}
		}
		return pixels
	})
	s.filename = widget.NewLabel("")
	s.filename.Wrapping = fyne.TextWrapBreak
	s.tooltip = container.NewStack(widgets.NewThemedRectangle(theme.ColorNameOverlayBackground), s.filename)
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
	s.cards = nil
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
	s.cards = nil
	var objects []fyne.CanvasObject
	s.feature.updateTiles()
	var visible []Point
	for _, cluster := range Clusters(s.positions(), Camera{s.centerX, s.centerY, s.scale}, float64(s.Size().Width), float64(s.Size().Height), 108) {
		px, py := float32(cluster.X), float32(cluster.Y)
		point := s.feature.points[cluster.Members[0]]
		members := make([]fileidentity.Occurrence, len(cluster.Members))
		for i, index := range cluster.Members {
			members[i] = s.feature.points[index].Source.Identity
		}
		open := func() {
			s.selected = point.Source.Identity
			s.feature.host.OpenLocationImage(point.Source.Identity)
		}
		photoY := py - 36
		var pin *widget.Button
		if len(cluster.Members) > 1 {
			open = func() {
				if !slices.Contains(members, s.selected) {
					s.selected = point.Source.Identity
				}
				s.feature.host.OpenLocationCluster(members)
			}
			pin = widget.NewButton(fmt.Sprintf(lang.L("%d images"), len(members)), open)
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
		s.cards = append(s.cards, mapCard{members: members, frame: frame, pin: pin, open: open})
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
	s.refreshSelection()
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
	r.s.refreshSelection()
	for _, object := range r.s.tileLayer.Objects {
		if img, ok := object.(*canvas.Image); ok {
			img.Image = mapstyle.ForTheme(img.Image)
		}
	}
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

func (f *Feature) HandleKey(key fyne.KeyName, modifiers fyne.KeyModifier) {
	s := f.surface
	switch key {
	case fyne.Key0:
		s.fit()
	case fyne.KeyPlus, fyne.KeyEqual:
		s.zoom(2, fyne.NewPos(s.Size().Width/2, s.Size().Height/2))
	case fyne.KeyMinus:
		s.zoom(.5, fyne.NewPos(s.Size().Width/2, s.Size().Height/2))
	case fyne.KeyLeft:
		s.moveDirection(-1, 0, modifiers)
	case fyne.KeyRight:
		s.moveDirection(1, 0, modifiers)
	case fyne.KeyUp:
		s.moveDirection(0, -1, modifiers)
	case fyne.KeyDown:
		s.moveDirection(0, 1, modifiers)
	case fyne.KeyReturn, fyne.KeyEnter:
		for _, card := range s.cards {
			if slices.Contains(card.members, s.selected) {
				card.open()
				return
			}
		}
	}
}

func (s *Surface) refreshSelection() {
	for _, card := range s.cards {
		selected := slices.Contains(card.members, s.selected)
		displayed := s.selected == (fileidentity.Occurrence{}) && slices.Contains(card.members, s.feature.displayed)
		card.frame.StrokeWidth = 1
		card.frame.StrokeColor = color.NRGBA{R: 190, G: 190, B: 190, A: 255}
		if selected || displayed {
			card.frame.StrokeColor = theme.Color(theme.ColorNamePrimary)
			card.frame.StrokeWidth = 2
			if selected {
				card.frame.StrokeWidth = 3
			}
		}
		card.frame.Refresh()
		if card.pin != nil {
			card.pin.Importance = widget.MediumImportance
			if selected || displayed {
				card.pin.Importance = widget.HighImportance
			}
			card.pin.Refresh()
		}
	}
}

func (s *Surface) moveDirection(dx, dy int, modifiers fyne.KeyModifier) {
	if modifiers == fyne.KeyModifierShift {
		s.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(float32(-60*dx), float32(-60*dy))})
		return
	}
	width, height := float64(s.Size().Width), float64(s.Size().Height)
	clusters := NavigationClusters(s.positions(), Camera{s.centerX, s.centerY, s.scale}, width, height, 108)
	current := -1
	for i, cluster := range clusters {
		for _, member := range cluster.Members {
			if s.feature.points[member].Source.Identity == s.selected {
				current = i
				break
			}
		}
	}
	next := NearestCluster(clusters, current, width, height, dx, dy)
	if next < 0 {
		return
	}
	target := clusters[next]
	s.selected = s.feature.points[target.Members[0]].Source.Identity
	s.manual = true
	// Expose the entire card and count without changing zoom or unnecessarily
	// moving a target that is already comfortably inside the viewport.
	marginX, marginY := math.Min(56, width/2), math.Min(62, height/2)
	moveX := target.X - math.Max(marginX, math.Min(target.X, width-marginX))
	moveY := target.Y - math.Max(marginY, math.Min(target.Y, height-marginY))
	if moveX != 0 || moveY != 0 {
		s.centerX += moveX / s.scale
		s.centerY += moveY / s.scale
		s.arrange()
	} else {
		s.refreshSelection()
	}
}
