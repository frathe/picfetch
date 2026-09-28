package favstore

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Entry is a compact menu observation. A count error preserves the Favorite's
// name and shortcut slot without presenting unknown membership as an empty list.
type Entry struct {
	Name     string
	Count    int
	CountErr error
}

// List reads names and complete validated counts without retaining definitions.
// A failed/cancelled enumeration returns no partial replacement menu.
func (s *Store) List(ctx context.Context, dir string) ([]Entry, error) {
	names, err := listNames(ctx, dir)
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(names))
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		definition, err := s.Open(ctx, Dir(dir, name))
		count := 0
		if err == nil {
			count = len(definition.Paths)
		}
		entries = append(entries, Entry{Name: name, Count: count, CountErr: err})
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func listNames(ctx context.Context, dir string) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	directory, err := os.Open(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = directory.Close() }()
	var names []string
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		entries, readErr := directory.ReadDir(64)
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if !entry.IsDir() || !ValidName(entry.Name()) {
				continue
			}
			info, err := os.Stat(filepath.Join(dir, entry.Name(), fileListName))
			if err == nil && !info.IsDir() {
				names = append(names, entry.Name())
			} else if err != nil && !errors.Is(err, os.ErrNotExist) {
				return nil, err
			}
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return nil, readErr
		}
	}
	sort.Slice(names, func(i, j int) bool {
		left, right := strings.ToLower(names[i]), strings.ToLower(names[j])
		if left == right {
			return names[i] < names[j]
		}
		return left < right
	})
	return names, ctx.Err()
}
