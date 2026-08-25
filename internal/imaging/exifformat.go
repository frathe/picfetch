package imaging

import (
	"fmt"
	"math"
	"time"
)

// formatExposureTime renders a shutter speed in seconds as Exif-style
// display text: "1/200 s" for anything faster than a second (the common
// case), or "2.5 s" for a full second or slower (long exposures).
func formatExposureTime(seconds float64) string {
	if seconds >= 1 {
		return fmt.Sprintf("%.1f s", seconds)
	}

	denominator := math.Round(1 / seconds)

	return fmt.Sprintf("1/%d s", int64(denominator))
}

// formatFocalLength renders a focal length in millimeters, dropping the
// decimal point for the common whole-number case.
func formatFocalLength(mm float64) string {
	if mm == math.Trunc(mm) {
		return fmt.Sprintf("%.0f mm", mm)
	}

	return fmt.Sprintf("%.1f mm", mm)
}

// formatExifDate reformats Exif's "YYYY:MM:DD HH:MM:SS" date/time encoding
// (colons instead of dashes in the date, so it doubles as a valid bare
// filename component on every OS) into the more readable
// "YYYY-MM-DD HH:MM:SS". Anything not matching that exact shape is passed
// through unchanged rather than discarded - still useful to show even if
// this reader doesn't recognize its layout.
func formatExifDate(raw string) string {
	if len(raw) == 19 && raw[4] == ':' && raw[7] == ':' {
		return raw[:4] + "-" + raw[5:7] + "-" + raw[8:]
	}

	return raw
}

// parseExifDateTime parses raw - the same "YYYY:MM:DD HH:MM:SS" Exif
// encoding formatExifDate reformats for display - into a time.Time. ok is
// false for anything not matching that exact layout, mirroring
// formatExifDate's own tolerant fallback (pass the raw string through
// unchanged rather than erroring). Interpreted in the local zone: Exif
// carries no timezone offset in the tags this reader looks at, so that's
// the best available guess, same as most photo software assumes.
func parseExifDateTime(raw string) (time.Time, bool) {
	t, err := time.ParseInLocation("2006:01:02 15:04:05", raw, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}
