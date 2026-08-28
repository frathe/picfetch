# Duplicate-model follow-ups — Implementation Plan

> **For agentic workers:** this plan is executed by dispatching one fresh
> subagent per task, with a review gate between tasks. Steps use checkbox
> (`- [ ]`) syntax for tracking.

**Goal:** Close all five open items in `todos.md` — the unguarded
cross-goroutine read of `viewer.FileAt`, the never-cleared `hideApplyAt`,
the missing nil-URI guard in the hashing pass, the O(n)-per-arrow-key
`NextVisible`, and the unconditional `hostRep` computation in
`applyVisibleFilter`.

**Architecture:** The four small items are local edits. The load-bearing
one — the `FileAt` race — is fixed by giving `internal/dupes` an
*immutable snapshot* of the file set instead of the live `Count()` /
`KeyAt(i)` pair. `appState` publishes a `dupes.Snapshot` (keys + generation
+ key→index map) through an `atomic.Pointer` on every write to `files`;
every `Model` method takes one snapshot at entry and uses it throughout.
That removes the torn-read window by construction, and the snapshot's
key→index map is what makes `InspectSource` O(1), folding TODO #4 into the
same change.

**Tech Stack:** Go 1.x, Fyne v2, `internal/dupes`, `internal/ui`,
`internal/ui/grid`. No new dependencies.

**Spec:** this document (design is in the "Design" section below; there is
no separate spec file).

---

## Global Constraints

Copied from `AGENTS.md`. **Every task's requirements implicitly include
this section.** Subagents must be handed these verbatim.

- **Do not run `git commit`.** No task in this plan commits. The final
  handoff is a suggested commit message for the user.
- Open work belongs in `todos.md`; **do not add `TODO`/`FIXME` comments to
  source**.
- Update `ARCHITECTURE.md` in the same change when packages are added,
  removed, renamed, or files move between packages.
- Every user-visible string is `lang.L("English text")` and the exact key
  must be added to every `translations/*.json` bundle. *(No task here adds
  a user-visible string; if one appears, that is a red flag.)*
- Do not add mutable package-level test seams. Runtime/test-configurable
  values belong on `viewer` or the owning feature.
- `internal/ui/grid` marshals through its per-instance `UIQueue`
  (`g.ui.Do`), never a direct `fyne.Do`. Every `g.ui.Do` call must be made
  from inside the `g.decodes.Go` body it belongs to.
- Report UI-boundary failures with `fyne.LogError`; viewer-independent
  packages return errors. Mark intentionally ignored errors explicitly
  (`_ =`).
- Imports are grouped `goimports -local github.com/frathe/picfetch`.
- Never sleep to guess completion in tests. Use `dropAndWait`, `waitFor*`,
  feature `Settle`, and existing completion channels.
- Per-task verification: `make fmt-check`, `go vet ./...`, `go build ./...`,
  and the focused package tests named in the task. The full
  `go test -timeout 20m -race ./...` is the final gate (Task 9).

---

## Design

### The problem with `Count()` + `KeyAt(i)`

`dupes.FileSet` today is:

```go
type FileSet interface {
	Count() int
	KeyAt(i int) string
	Generation() uint64
}
```

`internal/ui`'s `dupeFileSet` implements it over `viewer.FileCount()` /
`viewer.FileAt(i)`, which are bare reads of `v.state.files`
(`internal/ui/viewer.go:882`, `internal/ui/state.go`). `dupes.Model.Compute`
runs on a hash-pool worker (`internal/ui/grid/hashengine.go:151`) and calls
`Count()` once, then `KeyAt(i)` in a loop. Meanwhile the UI goroutine
replaces the slice (`setFiles`, `clearFiles`, `removeFile`,
`internal/ui/sort.go:53`). Two independent failures:

1. **Index panic.** A shrink landing between `Count()` and `KeyAt(i)`, or
   part-way through the loop, indexes past the end.
2. **Data race.** The slice header itself (ptr, len, cap) is written by one
   goroutine and read unsynchronized by another. `-race` reports it; a torn
   read (new pointer, old length) is the mechanism behind (1).

The `todos.md` entry proposes hoisting the key slice out from under
`Compute`'s lock. **That is not sufficient** — a hoisted
`for i := range n { keys[i] = set.KeyAt(i) }` still interleaves with a
shrink, and the slice-header race is untouched. The fix has to make the
reader see an immutable value.

### The fix: publish an immutable snapshot

`dupes` gains a value type:

```go
type Snapshot struct {
	keys  []string
	byKey map[string]int
	gen   uint64
}
```

and `FileSet` collapses to a single method:

```go
type FileSet interface {
	Snapshot() Snapshot
}
```

`appState` holds `published atomic.Pointer[dupes.Snapshot]` and republishes
under a bumped generation as the last act of every write to `files`.
`viewer.Generation()` reads the generation *out of the published snapshot*,
so keys and generation move together atomically — where the old
`fileSetRevision.advance()` was a separate counter that could (and in
`finishSort` did) move before the files did.

Every `Model` method takes one `s := m.set.Snapshot()` at entry and uses
`s.Count()` / `s.KeyAt(i)` / `s.Generation()` throughout. The count and the
keys a method sees can no longer disagree.

### Why this also fixes `NextVisible`

`Snapshot` carries `byKey`, so `InspectSource` becomes `s.IndexOf(key)` —
O(1) instead of a full scan per arrow key. Separately, the
skip-hidden-extras walk takes `m.mu` once per candidate via
`IsHiddenExtra`; a `visibility()` helper reads `hide` and `groups` once per
`NextVisible` call and a free `hiddenExtra(hide, groups, i)` function does
the per-index test with no lock at all.

### Deliberate behaviour change to review

`finishSort` (`internal/ui/sort.go:127`) currently calls
`v.fileSetRevision.advance()` **before** `onDone(ordered)` — the generation
moves before the files do. After Task 3 the bump happens *inside* the
mutation, i.e. slightly later. This is the correct order (a worker can no
longer see a new generation over an old file list), but it is a real change
and Task 3 must re-run the whole `internal/ui` suite, not just focused
tests.

### Task dependency order

```
Task 1 (micro-fixes)         independent
Task 2 (Snapshot type)  ──┬─→ Task 3 (viewer publishes) ──┐
                          └─→ Task 4 (test fakes)      ───┤
                                                          ├─→ Task 6 (flip FileSet) ─→ Task 7 (NextVisible)
                              Task 5 (race repro, RED) ───┘                                    │
                                                                                               ↓
                                                                                    Task 8 (docs) → Task 9 (CI)
```

Tasks 2, 3 and 4 are **purely additive** — they add `Snapshot()` alongside
the existing methods, so the tree compiles and every test stays green
throughout. Task 6 is where the interface actually flips, which is why
nothing breaks in between.

**The RED window is Task 5 → Task 6 only.** Task 5 lands a test that is
*expected to fail*. Task 6 makes it pass. No other task may be run while
that test is red.

---

## File Structure

| File | Change | Responsibility after the change |
|---|---|---|
| `internal/dupes/snapshot.go` | **create** | The `Snapshot` value type: `NewSnapshot`, `Count`, `KeyAt`, `Generation`, `IndexOf`. |
| `internal/dupes/snapshot_test.go` | **create** | Table tests for `Snapshot`. |
| `internal/dupes/dupes.go` | modify | `FileSet` becomes `Snapshot() Snapshot`; every generation read goes through a snapshot. |
| `internal/dupes/groups.go` | modify | `Compute` / `Members` take one snapshot at entry; `membersOf` extracted. |
| `internal/dupes/visible.go` | modify | `InspectSource` O(1); `visibility()` / `hiddenExtra()` batching; snapshot-driven walks. |
| `internal/dupes/dupes_test.go` | modify | `fakeSet` gains `Snapshot()`, loses the other three. |
| `internal/dupes/visible_test.go` | modify | New tests for O(1) inspect lookup and single-snapshot reads. |
| `internal/ui/state.go` | modify | `appState` owns `published atomic.Pointer[dupes.Snapshot]`; every mutator republishes. |
| `internal/ui/viewer.go` | modify | `Generation()` reads the published snapshot; `fileSetRevision` field removed. |
| `internal/ui/sort.go` | modify | `v.state.files = ordered` → `v.state.reorder(ordered)`; `advance()` call removed. |
| `internal/ui/visibility.go` | modify | `dupeFileSet.Snapshot()`; the three old methods and the stale "KeyAt has to stay a plain lookup" comment removed. |
| `internal/ui/dupes_race_test.go` | **create** | The `-race` reproducer. |
| `internal/ui/grid/hashengine.go` | modify | nil-URI guard; `beginPass` clears the stale hide-apply floor. |
| `internal/ui/grid/hashengine_test.go` | **create** | Unit tests for `beginPass` and the nil-URI skip. |
| `internal/ui/grid/search.go` | modify | `hostRep` computed only when browsing. |
| `internal/ui/grid/harness_test.go` | modify | `hostSet` gains `Snapshot()`, loses the other three; stale comment removed. |
| `ARCHITECTURE.md` | modify | `internal/dupes` section reflects the `Snapshot`-based `FileSet`. |
| `todos.md` | modify | All five entries moved to Done. |

---

## Subagent Dispatch Table

| Task | Agent type | Model | Why this model |
|---|---|---|---|
| 1 | `go-expert` | `sonnet` | Three localized, fully-specified edits plus two small unit tests. Mechanical. |
| 2 | `go-expert` | `sonnet` | One new self-contained value type with the code given verbatim. |
| 3 | `go-expert` | `opus` | Touches the file-set generation contract — load-bearing lifecycle semantics, and a deliberate ordering change in `finishSort`. |
| 4 | `go-expert` | `sonnet` | Two mechanical test-fake additions. |
| 5 | `go-expert` | `fable` | Must author a concurrency reproducer against an unfamiliar test harness and *prove it fails* — the hardest judgment call in the plan. |
| 6 | `go-expert` | `opus` | Interface flip across a whole package and three implementations; must preserve exact behaviour. |
| 7 | `go-expert` | `opus` | Rewrites concurrency-sensitive navigation with a documented "reproduces the old branch order exactly" contract. |
| 8 | `general-purpose` | `sonnet` | Documentation and bookkeeping. |
| 9 | *(none — I run it)* | — | Final CI-parity gate. |

**Every dispatch prompt must include:** the Global Constraints section
above, the task's own text verbatim, and the instruction *"read
`AGENTS.md` and the files you are about to edit before making any change;
do not commit."*

---

## Task 1: Three micro-fixes in `internal/ui/grid`

Closes `todos.md` items **"The hashing pass has no nil-URI guard"** (prio
25), **"`hideApplyAt` is never cleared between hashing passes"** (prio 10),
and **"`applyVisibleFilter` computes `hostRep` unconditionally"** (prio 10).

**Files:**
- Modify: `internal/ui/grid/hashengine.go:111` (nil guard),
  `internal/ui/grid/hashengine.go:138` (`e.hashJobs.Add`)
- Modify: `internal/ui/grid/search.go:131`
- Create: `internal/ui/grid/hashengine_test.go`

**Interfaces:**
- Consumes: nothing from other tasks.
- Produces: `func (e *hashEngine) beginPass(n int)` — used by nothing
  outside `hashengine.go`, but Task 6 will read this file, so the name
  matters.

- [ ] **Step 1: Write the failing tests**

Create `internal/ui/grid/hashengine_test.go`:

```go
package grid

import (
	"testing"
	"time"
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

	g := newTestOverview(t, host)

	if n := g.hashes.Run(func(_ dupes.Groups, _ int32, _ uint64) {}); n != 2 {
		t.Errorf("Run queued %d jobs, want 2 (the nil URI must be skipped)", n)
	}
}
```

> **Note for the implementer:** `hostWith` is in
> `internal/ui/grid/harness_test.go`. `newTestOverview` is illustrative —
> read `harness_test.go` and use whatever constructor the existing dupes
> tests use (around `internal/ui/grid/harness_test.go:96`, which calls
> `New(host, win, dupes.New(hostSet{host: host}))`). Adapt the third test
> to that harness; keep its assertion (2 jobs, no panic) exactly.
> `hostWith` writes real JPEGs, so the two real files really do get queued.
> Add the `dupes` import.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
go test -run 'TestBeginPass|TestRun_SkipsNilURI' ./internal/ui/grid/
```

Expected: the two `TestBeginPass` tests fail to compile
(`e.beginPass undefined`); `TestRun_SkipsNilURI` panics with a nil-pointer
dereference on `u.String()`.

- [ ] **Step 3: Add the nil guard in `Run`**

In `internal/ui/grid/hashengine.go`, in `Run`'s file loop, immediately
after `u := e.host.FileAt(i)`:

```go
		u := e.host.FileAt(i)
		// The URI string is the model's key throughout, so an index with
		// no URI has nothing this pass can dedup, cache, or hash by:
		// skip it rather than dereferencing it. Every neighbouring
		// helper (rememberHash, hashOf, ... in grid/dupes.go) applies
		// the same guard.
		if u == nil {
			continue
		}
```

Then **delete** the now-false final sentence of the comment block below it
("this loop already cannot survive a nil URI, since the key it dedups and
caches by is that URI's own string") and replace it with a pointer to the
guard:

```go
		// The URI string is the model's key throughout - it stores facts
		// about files, not fyne.URIs. Read straight off the model here
		// rather than through Overview's hashOf/pixelCountOf/
		// hashFailedOf wrappers, which exist to nil-guard the fyne.URI
		// the cell and Warm paths hand them; the guard above is this
		// loop's equivalent.
		key := u.String()
```

- [ ] **Step 4: Add `beginPass` and call it**

In `internal/ui/grid/hashengine.go`, replace `e.hashJobs.Add(int32(n))`
(just after the `if n == 0 { return 0 }` guard) with `e.beginPass(n)`, and
add the method next to `shouldScheduleHideApply`:

```go
// beginPass books n new jobs onto the counter and, when this is the first
// work in flight, clears the mid-window throttle floor.
//
// shouldScheduleHideApply leaves hideApplyAt at the previous pass's last
// mid-window timestamp when that pass ended, so without this a pass
// starting within hideApplyMinInterval of the old one's last apply would
// skip its own first mid-window apply. The last job always applies, so
// that was latency, not lost work - this removes the latency.
//
// The Load/Add pair is deliberately not atomic as a unit: a worker
// finishing between the two can only make this look like a continuing
// pass and keep a floor that is about to expire anyway. The cost of
// losing that race is one throttled apply, never a wrong result.
func (e *hashEngine) beginPass(n int) {
	if e.hashJobs.Load() == 0 {
		e.hideApplyAt.Store(0)
	}
	e.hashJobs.Add(int32(n))
}
```

Update the `hideApply`/`hideApplyAt` field comment in the `hashEngine`
struct to mention that `beginPass` clears the floor between passes.

- [ ] **Step 5: Make `hostRep` conditional**

In `internal/ui/grid/search.go`, inside `applyVisibleFilter`, replace:

```go
		needle := strings.ToLower(g.query)
		hostRep := g.dupes.RepresentativeOf(g.browseHost)
```

with:

```go
		needle := strings.ToLower(g.query)
		// Only the browseFilter branch below reads hostRep, and browse is
		// off (browseHost == -1) for every plain search or hide pass - the
		// common case. Computing it unconditionally spent a model-mutex
		// acquisition per filter pass on a value nothing read.
		hostRep := -1
		if browseFilter {
			hostRep = g.dupes.RepresentativeOf(g.browseHost)
		}
```

- [ ] **Step 6: Run the tests to verify they pass**

```bash
go test -run 'TestBeginPass|TestRun_SkipsNilURI' ./internal/ui/grid/
go test ./internal/ui/grid/
make fmt-check && go vet ./... && go build ./...
```

Expected: all PASS.

**Review gate (I check):** `git diff` shows exactly three source edits plus
one new test file; no `TODO` comments added; `beginPass` is called exactly
once; `hostRep`'s `-1` default is only ever read under `browseFilter`.

---

## Task 2: The `dupes.Snapshot` value type

**Files:**
- Create: `internal/dupes/snapshot.go`
- Create: `internal/dupes/snapshot_test.go`

**Interfaces:**
- Consumes: nothing.
- Produces — every later task depends on these exact signatures:
  - `func NewSnapshot(keys []string, gen uint64) Snapshot`
  - `func (s Snapshot) Count() int`
  - `func (s Snapshot) KeyAt(i int) string`
  - `func (s Snapshot) Generation() uint64`
  - `func (s Snapshot) IndexOf(key string) int`

- [ ] **Step 1: Write the failing tests**

Create `internal/dupes/snapshot_test.go`:

```go
package dupes

import "testing"

func TestNewSnapshot_CountKeyAtAndGeneration(t *testing.T) {
	s := NewSnapshot([]string{"a", "b", "c"}, 7)

	if got := s.Count(); got != 3 {
		t.Errorf("Count() = %d, want 3", got)
	}
	if got := s.KeyAt(1); got != "b" {
		t.Errorf("KeyAt(1) = %q, want %q", got, "b")
	}
	if got := s.Generation(); got != 7 {
		t.Errorf("Generation() = %d, want 7", got)
	}
}

func TestSnapshotKeyAt_OutOfRangeIsEmpty(t *testing.T) {
	s := NewSnapshot([]string{"a"}, 1)

	for _, i := range []int{-1, 1, 99} {
		if got := s.KeyAt(i); got != "" {
			t.Errorf("KeyAt(%d) = %q, want %q", i, got, "")
		}
	}
}

func TestSnapshotIndexOf(t *testing.T) {
	s := NewSnapshot([]string{"a", "b", "a"}, 1)

	tests := []struct {
		key  string
		want int
	}{
		{"a", 0}, // duplicates keep the lowest index, matching the scan this replaces
		{"b", 1},
		{"zzz", -1},
		{"", -1},
	}
	for _, tt := range tests {
		if got := s.IndexOf(tt.key); got != tt.want {
			t.Errorf("IndexOf(%q) = %d, want %d", tt.key, got, tt.want)
		}
	}
}

// The whole point of the type: a caller mutating its own slice afterwards
// must not be able to change what the snapshot reports.
func TestNewSnapshot_CopiesKeys(t *testing.T) {
	keys := []string{"a", "b"}
	s := NewSnapshot(keys, 1)

	keys[0] = "mutated"

	if got := s.KeyAt(0); got != "a" {
		t.Errorf("KeyAt(0) = %q after caller mutation, want %q", got, "a")
	}
}

func TestZeroSnapshot_IsAValidEmptySet(t *testing.T) {
	var s Snapshot

	if got := s.Count(); got != 0 {
		t.Errorf("Count() = %d, want 0", got)
	}
	if got := s.KeyAt(0); got != "" {
		t.Errorf("KeyAt(0) = %q, want %q", got, "")
	}
	if got := s.Generation(); got != 0 {
		t.Errorf("Generation() = %d, want 0", got)
	}
	if got := s.IndexOf("a"); got != -1 {
		t.Errorf("IndexOf(\"a\") = %d, want -1", got)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

```bash
go test -run 'Snapshot' ./internal/dupes/
```

Expected: compile failure, `undefined: NewSnapshot`.

- [ ] **Step 3: Write the implementation**

Create `internal/dupes/snapshot.go`:

```go
// The immutable view of a file set that every Model method reads through.

package dupes

// Snapshot is a FileSet frozen at one generation: the file keys in index
// order, the generation they belong to, and a key-to-index map.
//
// It exists because the model is read from hashing workers while the UI
// goroutine replaces the file set underneath them. The Count()/KeyAt(i)
// pair this replaces could not be read consistently: a shrink landing
// between the two, or part-way through a loop over them, indexed past the
// end of a slice whose header was itself being written unsynchronized.
// A method that takes one Snapshot at its top holds a count and a key
// list that cannot disagree.
//
// byKey is what makes IndexOf O(1), which is what makes InspectSource -
// and so every arrow key while inspecting - cheap enough to call per
// keystroke on a 50k-file drop.
//
// The zero value is a valid empty snapshot: Count 0, generation 0, no
// keys. That is what a viewer with no files loaded publishes.
type Snapshot struct {
	keys  []string
	byKey map[string]int
	gen   uint64
}

// NewSnapshot builds a Snapshot over keys at generation gen. keys is
// copied, so the caller keeps ownership of its own slice and the
// snapshot stays immutable however that slice is mutated later.
//
// A duplicate key keeps its lowest index, which is what the linear scan
// this replaces returned (InspectSource stopped at the first match).
func NewSnapshot(keys []string, gen uint64) Snapshot {
	cp := append([]string(nil), keys...)
	byKey := make(map[string]int, len(cp))
	for i, k := range cp {
		if _, seen := byKey[k]; !seen {
			byKey[k] = i
		}
	}

	return Snapshot{keys: cp, byKey: byKey, gen: gen}
}

// Count is how many files the snapshot holds.
func (s Snapshot) Count() int { return len(s.keys) }

// KeyAt is the key at i, or "" when i is out of range - the same empty
// key an absent URI produced before, which every caller already reads as
// "no facts are stored about this index".
func (s Snapshot) KeyAt(i int) string {
	if i < 0 || i >= len(s.keys) {
		return ""
	}

	return s.keys[i]
}

// Generation is the file-set revision these keys belong to. A change
// invalidates stored hashes - see Model.WipeIfStale.
func (s Snapshot) Generation() uint64 { return s.gen }

// IndexOf is the index of key, or -1 when key is empty or absent.
func (s Snapshot) IndexOf(key string) int {
	if key == "" {
		return -1
	}
	i, ok := s.byKey[key]
	if !ok {
		return -1
	}

	return i
}
```

- [ ] **Step 4: Run the tests to verify they pass**

```bash
go test -run 'Snapshot' ./internal/dupes/
go test ./internal/dupes/
make fmt-check && go vet ./... && go build ./...
```

Expected: all PASS.

**Review gate (I check):** `Snapshot` has no exported fields; `NewSnapshot`
really copies; the zero value works; nothing else in the package was
touched.

---

## Task 3: `appState` publishes the snapshot

**Files:**
- Modify: `internal/ui/state.go` (whole file)
- Modify: `internal/ui/viewer.go:148-153` (the `fileSetRevision` field),
  `internal/ui/viewer.go:621`, `internal/ui/viewer.go:832`,
  `internal/ui/viewer.go:909-911` (`Generation`)
- Modify: `internal/ui/sort.go:53`, `internal/ui/sort.go:127`
- Modify: `internal/ui/visibility.go` (add `Snapshot()` to `dupeFileSet`)

**Interfaces:**
- Consumes: `dupes.NewSnapshot`, `dupes.Snapshot` from Task 2.
- Produces:
  - `func (s *appState) snapshot() dupes.Snapshot`
  - `func (s *appState) reorder(files []fyne.URI)`
  - `func (s dupeFileSet) Snapshot() dupes.Snapshot`

**This task is additive.** `dupeFileSet` keeps its `Count`/`KeyAt`/
`Generation` methods; the `dupes.FileSet` interface is untouched. The tree
must compile and every existing test must pass at the end of this task.

- [ ] **Step 1: Write the failing test**

Add to `internal/ui/filestate_test.go` (next to the existing generation
test around line 185):

```go
// The published snapshot and the generation are one value, so a reader
// can never hold keys from one file set and a generation from the next.
func TestFileSnapshot_KeysAndGenerationMoveTogether(t *testing.T) {
	v := newTestViewer(t)
	files := uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg", "c.jpg")
	dropAndWait(t, v, files...)

	before := v.state.snapshot()
	if got := before.Count(); got != 3 {
		t.Fatalf("snapshot Count() = %d, want 3", got)
	}
	if got := before.KeyAt(0); got != files[0].String() {
		t.Errorf("snapshot KeyAt(0) = %q, want %q", got, files[0].String())
	}

	v.RemoveFile(2)

	after := v.state.snapshot()
	if got := after.Count(); got != 2 {
		t.Errorf("snapshot Count() = %d after removal, want 2", got)
	}
	if after.Generation() <= before.Generation() {
		t.Errorf("Generation() = %d after removal, want > %d",
			after.Generation(), before.Generation())
	}
	if got := v.Generation(); got != after.Generation() {
		t.Errorf("v.Generation() = %d, snapshot generation = %d; they must be one value",
			got, after.Generation())
	}
	// The old snapshot is immutable: it still describes the file set it
	// was published for.
	if got := before.Count(); got != 3 {
		t.Errorf("previously published snapshot Count() = %d, want 3", got)
	}
}
```

> **Helpers, verified:** `newTestViewer(t) *viewer` and
> `dropAndWait(t, v, uris...)` are in `internal/ui/harness_test.go`;
> `uitest.TempDirJPEGURIs(t, names ...string) []fyne.URI` is in
> `internal/uitest/uitest.go:88`. The neighbouring
> `TestGenerationTracksFileSetIdentityNotNavigation`
> (`internal/ui/filestate_test.go:178`) is the pattern to match.

- [ ] **Step 2: Run the test to verify it fails**

```bash
go test -run TestFileSnapshot_KeysAndGenerationMoveTogether ./internal/ui/
```

Expected: compile failure, `v.state.snapshot undefined`.

- [ ] **Step 3: Give `appState` the published snapshot**

In `internal/ui/state.go`, add the field and the two new methods, and call
`publish()` as the last act of `setFiles`, `clearFiles` and `removeFile`:

```go
type appState struct {
	files         []fyne.URI
	unsortedFiles []fyne.URI
	index         int
	sortMode      filesort.Mode
	mergeMode     bool

	// published is the immutable {keys, generation} view of files that
	// readers off the UI goroutine use instead of touching the slice -
	// internal/dupes, whose Model is read from hashing workers while the
	// UI goroutine replaces files underneath them.
	//
	// Every write to files republishes under a bumped generation as its
	// last act, so the two move together. The fileSetRevision counter
	// this replaces was separate, and in finishSort it advanced before
	// the files did: a worker could see the new generation over the old
	// list.
	published atomic.Pointer[dupes.Snapshot]
}

// publish replaces the published snapshot with one built from the
// current files, at the next generation. Mutators call it last; nothing
// else may.
func (s *appState) publish() {
	var gen uint64
	if prev := s.published.Load(); prev != nil {
		gen = prev.Generation()
	}

	keys := make([]string, len(s.files))
	for i, u := range s.files {
		if u != nil {
			keys[i] = u.String()
		}
	}

	snap := dupes.NewSnapshot(keys, gen+1)
	s.published.Store(&snap)
}

// snapshot is the current published view of the file set. Safe from any
// goroutine; it is the only read of the file set that is.
func (s *appState) snapshot() dupes.Snapshot {
	if p := s.published.Load(); p != nil {
		return *p
	}

	return dupes.Snapshot{}
}

// reorder replaces files with an already-sorted list of the same members,
// leaving unsortedFiles and index alone. It exists so a reorder goes
// through a mutator like every other write does, rather than assigning
// the field directly and skipping publish.
func (s *appState) reorder(files []fyne.URI) {
	s.files = append([]fyne.URI(nil), files...)
	s.publish()
}
```

Add `"sync/atomic"` and `"github.com/frathe/picfetch/internal/dupes"` to
the imports, in the right `goimports -local` groups.

- [ ] **Step 4: Route every mutation and every generation read through it**

1. `internal/ui/state.go` — append `s.publish()` to the end of `setFiles`,
   `clearFiles`, and `removeFile` (in `removeFile`, before `return target`).
2. `internal/ui/sort.go:53` — `v.state.files = ordered` becomes
   `v.state.reorder(ordered)`.
3. `internal/ui/sort.go:127` — delete `v.fileSetRevision.advance()`.
   Replace the surrounding comment's mention of the revision with: *"The
   generation bump now rides on the file-set write itself (appState.publish),
   so it happens inside onDone rather than ahead of it — a worker can no
   longer see the new generation over the old list."*
4. `internal/ui/viewer.go:621` and `:832` — delete both
   `v.fileSetRevision.advance()` calls (`clearFiles` and `removeFile`
   publish for them now).
5. `internal/ui/viewer.go:148-153` — delete the `fileSetRevision revision`
   field and its comment.
6. `internal/ui/viewer.go:909` — `Generation` becomes:

```go
// Generation is the current index-to-URI file-set revision. Navigation does
// not change it; replacement, reorder, removal, and clear operations do.
// It is read out of the published snapshot rather than a counter of its
// own, so the generation and the keys it describes are one value - see
// appState.publish.
func (v *viewer) Generation() uint64 {
	return v.state.snapshot().Generation()
}
```

7. `internal/ui/visibility.go` — add, next to the other `dupeFileSet`
   methods:

```go
// Snapshot is the viewer's published, immutable view of the file set:
// what dupes.Model reads instead of walking Count()/KeyAt(i) live while
// the UI goroutine replaces the slice underneath it.
func (s dupeFileSet) Snapshot() dupes.Snapshot { return s.v.state.snapshot() }
```

Add the `dupes` import. **Leave `Count`, `KeyAt` and `Generation` in place
for now** — Task 6 removes them.

8. Check that `revision` (`internal/ui/lifecycle.go:11`) still has other
   users (`requestLifecycle` embeds it). If `fileSetRevision` was its only
   *direct* user, leave the type alone — do not delete it.

- [ ] **Step 5: Run the tests to verify they pass**

```bash
go test -run TestFileSnapshot_KeysAndGenerationMoveTogether ./internal/ui/
go test -timeout 10m ./internal/ui/...
make fmt-check && go vet ./... && go build ./...
```

Expected: all PASS. **The whole `internal/ui` tree, not just focused
tests** — this task changes when the generation advances relative to
`finishSort`'s `onDone`, and `internal/ui/filestate_test.go`,
`internal/ui/grid/dupes_test.go` and `internal/ui/spiral/spiral_test.go`
all read generations.

- [ ] **Step 6: Report the generation-ordering outcome**

State explicitly in the report whether any test needed adjusting because
of the `finishSort` ordering change, and if so, which and why. Do **not**
silently relax an assertion — if one fails, report it and stop.

**Review gate (I check):** `grep -rn "fileSetRevision" internal/` returns
nothing; every `state.files` write is inside an `appState` method that ends
in `publish()`; `publish()` is called from nowhere else; no test assertion
was weakened.

---

## Task 4: Test fakes gain `Snapshot()`

**Files:**
- Modify: `internal/dupes/dupes_test.go:14-21` (`fakeSet`)
- Modify: `internal/ui/grid/harness_test.go:142-171` (`hostSet`)

**Interfaces:**
- Consumes: `dupes.NewSnapshot` (Task 2).
- Produces: `func (f *fakeSet) Snapshot() Snapshot`,
  `func (s hostSet) Snapshot() dupes.Snapshot` — Task 6 needs both to
  exist before it flips the interface.

Additive only. Nothing is removed in this task.

- [ ] **Step 1: Add `Snapshot()` to `fakeSet`**

In `internal/dupes/dupes_test.go`, next to the existing three methods:

```go
// Snapshot rebuilds from keys/gen on every call, which is what lets a
// test mutate those fields directly and have the next model call see the
// change - the production adapter (internal/ui's dupeFileSet) instead
// returns an already-published value.
func (f *fakeSet) Snapshot() Snapshot { return NewSnapshot(f.keys, f.gen) }
```

- [ ] **Step 2: Add `Snapshot()` to `hostSet`**

In `internal/ui/grid/harness_test.go`, next to the existing three methods:

```go
// Snapshot rebuilds from the host on every call. Production hands
// grid.New a model the viewer owns, whose adapter returns an
// already-published value; this stands in for it over a fakeHost, with
// the same nil-URI-becomes-empty-key rule.
func (s hostSet) Snapshot() dupes.Snapshot {
	n := s.host.FileCount()
	keys := make([]string, n)
	for i := range n {
		if u := s.host.FileAt(i); u != nil {
			keys[i] = u.String()
		}
	}

	return dupes.NewSnapshot(keys, s.host.Generation())
}
```

- [ ] **Step 3: Run the tests to verify nothing broke**

```bash
go test ./internal/dupes/ ./internal/ui/grid/
make fmt-check && go vet ./... && go build ./...
```

Expected: all PASS (these are additions; behaviour is unchanged).

**Review gate (I check):** both methods exist, both handle the nil/empty
cases, and the old three methods are still present.

---

## Task 5: The race reproducer — RED

**This task deliberately leaves a failing test in the tree.** Task 6 is
the only thing that may run next.

**Files:**
- Create: `internal/ui/dupes_race_test.go`

**Interfaces:**
- Consumes: `newTestUI`, `dropAndWait` and the temp-image helper from
  `internal/ui/harness_test.go`.
- Produces: nothing.

- [ ] **Step 1: Read the harness first**

Read `internal/ui/harness_test.go` in full, plus
`internal/ui/filestate_test.go`, before writing anything. You need the real
names and signatures of `newTestUI`, `dropAndWait`, and whatever creates
temp images. Do not guess them.

- [ ] **Step 2: Write the reproducer**

Create `internal/ui/dupes_race_test.go`. This is the shape; adapt the
harness calls to what you actually found:

```go
package ui

import (
	"sync"
	"testing"
)

// dupes.Model.Compute runs on a hash-pool worker (internal/ui/grid's
// hashengine) while the UI goroutine replaces v.state.files. Before the
// Snapshot migration, Compute reached the live slice through
// dupeFileSet's Count()/KeyAt(i): a shrink landing between those two
// reads, or part-way through the loop over them, indexed past the end,
// and the slice header itself was read with no synchronization at all.
//
// Run under -race. This is the regression guard for that; it must not
// report a race and must not panic.
func TestDupeCompute_ConcurrentWithFileRemoval(t *testing.T) {
	v, _, _ := newTestUI(t)
	files := uitest.TempImages(t, 48)
	dropAndWait(t, v, files...)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 500 {
			v.dupes.Compute()
		}
	}()

	for v.FileCount() > 1 {
		v.RemoveFile(v.FileCount() - 1)
	}

	wg.Wait()
}
```

Constraints:
- `RemoveFile` must stay on the test (UI) goroutine; only `Compute` runs on
  the second goroutine. That is the production shape.
- No `time.Sleep`. The `wg.Wait()` is the barrier.
- If `newTestUI`'s cleanup (`drain`) needs the goroutine finished before it
  runs, `wg.Wait()` before returning already guarantees it.

- [ ] **Step 3: Run it under `-race` and confirm it fails**

```bash
go test -race -run TestDupeCompute_ConcurrentWithFileRemoval ./internal/ui/
```

Expected: **FAIL** — either `WARNING: DATA RACE` naming
`internal/ui/state.go` / `internal/ui/viewer.go`'s `FileAt`, or an
`index out of range` panic inside `dupes.Compute`. Either outcome is a
valid RED.

If it passes, the reproducer is not exercising the race — increase the
file count and the iteration count, and make sure the removal loop really
overlaps the `Compute` loop (start the goroutine before the loop, as
written). Report back rather than weakening the test.

- [ ] **Step 4: Record the failure**

Paste the exact failure output into the task report. That output is the
evidence Task 6 has to invalidate.

**Review gate (I check):** the test fails under `-race` for the stated
reason (I re-run it myself and read the output); it does not fail for an
unrelated reason such as a harness misuse.

---

## Task 6: Flip `FileSet` to `Snapshot()` — GREEN

Closes `todos.md` item **"`viewer.FileAt` is unguarded and read off the UI
goroutine"** (prio 18).

**Files:**
- Modify: `internal/dupes/dupes.go` (the `FileSet` interface at :30-34, and
  every `m.set.` read: `:130`, `:145`, `:161`, `:172`, `:185`, `:233-236`)
- Modify: `internal/dupes/groups.go` (`Compute` :42-93, `Members` :132)
- Modify: `internal/dupes/visible.go` (`BeginInspect` :35-36,
  `InspectSource` :68-69, `NextVisible` :128, `FirstVisible` :163,
  `LastVisible` :174, `VisibleIndexesExcept` :189)
- Modify: `internal/ui/visibility.go` (remove `Count`/`KeyAt`/`Generation`
  and the stale comment)
- Modify: `internal/ui/grid/harness_test.go` (same removal on `hostSet`)
- Modify: `internal/dupes/dupes_test.go` (same removal on `fakeSet`)

**Interfaces:**
- Consumes: `Snapshot` (Task 2), `Snapshot()` on all three implementations
  (Tasks 3 and 4).
- Produces:
  - `type FileSet interface { Snapshot() Snapshot }`
  - `func (m *Model) wipeIfStale(gen uint64)` (unexported; `WipeIfStale`
    keeps its exported signature)

- [ ] **Step 1: Narrow the interface**

In `internal/dupes/dupes.go`:

```go
// FileSet is the ordered file set the model groups over. It hands back an
// immutable Snapshot rather than answering Count/KeyAt/Generation live:
// the model is read from hashing workers while the app replaces the file
// set on its UI goroutine, and a method that took a count and then a key
// could see the two disagree. See Snapshot.
type FileSet interface {
	Snapshot() Snapshot
}
```

- [ ] **Step 2: Take one snapshot per method**

Mechanical, everywhere in the package: replace `m.set.Count()` with
`s.Count()`, `m.set.KeyAt(i)` with `s.KeyAt(i)`, and `m.set.Generation()`
with `s.Generation()`, where `s := m.set.Snapshot()` is taken **once, as
the method's first statement**. Never call `m.set.Snapshot()` twice in one
method, and never take one while holding `m.mu`.

Split `WipeIfStale` so the generation can come from a snapshot the caller
already has:

```go
// WipeIfStale ensures the stored facts belong to set's current
// generation, wiping hashes, hashFailed, and native when they don't.
// Every read path in this file calls it first.
func (m *Model) WipeIfStale() {
	m.wipeIfStale(m.set.Snapshot().Generation())
}

// wipeIfStale is WipeIfStale against a generation the caller already
// holds, so a method that has taken a snapshot does not take a second
// one just to re-read the same number.
func (m *Model) wipeIfStale(gen uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wipeIfStaleLocked(gen)
}
```

`Compute` becomes (only the head changes; the grouping body is untouched):

```go
func (m *Model) Compute() Groups {
	m.computes.Add(1)
	s := m.set.Snapshot()
	n := s.Count()
	sizes := make([]int, n)
	reps := make([]int, n)
	for i := range n {
		reps[i] = i
	}
	m.wipeIfStale(s.Generation())

	m.mu.Lock()
	idx := make([]int, 0, n)
	hs := make([]uint64, 0, n)
	hashed := make([]bool, n)
	px := make([]int, n)
	dist := m.dist
	for i := range n {
		key := s.KeyAt(i)
		...
```

Update `Compute`'s doc comment: the "hoist the key slice out of the lock"
escape hatch it and `internal/ui/visibility.go` both describe is now moot —
`s` is already a value, so reading it under `m.mu` cannot reach back into
the model or deadlock a worker. Say that instead.

`NativeSizeAt` (`dupes.go:232`):

```go
func (m *Model) NativeSizeAt(i int) (w, h int, ok bool) {
	key := m.set.Snapshot().KeyAt(i)
	if key == "" {
		return 0, 0, false
	}
	sz, ok := m.NativeSize(key)
	if !ok || sz.X <= 0 || sz.Y <= 0 {
		return 0, 0, false
	}

	return sz.X, sz.Y, true
}
```

(`KeyAt` already returns `""` out of range, so the explicit bounds check
goes away. Behaviour is identical: an empty key was never stored.)

- [ ] **Step 3: Drop the three old methods from all three implementations**

- `internal/ui/visibility.go`: delete `dupeFileSet.Count`,
  `dupeFileSet.KeyAt`, `dupeFileSet.Generation`. Delete the paragraph of
  the type comment beginning *"KeyAt has to stay a plain lookup"* and
  replace the whole comment's second half with a note that the adapter now
  just forwards the viewer's published snapshot.
- `internal/ui/grid/harness_test.go`: same three deletions on `hostSet`,
  same stale-comment deletion.
- `internal/dupes/dupes_test.go`: same three deletions on `fakeSet`.

- [ ] **Step 4: Run the reproducer — it must now pass**

```bash
go test -race -run TestDupeCompute_ConcurrentWithFileRemoval ./internal/ui/
```

Expected: **PASS**, no `DATA RACE` warning.

- [ ] **Step 5: Run the full affected tree**

```bash
go test -race ./internal/dupes/ ./internal/ui/... 
make fmt-check && go vet ./... && go build ./...
```

Expected: all PASS. If a `internal/dupes` test fails, it is almost
certainly because `fakeSet.Snapshot()` rebuilds from `f.keys`/`f.gen` on
each call while the test mutates those between model calls — that is
intentional and matches the old live behaviour. Report any test you cannot
make pass without changing its assertion.

**Review gate (I check):** `grep -rn "set\.Count()\|set\.KeyAt(\|set\.Generation()" internal/`
returns nothing; no method calls `m.set.Snapshot()` more than once; no
snapshot is taken while `m.mu` is held; the reproducer passes under `-race`
and I re-run it myself.

---

## Task 7: `NextVisible` in O(1) lock acquisitions

Closes `todos.md` item **"`NextVisible` is O(n) per arrow key"** (prio 12).

**Files:**
- Modify: `internal/dupes/visible.go` (`InspectSource`, `InspectMembers`,
  `IsHiddenExtra`, `NextVisible`, `FirstVisible`, `LastVisible`,
  `VisibleIndexesExcept`)
- Modify: `internal/dupes/groups.go` (`Members` → thin wrapper over a new
  `membersOf`)
- Modify: `internal/dupes/visible_test.go` (new tests)

**Interfaces:**
- Consumes: `Snapshot.IndexOf` (Task 2), the flipped `FileSet` (Task 6).
- Produces (all unexported):
  - `func (m *Model) visibility() (hide bool, groups Groups)`
  - `func hiddenExtra(hide bool, groups Groups, i int) bool`
  - `func membersOf(groups Groups, n, i int) []int`
  - `func (m *Model) inspectMembers(s Snapshot) []int`

**Behaviour must not change.** `NextVisible`'s doc comment promises it
"reproduces the pre-extraction viewer's `nextVisibleIndex` exactly,
including its branch order". Preserve that: inspect-members first, then the
hide/delta check, then the walk.

- [ ] **Step 1: Write the failing tests**

Add to `internal/dupes/visible_test.go`:

```go
// countingSet reports how many times the model asked for a snapshot, so
// a test can prove a navigation step takes exactly one rather than one
// per candidate index.
type countingSet struct {
	inner     *fakeSet
	snapshots int
}

func (c *countingSet) Snapshot() Snapshot {
	c.snapshots++
	return c.inner.Snapshot()
}

// One arrow key is one snapshot, however many indices the walk skips.
// Before this, InspectSource rescanned the whole set for the inspect key
// on every step, and the skip-hidden-extras walk took the model mutex
// once per candidate.
func TestNextVisible_TakesOneSnapshotPerCall(t *testing.T) {
	set := &countingSet{inner: newFakeSet(64, 1)}
	m := New(set)
	set.snapshots = 0

	m.NextVisible(0, 1)

	if set.snapshots != 1 {
		t.Errorf("NextVisible took %d snapshots, want 1", set.snapshots)
	}
}

// InspectSource is a map lookup now, not a scan, but it must still
// answer the same question: where does the inspected file live today?
func TestInspectSource_FindsKeyAfterTheSetShifts(t *testing.T) {
	set := newFakeSet(4, 1) // keys "a" "b" "c" "d"
	m := New(set)
	m.BeginInspect(2) // "c"

	if got := m.InspectSource(); got != 2 {
		t.Fatalf("InspectSource() = %d, want 2", got)
	}

	set.keys = []string{"d", "c", "a"} // "c" moved to index 1

	if got := m.InspectSource(); got != 1 {
		t.Errorf("InspectSource() = %d after the set shifted, want 1", got)
	}

	set.keys = []string{"d", "a"} // "c" is gone

	if got := m.InspectSource(); got != -1 {
		t.Errorf("InspectSource() = %d after the file was removed, want -1", got)
	}
}
```

> **Note for the implementer:** `newFakeSet` is in
> `internal/dupes/dupes_test.go` and builds keys `"a"`, `"b"`, `"c"`, …
> `countingSet` may need to live in `dupes_test.go` next to `fakeSet`
> instead, if that is where the package's fakes are kept — follow the
> existing convention.

- [ ] **Step 2: Run the tests to verify they fail**

```bash
go test -run 'TestNextVisible_TakesOneSnapshotPerCall|TestInspectSource_FindsKeyAfterTheSetShifts' ./internal/dupes/
```

Expected: `TestNextVisible_TakesOneSnapshotPerCall` FAILs reporting more
than one snapshot (`InspectMembers` and `HideDuplicates` and the walk each
reach the set separately). `TestInspectSource_FindsKeyAfterTheSetShifts`
may already pass — that is fine, it is the behaviour-preservation guard.

- [ ] **Step 3: Add the batching helpers**

In `internal/dupes/visible.go`:

```go
// visibility reads hide and the installed group snapshot in one lock
// acquisition, for callers that then test many indices against them.
// The walks below used to call IsHiddenExtra per candidate, which is one
// mutex acquisition per candidate; at 50k files with hide on, that was
// the cost of a single arrow key.
func (m *Model) visibility() (hide bool, groups Groups) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.hide, m.groups
}

// hiddenExtra is IsHiddenExtra's test against an already-read hide flag
// and group snapshot: no lock, so a walk can call it per index.
func hiddenExtra(hide bool, groups Groups, i int) bool {
	if !hide {
		return false
	}
	if groups.Size(i) < 2 {
		return false
	}

	return i != groups.RepresentativeOf(i)
}

// IsHiddenExtra reports whether i is a non-representative member of a
// duplicate group while hide is on. Unhashed files are never extras:
// their installed group size is 0, which already fails the size check on
// its own. The single-index entry point; a walk should read visibility()
// once and call hiddenExtra directly.
func (m *Model) IsHiddenExtra(i int) bool {
	hide, groups := m.visibility()

	return hiddenExtra(hide, groups, i)
}
```

In `internal/dupes/groups.go`, extract the member scan so it can run
against an already-read snapshot:

```go
// membersOf is Members' body against an already-read Groups snapshot and
// count, so a caller that has both does not re-take the model mutex.
func membersOf(groups Groups, n, i int) []int {
	if groups.Size(i) < 2 {
		return nil
	}
	rep := groups.RepresentativeOf(i)
	var members []int
	for j := range n {
		if groups.RepresentativeOf(j) == rep {
			members = append(members, j)
		}
	}

	return members
}
```

and rewrite `Members` as a thin wrapper over it, preserving its exported
doc comment.

- [ ] **Step 4: Rewrite the navigation reads**

```go
// InspectSource is the current index of the file BeginInspect recorded,
// or -1 when inspect is off or that file is no longer in the set.
//
// A map lookup in the published Snapshot, not a scan: the arrow keys ask
// this once per step, and at 50k files a scan per keystroke was the
// dominant cost of moving through a drop with inspect on. IndexOf
// returns -1 for the empty key, which folds in the inspect-off case.
func (m *Model) InspectSource() int {
	m.mu.Lock()
	key := m.inspectKey
	m.mu.Unlock()

	return m.set.Snapshot().IndexOf(key)
}

// InspectMembers returns the inspected file's duplicate group in
// ascending index order, or nil when inspect is off or its file is gone.
func (m *Model) InspectMembers() []int {
	return m.inspectMembers(m.set.Snapshot())
}

// inspectMembers is InspectMembers against a snapshot the caller already
// holds. The inspect key and the group snapshot come out of one lock
// acquisition, so a NextVisible step pays one mutex for the whole ring
// lookup instead of one per member.
func (m *Model) inspectMembers(s Snapshot) []int {
	m.mu.Lock()
	key := m.inspectKey
	groups := m.groups
	m.mu.Unlock()

	src := s.IndexOf(key)
	if src < 0 {
		return nil
	}

	return membersOf(groups, s.Count(), src)
}

func (m *Model) NextVisible(from, delta int) int {
	s := m.set.Snapshot()
	n := s.Count()
	if n == 0 {
		return 0
	}
	if members := m.inspectMembers(s); len(members) >= 2 && delta != 0 {
		return stepInMembers(members, from, delta)
	}
	hide, groups := m.visibility()
	if !hide || delta == 0 {
		return from + delta
	}
	step := 1
	if delta < 0 {
		step = -1
	}
	i := from
	for k := 0; k < absInt(delta); k++ {
		start := i
		for {
			i = (i + step + n) % n
			if !hiddenExtra(hide, groups, i) {
				break
			}
			if i == start {
				return from
			}
		}
	}

	return i
}

func (m *Model) FirstVisible() int {
	s := m.set.Snapshot()
	hide, groups := m.visibility()
	for i := range s.Count() {
		if !hiddenExtra(hide, groups, i) {
			return i
		}
	}

	return 0
}

func (m *Model) LastVisible() int {
	s := m.set.Snapshot()
	hide, groups := m.visibility()
	for i := s.Count() - 1; i >= 0; i-- {
		if !hiddenExtra(hide, groups, i) {
			return i
		}
	}

	return 0
}

func (m *Model) VisibleIndexesExcept(current int) []int {
	s := m.set.Snapshot()
	hide, groups := m.visibility()
	var out []int
	for i := range s.Count() {
		if i != current && !hiddenExtra(hide, groups, i) {
			out = append(out, i)
		}
	}

	return out
}
```

Keep every existing doc comment on the exported methods; only add to them
where the batching is worth explaining. `NextVisible`'s numbered
branch-order comment stays exactly as it is — the order it describes is
preserved above.

`BeginInspect` also takes a snapshot now:

```go
func (m *Model) BeginInspect(i int) {
	key := m.set.Snapshot().KeyAt(i)
	m.mu.Lock()
	m.inspectKey = key
	m.mu.Unlock()
}
```

(`KeyAt` returns `""` out of range, which is exactly the clear-inspect
behaviour the old explicit bounds check produced.)

- [ ] **Step 5: Run the tests to verify they pass**

```bash
go test -run 'TestNextVisible|TestInspectSource|TestFirstVisible|TestLastVisible|TestVisibleIndexes|TestIsHiddenExtra' ./internal/dupes/
go test -race ./internal/dupes/ ./internal/ui/...
make fmt-check && go vet ./... && go build ./...
```

Expected: all PASS, including `TestNextVisible_TakesOneSnapshotPerCall`
now reporting exactly 1.

**Review gate (I check):** `NextVisible`'s three branches are in the
original order; no exported signature changed; `IsHiddenExtra` still exists
as the single-index entry point (`internal/ui/visibility.go` and
`internal/ui/grid/search.go` both call it); `m.mu` is never held across a
`Snapshot()` call.

---

## Task 8: Documentation and bookkeeping

**Files:**
- Modify: `ARCHITECTURE.md:46`, `:277-294`, `:360`
- Modify: `todos.md`
- Check: `needs_refactoring.md`

**Interfaces:**
- Consumes: the finished state of Tasks 1–7.
- Produces: nothing code-facing.

- [ ] **Step 1: Update `ARCHITECTURE.md`**

Three places:

1. Line 46 (`visibility.go` row): `dupeFileSet` now "adapts the viewer to
   `dupes.FileSet` by forwarding `appState`'s published `dupes.Snapshot`".
2. The `internal/dupes` section (from line 277): the `FileSet` interface is
   now a single `Snapshot() Snapshot`; add `snapshot.go` to the file table
   with the description *"`Snapshot`: the immutable {keys, generation,
   key→index} view every Model method reads through."*
3. Line 294 (`dupes.go` row): mention `Snapshot`-based generation reads.

Also add a line to the `internal/ui` section noting that `appState`
publishes the snapshot atomically with the generation and that
`viewer.Generation()` reads it from there.

- [ ] **Step 2: Move all five TODOs to Done**

In `todos.md`, delete the five entries under `## TODO` and add them under
`## Done` → `### What's Changed`, in the existing subsections:

- Under `#### Bugfix`:
  - "Hashing a folder no longer panics when the file set shrinks
    underneath the pool: the duplicate model reads an immutable snapshot of
    the file set instead of the live slice."
  - "A file the scanner produced no URI for no longer panics the hashing
    pass."
- Under `#### Internal`:
  - "`internal/dupes.FileSet` is a single `Snapshot()` method; every
    `Model` method takes one snapshot at entry."
  - "Arrow keys with inspect or hide-duplicates on no longer rescan the
    whole file set per step (`InspectSource` is an O(1) map lookup; the
    hidden-extra walk reads the model mutex once per call, not per
    candidate)."
  - "A new hashing pass no longer inherits the previous pass's
    hide-apply throttle floor."
  - "`applyVisibleFilter` only computes the browse representative when
    browsing."

Leave the `## not deemed worth implementing (edge cases)` section alone.

- [ ] **Step 3: Check `needs_refactoring.md`**

```bash
grep -n "FileAt\|NextVisible\|hideApply\|hostRep\|FileSet\|KeyAt" needs_refactoring.md
```

If any entry describes work these tasks completed, remove it. If none does,
change nothing and say so.

- [ ] **Step 4: Verify no stray TODO comments were added**

```bash
grep -rn "TODO\|FIXME" --include="*.go" internal/ main.go
```

Expected: no new hits (`AGENTS.md` forbids them in source).

**Review gate (I check):** `ARCHITECTURE.md` describes the code as it now
is (I diff it against the actual `internal/dupes` file list); every one of
the five `todos.md` entries is accounted for; no source `TODO` comments.

---

## Task 9: CI parity — I run this, not a subagent

- [ ] **Step 1: Full CI-matching verification**

```bash
make fmt-check
go vet ./...
go build ./...
go test -timeout 20m -race ./...
```

All four must pass from the repository root. Anything red goes back to the
owning task's subagent with the exact output.

- [ ] **Step 2: Confirm the reproducer is a real guard**

Temporarily revert `internal/ui/visibility.go`'s `Snapshot()` to the old
live `Count`/`KeyAt` shape, confirm
`TestDupeCompute_ConcurrentWithFileRemoval` goes red again under `-race`,
then restore. A regression test that cannot fail is not a regression test.

- [ ] **Step 3: Hand off**

Do **not** commit (`AGENTS.md`). Produce a suggested commit message for the
user covering all five items.

---

## Self-Review

**Spec coverage** — each `todos.md` entry maps to a task:

| TODO (priority) | Task |
|---|---|
| `viewer.FileAt` unguarded (18) | 2, 3, 4, 5, 6 |
| `hideApplyAt` never cleared (10) | 1 |
| Hashing pass nil-URI guard (25) | 1 |
| `NextVisible` O(n) (12) | 7 (with 2's `IndexOf`) |
| `applyVisibleFilter` `hostRep` (10) | 1 |

**Type consistency** — `NewSnapshot`, `Count`, `KeyAt`, `Generation`,
`IndexOf` are defined in Task 2 and used with those exact names in Tasks 3,
4, 6 and 7. `beginPass` (Task 1), `snapshot()` / `reorder()` / `publish()`
(Task 3), `wipeIfStale` (Task 6), and `visibility()` / `hiddenExtra()` /
`membersOf()` / `inspectMembers()` (Task 7) are each defined once and
referenced only after their defining task.

**Known deviations from the `todos.md` text**, both deliberate and both
argued in the Design section:
1. The `FileAt` fix is a snapshot published by `appState`, not the
   "hoist the key slice out from under the lock in `Compute`" the entry
   proposes — that proposal does not close the race.
2. `hideApplyAt` is cleared in a new `beginPass` guarded by
   `hashJobs.Load() == 0`, not unconditionally at `Run`'s top, so a second
   `Run` landing mid-pass cannot reset a live throttle floor.

**Deliberately out of scope.** `internal/ui/grid/search.go:140` calls
`IsHiddenExtra(i)` inside `applyVisibleFilter`'s loop over
`g.host.FileCount()` — the same one-mutex-per-index pattern Task 7 removes
from the navigation walks, in the grid's filter pass. It is not in
`todos.md` and fixing it means exporting a batched entry point from
`internal/dupes` (a `VisibleMask()` or similar), which is a design decision
of its own. Flag it as a new `todos.md` entry rather than folding it in
here.

**No test for the `hostRep` change.** It is a pure removal of a redundant
mutex acquisition with no observable behaviour; asserting it would mean
counting lock acquisitions through a fake model, which is more machinery
than the fix. The existing `internal/ui/grid` search and browse tests cover
that the filter still produces the right matches.
