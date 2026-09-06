package displays

import (
	"image"
	"regexp"
	"strconv"
	"strings"
)

// Parsing xrandr output lives outside linux.go on purpose. It is pure text
// handling with no X11 dependency, so keeping it clear of that file's cgo
// preamble lets it compile and be tested on any platform - only
// platformInspect, which shells out to xrandr and needs X11 to locate the
// window, has to be Linux-only.

var xrandrLine = regexp.MustCompile(`^(\S+)\s+connected(?:\s+primary)?\s+(\d+)x(\d+)([-+]\d+)([-+]\d+)`)

func parseXRandR(output string) ([]Display, error) {
	var displays []Display
	for line := range strings.SplitSeq(output, "\n") {
		match := xrandrLine.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		width, widthErr := strconv.Atoi(match[2])
		height, heightErr := strconv.Atoi(match[3])
		x, xErr := strconv.Atoi(match[4])
		y, yErr := strconv.Atoi(match[5])
		if widthErr != nil || heightErr != nil || xErr != nil || yErr != nil {
			return nil, &InvalidTopologyError{Reason: "xrandr returned invalid geometry"}
		}
		displays = append(displays, Display{
			ID:     ID(match[1]),
			Name:   match[1],
			Bounds: image.Rect(x, y, x+width, y+height),
		})
	}
	if len(displays) == 0 {
		return nil, &EmptyError{}
	}

	return displays, nil
}
