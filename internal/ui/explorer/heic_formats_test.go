package explorer

import (
	"slices"
	"testing"

	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/explorerpresets"
)

func TestHEICPortableFormatChoices(t *testing.T) {
	form, read := presetRuleFields(explorerpresets.Rule{}, func() {})
	choices := form.Items[1].Widget.(*widget.Select)
	for _, format := range []string{"heic", "heif"} {
		if !slices.Contains(choices.Options, format) {
			t.Fatalf("portable rule form omits %s without a decoder", format)
		}
		choices.SetSelected(format)
		if got := read(); got.Format != format || got.Validate() != nil {
			t.Fatalf("selected portable rule = %+v", got)
		}
	}
}
