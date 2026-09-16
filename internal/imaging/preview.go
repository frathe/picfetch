package imaging

import (
	"bytes"
	"context"
	"image"
	"image/gif"

	"fyne.io/fyne/v2"
)

// The preview decoder retains paletted native frames and at most two RGBA
// compositing canvases. One caller-owned decode lane bounds concurrent work.
const previewGIFDecodeBytes int64 = 256 * 1024 * 1024

// LoadAnimatedPreviewContext returns oriented, display-ready previews. GIFs
// preserve their source timing; other formats retain the static thumbnail path.
// animationBytes bounds retained RGBA pixels and frame objects; large source decodes fall back to
// a static first frame, with AnimationTruncated set. No preview is upscaled.
func LoadAnimatedPreviewContext(ctx context.Context, u fyne.URI, maxEdge int, animationBytes int64) (*LoadedImage, error) {
	return (Reader{}).Preview(ctx, u, maxEdge, animationBytes)
}

func decodePreview(ctx context.Context, data []byte, bounds image.Rectangle, maxEdge int, animationBytes int64) (*LoadedImage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	count, w, h, ok := probeGIF(data)
	truncated := false
	if ok && count > 1 && animationBytes > 0 {
		edge, fits := animatedPreviewEdge(count, w, h, maxEdge, animationBytes)
		truncated = !fits
		if fits {
			g, decodeErr := gif.DecodeAll(ctxReader{ctx: ctx, r: bytes.NewReader(data)})
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
	working, ok := gifWorkingBytes(count, w, h)
	if !ok || working > previewGIFDecodeBytes || maxEdge <= 0 {
		return 0, false
	}
	if budget < int64(count)*(gifPreviewFrameOverhead+4) {
		return 0, false
	}
	budget -= int64(count) * gifPreviewFrameOverhead
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
