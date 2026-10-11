package fileaccess

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"

	"fyne.io/fyne/v2"
)

// Manifest stores each selected scope once and the complete ordered source set.
// Sources without a captured scope use -1. Occurrences are never deduplicated.
type Manifest struct {
	Scopes  []Scope  `json:"scopes,omitempty"`
	Sources []Source `json:"sources"`
}

type Scope struct {
	Bookmark  []byte `json:"bookmark"`
	Directory bool   `json:"directory,omitempty"`
}

type Source struct {
	URI      string `json:"uri"`
	Scope    int    `json:"scope"`
	Relative string `json:"relative,omitempty"`
}

type scopeKey struct {
	digest    [32]byte
	directory bool
}

// Pack captures immutable metadata without resolving scopes or inspecting disk.
func Pack(ctx context.Context, files []fyne.URI) (Manifest, error) {
	result := Manifest{Sources: make([]Source, len(files))}
	indexes := make(map[scopeKey]int)
	for i, file := range files {
		if err := ctx.Err(); err != nil {
			return Manifest{}, err
		}
		if file == nil {
			return Manifest{}, errors.New("source manifest contains nil URI")
		}
		entry := Source{URI: file.String(), Scope: -1}
		if source, ok := file.(*sourceURI); ok {
			record := source.record
			key := scopeKey{digest: sha256.Sum256(record.Bookmark), directory: record.Directory}
			index, found := indexes[key]
			if !found {
				index = len(result.Scopes)
				indexes[key] = index
				result.Scopes = append(result.Scopes, Scope{Bookmark: append([]byte(nil), record.Bookmark...), Directory: record.Directory})
			}
			entry.Scope, entry.Relative = index, record.Relative
		}
		result.Sources[i] = entry
	}
	return result, ctx.Err()
}

// Unpack validates a complete captured set before publishing any of its URIs.
func Unpack(ctx context.Context, manifest Manifest) ([]fyne.URI, error) {
	scopes := make([]Scope, len(manifest.Scopes))
	for i, scope := range manifest.Scopes {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if len(scope.Bookmark) == 0 {
			return nil, errors.New("source manifest has an empty scope")
		}
		scopes[i] = Scope{Bookmark: append([]byte(nil), scope.Bookmark...), Directory: scope.Directory}
	}
	files := make([]fyne.URI, len(manifest.Sources))
	for i, source := range manifest.Sources {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		record := Record{URI: source.URI, Relative: source.Relative}
		if source.Scope < -1 || source.Scope >= len(scopes) {
			return nil, errors.New("source manifest references a missing scope")
		}
		if source.Scope >= 0 {
			scope := scopes[source.Scope]
			record.Bookmark, record.Directory = scope.Bookmark, scope.Directory
		}
		// The private constructor shares only this function's own immutable copy.
		uri, err := fromRecord(record, false)
		if err != nil {
			return nil, err
		}
		files[i] = uri
	}
	return files, ctx.Err()
}

// UnmarshalJSON requires an explicit scope reference. Missing metadata cannot
// silently assign the first captured grant to a different source occurrence.
func (s *Source) UnmarshalJSON(data []byte) error {
	var value struct {
		URI      string `json:"uri"`
		Scope    *int   `json:"scope"`
		Relative string `json:"relative,omitempty"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	if value.Scope == nil {
		return errors.New("source manifest omitted scope reference")
	}
	*s = Source{URI: value.URI, Scope: *value.Scope, Relative: value.Relative}
	return nil
}
