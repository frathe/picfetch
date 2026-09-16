package heicdecode

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"testing"
	"time"
)

func TestLimitsValidate(t *testing.T) {
	valid := DefaultLimits(32 * 1024 * 1024)
	valid.Timeout = time.Minute
	if err := valid.Validate(); err != nil {
		t.Fatalf("default limits: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*Limits)
	}{
		{"zero timeout", func(l *Limits) { l.Timeout = 0 }},
		{"timeout over hard ceiling", func(l *Limits) { l.Timeout = time.Minute + time.Nanosecond }},
		{"fractional WASM page", func(l *Limits) { l.WASMMemoryBytes = 1 }},
		{"guest exceeds OS budget", func(l *Limits) { l.OSProcessBytes = l.WASMMemoryBytes - 1 }},
		{"input over hard ceiling", func(l *Limits) { l.MaxInputBytes = hardMaxInputBytes + 1 }},
		{"zero pixels", func(l *Limits) { l.MaxPixels = 0 }},
		{"output too large", func(l *Limits) { l.MaxOutputBytes = hardMaxOutputBytes + 1 }},
		{"metadata too large", func(l *Limits) { l.MaxMetadataBytes = hardMaxMetadataBytes + 1 }},
		{"diagnostic too large", func(l *Limits) { l.MaxDiagnosticBytes = hardMaxDiagnosticBytes + 1 }},
		{"wrong live jobs", func(l *Limits) { l.MaxLiveJobs = 2 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limits := valid
			tt.mutate(&limits)
			if err := limits.Validate(); !errors.Is(err, ErrInvalidLimits) {
				t.Fatalf("Validate() error = %v, want ErrInvalidLimits", err)
			}
		})
	}
}

func TestDecodeResponseRejectsInvalidMessages(t *testing.T) {
	limits := DefaultLimits(1024)
	valid := responseBytes(t, 2, 1, pixelFormatRGBA8, 8, []byte{1, 2, 3, 4, 5, 6, 7, 8}, nil, nil)

	tests := []struct {
		name string
		data []byte
	}{
		{"truncated", valid[:len(valid)-1]},
		{"trailing", append(append([]byte(nil), valid...), 0)},
		{"unknown version", replaceUint16(valid, 4, protocolVersion+1)},
		{"unknown status", replaceUint16(valid, 6, 99)},
		{"zero width", replaceUint32(valid, 8, 0)},
		{"bad stride", replaceUint32(valid, 16, 3)},
		{"unsupported bit depth", replaceUint16(valid, 20, 10)},
		{"unknown pixel format", replaceUint16(valid, 22, 99)},
		{"pixel length mismatch", replaceUint64(valid, 24, 7)},
		{"metadata over limit", responseBytes(t, 2, 1, pixelFormatRGBA8, 8, make([]byte, 8), make([]byte, limits.MaxMetadataBytes+1), nil)},
		{"diagnostic over limit", responseBytes(t, 2, 1, pixelFormatRGBA8, 8, make([]byte, 8), nil, make([]byte, limits.MaxDiagnosticBytes+1))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ReadResponse(bytes.NewReader(tt.data), Decode, limits); !errors.Is(err, ErrInvalidResponse) {
				t.Fatalf("decodeResponse() error = %v, want ErrInvalidResponse", err)
			}
		})
	}
}

func TestDefaultLimitsClampUserInput(t *testing.T) {
	if got := DefaultLimits(0).MaxInputBytes; got != hardMaxInputBytes {
		t.Fatalf("zero user limit = %d, want %d", got, hardMaxInputBytes)
	}
	if got := DefaultLimits(512 * 1024 * 1024).MaxInputBytes; got != hardMaxInputBytes {
		t.Fatalf("large user limit = %d, want %d", got, hardMaxInputBytes)
	}
	if got := DefaultLimits(7).MaxInputBytes; got != 7 {
		t.Fatalf("small user limit = %d, want 7", got)
	}
	if got := DefaultLimits(7).Timeout; got != time.Minute {
		t.Fatalf("timeout = %v, want 60s", got)
	}
}

func responseBytes(t *testing.T, width, height uint32, format, depth uint16, pixels, metadata, diagnostic []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	out.WriteString(responseMagic)
	values := []any{uint16(protocolVersion), uint16(statusOK), width, height, width * 4, depth, format,
		uint64(len(pixels)), uint32(len(metadata)), uint32(len(diagnostic))}
	for _, value := range values {
		if err := binary.Write(&out, binary.LittleEndian, value); err != nil {
			t.Fatal(err)
		}
	}
	out.Write(pixels)
	out.Write(metadata)
	out.Write(diagnostic)
	return out.Bytes()
}

func replaceUint16(data []byte, offset int, value uint16) []byte {
	out := append([]byte(nil), data...)
	binary.LittleEndian.PutUint16(out[offset:offset+2], value)
	return out
}

func replaceUint32(data []byte, offset int, value uint32) []byte {
	out := append([]byte(nil), data...)
	binary.LittleEndian.PutUint32(out[offset:offset+4], value)
	return out
}

func replaceUint64(data []byte, offset int, value uint64) []byte {
	out := append([]byte(nil), data...)
	binary.LittleEndian.PutUint64(out[offset:offset+8], value)
	return out
}

func TestDecodeResponsePreservesStraightAlphaAndSixteenBitSamples(t *testing.T) {
	for _, tt := range []struct {
		name   string
		format uint16
		depth  uint16
		pixels []byte
	}{
		{"straight alpha", pixelFormatRGBA8, 8, []byte{255, 0, 0, 128}},
		{"sixteen bit", 2, 16, []byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xff}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			data := responseBytes(t, 1, 1, tt.format, tt.depth, tt.pixels, nil, nil)
			binary.LittleEndian.PutUint32(data[16:20], uint32(len(tt.pixels)))
			got, err := ReadResponse(bytes.NewReader(data), Decode, DefaultLimits(0))
			if err != nil {
				t.Fatal(err)
			}
			switch img := any(got.Image).(type) {
			case *image.NRGBA:
				if !bytes.Equal(img.Pix, tt.pixels) {
					t.Fatal("straight alpha samples changed")
				}
			case *image.NRGBA64:
				if !bytes.Equal(img.Pix, tt.pixels) {
					t.Fatal("sixteen-bit samples changed")
				}
			default:
				t.Fatalf("decoded image type %T loses straight alpha or precision", got.Image)
			}
		})
	}
}
