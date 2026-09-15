package heicdecode

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"io"
)

const (
	responseMagic       = "PHR1"
	protocolVersion     = 1
	statusOK            = 0
	pixelFormatRGBA8    = 1
	responseHeaderBytes = 40
)

var ErrInvalidResponse = errors.New("invalid HEIC helper response")

type response struct {
	Image      *image.RGBA
	Metadata   []byte
	Diagnostic string
}

func decodeResponse(reader io.Reader, limits Limits) (response, error) {
	if err := limits.Validate(); err != nil {
		return response{}, err
	}
	header := make([]byte, responseHeaderBytes)
	if _, err := io.ReadFull(reader, header); err != nil {
		return response{}, invalidResponse("header", err)
	}
	if string(header[:4]) != responseMagic || binary.LittleEndian.Uint16(header[4:6]) != protocolVersion {
		return response{}, invalidResponse("magic or protocol version", nil)
	}
	if binary.LittleEndian.Uint16(header[6:8]) != statusOK {
		return response{}, invalidResponse("status", nil)
	}
	width := uint64(binary.LittleEndian.Uint32(header[8:12]))
	height := uint64(binary.LittleEndian.Uint32(header[12:16]))
	stride := uint64(binary.LittleEndian.Uint32(header[16:20]))
	depth := binary.LittleEndian.Uint16(header[20:22])
	format := binary.LittleEndian.Uint16(header[22:24])
	pixelLength := binary.LittleEndian.Uint64(header[24:32])
	metadataLength := binary.LittleEndian.Uint32(header[32:36])
	diagnosticLength := binary.LittleEndian.Uint32(header[36:40])

	if width == 0 || height == 0 || width > uint64(limits.MaxPixels) || height > uint64(limits.MaxPixels) ||
		width > ^uint64(0)/height || width*height > uint64(limits.MaxPixels) {
		return response{}, invalidResponse("dimensions", nil)
	}
	if format != pixelFormatRGBA8 || depth != 8 || width > ^uint64(0)/4 || stride != width*4 ||
		stride > ^uint64(0)/height || pixelLength != stride*height || pixelLength > uint64(limits.MaxOutputBytes) {
		return response{}, invalidResponse("pixel layout", nil)
	}
	if metadataLength > limits.MaxMetadataBytes || diagnosticLength > limits.MaxDiagnosticBytes {
		return response{}, invalidResponse("bounded fields", nil)
	}
	total := pixelLength + uint64(metadataLength) + uint64(diagnosticLength)
	if total < pixelLength || total > uint64(limits.MaxOutputBytes)+uint64(limits.MaxMetadataBytes)+uint64(limits.MaxDiagnosticBytes) {
		return response{}, invalidResponse("payload length", nil)
	}
	payload := make([]byte, int(total))
	if _, err := io.ReadFull(reader, payload); err != nil {
		return response{}, invalidResponse("payload", err)
	}
	var trailing [1]byte
	if n, err := reader.Read(trailing[:]); n != 0 || !errors.Is(err, io.EOF) {
		return response{}, invalidResponse("trailing data", err)
	}
	pixelEnd := int(pixelLength)
	metadataEnd := pixelEnd + int(metadataLength)
	return response{
		Image:      &image.RGBA{Pix: payload[:pixelEnd:pixelEnd], Stride: int(stride), Rect: image.Rect(0, 0, int(width), int(height))},
		Metadata:   append([]byte(nil), payload[pixelEnd:metadataEnd]...),
		Diagnostic: string(payload[metadataEnd:]),
	}, nil
}

func invalidResponse(field string, cause error) error {
	if cause != nil {
		return fmt.Errorf("%w: %s: %v", ErrInvalidResponse, field, cause)
	}
	return fmt.Errorf("%w: %s", ErrInvalidResponse, field)
}
