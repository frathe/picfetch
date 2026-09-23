package heic

import (
	_ "embed"
	"fmt"
	"io"
	"os"
)

//go:embed testdata/probe8.heic
var probe8 string

//go:embed testdata/probe10.heic
var probe10 string

// WorkerMain handles private native work before desktop startup. The marker is
// removed before processing so any nested invocation requires explicit dispatch.
func WorkerMain() bool {
	if os.Getenv(workerEnvironment) != "1" {
		return false
	}
	_ = os.Unsetenv(workerEnvironment)
	if err := restrictWorker(); err != nil {
		_ = writeResponse(os.Stdout, Result{}, fmt.Errorf("restrict HEIC worker: %w", err))
		return true
	}
	var request wireRequest
	if err := readHeader(os.Stdin, &request); err != nil {
		_ = writeResponse(os.Stdout, Result{}, err)
		return true
	}
	if request.Check {
		_ = writeResponse(os.Stdout, Result{}, checkNative())
		return true
	}
	if err := validateRequest(request.Request, int64(request.Encoded)); err != nil {
		_ = writeResponse(os.Stdout, Result{}, err)
		return true
	}
	data := make([]byte, request.Encoded)
	if _, err := io.ReadFull(os.Stdin, data); err != nil {
		_ = writeResponse(os.Stdout, Result{}, ErrInvalid)
		return true
	}
	result, err := nativeRead(data, request.Request)
	_ = writeResponse(os.Stdout, result, err)
	return true
}

func checkNative() error {
	for _, fixture := range []string{probe8, probe10} {
		result, err := nativeRead([]byte(fixture), Request{Pixels: true, MaxEncodedBytes: 64 * 1024, MaxPixels: 64 * 64})
		if err != nil {
			return err
		}
		if result.Width != 64 || result.Height != 64 || result.Stride != 256 || len(result.Pixels) != 64*64*4 {
			return fmt.Errorf("HEIC probe returned unexpected dimensions")
		}
		for _, sample := range []struct{ x, y, want int }{{8, 8, 0}, {48, 8, 255}, {8, 48, 0}, {48, 48, 255}} {
			pixel := result.Pixels[sample.y*result.Stride+sample.x*4:][:4]
			for _, value := range pixel[:3] {
				if int(value) < sample.want-2 || int(value) > sample.want+2 {
					return fmt.Errorf("HEIC probe returned incorrect grayscale pixels")
				}
			}
			if pixel[3] != 255 {
				return fmt.Errorf("HEIC probe returned incorrect alpha")
			}
		}
	}
	return nil
}
