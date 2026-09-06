package imaging

import (
	"bytes"
	"image/color"
	"testing"

	"github.com/frathe/picfetch/internal/uitest"
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

// sofPayload builds a start-of-frame payload: precision, height, width (both
// big-endian, height first), then whatever extra bytes a case needs (usually
// a component count and per-component triplets, none of which jpegFrameSize
// reads).
func sofPayload(precision byte, height, width uint16, extra ...byte) []byte {
	p := []byte{precision, byte(height >> 8), byte(height), byte(width >> 8), byte(width)}
	return append(p, extra...)
}

func TestJPEGFrameSize(t *testing.T) {
	t.Run("non-JPEG and nil report ok=false", func(t *testing.T) {
		if _, _, ok := jpegFrameSize([]byte("\x89PNG")); ok {
			t.Fatal("ok = true, want false for non-JPEG data")
		}
		if _, _, ok := jpegFrameSize(nil); ok {
			t.Fatal("ok = true, want false for nil data")
		}
	})

	t.Run("SOI only, no segments at all", func(t *testing.T) {
		if _, _, ok := jpegFrameSize([]byte{0xFF, 0xD8}); ok {
			t.Fatal("ok = true, want false: there is no frame header to read")
		}
	})

	t.Run("baseline SOF0 from a real encoder", func(t *testing.T) {
		// 9x4 is asymmetric so a height/width swap fails this test.
		data := uitest.EncodeJPEG(t, 9, 4, color.White)
		w, h, ok := jpegFrameSize(data)
		if !ok {
			t.Fatal("ok = false, want true")
		}
		if w != 9 || h != 4 {
			t.Fatalf("got %dx%d, want 9x4", w, h)
		}
	})

	t.Run("progressive SOF2, hand-assembled", func(t *testing.T) {
		// image/jpeg cannot encode progressive JPEGs, so this segment is
		// built by hand. 21x6 is asymmetric and distinct from the baseline
		// case above, so a swap or a hardcoded return also fails this test.
		sof2 := jpegSegmentBytes(0xC2, sofPayload(8, 6, 21, 1, 0x11, 0))
		w, h, ok := jpegFrameSize(jpegWith(sof2))
		if !ok {
			t.Fatal("ok = false, want true")
		}
		if w != 21 || h != 6 {
			t.Fatalf("got %dx%d, want 21x6", w, h)
		}
	})

	t.Run("DHT (0xC4) is not mistaken for a frame header", func(t *testing.T) {
		dht := jpegSegmentBytes(0xC4, sofPayload(8, 999, 888))
		if _, _, ok := jpegFrameSize(jpegWith(dht)); ok {
			t.Fatal("ok = true, want false: a DHT payload must not be read as SOF")
		}
	})

	t.Run("reserved JPG marker (0xC8) is not mistaken for a frame header", func(t *testing.T) {
		reserved := jpegSegmentBytes(0xC8, sofPayload(8, 999, 888))
		if _, _, ok := jpegFrameSize(jpegWith(reserved)); ok {
			t.Fatal("ok = true, want false: 0xC8 must not be read as SOF")
		}
	})

	t.Run("DAC (0xCC) is not mistaken for a frame header", func(t *testing.T) {
		dac := jpegSegmentBytes(0xCC, sofPayload(8, 999, 888))
		if _, _, ok := jpegFrameSize(jpegWith(dac)); ok {
			t.Fatal("ok = true, want false: a DAC payload must not be read as SOF")
		}
	})

	t.Run("skips a leading DHT and finds the real SOF0", func(t *testing.T) {
		dht := jpegSegmentBytes(0xC4, []byte{0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
		sof0 := jpegSegmentBytes(0xC0, sofPayload(8, 12, 34, 1, 0x11, 0))
		w, h, ok := jpegFrameSize(jpegWith(dht, sof0))
		if !ok {
			t.Fatal("ok = false, want true")
		}
		if w != 34 || h != 12 {
			t.Fatalf("got %dx%d, want 34x12", w, h)
		}
	})

	t.Run("no SOF before SOS, only a COM segment", func(t *testing.T) {
		com := []byte{0xFF, 0xFE, 0x00, 0x05, 'h', 'i', 0x00}
		if _, _, ok := jpegFrameSize(jpegWith(com)); ok {
			t.Fatal("ok = true, want false: no frame header is present")
		}
	})

	t.Run("4-byte payload is too short", func(t *testing.T) {
		short := jpegSegmentBytes(0xC0, []byte{8, 0, 12, 0}) // missing the low width byte
		if _, _, ok := jpegFrameSize(jpegWith(short)); ok {
			t.Fatal("ok = true, want false: 4 bytes cannot hold height and width")
		}
	})

	t.Run("5-byte payload is exactly usable", func(t *testing.T) {
		exact := jpegSegmentBytes(0xC0, sofPayload(8, 12, 34)) // no components byte at all
		w, h, ok := jpegFrameSize(jpegWith(exact))
		if !ok {
			t.Fatal("ok = false, want true")
		}
		if w != 34 || h != 12 {
			t.Fatalf("got %dx%d, want 34x12", w, h)
		}
	})

	t.Run("zero width reports ok=false", func(t *testing.T) {
		zeroW := jpegSegmentBytes(0xC0, sofPayload(8, 12, 0, 1, 0x11, 0))
		if _, _, ok := jpegFrameSize(jpegWith(zeroW)); ok {
			t.Fatal("ok = true, want false for a zero width")
		}
	})

	t.Run("zero height (DNL deferral) reports ok=false", func(t *testing.T) {
		zeroH := jpegSegmentBytes(0xC0, sofPayload(8, 0, 34, 1, 0x11, 0))
		if _, _, ok := jpegFrameSize(jpegWith(zeroH)); ok {
			t.Fatal("ok = true, want false for a zero height")
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
