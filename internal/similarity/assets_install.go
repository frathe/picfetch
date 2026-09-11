package similarity

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/frathe/picfetch/internal/distribution"
)

// DownloadProgress reports aggregate transfer bytes, without source-image data.
type DownloadProgress struct {
	Received, Total int64
}

// ErrBundledRuntimeUnavailable requires package repair, never a code download.
var ErrBundledRuntimeUnavailable = errors.New("bundled runtime is unavailable; repair or update PicFetch through Microsoft Store")

type assetDownload struct {
	name, address, digest string
	size                  int64
}

func assetDownloads(asset runtimeAsset, bundled bool) []assetDownload {
	model := "https://huggingface.co/onnx-community/siglip2-base-patch16-224-ONNX/resolve/" + ModelRevision
	downloads := []assetDownload{
		{"vision_model.onnx", model + "/onnx/vision_model.onnx", "c0573e3f4140c3a7c4e9cc5912bd6b26a033b46a6a8e8af26cbea262b163bcad", 371807752},
		{"preprocessor_config.json", model + "/preprocessor_config.json", "9b36b57ebaf20f09bf4c22100ccc21877ea6bfe5aead0c00c59f8af8ccefacfc", 394},
	}
	if !bundled {
		downloads = append(downloads, asset.download())
	}
	return downloads
}

func (a runtimeAsset) download() assetDownload {
	version := a.version
	if version == "" {
		version = "1.29.0"
	}
	return assetDownload{"runtime." + a.archiveExtension, "https://github.com/microsoft/onnxruntime/releases/download/v" + version + "/" + a.directory + "." + a.archiveExtension, a.digest, a.size}
}

// CheckAssets reads and verifies local assets without making network requests.
func (c Client) CheckAssets(ctx context.Context) error {
	root, err := c.assetDirectory()
	if err != nil {
		return err
	}
	return VerifyAssets(ctx, root)
}

func (c Client) assetDirectory() (string, error) {
	if c.Assets != "" {
		return c.Assets, nil
	}
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	return defaultAssets(executable), nil
}

func userAssetDirectory() (string, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cache, "picfetch", "similarity-assets", ModelRevision), nil
}

// InstallAssets is an explicit download operation. Analyze never calls it.
// Only verified files are published; a canceled or rejected transfer removes its
// staging directory. Existing installations are left usable until publication.
func (c Client) InstallAssets(ctx context.Context, progress func(DownloadProgress)) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	assetRuntime, err := currentRuntime()
	if err != nil {
		return "", err
	}
	if distribution.StoreManaged {
		nativeRoot, err := runtimeDirectory("")
		if err != nil {
			return "", err
		}
		if err := verifyRuntime(ctx, nativeRoot, assetRuntime); err != nil {
			return "", fmt.Errorf("%w: %w", ErrBundledRuntimeUnavailable, err)
		}
	}
	if err := c.CheckAssets(ctx); err == nil {
		return c.assetDirectory()
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	root := c.Assets
	if root == "" {
		root = os.Getenv("PICFETCH_SIMILARITY_ASSETS")
	}
	if root == "" {
		var err error
		root, err = userAssetDirectory()
		if err != nil {
			return "", err
		}
	}
	if err := os.MkdirAll(filepath.Dir(root), 0700); err != nil {
		return "", err
	}
	staging, err := os.MkdirTemp(filepath.Dir(root), ".picfetch-model-")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(staging) }()

	downloads := assetDownloads(assetRuntime, distribution.StoreManaged)
	client := http.Client{Timeout: 30 * time.Minute}
	if c.HTTPClient != nil {
		client = *c.HTTPClient
	}
	client.Jar = nil
	client.CheckRedirect = func(request *http.Request, via []*http.Request) error {
		if len(via) >= 10 || !assetDownloadHost(request.URL) {
			return fmt.Errorf("asset download redirected outside its HTTPS providers")
		}
		return nil
	}
	var received int64
	for _, asset := range downloads {
		if err := downloadAsset(ctx, &client, staging, asset, func(n int) {
			received += int64(n)
			if progress != nil {
				progress(DownloadProgress{Received: received, Total: AssetDownloadBytes()})
			}
		}); err != nil {
			return "", err
		}
	}
	var files []string
	if !distribution.StoreManaged {
		files, err = unpackRuntime(ctx, staging)
		if err != nil {
			return "", err
		}
	}
	if err := VerifyAssets(ctx, staging); err != nil {
		return "", err
	}
	files = append([]string{"vision_model.onnx", "preprocessor_config.json"}, files...)
	for _, name := range files {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		destination := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
			return "", err
		}
		if err := os.Rename(filepath.Join(staging, name), destination); err != nil {
			return "", err
		}
	}
	return root, VerifyAssets(ctx, root)
}

func assetDownloadHost(address *url.URL) bool {
	if address.Scheme != "https" || address.User != nil {
		return false
	}
	host := strings.ToLower(address.Hostname())
	for _, domain := range []string{"huggingface.co", "hf.co", "github.com", "githubusercontent.com"} {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}
	return false
}

func downloadAsset(ctx context.Context, client *http.Client, directory string, asset assetDownload, progress func(int)) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.address, nil)
	if err != nil {
		return err
	}
	request.Header.Set("User-Agent", "PicFetch")
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("download %s: %w", asset.name, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: HTTP %d", asset.name, response.StatusCode)
	}
	if response.ContentLength >= 0 && response.ContentLength != asset.size {
		return fmt.Errorf("download %s: unexpected size", asset.name)
	}
	file, err := os.OpenFile(filepath.Join(directory, asset.name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	hash := sha256.New()
	reader := &assetReader{ctx: ctx, source: io.LimitReader(response.Body, asset.size+1), progress: progress}
	n, err := io.Copy(io.MultiWriter(file, hash), reader)
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if n != asset.size || fmt.Sprintf("%x", hash.Sum(nil)) != asset.digest {
		return fmt.Errorf("download %s: checksum or size mismatch", asset.name)
	}
	return ctx.Err()
}

type assetReader struct {
	ctx      context.Context
	source   io.Reader
	progress func(int)
}

func (r *assetReader) Read(buffer []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := r.source.Read(buffer)
	if r.progress != nil && n > 0 {
		r.progress(n)
	}
	return n, err
}

func unpackRuntime(ctx context.Context, directory string) ([]string, error) {
	asset, err := currentRuntime()
	if err != nil {
		return nil, err
	}
	return unpackRuntimeAsset(ctx, directory, asset)
}

func unpackRuntimeAsset(ctx context.Context, directory string, asset runtimeAsset) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if asset.archiveExtension == "zip" {
		return unpackRuntimeZIP(ctx, directory, asset)
	}
	archive, err := os.Open(filepath.Join(directory, "runtime.tgz"))
	if err != nil {
		return nil, err
	}
	defer func() { _ = archive.Close() }()
	compressed, err := gzip.NewReader(archive)
	if err != nil {
		return nil, err
	}
	defer func() { _ = compressed.Close() }()
	reader := tar.NewReader(&assetReader{ctx: ctx, source: io.LimitReader(compressed, 1<<30)})
	expected := asset.files()
	seen := make(map[string]bool, len(expected))
	var files []string
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		// The vendor tarball prefixes its relative names with "./".
		member := strings.TrimPrefix(header.Name, "./")
		name := ""
		for _, trusted := range expected {
			if member == trusted {
				name = trusted
				break
			}
		}
		if name == "" {
			continue
		}
		if seen[name] || header.Typeflag != tar.TypeReg || header.Size < 0 || header.Size > 100<<20 {
			return nil, fmt.Errorf("invalid runtime archive member")
		}
		if err := writeRuntimeFile(ctx, directory, name, reader, header.Size); err != nil {
			return nil, err
		}
		seen[name] = true
		files = append(files, name)
	}
	for _, name := range expected {
		if !seen[name] {
			return nil, fmt.Errorf("runtime archive is missing required files")
		}
	}
	return files, ctx.Err()
}

func unpackRuntimeZIP(ctx context.Context, directory string, asset runtimeAsset) ([]string, error) {
	return unpackRuntimeZIPFile(ctx, directory, asset, filepath.Join(directory, "runtime.zip"))
}

func unpackRuntimeZIPFile(ctx context.Context, directory string, asset runtimeAsset, archivePath string) ([]string, error) {
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = archive.Close() }()
	expected := make(map[string]bool)
	for _, name := range asset.files() {
		expected[name] = true
	}
	seen := make(map[string]bool)
	var files []string
	for _, entry := range archive.File {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		// Skip unlisted entries without decompressing them, including the large
		// PDB. Never normalize a ZIP name into the trusted allowlist.
		if !expected[entry.Name] {
			continue
		}
		if seen[entry.Name] || !entry.Mode().IsRegular() || entry.UncompressedSize64 > 100<<20 {
			return nil, fmt.Errorf("invalid runtime archive member")
		}
		reader, err := entry.Open()
		if err != nil {
			return nil, err
		}
		copyErr := writeRuntimeFile(ctx, directory, entry.Name, reader, int64(entry.UncompressedSize64))
		closeErr := reader.Close()
		if copyErr != nil {
			return nil, copyErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		seen[entry.Name] = true
		files = append(files, entry.Name)
	}
	if len(seen) != len(expected) {
		return nil, fmt.Errorf("runtime archive is missing required files")
	}
	return files, ctx.Err()
}

// name is a fixed allowlisted runtime path, never an arbitrary archive member.
func writeRuntimeFile(ctx context.Context, directory, name string, reader io.Reader, size int64) error {
	target := filepath.Join(directory, name)
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return err
	}
	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	n, copyErr := io.Copy(file, &assetReader{ctx: ctx, source: io.LimitReader(reader, size+1)})
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if n != size {
		return fmt.Errorf("invalid runtime archive member size")
	}
	return ctx.Err()
}
