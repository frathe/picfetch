package ui

import "github.com/frathe/picfetch/internal/appearance"

// ThemeMode reports the application-wide appearance selected in Settings.
func (v *viewer) ThemeMode() appearance.Mode {
	return v.settings.themeMode
}

// SetThemeMode applies an appearance immediately and keeps the normalized
// value for shutdown persistence.
func (v *viewer) SetThemeMode(mode appearance.Mode) {
	mode = mode.Normalized()
	v.settings.themeMode = mode
	appearance.Apply(v.app, mode)
}
