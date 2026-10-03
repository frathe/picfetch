package fileaccess

import (
	"errors"
	"sync"

	"fyne.io/fyne/v2"
)

type destinationURI struct {
	fyne.URI
	mu        sync.Mutex
	release   func()
	borrowers int
	closed    bool
}

// NewDestination owns a save-panel URL, even before its file exists. The caller
// must ReleaseDestination on every exit, including cancellation and write errors.
func NewDestination(uri fyne.URI, release func()) fyne.URI {
	return &destinationURI{URI: uri, release: release}
}

func (d *destinationURI) acquire() (fyne.URI, func(), error) {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil, nil, errors.New("save destination is closed")
	}
	d.borrowers++
	d.mu.Unlock()
	return d.URI, sync.OnceFunc(func() {
		d.mu.Lock()
		d.borrowers--
		release := d.retireLocked()
		d.mu.Unlock()
		if release != nil {
			release()
		}
	}), nil
}

func (d *destinationURI) retireLocked() func() {
	if !d.closed || d.borrowers != 0 {
		return nil
	}
	release := d.release
	d.release = nil
	return release
}

// ReleaseDestination closes admission without waiting for an active operation.
// The original URL retires only after its last Acquire borrower has returned.
// Plain and persistent source URIs have no destination ownership to discard.
func ReleaseDestination(uri fyne.URI) {
	d, ok := uri.(*destinationURI)
	if !ok {
		return
	}
	d.mu.Lock()
	d.closed = true
	release := d.retireLocked()
	d.mu.Unlock()
	if release != nil {
		release()
	}
}
