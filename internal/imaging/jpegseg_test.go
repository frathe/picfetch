package imaging

import (
	"bytes"
	"testing"
)

type jpegVisit struct {
	marker  byte
	payload []byte
}

func collectJPEGSegments(data []byte) []jpegVisit {
	var got []jpegVisit
	walkJPEGSegments(data, func(marker byte, payload []byte) bool {
		p := make([]byte, len(payload))
		copy(p, payload)
		got = append(got, jpegVisit{marker: marker, payload: p})
		return true
	})
	return got
}

func jpegWith(segments ...[]byte) []byte {
	data := []byte{0xFF, 0xD8}
	for _, s := range segments {
		data = append(data, s...)
	}
	data = append(data, 0xFF, 0xDA, 0x00, 0x08, 0, 0, 0, 0, 0, 0)
	return data
}

func TestWalkJPEGSegments(t *testing.T) {
	com := []byte{0xFF, 0xFE, 0x00, 0x05, 'h', 'i', 0x00}
	comPayload := []byte{'h', 'i', 0x00}

	t.Run("non-JPEG is a no-op", func(t *testing.T) {
		if got := collectJPEGSegments([]byte("\x89PNG")); len(got) != 0 {
			t.Fatalf("visits = %d, want 0", len(got))
		}
		if got := collectJPEGSegments(nil); len(got) != 0 {
			t.Fatalf("nil visits = %d, want 0", len(got))
		}
	})

	t.Run("SOI-only truncated file is a no-op", func(t *testing.T) {
		if got := collectJPEGSegments([]byte{0xFF, 0xD8}); len(got) != 0 {
			t.Fatalf("visits = %d, want 0", len(got))
		}
	})

	t.Run("one COM then SOS", func(t *testing.T) {
		got := collectJPEGSegments(jpegWith(com))
		if len(got) != 1 || got[0].marker != 0xFE || !bytes.Equal(got[0].payload, comPayload) {
			t.Fatalf("got %+v, want one COM payload %q", got, comPayload)
		}
	})

	t.Run("skips no-payload RST and TEM; still visits neighbours", func(t *testing.T) {
		app1 := wrapAsAPP1([]byte("Exif\x00\x00xxxx"))
		rst := []byte{0xFF, 0xD0}
		tem := []byte{0xFF, 0x01}
		got := collectJPEGSegments(jpegWith(app1, rst, tem, com))
		if len(got) != 2 || got[0].marker != 0xE1 || got[1].marker != 0xFE {
			t.Fatalf("got markers %v, want E1 then FE", got)
		}
		if !bytes.Equal(got[1].payload, comPayload) {
			t.Fatalf("COM payload = %q, want %q", got[1].payload, comPayload)
		}
	})

	t.Run("does not visit segments after SOS", func(t *testing.T) {
		after := []byte{0xFF, 0xFE, 0x00, 0x05, 'n', 'o', 0x00}
		data := jpegWith(com)
		data = append(data, after...)
		got := collectJPEGSegments(data)
		if len(got) != 1 || got[0].marker != 0xFE || !bytes.Equal(got[0].payload, comPayload) {
			t.Fatalf("got %+v, want only the pre-SOS COM", got)
		}
	})

	t.Run("malformed length stops the walk", func(t *testing.T) {
		short := []byte{0xFF, 0xE1, 0x00, 0x01} // segLen 1 < 2
		got := collectJPEGSegments(jpegWith(com, short, []byte{0xFF, 0xFE, 0x00, 0x05, 'z', 'z', 0x00}))
		if len(got) != 1 || got[0].marker != 0xFE {
			t.Fatalf("got %+v, want only the COM before the bad length", got)
		}
	})

	t.Run("truncated payload stops the walk", func(t *testing.T) {
		trunc := []byte{0xFF, 0xE1, 0x00, 0x10, 1, 2, 3} // claims 14 payload bytes
		got := collectJPEGSegments(append([]byte{0xFF, 0xD8}, append(com, trunc...)...))
		if len(got) != 1 || got[0].marker != 0xFE {
			t.Fatalf("got %+v, want only the COM before the truncated APP1", got)
		}
	})

	t.Run("non-0xFF at a marker boundary stops the walk", func(t *testing.T) {
		data := append([]byte{0xFF, 0xD8}, com...)
		data = append(data, 0x00, 0x00)
		data = append(data, com...)
		data = append(data, 0xFF, 0xDA, 0x00, 0x08, 0, 0, 0, 0, 0, 0)
		got := collectJPEGSegments(data)
		if len(got) != 1 || got[0].marker != 0xFE {
			t.Fatalf("got %+v, want only the COM before the non-FF byte", got)
		}
	})

	t.Run("false from fn stops the walk", func(t *testing.T) {
		com2 := []byte{0xFF, 0xFE, 0x00, 0x05, 'b', 'y', 0x00}
		var n int
		walkJPEGSegments(jpegWith(com, com2), func(marker byte, payload []byte) bool {
			n++
			return false
		})
		if n != 1 {
			t.Fatalf("callbacks = %d, want 1", n)
		}
	})
}

func TestJPEGSegmentBytes(t *testing.T) {
	payload := []byte("Exif\x00\x00hi")
	got := jpegSegmentBytes(0xE1, payload)
	want := wrapAsAPP1(payload)
	if !bytes.Equal(got, want) {
		t.Fatalf("jpegSegmentBytes = %x, want %x", got, want)
	}
	// Mutating the result must not change payload.
	got[4] = 'X'
	if payload[0] != 'E' {
		t.Fatal("jpegSegmentBytes must copy the payload")
	}
}
