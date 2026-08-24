# Grid Duplicate Finder Implementation Plan

> **Controller:** After every task, review the diff, fix issues, then start the next task. Do **not** `git commit` (`AGENTS.md`). Do **not** start the next task until the current one is accepted.
>
> Task subagents: read only your task, Global Constraints, and that task's Interfaces. Do not read this whole file.

**Goal:** Find visually similar shots with a dHash of already-decoded thumbnails. `D` **hides extras** (keeps one representative per group plus all uniques). Remaining grid cells show a **group-size badge**. Match tightness is a **Settings slider**. Viewer `D` and arrow keys skip hidden extras. Do **not** implement folder-cull (#1) or pick/reject (#2).

**Architecture:** Hashing stays in `internal/imaging`. The grid owns a URI-keyed hash map (not `ByteCache`). Hide-dupes is a second `applyFilter` predicate composed with `/` search. Distance is a setter on `Overview` (same shape as `SetCacheBytes`). `settingswin.Host` grows two methods. **`keys.go` handles `KeyD` when the grid is not visible.** Grid `Host` gets **no** new methods.

**Tech:** Go 1.26.7, Fyne v2.8, `internal/ui/grid`, `internal/imaging`, `internal/decodepool`. No hash library.

**Source todo:** `todos.md` — “Duplicate finder (perceptual hash) in the grid”.

## Confirmed (Florian, 2026-08-23)

| # | Decision |
|---|----------|
| 1 | Hamming **0–32**, default **10**, Settings **slider**, persist (0 is valid → `DuplicateDistanceSet` bool, same idea as `WindowPositionSet`). |
| 2 | **`D` hides extras.** Viewer: navigation shows unique shots. Grid: extras hidden; remaining cells badge **group size**. Not a duplicates-only gallery. |
| 3 | Escape: selection → search → **hide-dupes off** → close. |
| 4 | `D` while grid is up and not searching. While searching, `d`/`D` is a filename character. Viewer `D` when grid is closed. |
| 5 | Hash on every successful thumbnail decode / `Warm`. When `D` turns on, enqueue remaining unhashed files on the existing pool. Do not full-scan just because the grid opened. |
| 6 | Badge = group size (`"2"`, `"3"`, …). Shown only while hide-dupes is on and size ≥ 2. |
| 7 | #1 / #2 out of scope. |

**Hide semantics:** Keep the full file list on `appState`. Representative = lowest host index in the group. Uniques always visible. If the current image is an extra when `D` turns on, `ShowImage` the representative. `StepImage` / Home / End / `Advance` skip extras while hide is on. `Close` does **not** clear hide (viewer still hiding). `G` closes the grid; hide stays. Hashes survive `Close`/`Toggle`; wipe when `Generation()` changes.

## Global Constraints

- Do not `git commit`. Suggested commit messages are for the user.
- Every user-visible string is `lang.L("...")` with the same key in `translations/en.json` **and** `translations/de.json`. Guard: `TestTranslations_EveryLocaleCoversEnglish` in `main_test.go`.
- Feature packages talk through their own `Host`. Do not import `internal/ui` from `grid`. Do not pass `appState`. **Grid `Host` gains no methods.** `settingswin.Host` gains `DuplicateDistance() int` and `SetDuplicateDistance(int)`.
- Cross-feature decisions stay in `internal/ui`. Grid-local `D` / Escape staging stay in `grid.HandleKey`. **`keys.go` must handle `KeyD` when the grid is not visible** (before the navigation-length guard, same as `G`). While the grid is visible, keys already go to `grid.HandleKey` — do not duplicate `KeyD` there for the grid-visible case.
- Hash the **thumbnail** (`imaging.LoadThumbnail` result), never a second full-size decode.
- Store hashes in a URI-keyed map on `Overview`, **not** in the thumbnail `ByteCache`.
- Extra background work goes through `g.decodes.Go` so `Settle` / test `drain` wait. No second pool. No `time.Sleep`. Background hash jobs: `Go` **without** `Claim`; do **not** `thumbs.Add` if `ThumbCacheFull()`.
- `applyFilter`: `matches == nil` is identity; a non-nil empty slice is “nothing matches”. Hide-dupes uses that convention (list of visible host indices).
- Uniform-color JPEGs (`uitest.TempJPEGURI` with a solid color) all dHash to `0`. Grid tests must use **patterned** JPEGs.
- Tests: TDD. No `time.Sleep`. Use `Warm` / `Settle` / existing harness. Fyne test driver runs `fyne.Do` inline.
- Open work in `todos.md`; no `TODO`/`FIXME` in source. Do not mark the todo done until Florian accepts.
- English comments; match surrounding style. Use the **real identifiers in the files you open** (`fileIndex`, `applyFilter`, `filterGen`, `HandleKey`, `HandleRune`, `requestThumbnail`, `Settle`, `Warm`, `LoadThumbnail`, `ThumbCacheFull`, `SetCacheBytes`).
- Verify identifiers against the files you open. Do not invent grid `Host` methods.

## Subagent assignment

Run **strictly in order**. Do not dispatch implementers in parallel. Do not commit.

| Task | What | Type | Model |
|------|------|------|-------|
| 1 | imaging dHash + Hamming + grouping | `go-expert` | `composer-2.5-fast` |
| 2 | Hash-on-decode URI map | `go-expert` | `cursor-grok-4.6-xhigh` |
| 3 | Hide-dupes filter + `D` + viewer skip | `go-expert` | `cursor-grok-4.6-xhigh` |
| 4 | Cell badge overlay | `go-expert` | `claude-sonnet-5-thinking-high` |
| 5 | Hash remaining when D turns on | `go-expert` | `cursor-grok-4.6-xhigh` |
| 6 | Settings slider + preferences | `go-expert` | `cursor-grok-4.6-xhigh` |
| 7 | Manual EN/DE + `ARCHITECTURE.md` | `generalPurpose` | `composer-2.5-fast` |
| Per-task review | Spec + quality | `generalPurpose` | `claude-sonnet-5-thinking-high` |
| Final review | Whole change | `generalPurpose` | `claude-opus-5-thinking-high` |

## File map

| File | Role |
|------|------|
| `internal/imaging/dhash.go` | Create. `DifferenceHash`, `Hamming`, `DuplicateMaxDistance`, `DuplicateGroups` |
| `internal/imaging/dhash_test.go` | Create. Patterned-image tests |
| `internal/ui/grid/dupes.go` | Create. Hash map, groups, hide flag, `hashRemaining` |
| `internal/ui/grid/dupes_test.go` | Create |
| `internal/ui/grid/thumbs.go` | `rememberHash` after decode / cache hit / `Warm` |
| `internal/ui/grid/search.go` | `applyFilter` intersection |
| `internal/ui/grid/nav.go` | `KeyD`; Escape staging |
| `internal/ui/grid/grid.go` | Fields; cell stack + badge (Task 4) |
| `internal/ui/keys.go` | `KeyD` when grid not visible |
| `internal/ui/viewer.go` | `StepImage` / `Advance` / Home / End skip extras |
| `internal/ui/settingswin/` | Slider |
| `internal/preferences/` | Persist distance |
| `translations/en.json`, `de.json` | New keys |
| `internal/ui/help/manual.md`, `manual_de.md` | Docs |
| `ARCHITECTURE.md` | imaging dHash; grid hide-dupes |

---

### Task 1: imaging dHash, Hamming, grouping

**Subagent:** `go-expert` @ `composer-2.5-fast`

**Files:** Create `internal/imaging/dhash.go`, `internal/imaging/dhash_test.go`.

**Interfaces:**

```go
const DuplicateMaxDistance = 10 // default Hamming threshold (slider default)

func DifferenceHash(img image.Image) uint64
func Hamming(a, b uint64) int
func DuplicateGroups(hashes []uint64, maxDist int) [][]int
```

`DuplicateGroups`: indices `i,j` same group iff `Hamming(hashes[i], hashes[j]) <= maxDist` (union-find). **Omit groups of size 1.** Order of groups and of indices inside a group is first appearance in `hashes`. `maxDist < 0` is treated as `0`. Callers pass only known hashes — no unset sentinels.

Use `golang.org/x/image/draw` (already imported by `thumbnail.go`).

- [ ] **Step 1: Write failing tests** in `dhash_test.go`:

```go
package imaging

import (
	"image"
	"image/color"
	"slices"
	"testing"
)

func patterned(w, h, seed int) *image.Gray {
	im := image.NewGray(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			im.SetGray(x, y, color.Gray{Y: uint8((x*13 + y*7 + seed*31) & 0xff)})
		}
	}
	return im
}

func TestDifferenceHash_IdenticalImagesHaveDistanceZero(t *testing.T) {
	a := patterned(64, 48, 1)
	if Hamming(DifferenceHash(a), DifferenceHash(a)) != 0 {
		t.Fatal("identical images must hash with Hamming 0")
	}
}

func TestDifferenceHash_DifferentPatternsAreFar(t *testing.T) {
	a, b := patterned(64, 48, 1), patterned(64, 48, 99)
	if d := Hamming(DifferenceHash(a), DifferenceHash(b)); d <= DuplicateMaxDistance {
		t.Fatalf("Hamming(different) = %d, want > %d", d, DuplicateMaxDistance)
	}
}

func TestDifferenceHash_UniformImagesHashToZero(t *testing.T) {
	white := image.NewGray(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			white.SetGray(x, y, color.Gray{Y: 255})
		}
	}
	black := image.NewGray(image.Rect(0, 0, 32, 32))
	if DifferenceHash(white) != 0 || DifferenceHash(black) != 0 {
		t.Fatal("uniform images must dHash to 0")
	}
}

func TestHamming_KnownBits(t *testing.T) {
	if Hamming(0, 0) != 0 || Hamming(0, 1) != 1 || Hamming(0, ^uint64(0)) != 64 {
		t.Fatal("Hamming known bits")
	}
}

func TestDuplicateGroups_ClustersByHamming(t *testing.T) {
	got := DuplicateGroups([]uint64{0, 1, ^uint64(0)}, 2)
	if len(got) != 1 || !slices.Equal(got[0], []int{0, 1}) {
		t.Fatalf("got %v, want [[0 1]]", got)
	}
}

func TestDuplicateGroups_SingletonsOmitted(t *testing.T) {
	if got := DuplicateGroups([]uint64{0, ^uint64(0)}, 0); len(got) != 0 {
		t.Fatalf("got %v, want no groups", got)
	}
}

func TestDuplicateGroups_NegativeMaxDistTreatedAsZero(t *testing.T) {
	if got := DuplicateGroups([]uint64{0, 1}, -3); len(got) != 0 {
		t.Fatalf("got %v, want no groups at exact-only", got)
	}
}
```

- [ ] **Step 2: Run tests — expect FAIL** (undefined symbols)

```bash
go test -count=1 ./internal/imaging/ -run 'TestDifferenceHash|TestHamming|TestDuplicateGroups' -v
```

- [ ] **Step 3: Implement** `dhash.go`:

```go
package imaging

import (
	"image"
	"image/color"
	"math/bits"

	"golang.org/x/image/draw"
)

// DuplicateMaxDistance is the default Hamming threshold at or below which
// two dHashes count as the same shot. 10 is the usual dHash near-duplicate
// cutoff: bursts and re-exports match, unrelated photos do not. The
// settings slider may pass a different maxDist into DuplicateGroups.
const DuplicateMaxDistance = 10

const dhashWide, dhashHigh = 9, 8

// DifferenceHash is a 64-bit dHash of img: luma resized to 9×8, then one
// bit per adjacent horizontal pair (8 rows × 8 comparisons). Uniform
// images have no horizontal gradient and hash to 0, so two different
// solid colors collide — callers that need “not a duplicate” fixtures
// must use patterned pixels, not solid JPEGs.
func DifferenceHash(img image.Image) uint64 {
	dst := image.NewGray(image.Rect(0, 0, dhashWide, dhashHigh))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Src, nil)

	var h uint64
	var bit uint64
	for y := 0; y < dhashHigh; y++ {
		for x := 0; x < dhashWide-1; x++ {
			left, _ := color.GrayModel.Convert(dst.At(x, y)).(color.Gray)
			right, _ := color.GrayModel.Convert(dst.At(x+1, y)).(color.Gray)
			if right.Y > left.Y {
				h |= 1 << bit
			}
			bit++
		}
	}
	return h
}

// Hamming is the number of bits that differ between two dHashes.
func Hamming(a, b uint64) int {
	return bits.OnesCount64(a ^ b)
}

// DuplicateGroups returns connected components of indices whose hashes
// are within maxDist Hamming distance. Groups of size 1 are omitted.
func DuplicateGroups(hashes []uint64, maxDist int) [][]int {
	n := len(hashes)
	if n == 0 {
		return nil
	}
	if maxDist < 0 {
		maxDist = 0
	}
	parent := make([]int, n)
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		if parent[i] != i {
			parent[i] = find(parent[i])
		}
		return parent[i]
	}
	union := func(a, b int) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[rb] = ra
		}
	}
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if Hamming(hashes[i], hashes[j]) <= maxDist {
				union(i, j)
			}
		}
	}
	buckets := make(map[int][]int, n)
	order := make([]int, 0, n)
	for i := 0; i < n; i++ {
		r := find(i)
		if _, ok := buckets[r]; !ok {
			order = append(order, r)
		}
		buckets[r] = append(buckets[r], i)
	}
	var out [][]int
	for _, r := range order {
		if len(buckets[r]) >= 2 {
			out = append(out, buckets[r])
		}
	}
	return out
}
```

- [ ] **Step 4: Run tests — expect PASS**

```bash
go test -count=1 ./internal/imaging/ -run 'TestDifferenceHash|TestHamming|TestDuplicateGroups' -v
```

- [ ] **Step 5: Suggested commit (do not commit):** `imaging: add dHash, Hamming distance, and duplicate grouping`

---

### Task 2: Record hashes on thumbnail decode

**Subagent:** `go-expert` @ `cursor-grok-4.6-xhigh`

**Files:** Create `dupes.go`. Modify `grid.go` (fields), `thumbs.go` (`Warm`, `requestThumbnail`), `thumbs_test.go`.

**Interfaces:** On `Overview`:

```go
func (g *Overview) rememberHash(u fyne.URI, img image.Image)
func (g *Overview) hashOf(u fyne.URI) (uint64, bool)
func (g *Overview) wipeHashesIfStale()
```

URI-keyed `map[string]uint64` plus mutex. `rememberHash` is worker-safe. Wipe when `host.Generation()` changes. After successful `LoadThumbnail` **and** on cache hit if `hashOf` misses, `rememberHash`. `Warm` hashes too. **No hide-dupes, no badges, no `hashRemaining` in this task.**

- [ ] Tests: `TestWarm_RecordsDifferenceHash`, `TestRequestThumbnail_CacheHitFillsMissingHash`. Patterned JPEGs not required if you only assert presence. Fail then implement. `go test -count=1 ./internal/ui/grid/ -run 'TestWarm_|TestRequestThumbnail_' -v`

- [ ] Suggested commit: `grid: record dHashes when thumbnails decode`

---

### Task 3: Hide extras, D key, viewer skip, search intersection

**Subagent:** `go-expert` @ `cursor-grok-4.6-xhigh`

**Files:** `dupes.go`, `dupes_test.go`, `search.go` (`applyFilter`, top bar), `nav.go` (`HandleKey`, `escape`), `grid.go` (`Close` must **not** clear hide), `keys.go` (`KeyD` when grid not visible), `viewer.go` (`StepImage`, `Advance`, Home/End), translations, grid harness patterned JPEG helper.

**Interfaces:**

```go
func (g *Overview) HideDuplicates() bool
func (g *Overview) SetHideDuplicates(on bool)
func (g *Overview) SetDuplicateDistance(n int) // clamp 0–32; default DuplicateMaxDistance until Task 6
func (g *Overview) IsHiddenExtra(hostIndex int) bool
func (g *Overview) RepresentativeOf(hostIndex int) int
func (g *Overview) groupSize(hostIndex int) int // 0 unhashed, 1 unique, ≥2 dupes
```

**Filter:** Representative = lowest host index in each `DuplicateGroups` component. Unhashed files are visible (not extras). `applyFilter` keeps indices that pass the name check (if searching with a query) **and** `!IsHiddenExtra(i)` (if hide on). Neither → `matches = nil`.

**Keys:** Grid `HandleKey` `KeyD` when `!searching` toggles hide. `escape`: selection → search → hide off → `Close`. Viewer `keys.go` `KeyD` when grid not visible: toggle hide on the grid object; if current file is extra, `ShowImage(RepresentativeOf)`. `StepImage` / Home / End / `Advance` skip extras while hide is on (wrap; no-op if every file is extra — should not happen).

**Chrome:** hide on and not searching → `lang.L("Hiding duplicates")`. Searching keeps `Search: %s`. Empty: hide+search with no matches keeps existing empty copy. Do **not** use “No duplicate images” (that was the old show-only-dupes empty state).

Keys: `"Hiding duplicates"` → DE `"Duplikate ausblenden"`.

- [ ] Tests (patterned JPEGs): hide pair+unique → count 2 (one of pair + unique). Search intersection. `HandleKey` `KeyD` toggles; `HandleRune` while searching does not. Escape staging. Viewer `StepImage` skips extra. `Close` leaves hide on.

- [ ] Suggested commit: `grid: hide perceptual-duplicate extras with D`

---

### Task 4: Group-size badge on remaining cells

**Subagent:** `go-expert` @ `claude-sonnet-5-thinking-high`

**Files:** `grid.go` cell factory + update. Grep `Objects[` in `internal/ui/grid` and update **every** unpack.

Cell stack **four** objects, in order: image, tint, **`*canvas.Text` badge**, ring. After this task `Objects[3]` is the ring. Badge text = `strconv.Itoa(n)` when hide is on and `groupSize >= 2`, else `Hide()`. `TextSize` ~12, trailing/top, `theme.Color(theme.ColorNamePrimary)`.

- [ ] Test: hide on, duplicate representative shows `"2"`; unique hides badge; hide off hides badge.

- [ ] Suggested commit: `grid: badge remaining cells with duplicate group size`

---

### Task 5: Hash remaining files when hide turns on

**Subagent:** `go-expert` @ `cursor-grok-4.6-xhigh`

`hashRemaining()` from `SetHideDuplicates(true)` after setting the flag. Skip already hashed. Cache hit → `rememberHash` on UI thread. Else `decodes.Go` **without** `Claim`. If `!acquired` or generation stale, return. `LoadThumbnail`; `rememberHash`; if `!ThumbCacheFull()` then `AddIfFits` (never `Add`). `fyne.Do`: if still hide on and generation matches, rebuild groups + `applyFilter`. Dedup in-flight URIs with `sync.Map`. Do not start from `Toggle` open unless hide is on.

- [ ] Test: three patterned files, **no** `Warm`, toggle hide, `Settle`, count == 2.

- [ ] Suggested commit: `grid: hash remaining files when hide-duplicates turns on`

---

### Task 6: Settings slider + preferences

**Subagent:** `go-expert` @ `cursor-grok-4.6-xhigh`

**Files:** `internal/preferences`, `internal/ui/settingswin`, viewer getters/setters (`memlimits.go` / `settings` struct / `currentPreferences` / startup), `Overview.SetDuplicateDistance`, tests, translations. Bump settings window height if the slider does not fit.

Range **0–32**, step **1**, default **10**. Live label. `OnChanged` → `SetDuplicateDistance`. Persist with `DuplicateDistance` + `DuplicateDistanceSet` (0 is valid). Load: if not set, use `imaging.DuplicateMaxDistance`. Live rebuild groups if hide is on; if current image becomes extra, jump to representative.

Keys: `"Duplicate match distance"` → DE `"Duplikat-Erkennungsabstand"`. Hint: lower is stricter; 0 is exact thumbnail hash.

- [ ] Tests: persist 0; default 10 when unset; slider calls setter; live regroup.

- [ ] Suggested commit: `settings: add duplicate match-distance slider`

---

### Task 7: Manual, ARCHITECTURE.md, translation parity

**Subagent:** `generalPurpose` @ `composer-2.5-fast`

Docs must describe **hide extras**, badges on remaining cells, Settings slider, Escape order (selection, search, hide-dupes, grid), viewer `D`, `/` intersection. Do **not** say `D` shows only duplicates. Do **not** mark `todos.md` done.

- [ ] `go test -count=1 ./internal/ui/help/ -run Manual` and `go test -count=1 ./ -run TestTranslations`

- [ ] Suggested commit: `docs: describe hide-duplicates (D) and match-distance setting`

---

## Parent review checklist

- Identifiers match the files you open.
- No second full-size decode. No hashes in `ByteCache`. No grid `Host` growth.
- `keys.go` has viewer `KeyD`. No `time.Sleep`. No #1/#2.
- Cell stack consistent after Task 4. Extra `decodes.Go` jobs reach `Settle`.
- Both translation JSON files updated.

## Suggested overall commit (user)

```
grid: hide visual duplicates with thumbnail dHash, badges, and a match-distance slider
```
