package favstore

import (
	"context"
	"os"

	"github.com/frathe/picfetch/internal/trash"
)

// RemovalResult distinguishes a completed native move from its presentation.
type RemovalResult struct{ Committed bool }

// Remove submits only the captured owner. No directory handle survives into
// Trash: Windows may need to move the directory itself. The final identity
// check cannot exclude an unrelated external mutation after it returns.
func (_ *Store) Remove(ctx context.Context, target *Target) (result RemovalResult, err error) {
	defer func() { target.finishMutation(result.Committed, err) }()
	if err := target.current(ctx); err != nil {
		return RemovalResult{}, err
	}
	if !target.Occupied() {
		return RemovalResult{}, os.ErrNotExist
	}
	if err := ctx.Err(); err != nil {
		return RemovalResult{}, err
	}
	if err := trash.Move(target.owner.Path()); err != nil {
		return RemovalResult{}, err
	}
	target.owner.retired.Store(true)
	return RemovalResult{Committed: true}, nil
}
