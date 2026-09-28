package similarity

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
)

var managedAnalysisName = regexp.MustCompile(`^[0-9a-f]{64}\.json$`)
var temporaryAnalysisName = regexp.MustCompile(`^\.[A-Za-z0-9_-]{20,64}\.tmp$`)

type managedAnalysis struct {
	directory          *analysisDirectory
	name               string
	info               fs.FileInfo
	general, temporary bool
	favorite           *favoriteAnalysis
}
type analysisInventory struct {
	records   []managedAnalysis
	usage     CacheUsage
	favorites *favoriteInventory
}

func (i *analysisInventory) close() {
	i.favorites.close()
}
func (i *analysisInventory) scan(ctx context.Context, roots CacheRoots, progress func(CacheProgress)) error {
	var failures []error
	add := func(base, relative string, general bool, favorite *favoriteAnalysis) {
		if favorite != nil {
			base = favorite.owner.Path()
		}
		base, err := filepath.Abs(base)
		if err != nil {
			failures = append(failures, err)
			return
		}
		observation := &analysisDirectory{base: base, relative: relative, favorite: favorite}
		access, err := observation.open(ctx)
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		if err != nil {
			failures = append(failures, err)
			return
		}
		defer access.close()
		observation.parentInfo, err = access.parent.Stat(".")
		if err != nil {
			failures = append(failures, err)
			return
		}
		observation.directory, err = access.root.Stat(".")
		if err != nil {
			failures = append(failures, err)
			return
		}
		dir, err := access.root.Open(".")
		if err != nil {
			failures = append(failures, err)
			return
		}
		defer func() { _ = dir.Close() }()
		for {
			if err := access.current(ctx); err != nil {
				failures = append(failures, err)
				return
			}
			entries, readErr := dir.ReadDir(64)
			for _, entry := range entries {
				if ctx.Err() != nil {
					return
				}
				temporary := temporaryAnalysisName.MatchString(entry.Name())
				if !temporary && !managedAnalysisName.MatchString(entry.Name()) {
					continue
				}
				info, err := access.root.Lstat(entry.Name())
				if err != nil {
					failures = append(failures, err)
					continue
				}
				if !info.Mode().IsRegular() {
					continue
				}
				record := managedAnalysis{directory: observation, name: entry.Name(), info: info, general: general, temporary: temporary, favorite: favorite}
				i.records = append(i.records, record)
				size := &i.usage.Favorite
				if general {
					size = &i.usage.General
				}
				size.Bytes += uint64(info.Size())
				if !temporary {
					size.Records++
				}
				if progress != nil {
					progress(CacheProgress{Phase: "inspect", Records: len(i.records), Bytes: i.usage.General.Bytes + i.usage.Favorite.Bytes})
				}
			}
			if readErr != nil && !errors.Is(readErr, io.EOF) {
				failures = append(failures, readErr)
			}
			err = access.current(ctx)
			if err != nil {
				failures = append(failures, err)
			}
			if readErr != nil || err != nil {
				return
			}
		}
	}
	if roots.GeneralDir != "" {
		add(roots.GeneralDir, "v1", true, nil)
	}
	i.usage.General.Incomplete = len(failures) > 0 || ctx.Err() != nil
	generalFailures := len(failures)
	if roots.FavoritesDir != "" && ctx.Err() == nil {
		var err error
		i.favorites, err = openFavoriteInventory(ctx, roots.FavoritesDir)
		if err != nil {
			failures = append(failures, err)
		}
		for _, favorite := range i.favorites.favorites {
			if ctx.Err() != nil {
				break
			}
			add(roots.FavoritesDir, "analysis", false, favorite)
		}
	}
	if err := ctx.Err(); err != nil {
		failures = append(failures, err)
	}
	i.usage.Favorite.Incomplete = len(failures) > generalFailures || ctx.Err() != nil
	i.usage.Incomplete = i.usage.General.Incomplete || i.usage.Favorite.Incomplete
	return errors.Join(failures...)
}

// Windows can report an existing non-directory as missing when OpenRoot asks
// for a directory handle. An unusable cache is a partial-inventory error, not
// an empty cache; inspect only that failure through the same retained root.
func openAnalysisDirectory(parent *os.Root, name string) (*os.Root, error) {
	root, err := parent.OpenRoot(name)
	if !errors.Is(err, os.ErrNotExist) {
		return root, err
	}
	if _, statErr := parent.Lstat(name); statErr == nil {
		return nil, fmt.Errorf("analysis path %q exists but cannot be opened as a directory: %v", name, err)
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return nil, statErr
	}
	return nil, err
}
