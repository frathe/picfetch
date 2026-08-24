# Grid `-race` Failures: UI-Queue Seam Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Parent protocol (this session):** Do **not** start until Florian says to start the subagents. Dispatch **one implementer at a time**, in task order. After each task the **parent reads the diff, re-runs that task's verification command, and fixes drift** before dispatching the next. Do **not** `git commit` (`AGENTS.md`); the parent suggests one message at the very end.

**Goal:** Make `go test -race ./internal/ui/grid/` pass without giving up coverage of the grid-open, hashes-still-pending flow, by making the grid marshal background completions through a per-instance queue that its tests drain on the test goroutine.

**Architecture:** Fyne's test driver is not a marshaling point — `test.(*driver).DoFromGoroutine(f, _)` is a bare `f()` — so every `fyne.Do` body in the grid runs *on the decode-pool worker*, concurrently with the test goroutine. `Overview` therefore gains one field, `ui uiQueue`, a two-method unexported interface (`Do(func())`, `Drain() bool`). The app's implementation forwards to `fyne.Do` and drains nothing, so production behaviour is byte-for-byte what it is today; the grid's own test harness installs `uitest.UIQueue`, which defers callbacks and runs them, serialized, from `Settle()`. On top of that, a `parkDecodes` test helper fills the decode pool so a test can open a cold grid with no decode running underneath, which is what makes a "hashes still pending" window deterministic rather than a coin flip against four workers.

**Tech Stack:** Go 1.26.7 (toolchain 1.27.0), Fyne v2.8.0 test driver, `go test -race`, packages `internal/uitest` and `internal/ui/grid`.

## Why both halves are needed

Neither half is sufficient alone, and a reviewer who thinks one is redundant should read this before cutting it:

- **Queue without parking → flaky.** An unwarmed `Toggle()` spawns a decode per visible cell, and each finishing worker calls `rememberHash`. Whether all three files are hashed by the time `SetBrowsingDuplicates` runs decides whether `hashRemaining` returns 0 or 3 — that is, whether the "images are currently being analyzed" toast appears at all. Task 5's `-count=5` run is what would catch this.
- **Parking without the queue → still racy.** After `unpark()`, up to four workers run at once. A thumbnail completion writes `img.Image`, while the last hash job's completion runs `finishBrowse` → `applyFilter` → `wrap.Refresh()` → the cell-update callback → `requestThumbnail`, which writes `img.Image` too, plus `g.matches`, `g.groupSizes`, and `Label.SetText`. That is worker-versus-worker, and no amount of parking serializes it. The `hashJobs` last-job guard only serializes hash completions against each other.

## Evidence (parent, 2026-08-24)

`go test -race ./internal/ui/grid/` fails. Exactly three tests own every race stack:

| Test | Race stacks |
| --- | --- |
| `TestApplyFilter_BrowsePendingDoesNotCollapseGrid` | 125 |
| `TestSetDuplicateDistance_ExitsBrowseWhenGroupSplits` | 32 |
| `TestSetBrowsingDuplicates_HashesRemainingWithoutWarm` | 20 |

The race sites are **not** confined to the thumbnail paint. Distinct project-code frames, by frequency: `nav.go:41/43/44` (`setHighlight` → `wrap.Highlight`/`RefreshItem`), `grid.go:417` (`Toggle`), `grid.go:292` (the cell-update callback), `search.go:129/133/135` (`applyFilter` writing `g.matches`, refreshing the wrap), `dupes.go:377/380/392` (the hash-completion `fyne.Do`), `dupes.go:189` (`finishBrowse`), `thumbs.go:157/237/239` (the cache-hit paint versus the worker's completion paint). Raced objects include `canvas.Image.Image`, `widget.Label` internals, `gridWrapLayout` internals, `g.matches`, and `g.groupSizes`.

`go test -race ./internal/ui/` passes today (253 s) and must keep passing — it builds a real `Overview` through `features.go`, which will use the app's `fyneQueue`, so nothing about it changes.

Everything a worker touches *outside* a `fyne.Do` body is already synchronized: `imaging.ByteCache` guards every method with `c.mu`, hashes sit behind `hashMu`, `cellIDs`/`hashing` are `sync.Map`, `filterGen`/`hashJobs` are atomics, and `decodepool` is a semaphore plus `sync.Map` plus `WaitGroup`. The `fyne.Do` bodies are the entire unsynchronized surface, which is why moving that one boundary fixes all of it.

## Global Constraints

- Do **not** `git commit`. `AGENTS.md` forbids it; the parent suggests one message after Task 5.
- Do **not** add `TODO`/`FIXME` comments to source. Open work goes in `todos.md`.
- Do **not** add mutable **package-level** test seams. `Overview.ui` is a per-instance field, which is what `AGENTS.md` permits ("Runtime/test-configurable values belong on `viewer` or the owning feature").
- Do **not** change the production logic of `SetHideDuplicates`, `hashRemaining`, `finishBrowse`, `applyFilter`, `Toggle`, clustering, or dHash. The only production edits in this plan are the new `uiQueue` seam, the field, three call sites, and `Settle`'s loop.
- Leave the existing comments in `dupes.go` that mention the inline test driver **exactly as they are**. They are still accurate: `internal/ui` builds an `Overview` with the app's `fyneQueue`, so `fyne.Do` is still inline under *that* package's tests. Task 5 files a follow-up todo instead.
- Every user-visible string is `lang.L("English text")`. This work adds none, and must not change any.
- Report UI-boundary failures with `fyne.LogError`; viewer-independent packages return errors. This work adds no error paths.
- Formatting: `gofmt -l -w` on every file touched. Match CI before handoff: `gofmt`, `go vet ./...`, `go build ./...`, `go test -race ./...` from the repo root.
- Work from `/Users/ronin/Projects/picfetch`. `docs/superpowers/` is untracked and not yours to clean up.
- Golden screenshots are out of scope: nothing here changes a rendered pixel, so **do not** run `make golden` (it needs Docker) and do not touch `internal/ui/testdata/`.

## File map

| File | Change | Task |
| --- | --- | --- |
| `internal/uitest/uiqueue.go` | **Create.** `UIQueue` with `Do`/`Drain`/`Len`. | 1 |
| `internal/uitest/uiqueue_test.go` | **Create.** Four tests for the queue's own semantics. | 1 |
| `internal/ui/grid/uiqueue.go` | **Create.** Unexported `uiQueue` interface + `fyneQueue`. | 2 |
| `internal/ui/grid/grid.go` | **Modify.** One struct field (~line 168, beside `decodes`); one entry in `New`'s composite literal (~line 232). | 2 |
| `internal/ui/grid/thumbs.go` | **Modify.** `Settle` (lines 60–66) gains its loop; `fyne.Do` → `g.ui.Do` at lines 203 and 237. | 2 |
| `internal/ui/grid/dupes.go` | **Modify.** `fyne.Do` → `g.ui.Do` at line 377. Nothing else. | 2 |
| `internal/ui/grid/harness_test.go` | **Modify.** `newOverview` installs the queue; new `parkDecodes` helper; `context`/`sync` imports. | 3 |
| `internal/ui/grid/thumbs_test.go` | **Modify.** Two existing parked tests move onto `parkDecodes`. | 3 |
| `internal/ui/grid/dupes_test.go` | **Modify.** The three failing tests park before `Toggle`. | 4 |
| `ARCHITECTURE.md` | **Modify.** `internal/uitest` blurb + file table; the grid-overview entry in the "where to look for X" index. | 5 |
| `AGENTS.md` | **Modify.** One clause in "Concurrency and Fyne" so a future agent does not revert the seam. | 5 |
| `todos.md` | **Modify.** Retire the race item; file the production follow-up. | 5 |

## Reference reading for implementers

- `internal/ui/grid/thumbs.go`'s `Warm()` doc comment is the existing statement of this whole problem: "under the fyne test driver a decode's completion runs inline on the decoding goroutine".
- `internal/ui/grid/grid.go`'s `cellIDs` field comment says the same thing about why that map is a `sync.Map`.
- `internal/ui/grid/thumbs_test.go`'s `TestRequestThumbnail_RecycledBeforeDecodeBailsAndReleases` (lines 260–294) is the hand-rolled park loop `parkDecodes` generalizes.
- `internal/ui/grid/dupes_test.go`'s `TestSetHideDuplicates_HashesRemainingWithoutWarm` (~line 295) is the one pending-hash test that already avoids the race by never opening the grid.

---

### Task 1: `uitest.UIQueue`

**Files:**
- Create: `internal/uitest/uiqueue.go`
- Test: `internal/uitest/uiqueue_test.go`

**Interfaces:**
- Consumes: nothing. `sync` only; no Fyne, no viewer types (the package rule for `internal/uitest`).
- Produces: `type UIQueue struct{}` with `func (q *UIQueue) Do(f func())`, `func (q *UIQueue) Drain() bool`, `func (q *UIQueue) Len() int`. The zero value is usable, so callers write `&uitest.UIQueue{}`. Task 2's unexported `uiQueue` interface is satisfied by `*UIQueue`; Task 3 installs it.

**Model:** `claude-sonnet-5-thinking-high`, `subagent_type: generalPurpose`. Small, but it is a concurrency primitive whose locking discipline has to be exactly right (the lock must not be held across a callback).

**Reviewer (parent):** read the file, confirm `Drain` releases the lock before running callbacks, re-run the package test.

- [ ] **Step 1: Write the failing test**

Create `internal/uitest/uiqueue_test.go`:

```go
package uitest

import (
	"sync"
	"testing"
)

func TestUIQueue_DoDefersInsteadOfRunning(t *testing.T) {
	var q UIQueue
	ran := false

	q.Do(func() { ran = true })

	if ran {
		t.Fatal("Do must not run the callback on the calling goroutine - that is the whole point")
	}
	if q.Len() != 1 {
		t.Fatalf("Len() = %d, want 1", q.Len())
	}
	if !q.Drain() {
		t.Fatal("Drain() = false with one callback queued, want true")
	}
	if !ran {
		t.Error("Drain must run the queued callback")
	}
}

func TestUIQueue_DrainRunsInOrderThenReportsEmpty(t *testing.T) {
	var q UIQueue
	var order []int
	for i := range 3 {
		q.Do(func() { order = append(order, i) })
	}

	q.Drain()

	if len(order) != 3 || order[0] != 0 || order[1] != 1 || order[2] != 2 {
		t.Fatalf("order = %v, want [0 1 2]", order)
	}
	if q.Drain() {
		t.Error("Drain() = true on an empty queue, want false")
	}
}

func TestUIQueue_WorkQueuedDuringADrainWaitsForTheNext(t *testing.T) {
	var q UIQueue
	inner := false
	q.Do(func() { q.Do(func() { inner = true }) })

	if !q.Drain() {
		t.Fatal("first Drain() = false, want true - the outer callback was queued")
	}
	if inner {
		t.Fatal("a callback queued during a Drain must not run in that same Drain")
	}
	if !q.Drain() {
		t.Fatal("second Drain() = false, want true - the inner callback was queued")
	}
	if !inner {
		t.Error("the inner callback never ran")
	}
}

func TestUIQueue_ConcurrentDo(t *testing.T) {
	var q UIQueue
	var wg sync.WaitGroup
	for range 50 {
		wg.Go(func() { q.Do(func() {}) })
	}
	wg.Wait()

	if q.Len() != 50 {
		t.Fatalf("Len() = %d after 50 concurrent Do calls, want 50", q.Len())
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run:

```bash
go test -race -count=1 ./internal/uitest/
```

Expected: a build failure, `undefined: UIQueue`. If it compiles, stop and report `NEEDS_CONTEXT` — the tree does not match this plan.

- [ ] **Step 3: Write the implementation**

Create `internal/uitest/uiqueue.go`:

```go
package uitest

import "sync"

// UIQueue collects the callbacks a background goroutine hands over for the
// UI goroutine to run, and runs them on whoever calls Drain.
//
// It exists because Fyne's test driver is not a marshaling point: its
// DoFromGoroutine calls the function inline on the calling goroutine, so a
// worker's fyne.Do body runs *on the worker*, concurrently with the test
// goroutine that spawned it. Under -race that is a genuine data race on
// every widget and every unsynchronized field the body touches - the same
// code being perfectly safe in the app, where the real driver queues the
// callback onto the one UI goroutine.
//
// A feature that hands its completions to a UIQueue gets those production
// semantics back under test: the callback is deferred, and runs serialized
// on the goroutine that drains it - which for a test is the test goroutine
// itself, at a point of its own choosing.
//
// The zero value is ready to use. Do is safe from any goroutine; Drain is
// for one goroutine at a time, the one the test is running on.
type UIQueue struct {
	mu      sync.Mutex
	pending []func()
}

// Do queues f, and never runs it on the calling goroutine.
func (q *UIQueue) Do(f func()) {
	if f == nil {
		return
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	q.pending = append(q.pending, f)
}

// Drain runs everything queued so far, in the order it was queued, on the
// calling goroutine, and reports whether it ran anything.
//
// The lock is dropped before the callbacks run: a callback may queue more
// work, directly or by spawning a worker that does, and holding the lock
// across one would deadlock. Work queued during a Drain therefore lands in
// the *next* one, so a caller that needs quiescence loops until Drain
// reports false.
func (q *UIQueue) Drain() bool {
	q.mu.Lock()
	batch := q.pending
	q.pending = nil
	q.mu.Unlock()

	for _, f := range batch {
		f()
	}

	return len(batch) > 0
}

// Len is how many callbacks are waiting for a Drain - what a test asserts
// on to show that a worker deferred its paint rather than running it.
func (q *UIQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	return len(q.pending)
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run:

```bash
gofmt -l -w internal/uitest/ && go vet ./internal/uitest/ && go test -race -count=1 ./internal/uitest/
```

Expected: `gofmt` silent, `vet` silent, `ok github.com/frathe/picfetch/internal/uitest`, no `DATA RACE`.

- [ ] **Step 5: Do not commit.** Report the two file paths, the command you ran, and its output.

---

### Task 2: The `uiQueue` seam on `Overview`

**Files:**
- Create: `internal/ui/grid/uiqueue.go`
- Modify: `internal/ui/grid/grid.go` (struct field ~line 168; `New`'s literal ~line 232)
- Modify: `internal/ui/grid/thumbs.go` (`Settle` lines 60–66; `fyne.Do` at 203 and 237)
- Modify: `internal/ui/grid/dupes.go` (`fyne.Do` at line 377)

**Interfaces:**
- Consumes: nothing from Task 1. This task must compile and pass with `internal/uitest` untouched.
- Produces: unexported `type uiQueue interface { Do(f func()); Drain() bool }`, unexported `type fyneQueue struct{}` satisfying it, and the field `ui uiQueue` on `Overview`, set to `fyneQueue{}` by `New`. Task 3 assigns `g.ui = &uitest.UIQueue{}` from a test file, so `*uitest.UIQueue` must satisfy `uiQueue` exactly — method set `Do(func())` and `Drain() bool`, no more.

**Model:** `gpt-5.6-terra-medium`, `subagent_type: generalPurpose`. The code is fully prescribed below; the work is careful mechanical editing across four files plus a build/vet/test gate.

**Reviewer (parent):** confirm the non-race grid suite is unchanged (this task must be behaviour-neutral), and that `-race` still fails on exactly the same three tests — proof that nothing has silently started deferring in production.

- [ ] **Step 1: Create the seam**

Create `internal/ui/grid/uiqueue.go`:

```go
// The seam a finished background job hands its widget work across, and the
// app's own implementation of it.

package grid

import "fyne.io/fyne/v2"

// uiQueue is how a decode-pool worker gets its result onto the UI
// goroutine. fyneQueue in the app; this package's tests install
// uitest.UIQueue instead, because Fyne's test driver is not a marshaling
// point - its DoFromGoroutine runs the callback inline on the calling
// (worker) goroutine, so the completion bodies below would otherwise touch
// canvas.Image, widget.Label, g.matches and g.groupSizes concurrently with
// the test goroutine that spawned the worker.
//
// A field on Overview rather than a package-level var, for the reason
// AGENTS.md gives: it is per-instance configuration. internal/ui builds a
// real Overview under that same test driver and must keep the app's own
// behaviour, which it does because New installs fyneQueue.
type uiQueue interface {
	// Do arranges for f to run on the UI goroutine. It may return before
	// f has run, and must not run f on the calling goroutine.
	Do(f func())

	// Drain runs whatever Do deferred, on the calling goroutine, and
	// reports whether it ran anything. Always false for a queue backed by
	// a real UI goroutine: that goroutine drains itself.
	Drain() bool
}

// fyneQueue is the app's uiQueue - hand the callback to Fyne and let the
// driver marshal it onto the UI goroutine. Nothing here to drain.
type fyneQueue struct{}

func (fyneQueue) Do(f func()) { fyne.Do(f) }

func (fyneQueue) Drain() bool { return false }
```

- [ ] **Step 2: Add the field and initialize it**

In `internal/ui/grid/grid.go`, immediately after the `decodes` field (the `decodes *decodepool.Pool[*fyne.Container, int]` line, ~168) and before the `cellIDs` comment block, insert:

```go
	// ui is how a decode worker's completion reaches the UI goroutine -
	// see uiqueue.go for why that is a field and not a direct fyne.Do.
	ui uiQueue
```

In the same file, in `New`'s composite literal, add `ui` after `decodes` so the literal reads:

```go
	g := &Overview{
		host:       host,
		win:        win,
		sel:        selection.New(),
		thumbs:     imaging.NewThumbCache(imaging.DefaultThumbCacheBytes),
		decodes:    decodepool.New[*fyne.Container, int](thumbConcurrency),
		ui:         fyneQueue{},
		hashes:     make(map[string]uint64),
		hashFailed: make(map[string]struct{}),
		dupeDist:   imaging.DuplicateMaxDistance,
		browseHost: -1,
	}
```

- [ ] **Step 3: Route the three completions through the seam**

In `internal/ui/grid/thumbs.go`, replace `fyne.Do(func() {` with `g.ui.Do(func() {` in **both** places — the re-request inside the pre-decode bail (line 203) and the completion paint (line 237). Leave the surrounding comments and bodies untouched. `fyne` stays imported (the file uses `fyne.URI` and `fyne.Container`).

In `internal/ui/grid/dupes.go`, replace `fyne.Do(func() {` with `g.ui.Do(func() {` at line 377, inside `hashRemaining`'s deferred last-job block. Leave the comment above it and the body untouched. `fyne` stays imported (`fyne.URI`).

- [ ] **Step 4: Make `Settle` drain**

In `internal/ui/grid/thumbs.go`, replace the whole `Settle` function (lines 60–66) with:

```go
// Settle waits for every thumbnail decode spawned so far to finish -
// including its completion, which runs before the wait returns. The app
// never needs this; tests do, to keep a decode goroutine from touching
// widgets after the test that started it has moved on.
//
// The loop is what keeps that promise for a deferring uiQueue (see
// uiqueue.go): a drained completion can spawn further decodes -
// requestThumbnail's own re-request does exactly that, and applyFilter
// refreshes the wrap, which re-runs the cell-update callback - so waiting
// once is not enough. It ends on the first pass that finds the pool empty
// and nothing left to drain, which for the app's fyneQueue is always the
// first pass, since its Drain is a constant false.
func (g *Overview) Settle() {
	for {
		g.decodes.Wait()
		if !g.ui.Drain() {
			return
		}
	}
}
```

- [ ] **Step 5: Verify the build and that production behaviour is unchanged**

Run:

```bash
gofmt -l -w internal/ui/grid/ && go vet ./... && go build ./... && go test -count=1 ./internal/ui/grid/
```

Expected: `gofmt` and `vet` silent, build clean, `ok github.com/frathe/picfetch/internal/ui/grid`. The non-race suite passes today and must still pass: `fyneQueue.Do` is `fyne.Do` and `fyneQueue.Drain` is `false`, so nothing about the package's behaviour has moved.

- [ ] **Step 6: Verify `-race` still fails on the same three tests, and only those**

Run:

```bash
go test -race -count=1 -timeout 300s ./internal/ui/grid/ 2>&1 | grep -E '^--- FAIL'
```

Expected, in some order, exactly these three and nothing else:

```
--- FAIL: TestSetBrowsingDuplicates_HashesRemainingWithoutWarm
--- FAIL: TestApplyFilter_BrowsePendingDoesNotCollapseGrid
--- FAIL: TestSetDuplicateDistance_ExitsBrowseWhenGroupSplits
```

This is the point of the step: no test has been installed on the queue yet, so nothing should have changed. If a *fourth* test now fails, or one of these now passes, stop and report `DONE_WITH_CONCERNS` with the test name and the first race stack — do not press on.

- [ ] **Step 7: Do not commit.** Report the four file paths and both commands' output.

---

### Task 3: Install the queue in the harness and add `parkDecodes`

**Files:**
- Modify: `internal/ui/grid/harness_test.go` (imports; `newOverview` at lines 80–87; new `parkDecodes` helper)
- Modify: `internal/ui/grid/thumbs_test.go` (`TestRequestThumbnail_RecycledBeforeDecodeBailsAndReleases` ~260–294 and `TestRequestThumbnail_QueryChangeDiscardsInFlightDecode` ~303–333)

**Interfaces:**
- Consumes: `uitest.UIQueue` (Task 1); `Overview.ui` and the looping `Settle` (Task 2); the package const `thumbConcurrency` (`thumbs.go`); `decodepool.Pool.Go`.
- Produces: `func parkDecodes(t *testing.T, g *Overview) (unpark func())`, used by Task 4. Every `*Overview` built by `newOverview` now defers its completions.

**Model:** `claude-opus-5-thinking-high`, `subagent_type: generalPurpose`. This is the one task with genuinely unknown fallout: it changes completion timing for every test in the package at once, and deciding which of the ~60 tests needs a `Settle()` it did not need before takes judgement about what each one is actually asserting.

**Reviewer (parent):** read every test the implementer touched beyond the two named above, and confirm each added `Settle()` is a real synchronization need rather than a way to paper over a changed assertion.

- [ ] **Step 1: Install the queue in `newOverview`**

In `internal/ui/grid/harness_test.go`, replace `newOverview` (lines 80–87) with:

```go
// newOverview builds an Overview over host and a real (test-driver)
// window, closing the window when the test ends - the fixture behind every
// New call in this file, now that New needs a window to maximize on open.
//
// Every overview built here defers its background completions instead of
// letting Fyne's test driver run them inline on the decode worker (see
// uitest.UIQueue and uiqueue.go). Settle is what runs them, on the test
// goroutine, so a test that asserts on the effect of a decode - a painted
// cell, a rebuilt filter, a group that has finished hashing - has to
// Settle first.
func newOverview(t *testing.T, host Host) *Overview {
	t.Helper()

	win := test.NewWindow(nil)
	t.Cleanup(win.Close)

	g := New(host, win)
	g.ui = &uitest.UIQueue{}

	return g
}
```

`uitest` is already imported in this file.

- [ ] **Step 2: Add `parkDecodes`**

In the same file, add below `newOverview`:

```go
// parkDecodes fills the decode pool with jobs that block until the
// returned unpark runs, so a test can drive the grid - including opening it
// on a cold cache - with nothing actually decoding underneath. That is what
// makes a "hashes are still pending" window deterministic instead of a race
// against four workers that may or may not have hashed everything already.
//
// Unparks and Settles on cleanup, so a test that Fatals mid-window cannot
// leave a parked goroutine behind for the next one. Registered after
// newOverview's window close, and cleanups run last-registered-first, so
// the drain still happens while the widgets are alive. unpark is a OnceFunc:
// calling it yourself, which is the normal shape, is fine.
func parkDecodes(t *testing.T, g *Overview) (unpark func()) {
	t.Helper()

	// The pool waits for its slot on the spawned goroutine, so each parker
	// has to report that it really holds one before the caller can be sure
	// its own requests queue behind them.
	holding := make(chan struct{}, thumbConcurrency)
	parked := make(chan struct{})
	unpark = sync.OnceFunc(func() { close(parked) })

	for range thumbConcurrency {
		g.decodes.Go(context.Background(), func(bool) {
			holding <- struct{}{}
			<-parked
		})
	}
	for range thumbConcurrency {
		<-holding
	}

	t.Cleanup(func() {
		unpark()
		g.Settle()
	})

	return unpark
}
```

Add `"context"` and `"sync"` to the file's stdlib import group, so the block reads:

```go
import (
	"context"
	"image/color"
	"os"
	"sync"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/uitest"
)
```

- [ ] **Step 3: Move the two existing parked tests onto the helper**

In `internal/ui/grid/thumbs_test.go`, in `TestRequestThumbnail_RecycledBeforeDecodeBailsAndReleases`, replace the hand-rolled park block (the `holding`/`parked` declarations, the two `for range thumbConcurrency` loops) with a single call, and replace `close(parked)` with `unpark()`. The body becomes:

```go
func TestRequestThumbnail_RecycledBeforeDecodeBailsAndReleases(t *testing.T) {
	host := hostWith(t, "a.jpg", "b.jpg")
	g := newOverview(t, host)

	cell, img := newCell()
	g.cellIDs.Store(cell, 0)

	unpark := parkDecodes(t, g)

	g.requestThumbnail(cell, img, 0, host.gen)
	g.cellIDs.Store(cell, 1) // the cell scrolls on before a worker picks this up

	unpark()
	g.Settle()

	if img.Image != nil {
		t.Error("a decode whose cell scrolled away must not paint it")
	}
	if !g.decodes.Claim(cell, 0) {
		t.Error("the bailed decode should have released its claim")
	}
}
```

Keep the existing doc comment above the function, and drop only its now-inaccurate last sentence about filling the pool "from the test" if it no longer reads true; `parkDecodes` documents the technique.

Do the same to `TestRequestThumbnail_QueryChangeDiscardsInFlightDecode`:

```go
func TestRequestThumbnail_QueryChangeDiscardsInFlightDecode(t *testing.T) {
	host := hostWith(t, "a.jpg", "b.jpg")
	g := newOverview(t, host)

	cell, img := newCell()
	g.cellIDs.Store(cell, 0)

	unpark := parkDecodes(t, g)

	g.requestThumbnail(cell, img, 0, host.gen)

	// Display cell 0 now means b.jpg; the decode in flight is for a.jpg.
	typeQuery(g, "b")

	unpark()
	g.Settle()

	if img.Image != nil {
		t.Error("a decode started under a different query must not paint a.jpg into a cell now showing b.jpg")
	}
}
```

Those two park blocks are the only users of `context` in `thumbs_test.go` (lines 273 and 313 today), so remove `"context"` from that file's imports once both are gone. The build will tell you if you are wrong.

- [ ] **Step 4: Run the package and fix the fallout**

Run:

```bash
gofmt -l -w internal/ui/grid/ && go test -count=1 -timeout 300s ./internal/ui/grid/
```

Expected: the three known-failing tests may now fail on assertions rather than races, and **other** tests may fail for a new reason — a completion that used to run inline on the worker now waits for a `Settle()` the test never makes.

How to fix each failure, in order of preference:

1. If the test spawns background work (it opens the grid without `Warm()`, or calls `requestThumbnail`/`SetHideDuplicates`/`SetBrowsingDuplicates` on unhashed files) and then asserts on the result, add `g.Settle()` before the assertions. That is the correct fix and the one this design expects.
2. If the test's point is the *pending* state, park with `parkDecodes` so the window is deterministic, assert, then `unpark(); g.Settle()` and assert the settled state.
3. Do **not** weaken an assertion, delete a test, or add a sleep. Do **not** touch production files — Task 2 owns those, and they are already reviewed.

Leave the three tests named in Task 4 alone even if you can see how to fix them; Task 4 owns them and prescribes their bodies.

- [ ] **Step 5: Verify — everything green except the three Task 4 tests**

Run:

```bash
go test -count=1 -timeout 300s ./internal/ui/grid/ 2>&1 | grep -E '^--- FAIL' ; go test -race -count=1 -timeout 300s ./internal/ui/grid/ 2>&1 | grep -E '^--- FAIL'
```

Expected: the only names in either list are drawn from `TestSetBrowsingDuplicates_HashesRemainingWithoutWarm`, `TestApplyFilter_BrowsePendingDoesNotCollapseGrid`, and `TestSetDuplicateDistance_ExitsBrowseWhenGroupSplits`. Any other name is fallout you still have to fix.

If a run **hangs**, do not raise the timeout and call it fixed. A hang means either a parked test never unparked, or `Settle`'s loop is not converging because a drained callback keeps queueing more. Report `BLOCKED` with the `-timeout` goroutine dump.

- [ ] **Step 6: Do not commit.** Report every file you touched, every test you added a `Settle()` to and why, and both commands' output.

---

### Task 4: Park the three failing tests before they open the grid

**Files:**
- Modify: `internal/ui/grid/dupes_test.go` — three functions only: `TestSetBrowsingDuplicates_HashesRemainingWithoutWarm` (~537–561), `TestApplyFilter_BrowsePendingDoesNotCollapseGrid` (~563–597), `TestSetDuplicateDistance_ExitsBrowseWhenGroupSplits` (~599–615)

**Interfaces:**
- Consumes: `parkDecodes` and the queue-installing `newOverview` (Task 3); `hostPatterned` and `nearGrayPair` (same file, ~lines 61 and 92); `hostWith` (`harness_test.go`); the public `SetBrowsingDuplicates`/`SetHideDuplicates`/`SetDuplicateDistance`/`BrowsingDuplicates`/`Settle`, and unexported `count`/`rememberHash`.
- Produces: nothing later tasks consume.

**Model:** `claude-sonnet-5-thinking-high`, `subagent_type: generalPurpose`. The bodies are prescribed, but Step 5's repeat run is a flake hunt that needs someone who will read a stack rather than re-run until green. Escalate to `claude-opus-5-thinking-high` if a rewrite is still racy or flaky after two honest attempts.

**Reviewer (parent):** re-run `go test -race -count=5` yourself. Check that all three tests still open the grid — if any of them stopped calling `Toggle()`, the coverage this whole plan exists to keep has been thrown away and the task must be redone.

- [ ] **Step 1: Record the starting point**

Run:

```bash
go test -race -count=1 -timeout 120s -run 'TestSetBrowsingDuplicates_HashesRemainingWithoutWarm|TestApplyFilter_BrowsePendingDoesNotCollapseGrid|TestSetDuplicateDistance_ExitsBrowseWhenGroupSplits' ./internal/ui/grid/
```

Expected, as of Task 3 landing: `ok`. These three passed under `-race -count=200` once the queue was installed, so **this task is not red-to-green** — Task 3 already removed the *data* race that was failing them. What is left is the *logical* race this plan predicted in "Why both halves are needed", and it is what this task removes.

The mechanism, which is worth understanding before you touch the code: a decode worker calls `g.thumbs.Add` and `g.rememberHash` on the worker itself, **outside** the `g.ui.Do` completion. Deferring the completion therefore does nothing to stop a worker from having already hashed a file by the time the test's next line runs. So whether `hashRemaining` finds three files pending or zero still depends on decode timing, and each of the three tests reads that differently:

- `HashesRemainingWithoutWarm` asserts the "images are currently being analyzed" toast, which only appears when at least one file is still pending.
- `BrowsePendingDoesNotCollapseGrid` asserts `count() == 3`, which holds only while `groupSize(browseHost)` is still below two — that is, while the hashes have not landed.
- `ExitsBrowseWhenGroupSplits` injects a near-gray hash pair into files that are solid white on disk, so a real decode landing would overwrite both with `dHash(white) == 0` and make the pair inseparable at any distance.

All three currently win that race because a JPEG decode takes far longer than the handful of instructions between `Toggle()` and the assertion. None of them is *guaranteed* to win it, and a loaded CI box is exactly where that stops being true. Parking makes all three outcomes structural instead of timing-dependent.

If any of the three *fails* on this run, say so in your report — that would mean the machine is already losing the race, which is useful evidence, and the rewrite below is still the fix.

- [ ] **Step 2: Rewrite `TestSetBrowsingDuplicates_HashesRemainingWithoutWarm`**

```go
func TestSetBrowsingDuplicates_HashesRemainingWithoutWarm(t *testing.T) {
	host := hostPatterned(t,
		[]string{"sunset-a.jpg", "sunset-b.jpg", "moon.jpg"},
		[]int{1, 1, 99},
	)
	g := newOverview(t, host)
	// Parked before opening: an unwarmed Toggle spawns a decode per visible
	// cell, and each one that finishes remembers a hash. Whether they beat
	// SetBrowsingDuplicates below is what decides whether there is anything
	// left for it to hash - that is, whether the toast appears at all.
	// Parked, there provably is.
	unpark := parkDecodes(t, g)
	host.index = 0
	g.Toggle()

	g.SetBrowsingDuplicates(true)
	if len(host.toasts) != 1 || host.toasts[0] != lang.L("The images are currently being analyzed") {
		t.Fatalf("toasts = %v, want [%q] while hashing", host.toasts, lang.L("The images are currently being analyzed"))
	}

	unpark()
	g.Settle()

	if !g.BrowsingDuplicates() {
		t.Fatal("browse should turn on after remaining files hash")
	}
	if g.count() != 2 {
		t.Fatalf("count() = %d after hashing remaining, want 2", g.count())
	}
	if len(host.toasts) != 1 {
		t.Errorf("toasts after Settle = %v, want still one (no second toast)", host.toasts)
	}
}
```

Why this holds: with all four pool slots parked, `Toggle`'s decodes queue without running, so no thumbnail is cached and no hash recorded. `hashRemaining` therefore queues three jobs, returns 3, and `SetBrowsingDuplicates` shows the toast without calling `finishBrowse`. `ShowToast` runs on the test goroutine inside the setter, so asserting on it before `Settle` is safe. After `unpark()`, `Settle` waits the pool out and then drains the deferred completions here, in order: the last hash job's `finishBrowse` groups the two matching seeds and filters the grid to them, so `count()` is 2.

- [ ] **Step 3: Rewrite `TestApplyFilter_BrowsePendingDoesNotCollapseGrid`**

```go
func TestApplyFilter_BrowsePendingDoesNotCollapseGrid(t *testing.T) {
	host := hostPatterned(t,
		[]string{"sunset-a.jpg", "sunset-b.jpg", "moon.jpg"},
		[]int{1, 1, 99},
	)
	g := newOverview(t, host)
	unpark := parkDecodes(t, g)
	host.index = 0
	g.Toggle()

	g.SetBrowsingDuplicates(true)
	g.SetHideDuplicates(true)

	if g.count() != 3 {
		t.Fatalf("count() = %d while hashes pending, want 3 (not collapsed to the source cell)", g.count())
	}
	if !g.BrowsingDuplicates() {
		t.Fatal("BrowsingDuplicates() = false while hashes pending, want true")
	}

	unpark()
	g.Settle()

	if !g.BrowsingDuplicates() {
		t.Fatal("browse should stay on after remaining files hash")
	}
	if g.count() != 2 {
		t.Fatalf("count() = %d after hashing remaining, want 2", g.count())
	}
}
```

Why this holds, and what it is actually guarding: `SetBrowsingDuplicates` queues three hash jobs and records their URIs in `g.hashing`. `SetHideDuplicates` then calls `hashRemaining` again, which skips all three as already in flight, returns 0, and so *does* run `applyFilter` on the test goroutine. That is the interesting state — `browseFilter` is false while `groupSize(browseHost)` is still below two, and `hide` is false because browse wins, so `matches` stays nil and the grid shows all three cells instead of collapsing to the source. Parking is what makes that window exist at all rather than depend on decode timing.

- [ ] **Step 4: Rewrite `TestSetDuplicateDistance_ExitsBrowseWhenGroupSplits`**

```go
func TestSetDuplicateDistance_ExitsBrowseWhenGroupSplits(t *testing.T) {
	host := hostWith(t, "a.jpg", "b.jpg")
	g := newOverview(t, host)
	a, b := nearGrayPair()
	g.rememberHash(host.files[0], a)
	g.rememberHash(host.files[1], b)
	// Parked before opening, and left parked: hostWith's JPEGs are solid
	// white, so a decode that landed would rememberHash 0 over the
	// near-gray pair injected above, leaving the two files exact duplicates
	// at every distance - the split this test is named for could not
	// happen. parkDecodes unparks and Settles on cleanup.
	parkDecodes(t, g)
	g.Toggle()

	g.SetBrowsingDuplicates(true)
	if g.count() != 2 {
		t.Fatalf("setup count() = %d, want 2", g.count())
	}

	g.SetDuplicateDistance(0)
	if g.BrowsingDuplicates() {
		t.Fatal("distance 0 should exit browse when the pair splits")
	}
}
```

Note the bare `parkDecodes(t, g)` — this is the one test that never unparks itself, so binding the return value would not compile. Both hashes are already present, so `hashRemaining` starts nothing, returns 0, and `finishBrowse` runs on the test goroutine: the near-gray pair is within the default distance, so browse turns on over two cells. `SetDuplicateDistance(0)` re-runs `finishBrowse`; the pair differs by at least one bit, so the group drops below two members and browse clears itself.

- [ ] **Step 5: Verify, repeatedly**

Run:

```bash
gofmt -l -w internal/ui/grid/dupes_test.go && go test -race -count=5 -timeout 600s ./internal/ui/grid/
```

Expected: `ok`, no `DATA RACE`, across all five runs.

Because these three already passed before the rewrite, a green run here is necessary but not sufficient — it does not by itself show the rewrite achieved anything. What shows that is *which* assertions now hold by construction, and each rewritten test carries one:

- `HashesRemainingWithoutWarm`'s toast assertion can only pass if `hashRemaining` found work pending, which parking guarantees.
- `BrowsePendingDoesNotCollapseGrid`'s `count() == 3` can only pass while no hash has landed, which parking guarantees.
- `ExitsBrowseWhenGroupSplits`'s setup `count() == 2` can only pass if the injected near-gray hashes survived, which parking guarantees.

Say in your report which of those three you take as the evidence that determinism was actually achieved.

If a rewritten test fails intermittently, find the nondeterminism — some background work is still landing at a time the test does not control — and report it. Do not add a sleep, do not loop until green, and do not remove the `Toggle()`.

- [ ] **Step 6: Do not commit.** Report the diff of the three functions and the full `-count=5` output.

---

### Task 5: Full-suite gate and documentation

**Files:**
- Modify: `ARCHITECTURE.md` (the `internal/uitest` section, ~lines 547–557; the grid-overview entry in the "where to look for X" index, ~lines 717–719)
- Modify: `AGENTS.md` ("Concurrency and Fyne" section)
- Modify: `todos.md` (retire the race item; file the follow-up)

**Interfaces:**
- Consumes: everything Tasks 1–4 landed.
- Produces: documentation only.

**Model:** `claude-sonnet-5-thinking-high`, `subagent_type: generalPurpose`. `ARCHITECTURE.md` is written in a dense, specific, reason-giving voice and a cheaper model will pad it with filler that has to be rewritten. The long verification runs also need someone who will read a failure rather than retry it.

**Reviewer (parent):** re-run the root `-race` suite yourself before declaring the work done, and read the doc diff as prose, not as a checklist.

- [ ] **Step 1: Gate the whole module**

Run, in order:

```bash
gofmt -l .
go vet ./...
go build ./...
go test -race -count=1 -timeout 1800s ./...
```

Expected: `gofmt -l` prints nothing, `vet` and `build` are silent, every package is `ok`. Budget roughly 10 minutes for the last one — `internal/ui` alone takes about 250 seconds under `-race`.

If any package outside `internal/ui/grid` and `internal/uitest` fails, stop and report `BLOCKED` with the package, test name, and first stack. Do not edit another feature package to make this green.

- [ ] **Step 2: Document the queue in `ARCHITECTURE.md`**

Two edits in the `### internal/uitest` section. First, extend the opening paragraph so the package's remit covers the queue — replace:

```markdown
Test fixtures shared across the module's test suites: synthetic images in every format the viewer reads, the temp files
and URIs to hand them over by, and swap-in stubs for the OS-level seams. Imported only from `_test.go`
files, so it never reaches a production binary. Zero dependency on
`viewer`.
```

with:

```markdown
Test fixtures shared across the module's test suites: synthetic images in every format the viewer reads, the temp files
and URIs to hand them over by, swap-in stubs for the OS-level seams, and the UI queue a feature hands its background
completions to so they do not run on the worker that produced them. Imported only from `_test.go`
files, so it never reaches a production binary. Zero dependency on
`viewer`.
```

Second, add a row to that section's file table, after the `stubs.go` row:

```markdown
| `uiqueue.go` | `UIQueue` — `Do` defers a callback, `Drain` runs everything deferred so far on the calling goroutine and reports whether it ran anything. Fyne's test driver is not a marshaling point: `test.(*driver).DoFromGoroutine` calls the function inline on the caller, so a worker's `fyne.Do` body runs *on the worker*, racing the test goroutine on widgets and on whatever unsynchronized state it touches. A feature that marshals through a queue instead gets the app's semantics back under test — deferred, then serialized on the goroutine that drains it. `internal/ui/grid`'s `newOverview` installs one on every `Overview` it builds and `Overview.Settle` drains it; work queued *during* a drain lands in the next one, which is why `Settle` loops |
```

- [ ] **Step 3: Point the index at the seam**

In the "where to look for X" index, replace the grid-overview entry:

```markdown
- "How does the grid overview / thumbnail generation work?" → `internal/imaging/thumbnail.go` (decode + downsample) +
  `internal/ui/grid/grid.go` (`widget.GridWrap` wiring) + `internal/ui/grid/thumbs.go` (bounded-concurrency requests,
  generation/cell-recycling guards)
```

with:

```markdown
- "How does the grid overview / thumbnail generation work?" → `internal/imaging/thumbnail.go` (decode + downsample) +
  `internal/ui/grid/grid.go` (`widget.GridWrap` wiring) + `internal/ui/grid/thumbs.go` (bounded-concurrency requests,
  generation/cell-recycling guards) + `internal/ui/grid/uiqueue.go` (the `uiQueue` a finished decode hands its widget
  work across — `fyneQueue` in the app, `uitest.UIQueue` under the grid's own tests, which is what makes the package
  `-race` clean while its tests still open the grid on a cold cache)
```

- [ ] **Step 4: Amend `AGENTS.md` so the seam does not get reverted**

In the "Concurrency and Fyne" section, replace this bullet:

```markdown
- Scan, load, sort, and vector work each own a `requestLifecycle`; capture its token, check staleness before expensive work and before applying results, and marshal background UI updates through `fyne.Do`.
```

with:

```markdown
- Scan, load, sort, and vector work each own a `requestLifecycle`; capture its token, check staleness before expensive work and before applying results, and marshal background UI updates through `fyne.Do` — except in `internal/ui/grid`, which marshals through its per-instance `uiQueue` (`g.ui.Do`) so its tests can drain completions on the test goroutine instead of letting the Fyne test driver run them inline on the decode worker. Do not "simplify" that back to a direct `fyne.Do`.
```

- [ ] **Step 5: Retire the todo and file the follow-up**

In `todos.md`, delete the whole `### grid tests race under go test -race when they Toggle` block from under `## TODO`, and append this under `## Done`:

```markdown
## Grid completions are marshalled through a drainable queue

`go test -race ./internal/ui/grid/` was failing on three tests, and the
cause was never the grid: Fyne's test driver implements `DoFromGoroutine`
as a bare `f()`, so every `fyne.Do` body ran on the decode-pool worker,
racing the test goroutine on `canvas.Image`, `widget.Label`, `g.matches`
and `g.groupSizes`. `Overview` now marshals through a per-instance
`uiQueue`: `fyneQueue` (a straight `fyne.Do`) in the app, `uitest.UIQueue`
under the package's own tests, drained by the looping `Settle`. A
`parkDecodes` harness helper fills the decode pool so a test can open a
cold grid with nothing decoding under it, which is what makes a
"hashes still pending" window deterministic. All three tests still open
the grid, so the `G` → `D` → `Shift+D` flow keeps its coverage.
```

Then add this under `## TODO`:

```markdown
### grid: undo the production compromises made for the inline test driver
`SetHideDuplicates` skips its own `applyFilter` when hash jobs are pending,
and `hashRemaining` lets only the last pool job apply — both, by their own
comments, to avoid racing Fyne's test driver rather than to serve the user.
The visible cost is that `D` on a big cold folder leaves the top bar without
its "Hiding duplicates" label until every thumbnail has hashed, and that
hiding never advances progressively as hashes arrive. `internal/ui`'s tests
still build an `Overview` on the app's `fyneQueue`, so `fyne.Do` is still
inline there; undoing these needs that suite moved onto a drainable queue
too, or the compromises replaced with real synchronization.
```

- [ ] **Step 6: Verify the docs and re-gate**

Run:

```bash
gofmt -l . && go vet ./... && go test -race -count=1 -timeout 600s ./internal/ui/grid/ ./internal/uitest/
```

Expected: silent, then `ok` for both packages. Then re-read your `ARCHITECTURE.md` diff and check three things: the `uitest` file table still has one row per file that exists, the index entry names a file that exists, and nothing you wrote contradicts the `AGENTS.md` bullet you just edited.

- [ ] **Step 7: Do not commit.** Report the doc diff and the command output.

---

## Subagent roster

| Task | Deliverable | Model | `subagent_type` |
| --- | --- | --- | --- |
| 1 | `uitest.UIQueue` + its tests | `claude-sonnet-5-thinking-high` | `generalPurpose` |
| 2 | `uiQueue` seam on `Overview`, behaviour-neutral | `gpt-5.6-terra-medium` | `generalPurpose` |
| 3 | Harness installs the queue, `parkDecodes`, fallout fixed | `claude-opus-5-thinking-high` | `generalPurpose` |
| 4 | The three tests park before `Toggle` | `claude-sonnet-5-thinking-high` | `generalPurpose` |
| 5 | Full-suite gate, `ARCHITECTURE.md`/`AGENTS.md`/`todos.md` | `claude-sonnet-5-thinking-high` | `generalPurpose` |

Opus is used once, on Task 3, because that task's scope is genuinely unknown up front: it changes completion timing for every test in the package simultaneously, and there is no way to enumerate the fallout without judgement about what each affected test is asserting. Tasks 1, 2, 4 and 5 have their code written out here; they need care and verification, not discovery.

Dispatch rules:

- One implementer at a time, in order. Tasks 1 and 2 are independent of each other and *could* run in parallel, but they touch the same review budget — keep them sequential so the parent reviews one diff at a time.
- Hand each implementer only its own `### Task N` block plus the **Global Constraints** and **Why both halves are needed** sections. The task blocks are self-contained by design.
- Tell every implementer: no commits; do not touch files outside your task's file list; do not "improve" dHash, clustering, or the duplicate UX while you are in there.
- After each task the parent runs that task's own verification command before dispatching the next.

## Out of scope

- Undoing the production compromises in `dupes.go` (`SetHideDuplicates`, `hashRemaining`). Task 5 files this as a new todo instead.
- Moving `internal/ui`'s own harness onto a drainable queue. That suite passes under `-race` today and has no reason to change.
- The other packages with the same latent exposure (`deletion`, `favorites`, `exifwin`, `slideshow`, `spiral`, `toast`, and `internal/ui`'s own `fyne.Do` sites). `uitest.UIQueue` is deliberately general enough for them, but nothing here converts them.
- Golden screenshots, `make golden`, `internal/ui/testdata/`, translations, packaging.
- Anything about perceptual-hash accuracy, which was finished in `8f6cb04`.

## Suggested commit message (parent, after Task 5)

```
grid: marshal background completions through a drainable queue

Fyne's test driver implements DoFromGoroutine as a bare f(), so every
fyne.Do body in the grid ran on the decode-pool worker and raced the test
goroutine on canvas.Image, widget.Label, g.matches and g.groupSizes -
three tests failed under -race. Overview now marshals through a
per-instance uiQueue: a straight fyne.Do in the app, uitest.UIQueue under
the package's own tests, drained by Settle. A parkDecodes harness helper
holds the decode pool so a test can open a cold grid deterministically,
which lets all three tests keep exercising the grid-open, hashes-pending
flow instead of avoiding it.
```

## Self-review (parent)

- **Spec coverage.** All three named tests are Task 4; the package `-race` gate is Tasks 3 and 5; `todos.md` is Task 5.
- **Placeholder scan.** No TBD, no "add error handling", no "similar to Task N". Every code step carries its code; every verification step carries its command and expected output.
- **Type and name consistency.** `uiQueue` (interface, grid), `fyneQueue` (app impl, grid), `Overview.ui` (field), `uitest.UIQueue` with `Do`/`Drain`/`Len`, `parkDecodes(t, g) (unpark func())`. Task 1 defines `Drain() bool`; Tasks 2 and 3 consume exactly that signature. `*uitest.UIQueue` satisfies `uiQueue` because the interface names only `Do(func())` and `Drain() bool`.
- **Constraint check.** `Overview.ui` is per-instance, not a package-level var, so the `AGENTS.md` seam rule holds. No task commits. No `TODO` comments in source. No new `lang.L` keys, so no translation bundle changes.
- **Convergence check.** `Settle`'s loop terminates because a drained completion only re-spawns decodes whose thumbnails are by then cached, and a cache hit paints inline on the draining goroutine without queueing anything. Task 3 Step 5 treats a hang as a `BLOCKED` report rather than a timeout to raise.
