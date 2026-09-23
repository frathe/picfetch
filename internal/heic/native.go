//go:build (!linux && !darwin && !windows) || (!windows && !cgo) || (!amd64 && !arm64)

package heic

func nativeRead(_ []byte, _ Request) (Result, error) { return Result{}, ErrUnavailable }
