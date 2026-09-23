package explorerpresets_test

import (
	"context"
	"testing"

	"github.com/frathe/picfetch/internal/explorerpresets"
	"github.com/frathe/picfetch/internal/similarity"
)

func TestHEICFormatRulePortability(t *testing.T) {
	for _, format := range []string{"heic", "heif", "jpg", "avif"} {
		t.Run(format, func(t *testing.T) {
			store := explorerpresets.Store{Dir: t.TempDir()}
			preset, err := store.Save(context.Background(), explorerpresets.Preset{
				Name: "Saved " + format + " rule", Rule: explorerpresets.Rule{Format: format},
			})
			if err != nil {
				t.Fatalf("portable format rule refused without a decoder check: %v", err)
			}
			loaded, err := store.Load(context.Background())
			if err != nil || len(loaded) != 1 || loaded[0].ID != preset.ID || loaded[0].Rule.Format != format || !loaded[0].Compatible() {
				t.Fatalf("rule did not survive save/load: %+v, %v", loaded, err)
			}
			item := similarity.Item{Facts: similarity.ImageFacts{Version: similarity.FactsVersion, Format: format}}
			if !loaded[0].Rule.Matches(item) {
				t.Fatal("saved format rule cannot match retained analysis facts")
			}
			item.Facts.Format = "png"
			if loaded[0].Rule.Matches(item) {
				t.Fatal("format rule accepted a different file type")
			}
		})
	}
	for _, format := range []string{"heics", "heifs", "invented"} {
		if err := (explorerpresets.Rule{Format: format}).Validate(); err == nil {
			t.Errorf("unsupported format %q became a valid rule", format)
		}
	}
}
