package heicdecode

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
	"time"
)

func TestLimitsValidate(t *testing.T) {
	valid := DefaultLimits(32 * 1024 * 1024)
	if err := valid.Validate(); err != nil {
		t.Fatalf("default limits: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*Limits)
	}{
		{"zero timeout", func(l *Limits) { l.Timeout = 0 }},
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
			if _, err := decodeResponse(bytes.NewReader(tt.data), limits); !errors.Is(err, ErrInvalidResponse) {
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
	if got := DefaultLimits(7).Timeout; got != 30*time.Second {
		t.Fatalf("timeout = %v, want 30s", got)
	}
}

func responseBytes(t *testing.T, width, height uint32, format, depth uint16, pixels, metadata, diagnostic []byte) []byte {
	t.Helper()
	var out bytes.Buffer
	out.WriteString(responseMagic)
	values := []any{uint16(protocolVersion), uint16(statusOK), width, height, uint32(width * 4), depth, format,
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
