package ui

import (
	"image/color"
	"runtime"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"

	"github.com/frathe/picfetch/internal/uitest"
)

func TestPictureFrameSlidingMenuUsesInWindowMenuBar(t *testing.T) {
	v := newTestViewer(t)
	if v.pictureFrameSlidingMenu != inWindowMenuBar() {
		t.Fatalf("pictureFrameSlidingMenu = %v, want inWindowMenuBar() %v", v.pictureFrameSlidingMenu, inWindowMenuBar())
	}
}

func TestInWindowMenuBarFollowsOSMenu(t *testing.T) {
	// darwin keeps the system menu. Every other OS draws the bar in the
	// window, which is the bar picture-frame mode slides.
	if got, want := inWindowMenuBar(), runtime.GOOS != "darwin"; got != want {
		t.Fatalf("inWindowMenuBar() = %v, want %v on %s", got, want, runtime.GOOS)
	}
}

func TestPictureFrameMenu_DetachesInWindowBarUntilExit(t *testing.T) {
	v := newTestViewer(t)
	v.pictureFrameSlidingMenu = true
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))

	menu := v.win.MainMenu()
	if menu == nil {
		t.Fatal("window menu missing before picture-frame mode")
	}

	v.togglePictureFrameMode()
	t.Cleanup(func() { settleSlideshow(t, v) })
	if !v.slides.Active() {
		t.Fatal("picture-frame mode should be on")
	}
	if v.win.MainMenu() != nil {
		t.Fatal("picture-frame mode should detach the in-window menu")
	}
	if got := v.frameChrome.Slide(); got != 0 {
		t.Fatalf("slide on entry = %v, want 0", got)
	}
	if !v.frameChrome.Layer().Visible() {
		t.Fatal("the sliding bar should be shown in picture-frame mode")
	}

	v.togglePictureFrameMode()
	if v.slides.Active() {
		t.Fatal("picture-frame mode should be off")
	}
	if v.win.MainMenu() != menu {
		t.Fatal("leaving picture-frame mode should restore the same menu")
	}
	if v.frameChrome.Layer().Visible() {
		t.Fatal("the sliding bar should hide outside picture-frame mode")
	}
}

func TestPictureFrameMenu_DetachedMenuReachesTheWindowEdge(t *testing.T) {
	v := newTestViewer(t)
	v.pictureFrameSlidingMenu = true
	v.win.Resize(fyne.NewSize(800, 600))
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	if !v.win.Padded() {
		t.Fatal("the window should start padded")
	}

	// glfw leaves the content below the removed menu. Reproduce that inset.
	content := v.win.Content()
	content.Move(fyne.NewPos(4, 40))
	content.Resize(fyne.NewSize(700, 500))

	v.togglePictureFrameMode()
	t.Cleanup(func() { settleSlideshow(t, v) })

	canvasSize := v.win.Canvas().Size()
	if content.Position() != fyne.NewPos(0, 0) {
		t.Fatalf("content position = %v, want the window edge", content.Position())
	}
	if content.Size() != canvasSize {
		t.Fatalf("content size = %v, want the canvas %v", content.Size(), canvasSize)
	}
	if h := v.frameChrome.Size().Height; h < 24 {
		t.Fatalf("hot zone height = %v, want at least 24", h)
	}
	if y := v.frameChrome.Position().Y; y != 0 {
		t.Fatalf("hot zone Y = %v, want the top of the content", y)
	}

	v.togglePictureFrameMode()
	if !v.win.Padded() {
		t.Fatal("leaving picture-frame mode should restore window padding")
	}
	pad := theme.Size(theme.SizeNamePadding)
	if content.Position() != fyne.NewPos(pad, pad) {
		t.Fatalf("content position after exit = %v, want padded origin %v", content.Position(), pad)
	}
}

func TestPictureFrameMenu_KeepsBarWhenSlidingIsOff(t *testing.T) {
	v := newTestViewer(t)
	v.pictureFrameSlidingMenu = false
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))

	menu := v.win.MainMenu()
	v.togglePictureFrameMode()
	t.Cleanup(func() { settleSlideshow(t, v) })
	if v.win.MainMenu() != menu {
		t.Fatal("picture-frame mode should keep the window menu when sliding is off")
	}
	if v.frameChrome.Layer().Visible() {
		t.Fatal("the sliding bar should stay hidden when sliding is off")
	}
}

func TestPictureFrameMenu_ShutdownDropsTheSlidingBar(t *testing.T) {
	application := test.NewApp()
	v, win := buildTestStartupViewer(t, application)
	t.Cleanup(win.Close)
	t.Cleanup(func() { drain(t, v) })
	v.pictureFrameSlidingMenu = true
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	v.togglePictureFrameMode()
	if !v.frameChrome.Layer().Visible() {
		t.Fatal("picture-frame mode should show the sliding bar")
	}

	lifecycle, ok := application.Lifecycle().(interface{ OnStopped() func() })
	if !ok {
		t.Fatal("test app lifecycle does not expose its stopped hook")
	}
	original := lifecycle.OnStopped()
	registerShutdown(application, v)
	shutdown := lifecycle.OnStopped()
	application.Lifecycle().SetOnStopped(original)
	shutdown()

	if v.frameChrome.Layer().Visible() {
		t.Fatal("shutdown should hide the sliding bar")
	}
	v.frameChrome.Wait()
}
