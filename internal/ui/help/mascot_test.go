package help

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// Fyne's markdown ImageSegment loads a file URI, which a packaged app does
// not ship. These tests pin that the three mascot filenames in the manuals
// are rewritten to the bytes go:embed'd beside this package.

func TestNewManualView_BindsEmbeddedMascots(t *testing.T) {
	v := newManualView("before\n\n![Trane](TaneWithFrame.webp)\n\nafter\n", nil)
	if !hasEmbeddedMascot(v.text.Segments, "TaneWithFrame.webp") {
		t.Fatal("expected TaneWithFrame.webp to render from the embedded resource")
	}
}

func TestNewManualView_RebindsMascotsAfterSearch(t *testing.T) {
	v := newManualView("hello there\n\n![Trane](trane_wags.webp)\n\nunique-tail\n", nil)
	v.submit("unique-tail")
	if !hasEmbeddedMascot(v.text.Segments, "trane_wags.webp") {
		t.Fatal("search re-parse dropped the embedded mascot")
	}
}

// ParseMarkdown Refresh()es file-URI images before bind can rewrite them,
// which logs "Failed to load image" for TaneWithFrame.webp etc. against CWD.
func TestSubmit_DoesNotLoadMascotFromDisk(t *testing.T) {
	t.Chdir(t.TempDir())

	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	v := newManualView("hello there\n\n![Trane](trane_wags.webp)\n\nunique-tail\n", nil)
	buf.Reset()
	v.submit("unique-tail")
	if strings.Contains(buf.String(), "Failed to load image") {
		t.Fatalf("search re-parse loaded mascot from disk:\n%s", buf.String())
	}
}

func TestBindManualImages_RewritesShippedManuals(t *testing.T) {
	want := []string{"TaneWithFrame.webp", "trane_digging.webp", "trane_wags.webp"}
	for name, md := range manuals {
		rt := widget.NewRichTextFromMarkdown(md)
		bindManualImages(rt)
		if files := leftoverMascotFiles(rt.Segments); len(files) > 0 {
			t.Errorf("%s still has file-URI mascots: %v", name, files)
		}
		for _, file := range want {
			if !hasEmbeddedMascot(rt.Segments, file) {
				t.Errorf("%s missing embedded %s", name, file)
			}
		}
	}
}

func TestBindManualImages_LeavesUnknownFiles(t *testing.T) {
	rt := widget.NewRichTextFromMarkdown("![nope](missing.webp)\n")
	bindManualImages(rt)
	if _, ok := rt.Segments[0].(*widget.ImageSegment); !ok {
		t.Fatalf("unknown image became %T, want *widget.ImageSegment", rt.Segments[0])
	}
}

func TestNewManualView_LayoutDoesNotLoadMascotFromDisk(t *testing.T) {
	t.Chdir(t.TempDir())
	a := test.NewApp()
	t.Cleanup(a.Quit)

	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	v := newManualView("before\n\n![Trane](TaneWithFrame.webp)\n\nafter\n", nil)
	w := a.NewWindow("")
	w.SetContent(v.content())
	w.Resize(fyne.NewSize(manualW, 400))
	w.Show()
	v.text.Refresh()

	if strings.Contains(buf.String(), "Failed to load image") {
		t.Fatalf("layout loaded mascot from disk:\n%s", buf.String())
	}
}

func TestNewManualView_RealManualLayoutDoesNotLoadMascotFromDisk(t *testing.T) {
	t.Chdir(t.TempDir())
	a := test.NewApp()
	t.Cleanup(a.Quit)

	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	defer func() {
		if rec := recover(); rec != nil {
			t.Logf("render panic (test theme): %v", rec)
		}
		if strings.Contains(buf.String(), "Failed to load image") {
			t.Errorf("layout loaded mascot from disk:\n%s", buf.String())
		}
	}()

	v := newManualView(manualMD, nil)
	w := a.NewWindow("")
	w.SetContent(v.content())
	w.Resize(fyne.NewSize(manualW, manualH))
	w.Show()
	v.text.Refresh()
}

func hasEmbeddedMascot(segs []widget.RichTextSegment, name string) bool {
	found := false
	walkRichText(segs, func(s widget.RichTextSegment) {
		img, ok := s.Visual().(*canvas.Image)
		if !ok || img.Resource == nil {
			return
		}
		if img.Resource.Name() == name {
			found = true
		}
	})
	return found
}

func leftoverMascotFiles(segs []widget.RichTextSegment) []string {
	var out []string
	walkRichText(segs, func(s widget.RichTextSegment) {
		img, ok := s.(*widget.ImageSegment)
		if !ok || img.Source == nil {
			return
		}
		if _, embedded := mascotResource(img.Source.Name()); embedded {
			out = append(out, img.Source.Name())
		}
	})
	return out
}

func walkRichText(segs []widget.RichTextSegment, fn func(widget.RichTextSegment)) {
	for _, s := range segs {
		fn(s)
		switch t := s.(type) {
		case *widget.ParagraphSegment:
			walkRichText(t.Texts, fn)
		case *widget.ListSegment:
			walkRichText(t.Items, fn)
		}
	}
}

func TestMascotsAreEmbedded(t *testing.T) {
	for _, name := range []string{"TaneWithFrame.webp", "trane_digging.webp", "trane_wags.webp"} {
		res, ok := mascotResource(name)
		if !ok || res == nil || len(res.Content()) == 0 {
			t.Errorf("%s was not embedded", name)
		}
	}
}
