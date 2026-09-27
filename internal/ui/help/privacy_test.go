package help

import (
	"os"
	"regexp"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestPrivacyPolicyMenuShowsOfflineDocument(t *testing.T) {
	policy, err := os.ReadFile("../../../PRIVACY.md")
	if err != nil {
		t.Fatal(err)
	}
	parsed := widget.NewRichTextFromMarkdown(string(policy))
	walkRichText(parsed.Segments, func(segment widget.RichTextSegment) {
		if _, ok := segment.(*widget.ImageSegment); ok {
			t.Fatal("the bundled privacy policy must not load images during layout")
		}
	})
	if regexp.MustCompile(`[\x{2190}-\x{2193}]`).Match(policy) {
		t.Fatal("the privacy policy contains arrows unsupported by the UI font")
	}

	app := &discussionLinkApp{App: test.NewApp()}
	initialWindows := len(app.Driver().AllWindows())
	h := New(app, "PicFetch", nil)
	h.SetPrivacyPolicy(string(policy))
	t.Cleanup(func() {
		for _, win := range app.Driver().AllWindows()[initialWindows:] {
			win.Close()
		}
	})
	var action func()
	for _, item := range h.Menu().Items {
		if item.Label == lang.L("Privacy policy") {
			action = item.Action
		}
	}
	if action == nil {
		t.Fatal("Help has no Privacy policy action")
	}
	if len(app.Driver().AllWindows()) != initialWindows || app.opened != nil {
		t.Fatal("building Help must not open a window or browser")
	}
	h.SetAdmission(func() bool { return false })
	action()
	if len(app.Driver().AllWindows()) != initialWindows {
		t.Fatal("a refused privacy-policy action opened a window")
	}
	h.SetAdmission(nil)
	action()
	windows := app.Driver().AllWindows()
	if len(windows) != initialWindows+1 || windows[initialWindows].Title() != lang.L("Privacy policy") {
		t.Fatal("Privacy policy must open its own titled window")
	}
	win := windows[initialWindows]
	scroll, ok := win.Content().(*container.Scroll)
	if !ok || scroll.Direction != container.ScrollVerticalOnly {
		t.Fatal("the policy must be in a vertical scroll container")
	}
	text := findRichText(win.Content())
	if text == nil || text.Wrapping != fyne.TextWrapWord || len(text.Segments) != len(parsed.Segments) {
		t.Fatal("the complete policy must reach the visible surface with word wrapping")
	}
	for i, segment := range parsed.Segments {
		if text.Segments[i].Textual() != segment.Textual() {
			t.Fatalf("policy content was lost at segment %d", i)
		}
	}
	heading, ok := text.Segments[0].(*widget.TextSegment)
	if !ok || heading.Style != widget.RichTextStyleHeading || heading.Text != "PicFetch privacy policy" {
		t.Fatal("the policy heading must be rendered as Markdown")
	}
	test.Scroll(win.Canvas(), fyne.NewPos(100, 120), 0, -100)
	if scroll.Offset.Y <= 0 {
		t.Fatal("the policy must scroll with the mouse wheel")
	}
	if app.opened != nil {
		t.Fatal("reading the policy must not open a browser")
	}
	action()
	if windows := app.Driver().AllWindows(); len(windows) != initialWindows+1 || windows[initialWindows] != win {
		t.Fatal("reopening Privacy policy must raise the existing window")
	}
	win.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if len(app.Driver().AllWindows()) != initialWindows {
		t.Fatal("Escape must close Privacy policy")
	}
	action()
	if windows := app.Driver().AllWindows(); len(windows) != initialWindows+1 || windows[initialWindows] == win {
		t.Fatal("Privacy policy must reopen after closing")
	}
	app.Driver().AllWindows()[initialWindows].Close()
	h.Stop()
	action()
	if len(app.Driver().AllWindows()) != initialWindows {
		t.Fatal("Privacy policy must not reopen after shutdown")
	}
}
