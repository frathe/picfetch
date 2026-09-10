package similarity

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const AssetDownloadBytes int64 = 371807752 + 394 + 41578864

// DownloadProgress reports aggregate transfer bytes, without source-image data.
type DownloadProgress struct {
	Received, Total int64
}

type assetDownload struct {
	name, address, digest string
	size                  int64
}

func SupportedPlatform() bool { return runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" }

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

	model := "https://huggingface.co/onnx-community/siglip2-base-patch16-224-ONNX/resolve/" + ModelRevision
	downloads := []assetDownload{
		{"vision_model.onnx", model + "/onnx/vision_model.onnx", "c0573e3f4140c3a7c4e9cc5912bd6b26a033b46a6a8e8af26cbea262b163bcad", 371807752},
		{"preprocessor_config.json", model + "/preprocessor_config.json", "9b36b57ebaf20f09bf4c22100ccc21877ea6bfe5aead0c00c59f8af8ccefacfc", 394},
		{"runtime.tgz", "https://github.com/microsoft/onnxruntime/releases/download/v1.29.0/onnxruntime-osx-arm64-1.29.0.tgz", "d0706fc34f315d8c88639d0a8c81f2e09e815f282cabed3493c06a054352cf92", 41578864},
	}
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
				progress(DownloadProgress{Received: received, Total: AssetDownloadBytes})
			}
		}); err != nil {
			return "", err
		}
	}
	files, err := unpackRuntime(ctx, staging)
	if err != nil {
		return "", err
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
	prefix := "onnxruntime-osx-arm64-1.29.0/"
	expected := map[string]bool{runtimeLibrary: false, prefix + "LICENSE": false, prefix + "ThirdPartyNotices.txt": false}
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
		name := strings.TrimPrefix(header.Name, "./")
		found, needed := expected[name]
		if !needed {
			continue
		}
		if found || header.Typeflag != tar.TypeReg || header.Size < 0 || header.Size > 100<<20 {
			return nil, fmt.Errorf("invalid runtime archive member")
		}
		target := filepath.Join(directory, name)
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return nil, err
		}
		file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return nil, err
		}
		_, copyErr := io.Copy(file, reader)
		closeErr := file.Close()
		if copyErr != nil {
			return nil, copyErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		expected[name] = true
		files = append(files, name)
	}
	for _, found := range expected {
		if !found {
			return nil, fmt.Errorf("runtime archive is missing required files")
		}
	}
	return files, ctx.Err()
}
