package similarity

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/fileaccess"
	"github.com/frathe/picfetch/internal/heic"
)

type transferExport func(context.Context, fyne.URI) (fileaccess.Transfer, func(), error)

// captureWorkerAccess retains exact source authority through process exit.
// Optional cache/source failures keep the existing per-item/cache error paths;
// cancellation or unavailable model/code authority refuses launch entirely.
func captureWorkerAccess(ctx context.Context, req *request, app string, export transferExport) (func(), error) {
	var releases []func()
	var once sync.Once
	release := func() {
		once.Do(func() {
			for i := len(releases) - 1; i >= 0; i-- {
				releases[i]()
			}
		})
	}
	seen := map[string]bool{}
	add := func(uri fyne.URI, required bool) error {
		path := uri.Path()
		if seen[path] {
			return nil
		}
		seen[path] = true
		grant, stop, err := export(ctx, uri)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if required {
				return err
			}
			return nil
		}
		releases = append(releases, stop)
		req.Access = append(req.Access, grant)
		return nil
	}
	for _, path := range []string{app, filepath.Join(req.Assets, "vision_model.onnx"), filepath.Join(req.Assets, "preprocessor_config.json")} {
		if err := add(storage.NewFileURI(path), true); err != nil {
			release()
			return nil, err
		}
	}
	paths := req.Paths
	caches := []string{req.FavoritesDir, req.GeneralAnalysisDir}
	if req.Search != nil {
		paths = req.Search.Paths
		caches = []string{req.Search.Cache.Roots.FavoritesDir}
		if req.Search.Cache.LooseEnabled {
			caches = append(caches, req.Search.Cache.Roots.GeneralDir)
		}
	}
	for _, path := range paths {
		if err := add(fileaccess.SourceForPath(ctx, path), false); err != nil {
			release()
			return nil, err
		}
	}
	for _, path := range caches {
		if path == "" {
			continue
		}
		if err := add(storage.NewFileURI(path), false); err != nil {
			release()
			return nil, err
		}
	}
	// Bound the complete encoded request, including base64 bookmark expansion.
	payload, err := json.Marshal(req)
	if err == nil && len(payload) > workerRequestLimit {
		err = fmt.Errorf("worker request exceeds %d bytes", workerRequestLimit)
	}
	if err == nil {
		err = ctx.Err()
	}
	if err != nil {
		release()
		return nil, err
	}
	return release, nil
}

func importWorkerAccess(ctx context.Context, grants []fileaccess.Transfer) (func(), error) {
	var releases []func()
	release := func() {
		for i := len(releases) - 1; i >= 0; i-- {
			releases[i]()
		}
	}
	for _, grant := range grants {
		stop, err := fileaccess.Import(ctx, grant)
		if err != nil {
			release()
			return nil, err
		}
		releases = append(releases, stop)
	}
	return release, nil
}

// Test command adapters own their own transport. Production captures authority
// for both worker protocols before admission and retains it through Wait.
func (c Client) captureAccess(ctx context.Context, req *request) (func(), error) {
	req.captureHEIC(heic.FromContext(ctx))
	if c.command != nil {
		return func() {}, nil
	}
	return prepareWorkerAccess(ctx, req)
}
