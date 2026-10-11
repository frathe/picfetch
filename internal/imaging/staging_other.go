//go:build !darwin || !appleappstore

package imaging

import "path/filepath"

func writeStagingDirectory(path string) (string, func(), error) {
	return filepath.Dir(path), func() {}, nil
}
