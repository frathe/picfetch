package imaging

import (
	"context"
	"image"

	"golang.org/x/image/draw"

	"fyne.io/fyne/v2"
)

// ThumbnailSize is the maximum length, in pixels, of a generated
// thumbnail's longer edge; the shorter edge is scaled to preserve the
// source image's aspect ratio.
const ThumbnailSize = 200

// DefaultThumbCacheBytes is the shipped byte budget for NewThumbCache,
// until the settings window (internal/ui/settingswin) changes it. Each
// entry is capped to ThumbnailSize on its long edge, so 256 MB covers on
// the order of 1600 thumbnails - enough for a large drop's whole file set
// to stay warm while scrolling the grid, without the 625 MB an
// entry-bounded 4096 could quietly reach.
const DefaultThumbCacheBytes = 256 << 20

// NewThumbCache builds the byte-bounded cache callers use to hold generated
// thumbnails, keyed by URI string - a separate cache and budget from
// NewImgCache's full-size decodes, so populating it (e.g. from the grid
// overview) can never evict a full decode the normal viewing path still
// needs, or vice versa.
func NewThumbCache(budget int64) *ByteCache[image.Image] {
	return NewByteCache(budget, imageBytes)
}

// LoadThumbnailAndBounds is LoadThumbnail plus the file's EXIF-oriented
// display size from ReadAndProbe (SVG logical size, RAW preview size).
// Callers that need a representative by resolution use native, not
// thumb.Bounds - generated thumbs are capped at ThumbnailSize.
func LoadThumbnailAndBounds(u fyne.URI) (image.Image, image.Rectangle, error) {
	data, bounds, err := ReadAndProbe(context.Background(), u)
	if err != nil {
		return nil, image.Rectangle{}, err
	}

	// An SVG has no fixed pixels, so rather than rasterizing at full
	// logical size only for scaleToFit to discard nearly all of it,
	// rasterize straight at the thumbnail's own size - bounds is the
	// logical size here, so the aspect comes out identical. The Vector is
	// ephemeral on purpose: one raster, then discarded (see vector.go's
	// note on why thumbnails never share the display path's Vector).
	if isSVGData(data) {
		vec, err := ParseVector(data)
		if err != nil {
			return nil, image.Rectangle{}, err
		}
		w, h := fitEdge(bounds.Dx(), bounds.Dy(), ThumbnailSize)
		thumb, err := vec.RasterAt(w, h)
		if err != nil {
			return nil, image.Rectangle{}, err
		}
		return thumb, bounds, nil
	}

	loaded, err := DecodeLoaded(context.Background(), data, 0)
	if err != nil {
		return nil, image.Rectangle{}, err
	}
	return scaleToFit(loaded.Frames[0], ThumbnailSize), bounds, nil
}

// LoadThumbnail reads and decodes u exactly like LoadImage - full EXIF
// orientation correction included - then downsamples the first frame
// (animated GIFs show only their first frame here, same as every other
// still context in this app) to fit within ThumbnailSize on its longer
// edge. An SVG skips the decode-then-downsample round trip entirely and
// rasterizes straight at the thumbnail's size.
//
// The zero animation budget is what makes that "first frame only" literal:
// without it a long animation composited every one of its frames to a full
// RGBA canvas so this could keep one and discard the rest, which for a
// large GIF meant gigabytes of allocation per grid cell.
func LoadThumbnail(u fyne.URI) (image.Image, error) {
	thumb, _, err := LoadThumbnailAndBounds(u)
	return thumb, err
}

// scaleToFit downsamples src to fit within maxEdge x maxEdge, preserving
// aspect ratio; src is returned unchanged if it's already within bounds,
// rather than upscaled. Uses ApproxBiLinear rather than the higher-quality
// (but much slower on a large downscale) BiLinear kernel - draw.Interpolator's
// own doc recommends it as the speed/quality tradeoff for exactly this case,
// and a small preview doesn't need kernel-grade sharpness.
//
// ScaleForExport is the same shape at the other end of that tradeoff. The
// two are deliberately separate functions rather than one taking a kernel:
// nothing populating a thumbnail cache should ever be one argument away
// from paying export-grade sampling costs per cell.
func scaleToFit(src image.Image, maxEdge int) image.Image {
	return scaleWith(draw.ApproxBiLinear, src, maxEdge)
}

// ScaleForExport applies an export size limit of maxEdge to src: its longest
// edge is brought to that ceiling, the aspect ratio is preserved, and an
// image already inside the ceiling is returned as itself rather than
// enlarged to meet it. A maxEdge of zero or less is no ceiling at all, so an
// export at Original size never pays for a resampling pass that would change
// nothing.
//
// CatmullRom rather than the thumbnail path's ApproxBiLinear: this is a
// photo somebody is about to mail or post, where the approximate kernel's
// softness is visible, and it runs once per export rather than once per
// grid cell.
func ScaleForExport(src image.Image, maxEdge int) image.Image {
	return scaleWith(draw.CatmullRom, src, maxEdge)
}

// SizeLimitApplies reports whether an export size limit of maxEdge would
// actually change an image whose bounds are b - false both for no ceiling at
// all and for a ceiling the image is already inside, since neither enlarges
// anything.
//
// This is the one place that question is answered. internal/ui asks it
// before an export to decide whether the size belongs in the suggested
// filename and the completion toast, and Export asks it to decide whether
// the source's dimension tags have been made false; both read the same rule
// ScaleForExport itself follows, so no caller can drift from it.
func SizeLimitApplies(b image.Rectangle, maxEdge int) bool {
	w, h := fitEdge(b.Dx(), b.Dy(), maxEdge)

	return w != b.Dx() || h != b.Dy()
}

// scaleWith is what scaleToFit and ScaleForExport share: the fitting rule,
// the never-upscale check, and the destination image. Only the kernel
// differs, and it is the caller's whole reason for existing.
func scaleWith(kernel draw.Interpolator, src image.Image, maxEdge int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()

	dw, dh := fitEdge(w, h, maxEdge)
	if w <= 0 || h <= 0 || (dw == w && dh == h) {
		return src
	}

	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	kernel.Scale(dst, dst.Bounds(), src, b, draw.Src, nil)
	return dst
}

// fitEdge is the size the scalers above scale to, separated so
// LoadThumbnail's SVG branch can aim a rasterization at it directly and so
// SizeLimitApplies can ask what a ceiling would do without doing it: w x h
// reduced to fit within maxEdge on the longer side, aspect preserved, never
// upscaled. A maxEdge of zero or less is no ceiling, and returns w x h
// unchanged - "Original" in the export prompt's terms, and the answer that
// keeps a caller from having to special-case it.
func fitEdge(w, h, maxEdge int) (int, int) {
	if maxEdge <= 0 || (w <= maxEdge && h <= maxEdge) {
		return w, h
	}

	dw, dh := maxEdge, h*maxEdge/w
	if h > w {
		dw, dh = w*maxEdge/h, maxEdge
	}

	return max(dw, 1), max(dh, 1)
}
