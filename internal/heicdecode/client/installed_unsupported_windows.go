//go:build windows && !amd64 && !arm64

package client

import "context"

func prepareInstalled(_ context.Context, _ string, _ [32]byte, _ string) (string, func(), error) {
	return "", nil, ErrUnavailable
}
