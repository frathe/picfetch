//go:build !windows && !((darwin || linux) && (amd64 || arm64))

package similarity

import (
	"fmt"
	"os"
)

func controlInput() (*os.File, error) {
	return nil, fmt.Errorf("local similarity requires macOS/Linux/Windows amd64/arm64")
}
