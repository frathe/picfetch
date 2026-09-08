package deletion

import (
	"errors"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/trash"
	"github.com/frathe/picfetch/internal/uitest"
)

// targetsFor turns a host's whole file set into the target list the grid
// would hand over for a select-all.
func targetsFor(host *fakeHost, indices ...int) []Target {
	ts := make([]Target, 0, len(indices))
	for _, i := range indices {
		ts = append(ts, Target{URI: host.files[i]})
	}

	return ts
}

func TestRequestFiles_NamesTheCountRatherThanEachFile(t *testing.T) {
	host := &fakeHost{files: tempFiles(t, "a.jpg", "b.jpg", "c.jpg")}
	c := newConfirmer(t, host)

	c.RequestFiles(targetsFor(host, 0, 1, 2))

	if !c.Visible() {
		t.Fatal("the card should be visible after RequestFiles")
	}
	if !strings.Contains(c.card.Message().Text, "3") {
		t.Errorf("message = %q, want it to say how many files are about to go", c.card.Message().Text)
	}
}

// TestRequestFiles_OneTargetReadsExactlyLikeTheSingleFilePrompt keeps the
// existing wording - and the golden masters that render it - intact: a batch
// of one is the same question Shift+Delete has always asked.
func TestRequestFiles_OneTargetReadsExactlyLikeTheSingleFilePrompt(t *testing.T) {
	host := &fakeHost{files: tempFiles(t, "sunset.jpg")}
	c := newConfirmer(t, host)

	c.RequestFiles(targetsFor(host, 0))
	batch := c.card.Message().Text

	c.Cancel()
	c.Request()

	if batch != c.card.Message().Text {
		t.Errorf("RequestFiles message = %q, Request message = %q, want them identical for one file", batch, c.card.Message().Text)
	}
}

func TestRequestFiles_NoOpWithNoTargets(t *testing.T) {
	c := newConfirmer(t, &fakeHost{files: tempFiles(t, "a.jpg")})

	c.RequestFiles(nil)

	if c.Visible() {
		t.Error("RequestFiles should do nothing with an empty target list")
	}
}

func TestPerformDelete_BatchRemovesEveryTarget(t *testing.T) {
	stubTrashMove(t)
	host := &fakeHost{files: tempFiles(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg")}
	c := newConfirmer(t, host)

	paths := []string{host.files[0].Path(), host.files[2].Path()}
	keptPath := host.files[1].Path()

	c.RequestFiles(targetsFor(host, 0, 2))
	c.setSelection(true)
	c.confirmSelection()
	c.Settle()

	for _, p := range paths {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Errorf("%s should be gone from disk", p)
		}
	}
	if _, err := os.Stat(keptPath); err != nil {
		t.Errorf("%s was not selected and should still exist: %v", keptPath, err)
	}
	if want := []int{0, 2}; !slices.Equal(host.removedBatch, want) {
		t.Errorf("RemoveFiles call = %v, want %v", host.removedBatch, want)
	}
	if len(host.toasts) != 1 || !strings.Contains(host.toasts[0], "2") {
		t.Errorf("toasts = %v, want one reporting how many moved", host.toasts)
	}
}

// TestPerformDelete_BatchRemovesInOneCall is what keeps the indices valid:
// removing them one at a time would shift every later index out from under
// the list already captured.
func TestPerformDelete_BatchRemovesInOneCall(t *testing.T) {
	stubTrashMove(t)
	host := &fakeHost{files: tempFiles(t, "a.jpg", "b.jpg", "c.jpg")}
	c := newConfirmer(t, host)

	c.RequestFiles(targetsFor(host, 0, 1, 2))
	c.setSelection(true)
	c.confirmSelection()
	c.Settle()

	if host.removeCalls != 1 {
		t.Errorf("RemoveFiles was called %d times, want exactly 1 for one batch", host.removeCalls)
	}
}

func TestPerformDelete_BatchOfEverythingFallsBackToEmptyState(t *testing.T) {
	stubTrashMove(t)
	host := &fakeHost{files: tempFiles(t, "a.jpg", "b.jpg")}
	c := newConfirmer(t, host)

	c.RequestFiles(targetsFor(host, 0, 1))
	c.setSelection(true)
	c.confirmSelection()
	c.Settle()

	if len(host.emptied) != 1 {
		t.Errorf("ShowEmptyStateError calls = %v, want one - nothing is left to show", host.emptied)
	}
	if len(host.shown) != 0 {
		t.Errorf("ShowImage calls = %v, want none with an empty file set", host.shown)
	}
}

// TestPerformDelete_PartialFailureKeepsTheFilesItCouldNotMove: one bad file
// must not cost the user the rest of the batch, and the toast has to say so
// rather than claiming a clean run.
func TestPerformDelete_PartialFailureKeepsTheFilesItCouldNotMove(t *testing.T) {
	host := &fakeHost{files: tempFiles(t, "good.jpg", "bad.jpg", "alsogood.jpg")}
	c := newConfirmer(t, host)

	badPath := host.files[1].Path()
	stubTrashMoveExcept(t, badPath)

	c.RequestFiles(targetsFor(host, 0, 1, 2))
	c.setSelection(true)
	c.confirmSelection()
	c.Settle()

	if want := []int{0, 2}; !slices.Equal(host.removedBatch, want) {
		t.Errorf("RemoveFiles call = %v, want %v - only what actually moved", host.removedBatch, want)
	}
	if _, err := os.Stat(badPath); err != nil {
		t.Errorf("the file that failed to move should still be on disk: %v", err)
	}
	if len(host.toasts) != 1 {
		t.Fatalf("toasts = %v, want exactly one", host.toasts)
	}
	if !strings.Contains(host.toasts[0], "2") || !strings.Contains(host.toasts[0], "3") {
		t.Errorf("toast = %q, want it to report 2 of 3", host.toasts[0])
	}
}

// TestPerformDelete_TotalFailureRemovesNothing: every move failed, so the
// app's file set must be left exactly as it was.
func TestPerformDelete_TotalFailureRemovesNothing(t *testing.T) {
	host := &fakeHost{files: tempFiles(t, "a.jpg", "b.jpg")}
	c := newConfirmer(t, host)
	stubTrashMoveExcept(t, host.files[0].Path(), host.files[1].Path())

	c.RequestFiles(targetsFor(host, 0, 1))
	c.setSelection(true)
	c.confirmSelection()
	c.Settle()

	if host.removeCalls != 0 {
		t.Errorf("RemoveFiles was called %d times, want 0 when nothing moved", host.removeCalls)
	}
	if len(host.toasts) != 1 || !strings.Contains(host.toasts[0], "could not") {
		t.Errorf("toasts = %v, want one reporting the failure", host.toasts)
	}
}

func TestPerformDelete_BatchReconcilesAfterGenerationChanges(t *testing.T) {
	host := &fakeHost{files: tempFiles(t, "a.jpg", "b.jpg"), gen: 1}
	c := newConfirmer(t, host)
	orig := trash.Move
	t.Cleanup(func() { trash.Move = orig })
	trash.Move = func(path string) error {
		host.gen++
		return os.Remove(path)
	}
	c.RequestFiles(targetsFor(host, 0, 1))
	c.setSelection(true)
	c.confirmSelection()
	c.Settle()
	if host.removeCalls != 1 || len(host.files) != 0 || len(host.emptied) != 1 {
		t.Errorf("batch was not reconciled: calls=%d files=%v empty=%v", host.removeCalls, host.files, host.emptied)
	}
}

// TestHandleKey_EscapeCancelsABatch: the batch prompt is the same card, so
// backing out of it works the same way.
func TestHandleKey_EscapeCancelsABatch(t *testing.T) {
	host := &fakeHost{files: tempFiles(t, "a.jpg", "b.jpg")}
	c := newConfirmer(t, host)

	c.RequestFiles(targetsFor(host, 0, 1))
	c.HandleKey(&fyne.KeyEvent{Name: fyne.KeyEscape})

	if c.Visible() {
		t.Error("Escape should dismiss the batch prompt")
	}
	if host.removeCalls != 0 {
		t.Error("Escape must not delete anything")
	}
	if _, err := os.Stat(host.files[0].Path()); err != nil {
		t.Errorf("the files should still exist on disk: %v", err)
	}
}

// stubTrashMove above makes every move succeed; this one fails for the named
// paths and removes the rest, so a partial failure can be driven exactly.
func stubTrashMoveExcept(t *testing.T, failing ...string) {
	t.Helper()

	orig := trash.Move
	t.Cleanup(func() { trash.Move = orig })

	trash.Move = func(path string) error {
		if slices.Contains(failing, path) {
			return errors.New("permission denied")
		}

		return os.Remove(path)
	}
}

func TestPerformDelete_ReorderBeforeConfirmationKeepsTargetIdentity(t *testing.T) {
	stubTrashMove(t)
	files := tempFiles(t, "a.jpg", "b.jpg")
	host := &fakeHost{files: slices.Clone(files)}
	c := newConfirmer(t, host)
	c.RequestFiles(targetsFor(host, 0))
	host.files = []fyne.URI{files[1], files[0]}
	host.gen++
	c.setSelection(true)
	c.confirmSelection()
	c.Settle()
	assertDeletionIdentity(t, host, files[0], files[1])
}

func TestRequestFiles_CopiesCallerTargets(t *testing.T) {
	stubTrashMove(t)
	files := tempFiles(t, "a.jpg", "b.jpg")
	host := &fakeHost{files: slices.Clone(files)}
	c := newConfirmer(t, host)
	targets := targetsFor(host, 0)
	c.RequestFiles(targets)
	targets[0] = targetsFor(host, 1)[0]
	c.setSelection(true)
	c.confirmSelection()
	c.Settle()
	assertDeletionIdentity(t, host, files[0], files[1])
}

func TestPerformDelete_ReorderDuringMoveKeepsTargetIdentity(t *testing.T) {
	files := tempFiles(t, "a.jpg", "b.jpg")
	host := &fakeHost{files: slices.Clone(files)}
	c := newConfirmer(t, host)
	entered, release := make(chan struct{}), make(chan struct{})
	releaseWorker := sync.OnceFunc(func() { close(release) })
	uitest.StubTrashMove(t, func(path string) error {
		close(entered)
		<-release
		return os.Remove(path)
	})
	t.Cleanup(func() { releaseWorker(); c.Settle() })
	c.RequestFiles(targetsFor(host, 0))
	c.setSelection(true)
	c.confirmSelection()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("trash worker did not start")
	}
	host.files = []fyne.URI{files[1], files[0]}
	host.gen++
	releaseWorker()
	c.Settle()
	assertDeletionIdentity(t, host, files[0], files[1])
}

func TestRequestFiles_RepeatedURIMovesOnceAndRemovesEveryOccurrence(t *testing.T) {
	files := tempFiles(t, "a.jpg", "b.jpg")
	host := &fakeHost{files: []fyne.URI{files[0], files[0], files[1]}}
	c := newConfirmer(t, host)
	calls := 0
	uitest.StubTrashMove(t, func(path string) error {
		calls++
		return os.Remove(path)
	})
	c.RequestFiles(targetsFor(host, 0, 1))
	c.setSelection(true)
	c.confirmSelection()
	c.Settle()
	assertDeletionIdentity(t, host, files[0], files[1])
	if calls != 1 {
		t.Errorf("trash moves = %d, want one for the repeated URI", calls)
	}
}

func assertDeletionIdentity(t *testing.T, host *fakeHost, gone, kept fyne.URI) {
	t.Helper()
	if _, err := os.Stat(gone.Path()); !os.IsNotExist(err) {
		t.Errorf("confirmed file %s still exists: %v", gone.Name(), err)
	}
	if _, err := os.Stat(kept.Path()); err != nil {
		t.Errorf("unconfirmed file %s did not survive: %v", kept.Name(), err)
	}
	if len(host.files) != 1 || host.files[0].String() != kept.String() {
		t.Errorf("current files = %v, want only %s", host.files, kept.Name())
	}
}

func TestSettle_DrainsQueuedTrashCompletions(t *testing.T) {
	stubTrashMove(t)
	files := tempFiles(t, "a.jpg", "b.jpg")
	host := &fakeHost{files: slices.Clone(files)}
	c := newConfirmer(t, host)
	c.SetUIQueue(&uitest.UIQueue{})
	c.RequestFiles(targetsFor(host, 0))
	c.setSelection(true)
	c.confirmSelection()
	c.pending.Wait()
	if len(host.files) != 2 {
		t.Fatal("worker changed the host before its UI completion was drained")
	}
	c.Settle()
	assertDeletionIdentity(t, host, files[0], files[1])
}

func TestPerformDelete_ReplacementSetKeepsUnrelatedFiles(t *testing.T) {
	stubTrashMove(t)
	files := tempFiles(t, "a.jpg", "b.jpg", "c.jpg")
	host := &fakeHost{files: slices.Clone(files[:2])}
	c := newConfirmer(t, host)
	c.RequestFiles(targetsFor(host, 0))
	c.setSelection(true)
	c.confirmSelection()
	c.pending.Wait()
	host.files = slices.Clone(files[1:])
	host.gen++
	c.Settle()
	if !slices.Equal(host.files, files[1:]) || len(host.shown) != 0 || len(host.emptied) != 0 {
		t.Errorf("replacement set changed: files=%v shown=%v emptied=%v", host.files, host.shown, host.emptied)
	}
	for _, uri := range host.files {
		if _, err := os.Stat(uri.Path()); err != nil {
			t.Errorf("replacement file did not survive: %v", err)
		}
	}
}

type deletionCompletionQueue struct {
	uitest.UIQueue
	queued chan struct{}
}

func (q *deletionCompletionQueue) Do(f func()) {
	q.UIQueue.Do(f)
	q.queued <- struct{}{}
}

func TestPerformDelete_OverlappingConfirmationsOwnTheirResults(t *testing.T) {
	files := tempFiles(t, "a.jpg", "b.jpg", "c.jpg")
	host := &fakeHost{files: slices.Clone(files)}
	c := newConfirmer(t, host)
	queue := &deletionCompletionQueue{queued: make(chan struct{}, 2)}
	c.SetUIQueue(queue)
	entered, release := make(chan struct{}), make(chan struct{})
	releaseFirst := sync.OnceFunc(func() { close(release) })
	uitest.StubTrashMove(t, func(path string) error {
		if path == files[0].Path() {
			close(entered)
			<-release
		}
		return os.Remove(path)
	})
	t.Cleanup(func() { releaseFirst(); c.Settle() })
	c.RequestFiles(targetsFor(host, 0))
	c.setSelection(true)
	c.confirmSelection()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("first trash worker did not start")
	}
	c.RequestFiles(targetsFor(host, 1))
	c.setSelection(true)
	c.confirmSelection()
	select {
	case <-queue.queued:
	case <-time.After(5 * time.Second):
		t.Fatal("second confirmation did not complete independently")
	}
	queue.Drain()
	if !slices.Equal(host.files, []fyne.URI{files[0], files[2]}) {
		t.Fatalf("second confirmation removed the wrong file: %v", host.files)
	}
	host.files = []fyne.URI{files[2], files[0]}
	host.gen++
	releaseFirst()
	c.Settle()
	assertDeletionIdentity(t, host, files[0], files[2])
	if _, err := os.Stat(files[1].Path()); !os.IsNotExist(err) {
		t.Errorf("second confirmed file survived: %v", err)
	}
}

func TestClose_StopsUnstartedBatchMoves(t *testing.T) {
	files := tempFiles(t, "a.jpg", "b.jpg")
	host := &fakeHost{files: slices.Clone(files)}
	c := newConfirmer(t, host)
	entered, release := make(chan struct{}), make(chan struct{})
	releaseFirst := sync.OnceFunc(func() { close(release) })
	uitest.StubTrashMove(t, func(path string) error {
		if path == files[0].Path() {
			close(entered)
			<-release
		}
		return os.Remove(path)
	})
	t.Cleanup(func() { releaseFirst(); c.Settle() })
	c.RequestFiles(targetsFor(host, 0, 1))
	c.setSelection(true)
	c.confirmSelection()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("trash worker did not start")
	}
	c.Close()
	releaseFirst()
	c.Settle()
	if _, err := os.Stat(files[0].Path()); !os.IsNotExist(err) {
		t.Errorf("submitted OS move did not finish: %v", err)
	}
	if _, err := os.Stat(files[1].Path()); err != nil {
		t.Errorf("unstarted move ran after Close: %v", err)
	}
	if host.removeCalls != 0 || len(host.toasts) != 0 || len(host.emptied) != 0 {
		t.Error("closed feature published a result")
	}
}

func TestPerformDelete_PartialFailureAfterReorderKeepsFailedIdentity(t *testing.T) {
	files := tempFiles(t, "a.jpg", "b.jpg", "c.jpg")
	host := &fakeHost{files: slices.Clone(files)}
	c := newConfirmer(t, host)
	stubTrashMoveExcept(t, files[1].Path())
	c.RequestFiles(targetsFor(host, 0, 1, 2))
	c.setSelection(true)
	c.confirmSelection()
	c.pending.Wait()
	host.files = []fyne.URI{files[2], files[0], files[1]}
	host.gen++
	c.Settle()
	assertDeletionIdentity(t, host, files[0], files[1])
	if _, err := os.Stat(files[2].Path()); !os.IsNotExist(err) {
		t.Errorf("second successful move did not finish: %v", err)
	}
	if len(host.toasts) != 1 || !strings.Contains(host.toasts[0], "2 of 3") {
		t.Errorf("partial failure was not reported once: %v", host.toasts)
	}
}
