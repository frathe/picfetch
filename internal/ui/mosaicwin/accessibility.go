package mosaicwin

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// namedSelect supplies the accessible name Fyne's Select does not carry on
// its own. The visible label beside it uses the same localized name.
type namedSelect struct {
	*widget.Select
	name string
}

func newNamedSelect(name string, options []string, changed func(string)) *namedSelect {
	return &namedSelect{Select: widget.NewSelect(options, changed), name: name}
}

func (s *namedSelect) AccessibilityLabel() string {
	if s.Selected == "" {
		return s.name
	}
	return fmt.Sprintf("%s: %s", s.name, s.Selected)
}

func (*namedSelect) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleButton }

// namedSlider gives a standard Slider a localized, value-bearing accessible
// name while its valueLabel makes the same changing value visible on screen.
type namedSlider struct {
	*widget.Slider
	name       string
	format     func(float64) string
	valueLabel *widget.Label
}

func newNamedSlider(name string, minimum, maximum, step, value float64, format func(float64) string, changed func(float64)) *namedSlider {
	slider := widget.NewSlider(minimum, maximum)
	slider.Step = step
	slider.Value = value
	named := &namedSlider{
		Slider:     slider,
		name:       name,
		format:     format,
		valueLabel: widget.NewLabel(format(value)),
	}
	slider.OnChanged = func(next float64) {
		named.valueLabel.SetText(format(next))
		changed(next)
	}

	return named
}

func (s *namedSlider) AccessibilityLabel() string {
	return fmt.Sprintf("%s: %s", s.name, s.format(s.Value))
}

func (*namedSlider) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleButton }

// actionButton keeps Fyne's standard button rendering/accessibility and makes
// both common keyboard activation keys equivalent.
type actionButton struct {
	*widget.Button
}

func newActionButton(label string, tapped func()) *actionButton {
	return &actionButton{Button: widget.NewButton(label, tapped)}
}

func (b *actionButton) TypedKey(event *fyne.KeyEvent) {
	switch event.Name {
	case fyne.KeyEnter, fyne.KeyReturn, fyne.KeySpace:
		b.Tapped(nil)
	}
}
