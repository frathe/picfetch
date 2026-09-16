package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/frathe/picfetch/internal/heicdecode/client"
)

func TestFinalizeRefusesWrongBinaryTargetBeforeManifest(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	wrong := "windows"
	if runtime.GOOS == wrong {
		wrong = "linux"
	}
	root := t.TempDir()
	target, manifest, err := client.PackagePaths(root, wrong)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		t.Fatal(err)
	}
	if err = copyFile(executable, target); err != nil {
		t.Fatal(err)
	}
	if err = stage(options{mode: "finalize", root: "../..", out: root, system: wrong, architecture: runtime.GOARCH}); err == nil {
		t.Fatal("wrong binary format accepted")
	}
	if _, err = os.Stat(manifest); !os.IsNotExist(err) {
		t.Fatalf("invalid package created a manifest: %v", err)
	}
}

func TestPackageRefusesIncompleteBuildTarget(t *testing.T) {
	for _, opts := range []options{
		{mode: "finalize", out: t.TempDir(), system: "linux", architecture: "386"},
		{mode: "unknown", out: t.TempDir(), system: "linux", architecture: "amd64"},
		{mode: "build", system: "linux", architecture: "amd64"},
	} {
		if err := stage(opts); err == nil {
			t.Fatalf("invalid options accepted: %+v", opts)
		}
	}
}
