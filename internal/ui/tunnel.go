package ui

import "fyne.io/fyne/v2"

// tunnelSources freezes main order and the installed duplicate visibility.
// Unknown groups stay eligible; opening the egg starts no analysis.
func (v *viewer) tunnelSources() []fyne.URI {
	visibility := v.dupes.Visibility()
	collection := v.state.Observe()
	sources := make([]fyne.URI, 0, collection.Count())
	for i := range collection.Count() {
		uri := collection.FileAt(i)
		if uri != nil && visibility.Visible(i) {
			sources = append(sources, uri)
		}
	}
	return sources
}

// openSpiral serves the manual's owned navigation without applying the
// main window's admission or yielding its active region selection.
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
	if _, ok := v.admitCommand(commandRequest{command: commandSpiral}); !ok {
		return
	}
	var sources []fyne.URI
	if !v.spiral.Open() {
		sources = v.tunnelSources()
	}
	v.spiral.ShowForGesture(clockwise, sources)
}
