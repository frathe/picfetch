package favstore

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func putDefinition(t *testing.T, data string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	name := "Saved"
	if err := os.Mkdir(Dir(dir, name), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(Dir(dir, name), fileListName), []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir, name
}

func TestFavoriteMembershipLimits(t *testing.T) {
	const limit = 64 * 1024 * 1024
	for _, extra := range []int{0, 1} {
		name := "exact"
		if extra != 0 {
			name = "oversized"
		}
		t.Run(name, func(t *testing.T) {
			data := `{}` + strings.Repeat(" ", limit-2+extra)
			dir, name := putDefinition(t, data)
			_, loadErr := Load(dir, name)
			_, countErr := Count(dir, name)
			if extra == 0 && (loadErr != nil || countErr != nil) {
				t.Fatalf("exact boundary rejected: %v / %v", loadErr, countErr)
			}
			if extra != 0 && (loadErr == nil || countErr == nil) {
				t.Fatalf("oversized definition accepted: %v / %v", loadErr, countErr)
			}
		})
	}
}

func TestFavoriteMembership(t *testing.T) {
	t.Run("paths", func(t *testing.T) {
		base := t.TempDir()
		t.Chdir(base)
		dir, name := putDefinition(t, `{"0":"offline/../Photo.JPG","1":"photo.jpg","2":"Photo.JPG"}`)
		definition, err := Open(context.Background(), Dir(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(definition.Paths, []string{"offline/../Photo.JPG", "photo.jpg", "Photo.JPG"}) {
			t.Fatalf("stored spelling changed: %q", definition.Paths)
		}
		t.Chdir(t.TempDir())
		files := definition.Files()
		if files[0].Path() != filepath.ToSlash(filepath.Join(base, "Photo.JPG")) || files[1].Path() == files[2].Path() {
			t.Fatalf("captured relative paths or case changed: %v", files)
		}
	})
	t.Run("valid", func(t *testing.T) {
		for _, tc := range []struct {
			name, data string
			paths      []string
		}{
			{"empty", `{}`, nil},
			{"sparse_repeated", `{"10":"/a.jpg","2":"/b.jpg","0":"/a.jpg"}`, []string{"/a.jpg", "/b.jpg", "/a.jpg"}},
			{"noncanonical", `{"+3":"/c.jpg","01":"/a.jpg","-0":"/zero.jpg"}`, []string{"/zero.jpg", "/a.jpg", "/c.jpg"}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				dir, name := putDefinition(t, tc.data)
				files, err := Load(dir, name)
				if err != nil {
					t.Fatal(err)
				}
				var paths []string
				for _, file := range files {
					paths = append(paths, file.Path())
				}
				if !slices.Equal(paths, tc.paths) {
					t.Fatalf("ordered occurrences = %q; want %q", paths, tc.paths)
				}
				if count, err := Count(dir, name); err != nil || count != len(tc.paths) {
					t.Fatalf("count = %d, %v", count, err)
				}
			})
		}
	})
	t.Run("invalid", func(t *testing.T) {
		for _, data := range []string{
			`{"0":"/a","0":"/b"}`, `{"0":"/a","\u0030":"/b"}`,
			`{"0":"/a","00":"/b"}`, `{"+1":"/a","1":"/b"}`,
			`{"-1":"/a"}`, `{"1.0":"/a"}`, `{"999999999999999999999999":"/a"}`,
			`{"one":"/a"}`, `{" 1":"/a"}`, `[]`, `null`, `"path"`, `{} {}`,
			`{"0":null}`, `{"0":1}`, `{"0":true}`, `{"0":[]}`, `{"0":{}}`,
			`{"0":""}`, `{"0":"/a\u0000b"}`, `{"0":"/valid","99":""}`,
		} {
			t.Run(data, func(t *testing.T) {
				dir, name := putDefinition(t, data)
				if files, err := Load(dir, name); err == nil || files != nil {
					t.Errorf("invalid definition opened: %v, %v", files, err)
				}
				if _, err := Count(dir, name); err == nil {
					t.Error("invalid definition counted successfully")
				}
			})
		}
	})
}
