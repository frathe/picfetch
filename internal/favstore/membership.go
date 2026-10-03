package favstore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/fileaccess"
)

// MaxDefinitionBytes bounds the complete encoded Favorite definition.
const MaxDefinitionBytes = 64 * 1024 * 1024

// ErrDefinitionTooLarge rejects a complete encoded definition over the limit.
var ErrDefinitionTooLarge = errors.New("favorite definition exceeds 64 MiB")

func encodeMembership(ctx context.Context, files []fyne.URI) ([]byte, []string, error) {
	paths := make([]string, len(files))
	list := make(map[string]any, len(files))
	rawBytes := 0
	for i, file := range files {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		if file == nil {
			return nil, nil, fmt.Errorf("favorite file %d is nil", i)
		}
		source := file.Path()
		if source == "" || strings.ContainsRune(source, 0) {
			return nil, nil, fmt.Errorf("invalid path at file index %d", i)
		}
		if len(source) > MaxDefinitionBytes-rawBytes {
			return nil, nil, ErrDefinitionTooLarge
		}
		rawBytes += len(source)
		paths[i], list[strconv.Itoa(i)] = source, source
	}
	var scoped bool
	for _, file := range files {
		scoped = scoped || fileaccess.HasScope(file)
	}
	if scoped {
		manifest, err := fileaccess.Pack(ctx, files)
		if err != nil {
			return nil, nil, err
		}
		for _, scope := range manifest.Scopes {
			if len(scope.Bookmark) > MaxDefinitionBytes-rawBytes {
				return nil, nil, ErrDefinitionTooLarge
			}
			rawBytes += len(scope.Bookmark)
		}
		list["$access"] = manifest
	}

	data, err := json.Marshal(list)
	if err != nil {
		return nil, nil, err
	}
	if len(data) >= MaxDefinitionBytes {
		return nil, nil, ErrDefinitionTooLarge
	}
	return append(data, '\n'), paths, ctx.Err()
}

func readDefinition(ctx context.Context, reader io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(contextReader{ctx, reader}, MaxDefinitionBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxDefinitionBytes {
		return nil, ErrDefinitionTooLarge
	}
	return data, ctx.Err()
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p[:min(len(p), 32*1024)])
}

// decodeMembership preserves occurrences while rejecting ambiguous positions.
// Token decoding matters: unmarshalling a map silently accepts duplicate keys.
func decodeMembership(ctx context.Context, data []byte) ([]string, []fyne.URI, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return nil, nil, errors.New("favorite definition must be an object")
	}
	positions := map[int]string{}
	var access *fileaccess.Manifest
	for decoder.More() {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}
		token, err := decoder.Token()
		if err != nil {
			return nil, nil, err
		}
		key, ok := token.(string)
		if !ok {
			return nil, nil, errors.New("invalid favorite index")
		}
		if key == "$access" {
			if access != nil {
				return nil, nil, errors.New("duplicate favorite authority")
			}
			var manifest fileaccess.Manifest
			if err := decoder.Decode(&manifest); err != nil {
				return nil, nil, err
			}
			if manifest.Sources == nil {
				return nil, nil, errors.New("favorite authority has no membership")
			}
			access = &manifest
			continue
		}
		index, err := strconv.Atoi(key)
		if err != nil || index < 0 {
			return nil, nil, fmt.Errorf("invalid file index %q", key)
		}
		if _, exists := positions[index]; exists {
			return nil, nil, fmt.Errorf("duplicate file position %q", key)
		}

		token, err = decoder.Token()
		if err != nil {
			return nil, nil, err
		}
		path, ok := token.(string)
		if !ok || path == "" || strings.ContainsRune(path, 0) {
			return nil, nil, fmt.Errorf("invalid path at file index %q", key)
		}

		positions[index] = path
	}
	if _, err := decoder.Token(); err != nil {
		return nil, nil, err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return nil, nil, errors.New("trailing favorite document")
	}
	indexes := make([]int, 0, len(positions))
	for index := range positions {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	paths := make([]string, len(indexes))
	for i, index := range indexes {
		paths[i] = positions[index]
	}
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	var sources []fyne.URI
	if access != nil {
		if len(access.Sources) != len(paths) {
			return nil, nil, errors.New("favorite authority membership mismatch")
		}
		var err error
		sources, err = fileaccess.Unpack(ctx, *access)
		if err != nil {
			return nil, nil, err
		}
		for i, source := range sources {
			if source.Path() != paths[i] {
				return nil, nil, errors.New("favorite authority path mismatch")
			}
			if !fileaccess.HasScope(source) {
				sources[i] = nil
			}
		}
	}
	return paths, sources, nil
}
