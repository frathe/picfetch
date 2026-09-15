package fileidentity_test

import (
	"testing"

	"github.com/frathe/picfetch/internal/fileidentity"
)

func TestIndexOccurrences(t *testing.T) {
	paths := []string{"/a", "/b", "/a", "", "/a"}
	reads := 0
	index := fileidentity.NewIndex(len(paths), func(i int) string {
		reads++
		return paths[i]
	})
	for ordinal, position := range []int{0, 2, 4} {
		identity, ok := index.Capture("/a", position)
		want := fileidentity.Occurrence{Path: "/a", Ordinal: ordinal}
		if !ok || identity != want || index.Resolve(identity) != position {
			t.Fatalf("occurrence at %d: got %+v/%v, want %+v", position, identity, ok, want)
		}
	}
	for _, identity := range []fileidentity.Occurrence{{}, {Path: "/missing"}, {Path: "/a", Ordinal: -1}, {Path: "/a", Ordinal: 3}} {
		if got := index.Resolve(identity); got != -1 {
			t.Fatalf("absent occurrence %+v resolved to %d", identity, got)
		}
	}
	for _, position := range []int{-1, 1, 3, 5} {
		if identity, ok := index.Capture("/a", position); ok {
			t.Fatalf("unrelated position %d captured as %+v", position, identity)
		}
	}
	if _, ok := index.Capture("", 3); ok {
		t.Fatal("empty source acquired an occurrence")
	}
	if reads != len(paths) {
		t.Fatalf("immutable lookups reread the collection: %d reads", reads)
	}
}

func TestIndexSnapshotAndReconciliation(t *testing.T) {
	paths := []string{"/a", "/b", "/a", "/c"}
	before := fileidentity.NewIndex(len(paths), func(i int) string { return paths[i] })
	bookmark, ok := before.Capture("/a", 2)
	if !ok {
		t.Fatal("second occurrence was not captured")
	}
	paths[0], paths[1] = paths[1], paths[0]
	after := fileidentity.NewIndex(len(paths), func(i int) string { return paths[i] })
	if before.Resolve(fileidentity.Occurrence{Path: "/a"}) != 0 || after.Resolve(fileidentity.Occurrence{Path: "/a"}) != 1 || after.Resolve(bookmark) != 2 {
		t.Fatal("reorder mutated the captured index or lost an occurrence")
	}
	paths = []string{"/b", "/a", "/c"}
	remaining := fileidentity.NewIndex(len(paths), func(i int) string { return paths[i] })
	if remaining.Resolve(bookmark) != -1 || remaining.Resolve(fileidentity.Occurrence{Path: bookmark.Path}) != 1 {
		t.Fatal("missing occurrence silently fell back to a different occurrence")
	}
	if before.Resolve(bookmark) != 2 || (fileidentity.Index{}).Resolve(bookmark) != -1 {
		t.Fatal("captured and empty indexes lost their independent state")
	}
}
