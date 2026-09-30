package similarity

import (
	"context"
	"errors"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/fileaccess"
)

func TestWorkerAccessCapturesExactPathsUntilRetirement(t *testing.T) {
	root := storage.NewFileURI(t.TempDir()).Path()
	source, err := fileaccess.FromRecord(fileaccess.Record{URI: storage.NewFileURI(root + "/photos/one.png").String(), Bookmark: []byte("persistent")})
	if err != nil {
		t.Fatal(err)
	}
	ctx := fileaccess.WithSources(context.Background(), []fyne.URI{source})
	req := request{Assets: root + "/models", Paths: []string{source.Path(), source.Path()}, GeneralAnalysisDir: root + "/cache"}
	active := map[string]bool{}
	release, err := captureWorkerAccess(ctx, &req, root+"/PicFetch.app", func(_ context.Context, uri fyne.URI) (fileaccess.Transfer, func(), error) {
		path := uri.Path()
		if path == source.Path() && uri != source {
			t.Fatal("source authority flattened")
		}
		active[path] = true
		return fileaccess.Transfer{Path: path, Bookmark: []byte("implicit")}, func() { active[path] = false }, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Use absolute host paths: Windows roots need a volume as well as slashes.
	want := []string{root + "/PicFetch.app", root + "/models/vision_model.onnx", root + "/models/preprocessor_config.json", root + "/photos/one.png", root + "/cache"}
	if len(req.Access) != len(want) {
		t.Fatalf("unexpected grants: %+v", req.Access)
	}
	for _, path := range want {
		if !active[path] {
			t.Fatalf("access lost: %s", path)
		}
	}
	if active[root+"/photos"] {
		t.Fatal("source expanded to parent")
	}
	release()
	release()
	for path, on := range active {
		if on {
			t.Fatalf("access leaked: %s", path)
		}
	}
}

func TestWorkerAccessFailureRetiresPriorGrants(t *testing.T) {
	req := request{Assets: "/models"}
	retired := 0
	_, err := captureWorkerAccess(context.Background(), &req, "/App.app", func(_ context.Context, uri fyne.URI) (fileaccess.Transfer, func(), error) {
		if uri.Path() != "/App.app" {
			return fileaccess.Transfer{}, nil, errors.New("missing models")
		}
		return fileaccess.Transfer{Path: uri.Path()}, func() { retired++ }, nil
	})
	if err == nil || retired != 1 {
		t.Fatalf("err=%v retired=%d", err, retired)
	}
}
