# finishLoad Named Steps Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Controller extra (this session):** After every task, the parent agent reviews the full diff and fixes it before dispatching the next task. Do not start Task N+1 until that review lands. Do not commit (`AGENTS.md`). End with a suggested commit message for the user. Dispatch **one implementer at a time** — these tasks edit the same function.

**Goal:** Turn `viewer.finishLoad` in `internal/ui/load.go` from a ~117-line linear pipeline into a short orchestrator that calls named unexported steps, with no behavior change.

**Architecture:** Extract-method only. Helpers stay unexported methods on `viewer` in the same file, immediately after `finishLoad`. Call order, comments that explain *why* (test-driver races, cache mutation, fade sequencing), and the `preloadNeighbors` → `done()` pairing stay exactly as they are today. Do not introduce a pipeline type, a new package, or a shared helper with `applyRotationLayout`.

**Tech Stack:** Go (see `go.mod`), Fyne v2, existing `internal/ui` test harness (`newTestViewer` / `newTestUI`, `dropAndWait`, `waitUntilLoaded`). No new dependencies.

## Status check (2026-08-25) — this todo is old but not done

Checked against the tree before writing this plan. `todos.md` still lists `finishLoad` under `## TODO`. `internal/ui/load.go:191-308` is still one function. Recent work (`ARCHITECTURE.md` trim, toast import grouping, `goimports` CI) explicitly left this bullet alone with the note “decompose only if it needs to change anyway.” This plan is the user choosing to do that decomposition now, as a readability refactor, without waiting for a behavior change.

| `todos.md` TODO | Still open? | This plan |
|-----------------|-------------|-----------|
| `finishLoad` (`internal/ui/load.go:192-305`) | **Yes.** One function, lines 191–308 (~117 lines of body + doc). | **This plan.** |
| `internal/imaging/exif.go` (~687 lines) | Yes. Still one file. | **Out of scope.** Parse/format split is cosmetic. |

If the user instead wants the `exif.go` split, stop; do not execute this plan.

## Approaches considered

1. **Named unexported `viewer` methods in `load.go` (this plan).** Matches the todo (“decompose into named steps”). Each comment block that explains a constraint becomes the helper that owns that constraint. `finishLoad` becomes the ordered list. Reviewers can reject a single extract without touching the rest.
2. **New `finishload.go`.** Same methods, different file. `load.go` is already the load/display/preload/animate file (~628 lines); splitting the steps away from `ShowImage`/`attemptLoad`/`preloadNeighbors` makes the call chain harder to read. Rejected.
3. **Pipeline struct / `finishLoad` object with a `Run` method.** Heavy for a function that has one caller shape (cache hit + decode completion) and no branching policy beyond what the steps already contain. Rejected (YAGNI).
4. **Unify window fit with `applyRotationLayout` (`rotate.go`).** They look similar and must not be merged: `finishLoad` skips resize when the **grid is visible** (batch-delete re-show) and sizes from **unrotated** `loaded.Frames[0].Bounds()` after forcing `rotation = 0`; `applyRotationLayout` does **not** skip the grid, rotates vector logical axes, and sizes from `displayedDimensions()`. Unifying would be a behavior change or a helper with so many flags it is worse than duplication. Rejected.

## Locked decisions (confirm or override before Task 1)

These are the plan defaults. Subagents must not reopen them.

1. **Helpers live in `internal/ui/load.go`**, immediately after `finishLoad`, before `preloadNeighbors`.
2. **Drop the unused `i int` parameter** (`_ int` today). `attemptLoad` already wrote `v.state.index` before both call sites. Title/index use `v.state.index`, not the argument.
3. **Do not share code with `applyRotationLayout`.**
4. **No new tests whose only job is to prove a helper exists.** This is extract-method. The public-path tests listed per task *are* the spec. Do not add tests that call unexported helpers just to assert they were named.
5. **TDD of “write a failing test for a missing helper” does not apply.** Inventing a test that fails because `installLoadedFrames` is undefined is not a behavior spec. Each task: run covering tests (green) → extract with comments moved, not deleted → run covering tests (still green).
6. **Do not commit.** Suggested commit message only, at the end of Task 5.

## Global Constraints

- Do not commit. `AGENTS.md`: “Do not run `git commit`. End with a suggested commit message for the user.”
- Do not change `internal/imaging/exif.go`, menu code, or any file other than those listed in the active task.
- Do not add `TODO`/`FIXME` comments to source. Open work stays in `todos.md`.
- Do not update `ARCHITECTURE.md` except the one locator sentence in Task 5 (no new package, no file move until that sentence).
- Preserve existing comments that state constraints the code cannot show (test-driver races, ByteCache mutation, fade sequencing, `done()` ordering). Move them onto the extracted function. Do not replace a why-comment with the function name alone.
- Preserve `gofmt` / `goimports -local github.com/frathe/picfetch` grouping.
- Do not add goroutines, package-level test seams, or a second `fyne.Do` hop. `startLoadedAnimation` must keep `go v.animate(...)` after `ForceRepaint`, same as today.
- Subagents must not start Task N+1 themselves. They stop after their task’s verification and report.
- **Load-bearing call order in `finishLoad` (do not reorder, even if tests stay green):**

```
installLoadedFrames        // displayFrames + vector copy-on-write + rotation/frame idx = 0
presentLoadedImage         // slideshow: Translucency=1 → redraw → fade-in; then show/hide widgets
syncLoadedFileInfo         // overlay/EXIF after pixels exist
fitWindowToLoadedImage     // ResetToFit always; resize unless slideshow or grid
applyLoadedTitle           // title + SetAnimDuration (0 for static, so GIF duration cannot leak)
clearLoadingChrome         // loading=false, bar hide, updateFileMenuState, ForceRepaint
startLoadedAnimation       // AFTER ForceRepaint; test driver would race if spawned first
preloadNeighbors(token)    // BEFORE done(); waiters may mutate v.state.files once done() runs
done()
```

## Subagent models

Use the least powerful listed model that can handle the role. Available slugs: `composer-2.5-fast`, `cursor-grok-4.5-high-fast`, `cursor-grok-4.6-xhigh`, `claude-opus-5-thinking-high`.

Implementers use `subagent_type: go-expert`. Reviewers use `subagent_type: go-expert`. Do **not** use Opus for any implementation task — the work splits cleanly. Opus is reserved for the final whole-branch review (subtle ordering / comment integrity across the whole function).

| Role | Model | Why |
|------|--------|-----|
| Task 1 implementer | `cursor-grok-4.6-xhigh` | First extract sets the comment-migration pattern and the vector copy-on-write invariant. |
| Task 2 implementer | `cursor-grok-4.5-high-fast` | Mechanical extract once Task 1’s pattern exists; fade order is already specified in the brief. |
| Task 3 implementer | `cursor-grok-4.5-high-fast` | Mechanical extract; grid-skip and title/`SetAnimDuration` coupling are specified. |
| Task 4 implementer | `cursor-grok-4.6-xhigh` | `ForceRepaint`→animate and `preloadNeighbors`→`done()` are load-bearing; also drops `_ int`. |
| Task 5 implementer | `cursor-grok-4.5-high-fast` | Docs + full suite; no design. |
| Task reviewer (Tasks 1, 4) | `cursor-grok-4.6-xhigh` | Must catch comment loss and order swaps the suite may not fail on. |
| Task reviewer (Tasks 2, 3, 5) | `cursor-grok-4.5-high-fast` | Mid-tier floor; diffs are transcription. |
| Parent review / fix after each task | this session (do not dispatch) | User asked the parent to review and fix after every step. |
| Final whole-branch review | `claude-opus-5-thinking-high` | One pass over the finished orchestrator vs the order table. Too easy for a cheap model to rubber-stamp. |

## File structure

- Modify: `internal/ui/load.go` — `finishLoad` plus new helpers; two `finishLoad` call sites in `attemptLoad`.
- Modify: `todos.md` — move the `finishLoad` bullet to `## Done` (Task 5).
- Modify: `ARCHITECTURE.md` — one locator wording tweak on the `load.go` row (Task 5).
- Do not create: `internal/ui/finishload.go`.
- Do not modify: `internal/ui/rotate.go`, `internal/ui/vector.go`, tests (unless a task’s extract accidentally breaks one — then fix the extract, not the test).

## Target signatures (all tasks)

```go
func (v *viewer) finishLoad(token requestToken, u fyne.URI, loaded *imaging.LoadedImage, done func())

func (v *viewer) installLoadedFrames(loaded *imaging.LoadedImage)
func (v *viewer) presentLoadedImage()
func (v *viewer) syncLoadedFileInfo(loaded *imaging.LoadedImage)
func (v *viewer) fitWindowToLoadedImage(loaded *imaging.LoadedImage)
func (v *viewer) applyLoadedTitle(u fyne.URI, loaded *imaging.LoadedImage)
func (v *viewer) clearLoadingChrome()
func (v *viewer) startLoadedAnimation(token requestToken, loaded *imaging.LoadedImage)
```

`preloadNeighbors` already exists and stays as-is. Do not wrap `done()`.

After Task 4, `finishLoad` must be exactly this body (comments may be slightly tighter but must still state the two test-driver constraints):

```go
func (v *viewer) finishLoad(token requestToken, u fyne.URI, loaded *imaging.LoadedImage, done func()) {
	v.installLoadedFrames(loaded)
	v.presentLoadedImage()
	v.syncLoadedFileInfo(loaded)
	v.fitWindowToLoadedImage(loaded)
	v.applyLoadedTitle(u, loaded)
	v.clearLoadingChrome()
	v.startLoadedAnimation(token, loaded)
	v.preloadNeighbors(token)
	done()
}
```

---

### Task 1: Extract `installLoadedFrames`

**Model:** `cursor-grok-4.6-xhigh` (implementer), `cursor-grok-4.6-xhigh` (task reviewer)

**Files:**
- Modify: `internal/ui/load.go` — `finishLoad` body start; add helper after `finishLoad`
- Test: existing `internal/ui/vector_test.go` (do not edit)

**Interfaces:**
- Consumes: `loaded *imaging.LoadedImage` with `Frames`, `Vector`; `v.clearVector`; `v.zoom.SetLogicalSize`; `v.vector.{svg,logical,raster}`
- Produces: `installLoadedFrames` as specified. Leaves `displayFrameIdx == 0` and `rotation == 0`. Does **not** change `finishLoad`’s signature yet (`_ int` stays until Task 4).

- [ ] **Step 1: Run covering tests on the current tree**

Run:

```bash
go test -race -count=1 ./internal/ui/ -run 'TestSVGReRenderNeverMutatesTheCachedEntry|TestSVGThenRasterClearsVectorState|TestRasterFormatKeepsNoVectorState|TestSVGDisplaysAtLogicalSize|TestCloseFilesClearsVector|TestViewerShow_LoadsAndNavigates'
```

Expected: PASS. If anything fails before your edit, stop and report BLOCKED (pre-existing).

- [ ] **Step 2: Extract `installLoadedFrames`**

Replace the top of `finishLoad` (today: bounds, `displayFrames`, `clearVector`, vector branch, `displayFrameIdx = 0`, `rotation = 0`) with `v.installLoadedFrames(loaded)`.

Insert this helper immediately after `finishLoad` (before `preloadNeighbors`). Preserve the cache-mutation comment verbatim on the helper, not only in `finishLoad`:

```go
// installLoadedFrames copies loaded onto the viewer's display buffers and
// resets view-only rotation and GIF frame index for a fresh navigation.
//
// A vector's frame is replaced in place by every re-render, so it
// must not share the backing array of the cached LoadedImage -
// writing through that would mutate the cache entry and invalidate
// the byte weight ByteCache computed for it.
func (v *viewer) installLoadedFrames(loaded *imaging.LoadedImage) {
	b := loaded.Frames[0].Bounds()

	v.displayFrames = loaded.Frames
	v.clearVector()

	if loaded.Vector != nil {
		v.displayFrames = []image.Image{loaded.Frames[0]}

		v.vector.svg = loaded.Vector
		v.vector.logical = fyne.NewSize(float32(b.Dx()), float32(b.Dy()))
		v.vector.raster = image.Pt(b.Dx(), b.Dy())
		v.zoom.SetLogicalSize(v.vector.logical)
	}

	v.displayFrameIdx = 0
	v.rotation = 0
}
```

Do not compute window size or title here. Do not call `redrawRotatedFrame`.

After this task, `finishLoad` still contains the fade/overlay/zoom/title/anim/preload inline, and still starts with `b := loaded.Frames[0].Bounds()` **or** uses `loaded.Frames[0].Bounds()` at the remaining resize/title sites. Either is fine: later tasks switch those sites to `loaded` themselves. If you keep a local `b`, it must remain `loaded.Frames[0].Bounds()` (unrotated source), not `v.displayFrames[0]` after a later rotation — rotation is now 0, so they match, but do not start reading `v.img.Image.Bounds()`.

- [ ] **Step 3: Re-run covering tests**

Same command as Step 1. Expected: PASS.

- [ ] **Step 4: Stop**

Do not extract further steps. Do not commit. Report DONE with the test command and output summary.

---

### Task 2: Extract `presentLoadedImage` and `syncLoadedFileInfo`

**Model:** `cursor-grok-4.5-high-fast` (implementer), `cursor-grok-4.5-high-fast` (task reviewer)

**Files:**
- Modify: `internal/ui/load.go` — next two blocks of `finishLoad`; two helpers after `installLoadedFrames`
- Test: existing `internal/ui/slideshow_test.go`, `internal/ui/load_test.go` (do not edit)

**Interfaces:**
- Consumes: Task 1’s `installLoadedFrames`. `v.slides.Active`, `v.startFade`, `v.redrawRotatedFrame`, `v.syncInfoOverlayVisibility`, `v.exif.Refresh`
- Produces: `presentLoadedImage`, `syncLoadedFileInfo` as specified

- [ ] **Step 1: Run covering tests**

```bash
go test -race -count=1 ./internal/ui/ -run 'TestShowImage_InPictureFrameModeEndsFullyOpaque|TestTogglePictureFrameMode_ExitResetsFade|TestViewerShow_LoadsAndNavigates|TestSVGReRenderNeverMutatesTheCachedEntry'
```

Expected: PASS.

- [ ] **Step 2: Extract `presentLoadedImage`**

Move the slideshow fade-swap and widget show/hide (today: `Translucency = 1` if slides, `redrawRotatedFrame`, `startFade(1, 0)` if slides, `img.Show`, `dropzone.Hide`, `emptyStateArt.Hide`) into:

```go
// presentLoadedImage puts loaded pixels on the canvas and hides the
// drop-zone / empty-state chrome.
//
// In picture-frame mode, the outgoing image was left fading toward
// invisible by ShowImage's startFade(0, 1) (or already is, if that
// fade had time to finish); forcing it the rest of the way there
// right before the swap hides the new pixels landing mid-fade, then
// the fade-in takes over from a clean, fully-invisible start.
func (v *viewer) presentLoadedImage() {
	if v.slides.Active() {
		v.img.Translucency = 1
	}
	v.redrawRotatedFrame()
	if v.slides.Active() {
		v.startFade(1, 0)
	}
	v.img.Show()
	v.dropzone.Hide()
	v.emptyStateArt.Hide()
}
```

The inner order `Translucency = 1` → `redrawRotatedFrame` → `startFade(1, 0)` is load-bearing. Do not collapse the two `slides.Active()` checks into one block that fades before redraw.

- [ ] **Step 3: Extract `syncLoadedFileInfo`**

Move `currentFileSize` / `currentHasEXIF` / `currentPreview` / `syncInfoOverlayVisibility` / `exif.Refresh` into:

```go
func (v *viewer) syncLoadedFileInfo(loaded *imaging.LoadedImage) {
	v.currentFileSize = loaded.FileSize
	v.currentHasEXIF = loaded.HasEXIF
	v.currentPreview = loaded.Preview
	v.syncInfoOverlayVisibility()
	v.exif.Refresh()
}
```

Call it after `presentLoadedImage`, not before (pixels first, then overlay/EXIF panel).

- [ ] **Step 4: Re-run covering tests**

Same command as Step 1. Expected: PASS.

- [ ] **Step 5: Stop**

Do not extract zoom/title/anim. Do not commit.

---

### Task 3: Extract `fitWindowToLoadedImage` and `applyLoadedTitle`

**Model:** `cursor-grok-4.5-high-fast` (implementer), `cursor-grok-4.5-high-fast` (task reviewer)

**Files:**
- Modify: `internal/ui/load.go`
- Test: existing `internal/ui/batch_test.go`, `internal/ui/load_test.go`, `internal/ui/slideshow_test.go`, `internal/ui/windowsize_test.go` (do not edit)

**Interfaces:**
- Consumes: `v.zoom.ResetToFit`, `v.undoGridMaximize`, `resizeToImage`, `v.slides.SetAnimDuration`, `v.setTitle`, `lang.L`
- Produces: `fitWindowToLoadedImage`, `applyLoadedTitle` as specified

- [ ] **Step 1: Run covering tests**

```bash
go test -race -count=1 ./internal/ui/ -run 'TestBatchDelete_LeavesTheWindowMaximized$|TestBatchDelete_LeavesTheWindowMaximizedOnAColdReload|TestViewerShow_RAWPreviewMarksTheTitle|TestShow_TracksAnimatedGIFLoopDuration|TestResizeToImage|TestViewerShow_LoadsAndNavigates'
```

Expected: PASS. (`TestBatchDelete_LeavesTheWindowMaximized$` avoids the ColdReload name accidentally matching twice; both names are listed explicitly anyway.)

- [ ] **Step 2: Extract `fitWindowToLoadedImage`**

Move `zoom.ResetToFit` and the conditional window resize. Keep the picture-frame / grid skip comments on the helper:

```go
// fitWindowToLoadedImage starts every navigation at fit-to-window and
// resizes the window to the new image, except when that resize would
// fight an overlay that already owns the window size.
//
// ResetToFit is applied directly (not just left for the resize below
// to trigger) since picture-frame mode skips that resize entirely.
//
// In picture-frame mode the window is already full-screen and
// ImageFillContain scales the image to fit it without stretching, so
// there's nothing to resize to - and resizing a full-screen window is
// asking for platform-specific trouble. The grid overview is skipped on
// the same grounds: it fills the window it maximized, and undoGridMaximize
// would shrink that window while the grid is still drawn over it.
func (v *viewer) fitWindowToLoadedImage(loaded *imaging.LoadedImage) {
	v.zoom.ResetToFit()

	if !v.slides.Active() && !v.grid.Visible() {
		b := loaded.Frames[0].Bounds()
		v.undoGridMaximize()
		resizeToImage(v.win, b, v.settings.maxWinW, v.settings.maxWinH)
	}
}
```

Do **not** call `applyRotationLayout`. Do **not** drop the `!v.grid.Visible()` conjunct. Size from `loaded.Frames[0].Bounds()`, not `displayedDimensions()` (rotation is 0, but stay identical to today’s source).

- [ ] **Step 3: Extract `applyLoadedTitle`**

Move title construction and `SetAnimDuration`. Keep the “GIF duration must not leak into the next static image” comment:

```go
func (v *viewer) applyLoadedTitle(u fyne.URI, loaded *imaging.LoadedImage) {
	b := loaded.Frames[0].Bounds()
	title := fmt.Sprintf("%s — %d x %d", u.Name(), b.Dx(), b.Dy())
	if loaded.Preview {
		title += " " + lang.L("(preview)")
	}

	// The slideshow uses this so an animated GIF always gets to play at
	// least one full loop before auto-advancing - see
	// internal/ui/slideshow. Set unconditionally (0 for a static image) so
	// a GIF's duration never leaks into the next, static image.
	animDuration := time.Duration(0)
	if len(loaded.Frames) > 1 {
		title += " (animated)"
		for _, d := range loaded.Delays {
			animDuration += d
		}
	}
	v.slides.SetAnimDuration(animDuration)

	if n := len(v.state.files); n > 1 {
		title = fmt.Sprintf("%s  (%d/%d)", title, v.state.index+1, n)
	}

	v.setTitle(title)
}
```

Use `len(loaded.Frames)`, not `len(v.displayFrames)`: a vector’s display slice is one element even though that does not matter today (vectors are not GIF), and matching the old source avoids a silent spec change.

Do not spawn `animate` here.

- [ ] **Step 4: Re-run covering tests**

Same command as Step 1. Expected: PASS.

- [ ] **Step 5: Stop**

Do not touch loading-bar / `ForceRepaint` / `animate` / `done()`. Do not commit.

---

### Task 4: Extract remaining steps, drop unused `i`, finish the orchestrator

**Model:** `cursor-grok-4.6-xhigh` (implementer), `cursor-grok-4.6-xhigh` (task reviewer)

**Files:**
- Modify: `internal/ui/load.go` — `finishLoad` signature and both call sites in `attemptLoad`; add `clearLoadingChrome` and `startLoadedAnimation`; leave `preloadNeighbors(token)` then `done()` in `finishLoad`
- Test: existing `internal/ui/animate_test.go`, `internal/ui/imgcache_test.go`, `internal/ui/save_test.go` (do not edit)

**Interfaces:**
- Consumes: Tasks 1–3 helpers. `v.anim.Begin`, `v.animate`, `v.ForceRepaint`, `v.updateFileMenuState`, `v.preloadNeighbors`
- Produces: final `finishLoad` signature and body as in “Target signatures”. Call sites:

```go
v.finishLoad(token, u, loaded, done)
```

(both the cache-hit path and the decode-completion path in `attemptLoad`).

- [ ] **Step 1: Run covering tests**

```bash
go test -race -count=1 ./internal/ui/ -run 'TestViewerShow_AnimatesGIF|TestViewerShow_NavigatingAwayStopsAnimation|TestInvalidateLoad_WakesAnimateImmediately|TestFinishLoad_PreloadsBothNeighbors|TestAttemptLoad_CacheHitServesFileRemovedFromDisk|TestCanSaveRotation_FalseWhileLoading|TestBatchDelete_LeavesTheWindowMaximized'
```

Expected: PASS.

- [ ] **Step 2: Extract `clearLoadingChrome` and `startLoadedAnimation`**

```go
func (v *viewer) clearLoadingChrome() {
	v.loading.Store(false)
	v.loadingBar.Hide()
	v.updateFileMenuState() // rotation just reset to 0, and loading has just cleared - see canSaveRotation
	v.ForceRepaint()
}

// startLoadedAnimation runs only after clearLoadingChrome's ForceRepaint.
// Animated GIFs keep playing until a newer load request (a navigation or
// a fresh drop) supersedes this one; animate checks the shared token and
// waits on its context. Under the real driver both go through the same
// serialized fyne.Do queue either way, but the fyne test driver runs
// fyne.Do synchronously on the calling goroutine, so spawning animate
// first let its own first-frame Refresh race with this goroutine's
// still-running ForceRepaint.
func (v *viewer) startLoadedAnimation(token requestToken, loaded *imaging.LoadedImage) {
	if len(loaded.Frames) <= 1 {
		return
	}
	stopped := v.anim.Begin()
	go v.animate(token, loaded.Frames, loaded.Delays, stopped)
}
```

Keep `startLoadedAnimation` as a **separate call after** `clearLoadingChrome` in `finishLoad`. Do not fold `go v.animate` into `clearLoadingChrome`. Do not call `done()` inside either helper.

- [ ] **Step 3: Drop `_ int` and collapse `finishLoad` to the orchestrator**

New signature (update the existing doc comment so it still describes cache-hit vs decode-completion sharing, and add that the body is ordered steps whose constraints live on the helpers):

```go
func (v *viewer) finishLoad(token requestToken, u fyne.URI, loaded *imaging.LoadedImage, done func()) {
	v.installLoadedFrames(loaded)
	v.presentLoadedImage()
	v.syncLoadedFileInfo(loaded)
	v.fitWindowToLoadedImage(loaded)
	v.applyLoadedTitle(u, loaded)
	v.clearLoadingChrome()
	v.startLoadedAnimation(token, loaded)
	// Must run - and finish reading v.state.files/v.state.index - before the
	// load signal finishes below: that finish is what a waiter (a test's
	// waitUntilLoaded, or a future navigation) synchronizes on to know
	// this call is done touching viewer state. Under the fyne test
	// driver, this whole function already runs on whatever goroutine
	// called fyne.Do rather than a dedicated UI goroutine (see
	// attemptLoad's token comment), so finishing the signal first would
	// let a waiter go on to mutate v.state.files - via reset() or a fresh
	// drop - concurrently with this read.
	v.preloadNeighbors(token)
	done()
}
```

Update both `attemptLoad` call sites. Do not pass `i`. Leave `preloadNeighbors`’s own comment as-is (it already points at `finishLoad`).

- [ ] **Step 4: Re-run covering tests plus a wider `internal/ui` slice**

```bash
go test -race -count=1 ./internal/ui/ -run 'TestViewerShow_AnimatesGIF|TestViewerShow_NavigatingAwayStopsAnimation|TestInvalidateLoad_WakesAnimateImmediately|TestFinishLoad_PreloadsBothNeighbors|TestAttemptLoad_CacheHitServesFileRemovedFromDisk|TestCanSaveRotation_FalseWhileLoading|TestBatchDelete_LeavesTheWindowMaximized|TestSVGReRenderNeverMutatesTheCachedEntry|TestShowImage_InPictureFrameModeEndsFullyOpaque|TestShow_TracksAnimatedGIFLoopDuration|TestViewerShow_RAWPreviewMarksTheTitle'
```

Expected: PASS.

- [ ] **Step 5: Stop**

Do not edit `todos.md` or `ARCHITECTURE.md` (Task 5). Do not commit.

---

### Task 5: Docs, locator, full verification

**Model:** `cursor-grok-4.5-high-fast` (implementer), `cursor-grok-4.5-high-fast` (task reviewer)

**Files:**
- Modify: `todos.md`
- Modify: `ARCHITECTURE.md` — `internal/ui` file table, `load.go` row only
- Do not modify Go sources unless `gofmt`/`goimports` requires it after Task 4

**Interfaces:**
- Consumes: Task 4’s finished orchestrator
- Produces: `todos.md` records the outcome; `ARCHITECTURE.md` still locates `finishLoad` in `load.go`

- [ ] **Step 1: Confirm Go sources match the orchestrator**

Read `finishLoad` in `internal/ui/load.go`. If it is not the eight-step sequence from Task 4, stop and report BLOCKED — do not “finish” docs on a partial extract.

- [ ] **Step 2: Update the `load.go` locator in `ARCHITECTURE.md`**

Current cell (package map, `internal/ui` file table):

`ShowImage` / `attemptLoad` / `finishLoad`, neighbor preload (`AddIfFits`), GIF `animate`, `resizeToImage` / `syncWindowToZoom`.

Replace with:

`ShowImage` / `attemptLoad` / `finishLoad` (named steps in this file), neighbor preload (`AddIfFits`), GIF `animate`, `resizeToImage` / `syncWindowToZoom`.

Do not list every helper in the table (the architecture trim forbids per-function commentary). Do not add a new file row.

- [ ] **Step 3: Move the todo to Done**

Remove this bullet from `todos.md` `## TODO`:

```
- `finishLoad` (`internal/ui/load.go:192-305`) is a 114-line do-everything pipeline (vector setup, fade, overlay, zoom,
  resize, title, animation, preload). It is linear and well-commented; decompose into named steps.
```

Add under `## Done` (keep the `exif.go` bullet in `## TODO`):

```
- `finishLoad` is an orchestrator of named steps in `internal/ui/load.go`
  (`installLoadedFrames` … `startLoadedAnimation`, then `preloadNeighbors` / `done()`);
  behavior unchanged (2026-08-25).
```

- [ ] **Step 4: Format check, vet, build, full race suite**

From the repository root:

```bash
make fmt-check
go vet ./...
go build ./...
go test -race ./...
```

Expected: all PASS. `fmt-check` uses `goimports -local github.com/frathe/picfetch`. If `fmt-check` fails on `load.go`, run `make fmt` and re-check — do not hand-format import groups.

- [ ] **Step 5: Stop**

Do not commit. Suggested commit message for the user (whole change, Tasks 1–5):

```
Refactor finishLoad into named steps in load.go.

The pipeline was already linear and correct; naming the steps keeps the
load-bearing order (vector copy-on-write, fade, ForceRepaint-before-animate,
preload-before-done) reviewable without a 100-line function.
```

---

## Controller checklist (parent session, not a subagent)

After **each** implementer returns:

1. Read the full `internal/ui/load.go` diff (not just the helper). Confirm order against the Global Constraints table.
2. Confirm why-comments moved onto the helper that now owns the constraint.
3. Fix any drift yourself before the task reviewer, then before Task N+1.
4. If a reviewer flags a plan-mandated choice (same-file helpers, unused `i` dropped, no `applyRotationLayout` merge), ask Florian which governs — do not silently undo the plan.

## Self-review (plan vs spec)

1. **Coverage:** Named-step decomposition → Tasks 1–4. Docs/todo → Task 5. `exif.go` explicitly out of scope.
2. **Placeholders:** None. Signatures, helper bodies, commands, and expected PASS are spelled out.
3. **Types:** `finishLoad(token requestToken, u fyne.URI, loaded *imaging.LoadedImage, done func())` is consistent from the target block through Task 4.
4. **Order risk:** Tasks that could silently reorder (`presentLoadedImage`, `startLoadedAnimation`, `done()`) keep the original comments and are assigned `cursor-grok-4.6-xhigh` plus parent review.
5. **Opus:** Not used for implementation; the work splits. Used only for final whole-branch review.
