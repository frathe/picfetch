package help

import (
	_ "embed"
	"time"

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

// ShowFinis opens or raises the companion, retaining the current view on reuse.
func (h *Help) ShowFinis() {
	if h.finis == nil {
		view, err := newFinisView()
		if err != nil {
			fyne.LogError("Could not load Finis", err)
			return
		}
		h.finis = view
		view.clue.onTapped = h.showEmptyManualSearch
	}
	h.finisWin.Show(h.app, lang.L("Finis"), fyne.NewSize(480, 420), func() fyne.CanvasObject {
		return h.finis
	}, func() { h.finis = nil })
	h.finisWin.Window().SetPadded(false)
}

// finisView fills its window so the pointer can guide his gaze from outside
// the portrait too. All state changes run on Fyne's event thread; there are
// no timers or system-wide pointer monitors to stop when the window closes.
type finisView struct {
	widget.BaseWidget
	gaze    *widgets.Gaze
	pointer fyne.Position
	inside  bool
	circles widgets.CircleGesture
	clue    *finisClue
}

var _ desktop.Hoverable = (*finisView)(nil)

func newFinisView() (*finisView, error) {
	frames, err := widgets.DecodeGazeAtlas(finisAtlas, nil)
	if err != nil {
		return nil, err
	}
	view := &finisView{gaze: widgets.NewGaze(frames, fyne.NewSize(widgets.GazeWidth, widgets.GazeHeight))}
	view.clue = newFinisClue()
	view.clue.Hide()
	view.ExtendBaseWidget(view)
	return view, nil
}

func (v *finisView) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.New(finisLayout{clue: v.clue}, v.gaze.Portrait(), v.clue))
}

func (v *finisView) MouseIn(event *desktop.MouseEvent) { v.MouseMoved(event) }

func (v *finisView) MouseMoved(event *desktop.MouseEvent) {
	if !v.Visible() {
		return
	}
	v.pointer = event.Position
	v.inside = true
	v.updateGaze()
	if !v.clue.Visible() && v.circles.Move(fyne.NewPos(
		(v.pointer.X-v.Size().Width/2)/24, (v.pointer.Y-(v.Size().Height/2-40))/24), time.Now()) {
		v.clue.Show()
	}
}

func (v *finisView) MouseOut() {
	v.inside = false
	v.circles.Reset()
	v.gaze.Rest()
}

func (v *finisView) Resize(size fyne.Size) {
	if size != v.Size() {
		v.circles.Reset()
	}
	v.BaseWidget.Resize(size)
	if v.inside {
		v.updateGaze()
	}
}

func (v *finisView) Hide() {
	v.MouseOut()
	v.BaseWidget.Hide()
}

type finisLayout struct{ clue *finisClue }

func (l finisLayout) MinSize(_ []fyne.CanvasObject) fyne.Size {
	clue := l.clue.MinSize()
	// Reserve space before reveal too: showing the clue never moves the head.
	return fyne.NewSize(max(widgets.GazeWidth, clue.Width+16), widgets.GazeHeight+2*(clue.Height+8))
}

func (l finisLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	portrait := objects[0]
	portrait.Resize(fyne.NewSize(widgets.GazeWidth, widgets.GazeHeight))
	portrait.Move(fyne.NewPos((size.Width-widgets.GazeWidth)/2, (size.Height-widgets.GazeHeight)/2))
	minimum := l.clue.MinSize()
	width := min(size.Width-16, max(l.clue.preferredWidth(), minimum.Width))
	l.clue.Resize(fyne.NewSize(width, minimum.Height))
	l.clue.Move(fyne.NewPos((size.Width-width)/2, portrait.Position().Y-l.clue.Size().Height-8))
}

func (v *finisView) updateGaze() {
	// The face is 64 logical pixels below the top of the centered portrait.
	v.gaze.LookAt(fyne.NewPos(v.pointer.X-v.Size().Width/2,
		v.pointer.Y-(v.Size().Height/2-widgets.GazeHeight/2+64)), 24)
}
