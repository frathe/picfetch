package similarity

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
)

var managedAnalysisName = regexp.MustCompile(`^[0-9a-f]{64}\.json$`)
var temporaryAnalysisName = regexp.MustCompile(`^\.[A-Za-z0-9_-]{20,64}\.tmp$`)

type managedAnalysis struct {
	root               *os.Root
	name               string
	info               fs.FileInfo
	general, temporary bool
	favorite           *favoriteAnalysis
}
type analysisInventory struct {
	records   []managedAnalysis
	usage     CacheUsage
	roots     []*os.Root
	favorites *favoriteInventory
}

func (i *analysisInventory) close() {
	i.favorites.close()
	for _, root := range i.roots {
		_ = root.Close()
	}
}
func (i *analysisInventory) scan(ctx context.Context, roots CacheRoots, progress func(CacheProgress)) error {
	var failures []error
	add := func(base, relative string, general bool, favorite *favoriteAnalysis) {
		var root *os.Root
		var err error
		if favorite != nil {
			root, err = openAnalysisDirectory(favorite.root, "analysis")
		} else {
			parent, openErr := os.OpenRoot(base)
			if errors.Is(openErr, os.ErrNotExist) {
				return
			}
			if openErr != nil {
				failures = append(failures, openErr)
				return
			}
			root, err = openAnalysisDirectory(parent, relative)
			_ = parent.Close()
		}
		if errors.Is(err, os.ErrNotExist) {
			return
		}
		if err != nil {
			failures = append(failures, err)
			return
		}
		i.roots = append(i.roots, root)
		dir, err := root.Open(".")
		if err != nil {
			failures = append(failures, err)
			return
		}
		entries, err := dir.ReadDir(-1)
		_ = dir.Close()
		if err != nil {
			failures = append(failures, err)
		}
		for _, entry := range entries {
			if ctx.Err() != nil {
				return
			}
			temporary := temporaryAnalysisName.MatchString(entry.Name())
			if !temporary && !managedAnalysisName.MatchString(entry.Name()) {
				continue
			}
			info, err := root.Lstat(entry.Name())
			if err != nil {
				failures = append(failures, err)
				continue
			}
			if !info.Mode().IsRegular() {
				continue
			}
			record := managedAnalysis{root: root, name: entry.Name(), info: info, general: general, temporary: temporary, favorite: favorite}
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
