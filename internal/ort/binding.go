//go:build cgo && !(darwin && amd64)

// Package ort selects the ONNX Runtime C API binding that matches the platform's
// pinned native library. It exposes only the operations used by PicFetch.
package ort

import upstream "github.com/yalue/onnxruntime_go"

type AdvancedSession = upstream.AdvancedSession
type SessionOptions = upstream.SessionOptions
type Shape = upstream.Shape
type Value = upstream.Value
type TensorData = upstream.TensorData
type Tensor[T TensorData] = upstream.Tensor[T]

func SetSharedLibraryPath(path string)   { upstream.SetSharedLibraryPath(path) }
func InitializeEnvironment() error       { return upstream.InitializeEnvironment() }
func DestroyEnvironment() error          { return upstream.DestroyEnvironment() }
func DisableTelemetry() error            { return upstream.DisableTelemetry() }
func NewShape(dimensions ...int64) Shape { return upstream.NewShape(dimensions...) }

func NewEmptyTensor[T TensorData](shape Shape) (*Tensor[T], error) {
	return upstream.NewEmptyTensor[T](shape)
}

func NewSessionOptions() (*SessionOptions, error) {
	return upstream.NewSessionOptions()
}

func NewAdvancedSession(path string, inputNames, outputNames []string,
	inputs, outputs []Value, options *SessionOptions) (*AdvancedSession, error) {
	return upstream.NewAdvancedSession(path, inputNames, outputNames, inputs, outputs, options)
}
