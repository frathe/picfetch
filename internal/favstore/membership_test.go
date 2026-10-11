package favstore

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
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
	t.Run("write", func(t *testing.T) {
		for _, tc := range []struct {
			name, path string
			valid      bool
		}{
			{"empty", "", false},
			{"nul", "a\x00b", false},
			{"exact", strings.Repeat("x", limit-9), true},
			{"oversized", strings.Repeat("x", limit-8), false},
			{"escaped_oversized", strings.Repeat("<", limit/6), false},
		} {
			t.Run(tc.name, func(t *testing.T) {
				dir, name := putDefinition(t, `{"0":"/original"}`)
				err := Save(dir, name, []fyne.URI{pathURI{value: tc.path}})
				if (err == nil) != tc.valid {
					t.Fatalf("Save valid=%v: %v", tc.valid, err)
				}
				definition, err := Open(context.Background(), Dir(dir, name))
				if err != nil {
					t.Fatal(err)
				}
				want := "/original"
				if tc.valid {
					want = tc.path
					info, err := os.Stat(filepath.Join(Dir(dir, name), fileListName))
					if err != nil || info.Size() != limit {
						t.Fatalf("exact encoded boundary: %v, %v", info, err)
					}
				}
				if len(definition.Paths) != 1 || definition.Paths[0] != want {
					t.Fatal("write did not preserve the complete expected list")
				}
			})
		}
	})
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
			entries, listErr := (&Store{}).List(context.Background(), dir)
			if listErr != nil || len(entries) != 1 || (entries[0].CountErr != nil) != (extra != 0) {
				t.Fatalf("menu count boundary: %+v, %v", entries, listErr)
			}
			if extra == 0 && (loadErr != nil || countErr != nil) {
				t.Fatalf("exact boundary rejected: %v / %v", loadErr, countErr)
			}
			if extra != 0 && (loadErr == nil || countErr == nil) {
				t.Fatalf("oversized definition accepted: %v / %v", loadErr, countErr)
			}
		})
	}
}

type pathURI struct {
	fyne.URI
	value string
}

func (u pathURI) Path() string { return u.value }

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
				entries, err := (&Store{}).List(context.Background(), dir)
				if err != nil || len(entries) != 1 || entries[0].Count != len(tc.paths) || entries[0].CountErr != nil {
					t.Fatalf("menu count disagrees: %+v, %v", entries, err)
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
			`{"0":"/a","$access":null}`,
			`{"0":"/a","$access":{"sources":[]}}`,
			`{"0":"/a","$access":{"sources":[{"uri":"file:///b","scope":-1}]}}`,
			`{"0":"/a","$access":{"sources":[{"uri":"file:///a","scope":2}]}}`,
			`{"$access":{"sources":[]},"$access":{"sources":[]}}`,
			`{"0":null}`, `{"0":1}`, `{"0":true}`, `{"0":[]}`, `{"0":{}}`,
			`{"0":""}`, `{"0":"/a\u0000b"}`, `{"0":"/valid","99":""}`,
			`{"0":"/valid","1":{"uri":"file:///photo.jpg","directory":true}}`,
			`{"0":{"uri":"file:///photo.jpg","bookmark":"eA==","directory":true,"relative":"../photo.jpg"}}`,
			`{"0":{"uri":"https://example.com/photo.jpg","bookmark":"eA=="}}`,
			`{"0":{"uri":"file:///photo.jpg","bookmark":"not base64"}}`,
		} {
			t.Run(data, func(t *testing.T) {
				dir, name := putDefinition(t, data)
				if files, err := Load(dir, name); err == nil || files != nil {
					t.Errorf("invalid definition opened: %v, %v", files, err)
				}
				if _, err := Count(dir, name); err == nil {
					t.Error("invalid definition counted successfully")
				}
				entries, err := (&Store{}).List(context.Background(), dir)
				if err != nil || len(entries) != 1 || entries[0].Name != name || entries[0].CountErr == nil {
					t.Fatalf("invalid menu count lost fallback: %+v, %v", entries, err)
				}
			})
		}
	})
}
