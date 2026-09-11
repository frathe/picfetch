package similarity

import (
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/favstore"
)

func TestFavoriteAnalysisFollowsOpenedDirectory(t *testing.T) {
	for _, replacement := range []bool{false, true} {
		name := "moved"
		if replacement {
			name = "replaced"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			original := filepath.Join(dir, "original.jpg")
			other := filepath.Join(dir, "other.jpg")
			if err := favstore.Save(dir, "Trip", []fyne.URI{storage.NewFileURI(original)}); err != nil {
				t.Fatal(err)
			}
			root, err := os.OpenRoot(favstore.Dir(dir, "Trip"))
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = root.Close() }()
			if err := os.Rename(favstore.Dir(dir, "Trip"), filepath.Join(dir, "Moved")); err != nil {
				t.Fatal(err)
			}
			if replacement {
				if err := favstore.Save(dir, "Trip", []fyne.URI{storage.NewFileURI(other)}); err != nil {
					t.Fatal(err)
				}
			}
			favorite, err := loadFavoriteAnalysis(root)
			if err != nil {
				t.Fatalf("moved favorite lost cache admission: %v", err)
			}
			if len(favorite.members) != 1 || !favorite.members[original] || favorite.members[other] || !favorite.current() {
				t.Fatalf("opened directory associated with replacement membership: %v", favorite.members)
			}
		})
	}
}
