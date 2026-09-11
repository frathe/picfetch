package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/filescan"
	"github.com/frathe/picfetch/internal/similarity"
)

func scanLibrary(ctx context.Context, directory string) ([]fyne.URI, error) {
	info, err := os.Stat(directory)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("library must be a directory")
	}
	scanError := similarity.RegisterLocalFiles()
	files, truncated := filescan.Images(ctx, []fyne.URI{storage.NewFileURI(directory)}, filescan.DefaultMax, nil)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := scanError(); err != nil {
		return nil, fmt.Errorf("incomplete input scan: %w", err)
	}
	if truncated {
		return nil, fmt.Errorf("input scan truncated; refusing an incomplete manifest")
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("library contains no supported images")
	}
	return files, nil
}

// Round-robin folders/formats after a fixed seeded ordering, so a capped smoke
// corpus cannot silently consist only of the first lexical JPEG directory.
func smokeSample(files []fyne.URI) []fyne.URI {
	buckets := map[string][]fyne.URI{}
	for _, file := range files {
		key := filepath.Dir(file.Path()) + "\x00" + strings.ToLower(file.Extension())
		buckets[key] = append(buckets[key], file)
	}
	keys := make([]string, 0, len(buckets))
	for key, bucket := range buckets {
		keys = append(keys, key)
		sort.Slice(bucket, func(i, j int) bool {
			a := sha256.Sum256([]byte("42:" + bucket[i].String()))
			b := sha256.Sum256([]byte("42:" + bucket[j].String()))
			return fmt.Sprintf("%x", a) < fmt.Sprintf("%x", b)
		})
	}
	sort.Strings(keys)
	var sample []fyne.URI
	for row := 0; len(sample) < min(512, len(files)); row++ {
		for _, key := range keys {
			if row < len(buckets[key]) {
				sample = append(sample, buckets[key][row])
				if len(sample) == 512 {
					return sample
				}
			}
		}
	}
	return sample
}
