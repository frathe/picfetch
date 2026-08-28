package infoview

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/test"
)

// New reads the current theme (theme.Color), so every test in this file
// needs an app to read it from - see internal/ui/widgets/style_test.go for
// the same pattern.
func TestMain(m *testing.M) {
	test.NewApp()
	m.Run()
}

func TestFormatFileSize(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1023, "1023 B"},
		{1024, "1.0 KiB"},
		{1500, "1.5 KiB"},
		{1_048_576, "1.0 MiB"},
		{1_500_000, "1.4 MiB"},
		{1_073_741_824, "1.0 GiB"},
	}

	for _, c := range cases {
		if got := formatFileSize(c.n); got != c.want {
			t.Errorf("formatFileSize(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

func TestToggle_FlipsAndReportsVisible(t *testing.T) {
	c := New(func() {})
	if c.Visible() {
		t.Fatal("new Card should start with the preference off")
	}

	if got := c.Toggle(); !got || !c.Visible() {
		t.Errorf("Toggle() = %v, want true and Visible() true", got)
	}
	if got := c.Toggle(); got || c.Visible() {
		t.Errorf("Toggle() = %v, want false and Visible() false", got)
	}
}

func TestUpdate_NoopWhileNotVisible(t *testing.T) {
	c := New(func() {})
	c.SetFile(100, false, false)

	c.Update(State{Name: "a.jpg", Count: 1, Width: 4, Height: 4, ZoomPercent: 100})

	if c.Text().Text != "" {
		t.Errorf("text = %q, want Update to no-op while the card is toggled off", c.Text().Text)
	}
}

func TestUpdate_PreviewSuffix(t *testing.T) {
	c := New(func() {})
	c.Toggle()
	c.SetFile(100, false, true)

	c.Update(State{Name: "photo.nef", Count: 1, Width: 16, Height: 8, ZoomPercent: 100})

	if !strings.Contains(c.Text().Text, lang.L("(preview)")) {
		t.Errorf("text = %q, want it to contain %q", c.Text().Text, lang.L("(preview)"))
	}
}

func TestUpdate_NoPreviewSuffixWhenNotAPreview(t *testing.T) {
	c := New(func() {})
	c.Toggle()
	c.SetFile(100, false, false)

	c.Update(State{Name: "a.jpg", Count: 1, Width: 4, Height: 4, ZoomPercent: 100})

	if strings.Contains(c.Text().Text, lang.L("(preview)")) {
		t.Errorf("text = %q, should not contain %q", c.Text().Text, lang.L("(preview)"))
	}
}

// TestUpdate_PositionSuffixOnlyWhenMoreThanOneFile covers the (i/n) suffix
// boundary directly: absent at Count==1, present and 1-indexed from
// Count==2 up.
func TestUpdate_PositionSuffixOnlyWhenMoreThanOneFile(t *testing.T) {
	c := New(func() {})
	c.Toggle()
	c.SetFile(100, false, false)

	c.Update(State{Name: "a.jpg", Index: 0, Count: 1, Width: 4, Height: 4, ZoomPercent: 100})
	if strings.Contains(c.Text().Text, "(1/1)") {
		t.Errorf("text = %q, a single file should carry no position suffix", c.Text().Text)
	}

	c.Update(State{Name: "a.jpg", Index: 0, Count: 2, Width: 4, Height: 4, ZoomPercent: 100})
	if !strings.Contains(c.Text().Text, "(1/2)") {
		t.Errorf("text = %q, want it to contain %q", c.Text().Text, "(1/2)")
	}

	c.Update(State{Name: "a.jpg", Index: 1, Count: 2, Width: 4, Height: 4, ZoomPercent: 100})
	if !strings.Contains(c.Text().Text, "(2/2)") {
		t.Errorf("text = %q, want it to contain %q", c.Text().Text, "(2/2)")
	}
}

func TestUpdate_RendersDimensionsAndZoom(t *testing.T) {
	c := New(func() {})
	c.Toggle()
	c.SetFile(1024, false, false)

	c.Update(State{Name: "a.jpg", Count: 1, Width: 40, Height: 20, ZoomPercent: 150})

	got := c.Text().Text
	for _, part := range []string{"a.jpg", "40 x 20", "1.0 KiB", "150%"} {
		if !strings.Contains(got, part) {
			t.Errorf("text = %q, want it to contain %q", got, part)
		}
	}
}

func TestSync_HidesCardWhenNoImage(t *testing.T) {
	c := New(func() {})
	c.Toggle()

	c.Sync(false, State{})

	if c.Object().Visible() {
		t.Error("card should stay hidden with no image on screen, even with the preference on")
	}
}

func TestSync_HidesCardWhenPreferenceOff(t *testing.T) {
	c := New(func() {})

	c.Sync(true, State{Name: "a.jpg", Count: 1})

	if c.Object().Visible() {
		t.Error("card should stay hidden while the preference is off")
	}
}

func TestSync_ShowsCardAndRefreshesTextWhenBothTrue(t *testing.T) {
	c := New(func() {})
	c.Toggle()
	c.SetFile(1024, false, false)

	c.Sync(true, State{Name: "a.jpg", Count: 1, Width: 4, Height: 4, ZoomPercent: 100})

	if !c.Object().Visible() {
		t.Error("card should show once both the preference and hasImage are true")
	}
	if !strings.HasPrefix(c.Text().Text, "a.jpg\n") {
		t.Errorf("text = %q, want it to start with the file name", c.Text().Text)
	}
}

// TestSync_ExifLinkFollowsHasEXIF is the exifLink show/hide rule: settled
// in Sync from whichever facts SetFile last recorded, not in Update.
func TestSync_ExifLinkFollowsHasEXIF(t *testing.T) {
	c := New(func() {})
	c.Toggle()

	c.SetFile(100, true, false)
	c.Sync(true, State{Name: "a.jpg", Count: 1})
	if !c.ExifLink().Visible() {
		t.Error("exifLink should show for a file that has EXIF metadata")
	}

	c.SetFile(100, false, false)
	c.Sync(true, State{Name: "a.jpg", Count: 1})
	if c.ExifLink().Visible() {
		t.Error("exifLink should hide for a file with no EXIF metadata")
	}
}

func TestAfterMetadataRemoved_ClearsEXIFAndUpdatesSizeWhenKnown(t *testing.T) {
	c := New(func() {})
	c.SetFile(1000, true, false)

	c.AfterMetadataRemoved(800, true)

	c.Toggle()
	c.Sync(true, State{Name: "a.jpg", Count: 1, Width: 4, Height: 4, ZoomPercent: 100})
	if c.ExifLink().Visible() {
		t.Error("exifLink should hide once EXIF has been stripped")
	}
	if !strings.Contains(c.Text().Text, "800 B") {
		t.Errorf("text = %q, want the new (smaller) size", c.Text().Text)
	}
}

func TestAfterMetadataRemoved_KeepsOldSizeWhenUnknown(t *testing.T) {
	c := New(func() {})
	c.SetFile(1000, true, false)

	c.AfterMetadataRemoved(0, false)

	c.Toggle()
	c.Sync(true, State{Name: "a.jpg", Count: 1, Width: 4, Height: 4, ZoomPercent: 100})
	if c.ExifLink().Visible() {
		t.Error("exifLink should hide regardless of whether the new size was known")
	}
	if !strings.Contains(c.Text().Text, "1000 B") {
		t.Errorf("text = %q, want the old size preserved when the new one couldn't be read", c.Text().Text)
	}
}

func TestFileSizeAndHasEXIF_ReflectLastSetFile(t *testing.T) {
	c := New(func() {})

	if got := c.FileSize(); got != 0 {
		t.Errorf("FileSize() on a fresh Card = %d, want 0", got)
	}
	if c.HasEXIF() {
		t.Error("HasEXIF() on a fresh Card should be false")
	}

	c.SetFile(1234, true, false)
	if got := c.FileSize(); got != 1234 {
		t.Errorf("FileSize() = %d, want 1234", got)
	}
	if !c.HasEXIF() {
		t.Error("HasEXIF() should be true after SetFile(_, true, _)")
	}
}

func TestOnShowExif_FiresOnTap(t *testing.T) {
	var fired bool
	c := New(func() { fired = true })

	c.ExifLink().OnTapped()

	if !fired {
		t.Error("tapping exifLink should call onShowExif")
	}
}
