package appearance

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

func TestModePreferenceRoundTripAndFallback(t *testing.T) {
	for _, mode := range Modes() {
		if got := FromPref(mode.PrefValue()); got != mode {
			t.Errorf("FromPref(%q) = %v, want %v", mode.PrefValue(), got, mode)
		}
	}
	if got := FromPref("future-mode"); got != System {
		t.Errorf("FromPref(unknown) = %v, want System", got)
	}
	if got := Mode(255).Normalized(); got != System {
		t.Errorf("invalid mode normalized to %v, want System", got)
	}
}

func TestApplyForcesVariantsAndRestoresSystemTheme(t *testing.T) {
	application := test.NewApp()
	base := &variantTheme{}
	application.Settings().SetTheme(base)

	Apply(application, Light)
	if got := application.Settings().Theme().Color(theme.ColorNameBackground, theme.VariantDark); got != color.White {
		t.Errorf("Light background = %v, want white", got)
	}

	Apply(application, Dark)
	if got := application.Settings().Theme().Color(theme.ColorNameBackground, theme.VariantLight); got != color.Black {
		t.Errorf("Dark background = %v, want black", got)
	}

	Apply(application, System)
	if got := application.Settings().Theme(); got != base {
		t.Errorf("System theme = %T %p, want original %T %p", got, got, base, base)
	}
}

type variantTheme struct{}

func (*variantTheme) Color(_ fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if variant == theme.VariantLight {
		return color.White
	}
	return color.Black
}

func (*variantTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (*variantTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (*variantTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
