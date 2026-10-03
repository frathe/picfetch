package favstore

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/fileaccess"
)

// ErrConflict requires a fresh target observation and user decision.
var ErrConflict = errors.New("favorite target changed")

// Target captures the intended name before a mutation is confirmed.
type Target struct {
	dir, name, base string
	anchor          string
	anchorInfo      os.FileInfo
	missing         []string
	owner           *Owner
	retired         atomic.Bool
}

// Occupied reports the captured name's occupancy, not its membership validity.
func (t *Target) Occupied() bool { return t.owner != nil }

func (t *Target) finishMutation(committed bool, err error) {
	if committed || errors.Is(err, ErrConflict) && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		t.retired.Store(true)
	}
}

// Capture observes a mutation target without reading its membership.
func (_ *Store) Capture(ctx context.Context, dir, name string) (*Target, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !ValidName(name) {
		return nil, fmt.Errorf("invalid favorite name %q", name)
	}
	base, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(base, dir)
	}
	target := &Target{dir: filepath.Clean(dir), name: name, base: base}
	target.anchor = target.dir
	for {
		target.anchorInfo, err = os.Stat(target.anchor)
		if err == nil {
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		target.missing = append(target.missing, target.anchor)
		target.anchor = filepath.Dir(target.anchor)
	}
	if !target.anchorInfo.IsDir() {
		return nil, fmt.Errorf("favorite root is not a directory: %s", target.anchor)
	}
	if len(target.missing) == 0 {
		target.owner, err = Observe(ctx, Dir(target.dir, name))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		if target.owner != nil {
			target.owner.base = base
		}
	}
	return target, ctx.Err()
}

func (t *Target) current(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if t.retired.Load() {
		return ErrConflict
	}
	info, err := os.Stat(t.anchor)
	if err != nil || !os.SameFile(info, t.anchorInfo) {
		return errors.Join(ErrConflict, err)
	}
	for _, missing := range t.missing {
		if _, err := os.Lstat(missing); !errors.Is(err, os.ErrNotExist) {
			return errors.Join(ErrConflict, err)
		}
	}
	if t.owner == nil {
		if _, err := os.Lstat(Dir(t.dir, t.name)); !errors.Is(err, os.ErrNotExist) {
			return errors.Join(ErrConflict, err)
		}
		return ctx.Err()
	}
	access, err := t.owner.AcquireDirectory(ctx)
	if err != nil {
		return errors.Join(ErrConflict, err)
	}
	defer func() { _ = access.Close() }()
	return mutationCurrent(ctx, access)
}

func mutationCurrent(ctx context.Context, access *Access) error {
	if err := access.Current(ctx); err != nil {
		return errors.Join(ErrConflict, err)
	}
	if access.owner.list == nil {
		if _, err := access.Root.Stat(fileListName); !errors.Is(err, os.ErrNotExist) {
			return errors.Join(ErrConflict, err)
		}
	}
	return ctx.Err()
}

// acquire creates only paths observed absent, then confines publication to the
// captured directory. Existing paths never become implicit overwrite consent.
func (t *Target) acquire(ctx context.Context) (*Access, error) {
	if err := t.current(ctx); err != nil {
		return nil, err
	}
	anchor, err := os.OpenRoot(t.anchor)
	if err != nil {
		return nil, err
	}
	defer func() { _ = anchor.Close() }()
	info, err := anchor.Stat(".")
	if err != nil || !os.SameFile(info, t.anchorInfo) {
		return nil, errors.Join(ErrConflict, err)
	}
	for i := len(t.missing) - 1; i >= 0; i-- {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		relative, err := filepath.Rel(t.anchor, t.missing[i])
		if err != nil {
			return nil, err
		}
		if err := anchor.Mkdir(relative, 0700); err != nil {
			return nil, errors.Join(ErrConflict, err)
		}
	}
	relative, err := filepath.Rel(t.anchor, t.dir)
	if err != nil {
		return nil, err
	}
	base, err := anchor.OpenRoot(relative)
	if err != nil {
		return nil, err
	}
	defer func() { _ = base.Close() }()
	if t.owner == nil {
		if err := base.Mkdir(t.name, 0700); err != nil {
			return nil, errors.Join(ErrConflict, err)
		}
	}
	root, err := base.OpenRoot(t.name)
	if err != nil {
		return nil, err
	}
	owner := t.owner
	if owner == nil {
		info, err := root.Stat(".")
		if err != nil {
			_ = root.Close()
			return nil, err
		}
		owner = &Owner{dir: Dir(t.dir, t.name), base: t.base, directory: info}
	}
	access := &Access{Root: root, owner: owner, directoryOnly: true}
	if err := mutationCurrent(ctx, access); err != nil {
		_ = root.Close()
		return nil, err
	}
	return access, nil
}

// SaveResult distinguishes a completed publication from cancelled presentation.
type SaveResult struct {
	Committed  bool
	Definition Definition
}

// Save writes the captured target and returns its committed ownership.
func (s *Store) Save(ctx context.Context, target *Target, files []fyne.URI) (result SaveResult, err error) {
	defer func() { target.finishMutation(result.Committed, err) }()
	data, paths, err := encodeMembership(ctx, files)
	if err != nil {
		return SaveResult{}, err
	}
	access, err := target.acquire(ctx)
	if err != nil {
		return SaveResult{}, err
	}
	defer func() { _ = access.Close() }()
	if err := access.Root.Chmod(".", 0700); err != nil {
		return SaveResult{}, err
	}
	name := ".file-list-" + rand.Text() + ".json"
	file, err := access.Root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return SaveResult{}, err
	}
	defer func() { _ = file.Close(); _ = access.Root.Remove(name) }()
	remaining := data
	for len(remaining) > 0 {
		if err := ctx.Err(); err != nil {
			return SaveResult{}, err
		}
		chunk := min(len(remaining), 32*1024)
		if _, err := file.Write(remaining[:chunk]); err != nil {
			return SaveResult{}, err
		}
		remaining = remaining[chunk:]
	}
	if err := file.Sync(); err != nil {
		return SaveResult{}, err
	}
	list, err := file.Stat()
	if err != nil {
		return SaveResult{}, err
	}
	if err := file.Close(); err != nil {
		return SaveResult{}, err
	}
	if s.beforePublish != nil {
		s.beforePublish()
	}
	anchorInfo, err := os.Stat(target.anchor)
	if err != nil || !os.SameFile(anchorInfo, target.anchorInfo) {
		return SaveResult{}, errors.Join(ErrConflict, err)
	}
	if err := mutationCurrent(ctx, access); err != nil {
		return SaveResult{}, err
	}
	if err := access.Root.Rename(name, fileListName); err != nil {
		return SaveResult{}, err
	}
	// Capture from the file just published, never reopen a pathname which could
	// already denote a replacement. The caller retains this committed effect even
	// if cancellation or external replacement follows the rename.
	owner := &Owner{dir: access.owner.dir, base: target.base, directory: access.owner.directory, list: list}
	owner.version = definitionVersion(data, list)
	sources := make([]fyne.URI, len(files))
	for i, file := range files {
		if fileaccess.HasScope(file) {
			sources[i] = file
		}
	}
	result = SaveResult{Committed: true, Definition: Definition{Owner: owner, Paths: paths, base: target.base, sources: sources}}
	if s.afterPublish != nil {
		s.afterPublish()
	}
	return result, nil
}
