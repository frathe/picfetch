package ui

import "fyne.io/fyne/v2"

// tunnelSources freezes main order and the installed duplicate visibility.
// Unknown groups stay eligible; opening the egg starts no analysis.
func (v *viewer) tunnelSources() []fyne.URI {
	visibility := v.dupes.Visibility()
	sources := make([]fyne.URI, 0, len(v.state.files))
	for i, uri := range v.state.files {
		if uri != nil && visibility.Visible(i) {
			sources = append(sources, uri)
		}
	}
	return sources
}

func (v *viewer) openSpiral() {
	if v.stopping {
		return
	}
	var sources []fyne.URI
	if !v.spiral.Open() {
		sources = v.tunnelSources()
	}
	v.spiral.Show(sources)
}

func (v *viewer) openSpiralForGesture(clockwise bool) {
	if v.stopping {
		return
	}
	var sources []fyne.URI
	if !v.spiral.Open() {
		sources = v.tunnelSources()
	}
	v.spiral.ShowForGesture(clockwise, sources)
}
