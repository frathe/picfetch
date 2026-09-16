//go:build windows && (amd64 || arm64)

package worker

import (
	"context"
	"errors"
	"fmt"
	"net"
	"unsafe"

	"golang.org/x/sys/windows"

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
	server, err := windows.UTF16PtrFromString(host)
	if err != nil {
		return err
	}
	query := windows.NewLazySystemDLL("Firewallapi.dll").NewProc("NetworkIsolationDiagnoseConnectFailureAndGetInfo")
	if err = query.Find(); err != nil {
		return err
	}
	// NETISO_ERROR_TYPE: 0 means no isolation failure, 1..3 identify missing
	// network capabilities, and 4 is the invalid/sentinel value.
	reason := uint32(4)
	code, _, _ := query.Call(uintptr(unsafe.Pointer(server)), uintptr(unsafe.Pointer(&reason)))
	if code != 0 {
		return fmt.Errorf("query HEIC network isolation: %w", windows.Errno(code))
	}
	if reason < 1 || reason > 3 {
		return fmt.Errorf("HEIC network isolation did not confirm a missing capability: %d", reason)
	}
	return ctx.Err()
}
