package imaging

import (
	"testing"
	"time"
)

func TestFormatExposureTime(t *testing.T) {
	cases := []struct {
		seconds float64
		want    string
	}{
		{1.0 / 200, "1/200 s"},
		{1.0 / 4000, "1/4000 s"},
		{2.5, "2.5 s"},
		{1, "1.0 s"},
	}

	for _, c := range cases {
		if got := formatExposureTime(c.seconds); got != c.want {
			t.Errorf("formatExposureTime(%v) = %q, want %q", c.seconds, got, c.want)
		}
	}
}

func TestFormatFocalLength(t *testing.T) {
	cases := []struct {
		mm   float64
		want string
	}{
		{50, "50 mm"},
		{18.5, "18.5 mm"},
	}

	for _, c := range cases {
		if got := formatFocalLength(c.mm); got != c.want {
			t.Errorf("formatFocalLength(%v) = %q, want %q", c.mm, got, c.want)
		}
	}
}

func TestFormatExifDate(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"2024:08:12 14:33:02", "2024-08-12 14:33:02"},
		{"garbage", "garbage"},
	}

	for _, c := range cases {
		if got := formatExifDate(c.raw); got != c.want {
			t.Errorf("formatExifDate(%q) = %q, want %q", c.raw, got, c.want)
		}
	}
}

func TestParseExifDateTime(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		got, ok := parseExifDateTime("2024:08:12 14:33:02")
		if !ok {
			t.Fatal("ok = false, want true")
		}
		want := time.Date(2024, 8, 12, 14, 33, 2, 0, time.Local)
		if !got.Equal(want) {
			t.Errorf("parseExifDateTime() = %v, want %v", got, want)
		}
	})

	t.Run("malformed", func(t *testing.T) {
		if _, ok := parseExifDateTime("garbage"); ok {
			t.Error("ok = true, want false for a malformed date")
		}
	})
}
