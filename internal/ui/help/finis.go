package help

import (
	_ "embed"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/ui/widgets"
)

// finisAtlas contains the 17 gaze cells of the user's original Finis atlas.
// The single row holds 16 gaze directions clockwise from up, then neutral.
// Decode only when summoned; mouse movement reuses cropped frames.
//
//go:embed finis.webp
var finisAtlas []byte

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
	gaze    *widgets.Gaze
	pointer fyne.Position
	inside  bool
}

var _ desktop.Hoverable = (*finisView)(nil)

func newFinisView() (*finisView, error) {
	frames, err := widgets.DecodeGazeAtlas(finisAtlas, nil)
	if err != nil {
		return nil, err
	}
	view := &finisView{gaze: widgets.NewGaze(frames, fyne.NewSize(widgets.GazeWidth, widgets.GazeHeight))}
	view.ExtendBaseWidget(view)
	return view, nil
}

func (v *finisView) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewCenter(v.gaze.Portrait()))
}

func (v *finisView) MouseIn(event *desktop.MouseEvent) { v.MouseMoved(event) }

func (v *finisView) MouseMoved(event *desktop.MouseEvent) {
	v.pointer = event.Position
	v.inside = true
	v.updateGaze()
}

func (v *finisView) MouseOut() {
	v.inside = false
	v.gaze.Rest()
}

func (v *finisView) Resize(size fyne.Size) {
	v.BaseWidget.Resize(size)
	if v.inside {
		v.updateGaze()
	}
}

func (v *finisView) updateGaze() {
	// The face is 64 logical pixels below the top of the centered portrait.
	v.gaze.LookAt(fyne.NewPos(v.pointer.X-v.Size().Width/2,
		v.pointer.Y-(v.Size().Height/2-widgets.GazeHeight/2+64)), 24)
}
