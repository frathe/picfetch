//go:build heicnative && (darwin || linux) && (amd64 || arm64)

package ui

import (
	"os"
	"testing"
)

func verifyNativeHEICIdentity(t *testing.T) {
	t.Helper()
	t.Logf("native execution uid=%d", os.Getuid())
}
func preserveNativeHEICInstallation(_ *testing.T, _ string) func() { return func() {} }
