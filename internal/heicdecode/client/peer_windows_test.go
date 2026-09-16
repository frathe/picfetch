//go:build windows && (amd64 || arm64)

package client

import (
	"io"
	"os"

	"github.com/frathe/picfetch/internal/heicdecode/winisolation"
)

func installOwnedPeer(source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { _ = input.Close() }()
	output, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	return winisolation.PrepareExecutable(destination)
}
