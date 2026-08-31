# Copy Selection / Settings Architectural Cleanup Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the ten verified findings from the architectural review of `feature/copy-selection`: state duplicated across seams (settings snapshot, captured image, key keep-list), the hand-enumerated "yield Copy Selection" invariant, and four smaller memory/pipeline/duplication cleanups.

**Implementation status:** All implementation and verification steps are complete. Per-task commit steps remain intentionally unchecked: `AGENTS.md` forbids agents from running `git commit`, and the completed work is already present in the branch's existing commits.

**Architecture:** Two mechanism-level moves carry most of the weight: (1) the Settings window pushes *patch* diffs (prev vs next snapshot) instead of being diffed against live viewer state, and (2) the Copy Selection yield moves into the one chokepoint every displayed-image change funnels through (`ShowImage`), with a completeness test guarding the remaining per-callback menu wrapping. The rest are local, verified-equivalent replacements.

**Tech Stack:** Go 1.26, Fyne v2. All tests run headless under Fyne's test driver: `go test ./internal/...`.

**Spec:** The "Review findings" section below — this plan implements exactly those findings, no more.

## Review findings (the spec)

Verified findings from the architectural review of this branch (all CONFIRMED against the code):

1. **Settings snapshot revert** — `internal/ui/settingswin/settingswin.go:188` + `internal/ui/memlimits.go:234`: `apply()` pushes the whole Show-time snapshot; `ApplySettings` diffs it against *live* state, so a preference changed via main-window shortcut (M/S/Shift+P/Up/Down) while Settings is open gets silently reverted by the next unrelated control edit.
2. **Navigation bypasses the yield** — `internal/ui/viewer.go:922`: `StepImage` (EXIF-window arrows) and background `ShowImage` callers (`jumpIfHiddenExtra`, sort completion) never yield Copy Selection; the image swaps under an active selection, and a paused animation stays frozen.
3. **Key0 desync** — `internal/ui/copyselection.go:118`: `Key0` is on the keep list as a "zoom key", but its handler also calls `resetRotation()`, changing orientation under the pinned capture.
4. **Menu wrapping is enumeration, and Favorites fell through** — `internal/ui/menu.go:62`: 20 hand-wrapped fields, no completeness check; `internal/ui/favorites/favorites.go:83/93/156` menu items never yield.
5. **Silent discard** — `internal/ui/drop.go:84`: a drop or favorite open during a pending copy is dropped with no feedback; `openFavorite` even runs `SyncFavoritePreviews` before the open no-ops.
6. **Duplicate RGBA pinned** — `internal/ui/copyselection.go:66`: capture re-rotates via `display.Rotated()` (fresh full-size allocation) though `v.img.Image` already holds the identical oriented frame.
7. **Geometry callback overhead** — `internal/ui/features.go:54`: dispatches a queued closure per geometry tick even while inactive; while active runs `ForceRepaint` (full window-tree refresh) per tick.
8. **Pause leak (latent)** — `internal/ui/copyselection.go:74`: nil-capture failure returns without releasing the acquired animation pause.
9. **Forked encode pipeline** — `internal/ui/copyselection/copyselection.go:149`: production runs `Source.Encode`; the test-only `SetEncode` seam runs a hand-copied pixels→cropBounds→encode sequence.
10. **Oriented-size duplication** — `internal/ui/copyselection/source.go:50`: the round-then-swap logical-size computation exists in four places (`displayedDimensions`, capture, `Bounds`, `cropBounds`).

**Explicitly NOT bugs — do not "fix" these:** the zoom-transform re-derivation in `regionCopyView` (legitimate use of the published `zoom.Geometry` contract) and the `yieldCopySelection` nil-`regionCopy` path (provably unreachable). Seven further minor cleanups were reviewed and deliberately left out of scope: do not expand into them.

## Global Constraints

- Go 1.26.7, module `github.com/frathe/picfetch`. Build with `go build ./...`, vet with `go vet ./...`.
- Every new user-visible string goes through `lang.L("...")` AND gets a key added to **both** `translations/en.json` and `translations/de.json`.
- **No Unicode arrows in UI strings** (the bundled NotoSans has no arrow glyphs). Use plain hyphens or words.
- Match the repo's comment style: comments state constraints and reasons, never narrate the edit ("why", not "what changed").
- Do not reorder tasks: Tasks 2–5 build on each other's yield semantics, and Tasks 6 and 9 edit the same function.
- Run the named tests after every step that says so; paste failures into your worklog rather than proceeding.
- Commit after each task with the given message. Repo commit style is a capitalized sentence with a trailing period (see `git log --oneline -5`).

---

### Task 1: Settings apply becomes patch semantics (finding 1)

The Settings window today holds a snapshot seeded once at Show and pushes the *whole* snapshot on every control edit; the viewer diffs it against live state, so live shortcut changes look like "edits" and get reverted. Fix: pass the snapshot **before and after** the one control edit, and apply only fields that differ between those two.

**Files:**
- Modify: `internal/ui/settingswin/settingswin.go` (Host interface ~line 55, `apply` ~line 188)
- Modify: `internal/ui/memlimits.go` (`ApplySettings` ~line 229)
- Modify: `internal/ui/settingswin/settingswin_test.go` (fakeHost ~line 42)
- Test: `internal/ui/settings_apply_test.go`

**Interfaces:**
- Consumes: nothing from other tasks.
- Produces: `ApplySettings(prev, next preferences.State)` on the viewer and on `settingswin.Host`. Later tasks do not depend on it.

- [x] **Step 1: Write the failing test**

Append to `internal/ui/settings_apply_test.go`:

```go
// The main window stays live while Settings is open: M, S, Shift+P and
// Up/Down all change standing preferences the Settings form snapshotted at
// Show. Applying an unrelated Settings edit must not push those stale
// snapshot values back over the live ones.
func TestApplySettings_DoesNotRevertLiveShortcutChanges(t *testing.T) {
	v := newTestViewer(t)

	prev := v.settingsState() // the snapshot Show would seed the form with

	v.toggleMergeMode() // the user presses M in the main window
	if !v.MergeMode() {
		t.Fatal("precondition: merge mode is off after toggling")
	}

	next := prev
	next.StaticWindowSize = !next.StaticWindowSize // an unrelated control edit
	v.ApplySettings(prev, next)

	if !v.MergeMode() {
		t.Error("applying an unrelated setting reverted the live merge-mode change")
	}
	if v.StaticWindowSize() != next.StaticWindowSize {
		t.Errorf("StaticWindowSize = %v, want %v", v.StaticWindowSize(), next.StaticWindowSize)
	}
}
```

- [x] **Step 2: Run it — expect a COMPILE failure**

Run: `go test ./internal/ui -run TestApplySettings_DoesNotRevertLiveShortcutChanges`
Expected: compile error — `too many arguments in call to v.ApplySettings` (the signature is still one-argument).

- [x] **Step 3: Change the viewer's ApplySettings to patch semantics**

In `internal/ui/memlimits.go`, replace the whole `ApplySettings` function (currently lines 229–280, from the doc comment through the closing brace) with:

```go
// ApplySettings is the Settings window's write path. prev and next are the
// form snapshot before and after the one control edit that triggered this
// push, so only the edited field runs its setter — never a field a
// main-window shortcut changed while the Settings window was open (the old
// live-state diff mistook those for edits and silently reverted them).
// Does not persist; shutdown Save still goes through currentPreferences.
func (v *viewer) ApplySettings(prev, next preferences.State) {
	if next.ThemeMode != prev.ThemeMode {
		v.SetThemeMode(next.ThemeMode)
	}
	// Also guarded against the live mode: SetSortMode restarts a background
	// sort, which re-selecting the already-active order must not do.
	if mode := filesort.FromPref(next.SortMode); next.SortMode != prev.SortMode && mode != v.SortMode() {
		v.SetSortMode(mode)
	}
	if next.MergeMode != prev.MergeMode {
		v.SetMergeMode(next.MergeMode)
	}
	if next.SlideShuffle != prev.SlideShuffle {
		v.SetSlideShuffle(next.SlideShuffle)
	}
	if next.SlideInterval != prev.SlideInterval {
		v.SetSlideInterval(next.SlideInterval)
	}
	if next.MaxScanFiles != prev.MaxScanFiles {
		v.SetMaxScan(next.MaxScanFiles)
	}
	if next.MaxWindowWidth != prev.MaxWindowWidth {
		v.SetMaxWindowWidth(next.MaxWindowWidth)
	}
	if next.MaxWindowHeight != prev.MaxWindowHeight {
		v.SetMaxWindowHeight(next.MaxWindowHeight)
	}
	if next.StaticWindowSize != prev.StaticWindowSize {
		v.SetStaticWindowSize(next.StaticWindowSize)
	}
	if next.MaxImageCacheMB != prev.MaxImageCacheMB {
		v.SetMaxImageCacheMB(next.MaxImageCacheMB)
	}
	if next.MaxThumbCacheMB != prev.MaxThumbCacheMB {
		v.SetMaxThumbCacheMB(next.MaxThumbCacheMB)
	}
	if next.MaxFileSizeMB != prev.MaxFileSizeMB {
		v.SetMaxFileSizeMB(next.MaxFileSizeMB)
	}
	if next.FavoritePreviewCache != prev.FavoritePreviewCache {
		v.SetFavoritePreviewCache(next.FavoritePreviewCache)
	}
	if next.CheckForUpdates != prev.CheckForUpdates {
		v.SetCheckForUpdates(next.CheckForUpdates)
	}
	if next.DuplicateDistance != prev.DuplicateDistance || next.DuplicateDistanceSet != prev.DuplicateDistanceSet {
		v.SetDuplicateDistance(next.DuplicateDistance)
	}
}
```

- [x] **Step 4: Change the settingswin side**

In `internal/ui/settingswin/settingswin.go`:

Replace the Host interface's `ApplySettings(preferences.State)` line (inside the interface at ~line 56) with:

```go
	ApplySettings(prev, next preferences.State)
```

Also update the Host doc comment's sentence "ApplySettings pushes the whole form back so live side effects (cache retune, appearance, sort) can run" to:

```go
// Host is what the settings window needs from the app after it has been
// seeded with a preferences.State snapshot: ApplySettings receives the form
// snapshot before and after the one control edit, so the app applies only
// the edited fields (patch semantics — a stale snapshot must never revert a
// preference a main-window shortcut changed while this window was open),
// and the two update verbs stay out of that snapshot because they are
// requests, not standing values.
```

Replace `apply` (lines 186–191) with:

```go
// apply mutates the seeded snapshot and pushes the before/after pair
// through the host, which applies only what actually changed between the
// two. Invalid mid-edit input never reaches here.
func (w *Window) apply(mutate func(*preferences.State)) {
	prev := w.prefs
	mutate(&w.prefs)
	w.host.ApplySettings(prev, w.prefs)
}
```

- [x] **Step 5: Update the settingswin test fake**

In `internal/ui/settingswin/settingswin_test.go`, add a `prevCalls` field to the fakeHost struct (after `applyCalls []preferences.State` at line 45):

```go
	prevCalls []preferences.State
```

and replace the `ApplySettings` method (lines 52–55) with:

```go
func (f *fakeHost) ApplySettings(prev, next preferences.State) {
	f.prefs = next
	f.prevCalls = append(f.prevCalls, prev)
	f.applyCalls = append(f.applyCalls, next)
}
```

Then append a window-level test at the end of the file:

```go
// Each push must carry the snapshot as it was before that one edit — the
// second call's prev is the first call's next — which is what lets the app
// apply only the edited field.
func TestApply_PassesThePreviousSnapshot(t *testing.T) {
	host := &fakeHost{prefs: preferences.State{}}
	w := showSettings(t, host)

	w.mergeCheck.SetChecked(true)
	w.shuffleCheck.SetChecked(true)

	if len(host.applyCalls) != 2 || len(host.prevCalls) != 2 {
		t.Fatalf("apply calls = %d/%d prev, want 2/2", len(host.applyCalls), len(host.prevCalls))
	}
	if host.prevCalls[0].MergeMode {
		t.Error("first prev already has the edit applied")
	}
	if host.prevCalls[1] != host.applyCalls[0] {
		t.Error("second prev is not the first push's next — the chain is broken")
	}
}
```

- [x] **Step 6: Fix the remaining callers**

Run `go build ./...`. The remaining compile errors are the existing calls in `internal/ui/settings_apply_test.go` (four tests call `v.ApplySettings(next)` after `next := v.settingsState()`). For each, rename the first snapshot to `prev`, copy it, and pass both. The pattern, using the first test as the example (lines 40–44):

```go
	prev := v.settingsState()
	next := prev
	next.ThemeMode = appearance.Dark
	v.ApplySettings(prev, next)
```

Apply the same mechanical change to every `v.ApplySettings(` call in that file. Do not change any assertion.

- [x] **Step 7: Run the affected tests**

Run: `go build ./... && go test ./internal/ui/settingswin ./internal/ui -run 'TestApply|TestApplySettings|TestSettings' -count=1`
Expected: PASS, including the new tests from Steps 1 and 5.

- [ ] **Step 8: Commit**

```bash
git add internal/ui/memlimits.go internal/ui/settingswin/settingswin.go internal/ui/settingswin/settingswin_test.go internal/ui/settings_apply_test.go
git commit -m "Apply Settings edits as patches so live shortcut changes survive."
```

---

### Task 2: Yield feedback + the ShowImage chokepoint (findings 2 and 5)

Two changes that belong together: `yieldCopySelection` gains a toast when it refuses (so a blocked drop/favorite/menu command is no longer silent), and `ShowImage` — the one function every displayed-image change funnels through (`StepImage`, `Advance`, `jumpIfHiddenExtra`, sort completion, drop completion) — yields the mode itself, closing the EXIF-window/background bypass. The five existing entry-point yields stay: they refuse *earlier* (before scans, dialogs, or state changes start), while `ShowImage` is the backstop that makes the invariant unconditional.

Main-window arrow behavior is unchanged: while the mode is active, `Feature.HandleKey` consumes navigation keys before the dispatcher ever reaches `StepImage`.

**Files:**
- Modify: `internal/ui/copyselection.go` (`yieldCopySelection`, lines 105–114)
- Modify: `internal/ui/load.go` (`ShowImage`, line 22)
- Modify: `translations/en.json`, `translations/de.json`
- Test: create `internal/ui/copyselection_yield_test.go`

**Interfaces:**
- Consumes: nothing from other tasks.
- Produces: `yieldCopySelection() bool` now shows a toast on refusal; `ShowImage` is guaranteed to yield. Tasks 3–5 rely on both.

- [x] **Step 1: Write the failing tests**

Create `internal/ui/copyselection_yield_test.go`:

```go
package ui

import (
	"image"
	"image/color"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/uitest"
)

// The EXIF window's arrow keys call StepImage directly, bypassing the key
// dispatcher's yield. The chokepoint yield in ShowImage must cancel an idle
// selection there too, instead of swapping the image under it.
func TestStepImageYieldsIdleCopySelection(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 40, 20, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 40, 20, color.White)
	dropAndWait(t, v, a, b)
	start := v.CurrentIndex()

	v.startRegionCopy()
	if !v.regionCopy.State().Active {
		t.Fatal("precondition: Copy Selection did not start")
	}

	v.StepImage(1) // what exifwin's Left/Right handler calls
	waitUntilLoaded(t, v)

	if v.regionCopy.State().Active {
		t.Error("navigation left Copy Selection active over a different image")
	}
	if v.CurrentIndex() == start {
		t.Error("yielded navigation did not advance the image")
	}
}

// A pending copy blocks navigation instead of being cancelled, and the
// refusal is visible: the shared yield shows a toast rather than silently
// dropping the command.
func TestStepImageBlockedWhileCopyPending(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v,
		regionCopyPNGURI(t, "a.png", markedRegionCopyImage(10, 8)),
		regionCopyPNGURI(t, "b.png", markedRegionCopyImage(10, 8)))
	start := v.CurrentIndex()

	release := make(chan struct{})
	uitest.StubClipboardCopy(t, func([]byte) error { <-release; return nil })
	selectRegion(t, v, image.Rect(2, 2, 8, 6))
	v.regionCopy.HandleKey(fyne.KeyReturn)
	if !v.regionCopy.State().Busy {
		t.Fatal("precondition: no copy is pending")
	}

	v.StepImage(1)

	if v.CurrentIndex() != start {
		t.Errorf("navigation during a pending copy moved the image: index %d, want %d", v.CurrentIndex(), start)
	}
	if !v.toast.card.Visible() {
		t.Error("blocked navigation gave no feedback toast")
	}

	close(release)
	waitForClipboard(t, v)
	settleToast(t, v)
}

// An OS drop during a pending copy is refused early (before any state is
// touched) — but no longer silently.
func TestHandleDropBlockedWhileCopyPendingShowsToast(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, regionCopyPNGURI(t, "photo.png", markedRegionCopyImage(10, 8)))

	release := make(chan struct{})
	uitest.StubClipboardCopy(t, func([]byte) error { <-release; return nil })
	selectRegion(t, v, image.Rect(2, 2, 8, 6))
	v.regionCopy.HandleKey(fyne.KeyReturn)
	if !v.regionCopy.State().Busy {
		t.Fatal("precondition: no copy is pending")
	}

	v.handleDrop([]fyne.URI{uitest.TempJPEGURI(t, "late.jpg", 8, 8, color.White)})

	if v.FileCount() != 1 {
		t.Errorf("blocked drop changed the file set: count = %d, want 1", v.FileCount())
	}
	if !v.toast.card.Visible() {
		t.Error("blocked drop gave no feedback toast")
	}

	close(release)
	waitForClipboard(t, v)
	settleToast(t, v)
}
```

- [x] **Step 2: Run them — expect failures**

Run: `go test ./internal/ui -run 'TestStepImageYieldsIdleCopySelection|TestStepImageBlockedWhileCopyPending|TestHandleDropBlockedWhileCopyPendingShowsToast' -count=1`
Expected: FAIL — the first test fails with "navigation left Copy Selection active", the other two fail on the missing toast.

- [x] **Step 3: Add the toast to yieldCopySelection**

In `internal/ui/copyselection.go`, replace `yieldCopySelection` (lines 105–114) with:

```go
// yieldCopySelection lets another PicFetch command run. A pending copy
// blocks that command — with a toast, so the refusal is visible wherever
// the command came from (drop, menu, favorite, EXIF window); idle mode
// cancels first. Zoom and pan must not call this. Window close and
// shutdown remain available while busy.
func (v *viewer) yieldCopySelection() bool {
	if v.regionCopy != nil && v.regionCopy.State().Busy {
		v.ShowToast(lang.L("finishing the copy - try again in a moment"))
		return false
	}
	v.cancelRegionCopy()
	return true
}
```

(`lang` is already imported in this file.)

- [x] **Step 4: Add the translation keys**

In `translations/en.json`, add (anywhere alongside the other toast strings, e.g. after `"could not copy the image: %v"`):

```json
   "finishing the copy - try again in a moment": "finishing the copy - try again in a moment",
```

In `translations/de.json`, add the same key in the matching spot:

```json
   "finishing the copy - try again in a moment": "Kopieren wird noch abgeschlossen - bitte gleich noch einmal versuchen",
```

- [x] **Step 5: Add the chokepoint yield to ShowImage**

In `internal/ui/load.go`, inside `ShowImage` directly after the `len(v.state.files) == 0` guard (lines 23–25), insert:

```go
	// Copy Selection pins a source captured from the displayed image and,
	// for animations, holds the frame-advance pause. Yield here — the one
	// chokepoint every displayed-image change funnels through — so no
	// caller (EXIF-window arrows, sort completion, jumpIfHiddenExtra) can
	// swap the image under an active selection. The per-entry-point yields
	// stay for their earlier, cheaper refusal; this is the backstop.
	if !v.yieldCopySelection() {
		return
	}
```

- [x] **Step 6: Run the new tests, then the package**

Run: `go test ./internal/ui -run 'TestStepImageYieldsIdleCopySelection|TestStepImageBlockedWhileCopyPending|TestHandleDropBlockedWhileCopyPendingShowsToast' -count=1`
Expected: PASS.

Run: `go test ./internal/ui -count=1`
Expected: PASS. If an existing test fails, read it before touching anything: the likely cause is a test that navigates while it deliberately holds the mode open — decide whether the test's expectation or your guard placement is wrong (the guard must sit before any state or lifecycle mutation in `ShowImage`, and must not fire when the mode is inactive). Do not weaken the new tests to get green.

- [ ] **Step 7: Commit**

```bash
git add internal/ui/copyselection.go internal/ui/load.go internal/ui/copyselection_yield_test.go translations/en.json translations/de.json
git commit -m "Yield Copy Selection at the ShowImage chokepoint and toast refusals."
```

---

### Task 3: Key0 leaves the Copy Selection keep list (finding 3)

`0` is kept as a "zoom key", but its dispatcher case also calls `resetRotation()` — changing orientation while the mode pins bounds captured under the old orientation, so Enter copies the wrong pixels. `0` must yield exactly like `R` does. The keep list's classification rule gets written down so the next key added to it is checked against it.

**Files:**
- Modify: `internal/ui/copyselection.go` (`copySelectionKeepsKey`, lines 116–124)
- Test: `internal/ui/copyselection_yield_test.go` (from Task 2)

**Interfaces:**
- Consumes: Task 2's yield semantics (0 now cancels the mode via the dispatcher's yield).
- Produces: nothing later tasks use.

- [x] **Step 1: Write the failing test**

Append to `internal/ui/copyselection_yield_test.go`:

```go
// 0 is not a pure zoom key: its handler also clears view rotation. Under
// an active selection that would change orientation while the captured
// bounds stay axis-swapped, so 0 must yield the mode the same way R does.
func TestKey0YieldsCopySelectionAndResetsRotation(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "photo.jpg", 40, 20, color.White))

	v.rotateBy(1)
	if v.display.Rotation() == 0 {
		t.Fatal("precondition: rotation did not apply")
	}

	v.startRegionCopy()
	if !v.regionCopy.State().Active {
		t.Fatal("precondition: Copy Selection did not start")
	}

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.Key0})

	if v.regionCopy.State().Active {
		t.Error("0 changed orientation but left Copy Selection active")
	}
	if v.display.Rotation() != 0 {
		t.Errorf("rotation after 0 = %d, want 0", v.display.Rotation())
	}
}
```

- [x] **Step 2: Run it — expect failure**

Run: `go test ./internal/ui -run TestKey0YieldsCopySelectionAndResetsRotation -count=1`
Expected: FAIL with "0 changed orientation but left Copy Selection active".

- [x] **Step 3: Remove Key0 from the keep list**

In `internal/ui/copyselection.go`, replace `copySelectionKeepsKey` (lines 116–124) with:

```go
// copySelectionKeepsKey is the viewer-side keep list: keys whose dispatcher
// case touches nothing but v.zoom stay available without cancelling.
// Feature.HandleKey owns Escape, copy, and navigation. Key0 is deliberately
// absent: its case also calls resetRotation, and an orientation change must
// yield the mode exactly as R does — check any key added here against its
// full handleKeyEvent case, not its name.
func copySelectionKeepsKey(key fyne.KeyName) bool {
	switch key {
	case fyne.Key1, fyne.KeyPlus, fyne.KeyEqual, fyne.KeyMinus:
		return true
	}
	return false
}
```

- [x] **Step 4: Run the test and the package**

Run: `go test ./internal/ui -run TestKey0 -count=1 && go test ./internal/ui -count=1`
Expected: PASS (no existing test asserts 0 keeps the mode — verified before this plan was written).

- [ ] **Step 5: Commit**

```bash
git add internal/ui/copyselection.go internal/ui/copyselection_yield_test.go
git commit -m "Drop Key0 from the Copy Selection keep list; it resets rotation."
```

---

### Task 4: Favorites menu actions run through the host's command entry (findings 4 and 5)

The Favorites menu items call `AddCurrentList` / `ShowManage` / `openFavorite` directly, bypassing the yield that every other menu item gets. Instead of wrapping items from the outside (the enumeration smell), the favorites package routes every menu action through one host-supplied runner. This also fixes the preview-sync-before-noop: a refused favorite open no longer runs `SyncFavoritePreviews` at all.

**Files:**
- Modify: `internal/ui/favorites/favorites.go` (Host interface line 35, `New` line 81, `refreshMenu` line 146)
- Modify: `internal/ui/menu.go` (add the viewer's `RunCommand`)
- Modify: `internal/ui/favorites/favorites_test.go` (fakeHost line 21)
- Test: `internal/ui/favorites/favorites_test.go` and `internal/ui/copyselection_yield_test.go`

**Interfaces:**
- Consumes: `yieldCopySelection` (Task 2's toast version).
- Produces: `favorites.Host` gains `RunCommand(fn func())`; the viewer implements it in `menu.go`. Task 5's exempt list does not include favorites (they are not `menus.Callbacks` fields).

- [x] **Step 1: Extend the favorites fake host and write the failing package test**

In `internal/ui/favorites/favorites_test.go`, add two fields to the `fakeHost` struct (after `refreshMenus int` at line 35):

```go
	// blockCommands makes RunCommand refuse, the way the real host does
	// while a copy is pending; runCommands counts arrivals either way.
	blockCommands bool
	runCommands   int
```

and add the method after `RefreshMenus` (line 50):

```go
func (h *fakeHost) RunCommand(fn func()) {
	h.runCommands++
	if h.blockCommands {
		return
	}
	fn()
}
```

Then append the test:

```go
// Every Favorites menu action goes through Host.RunCommand, so the host's
// command-entry rules (yielding Copy Selection) cover this menu without
// internal/ui wrapping its items from the outside. A refused command runs
// nothing — not even the preview sync openFavorite fires before opening.
func TestMenuActionsRunThroughHostRunCommand(t *testing.T) {
	host := &fakeHost{files: []fyne.URI{storage.NewFileURI("/tmp/a.jpg")}}
	f := newFeature(t, host)
	f.writeFavorite("trip")
	if len(f.names) != 1 {
		t.Fatalf("setup: favorites = %v, want [trip]", f.names)
	}
	favoriteItem := f.menu.Items[2] // addItem, separator, then the favorite

	host.blockCommands = true
	host.calls = nil
	before := host.runCommands

	f.addItem.Action()
	f.manageItem.Action()
	favoriteItem.Action()

	if host.runCommands != before+3 {
		t.Errorf("RunCommand arrivals = %d, want %d", host.runCommands, before+3)
	}
	if f.addDialog != nil || f.manageDialog != nil {
		t.Error("a refused command still opened its dialog")
	}
	if len(host.calls) != 0 {
		t.Errorf("a refused favorite open still did work: calls = %v", host.calls)
	}

	host.blockCommands = false
	favoriteItem.Action()
	if len(host.calls) != 2 || host.calls[0] != "sync" || host.calls[1] != "open" {
		t.Errorf("allowed favorite open calls = %v, want [sync open]", host.calls)
	}
}
```

- [x] **Step 2: Run it — expect failure**

Run: `go test ./internal/ui/favorites -run TestMenuActionsRunThroughHostRunCommand`
Expected: FAIL with `RunCommand arrivals = 0, want 3` — the menu items don't route through the host yet. (A compile error here means the fake from Step 1 was mistyped; fix that first.)

- [x] **Step 3: Add RunCommand to favorites.Host and route the menu items**

In `internal/ui/favorites/favorites.go`, add to the `Host` interface (after `RefreshMenus()`):

```go
	// RunCommand runs a menu-initiated PicFetch command under the host's
	// command-entry rules — today: yield Copy Selection, cancelling an idle
	// selection and refusing while a copy is pending. Every action this
	// menu can start goes through it; the keyboard shortcuts reaching the
	// same actions are wrapped on the host's side instead (internal/ui's
	// yielding shortcut adder).
	RunCommand(fn func())
```

In `New` (line 83), change the addItem construction to:

```go
	f.addItem = fyne.NewMenuItem(lang.L("Add Current List to Favorites…"), func() { f.host.RunCommand(f.AddCurrentList) })
```

and the manageItem construction (line 93) to:

```go
	f.manageItem = fyne.NewMenuItem(lang.L("Manage Favorites…"), func() { f.host.RunCommand(f.ShowManage) })
```

In `refreshMenu` (line 156), change the per-favorite item to:

```go
		item := fyne.NewMenuItem(f.menuLabel(favoriteName), func() {
			f.host.RunCommand(func() { f.openFavorite(favoriteName) })
		})
```

- [x] **Step 4: Implement RunCommand on the viewer**

In `internal/ui/menu.go`, after `yieldThenMode` (line 108), add:

```go
// RunCommand is favorites.Host's command entry: the same yield every
// menus.Callbacks field gets from yieldingMenuCallbacks, supplied to the
// favorites package as a runner so its menu items are covered from the
// inside instead of wrapped item-by-item out here.
func (v *viewer) RunCommand(fn func()) {
	if !v.yieldCopySelection() {
		return
	}
	fn()
}
```

- [x] **Step 5: Fix any other Host implementors**

Run: `go build ./... && go vet ./...`. If any other type implements `favorites.Host` (search with `grep -rn "favorites.Host\|SyncFavoritePreviews" --include="*.go" internal | grep -v internal/ui/favorites`), give it the same trivial `RunCommand(fn func()) { fn() }` unless it is the viewer. The known implementors are the viewer (real) and the favorites test fakes (`fakeHost`, which `addGuardDuringSave` embeds, so it inherits the method).

- [x] **Step 6: Write the viewer-level integration test**

Append to `internal/ui/copyselection_yield_test.go`:

```go
// The Favorites menu's own items must yield like every other menu command;
// they were the gap in the hand-wrapped menu callbacks.
func TestFavoritesMenuYieldsCopySelection(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "photo.jpg", 40, 20, color.White))

	v.startRegionCopy()
	if !v.regionCopy.State().Active {
		t.Fatal("precondition: Copy Selection did not start")
	}

	// Items[0] is "Add Current List to Favorites…" — see favorites.New.
	v.favorites.Menu().Items[0].Action()

	if v.regionCopy.State().Active {
		t.Error("the Favorites menu ran a command without yielding Copy Selection")
	}
}
```

- [x] **Step 7: Run the tests**

Run: `go test ./internal/ui/favorites -count=1 && go test ./internal/ui -run 'TestFavoritesMenuYieldsCopySelection' -count=1`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/ui/favorites/favorites.go internal/ui/favorites/favorites_test.go internal/ui/menu.go internal/ui/copyselection_yield_test.go
git commit -m "Route Favorites menu actions through a host RunCommand that yields."
```

---

### Task 5: Completeness check for yieldingMenuCallbacks (finding 4)

`yieldingMenuCallbacks` restates "every command yields" once per field; a field added to `menus.Callbacks` later compiles unwrapped with no test failing. This reflection test enumerates the struct's func fields, so a future unwrapped field fails CI with a message telling the author exactly what to do.

**Files:**
- Test: create `internal/ui/menu_yield_test.go`

**Interfaces:**
- Consumes: `v.yieldingMenuCallbacks` (menu.go:62), `menus.Callbacks` (internal/ui/menus/menus.go:32), Task 2's yield.
- Produces: nothing.

- [x] **Step 1: Write the test**

Create `internal/ui/menu_yield_test.go`:

```go
package ui

import (
	"image/color"
	"reflect"
	"testing"

	"github.com/frathe/picfetch/internal/ui/menus"
	"github.com/frathe/picfetch/internal/uitest"
)

// Every menus.Callbacks field must run through the viewer's yield unless it
// is one of the exempt keep-the-mode commands. Enumerated by reflection so
// a field added to menus.Callbacks later fails here until it is either
// wrapped in yieldingMenuCallbacks or added to this exempt list on purpose.
func TestYieldingMenuCallbacksWrapsEveryField(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "photo.jpg", 40, 20, color.White))

	exempt := map[string]bool{
		"ZoomIn":        true, // zoom keeps the mode — see yieldingMenuCallbacks
		"ZoomOut":       true,
		"CopySelection": true, // Copy Selection itself must not cancel the mode
	}

	var c menus.Callbacks
	cv := reflect.ValueOf(&c).Elem()
	ct := cv.Type()

	ran := make(map[string]int, ct.NumField())
	for i := 0; i < ct.NumField(); i++ {
		field := ct.Field(i)
		if field.Type.Kind() != reflect.Func {
			continue
		}
		name := field.Name
		cv.Field(i).Set(reflect.MakeFunc(field.Type, func([]reflect.Value) []reflect.Value {
			ran[name]++
			return nil
		}))
	}

	wrapped := reflect.ValueOf(v.yieldingMenuCallbacks(c))

	for i := 0; i < ct.NumField(); i++ {
		field := ct.Field(i)
		if field.Type.Kind() != reflect.Func {
			continue
		}

		v.startRegionCopy()
		if !v.regionCopy.State().Active {
			t.Fatalf("%s: could not start Copy Selection for the probe", field.Name)
		}

		args := make([]reflect.Value, field.Type.NumIn())
		for j := range args {
			args[j] = reflect.Zero(field.Type.In(j))
		}
		wrapped.Field(i).Call(args)

		active := v.regionCopy.State().Active
		switch {
		case exempt[field.Name] && !active:
			t.Errorf("%s is exempt from the yield but ended Copy Selection", field.Name)
		case !exempt[field.Name] && active:
			t.Errorf("%s does not yield Copy Selection — wrap it in yieldingMenuCallbacks or, if it must keep the mode, add it to this test's exempt list", field.Name)
		}
		v.cancelRegionCopy()

		if ran[field.Name] != 1 {
			t.Errorf("%s ran %d times, want exactly once", field.Name, ran[field.Name])
		}
	}
}
```

- [x] **Step 2: Run it — expect PASS, then prove it can fail**

Run: `go test ./internal/ui -run TestYieldingMenuCallbacksWrapsEveryField -count=1`
Expected: PASS (every current field is wrapped).

Now verify the test actually detects the defect it guards against: in `internal/ui/menu.go`, temporarily delete the line `c.Trash = v.yieldThen(c.Trash)`, re-run the test, and confirm it FAILS with "Trash does not yield Copy Selection". Restore the line, re-run, confirm PASS. Do not commit the temporary edit.

- [ ] **Step 3: Commit**

```bash
git add internal/ui/menu_yield_test.go
git commit -m "Guard yieldingMenuCallbacks completeness with a reflection test."
```

---

### Task 6: Capture the displayed frame instead of re-rotating, and release the pause on failure (findings 6 and 8)

`captureRegionCopySource` calls `display.Rotated()`, which allocates a fresh full-size RGBA for any rotated image — a duplicate of the oriented frame `redrawRotatedFrame` already stored in `v.img.Image` — and the Source pins it for the whole mode (~48MB for a rotated 12MP photo). Capture `v.img.Image` instead. Same function, second defect: when capture yields nil, the acquired animation pause leaks; release it where it was acquired.

Equivalence argument (verified during review — rely on it): every raster path writes the displayed frame through `redrawRotatedFrame` (`internal/ui/rotate.go:65`), which stores exactly `display.Rotated()` into `v.img.Image`; SVG takes the `VectorSource` branch before the raster code runs; while the mode is active nothing rewrites `v.img.Image` (animations are paused by this very function, and every rotation/navigation path yields the mode first after Tasks 2–3).

**Files:**
- Modify: `internal/ui/copyselection.go` (`captureRegionCopySource`, lines 54–78)
- Test: `internal/ui/copyselection_yield_test.go`

**Interfaces:**
- Consumes: Tasks 2 and 3 (the "nothing rewrites v.img.Image while active" argument needs their yields in place — do not reorder).
- Produces: `captureRegionCopySource` now returns `animated == false` whenever `ok == false`; it owns releasing the pause on its own failure. `startRegionCopy` still owns the release when `Feature.Start` fails afterwards.

- [x] **Step 1: Write the failing test**

Append to `internal/ui/copyselection_yield_test.go` (add `"fyne.io/fyne/v2/storage"` to the imports):

```go
// A capture that produces no frame must release the animation pause it
// acquired: the caller only cleans up after a failed Start, not a failed
// capture, and a held pause freezes the animation loop for the session.
func TestCaptureRegionCopySourceReleasesPauseWhenCaptureFails(t *testing.T) {
	v := newTestViewer(t)
	path := uitest.WriteTempFile(t, "anim.gif", uitest.EncodeAnimatedGIF(t, 4, 3,
		[]color.Color{color.NRGBA{R: 255, A: 255}, color.NRGBA{B: 255, A: 255}},
		[]int{1000, 1000}))
	dropAndWait(t, v, storage.NewFileURI(path))
	if v.display.Count() < 2 {
		t.Fatal("precondition: the dropped GIF is not animated")
	}

	v.img.Image = nil // simulate the display layer handing back no frame
	_, animated, ok := v.captureRegionCopySource()
	if ok || animated {
		t.Fatalf("capture with no frame = (animated=%v, ok=%v), want (false, false)", animated, ok)
	}

	if !v.animationPause.pause(func() {}) {
		t.Fatal("failed capture left the animation pause held")
	}
	v.animationPause.unpause()
}
```

- [x] **Step 2: Run it — expect failure**

Run: `go test ./internal/ui -run TestCaptureRegionCopySourceReleasesPauseWhenCaptureFails -count=1`
Expected: FAIL. Today the nil-frame path is only reachable once capture reads `v.img.Image` (Step 3), so before the change the test fails at the `(animated=%v, ok=%v)` check — `display.Rotated()` still returns a frame — or, if it does return nil, at the held-pause check. Either failure is the defect being pinned.

- [x] **Step 3: Rewrite the raster half of captureRegionCopySource**

In `internal/ui/copyselection.go`, replace lines 64–77 (from `animated = v.display.Count() > 1` through the `return copyselection.RasterSource(...)` line) with:

```go
	animated = v.display.Count() > 1
	var raster image.Image
	// v.img.Image is the displayed oriented frame redrawRotatedFrame keeps
	// current for every raster path. Capturing it instead of re-running
	// display.Rotated() avoids a second full-size RGBA that the Source
	// would pin for the whole mode. It stays stable while the mode is
	// active: animations are paused right here, and every rotation or
	// navigation path yields the mode before touching the frame.
	capture := func() { raster = v.img.Image }
	if animated {
		if !v.animationPause.pause(capture) {
			return copyselection.Source{}, false, false
		}
	} else {
		capture()
	}
	if raster == nil {
		// Release what this function acquired: the caller cleans up only
		// after a failed Start, not after a failed capture.
		if animated {
			v.animationPause.unpause()
		}
		return copyselection.Source{}, false, false
	}
	return copyselection.RasterSource(raster), animated, true
```

- [x] **Step 4: Run the new test and every Copy Selection test**

Run: `go test ./internal/ui -run 'TestCaptureRegionCopySourceReleasesPauseWhenCaptureFails|TestCopySelection' -count=1 && go test ./internal/ui/copyselection -count=1`
Expected: PASS — the existing pixel-equality tests (`copyselection_pixels_test.go`, `copyselection_worker_test.go`, `copyselection_sources_test.go`) are the equivalence proof for the capture change, including the rotated and animated cases.

- [ ] **Step 5: Commit**

```bash
git add internal/ui/copyselection.go internal/ui/copyselection_yield_test.go
git commit -m "Capture Copy Selection from the displayed frame and release the pause on failure."
```

---

### Task 7: Geometry callback stops dispatching while inactive and stops repainting the world (finding 7)

The zoom geometry callback fires on every zoom/pan/resize frame for the app's lifetime. The `Active` check sits *inside* the queued closure, so the common inactive path pays two closure allocations and a queue hop per frame; while active, `ForceRepaint()` re-refreshes the entire window tree — including re-uploading the photo texture — per scroll tick. `ViewChanged → syncChrome` already refreshes the selection overlay, which is all that needs painting.

Reading `regionCopy.State()` synchronously in the callback is safe: `zoom.SetOnGeometryChanged`'s contract (internal/ui/zoom/zoom.go:173) forbids *mutating widgets* synchronously, not reading; the mutation still crosses `regionCopyDo`.

**Files:**
- Modify: `internal/ui/features.go` (lines 54–69)
- Test: `internal/ui/copyselection_yield_test.go`

**Interfaces:**
- Consumes: nothing from other tasks.
- Produces: nothing.

- [x] **Step 1: Write the failing test**

Append to `internal/ui/copyselection_yield_test.go` (add `"fyne.io/fyne/v2/test"` and `"fyne.io/fyne/v2"` to the imports if not already present):

```go
// Zoom geometry changes fire for the app's whole lifetime; while Copy
// Selection is inactive they must not queue UI work at all.
func TestZoomGeometryCallbackSkipsInactiveMode(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "photo.jpg", 800, 400, color.White))

	dispatches := 0
	v.regionCopyDo = func(f func()) {
		dispatches++
		f()
	}

	test.Scroll(v.win.Canvas(), fyne.NewPos(100, 100), 0, 20)

	if dispatches != 0 {
		t.Fatalf("geometry dispatches while Copy Selection is inactive = %d, want 0", dispatches)
	}
}
```

- [x] **Step 2: Run it — expect failure**

Run: `go test ./internal/ui -run TestZoomGeometryCallbackSkipsInactiveMode -count=1`
Expected: FAIL with a non-zero dispatch count.

- [x] **Step 3: Rewrite the callback**

In `internal/ui/features.go`, replace the whole `view.zoom.SetOnGeometryChanged(...)` call (lines 54–69) with:

```go
	view.zoom.SetOnGeometryChanged(func(geometry zoom.Geometry) {
		// Delivered synchronously, possibly inside imageRenderer.Layout —
		// see zoom.SetOnGeometryChanged. Reading mode state here is safe;
		// mutating widgets is not, so the update still crosses
		// regionCopyDo. The Active read is hoisted so the common inactive
		// frame pays no closure or queue hop, and re-checked inside because
		// the mode can end between queueing and running. No ForceRepaint:
		// ViewChanged's own chrome sync refreshes the selection overlay,
		// and zoom's apply already painted the image.
		if !view.regionCopy.State().Active {
			return
		}
		do := view.regionCopyDo
		if do == nil {
			do = fyne.Do
		}
		do(func() {
			if !view.regionCopy.State().Active {
				return
			}
			view.regionCopy.ViewChanged(copyselection.View{
				Position: geometry.Position,
				Size:     geometry.Size,
			})
		})
	})
```

- [x] **Step 4: Run the tests**

Run: `go test ./internal/ui -run 'TestZoomGeometryCallbackSkipsInactiveMode|TestCopySelection' -count=1`
Expected: PASS — `TestCopySelectionZoomPanResize` still counts dispatches because it starts the mode before scrolling; it now also implicitly verifies the active path still delivers.

- [ ] **Step 5: Commit**

```bash
git add internal/ui/features.go internal/ui/copyselection_yield_test.go
git commit -m "Skip geometry dispatch while Copy Selection is inactive and stop full repaints."
```

---

### Task 8: One crop-and-encode pipeline (finding 9)

`Feature.Encode` forks: production runs `Source.Encode` (pixels → cropBounds → PNG) while the test seam runs a hand-copied pixels → cropBounds → `f.encode` sequence. A step added to one pipeline later silently misses the other. Collapse to a single `encodeWith` pipeline with the encoder injected; the seam swaps only the final encode step.

**Files:**
- Modify: `internal/ui/copyselection/source.go` (`Encode`, lines 34–45)
- Modify: `internal/ui/copyselection/copyselection.go` (`New` line 57, `Encode` line 149, `SetEncode` line 165)

**Interfaces:**
- Consumes: nothing from other tasks.
- Produces: `Source.encodeWith(encode func(image.Image, image.Rectangle) ([]byte, error), bounds image.Rectangle) ([]byte, error)` (unexported); `Feature.SetEncode(nil)` now restores the PNG default (the worker test at `internal/ui/copyselection_worker_test.go:130` already relies on nil meaning "back to PNG").

- [x] **Step 1: Rewrite Source.Encode around encodeWith**

In `internal/ui/copyselection/source.go`, replace `Encode` (lines 34–45) with:

```go
// Encode returns a zero-origin PNG of bounds from the captured source.
func (s Source) Encode(bounds image.Rectangle) ([]byte, error) {
	return s.encodeWith(PNG, bounds)
}

// encodeWith is the one crop-and-encode pipeline: resolve pixels, map
// bounds into them, hand both to encode. Feature.Encode injects its
// configurable encoder here; any step added to this pipeline reaches the
// seam-installed test encoders too, which is the point of there being
// exactly one.
func (s Source) encodeWith(encode func(image.Image, image.Rectangle) ([]byte, error), bounds image.Rectangle) ([]byte, error) {
	pixels, err := s.pixels()
	if err != nil {
		return nil, err
	}
	crop, err := s.cropBounds(bounds, pixels.Bounds())
	if err != nil {
		return nil, err
	}
	return encode(pixels, crop)
}
```

- [x] **Step 2: Default the Feature's encoder and drop the fork**

In `internal/ui/copyselection/copyselection.go`:

In `New` (line 58), change the constructor line to seed the default encoder:

```go
	f := &Feature{callbacks: callbacks, encode: PNG}
```

Replace `Encode` (lines 148–162) with:

```go
// Encode returns a zero-origin encoding of bounds from the source captured
// at Start — PNG unless SetEncode replaced the final step.
func (f *Feature) Encode(bounds image.Rectangle) ([]byte, error) {
	return f.source.encodeWith(f.encode, bounds)
}
```

Replace `SetEncode` (lines 164–167) with:

```go
// SetEncode replaces the encoding step that runs after crop, for tests
// that need a failing or recording encoder. nil restores the PNG default.
// Every call still runs the one Source pipeline; only the last step swaps.
func (f *Feature) SetEncode(fn func(image.Image, image.Rectangle) ([]byte, error)) {
	if fn == nil {
		fn = PNG
	}
	f.encode = fn
}
```

- [x] **Step 3: Run both packages' suites**

Run: `go test ./internal/ui/copyselection ./internal/ui -run '.' -count=1`
Expected: PASS. The worker tests (`TestCopySelectionEncodeFailure` installs a failing encoder, then `SetEncode(nil)` and retries) exercise both the seam and the restored default through the unified pipeline.

- [ ] **Step 4: Commit**

```bash
git add internal/ui/copyselection/source.go internal/ui/copyselection/copyselection.go
git commit -m "Collapse Copy Selection encoding onto one pipeline with an injected encoder."
```

---

### Task 9: One definition each for logical-size rounding and orientation swap (finding 10)

The round-SVG-logical-size and swap-axes-when-rotation-is-odd computations exist in four places. Consolidate: one rounding helper in `internal/ui` (shared by `displayedDimensions` and the capture site), one orientation helper in `internal/ui/copyselection` (shared by `Bounds` and `cropBounds`). `applyRotationLayout`'s float32 swap stays — it feeds zoom a `fyne.Size`, a different space — but gets a cross-reference.

**Files:**
- Modify: `internal/ui/rotate.go` (`displayedDimensions` line 136, `applyRotationLayout` line 86)
- Modify: `internal/ui/copyselection.go` (vector branch of `captureRegionCopySource`, line 55)
- Modify: `internal/ui/copyselection/source.go` (`Bounds` line 50, `cropBounds` line 90)

**Interfaces:**
- Consumes: Task 6's version of `captureRegionCopySource` (this task edits its vector branch; Task 6 edited the raster branch).
- Produces: `roundedLogical(l fyne.Size) (w, h int)` in package `ui`; `(s Source) orientedLogical() (w, h int)` in package `copyselection`.

- [x] **Step 1: Add the rounding helper and use it in displayedDimensions**

In `internal/ui/rotate.go`, append at the end of the file:

```go
// roundedLogical is the one rounding from a vector's float logical size to
// the integer pixel space the app reports and crops in. displayedDimensions
// and the Copy Selection capture must agree on it exactly, or the crop
// space drifts off-by-one from the dimensions the UI reports.
func roundedLogical(l fyne.Size) (w, h int) {
	return int(l.Width + 0.5), int(l.Height + 0.5)
}
```

In `displayedDimensions` (lines 136–148), replace the vector branch (the lines after the `if v.vector.svg == nil` block) with:

```go
	w, h = roundedLogical(v.vector.logical)
	if v.display.Rotation()%2 != 0 {
		w, h = h, w
	}

	return w, h
```

(Swapping before or after rounding is equivalent — each axis rounds independently.)

- [x] **Step 2: Use it at the capture site**

In `internal/ui/copyselection.go`, replace the vector branch of `captureRegionCopySource` (lines 55–62) with:

```go
	if v.vector.svg != nil {
		w, h := roundedLogical(v.vector.logical)
		return copyselection.VectorSource(
			v.vector.svg,
			image.Pt(w, h),
			v.display.Rotation(),
			v.vector.rasterize,
		), false, true
	}
```

- [x] **Step 3: Add the orientation helper in copyselection**

In `internal/ui/copyselection/source.go`, add after `VectorSource` (line 32):

```go
// orientedLogical is the captured SVG's logical size with view-only quarter
// turns applied — the single definition of the mode's vector image-space.
// Bounds and cropBounds both read it, so the rectangle the selection is
// drawn against and the rectangle the crop is validated against cannot
// drift apart.
func (s Source) orientedLogical() (w, h int) {
	w, h = s.logical.X, s.logical.Y
	if s.rotation%2 != 0 {
		w, h = h, w
	}
	return w, h
}
```

In `Bounds` (lines 50–66), replace the vector branch with:

```go
	if s.vector != nil {
		w, h := s.orientedLogical()
		if w <= 0 || h <= 0 {
			return image.Rectangle{}
		}
		return image.Rect(0, 0, w, h)
	}
```

In `cropBounds` (lines 95–98), replace:

```go
	logical := image.Rect(0, 0, s.logical.X, s.logical.Y)
	if s.rotation%2 != 0 {
		logical = image.Rect(0, 0, s.logical.Y, s.logical.X)
	}
```

with:

```go
	w, h := s.orientedLogical()
	logical := image.Rect(0, 0, w, h)
```

- [x] **Step 4: Cross-reference the float swap that stays**

In `internal/ui/rotate.go`, inside `applyRotationLayout`'s vector branch (the comment block above `if v.vector.svg != nil`, line 82), append one sentence to the existing comment:

```go
	// The float swap below is the fyne.Size twin of the integer
	// swap-and-round in displayedDimensions/roundedLogical; zoom needs the
	// unrounded size, so the two deliberately do not share code.
```

- [x] **Step 5: Run the affected suites**

Run: `go test ./internal/ui/copyselection ./internal/ui -count=1`
Expected: PASS — the SVG source tests (`source_test.go`, `copyselection_sources_test.go`) and rotation tests (`rotate_test.go`) pin the behavior these helpers must reproduce exactly.

- [ ] **Step 6: Commit**

```bash
git add internal/ui/rotate.go internal/ui/copyselection.go internal/ui/copyselection/source.go
git commit -m "Share one oriented-logical-size definition per package."
```

---

### Task 10: Full verification

- [x] **Step 1: Full build, vet, and test run**

Run: `go build ./... && go vet ./... && go test ./... -count=1`
Expected: everything passes. `-count=1` defeats the test cache — the point is a fresh run of the whole module.

- [x] **Step 2: Race check on the touched packages**

Run: `go test -race ./internal/ui ./internal/ui/copyselection ./internal/ui/favorites ./internal/ui/settingswin -count=1`
Expected: PASS with no race reports. The capture change (Task 6) and the geometry callback (Task 7) are the reason this step exists — do not skip it.

- [x] **Step 3: Confirm the working tree is clean and every commit landed**

Run: `git status && git log --oneline main..HEAD`
Expected: clean tree; the nine task commits on top of the branch's existing history.
