package grid

import (
	"testing"
	"time"

	"github.com/frathe/picfetch/internal/dupes"
)

// A pass that starts with nothing in flight must clear the previous
// pass's mid-window timestamp, or its own first mid-window apply is
// swallowed by a 250 ms floor left over from work that already finished.
func TestBeginPass_ClearsHideApplyFloorWhenNothingInFlight(t *testing.T) {
	e := &hashEngine{}
	e.hideApplyAt.Store(time.Now().UnixMilli())

	e.beginPass(3)

	if got := e.hideApplyAt.Load(); got != 0 {
		t.Errorf("hideApplyAt = %d, want 0", got)
	}
	if got := e.hashJobs.Load(); got != 3 {
		t.Errorf("hashJobs = %d, want 3", got)
	}
}

// Jobs added while a pass is still draining are the same pass, so the
// floor it has already established must survive.
func TestBeginPass_KeepsHideApplyFloorWhileJobsInFlight(t *testing.T) {
	e := &hashEngine{}
	stamp := time.Now().UnixMilli()
	e.hideApplyAt.Store(stamp)
	e.hashJobs.Store(2)

	e.beginPass(3)

	if got := e.hideApplyAt.Load(); got != stamp {
		t.Errorf("hideApplyAt = %d, want %d", got, stamp)
	}
	if got := e.hashJobs.Load(); got != 5 {
		t.Errorf("hashJobs = %d, want 5", got)
	}
}

// A nil URI at some index must be skipped, not dereferenced: Run reads
// u.String() as the model's key three lines after FileAt.
func TestRun_SkipsNilURI(t *testing.T) {
	host := hostWith(t, "a.jpg", "b.jpg")
	host.files = append(host.files, nil)

	g := newOverview(t, host)

	n := g.hashes.Run(func(_ dupes.Groups, _ int32, _ uint64) {})
	// The two real files really do reach the decode pool, so this drains
	// them before the harness closes the window under them - the same
	// barrier every hashing test in dupes_test.go takes.
	g.Settle()

	if n != 2 {
		t.Errorf("Run queued %d jobs, want 2 (the nil URI must be skipped)", n)
	}
}
