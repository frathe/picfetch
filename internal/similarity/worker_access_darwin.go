//go:build darwin && appleappstore

package similarity

import (
	"context"
	"os"
	"path/filepath"

	"github.com/frathe/picfetch/internal/fileaccess"
	"github.com/frathe/picfetch/internal/macbundle"
)

func prepareWorkerAccess(ctx context.Context, req *request) (func(), error) {
	root, err := macbundle.CurrentRuntimeDirectory()
	if err != nil {
		return nil, err
	}
	// These app-owned roots must exist before Foundation can grant their child
	// creation. Cache failures remain best-effort and are reported by the worker.
	caches := []string{req.FavoritesDir, req.GeneralAnalysisDir}
	if req.Search != nil {
		caches = []string{req.Search.Cache.Roots.FavoritesDir}
		if req.Search.Cache.LooseEnabled {
			caches = append(caches, req.Search.Cache.Roots.GeneralDir)
		}
	}
	for _, path := range caches {
		if path != "" {
			_ = os.MkdirAll(path, 0700)
		}
	}
	return captureWorkerAccess(ctx, req, filepath.Dir(filepath.Dir(root)), fileaccess.Export)
}
