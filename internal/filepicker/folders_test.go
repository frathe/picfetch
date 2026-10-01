package filepicker

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"slices"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/fileaccess"
)

func folderFixture(t *testing.T, name string, directory bool) fyne.URI {
	t.Helper()
	uri, err := fileaccess.FromRecord(fileaccess.Record{URI: storage.NewFileURI(name).String(), Bookmark: []byte(name), Directory: directory})
	if err != nil {
		t.Fatal(err)
	}
	return uri
}

func TestFolderApprovalSurvivesNewAuthorizer(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	root := t.TempDir()
	folder := folderFixture(t, root, true)
	first := folderFixture(t, filepath.Join(root, "first.jpg"), false)
	second := folderFixture(t, filepath.Join(root, "second.jpg"), false)
	prompts, active := 0, 0
	for _, file := range []fyne.URI{first, second} {
		authorizer := NewFolderAuthorizer(app.Preferences())
		authorizer.choose = func(_ context.Context, _ string) (fyne.URI, error) {
			prompts++
			return folder, nil
		}
		authorizer.acquire = func(_ context.Context, uri fyne.URI) (fyne.URI, func(), error) {
			active++
			return uri, func() { active-- }, nil
		}
		got, err := authorizer.authorize(context.Background(), []fyne.URI{file})
		if err != nil || len(got) != 1 {
			t.Fatalf("result %v, %v", got, err)
		}
		record := fileaccess.Snapshot(got[0])
		if got[0].String() != file.String() || !record.Directory || record.Relative != file.Name() {
			t.Fatalf("lost selected image or folder authority: %+v", record)
		}
		if active != 0 {
			t.Fatal("native scope retained after authorization")
		}
	}
	if prompts != 1 {
		t.Fatalf("folder prompts across launches = %d, want 1", prompts)
	}
}

func TestSavedFolderApprovalValidation(t *testing.T) {
	for _, scenario := range []string{"renewed", "moved", "wrong-folder", "revoked", "offline", "malformed", "file-grant", "cancel-lookup", "cancel-panel", "multiple"} {
		t.Run(scenario, func(t *testing.T) {
			app := test.NewApp()
			t.Cleanup(app.Quit)
			root, other := t.TempDir(), t.TempDir()
			folder := folderFixture(t, root, true)
			file := folderFixture(t, filepath.Join(root, "image.jpg"), false)
			record := fileaccess.Snapshot(folder)
			if scenario == "moved" {
				record.URI = storage.NewFileURI(filepath.Join(other, "old")).String()
			}
			if scenario == "file-grant" {
				record = fileaccess.Snapshot(file)
			}
			encoded, err := json.Marshal([]fileaccess.Record{record})
			if err != nil {
				t.Fatal(err)
			}
			original := string(encoded)
			if scenario == "malformed" {
				original = "invalid json"
			}
			app.Preferences().SetString(keySiblingFolderGrants, original)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			authorizer := NewFolderAuthorizer(app.Preferences())
			prompts, acquired, released := 0, 0, 0
			authorizer.choose = func(_ context.Context, _ string) (fyne.URI, error) {
				prompts++
				if scenario == "cancel-panel" {
					cancel()
					return folder, nil
				}
				return nil, nil
			}
			authorizer.acquire = func(_ context.Context, uri fyne.URI) (fyne.URI, func(), error) {
				if !fileaccess.Snapshot(uri).Directory {
					return uri, func() {}, nil
				}
				if scenario == "revoked" || scenario == "cancel-panel" {
					return nil, nil, errors.New("revoked")
				}
				acquired++
				resolved := fileaccess.Snapshot(folder)
				resolved.Bookmark = []byte("renewed")
				if scenario == "wrong-folder" {
					resolved.URI = storage.NewFileURI(other).String()
				}
				if scenario == "offline" {
					resolved.URI = storage.NewFileURI(filepath.Join(other, "missing")).String()
				}
				uri, err := fileaccess.FromRecord(resolved)
				if err != nil {
					t.Fatal(err)
				}
				if scenario == "cancel-lookup" {
					cancel()
				}
				return uri, func() { released++ }, nil
			}
			inputs := []fyne.URI{file}
			if scenario == "multiple" {
				inputs = append(inputs, folderFixture(t, filepath.Join(root, "second.jpg"), false))
			}
			got, err := authorizer.authorize(ctx, inputs)
			if acquired != released {
				t.Fatalf("scopes acquired %d released %d", acquired, released)
			}
			switch scenario {
			case "renewed", "moved":
				if err != nil || len(got) != 1 || prompts != 0 || string(fileaccess.Snapshot(got[0]).Bookmark) != "renewed" {
					t.Fatalf("saved access not reused: result=%v prompts=%d err=%v", got, prompts, err)
				}
				var saved []fileaccess.Record
				if err := json.Unmarshal([]byte(app.Preferences().String(keySiblingFolderGrants)), &saved); err != nil {
					t.Fatal(err)
				}
				if len(saved) != 1 || saved[0].URI != folder.String() || string(saved[0].Bookmark) != "renewed" || saved[0].Relative != "" {
					t.Fatalf("renewal not persisted: %+v", saved)
				}
			case "cancel-lookup", "cancel-panel":
				if !errors.Is(err, context.Canceled) || got != nil {
					t.Fatalf("cancelled operation delivered %v, %v", got, err)
				}
				wantPrompts := 0
				if scenario == "cancel-panel" {
					wantPrompts = 1
				}
				if prompts != wantPrompts {
					t.Fatalf("prompts %d, want %d", prompts, wantPrompts)
				}
			default:
				wantPrompts := 1
				if scenario == "multiple" {
					wantPrompts = 0
				}
				if err != nil || !slices.Equal(got, inputs) || prompts != wantPrompts {
					t.Fatalf("selection=%v prompts=%d err=%v", got, prompts, err)
				}
			}
			if scenario != "renewed" && scenario != "moved" && app.Preferences().String(keySiblingFolderGrants) != original {
				t.Fatal("unsuccessful or cancelled authorization changed saved grants")
			}
		})
	}
}

func TestSiblingApprovalUsesResolvedImageParent(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	old, current := t.TempDir(), t.TempDir()
	source := folderFixture(t, filepath.Join(old, "photo.jpg"), false)
	resolved := folderFixture(t, filepath.Join(current, "photo.jpg"), false)
	folder := folderFixture(t, current, true)
	authorizer := NewFolderAuthorizer(app.Preferences())
	active := 0
	authorizer.acquire = func(_ context.Context, uri fyne.URI) (fyne.URI, func(), error) {
		if uri != source {
			t.Fatal("unexpected source acquisition")
		}
		active++
		return resolved, func() { active-- }, nil
	}
	authorizer.choose = func(_ context.Context, directory string) (fyne.URI, error) {
		if filepath.Clean(directory) != filepath.Clean(current) {
			t.Fatalf("permission panel starts at stale folder %q; want %q", directory, current)
		}
		if active != 1 {
			t.Fatal("source resolution authority ended before folder selection")
		}
		return folder, nil
	}
	got, err := authorizer.authorize(context.Background(), []fyne.URI{source})
	if err != nil || len(got) != 1 {
		t.Fatalf("authorization %v, %v", got, err)
	}
	if got[0].Path() != resolved.Path() || fileaccess.Snapshot(got[0]).Relative != "photo.jpg" || active != 0 {
		t.Fatalf("lost resolved selection or scope release: %v active=%d", got, active)
	}
}
