package imaging

import (
	"bytes"
	"context"
	"encoding/binary"
	"image/jpeg"
)

const jpegRemovalWorkingBytes int64 = 256 * 1024 * 1024
const jpegRemovalScratchBytes int64 = 16 * 1024 * 1024
const jpegRemovalSourceBytes = (jpegRemovalWorkingBytes - jpegRemovalScratchBytes) / 4

type jpegRemovalMemory struct {
	working  int64
	oriented int64
}

// jpegRemovalAdmission reads headers without allocating image planes or copied
// JPEG data. Account MCU padding, all component samples, progressive int32
// coefficient blocks, possible RGB conversion, and an oriented output plus its
// validation decode. Four encoded copies and ICC/small-object scratch are
// reserved separately. This is an operation budget, not the foreground cache.
func jpegRemovalAdmission(ctx context.Context, data []byte) (jpegRemovalMemory, error) {
	if int64(len(data)) > jpegRemovalSourceBytes {
		return jpegRemovalMemory{}, ErrJPEGMetadataMemory
	}
	memory := jpegRemovalMemory{working: jpegRemovalScratchBytes + 4*int64(len(data))}
	cfg, err := jpeg.DecodeConfig(contextRead{ctx: ctx, in: bytes.NewReader(data)})
	if cancelled := ctx.Err(); cancelled != nil {
		return memory, cancelled
	}
	if err != nil || checkDimensions(cfg.Width, cfg.Height) != nil {
		return memory, ErrJPEGMetadataStructure
	}
	frame, payload, err := jpegRemovalFrame(ctx, data)
	if err != nil {
		return memory, err
	}
	horizontal, vertical := int64(1), int64(1)
	if len(frame.ids) == 3 {
		for i := range frame.ids {
			sampling := payload[7+3*i]
			horizontal = max(horizontal, int64(sampling>>4))
			vertical = max(vertical, int64(sampling&15))
		}
	}
	columns := (int64(cfg.Width) + 8*horizontal - 1) / (8 * horizontal)
	rows := (int64(cfg.Height) + 8*vertical - 1) / (8 * vertical)
	plane := columns * rows * 64 * horizontal * vertical
	samples := plane
	if len(frame.ids) == 3 {
		samples = 0
		for i := range frame.ids {
			sampling := payload[7+3*i]
			samples += columns * rows * 64 * int64(sampling>>4) * int64(sampling&15)
		}
	}
	if frame.progressive {
		memory.working += 4 * samples
		// Strict entropy validation retains one nonzero-coefficient mask
		// per padded block, conservatively alongside the decoder arrays.
		memory.working += samples / 8
	}
	if len(frame.ids) == 3 {
		// Go's flexible sampling path may expand all three components
		// to full planes, then retain them during RGB conversion.
		memory.working += 7 * plane
		// The new RGB encode is 4:2:0 and pads independently to 16x16
		// MCUs. Reserve RGBA orientation plus its validation planes.
		outputPlane := ((int64(cfg.Width) + 15) / 16) * 16 * ((int64(cfg.Height) + 15) / 16) * 16
		memory.oriented = memory.working + 4*plane + 2*outputPlane
	} else {
		memory.working += plane
		memory.oriented = memory.working + 2*plane
	}
	if memory.working > jpegRemovalWorkingBytes {
		return memory, ErrJPEGMetadataMemory
	}
	return memory, ctx.Err()
}

// jpegRemovalFrame retains legal marker fill accepted by the complete removal
// parser. The tolerant viewing/export walker has a different header policy.
func jpegRemovalFrame(ctx context.Context, data []byte) (jpegScanPolicy, []byte, error) {
	var frame jpegScanPolicy
	for pos := 2; pos < len(data); {
		if err := ctx.Err(); err != nil {
			return frame, nil, err
		}
		marker, after, err := jpegRemovalMarker(ctx, data, pos)
		if err != nil {
			return frame, nil, err
		}
		pos = after
		if marker == 0x00 || marker == 0x01 || marker >= 0xd0 && marker <= 0xda {
			return frame, nil, ErrJPEGMetadataProcess
		}
		if len(data)-pos < 2 {
			return frame, nil, ErrJPEGMetadataStructure
		}
		length := int(binary.BigEndian.Uint16(data[pos : pos+2]))
		if length < 2 || length > len(data)-pos {
			return frame, nil, ErrJPEGMetadataStructure
		}
		payload := data[pos+2 : pos+length]
		pos += length
		if marker == 0xc0 || marker == 0xc2 {
			err := frame.frame(marker, payload)
			return frame, payload, err
		}
	}
	return frame, nil, ErrJPEGMetadataProcess
}

// jpegRemovalMarker shares bounded fill traversal between admission and the
// complete parser. Callers retain the original marker span when writing output.
func jpegRemovalMarker(ctx context.Context, data []byte, pos int) (byte, int, error) {
	if pos >= len(data) || data[pos] != 0xff {
		return 0, pos, ErrJPEGMetadataStructure
	}
	nextCheck := pos
	for pos < len(data) && data[pos] == 0xff {
		if pos >= nextCheck {
			if err := ctx.Err(); err != nil {
				return 0, pos, err
			}
			nextCheck = pos + 4096
		}
		pos++
	}
	if pos == len(data) {
		return 0, pos, ErrJPEGMetadataStructure
	}
	return data[pos], pos + 1, nil
}
