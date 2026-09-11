package similarity

import (
	"errors"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/storage/repository"
)

// RegisterLocalFiles installs the read-only repository for a driverless worker.
// The returned check reports directory scan errors the ordinary scanner skips.
func RegisterLocalFiles() func() error {
	local := &localFiles{}
	repository.Register("file", local)
	return func() error { return errors.Join(local.scanErrors...) }
}

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
