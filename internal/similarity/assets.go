package similarity

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/frathe/picfetch/internal/distribution"
)

//go:embed assets.sha256
var assetChecksums string

const ModelRevision = "ba1f3b0843f24bc5417d38e19c37b287d719b2f4"

// Each runtime is pinned independently; model files are shared across platforms.
type runtimeAsset struct {
	directory, library, digest string
	version                    string
	size                       int64
	archiveExtension           string
	supportLibraries           []string
	privacyNotice              bool
}

func platformRuntime(goos, goarch string) (runtimeAsset, bool) {
	switch {
	case goos == "darwin" && goarch == "amd64":
		return runtimeAsset{directory: "onnxruntime-osx-x86_64-1.23.2", library: "lib/libonnxruntime.1.23.2.dylib", version: "1.23.2", digest: "d10359e16347b57d9959f7e80a225a5b4a66ed7d7e007274a15cae86836485a6", size: 11676322, archiveExtension: "tgz", privacyNotice: true}, true
	case goos == "darwin" && goarch == "arm64":
		return runtimeAsset{directory: "onnxruntime-osx-arm64-1.29.0", library: "lib/libonnxruntime.1.29.0.dylib", digest: "d0706fc34f315d8c88639d0a8c81f2e09e815f282cabed3493c06a054352cf92", size: 41578864, archiveExtension: "tgz"}, true
	case goos == "linux" && goarch == "amd64":
		return runtimeAsset{directory: "onnxruntime-linux-x64-1.29.0", library: "lib/libonnxruntime.so.1.29.0", digest: "c3fddc4f139a045b0c4902c57410f0694f1c2fdf9b6939fbe38b1aeae7cd14ba", size: 11082880, archiveExtension: "tgz"}, true
	case goos == "linux" && goarch == "arm64":
		return runtimeAsset{directory: "onnxruntime-linux-aarch64-1.29.0", library: "lib/libonnxruntime.so.1.29.0", digest: "e1799098ebc054b370f6176a450f158720f297818c613e5dc99b92e2ec82346f", size: 10027600, archiveExtension: "tgz"}, true
	case goos == "windows" && goarch == "amd64":
		return runtimeAsset{directory: "onnxruntime-win-x64-1.29.0", library: "lib/onnxruntime.dll", digest: "c9b4b7086b529ad814f428c1bad028e20a25d7dc0699836775faace4ab5b78b2", size: 79645520, archiveExtension: "zip", supportLibraries: []string{"lib/onnxruntime_providers_shared.dll"}}, true
	case goos == "windows" && goarch == "arm64":
		return runtimeAsset{directory: "onnxruntime-win-arm64-1.29.0", library: "lib/onnxruntime.dll", digest: "a094a49c3ced0f9fca554647cc7566ae99d93a63a8ce6bf47975561c2de7608e", size: 81679033, archiveExtension: "zip", supportLibraries: []string{"lib/onnxruntime_providers_shared.dll"}}, true
	default:
		return runtimeAsset{}, false
	}
}

func SupportedPlatform() bool {
	return AssetPlatformSupported()
}

// AssetPlatformSupported reports whether pinned downloads exist for this host.
// It does not imply that local analysis is available.
func AssetPlatformSupported() bool {
	_, ok := platformRuntime(runtime.GOOS, runtime.GOARCH)
	return ok
}

func currentRuntime() (runtimeAsset, error) {
	asset, ok := platformRuntime(runtime.GOOS, runtime.GOARCH)
	if !ok {
		return runtimeAsset{}, fmt.Errorf("local similarity assets require macOS/Linux/Windows amd64/arm64")
	}
	return asset, nil
}

func (a runtimeAsset) files() []string {
	names := append([]string{a.library, "LICENSE", "ThirdPartyNotices.txt"}, a.supportLibraries...)
	if a.archiveExtension == "zip" || a.privacyNotice {
		names = append(names, "Privacy.md")
	}
	for i := range names {
		names[i] = a.directory + "/" + names[i]
	}
	return names
}

// AssetDownloadBytes is the full transfer size for the current platform.
func AssetDownloadBytes() int64 {
	asset, _ := platformRuntime(runtime.GOOS, runtime.GOARCH)
	var total int64
	for _, download := range assetDownloads(asset, distribution.StoreManaged) {
		total += download.size
	}
	return total
}

func VerifyAssets(ctx context.Context, root string) error {
	asset, err := currentRuntime()
	if err != nil {
		return err
	}
	nativeRoot, err := runtimeDirectory(root)
	if err != nil {
		return err
	}
	return verifyAssetDirectories(ctx, root, nativeRoot, asset)
}

// Store native code is installed beside the executable. Model/cache overrides
// must never select another runtime or cause it to be copied into the cache.
func runtimeDirectory(modelRoot string) (string, error) {
	if !distribution.StoreManaged {
		return modelRoot, nil
	}
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(executable), nil
}

func verifyAssetDirectories(ctx context.Context, modelRoot, nativeRoot string, asset runtimeAsset) error {
	if err := verifyAssetFiles(ctx, modelRoot, "vision_model.onnx", "preprocessor_config.json"); err != nil {
		return err
	}
	return verifyRuntime(ctx, nativeRoot, asset)
}

func verifyRuntime(ctx context.Context, root string, asset runtimeAsset) error {
	names := []string{asset.directory + "/" + asset.library}
	for _, name := range asset.supportLibraries {
		names = append(names, asset.directory+"/"+name)
	}
	return verifyAssetFiles(ctx, root, names...)
}

func verifyAssetFiles(ctx context.Context, root string, names ...string) error {
	for line := range strings.SplitSeq(strings.TrimSpace(assetChecksums), "\n") {
		if err := ctx.Err(); err != nil {
			return err
		}
		fields := strings.Fields(line)
		if !slices.Contains(names, fields[1]) {
			continue
		}
		f, err := os.Open(filepath.Join(root, fields[1]))
		if err != nil {
			return fmt.Errorf("assets: %w; run make explorer-setup", err)
		}
		hash := sha256.New()
		_, err = io.Copy(hash, &assetReader{ctx: ctx, source: f})
		closeErr := f.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
		if fmt.Sprintf("%x", hash.Sum(nil)) != fields[0] {
			return fmt.Errorf("assets: checksum mismatch for %s", fields[1])
		}
	}
	return ctx.Err()
}
