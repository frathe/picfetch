package help

import (
	"net/http"
	"os"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestLicensesWheelScrollsOverCodeBlocks(t *testing.T) {
	h := New(test.NewApp(), "PicFetch", nil)
	h.SetLicenses("# Notices\n\n```text\n" + strings.Repeat("Copyright Example. Permission to redistribute.\n", 80) + "```\n")
	h.ShowLicenses()
	win := h.licensesWin.Window()
	defer win.Close()
	scroll := win.Content().(*container.Scroll)
	for _, target := range []struct {
		name string
		pos  fyne.Position
	}{
		{"heading", fyne.NewPos(100, 20)},
		{"code block", fyne.NewPos(100, 120)},
	} {
		t.Run(target.name, func(t *testing.T) {
			scroll.ScrollToTop()
			test.Scroll(win.Canvas(), target.pos, 0, -100)
			if scroll.Offset.Y <= 0 {
				t.Fatalf("wheel over %s did not scroll the license document", target.name)
			}
		})
	}
	before := scroll.Offset.Y
	test.Scroll(win.Canvas(), fyne.NewPos(100, 120), 0, 40)
	if scroll.Offset.Y >= before {
		t.Fatal("wheel over the code block did not scroll back up")
	}
}

func TestLicensesMenuShowsOfflineMarkdown(t *testing.T) {
	app := &discussionLinkApp{App: test.NewApp()}
	h := New(app, "PicFetch", nil)
	h.SetLicenses("# Third-Party Notices\n\n## Example component\n\n```text\nCopyright Example\nPermission to redistribute.\n```\n")
	h.SetImageClient(&http.Client{Transport: releaseImageTransport(func(_ *http.Request) (*http.Response, error) {
		t.Fatal("opening Licenses must not fetch remote resources")
		return nil, nil
	})})
	var action func()
	for _, item := range h.Menu().Items {
		if item.Label == lang.L("Licenses") {
			action = item.Action
		}
	}
	if action == nil {
		t.Fatal("Help has no Licenses action")
	}
	if h.licensesWin.Open() {
		t.Fatal("building Help must not open Licenses")
	}
	t.Cleanup(func() {
		if win := h.licensesWin.Window(); win != nil {
			win.Close()
		}
	})
	action()
	win := h.licensesWin.Window()
	if win == nil || win.Title() != lang.L("Licenses") {
		t.Fatal("Licenses must open its own titled window")
	}
	text := findRichText(win.Content())
	if text == nil || text.Wrapping != fyne.TextWrapWord {
		t.Fatal("the notice text must reach the visible surface with word wrapping")
	}
	var plain strings.Builder
	for _, segment := range text.Segments {
		plain.WriteString(segment.Textual())
	}
	for _, want := range []string{"Third-Party Notices", "Example component", "Copyright Example", "Permission to redistribute."} {
		if !strings.Contains(plain.String(), want) {
			t.Errorf("Licenses omitted %q", want)
		}
	}
	heading, ok := text.Segments[0].(*widget.TextSegment)
	if !ok || heading.Style != widget.RichTextStyleHeading {
		t.Fatal("notice headings must be rendered as Markdown")
	}
	if app.opened != nil {
		t.Fatal("opening Licenses must not open the browser")
	}
	action()
	if h.licensesWin.Window() != win {
		t.Fatal("reopening Licenses must raise the existing window")
	}
	win.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if h.licensesWin.Open() {
		t.Fatal("Escape must close Licenses")
	}
	action()
	if h.licensesWin.Window() == nil || h.licensesWin.Window() == win {
		t.Fatal("Licenses must reopen after closing")
	}
	h.licensesWin.Window().Close()
	h.Stop()
	action()
	if h.licensesWin.Open() {
		t.Fatal("Licenses must not reopen after application shutdown")
	}
}

func TestLicensesReleaseDocumentIsReadableOffline(t *testing.T) {
	notices, err := os.ReadFile("../../../THIRD-PARTY-NOTICES.md")
	if err != nil {
		t.Fatal(err)
	}
	// Markdown images can initiate synchronous URI reads during layout. Keep
	// the entire shipped document text-only so opening it requires no I/O.
	parsed := widget.NewRichTextFromMarkdown(string(notices))
	walkRichText(parsed.Segments, func(segment widget.RichTextSegment) {
		if _, ok := segment.(*widget.ImageSegment); ok {
			t.Fatal("release notices must not contain images requiring runtime I/O")
		}
	})
	h := New(test.NewApp(), "PicFetch", nil)
	h.SetLicenses(string(notices))
	h.ShowLicenses()
	defer h.licensesWin.Window().Close()
	text := findRichText(h.licensesWin.Window().Content())
	if text == nil || len(text.Segments) != len(parsed.Segments) {
		t.Fatal("the complete notice document must be present in the window")
	}
	for i, segment := range parsed.Segments {
		if text.Segments[i].Textual() != segment.Textual() {
			t.Fatalf("notice content was lost at segment %d", i)
		}
		if _, ok := segment.(*widget.CodeBlockSegment); ok {
			block, ok := text.Segments[i].(*widget.TextSegment)
			if !ok || block.Style != widget.RichTextStyleCodeBlock {
				t.Fatalf("license block %d must retain its monospace block style", i)
			}
		}
	}
	walkRichText(text.Segments, func(segment widget.RichTextSegment) {
		if _, ok := segment.(*widget.CodeBlockSegment); ok {
			t.Fatal("license blocks must not contain nested scrollers")
		}
	})
}
