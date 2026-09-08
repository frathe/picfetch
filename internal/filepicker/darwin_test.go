//go:build darwin

package filepicker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDarwinPathTransport_RoundTripsNativeURLPaths(t *testing.T) {
	dir := t.TempDir()
	// NSURL returns the filesystem spelling, including decomposed accents.
	// Use that native spelling to isolate transport from Unicode normalization.
	var paths []string
	for _, name := range []string{"line\nnext.jpg", "tail\r", "tail\n", " spaces ", "cafe\u0301 東京 😀.png"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("temporary fixture"), 0o600); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	out, err := darwinPathTransport(paths)
	selected, err := decodePickedPaths(out, err)
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != len(paths) {
		t.Fatalf("selected %d paths, want %d", len(selected), len(paths))
	}
	for i, uri := range selected {
		if uri.Path() != paths[i] {
			t.Errorf("native path %d = %q, want %q", i, uri.Path(), paths[i])
		}
		if _, err := os.Stat(uri.Path()); err != nil {
			t.Errorf("native result no longer names its temporary file: %v", err)
		}
	}
}
