package settingswin

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/heic"
)

// SetHEICCheckAction connects the app-owned support check before opening.
func (w *Window) SetHEICCheckAction(check func()) {
	w.heicCheckAction = check
	w.SetHEICStatus(w.heicStatus)
}

// SetHEICStatus receives an immutable app snapshot on UI. Closed windows retain
// the value for their next opening without retaining or changing stale widgets.
func (w *Window) SetHEICStatus(state heic.State) {
	w.heicStatus = state
	if w.heicStatusLabel == nil || w.heicCheckButton == nil {
		return
	}
	text := lang.L("HEIC support: not checked")
	switch {
	case state.Checking:
		text = lang.L("HEIC support: checking")
	case state.Err != nil && state.Available:
		text = lang.L("HEIC check failed; previous support remains available")
	case state.Err != nil:
		text = lang.L("HEIC support check failed; try checking again")
	case state.Known && state.Available:
		text = lang.L("HEIC support: available")
	case state.Known:
		text = lang.L("HEIC support: unavailable")
	}
	w.heicStatusLabel.SetText(text)
	if state.Checking || w.heicCheckAction == nil {
		w.heicCheckButton.Disable()
	} else {
		w.heicCheckButton.Enable()
	}
}

func (w *Window) buildHEICSupport() fyne.CanvasObject {
	status := widget.NewLabel("")
	status.Wrapping = fyne.TextWrapWord
	button := widget.NewButton(lang.L("Check HEIC support"), nil)
	w.heicStatusLabel, w.heicCheckButton = status, button
	button.OnTapped = func() {
		if w.heicCheckButton == button && !button.Disabled() && w.heicCheckAction != nil {
			w.heicCheckAction()
		}
	}
	w.SetHEICStatus(w.heicStatus)
	return container.NewVBox(status, button)
}
