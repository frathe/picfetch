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
	"strings"
)

//go:embed assets.sha256
var assetChecksums string

const ModelRevision = "ba1f3b0843f24bc5417d38e19c37b287d719b2f4"

// Each runtime is pinned independently; model files are shared across platforms.
type runtimeAsset struct {
	directory, library, digest string
	size                       int64
}

func platformRuntime(goos, goarch string) (runtimeAsset, bool) {
	switch {
	case goos == "darwin" && goarch == "arm64":
		return runtimeAsset{"onnxruntime-osx-arm64-1.29.0", "lib/libonnxruntime.1.29.0.dylib", "d0706fc34f315d8c88639d0a8c81f2e09e815f282cabed3493c06a054352cf92", 41578864}, true
	case goos == "linux" && goarch == "amd64":
		return runtimeAsset{"onnxruntime-linux-x64-1.29.0", "lib/libonnxruntime.so.1.29.0", "c3fddc4f139a045b0c4902c57410f0694f1c2fdf9b6939fbe38b1aeae7cd14ba", 11082880}, true
	default:
		return runtimeAsset{}, false
	}
}

func SupportedPlatform() bool {
	_, ok := platformRuntime(runtime.GOOS, runtime.GOARCH)
	return ok
}

func currentRuntime() (runtimeAsset, error) {
	asset, ok := platformRuntime(runtime.GOOS, runtime.GOARCH)
	if !ok {
		return runtimeAsset{}, fmt.Errorf("local similarity requires Apple Silicon macOS or Linux amd64")
	}
	return asset, nil
}

// AssetDownloadBytes is the full transfer size for the current platform.
func AssetDownloadBytes() int64 {
	asset, _ := platformRuntime(runtime.GOOS, runtime.GOARCH)
	return 371807752 + 394 + asset.size
}

func VerifyAssets(ctx context.Context, root string) error {
	asset, platformErr := currentRuntime()
	library := ""
	if platformErr == nil {
		library = asset.directory + "/" + asset.library
	}
	for line := range strings.SplitSeq(strings.TrimSpace(assetChecksums), "\n") {
		if err := ctx.Err(); err != nil {
			return err
		}
		fields := strings.Fields(line)
		if strings.HasPrefix(fields[1], "onnxruntime-") && fields[1] != library {
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
	if err := ctx.Err(); err != nil {
		return err
	}
	return platformErr
}
