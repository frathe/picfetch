//go:build !windows

package consolehelp

import "os"

func enableANSI(_ *os.File) (func(), error) {
	return func() {}, nil
}
