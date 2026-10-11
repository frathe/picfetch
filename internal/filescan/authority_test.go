//go:build !darwin || !appleappstore

package filescan

import (
	"context"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/fileaccess"
	"github.com/frathe/picfetch/internal/uitest"
)

// These structural tests use the non-Store bookmark resolver. Actual native
// security-scope acquisition requires the signed sandbox qualification fixture.
func TestScanPreservesDirectoryAuthority(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"first.jpg", "nested/second.jpg"} {
		if err := os.WriteFile(filepath.Join(dir, name), uitest.EncodeJPEG(t, 4, 4, color.White), 0600); err != nil {
			t.Fatal(err)
		}
	}
	root, err := fileaccess.FromRecord(fileaccess.Record{URI: storage.NewFileURI(dir).String(), Bookmark: []byte("folder"), Directory: true})
	if err != nil {
		t.Fatal(err)
	}
	files, truncated := Images(context.Background(), []fyne.URI{root}, 10, nil)
	if len(files) != 2 || truncated {
		t.Fatalf("scan: %v, truncated=%v", files, truncated)
	}
	seen := map[string]bool{}
	for _, file := range files {
		record := fileaccess.Snapshot(file)
		if string(record.Bookmark) != "folder" || !record.Directory {
			t.Fatal("scan discarded selected scope")
		}
		seen[record.Relative] = true
	}
	if !seen["first.jpg"] || !seen["nested/second.jpg"] {
		t.Fatalf("relative identities: %v", seen)
	}
	replayed, _ := ReplayWithAdmission(context.Background(), []fyne.URI{files[0], files[0]}, 10, nil, func(_ fyne.URI) bool { return true })
	if len(replayed) != 2 || replayed[0] != files[0] || replayed[1] != files[0] {
		t.Fatal("replay changed captured occurrences")
	}
}

func TestSiblingScanRequiresDirectoryAuthority(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"first.jpg", "second.jpg"} {
		if err := os.WriteFile(filepath.Join(dir, name), uitest.EncodeJPEG(t, 4, 4, color.White), 0600); err != nil {
			t.Fatal(err)
		}
	}
	selected := storage.NewFileURI(filepath.Join(dir, "first.jpg"))
	file, err := fileaccess.FromRecord(fileaccess.Record{URI: selected.String(), Bookmark: []byte("file")})
	if err != nil {
		t.Fatal(err)
	}
	files, _ := Siblings(context.Background(), file, 10, nil)
	if len(files) != 1 || files[0].String() != selected.String() {
		t.Fatalf("file grant broadened: %v", files)
	}
	root, err := fileaccess.FromRecord(fileaccess.Record{URI: storage.NewFileURI(dir).String(), Bookmark: []byte("folder"), Directory: true})
	if err != nil {
		t.Fatal(err)
	}
	child, err := fileaccess.Child(root, selected)
	if err != nil {
		t.Fatal(err)
	}
	files, _ = Siblings(context.Background(), child, 10, nil)
	if len(files) != 2 {
		t.Fatalf("captured directory did not admit siblings: %v", files)
	}
	for _, source := range files {
		if string(fileaccess.Snapshot(source).Bookmark) != "folder" {
			t.Fatal("sibling lost authority")
		}
	}
}
