package ort

import (
	"go/build"
	"slices"
	"testing"
)

// Both wrappers define the same C symbols. Every target must select exactly
// one, with API 23 confined to Intel macOS's last official runtime.
func TestPlatformBinding(t *testing.T) {
	for _, goos := range []string{"darwin", "linux", "windows"} {
		for _, arch := range []string{"amd64", "arm64"} {
			t.Run(goos+"/"+arch, func(t *testing.T) {
				context := build.Default
				context.GOOS, context.GOARCH, context.CgoEnabled = goos, arch, true
				pkg, err := context.ImportDir(".", build.ImportComment)
				if err != nil {
					t.Fatal(err)
				}
				legacy := "github.com/frathe/picfetch/internal/ortlegacy"
				current := "github.com/yalue/onnxruntime_go"
				want, reject := current, legacy
				if goos == "darwin" && arch == "amd64" {
					want, reject = legacy, current
				}
				if !slices.Contains(pkg.Imports, want) || slices.Contains(pkg.Imports, reject) {
					t.Fatalf("imports %v; want only binding %s", pkg.Imports, want)
				}
			})
		}
	}
}
