//go:build !linux || (!amd64 && !arm64)

package heic

func restrictWorker() error { return nil }
