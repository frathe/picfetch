package uitest

import (
	"io"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage/repository"
)

// ReaderURI preserves u's file identity and metadata while serving reads from
// the instance-owned factory. The immutable adapter is registered during init,
// before any workers run: Fyne's repository registry is not synchronized.
func ReaderURI(u fyne.URI, open func() (io.ReadCloser, error)) fyne.URI {
	return &storageURI{URI: u, open: open}
}

// DirectoryURI preserves a directory's path while serving listings from an
// instance-owned callback, so tests can control an in-flight directory read.
func DirectoryURI(u fyne.URI, list func() ([]fyne.URI, error)) fyne.URI {
	return &storageURI{URI: u, list: list}
}

type storageURI struct {
	fyne.URI
	open func() (io.ReadCloser, error)
	list func() ([]fyne.URI, error)
}

func (_ *storageURI) Scheme() string { return "picfetch-test-storage" }
func (u *storageURI) String() string {
	return u.Scheme() + strings.TrimPrefix(u.URI.String(), u.URI.Scheme())
}

type storageRepository struct{}

func init() { repository.Register("picfetch-test-storage", storageRepository{}) }

func (storageRepository) Exists(_ fyne.URI) (bool, error) { return true, nil }
func (storageRepository) Destroy(_ string)                {}
func (storageRepository) CanRead(u fyne.URI) (bool, error) {
	return u.(*storageURI).open != nil, nil
}
func (storageRepository) Reader(u fyne.URI) (fyne.URIReadCloser, error) {
	open := u.(*storageURI).open
	if open == nil {
		return nil, repository.ErrOperationNotSupported
	}
	r, err := open()
	if err != nil {
		return nil, err
	}
	return uriReader{ReadCloser: r, u: u}, nil
}

func (storageRepository) CanList(u fyne.URI) (bool, error) {
	return u.(*storageURI).list != nil, nil
}
func (storageRepository) List(u fyne.URI) ([]fyne.URI, error) {
	list := u.(*storageURI).list
	if list == nil {
		return nil, repository.ErrOperationNotSupported
	}
	return list()
}
func (storageRepository) CreateListable(_ fyne.URI) error {
	return repository.ErrOperationNotSupported
}

type uriReader struct {
	io.ReadCloser
	u fyne.URI
}

func (r uriReader) URI() fyne.URI { return r.u }

// ReadCloser supplies controlled read/close effects without replacing a global
// storage function. Tests own synchronization and wait for completion before
// inspecting state captured by these callbacks.
type ReadCloser struct {
	ReadFunc  func([]byte) (int, error)
	CloseFunc func() error
}

func (r ReadCloser) Read(p []byte) (int, error) { return r.ReadFunc(p) }
func (r ReadCloser) Close() error               { return r.CloseFunc() }
