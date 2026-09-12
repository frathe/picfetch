package imaging

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/gif"

	"fyne.io/fyne/v2"
)

// The preview decoder retains paletted native frames and at most two RGBA
// compositing canvases. One caller-owned decode lane bounds concurrent work.
const previewGIFDecodeBytes int64 = 256 << 20
const previewGIFMaxFrames = 4096

// LoadAnimatedPreviewContext returns oriented, display-ready previews. GIFs
// preserve their source timing; other formats retain the static thumbnail path.
// animationBytes bounds retained RGBA pixels; large native decodes fall back to
// a static first frame, with AnimationTruncated set. No preview is upscaled.
func LoadAnimatedPreviewContext(ctx context.Context, u fyne.URI, maxEdge int, animationBytes int64) (*LoadedImage, error) {
	if maxEdge <= 0 {
		return nil, fmt.Errorf("preview edge must be positive: %d", maxEdge)
	}
	data, bounds, err := ReadAndProbe(ctx, u)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	count, w, h, ok := probeGIF(data)
	truncated := false
	if ok && count > 1 && animationBytes > 0 {
		edge, fits := animatedPreviewEdge(count, w, h, maxEdge, animationBytes)
		truncated = !fits
		if fits {
			g, decodeErr := gif.DecodeAll(bytes.NewReader(data))
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if decodeErr == nil && len(g.Image) > 1 {
				// Recheck the decoder's actual output, as full-size GIF decoding does.
				edge, fits = animatedPreviewEdge(len(g.Image), g.Config.Width, g.Config.Height, maxEdge, animationBytes)
				truncated = !fits
				if fits {
					frames, delays, err := compositeGIFFrames(ctx, g, edge)
					if err != nil {
						return nil, err
					}
					if err := ctx.Err(); err != nil {
						return nil, err
					}
					return &LoadedImage{Frames: frames, Delays: delays}, nil
				}
			}
		}
	}
	pixels, err := decodeThumbnailAtEdge(ctx, data, bounds, maxEdge)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return &LoadedImage{Frames: []image.Image{pixels}, AnimationTruncated: truncated}, nil
}

func animatedPreviewEdge(count, w, h, maxEdge int, budget int64) (int, bool) {
	if count <= 0 || count > previewGIFMaxFrames || w <= 0 || h <= 0 || int64(w)*int64(h)*int64(count+8) > previewGIFDecodeBytes {
		return 0, false
	}
	if budget < 4*int64(count) {
		return 0, false
	}
	// Keep the largest edge that fits all composited frames. Integer sizing
	// matters for very narrow images, where the shorter edge stays one pixel.
	low, high := 1, min(maxEdge, max(w, h))
	for low < high {
		mid := low + (high-low+1)/2
		width, height := fitEdge(w, h, mid)
		if int64(width)*int64(height)*4*int64(count) <= budget {
			low = mid
		} else {
			high = mid - 1
		}
	}
	return low, true
}
