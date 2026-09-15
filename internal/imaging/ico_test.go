package imaging

import (
	"bytes"
	"context"
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"testing"

	"fyne.io/fyne/v2/storage"
	_ "github.com/fyne-io/image/ico" // mirrors the desktop driver's earlier format registration
)

type icoFixture struct {
	w, h int
	data []byte
}

func pngIcon(t *testing.T, w, h int) icoFixture {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	img.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return icoFixture{w, h, buf.Bytes()}
}

func iconFile(entries ...icoFixture) []byte {
	data := make([]byte, 6+16*len(entries))
	binary.LittleEndian.PutUint16(data[2:4], 1)
	binary.LittleEndian.PutUint16(data[4:6], uint16(len(entries)))
	for i, entry := range entries {
		dir := data[6+16*i : 6+16*(i+1)]
		dir[0], dir[1] = byte(entry.w), byte(entry.h)
		binary.LittleEndian.PutUint16(dir[4:6], 1)
		binary.LittleEndian.PutUint16(dir[6:8], 32)
		binary.LittleEndian.PutUint32(dir[8:12], uint32(len(entry.data)))
		binary.LittleEndian.PutUint32(dir[12:16], uint32(len(data)))
		data = append(data, entry.data...)
	}
	return data
}

func TestICOProbeMatchesSelectedImage(t *testing.T) {
	data := iconFile(pngIcon(t, 8, 8), pngIcon(t, 16, 12))
	u := storage.NewFileURI(writeTempFile(t, "multi.ico", data))
	_, bounds, err := ReadAndProbe(context.Background(), u)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := DecodeLoaded(context.Background(), data, 0)
	if err != nil {
		t.Fatal(err)
	}
	if want := image.Rect(0, 0, 16, 12); bounds != want || loaded.Frames[0].Bounds() != want {
		t.Fatalf("probe %v and decoded %v must select %v", bounds, loaded.Frames[0].Bounds(), want)
	}
}

func TestICOValidatesDirectoryAndPayloadBounds(t *testing.T) {
	valid := iconFile(pngIcon(t, 8, 8))
	for length := 0; length < len(valid); length++ {
		if _, err := decodeICOConfig(bytes.NewReader(valid[:length])); err == nil {
			t.Fatalf("accepted truncated icon of %d bytes", length)
		}
	}
	for _, field := range []int{6, 7, 9, 18} {
		data := bytes.Clone(valid)
		data[field]++ // dimensions, reserved byte, or an offset beyond its span
		if _, err := decodeICO(bytes.NewReader(data)); err == nil {
			t.Fatalf("accepted inconsistent directory field %d", field)
		}
	}
	oversizedDirectory := make([]byte, 6+16*(maxICOEntries+1))
	copy(oversizedDirectory, valid[:6])
	binary.LittleEndian.PutUint16(oversizedDirectory[4:6], maxICOEntries+1)
	if _, err := decodeICOConfig(bytes.NewReader(oversizedDirectory)); err == nil {
		t.Fatal("accepted excessive entry count")
	}
}

func TestICORejectionDoesNotUseRawPreviewFallback(t *testing.T) {
	data := append([]byte{0, 0, 1, 0, 0, 0}, encodeJPEG(t, 2, 2, color.White)...)
	u := storage.NewFileURI(writeTempFile(t, "invalid.ico", data))
	if _, _, err := ReadAndProbe(context.Background(), u); err == nil {
		t.Fatal("invalid ICO must fail admission despite an embedded preview")
	}
	if _, err := DecodeLoaded(context.Background(), data, 0); err == nil {
		t.Fatal("invalid ICO must fail decoding despite an embedded preview")
	}
}

func TestICOUsesOffsetsAndDecodesOnlySelectedEntry(t *testing.T) {
	small := pngIcon(t, 8, 8)
	// Configuration alone is sufficient for an unselected image; it has no
	// pixel stream. Decoding every entry would incorrectly fail this load.
	small.data = small.data[:33]
	large := pngIcon(t, 256, 256)
	data := iconFile(large, small)
	// Exchange directory records without moving their referenced payloads.
	first := bytes.Clone(data[6:22])
	copy(data[6:22], data[22:38])
	copy(data[22:38], first)
	for _, decode := range []func(*bytes.Reader) (image.Rectangle, error){
		func(r *bytes.Reader) (image.Rectangle, error) {
			cfg, err := decodeICOConfig(r)
			return image.Rect(0, 0, cfg.Width, cfg.Height), err
		},
		func(r *bytes.Reader) (image.Rectangle, error) {
			img, err := decodeICO(r)
			if err != nil {
				return image.Rectangle{}, err
			}
			return img.Bounds(), nil
		},
	} {
		bounds, err := decode(bytes.NewReader(data))
		if err != nil || bounds != image.Rect(0, 0, 256, 256) {
			t.Fatalf("zero-byte dimension/offset selection: %v, %v", bounds, err)
		}
	}
}

func dibIcon(bits int) icoFixture {
	const w, h = 2, 3
	palette := 0
	if bits <= 8 {
		palette = 4 * (1 << bits)
	}
	stride := (w*bits + 31) / 32 * 4
	data := make([]byte, 40+palette+stride*h+4*h)
	binary.LittleEndian.PutUint32(data[:4], 40)
	binary.LittleEndian.PutUint32(data[4:8], w)
	binary.LittleEndian.PutUint32(data[8:12], 2*h)
	binary.LittleEndian.PutUint16(data[12:14], 1)
	binary.LittleEndian.PutUint16(data[14:16], uint16(bits))
	if palette != 0 {
		data[42] = 255 // palette entry zero: red
	} else {
		for row := range h {
			for x := range w {
				data[40+row*stride+x*bits/8+2] = 255
			}
		}
	}
	data[40+palette+stride*h] = 0x80 // bottom left pixel is transparent
	return icoFixture{w, h, data}
}

func TestICODIBColorsMasksAndDimensions(t *testing.T) {
	for _, bits := range []int{1, 2, 4, 8, 24, 32} {
		entry := dibIcon(bits)
		data := iconFile(entry)
		img, err := decodeICO(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("%d-bit DIB: %v", bits, err)
		}
		if img.Bounds() != image.Rect(0, 0, entry.w, entry.h) {
			t.Fatalf("%d-bit non-square DIB: %v", bits, img.Bounds())
		}
		if got := color.NRGBAModel.Convert(img.At(1, 0)); got != (color.NRGBA{R: 255, A: 255}) {
			t.Fatalf("%d-bit color: %v", bits, got)
		}
		if _, _, _, a := img.At(0, 2).RGBA(); a != 0 {
			t.Fatalf("%d-bit mask: alpha %d", bits, a)
		}
		// A short mask must fail before either decoder or mask indexing.
		entry.data = entry.data[:len(entry.data)-1]
		if _, err := decodeICO(bytes.NewReader(iconFile(entry))); err == nil {
			t.Fatalf("accepted truncated %d-bit mask", bits)
		}
	}
}

func TestICO32BitAlpha(t *testing.T) {
	entry := dibIcon(32)
	entry.data[43] = 128
	entry.data = entry.data[:len(entry.data)-12] // optional mask omitted
	img, err := decodeICO(bytes.NewReader(iconFile(entry)))
	if err != nil {
		t.Fatal(err)
	}
	if got := color.NRGBAModel.Convert(img.At(0, 2)); got != (color.NRGBA{R: 255, A: 128}) {
		t.Fatalf("32-bit alpha: %v", got)
	}
}
