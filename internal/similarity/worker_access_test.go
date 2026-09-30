package similarity

import (
	"context"
	"errors"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/fileaccess"
)

func TestWorkerAccessCapturesExactPathsUntilRetirement(t *testing.T) {
	source, err := fileaccess.FromRecord(fileaccess.Record{URI: "file:///photos/one.png", Bookmark: []byte("persistent")})
	if err != nil {
		t.Fatal(err)
	}
	ctx := fileaccess.WithSources(context.Background(), []fyne.URI{source})
	req := request{Assets: "/models", Paths: []string{source.Path(), source.Path()}, GeneralAnalysisDir: "/cache"}
	active := map[string]bool{}
	release, err := captureWorkerAccess(ctx, &req, "/Applications/PicFetch.app", func(_ context.Context, uri fyne.URI) (fileaccess.Transfer, func(), error) {
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
	// Fyne URI paths use slashes on Windows too.
	want := []string{"/Applications/PicFetch.app", "/models/vision_model.onnx", "/models/preprocessor_config.json", "/photos/one.png", "/cache"}
	if len(req.Access) != len(want) {
		t.Fatalf("unexpected grants: %+v", req.Access)
	}
	for _, path := range want {
		if !active[path] {
			t.Fatalf("access lost: %s", path)
		}
	}
	if active["/photos"] {
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
