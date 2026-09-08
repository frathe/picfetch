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
	return &readerURI{URI: u, open: open}
}

type readerURI struct {
	fyne.URI
	open func() (io.ReadCloser, error)
}

func (_ *readerURI) Scheme() string { return "picfetch-test-reader" }
func (u *readerURI) String() string {
	return u.Scheme() + strings.TrimPrefix(u.URI.String(), u.URI.Scheme())
}

type readerRepository struct{}

func init() { repository.Register("picfetch-test-reader", readerRepository{}) }

func (readerRepository) Exists(_ fyne.URI) (bool, error)  { return true, nil }
func (readerRepository) CanRead(_ fyne.URI) (bool, error) { return true, nil }
func (readerRepository) Destroy(_ string)                 {}
func (readerRepository) Reader(u fyne.URI) (fyne.URIReadCloser, error) {
	r, err := u.(*readerURI).open()
	if err != nil {
		return nil, err
	}
	return uriReader{ReadCloser: r, u: u}, nil
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
