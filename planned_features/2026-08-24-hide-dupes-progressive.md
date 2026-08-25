# Progressive Hide-Duplicates Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Parent protocol (this session):** Do **not** start until Florian says to start the subagents. Dispatch **one implementer at a time**, in task order. After each task the **parent reads the diff, re-runs that task's verification command, and fixes drift** before dispatching a reviewer and then the next implementer. Do **not** `git commit` (`AGENTS.md`); the parent suggests one message at the very end.
>
> **Predecessor:** `docs/superpowers/plans/2026-08-24-grid-toggle-race.md` (and `.superpowers/sdd/uiqueue/progress.md`) already landed the per-instance `uiQueue` / `uitest.UIQueue` / looping `Settle` / `parkDecodes`. That work is in the working tree and **must not be reverted**. This plan is the follow-up filed in `todos.md` under *grid: undo the production compromises made for the inline test driver*, plus the cheap *middle of the hashing window* coverage item.

**Goal:** When the user presses `D` on a large cold folder, the grid immediately shows the "Hiding duplicates" chrome and extras disappear as hashes land, instead of waiting until every remaining thumbnail has hashed.

**Architecture:** The two production compromises in `dupes.go` exist only because Fyne's test driver runs `fyne.Do` inline on the decode worker. The grid package already marshals through `g.ui.Do`, so those compromises can come out: `SetHideDuplicates` always `applyFilter`s on the caller, and every hide-duplicates hash job `rebuildFilter(false)`s through `g.ui.Do`. Incremental applies capture the highlighted *host* index before rebuilding `matches`, then restore it — they must not `setHighlight(0)` / `ScrollTo(0)`, or hashing a big folder would yank the viewport to the top on every completion. Browse (`Shift+D`) stays last-job `finishBrowse`. `internal/ui`'s tests install `*uitest.UIQueue` through a new exported `SetUIQueue`, so multiple in-flight `g.ui.Do` bodies cannot run on four workers at once under that suite either.

**Tech Stack:** Go 1.26.7 (toolchain 1.27.0), Fyne v2.8.0, `go test -race`, packages `internal/ui/grid`, `internal/ui`, `internal/uitest`.

## Default resolutions (locked unless Florian overrides before dispatch)

These were the open product questions. Implementers treat them as spec:

1. **Immediate chrome.** `SetHideDuplicates(true)` always applies the filter on the caller, even when `hashRemaining` returned a positive pending count. Unhashed files are not extras (`IsHiddenExtra`), so a cold folder stays fully visible at first; the top bar still reads "Hiding duplicates".
2. **Progressive hide, last-job browse.** Every hash-job completion that finds hide on and browse off calls `rebuildFilter(false)` + `jumpIfHiddenExtra`. `finishBrowse` still runs only when `hashJobs` hits 0. Do not toast on the hide path. Do not show a partial browse group.
3. **Keep the file under the ring.** Hash-completion applies capture `fileIndex(g.highlight)` *before* rebuilding `matches`. They do not `ScrollTo`. User-initiated `applyFilter()` (search, `D`, Escape, distance, selection-resync) still resets to cell 0 and scrolls to 0, unchanged.
4. **Queue the app suite.** Export `grid.UIQueue` (rename of the existing unexported interface) and `(*Overview).SetUIQueue`. `newTestUI` always installs `&uitest.UIQueue{}` after `buildStartupViewer`. Do not add a mutex around `applyFilter`; Fyne widgets are not mutex-safe. Do not import `internal/uitest` from production `grid` files.
5. **`hashRemaining` is not a getter.** A second call after `SetHideDuplicates` would see in-flight keys in `g.hashing` and return 0. Mid-window tests assert `g.hashJobs.Load()`, chrome, and `count()`, never a second `hashRemaining()`.
6. **No commits.** `AGENTS.md` forbids `git commit`. Skip every commit step. `gofmt` every touched file.

## Why the compromises are safe to remove now

- **Queue without parking was flaky; parking without the queue was racy.** That is already fixed in the predecessor. Grid-package tests drain `g.ui` on the test goroutine.
- **Last-job-only `applyFilter`** serializes workers under the inline driver. Once every apply goes through `g.ui.Do`, the app's real driver serializes on the UI goroutine and the test queue serializes on `Drain`. Last-job-only is what prevents progressive hide.
- **Skipping the caller `applyFilter` when `pending > 0`** avoids a caller-vs-worker race on `g.matches` / widgets under the inline driver. The caller is already on the UI/test goroutine; the race only exists if a worker's `fyne.Do` runs `applyFilter` *on the worker at the same time*. With a deferring queue, the worker only *enqueues*; `Settle`/`Drain` runs the body later on the test goroutine, after the caller has returned. Production `fyne.Do` enqueues onto the UI goroutine, which is also not the worker. So the caller can apply immediately.

Naive "call `applyFilter` on every job" is still wrong: `applyFilter` always `setHighlight(0)` and `ScrollTo(0)` (`search.go`). That is correct for a keystroke. It is not correct for a hash landing while the user is looking at cell 40.

## Global Constraints

- Do **not** `git commit`. `AGENTS.md` forbids it; the parent suggests one message after Task 5.
- Do **not** add `TODO`/`FIXME` comments to source. Open work goes in `todos.md`.
- Do **not** add mutable **package-level** test seams. `Overview.ui` stays a per-instance field; `SetUIQueue` is a per-instance setter (what `AGENTS.md` permits).
- Do **not** import `internal/uitest` from production `grid` files (`uiqueue.go`, `dupes.go`, `search.go`, `grid.go`, `thumbs.go`, …). `*uitest.UIQueue` is passed in from `_test.go` files and from `internal/ui/harness_test.go`.
- Do **not** change dHash, `DuplicateGroups`, `IsHiddenExtra` (unhashed ⇒ not extra), browse toast copy, or Escape staging.
- Do **not** call `g.ui.Do` from anywhere except inside a `g.decodes.Go` body. `Settle`'s barrier is `decodes.Wait()`; a completion queued from an untracked goroutine slips past it. `rebuildFilter(false)` from a hash job is invoked *inside* the existing `g.ui.Do` that already sits in that `Go` body.
- Every user-visible string is `lang.L("English text")`. This work adds none. The chrome string stays `lang.L("Hiding duplicates")`.
- Every `g.ui.Do` must still sit on the return path of its `decodes.Go` fn (see `thumbs.go` above the `requestThumbnail` `Go` call).
- Formatting: `gofmt -l -w` on every file touched. Match CI before handoff: `gofmt`, `go vet ./...`, `go build ./...`, `go test -race ./...` from the repo root.
- Work from `/Users/ronin/Projects/picfetch`. Golden screenshots are out of scope: **do not** run `make golden` and do not touch `internal/ui/testdata/`.
- `hashRemaining` starts work; tests must not call it twice to "read" a pending count. Use `g.hashJobs.Load()`.

## File map

| File | Change | Task |
| --- | --- | --- |
| `internal/ui/grid/dupes_test.go` | New tests: immediate chrome + mid-window `hashJobs`; keep-view; one-job progressive hide via the pool. | 1, 2, 3 |
| `internal/ui/grid/dupes.go` | `SetHideDuplicates` always filters; `hashRemaining` applies hide on every job; comments. | 1, 3 |
| `internal/ui/grid/search.go` | `applyFilter` wraps `rebuildFilter(true)`; new `rebuildFilter` / `restoreHighlight` / `displayIndexOfHost`. | 2 |
| `internal/ui/grid/grid.go` | `hashJobs` field comment no longer mentions last-job / inline driver. | 3 |
| `internal/ui/grid/uiqueue.go` | Export `UIQueue`; add `SetUIQueue`. Update the "internal/ui keeps fyneQueue" sentence. | 4 |
| `internal/ui/grid/harness_test.go` | `newOverview` calls `SetUIQueue` instead of assigning `g.ui`. | 4 |
| `internal/ui/harness_test.go` | `newTestUI` installs `&uitest.UIQueue{}`. | 4 |
| `ARCHITECTURE.md` | Hide-duplicates blurb: progressive apply + `SetUIQueue` in the app suite. | 5 |
| `AGENTS.md` | Concurrency bullet: app tests also install the drainable queue. | 5 |
| `todos.md` | Retire both follow-up TODOs into Done. | 5 |

## Subagent roster

Cursor's Task tool does **not** offer Opus. Use `cursor-grok-4.6-xhigh` wherever this plan would have used Opus (judgment, concurrency, final review).

| Task | Role | `subagent_type` | Model | Why |
| --- | --- | --- | --- | --- |
| 1 | Implementer | `go-expert` | `composer-2.5-fast` | Exact test + a 15-line production edit. Transcription. |
| 1 | Reviewer | `generalPurpose` | `cursor-grok-4.5-high-fast` | Small diff, spec check. |
| 2 | Implementer | `go-expert` | `cursor-grok-4.6-xhigh` | Viewport restore is easy to get wrong (`fileIndex` after rebuild is stale). |
| 2 | Reviewer | `generalPurpose` | `cursor-grok-4.6-xhigh` | Same reason. |
| 3 | Implementer | `go-expert` | `cursor-grok-4.6-xhigh` | Completion contract, browse last-job, `g.ui.Do` placement. |
| 3 | Reviewer | `generalPurpose` | `cursor-grok-4.6-xhigh` | Concurrency. |
| 4 | Implementer | `go-expert` | `cursor-grok-4.5-high-fast` | Exported setter + harness; then a long `internal/ui` race run. |
| 4 | Reviewer | `generalPurpose` | `cursor-grok-4.5-high-fast` | API/layering. |
| 5 | Implementer | `generalPurpose` | `composer-2.5-fast` | Docs + `todos.md` + full-suite gate. |
| 5 | Reviewer | `generalPurpose` | `cursor-grok-4.5-high-fast` | Doc/spec check. |
| Final | Whole-branch review | `generalPurpose` | `cursor-grok-4.6-xhigh` | Concurrency + cross-package. |

Parent (this session, after Florian says start): review every implementer diff, re-run that task's test command, fix drift, then dispatch the reviewer. Do not parallelize implementers.

## Reference reading for implementers

- `internal/ui/grid/dupes.go` `SetHideDuplicates` (lines 110–130) and `hashRemaining` (lines 333–404), including the two comments that name the inline test driver — those comments are what this plan deletes/replaces.
- `internal/ui/grid/search.go` `applyFilter` (lines 88–142): the `setHighlight(0)` / `ScrollTo(0)` pair is why hash completions cannot call `applyFilter` as it stands.
- `internal/ui/grid/dupes.go` `IsHiddenExtra`: unhashed ⇒ not extra. That is why immediate apply on a cold folder does not collapse the grid.
- `internal/ui/grid/dupes_test.go` `TestSetHideDuplicates_HidesExtrasKeepsUniques` (chrome assertions) and `TestApplyFilter_BrowsePendingDoesNotCollapseGrid` (parked pending window).
- `internal/ui/grid/harness_test.go` `parkDecodes` preconditions: call before any decode; never `Settle` while parked.
- `internal/ui/grid/uiqueue.go` and `internal/uitest/uiqueue.go`.
- `AGENTS.md` "Concurrency and Fyne" (`g.ui.Do` only from inside `decodes.Go`).

---

### Task 1: Immediate "Hiding duplicates" chrome while hashes are pending

**Files:**
- Modify: `internal/ui/grid/dupes_test.go` (append the two tests below after `TestSetHideDuplicates_HashesRemainingWithoutWarm`)
- Modify: `internal/ui/grid/dupes.go` (`SetHideDuplicates` only)

**Interfaces:**
- Consumes: `parkDecodes`, `hostPatterned`, `newOverview`, `rememberHash`, `hashJobs`, `searchLabel`, `count()`, `lang.L("Hiding duplicates")` — all already exist.
- Produces: `SetHideDuplicates(true)` always calls `applyFilter()` (and `jumpIfHiddenExtra` when `on`) after `hashRemaining`, even when pending > 0. Hash jobs still last-job-apply until Task 3.

- [ ] **Step 1: Write the failing tests**

Append to `internal/ui/grid/dupes_test.go`. Do not call `hashRemaining` in the tests; `SetHideDuplicates` already does.

```go
func TestSetHideDuplicates_PendingShowsChromeAndLeavesUnhashedVisible(t *testing.T) {
	host := hostPatterned(t,
		[]string{"sunset-a.jpg", "sunset-b.jpg", "moon.jpg"},
		[]int{1, 1, 99},
	)
	g := newOverview(t, host)
	g.rememberHash(host.files[0], mustThumb(t, host.files[0]))

	unpark := parkDecodes(t, g)
	g.Toggle()
	g.SetHideDuplicates(true)

	if !g.HideDuplicates() {
		t.Fatal("HideDuplicates() = false after SetHideDuplicates(true)")
	}
	if got, want := g.searchLabel.Text, lang.L("Hiding duplicates"); got != want {
		t.Errorf("searchLabel = %q, want %q while hashes are still pending", got, want)
	}
	if g.count() != 3 {
		t.Fatalf("count() = %d while the extra is still unhashed, want 3", g.count())
	}
	if got := g.hashJobs.Load(); got != 2 {
		t.Fatalf("hashJobs = %d, want 2 (file 0 already hashed; 1 and 2 still pending)", got)
	}

	unpark()
	g.Settle()

	if g.count() != 2 {
		t.Fatalf("count() = %d after remaining hashes land, want 2", g.count())
	}
	if !g.IsHiddenExtra(1) {
		t.Error("the pair's extra should hide once its hash lands")
	}
}

func mustThumb(t *testing.T, u fyne.URI) image.Image {
	t.Helper()
	thumb, err := imaging.LoadThumbnail(u)
	if err != nil {
		t.Fatalf("LoadThumbnail(%s): %v", u.Name(), err)
	}
	return thumb
}
```

`imaging` is already imported in `dupes_test.go`. `image` is already imported.

This is also the "middle of the hashing window" coverage `todos.md` asked for: one of three files hashed, pool parked, hide on, `hashJobs == 2`, then unpark+Settle to the fully-hashed end state.

- [ ] **Step 2: Run the tests and confirm they fail**

```bash
go test -race -count=1 -run 'TestSetHideDuplicates_PendingShowsChromeAndLeavesUnhashedVisible' ./internal/ui/grid/ -v
```

Expected: FAIL. `searchLabel` is still empty / not `"Hiding duplicates"` because `SetHideDuplicates` skipped `applyFilter` when `pending > 0`. (If `hashJobs` is also wrong, stop and re-read `hashRemaining` — do not "fix" the assertion to match a spawn bug.)

- [ ] **Step 3: Always apply on the caller**

Replace `SetHideDuplicates` in `internal/ui/grid/dupes.go` with:

```go
func (g *Overview) SetHideDuplicates(on bool) {
	if g.hideDupes == on {
		return
	}
	g.hideDupes = on
	if on {
		_ = g.hashRemaining()
	}
	g.applyFilter()
	if on {
		g.jumpIfHiddenExtra()
	}
}
```

Delete the comment that begins `Background hash jobs applyFilter from the last pool worker`. Do not touch `hashRemaining` in this task.

- [ ] **Step 4: Re-run Task 1's test plus the existing hide/browse pending tests**

```bash
gofmt -w internal/ui/grid/dupes.go internal/ui/grid/dupes_test.go
go test -race -count=1 -run 'TestSetHideDuplicates_|TestApplyFilter_BrowsePending|TestSetBrowsingDuplicates_HashesRemaining' ./internal/ui/grid/ -v
```

Expected: PASS. `TestApplyFilter_BrowsePendingDoesNotCollapseGrid` still sees `count() == 3` while pending: hide is ignored while browse is on (`hide := g.hideDupes && !browsing`), and `browseFilter` is false until `groupSize >= 2`.

- [ ] **Step 5: Do not commit.** Parent reviews the diff and re-runs Step 4's command.

---

### Task 2: `rebuildFilter` keeps the highlighted file on incremental applies

**Files:**
- Modify: `internal/ui/grid/search.go` (`applyFilter` and the three new helpers)
- Modify: `internal/ui/grid/dupes_test.go` (keep-view test)

**Interfaces:**
- Consumes: Task 1's always-apply `SetHideDuplicates`. Existing `applyFilter()` call sites (`search.go`, `nav.go`, `dupes.go` `finishBrowse`/`SetBrowsingDuplicates(false)`/`SetDuplicateDistance`, `selection.go`) stay as `applyFilter()` and keep reset-to-0 behaviour.
- Produces:
  - `func (g *Overview) applyFilter()` — unchanged signature; body is `g.rebuildFilter(true)`.
  - `func (g *Overview) rebuildFilter(resetView bool)` — unexported. `resetView == true` is today's `applyFilter`. `resetView == false` captures the highlighted host **before** mutating `matches`, rebuilds, `wrap.Refresh` if visible, restores the ring onto that host (or its representative, or 0), does **not** `ScrollTo`, then `syncTopBar`.
  - `func (g *Overview) displayIndexOfHost(hostIdx int) int` — like `displayIndexOf` but returns `-1` when the host is not visible, never `0` as a miss sentinel (`displayIndexOf` must stay as-is; `finishBrowse` uses it).
  - `func (g *Overview) restoreHighlight(host int)` — visible-grid only, called from `rebuildFilter(false)`.

- [ ] **Step 1: Write the failing keep-view test**

Append to `dupes_test.go`. This test does **not** wait on the pool for the incremental step: it `rememberHash`es the extra on the test goroutine and calls `rebuildFilter(false)` directly (same package). That is what pins the host-capture-before-rebuild rule. Task 3 wires the pool to the same method.

```go
func TestRebuildFilter_KeepsHighlightedHostWhenAnExtraDisappears(t *testing.T) {
	host := hostPatterned(t,
		[]string{"sunset-a.jpg", "sunset-b.jpg", "moon.jpg"},
		[]int{1, 1, 99},
	)
	g := newOverview(t, host)
	g.rememberHash(host.files[0], mustThumb(t, host.files[0]))
	g.rememberHash(host.files[2], mustThumb(t, host.files[2]))

	unpark := parkDecodes(t, g)
	g.Toggle()
	g.SetHideDuplicates(true)

	if g.count() != 3 {
		t.Fatalf("setup count() = %d, want 3 (extra still unhashed)", g.count())
	}
	g.setHighlight(2)
	if g.fileIndex(g.Highlight()) != 2 {
		t.Fatalf("setup highlight host = %d, want 2", g.fileIndex(g.Highlight()))
	}

	g.rememberHash(host.files[1], mustThumb(t, host.files[1]))
	g.rebuildFilter(false)

	if g.count() != 2 {
		t.Fatalf("count() = %d after extra hashes, want 2", g.count())
	}
	if !g.IsHiddenExtra(1) {
		t.Fatal("index 1 should now be a hidden extra")
	}
	if g.fileIndex(g.Highlight()) != 2 {
		t.Fatalf("Highlight host = %d, want 2 (moon.jpg stayed under the ring; display index may have moved)", g.fileIndex(g.Highlight()))
	}
	if g.Highlight() != 1 {
		t.Fatalf("Highlight() = %d, want 1 (host 2 is now display index 1)", g.Highlight())
	}

	unpark()
	g.Settle()
}
```

Until `rebuildFilter` exists this fails to compile — that is the RED. If a Task 2 implementer finds `rebuildFilter` already named something else from a rogue edit, they still need this behaviour; they do not invent a second helper.

- [ ] **Step 2: Confirm RED**

```bash
go test -count=1 -run TestRebuildFilter_KeepsHighlightedHostWhenAnExtraDisappears ./internal/ui/grid/ -v
```

Expected: FAIL compile: `g.rebuildFilter undefined`.

- [ ] **Step 3: Implement `rebuildFilter` in `search.go`**

Replace `applyFilter` with the following. Keep `applyFilter`'s doc comment on `applyFilter` itself; put the incremental contract on `rebuildFilter`.

```go
// applyFilter recomputes the visible subset from the current query and
// redraws the grid around it. An empty query - which is what an
// just-opened search bar has - matches everything, so opening search
// changes nothing on screen until a character is typed.
//
// The whole set is rescanned per keystroke rather than narrowed from the
// previous result: Backspace widens the match set again, and a
// strings.Contains over a few thousand names is not worth a cache.
func (g *Overview) applyFilter() {
	g.rebuildFilter(true)
}

// rebuildFilter is applyFilter with a choice of viewport. resetView true
// (search, D, Escape, distance, selection resync) jumps the ring to cell 0
// and scrolls there. resetView false is for a hash landing while hide is
// already on: keep the same host file under the ring so a long cold-folder
// hash does not yank the user to the top on every completion. The host
// index is captured before matches is rebuilt - fileIndex after a hide
// shrinks the grid would read a shifted or out-of-range display index.
func (g *Overview) rebuildFilter(resetView bool) {
	keepHost := -1
	if !resetView {
		keepHost = g.fileIndex(g.highlight)
	}

	g.rebuildGroups()
	g.matches = nil

	browsing := g.browseHost >= 0
	browseFilter := browsing && g.groupSize(g.browseHost) >= 2
	nameFilter := g.searching && g.query != ""
	hide := g.hideDupes && !browsing
	if nameFilter || hide || browseFilter {
		needle := strings.ToLower(g.query)
		hostRep := g.RepresentativeOf(g.browseHost)
		g.matches = make([]int, 0, g.host.FileCount())
		for i := range g.host.FileCount() {
			if nameFilter && !strings.Contains(strings.ToLower(g.host.FileAt(i).Name()), needle) {
				continue
			}
			if browseFilter && g.RepresentativeOf(i) != hostRep {
				continue
			}
			if hide && g.IsHiddenExtra(i) {
				continue
			}
			g.matches = append(g.matches, i)
		}
	}

	g.filterGen.Add(1)

	// GridWrap's renderer does not exist until the overlay has been shown.
	// Hide-duplicates can turn on with the grid closed (viewer D), and a
	// hashRemaining completion can apply with it still closed; touching
	// wrap here would panic. Toggle scrolls and highlights when it opens.
	if g.visible {
		g.wrap.Refresh()
		if resetView {
			g.setHighlight(0)
			if g.count() > 0 {
				g.wrap.ScrollTo(0)
			}
		} else {
			g.restoreHighlight(keepHost)
		}
	} else if resetView {
		g.highlight = 0
	}

	g.syncTopBar()
}

// displayIndexOfHost maps a host index to a display index, or -1 when that
// file is not currently shown. Distinct from displayIndexOf, which returns
// 0 on a miss (finishBrowse's "scroll somewhere" fallback).
func (g *Overview) displayIndexOfHost(hostIdx int) int {
	if hostIdx < 0 {
		return -1
	}
	if g.matches == nil {
		if hostIdx >= g.host.FileCount() {
			return -1
		}
		return hostIdx
	}
	for i, h := range g.matches {
		if h == hostIdx {
			return i
		}
	}
	return -1
}

func (g *Overview) restoreHighlight(host int) {
	if g.count() == 0 {
		g.setHighlight(0)
		return
	}
	id := 0
	if d := g.displayIndexOfHost(host); d >= 0 {
		id = d
	} else if d := g.displayIndexOfHost(g.RepresentativeOf(host)); d >= 0 {
		id = d
	}
	if id >= g.count() {
		id = g.count() - 1
	}
	g.setHighlight(id)
}
```

Do not change `displayIndexOf` in `dupes.go`.

- [ ] **Step 4: Run Task 2's test plus search/hide tests that reset the viewport**

```bash
gofmt -w internal/ui/grid/search.go internal/ui/grid/dupes_test.go
go test -race -count=1 -run 'TestRebuildFilter_KeepsHighlightedHostWhenAnExtraDisappears|TestSetHideDuplicates_|TestSyncTopBar_|TestHandleRune_|TestApplyFilter_' ./internal/ui/grid/ -v
```

Expected: PASS. Search tests still land on highlight 0 because they call `applyFilter()` → `rebuildFilter(true)`.

- [ ] **Step 5: Do not commit.** Parent reviews. Particular check: `keepHost = g.fileIndex(g.highlight)` is **before** `g.matches = nil` / the rebuild loop, not after.

---

### Task 3: Every hide hash job applies; browse stays last-job

**Files:**
- Modify: `internal/ui/grid/dupes.go` (`hashRemaining` defer / `g.ui.Do` body, and its doc comment)
- Modify: `internal/ui/grid/grid.go` (`hashJobs` field comment, ~lines 199–203)
- Modify: `internal/ui/grid/dupes_test.go` (one-pending-job pool test)

**Interfaces:**
- Consumes: `rebuildFilter(false)` from Task 2; `g.ui.Do` already wrapping the last-job apply; `hashJobs` atomic still counting pool jobs.
- Produces: every hash job's `defer` enqueues one `g.ui.Do`. Inside that callback, after the generation check: if `browseHost >= 0` and `remaining == 0`, `finishBrowse()`; else if `browseHost >= 0`, return; else if `hideDupes`, `rebuildFilter(false)` then `jumpIfHiddenExtra()`. `remaining := g.hashJobs.Add(-1)` is captured **before** `g.ui.Do` so the last job is identified even if another worker decrements first… **No.** `Add(-1)` is atomic and must stay in the worker `defer` (today's place). Capture `remaining := g.hashJobs.Add(-1)` in the worker, close over that value in the `Do` callback. Last browse job is `remaining == 0`. Hide jobs apply regardless of `remaining`.

The `g.ui.Do` call stays inside the `g.decodes.Go` body (the `defer` already runs there). Do not spawn a new goroutine.

- [ ] **Step 1: Write the pool-driven progressive test**

One remaining hash job (the extra). Immediate apply from Task 1 leaves all three visible; unpark+Settle runs that one job, which must now apply without waiting for a "last of many" that is also the only one.

```go
func TestSetHideDuplicates_OnePendingJobHidesExtraWithoutWaitingForAPeer(t *testing.T) {
	host := hostPatterned(t,
		[]string{"sunset-a.jpg", "sunset-b.jpg", "moon.jpg"},
		[]int{1, 1, 99},
	)
	g := newOverview(t, host)
	g.rememberHash(host.files[0], mustThumb(t, host.files[0]))
	g.rememberHash(host.files[2], mustThumb(t, host.files[2]))

	unpark := parkDecodes(t, g)
	g.Toggle()
	g.SetHideDuplicates(true)

	if g.count() != 3 {
		t.Fatalf("count() = %d with the extra still unhashed, want 3", g.count())
	}
	if got := g.hashJobs.Load(); got != 1 {
		t.Fatalf("hashJobs = %d, want 1", got)
	}

	unpark()
	g.Settle()

	if g.count() != 2 {
		t.Fatalf("count() = %d after the one remaining job, want 2", g.count())
	}
	if !g.IsHiddenExtra(1) {
		t.Error("the extra should hide when its own job completes")
	}
}
```

Today this already PASSES at end-state because the one job *is* the last job. It is still required: after Task 3, hide apply no longer depends on `remaining == 0`, and this is the regression lock that a hide job with `remaining == 0` still applies (the last hide job must not skip `rebuildFilter` now that browse owns the `remaining == 0` branch). Keep it.

Add a second test that would FAIL today if we had two pending jobs and tried to observe the first completion — we cannot complete exactly one of two parked jobs without a finer harness. Do **not** build a finer harness. Task 2's direct `rebuildFilter(false)` plus this one-job pool test are the coverage.

- [ ] **Step 2: Run it (green against last-job-only, still required)**

```bash
go test -race -count=1 -run TestSetHideDuplicates_OnePendingJobHidesExtraWithoutWaitingForAPeer ./internal/ui/grid/ -v
```

Expected: PASS already. Do not skip the test.

- [ ] **Step 3: Change `hashRemaining`'s completion**

Replace `hashRemaining`'s function comment and the `g.decodes.Go` body in `dupes.go` with:

```go
// hashRemaining hashes every file that does not already have a dHash.
// Cache hits run on this goroutine; the rest join the thumbnail pool
// without a per-cell Claim so Settle still waits, and they do not Add to
// a full thumbnail cache. Each pool job applyFilters hide-duplicates
// through g.ui.Do (rebuildFilter(false)) so extras disappear as hashes
// land. Browse still waits for the last job (finishBrowse) so a partial
// group is never shown. g.ui.Do stays inside this Go body: Settle's
// barrier is decodes.Wait, which only covers completions the pool spawned.
func (g *Overview) hashRemaining() int {
	gen := g.host.Generation()
	g.wipeHashesIfStale()

	type hashJob struct {
		file fyne.URI
		key  string
	}
	var jobs []hashJob
	for i := 0; i < g.host.FileCount(); i++ {
		u := g.host.FileAt(i)
		if _, ok := g.hashOf(u); ok {
			continue
		}
		if g.hashFailedOf(u) {
			continue
		}
		if thumb, ok := g.thumbs.Get(u.String()); ok {
			g.rememberHash(u, thumb)
			continue
		}
		key := u.String()
		if _, loaded := g.hashing.LoadOrStore(key, true); loaded {
			continue
		}
		jobs = append(jobs, hashJob{file: u, key: key})
	}
	n := len(jobs)
	if n == 0 {
		return 0
	}
	g.hashJobs.Add(int32(n))
	for _, j := range jobs {
		file, key := j.file, j.key
		g.decodes.Go(context.Background(), func(acquired bool) {
			defer func() {
				g.hashing.Delete(key)
				remaining := g.hashJobs.Add(-1)
				g.ui.Do(func() {
					if gen != g.host.Generation() {
						return
					}
					if g.browseHost >= 0 {
						if remaining == 0 {
							g.finishBrowse()
						}
						return
					}
					if g.hideDupes {
						g.rebuildFilter(false)
						g.jumpIfHiddenExtra()
					}
				})
			}()
			if !acquired || gen != g.host.Generation() {
				return
			}
			thumb, err := imaging.LoadThumbnail(file)
			if err != nil || thumb == nil {
				g.rememberHashFail(file)
				return
			}
			g.rememberHash(file, thumb)
			if !g.ThumbCacheFull() {
				g.thumbs.AddIfFits(file.String(), thumb)
			}
		})
	}
	return n
}
```

In `grid.go`, replace the `hashJobs` clause of the field comment (the sentence that currently says `hashJobs counts those pool jobs so only the last one applyFilters (Fyne's test driver runs Do inline on the worker).`) with:

```go
	// hashing dedups in-flight hashRemaining jobs by URI string. hashJobs
	// counts those pool jobs so the last one can finishBrowse; hide applies
	// on every job via g.ui.Do.
```

Keep the rest of that comment block (`hideDupes`, `browseHost`, `dupeDist`, `groupSizes`) intact.

- [ ] **Step 4: Run the hide/browse hash suite**

```bash
gofmt -w internal/ui/grid/dupes.go internal/ui/grid/grid.go internal/ui/grid/dupes_test.go
go test -race -count=1 -run 'TestSetHideDuplicates_|TestSetBrowsingDuplicates_|TestApplyFilter_BrowsePending|TestRebuildFilter_|TestSetDuplicateDistance_' ./internal/ui/grid/ -v
go test -race -count=5 ./internal/ui/grid/
```

Expected: PASS, including `-count=5`. Browse tests still toast once and finish only after Settle. Hide chrome test from Task 1 still ends at `count() == 2`.

- [ ] **Step 5: Do not commit.** Parent reviews: every `g.ui.Do` is inside `decodes.Go`; browse does not `rebuildFilter` on intermediate jobs; hide does not require `remaining == 0`.

---

### Task 4: `SetUIQueue` and install it in `newTestUI`

**Files:**
- Modify: `internal/ui/grid/uiqueue.go`
- Modify: `internal/ui/grid/grid.go` (field type `ui uiQueue` → `ui UIQueue` if the interface is renamed there; `New` still `ui: fyneQueue{}`)
- Modify: `internal/ui/grid/harness_test.go` (`newOverview` uses `SetUIQueue`)
- Modify: `internal/ui/harness_test.go` (`newTestUI` installs the queue)

**Interfaces:**
- Consumes: existing unexported `uiQueue` with `Do(func())` and `Drain() bool`; `fyneQueue`; `*uitest.UIQueue`.
- Produces:
  - Exported `type UIQueue interface { Do(func()); Drain() bool }` (rename of `uiQueue`; delete the old name — do not keep both).
  - `func (g *Overview) SetUIQueue(q UIQueue)` — `nil` restores `fyneQueue{}`. Production `New` still installs `fyneQueue{}`. App tests call this; the app itself never does.
  - `newOverview`: `g.SetUIQueue(&uitest.UIQueue{})` instead of `g.ui = &uitest.UIQueue{}`.
  - `newTestUI`: after `v, win = buildStartupViewer(testApp)` (and the existing toast/vector/wallpaper tweaks are fine either before or after), `v.grid.SetUIQueue(&uitest.UIQueue{})`. Import `"github.com/frathe/picfetch/internal/uitest"` in `harness_test.go` if not already present.

Update `uiqueue.go`'s package comment that currently says internal/ui "must keep the app's own behaviour, which it does because New installs fyneQueue". After this task that sentence is false. Replace the last paragraph of the `UIQueue` / `uiQueue` doc with: production `New` installs `fyneQueue`; `internal/ui/grid`'s tests and `internal/ui`'s `newTestUI` install `*uitest.UIQueue` via `SetUIQueue` so Drain/Settle run on the test goroutine.

- [ ] **Step 1: Write a tiny setter test in `internal/ui/grid`**

Add `internal/ui/grid/uiqueue_test.go`:

```go
package grid

import (
	"testing"

	"github.com/frathe/picfetch/internal/uitest"
)

func TestSetUIQueue_NilRestoresFyneQueue(t *testing.T) {
	g := newOverview(t, hostWith(t, "a.jpg"))
	g.SetUIQueue(nil)
	if _, ok := g.ui.(fyneQueue); !ok {
		t.Fatalf("SetUIQueue(nil) ui type = %T, want fyneQueue", g.ui)
	}
	g.SetUIQueue(&uitest.UIQueue{})
	if _, ok := g.ui.(*uitest.UIQueue); !ok {
		t.Fatalf("SetUIQueue(&uitest.UIQueue{}) ui type = %T, want *uitest.UIQueue", g.ui)
	}
}
```

- [ ] **Step 2: Confirm RED**

```bash
go test -count=1 -run TestSetUIQueue_NilRestoresFyneQueue ./internal/ui/grid/ -v
```

Expected: FAIL compile: `SetUIQueue undefined`.

- [ ] **Step 3: Export the interface and setter**

`internal/ui/grid/uiqueue.go` becomes:

```go
// The seam a finished background job hands its widget work across, and the
// app's own implementation of it.

package grid

import "fyne.io/fyne/v2"

// UIQueue is how a decode-pool worker gets its result onto the UI
// goroutine. fyneQueue in the app; tests install *uitest.UIQueue via
// SetUIQueue, because Fyne's test driver is not a marshaling point - its
// DoFromGoroutine runs the callback inline on the calling (worker)
// goroutine, so the completion bodies would otherwise touch canvas.Image,
// widget.Label, g.matches and g.groupSizes concurrently with the test
// goroutine that spawned the worker.
//
// A field on Overview rather than a package-level var, for the reason
// AGENTS.md gives: it is per-instance configuration. Production New
// installs fyneQueue. internal/ui/grid's newOverview and internal/ui's
// newTestUI replace it with *uitest.UIQueue so Drain/Settle run on the
// test goroutine.
type UIQueue interface {
	// Do arranges for f to run on the UI goroutine. It may return before
	// f has run, and must not run f on the calling goroutine.
	Do(f func())

	// Drain runs whatever Do deferred, on the calling goroutine, and
	// reports whether it ran anything. Always false for a queue backed by
	// a real UI goroutine: that goroutine drains itself.
	Drain() bool
}

// SetUIQueue replaces the marshaler background completions use. Production
// never calls this. Tests pass *uitest.UIQueue so completions are drained
// by Settle on the test goroutine. A nil q restores the app's fyneQueue.
func (g *Overview) SetUIQueue(q UIQueue) {
	if q == nil {
		g.ui = fyneQueue{}
		return
	}
	g.ui = q
}

// fyneQueue is the app's UIQueue - hand the callback to Fyne and let the
// driver marshal it onto the UI goroutine. Nothing here to drain.
type fyneQueue struct{}

func (fyneQueue) Do(f func()) { fyne.Do(f) }

func (fyneQueue) Drain() bool { return false }
```

In `grid.go`, the field `ui uiQueue` becomes `ui UIQueue`. `New` still has `ui: fyneQueue{}`.

In `harness_test.go` `newOverview`, replace `g.ui = &uitest.UIQueue{}` with `g.SetUIQueue(&uitest.UIQueue{})`.

In `internal/ui/harness_test.go`, add the `uitest` import and, immediately after `v, win = buildStartupViewer(testApp)`:

```go
	v.grid.SetUIQueue(&uitest.UIQueue{})
```

Do not change `features.go`. Production stays on `fyneQueue`.

- [ ] **Step 4: Run grid tests, then the app suite**

```bash
gofmt -w internal/ui/grid/uiqueue.go internal/ui/grid/uiqueue_test.go internal/ui/grid/grid.go internal/ui/grid/harness_test.go internal/ui/harness_test.go
go test -race -count=1 ./internal/ui/grid/ ./internal/uitest/
go test -race -count=1 -timeout 1800s ./internal/ui/
```

Expected: PASS. `internal/ui` is ~4 minutes. If a test now fails because it Toggled a cold grid and asserted a painted cell without `Settle`, that is a real fallout of the queue — fix by calling `warmThumbs` (already the package convention) or `v.grid.Settle()` before the assertion. Do not revert `SetUIQueue`. Do not reintroduce last-job-only.

- [ ] **Step 5: Do not commit.** Parent reviews: `internal/ui/grid` production `.go` files still do not import `uitest`; `features.go` untouched.

---

### Task 5: Docs, todos, full-suite gate

**Files:**
- Modify: `ARCHITECTURE.md` (hide-duplicates index entry; grid package blurb if it still implies last-job apply)
- Modify: `AGENTS.md` (Concurrency and Fyne bullet: app tests install the queue too)
- Modify: `todos.md` (move both follow-up TODOs into Done)

**Interfaces:** none.

- [ ] **Step 1: Edit `ARCHITECTURE.md`**

In the hide-duplicates "where to look" bullet (~line 643), after the sentence about extras being a display filter, add one sentence:

`SetHideDuplicates` applies immediately (chrome shows while hashes are pending; unhashed files stay visible). Each `hashRemaining` job rebuilds the hide filter through `g.ui.Do` without resetting the highlight; `Shift+D` browse still waits for the last job before `finishBrowse`.

In the `grid/` package-map paragraph, keep the `uiQueue` explanation and add that `internal/ui`'s `newTestUI` also installs `*uitest.UIQueue` via `SetUIQueue`.

- [ ] **Step 2: Edit `AGENTS.md`**

The concurrency bullet that says `internal/ui/grid` marshals through `uiQueue` so *its* tests can drain completions: extend it so it also names `internal/ui`'s `newTestUI` (`v.grid.SetUIQueue(&uitest.UIQueue{})`). The "do not simplify back to a direct `fyne.Do`" rule stays. The "every `g.ui.Do` from inside `g.decodes.Go`" rule stays.

- [ ] **Step 3: Edit `todos.md`**

Remove these two ACTIVE DEVELOPMENT items:

- `grid: undo the production compromises made for the inline test driver`
- `grid: no test covers the middle of the hashing window`

Add a Done entry (same style as the uiQueue Done paragraph) stating: hide-duplicates applies immediately and progressively; browse still waits for the last hash job; `newTestUI` installs `uitest.UIQueue`; mid-window coverage is `TestSetHideDuplicates_PendingShowsChromeAndLeavesUnhashedVisible`.

- [ ] **Step 4: Full gate**

```bash
gofmt -l .
go vet ./...
go build ./...
go test -race -count=1 -timeout 1800s ./...
```

Expected: `gofmt -l .` empty, vet/build clean, all 29 packages ok, 0 data races.

- [ ] **Step 5: Do not commit.** Parent suggests one commit message covering predecessor uiQueue work *and* this follow-up if Florian wants them together, or two if they prefer to split.

---

## Self-review (planner)

1. **Spec coverage:** Immediate chrome → Task 1. Progressive hide without viewport yank → Tasks 2–3. Browse last-job preserved → Task 3. App-suite drainable queue → Task 4. Mid-hash window → Task 1's `hashJobs == 2` test. Docs/todos → Task 5.
2. **Placeholders:** none. `rebuildFilter(false)` is fully specified including host capture order.
3. **Type consistency:** `UIQueue` exported in Task 4; `SetUIQueue(q UIQueue)`; `rebuildFilter(resetView bool)` unexported; `hashJobs.Load()` for pending counts; chrome string `lang.L("Hiding duplicates")`.
4. **Out of scope:** dHash/area-average, distance default, Windows Ctrl+click, golden screenshots, mutex "fixes", changing browse toast, importing `uitest` into production grid.

## Suggested commit message (parent, after Task 5, not for implementers)

```
Fix grid hide-duplicates stalling until every thumbnail has hashed.

D now shows "Hiding duplicates" immediately and extras drop out as hashes
land, without yanking the highlight back to the top. Browse still waits
for the last job. App tests drain grid completions through uitest.UIQueue
the same way the grid package already did.
```
