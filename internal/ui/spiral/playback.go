package spiral

import (
	"image"
	"sort"
	"time"

	"github.com/frathe/picfetch/internal/imaging"
)

type tunnelPreview struct {
	source   int
	frames   []image.Image
	ends     []time.Duration
	duration time.Duration
}

func newTunnelPreview(source int, loaded *imaging.LoadedImage) *tunnelPreview {
	p := &tunnelPreview{source: source, frames: loaded.Frames}
	for _, delay := range loaded.Delays {
		p.duration += delay
		p.ends = append(p.ends, p.duration)
	}
	return p
}

// Each flight owns its playback origin. The existing UI frame clock selects
// the current GIF frame directly, so stalls never queue a catch-up burst.
type tunnelPlayback struct {
	preview *tunnelPreview
	born    time.Duration
	frame   int
}

func (p *tunnelPlayback) advance(elapsed time.Duration) (image.Image, bool) {
	if p.preview == nil || p.preview.duration == 0 {
		return nil, false
	}
	phase := max(0, elapsed-p.born) % p.preview.duration
	frame := sort.Search(len(p.preview.ends), func(i int) bool { return p.preview.ends[i] > phase })
	if frame == p.frame {
		return nil, false
	}
	p.frame = frame
	return p.preview.frames[frame], true
}
