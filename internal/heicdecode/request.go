package heicdecode

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// Operation identifies a single still-image operation. Sequences are unsupported.
type Operation uint16

const (
	Decode Operation = 1 + iota
	DecodeConfig
	DecodeExif
	requestMagic       = "PHQ2"
	requestHeaderBytes = 24
)

var ErrInvalidRequest = errors.New("invalid HEIC helper request")

// Request carries bytes only. A caller must acquire family-wide admission before
// reading Input. This framing package does not grant worker admission.
type Request struct {
	Operation      Operation
	Input          []byte
	MaxPixels      int64
	MaxOutputBytes int64
}

func (o Operation) valid() bool { return o >= Decode && o <= DecodeExif }

// WriteRequest validates a bounded request before writing any bytes.
func WriteRequest(w io.Writer, request Request, limits Limits) error {
	if err := limits.Validate(); err != nil {
		return err
	}
	if request.MaxPixels == 0 {
		request.MaxPixels = limits.MaxPixels
	}
	if request.MaxOutputBytes == 0 {
		request.MaxOutputBytes = limits.MaxOutputBytes
	}
	if err := validateRequest(request.Operation, uint64(len(request.Input)), request.MaxPixels, request.MaxOutputBytes, limits); err != nil {
		return err
	}
	var header [requestHeaderBytes]byte
	copy(header[:4], requestMagic)
	binary.LittleEndian.PutUint16(header[4:6], protocolVersion)
	binary.LittleEndian.PutUint16(header[6:8], uint16(request.Operation))
	binary.LittleEndian.PutUint64(header[8:16], uint64(len(request.Input)))
	binary.LittleEndian.PutUint32(header[16:20], uint32(request.MaxPixels))
	binary.LittleEndian.PutUint32(header[20:24], uint32(request.MaxOutputBytes))
	if err := writeAll(w, header[:]); err != nil {
		return err
	}
	return writeAll(w, request.Input)
}

// ReadRequest reads one request and requires EOF. The launcher must close stdin
// after writing and enforce the wall-clock deadline while this function reads.
func ReadRequest(r io.Reader, limits Limits) (Request, error) {
	if err := limits.Validate(); err != nil {
		return Request{}, err
	}
	var header [requestHeaderBytes]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return Request{}, fmt.Errorf("%w: header: %v", ErrInvalidRequest, err)
	}
	if string(header[:4]) != requestMagic || binary.LittleEndian.Uint16(header[4:6]) != protocolVersion {
		return Request{}, ErrInvalidRequest
	}
	req := Request{Operation: Operation(binary.LittleEndian.Uint16(header[6:8])), MaxPixels: int64(binary.LittleEndian.Uint32(header[16:20])), MaxOutputBytes: int64(binary.LittleEndian.Uint32(header[20:24]))}
	n := binary.LittleEndian.Uint64(header[8:16])
	if err := validateRequest(req.Operation, n, req.MaxPixels, req.MaxOutputBytes, limits); err != nil {
		return Request{}, err
	}
	req.Input = make([]byte, int(n))
	if _, err := io.ReadFull(r, req.Input); err != nil {
		return Request{}, fmt.Errorf("%w: input: %v", ErrInvalidRequest, err)
	}
	if err := requireEOF(r); err != nil {
		return Request{}, fmt.Errorf("%w: trailing data: %v", ErrInvalidRequest, err)
	}
	return req, nil
}

func validateRequest(op Operation, n uint64, pixels, output int64, limits Limits) error {
	if !op.valid() || n == 0 || n > uint64(limits.MaxInputBytes) || pixels <= 0 || pixels > limits.MaxPixels || output <= 0 || output > limits.MaxOutputBytes {
		return ErrInvalidRequest
	}
	return nil
}

func requireEOF(r io.Reader) error {
	var trailing [1]byte
	if n, err := r.Read(trailing[:]); n != 0 || !errors.Is(err, io.EOF) {
		return errors.New("expected EOF")
	}
	return nil
}

func writeAll(w io.Writer, p []byte) error {
	n, err := w.Write(p)
	if err == nil && n != len(p) {
		return io.ErrShortWrite
	}
	return err
}
