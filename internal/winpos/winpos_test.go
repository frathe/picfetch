package winpos

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// The fyne test driver's windows implement neither driver.NativeWindow nor
// desktop.Window, mirroring every headless test in internal/ui (see
// startWindowPosPolling's own doc comment in internal/ui/windowtrack.go) -
// Get and Set must degrade to a harmless no-op there rather than panicking
// on a failed type assertion.

func TestGet_NonNativeWindowReportsNotOK(t *testing.T) {
	win := test.NewWindow(nil)
	defer win.Close()

	if _, _, ok := Get(win); ok {
		t.Error("Get on a non-native test window should report ok=false")
	}
}

func TestSet_NonDesktopWindowDoesNotPanic(t *testing.T) {
	win := test.NewWindow(nil)
	defer win.Close()

	Set(win, 100, 200)
}

type fixedNativeWindow struct {
	fyne.Window
	calls int
}

func (w *fixedNativeWindow) RunNative(_ func(any)) { w.calls++ }

func TestFixedWindowSkipsNativeMaximize(t *testing.T) {
	w := &fixedNativeWindow{Window: test.NewWindow(nil)}
	defer w.Close()
	w.SetFixedSize(true)
	Maximize(w)
	Unmaximize(w)
	if w.calls != 0 {
		t.Fatal("fixed window admitted a native geometry change")
	}
	w.SetFixedSize(false)
	Maximize(w)
	if w.calls != 1 {
		t.Fatal("ordinary maximize was disabled")
	}
}
