package heicdecode

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"unicode/utf8"
)

const (
	responseMagic       = "PHR2"
	protocolVersion     = 2
	statusOK            = 0
	pixelFormatRGBA8    = 1 // Straight alpha; v1's premultiplied interpretation was never shipped.
	pixelFormatNRGBA64  = 2
	responseHeaderBytes = 40
)

var ErrInvalidResponse = errors.New("invalid HEIC helper response")

// Status classifies a bounded worker refusal, independently of transport errors.
type Status uint16

const (
	StatusRejected Status = 1 + iota
	StatusUnsupported
	StatusResourceLimit
	StatusInternal
	StatusUnavailable
)

// Failure is a validated worker refusal. It does not imply an OS memory control
// was installed or identify a process's termination cause.
type Failure struct {
	Status     Status
	Diagnostic string
}

func (f *Failure) Error() string {
	return fmt.Sprintf("HEIC worker status %d: %s", f.Status, f.Diagnostic)
}

// Response holds only fully validated pixels and normalized metadata.
type Response struct {
	Image    image.Image
	Config   image.Config
	Metadata *Metadata
}

type responseHeader struct {
	status                Status
	width, height, stride uint64
	depth, format         uint16
	pixels                uint64
	metadata, diagnostic  uint32
}

func parseResponseHeader(b []byte) (responseHeader, error) {
	if string(b[:4]) != responseMagic || binary.LittleEndian.Uint16(b[4:6]) != protocolVersion {
		return responseHeader{}, invalidResponse("magic or protocol version", nil)
	}
	return responseHeader{
		status: Status(binary.LittleEndian.Uint16(b[6:8])), width: uint64(binary.LittleEndian.Uint32(b[8:12])), height: uint64(binary.LittleEndian.Uint32(b[12:16])), stride: uint64(binary.LittleEndian.Uint32(b[16:20])), depth: binary.LittleEndian.Uint16(b[20:22]), format: binary.LittleEndian.Uint16(b[22:24]), pixels: binary.LittleEndian.Uint64(b[24:32]), metadata: binary.LittleEndian.Uint32(b[32:36]), diagnostic: binary.LittleEndian.Uint32(b[36:40]),
	}, nil
}

func (h responseHeader) bytes() []byte {
	b := make([]byte, responseHeaderBytes)
	copy(b, responseMagic)
	binary.LittleEndian.PutUint16(b[4:6], protocolVersion)
	binary.LittleEndian.PutUint16(b[6:8], uint16(h.status))
	binary.LittleEndian.PutUint32(b[8:12], uint32(h.width))
	binary.LittleEndian.PutUint32(b[12:16], uint32(h.height))
	binary.LittleEndian.PutUint32(b[16:20], uint32(h.stride))
	binary.LittleEndian.PutUint16(b[20:22], h.depth)
	binary.LittleEndian.PutUint16(b[22:24], h.format)
	binary.LittleEndian.PutUint64(b[24:32], h.pixels)
	binary.LittleEndian.PutUint32(b[32:36], h.metadata)
	binary.LittleEndian.PutUint32(b[36:40], h.diagnostic)
	return b
}

func (h responseHeader) validate(op Operation, l Limits) error {
	if !op.valid() || h.status > StatusUnavailable {
		return invalidResponse("operation or status", nil)
	}
	if h.metadata > l.MaxMetadataBytes || h.diagnostic > l.MaxDiagnosticBytes {
		return invalidResponse("bounded fields", nil)
	}
	if h.status != statusOK || op == DecodeExif {
		if h.width != 0 || h.height != 0 || h.stride != 0 || h.depth != 0 || h.format != 0 || h.pixels != 0 || (h.status != statusOK && h.metadata != 0) || (h.status == statusOK && h.diagnostic != 0) {
			return invalidResponse("non-pixel layout", nil)
		}
		return nil
	}
	if h.width == 0 || h.height == 0 || h.width > uint64(l.MaxPixels) || h.height > uint64(l.MaxPixels) || h.width*h.height > uint64(l.MaxPixels) {
		return invalidResponse("dimensions", nil)
	}
	var size uint64
	switch {
	case h.format == pixelFormatRGBA8 && h.depth == 8:
		size = 4
	case h.format == pixelFormatNRGBA64 && h.depth == 16:
		size = 8
	default:
		return invalidResponse("pixel format or depth", nil)
	}
	// Width and height are bounded above by 64M before either product.
	if h.stride != h.width*size || h.stride*h.height > uint64(l.MaxOutputBytes) || h.diagnostic != 0 {
		return invalidResponse("pixel layout", nil)
	}
	if (op == Decode && h.pixels != h.stride*h.height) || (op == DecodeConfig && h.pixels != 0) {
		return invalidResponse("pixel length", nil)
	}
	return nil
}

// ReadResponse checks every field before allocation and requires EOF before
// returning any image. The launcher owns cancellation/deadline and pipe closure.
func ReadResponse(r io.Reader, op Operation, limits Limits) (Response, error) {
	if err := limits.Validate(); err != nil {
		return Response{}, err
	}
	var b [responseHeaderBytes]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return Response{}, invalidResponse("header", err)
	}
	h, err := parseResponseHeader(b[:])
	if err != nil {
		return Response{}, err
	}
	if err = h.validate(op, limits); err != nil {
		return Response{}, err
	}
	// All three lengths are independently bounded; their sum fits a 32-bit int.
	payload := make([]byte, int(h.pixels)+int(h.metadata)+int(h.diagnostic))
	if _, err = io.ReadFull(r, payload); err != nil {
		return Response{}, invalidResponse("payload", err)
	}
	if err = requireEOF(r); err != nil {
		return Response{}, invalidResponse("trailing data", err)
	}
	if h.status != statusOK {
		if !utf8.Valid(payload) {
			return Response{}, invalidResponse("diagnostic encoding", nil)
		}
		return Response{}, &Failure{Status: h.status, Diagnostic: string(payload)}
	}
	metadata, err := readMetadata(payload[int(h.pixels):])
	if err != nil {
		return Response{}, invalidResponse("metadata", err)
	}
	result := Response{Metadata: metadata}
	if op == DecodeExif {
		return result, nil
	}
	model := color.NRGBAModel
	if h.depth == 16 {
		model = color.NRGBA64Model
	}
	result.Config = image.Config{Width: int(h.width), Height: int(h.height), ColorModel: model}
	if op == Decode {
		pix := payload[:int(h.pixels):int(h.pixels)]
		rect := image.Rect(0, 0, int(h.width), int(h.height))
		if h.depth == 16 {
			result.Image = &image.NRGBA64{Pix: pix, Stride: int(h.stride), Rect: rect}
		} else {
			result.Image = &image.NRGBA{Pix: pix, Stride: int(h.stride), Rect: rect}
		}
	}
	return result, nil
}

// WriteResponse streams canonical straight-alpha pixels after checking the
// same header invariants as the reader. It never encodes a second image format.
func WriteResponse(w io.Writer, op Operation, result Response, limits Limits) error {
	if err := limits.Validate(); err != nil {
		return err
	}
	h := responseHeader{}
	var pixels, metadata []byte
	if result.Metadata != nil {
		if err := result.Metadata.validate(); err != nil {
			return err
		}
		var err error
		metadata, err = json.Marshal(result.Metadata)
		if err != nil {
			return err
		}
		h.metadata = uint32(len(metadata))
	}
	if op == Decode {
		switch img := result.Image.(type) {
		case *image.NRGBA:
			if img == nil {
				return invalidResponse("nil image", nil)
			}
			h.width, h.height, h.stride = uint64(img.Rect.Dx()), uint64(img.Rect.Dy()), uint64(img.Stride)
			h.depth, h.format = 8, pixelFormatRGBA8
			pixels = img.Pix
		case *image.NRGBA64:
			if img == nil {
				return invalidResponse("nil image", nil)
			}
			h.width, h.height, h.stride = uint64(img.Rect.Dx()), uint64(img.Rect.Dy()), uint64(img.Stride)
			h.depth, h.format = 16, pixelFormatNRGBA64
			pixels = img.Pix
		default:
			return invalidResponse("image type", nil)
		}
		if result.Image.Bounds().Min != (image.Point{}) {
			return invalidResponse("image origin", nil)
		}
		h.pixels = uint64(len(pixels))
	} else if op == DecodeConfig {
		h.width, h.height = uint64(result.Config.Width), uint64(result.Config.Height)
		switch result.Config.ColorModel {
		case color.NRGBAModel:
			h.depth, h.format, h.stride = 8, pixelFormatRGBA8, h.width*4
		case color.NRGBA64Model:
			h.depth, h.format, h.stride = 16, pixelFormatNRGBA64, h.width*8
		default:
			return invalidResponse("config color model", nil)
		}
	}
	if err := h.validate(op, limits); err != nil {
		return err
	}
	for _, p := range [][]byte{h.bytes(), pixels, metadata} {
		if len(p) > 0 {
			if err := writeAll(w, p); err != nil {
				return err
			}
		}
	}
	return nil
}

// WriteFailure emits a fixed-schema refusal without image or metadata content.
func WriteFailure(w io.Writer, status Status, diagnostic string, limits Limits) error {
	if err := limits.Validate(); err != nil {
		return err
	}
	if status < StatusRejected || status > StatusUnavailable || !utf8.ValidString(diagnostic) || len(diagnostic) > int(limits.MaxDiagnosticBytes) {
		return invalidResponse("failure", nil)
	}
	h := responseHeader{status: status, diagnostic: uint32(len(diagnostic))}
	if err := writeAll(w, h.bytes()); err != nil {
		return err
	}
	return writeAll(w, []byte(diagnostic))
}

func invalidResponse(field string, cause error) error {
	if cause != nil {
		return fmt.Errorf("%w: %s: %v", ErrInvalidResponse, field, cause)
	}
	return fmt.Errorf("%w: %s", ErrInvalidResponse, field)
}
