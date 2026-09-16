package client

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestStagedHelperPublication(t *testing.T) {
	source := filepath.Join(t.TempDir(), "packaged.exe")
	data := []byte("owned packaged helper")
	if err := os.WriteFile(source, data, 0700); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(data)
	root := t.TempDir()
	prepared := 0
	prepare := func(path string) error {
		prepared++
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, data) {
			t.Fatal("permissions prepared before verified copy existed")
		}
		if path == source {
			t.Fatal("changed installed helper permissions")
		}
		return nil
	}
	staged, err := stageHelper(context.Background(), source, digest, root, prepare)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(staged, hex.EncodeToString(digest[:])) {
		t.Fatal("copy is not content addressed")
	}
	first, err := os.Stat(staged)
	if err != nil {
		t.Fatal(err)
	}
	reused, err := stageHelper(context.Background(), source, digest, root, prepare)
	if err != nil || reused != staged {
		t.Fatalf("reuse: %q %v", reused, err)
	}
	second, err := os.Stat(reused)
	if err != nil || !os.SameFile(first, second) {
		t.Fatal("valid copy was replaced")
	}
	if err = os.WriteFile(staged, []byte("damaged"), 0700); err != nil {
		t.Fatal(err)
	}
	repaired, err := stageHelper(context.Background(), source, digest, root, prepare)
	if err != nil || repaired != staged {
		t.Fatalf("repair: %q %v", repaired, err)
	}
	got, err := os.ReadFile(repaired)
	if err != nil || !bytes.Equal(got, data) || prepared != 3 {
		t.Fatal("damaged copy not repaired/prepared")
	}
	if err = os.WriteFile(source, []byte("changed installed source"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err = stageHelper(context.Background(), source, digest, t.TempDir(), prepare); err == nil {
		t.Fatal("published a changed source")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = stageHelper(ctx, source, digest, root, prepare); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
}

func TestStagedHelperPreparationFailure(t *testing.T) {
	source := filepath.Join(t.TempDir(), "packaged.exe")
	data := []byte("owned helper")
	if err := os.WriteFile(source, data, 0700); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	denied := errors.New("owned permission denial")
	_, err := stageHelper(context.Background(), source, sha256.Sum256(data), root, func(_ string) error { return denied })
	if !errors.Is(err, denied) {
		t.Fatalf("preparation error lost: %v", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 0 {
		t.Fatalf("failed preparation published or leaked a copy: %v, %v", entries, err)
	}
}
