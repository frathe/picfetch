package ui

import (
	"image"
	"sync"

	"fyne.io/fyne/v2"
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
	gaze    *widgets.Gaze
	pointer fyne.Position
	known   bool
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
	p.known = false
	p.renderFrame()
	p.BaseWidget.Hide()
}

func (p *tranePet) lookAt(pointer fyne.Position) {
	if !p.Visible() {
		return
	}
	p.pointer, p.known = pointer, true
	p.renderFrame()
}

func (p *tranePet) forgetPointer() {
	p.known = false
	p.renderFrame()
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
	if !p.known {
		p.gaze.Rest()
		return
	}
	// ImageFillContain centers the cell inside the portrait. Aim from
	// the face, accounting for both letterboxing and window resizing.
	portrait := p.gaze.Portrait()
	size := portrait.Size()
	scale := min(size.Width/widgets.GazeWidth, size.Height/widgets.GazeHeight)
	origin := fyne.CurrentApp().Driver().AbsolutePositionForObject(portrait)
	p.gaze.LookAt(fyne.NewPos(p.pointer.X-origin.X-size.Width/2,
		p.pointer.Y-origin.Y-(size.Height-widgets.GazeHeight*scale)/2-64*scale), 18*scale)
}
