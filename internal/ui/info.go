// The persistent info overlay (I key). The card's own widgets, text
// formatting, and toggle preference live in internal/ui/infoview; this
// file is the thin viewer-side glue that builds the infoview.State
// snapshot from state.files/zoom/vector and calls the card.

package ui

import "github.com/frathe/picfetch/internal/ui/infoview"

// toggleInfoOverlay is the I key: flips the persistent info card (filename,
// position, pixel dimensions, file size, zoom level) on or off. Modeled on
// the toast card (toast.go) - a pinned overlay rather than a new window -
// but, unlike a toast, it never auto-hides: once on, it stays up across
// navigation and zoom changes until toggled off again.
func (v *viewer) toggleInfoOverlay() {
	if !v.cancelRegionCopyBeforeAction() {
		return
	}
	v.info.Toggle()
	v.syncInfoOverlayVisibility()
	v.ForceRepaint()
	v.syncMenus()
}

// syncInfoOverlayVisibility shows or hides the info card to match its
// standing preference, but only while there's actually an image on screen
// to describe - called both from toggleInfoOverlay (the preference itself
// just changed) and finishLoad (a fresh image just appeared, which the
// still-hidden card needs to be shown for if the preference was already
// on).
//
// The "Show EXIF data" link is settled inside infoview.Card.Sync rather
// than in Update: Sync is the one path that runs when the file on screen
// changes, while updateInfoOverlay also runs on every zoom change - and a
// zoom can't add or remove a file's metadata.
func (v *viewer) syncInfoOverlayVisibility() {
	hasImage := len(v.state.files) > 0 && v.img.Image != nil
	var s infoview.State
	if hasImage {
		s = v.infoState()
	}
	v.info.Sync(hasImage, s)
}

// updateInfoOverlay refreshes the info card's text from current viewer
// state. A no-op whenever the card isn't supposed to be showing anything -
// toggled off, or no image loaded - so internal/ui/zoom can call it as its
// onChanged callback after every zoom change, unconditionally, without
// checking visibility itself first.
func (v *viewer) updateInfoOverlay() {
	if !v.info.Visible() || len(v.state.files) == 0 || v.img.Image == nil {
		return
	}
	v.info.Update(v.infoState())
}

// infoState builds the info card's State snapshot from current viewer
// state - the one function updateInfoOverlay and syncInfoOverlayVisibility
// share, so a zoom-change refresh and a navigation refresh always agree on
// what the card shows. Only safe to call once an image is actually on
// screen; every caller already checks that first.
func (v *viewer) infoState() infoview.State {
	w, h := v.displayedDimensions()
	return infoview.State{
		Name:        v.state.files[v.state.index].Name(),
		Index:       v.state.index,
		Count:       len(v.state.files),
		Width:       w,
		Height:      h,
		ZoomPercent: v.zoom.Percent(),
	}
}
