package favorites

import (
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/favstore"
)

func TestFindMoreLikeThisFavoritesCapturesBeforeNaming(t *testing.T) {
	original := storage.NewFileURI(filepath.Join(t.TempDir(), "chosen.jpg"))
	other := storage.NewFileURI(filepath.Join(t.TempDir(), "other.jpg"))
	host := &fakeHost{files: []fyne.URI{other}}
	f := newFeature(t, host)
	files := []fyne.URI{original}
	f.AddFiles(files)
	if f.addPanel == nil {
		t.Fatal("captured-list naming did not open")
	}
	files[0] = other
	host.files = nil
	f.addPanel.entry.SetText("Matches")
	f.addPanel.entry.OnSubmitted("Matches")
	saved, err := favstore.Load(f.dir, "Matches")
	if err != nil || len(saved) != 1 || saved[0].String() != original.String() {
		t.Fatalf("saved captured result %v: %v", saved, err)
	}
}
