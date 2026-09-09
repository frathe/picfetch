package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/storage/repository"
)

// The command has no Fyne driver, which normally installs the file repository.
// This read-only adapter lets it reuse filescan.Images and imaging.ReadAndProbe
// without opening a desktop window. Failed directory reads must fail the trial
// even though the viewer's best-effort scanner normally continues past them.
type localFiles struct{ scanErrors []error }

var _ repository.ListableRepository = (*localFiles)(nil)

func (r *localFiles) CreateListable(_ fyne.URI) error { return repository.ErrOperationNotSupported }

func (r *localFiles) Exists(u fyne.URI) (bool, error) {
	_, err := os.Stat(u.Path())
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}
func (r *localFiles) CanRead(u fyne.URI) (bool, error) { return r.Exists(u) }
func (r *localFiles) Reader(u fyne.URI) (fyne.URIReadCloser, error) {
	f, err := os.Open(u.Path())
	if err != nil {
		return nil, err
	}
	return localReader{File: f, uri: u}, nil
}
func (r *localFiles) Destroy(_ string) {}
func (r *localFiles) CanList(u fyne.URI) (bool, error) {
	info, err := os.Stat(u.Path())
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}
func (r *localFiles) List(u fyne.URI) ([]fyne.URI, error) {
	entries, err := os.ReadDir(u.Path())
	if err != nil {
		r.scanErrors = append(r.scanErrors, err)
		return nil, err
	}
	result := make([]fyne.URI, 0, len(entries))
	for _, entry := range entries {
		result = append(result, storage.NewFileURI(filepath.Join(u.Path(), entry.Name())))
	}
	return result, nil
}

type localReader struct {
	*os.File
	uri fyne.URI
}

func (r localReader) URI() fyne.URI { return r.uri }

// Round-robin folders/formats after a fixed seeded ordering, so a capped smoke
// corpus cannot silently consist only of the first lexical JPEG directory.
func smokeSample(files []fyne.URI) []fyne.URI {
	buckets := map[string][]fyne.URI{}
	for _, file := range files {
		key := filepath.Dir(file.Path()) + "\x00" + strings.ToLower(file.Extension())
		buckets[key] = append(buckets[key], file)
	}
	keys := make([]string, 0, len(buckets))
	for key, bucket := range buckets {
		keys = append(keys, key)
		sort.Slice(bucket, func(i, j int) bool {
			a := sha256.Sum256([]byte("42:" + bucket[i].String()))
			b := sha256.Sum256([]byte("42:" + bucket[j].String()))
			return fmt.Sprintf("%x", a) < fmt.Sprintf("%x", b)
		})
	}
	sort.Strings(keys)
	var sample []fyne.URI
	for row := 0; len(sample) < min(512, len(files)); row++ {
		for _, key := range keys {
			if row < len(buckets[key]) {
				sample = append(sample, buckets[key][row])
				if len(sample) == 512 {
					return sample
				}
			}
		}
	}
	return sample
}
