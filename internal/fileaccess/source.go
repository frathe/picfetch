// Package fileaccess carries immutable filesystem authority with source URIs.
// Native scopes belong to bounded operations, rather than to viewer lifetime.
package fileaccess

import (
	"context"
	"errors"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
)

// Record persists one occurrence's URI and its selected scope. Relative names
// refer only to children of a selected directory, never to its parent.
// Directory describes the selected scope, including when URI names its child.
type Record struct {
	URI       string `json:"uri"`
	Bookmark  []byte `json:"bookmark,omitempty"`
	Directory bool   `json:"directory,omitempty"`
	Relative  string `json:"relative,omitempty"`
}

type resolution struct {
	path     string
	bookmark []byte
}

type sourceURI struct {
	fyne.URI
	record  Record
	resolve func(context.Context, Record) (resolution, func(), error)
}

// FromRecord restores metadata without starting native access or inspecting disk.
func FromRecord(record Record) (fyne.URI, error) {
	return fromRecord(record, true)
}

func fromRecord(record Record, copyBookmark bool) (fyne.URI, error) {
	uri, err := storage.ParseURI(record.URI)
	if err != nil {
		return nil, err
	}
	if len(record.Bookmark) == 0 {
		if record.Relative != "" || record.Directory {
			return nil, errors.New("source scope missing bookmark")
		}
		return uri, nil
	}
	if uri.Scheme() != "file" || !(path.IsAbs(uri.Path()) || filepath.IsAbs(uri.Path())) || strings.ContainsRune(uri.Path(), 0) {
		return nil, errors.New("bookmarked source requires an absolute file URI")
	}
	if record.Relative != "" && (!record.Directory || !validRelative(record.Relative)) {
		return nil, errors.New("source escapes its selected scope")
	}
	record.URI = uri.String()
	if copyBookmark {
		record.Bookmark = append([]byte(nil), record.Bookmark...)
	}
	return &sourceURI{URI: uri, record: record, resolve: resolveBookmark}, nil
}

func validRelative(name string) bool {
	return name != "." && !path.IsAbs(name) && !filepath.IsAbs(name) && path.Clean(name) == name &&
		name != ".." && !strings.HasPrefix(name, "../") && !strings.ContainsRune(name, 0)
}

// HasScope reports whether a URI carries captured authority, without native I/O.
func HasScope(uri fyne.URI) bool {
	_, ok := uri.(*sourceURI)
	return ok
}

// Snapshot returns independent persistent metadata; it acquires no native scope.
func Snapshot(uri fyne.URI) Record {
	if source, ok := uri.(*sourceURI); ok {
		record := source.record
		record.Bookmark = append([]byte(nil), record.Bookmark...)
		return record
	}
	if uri == nil {
		return Record{}
	}
	return Record{URI: uri.String()}
}

// Child propagates only captured directory authority to a discovered child.
func Child(parent, child fyne.URI) (fyne.URI, error) {
	source, ok := parent.(*sourceURI)
	if !ok {
		return child, nil
	}
	if !source.record.Directory || child == nil || child.Scheme() != "file" {
		return nil, errors.New("source does not grant directory access")
	}
	relative, err := filepath.Rel(parent.Path(), child.Path())
	relative = filepath.ToSlash(relative)
	if err != nil || !validRelative(relative) {
		return nil, errors.New("child escapes selected directory")
	}
	record := source.record
	record.URI = child.String()
	record.Relative = path.Join(record.Relative, relative)
	return &sourceURI{URI: child, record: record, resolve: source.resolve}, nil
}

// Acquire resolves one operation's authority. Call release after all I/O and
// native completion using the returned URI; release is safe to call repeatedly.
func Acquire(ctx context.Context, uri fyne.URI) (fyne.URI, func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if destination, ok := uri.(*destinationURI); ok {
		return destination.acquire()
	}
	source, ok := uri.(*sourceURI)
	if !ok {
		return uri, func() {}, nil
	}
	root, stop, err := source.resolve(ctx, Snapshot(source))
	if err != nil {
		return nil, nil, err
	}
	var once sync.Once
	release := func() { once.Do(stop) }
	if err := ctx.Err(); err != nil {
		release()
		return nil, nil, err
	}
	resolved := root.path
	if source.record.Relative != "" {
		resolved = filepath.Join(root.path, filepath.FromSlash(source.record.Relative))
	}
	resolvedURI := storage.NewFileURI(resolved)
	record := source.record
	record.URI = resolvedURI.String()
	if len(root.bookmark) != 0 {
		record.Bookmark = append([]byte(nil), root.bookmark...)
	}
	return &sourceURI{URI: resolvedURI, record: record, resolve: source.resolve}, release, nil
}

// Parent retains only already selected directory authority. A selected file or
// a selected directory's own parent requires another explicit user grant.
func Parent(uri fyne.URI) (fyne.URI, error) {
	source, ok := uri.(*sourceURI)
	if !ok {
		return storage.Parent(uri)
	}
	if !source.record.Directory || source.record.Relative == "" {
		return nil, errors.New("parent is outside selected scope")
	}
	parent, err := storage.Parent(uri)
	if err != nil {
		return nil, err
	}
	record := source.record
	record.URI = parent.String()
	record.Relative = path.Dir(record.Relative)
	if record.Relative == "." {
		record.Relative = ""
	}
	return &sourceURI{URI: parent, record: record, resolve: source.resolve}, nil
}
