//go:build !darwin || !amd64

package similarity

func runtimeVersionSupported() bool { return true }
