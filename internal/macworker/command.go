// Package macworker launches the bundled broker for the Mac App Store's
// network-free XPC service. Packaging supplies both signed executables.
package macworker

import (
	"context"
	"os/exec"
	"path/filepath"
	"syscall"
)

// Mode selects one private worker protocol; the native service validates it.
type Mode string

const (
	HEIC       Mode = "heic"
	Similarity Mode = "similarity"
)

// Command selects the broker beside the main application executable. It sends
// termination rather than killing the broker so Wait includes service cleanup.
func Command(ctx context.Context, executable string, mode Mode) *exec.Cmd {
	cmd := exec.CommandContext(ctx, filepath.Join(filepath.Dir(executable), "picfetch-worker-client"), string(mode))
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	return cmd
}
