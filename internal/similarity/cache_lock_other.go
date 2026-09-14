//go:build !darwin && !linux && !windows

package similarity

import (
	"context"
	"fmt"
	"os"
)

func lockCacheFile(_ context.Context, _ *os.File) error {
	return fmt.Errorf("analysis cache locking is unavailable on this platform")
}
func unlockCacheFile(_ *os.File) {}
