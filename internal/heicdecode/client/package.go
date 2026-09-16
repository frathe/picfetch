package client

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/frathe/picfetch/internal/heicdecode"
)

// InstallationRoot discovers only the package containing the running binary.
// A macOS standalone executable is deliberately not a complete app package.
func InstallationRoot(executable, system string) (string, error) {
	if !filepath.IsAbs(executable) {
		return "", ErrUnavailable
	}
	executable, err := filepath.EvalSymlinks(executable)
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(executable)
	switch system {
	case "darwin":
		contents := filepath.Dir(dir)
		root := filepath.Dir(contents)
		if filepath.Base(dir) != "MacOS" || filepath.Base(contents) != "Contents" || !strings.HasSuffix(strings.ToLower(root), ".app") {
			return "", ErrUnavailable
		}
		return root, nil
	case "linux", "windows":
		return dir, nil
	default:
		return "", ErrUnavailable
	}
}

// OpenInstalled creates the one app-owned client from a complete local package.
// Preparation never chooses another decoder or downloads a missing helper.
func OpenInstalled(ctx context.Context, executable, privateDir string, limits heicdecode.Limits) (*Client, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	root, err := InstallationRoot(executable, runtime.GOOS)
	if err != nil {
		return nil, err
	}
	helper, digest, err := LoadPackage(root, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return nil, err
	}
	helper, release, err := prepareInstalled(ctx, helper, digest, privateDir)
	if err != nil {
		return nil, err
	}
	owner, err := New(Config{Executable: helper, SHA256: digest, Limits: limits})
	if err != nil {
		release()
		return nil, err
	}
	owner.release = release
	return owner, nil
}

// PackageManifest records the post-signing helper identity within a trusted
// installed package. It is not an independent signature: package authenticity
// belongs to the enclosing TUF archive or platform bundle/package signature.
// No manifest field selects a path or relaxes the runtime resource policy.
type PackageManifest struct {
	Version          int    `json:"version"`
	GOOS             string `json:"goos"`
	GOARCH           string `json:"goarch"`
	ExecutableSHA256 string `json:"executableSHA256"`
	GuestSHA256      string `json:"guestSHA256"`
}

// PackagePaths defines the one supported helper and manifest location for each
// OS. root is the .app bundle on macOS, or the installation directory elsewhere.
func PackagePaths(root, system string) (executable, manifest string, err error) {
	if !filepath.IsAbs(root) {
		return "", "", errors.New("HEIC package root must be absolute")
	}
	switch system {
	case "darwin":
		return filepath.Join(root, "Contents", "Helpers", "HEICWorker.app", "Contents", "MacOS", "picfetch-heic-worker"), filepath.Join(root, "Contents", "Resources", "heic", "manifest.json"), nil
	case "linux":
		return filepath.Join(root, "heic", "picfetch-heic-worker"), filepath.Join(root, "heic", "manifest.json"), nil
	case "windows":
		return filepath.Join(root, "heic", "picfetch-heic-worker.exe"), filepath.Join(root, "heic", "manifest.json"), nil
	default:
		return "", "", errors.New("unsupported HEIC package platform")
	}
}

// LoadPackage refuses absent/malformed/wrong-target packages and pins the
// installed executable. Client verifies that pin again at every launch. The
// caller supplies the trusted installation root, never an image-selected path.
func LoadPackage(root, system, architecture string) (string, [32]byte, error) {
	var empty [32]byte
	if architecture != "amd64" && architecture != "arm64" {
		return "", empty, ErrUnavailable
	}
	executable, manifest, err := PackagePaths(root, system)
	if err != nil {
		return "", empty, err
	}
	file, err := os.Open(manifest)
	if err != nil {
		return "", empty, err
	}
	data, readErr := io.ReadAll(io.LimitReader(file, 4097))
	closeErr := file.Close()
	if readErr != nil {
		return "", empty, readErr
	}
	if closeErr != nil {
		return "", empty, closeErr
	}
	if len(data) > 4096 {
		return "", empty, errors.New("oversized HEIC package manifest")
	}
	var record PackageManifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&record); err != nil {
		return "", empty, err
	}
	if err = decoder.Decode(new(any)); err != io.EOF {
		return "", empty, errors.New("trailing HEIC package manifest data")
	}
	if record.Version != 1 || record.GOOS != system || record.GOARCH != architecture {
		return "", empty, errors.New("HEIC package target mismatch")
	}
	pin, err := packageDigest(record.ExecutableSHA256)
	if err != nil {
		return "", empty, err
	}
	if _, err = packageDigest(record.GuestSHA256); err != nil {
		return "", empty, err
	}
	helper, err := os.Open(executable)
	if err != nil {
		return "", empty, err
	}
	defer func() { _ = helper.Close() }()
	info, err := helper.Stat()
	if err != nil {
		return "", empty, err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 64*1024*1024 {
		return "", empty, errors.New("invalid packaged HEIC executable")
	}
	hash := sha256.New()
	if _, err = io.Copy(hash, io.LimitReader(helper, 64*1024*1024+1)); err != nil {
		return "", empty, err
	}
	if !bytes.Equal(hash.Sum(nil), pin[:]) {
		return "", empty, errors.New("packaged HEIC executable hash mismatch")
	}
	return executable, pin, nil
}

func packageDigest(value string) ([32]byte, error) {
	var digest [32]byte
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != len(digest) || hex.EncodeToString(decoded) != value {
		return digest, fmt.Errorf("invalid HEIC package digest")
	}
	copy(digest[:], decoded)
	if digest == [32]byte{} {
		return digest, errors.New("empty HEIC package digest")
	}
	return digest, nil
}
