package ui

import (
	"fmt"
	"sync"
	"testing"

	"github.com/frathe/picfetch/internal/uitest"
)

// dupes.Model.Compute runs on a hash-pool worker (internal/ui/grid's
// hashengine) while the UI goroutine replaces v.state.files. Reaching the
// live slice through dupeFileSet's Count()/KeyAt(i) pair means a shrink
// landing between those two reads, or part-way through the loop over
// them, indexes past the end - and the slice header itself is read with
// no synchronization at all.
//
// Run under -race. This is the regression guard for that: it must not
// report a race and must not panic.
func TestDupeCompute_ConcurrentWithFileRemoval(t *testing.T) {
	names := make([]string, 48)
	for i := range names {
		names[i] = fmt.Sprintf("f%02d.jpg", i)
	}
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempDirJPEGURIs(t, names...)...)

	var wg sync.WaitGroup
	wg.Go(func() {
		for range 500 {
			v.dupes.Compute()
		}
	})

	for v.FileCount() > 1 {
		v.RemoveFile(v.FileCount() - 1)
	}

	wg.Wait()
}
