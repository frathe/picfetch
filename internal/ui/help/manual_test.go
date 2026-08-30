package help

import (
	"path"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

// manuals is shared by every check below that should hold for both shipped
// editions, keyed by embed filename for error messages.
var manuals = map[string]string{
	"manual.md":    manualMD,
	"manual_de.md": manualDE,
}

func TestManualIsEmbedded(t *testing.T) {
	for name, md := range manuals {
		if len(md) == 0 {
			t.Errorf("%s was not embedded", name)
			continue
		}

		if !strings.HasPrefix(md, "# PicFetch") {
			t.Errorf("%s does not start with the expected heading, got %q", name, firstLine(md))
		}
	}
}

// TestManualsShareMascotImages keeps the English and German editions in lockstep
// on the three Trane pictures: same files, same order, so a translation update
// cannot drop one or shuffle them.
func TestManualsShareMascotImages(t *testing.T) {
	want := []string{"TaneWithFrame.webp", "trane_digging.webp", "trane_wags.webp"}
	for name, md := range manuals {
		if got := mascotRefs(md); !reflect.DeepEqual(got, want) {
			t.Errorf("%s mascot images = %v, want %v in that order", name, got, want)
		}
	}
}

var mascotMarkdownRef = regexp.MustCompile(`]\(([^)]+\.webp)\)`)

func mascotRefs(md string) []string {
	var out []string
	for _, m := range mascotMarkdownRef.FindAllStringSubmatch(md, -1) {
		out = append(out, path.Base(m[1]))
	}
	return out
}

// TestManualHasNoMarkdownTables guards the in-app rendering: Fyne's markdown
// support has no table extension, so a table in either manual would show up
// as a wall of pipe characters in the help window.
func TestManualHasNoMarkdownTables(t *testing.T) {
	for name, md := range manuals {
		for i, line := range strings.Split(md, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "|") {
				t.Errorf("%s:%d looks like a markdown table row, which Fyne cannot render: %q", name, i+1, line)
			}
		}
	}
}

// TestManualHasNoUnicodeArrows guards a Fyne font quirk. The theme's regular
// face (NotoSans) carries no arrow glyphs at all, so the shaper falls back to
// the symbol face - InterSymbols, a 23-glyph subset of Inter - and go-text
// keeps that face across the next character instead of splitting the run.
// The subset holds arrows and little else: no space, no "/", no hyphen. So
// whatever follows an arrow shapes to .notdef and the painter substitutes
// U+FFFD for it, and "Windows-Sicherheit \u2192 Viren-" reaches the user as
// "Windows-Sicherheit \u2192\ufffd Viren-".
//
// An earlier version of this check allowed arrows inside code spans, on the
// grounds that the monospace face carries the whole range. That held for the
// span itself and broke everywhere the surrounding markup pulled the text
// back into the regular face. Nothing is gained by drawing the fine line:
// use ASCII throughout - "->" for menu paths and cycles, the key names
// "Left", "Right", "Up" and "Down" for the keys themselves.
func TestManualHasNoUnicodeArrows(t *testing.T) {
	arrow := regexp.MustCompile(`[\x{2190}\x{2191}\x{2192}\x{2193}]`)

	for name, md := range manuals {
		for i, line := range strings.Split(md, "\n") {
			if arrow.MatchString(line) {
				t.Errorf("%s:%d has a Unicode arrow: %q - write the key name (Left/Right/Up/Down) or ASCII \"->\"", name, i+1, strings.TrimSpace(line))
			}
		}
	}
}

// TestManualDocumentsItsOwnShortcut is a cheap consistency check that each
// manual keeps mentioning the key that opens it. F1 is a physical key name,
// not translated, so it reads the same in both editions.
func TestManualDocumentsItsOwnShortcut(t *testing.T) {
	for name, md := range manuals {
		if !strings.Contains(md, "F1") {
			t.Errorf("%s does not mention the F1 shortcut", name)
		}
	}
}

// TestCurrentManual_GermanLocaleUsesGermanManual and
// TestCurrentManual_OtherLocaleFallsBackToEnglish cover currentManual's
// locale switch (manual.go) via the systemLocale var, the way this codebase
// stubs an OS-dependent lookup elsewhere (e.g. clipboard's lookupXClip).
func TestCurrentManual_GermanLocaleUsesGermanManual(t *testing.T) {
	orig := systemLocale
	t.Cleanup(func() { systemLocale = orig })

	for _, loc := range []fyne.Locale{"de", "de-DE", "de-AT", "de-CH"} {
		systemLocale = func() fyne.Locale { return loc }
		if got := currentManual(); got != manualDE {
			t.Errorf("currentManual() for locale %q did not return the German manual", loc)
		}
	}
}

func TestCurrentManual_OtherLocaleFallsBackToEnglish(t *testing.T) {
	orig := systemLocale
	t.Cleanup(func() { systemLocale = orig })

	for _, loc := range []fyne.Locale{"en", "en-US", "fr", "fr-FR"} {
		systemLocale = func() fyne.Locale { return loc }
		if got := currentManual(); got != manualMD {
			t.Errorf("currentManual() for locale %q did not fall back to the English manual", loc)
		}
	}
}

func TestHelpMenu(t *testing.T) {
	help := New(nil, "PicFetch", nil).Menu()

	if help.Label != "Help" {
		t.Errorf("expected menu label %q, got %q", "Help", help.Label)
	}

	if got := len(help.Items); got != 3 {
		t.Fatalf("expected 3 help items, got %d", got)
	}

	manual := help.Items[0]

	if manual.Label != "Manual" {
		t.Errorf("expected item label %q, got %q", "Manual", manual.Label)
	}

	if manual.Action == nil {
		t.Error("manual menu item has no action")
	}

	shortcut, ok := manual.Shortcut.(*desktop.CustomShortcut)
	if !ok {
		t.Fatalf("Manual item Shortcut = %#v, want a *desktop.CustomShortcut for F1", manual.Shortcut)
	}
	if shortcut.KeyName != fyne.KeyF1 || shortcut.Modifier != 0 {
		t.Errorf("Manual accelerator = %+v, want {KeyF1, 0}", shortcut)
	}

	if !help.Items[1].IsSeparator {
		t.Error("expected a separator between Manual and About")
	}

	about := help.Items[2]

	if about.Label != "About" {
		t.Errorf("expected item label %q, got %q", "About", about.Label)
	}

	if about.Action == nil {
		t.Error("about menu item has no action")
	}
}

func firstLine(s string) string {
	if before, _, ok := strings.Cut(s, "\n"); ok {
		return before
	}

	return s
}
