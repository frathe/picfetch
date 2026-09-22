//go:build !linux && !darwin && !windows

package heic

func systemVersion() string { return "" }
