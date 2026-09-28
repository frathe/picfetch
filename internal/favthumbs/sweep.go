package favthumbs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/favstore"
)

// Sweep deletes previews under owner that no file in files maps to.
//
// A preview is kept when either its base name matches EntryName(f) for some
// f in files - the current version of a source that could be stat-ed - or
// its base name begins with pathHash(f)+"-" for some f whose EntryName
// reported false. That second case is the offline-volume guard: a source
// that cannot be stat-ed right now has an unknown current version, so every
// preview that could belong to it is retained rather than destroyed on the
// strength of a stat error that may only be temporary.
func Sweep(owner *favstore.Owner, files []fyne.URI) error {
	return sweepContext(context.Background(), owner, files)
}

func sweepContext(ctx context.Context, owner *favstore.Owner, files []fyne.URI) error {
	access, err := acquirePreview(ctx, owner)
	if err != nil {
		return err
	}
	defer func() { _ = access.Close() }()
	dir, err := access.Root.Open(SubDir)
	if err != nil {
		// No thumbs directory means nothing to prune - not an error, since
		// a favorite that has never had a preview written is the common
		// case for a fresh save, not a fault.
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	entries, err := dir.Readdir(-1)
	_ = dir.Close()
	if err != nil {
		return err
	}

	expected := make(map[string]bool, len(files))
	var offlineHashes []string
	for _, f := range files {
		if err := ctx.Err(); err != nil {
			return err
		}
		if name, ok := EntryName(f); ok {
			expected[name] = true
			continue
		}
		// f could not be stat-ed: an offline volume, removed media, or
		// similar transient state. Its current version is unknown, so
		// every preview that could belong to it is retained below rather
		// than treated as stale on the strength of a stat error that may
		// not even be permanent.
		if hash, ok := pathHash(f); ok {
			offlineHashes = append(offlineHashes, hash)
		}
	}

	var firstErr error
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		// Candidates are only regular files: a subdirectory or symlink that
		// happens to carry a .jpg/.png name is not something Write ever
		// produced, so it is left alone rather than risk removing something
		// this package doesn't own.
		if !entry.Mode().IsRegular() {
			continue
		}
		name := entry.Name()
		ext := filepath.Ext(name)
		if ext != ".jpg" && ext != ".png" {
			continue
		}
		base := strings.TrimSuffix(name, ext)
		if expected[base] || retainedByOfflineGuard(base, offlineHashes) {
			continue
		}

		// One file failing to remove (permissions, a concurrent delete)
		// should not stop the rest of the sweep from running; report the
		// first error once the whole directory has been walked.
		if err := removeStalePreview(ctx, access, filepath.Join(SubDir, name), entry); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// A sweep must not unlink a replacement installed after its inventory.
// Publication and cleanup share the same brief critical section.
func removeStalePreview(ctx context.Context, access *favstore.Access, path string, observed os.FileInfo) error {
	previewCommitMu.Lock()
	defer previewCommitMu.Unlock()
	if err := access.Current(ctx); err != nil {
		return err
	}
	current, err := access.Root.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !os.SameFile(observed, current) || observed.Size() != current.Size() || !observed.ModTime().Equal(current.ModTime()) {
		return nil
	}
	return access.Root.Remove(path)
}

// retainedByOfflineGuard reports whether base begins with one of hashes
// followed by a dash. Matching on the hash plus the dash, not a bare
// prefix, keeps one hash from accidentally prefix-matching a different
// hash that happens to extend it.
func retainedByOfflineGuard(base string, hashes []string) bool {
	for _, h := range hashes {
		if strings.HasPrefix(base, h+"-") {
			return true
		}
	}
	return false
}
