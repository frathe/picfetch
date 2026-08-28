// Package infoview owns the persistent info overlay (the I key, see
// internal/ui/info.go's toggleInfoOverlay): its three widgets - the text
// label, the "Show EXIF data" link, and the card container - the current
// file's raw facts (byte size, EXIF presence, RAW-preview flag), and its
// own toggle preference and text formatting. It has no window of its own
// and reads nothing outside itself: internal/ui builds the State snapshot
// Update renders from, since only it has state.files/zoom/vector to read.
package infoview

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/ui/widgets"
)

// State is the snapshot Update renders: the position/name info gathered
// from state.files, and the pixel dimensions and zoom level gathered from
// vector/zoom - none of which this package has access to. Built by
// internal/ui's viewer.infoState, the one function every caller shares.
type State struct {
	Name          string
	Index, Count  int
	Width, Height int
	ZoomPercent   int
}

// Card is the persistent info overlay - unlike the toast (internal/ui's
// toast.go) it never auto-hides itself, and it's several distinct lines
// rather than one centered message, so it uses the theme's own
// overlay-background/foreground pairing (the same one dialogs use) instead
// of the toast's fixed, deliberately loud warning colors - legible in both
// light and dark themes without hardcoding either.
//
// visible is the standing I-key preference: it survives navigation and
// drops the same way sortMode/mergeMode do, independent of whether the
// card widgets are actually shown right now - see Sync. fileSize/hasEXIF/
// preview are the current file's raw facts, carried on
// imaging.LoadedImage and handed in through SetFile so a cache hit gets
// them too.
type Card struct {
	visible bool

	text     *widget.Label
	exifLink *widget.Hyperlink
	card     *fyne.Container

	fileSize int64
	hasEXIF  bool
	preview  bool
}

// New builds the info card. onShowExif backs the "Show EXIF data" link
// right below the card's own text (the click equivalent of the E key, see
// internal/ui/exifwin); like the dropzone's own tap callbacks, it only
// ever runs on a later tap, so it may close over a not-yet-assigned viewer
// variable.
func New(onShowExif func()) *Card {
	bg := canvas.NewRectangle(theme.Color(theme.ColorNameOverlayBackground))
	bg.CornerRadius = widgets.CardRadius
	text := widget.NewLabel("")
	text.Alignment = fyne.TextAlignLeading

	exifLink := widget.NewHyperlink(lang.L("Show EXIF data"), nil)
	exifLink.OnTapped = onShowExif

	card := container.NewStack(bg, container.NewPadded(container.NewVBox(text, exifLink)))
	card.Hide()

	return &Card{text: text, exifLink: exifLink, card: card}
}

// Object hands the card's container to build.go's overlay stack.
func (c *Card) Object() fyne.CanvasObject { return c.card }

// Text is the card's own text label, exposed so callers can read its
// rendered content directly - the same "reach through an exported
// accessor" shape internal/ui/menus uses for its menu items.
func (c *Card) Text() *widget.Label { return c.text }

// ExifLink is the "Show EXIF data" link inside the card, exposed so
// callers can drive its OnTapped and read its visibility directly.
func (c *Card) ExifLink() *widget.Hyperlink { return c.exifLink }

// FileSize is the current file's raw byte count, as last recorded by
// SetFile or AfterMetadataRemoved.
func (c *Card) FileSize() int64 { return c.fileSize }

// HasEXIF reports whether the current file carries EXIF metadata, as last
// recorded by SetFile or AfterMetadataRemoved.
func (c *Card) HasEXIF() bool { return c.hasEXIF }

// SetFile records the current file's raw facts - its byte size, whether it
// carries EXIF metadata, and whether it's a RAW's embedded preview -
// called once a file is decoded (a cache hit carries the same facts on
// imaging.LoadedImage, so this covers both paths).
func (c *Card) SetFile(fileSize int64, hasEXIF, preview bool) {
	c.fileSize = fileSize
	c.hasEXIF = hasEXIF
	c.preview = preview
}

// AfterMetadataRemoved records that the file currently on screen just lost
// its EXIF data (an EXIF strip via exifwin) and, if its new size could be
// read from disk, that new byte count - sizeKnown false leaves fileSize
// untouched. One call expresses the whole "same file, new size, no EXIF
// now" transition rather than a setter per field. The preview flag is
// left alone: stripping metadata can't turn a RAW preview into something
// else.
func (c *Card) AfterMetadataRemoved(fileSize int64, sizeKnown bool) {
	if sizeKnown {
		c.fileSize = fileSize
	}
	c.hasEXIF = false
}

// Toggle flips the standing I-key preference and reports its new value.
func (c *Card) Toggle() bool {
	c.visible = !c.visible
	return c.visible
}

// Visible reports the standing I-key preference - not whether the card
// widgets are actually on screen right now, which a caller reads off
// Object() instead (fyne.CanvasObject's own Visible()).
func (c *Card) Visible() bool { return c.visible }

// Sync shows or hides the card to match the preference, but only while
// there's actually an image on screen to describe (hasImage) - called
// both when the preference itself just changed and when a fresh image
// just appeared, which the still-hidden card needs to be shown for if the
// preference was already on. Refreshes the card's text via s before
// showing it, so a toggle-on never briefly displays whatever text it last
// held.
//
// The "Show EXIF data" link is settled here too, rather than in Update:
// this is the one path that runs when the file on screen changes, while
// Update also runs on every zoom change - and a zoom can't add or remove
// a file's metadata.
func (c *Card) Sync(hasImage bool, s State) {
	if c.visible && hasImage {
		c.Update(s)
		if c.hasEXIF {
			c.exifLink.Show()
		} else {
			c.exifLink.Hide()
		}
		c.card.Show()
	} else {
		c.card.Hide()
	}
}

// Update refreshes the card's text from s. A no-op whenever the card
// isn't currently toggled on - mirroring Sync's own guard - so
// internal/ui's updateInfoOverlay (internal/ui/zoom's onChanged callback)
// can call it after every zoom change, unconditionally, without checking
// visibility itself first.
func (c *Card) Update(s State) {
	if !c.visible {
		return
	}

	name := s.Name
	if c.preview {
		name += " " + lang.L("(preview)")
	}
	if s.Count > 1 {
		name = fmt.Sprintf("%s  (%d/%d)", name, s.Index+1, s.Count)
	}

	lines := []string{
		name,
		fmt.Sprintf("%d x %d", s.Width, s.Height),
		formatFileSize(c.fileSize),
		fmt.Sprintf(lang.L("Zoom: %d%%"), s.ZoomPercent),
	}
	c.text.SetText(strings.Join(lines, "\n"))
}

// formatFileSize renders n bytes as a short human-readable size (e.g.
// "2.3 MiB"), matching the binary (1024-based) units most OS file browsers
// use for a single file's size, rather than SI (1000-based) ones. The
// example says MiB, not MB, because that is what the format string below
// actually emits - the unit letter comes from "KMGTPE" with an "iB"
// suffix.
func formatFileSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}

	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
