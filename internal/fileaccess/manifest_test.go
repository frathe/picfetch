package fileaccess

import (
	"context"
	"encoding/json"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
)

func TestManifestDeduplicatesScopesAndPreservesOccurrences(t *testing.T) {
	root, err := FromRecord(Record{URI: "file:///selected", Bookmark: []byte("scope"), Directory: true})
	if err != nil {
		t.Fatal(err)
	}
	first, err := Child(root, storage.NewFileURI("/selected/first.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := Child(root, storage.NewFileURI("/selected/second.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	files := []fyne.URI{first, second, first, storage.NewFileURI("/plain.jpg")}
	manifest, err := Pack(context.Background(), files)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Scopes) != 1 || len(manifest.Sources) != 4 {
		t.Fatalf("manifest: %+v", manifest)
	}
	restored, err := Unpack(context.Background(), manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifest.Scopes[0].Bookmark[0] = 'X'
	for i, source := range restored {
		if source.String() != files[i].String() {
			t.Fatalf("occurrence %d changed", i)
		}
		if i < 3 && string(Snapshot(source).Bookmark) != "scope" {
			t.Fatal("mutable manifest escaped")
		}
	}
}

func TestManifestRejectsMissingScopeAndTraversal(t *testing.T) {
	for _, manifest := range []Manifest{
		{Scopes: []Scope{{Bookmark: []byte("scope"), Directory: true}}, Sources: []Source{{URI: "file:///selected/a.jpg", Scope: 1}}},
		{Scopes: []Scope{{Bookmark: []byte("scope"), Directory: true}}, Sources: []Source{{URI: "file:///selected/a.jpg", Scope: 0, Relative: "../outside"}}},
		{Scopes: []Scope{{}}, Sources: []Source{{URI: "file:///selected/a.jpg", Scope: 0}}},
		{Sources: []Source{{URI: "file:///selected/a.jpg", Scope: -1, Relative: "nested/a.jpg"}}},
	} {
		if _, err := Unpack(context.Background(), manifest); err == nil {
			t.Fatalf("accepted invalid manifest %+v", manifest)
		}
	}
}

func TestManifestJSONRequiresScopeReference(t *testing.T) {
	var manifest Manifest
	if err := json.Unmarshal([]byte(`{"scopes":[{"bookmark":"c2NvcGU="}],"sources":[{"uri":"file:///selected/a.jpg"}]}`), &manifest); err == nil {
		t.Fatal("missing reference silently selected first scope")
	}
}
