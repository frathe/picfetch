# Fix Duplicate Grouping (Non-Transitive Stars) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Controller:** After every task, review the diff, fix issues, then start the next task. Do **not** `git commit` (`AGENTS.md`). Do **not** start the next task until the current one is accepted.
>
> Task subagents: read only your task, Global Constraints, and that task's Interfaces. Do not read this whole file. **If an identifier in this plan disagrees with a file you open, the file wins.**
>
> Execution starts only when Florian says to start the subagents.

**Goal:** Stop hide-duplicates (`D`) and browse-duplicates (`Shift+D`) from putting thousands of unrelated mixed-format photos into one group headed by the first unsorted file.

**Architecture:** The bug is in `imaging.DuplicateGroups`: it builds **connected components** (union-find / single-linkage). If A is within Hamming `maxDist` of B and B of C, A and C share a group even when they are far apart. The grid then picks the **lowest host index** as representative, so file 0 absorbs every chain. Replace that with **star clustering**: walk indices in order; each unassigned file becomes a representative; later files join it only when they are within `maxDist` of **that representative**. Hide, badges, and `Shift+D` all call `DuplicateGroups` already — change it in place; do not add a second grouping API.

**Tech Stack:** Go 1.26.7, existing `internal/imaging` dHash, `internal/ui/grid` hide/browse filters. No new packages, no hash library, no default-distance change unless Florian overrides before Task 1.

**Source todo:** `todos.md` — “bug in duplication detection”.

## Diagnosis (tester report)

Tester: large mixed-format scan → grid → `D` → **first unsorted image is assigned thousands of duplicates** → highlight it → `Shift+D` → **many different images that are not copies of that shot**.

That is exactly connected-component grouping plus `RepresentativeOf` = lowest host index:

1. `DuplicateGroups` unions every pair with `Hamming <= dupeDist` (default 10).
2. `rebuildGroups` maps the giant component onto host indices and sets `groupReps[i] = min(group)`.
3. Unsorted file 0 becomes the representative; the badge shows the component size; `Shift+D` lists the whole blob.

A secondary hypothesis (uniform dHash `0` from blank/failed thumbs) would also make a giant group, **even with star clustering**, because `Hamming(0, 0) == 0`. Locked default: **do not** special-case hash `0` in this plan. Identical solid-color images should still group. If Florian overrides, add it as a follow-up, not as part of Task 1–3.

## Locked defaults (override before Task 1)

| # | Decision |
|---|----------|
| 1 | **Star clustering** in `DuplicateGroups` (distance to representative). Not complete-linkage. Not “browse uses a different algorithm than hide”. |
| 2 | **Keep** `DuplicateMaxDistance = 10` and the Settings slider 0–32. |
| 3 | **Do not** exclude hash `0` from grouping. |
| 4 | **Do not** change `Shift+D` to “all files within `maxDist` of the highlighted file, ignoring the representative”. Browse stays “the group `RepresentativeOf(source)` owns”. After the fix, that group is a star, so file 0 no longer swallows a chain. |
| 5 | **Do not** add diagnostics, logging, or a second hash (aHash). |
| 6 | `rebuildGroups` / `dupes.go` production logic stays as-is once `DuplicateGroups` is correct. Task 2 only adds grid tests (and a one-line comment if the grouping comment is now wrong). |

## Global Constraints

- Do not `git commit`. Suggested commit messages are for the user.
- Every user-visible string is `lang.L("...")` with the same key in `translations/en.json` **and** `translations/de.json`. This plan adds **no** new `lang.L` keys.
- Feature packages talk through their own `Host`. Do not import `internal/ui` from `grid`. Do not pass `appState`. **Grid `Host` gains no methods.**
- Hash the **thumbnail**, never a second full-size decode. This bugfix does not change when hashes are recorded.
- Uniform-color JPEGs (`uitest.TempJPEGURI` with a solid color) all dHash to `0`. Grid tests that inject hashes must **not** `Warm` or `Toggle` before the assertion: `Toggle`/`requestThumbnail` would decode those white JPEGs and overwrite injected hashes with `0`. Follow `TestSetDuplicateDistance_RegroupsLive` (inject via `rememberHash` or a direct `g.hashes` write, then `SetHideDuplicates` / `SetBrowsingDuplicates`).
- Tests: TDD. No `time.Sleep`. Fyne test driver runs `fyne.Do` inline.
- Open work in `todos.md`; no `TODO`/`FIXME` in source. Do **not** mark the todo done until Florian accepts.
- English comments; match surrounding style. Verify identifiers against the files you open.
- Do not rename `DuplicateGroups`. Callers in `internal/ui/grid/dupes.go` (`rebuildGroups`) must keep compiling unchanged.
- Do not extract a new package. Do not “optimize” the O(n²) loop (thousands of files is fine).
- `AGENTS.md`: update `ARCHITECTURE.md` in the same change when behavior described there changes (Task 3).

---

## Subagent assignment

Run **strictly in order**. Do not dispatch implementers in parallel. Do not commit.

| Task | What | Type | Model |
|------|------|------|-------|
| 1 | Replace union-find `DuplicateGroups` with star clustering + imaging tests | `go-expert` | `composer-2.5-fast` |
| 2 | Grid hide/`Shift+D` regression for a Hamming chain | `go-expert` | `cursor-grok-4.6-xhigh` |
| 3 | Manual EN/DE + `ARCHITECTURE.md` grouping wording | `generalPurpose` | `composer-2.5-fast` |
| Per-task review | Spec + quality | `generalPurpose` | `claude-sonnet-5-thinking-high` |
| Parent fix-up | After each task | controller (this session) | inherit |
| Final review | Whole change | `generalPurpose` | `claude-opus-5-thinking-high` |

Task 1 is transcription-plus-tests from the code below (cheap model). Task 2 coordinates grid test harness + hide/browse flags (standard model). Task 3 is copy. The algorithm is specified; do **not** use Opus for implementation unless Task 1 reports `BLOCKED`.

---

## File map

| File | Role |
|------|------|
| `internal/imaging/dhash.go` | Change `DuplicateGroups` (and its doc comment) from union-find to star clustering |
| `internal/imaging/dhash_test.go` | Chain test (must fail on old code); separated-hashes test; keep existing tests passing |
| `internal/ui/grid/dupes_test.go` | Hide + browse tests on an injected A~B~C chain |
| `internal/ui/grid/dupes.go` | Comment only, if it still says groups are connected components (today it does not — verify) |
| `ARCHITECTURE.md` | `dhash.go` row + “How does hide-duplicates work?”: union-find → star |
| `internal/ui/help/manual.md` | One sentence: match is to the representative, not a chain |
| `internal/ui/help/manual_de.md` | Same sentence in German |

### Identifier lock (verify in the files you open)

As of this plan: `DifferenceHash`, `Hamming`, `DuplicateGroups`, `DuplicateMaxDistance`, `rebuildGroups`, `SetHideDuplicates`, `SetBrowsingDuplicates`, `IsHiddenExtra`, `RepresentativeOf`, `groupSize`, `hashOf`, `rememberHash`, `browseHost`. Package `grid` tests may write `g.hashes` / `g.hashGen` under `g.hashMu` (same package). `hostWith` builds **white** 8×8 JPEGs — do not `Warm` them if you need distinct hashes.

---

### Task 1: Star clustering in `DuplicateGroups`

**Files:**
- Modify: `internal/imaging/dhash.go` (`DuplicateGroups` and its comment)
- Test: `internal/imaging/dhash_test.go`

**Interfaces:**
- Consumes: existing `Hamming(a, b uint64) int`
- Produces: `DuplicateGroups(hashes []uint64, maxDist int) [][]int` — same signature. Each returned group is indices into `hashes`, sorted by first-seen (representative at `grp[0]`, later members in increasing index). Size-1 groups omitted. `maxDist < 0` treated as `0`. Empty input returns `nil`.

**Star rule (implement exactly this):**

```
assigned[i] = -1 for all i
for i in 0..n-1:
  if assigned[i] >= 0: continue
  grp = [i]
  assigned[i] = i
  for j in i+1..n-1:
    if assigned[j] >= 0: continue
    if Hamming(hashes[i], hashes[j]) <= maxDist:
      assigned[j] = i
      append j to grp
  if len(grp) >= 2: emit grp
```

File `i` is the representative. File `j` joins **only** if it is within `maxDist` of `i`, not of some other member.

Chain fixture used in tests (copy verbatim):

```go
const (
	chainA uint64 = 0
	chainB uint64 = 0x3FF   // bits 0–9;  Hamming(A,B)=10
	chainC uint64 = 0xFFFFF // bits 0–19; Hamming(B,C)=10, Hamming(A,C)=20
)
```

At `maxDist == 10`: old union-find returns `[[0 1 2]]`; star returns `[[0 1]]` (C is a singleton). At `maxDist == 20`: both return `[[0 1 2]]` because C is within 20 of A.

- [ ] **Step 1: Write the failing tests**

Append to `internal/imaging/dhash_test.go` (keep existing tests). Add `chainA`/`chainB`/`chainC` next to the other tests, or as unexported constants in that file:

```go
func TestDuplicateGroups_ChainIsNotTransitive(t *testing.T) {
	// A~B and B~C at distance 10, A far from C (20). Union-find would
	// emit one group of three; a star around A must keep C out.
	got := DuplicateGroups([]uint64{chainA, chainB, chainC}, 10)
	if len(got) != 1 || !slices.Equal(got[0], []int{0, 1}) {
		t.Fatalf("got %v, want [[0 1]] (C must not join A's star)", got)
	}
}

func TestDuplicateGroups_ChainJoinsWhenWithinRepDistance(t *testing.T) {
	got := DuplicateGroups([]uint64{chainA, chainB, chainC}, 20)
	if len(got) != 1 || !slices.Equal(got[0], []int{0, 1, 2}) {
		t.Fatalf("got %v, want [[0 1 2]] at maxDist 20", got)
	}
}

func TestDuplicateGroups_SeparatedHashesStaySingletons(t *testing.T) {
	hashes := []uint64{
		0x0000000000000000,
		0xFFFF000000000000,
		0x0000FFFF00000000,
		0x00000000FFFF0000,
		0x000000000000FFFF,
		0x00FF00FF00FF00FF,
		0xFF00FF00FF00FF00,
		0xF0F0F0F0F0F0F0F0,
	}
	got := DuplicateGroups(hashes, DuplicateMaxDistance)
	if len(got) != 0 {
		t.Fatalf("got %v, want no groups (each hash is far from the others)", got)
	}
}
```

Define the `chainA`/`chainB`/`chainC` constants in `dhash_test.go` only (not in `dhash.go`).

Keep `TestDuplicateGroups_ClustersByHamming` as-is: `DuplicateGroups([]uint64{0, 1, ^uint64(0)}, 2)` must still return `[[0 1]]`.

- [ ] **Step 2: Run tests to verify the chain test fails**

Run: `go test -run 'TestDuplicateGroups_ChainIsNotTransitive|TestDuplicateGroups_ClustersByHamming' ./internal/imaging/ -v`

Expected: `TestDuplicateGroups_ChainIsNotTransitive` **FAIL** with `got [[0 1 2]], want [[0 1]]` (or equivalent). `TestDuplicateGroups_ClustersByHamming` still PASS. If the chain test already passes, stop and report `BLOCKED` — the production function is not the union-find described in this plan.

- [ ] **Step 3: Replace `DuplicateGroups` with star clustering**

In `internal/imaging/dhash.go`, replace the function (including the comment that currently says “connected components”) with:

```go
// DuplicateGroups partitions indices into groups of near-duplicates.
// Each group has a representative at the lowest index; every other
// member is within maxDist Hamming distance of that representative.
// Membership is not transitive: A~B and B~C do not put A and C in the
// same group unless both are within maxDist of the representative.
// Groups of size 1 are omitted.
func DuplicateGroups(hashes []uint64, maxDist int) [][]int {
	n := len(hashes)
	if n == 0 {
		return nil
	}
	if maxDist < 0 {
		maxDist = 0
	}

	assigned := make([]int, n)
	for i := range assigned {
		assigned[i] = -1
	}

	var out [][]int
	for i := range n {
		if assigned[i] >= 0 {
			continue
		}
		grp := []int{i}
		assigned[i] = i
		for j := i + 1; j < n; j++ {
			if assigned[j] >= 0 {
				continue
			}
			if Hamming(hashes[i], hashes[j]) <= maxDist {
				assigned[j] = i
				grp = append(grp, j)
			}
		}
		if len(grp) >= 2 {
			out = append(out, grp)
		}
	}
	return out
}
```

Delete the old `parent` / `find` / `union` / `buckets` code. Do not leave a unused helper.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -race ./internal/imaging/`

Expected: PASS, including the three new tests and `TestDuplicateGroups_ClustersByHamming`, `TestDuplicateGroups_SingletonsOmitted`, `TestDuplicateGroups_NegativeMaxDistTreatedAsZero`.

- [ ] **Step 5: Suggested commit (do not commit)**

`imaging: group duplicates by distance to representative, not transitive chains`

---

### Task 2: Grid hide and browse must not swallow the chain

**Files:**
- Test: `internal/ui/grid/dupes_test.go`
- Modify only if a comment in `internal/ui/grid/dupes.go` still describes connected components (as of this plan it does not — do not add a comment for its own sake)

**Interfaces:**
- Consumes: Task 1's `DuplicateGroups` star semantics; `Overview.rebuildGroups` via `SetHideDuplicates` / `SetBrowsingDuplicates`
- Produces: two tests that lock the tester's `D` + `Shift+D` path on a three-file chain

`rebuildGroups` already does:

```go
groups := imaging.DuplicateGroups(hs, dist)
```

Do **not** reimplement grouping in the grid. If Task 2 tests fail after Task 1, the bug is in how `rebuildGroups` maps `grp` indices through `idx` — fix that mapping, do not fork a second algorithm.

Inject hashes directly (package `grid` tests). Do **not** `Warm` or `Toggle`. `hostWith` JPEGs are solid white.

Helper to add in `dupes_test.go`:

```go
func injectHashes(t *testing.T, g *Overview, host *fakeHost, hs []uint64) {
	t.Helper()
	if len(hs) != len(host.files) {
		t.Fatalf("injectHashes: %d hashes for %d files", len(hs), len(host.files))
	}
	g.hashMu.Lock()
	defer g.hashMu.Unlock()
	if g.hashes == nil {
		g.hashes = make(map[string]uint64)
	}
	g.hashGen = host.Generation()
	for i, h := range hs {
		g.hashes[host.files[i].String()] = h
	}
}
```

Use the same bit patterns as Task 1 (`0`, `0x3FF`, `0xFFFFF`). Importing unexported `chainA` from `imaging` is impossible; **repeat the three literals** in the grid test (do not export them from `imaging`).

- [ ] **Step 1: Write the failing tests**

Append to `internal/ui/grid/dupes_test.go`:

```go
func TestSetHideDuplicates_ChainDoesNotHideUnrelated(t *testing.T) {
	host := hostWith(t, "a.jpg", "b.jpg", "c.jpg")
	g := newOverview(t, host)
	injectHashes(t, g, host, []uint64{0, 0x3FF, 0xFFFFF})
	g.SetDuplicateDistance(imaging.DuplicateMaxDistance)

	g.SetHideDuplicates(true)

	if g.count() != 2 {
		t.Fatalf("count() = %d, want 2 (A visible, B hidden extra, C unique)", g.count())
	}
	if !g.IsHiddenExtra(1) {
		t.Error("B is within distance 10 of A and must be an extra")
	}
	if g.IsHiddenExtra(2) {
		t.Error("C is Hamming 20 from A and must not be hidden as A's extra")
	}
	if g.RepresentativeOf(2) != 2 {
		t.Errorf("RepresentativeOf(2) = %d, want 2", g.RepresentativeOf(2))
	}
	if g.groupSize(0) != 2 {
		t.Errorf("groupSize(0) = %d, want 2 (not the whole set)", g.groupSize(0))
	}
}

func TestSetBrowsingDuplicates_ChainDoesNotListUnrelated(t *testing.T) {
	host := hostWith(t, "a.jpg", "b.jpg", "c.jpg")
	host.index = 0
	g := newOverview(t, host)
	injectHashes(t, g, host, []uint64{0, 0x3FF, 0xFFFFF})
	g.SetDuplicateDistance(imaging.DuplicateMaxDistance)

	g.SetBrowsingDuplicates(true)

	if !g.BrowsingDuplicates() {
		t.Fatal("A has a duplicate (B); browse must turn on")
	}
	if g.count() != 2 {
		t.Fatalf("count() = %d, want 2 (A and B, not C)", g.count())
	}
	seen := map[int]bool{g.fileIndex(0): true, g.fileIndex(1): true}
	if !seen[0] || !seen[1] || seen[2] {
		t.Fatalf("visible hosts = %v, want 0 and 1 only", seen)
	}
}
```

If `groupSize` is unexported, the first test is in package `grid` and **can** call it (it is unexported on `Overview` in the same package).

- [ ] **Step 2: Run tests to verify they fail on old grouping, pass on Task 1**

Run: `go test -run 'TestSetHideDuplicates_ChainDoesNotHideUnrelated|TestSetBrowsingDuplicates_ChainDoesNotListUnrelated' ./internal/ui/grid/ -v`

Expected after Task 1: PASS. Expected if someone runs this task against union-find `DuplicateGroups`: FAIL (`count() = 1` on hide, `count() = 3` on browse, or `IsHiddenExtra(2) == true`).

If hide count is 1 (everything swallowed) after Task 1: `rebuildGroups` is mapping wrong — inspect `idx[grp[i]]` vs treating `grp` values as host indices. Fix `rebuildGroups` only if that bug is real; the current code uses `idx[gi]` and should be correct.

- [ ] **Step 3: Run the existing dupe tests so the chain tests did not break pairs**

Run: `go test -race -run 'TestSetHideDuplicates_|TestSetBrowsingDuplicates_|TestWarm_RecordsDifferenceHash|TestSetDuplicateDistance_RegroupsLive' ./internal/ui/grid/`

Expected: PASS. `pairAndUnique` (identical patterned seeds) must still hide one extra and browse a group of 2.

- [ ] **Step 4: Suggested commit (do not commit)**

`grid: lock hide/browse against transitive duplicate chains`

---

### Task 3: Docs — representative match, not a chain

**Files:**
- Modify: `ARCHITECTURE.md` (dhash row ~line 161; hide-duplicates index ~line 640)
- Modify: `internal/ui/help/manual.md` (hide-duplicates bullet)
- Modify: `internal/ui/help/manual_de.md` (same bullet)

**Interfaces:**
- Consumes: Task 1 semantics (star / distance to representative)
- Produces: docs that no longer say “connected components” / “union-find grouping”

Do **not** add translation keys. Do **not** mark `todos.md` done.

- [ ] **Step 1: `ARCHITECTURE.md`**

In the `dhash.go` table row, replace “union-find grouping by Hamming distance” with wording equivalent to:

> star grouping by Hamming distance to the lowest-index representative (not transitive connected components)

In the “How does hide-duplicates work?” index line, keep the file pointers; add that groups are stars around the representative, not union-find components.

Do not rewrite the rest of those paragraphs.

- [ ] **Step 2: English manual**

In `internal/ui/help/manual.md`, in the `D` hide-duplicates bullet, after the sentence that names the Settings slider (0–32, default 10), insert:

```markdown
  Two files count as copies of the same shot when each is close enough to
  the group's representative (the earliest file in the current order). A
  chain of similar-looking photos does not merge into one giant group.
```

Keep the rest of the bullet. Do not claim `D` shows only duplicates. Do not mention union-find, Hamming, or `DuplicateGroups` in the user manual.

- [ ] **Step 3: German manual**

In `internal/ui/help/manual_de.md`, in the matching `D` bullet, after the slider sentence, insert:

```markdown
  Zwei Dateien gelten als Kopien derselben Aufnahme, wenn jede dem
  Vertreter der Gruppe (der frühesten Datei in der aktuellen Reihenfolge)
  nahe genug ist. Eine Kette ähnlich aussehender Fotos wird nicht zu einer
  Riesengruppe zusammengefasst.
```

- [ ] **Step 4: Suggested commit (do not commit)**

`docs: describe duplicate groups as distance to representative`

---

## Out of scope

- Lowering `DuplicateMaxDistance` or changing the Settings slider range.
- Excluding dHash `0` (uniform images) from grouping.
- A second perceptual hash, full-size hashing, or format-specific hash tweaks.
- Making `Shift+D` query “neighbors of the highlighted file” instead of the representative's group.
- Performance work on the O(n²) loop.
- Marking the `todos.md` bug done.
- Commits.

## Verification (controller, after all tasks)

From the repository root, matching CI:

```bash
make fmt
make vet
go build ./...
go test -race ./internal/imaging/ ./internal/ui/grid/ ./internal/ui/
```

Then, if those pass: `go test -race ./...`

Do not regenerate goldens (`make golden`) — this change is grouping logic and copy, not pixels.

## Self-review (plan vs todo)

| Tester / requirement | Task |
|----------------------|------|
| First unsorted file assigned thousands of extras | Task 1 (no transitive blob); Task 2 hide `groupSize(0)==2` |
| `Shift+D` shows unrelated images | Task 2 browse count 2, C absent |
| Existing near-dupe pairs still group | Task 1 existing Hamming-1 test; Task 2 existing `pairAndUnique` tests |
| Docs match behavior | Task 3 |
| Hash-0 / default-distance / browse-by-source | Out of scope unless Florian overrides before Task 1 |
