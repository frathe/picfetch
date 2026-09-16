//go:build !windows || (!amd64 && !arm64)

package worker

import (
	"context"
	"errors"
)

func confirmNetworkDenial(_ context.Context, _ string, _ error) error {
	return errors.New("explicit native permission refusal required")
}
