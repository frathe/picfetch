package imaging

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"time"
)

// minFrameDelay is substituted for a zero (or negative) frame delay, which
// most GIF encoders use to mean "as fast as possible" but which would
// otherwise spin the UI thread pointlessly.
const minFrameDelay = 100 * time.Millisecond

const (
	maxGIFFrames = 4096
	// Includes frame structs, palette interfaces/colors and growing decoder
	// slices, with headroom. This is conservative admission accounting, not
	// an OS-enforced heap limit.
	gifFrameOverhead        = 8 * 1024
	gifDecodeScratch        = 64 * 1024
	gifPreviewFrameOverhead = 128
)

// IsAnimatedGIF reports whether data describes a GIF with more than one
// image frame. It performs the same bounded structural probe used for
// animation admission and does not decode pixels.
// Comparison calls it to keep first-frame-only records out of the image cache.
//
//goland:noinspection GoUnusedExportedFunction
func IsAnimatedGIF(data []byte) bool {
	count, _, _, ok := probeGIF(data)
	return ok && count > 1
}

// gifWorkingBytes estimates paletted source frames, per-frame storage,
// decoder scratch and two full RGBA compositing canvases. Retained output
// pixels are charged separately, since previews can be smaller than the source.
func gifWorkingBytes(count, w, h int) (int64, bool) {
	if count <= 0 || count > maxGIFFrames || checkDimensions(w, h) != nil {
		return 0, false
	}
	pixels := int64(w) * int64(h)
	return gifDecodeScratch + pixels*int64(count+8) + int64(count)*gifFrameOverhead, true
}

func gifAnimationFits(count, w, h int, budget int64) bool {
	working, ok := gifWorkingBytes(count, w, h)
	if !ok || budget < working {
		return false
	}
	return int64(w)*int64(h)*4 <= (budget-working)/int64(count)
}

// decodeAnimatedGIF checks the complete animation estimate before DecodeAll.
// A refusal returns no frames so the caller can decode only the first frame.
// Zero budget explicitly disables animation without reporting truncation.
func decodeAnimatedGIF(ctx context.Context, data []byte, budget int64) ([]image.Image, []time.Duration, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, false, err
	}
	count, w, h, ok := probeGIF(data)
	if !ok || count <= 1 {
		return nil, nil, false, nil
	}
	if !gifAnimationFits(count, w, h, budget) {
		return nil, nil, budget > 0, nil
	}
	g, err := gif.DecodeAll(ctxReader{ctx: ctx, r: bytes.NewReader(data)})
	if cancelled := ctx.Err(); cancelled != nil {
		return nil, nil, false, cancelled
	}
	if err != nil || len(g.Image) <= 1 {
		return nil, nil, false, nil
	}
	// Retain a check against actual decoder output as well as the block probe.
	if !gifAnimationFits(len(g.Image), g.Config.Width, g.Config.Height, budget) {
		return nil, nil, true, nil
	}
	frames, delays, err := compositeGIFFrames(ctx, g, 0)
	return frames, delays, false, err
}

// compositeGIFFrames is shared by full-size viewing and bounded previews.
// Composite on the logical canvas before scaling so offsets, transparency and
// disposal retain the same meaning at every preview size.
func compositeGIFFrames(ctx context.Context, g *gif.GIF, maxEdge int) ([]image.Image, []time.Duration, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	bounds := image.Rect(0, 0, g.Config.Width, g.Config.Height)
	canvasImg := image.NewRGBA(bounds)

	frames := make([]image.Image, 0, len(g.Image))
	delays := make([]time.Duration, 0, len(g.Image))

	for i, frame := range g.Image {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		disposal := byte(gif.DisposalNone)
		if i < len(g.Disposal) {
			disposal = g.Disposal[i]
		}

		// DisposalPrevious means "after this frame, restore the canvas to
		// how it looked before this frame was drawn", so snapshot now.
		beforeFrame := canvasImg
		if disposal == gif.DisposalPrevious {
			beforeFrame = copyRGBA(canvasImg)
		}

		draw.Draw(canvasImg, frame.Bounds(), frame, frame.Bounds().Min, draw.Over)
		snapshot := scaleToFit(canvasImg, maxEdge)
		if snapshot == canvasImg {
			snapshot = copyRGBA(canvasImg)
		}
		frames = append(frames, snapshot)

		delay := time.Duration(g.Delay[i]) * 10 * time.Millisecond
		if delay <= 0 {
			delay = minFrameDelay
		}
		delays = append(delays, delay)

		switch disposal {
		case gif.DisposalBackground:
			draw.Draw(canvasImg, frame.Bounds(), image.NewUniform(color.Transparent), image.Point{}, draw.Src)
		case gif.DisposalPrevious:
			canvasImg = beforeFrame
		}
	}

	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	return frames, delays, nil
}

// compositeGIFCanvas restores the logical canvas around a partial first
// frame. DecodeConfig reads only the header, so a frozen animation never pays
// for decoding later frames. Its transparent backing matches decodeAnimatedGIF.
func compositeGIFCanvas(data []byte, first image.Image) (image.Image, error) {
	config, err := gif.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if err := checkDimensions(config.Width, config.Height); err != nil {
		return nil, err
	}
	bounds := image.Rect(0, 0, config.Width, config.Height)
	if first.Bounds() == bounds {
		return first, nil
	}
	canvas := image.NewRGBA(bounds)
	draw.Draw(canvas, first.Bounds(), first, first.Bounds().Min, draw.Over)
	return canvas, nil
}

// The three block markers a GIF's top-level grammar can start a block with,
// per the GIF89a spec (sections 15-27).
const (
	gifExtensionIntroducer = 0x21
	gifImageDescriptor     = 0x2C
	gifTrailer             = 0x3B
)

// The extension labels that need handling of their own in skipGIFExtension -
// the ones whose payload is not simply a chain of data sub-blocks.
const (
	gifExtPlainText      = 0x01
	gifExtGraphicControl = 0xF9
	gifExtApplication    = 0xFF
)

// probeGIF walks data's block structure to count its image descriptors and
// read its logical screen size, touching no pixel data at all: every block is
// skipped by the length fields the format already carries, so the cost is a
// walk over the encoded bytes rather than a decode of them. That is what lets
// decodeAnimatedGIF apply its budget before gif.DecodeAll allocates a
// paletted image per frame - the standard library offers no way to learn a
// frame count short of decoding every frame.
//
// ok is false for anything the walk cannot account for byte-for-byte:
// truncated input, an unknown block marker, a sub-block chain running past
// the end. Callers treat that as "not an animation" and fall back to
// image.Decode, which is exactly what already happened when gif.DecodeAll
// returned an error - and it errors on the same structural faults, so the
// fallback path is unchanged for such a file.
//
// Where the two deliberately differ is leniency: this walk skips any
// extension by its sub-block lengths, including labels image/gif rejects
// outright. Erring that way is safe - such a file gets counted, clears the
// budget gate, and is then refused by gif.DecodeAll itself, landing on the
// same static fallback. Erring the other way, by being stricter than the
// decoder, would stop a perfectly readable GIF from animating.
func probeGIF(data []byte) (frames, w, h int, ok bool) {
	// Signature (6 bytes) plus the logical screen descriptor (7).
	if len(data) < 13 {
		return 0, 0, 0, false
	}

	// The same "GIF8?a" pattern image.RegisterFormat matches on, rather than
	// image/gif's own stricter 87a/89a test - see the leniency note above.
	if string(data[:4]) != "GIF8" || data[5] != 'a' {
		return 0, 0, 0, false
	}

	w = int(data[6]) | int(data[7])<<8
	h = int(data[8]) | int(data[9])<<8

	// Bit 7 of the packed field flags a global color table; bits 0-2 give its
	// size as 2^(n+1) entries of three bytes each.
	p := 13
	if packed := data[10]; packed&0x80 != 0 {
		p += 3 << ((packed & 0x07) + 1)
	}

	for {
		if p >= len(data) {
			return 0, 0, 0, false
		}

		switch data[p] {
		case gifTrailer:
			return frames, w, h, true

		case gifExtensionIntroducer:
			if p, ok = skipGIFExtension(data, p); !ok {
				return 0, 0, 0, false
			}

		case gifImageDescriptor:
			// Marker, eight bytes of geometry, then the packed field, whose
			// bit 7 and bits 0-2 describe a local color table exactly as the
			// global one above.
			if p+10 > len(data) {
				return 0, 0, 0, false
			}

			fields := data[p+9]
			p += 10

			if fields&0x80 != 0 {
				p += 3 << ((fields & 0x07) + 1)
			}

			// One byte of LZW minimum code size, then the pixel data - the
			// only bytes here that would cost anything to actually decode,
			// and the whole reason to skip them by length instead.
			if p, ok = skipGIFSubBlocks(data, p+1); !ok {
				return 0, 0, 0, false
			}

			frames++

		default:
			return 0, 0, 0, false
		}
	}
}

// skipGIFExtension advances past the whole extension block starting at p,
// which must be its introducer byte, and reports the offset just past it.
//
// It mirrors image/gif's readExtension rather than assuming every extension
// is a plain chain of data sub-blocks, because three of the four labels
// aren't: a graphic control is a fixed six bytes with no chain at all, and
// plain text and application extensions each carry a fixed-size or
// length-prefixed field *before* their chain. Walking those last two as a
// bare chain ends the extension one block early whenever that leading length
// byte is zero - the decoder reads it as an empty field and then still
// expects the chain - which would report a perfectly readable GIF as
// unparseable, the one direction probeGIF must never err in.
//
// Unknown labels, which image/gif rejects outright, are skipped as a plain
// chain here; see probeGIF on why that leniency is the safe direction.
func skipGIFExtension(data []byte, p int) (int, bool) {
	if p+2 > len(data) {
		return 0, false
	}

	label := data[p+1]
	p += 2

	switch label {
	case gifExtGraphicControl:
		// Block size, four bytes of payload, and its own terminator - read
		// as one fixed run, with no sub-block chain following it.
		if p+6 > len(data) {
			return 0, false
		}

		return p + 6, true

	case gifExtPlainText:
		// Thirteen bytes taken as read, without consulting the block size
		// byte they begin with - exactly what the decoder does.
		p += 13

	case gifExtApplication:
		if p >= len(data) {
			return 0, false
		}

		p += 1 + int(data[p])
	}

	if p > len(data) {
		return 0, false
	}

	return skipGIFSubBlocks(data, p)
}

// skipGIFSubBlocks advances past the sub-block chain starting at p - each
// block a length byte followed by that many bytes of payload, the chain ended
// by a zero length - and reports the offset just past its terminator. ok is
// false if the chain runs off the end of data, which is the one thing a
// length-driven walk cannot recover from.
func skipGIFSubBlocks(data []byte, p int) (int, bool) {
	for {
		if p >= len(data) {
			return 0, false
		}

		n := int(data[p])
		p++

		if n == 0 {
			return p, true
		}

		if p+n > len(data) {
			return 0, false
		}

		p += n
	}
}

func copyRGBA(src *image.RGBA) *image.RGBA {
	dst := image.NewRGBA(src.Bounds())
	copy(dst.Pix, src.Pix)
	return dst
}
