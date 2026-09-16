package client

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPackagedHelperPinsOnlyFixedTarget(t *testing.T) {
	for _, system := range []string{"darwin", "linux", "windows"} {
		t.Run(system, func(t *testing.T) {
			root := t.TempDir()
			executable, manifest, err := PackagePaths(root, system)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.MkdirAll(filepath.Dir(executable), 0700); err != nil {
				t.Fatal(err)
			}
			if err = os.MkdirAll(filepath.Dir(manifest), 0700); err != nil {
				t.Fatal(err)
			}
			helper := []byte("owned package executable")
			if err = os.WriteFile(executable, helper, 0700); err != nil {
				t.Fatal(err)
			}
			digest := sha256.Sum256(helper)
			guest := sha256.Sum256([]byte("owned package guest"))
			record := PackageManifest{Version: 1, GOOS: system, GOARCH: "arm64", ExecutableSHA256: hex.EncodeToString(digest[:]), GuestSHA256: hex.EncodeToString(guest[:])}
			data, err := json.Marshal(record)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(manifest, data, 0600); err != nil {
				t.Fatal(err)
			}
			got, pin, err := LoadPackage(root, system, "arm64")
			if err != nil || got != executable || pin != digest {
				t.Fatalf("package: path=%q pin=%x error=%v", got, pin, err)
			}
			if _, _, err = LoadPackage(root, system, "amd64"); err == nil {
				t.Fatal("wrong architecture accepted")
			}
			if err = os.WriteFile(executable, []byte("changed helper"), 0700); err != nil {
				t.Fatal(err)
			}
			if _, _, err = LoadPackage(root, system, "arm64"); err == nil {
				t.Fatal("changed helper accepted")
			}
		})
	}
}

func TestPackagedHelperRejectsUntrustedManifestChoices(t *testing.T) {
	root := t.TempDir()
	if _, _, err := LoadPackage(root, "linux", "amd64"); err == nil {
		t.Fatal("missing package accepted")
	}
	_, manifest, err := PackagePaths(root, "linux")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(filepath.Dir(manifest), 0700); err != nil {
		t.Fatal(err)
	}
	for _, data := range []string{`{}`, `{"Version":1,"Executable":"/other/program"}`, string(make([]byte, 4097))} {
		if err = os.WriteFile(manifest, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if _, _, err = LoadPackage(root, "linux", "amd64"); err == nil {
			t.Fatalf("invalid manifest accepted: %.80q", data)
		}
	}
	if _, _, err = PackagePaths(root, "unsupported"); err == nil {
		t.Fatal("unsupported package target accepted")
	}
	if _, _, err = PackagePaths("relative", "linux"); err == nil {
		t.Fatal("relative installation root accepted")
	}
}
