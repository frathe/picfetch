package similarity

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// StageWindowsRuntime verifies the pinned architecture's release archive and extracts only
// the runtime libraries and upstream notices into a new package directory.
// Packaging calls this explicitly; the Store application never downloads code.
func StageWindowsRuntime(ctx context.Context, arch, archivePath, root string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	asset, ok := platformRuntime("windows", arch)
	if !ok {
		return fmt.Errorf("unsupported Windows runtime architecture %q", arch)
	}
	archive, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer func() { _ = archive.Close() }()
	info, err := archive.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() != asset.size {
		// Windows is a proper name.
		//goland:noinspection GoErrorStringFormat
		return fmt.Errorf("Windows runtime archive has an unexpected size or type")
	}
	staging, err := os.MkdirTemp("", "picfetch-package-runtime-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(staging) }()
	verifiedPath := filepath.Join(staging, "runtime.zip")
	file, err := os.OpenFile(verifiedPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	hash := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(file, hash), &assetReader{ctx: ctx, source: io.LimitReader(archive, asset.size+1)})
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if n != asset.size || fmt.Sprintf("%x", hash.Sum(nil)) != asset.digest {
		// Windows is a proper name.
		//goland:noinspection GoErrorStringFormat
		return fmt.Errorf("Windows runtime archive checksum mismatch")
	}
	if _, err := unpackRuntimeZIPFile(ctx, root, asset, verifiedPath); err != nil {
		return err
	}
	return verifyRuntime(ctx, root, asset)
}
