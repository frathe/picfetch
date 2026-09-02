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

func TestManualDocumentsCopySelection(t *testing.T) {
	for name, md := range manuals {
		if !strings.Contains(md, "Opt") || !strings.Contains(md, "Alt+Shift+C") {
			t.Errorf("%s does not document the Copy selection shortcut Opt/Alt+Shift+C", name)
		}
	}
	if !strings.Contains(manualMD, "Actions -> Copy selection") {
		t.Error("manual.md does not document Actions -> Copy selection")
	}
	if !strings.Contains(manualDE, "Aktionen -> Auswahl kopieren") {
		t.Error("manual_de.md does not document Aktionen -> Auswahl kopieren")
	}
}

func TestManualDocumentsComparisonEntryAndExit(t *testing.T) {
	for name, phrase := range map[string]string{
		"manual.md":    "Actions -> Compare selected images",
		"manual_de.md": "Aktionen -> Ausgewählte Bilder vergleichen",
	} {
		if !strings.Contains(manuals[name], phrase) {
			t.Errorf("%s does not document %s", name, phrase)
		}
	}
	if !strings.Contains(manualMD, "Cmd/Ctrl+D") || !strings.Contains(manualMD, "Back to Grid") {
		t.Error("manual.md does not document the comparison shortcut and exit")
	}
	if !strings.Contains(manualMD, "physical **`Ctrl+D`** also works on macOS") {
		t.Error("manual.md does not document the physical Ctrl+D comparison shortcut on macOS")
	}
	if !strings.Contains(manualDE, "Cmd/Strg+D") || !strings.Contains(manualDE, "Zurück zur Rasteransicht") {
		t.Error("manual_de.md does not document the comparison shortcut and exit")
	}
	if !strings.Contains(manualDE, "physische **`Ctrl+D`** funktioniert unter macOS ebenfalls") {
		t.Error("manual_de.md does not document the physical Ctrl+D comparison shortcut on macOS")
	}
}

func TestManualDocumentsComparisonIdentityAndSwap(t *testing.T) {
	for _, phrase := range []string{
		"bottom-corner badges",
		"**Swap**",
		"Compare: left.jpg | right.jpg - PicFetch",
	} {
		if !strings.Contains(manualMD, phrase) {
			t.Errorf("manual.md does not document comparison identity/Swap phrase %q", phrase)
		}
	}
	for _, phrase := range []string{
		"Abzeichen in den unteren Ecken",
		"**Tauschen**",
		"Vergleich: links.jpg | rechts.jpg - PicFetch",
	} {
		if !strings.Contains(manualDE, phrase) {
			t.Errorf("manual_de.md does not document comparison identity/Swap phrase %q", phrase)
		}
	}
}

func TestManualDocumentsComparisonCommandIsolation(t *testing.T) {
	for _, phrase := range []string{
		"exclusive main-window mode",
		"Return to Grid View before opening files",
		"F1",
	} {
		if !strings.Contains(manualMD, phrase) {
			t.Errorf("manual.md does not document comparison isolation phrase %q", phrase)
		}
	}
	for _, phrase := range []string{
		"exklusiver Hauptfenster-Modus",
		"Kehren Sie zur Rasteransicht zurück, bevor Sie Dateien öffnen",
		"F1",
	} {
		if !strings.Contains(manualDE, phrase) {
			t.Errorf("manual_de.md does not document comparison isolation phrase %q", phrase)
		}
	}
}

func TestManualDocumentsComparisonCameraAndPhotoControls(t *testing.T) {
	english := strings.Join(strings.Fields(manualMD), " ")
	for _, phrase := range []string{
		"zoom and pan operate one overhead camera above the two photos",
		"**`0`** frames both photos in their current arrangement with one camera move",
		"**`1`** returns the camera to its 1x home view",
		"using Shift+scroll moves both views by the same screen distance",
		"Here **`0`** fits and centres only that photo",
		"Changing the link state never moves or resizes either photo",
		"top-left **Unlink** button",
		"remain inactive until both images are ready",
		"button changes to **Link**",
		"status **Unlinked** appears immediately beside it",
	} {
		if !strings.Contains(english, phrase) {
			t.Errorf("manual.md does not document comparison camera/photo phrase %q", phrase)
		}
	}
	german := strings.Join(strings.Fields(manualDE), " ")
	for _, phrase := range []string{
		"Zoom und Verschieben eine gemeinsame Kamera über den beiden Fotos",
		"**`0`** rahmt beide Fotos in ihrer aktuellen Anordnung mit einer Kamerabewegung ein",
		"**`1`** setzt die Kamera auf ihre 1x-Ausgangsansicht",
		"Shift+Scrollen bewegt beide Ansichten um dieselbe Bildschirmstrecke",
		"passt **`0`** nur dieses Foto in die aktuelle Kameraansicht ein",
		"Das Umschalten der Kopplung bewegt oder skaliert keines der Fotos",
		"Schaltfläche **Entkoppeln** oben links",
		"bleiben inaktiv, bis beide Bilder bereit sind",
		"wechselt die Schaltfläche zu **Koppeln**",
		"Status **Entkoppelt** erscheint direkt daneben",
	} {
		if !strings.Contains(german, phrase) {
			t.Errorf("manual_de.md does not document comparison camera/photo phrase %q", phrase)
		}
	}
}

func TestManualDocumentsComparisonSwipe(t *testing.T) {
	english := strings.Join(strings.Fields(manualMD), " ")
	for _, phrase := range []string{
		"**Swipe** switches both images to the full comparison viewport",
		"Drag the divider to change the reveal; dragging elsewhere continues to pan both images",
		"**`Left`** / **`Right`** move the divider by 5 percentage points",
		"**`Shift+Left`** / **`Shift+Right`** by 1 point",
		"**`Home`** / **`End`** move it to 0%/100%",
		"**Side by side** returns",
	} {
		if !strings.Contains(english, phrase) {
			t.Errorf("manual.md does not document swipe comparison phrase %q", phrase)
		}
	}

	german := strings.Join(strings.Fields(manualDE), " ")
	for _, phrase := range []string{
		"**Wischen** legt beide Bilder über den vollständigen Vergleichsbereich",
		"Ziehen Sie die Trennlinie, um die Aufteilung zu ändern; Ziehen an anderer Stelle verschiebt weiterhin beide Bilder",
		"Solange Wischen aktiv ist, verschieben **`Left`** / **`Right`** die Trennlinie um 5 Prozentpunkte",
		"**`Shift+Left`** / **`Shift+Right`** um 1 Prozentpunkt",
		"**`Home`** / **`End`** setzen sie auf 0 %/100 %",
		"**Nebeneinander** kehrt",
	} {
		if !strings.Contains(german, phrase) {
			t.Errorf("manual_de.md does not document swipe comparison phrase %q", phrase)
		}
	}
}

func TestManualDocumentsComparisonSourceFidelity(t *testing.T) {
	english := strings.Join(strings.Fields(manualMD), " ")
	for _, phrase := range []string{
		"full decoded resolution",
		"canonical EXIF-corrected orientation",
		"SVGs re-render at their effective screen-pixel size",
		"RAW files use the same embedded JPEG preview",
		"Animated inputs stay frozen on their first decoded frame",
		"overview remains visible while sharper detail tiles arrive in the background",
		"Pan and zoom update that stable GPU surface directly",
		"combined decoded memory",
		"encoded-input and vector-raster limits",
		"never downsampled or removed",
	} {
		if !strings.Contains(english, phrase) {
			t.Errorf("manual.md does not document comparison source fidelity phrase %q", phrase)
		}
	}

	german := strings.Join(strings.Fields(manualDE), " ")
	for _, phrase := range []string{
		"vollen dekodierten Auflösung",
		"kanonische EXIF-korrigierte Ausrichtung",
		"SVGs werden für ihre effektive Bildschirm-Pixelgröße neu gerendert",
		"RAW-Dateien verwenden dieselbe eingebettete JPEG-Vorschau",
		"Animierte Eingaben bleiben auf ihrem ersten dekodierten Einzelbild eingefroren",
		"Übersicht bleibt sichtbar, während schärfere Detailkacheln im Hintergrund eintreffen",
		"Verschieben und Zoomen aktualisieren diese stabile GPU-Fläche direkt",
		"kombinierten dekodierten Speicher",
		"Grenzen für kodierte Eingaben und Vektor-Raster",
		"weder verkleinert noch entfernt",
	} {
		if !strings.Contains(german, phrase) {
			t.Errorf("manual_de.md does not document comparison source fidelity phrase %q", phrase)
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
