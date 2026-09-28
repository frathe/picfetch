// Package favstore persists named lists of image files.
package favstore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
)

const fileListName = "file-list.json"

// DefaultDir returns the directory used for favorites in production. When the
// user configuration directory is unavailable, it creates an isolated
// temporary directory rather than placing path metadata directly in the shared
// temporary directory.
func DefaultDir() (string, error) {
	base, err := os.UserConfigDir()
	return defaultDir(base, err, os.TempDir())
}

func defaultDir(base string, configErr error, tempDir string) (string, error) {
	if configErr == nil && base != "" {
		return filepath.Join(base, "picfetch", "favorites"), nil
	}

	privateDir, err := os.MkdirTemp(tempDir, "picfetch-")
	if err != nil {
		return "", fmt.Errorf("create private favorites directory: %w", err)
	}
	return filepath.Join(privateDir, "favorites"), nil
}

// ValidName reports whether name is safe to use as one directory component.
func ValidName(name string) bool {
	return name != "" &&
		name != "." &&
		name != ".." &&
		!strings.ContainsAny(name, `/\:*?"<>|`)
}

// Dir returns the directory holding the favorite named name, or "" when
// name is not a valid favorite name.
func Dir(dir, name string) string {
	if !ValidName(name) {
		return ""
	}
	return filepath.Join(dir, name)
}

// Exists reports whether a favorite with name exists.
func Exists(dir, name string) bool {
	if !ValidName(name) {
		return false
	}
	info, err := os.Stat(filepath.Join(dir, name, fileListName))
	return err == nil && !info.IsDir()
}

// List returns favorite names sorted case-insensitively.
func List(dir string) ([]string, error) {
	return listNames(context.Background(), dir)
}

// Save atomically writes files as the favorite named name.
func Save(dir, name string, files []fyne.URI) error {
	store := &Store{}
	target, err := store.Capture(context.Background(), dir, name)
	if err != nil {
		return err
	}
	_, err = store.Save(context.Background(), target, files)
	return err
}

// readList returns the complete definition through the shared ownership path.
func readList(dir, name string) (Definition, error) {
	if !ValidName(name) {
		return Definition{}, fmt.Errorf("invalid favorite name %q", name)
	}
	return Open(context.Background(), Dir(dir, name))
}

// Load returns the files stored in the favorite named name.
func Load(dir, name string) ([]fyne.URI, error) {
	list, err := readList(dir, name)
	if err != nil {
		return nil, err
	}

	return list.Files(), nil
}

// Count returns how many files the favorite named name stores.
//
// It validates each key exactly as Load does rather than just returning
// len(list): a hand-edited file-list.json with a non-numeric or negative
// key is not a valid stored list, and Count reporting a count for it anyway
// would let a menu label claim a number Load then refuses to open - the two
// must agree on what counts as a stored list, not just on how many keys are
// present. A missing favorite, an unreadable file, and malformed JSON are
// likewise errors here, never 0 - 0 is what a favorite saved with no files
// legitimately reports, and callers need to tell the two apart.
func Count(dir, name string) (int, error) {
	list, err := readList(dir, name)
	if err != nil {
		return 0, err
	}

	return len(list.Paths), nil
}

// Remove moves the favorite named name to the operating system's trash.
func Remove(dir, name string) error {
	store := &Store{}
	target, err := store.Capture(context.Background(), dir, name)
	if err != nil {
		return err
	}
	_, err = store.Remove(context.Background(), target)
	return err
}
