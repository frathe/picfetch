package fileaccess

import (
	"context"
	"errors"
	"os"

	"fyne.io/fyne/v2"
)

// Stat observes source metadata within its operation-bound native authority.
func Stat(ctx context.Context, uri fyne.URI) (os.FileInfo, error) {
	if uri == nil {
		return nil, errors.New("missing source")
	}
	resolved, release, err := Acquire(ctx, uri)
	if err != nil {
		return nil, err
	}
	defer release()
	return os.Stat(resolved.Path())
}
