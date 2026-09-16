package client

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

const stagedExecutable = "picfetch-heic-worker.exe"

// stageHelper runs while the caller holds the private cache's publication lock.
// Preparation operates on owned copies only, before their directory is published.
func stageHelper(ctx context.Context, source string, digest [32]byte, root string, prepare func(string) error) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	directory := filepath.Join(root, runtime.GOARCH+"-"+hex.EncodeToString(digest[:]))
	target := filepath.Join(directory, stagedExecutable)
	if info, err := os.Lstat(directory); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return "", ErrUnavailable
		}
		if err = verifyStaged(ctx, target, digest); err == nil {
			if err = prepare(target); err != nil {
				return "", err
			}
			return target, ctx.Err()
		}
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		// A Windows lease prevents removing any copy used by another app instance.
		// Remove only the known file and empty directory; never recurse into a cache entry.
		if err = removeStaged(directory); err != nil {
			return "", err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	temporary, err := os.MkdirTemp(root, ".stage-")
	if err != nil {
		return "", err
	}
	defer func() { _ = removeStaged(temporary) }()
	copied := filepath.Join(temporary, stagedExecutable)
	if err = copyStaged(ctx, source, copied); err != nil {
		return "", err
	}
	if err = verifyStaged(ctx, copied, digest); err != nil {
		return "", err
	}
	if err = prepare(copied); err != nil {
		return "", err
	}
	if err = ctx.Err(); err != nil {
		return "", err
	}
	if err = os.Rename(temporary, directory); err != nil {
		return "", err
	}
	return target, nil
}

func removeStaged(directory string) error {
	if err := os.Remove(filepath.Join(directory, stagedExecutable)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Remove(directory)
}

func copyStaged(ctx context.Context, source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { _ = input.Close() }()
	info, err := input.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 64*1024*1024 {
		return ErrUnavailable
	}
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0700)
	if err != nil {
		return err
	}
	defer func() { _ = output.Close() }()
	n, err := io.Copy(output, io.LimitReader(stagingReader{ctx, input}, 64*1024*1024+1))
	if err != nil {
		return err
	}
	if n != info.Size() {
		return fmt.Errorf("%w: packaged helper size changed", ErrUnavailable)
	}
	if err = output.Sync(); err != nil {
		return err
	}
	return output.Close()
}

func verifyStaged(ctx context.Context, path string, digest [32]byte) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 64*1024*1024 {
		return ErrUnavailable
	}
	input, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = input.Close() }()
	hash := sha256.New()
	n, err := io.Copy(hash, io.LimitReader(stagingReader{ctx, input}, 64*1024*1024+1))
	if err != nil {
		return err
	}
	if n != info.Size() || string(hash.Sum(nil)) != string(digest[:]) {
		return fmt.Errorf("%w: staged helper identity", ErrUnavailable)
	}
	return ctx.Err()
}

type stagingReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r stagingReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
