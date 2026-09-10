package similarity

import (
	"context"
	"errors"
	"fmt"
	"net"
	"syscall"
	"time"
)

// VerifyOffline checks OS-enforced network denial without sending an image,
// path, or payload. A timeout, refused port, offline network, or environment flag
// cannot substitute for OS denial.
func VerifyOffline(ctx context.Context) error {
	for _, network := range []string{"tcp4", "udp4"} {
		dialer := net.Dialer{Timeout: time.Second}
		connection, err := dialer.DialContext(ctx, network, "192.0.2.1:443")
		if err == nil {
			if network == "udp4" {
				_, err = connection.Write(nil)
			}
			_ = connection.Close()
		}
		if !errors.Is(err, syscall.EPERM) && !errors.Is(err, syscall.EACCES) {
			return fmt.Errorf("offline denial not verified for %s: %v", network, err)
		}
	}
	return ctx.Err()
}
