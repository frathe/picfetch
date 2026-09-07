// Package favthumbs persists grid previews for a favorite's files under
// that favorite's own directory.
package favthumbs

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2"
)

// SubDir is the directory name, under a favorite, that holds previews.
const SubDir = "thumbs"

// Dir returns the preview directory for the favorite rooted at favDir.
func Dir(favDir string) string {
	return filepath.Join(favDir, SubDir)
}

// pathHash returns the stable, stat-independent half of src's entry name:
// the first 8 bytes of SHA-256 over its path, hex-encoded. Unlike the
// mtime/size half that EntryName adds on top, this half never depends on
// src actually existing right now, which is what lets Sweep recognize which
// previews belong to a source it currently cannot stat.
func pathHash(src fyne.URI) (string, bool) {
	if src == nil {
		return "", false
	}

	sum := sha256.Sum256([]byte(src.Path()))
	return hex.EncodeToString(sum[:8]), true
}

// EntryName returns the preview file's base name (no extension) for the
// source src: "<pathhash>-<mtime seconds>.<nanoseconds>-<size>". It reports
// false when src cannot be stat-ed, telling the caller to leave any existing
// preview for src alone rather than treat it as stale.
func EntryName(src fyne.URI) (string, bool) {
	hash, ok := pathHash(src)
	if !ok {
		return "", false
	}

	info, err := os.Stat(src.Path())
	if err != nil {
		return "", false
	}

	return entryName(hash, info.ModTime(), info.Size()), true
}

func entryName(hash string, modified time.Time, size int64) string {
	return fmt.Sprintf("%s-%d.%09d-%d", hash, modified.Unix(), modified.Nanosecond(), size)
}
