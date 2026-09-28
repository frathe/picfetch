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
)

// MaxDefinitionBytes bounds the complete encoded Favorite definition.
const MaxDefinitionBytes = 64 * 1024 * 1024

// ErrDefinitionTooLarge rejects a complete encoded definition over the limit.
var ErrDefinitionTooLarge = errors.New("favorite definition exceeds 64 MiB")

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
func decodeMembership(ctx context.Context, data []byte) ([]string, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return nil, errors.New("favorite definition must be an object")
	}
	positions := map[int]string{}
	for decoder.More() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		token, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		key, ok := token.(string)
		if !ok {
			return nil, errors.New("invalid favorite index")
		}
		index, err := strconv.Atoi(key)
		if err != nil || index < 0 {
			return nil, fmt.Errorf("invalid file index %q", key)
		}
		if _, exists := positions[index]; exists {
			return nil, fmt.Errorf("duplicate file position %q", key)
		}
		token, err = decoder.Token()
		if err != nil {
			return nil, err
		}
		path, ok := token.(string)
		if !ok || path == "" || strings.ContainsRune(path, 0) {
			return nil, fmt.Errorf("invalid path at file index %q", key)
		}
		positions[index] = path
	}
	if _, err := decoder.Token(); err != nil {
		return nil, err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("trailing favorite document")
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
		return nil, err
	}
	return paths, nil
}
