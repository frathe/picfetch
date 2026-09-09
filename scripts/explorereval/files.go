package main

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
)

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
