package fileaccess

import (
	"context"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
)

// Reader retains source authority until the returned reader closes. Failed
// opens release immediately, and concurrent or repeated Close calls release once.
func Reader(ctx context.Context, source fyne.URI) (fyne.URIReadCloser, error) {
	uri, release, err := Acquire(ctx, source)
	if err != nil {
		return nil, err
	}
	reader, err := storage.Reader(uri)
	if err != nil {
		release()
		return nil, err
	}
	return &scopedReader{URIReadCloser: reader, release: release}, nil
}

type scopedReader struct {
	fyne.URIReadCloser
	release func()
	once    sync.Once
	err     error
}

func (r *scopedReader) Close() error {
	r.once.Do(func() { r.err = r.URIReadCloser.Close(); r.release() })
	return r.err
}
