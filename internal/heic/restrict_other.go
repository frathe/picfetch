//go:build (!linux && !windows) || (!amd64 && !arm64)

package heic

func restrictWorker() error { return nil }
