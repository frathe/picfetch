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
	ErrJPEGMetadataMemory      = errors.New("JPEG metadata removal exceeds its working-memory limit")
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
	encodeLimit int64
	icc         [][]byte
}

type jpegRemovalICCSpan struct {
	at   int
	data []byte
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
	memory, err := jpegRemovalAdmission(ctx, data)
	if err != nil {
		return p, err
	}
	p.encodeLimit = (jpegRemovalWorkingBytes - memory.oriented) / 4
	p.output = []byte{0xff, 0xd8}
	var frame, scan, jfif, adobe, exif, exifColor bool
	adobeTransform := byte(0xff)
	var policy jpegScanPolicy
	var profile removalICC
	var iccSpans []jpegRemovalICCSpan
	jfifEnd := 0
	for pos := 2; pos < len(data); {
		if err := ctx.Err(); err != nil {
			return p, err
		}
		start := pos
		marker, after, err := jpegRemovalMarker(ctx, data, pos)
		if err != nil {
			return p, err
		}
		pos = after
		if marker == 0xd9 {
			if !frame || !scan || !policy.complete() {
				return p, ErrJPEGMetadataStructure
			}
			if p.orientation != 1 && p.encodeLimit <= 0 {
				return p, ErrJPEGMetadataMemory
			}
			if adobe && ((jfif && adobeTransform != 1) || (p.components == 1 && adobeTransform != 0)) {
				return p, ErrJPEGMetadataProcess
			}
			if p.components == 3 && !jfif && !adobe && !bytes.Equal(policy.ids, []byte{1, 2, 3}) && !bytes.Equal(policy.ids, []byte("RGB")) {
				return p, ErrJPEGMetadataProcess
			}
			if bytes.Equal(policy.ids, []byte("RGB")) && (jfif || adobe && adobeTransform != 0) {
				return p, ErrJPEGMetadataProcess
			}
			if exifColor && profile.total != 0 {
				return p, ErrJPEGMetadataProcess
			}
			p.output = append(p.output, data[start:pos]...)
			segments, iccClean, err := profile.normalized(ctx, p.components)
			if err != nil {
				return p, err
			}
			p.icc = segments
			if len(segments) > 0 && iccClean {
				// Qualified normalized profiles need no sanitization. Preserve
				// their original marker fill, chunking and scan placement.
				p.output = restoreRemovalICC(p.output, iccSpans)
			} else if len(segments) > 0 {
				// Keep a leading JFIF directly after SOI.
				at := 2
				if jfifEnd != 0 {
					at = jfifEnd
				}
				out := append([]byte(nil), p.output[:at]...)
				for _, segment := range segments {
					out = append(out, segment...)
				}
				p.output = append(out, p.output[at:]...)
			}
			cfg, err := jpeg.DecodeConfig(contextRead{ctx: ctx, in: bytes.NewReader(p.output)})
			if cancelled := ctx.Err(); cancelled != nil {
				return p, cancelled
			}
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
			end, err := policy.entropyEnd(ctx, data, pos, payload)
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
				p.output = appendRemovalSegment(p.output, data[start:after], header)
				jfifEnd = len(p.output)
			}
			continue
		case marker == 0xee:
			if adobe || scan || len(payload) < 12 || string(payload[:5]) != "Adobe" || binary.BigEndian.Uint16(payload[5:7]) != 100 || !bytes.Equal(payload[7:11], []byte{0, 0, 0, 0}) || payload[11] > 1 || (jfif && payload[11] != 1) {
				return p, ErrJPEGMetadataProcess
			}
			adobe = true
			adobeTransform = payload[11]
			p.output = appendRemovalSegment(p.output, data[start:after], payload[:12])
			continue
		case marker == 0xe1:
			if bytes.HasPrefix(payload, []byte("Exif\x00\x00")) {
				if exif {
					return p, ErrJPEGMetadataOrientation
				}
				exif = true
				var err error
				p.orientation, exifColor, err = removalEXIF(payload[6:])
				if err != nil {
					return p, err
				}
			}
			continue
		case marker == 0xe2 && bytes.HasPrefix(payload, []byte("ICC_PROFILE\x00")):
			if err := profile.add(payload); err != nil {
				return p, err
			}
			iccSpans = append(iccSpans, jpegRemovalICCSpan{at: len(p.output), data: data[start:pos]})
			continue
		case marker == 0xe8 && bytes.HasPrefix(payload, []byte("SPIFF\x00")):
			return p, ErrJPEGMetadataProcess
		case marker == 0xfe || marker >= 0xe0 && marker <= 0xef:
			continue
		default:
			return p, ErrJPEGMetadataProcess
		}
		p.output = append(p.output, data[start:pos]...)
	}
	return p, ErrJPEGMetadataStructure
}

func appendRemovalSegment(output, prefix, payload []byte) []byte {
	output = append(output, prefix...)
	output = binary.BigEndian.AppendUint16(output, uint16(len(payload)+2))
	return append(output, payload...)
}

func restoreRemovalICC(output []byte, spans []jpegRemovalICCSpan) []byte {
	size := len(output)
	for _, span := range spans {
		size += len(span.data)
	}
	restored := make([]byte, 0, size)
	pos := 0
	for _, span := range spans {
		restored = append(restored, output[pos:span.at]...)
		restored = append(restored, span.data...)
		pos = span.at
	}
	return append(restored, output[pos:]...)
}

// removalEXIF qualifies orientation and the color declarations which would be
// lost with APP1. Only default sRGB interpretation is qualified for removal.
// The boolean reports an explicit color declaration whose agreement with a
// retained ICC profile would also need qualification.
func removalEXIF(tiff []byte) (int, bool, error) {
	bo, ok := tiffOrder(tiff)
	if !ok {
		return 0, false, ErrJPEGMetadataOrientation
	}
	const (
		primary = iota
		exif
		interop
	)
	// Follow only IFD0 -> Exif -> Interoperability. GPS and thumbnail IFDs
	// contain no qualified primary-image color declarations.
	offsets := [3]uint64{uint64(bo.Uint32(tiff[4:8]))}
	var ends [3]uint64
	orient, foundOrientation, foundSpace, foundIndex := 1, false, false, false
	foundPositioning := false
	for level := range offsets {
		offset := offsets[level]
		if level != primary && offset == 0 {
			continue
		}
		invalid := ErrJPEGMetadataProcess
		if level == primary {
			invalid = ErrJPEGMetadataOrientation
		}
		header, ok := tiffSpan(tiff, offset, 2)
		if !ok || offset < 8 {
			return 0, false, invalid
		}
		count := uint64(bo.Uint16(header))
		entries, ok := tiffSpan(tiff, offset+2, count*12+4)
		if !ok {
			return 0, false, invalid
		}
		ends[level] = offset + 2 + count*12 + 4
		for previous := 0; previous < level; previous++ {
			if offset < ends[previous] && offsets[previous] < ends[level] {
				return 0, false, ErrJPEGMetadataProcess
			}
		}
		for i := uint64(0); i < count; i++ {
			e := entries[i*12 : i*12+12]
			tag, kind, values := bo.Uint16(e[:2]), bo.Uint16(e[2:4]), bo.Uint32(e[4:8])
			switch {
			case level == primary && tag == 0x0213:
				// Centered chroma is also the EXIF default when this tag is absent.
				if foundPositioning || kind != 3 || values != 1 || bo.Uint16(e[8:10]) != 1 {
					return 0, false, ErrJPEGMetadataProcess
				}
				foundPositioning = true
			case level == primary && (tag == 0x012d || tag == 0x013e || tag == 0x013f || tag == 0x0211 || tag == 0x0214), level == exif && tag == 0xa500:
				// Explicit transfer/colorimetry tags need their own transform
				// qualification; ICC presence does not establish precedence.
				return 0, false, ErrJPEGMetadataProcess
			case level == primary && tag == 0x112:
				if foundOrientation || kind != 3 || values != 1 {
					return 0, false, ErrJPEGMetadataOrientation
				}
				foundOrientation = true
				orient = int(bo.Uint16(e[8:10]))
				if orient < 1 || orient > 8 {
					return 0, false, ErrJPEGMetadataOrientation
				}
			case level == primary && tag == 0x8769 || level == exif && tag == 0xa005:
				if offsets[level+1] != 0 || kind != 4 || values != 1 || bo.Uint32(e[8:12]) < 8 {
					return 0, false, ErrJPEGMetadataProcess
				}
				offsets[level+1] = uint64(bo.Uint32(e[8:12]))
			case level == exif && tag == 0xa001:
				if foundSpace || kind != 3 || values != 1 || bo.Uint16(e[8:10]) != 1 {
					return 0, false, ErrJPEGMetadataProcess
				}
				foundSpace = true
			case level == interop && tag == 1:
				if foundIndex || kind != 2 || values != 4 || string(e[8:12]) != "R98\x00" {
					return 0, false, ErrJPEGMetadataProcess
				}
				foundIndex = true
			}
		}
	}
	return orient, foundSpace || foundIndex, nil
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
				// Exactly one destination is allocated: gray == nil implies rgba != nil.
				//goland:noinspection GoMaybeNil
				rgba.SetRGBA64(dx, dy, pixel)
			}
		}
	}
	return output, ctx.Err()
}

type jpegRemovalEncodeCanceled struct{ err error }

type jpegRemovalEncodeWriter struct {
	contextWrite
	limit   int64
	written int64
}

func (w *jpegRemovalEncodeWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > w.limit-w.written {
		panic(jpegRemovalEncodeCanceled{err: ErrJPEGMetadataMemory})
	}
	n, err := w.contextWrite.Write(p)
	if err != nil {
		// image/jpeg records writer errors but finishes all pixel blocks. This
		// private signal exits those loops at the next buffered output write.
		panic(jpegRemovalEncodeCanceled{err: err})
	}
	w.written += int64(n)
	return n, nil
}

func encodeJPEGRemoval(ctx context.Context, pixels image.Image, profile [][]byte, limit int64) (output []byte, err error) {
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
	writer := &jpegRemovalEncodeWriter{contextWrite: contextWrite{ctx: ctx, out: &encoded}, limit: limit}
	if err := jpeg.Encode(writer, pixels, &jpeg.Options{Quality: jpegSaveQuality}); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return injectJPEGMetadata(encoded.Bytes(), profile)
}
