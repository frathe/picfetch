package help

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/uitest"
)

func TestReleaseNotesMenuOpensBundledNotesAndHistory(t *testing.T) {
	app := &releaseNotesApp{discussionLinkApp: discussionLinkApp{App: test.NewApp()}, version: "1.2.3"}
	h := newReleaseNotesHelp(t, app)
	var action func()
	for _, item := range h.Menu().Items {
		if item.Label == lang.L("Release Notes") {
			action = item.Action
		}
	}
	if action == nil {
		t.Fatal("Help has no Release Notes action")
	}
	if h.WhatsNewOpen() || app.opened != nil {
		t.Fatal("building the menu should not open notes or the browser")
	}
	action()
	win := h.whatsNewWin.Window()
	if win == nil {
		t.Fatal("Release Notes did not open the notes window")
	}
	defer func() {
		if current := h.whatsNewWin.Window(); current != nil {
			current.Close()
		}
	}()
	if want := fmt.Sprintf(lang.L("What's New in %s"), app.version); win.Title() != want {
		t.Errorf("title = %q, want %q", win.Title(), want)
	}
	rt := findRichText(win.Content())
	if rt == nil || !strings.Contains(richTextPlain(rt.Segments), "What's Changed") {
		t.Fatal("Release Notes must show the bundled release body")
	}
	assertReleaseHistoryLink(t, win.Content(), app)
	action()
	if h.whatsNewWin.Window() != win {
		t.Fatal("reopening Release Notes must raise the existing window")
	}
	win.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if h.WhatsNewOpen() {
		t.Fatal("Escape did not close the release notes")
	}
	action()
	if h.whatsNewWin.Window() == nil || h.whatsNewWin.Window() == win {
		t.Fatal("closed release notes must be reopenable")
	}
}

func newReleaseNotesHelp(t *testing.T, app fyne.App) *Help {
	t.Helper()
	h := New(app, "PicFetch", nil)
	h.SetUIQueue(&uitest.UIQueue{})
	data := releaseTestPNG(t)
	h.imageClient = &http.Client{Transport: releaseImageTransport(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(data))}, nil
	})}
	t.Cleanup(func() {
		h.Stop()
		h.Settle()
		if window := h.whatsNewWin.Window(); window != nil {
			window.Close()
		}
	})
	return h
}

type releaseImageTransport func(*http.Request) (*http.Response, error)

func (f releaseImageTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func releaseTestPNG(t *testing.T) []byte {
	t.Helper()
	picture := image.NewNRGBA(image.Rect(0, 0, 8, 12))
	for y := range 12 {
		for x := range 8 {
			picture.SetNRGBA(x, y, color.NRGBA{R: 180, A: 255})
		}
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, picture); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestReleaseNotesImagesLoadWithoutBlocking(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	data := releaseTestPNG(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "raw=true" {
			t.Errorf("image query = %q, want raw=true", r.URL.RawQuery)
		}
		close(started)
		select {
		case <-release:
			_, _ = w.Write(data)
		case <-r.Context().Done():
		}
	}))
	defer server.Close()
	h := newReleaseNotesHelp(t, test.NewApp())
	h.imageClient = server.Client()
	h.ShowWhatsNew("1.2.3", "# Notes\n\n![Art]("+server.URL+"/new-image.png?raw=true)\n\nStill readable")
	defer h.Stop()
	text := findRichText(h.whatsNewWin.Window().Content())
	if !strings.Contains(richTextPlain(text.Segments), "Still readable") {
		t.Fatal("note text must be readable while the image loads")
	}
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("release-note image URL was never requested")
	}
	close(release)
	h.Settle()
	var loaded bool
	walkRichText(text.Segments, func(segment widget.RichTextSegment) {
		if strings.Contains(segment.Textual(), lang.L("Loading image...")) {
			t.Error("image is still loading after settlement")
		}
		var walk func(fyne.CanvasObject)
		walk = func(object fyne.CanvasObject) {
			if group, ok := object.(*fyne.Container); ok {
				for _, child := range group.Objects {
					walk(child)
				}
			}
			if picture, ok := object.(*canvas.Image); ok && picture.Image != nil && picture.Image.Bounds().Dx() == 8 {
				loaded = true
			}
		}
		walk(segment.Visual())
	})
	if !loaded {
		t.Fatal("downloaded image did not reach the notes surface")
	}
}

func TestReleaseNotesPreparesNestedImagesBeforeLayout(t *testing.T) {
	parsed := widget.NewRichTextFromMarkdown("- ![list](https://example.invalid/list.png)\n\n| ![heading](https://example.invalid/header.png) |\n| --- |\n| ![cell](https://example.invalid/cell.png) |")
	var requests []releaseImageRequest
	parsed.Segments = prepareReleaseImages(parsed.Segments, &requests)
	if len(requests) != 3 {
		t.Fatalf("prepared %d image requests, want list, table heading and table cell", len(requests))
	}
}

func TestReleaseNotesWithoutImagesDoesNotFetch(t *testing.T) {
	h := newReleaseNotesHelp(t, test.NewApp())
	h.imageClient = &http.Client{Transport: releaseImageTransport(func(_ *http.Request) (*http.Response, error) {
		t.Error("notes without images must not start an image request")
		return nil, errors.New("unexpected image request")
	})}
	h.ShowWhatsNew("1.2.3", "# Changes\n\nPlain release notes with a [changelog](https://example.invalid/releases).")
	h.Settle()
	text := findRichText(h.whatsNewWin.Window().Content())
	if !strings.Contains(richTextPlain(text.Segments), "Plain release notes") {
		t.Fatal("notes without images were not rendered")
	}
	if len(releaseImageSegments(text)) != 0 {
		t.Fatal("notes without images must not contain image placeholders")
	}
}

func TestReleaseNotesImageCloseAndStopCancelRequests(t *testing.T) {
	for _, terminal := range []bool{false, true} {
		t.Run(fmt.Sprintf("terminal=%v", terminal), func(t *testing.T) {
			h := newReleaseNotesHelp(t, test.NewApp())
			started, cancelled := make(chan struct{}), make(chan struct{})
			h.imageClient = &http.Client{Transport: releaseImageTransport(func(request *http.Request) (*http.Response, error) {
				close(started)
				<-request.Context().Done()
				close(cancelled)
				return nil, request.Context().Err()
			})}
			h.ShowWhatsNew("1.2.3", "![Art](https://example.invalid/different-release.png)")
			waitReleaseImageSignal(t, started)
			if terminal {
				h.Stop()
			}
			h.whatsNewWin.Window().Close()
			waitReleaseImageSignal(t, cancelled)
			h.Settle()
			h.ShowWhatsNew("1.2.3", "New window")
			if h.WhatsNewOpen() == terminal {
				t.Fatalf("open after retry = %v, terminal = %v", h.WhatsNewOpen(), terminal)
			}
		})
	}
}

func TestReleaseNotesQueuedImageCannotReachClosedWindow(t *testing.T) {
	h := newReleaseNotesHelp(t, test.NewApp())
	h.ShowWhatsNew("1.2.3", "![Old](https://example.invalid/old.png)")
	old := releaseImageSegments(findRichText(h.whatsNewWin.Window().Content()))[0]
	h.Wait() // The download finished, but its UI delivery is still queued.
	if old.picture != nil {
		t.Fatal("image was published outside the UI queue")
	}
	h.whatsNewWin.Window().Close()
	h.ShowWhatsNew("1.2.3", "- New release\n\n  ![New](https://example.invalid/new.png)")
	current := releaseImageSegments(findRichText(h.whatsNewWin.Window().Content()))[0]
	h.Settle()
	if old.picture != nil || current.picture == nil {
		t.Fatal("stale delivery reached the closed window or current image was lost")
	}
}

func TestReleaseNotesFailedImageKeepsNotesReadable(t *testing.T) {
	h := newReleaseNotesHelp(t, test.NewApp())
	h.imageClient = &http.Client{Transport: releaseImageTransport(func(_ *http.Request) (*http.Response, error) {
		return nil, errors.New("offline")
	})}
	h.ShowWhatsNew("1.2.3", "# Changes\n\n![Art](https://example.invalid/art.png)\n\nRelease text")
	h.Settle()
	text := findRichText(h.whatsNewWin.Window().Content())
	art := releaseImageSegments(text)[0]
	if art.picture != nil || art.message != lang.L("Image unavailable") || !strings.Contains(richTextPlain(text.Segments), "Release text") {
		t.Fatal("image failure must retain the notes and replace the loading status")
	}
}

func releaseImageSegments(text *widget.RichText) []*releaseImageSegment {
	var images []*releaseImageSegment
	walkRichText(text.Segments, func(segment widget.RichTextSegment) {
		if art, ok := segment.(*releaseImageSegment); ok {
			images = append(images, art)
		}
	})
	return images
}

func waitReleaseImageSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(3 * time.Second):
		t.Fatal("image worker did not reach the expected lifecycle event")
	}
}

func TestReleaseNotesBundleMatchesPublishedVersion(t *testing.T) {
	published, err := os.ReadFile("../../../.github/release-notes.md")
	if err != nil {
		t.Fatal(err)
	}
	if strings.ReplaceAll(string(published), "\r\n", "\n") != strings.ReplaceAll(releaseNotesMD, "\r\n", "\n") {
		t.Fatal("bundled release notes differ from the published source; regenerate both with make release")
	}
	metadata, err := os.ReadFile("../../../FyneApp.toml")
	if err != nil {
		t.Fatal(err)
	}
	match := regexp.MustCompile(`(?m)^Version = "([^"]+)"`).FindSubmatch(metadata)
	if len(match) != 2 {
		t.Fatal("FyneApp.toml has no version")
	}
	if !strings.HasSuffix(strings.TrimSpace(releaseNotesMD), "...v"+string(match[1])) {
		t.Fatal("bundled notes do not describe the version in FyneApp.toml")
	}
	if regexp.MustCompile(`[\x{2190}-\x{2193}]`).MatchString(releaseNotesMD) {
		t.Fatal("release notes contain Unicode arrows unsupported by the UI font; use ASCII or key names")
	}
}

func assertReleaseHistoryLink(t *testing.T, content fyne.CanvasObject, app *releaseNotesApp) {
	t.Helper()
	var link *widget.Hyperlink
	var walk func(fyne.CanvasObject)
	walk = func(obj fyne.CanvasObject) {
		switch o := obj.(type) {
		case *widget.Hyperlink:
			if o.Text == lang.L("Browse all releases") {
				link = o
			}
		case *fyne.Container:
			for _, child := range o.Objects {
				walk(child)
			}
		}
	}
	walk(content)
	if link == nil || !link.Visible() {
		t.Fatal("notes surface has no visible Browse all releases link")
	}
	if app.opened != nil {
		t.Fatal("showing release notes must not open the browser")
	}
	// Hyperlinks only react over their text, not the empty width of the footer.
	test.TapAt(link, fyne.NewPos(12, link.Size().Height/2))
	if app.opened == nil || app.opened.String() != "https://github.com/frathe/picfetch/releases" {
		t.Fatalf("history opened %v, want the public releases page", app.opened)
	}
}

type releaseNotesApp struct {
	discussionLinkApp
	version string
}

func (a *releaseNotesApp) Metadata() fyne.AppMetadata {
	return fyne.AppMetadata{Version: a.version}
}

func TestShowWhatsNew_OpensTitleBodyAndRaises(t *testing.T) {
	app := &releaseNotesApp{discussionLinkApp: discussionLinkApp{App: test.NewApp()}}
	h := New(app, "PicFetch", nil)

	h.ShowWhatsNew("v0.2.6", "# Hi\n\n- item")

	if !h.WhatsNewOpen() {
		t.Fatal("WhatsNewOpen should be true after ShowWhatsNew")
	}
	win := h.whatsNewWin.Window()
	if win == nil {
		t.Fatal("ShowWhatsNew did not open a window")
	}

	wantTitle := fmt.Sprintf(lang.L("What's New in %s"), "0.2.6")
	if win.Title() != wantTitle {
		t.Errorf("title = %q, want %q", win.Title(), wantTitle)
	}

	rt := findRichText(win.Content())
	if rt == nil {
		t.Fatal("window content has no RichText")
	}
	if len(rt.Segments) == 0 {
		t.Error("RichText has no segments")
	}
	assertReleaseHistoryLink(t, win.Content(), app)

	h.ShowWhatsNew("v0.2.6", "# other")
	if h.whatsNewWin.Window() != win {
		t.Error("a second ShowWhatsNew call should raise the existing window, not open a new one")
	}

	win.Close()
	if h.WhatsNewOpen() {
		t.Error("closing the What's New window should leave the singleton closed")
	}
}

func TestShowWhatsNew_EmptyBodyUsesFallback(t *testing.T) {
	h := New(test.NewApp(), "PicFetch", nil)

	h.ShowWhatsNew("v0.2.6", "")

	win := h.whatsNewWin.Window()
	if win == nil {
		t.Fatal("ShowWhatsNew did not open a window")
	}
	defer win.Close()

	rt := findRichText(win.Content())
	if rt == nil {
		t.Fatal("window content has no RichText")
	}
	got := richTextPlain(rt.Segments)
	want := lang.L("This release has no notes.")
	if !strings.Contains(got, want) {
		t.Errorf("body %q does not contain fallback %q", got, want)
	}
}

func findRichText(obj fyne.CanvasObject) *widget.RichText {
	switch o := obj.(type) {
	case *widget.RichText:
		return o
	case *container.Scroll:
		return findRichText(o.Content)
	case *fyne.Container:
		for _, c := range o.Objects {
			if rt := findRichText(c); rt != nil {
				return rt
			}
		}
	}
	return nil
}

func richTextPlain(segs []widget.RichTextSegment) string {
	var b strings.Builder
	var walk func([]widget.RichTextSegment)
	walk = func(segs []widget.RichTextSegment) {
		for _, s := range segs {
			switch t := s.(type) {
			case *widget.TextSegment:
				b.WriteString(t.Text)
			case *widget.ParagraphSegment:
				walk(t.Texts)
			case *widget.ListSegment:
				walk(t.Items)
			}
		}
	}
	walk(segs)
	return b.String()
}
