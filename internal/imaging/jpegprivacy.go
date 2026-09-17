package imaging

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
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
	if !bytes.Equal(data, p.output) {
		state = JPEGMetadataRemovable
	}
	return JPEGMetadataInspection{State: state}
}

type jpegRemoval struct {
	output []byte
}

type jpegRemovalICCSpan struct {
	at   int
	data []byte
}

// prepareJPEGRemoval deliberately does not change the tolerant header reader used
// by viewing and metadata-preserving export. Only this operation promises removal.
func prepareJPEGRemoval(ctx context.Context, data []byte) (jpegRemoval, error) {
	var p jpegRemoval
	if err := ctx.Err(); err != nil {
		return p, err
	}
	if len(data) < 4 || !bytes.Equal(data[:2], []byte{0xff, 0xd8}) {
		return p, errNotJPEG
	}
	if int64(len(data)) > MaxEncodedBytes() {
		return p, &InputTooLargeError{limit: MaxEncodedBytes()}
	}
	_, err := jpegRemovalAdmission(ctx, data)
	if err != nil {
		return p, err
	}
	p.output = []byte{0xff, 0xd8}
	var frame, scan, jfif, adobe, exif bool
	var components int
	adobeTransform := byte(0xff)
	var policy jpegScanPolicy
	var profile removalICC
	var iccSpans []jpegRemovalICCSpan
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
			if adobe && ((jfif && adobeTransform != 1) || (components == 1 && adobeTransform != 0)) {
				return p, ErrJPEGMetadataProcess
			}
			if components == 3 && !jfif && !adobe && !bytes.Equal(policy.ids, []byte{1, 2, 3}) && !bytes.Equal(policy.ids, []byte("RGB")) {
				return p, ErrJPEGMetadataProcess
			}
			if bytes.Equal(policy.ids, []byte("RGB")) && (jfif || adobe && adobeTransform != 0) {
				return p, ErrJPEGMetadataProcess
			}
			p.output = append(p.output, data[start:pos]...)
			segments, iccClean, err := profile.normalized(ctx, components)
			if err != nil {
				return p, err
			}
			if len(segments) > 0 && iccClean {
				// Qualified normalized profiles need no sanitization. Preserve
				// their original marker fill, chunking and scan placement.
				p.output = restoreRemovalICC(p.output, iccSpans)
			} else if len(segments) > 0 {
				// Preserve the first profile's position relative to EXIF and
				// other rendering declarations, including a leading JFIF.
				at := iccSpans[0].at
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
			_, err = jpeg.Decode(contextRead{ctx: ctx, in: bytes.NewReader(p.output)})
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
			components = len(policy.ids)
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
				p.output = appendRemovalSegment(p.output, data[start:after], header)
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
				rendering, err := removalEXIF(payload[6:])
				if err != nil {
					return p, err
				}
				if len(rendering) != 0 {
					p.output = appendRemovalSegment(p.output, data[start:after], rendering)
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
