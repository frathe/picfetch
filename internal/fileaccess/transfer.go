package fileaccess

import (
	"context"
	"errors"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
)

// Transfer is ephemeral interprocess authority, never a persisted source record.
// Path identifies the captured source; Bookmark has implicit scope for another
// process. Its rights are bounded by the exporting process's existing access.
type Transfer struct {
	Path     string
	Bookmark []byte
}

type sourcesKey struct{}

// WithSources captures an immutable path lookup before work leaves the UI.
// It performs no native I/O and does not start security-scoped access.
func WithSources(ctx context.Context, sources []fyne.URI) context.Context {
	captured := make(map[string]fyne.URI, len(sources))
	for _, uri := range sources {
		if uri != nil {
			captured[uri.Path()] = uri
		}
	}
	return context.WithValue(ctx, sourcesKey{}, captured)
}

// SourceForPath returns captured authority, or a plain URI for an app-owned path.
// A plain URI does not manufacture permission in a sandbox.
func SourceForPath(ctx context.Context, path string) fyne.URI {
	sources, _ := ctx.Value(sourcesKey{}).(map[string]fyne.URI)
	if uri := sources[path]; uri != nil {
		return uri
	}
	return storage.NewFileURI(path)
}

// Export captures a fresh implicit bookmark while the original authority is
// active. The caller retains release until the receiving worker has finished.
func Export(ctx context.Context, uri fyne.URI) (Transfer, func(), error) {
	return exportTransfer(ctx, uri, createTransfer)
}

func exportTransfer(ctx context.Context, uri fyne.URI, create func(string) ([]byte, error)) (Transfer, func(), error) {
	if uri == nil || uri.Scheme() != "file" {
		return Transfer{}, nil, errors.New("worker source must be a local file")
	}
	resolved, release, err := Acquire(ctx, uri)
	if err != nil {
		return Transfer{}, nil, err
	}
	bookmark, err := create(resolved.Path())
	if err == nil {
		err = ctx.Err()
	}
	if err != nil {
		release()
		return Transfer{}, nil, err
	}
	return Transfer{Path: resolved.Path(), Bookmark: bookmark}, release, nil
}

// Import activates only the transferred authority for one bounded worker.
// Native implicit resolution starts access; release balances that acquisition.
func Import(ctx context.Context, transfer Transfer) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(transfer.Bookmark) == 0 || len(transfer.Bookmark) > 1024*1024 {
		return nil, errors.New("invalid worker bookmark size")
	}
	release, err := resolveTransfer(transfer)
	if err != nil {
		return nil, err
	}
	var once sync.Once
	stop := func() { once.Do(release) }
	if err := ctx.Err(); err != nil {
		stop()
		return nil, err
	}
	return stop, nil
}
