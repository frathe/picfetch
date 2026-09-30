//go:build !(darwin && appleappstore)

package fileaccess

import (
	"context"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2/storage"
)

func resolveBookmark(_ context.Context, record Record) (resolution, func(), error) {
	uri, err := storage.ParseURI(record.URI)
	if err != nil {
		return resolution{}, nil, err
	}
	root := uri.Path()
	if record.Relative != "" {
		for range strings.Split(record.Relative, "/") {
			root = filepath.Dir(root)
		}
	}
	return resolution{path: root}, func() {}, nil
}
