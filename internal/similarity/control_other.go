//go:build !windows && !(darwin && arm64) && !(linux && amd64)

package similarity

import (
	"fmt"
	"os"
)

func controlInput() (*os.File, error) {
	return nil, fmt.Errorf("local similarity requires Apple Silicon macOS or Linux amd64")
}
