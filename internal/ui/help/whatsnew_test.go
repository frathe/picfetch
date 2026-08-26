package help

import (
	"fmt"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestShowWhatsNew_OpensTitleBodyAndRaises(t *testing.T) {
	h := New(test.NewApp(), "PicFetch", nil)

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
