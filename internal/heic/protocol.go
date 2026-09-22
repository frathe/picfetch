package heic

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const (
	headerLimit   = 16 * 1024
	metadataLimit = 1024 * 1024
	pixelLimit    = 200_000_000
	encodedLimit  = 2 * 1024 * 1024 * 1024
)

type wireRequest struct {
	Request
	Check   bool
	Encoded int
}

type wireResponse struct {
	Width, Height, Stride int
	PixelBytes, EXIFBytes int
	Provider              string
	Error, Message        string
}

func validateRequest(request Request, encoded int64) error {
	if request.MaxEncodedBytes <= 0 || request.MaxEncodedBytes > encodedLimit || request.MaxPixels <= 0 || request.MaxPixels > pixelLimit {
		return fmt.Errorf("%w: invalid resource policy", ErrInvalid)
	}
	if encoded <= 0 || encoded > request.MaxEncodedBytes {
		return fmt.Errorf("%w: encoded data exceeds limit", ErrInvalid)
	}
	return nil
}

func writeHeader(w io.Writer, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if len(data) > headerLimit {
		return ErrProtocol
	}
	if err := binary.Write(w, binary.BigEndian, uint32(len(data))); err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

func readHeader(r io.Reader, value any) error {
	var size uint32
	if err := binary.Read(r, binary.BigEndian, &size); err != nil {
		return err
	}
	if size == 0 || size > headerLimit {
		return ErrProtocol
	}
	data := make([]byte, size)
	if _, err := io.ReadFull(r, data); err != nil {
		return err
	}
	if err := json.Unmarshal(data, value); err != nil {
		return fmt.Errorf("%w: header JSON", ErrProtocol)
	}
	return nil
}

func readResponse(r io.Reader, request wireRequest) (Result, error) {
	var response wireResponse
	if err := readHeader(r, &response); err != nil {
		return Result{}, err
	}
	if response.Error != "" {
		var cause error
		switch response.Error {
		case "unavailable":
			cause = ErrUnavailable
		case "unsupported":
			cause = ErrUnsupported
		case "invalid":
			cause = ErrInvalid
		case "failed":
			return Result{}, fmt.Errorf("HEIC worker: %s", response.Message)
		default:
			return Result{}, ErrProtocol
		}
		return Result{}, fmt.Errorf("%w: %s", cause, response.Message)
	}
	if request.Check {
		if response.Width != 0 || response.Height != 0 || response.Stride != 0 || response.PixelBytes != 0 || response.EXIFBytes != 0 {
			return Result{}, ErrProtocol
		}
		return Result{}, requireEOF(r)
	}
	if response.Width <= 0 || response.Height <= 0 || response.Width > pixelLimit || response.Height > pixelLimit ||
		int64(response.Width)*int64(response.Height) > request.MaxPixels || response.Stride != response.Width*4 ||
		response.EXIFBytes < 0 || response.EXIFBytes > metadataLimit {
		return Result{}, ErrProtocol
	}
	want := int64(0)
	if request.Pixels {
		want = int64(response.Stride) * int64(response.Height)
	}
	if int64(response.PixelBytes) != want {
		return Result{}, ErrProtocol
	}
	result := Result{Width: response.Width, Height: response.Height, Stride: response.Stride, Provider: response.Provider,
		EXIF: make([]byte, response.EXIFBytes), Pixels: make([]byte, response.PixelBytes)}
	if _, err := io.ReadFull(r, result.EXIF); err != nil {
		return Result{}, fmt.Errorf("%w: truncated metadata: %v", ErrProtocol, err)
	}
	if _, err := io.ReadFull(r, result.Pixels); err != nil {
		return Result{}, fmt.Errorf("%w: truncated pixels: %v", ErrProtocol, err)
	}
	if err := requireEOF(r); err != nil {
		return Result{}, err
	}
	return result, nil
}

func requireEOF(r io.Reader) error {
	var extra [1]byte
	if n, err := r.Read(extra[:]); n != 0 || !errors.Is(err, io.EOF) {
		return ErrProtocol
	}
	return nil
}

func writeResponse(w io.Writer, result Result, err error) error {
	response := wireResponse{Width: result.Width, Height: result.Height, Stride: result.Stride,
		PixelBytes: len(result.Pixels), EXIFBytes: len(result.EXIF), Provider: result.Provider}
	if err != nil {
		response = wireResponse{Message: err.Error(), Error: "failed"}
		switch {
		case errors.Is(err, ErrUnavailable):
			response.Error = "unavailable"
		case errors.Is(err, ErrUnsupported):
			response.Error = "unsupported"
		case errors.Is(err, ErrInvalid):
			response.Error = "invalid"
		}
	}
	if err := writeHeader(w, response); err != nil {
		return err
	}
	if _, err := w.Write(result.EXIF); err != nil {
		return err
	}
	_, err = w.Write(result.Pixels)
	return err
}
