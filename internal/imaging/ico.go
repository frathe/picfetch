package imaging

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"

	"golang.org/x/image/bmp"
)

// Icons have at most 256 pixels per axis. Separate encoded and directory
// limits keep configuration probing bounded even when entries share payloads.
const (
	maxICOBytes   = 16 * 1024 * 1024
	maxICOEntries = 256
)

var errInvalidICO = errors.New("invalid or unsupported ICO")

// The desktop driver also registers an ICO decoder. Dispatch explicitly so
// the viewer's admission cannot depend on global decoder registration order.
func decodeRasterConfig(data []byte) (image.Config, string, error) {
	if bytes.HasPrefix(data, []byte("\x00\x00\x01\x00")) {
		cfg, err := decodeICOConfig(bytes.NewReader(data))
		return cfg, "ico", err
	}
	return image.DecodeConfig(bytes.NewReader(data))
}

func decodeRaster(data []byte) (image.Image, string, error) {
	if bytes.HasPrefix(data, []byte("\x00\x00\x01\x00")) {
		img, err := decodeICO(bytes.NewReader(data))
		return img, "ico", err
	}
	return image.Decode(bytes.NewReader(data))
}

type icoImage struct {
	data                  []byte
	w, h                  int
	header, palette, bits int // zero header denotes PNG
	stride, pixelEnd      int
}

func readICO(r io.Reader) (icoImage, error) {
	data, err := io.ReadAll(io.LimitReader(r, maxICOBytes+1))
	if err != nil {
		return icoImage{}, err
	}
	if len(data) < 6 || len(data) > maxICOBytes || string(data[:4]) != "\x00\x00\x01\x00" {
		return icoImage{}, errInvalidICO
	}
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	directoryEnd := 6 + 16*count
	if count == 0 || count > maxICOEntries || directoryEnd > len(data) {
		return icoImage{}, errInvalidICO
	}
	var selected icoImage
	var examined uint64
	for i := range count {
		entry := data[6+16*i : 6+16*(i+1)]
		size := uint64(binary.LittleEndian.Uint32(entry[8:12]))
		offset := uint64(binary.LittleEndian.Uint32(entry[12:16]))
		examined += size
		if size == 0 || offset < uint64(directoryEnd) || offset > uint64(len(data)) || size > uint64(len(data))-offset || examined > maxICOBytes || entry[3] != 0 {
			return icoImage{}, errInvalidICO
		}
		w, h := int(entry[0]), int(entry[1])
		if w == 0 {
			w = 256
		}
		if h == 0 {
			h = 256
		}
		candidate, err := inspectICOImage(data[offset:offset+size], w, h)
		if err != nil {
			return icoImage{}, err
		}
		if w*h > selected.w*selected.h {
			selected = candidate
		}
	}
	return selected, nil
}

func inspectICOImage(data []byte, w, h int) (icoImage, error) {
	entry := icoImage{data: data, w: w, h: h}
	if bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
		cfg, err := png.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			return icoImage{}, err
		}
		if cfg.Width != w || cfg.Height != h {
			return icoImage{}, errInvalidICO
		}
		return entry, nil
	}
	if len(data) < 40 {
		return icoImage{}, errInvalidICO
	}
	// Accept the common Windows uncompressed DIB layouts. Other header types,
	// compression methods and channel masks need their own qualified adapter.
	header := binary.LittleEndian.Uint32(data[:4])
	if (header != 40 && header != 108 && header != 124) || uint64(header) > uint64(len(data)) ||
		binary.LittleEndian.Uint32(data[4:8]) != uint32(w) ||
		binary.LittleEndian.Uint32(data[8:12]) != uint32(2*h) ||
		binary.LittleEndian.Uint16(data[12:14]) != 1 || binary.LittleEndian.Uint32(data[16:20]) != 0 {
		return icoImage{}, errInvalidICO
	}
	bits := int(binary.LittleEndian.Uint16(data[14:16]))
	colors := binary.LittleEndian.Uint32(data[32:36])
	switch bits {
	case 1, 2, 4, 8:
		if colors == 0 {
			colors = 1 << bits
		}
		if colors > 1<<bits {
			return icoImage{}, errInvalidICO
		}
	case 24, 32:
		if colors != 0 {
			return icoImage{}, errInvalidICO
		}
	default:
		return icoImage{}, errInvalidICO
	}
	entry.header, entry.palette, entry.bits = int(header), int(colors)*4, bits
	entry.stride = (w*bits + 31) / 32 * 4
	entry.pixelEnd = entry.header + entry.palette + entry.stride*h
	maskBytes := (w + 31) / 32 * 4 * h
	if entry.pixelEnd > len(data) || (len(data)-entry.pixelEnd < maskBytes && (bits != 32 || len(data) != entry.pixelEnd)) {
		return icoImage{}, errInvalidICO
	}
	return entry, nil
}

func decodeICOConfig(r io.Reader) (image.Config, error) {
	entry, err := readICO(r)
	if err != nil {
		return image.Config{}, err
	}
	return image.Config{ColorModel: color.NRGBAModel, Width: entry.w, Height: entry.h}, nil
}

func decodeICO(r io.Reader) (image.Image, error) {
	entry, err := readICO(r)
	if err != nil {
		return nil, err
	}
	if entry.header == 0 {
		return png.Decode(bytes.NewReader(entry.data))
	}
	// Supply a normalized BITMAPINFOHEADER with the actual image height to the
	// existing BMP decoder. Neither source offsets nor sizes reach it unchecked.
	var header [54]byte
	copy(header[:2], "BM")
	copy(header[14:], entry.data[:40])
	binary.LittleEndian.PutUint32(header[2:6], uint32(54+entry.palette+entry.stride*entry.h))
	binary.LittleEndian.PutUint32(header[10:14], uint32(54+entry.palette))
	binary.LittleEndian.PutUint32(header[14:18], 40)
	binary.LittleEndian.PutUint32(header[22:26], uint32(entry.h))
	binary.LittleEndian.PutUint32(header[34:38], uint32(entry.stride*entry.h))
	pixels, err := bmp.Decode(io.MultiReader(bytes.NewReader(header[:]), bytes.NewReader(entry.data[entry.header:entry.pixelEnd])))
	if err != nil {
		return nil, err
	}
	out := image.NewNRGBA(image.Rect(0, 0, entry.w, entry.h))
	draw.Draw(out, out.Bounds(), pixels, image.Point{}, draw.Src)
	// A 32-bit icon with any nonzero alpha uses its alpha channel. Legacy
	// all-zero alpha icons instead use the one-bit AND mask, or stay opaque.
	hasAlpha := false
	if entry.bits == 32 {
		for p := entry.header + entry.palette + 3; p < entry.pixelEnd; p += 4 {
			hasAlpha = hasAlpha || entry.data[p] != 0
		}
	}
	mask := entry.data[entry.pixelEnd:]
	maskStride := (entry.w + 31) / 32 * 4
	for y := range entry.h {
		row := entry.h - 1 - y
		for x := range entry.w {
			alpha := byte(255)
			if hasAlpha {
				alpha = entry.data[entry.header+entry.palette+row*entry.stride+x*4+3]
			} else if len(mask) != 0 && mask[row*maskStride+x/8]&(0x80>>uint(x%8)) != 0 {
				alpha = 0
			}
			out.Pix[y*out.Stride+x*4+3] = alpha
		}
	}
	return out, nil
}
