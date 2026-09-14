package ui

import (
	"image"
	"image/color"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/ui/assets"
	"github.com/frathe/picfetch/internal/ui/widgets"
)

// Share immutable decoded cells between viewers; neither animation nor cursor
// motion decodes artwork or changes source pixels.
var traneFrames = sync.OnceValues(func() (widgets.GazeFrames, error) {
	return widgets.DecodeGazeAtlas(assets.TraneWebP, removeTraneSpill)
})

// Correct magenta spill only within five source pixels of transparency.
// The tongue, collar, and all other interior colors are outside this band.
// Copy nearby opaque colors without changing any alpha or the source WebP.
func removeTraneSpill(frame *image.NRGBA) {
	width, height := frame.Bounds().Dx(), frame.Bounds().Dy()
	pink := make([]bool, width*height)
	for index := range pink {
		pixel := frame.Pix[index*4 : index*4+4]
		pink[index] = pixel[3] > 0 && int(pixel[0]) > int(pixel[1])+35 && int(pixel[2]) > int(pixel[1])+20
	}
	for index, spill := range pink {
		if !spill {
			continue
		}
		x, y := index%width, index/width
		edge := false
		for ny := max(0, y-5); ny <= min(height-1, y+5) && !edge; ny++ {
			for nx := max(0, x-5); nx <= min(width-1, x+5); nx++ {
				if frame.Pix[(ny*width+nx)*4+3] == 0 {
					edge = true
					break
				}
			}
		}
		if !edge {
			continue
		}
		nearest, distance := -1, 12*12+1
		for ny := max(0, y-12); ny <= min(height-1, y+12); ny++ {
			for nx := max(0, x-12); nx <= min(width-1, x+12); nx++ {
				candidate := ny*width + nx
				d := (nx-x)*(nx-x) + (ny-y)*(ny-y)
				if d < distance && !pink[candidate] && frame.Pix[candidate*4+3] >= 240 {
					nearest, distance = candidate, d
				}
			}
		}
		if nearest >= 0 {
			copy(frame.Pix[index*4:index*4+3], frame.Pix[nearest*4:nearest*4+3])
		}
	}
}

// Trane has no timers or background work: only pointer and layout events
// choose a pose, on Fyne's UI thread.
type tranePet struct {
	widget.BaseWidget
	gaze      *widgets.Gaze
	pointer   fyne.Position
	known     bool
	circles   widgets.CircleGesture
	center    fyne.Position
	scale     float32
	onCircles func()
}

func newTranePet() *tranePet {
	frames, err := traneFrames()
	gaze := widgets.NewGaze(frames, fyne.NewSize(widgets.WelcomeArtSize, widgets.WelcomeArtSize))
	if err != nil {
		fyne.LogError("Could not load Trane pet", err)
		gaze.Portrait().Resource = fyne.NewStaticResource("welcome.webp", assets.WelcomeWebP)
	}
	pet := &tranePet{gaze: gaze}
	pet.ExtendBaseWidget(pet)
	return pet
}

func (p *tranePet) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(p.gaze.Portrait())
}

func (p *tranePet) Hide() {
	p.forgetPointer()
	p.BaseWidget.Hide()
}

func (p *tranePet) lookAt(pointer fyne.Position) {
	if !p.Visible() {
		return
	}
	p.pointer, p.known = pointer, true
	p.renderFrame()
	if p.scale > 0 && p.circles.Move(fyne.NewPos(
		(pointer.X-p.center.X)/(18*p.scale), (pointer.Y-p.center.Y)/(18*p.scale)), time.Now()) && p.onCircles != nil {
		p.onCircles()
	}
}

func (p *tranePet) forgetPointer() {
	p.known = false
	p.circles.Reset()
	p.renderFrame()
}

// welcomePointer covers the complete dropzone above its hoverable children.
// It owns only hover input: tap hit testing still reaches the restore link
// or the enclosing open-files area underneath it.
type welcomePointer struct {
	widget.BaseWidget
	area *widgets.TappableArea
	link *widget.Hyperlink
}

func newWelcomePointer(area *widgets.TappableArea, link *widget.Hyperlink) *welcomePointer {
	p := &welcomePointer{area: area, link: link}
	p.ExtendBaseWidget(p)
	return p
}

func (p *welcomePointer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(color.Transparent))
}

func (p *welcomePointer) MouseIn(event *desktop.MouseEvent) {
	p.area.MouseIn(event)
	p.updateLink(event)
}

func (p *welcomePointer) MouseMoved(event *desktop.MouseEvent) {
	p.area.MouseMoved(event)
	p.updateLink(event)
}

func (p *welcomePointer) MouseOut() {
	p.area.MouseOut()
	p.link.MouseOut()
}

func (p *welcomePointer) updateLink(event *desktop.MouseEvent) {
	if !p.link.Visible() {
		p.link.MouseOut()
		return
	}
	// The overlay owns movement, but the existing link still owns its
	// text hit test, underline and cursor feedback.
	local := *event
	local.Position = event.AbsolutePosition.Subtract(fyne.CurrentApp().Driver().AbsolutePositionForObject(p.link))
	p.link.MouseMoved(&local)
}

// Gaze must be recalculated after the complete layout: the border layout
// resizes Trane before moving his parent container to the right edge.
type welcomeLayout struct {
	base fyne.Layout
	pet  *tranePet
}

func (l welcomeLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return l.base.MinSize(objects)
}

func (l welcomeLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	l.base.Layout(objects, size)
	if l.pet.Visible() {
		l.pet.renderFrame()
	}
}

func (p *tranePet) renderFrame() {
	// ImageFillContain centers the cell inside the portrait. Aim from
	// the face, accounting for both letterboxing and window resizing.
	portrait := p.gaze.Portrait()
	size := portrait.Size()
	scale := min(size.Width/widgets.GazeWidth, size.Height/widgets.GazeHeight)
	origin := fyne.CurrentApp().Driver().AbsolutePositionForObject(portrait)
	center := origin.Add(fyne.NewPos(size.Width/2, (size.Height-widgets.GazeHeight*scale)/2+64*scale))
	if center != p.center || scale != p.scale {
		p.circles.Reset()
		p.center, p.scale = center, scale
	}
	if !p.known {
		p.gaze.Rest()
		return
	}
	p.gaze.LookAt(p.pointer.Subtract(center), 18*scale)
}
