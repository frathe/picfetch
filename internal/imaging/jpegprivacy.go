package imaging

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"image"
	"image/jpeg"
)

// JPEGMetadataState describes the result of inspecting the complete primary JPEG.
type JPEGMetadataState uint8

const (
	JPEGMetadataUnsupported JPEGMetadataState = iota
	JPEGMetadataClean
	JPEGMetadataRemovable
)

// JPEGMetadataInspection never equates an inspection failure with a clean file.
type JPEGMetadataInspection struct {
	State JPEGMetadataState
	Err   error
}

// ErrJPEGMetadataNotJPEG lets the EXIF window distinguish non-JPEG sources.
// Qodana's differential analysis misses those cross-package references.
//
//goland:noinspection GoUnusedGlobalVariable
var ErrJPEGMetadataNotJPEG = errNotJPEG

var (
	ErrJPEGMetadataStructure   = errors.New("JPEG structure is incomplete or invalid")
	ErrJPEGMetadataProcess     = errors.New("JPEG process or color interpretation is not qualified for metadata removal")
	ErrJPEGMetadataProfile     = errors.New("ICC profile is not qualified for metadata removal")
	ErrJPEGMetadataOrientation = errors.New("JPEG orientation is ambiguous or invalid")
)

// InspectJPEGMetadata uses the same cancellable policy as the mutation operation.
// Call it on a worker: validation includes decoding the bounded primary image.
func InspectJPEGMetadata(ctx context.Context, data []byte) JPEGMetadataInspection {
	p, err := prepareJPEGRemoval(ctx, data)
	if err != nil {
		return JPEGMetadataInspection{State: JPEGMetadataUnsupported, Err: err}
	}
	state := JPEGMetadataClean
	if !bytes.Equal(data, p.output) || p.orientation != 1 {
		state = JPEGMetadataRemovable
	}
	return JPEGMetadataInspection{State: state}
}

type jpegRemoval struct {
	output      []byte
	jfif        []byte
	orientation int
	components  int
	pixels      image.Image
}

// prepareJPEGRemoval deliberately does not change the tolerant header reader used
// by viewing and metadata-preserving export. Only this operation promises removal.
func prepareJPEGRemoval(ctx context.Context, data []byte) (jpegRemoval, error) {
	p := jpegRemoval{orientation: 1}
	if err := ctx.Err(); err != nil {
		return p, err
	}
	if len(data) < 4 || !bytes.Equal(data[:2], []byte{0xff, 0xd8}) {
		return p, errNotJPEG
	}
	if int64(len(data)) > MaxEncodedBytes() {
		return p, &InputTooLargeError{limit: MaxEncodedBytes()}
	}
	p.output = []byte{0xff, 0xd8}
	var frame, scan, jfif, adobe, exif bool
	adobeTransform := byte(0xff)
	var policy jpegScanPolicy
	var profile removalICC
	for pos := 2; pos < len(data); {
		if err := ctx.Err(); err != nil {
			return p, err
		}
		start := pos
		if data[pos] != 0xff {
			return p, ErrJPEGMetadataStructure
		}
		for pos < len(data) && data[pos] == 0xff {
			pos++
		}
		if pos == len(data) {
			return p, ErrJPEGMetadataStructure
		}
		marker := data[pos]
		pos++
		if marker == 0xd9 {
			if !frame || !scan || !policy.complete() {
				return p, ErrJPEGMetadataStructure
			}
			if adobe && ((jfif && adobeTransform != 1) || (p.components == 1 && adobeTransform != 0)) {
				return p, ErrJPEGMetadataProcess
			}
			if bytes.Equal(policy.ids, []byte("RGB")) && (jfif || adobe && adobeTransform != 0) {
				return p, ErrJPEGMetadataProcess
			}
			p.output = append(p.output, 0xff, 0xd9)
			segments, err := profile.normalized(ctx, p.components)
			if err != nil {
				return p, err
			}
			if len(segments) > 0 {
				// Keep a leading JFIF directly after SOI.
				at := 2
				if len(p.output) > 6 && p.output[2] == 0xff && p.output[3] == 0xe0 {
					at += 2 + int(binary.BigEndian.Uint16(p.output[4:6]))
				}
				out := append([]byte(nil), p.output[:at]...)
				for _, segment := range segments {
					out = append(out, segment...)
				}
				p.output = append(out, p.output[at:]...)
			}
			cfg, err := jpeg.DecodeConfig(bytes.NewReader(p.output))
			if err != nil || cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > maxImagePixels {
				return p, ErrJPEGMetadataStructure
			}
			p.pixels, err = jpeg.Decode(contextRead{ctx: ctx, in: bytes.NewReader(p.output)})
			if err != nil {
				return p, errors.Join(ErrJPEGMetadataStructure, err)
			}
			return p, ctx.Err()
		}
		if pos+2 > len(data) {
			return p, ErrJPEGMetadataStructure
		}
		length := int(binary.BigEndian.Uint16(data[pos:]))
		if length < 2 || length > len(data)-pos {
			return p, ErrJPEGMetadataStructure
		}
		payload := data[pos+2 : pos+length]
		pos += length
		switch {
		case marker == 0xc0 || marker == 0xc2:
			if err := policy.frame(marker, payload); err != nil {
				return p, err
			}
			p.components = len(policy.ids)
			frame = true
		case marker == 0xda:
			if err := policy.scan(payload); err != nil {
				return p, err
			}
			scan = true
			p.output = append(p.output, data[start:pos]...)
			end, err := jpegScanEnd(ctx, data, pos)
			if err != nil {
				return p, err
			}
			p.output = append(p.output, data[pos:end]...)
			pos = end
			continue
		case marker == 0xdb || marker == 0xc4 || marker == 0xdd:
			if err := policy.table(marker, payload); err != nil {
				return p, err
			}
		case marker == 0xe0:
			if bytes.HasPrefix(payload, []byte("JFIF\x00")) {
				if start != 2 || jfif || scan || len(payload) < 14 || payload[5] != 1 || payload[6] > 2 || payload[7] > 2 || binary.BigEndian.Uint16(payload[8:10]) == 0 || binary.BigEndian.Uint16(payload[10:12]) == 0 || len(payload) < 14+3*int(payload[12])*int(payload[13]) {
					return p, ErrJPEGMetadataStructure
				}
				jfif = true
				header := append([]byte(nil), payload[:14]...)
				header[12], header[13] = 0, 0
				p.jfif = header
				p.output = append(p.output, jpegSegmentBytes(marker, header)...)
			}
			continue
		case marker == 0xee:
			if adobe || scan || len(payload) < 12 || string(payload[:5]) != "Adobe" || binary.BigEndian.Uint16(payload[5:7]) != 100 || !bytes.Equal(payload[7:11], []byte{0, 0, 0, 0}) || payload[11] > 1 || (jfif && payload[11] != 1) {
				return p, ErrJPEGMetadataProcess
			}
			adobe = true
			adobeTransform = payload[11]
			p.output = append(p.output, jpegSegmentBytes(marker, payload[:12])...)
			continue
		case marker == 0xe1:
			if bytes.HasPrefix(payload, []byte("Exif\x00\x00")) {
				if exif {
					return p, ErrJPEGMetadataOrientation
				}
				exif = true
				var err error
				p.orientation, err = removalOrientation(payload[6:])
				if err != nil {
					return p, err
				}
			}
			continue
		case marker == 0xe2 && bytes.HasPrefix(payload, []byte("ICC_PROFILE\x00")):
			if err := profile.add(payload); err != nil {
				return p, err
			}
			continue
		case marker == 0xfe || marker >= 0xe0 && marker <= 0xef:
			continue
		default:
			return p, ErrJPEGMetadataProcess
		}
		p.output = append(p.output, data[start:pos]...)
	}
	return p, ErrJPEGMetadataStructure
}

func jpegScanEnd(ctx context.Context, data []byte, pos int) (int, error) {
	markerStart := -1
	nextCheck := pos
	for ; pos < len(data); pos++ {
		if pos >= nextCheck {
			if err := ctx.Err(); err != nil {
				return 0, err
			}
			nextCheck = pos + 4096
		}
		marker := data[pos]
		if markerStart < 0 {
			if marker == 0xff {
				markerStart = pos
			}
			continue
		}
		if marker == 0xff {
			continue
		}
		if marker == 0 || marker >= 0xd0 && marker <= 0xd7 {
			markerStart = -1
			continue
		}
		return markerStart, nil
	}
	return 0, ErrJPEGMetadataStructure
}

func removalOrientation(tiff []byte) (int, error) {
	bo, ok := tiffOrder(tiff)
	if !ok {
		return 0, ErrJPEGMetadataOrientation
	}
	offset := uint64(bo.Uint32(tiff[4:8]))
	header, ok := tiffSpan(tiff, offset, 2)
	if !ok || offset < 8 {
		return 0, ErrJPEGMetadataOrientation
	}
	count := uint64(bo.Uint16(header))
	entries, ok := tiffSpan(tiff, offset+2, count*12+4)
	if !ok {
		return 0, ErrJPEGMetadataOrientation
	}
	orient, found := 1, false
	for i := uint64(0); i < count; i++ {
		e := entries[i*12 : i*12+12]
		if bo.Uint16(e[:2]) != 0x112 {
			continue
		}
		if found || bo.Uint16(e[2:4]) != 3 || bo.Uint32(e[4:8]) != 1 {
			return 0, ErrJPEGMetadataOrientation
		}
		found = true
		orient = int(bo.Uint16(e[8:10]))
		if orient < 1 || orient > 8 {
			return 0, ErrJPEGMetadataOrientation
		}
	}
	return orient, nil
}

// orientJPEGRemoval checks cancellation between rows and writes the final color
// model directly, avoiding a second full-image conversion for gray sources.
func orientJPEGRemoval(ctx context.Context, source image.Image, orientation, components int) (image.Image, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	size := image.Rect(0, 0, width, height)
	if orientation >= 5 {
		size = image.Rect(0, 0, height, width)
	}
	var output image.Image
	var gray *image.Gray
	var rgba *image.RGBA
	if components == 1 {
		gray = image.NewGray(size)
		output = gray
	} else {
		rgba = image.NewRGBA(size)
		output = rgba
	}
	pixels := orientationPixels(source)
	for y := range height {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		for x := range width {
			dx, dy := x, y
			switch orientation {
			case 2:
				dx = width - 1 - x
			case 3:
				dx, dy = width-1-x, height-1-y
			case 4:
				dy = height - 1 - y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = height-1-y, x
			case 7:
				dx, dy = height-1-y, width-1-x
			case 8:
				dx, dy = y, width-1-x
			}
			pixel := pixels.RGBA64At(bounds.Min.X+x, bounds.Min.Y+y)
			if gray != nil {
				gray.Pix[dy*gray.Stride+dx] = uint8(pixel.R >> 8)
			} else {
				rgba.SetRGBA64(dx, dy, pixel)
			}
		}
	}
	return output, ctx.Err()
}

type jpegRemovalEncodeCanceled struct{ err error }

type jpegRemovalEncodeWriter struct{ contextWrite }

func (w jpegRemovalEncodeWriter) Write(p []byte) (int, error) {
	n, err := w.contextWrite.Write(p)
	if err != nil {
		// image/jpeg records writer errors but finishes all pixel blocks. This
		// private signal exits those loops at the next buffered output write.
		panic(jpegRemovalEncodeCanceled{err: err})
	}
	return n, nil
}

func encodeJPEGRemoval(ctx context.Context, pixels image.Image, original []byte) (output []byte, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if cancelled, ok := recovered.(jpegRemovalEncodeCanceled); ok {
				output, err = nil, cancelled.err
			} else {
				panic(recovered)
			}
		}
	}()
	var encoded bytes.Buffer
	writer := jpegRemovalEncodeWriter{contextWrite{ctx: ctx, out: &encoded}}
	if err := jpeg.Encode(writer, pixels, &jpeg.Options{Quality: jpegSaveQuality}); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return injectJPEGMetadata(encoded.Bytes(), jpegICCSegments(original))
}
