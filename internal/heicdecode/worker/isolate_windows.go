//go:build windows && (amd64 || arm64)

package worker

import (
	"context"
	"errors"
	"net"

	"github.com/frathe/picfetch/internal/heicdecode"
	"github.com/frathe/picfetch/internal/heicdecode/winisolation"
)

func isolate(limits heicdecode.Limits) (int64, error) {
	if err := winisolation.Verify(limits); err != nil {
		return 0, err
	}
	return limits.OSProcessBytes, nil
}

// Windows drops blocked AppContainer loopback packets. A timeout therefore
// needs independent policy evidence; it never suffices by itself.
func confirmNetworkDenial(ctx context.Context, host string, probeErr error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	var networkErr net.Error
	if host != "127.0.0.1" || !errors.As(probeErr, &networkErr) || !networkErr.Timeout() {
		return errors.New("expected a timed-out owned loopback probe")
	}
	// Start already refused an explicit loopback exemption before creation.
	// That privileged query is denied inside the zero-capability container.
	if err := winisolation.VerifyAppContainer(); err != nil {
		return err
	}
	return ctx.Err()
}
