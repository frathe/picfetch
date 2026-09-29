package ui

import (
	"image/color"
	"runtime"
	"testing"

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
