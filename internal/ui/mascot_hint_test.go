package ui

import (
	"math"
	"testing"
	"testing/synctest"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Fyne's test driver deliberately returns the last-created window for every
// CanvasForObject call, including its AbsolutePositionForObject implementation.
// Override only absolute geometry: rendering and input stay on the real test
// driver, while head/link coordinates resolve in the actual owning surface
// when the discovery flow moves between the welcome, Finis and manual windows.
type mascotTestApp struct{ fyne.App }

func (a mascotTestApp) Driver() fyne.Driver { return mascotTestDriver{a.App.Driver()} }

type mascotTestDriver struct{ fyne.Driver }

func (d mascotTestDriver) AbsolutePositionForObject(target fyne.CanvasObject) fyne.Position {
	for _, window := range d.AllWindows() {
		if position, found := mascotObjectPosition(window.Content(), target, fyne.Position{}); found {
			return position
		}
	}
	return fyne.Position{}
}

func mascotObjectPosition(root, target fyne.CanvasObject, parent fyne.Position) (fyne.Position, bool) {
	if root == nil {
		return fyne.Position{}, false
	}
	position := parent.Add(root.Position())
	if root == target {
		return position, true
	}
	var children []fyne.CanvasObject
	switch object := root.(type) {
	case *fyne.Container:
		children = object.Objects
	case fyne.Widget:
		children = test.WidgetRenderer(object).Objects()
	}
	for _, child := range children {
		if found, ok := mascotObjectPosition(child, target, position); ok {
			return found, true
		}
	}
	return fyne.Position{}, false
}

func newMascotTestUI(t *testing.T) (*viewer, fyne.Window, func() bool) {
	t.Helper()
	previous := testApp
	testApp = mascotTestApp{previous}
	t.Cleanup(func() { testApp = previous; fyne.SetCurrentApp(previous) })
	return newTestUI(t)
}

func mascotWindow(title string) fyne.Window {
	for _, window := range testApp.Driver().AllWindows() {
		if window.Title() == title {
			return window
		}
	}
	return nil
}

func mascotOrbit(window fyne.Window, center fyne.Position, radius fyne.Size, start, end int, direction float64) {
	for step := start; step <= end; step++ {
		angle := direction * float64(step) * math.Pi / 16
		test.MoveMouse(window.Canvas(), center.Add(fyne.NewPos(
			float32(math.Cos(angle))*radius.Width, float32(math.Sin(angle))*radius.Height)))
	}
}

func TestMascotHintTraneSummonsFinis(t *testing.T) {
	v, window, _ := newMascotTestUI(t)
	t.Cleanup(func() {
		if companion := mascotWindow("Finis"); companion != nil {
			companion.Close()
		}
	})
	center := mascotTraneCenter(v, window)
	radius := fyne.NewSize(42, 38)
	mascotOrbit(window, center, radius, 0, 9*32, 1)
	if mascotWindow("Finis") != nil {
		t.Fatal("nine welcome circles opened Finis")
	}
	mascotOrbit(window, center, radius, 9*32+1, 10*32, 1)
	first := mascotWindow("Finis")
	if first == nil {
		t.Fatal("ten real canvas circles around welcome Trane did not open Finis")
	}
	mascotOrbit(window, center, radius, 0, 10*32, -1)
	if mascotWindow("Finis") != first {
		t.Fatal("another complete attempt replaced the companion window")
	}
	first.Close()
	mascotOrbit(window, center, radius, 0, 9*32, 1)
	if mascotWindow("Finis") != nil {
		t.Fatal("repeat activation retained completed progress")
	}
	mascotOrbit(window, center, radius, 9*32+1, 10*32, 1)
	if companion := mascotWindow("Finis"); companion == nil || companion == first {
		t.Fatal("a new full attempt did not reopen Finis after close")
	}
}

func TestMascotHintTraneResets(t *testing.T) {
	for _, cause := range []string{"exit", "resize", "hide", "expiry"} {
		t.Run(cause, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				v, window, _ := newMascotTestUI(t)
				t.Cleanup(func() {
					if companion := mascotWindow("Finis"); companion != nil {
						companion.Close()
					}
				})
				center := mascotTraneCenter(v, window)
				radius := fyne.NewSize(42, 38)
				mascotOrbit(window, center, radius, 0, 9*32, 1)
				switch cause {
				case "exit":
					test.MoveMouse(window.Canvas(), fyne.NewPos(-10, -10))
				case "resize":
					window.Resize(window.Canvas().Size().AddWidthHeight(180, 120))
				case "hide":
					v.welcomeArt.Hide()
					mascotOrbit(window, center, radius, 0, 10*32, 1)
					if mascotWindow("Finis") != nil {
						t.Fatal("hidden welcome art triggered")
					}
					v.showWelcomeState()
				case "expiry":
					time.Sleep(20*time.Second + time.Nanosecond)
				}
				center = mascotTraneCenter(v, window)
				mascotOrbit(window, center, radius, 0, 32, 1)
				if mascotWindow("Finis") != nil {
					t.Fatal("one circle after a break completed the old attempt")
				}
				mascotOrbit(window, center, radius, 33, 11*32, 1)
				if mascotWindow("Finis") == nil {
					t.Fatal("quiet retry did not summon Finis")
				}
			})
		})
	}
}

func mascotTraneCenter(v *viewer, window fyne.Window) fyne.Position {
	picture := traneSurfaceImage(window.Content(), v.welcomeArt, false)
	scale := min(picture.Size().Width/192, picture.Size().Height/208)
	return testApp.Driver().AbsolutePositionForObject(picture).Add(fyne.NewPos(
		picture.Size().Width/2, (picture.Size().Height-208*scale)/2+64*scale))
}

func TestMascotHintTraneWholeSurface(t *testing.T) {
	v, window, _ := newMascotTestUI(t)
	t.Cleanup(func() {
		if companion := mascotWindow("Finis"); companion != nil {
			companion.Close()
		}
	})
	v.restoreLink.SetText("Restore previous session")
	v.restoreLink.Show()
	window.Resize(fyne.NewSize(900, 650))
	center := mascotTraneCenter(v, window)
	link := testApp.Driver().AbsolutePositionForObject(v.restoreLink).Add(
		fyne.NewPos(v.restoreLink.Size().Width/2, v.restoreLink.Size().Height/2))
	// This noncircular closed path covers the text/link half of the dropzone
	// as well as the portrait. Dense linear samples avoid pointer jumps.
	vertices := []fyne.Position{
		center.Add(fyne.NewPos(42, 0)), center.Add(fyne.NewPos(0, 50)),
		link, center.Add(fyne.NewPos(0, -38)), center.Add(fyne.NewPos(42, 0)),
	}
	for range 10 {
		for i := 1; i < len(vertices); i++ {
			from, to := vertices[i-1], vertices[i]
			for step := 0; step <= 40; step++ {
				test.MoveMouse(window.Canvas(), fyne.NewPos(
					from.X+(to.X-from.X)*float32(step)/40, from.Y+(to.Y-from.Y)*float32(step)/40))
			}
		}
	}
	if mascotWindow("Finis") == nil {
		t.Fatal("valid circles crossing the restore link did not use the whole welcome surface")
	}
	test.MoveMouse(window.Canvas(), link)
	if v.restoreLink.Cursor() != desktop.PointerCursor {
		t.Fatalf("whole-surface tracking suppressed the restore link hover feedback: point %v, link %v/%v, visible %v", link, testApp.Driver().AbsolutePositionForObject(v.restoreLink), v.restoreLink.Size(), v.restoreLink.Visible())
	}
	test.MoveMouse(window.Canvas(), fyne.NewPos(-10, -10))
	if v.restoreLink.Cursor() != desktop.DefaultCursor {
		t.Fatal("restore link retained hover feedback after leaving the dropzone")
	}
}

func mascotVisibleObjects(root fyne.CanvasObject, visit func(fyne.CanvasObject)) {
	if !root.Visible() {
		return
	}
	visit(root)
	var children []fyne.CanvasObject
	switch object := root.(type) {
	case *fyne.Container:
		children = object.Objects
	case fyne.Widget:
		children = test.WidgetRenderer(object).Objects()
	}
	for _, child := range children {
		mascotVisibleObjects(child, visit)
	}
}

func mascotClues(window fyne.Window) []*widget.Label {
	var labels []*widget.Label
	mascotVisibleObjects(window.Content(), func(object fyne.CanvasObject) {
		if label, ok := object.(*widget.Label); ok && label.Text == "please hypnotize me (search for it)" {
			labels = append(labels, label)
		}
	})
	return labels
}

func mascotManualEntry(t *testing.T) testTextEntry {
	t.Helper()
	window := mascotWindow("PicFetch Manual")
	if window == nil {
		t.Fatal("manual window absent")
	}
	var entry testTextEntry
	mascotVisibleObjects(window.Content(), func(object fyne.CanvasObject) {
		if candidate, ok := object.(testTextEntry); ok {
			entry = candidate
		}
	})
	if entry == nil {
		t.Fatal("real manual has no search entry")
	}
	return entry
}

func mascotUseNormalTheme(t *testing.T) {
	t.Helper()
	previous := testApp.Settings().Theme()
	testApp.Settings().SetTheme(theme.DefaultTheme())
	t.Cleanup(func() {
		for _, title := range []string{"Finis", "PicFetch Manual"} {
			if window := mascotWindow(title); window != nil {
				window.Close()
			}
		}
		testApp.Settings().SetTheme(previous)
	})
}

func mascotFinisCenter(window fyne.Window) fyne.Position {
	size := window.Canvas().Size()
	return fyne.NewPos(size.Width/2, size.Height/2-40)
}

func TestMascotHintFinisDiscovery(t *testing.T) {
	for _, route := range []string{"Trane", "manual"} {
		t.Run(route, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				v, main, _ := newMascotTestUI(t)
				mascotUseNormalTheme(t)
				if route == "Trane" {
					mascotOrbit(main, mascotTraneCenter(v, main), fyne.NewSize(42, 38), 0, 320, 1)
				} else {
					v.help.ShowManual()
					entry := mascotManualEntry(t)
					entry.SetText("finis")
					entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
				}
				companion := mascotWindow("Finis")
				if companion == nil {
					t.Fatal("opening route did not show Finis")
				}
				// Stages have no shared deadline or required direction.
				time.Sleep(time.Hour)
				center := mascotFinisCenter(companion)
				radius := fyne.NewSize(170, 130)
				mascotOrbit(companion, center, radius, 0, 9*32, -1)
				if len(mascotClues(companion)) != 0 {
					t.Fatal("nine Finis circles revealed the clue")
				}
				mascotOrbit(companion, center, radius, 9*32+1, 10*32, -1)
				if len(mascotClues(companion)) != 1 {
					t.Fatal("ten Finis circles did not reveal one clue on the real surface")
				}
				test.MoveMouse(companion.Canvas(), fyne.NewPos(-10, -10))
				companion.Resize(fyne.NewSize(720, 540))
				mascotOrbit(companion, mascotFinisCenter(companion), radius, 0, 20*32, 1)
				mascotOrbit(main, mascotTraneCenter(v, main), fyne.NewSize(42, 38), 0, 320, 1)
				time.Sleep(time.Hour)
				if mascotWindow("Finis") != companion || len(mascotClues(companion)) != 1 {
					t.Fatal("exit, resize, repeated circles or Trane replaced the persistent clue")
				}
				companion.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyEscape})
				v.help.ShowFinis()
				reopened := mascotWindow("Finis")
				if reopened == nil || reopened == companion || len(mascotClues(reopened)) != 0 {
					t.Fatal("close/reopen retained the previous reveal")
				}
			})
		})
	}
}

func TestMascotHintCompleteDiscovery(t *testing.T) {
	for _, manualState := range []string{"new", "existing query"} {
		t.Run(manualState, func(t *testing.T) {
			v, main, _ := newMascotTestUI(t)
			mascotUseNormalTheme(t)
			var previousManual fyne.Window
			if manualState == "existing query" {
				v.help.ShowManual()
				previousManual = mascotWindow("PicFetch Manual")
				entry := mascotManualEntry(t)
				entry.SetText("image")
				entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
				previousManual.Canvas().Unfocus()
			}
			mascotOrbit(main, mascotTraneCenter(v, main), fyne.NewSize(42, 38), 0, 320, 1)
			companion := mascotWindow("Finis")
			if companion == nil {
				t.Fatal("Trane did not summon Finis")
			}
			mascotOrbit(companion, mascotFinisCenter(companion), fyne.NewSize(170, 130), 0, 320, -1)
			clues := mascotClues(companion)
			if len(clues) != 1 {
				t.Fatal("Finis did not reveal the clue")
			}
			pointer := false
			mascotVisibleObjects(companion.Content(), func(object fyne.CanvasObject) {
				if cursor, ok := object.(desktop.Cursorable); ok && cursor.Cursor() == desktop.PointerCursor {
					pointer = true
				}
			})
			if !pointer {
				t.Error("clue has no pointer affordance")
			}
			label := clues[0]
			point := testApp.Driver().AbsolutePositionForObject(label).Add(fyne.NewPos(label.Size().Width/2, label.Size().Height/2))
			test.TapCanvas(companion.Canvas(), point)
			manual := mascotWindow("PicFetch Manual")
			if previousManual != nil && manual != previousManual {
				t.Fatal("click replaced the existing manual")
			}
			entry := mascotManualEntry(t)
			entry.TypedShortcut(&fyne.ShortcutSelectAll{})
			if entry.SelectedText() != "" || manual.Canvas().Focused() != entry {
				t.Fatal("clue did not leave an empty focused search")
			}
			if v.spiral.Open() || mascotWindow("Finis") != companion || len(mascotClues(companion)) != 1 {
				t.Fatal("click opened Spiral or dismissed Finis's clue")
			}
			test.Type(entry, "please hypnotize me")
			entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
			if !v.spiral.Open() {
				t.Fatal("typing the clue did not open the viewer's Spiral")
			}
		})
	}
}

func TestMascotHintFinisResets(t *testing.T) {
	for _, cause := range []string{"exit", "resize", "hide", "close", "expiry"} {
		t.Run(cause, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				v, _, _ := newMascotTestUI(t)
				mascotUseNormalTheme(t)
				v.help.ShowFinis()
				window := mascotWindow("Finis")
				radius := fyne.NewSize(80, 60)
				mascotOrbit(window, mascotFinisCenter(window), radius, 0, 9*32, 1)
				switch cause {
				case "exit":
					test.MoveMouse(window.Canvas(), fyne.NewPos(-10, -10))
				case "resize":
					window.Resize(fyne.NewSize(720, 540))
				case "hide":
					window.Content().Hide()
					mascotOrbit(window, mascotFinisCenter(window), radius, 0, 320, 1)
					window.Content().Show()
				case "close":
					window.Close()
					v.help.ShowFinis()
					window = mascotWindow("Finis")
				case "expiry":
					time.Sleep(20*time.Second + time.Nanosecond)
				}
				mascotOrbit(window, mascotFinisCenter(window), radius, 0, 32, 1)
				if len(mascotClues(window)) != 0 {
					t.Fatal("one circle after a break revealed the clue")
				}
				mascotOrbit(window, mascotFinisCenter(window), radius, 33, 11*32, 1)
				if len(mascotClues(window)) != 1 {
					t.Fatal("Finis did not accept a fresh complete attempt")
				}
			})
		})
	}
}
