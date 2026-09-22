package settingswin

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/ui/help"
)

func TestHEICSettingsGuideIsInGeneral(t *testing.T) {
	w := showSettings(t, &fakeHost{})
	general := settingsTabs(t, w).Items[0].Content
	if findHEICGuideButton(general) == nil {
		t.Fatal("General Settings has no HEIC installation instructions action")
	}
}

func TestHEICSettingsGuideActionAndClosedWindow(t *testing.T) {
	host := &fakeHost{}
	w := showSettings(t, host)
	documentation := help.New(testApp, "PicFetch", nil)
	calls := 0
	w.SetHEICGuideAction(func() {
		calls++
		documentation.ShowHEICGuide()
	})
	button := findHEICGuideButton(settingsTabs(t, w).Items[0].Content)
	if button == nil {
		t.Fatal("General Settings has no HEIC installation instructions action")
	}
	test.Tap(button)
	if calls != 1 {
		t.Fatal("the guide button must invoke the Help action exactly once")
	}
	var guide fyne.Window
	for _, win := range testApp.Driver().AllWindows() {
		if win.Title() == lang.L("HEIC installation instructions") {
			guide = win
		}
	}
	if guide == nil {
		t.Fatal("the Settings action must open the actual Help window")
	}
	t.Cleanup(guide.Close)
	if len(host.applyCalls) != 0 || len(host.updateCallbacks) != 0 || host.performCalls != 0 {
		t.Fatal("reading the guide must not change preferences or check updates")
	}
	w.win.Window().Close()
	button.OnTapped()
	if calls != 1 {
		t.Fatal("the closed Settings window must ignore an obsolete guide action")
	}
	w.Show(host.prefs, false)
	button.OnTapped()
	if calls != 1 {
		t.Fatal("reopening Settings must not reactivate the old guide action")
	}
	current := findHEICGuideButton(settingsTabs(t, w).Items[0].Content)
	if current == nil {
		t.Fatal("the reopened Settings tree is missing the guide action")
	}
	test.Tap(current)
	if calls != 2 {
		t.Fatal("the reopened Settings window must have a working guide action")
	}
}

func findHEICGuideButton(root fyne.CanvasObject) *widget.Button {
	switch object := root.(type) {
	case *widget.Button:
		if object.Text == lang.L("HEIC installation instructions") {
			return object
		}
	case *fyne.Container:
		for _, child := range object.Objects {
			if button := findHEICGuideButton(child); button != nil {
				return button
			}
		}
	case *container.Scroll:
		return findHEICGuideButton(object.Content)
	}
	return nil
}
