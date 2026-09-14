package help

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// finisClue is one wrapped speech bubble belonging to an open Finis view.
// The canonical phrase stays identical to manual submission in every locale.
type finisClue struct {
	widget.BaseWidget
	text     *widget.Label
	hint     string
	onTapped func()
}

func (_ *finisClue) Cursor() desktop.Cursor { return desktop.PointerCursor }

func (c *finisClue) Tapped(_ *fyne.PointEvent) {
	if c.onTapped != nil {
		c.onTapped()
	}
}

func newFinisClue() *finisClue {
	hint := lang.L("(search for it)")
	text := widget.NewLabel(lang.L(secretPhrase) + " " + hint)
	text.Alignment = fyne.TextAlignCenter
	text.Wrapping = fyne.TextWrapWord
	clue := &finisClue{text: text, hint: hint}
	clue.ExtendBaseWidget(clue)
	return clue
}

func (c *finisClue) preferredWidth() float32 {
	th := c.Theme()
	text := fyne.MeasureText(c.text.Text, th.Size(theme.SizeNameText), c.text.TextStyle)
	return text.Width + 24 + 2*th.Size(theme.SizeNameInnerPadding)
}

func (c *finisClue) CreateRenderer() fyne.WidgetRenderer {
	background := canvas.NewRectangle(theme.Color(theme.ColorNameMenuBackground))
	background.CornerRadius = 12
	background.StrokeWidth = 1
	background.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	tail := canvas.NewArbitraryPolygon([]fyne.Position{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: .5, Y: 1}}, background.FillColor)
	tail.NormalizedPoints = true
	return &finisClueRenderer{clue: c, background: background, tail: tail}
}

type finisClueRenderer struct {
	clue       *finisClue
	background *canvas.Rectangle
	tail       *canvas.ArbitraryPolygon
}

func (r *finisClueRenderer) Layout(size fyne.Size) {
	r.background.Resize(fyne.NewSize(size.Width, size.Height-10))
	r.tail.Resize(fyne.NewSize(18, 11))
	r.tail.Move(fyne.NewPos(size.Width/2-9, size.Height-11))
	r.clue.text.Move(fyne.NewPos(12, 12))
	r.clue.text.Resize(fyne.NewSize(size.Width-24, size.Height-34))
}

func (r *finisClueRenderer) MinSize() fyne.Size {
	// Each phrase fits on one line at the minimum width, so two lines
	// always suffice. Measuring does not resize the live label during a
	// MinSize query; native layout may query it after laying out the text.
	th := r.clue.Theme()
	textSize := th.Size(theme.SizeNameText)
	phrase := fyne.MeasureText(lang.L(secretPhrase), textSize, fyne.TextStyle{})
	hint := fyne.MeasureText(r.clue.hint, textSize, fyne.TextStyle{})
	padding := 2 * th.Size(theme.SizeNameInnerPadding)
	return fyne.NewSize(max(260, max(phrase.Width, hint.Width)+24+padding),
		2*max(phrase.Height, hint.Height)+th.Size(theme.SizeNameLineSpacing)+padding+34)
}

func (r *finisClueRenderer) Refresh() {
	r.background.FillColor = theme.Color(theme.ColorNameMenuBackground)
	r.background.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	r.tail.FillColor = r.background.FillColor
	r.background.Refresh()
	r.tail.Refresh()
	r.clue.text.Refresh()
}

func (r *finisClueRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.background, r.tail, r.clue.text}
}

func (_ *finisClueRenderer) Destroy() {}
