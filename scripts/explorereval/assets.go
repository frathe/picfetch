package main

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

//go:embed assets.sha256
var assetChecksums string

const modelRevision = "ba1f3b0843f24bc5417d38e19c37b287d719b2f4"
const runtimeLibrary = "onnxruntime-osx-arm64-1.29.0/lib/libonnxruntime.1.29.0.dylib"

func verifyAssets(ctx context.Context, root string) error {
	for line := range strings.SplitSeq(strings.TrimSpace(assetChecksums), "\n") {
		if err := ctx.Err(); err != nil {
			return err
		}
		fields := strings.Fields(line)
		f, err := os.Open(filepath.Join(root, fields[1]))
		if err != nil {
			return fmt.Errorf("assets: %w; run make explorer-setup", err)
		}
		hash := sha256.New()
		_, err = io.Copy(hash, f)
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
