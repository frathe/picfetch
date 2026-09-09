//go:build !darwin || !arm64

package similarity

import (
	"fmt"
	"os"
)

func controlInput() (*os.File, error) {
	return nil, fmt.Errorf("local similarity currently requires Apple Silicon macOS")
}
