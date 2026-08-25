# Restore Never-Started Canary Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Parent session protocol (overrides the skill's "don't pause" rule):** After each task, the parent reviews the diff, fixes anything wrong, then dispatches the next subagent. Do not start Task N+1 until the parent has signed off on Task N. Do not `git commit`.

**Goal:** Restore a loud `t.Fatal` when a named harness wait helper is asked to wait on a `completion.Signal` that has never begun, without changing `drain` or `waitFor`.

**Architecture:** `completion.Signal.Wait` returning immediately on a never-begun signal is load-bearing for `drain` (every test's `t.Cleanup`). The canary belongs on the *named* wait helpers that mean "this test just started this operation", matching `settleChooser` / `settleWallpaper` / `settleFavoritePreviews` / `exifwin.waitForWarm`. `Begun()` is monotonic ("ever started"), not "pending now".

**Tech Stack:** Go tests, `internal/completion.Signal`, `internal/ui` harness (`harness_test.go`).

## Status: not already done

Checked 2026-08-25 against `todos.md` and the current tree.

| Helper | File | Canary today? |
| --- | --- | --- |
| `settleChooser` | `internal/ui/harness_test.go` | yes (`!v.chooser.Begun()`) |
| `settleWallpaper` | `internal/ui/wallpaper_test.go` | yes |
| `settleFavoritePreviews` | `internal/ui/favthumbs_test.go` | yes |
| `settleToast` | `internal/ui/harness_test.go` | yes (`v.toast.stop == nil`, pending-*now*) |
| `waitForWarm` | `internal/ui/exifwin/exifwin_test.go` | yes (out of scope) |
| `waitUntilLoaded` | `internal/ui/harness_test.go` | **no** |
| `waitForScan` | `internal/ui/harness_test.go` | **no** |
| `waitForSort` | `internal/ui/harness_test.go` | **no** |
| `waitForAnimStopped` | `internal/ui/harness_test.go` | **no** |
| `waitForClipboard` | `internal/ui/batch_test.go` | **no** |

`waitFor` itself must stay silent on never-begun: `drain` loops it over clipboard, wallpaper, favThumb, chooser, scan, sort, load, and anim on every test, most of which never started those ops.

## Approaches considered

1. **Canary inside `waitFor`** — one line, covers every caller. **Reject.** Breaks `drain`; every test that never scanned/copied/animated would fatal at cleanup.
2. **Make `Signal.Wait` error on never-begun** — **Reject.** The type's contract (and `ARCHITECTURE.md`) is that a never-begun wait is the "nothing to drain" case.
3. **Per-helper `if !s.Begun() { t.Fatal(...) }`** — **Accept.** Matches the todo, matches the four helpers that already have it, leaves `drain`/`waitFor` alone.

Do not invent a `requireBegun` helper unless a later task is adding more than these five; the existing canaries are inline.

## Global Constraints

- Do not change `internal/completion` (`Wait` on never-begun stays immediate nil).
- Do not add a canary to `waitFor`, `waitHandle`, or `drain`.
- Do not change `settleToast` (its `stop == nil` check is the pending-now form; leave it).
- Do not touch `internal/ui/exifwin` (`waitForWarm` already has the canary).
- Do not edit the in-progress Actions-menu work (`actionmenu.go`, `menu.go`, translations, etc.). This task's files are listed per task.
- Every user-visible string is `lang.L(...)` — this work has none.
- Do not `git commit`. Do not push.
- Match existing fatal-message tone: `"the image load never started"`, not "pending to settle" (`Begun()` is not "in flight now").
- After each task, parent reviews before the next subagent starts.
- `Begun()` is monotonic: a helper called after an earlier completed generation of the *same* signal still passes the canary and `Wait` returns immediately. That matches the old closed-channel behavior. The canary only catches "this viewer never started this op at all".

### Subagent roster

| Role | When | `subagent_type` | Model |
| --- | --- | --- | --- |
| Implementer (Go) | Tasks 1–4, Task 6 | `go-expert` | `composer-2.5-fast` (mechanical, tiny diffs; plan text has the complete code) |
| Implementer (race suite) | Task 5 Step 1–2 if the suite is expected to pass | `shell` | `composer-2.5-fast` |
| Implementer (diagnosis) | Task 5 if `-race` fails, or any task that cannot be split | `go-expert` | `claude-opus-5-thinking-high` |
| Final whole-branch review | After Task 6 | `go-expert` | `claude-opus-5-thinking-high` |
| Parent review | After every task | this session, not a subagent | — |

Each implementer prompt must include: the task text below, the Global Constraints, "read `ARCHITECTURE.md` and `AGENTS.md` first", "do not commit", and the exact files it may touch.

## File map

- Modify: `internal/ui/harness_test.go` — canaries on `waitUntilLoaded`, `waitForScan`, `waitForSort`, `waitForAnimStopped`; move `waitForClipboard` here; comment on `waitFor`/`drain`.
- Modify: `internal/ui/batch_test.go` — delete the local `waitForClipboard` (call sites stay).
- Modify: `internal/ui/clipboard_test.go` — `waitFor(t, "the clipboard copy", &v.clipboard)` becomes `waitForClipboard(t, v)`.
- Modify: `todos.md` — move the item to Done.
- Modify: `ARCHITECTURE.md` — one sentence under `internal/completion` / wait-helper notes: named waiters require `Begun()`, `drain` does not.

No production code changes.

---

### Task 1: `waitUntilLoaded` canary

**Files:**
- Modify: `internal/ui/harness_test.go` (`waitUntilLoaded`, ~278–299)
- Test: `go test -count=1 ./internal/ui/` (see Step 4)

**Interfaces:**
- Consumes: `v.load` (`completion.Signal`), `(*Signal).Begun() bool`, existing `waitFor`.
- Produces: `waitUntilLoaded` fatals with `"the image load never started"` when `!v.load.Begun()`.

`ShowImage` is the only production Begin of `v.load`. `handleDrop` with a non-empty image set eventually calls it via `startSort`'s completion. `dropAndWait` therefore always Begins load. `dropAndWaitScan` does not, and must not call `waitUntilLoaded`.

- [ ] **Step 1: Add the canary**

In `waitUntilLoaded`, immediately after `t.Helper()`, before `waitFor`:

```go
func waitUntilLoaded(t *testing.T, v *viewer) {
	t.Helper()

	if !v.load.Begun() {
		t.Fatal("the image load never started")
	}

	waitFor(t, "the image to finish loading", &v.load)

	// Also wait out the neighbor preloads finishLoad kicked off (they're
	// registered with preloads before the load signal finishes): a preload
	// goroutine that outlives its test keeps reading files - and shared
	// library state like the MIME map - under whatever test runs next,
	// which -race rightly reports. "Loaded" here deliberately means
	// "loaded, and everything that load spawned has settled".
	settled := make(chan struct{})
	go func() {
		v.preloads.Wait()
		close(settled)
	}()
	select {
	case <-settled:
	case <-time.After(testTimeout):
		t.Fatal("timed out waiting for neighbor preloads to settle")
	}
}
```

Do not add a dedicated "canary fires" unit test (nested `testing.T` / subprocess). The todo's proof is "the suite still passes" plus the `-race` run in Task 5.

- [ ] **Step 2: Run the package tests (no race yet)**

Run: `go test -count=1 ./internal/ui/`

Expected: PASS. If a test fatals `"the image load never started"`, stop and report the test name to the parent — that is a vacuous wait the canary is supposed to catch, and the parent decides whether to fix the call site (use `dropAndWait` / `ShowImage` first, or switch to `dropAndWaitScan`) rather than weaken the helper.

Likely-safe: every current `waitUntilLoaded` call follows `dropAndWait`, `ShowImage`, or a delete that re-shows. `handleDrop([]fyne.URI{})` returns without Begin — no current test waits a load after that.

- [ ] **Step 3: Parent review**

Parent checks: only `waitUntilLoaded` changed; `drain` / `waitFor` untouched; message is `"the image load never started"`.

---

### Task 2: `waitForScan` and `waitForSort` canaries

**Files:**
- Modify: `internal/ui/harness_test.go` (`waitForScan`, `waitForSort`, ~301–311)

**Interfaces:**
- Consumes: `v.scanOp.done`, `v.sortOp.done`.
- Produces: scan helper fatals `"the scan never started"`; sort helper fatals `"the sort never started"`.

`handleDrop` with `len(uris) > 0` always `scanOp.begin()`s. Empty URI list returns without Begin.

`startSort` always `sortOp.begin()`s. `SetSortMode` with `len(v.state.files) == 0` returns without `startSort` — `TestSetSortMode_SafeWithNoFilesLoaded` and `TestActionsMenu_SetSortModeWithNoFiles` correctly do **not** call `waitForSort`.

`dropAndWait` calls both helpers. `dropAndWaitScan` calls only scan, because `applyScanResult` never reaches `startSort` when nothing is displayable.

- [ ] **Step 1: Add both canaries**

```go
func waitForScan(t *testing.T, v *viewer) {
	t.Helper()

	if !v.scanOp.done.Begun() {
		t.Fatal("the scan never started")
	}

	waitFor(t, "the scan", &v.scanOp.done)
}

func waitForSort(t *testing.T, v *viewer) {
	t.Helper()

	if !v.sortOp.done.Begun() {
		t.Fatal("the sort never started")
	}

	waitFor(t, "the sort", &v.sortOp.done)
}
```

- [ ] **Step 2: Run the package tests**

Run: `go test -count=1 ./internal/ui/`

Expected: PASS.

If `"the sort never started"` fires from `dropAndWait`, the call site dropped something that scanned to zero images — it should have used `dropAndWaitScan`. Do not remove the canary from `dropAndWait`; fix the call site.

- [ ] **Step 3: Parent review**

---

### Task 3: `waitForAnimStopped` canary

**Files:**
- Modify: `internal/ui/harness_test.go` (`waitForAnimStopped`, ~338–345)
- Call sites today: `internal/ui/animate_test.go` only (`TestViewerAdvancesFramesUntilInvalidated`, `TestInvalidateLoad_WakesAnimateImmediately`). Both load an animated GIF first. The second already asserts `v.anim.Begun()` at the call site; keep that assertion — it documents the test's own setup, the helper canary is the backstop.

**Interfaces:**
- Consumes: `v.anim`.
- Produces: fatals `"the animation never started"`.

`v.anim.Begin()` lives in `load.go` when playback actually starts. An over-budget GIF (`TestImgCache_OverBudgetAnimationStillDisplaysFirstFrame`) never Begins `v.anim` and must not call this helper (it currently does not).

- [ ] **Step 1: Add the canary**

```go
func waitForAnimStopped(t *testing.T, v *viewer) {
	t.Helper()

	if !v.anim.Begun() {
		t.Fatal("the animation never started")
	}

	waitFor(t, "the animation to stop", &v.anim)
}
```

- [ ] **Step 2: Run focused tests**

Run: `go test -count=1 -run 'TestViewerAdvancesFramesUntilInvalidated|TestInvalidateLoad_WakesAnimateImmediately|TestImgCache_OverBudget' ./internal/ui/`

Expected: PASS.

Then: `go test -count=1 ./internal/ui/`

Expected: PASS.

- [ ] **Step 3: Parent review**

---

### Task 4: Move `waitForClipboard` into the harness and add its canary

**Files:**
- Create (by moving): `waitForClipboard` in `internal/ui/harness_test.go` (place it with the other named waiters, after `waitForAnimStopped`)
- Modify: `internal/ui/batch_test.go` — delete the local function (lines 40–47); leave the four call sites
- Modify: `internal/ui/clipboard_test.go` — `TestCopyImageToClipboard_DispatchFailureShowsToast` currently calls `waitFor` directly (~105). Switch it to `waitForClipboard`.

**Interfaces:**
- Consumes: `v.clipboard`.
- Produces: package-level `waitForClipboard(t *testing.T, v *viewer)` that fatals `"the clipboard copy never started"` when `!v.clipboard.Begun()`.

`copyImageToClipboard` returns without Begin when `v.img.Image == nil`. `copyGridSelection` returns without Begin when `len(paths) == 0`. Current call sites all copy after a successful load / grid selection.

`clipboard_test.go`'s `TestReportClipboardError_ShowsToast` must keep **not** waiting on `v.clipboard` — that path never Begins the signal.

Do not add a canary to the raw `waitFor` call in `filestate_test.go` (`waitFor` on local Signals the test itself just `Begin()`s).

- [ ] **Step 1: Add the helper to the harness**

```go
// waitForClipboard waits out the goroutine a clipboard copy runs on -
// v.clipboard is finished once that goroutine has fully run, error toast
// included, so reading widget state afterwards is race-free.
func waitForClipboard(t *testing.T, v *viewer) {
	t.Helper()

	if !v.clipboard.Begun() {
		t.Fatal("the clipboard copy never started")
	}

	waitFor(t, "the clipboard copy", &v.clipboard)
}
```

Keep the comment; it is the same contract as today's `batch_test.go` helper.

- [ ] **Step 2: Delete the duplicate from `batch_test.go`**

Remove:

```go
// waitForClipboard waits out the goroutine a clipboard copy runs on -
// v.clipboard is finished once that goroutine has fully run, error toast
// included, so reading widget state afterwards is race-free.
func waitForClipboard(t *testing.T, v *viewer) {
	t.Helper()

	waitFor(t, "the clipboard copy", &v.clipboard)
}
```

Leave `waitForClipboard(t, v)` call sites in that file unchanged.

- [ ] **Step 3: Switch `clipboard_test.go`**

In `TestCopyImageToClipboard_DispatchFailureShowsToast`, replace:

```go
	waitFor(t, "the clipboard copy", &v.clipboard)
```

with:

```go
	waitForClipboard(t, v)
```

Keep the comment above it (why waiting beats polling the toast).

- [ ] **Step 4: Run focused tests**

Run: `go test -count=1 -run 'TestCopy|TestBatchCopy|TestShiftDelete' ./internal/ui/`

Expected: PASS.

Then: `go test -count=1 ./internal/ui/`

Expected: PASS.

- [ ] **Step 5: Parent review**

Confirm `waitFor` in `drain` still has no Begun check. Confirm `TestReportClipboardError_ShowsToast` still does not call `waitForClipboard`.

---

### Task 5: Full `-race` proof

**Files:** none, unless a test fatals a canary — then only that test's call site (parent decides).

This is the todo's actual acceptance criterion: "it needs a full `-race` run to prove no test legitimately waits on a signal that never began."

- [ ] **Step 1: Format / vet / build (CI prelude)**

Run from repo root:

```
make fmt
go vet ./...
go build ./...
```

Expected: clean.

- [ ] **Step 2: Full race suite**

Run: `go test -race ./...`

Expected: PASS. This is slow (minutes). Do not cancel early. Do not skip packages.

If a canary fires: copy the test name and the fatal line. Do **not** remove the canary. Either:

- the test was waiting vacuously → start the operation first, or drop the wait and keep a positive assertion, or
- `dropAndWait` was used for a no-image drop → switch to `dropAndWaitScan`.

If a *race* fires that is unrelated to this change: stop and report to the parent; do not "fix" unrelated races in this task.

- [ ] **Step 3: Parent review of the test log**

---

### Task 6: Docs and todo list

**Files:**
- Modify: `todos.md` — move "Restore the never started canary…" from `## TODO` to `## Done` (keep the explanation; add a one-liner that the five helpers now fatal via `Begun()`, `drain`/`waitFor` still do not).
- Modify: `ARCHITECTURE.md` — in the `internal/completion` section (the paragraph that says `Wait` returns immediately for a Signal that has never begun), add that **named** `internal/ui` wait helpers (`waitUntilLoaded`, `waitForScan`, `waitForSort`, `waitForAnimStopped`, `waitForClipboard`, plus the existing settle* helpers) require `Begun()` first, while `drain` and `waitFor` must not.

Do not add a new package-map row for `harness_test.go` unless you are already editing that table for another reason.

- [ ] **Step 1: `todos.md`**

Under `## Done`, after the Menu Actions block, add the completed item (preserve the original explanation, then the outcome). Clear it from `## TODO` so that section is empty again (or only leftover later items).

- [ ] **Step 2: `ARCHITECTURE.md`**

Append to the `Wait` / never-begun sentence around the `internal/completion` package description, without contradicting "callers don't each need their own nil check" — that sentence is still true for `drain`. Named test waiters are the exception, on purpose.

Suggested addition (edit to fit the surrounding prose, do not leave it as a bullet dump):

Named wait helpers in `internal/ui`'s harness (`waitUntilLoaded`, `waitForScan`, `waitForSort`, `waitForAnimStopped`, `waitForClipboard`) fatal when `!Begun()` so a test that forgot to start the operation fails with a named message instead of returning immediately. `drain` and `waitFor` keep the never-begun-is-nothing behavior, because cleanup must wait out whichever subset of operations that test happened to start.

- [ ] **Step 3: `waitFor` comment in `harness_test.go`**

On `waitFor`, add one sentence that it is deliberately *not* the canary: `drain` uses it, and a never-begun Signal must still return immediately.

- [ ] **Step 4: Parent review**

---

## Out of scope

- `waitHandle` / zero `Handle` (still waits for nothing; current call sites capture `Current()` after Begin).
- `settleSlideshow`, `waitForAnimFrame`, `waitForCached`, `waitForPending` (not `completion.Signal` waiters, or not the five named in the todo).
- Changing `settleToast` to `toast.hidden.Begun()` (worse: monotonic; `stop == nil` is the right check).
- Production code, translations, golden screenshots.

## Self-review

1. **Spec coverage:** five missing canaries, drain/`waitFor` unchanged, clipboard helper shared, full `-race`, todos + ARCHITECTURE. All have tasks.
2. **Placeholders:** none.
3. **Type consistency:** `Begun() bool` on `*completion.Signal`; field names `v.load`, `v.scanOp.done`, `v.sortOp.done`, `v.anim`, `v.clipboard`.

## Suggested commit message (for the user, after all tasks)

```
test: restore never-started canaries on harness wait helpers

Signal.Wait returns immediately when work never began, which is what
drain needs and what made five wait helpers fail silent. Named waiters
now Fatal via Begun(); drain and waitFor stay unchanged.
```
