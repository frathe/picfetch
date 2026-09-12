package ui

import (
	"bytes"
	"image/color"
	"io"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/uitest"
)

func TestHypnoTunnel(t *testing.T) {
	t.Run("highest_resolution_snapshot", func(t *testing.T) {
		v := newTestViewer(t)
		small := uitest.PatternedJPEGURISize(t, "a.jpg", 1, 64, 48)
		large := uitest.PatternedJPEGURISize(t, "b.jpg", 1, 192, 144)
		unique := uitest.PatternedJPEGURI(t, "c.jpg", 8)
		dropAndWait(t, v, small, large, unique)
		if err := v.grid.Warm(); err != nil {
			t.Fatal(err)
		}
		v.grid.SetHideDuplicates(true)
		v.grid.Settle()
		waitUntilLoaded(t, v)
		got := v.tunnelSources()
		if !slices.Equal(got, []fyne.URI{large, unique}) {
			t.Fatalf("snapshot ignored canonical representatives: %v", got)
		}
		v.grid.SetHideDuplicates(false)
		if !slices.Equal(got, []fyne.URI{large, unique}) {
			t.Fatal("frozen snapshot changed with duplicate mode")
		}
		if !slices.Equal(v.tunnelSources(), []fyne.URI{small, large, unique}) {
			t.Fatal("new snapshot did not use current mode")
		}
	})
	t.Run("shutdown_admission", func(t *testing.T) {
		v := newTestViewer(t)
		v.stopping = true
		v.openSpiral()
		v.openSpiralForGesture(true)
		if v.spiral.Open() {
			t.Fatal("shutdown admitted a new spiral window")
		}
	})
	t.Run("shutdown_blocked_preview", func(t *testing.T) {
		v := newTestViewer(t)
		u := uitest.TempJPEGURI(t, "held.jpg", 8, 4, color.White)
		data, err := os.ReadFile(u.Path())
		if err != nil {
			t.Fatal(err)
		}
		started, release := make(chan struct{}), make(chan struct{})
		var once sync.Once
		unblock := func() { once.Do(func() { close(release) }) }
		defer unblock()
		held := uitest.ReaderURI(u, func() (io.ReadCloser, error) {
			reader := bytes.NewReader(data)
			return uitest.ReadCloser{
				ReadFunc: func(p []byte) (int, error) {
					close(started)
					<-release
					return reader.Read(p)
				},
				CloseFunc: func() error { return nil },
			}, nil
		})
		v.state.setFiles([]fyne.URI{held}, []fyne.URI{held})
		v.openSpiral()
		select {
		case <-started:
		case <-time.After(testTimeout):
			t.Fatal("preview read did not start")
		}
		lifecycle := testApp.Lifecycle().(interface{ OnStopped() func() })
		original := lifecycle.OnStopped()
		registerShutdown(testApp, v)
		shutdown := lifecycle.OnStopped()
		testApp.Lifecycle().SetOnStopped(original)
		shutdown()
		done := make(chan struct{})
		go func() {
			v.waitForShutdown()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(testTimeout):
			unblock()
			<-done
			t.Fatal("post-event-loop shutdown waited for blocked preview I/O")
		}
		if v.spiral.Open() {
			t.Fatal("shutdown left Spiral open")
		}
		unblock()
		v.spiral.Settle()
	})
	t.Run("entry", func(t *testing.T) {
		v := newTestViewer(t)
		previousTheme := testApp.Settings().Theme()
		testApp.Settings().SetTheme(theme.DefaultTheme())
		t.Cleanup(func() {
			for _, w := range testApp.Driver().AllWindows() {
				if w != nil && w.Title() == "PicFetch Manual" {
					w.Close()
					break
				}
			}
			testApp.Settings().SetTheme(previousTheme)
		})
		v.help.ShowManual()
		var entry *widget.Entry
		var visit func(fyne.CanvasObject)
		visit = func(o fyne.CanvasObject) {
			if e, ok := o.(*widget.Entry); ok {
				entry = e
			}
			if c, ok := o.(*fyne.Container); ok {
				for _, child := range c.Objects {
					visit(child)
				}
			}
		}
		for _, w := range testApp.Driver().AllWindows() {
			if w != nil && w.Title() == "PicFetch Manual" {
				visit(w.Content())
			}
		}
		if entry == nil {
			t.Fatal("manual search is absent from the window")
		}
		entry.OnSubmitted("please hypnotize me")
		if !v.spiral.Open() {
			t.Fatal("manual did not open viewer's Spiral")
		}
		for _, w := range testApp.Driver().AllWindows() {
			if w != nil && w.Title() == "PicFetch Manual" {
				w.Close()
			}
		}
		if v.help.ManualOpen() {
			t.Fatal("manual did not close before testing Spiral's F1 binding")
		}
		for _, w := range testApp.Driver().AllWindows() {
			if w != nil && w.Title() == "Hypno Spiral" {
				w.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyF1})
			}
		}
		if !v.help.ManualOpen() {
			t.Fatal("F1 in Spiral did not open the viewer's manual")
		}
		reads := v.dupes.VisibilityReads()
		v.spiralGesture(true)
		if !v.spiral.Open() || v.dupes.VisibilityReads() != reads {
			t.Fatal("gesture must raise the open session without another snapshot")
		}
	})
	t.Run("snapshot", func(t *testing.T) {
		v := newTestViewer(t)
		uris := uitest.TempDirJPEGURIs(t, "c.jpg", "a.jpg", "b.jpg")
		v.state.setFiles(uris, uris)
		v.dupes.SetHideDuplicates(true) // Unanalysed files remain eligible.
		reads := v.dupes.VisibilityReads()
		got := v.tunnelSources()
		if !slices.Equal(got, uris) || v.dupes.VisibilityReads() != reads+1 {
			t.Fatal("snapshot must retain main order with one visibility read")
		}
		v.state.clearFiles()
		if !slices.Equal(got, uris) {
			t.Fatal("snapshot changed with the viewer")
		}
	})
}
