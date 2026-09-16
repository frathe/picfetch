package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	heicclient "github.com/frathe/picfetch/internal/heicdecode/client"
)

func TestDownloadedCompanionsRemainVerifiedAfterPersistence(t *testing.T) {
	helper := []byte("owned helper executable")
	digest := sha256.Sum256(helper)
	guest := sha256.Sum256([]byte("owned guest"))
	manifest, err := json.Marshal(heicclient.PackageManifest{Version: 1, GOOS: "linux", GOARCH: "amd64", ExecutableSHA256: hex.EncodeToString(digest[:]), GuestSHA256: hex.EncodeToString(guest[:])})
	if err != nil {
		t.Fatal(err)
	}
	archivePath := writeTarGz(t, t.TempDir(), "picfetch-linux-amd64.tar.gz", map[string][]byte{
		"picfetch-linux-amd64":      []byte("owned main executable"),
		"heic/picfetch-heic-worker": helper,
		"heic/manifest.json":        manifest,
		"heic/notices/h265-LICENSE": []byte("owned fixture notice"),
	})
	archive, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(archive)
	server := serveDownload(t, "picfetch-linux-amd64.tar.gz", archive, okAttest)
	stageDir := t.TempDir()
	release := Release{Version: "v0.2.6", AssetName: "picfetch-linux-amd64.tar.gz", AssetURL: server.URL + "/picfetch-linux-amd64.tar.gz", AssetDigest: hex.EncodeToString(sum[:])}
	staged, err := downloadClient(t, server, &fakeVerifier{}, stageDir).Download(context.Background(), release)
	if err != nil {
		t.Fatal(err)
	}
	if err = ValidateStage(staged); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadStage(stageDir)
	if err != nil {
		t.Fatal(err)
	}
	withoutProof := loaded
	withoutProof.verification.CompanionDigests = nil
	if err = ValidateStage(withoutProof); err == nil {
		t.Fatal("helper package accepted after removing its companion provenance")
	}
	companion := filepath.Join(filepath.Dir(loaded.BinaryPath), "heic", "notices", "h265-LICENSE")
	if err = os.WriteFile(companion, []byte("changed after archive verification"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = ValidateStage(loaded); err == nil {
		t.Fatal("changed companion accepted after provenance round trip")
	}
}

func TestApplyInstallsCompanionsAndRollsBackOnBinaryFailure(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "install", true: "binary failure"}[fail], func(t *testing.T) {
			directory := t.TempDir()
			installed := filepath.Join(directory, "installed", "picfetch")
			staged := filepath.Join(directory, "staged", "picfetch")
			for path, data := range map[string]string{installed: "old main", staged: "new main", filepath.Join(filepath.Dir(installed), "heic", "helper"): "old helper", filepath.Join(filepath.Dir(staged), "heic", "helper"): "new helper"} {
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(data), 0700); err != nil {
					t.Fatal(err)
				}
			}
			files, err := companionDigests(filepath.Join(filepath.Dir(staged), "heic"))
			if err != nil {
				t.Fatal(err)
			}
			stage := Stage{BinaryPath: staged, verification: stageVerification{GOOS: "linux", CompanionDigests: files}}
			if fail {
				if err = os.Remove(staged); err != nil {
					t.Fatal(err)
				}
			}
			err = Apply(stage, installed, ApplyOptions{})
			if (err != nil) != fail {
				t.Fatalf("apply: %v", err)
			}
			main, readErr := os.ReadFile(installed)
			if readErr != nil {
				t.Fatal(readErr)
			}
			helper, readErr := os.ReadFile(filepath.Join(filepath.Dir(installed), "heic", "helper"))
			if readErr != nil {
				t.Fatal(readErr)
			}
			wanted := "new"
			if fail {
				wanted = "old"
			}
			if string(main) != wanted+" main" || string(helper) != wanted+" helper" {
				t.Fatalf("installed main=%q helper=%q", main, helper)
			}
		})
	}
}
