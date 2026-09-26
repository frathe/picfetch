package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// PreparationProgress shows measured checks, then an unmeasured finishing phase.
// Hiding it also stops the waiting animation; hidden parents alone do not do so.
type PreparationProgress struct {
	widget.BaseWidget
	checks  *widget.ProgressBar
	waiting *widget.ProgressBarInfinite
}

func NewPreparationProgress() *PreparationProgress {
	p := &PreparationProgress{checks: widget.NewProgressBar(), waiting: widget.NewProgressBarInfinite()}
	p.ExtendBaseWidget(p)
	// Fyne starts the infinite animation when its renderer is first created,
	// even if the widget was hidden. Initialize it before retiring that phase.
	_ = p.waiting.MinSize()
	p.Hide()
	return p
}

func (p *PreparationProgress) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewStack(p.checks, p.waiting))
}

// Update never presents completed checks as completion of the final grouping.
func (p *PreparationProgress) Update(completed, total int) {
	if total > 0 && completed < total {
		p.waiting.Stop()
		p.waiting.Hide()
		p.checks.Max = float64(total)
		p.checks.SetValue(float64(max(0, completed)))
		p.checks.Show()
	} else {
		p.checks.Hide()
		p.waiting.Show()
	}
	p.BaseWidget.Show()
}

func (p *PreparationProgress) Hide() {
	p.waiting.Stop()
	p.waiting.Hide()
	p.checks.Hide()
	p.BaseWidget.Hide()
}
