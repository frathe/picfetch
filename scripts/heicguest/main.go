//go:build wasip1 && wasm

// The development guest is WASI-only. It is not embedded in or launched by the
// application until mandatory host isolation and memory controls are qualified.
package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"io"
	"os"

	"github.com/gen2brain/h265/heic"

	"github.com/frathe/picfetch/internal/heicdecode"
)

func main() {
	limits := heicdecode.DefaultLimits(0)
	request, err := heicdecode.ReadRequest(os.Stdin, limits)
	if err != nil {
		_ = heicdecode.WriteFailure(os.Stdout, heicdecode.StatusRejected, "invalid request", limits)
		return
	}
	limits.MaxPixels = request.MaxPixels
	limits.MaxOutputBytes = request.MaxOutputBytes
	result, status := decode(request, limits)
	if status != 0 {
		_ = heicdecode.WriteFailure(os.Stdout, status, "decode refused", limits)
		return
	}
	if err = heicdecode.WriteResponse(os.Stdout, request.Operation, result, limits); err != nil {
		// A transport failure may leave a partial response. The parent rejects it.
		os.Exit(1)
	}
}

func decode(request heicdecode.Request, limits heicdecode.Limits) (result heicdecode.Response, status heicdecode.Status) {
	defer func() {
		if recover() != nil {
			result = heicdecode.Response{}
			status = heicdecode.StatusInternal
		}
	}()
	if status = stillContainer(request.Input); status != 0 {
		return result, status
	}
	reader := bytes.NewReader(request.Input)
	exif, err := heic.DecodeExif(reader)
	if err != nil && !errors.Is(err, heic.ErrNoExif) {
		return result, heicdecode.StatusRejected
	}
	if exif != nil {
		result.Metadata = &heicdecode.Metadata{Orientation: exif.Orientation, Make: exif.Make, Model: exif.Model, Software: exif.Software, DateTime: exif.DateTime, DateTimeOriginal: exif.DateTimeOriginal, ExposureTime: exif.ExposureTime, FNumber: exif.FNumber, ISOSpeed: exif.ISOSpeed, FocalLength: exif.FocalLength, Flash: exif.Flash, GPSLatitude: exif.GPSLatitude, GPSLongitude: exif.GPSLongitude, GPSAltitude: exif.GPSAltitude, Copyright: exif.Copyright, Artist: exif.Artist}
	}
	if request.Operation == heicdecode.DecodeExif {
		return result, 0
	}
	// Config also decodes so it reports transformed dimensions consistently.
	// A header-only transformed-config API must be qualified before optimizing it.
	if _, err = reader.Seek(0, io.SeekStart); err != nil {
		return result, heicdecode.StatusInternal
	}
	img, err := heic.Decode(reader, heic.Options{AutoRotate: true, FrameSizeLimit: int(limits.MaxPixels), Threads: 1})
	if err != nil {
		if errors.Is(err, heic.ErrUnsupported) {
			return result, heicdecode.StatusUnsupported
		}
		return result, heicdecode.StatusRejected
	}
	img, status = canonicalPixels(img, limits)
	if status != 0 {
		return heicdecode.Response{}, status
	}
	result.Config = image.Config{Width: img.Bounds().Dx(), Height: img.Bounds().Dy(), ColorModel: img.ColorModel()}
	if request.Operation == heicdecode.Decode {
		result.Image = img
	}
	return result, 0
}

// Reject all movie boxes, including a still item accompanied by a sequence. The
// upstream still API otherwise falls back to decoding every sequence frame.
// This scan runs exclusively inside the guest, before upstream container work.
func stillContainer(data []byte) heicdecode.Status {
	seenType, seenMeta := false, false
	for len(data) > 0 {
		if len(data) < 8 {
			return heicdecode.StatusRejected
		}
		size := uint64(binary.BigEndian.Uint32(data[:4]))
		kind := string(data[4:8])
		header := uint64(8)
		if size == 1 {
			if len(data) < 16 {
				return heicdecode.StatusRejected
			}
			size = binary.BigEndian.Uint64(data[8:16])
			header = 16
		} else if size == 0 {
			size = uint64(len(data))
		}
		if size < header || size > uint64(len(data)) {
			return heicdecode.StatusRejected
		}
		switch kind {
		case "moov":
			return heicdecode.StatusUnsupported
		case "ftyp":
			seenType = true
		case "meta":
			seenMeta = true
		}
		data = data[int(size):]
	}
	if !seenType || !seenMeta {
		return heicdecode.StatusRejected
	}
	return 0
}

func canonicalPixels(img image.Image, limits heicdecode.Limits) (image.Image, heicdecode.Status) {
	if img == nil {
		return nil, heicdecode.StatusRejected
	}
	var pix []byte
	var stride, size int
	var rect image.Rectangle
	switch m := img.(type) {
	case *image.NRGBA:
		if m == nil {
			return nil, heicdecode.StatusRejected
		}
		pix, stride, size, rect = m.Pix, m.Stride, 4, m.Rect
	case *image.NRGBA64:
		if m == nil {
			return nil, heicdecode.StatusRejected
		}
		pix, stride, size, rect = m.Pix, m.Stride, 8, m.Rect
	default:
		return nil, heicdecode.StatusUnsupported
	}
	width, height := int64(rect.Dx()), int64(rect.Dy())
	if width <= 0 || height <= 0 || width > limits.MaxPixels || height > limits.MaxPixels || width*height > limits.MaxPixels || width*height*int64(size) > limits.MaxOutputBytes {
		return nil, heicdecode.StatusResourceLimit
	}
	row := int(width) * size
	if stride < row || int64(stride)*(height-1)+int64(row) > int64(len(pix)) {
		return nil, heicdecode.StatusRejected
	}
	if rect.Min == (image.Point{}) && stride == row && len(pix) == row*int(height) {
		return img, 0
	}
	// Normalize padded/cropped output without color conversion or bit-depth loss.
	out := make([]byte, row*int(height))
	for y := 0; y < int(height); y++ {
		copy(out[y*row:(y+1)*row], pix[y*stride:y*stride+row])
	}
	bounds := image.Rect(0, 0, int(width), int(height))
	if size == 8 {
		return &image.NRGBA64{Pix: out, Stride: row, Rect: bounds}, 0
	}
	return &image.NRGBA{Pix: out, Stride: row, Rect: bounds}, 0
}
