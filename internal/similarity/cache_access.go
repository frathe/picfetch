package similarity

import (
	"context"
	"errors"
	"os"

	"github.com/frathe/picfetch/internal/favstore"
)

var errAnalysisChanged = errors.New("analysis cache observation changed")

// Deferred maintenance retains identities, never roots. Favorite ownership is
// shared; the cache subdirectory and individual record remain consumer facts.
type analysisDirectory struct {
	base, relative        string
	parentInfo, directory os.FileInfo
	favorite              *favoriteAnalysis
}

type analysisAccess struct {
	observation  *analysisDirectory
	parent, root *os.Root
	owner        *favstore.Access
}

func (d *analysisDirectory) open(ctx context.Context) (*analysisAccess, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	access := &analysisAccess{observation: d}
	var err error
	if d.favorite != nil {
		access.owner, err = d.favorite.owner.AcquireDirectory(ctx)
		if err == nil {
			access.parent = access.owner.Root
		}
	} else {
		access.parent, err = os.OpenRoot(d.base)
	}
	if err != nil {
		return nil, err
	}
	access.root, err = openAnalysisDirectory(access.parent, d.relative)
	if err != nil {
		access.close()
		return nil, err
	}
	return access, nil
}

func (a *analysisAccess) close() {
	if a.root != nil {
		_ = a.root.Close()
	}
	if a.parent != nil {
		_ = a.parent.Close()
	}
}

func (a *analysisAccess) current(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	d := a.observation
	if a.owner != nil {
		if err := a.owner.Current(ctx); err != nil {
			return err
		}
	} else {
		parent, err := os.Stat(d.base)
		if err != nil {
			return err
		}
		opened, err := a.parent.Stat(".")
		if err != nil {
			return err
		}
		if !os.SameFile(parent, d.parentInfo) || !os.SameFile(opened, d.parentInfo) {
			return errAnalysisChanged
		}
	}
	directory, err := a.parent.Stat(d.relative)
	if err != nil {
		return err
	}
	opened, err := a.root.Stat(".")
	if err != nil {
		return err
	}
	if !os.SameFile(directory, d.directory) || !os.SameFile(opened, d.directory) {
		return errAnalysisChanged
	}
	return ctx.Err()
}

func (a *analysisAccess) currentRecord(ctx context.Context, record managedAnalysis) error {
	if err := a.current(ctx); err != nil {
		return err
	}
	info, err := a.root.Lstat(record.name)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || !sameVersion(record.info, info) {
		return errAnalysisChanged
	}
	return ctx.Err()
}

func (r managedAnalysis) remove(ctx context.Context, staleOnly bool) (removed, unavailable bool, err error) {
	// Observed changes are conservative skips, not failed unlink operations.
	defer func() {
		if errors.Is(err, errAnalysisChanged) || errors.Is(err, favstore.ErrRetired) || errors.Is(err, os.ErrNotExist) {
			removed, unavailable, err = false, true, nil
		}
	}()
	access, err := r.directory.open(ctx)
	if err != nil {
		return false, false, err
	}
	defer access.close()
	if err := access.currentRecord(ctx, r); err != nil {
		return false, false, err
	}
	if staleOnly {
		stale, unavailable := r.stale(access.root)
		if !stale {
			return false, unavailable, nil
		}
	}
	// Classification and inventory are observations, not deletion authority.
	// Another writer may replace either the list or record without this lease.
	if err := access.currentRecord(ctx, r); err != nil {
		return false, false, err
	}
	if err := access.root.Remove(r.name); err != nil {
		return false, false, err
	}
	return true, false, nil
}
