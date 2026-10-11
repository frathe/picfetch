package session

import (
	"reflect"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/fileaccess"
)

func TestLoadSession_NothingSavedReturnsNil(t *testing.T) {
	app := test.NewApp()

	if got := Load(app); got != nil {
		t.Errorf("Load() = %v, want nil", got)
	}
}

func TestSaveSession_RoundTrip(t *testing.T) {
	app := test.NewApp()

	want := []fyne.URI{
		storage.NewFileURI("/tmp/a.jpg"),
		storage.NewFileURI("/tmp/b.png"),
	}
	Save(app, want)

	got := Load(app)
	if len(got) != len(want) {
		t.Fatalf("Load() returned %d URIs, want %d", len(got), len(want))
	}
	for i, u := range got {
		if u.String() != want[i].String() {
			t.Errorf("Load()[%d] = %q, want %q", i, u.String(), want[i].String())
		}
	}
}

func TestSaveSession_EmptyFilesClearsPreviouslySavedSession(t *testing.T) {
	app := test.NewApp()

	Save(app, []fyne.URI{storage.NewFileURI("/tmp/a.jpg")})
	if Load(app) == nil {
		t.Fatal("expected a saved session before clearing")
	}

	Save(app, nil)

	if got := Load(app); got != nil {
		t.Errorf("Load() after clearing = %v, want nil", got)
	}
	if app.Cache().Exists(cacheKey) {
		t.Error("cache entry should be removed, not left empty")
	}
}

func TestLoadSession_CorruptCacheEntryReturnsNil(t *testing.T) {
	app := test.NewApp()

	w, err := app.Cache().Write(cacheKey)
	if err != nil {
		t.Fatalf("Cache().Write: %v", err)
	}
	if _, err := w.Write([]byte("not valid json")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	if got := Load(app); got != nil {
		t.Errorf("Load() = %v, want nil for corrupt cache entry", got)
	}
}

func TestSaveSession_PreservesAuthorityForEachOccurrence(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	var files []fyne.URI
	for _, bookmark := range []string{"first-folder", "second-folder"} {
		source, err := fileaccess.FromRecord(fileaccess.Record{URI: "file:///photos/a.jpg", Bookmark: []byte(bookmark), Directory: true, Relative: "a.jpg"})
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, source)
	}
	files = append(files, storage.NewFileURI("/photos/plain.jpg"), files[0])
	Save(app, files)
	got := Load(app)
	if len(got) != 4 {
		t.Fatalf("restored %d occurrences, want 4", len(got))
	}
	for i := range files {
		if !reflect.DeepEqual(fileaccess.Snapshot(got[i]), fileaccess.Snapshot(files[i])) {
			t.Fatalf("occurrence %d lost its authority: %+v", i, fileaccess.Snapshot(got[i]))
		}
	}
}

func TestLoadSessionRejectsMisalignedAuthority(t *testing.T) {
	for _, data := range []string{
		`{"files":["file:///photos/a.jpg"],"access":{"sources":[]}}`,
		`{"files":["file:///photos/a.jpg"],"access":{"sources":[{"uri":"file:///photos/b.jpg","scope":-1}]}}`,
		`{"files":["file:///photos/a.jpg"],"access":{"scopes":[{"bookmark":"eA==","directory":true}],"sources":[{"uri":"file:///photos/a.jpg","scope":0,"relative":"../a.jpg"}]}}`,
	} {
		t.Run(data, func(t *testing.T) {
			app := test.NewApp()
			t.Cleanup(app.Quit)
			w, err := app.Cache().Write(cacheKey)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write([]byte(data)); err != nil {
				t.Fatal(err)
			}
			if err := w.Close(); err != nil {
				t.Fatal(err)
			}
			if got := Load(app); got != nil {
				t.Fatalf("restored partial or broadened authority: %v", got)
			}
		})
	}
}

func TestSaveSession_PreservesScopedOccurrences(t *testing.T) {
	app := test.NewApp()
	root, err := fileaccess.FromRecord(fileaccess.Record{URI: "file:///selected", Bookmark: []byte("scope"), Directory: true})
	if err != nil {
		t.Fatal(err)
	}
	source, err := fileaccess.Child(root, storage.NewFileURI("/selected/photo.jpg"))
	if err != nil {
		t.Fatal(err)
	}
	Save(app, []fyne.URI{source, source})
	files := Load(app)
	if len(files) != 2 {
		t.Fatalf("occurrences = %d", len(files))
	}
	for _, uri := range files {
		record := fileaccess.Snapshot(uri)
		if uri.String() != source.String() || string(record.Bookmark) != "scope" || record.Relative != "photo.jpg" {
			t.Fatalf("lost saved authority: %+v", record)
		}
	}
}
