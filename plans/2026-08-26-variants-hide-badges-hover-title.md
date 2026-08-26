# Variants Grid: Hide Count Pills and Hover Title

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** In Show variants (`Shift+D`) mode, hide duplicate-count pills on every cell, and set the window title for the highlighted file to `(n/m) [WxH] /absolute/path.jpg` with no merge/sort/shuffle prefixes.

**Architecture:** Badge visibility stays inside `internal/ui/grid` (`applyDupBadge`). Native pixel size already exists as a URI-keyed pixel *count* used to pick the highest-resolution representative; extend that store to keep width×height so the title can show dimensions without a new decode. Title formatting stays on `viewer.HighlightChanged` / `applyTitle` (grid reports the host index; it does not format window titles). Hover already drives `GridWrap.OnHighlighted` → `setHighlight` → `HighlightChanged`, so keyboard and pointer share one path.

**Tech Stack:** Go, Fyne `widget.GridWrap`, `internal/imaging.LoadThumbnailAndBounds` / `ReadAndProbe` (EXIF-oriented native size), Fyne test driver (`newTestUI` / `newTestViewer` / `dropAndWait` / `Settle`).

## Global Constraints

- Do not pass `appState` into `grid`; keep the 10-method `Host`. Add a getter on `Overview`, not a new Host method, for native size.
- Do not add a parallel open/load path; dimensions come from the existing probe recorded during Warm / `hashRemaining` / thumbnail decode (`rememberNative`).
- Do not add mutable package-level test seams. Runtime values stay on `Overview` / `viewer`.
- Every user-visible *word* goes through `lang.L` and every `translations/*.json` bundle. This title format is numbers, brackets, and a path — no new translation keys unless a later decision adds words.
- Report UI-boundary failures with `fyne.LogError`; ignored errors marked `_ =`.
- UI tests use `newTestUI` / `newTestViewer`, `dropAndWait`, `waitFor*`, `grid.Settle`. Never `time.Sleep` to guess completion.
- Update `internal/ui/help/manual.md` and `manual_de.md` for documented behavior. `ARCHITECTURE.md` only if the package map changes (it should not).
- Open work belongs in `todos.md`; no `TODO`/`FIXME` in source.
- Match CI: `gofmt`, `go vet ./...`, `go build ./...`, focused tests while iterating, then `go test -race ./...` from the repo root before handoff.
- Subagents do **not** `git commit`. After each task the parent reviews the diff, fixes if needed, then commits and pushes.

---

## Decisions (locked)

1. **Title scope.** Variants browse grid only. Hide-duplicates and the normal grid keep `filename  (n/m)`.
2. **Hover vs keyboard.** Title follows the **highlighted** cell (hover *and* keyboard). They already share `OnHighlighted`.
3. **Exact format (user).** `(2/7) [1440x780] /absolute/path.jpg`
   - Position first, then a space, then `[WxH]`, then a space, then the filesystem path (`URI.Path()`), **not** wrapped in parentheses, not `file://`, not basename-only.
   - No spaces around `x` inside the brackets.
   - `(n/m)` is the highlighted file’s **host index in the full loaded set** (same as today’s grid counter: `i+1` and `len(files)`), not the index inside the variant group. Omit `(n/m)` only when `len(files) == 1` (variants browse itself requires a group of 2+, so this is defensive).
   - Example for file index 1 of 3 at 192×144: `(2/3) [192x144] /tmp/.../b.jpg`
4. **Mode prefixes (user).** While the variants grid is showing, **do not** prepend `[merge]`, sort-order, or `[shuffle]`. `applyTitle` skips those prefixes when `BrowsingDuplicates()` is on and `gridTitle` is set. Leaving variants restores the usual prefixes. `TestGridHighlight_TitleKeepsTheModePrefixes` (normal grid) must still pass.
5. **Position counter (user).** Prepend `(n/m)` as in decision 3. Do **not** also append a second counter.
6. **Unknown size.** If native WxH is missing or 0×0: `(2/7) /absolute/path.jpg` (counter + path, no `[?]`). `finishBrowse` / `setHighlight` refreshes when the probe lands.
7. **Pointer leaving a cell.** No MouseOut title restore. Closing variants / the grid hands the title back via `HighlightChanged(-1)`.
8. **Inspect / image view.** Unchanged (`name — W x H  (n/m)`). Image-view titles keep merge/sort/shuffle prefixes.

---

## File map

| File | Responsibility |
|------|----------------|
| `internal/ui/grid/nav.go` | `applyDupBadge`: hide the chip while `BrowsingDuplicates()`. `SimulateHover` for tests (GridWrap `OnHighlighted`). |
| `internal/ui/grid/dupes_test.go` | Badge visibility tests; native-size getter tests. |
| `internal/ui/grid/grid.go` | `Overview` fields: replace URI→pixel-count map with URI→`image.Point` (or equivalent) so WxH survives. |
| `internal/ui/grid/dupes.go` | `rememberNative`, `pixelCountOf` (derived), new `NativeSize(hostIndex int) (w, h int, ok bool)`, `computeDuplicateGroups` pixel reads, map init/wipe/clear. |
| `internal/ui/grid/thumbs.go` | No behavior change if `rememberNative` keeps the same signature. |
| `internal/ui/viewer.go` | `gridHighlightTitle` + `applyTitle`: variants title `(n/m) [WxH] path` with prefixes suppressed; otherwise existing basename + counter and prefixes. |
| `internal/ui/grid_test.go` | Viewer-level title tests (hover + leave variants). |
| `internal/ui/help/manual.md` | Grid overview + window-title sections. |
| `internal/ui/help/manual_de.md` | Same in German. |
| `todos.md` | Not required unless this item is tracked there; do not add source TODOs. |

No new packages. `ARCHITECTURE.md` unchanged unless a file’s one-line description needs a native-size mention (optional, skip unless the table becomes wrong).

---

## Delegation

Execute **strictly in order**. Parent reviews the full diff after every task and fixes before the next dispatch. One implementer subagent per task; do not give a subagent later tasks.

| Task | Subagent type | Model | Why |
|------|---------------|-------|-----|
| 1 Hide pills | `go-expert` | `cursor-grok-4.6-high-fast` | Tiny, local predicate + one test. Fast model is enough. |
| 2 Native WxH store | `go-expert` | `claude-sonnet-5-thinking-high` | Touches the hash/native maps, grouping, and several tests. Needs careful Go and lock discipline. |
| 3 Variants title | `go-expert` | `claude-sonnet-5-thinking-high` | Viewer/grid composition, title prefixes, Host stay-narrow. |
| 4 Viewer hover tests | `go-expert` | `cursor-grok-4.6-high-fast` | Tests only against Task 3 API; follow existing `grid_test.go` / `step_test.go` fixtures. |
| 5 Manuals | `generalPurpose` | `composer-2.5` | EN/DE copy only; no Go. |

Do **not** use Opus unless a task is blocked after one failed implementer pass. If Task 2’s map migration keeps failing review, re-dispatch Task 2 on `claude-opus-5-thinking-high`.

Parent review checklist (every task): Host still 10 methods; no package-level test seams; badges hidden iff `BrowsingDuplicates()`; `pixelCountOf` still native Dx×Dy (not thumbnail bounds); variants title is `(n/m) [WxH] path` with **no** merge/sort/shuffle prefixes; tests wait via `Settle` / `dropAndWait`, not sleep; `lang.L` unused for this format; manuals only in Task 5.

---

### Task 1: Hide duplicate-count pills in Show variants

**Files:**
- Modify: `internal/ui/grid/nav.go` (`applyDupBadge`)
- Test: `internal/ui/grid/dupes_test.go` (`TestApplyDupBadge_ShowsGroupSize` plus a dedicated browse-mode test)

**Interfaces:**
- Consumes: `Overview.hideDupes`, `Overview.BrowsingDuplicates()`, `Overview.groupSize`
- Produces: chip hidden when browse is on, even if hide-duplicates stays on (hide stays on during variants by design)

- [ ] **Step 1: Write the failing test**

Add after `TestApplyDupBadge_ShowsGroupSize` in `internal/ui/grid/dupes_test.go`:

```go
func TestApplyDupBadge_HiddenWhileBrowsingDuplicates(t *testing.T) {
	g, _ := pairAndUnique(t)

	g.SetHideDuplicates(true)
	cell := newGridCell()
	_, _, _, badge := unpackGridCell(cell)
	g.applyDupBadge(badge, 0, fyne.NewSize(cellSize, cellSize))
	if !badge.chip.Visible() {
		t.Fatal("setup: hide-duplicates should show the group-size chip")
	}

	g.SetBrowsingDuplicates(true)
	g.applyDupBadge(badge, 0, fyne.NewSize(cellSize, cellSize))
	if badge.chip.Visible() {
		t.Fatal("variants browse must hide the group-size chip")
	}

	g.SetBrowsingDuplicates(false)
	g.applyDupBadge(badge, 0, fyne.NewSize(cellSize, cellSize))
	if !badge.chip.Visible() || badge.label.Text != "2" {
		t.Errorf("after leaving browse, chip visible=%v text=%q, want visible \"2\"", badge.chip.Visible(), badge.label.Text)
	}
}
```

`pairAndUnique` already builds a 2-file duplicate group plus a unique. `SetBrowsingDuplicates(true)` on host index 0 is valid. Direct `applyDupBadge` calls are the existing style (`TestApplyDupBadge_ShowsGroupSize`); `wrap.Refresh` is not required for this unit test.

- [ ] **Step 2: Run the test and confirm it fails**

```bash
go test -run TestApplyDupBadge_HiddenWhileBrowsingDuplicates -count=1 ./internal/ui/grid/
```

Expected: FAIL with the variants-browse fatal (chip still visible). `hideDupes` stays true during browse, which is why today’s predicate shows the pill.

- [ ] **Step 3: Minimal implementation**

In `internal/ui/grid/nav.go`, change `applyDupBadge`:

```go
func (g *Overview) applyDupBadge(b *dupBadge, hostIndex int, cell fyne.Size) {
	n := g.groupSize(hostIndex)
	if !g.hideDupes || g.BrowsingDuplicates() || n < 2 {
		b.chip.Hide()
		return
	}
	b.label.Text = strconv.Itoa(n)
	b.label.Refresh()
	sz := b.chip.MinSize()
	b.chip.Resize(sz)
	w := cell.Width
	if w <= 0 {
		w = cellSize
	}
	b.chip.Move(fyne.NewPos(w-sz.Width-dupBadgeMargin, dupBadgeMargin))
	b.chip.Show()
}
```

`SetBrowsingDuplicates` already calls `applyFilter` → `wrap.Refresh` → cell update → `applyDupBadge`, so live cells hide without extra wiring.

- [ ] **Step 4: Run tests**

```bash
go test -run 'TestApplyDupBadge|TestDupBadge_' -count=1 ./internal/ui/grid/
```

Expected: PASS, including `TestDupBadge_TopRightClearsTheHighlightRing` (hide on, browse off).

- [ ] **Step 5: Stop for parent review**

Do not commit. Report files changed and the test command output.

---

### Task 2: Keep native width×height, not only pixel count

**Files:**
- Modify: `internal/ui/grid/grid.go` (field on `Overview`)
- Modify: `internal/ui/grid/dupes.go` (`ensureHashGenLocked`, `adoptHashGen`, `rememberNative`, `pixelCountOf`, `clearHashes`, `computeDuplicateGroups`, New init is in `grid.go`)
- Modify: `internal/ui/grid/dupes_test.go` (any test that writes `g.pixels` directly; add `NativeSize` tests)
- Test: `internal/ui/grid/dupes_test.go`

**Interfaces:**
- Consumes: `rememberNative(u fyne.URI, native image.Rectangle)` already called from `Warm`, `requestThumbnail`, and `hashRemaining`
- Produces: `func (g *Overview) NativeSize(hostIndex int) (w, h int, ok bool)` — `ok` is false when the index is out of range, the URI is missing, or either edge is ≤ 0. `pixelCountOf` remains and returns `w*h` for grouping/representative selection.

Do **not** change `Host`. Do **not** probe on the UI goroutine from `NativeSize`.

- [ ] **Step 1: Write failing tests**

Add to `internal/ui/grid/dupes_test.go` (same package, can call unexported helpers):

```go
func TestNativeSize_ReportsProbeWidthAndHeight(t *testing.T) {
	u := uitest.TempJPEGURI(t, "big.jpg", 800, 400, color.RGBA{R: 200, G: 20, B: 20, A: 255})
	host := &fakeHost{files: []fyne.URI{u}}
	g := newOverview(t, host)
	if err := g.Warm(); err != nil {
		t.Fatalf("Warm: %v", err)
	}

	w, h, ok := g.NativeSize(0)
	if !ok {
		t.Fatal("NativeSize(0) ok=false, want the Warm probe")
	}
	if w != 800 || h != 400 {
		t.Errorf("NativeSize(0) = %dx%d, want 800x400 (not thumbnail size)", w, h)
	}

	px, pok := g.pixelCountOf(u)
	if !pok || px != 800*400 {
		t.Errorf("pixelCountOf = %d ok=%v, want %d true (grouping still uses Dx*Dy)", px, pok, 800*400)
	}

	if _, _, ok := g.NativeSize(-1); ok {
		t.Error("NativeSize(-1) must be !ok")
	}
	if _, _, ok := g.NativeSize(1); ok {
		t.Error("NativeSize past FileCount must be !ok")
	}
}

func TestNativeSize_UnknownWhenUnprobed(t *testing.T) {
	u := uitest.TempJPEGURI(t, "a.jpg", 8, 8, color.White)
	host := &fakeHost{files: []fyne.URI{u}}
	g := newOverview(t, host)
	if _, _, ok := g.NativeSize(0); ok {
		t.Fatal("unprobed file must not report a size")
	}
}
```

Keep `TestWarm_RecordsNativePixelCountNotThumbnailSize` — it must still pass after the map change.

- [ ] **Step 2: Run tests and confirm NativeSize fails**

```bash
go test -run 'TestNativeSize_' -count=1 ./internal/ui/grid/
```

Expected: FAIL compile (`NativeSize` undefined) or FAIL assertions.

- [ ] **Step 3: Store `image.Point` and derive pixel count**

Replace the URI→int `pixels` map with URI→`image.Point` named `native` (or keep the field name `pixels` only if every comment is updated — prefer `native`).

In `internal/ui/grid/grid.go` on `Overview`, replace:

```go
	// hashes maps URI string → dHash. Not stored in thumbs: a hash is 8
	// bytes and must survive thumbnail eviction. native maps URI string
	// → EXIF-oriented pixel size (Dx, Dy) for the same generation;
	// absent means unknown. Thumbnails are capped, so size cannot be
	// recovered from the thumb cache. hashGen is the host Generation
	// those entries belong to; a newer drop wipes hashes, hashFailed,
	// and native.
	hashMu sync.Mutex
	hashes map[string]uint64
	native map[string]image.Point
```

`image` is already imported in `grid.go`.

Init in `New` (`pixels: make(map[string]int)` → `native: make(map[string]image.Point)`).

In `dupes.go`:

- Every `g.pixels = make(map[string]int)` becomes `g.native = make(map[string]image.Point)` (also the nil-init branches in `ensureHashGenLocked` / `adoptHashGen`).
- `rememberNative`:

```go
func (g *Overview) rememberNative(u fyne.URI, native image.Rectangle) {
	if u == nil {
		return
	}
	sz := image.Pt(max(native.Dx(), 0), max(native.Dy(), 0))
	gen := g.host.Generation()
	g.hashMu.Lock()
	defer g.hashMu.Unlock()
	g.ensureHashGenLocked(gen)
	g.native[u.String()] = sz
}
```

- `pixelCountOf` stays; implement via `nativeSizeOf`:

```go
func (g *Overview) nativeSizeOf(u fyne.URI) (image.Point, bool) {
	if u == nil {
		return image.Point{}, false
	}
	g.wipeHashesIfStale()
	g.hashMu.Lock()
	defer g.hashMu.Unlock()
	sz, ok := g.native[u.String()]
	return sz, ok
}

func (g *Overview) pixelCountOf(u fyne.URI) (int, bool) {
	sz, ok := g.nativeSizeOf(u)
	if !ok {
		return 0, false
	}
	return sz.X * sz.Y, true
}

// NativeSize is the EXIF-oriented pixel size of the file at hostIndex.
// ok is false when the index is out of range or no probe has been stored,
// or when a stored size has a non-positive edge (failed/empty probe).
func (g *Overview) NativeSize(hostIndex int) (w, h int, ok bool) {
	if hostIndex < 0 || hostIndex >= g.host.FileCount() {
		return 0, 0, false
	}
	sz, ok := g.nativeSizeOf(g.host.FileAt(hostIndex))
	if !ok || sz.X <= 0 || sz.Y <= 0 {
		return 0, 0, false
	}
	return sz.X, sz.Y, true
}
```

- `computeDuplicateGroups` currently copies `g.pixels` into `px []int` under `hashMu`. Change that read to `p.X * p.Y` from `g.native`. Do not change `DuplicateGroups` or representative rules (max pixels, then lowest host index).

- `clearHashes` / stale wipe must drop `native` the same way they dropped `pixels`.

Tests that assign `g.pixels = map[string]int{...}` must assign `g.native` with `image.Pt` values whose product matches the old counts (see around the representative-resolution tests near the end of `dupes_test.go`). Tests that only call `pixelCountOf` should keep working.

Zero rectangle probes (`hashRemaining` stores `image.Rectangle{}` when `ReadAndProbe` fails) become `Pt(0,0)`, `pixelCountOf` = 0 with ok true (same as today for grouping), `NativeSize` = !ok.

- [ ] **Step 4: Run tests**

```bash
go test -count=1 ./internal/ui/grid/
```

Expected: PASS. If anything still references `g.pixels`, fix it in this task.

- [ ] **Step 5: Stop for parent review**

Confirm grouping still prefers the highest `Dx*Dy` (existing `TestStepImage_HideDuplicatesShowsHighestResolution` is viewer-level and can wait for Task 4; grid-level representative tests in `dupes_test.go` must pass now).

---

### Task 3: Variants highlight title `(n/m) [WxH] path`

**Files:**
- Modify: `internal/ui/viewer.go` (`HighlightChanged`, `gridHighlightTitle`, `applyTitle`)
- Test: `internal/ui/grid_test.go` (first title assertion can live here; Task 4 adds hover/leave coverage)

**Interfaces:**
- Consumes: `v.grid.BrowsingDuplicates()`, `v.grid.NativeSize(i)`, `v.state.files[i].Name()`, `v.state.files[i].Path()`
- Produces: `gridTitle` consumed by `applyTitle`. While `BrowsingDuplicates()` and `gridTitle != ""`, `applyTitle` skips `[merge]` / `[shuffle]` / sort-order prefixes.

Do **not** decode in `HighlightChanged`. Do **not** change `Host.HighlightChanged(i int)`.

- [ ] **Step 1: Write a failing viewer test**

In `internal/ui/grid_test.go`, next to `TestGridHighlight_NamesTheHighlightedFileInTheTitle`:

```go
func TestGridHighlight_VariantsTitleUsesSizeAndPath(t *testing.T) {
	v := newTestViewer(t)
	small := uitest.PatternedJPEGURISize(t, "a.jpg", 1, 64, 48)
	large := uitest.PatternedJPEGURISize(t, "b.jpg", 1, 192, 144)
	unique := uitest.PatternedJPEGURI(t, "c.jpg", 99)
	dropAndWait(t, v, small, large, unique)
	if err := v.grid.Warm(); err != nil {
		t.Fatalf("Warm: %v", err)
	}

	v.SetMergeMode(true)
	v.grid.SetHideDuplicates(true)
	v.grid.Settle()
	v.grid.Toggle()
	v.grid.SetBrowsingDuplicates(true)
	v.grid.Settle()
	if !v.grid.BrowsingDuplicates() || !v.grid.Visible() {
		t.Fatal("premises: variants grid up")
	}

	title := v.win.Title()
	wantSmall := fmt.Sprintf("(1/3) [64x48] %s", small.Path())
	wantLarge := fmt.Sprintf("(2/3) [192x144] %s", large.Path())
	if title != wantSmall && title != wantLarge {
		t.Fatalf("variants title = %q, want %q or %q", title, wantSmall, wantLarge)
	}
	if strings.HasPrefix(title, "[merge]") {
		t.Errorf("variants title = %q, must not include [merge]", title)
	}
	if strings.Contains(title, "a.jpg  (") || strings.Contains(title, " — ") {
		t.Errorf("variants title = %q, must not use basename or image-view format", title)
	}

	v.grid.SetBrowsingDuplicates(false)
	v.grid.Settle()
	title = v.win.Title()
	if strings.Contains(title, "[64x48]") || strings.Contains(title, "[192x144]") {
		t.Errorf("after leaving variants, title = %q, want the basename grid title back", title)
	}
	if !strings.HasPrefix(title, "[merge] ") {
		t.Errorf("after leaving variants, title = %q, want [merge] restored", title)
	}
	if !strings.Contains(title, ".jpg") {
		t.Errorf("after leaving variants, title = %q, want a file name", title)
	}
}
```

Add `"fmt"` to `grid_test.go` imports if missing.

- [ ] **Step 2: Run it and confirm it fails**

```bash
go test -run TestGridHighlight_VariantsTitleUsesSizeAndPath -count=1 ./internal/ui/
```

Expected: FAIL (title still `[merge] a.jpg  (n/m)` or similar).

- [ ] **Step 3: Format the title in the viewer and skip prefixes**

`applyTitle` today always prepends merge/sort/shuffle. Skip those while variants are showing:

```go
func (v *viewer) applyTitle() {
	title := v.baseTitle
	if v.gridTitle != "" {
		title = v.gridTitle
	}
	hidePrefixes := v.grid != nil && v.grid.BrowsingDuplicates() && v.gridTitle != ""
	if !hidePrefixes {
		if v.state.MergeMode() {
			title = lang.L("[merge]") + " " + title
		}
		if v.slides.Shuffle() {
			title = lang.L("[shuffle]") + " " + title
		}
		if p := filesort.Label(v.state.SortMode()); p != "" {
			title = p + " " + title
		}
	}
	v.win.SetTitle(title)
}
```

Update the `applyTitle` comment to say Show-variants replaces the base title **and** hides mode prefixes, so the bar is only position, size, and path.

Replace the title body of `HighlightChanged` with a helper. Keep the Actions-menu refresh that follows unchanged.

```go
func (v *viewer) HighlightChanged(i int) {
	if i < 0 || i >= len(v.state.files) {
		v.gridTitle = ""
		v.applyTitle()
	} else {
		v.gridTitle = v.gridHighlightTitle(i)
		v.applyTitle()
	}

	if v.actionsHideItem == nil {
		return
	}
	hideChecked := v.actionsHideItem.Checked
	hideDisabled := v.actionsHideItem.Disabled
	variantChecked := v.actionsShowVariantItem.Checked
	variantDisabled := v.actionsShowVariantItem.Disabled
	v.applyActionsMenuState()
	if v.actionsHideItem.Checked != hideChecked ||
		v.actionsHideItem.Disabled != hideDisabled ||
		v.actionsShowVariantItem.Checked != variantChecked ||
		v.actionsShowVariantItem.Disabled != variantDisabled {
		v.refreshMainMenu()
	}
}

// gridHighlightTitle names the file under the grid ring. Hide-duplicates
// and the unfiltered grid keep the basename and a trailing position
// counter. Show-variants compares copies of one shot, so the title is
// `(index/count) [WxH] /absolute/path` from the already-probed native
// size — not a new decode. applyTitle strips mode prefixes for this form.
func (v *viewer) gridHighlightTitle(i int) string {
	u := v.state.files[i]
	if v.grid.BrowsingDuplicates() {
		head := ""
		if n := len(v.state.files); n > 1 {
			head = fmt.Sprintf("(%d/%d) ", i+1, n)
		}
		if w, h, ok := v.grid.NativeSize(i); ok {
			return fmt.Sprintf("%s[%dx%d] %s", head, w, h, u.Path())
		}
		return head + u.Path()
	}
	title := u.Name()
	if n := len(v.state.files); n > 1 {
		title = fmt.Sprintf("%s  (%d/%d)", title, i+1, n)
	}
	return title
}
```

Replace the `HighlightChanged` doc comment so it no longer says dimensions are always absent:

```go
// HighlightChanged names the grid overview's highlighted file in the window
// title (internal/ui/grid's Host): with the image view hidden behind the
// overlay, the title is the only place a thumbnail's file name is spelled
// out in full. i is -1 when nothing is highlighted - the grid closing, or a
// search matching no file - which hands the title back to the image view.
// The hide-duplicates and unfiltered grids still omit pixel size: that
// would cost a full decode of a file nobody has picked yet. Show-variants
// reuses the already-probed native size and the full path instead, and
// applyTitle omits [merge]/[shuffle]/sort prefixes for that title.
// Show-variants enablement follows the highlighted host while the grid is
// open, so this also reapplies Actions; native Refresh only if Hide or
// Show-variants Checked/Disabled actually changed.
```

`finishBrowse` already calls `setHighlight`, which notifies even when the index is unchanged, so a late size backfill refreshes the title.

- [ ] **Step 4: Run title tests**

```bash
go test -run 'TestGridHighlight_' -count=1 ./internal/ui/
```

Expected: PASS, including `TestGridHighlight_TitleKeepsTheModePrefixes` (normal grid still has `[merge]`).

- [ ] **Step 5: Stop for parent review**

---

### Task 4: Hover (and arrows) update the variants title; closing restores it

**Files:**
- Modify: `internal/ui/grid/nav.go` (add `SimulateHover`)
- Test: `internal/ui/grid_test.go`

**Interfaces:**
- Consumes: `Overview.SimulateHover(id int)` → existing `wrap.OnHighlighted`; `v.win.Title()`; `SetBrowsingDuplicates` / `Toggle`
- Produces: viewer-level proof that moving the ring in variants changes `(n/m) [WxH] path`, hides `[merge]`, and that closing the grid drops that format (prefixes return on the image-view title)

- [ ] **Step 1: Write the failing tests**

Add to `internal/ui/grid/nav.go` (next to `setHighlight`) so package `ui` can drive the same callback GridWrap uses on pointer enter. This is not a new hover path; it is the test/production seam for `OnHighlighted`:

```go
// SimulateHover moves the ring as GridWrap does when the pointer enters
// the cell at display index id.
func (g *Overview) SimulateHover(id int) {
	if g.wrap != nil && g.wrap.OnHighlighted != nil {
		g.wrap.OnHighlighted(id)
	}
}
```

Add to `internal/ui/grid_test.go`:

```go
func TestGridHighlight_VariantsHoverUpdatesTitleAndHidesMergePrefix(t *testing.T) {
	v := newTestViewer(t)
	small := uitest.PatternedJPEGURISize(t, "a.jpg", 1, 64, 48)
	large := uitest.PatternedJPEGURISize(t, "b.jpg", 1, 192, 144)
	unique := uitest.PatternedJPEGURI(t, "c.jpg", 99)
	dropAndWait(t, v, small, large, unique)
	if err := v.grid.Warm(); err != nil {
		t.Fatalf("Warm: %v", err)
	}

	v.SetMergeMode(true)
	v.grid.SetHideDuplicates(true)
	v.grid.Settle()
	v.grid.Toggle()
	v.grid.SetBrowsingDuplicates(true)
	v.grid.Settle()
	if !v.grid.Visible() || !v.grid.BrowsingDuplicates() {
		t.Fatal("premises: variants grid up")
	}

	startID := v.grid.Highlight()
	otherID := 1 - startID

	before := v.win.Title()
	if strings.HasPrefix(before, "[merge]") {
		t.Fatalf("variants title = %q, must not include [merge]", before)
	}

	v.grid.SimulateHover(otherID)
	after := v.win.Title()
	if after == before {
		t.Fatalf("title did not change after hovering the other variant, still %q", after)
	}
	if strings.HasPrefix(after, "[merge]") {
		t.Errorf("title = %q, must not include [merge]", after)
	}

	wantSmall := fmt.Sprintf("(1/3) [64x48] %s", small.Path())
	wantLarge := fmt.Sprintf("(2/3) [192x144] %s", large.Path())
	if after != wantSmall && after != wantLarge {
		t.Errorf("after hover, title = %q, want %q or %q", after, wantSmall, wantLarge)
	}
	if strings.Contains(before, small.Path()) && !strings.Contains(after, large.Path()) {
		t.Errorf("hovered off small, title = %q, want large path", after)
	}
	if strings.Contains(before, large.Path()) && !strings.Contains(after, small.Path()) {
		t.Errorf("hovered off large, title = %q, want small path", after)
	}

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
	arrow := v.win.Title()
	if strings.HasPrefix(arrow, "[merge]") {
		t.Errorf("after Right, title = %q, must not include [merge]", arrow)
	}
	if arrow != wantSmall && arrow != wantLarge {
		t.Errorf("after Right, title = %q, want %q or %q", arrow, wantSmall, wantLarge)
	}

	v.grid.Toggle()
	if v.grid.Visible() {
		t.Fatal("grid should be closed")
	}
	closed := v.win.Title()
	if strings.Contains(closed, "[64x48]") || strings.Contains(closed, "[192x144]") {
		t.Errorf("closed-grid title = %q, want image-view title, not variants format", closed)
	}
	if !strings.HasPrefix(closed, "[merge] ") {
		t.Errorf("closed-grid title = %q, want [merge] restored on the image-view title", closed)
	}
}
```

`Overview` already has exported `Highlight()`. Display index 0 is the lower host index of the group (small at 64×48), index 1 is large, because `applyVisibleFilter` walks host order. `startID := v.grid.Highlight(); otherID := 1 - startID` works for a 2-cell group.

- [ ] **Step 2: Run the test**

```bash
go test -run TestGridHighlight_VariantsHoverUpdatesTitleAndHidesMergePrefix -count=1 ./internal/ui/
```

Expected: FAIL compile (`SimulateHover` undefined) until Step 1’s method is added; then PASS if Task 3 already formats on `HighlightChanged`. If the title does not change, `OnHighlighted` is a no-op because `id == g.highlight` — then `otherID` was wrong; fix the test, not production.

- [ ] **Step 3: Implement only if the title does not follow the ring**

If `SimulateHover` moves `Highlight()` but not `win.Title()`, `setHighlight` is not notifying while browse-filtered. Fix `setHighlight` in `nav.go` so the `g.visible` notify still runs. Do not format titles inside `grid`.

- [ ] **Step 4: Re-run**

```bash
go test -run 'TestGridHighlight_|TestApplyDupBadge_HiddenWhileBrowsing|TestNativeSize_' -count=1 ./internal/ui/ ./internal/ui/grid/
```

Expected: PASS.

- [ ] **Step 5: Stop for parent review**

---

### Task 5: Manual (EN + DE)

**Files:**
- Modify: `internal/ui/help/manual.md` (Grid overview `Shift+D` bullets; optionally “The window title”)
- Modify: `internal/ui/help/manual_de.md` (same)

No new `lang.L` keys. `main_test.go` locale parity does not cover the manuals.

- [ ] **Step 1: English**

In `internal/ui/help/manual.md`, at the `Shift+D` / variants bullets (~391–405), add:

- While that variants grid is showing, the duplicate-count badges are hidden (every cell is already a member of the same group).
- The window title names the highlighted thumbnail as `(position) [widthxheight] full-path`, for example `(2/7) [1440x780] /photos/vacation/IMG_0123.jpg`. `[merge]`, sort-order, and `[shuffle]` prefixes are hidden while variants are showing. Arrow keys and hovering the pointer over a thumbnail both move the highlight, so both update the title. Leaving variants restores the usual file-name title and those prefixes.

Keep the existing commit/inspect/`Esc` loop text.

Optional one-liner under “The window title” (~103) that the grid normally shows the file name, and Show variants uses `(n/m) [WxH] /path` with no mode prefixes.

- [ ] **Step 2: German**

Mirror in `internal/ui/help/manual_de.md` at the matching `Shift+D` bullets (~438–457):

- In der Variantenansicht sind die Zähl-Badges ausgeblendet.
- Die Fenstertitelzeile zeigt die hervorgehobene Miniaturansicht als `(Position) [BreitexHöhe] vollständiger-Pfad`, z. B. `(2/7) [1440x780] /photos/vacation/IMG_0123.jpg`. `[merge]`, Sortier- und `[shuffle]`-Präfixe sind in der Variantenansicht ausgeblendet. Pfeiltasten und Zeigen mit der Maus bewegen die Hervorhebung und damit den Titel. Verlassen der Variantenansicht stellt den Dateinamen-Titel und jene Präfixe wieder her.

- [ ] **Step 3: No code**

Do not regenerate goldens. Do not touch `translations/*.json`.

- [ ] **Step 4: Parent review**

EN and DE must describe the same behavior as Tasks 1–4.

---

## Verification (parent, after Task 5)

From the repository root:

```bash
gofmt -l .
go vet ./...
go build ./...
go test -run 'TestApplyDupBadge|TestNativeSize_|TestGridHighlight_|TestSetBrowsingDuplicates|TestWarm_RecordsNative' -count=1 ./internal/ui/ ./internal/ui/grid/
go test -race -timeout 20m ./...
```

Expected: no `gofmt` output, vet/build clean, focused tests PASS, full race suite PASS.

No golden screenshots: this is title strings and chip visibility, not pixels.

## Out of scope

- Changing image-view title format (`name — W x H`).
- Showing WxH on the hide-duplicates grid (deliberately not decoded today).
- Hiding badges anywhere except `BrowsingDuplicates()`.
- New Host methods, package-level seams, or a hover-only title channel (Q2 is locked: highlight drives the title).
- Windows Ctrl+click grid selection (tracked as not-worth-it / Fyne).
