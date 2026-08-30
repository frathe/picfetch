package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// TestTranslationsHaveNoUnicodeArrows extends the manuals' arrow ban to every
// string the app puts on screen through lang.L. The quirk is the same one
// internal/ui/help documents: NotoSans has no arrow glyphs, the shaper falls
// back to InterSymbols, and that 23-glyph subset has no space, no "/" and no
// hyphen - so the character after an arrow shapes to .notdef and is painted
// as U+FFFD.
//
// The catalogues are the right place to check because every translated
// string appears here as its own key, so one scan covers wording written
// anywhere in the tree. It caught the spiral overlay's "Turn speed: %s
// (<-/->)", which the manual-only guard had no way to see.
func TestTranslationsHaveNoUnicodeArrows(t *testing.T) {
	arrow := regexp.MustCompile(`[\x{2190}\x{2191}\x{2192}\x{2193}]`)

	paths, err := filepath.Glob(filepath.Join("..", "..", "translations", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) == 0 {
		t.Fatal("no translation catalogues found - this test would pass on an empty set")
	}

	for _, path := range paths {
		name := filepath.Base(path)
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var catalogue map[string]string
		if err := json.Unmarshal(raw, &catalogue); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for key, value := range catalogue {
			if arrow.MatchString(key) {
				t.Errorf("%s has a Unicode arrow in the key %q - write the key name (Left/Right/Up/Down) or ASCII \"->\"", name, key)
			}
			if arrow.MatchString(value) {
				t.Errorf("%s has a Unicode arrow in the translation of %q: %q", name, key, value)
			}
		}
	}
}
