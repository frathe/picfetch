package widgets

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"golang.org/x/image/webp"
)

const (
	GazeWidth   = 192
	GazeHeight  = 208
	gazeNeutral = 16
)

// GazeFrames contains sixteen clockwise directions starting at up, followed
// by the neutral pose. Decoded frames are immutable and may be shared.
type GazeFrames [17]image.Image

// DecodeGazeAtlas extracts the gaze rows and neutral cell of a Codex v2 atlas.
// prepare optionally adjusts each owned frame before it becomes immutable.
func DecodeGazeAtlas(data []byte, prepare func(*image.NRGBA)) (GazeFrames, error) {
	var frames GazeFrames
	atlas, err := webp.Decode(bytes.NewReader(data))
	if err != nil {
		return frames, err
	}
	if atlas.Bounds() != image.Rect(0, 0, 8*GazeWidth, 11*GazeHeight) {
		return frames, fmt.Errorf("unexpected gaze atlas bounds: %v", atlas.Bounds())
	}
	for index := range frames {
		column, row := index%8, 9+index/8
		if index == gazeNeutral {
			column, row = 6, 0
		}
		frame := image.NewNRGBA(image.Rect(0, 0, GazeWidth, GazeHeight))
		draw.Draw(frame, frame.Bounds(), atlas, image.Pt(column*GazeWidth, row*GazeHeight), draw.Src)
		if prepare != nil {
			prepare(frame)
		}
		frames[index] = frame
	}
	return frames, nil
}

// Gaze presents a character's pose on the UI thread without timers or workers.
// Its host owns layout and converts pointer coordinates to a face-relative offset.
type Gaze struct {
	portrait *canvas.Image
	frames   GazeFrames
}

func NewGaze(frames GazeFrames, minSize fyne.Size) *Gaze {
	portrait := canvas.NewImageFromImage(frames[gazeNeutral])
	portrait.FillMode = canvas.ImageFillContain
	portrait.ScaleMode = canvas.ImageScaleSmooth
	portrait.SetMinSize(minSize)
	return &Gaze{portrait: portrait, frames: frames}
}

// Portrait is the canvas image to include in the host's renderer.
func (g *Gaze) Portrait() *canvas.Image { return g.portrait }

// LookAt selects the closest direction, resting inside the face's dead zone.
// Both offset and deadZone are in the host's displayed coordinate scale.
func (g *Gaze) LookAt(offset fyne.Position, deadZone float32) {
	dx, dy := float64(offset.X), float64(offset.Y)
	pose := gazeNeutral
	if math.Hypot(dx, dy) >= float64(deadZone) {
		pose = (int(math.Round(math.Atan2(dx, -dy)/(math.Pi/8))) + 16) % 16
	}
	g.setPose(pose)
}

func (g *Gaze) Rest() { g.setPose(gazeNeutral) }

func (g *Gaze) setPose(pose int) {
	frame := g.frames[pose]
	if frame != nil && g.portrait.Image != frame {
		g.portrait.Image = frame
		g.portrait.Refresh()
	}
}
