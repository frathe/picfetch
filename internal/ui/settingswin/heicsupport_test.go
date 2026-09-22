package settingswin

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
)

func TestHEICSettings(t *testing.T) {
	w := showSettings(t, &fakeHost{})
	general := settingsTabs(t, w).Items[0].Content
	var check *widget.Button
	var status *widget.Label
	var visit func(fyne.CanvasObject)
	visit = func(object fyne.CanvasObject) {
		switch object := object.(type) {
		case *container.Scroll:
			visit(object.Content)
		case *fyne.Container:
			for _, child := range object.Objects {
				visit(child)
			}
		case *widget.Button:
			if object.Text == lang.L("Check HEIC support") {
				check = object
			}
		case *widget.Label:
			if object.Text == lang.L("HEIC support: not checked") {
				status = object
			}
		}
	}
	visit(general)
	if check == nil || status == nil {
		t.Fatal("General Settings must contain the HEIC check action and status")
	}
	if !check.Disabled() {
		t.Fatal("an unconnected check action must be disabled")
	}
}
