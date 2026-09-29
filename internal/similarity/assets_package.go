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
	return stageRuntime(ctx, asset, archivePath, root)
}

// StageMacRuntime verifies the architecture-pinned upstream archive and stages
// its native library and notices before signing. Signing changes Mach-O bytes;
// the signed bundle needs code-signature validation instead of this checksum.
func StageMacRuntime(ctx context.Context, arch, archivePath, root string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	asset, ok := platformRuntime("darwin", arch)
	if !ok {
		return fmt.Errorf("unsupported macOS runtime architecture %q", arch)
	}
	// Both pinned macOS releases provide this notice. Keep it with the native
	// payload even where ordinary asset installation does not require it.
	asset.privacyNotice = true
	return stageRuntime(ctx, asset, archivePath, root)
}

func stageRuntime(ctx context.Context, asset runtimeAsset, archivePath, root string) error {
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
		return fmt.Errorf("runtime archive has an unexpected size or type")
	}
	staging, err := os.MkdirTemp("", "picfetch-package-runtime-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(staging) }()
	verifiedPath := filepath.Join(staging, "runtime."+asset.archiveExtension)
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
		return fmt.Errorf("runtime archive checksum mismatch")
	}
	if _, err := unpackRuntimeAsset(ctx, staging, asset); err != nil {
		return err
	}
	if err := verifyRuntime(ctx, staging, asset); err != nil {
		return err
	}
	for _, name := range asset.files() {
		if err := stageRuntimeFile(ctx, staging, root, name); err != nil {
			return err
		}
	}
	return ctx.Err()
}

func stageRuntimeFile(ctx context.Context, staging, root, name string) error {
	file, err := os.Open(filepath.Join(staging, name))
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	return writeRuntimeFile(ctx, root, name, file, info.Size())
}
