package help

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/draw"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/image/webp"
)

// finisAtlas is the user's Finis character, supplied as a Codex v2 atlas.
// Rows 9 and 10 contain 16 gaze directions clockwise from up; (6, 0) is
// neutral. Decode only when summoned; mouse movement reuses cropped frames.
//
//go:embed finis.webp
var finisAtlas []byte

const (
	finisWidth   = 192
	finisHeight  = 208
	finisNeutral = 16
)

func (h *Help) showFinis() {
	if h.finis == nil {
		view, err := newFinisView()
		if err != nil {
			fyne.LogError("Could not load Finis", err)
			return
		}
		h.finis = view
	}
	h.finisWin.Show(h.app, lang.L("Finis"), fyne.NewSize(480, 420), func() fyne.CanvasObject {
		return h.finis
	}, func() { h.finis = nil })
}

// finisView fills its window so the pointer can guide his gaze from outside
// the portrait too. All state changes run on Fyne's event thread; there are
// no timers or system-wide pointer monitors to stop when the window closes.
type finisView struct {
	widget.BaseWidget
	frames   [17]image.Image
	portrait *canvas.Image
	pose     int
	pointer  fyne.Position
	inside   bool
}

var _ desktop.Hoverable = (*finisView)(nil)

func newFinisView() (*finisView, error) {
	atlas, err := webp.Decode(bytes.NewReader(finisAtlas))
	if err != nil {
		return nil, err
	}
	if atlas.Bounds() != image.Rect(0, 0, 8*finisWidth, 11*finisHeight) {
		return nil, fmt.Errorf("unexpected Finis atlas bounds: %v", atlas.Bounds())
	}
	view := &finisView{pose: finisNeutral}
	for index := range view.frames {
		column, row := index%8, 9+index/8
		if index == finisNeutral {
			column, row = 6, 0
		}
		frame := image.NewNRGBA(image.Rect(0, 0, finisWidth, finisHeight))
		draw.Draw(frame, frame.Bounds(), atlas, image.Pt(column*finisWidth, row*finisHeight), draw.Src)
		view.frames[index] = frame
	}
	view.portrait = canvas.NewImageFromImage(view.frames[finisNeutral])
	view.portrait.FillMode = canvas.ImageFillContain
	view.portrait.ScaleMode = canvas.ImageScaleSmooth
	view.portrait.SetMinSize(fyne.NewSize(finisWidth, finisHeight))
	view.ExtendBaseWidget(view)
	return view, nil
}

func (v *finisView) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewCenter(v.portrait))
}

func (v *finisView) MouseIn(event *desktop.MouseEvent) { v.MouseMoved(event) }

func (v *finisView) MouseMoved(event *desktop.MouseEvent) {
	v.pointer = event.Position
	v.inside = true
	v.updateGaze()
}

func (v *finisView) MouseOut() {
	v.inside = false
	v.setPose(finisNeutral)
}

func (v *finisView) Resize(size fyne.Size) {
	v.BaseWidget.Resize(size)
	if v.inside {
		v.updateGaze()
	}
}

func (v *finisView) updateGaze() {
	// The face is 64 logical pixels below the top of the centered portrait.
	dx := float64(v.pointer.X - v.Size().Width/2)
	dy := float64(v.pointer.Y - (v.Size().Height/2 - finisHeight/2 + 64))
	if math.Hypot(dx, dy) < 24 {
		v.setPose(finisNeutral)
		return
	}
	// atan2(dx, -dy) starts at up and increases clockwise. Round to the
	// nearest of 16 sectors, wrapping negative angles through the left side.
	sector := int(math.Round(math.Atan2(dx, -dy) / (math.Pi / 8)))
	v.setPose((sector + 16) % 16)
}

func (v *finisView) setPose(pose int) {
	if pose == v.pose {
		return
	}
	v.pose = pose
	v.portrait.Image = v.frames[pose]
	v.portrait.Refresh()
}
