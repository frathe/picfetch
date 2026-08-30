package main

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"
)

func TestArgsToURIs_ResolvesRelativeToAbsolute(t *testing.T) {
	uris := argsToURIs([]string{"one.jpg", filepath.Join("sub", "two.png")})

	if len(uris) != 2 {
		t.Fatalf("len(uris) = %d, want 2", len(uris))
	}

	wantOne, err := filepath.Abs("one.jpg")
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	if uris[0].Path() != wantOne {
		t.Errorf("uris[0].Path() = %q, want %q", uris[0].Path(), wantOne)
	}

	wantTwo, err := filepath.Abs(filepath.Join("sub", "two.png"))
	if err != nil {
		t.Fatalf("filepath.Abs: %v", err)
	}
	if uris[1].Path() != wantTwo {
		t.Errorf("uris[1].Path() = %q, want %q", uris[1].Path(), wantTwo)
	}
}

func TestArgsToURIs_EmptyArgsReturnsEmpty(t *testing.T) {
	if uris := argsToURIs(nil); len(uris) != 0 {
		t.Errorf("len(uris) = %d, want 0", len(uris))
	}
}

func TestArgsToURIs_SkipsEmptyPath(t *testing.T) {
	uris := argsToURIs([]string{"", "photo.jpg"})

	if len(uris) != 1 {
		t.Fatalf("len(uris) = %d, want 1", len(uris))
	}
	if uris[0].Name() != "photo.jpg" {
		t.Errorf("uris[0].Name() = %q, want %q", uris[0].Name(), "photo.jpg")
	}
}

func TestArgsToURIs_AbsoluteArgUnchanged(t *testing.T) {
	uris := argsToURIs([]string{"/already/absolute.jpg"})

	if len(uris) != 1 {
		t.Fatalf("len(uris) = %d, want 1", len(uris))
	}
	if uris[0].Path() != "/already/absolute.jpg" {
		t.Errorf("uris[0].Path() = %q, want %q", uris[0].Path(), "/already/absolute.jpg")
	}
}

// --- translations -----------------------------------------------------------

// loadBundle reads one embedded translation bundle, failing the test if it
// isn't valid JSON - which is the failure mode worth catching early:
// lang.AddTranslationsFS only logs a load error, so a malformed bundle
// ships as an app that silently falls back to the untranslated keys.
func loadBundle(t *testing.T, name string) map[string]string {
	t.Helper()

	data, err := fs.ReadFile(translationsFS, "translations/"+name)
	if err != nil {
		t.Fatalf("reading the embedded %s: %v", name, err)
	}

	var bundle map[string]string
	if err := json.Unmarshal(data, &bundle); err != nil {
		t.Fatalf("parsing %s: %v", name, err)
	}
	if len(bundle) == 0 {
		t.Fatalf("%s is empty", name)
	}

	return bundle
}

// TestTranslations_EveryLocaleCoversEnglish is the guard against the way
// this actually goes wrong: a new lang.L string gets added to en.json and
// nowhere else, and nothing complains until a German user sees an English
// word in the middle of their UI. English is the source locale, so it
// defines the key set every other bundle has to cover exactly - a key only
// another locale has is just as broken, since it means an English string was
// renamed and its translation left stranded.
func TestTranslations_EveryLocaleCoversEnglish(t *testing.T) {
	en := loadBundle(t, "en.json")

	locales, err := fs.Glob(translationsFS, "translations/*.json")
	if err != nil {
		t.Fatalf("listing embedded translations: %v", err)
	}
	if len(locales) < 2 {
		t.Fatalf("found %d translation bundle(s), want English and at least one translated locale", len(locales))
	}

	for _, path := range locales {
		locale := filepath.Base(path)
		if locale == "en.json" {
			continue
		}
		other := loadBundle(t, locale)

		for key := range en {
			if _, ok := other[key]; !ok {
				t.Errorf("%s is missing a translation for %q", locale, key)
			}
		}
		for key := range other {
			if _, ok := en[key]; !ok {
				t.Errorf("%s translates %q, which no longer exists in en.json", locale, key)
			}
		}
	}
}

// TestTranslations_EnglishMapsEachKeyToItself pins the convention this app
// uses: the lang.L argument in the source *is* the English text, so en.json
// is an identity mapping. A value that differs from its key means the
// English UI silently says something other than what the source reads,
// which here is always a typo rather than an intent.
func TestTranslations_EnglishMapsEachKeyToItself(t *testing.T) {
	for key, value := range loadBundle(t, "en.json") {
		if key != value {
			t.Errorf("en.json maps %q to %q, want the key itself", key, value)
		}
	}
}

// TestTranslations_NoArrowFollowedByASpace guards the same Fyne font quirk
// internal/ui/help's manual check guards, on the side the manual cannot see.
// The theme's regular face (NotoSans) has no arrow glyphs, so the shaper
// falls back to the symbol face - InterSymbols, a 23-glyph subset of Inter
// holding the arrows and nothing else. go-text does not let a space end a
// run (shaping.ignoreFaceChange treats every Zs as face-neutral), so the
// space right after an arrow is shaped with that subset, comes out as
// .notdef, and painter.DrawStringOffset substitutes U+FFFD for it, so a
// label reading "Security -> Virus" spelled with U+2192 draws a replacement
// box where the space should be.
//
// Arrows themselves are fine and the spiral overlay ships several - it is
// only the space after one that breaks, which is why this checks the pair
// rather than banning the character.
func TestTranslations_NoArrowFollowedByASpace(t *testing.T) {
	locales, err := fs.Glob(translationsFS, "translations/*.json")
	if err != nil {
		t.Fatalf("listing embedded translations: %v", err)
	}

	for _, path := range locales {
		locale := filepath.Base(path)
		for key, value := range loadBundle(t, locale) {
			for _, s := range []string{key, value} {
				if i := arrowBeforeSpace(s); i >= 0 {
					t.Errorf("%s: %q has an arrow followed by a space at offset %d - the space renders as a box, use \"->\" instead", locale, s, i)
				}
			}
		}
	}
}

// arrowBeforeSpace returns the byte offset of the first arrow that is
// directly followed by a space, or -1.
func arrowBeforeSpace(s string) int {
	var prev rune
	var prevAt int
	for i, r := range s {
		if strings.ContainsRune("\u2190\u2191\u2192\u2193", prev) && unicode.IsSpace(r) {
			return prevAt
		}
		prev, prevAt = r, i
	}
	return -1
}

// TestFyneAppToml_IconExists pins the file fyne install reads when someone
// runs `fyne install github.com/frathe/picfetch@latest`. The CLI clones the
// tagged module into a temp dir and looks up [Details] Icon in FyneApp.toml;
// if that field is empty it falls back to Icon.png at the module root, which
// this repo does not ship (the PNG lives under assets/). An empty Icon is
// exactly the "Missing application icon" failure that command produces.
func TestFyneAppToml_IconExists(t *testing.T) {
	data, err := os.ReadFile("FyneApp.toml")
	if err != nil {
		t.Fatalf("reading FyneApp.toml: %v", err)
	}

	icon := fyneAppQuotedField(string(data), "Icon")
	if icon == "" {
		t.Fatal(`FyneApp.toml has no Icon; fyne install defaults to Icon.png at the module root and fails`)
	}
	if filepath.Ext(icon) != ".png" {
		t.Fatalf("FyneApp.toml Icon = %q, want a .png path", icon)
	}
	if _, err := os.Stat(icon); err != nil {
		t.Fatalf("FyneApp.toml Icon %q does not exist: %v", icon, err)
	}
}

// fyneAppQuotedField returns the first `Key = "value"` assignment in a
// FyneApp.toml body. bump_version.sh parses Version the same way; both
// depend on fyne's encoder keeping quoted strings on their own line.
func fyneAppQuotedField(toml, key string) string {
	prefix := key + ` = "`
	for line := range strings.SplitSeq(toml, "\n") {
		line = strings.TrimSpace(line)
		rest, ok := strings.CutPrefix(line, prefix)
		if !ok {
			continue
		}
		value, _ := strings.CutSuffix(rest, `"`)
		return value
	}
	return ""
}
