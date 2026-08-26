# Hide-duplicates highest-resolution representative Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Controller extra (this session):** After every task, the parent agent reviews the diff and fixes it before dispatching the next task. Do not start Task N+1 until that review lands. Do not commit (`AGENTS.md`). End with a suggested commit message for the user. Do not dispatch Task 1 until Florian has confirmed the locked product decisions (or explicitly said to proceed with the defaults).

**Goal:** While hide-duplicates is on, each duplicate group keeps the highest-resolution member visible (grid cell, viewer jump, Left/Right skip) instead of the earliest file in the current sort order.

**Architecture:** Keep grouping in `imaging.DuplicateGroups` (dHash + complete linkage). Change only how `grid.computeDuplicateGroups` picks `groupReps`: max EXIF-oriented pixel count (`width*height` from `ReadAndProbe` bounds), tie-break lowest host index. Store native pixel counts next to dHashes on `Overview` (same mutex/generation), filled from `LoadThumbnailAndBounds` and backfilled by `hashRemaining` for cache-hit / favorites-prewarm thumbs that were hashed without a size.

**Tech Stack:** Go, existing `internal/imaging` probe/thumbnail pipeline, `internal/ui/grid` hide-duplicates, `internal/ui` `StepImage` / `IsHiddenExtra` (already skip non-representatives — no viewer algorithm change).

## Why this approach

Today the visible stand-in for a group is **the lowest host index** after the current sort. Re-saving a shot at a smaller size, or sorting by name, can hide the file the user actually wants to look at.

Thumbnail bounds are useless: every generated thumb is capped at `imaging.ThumbnailSize` (200px long edge). Native size is already computed by `ReadAndProbe` inside `LoadThumbnail` and then thrown away. Plumb that rectangle out, remember `Dx()*Dy()` per URI, pick the max.

Alternatives rejected:

- **Longest edge as “resolution”.** A 5000×2000 (10 MP) would beat a 4000×3000 (12 MP). “Highest resolution” means pixel count.
- **File size as a proxy.** A heavily compressed original can be smaller on disk than a lightly compressed downscale.
- **Full-image decode just for size.** `ReadAndProbe` already has oriented header bounds; decoding pixels is what `LoadThumbnail` already does when hashing from a miss.
- **Changing `imaging.DuplicateGroups` to take sizes.** Grouping and “which member is shown” are different jobs. Tests and the Hamming slider stay on hashes only.
- **Doing the variants-loop todo in this branch.** That is a navigation-mode change (arrow wrap + Esc stack). This branch only changes *which file is the hidden extra*. Return-from-browse already `ShowImage`s the highlighted cell and does **not** call `jumpIfHiddenExtra`.

## Locked product decisions

These are the defaults. Change them only if Florian says so before Task 1.

1. **Metric.** Highest resolution = largest **oriented pixel count** `bounds.Dx() * bounds.Dy()` from `ReadAndProbe` (EXIF orientation 5–8 already swaps width/height). Not long edge, not file bytes, not thumbnail size.
2. **Tie-break.** Equal pixel count → **lowest host index** (today’s rule). Existing same-size fixtures (`PatternedJPEGURI` is always 64×48) keep passing without edits.
3. **Unknown size.** Missing pixel count is `0`. A known-size member always beats an unknown one. If every member is unknown, lowest host index (today).
4. **RAW.** Use the **embedded preview** dimensions `ReadAndProbe` already returns, not sensor megapixels. That is what the viewer displays (`LoadedImage.Preview`).
5. **SVG.** Use the logical / viewBox size `ReadAndProbe` already returns. Two SVGs hashing as dupes is exotic; still apply the same max-pixels rule.
6. **When the winner can change.** As `hashRemaining` jobs land, groups rebuild on the existing coalesced apply path (`jumpIfHiddenExtra` included). A later-hashed larger file may replace the visible representative and jump the viewer off a now-hidden extra. That is desired.
7. **Sort.** Reordering the set does **not** change the winner except via the equal-pixels tie-break (host indices move). Pixel counts are per URI.
8. **Browse / Return.** `OnSelected` still closes the grid and `ShowImage`s the clicked host index, even if that file is a hidden extra. Do **not** jump to the representative on Return/click. The next todo (variant arrow loop + Esc back to the variants grid) stays **out of scope**.
9. **`D` while parked on an extra.** `jumpIfHiddenExtra` still jumps to the representative — now the highest-res member.
10. **No new `lang.L` keys.** Manual EN/DE wording only.
11. **No golden screenshot regeneration.**
12. **Out of scope:** Shift+D variant-loop / Esc-to-variants-grid todo; Windows Ctrl+click; changing Hamming grouping; probing on the UI goroutine.

## Open points (answer before Task 1)

If Florian does not answer, the locked decisions above govern.

1. **Scope.** This plan implements only `todos.md` “if duplicates are hidden, the highest resolution image should be shown.” The following item (Return shows the selected variant; arrows loop variants; Esc → variants grid → normal grid) is a **separate** plan. Confirm, or say to fold it in (that is a different, larger design).
2. **Metric.** Confirm pixel count vs longest edge vs “I meant file size”.
3. **RAW.** Confirm preview pixels (default) vs “leave RAW groups on lowest-index until we have sensor size” (would need new imaging work; do not invent it here).

## Global Constraints

- Do not commit. `AGENTS.md`: “Do not run `git commit`. End with a suggested commit message for the user.”
- Do not add `TODO`/`FIXME` comments. Open work stays in `todos.md`.
- Every user-visible string is `lang.L("English text")` with the same key in every `translations/*.json` bundle. This feature adds none.
- Do not pass `appState` into `internal/ui/grid`. Pixel maps live on `Overview`.
- `g.ui.Do` stays inside the `g.decodes.Go` body that owns it. Do not probe (`ReadAndProbe` / `LoadThumbnailAndBounds`) on the UI goroutine — `requestThumbnail` cache-hits run there today and may only `rememberHash` a cached thumb.
- `hashRemaining` must backfill **missing pixel counts even when a dHash already exists**. Favorites pre-warm (`StoreThumb`) plus a cache-hit `rememberHash` would otherwise leave every size at 0 and keep today’s lowest-index winner.
- Drive grid tests with `Warm` / `Settle` / existing `parkDecodes`. Drive viewer tests with `dropAndWait` / `waitUntilLoaded`. Never `time.Sleep` to guess completion.
- Preserve `gofmt` / `goimports -local github.com/frathe/picfetch`. Tabs, not spaces.
- Subagents must not start Task N+1 themselves. They stop after their task’s verification and report.
- Do not change `imaging.DuplicateGroups` behavior. You may clarify its comment that `grp[0]` is lowest **hashes-slice** index, not the grid’s visible representative.
- Existing `rememberHash`-only tests (no native size) must remain green via the unknown-size → lowest-index fallback.

## Subagent models

Use the least powerful listed model that can handle the role. Available slugs: `composer-2.5-fast`, `cursor-grok-4.5-high-fast`, `cursor-grok-4.6-xhigh`, `claude-opus-5-thinking-high`.

| Role | Model | Why |
|------|--------|-----|
| Task 1 implementer | `cursor-grok-4.5-high-fast` | Mechanical `LoadThumbnail` wrap + tests; complete code in the brief. |
| Task 2 implementer | `cursor-grok-4.6-xhigh` | Pixel map + hashRemaining backfill + UI-thread rule. Easy to freeze the UI or skip already-hashed files. |
| Task 3 implementer | `cursor-grok-4.6-xhigh` | Representative pick + injected fixtures; off-by-one / tie-break bugs. |
| Task 4 implementer | `cursor-grok-4.6-xhigh` | Viewer `StepImage` + jump; harness waits. |
| Task 5 implementer | `cursor-grok-4.5-high-fast` | Manual / ARCHITECTURE / todos copy. |
| Task reviewer (Tasks 1, 5) | `cursor-grok-4.5-high-fast` | Mid-tier floor. |
| Task reviewer (Tasks 2, 3, 4) | `cursor-grok-4.6-xhigh` | Concurrency and grouping bugs are easy to miss. |
| Parent review / fix after each task | this session (do not dispatch) | Review and fix after every step. |
| Final whole-branch review | `claude-opus-5-thinking-high` | Cross-task: thumbs never used as size, UI never probes, Return still shows the clicked extra. |

Subagent type: `generalPurpose` for implementers and reviewers. Do not use `go-expert` to write the code (it is for design questions). Do not dispatch two implementers in parallel.

If Task 2 or 3 reports `BLOCKED` on a real design hole (not missing context), re-dispatch **that task only** with `claude-opus-5-thinking-high`.

## File structure

- Modify: `internal/imaging/thumbnail.go` — add `LoadThumbnailAndBounds`; `LoadThumbnail` wraps it
- Modify: `internal/imaging/thumbnail_test.go` — bounds vs thumb size, EXIF swap
- Modify: `internal/ui/grid/grid.go` — `pixels` map on `Overview`; `New` initializes it
- Modify: `internal/ui/grid/dupes.go` — remember/wipe/backfill pixels; representative = max pixels
- Modify: `internal/ui/grid/thumbs.go` — `Warm` / miss-path `requestThumbnail` record bounds
- Modify: `internal/ui/grid/dupes_test.go` — pixel storage, backfill, highest-res pick, jump
- Modify: `internal/ui/step_test.go` — hide-dupes viewer shows highest-res, arrows skip extras
- Modify: `internal/ui/help/manual.md`, `internal/ui/help/manual_de.md`
- Modify: `ARCHITECTURE.md` — thumbnail + hide-duplicates locator
- Modify: `todos.md` — point this item at this plan (do not move it to Done until the branch is accepted)

Do not add a new grid file. Do not change `favthumbs` (backfill covers `StoreThumb`). Do not change `OnSelected`. Do not regenerate goldens.

## Current code the implementers must not break

`computeDuplicateGroups` today (replace only the inner `rep :=` loop in Task 3; Task 2 only copies `pixels` under the same lock):

```go
	groups := imaging.DuplicateGroups(hs, dist)
	for _, grp := range groups {
		rep := idx[grp[0]]
		for _, gi := range grp {
			if idx[gi] < rep {
				rep = idx[gi]
			}
		}
		// ...
	}
```

`jumpIfHiddenExtra` / `IsHiddenExtra` / `StepImage` already key off `groupReps`. Changing the pick is enough for viewer skip and the `D` jump.

`hashRemaining` today skips any URI that already has a dHash. After Task 2 it must **not** skip a hashed URI that still lacks pixels.

`requestThumbnail` cache-hit (UI goroutine) today:

```go
	if thumb, ok := g.thumbs.Get(cacheKey); ok {
		img.Image = thumb
		img.Refresh()
		if _, hashed := g.hashOf(u); !hashed {
			g.rememberHash(u, thumb)
		}
		return
	}
```

Keep that shape. Do **not** call `ReadAndProbe` here. Missing pixels are hashRemaining’s job when hide/browse starts.

`Close` / `OnSelected`: `Close` ends browse without `jumpIfHiddenExtra`; then `ShowImage(clicked)`. Keep that.

---

### Task 1: `LoadThumbnailAndBounds`

**Files:**
- Modify: `internal/imaging/thumbnail.go`
- Test: `internal/imaging/thumbnail_test.go`

**Interfaces:**
- Consumes: existing `ReadAndProbe`, `DecodeLoaded`, `ParseVector` / `RasterAt`, `scaleToFit` / `fitEdge`
- Produces: `func LoadThumbnailAndBounds(u fyne.URI) (thumb image.Image, native image.Rectangle, err error)`
  - `native` is the `ReadAndProbe` bounds (EXIF-oriented, SVG logical size, RAW preview size)
  - `thumb` is the same image `LoadThumbnail` returns today
  - On error: `thumb == nil`, `native` zero rectangle, `err != nil`
- Produces: `LoadThumbnail` becomes a one-line wrapper that discards `native`
- Callers of `LoadThumbnail` (`favthumbs`, existing tests, grid until Task 2) stay compiling

Do **not** add a second file read. One `ReadAndProbe` per call.

- [ ] **Step 1: Write the failing tests**

Append to `internal/imaging/thumbnail_test.go`:

```go
func TestLoadThumbnailAndBounds_NativeSizeIsNotThumbnailSize(t *testing.T) {
	path := writeTempFile(t, "photo.jpg", encodeJPEG(t, 800, 400, color.RGBA{R: 200, G: 20, B: 20, A: 255}))
	u := storage.NewFileURI(path)

	thumb, native, err := LoadThumbnailAndBounds(u)
	if err != nil {
		t.Fatalf("LoadThumbnailAndBounds: %v", err)
	}
	if native.Dx() != 800 || native.Dy() != 400 {
		t.Errorf("native = %dx%d, want 800x400", native.Dx(), native.Dy())
	}
	tb := thumb.Bounds()
	if tb.Dx() != ThumbnailSize || tb.Dy() != ThumbnailSize/2 {
		t.Errorf("thumb = %dx%d, want %dx%d", tb.Dx(), tb.Dy(), ThumbnailSize, ThumbnailSize/2)
	}
}

func TestLoadThumbnailAndBounds_AccountsForEXIFOrientation(t *testing.T) {
	path := writeTempFile(t, "rotated.jpg", halfRedHalfBlueJPEG(t, 20, 10, 6))

	_, native, err := LoadThumbnailAndBounds(storage.NewFileURI(path))
	if err != nil {
		t.Fatalf("LoadThumbnailAndBounds: %v", err)
	}
	if native.Dx() != 10 || native.Dy() != 20 {
		t.Errorf("native = %dx%d, want 10x20 after orientation 6", native.Dx(), native.Dy())
	}
}

func TestLoadThumbnail_WrapsLoadThumbnailAndBounds(t *testing.T) {
	path := writeTempFile(t, "photo.jpg", encodeJPEG(t, 800, 400, color.RGBA{R: 200, G: 20, B: 20, A: 255}))
	u := storage.NewFileURI(path)

	a, err := LoadThumbnail(u)
	if err != nil {
		t.Fatalf("LoadThumbnail: %v", err)
	}
	b, _, err := LoadThumbnailAndBounds(u)
	if err != nil {
		t.Fatalf("LoadThumbnailAndBounds: %v", err)
	}
	if a.Bounds() != b.Bounds() {
		t.Errorf("LoadThumbnail bounds %v, LoadThumbnailAndBounds thumb %v", a.Bounds(), b.Bounds())
	}
}
```

`halfRedHalfBlueJPEG` / `writeTempFile` / `encodeJPEG` already exist in package `imaging` tests.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -count=1 -run 'TestLoadThumbnailAndBounds_|TestLoadThumbnail_Wraps' ./internal/imaging/`

Expected: FAIL, `LoadThumbnailAndBounds` undefined.

- [ ] **Step 3: Write the implementation**

In `internal/imaging/thumbnail.go`, replace `LoadThumbnail` with:

```go
// LoadThumbnailAndBounds is LoadThumbnail plus the file's EXIF-oriented
// display size from ReadAndProbe (SVG logical size, RAW preview size).
// Callers that need a representative by resolution use native, not
// thumb.Bounds — generated thumbs are capped at ThumbnailSize.
func LoadThumbnailAndBounds(u fyne.URI) (image.Image, image.Rectangle, error) {
	data, bounds, err := ReadAndProbe(context.Background(), u)
	if err != nil {
		return nil, image.Rectangle{}, err
	}

	if isSVGData(data) {
		vec, err := ParseVector(data)
		if err != nil {
			return nil, image.Rectangle{}, err
		}
		w, h := fitEdge(bounds.Dx(), bounds.Dy(), ThumbnailSize)
		thumb, err := vec.RasterAt(w, h)
		if err != nil {
			return nil, image.Rectangle{}, err
		}
		return thumb, bounds, nil
	}

	loaded, err := DecodeLoaded(context.Background(), data, 0)
	if err != nil {
		return nil, image.Rectangle{}, err
	}
	return scaleToFit(loaded.Frames[0], ThumbnailSize), bounds, nil
}

// LoadThumbnail reads and decodes u exactly like LoadImage - full EXIF
// orientation correction included - then downsamples the first frame
// (animated GIFs show only their first frame here, same as every other
// still context in this app) to fit within ThumbnailSize on its longer
// edge. An SVG skips the decode-then-downsample round trip entirely and
// rasterizes straight at the thumbnail's size.
//
// The zero animation budget is what makes that "first frame only" literal:
// without it a long animation composited every one of its frames to a full
// RGBA canvas so this could keep one and discard the rest, which for a
// large GIF meant gigabytes of allocation per grid cell.
func LoadThumbnail(u fyne.URI) (image.Image, error) {
	thumb, _, err := LoadThumbnailAndBounds(u)
	return thumb, err
}
```

Keep the existing `LoadThumbnail` doc comment on `LoadThumbnail` (not only on the new function). Existing tests that call `LoadThumbnail` must still pass.

Check `ParseVector` / `RasterAt` error signatures against the current `LoadThumbnail` body: if today’s SVG arm does not name an error from `RasterAt`, keep that exact call shape (only add `bounds` to the return). Do not invent new error wrapping.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -count=1 ./internal/imaging/`

Expected: PASS (full imaging package, not only the new names).

- [ ] **Step 5: Do not commit**

Report files changed and the test command output.

---

### Task 2: Store native pixel counts next to hashes

**Files:**
- Modify: `internal/ui/grid/grid.go`
- Modify: `internal/ui/grid/dupes.go`
- Modify: `internal/ui/grid/thumbs.go`
- Test: `internal/ui/grid/dupes_test.go`

**Interfaces:**
- Consumes: `imaging.LoadThumbnailAndBounds`, `imaging.ReadAndProbe`
- Produces (unexported, same package):
  - `Overview.pixels map[string]int` — URI string → `Dx*Dy`; absent means unknown
  - `func (g *Overview) rememberNative(u fyne.URI, native image.Rectangle)`
  - `func (g *Overview) pixelCount(u fyne.URI) (int, bool)` — test helper; or tests may read `g.pixels` under the same conventions as `g.hashes`
  - `hashRemaining` jobs: skip only when **both** a hash and a pixel count exist (or `hashFailed`)
  - Pixel-only jobs (hash present, pixels missing): `ReadAndProbe` on the **decode worker**, then `rememberNative`. Do not re-dHash. Do not `LoadThumbnail` again if the thumb is cached.
  - Hash+size jobs (current miss path): `LoadThumbnailAndBounds` + `rememberHash` + `rememberNative` + existing `AddIfFits` rules
- Does **not** yet change representative selection (still lowest host index). Task 3 does that.

**Generation wipe:** every path that resets `g.hashes` must also reset `g.pixels`. Extract:

```go
func (g *Overview) ensureHashGenLocked(gen uint64) {
	if g.hashGen != gen {
		g.hashes = make(map[string]uint64)
		g.hashFailed = make(map[string]struct{})
		g.pixels = make(map[string]int)
		g.hashGen = gen
	}
	if g.hashes == nil {
		g.hashes = make(map[string]uint64)
	}
	if g.hashFailed == nil {
		g.hashFailed = make(map[string]struct{})
	}
	if g.pixels == nil {
		g.pixels = make(map[string]int)
	}
}
```

Call it from `rememberHash`, `rememberHashFail`, `rememberNative`, `wipeHashesIfStale` (those functions still take `hashMu`). `clearHashes` always replaces all three maps (and does not need a gen check). `New` initializes `pixels` the same way as `hashes`.

`rememberNative`:

```go
func (g *Overview) rememberNative(u fyne.URI, native image.Rectangle) {
	if u == nil {
		return
	}
	px := native.Dx() * native.Dy()
	if px < 0 {
		px = 0
	}
	gen := g.host.Generation()
	g.hashMu.Lock()
	defer g.hashMu.Unlock()
	g.ensureHashGenLocked(gen)
	g.pixels[u.String()] = px
}
```

**Warm:**

```go
		if thumb, ok := g.thumbs.Get(u.String()); ok {
			if _, hashed := g.hashOf(u); !hashed {
				g.rememberHash(u, thumb)
			}
			continue
		}

		thumb, native, err := imaging.LoadThumbnailAndBounds(u)
		if err != nil {
			return err
		}
		g.thumbs.Add(u.String(), thumb)
		g.rememberHash(u, thumb)
		g.rememberNative(u, native)
```

Warm’s cache-hit branch does **not** probe (tests/app can rely on hashRemaining). Tests that `Warm` a cold cache get pixels.

**requestThumbnail miss** (inside `g.decodes.Go`, after the second cache look): replace `LoadThumbnail` with `LoadThumbnailAndBounds`; `rememberHash` + `rememberNative`. Cache-hit on the UI goroutine: still `rememberHash` only.

**hashRemaining skip and job body:**

```go
		_, hashed := g.hashOf(u)
		_, sized := g.pixelCountOf(u) // implement as hashMu-locked lookup, same as hashOf
		if hashed && sized {
			continue
		}
		if g.hashFailedOf(u) {
			continue
		}
```

`hashFailed` still skips unreadable files entirely.

In the worker, after the existing thumb load / cache path:

```go
			if !hashed {
				g.rememberHash(file, thumb)
			}
			if !sized {
				if native, ok := nativeFromThisJob; ok {
					g.rememberNative(file, native)
				} else {
					_, b, err := imaging.ReadAndProbe(context.Background(), file)
					if err == nil {
						g.rememberNative(file, b)
					}
				}
			}
```

`nativeFromThisJob` is the `LoadThumbnailAndBounds` bounds when this job decoded; empty when the job only had a cached thumb. Do **not** `ReadAndProbe` if `LoadThumbnailAndBounds` already returned bounds.

Keep `shouldScheduleHideApply`, `computeDuplicateGroups` on the worker, `g.ui.Do` inside the `Go` body, browse-waits-for-last-job, `AddIfFits` / `ThumbCacheFull` rules exactly as they are.

- [ ] **Step 1: Write the failing tests**

Append to `internal/ui/grid/dupes_test.go`:

```go
func TestWarm_RecordsNativePixelCountNotThumbnailSize(t *testing.T) {
	u := uitest.TempJPEGURI(t, "big.jpg", 80, 40, color.RGBA{R: 200, G: 20, B: 20, A: 255})
	host := &fakeHost{files: []fyne.URI{u}}
	g := newOverview(t, host)
	if err := g.Warm(); err != nil {
		t.Fatalf("Warm: %v", err)
	}
	px, ok := g.pixelCountOf(u)
	if !ok {
		t.Fatal("Warm should record native pixels")
	}
	if px != 80*40 {
		t.Errorf("pixels = %d, want %d (not the thumbnail)", px, 80*40)
	}
}

func TestWipeHashesIfStale_DropsPixelsOnGenerationChange(t *testing.T) {
	host := hostWith(t, "a.jpg")
	g := newOverview(t, host)
	if err := g.Warm(); err != nil {
		t.Fatalf("Warm: %v", err)
	}
	host.gen++
	g.wipeHashesIfStale()
	if _, ok := g.pixelCountOf(host.files[0]); ok {
		t.Fatal("pixels must drop when host.Generation changes")
	}
}

func TestHashRemaining_BackfillsPixelsForAlreadyHashedFiles(t *testing.T) {
	u := uitest.TempJPEGURI(t, "big.jpg", 80, 40, color.RGBA{R: 200, G: 20, B: 20, A: 255})
	host := &fakeHost{files: []fyne.URI{u}}
	g := newOverview(t, host)
	thumb := mustThumb(t, u)
	g.thumbs.Add(u.String(), thumb)
	g.rememberHash(u, thumb)
	if _, ok := g.pixelCountOf(u); ok {
		t.Fatal("setup: pixels should be missing")
	}

	g.SetHideDuplicates(true)
	g.Settle()

	px, ok := g.pixelCountOf(u)
	if !ok {
		t.Fatal("hashRemaining should backfill pixels for a hashed file")
	}
	if px != 80*40 {
		t.Errorf("pixels = %d, want %d", px, 80*40)
	}
}

func TestHashRemaining_DoesNotRequeueWhenHashAndPixelsExist(t *testing.T) {
	host := hostWith(t, "a.jpg")
	g := newOverview(t, host)
	if err := g.Warm(); err != nil {
		t.Fatalf("Warm: %v", err)
	}
	if n := g.hashRemaining(); n != 0 {
		t.Fatalf("hashRemaining() = %d, want 0 after Warm", n)
	}
}
```

Name the lookup `pixelCountOf` to match `hashOf` / `hashFailedOf`. Implement it in Step 3; the tests failing because it is missing is the red phase.

Solid-color JPEGs dHash to 0 (ungroupable). These tests must not assert grouping.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -count=1 -run 'TestWarm_RecordsNative|TestWipeHashesIfStale_DropsPixels|TestHashRemaining_Backfills|TestHashRemaining_DoesNotRequeue' ./internal/ui/grid/`

Expected: FAIL (`pixelCountOf` undefined and/or pixels not recorded).

- [ ] **Step 3: Implement storage, Warm, requestThumbnail miss, hashRemaining backfill**

Follow the Interfaces block. Existing `TestWipeHashesIfStale_DropsHashesOnGenerationChange` must still pass. Existing `TestSetHideDuplicates_HashesRemainingWithoutWarm` and pending-job tests must still pass.

`pixelCountOf`:

```go
func (g *Overview) pixelCountOf(u fyne.URI) (int, bool) {
	if u == nil {
		return 0, false
	}
	g.wipeHashesIfStale()
	g.hashMu.Lock()
	defer g.hashMu.Unlock()
	n, ok := g.pixels[u.String()]
	return n, ok
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -count=1 ./internal/ui/grid/`

Expected: PASS (full grid package). Representative pick is still lowest index.

- [ ] **Step 5: Do not commit**

Report files changed and the test command output.

---

### Task 3: Pick the highest-pixel member as representative

**Files:**
- Modify: `internal/ui/grid/dupes.go` (`computeDuplicateGroups` inner loop, `RepresentativeOf` comment)
- Modify: `internal/imaging/dhash.go` — optional one-line comment clarification only
- Test: `internal/ui/grid/dupes_test.go`

**Interfaces:**
- Consumes: `g.pixels` copied under the same `hashMu` section that copies hashes (do not look up pixels after unlocking — a concurrent wipe could drop the map)
- Produces: for each `DuplicateGroups` group, `rep` is the host index with max pixel count; ties → smaller host index
- `IsHiddenExtra` / `jumpIfHiddenExtra` / badges unchanged

Copy pixels in the existing lock:

```go
	g.hashMu.Lock()
	idx := make([]int, 0, n)
	hs := make([]uint64, 0, n)
	hashed := make([]bool, n)
	px := make([]int, n)
	dist := g.dupeDist
	for i := range n {
		u := g.host.FileAt(i)
		if h, ok := g.hashes[u.String()]; ok {
			idx = append(idx, i)
			hs = append(hs, h)
			hashed[i] = true
		}
		if p, ok := g.pixels[u.String()]; ok {
			px[i] = p
		}
	}
	g.hashMu.Unlock()
```

Inner pick (replace lowest-index loop):

```go
		rep := idx[grp[0]]
		repPx := px[rep]
		for _, gi := range grp {
			hi := idx[gi]
			if px[hi] > repPx || (px[hi] == repPx && hi < rep) {
				rep, repPx = hi, px[hi]
			}
		}
```

Update `RepresentativeOf`’s comment from “lowest host index” to “highest native pixel count in the group, lowest host index on a tie; itself when unique, unhashed, or out of range”.

- [ ] **Step 1: Write the failing tests**

Use **injected** hashes and pixels so grouping does not depend on dHash of solid JPEGs. Hash `0` is ungroupable — use a non-zero value.

```go
func TestComputeDuplicateGroups_PicksHighestPixelCount(t *testing.T) {
	host := hostWith(t, "small.jpg", "large.jpg", "unique.jpg")
	g := newOverview(t, host)
	const h uint64 = 0x1111111111111111
	g.hashes = map[string]uint64{
		host.files[0].String(): h,
		host.files[1].String(): h,
		host.files[2].String(): 0x2222222222222222,
	}
	g.pixels = map[string]int{
		host.files[0].String(): 100,
		host.files[1].String(): 400,
		host.files[2].String(): 9999,
	}
	g.hashGen = host.gen
	g.rebuildGroups()

	if g.RepresentativeOf(0) != 1 || g.RepresentativeOf(1) != 1 {
		t.Errorf("rep of pair = %d/%d, want 1 (larger file)", g.RepresentativeOf(0), g.RepresentativeOf(1))
	}
	if g.RepresentativeOf(2) != 2 {
		t.Errorf("unique rep = %d, want 2", g.RepresentativeOf(2))
	}

	g.SetHideDuplicates(true)
	if g.count() != 2 {
		t.Fatalf("count() = %d, want 2", g.count())
	}
	if g.fileIndex(0) != 1 || g.fileIndex(1) != 2 {
		t.Fatalf("visible = [%d, %d], want [1, 2] (large + unique)", g.fileIndex(0), g.fileIndex(1))
	}
	if !g.IsHiddenExtra(0) || g.IsHiddenExtra(1) || g.IsHiddenExtra(2) {
		t.Error("only the smaller pair member is a hidden extra")
	}
}

func TestComputeDuplicateGroups_EqualPixelsKeepsLowestIndex(t *testing.T) {
	host := hostWith(t, "a.jpg", "b.jpg")
	g := newOverview(t, host)
	const h uint64 = 0x1111111111111111
	g.hashes = map[string]uint64{
		host.files[0].String(): h,
		host.files[1].String(): h,
	}
	g.pixels = map[string]int{
		host.files[0].String(): 100,
		host.files[1].String(): 100,
	}
	g.hashGen = host.gen
	g.rebuildGroups()
	if g.RepresentativeOf(1) != 0 {
		t.Errorf("RepresentativeOf(1) = %d, want 0 (tie-break)", g.RepresentativeOf(1))
	}
}

func TestComputeDuplicateGroups_UnknownPixelsLoseToKnown(t *testing.T) {
	host := hostWith(t, "unknown.jpg", "known.jpg")
	g := newOverview(t, host)
	const h uint64 = 0x1111111111111111
	g.hashes = map[string]uint64{
		host.files[0].String(): h,
		host.files[1].String(): h,
	}
	g.pixels = map[string]int{
		host.files[1].String(): 50,
	}
	g.hashGen = host.gen
	g.rebuildGroups()
	if g.RepresentativeOf(0) != 1 {
		t.Errorf("RepresentativeOf(0) = %d, want 1 (known size wins)", g.RepresentativeOf(0))
	}
}

func TestSetHideDuplicates_JumpsToHighestResolution(t *testing.T) {
	host := hostWith(t, "small.jpg", "large.jpg")
	g := newOverview(t, host)
	const h uint64 = 0x1111111111111111
	g.hashes = map[string]uint64{
		host.files[0].String(): h,
		host.files[1].String(): h,
	}
	g.pixels = map[string]int{
		host.files[0].String(): 100,
		host.files[1].String(): 400,
	}
	g.hashGen = host.gen
	host.index = 0

	g.SetHideDuplicates(true)

	if len(host.shown) == 0 || host.shown[len(host.shown)-1] != 1 {
		t.Errorf("ShowImage calls = %v, want a jump to representative 1", host.shown)
	}
}
```

`TestSetHideDuplicates_JumpsToRepresentative` (pairAndUnique, equal sizes, jump to 0) must stay and still pass.

Assigning `g.hashes` / `g.pixels` directly matches existing tests around `dupes_test.go` (~line 892). Set `g.hashGen = host.gen` so `wipeHashesIfStale` does not drop the maps.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -count=1 -run 'TestComputeDuplicateGroups_PicksHighest|TestComputeDuplicateGroups_EqualPixels|TestComputeDuplicateGroups_UnknownPixels|TestSetHideDuplicates_JumpsToHighest' ./internal/ui/grid/`

Expected: FAIL (still picking index 0).

- [ ] **Step 3: Change the representative loop**

Only the pick loop + comments. Do not change `DuplicateGroups`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -count=1 ./internal/ui/grid/`

Expected: PASS. Especially `TestSetHideDuplicates_HidesExtrasKeepsUniques` (equal 64×48 patterned JPEGs → still index 0).

- [ ] **Step 5: Do not commit**

Report files changed and the test command output.

---

### Task 4: Viewer navigation follows the new representative

**Files:**
- Modify: `internal/ui/step_test.go` (tests only; `viewer.go` skip logic is already `IsHiddenExtra`)
- Test: `internal/ui/step_test.go`
- Optionally: `internal/ui/grid/dupes_test.go` only if Task 3 missed a Warm-based integration case

**Interfaces:**
- Consumes: `dropAndWait`, `grid.Warm`, `SetHideDuplicates`, `StepImage`, `waitUntilLoaded`
- Produces: a viewer test where the later file is larger, hide-on shows it, arrows skip the smaller extra

`PatternedJPEGURI` is fixed 64×48. Add a sized helper in `internal/uitest/uitest.go`:

```go
// PatternedJPEGURISize is PatternedJPEGURI at an explicit size. Same seed
// at two sizes is the hide-duplicates fixture for “same shot, different
// resolution”.
func PatternedJPEGURISize(t *testing.T, name string, seed, w, h int) fyne.URI {
	t.Helper()
	if w <= 0 || h <= 0 {
		t.Fatalf("PatternedJPEGURISize: invalid size %dx%d", w, h)
	}
	img := image.NewGray(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetGray(x, y, color.Gray{Y: uint8(x*13 + y*7 + seed*31)})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode patterned jpeg: %v", err)
	}
	return storage.NewFileURI(WriteTempFile(t, name, buf.Bytes()))
}
```

Keep `PatternedJPEGURI` as `PatternedJPEGURISize(t, name, seed, 64, 48)` or leave it inlined — do **not** break existing callers either way.

If two sizes of the same seed do not group at default distance 6, the test may `v.grid.SetDuplicateDistance(32)` after Warm and before hide. Prefer proving they group: after Warm, `if v.grid.RepresentativeOf(0) == 0 && v.grid.RepresentativeOf(1) == 1` and both unique, fatal with Hamming so the fixture can be adjusted. Do not silently pass a test that never hid an extra.

- [ ] **Step 1: Write the failing test**

In `internal/ui/step_test.go`:

```go
func TestStepImage_HideDuplicatesShowsHighestResolution(t *testing.T) {
	v := newTestViewer(t)
	small := uitest.PatternedJPEGURISize(t, "small.jpg", 1, 64, 48)
	large := uitest.PatternedJPEGURISize(t, "large.jpg", 1, 192, 144)
	other := uitest.PatternedJPEGURI(t, "other.jpg", 99)
	dropAndWait(t, v, small, large, other)
	if err := v.grid.Warm(); err != nil {
		t.Fatalf("Warm: %v", err)
	}

	v.grid.SetHideDuplicates(true)
	waitUntilLoaded(t, v)
	if v.state.index != 1 {
		t.Fatalf("index = %d, want 1 (larger copy of seed 1)", v.state.index)
	}

	v.StepImage(1)
	waitUntilLoaded(t, v)
	if v.state.index != 2 {
		t.Fatalf("after StepImage(1) index = %d, want 2 (skipped small extra at 0)", v.state.index)
	}

	v.StepImage(1)
	waitUntilLoaded(t, v)
	if v.state.index != 1 {
		t.Fatalf("after wrap index = %d, want 1", v.state.index)
	}
}
```

If grouping fails at default distance, set distance 32 as documented above, still asserting hide count / extras, not merely that StepImage moved.

`TestStepImage_SkipsHiddenExtras` (equal sizes, skip index 1) stays.

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test -count=1 -run TestStepImage_HideDuplicatesShowsHighestResolution ./internal/ui/`

Expected: FAIL if Task 3 is complete but Warm never stored pixels — should **pass** if Tasks 2–3 landed. If it passes on the first run because the branch already behaves, that is success; do not weaken the assertion.

This task exists because Task 3 used injected maps. This is the real `LoadThumbnailAndBounds` → `Warm` → `jumpIfHiddenExtra` → `StepImage` path.

If the test fails because the two sizes do not group, fix the fixture (distance or sizes) in this task — do not revert Task 3’s pick rule.

- [ ] **Step 3: Fix only if the integration path is broken**

Likely holes: `Warm` cache-hit skipping `rememberNative` (use cold cache in the test — `dropAndWait` may have filled thumbs via the viewer; if so, `hashRemaining` on `SetHideDuplicates` must backfill — that is Task 2). If pixels are still missing after Settle, that is a Task 2 bug: fix it here only if it is a one-line call-site miss, otherwise report `BLOCKED` naming Task 2.

Do not change `OnSelected`.

- [ ] **Step 4: Run tests to verify they pass**

Run:

```
go test -count=1 -run 'TestStepImage_' ./internal/ui/
go test -count=1 ./internal/ui/grid/
```

Expected: PASS.

- [ ] **Step 5: Do not commit**

Report files changed and the test command output.

---

### Task 5: Docs and todos

**Files:**
- Modify: `internal/ui/help/manual.md` (the sentence that says “the earliest file in the current order”)
- Modify: `internal/ui/help/manual_de.md` (the matching “früheste Datei in der aktuellen Reihenfolge”)
- Modify: `ARCHITECTURE.md` — `thumbnail.go` row and/or hide-duplicates “Where to look” line
- Modify: `todos.md` — leave the item under TODO, add a pointer to this plan; do **not** move it to Done

**Interfaces:** none.

English replacement for the representative clause:

> keeps one representative (the highest-resolution file: most pixels after EXIF orientation; equal sizes keep the earliest file in the current order).

German:

> behält einen Vertreter (die Datei mit der höchsten Auflösung: die meisten Pixel nach EXIF-Ausrichtung; bei gleicher Größe die früheste Datei in der aktuellen Reihenfolge).

ARCHITECTURE `thumbnail.go` row: mention `LoadThumbnailAndBounds` returns native `ReadAndProbe` size for hide-duplicates.

Hide-duplicates “Where to look” can stay `dhash.go` + `dupes.go`.

- [ ] **Step 1: Edit the four files**

No tests. Do not add `lang.L` keys. Do not touch README unless a duplicates sentence there still says “earliest” (today it does not).

- [ ] **Step 2: Locale / fmt**

Run: `make fmt-check`

If you touched only markdown, `make fmt-check` still must pass on the branch’s Go files.

- [ ] **Step 3: Do not commit**

Report files changed.

---

## Execution notes for the parent agent

1. Confirm the three open points (or “use the defaults”).
2. Dispatch Task 1 → review/fix → Task 2 → review/fix → … → Task 5.
3. After Task 5, run CI-matching verification from the repo root:

```
make fmt-check
go vet ./...
go build ./...
go test -race ./...
```

4. Suggested commit message (user commits):

```
Show the highest-resolution file when hiding duplicates.

Hide-duplicates kept the earliest file in the current sort order, which
could bury a larger original behind a downscale. Representatives are now
max oriented pixel count, with lowest index as the tie-break.
```

## Self-review

- Spec coverage: metric, tie-break, unknown size, RAW/SVG, live rebuild, D-jump, viewer skip, Return-not-jump, docs — each has a task.
- No TBD / “add tests later”.
- `LoadThumbnailAndBounds` name is used in Tasks 1–4 consistently.
- `pixelCountOf` / `rememberNative` / `pixels` map names are used consistently.
- `OnSelected` is explicitly unchanged so the next todo is not pre-empted.