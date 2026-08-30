// Package appearance defines PicFetch's application-wide appearance modes
// and applies them without replacing Fyne's underlying theme. System mode
// leaves that theme adaptive, while Light and Dark force only the color
// variant so custom fonts, icons, and sizes remain intact.
package appearance

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
)

// Mode selects whether PicFetch follows the system appearance or forces one
// color variant.
type Mode uint8

const (
	System Mode = iota
	Light
	Dark
)

const (
	prefSystem = "system"
	prefLight  = "light"
	prefDark   = "dark"
)

// Modes returns every appearance mode in Settings-picker order.
func Modes() []Mode {
	return []Mode{System, Light, Dark}
}

// DisplayName returns the translated label shown in Settings.
func DisplayName(mode Mode) string {
	switch mode.Normalized() {
	case Light:
		return lang.L("Light")
	case Dark:
		return lang.L("Dark")
	default:
		return lang.L("System default")
	}
}

// Normalized maps values from a newer or corrupt caller to the safe default.
func (mode Mode) Normalized() Mode {
	switch mode {
	case Light, Dark:
		return mode
	default:
		return System
	}
}

// FromPref decodes the stable string stored in Fyne preferences.
func FromPref(value string) Mode {
	switch value {
	case prefLight:
		return Light
	case prefDark:
		return Dark
	default:
		return System
	}
}

// PrefValue returns the stable string written to Fyne preferences.
func (mode Mode) PrefValue() string {
	switch mode.Normalized() {
	case Light:
		return prefLight
	case Dark:
		return prefDark
	default:
		return prefSystem
	}
}

// forcedVariant keeps the application's current theme and overrides only the
// color variant passed to it. In particular, System can unwrap this value and
// recover the exact adaptive or custom theme that was active before forcing a
// mode.
type forcedVariant struct {
	fyne.Theme
	variant fyne.ThemeVariant
}

func (f *forcedVariant) Color(name fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	return f.Theme.Color(name, f.variant)
}

// Apply changes the application's appearance immediately. Returning to
// System restores the original theme rather than manufacturing a replacement,
// so Fyne's operating-system theme watcher can keep changing its variant.
func Apply(application fyne.App, mode Mode) {
	current := application.Settings().Theme()
	base := current
	if forced, ok := current.(*forcedVariant); ok {
		base = forced.Theme
	}
	if base == nil {
		base = theme.DefaultTheme()
	}

	switch mode.Normalized() {
	case Light:
		application.Settings().SetTheme(&forcedVariant{Theme: base, variant: theme.VariantLight})
	case Dark:
		application.Settings().SetTheme(&forcedVariant{Theme: base, variant: theme.VariantDark})
	default:
		if current != base {
			application.Settings().SetTheme(base)
		}
	}
}
