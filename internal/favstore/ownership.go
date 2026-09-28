package favstore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path"
	"path/filepath"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
)

// ErrRetired means an observation no longer names the captured Favorite.
var ErrRetired = errors.New("favorite owner retired")

// Definition is the complete ordered saved list and its captured owner.
type Definition struct {
	Owner *Owner
	Paths []string
	base  string
}

// Files interprets the saved occurrences as native file URIs.
func (d Definition) Files() []fyne.URI {
	files := make([]fyne.URI, len(d.Paths))
	for i, source := range d.Paths {
		files[i] = storage.NewFileURI(interpretedPath(d.base, source))
	}
	return files
}

// Owner retains a Favorite observation without keeping idle directory handles.
type Owner struct {
	dir             string
	base            string
	version         string
	directory, list os.FileInfo
	retired         atomic.Bool
}

// Path is the absolute directory pathname captured for this owner.
func (o *Owner) Path() string { return o.dir }

// Version fingerprints complete encoded membership and its modification time.
// Consumers may use it for their own namespaces; currentness uses file identity.
func (o *Owner) Version() string { return o.version }

// SourceKey interprets a source under this operation's captured working directory.
// It performs lexical normalization only, without inspecting the source.
func (o *Owner) SourceKey(source string) string { return sourceKey(o.base, source) }

func sourceKey(base, source string) string {
	return filepath.Clean(storage.NewFileURI(interpretedPath(base, source)).Path())
}

func interpretedPath(base, value string) string {
	// Fyne also accepts slash-rooted paths on Windows. Match its URI convention
	// while resolving genuine relative paths against our captured base.
	if path.IsAbs(value) || filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(base, value)
}

// Access confines one bounded operation to a Favorite directory.
type Access struct {
	Root          *os.Root
	owner         *Owner
	directoryOnly bool
}

// Close releases this operation's directory access.
func (a *Access) Close() error { return a.Root.Close() }

// Acquire opens access to this observation's directory.
func (o *Owner) Acquire(ctx context.Context) (*Access, error) {
	if o.list == nil {
		return nil, ErrRetired
	}
	return o.acquire(ctx, false)
}

// AcquireDirectory admits explicit maintenance of a captured directory even
// when its membership is unknown. It never grants membership or write policy.
func (o *Owner) AcquireDirectory(ctx context.Context) (*Access, error) {
	return o.acquire(ctx, true)
}

func (o *Owner) acquire(ctx context.Context, directoryOnly bool) (*Access, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if o.retired.Load() {
		return nil, ErrRetired
	}
	root, err := os.OpenRoot(o.dir)
	if err != nil {
		return nil, o.retire(err)
	}
	a := &Access{Root: root, owner: o, directoryOnly: directoryOnly}
	if err := a.Current(ctx); err != nil {
		_ = root.Close()
		return nil, err
	}
	return a, nil
}

func (o *Owner) retire(err error) error {
	o.retired.Store(true)
	return errors.Join(ErrRetired, err)
}

// Current rechecks both the original pathname and the observed list. It does
// not exclude an external mutation after the check; effects stay root-relative.
func (a *Access) Current(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	o := a.owner
	if o.retired.Load() {
		return ErrRetired
	}
	directory, err := os.Stat(o.dir)
	if err != nil || !os.SameFile(directory, o.directory) {
		return o.retire(err)
	}
	opened, err := a.Root.Stat(".")
	if err != nil || !os.SameFile(opened, o.directory) {
		return o.retire(err)
	}
	if !a.directoryOnly || o.list != nil {
		list, err := a.Root.Stat(fileListName)
		if err != nil || !sameList(o.list, list) {
			return o.retire(err)
		}
	}
	return ctx.Err()
}

func sameList(a, b os.FileInfo) bool {
	return a != nil && b != nil && os.SameFile(a, b) && a.Size() == b.Size() && a.ModTime().Equal(b.ModTime())
}

// Open validates a Favorite and captures its ownership.
func Open(ctx context.Context, dir string) (Definition, error) {
	return (&Store{}).Open(ctx, dir)
}

// Store performs independent Favorite operations. Its zero value uses native
// storage; each operation owns its handles and captures its working directory.
type Store struct {
	readFile func(context.Context, *os.File) ([]byte, error)
}

// Open reads the complete definition through this store's filesystem seam.
func (s *Store) Open(ctx context.Context, dir string) (Definition, error) {
	if err := ctx.Err(); err != nil {
		return Definition{}, err
	}
	base, err := os.Getwd()
	if err != nil {
		return Definition{}, err
	}
	return s.open(ctx, dir, base)
}

func (s *Store) open(ctx context.Context, dir, base string) (Definition, error) {
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(base, dir)
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return Definition{}, err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return Definition{}, err
	}
	defer func() { _ = root.Close() }()
	directory, err := root.Stat(".")
	if err != nil {
		return Definition{}, err
	}
	list, err := root.Stat(fileListName)
	if err != nil {
		return Definition{}, err
	}
	file, err := root.Open(fileListName)
	if err != nil {
		return Definition{}, err
	}
	defer func() { _ = file.Close() }()
	opened, err := file.Stat()
	if err != nil {
		return Definition{}, err
	}
	if !sameList(list, opened) {
		return Definition{}, ErrRetired
	}
	read := s.readFile
	if read == nil {
		read = func(ctx context.Context, f *os.File) ([]byte, error) { return readDefinition(ctx, f) }
	}
	data, err := read(ctx, file)
	if err != nil {
		return Definition{}, err
	}
	paths, err := decodeMembership(ctx, data)
	if err != nil {
		return Definition{}, err
	}
	hash := sha256.New()
	_, _ = hash.Write(data)
	_, _ = hash.Write([]byte(list.ModTime().UTC().Format("20060102T150405.000000000")))
	owner := &Owner{dir: dir, base: base, directory: directory, list: list, version: hex.EncodeToString(hash.Sum(nil))}
	access := &Access{Root: root, owner: owner}
	if err := access.Current(ctx); err != nil {
		return Definition{}, err
	}
	return Definition{Owner: owner, Paths: paths, base: base}, nil
}

// Observe captures target identity without admitting its membership. This also
// permits explicit inspection/clearing of a damaged Favorite. A returned owner
// can accompany a list-stat error, retaining only safe directory access.
func Observe(ctx context.Context, dir string) (*Owner, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	base, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(base, dir)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	directory, err := root.Stat(".")
	if err != nil {
		return nil, err
	}
	owner := &Owner{dir: dir, base: base, directory: directory}
	owner.list, err = root.Stat(fileListName)
	return owner, err
}
