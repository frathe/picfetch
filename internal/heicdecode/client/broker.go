package client

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"

	"github.com/frathe/picfetch/internal/heicdecode"
)

const (
	brokerIntent uint16 = 1 + iota
	brokerGrant
	brokerInput
	brokerResult
	maxServices = 8
)

type brokerHeader struct {
	kind      uint16
	operation heicdecode.Operation
	size      uint64
}

// Serve shares this Client's admission lane with one analysis connection.
// Its context belongs to the analysis process's lifetime. Stop closes active
// and idle connections; Wait joins their complete service work. No source path
// or native executable selection crosses this byte-only protocol.
func (c *Client) Serve(ctx context.Context, input io.ReadCloser, output io.WriteCloser) error {
	finish, err := c.beginService()
	if err != nil {
		_ = input.Close()
		_ = output.Close()
		return err
	}
	defer finish()
	return c.serve(ctx, input, output)
}

func (c *Client) beginService() (func(), error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil, context.Canceled
	}
	if c.services >= maxServices {
		return nil, ErrBusy
	}
	c.services++
	c.work.Add(1)
	return func() { c.mu.Lock(); c.services--; c.mu.Unlock(); c.work.Done() }, nil
}

func (c *Client) serve(ctx context.Context, input io.ReadCloser, output io.WriteCloser) error {
	closeIO := func() { _ = input.Close(); _ = output.Close() }
	defer closeIO()
	ctx, cancel := context.WithCancel(ctx)
	stopParent := watchCancellation(c.ctx, cancel)
	stopIO := watchCancellation(ctx, closeIO)
	defer func() { cancel(); stopIO(); stopParent() }()
	for {
		header, err := readBrokerHeader(input)
		if err != nil {
			return err
		}
		if header.kind != brokerIntent || header.size != 0 {
			return heicdecode.ErrInvalidRequest
		}
		err = c.serveRequest(ctx, header.operation, input, output, closeIO)
		if err != nil {
			return err
		}
	}
}

// Read one control byte while queued so a disconnected analysis peer can retire
// immediately. No encoded allocation or bulk read occurs before native-ready
// admission. Every exit joins this reader, including pre-admission refusal.
func (c *Client) serveRequest(ctx context.Context, op heicdecode.Operation, input io.Reader, output io.Writer, closeIO func()) (resultErr error) {
	ctx, cancel := context.WithCancel(ctx)
	stopIO := watchCancellation(ctx, closeIO)
	var first [1]byte
	var peekErr error
	peekDone := make(chan struct{})
	go func() {
		defer close(peekDone)
		_, peekErr = io.ReadFull(input, first[:])
		if peekErr != nil {
			cancel()
		}
	}()
	received := false
	defer func() {
		if resultErr != nil || !received {
			closeIO()
		}
		<-peekDone
		stopIO()
		cancel()
	}()
	resultErr = c.withAdmission(ctx, Background, func(ctx context.Context) error {
		// Include source transfer and downstream output in the admitted deadline.
		stopAdmittedIO := watchCancellation(ctx, closeIO)
		defer stopAdmittedIO()
		var response heicdecode.Response
		var inputErr error
		decodeErr := c.verifyExecutable(ctx)
		if decodeErr == nil {
			response, decodeErr = c.run(ctx, op, func(ctx context.Context, maxBytes int64) (data []byte, resultErr error) {
				defer func() { inputErr = resultErr }()
				if err := ctx.Err(); err != nil {
					return nil, err
				}
				if err := writeBrokerHeader(output, brokerHeader{kind: brokerGrant, operation: op, size: uint64(maxBytes)}); err != nil {
					return nil, err
				}
				<-peekDone
				if peekErr != nil {
					return nil, peekErr
				}
				reader := io.MultiReader(bytes.NewReader(first[:]), input)
				data, resultErr = readBrokerInput(reader, op, c.config.Limits)
				received = resultErr == nil
				return data, resultErr
			})
		}
		if inputErr != nil {
			return inputErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		return writeBrokerResult(output, op, response, decodeErr, c.config.Limits)
	})
	if resultErr == nil && !received {
		return io.EOF
	}
	return resultErr
}

func readBrokerInput(input io.Reader, op heicdecode.Operation, limits heicdecode.Limits) ([]byte, error) {
	encoded, err := readBrokerHeader(input)
	if err != nil {
		return nil, err
	}
	if encoded.kind != brokerInput || encoded.operation != op || encoded.size <= 24 || encoded.size > uint64(limits.MaxInputBytes)+24 {
		return nil, heicdecode.ErrInvalidRequest
	}
	frame := &io.LimitedReader{R: input, N: int64(encoded.size)}
	request, err := heicdecode.ReadRequest(frame, limits)
	if err != nil {
		return nil, err
	}
	if frame.N != 0 || request.Operation != op || request.MaxPixels != limits.MaxPixels || request.MaxOutputBytes != limits.MaxOutputBytes {
		return nil, heicdecode.ErrInvalidRequest
	}
	return request.Input, nil
}

func readBrokerHeader(r io.Reader) (brokerHeader, error) {
	var data [16]byte
	if _, err := io.ReadFull(r, data[:]); err != nil {
		return brokerHeader{}, err
	}
	h := brokerHeader{kind: binary.LittleEndian.Uint16(data[4:]), operation: heicdecode.Operation(binary.LittleEndian.Uint16(data[6:])), size: binary.LittleEndian.Uint64(data[8:])}
	if string(data[:4]) != "PHB1" || h.kind < brokerIntent || h.kind > brokerResult || h.operation < heicdecode.Decode || h.operation > heicdecode.DecodeExif {
		return brokerHeader{}, heicdecode.ErrInvalidRequest
	}
	return h, nil
}

func writeBrokerHeader(w io.Writer, h brokerHeader) error {
	if h.kind < brokerIntent || h.kind > brokerResult || h.operation < heicdecode.Decode || h.operation > heicdecode.DecodeExif {
		return heicdecode.ErrInvalidRequest
	}
	var data [16]byte
	copy(data[:4], "PHB1")
	binary.LittleEndian.PutUint16(data[4:], h.kind)
	binary.LittleEndian.PutUint16(data[6:], uint16(h.operation))
	binary.LittleEndian.PutUint64(data[8:], h.size)
	n, err := w.Write(data[:])
	if err == nil && n != len(data) {
		return io.ErrShortWrite
	}
	return err
}

func readBrokerResult(r io.Reader, size uint64, op heicdecode.Operation, limits heicdecode.Limits) (heicdecode.Response, error) {
	if size < 40 || size > uint64(limits.MaxOutputBytes)+uint64(limits.MaxMetadataBytes)+uint64(limits.MaxDiagnosticBytes)+40 {
		return heicdecode.Response{}, heicdecode.ErrInvalidResponse
	}
	frame := &io.LimitedReader{R: r, N: int64(size)}
	result, err := heicdecode.ReadResponse(frame, op, limits)
	if frame.N != 0 {
		return heicdecode.Response{}, heicdecode.ErrInvalidResponse
	}
	return result, err
}

func writeBrokerResult(w io.Writer, op heicdecode.Operation, result heicdecode.Response, decodeErr error, limits heicdecode.Limits) error {
	encode := func(w io.Writer) error {
		if decodeErr == nil {
			return heicdecode.WriteResponse(w, op, result, limits)
		}
		var failure *heicdecode.Failure
		if errors.As(decodeErr, &failure) {
			return heicdecode.WriteFailure(w, failure.Status, failure.Diagnostic, limits)
		}
		return heicdecode.WriteFailure(w, heicdecode.StatusUnavailable, "helper request failed", limits)
	}
	// Validation/counting touches only lengths and bounded metadata. Pixel
	// slices are streamed on the second pass, never copied into an envelope.
	var count byteCounter
	if err := encode(&count); err != nil {
		return err
	}
	if err := writeBrokerHeader(w, brokerHeader{kind: brokerResult, operation: op, size: uint64(count)}); err != nil {
		return err
	}
	return encode(w)
}

type byteCounter int64

func (c *byteCounter) Write(p []byte) (int, error) { *c += byteCounter(len(p)); return len(p), nil }
