package imaging

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// WriteResult distinguishes an accomplished replacement from cancelled work.
// Cancelling presentation after Committed became true cannot undo the file.
type WriteResult struct {
	Path      string
	Committed bool
}

// fileTransactions coordinates this process's image writers. It is production
// state, never replaced/configured by tests. A claim covers the complete
// read-transform-write transaction, and unrelated paths have independent turns.
var fileTransactions pathTransactions

type pathTransactions struct {
	mu    sync.Mutex
	paths map[string]*pathTurn
}

type pathTurn struct {
	turn chan struct{}
	refs int
}

func (p *pathTransactions) acquire(ctx context.Context, path string) (func(), error) {
	p.mu.Lock()
	if p.paths == nil {
		p.paths = make(map[string]*pathTurn)
	}
	t := p.paths[path]
	if t == nil {
		t = &pathTurn{turn: make(chan struct{}, 1)}
		p.paths[path] = t
	}
	t.refs++
	p.mu.Unlock()

	releaseRef := func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		t.refs--
		if t.refs == 0 {
			delete(p.paths, path)
		}
	}
	select {
	case t.turn <- struct{}{}:
		release := func() { <-t.turn; releaseRef() }
		if err := ctx.Err(); err != nil {
			release()
			return nil, err
		}
		return release, nil
	case <-ctx.Done():
		releaseRef()
		return nil, ctx.Err()
	}
}

func (p *pathTransactions) write(ctx context.Context, path string, create bool, write func(string) (bool, error)) (WriteResult, error) {
	if err := ctx.Err(); err != nil {
		return WriteResult{}, err
	}
	resolved, err := resolvedWritePath(path, create)
	if err != nil {
		return WriteResult{}, err
	}
	result := WriteResult{Path: resolved}
	release, err := p.acquire(ctx, resolved)
	if err != nil {
		return result, err
	}
	defer release()
	result.Committed, err = write(resolved)
	return result, err
}

func resolvedWritePath(path string, create bool) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if !create {
		return filepath.EvalSymlinks(abs)
	}
	dir, err := filepath.EvalSymlinks(filepath.Dir(abs))
	if err != nil {
		return "", err
	}
	// Export confirms a destination name, not the target of a leaf symlink.
	// Keep the leaf unresolved so a link introduced after this check is
	// replaced by the atomic rename instead of redirecting the write.
	resolved := filepath.Join(dir, filepath.Base(abs))
	if info, err := os.Lstat(resolved); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return "", &os.PathError{Op: "export", Path: abs, Err: errors.New("destination is a symbolic link")}
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	return resolved, nil
}

func readFileContext(ctx context.Context, path string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return io.ReadAll(contextRead{ctx: ctx, in: f})
}

type contextRead struct {
	ctx context.Context
	in  io.Reader
}

func (r contextRead) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := r.in.Read(p)
	if cancelled := r.ctx.Err(); cancelled != nil {
		return n, cancelled
	}
	return n, err
}

type contextWrite struct {
	ctx context.Context
	out io.Writer
}

func (w contextWrite) Write(p []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := w.out.Write(p)
	if cancelled := w.ctx.Err(); cancelled != nil {
		return n, cancelled
	}
	return n, err
}
