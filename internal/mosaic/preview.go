package mosaic

import (
	"image"
	"image/draw"
	"time"

	xdraw "golang.org/x/image/draw"
)

const previewInterval = 250 * time.Millisecond

// previewSnapshots samples the two live layers only between completed
// placements on their owning worker. Each published image has its own pixels;
// neither the renderer nor a later snapshot can change it.
type previewSnapshots struct {
	clock func() time.Time
	last  time.Time
}

func (p *previewSnapshots) next(background, primary *image.NRGBA) image.Image {
	clock := p.clock
	if clock == nil {
		clock = time.Now
	}
	now := clock()
	if !p.last.IsZero() && now.Sub(p.last) < previewInterval {
		return nil
	}
	p.last = now
	return previewComposite(background, primary)
}

// Scaling each layer directly into the snapshot avoids a full-size composite
// copy and bounds allocation/work independently of the output dimensions.
// RGBA selects the resampler's allocation-free pixel loops.
// This filter affects only the transient preview, never the finished mosaic.
func previewComposite(background, primary *image.NRGBA) *image.RGBA {
	size := background.Bounds().Size()
	if longest := max(size.X, size.Y); longest > 960 {
		size = image.Pt(max(1, size.X*960/longest), max(1, size.Y*960/longest))
	}
	preview := image.NewRGBA(image.Rectangle{Max: size})
	xdraw.ApproxBiLinear.Scale(preview, preview.Bounds(), background, background.Bounds(), draw.Src, nil)
	xdraw.ApproxBiLinear.Scale(preview, preview.Bounds(), primary, primary.Bounds(), draw.Over, nil)
	return preview
}
