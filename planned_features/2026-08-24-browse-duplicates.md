# Browse Duplicates (Shift+D) Implementation Plan

> **Controller:** After every task, review the diff, fix issues, then start the next task. Do **not** `git commit` (`AGENTS.md`). Do **not** start the next task until the current one is accepted.
>
> Task subagents: read only your task, Global Constraints, and that task's Interfaces. Do not read this whole file. **If an identifier in this plan disagrees with a file you open, the file wins.**

**Goal:** `Shift+D` on a duplicated shot opens (or filters) the grid to **only that shot's duplicate group**, so the user can compare the copies side by side.

**Architecture:** Reuse the existing dHash groups on `Overview`. Add a grid-local browse flag (`browseHost int`, `-1` = off) as a third `applyFilter` predicate. While browsing, **do not** apply hide-extras (the whole point is to *see* extras). `Host` gains **one** method: `ShowToast(string)` (same shape as `deletion` / `favorites` / `exifwin`). The grid reads Shift through existing `Modifiers()`, the viewer reads it through existing `keyModifiers()`. `viewer` already implements `ShowToast`.

**Tech:** Go 1.26.7, Fyne v2.8, `internal/ui/grid`, `internal/ui/keys.go`. No new packages, no new hash work.

**Source todo:** `todos.md` — “browse duplicates”.

## Recommended defaults (locked unless Florian overrides)

These are the defaults the tasks implement. Open questions at the bottom of the user message can change them **before** Task 1 starts.

| # | Decision |
|---|----------|
| 1 | **Grid `Shift+D`:** highlighted cell's group. **Viewer `Shift+D`:** current image's group; **opens the grid** if it was closed. |
| 2 | Hash remaining first (same as `D`), then re-check group size. If hashing **jobs are still running** (`hashRemaining` returned `pending > 0`): info toast `lang.L("The images are currently being analyzed")` (DE `"Die Bilder werden gerade analysiert"`), keep `browseHost` set, and `finishBrowse` when the last job lands. If hashes are already ready and group size **&lt; 2**: **silent no-op** (no toast). Do **not** toast when hide-duplicates (`D`) hashes remaining. |
| 3 | **Escape:** selection → search → **browse off** → hide-dupes off → close. |
| 4 | **`G` / `Close` clear browse** (like search). Hide-dupes stays. Reopening `G` is the hide/full grid, not the last group. |
| 5 | **`/` while browsing** intersects (name match AND in group). **`Shift+D` while searching** is ignored (letter `D` in the query). |
| 6 | **`D` while browsing** still toggles hide, but browse **overrides** the hide predicate so extras of *this* group stay visible. Plain `D` does **not** exit browse. |
| 7 | **`Shift+D` while already browsing** turns browse **off** (toggle). Only one group is on screen, so it cannot switch groups. |
| 8 | **Picture-frame (`slides.Active()`):** ignore viewer `Shift+D`, same as `G`. |
| 9 | **Top bar** while browsing and not searching: `lang.L("Showing duplicates")` plus the existing `"%d of %d"`. Search chrome still wins if `/` is open. (Parallel to `"Hiding duplicates"`.) |
| 10 | **Badges** stay hide-only (`hideDupes && n >= 2`). No badge change in browse mode. |
| 11 | After the filter applies, **highlight the source file** (the one `Shift+D` was pressed on), not display index 0. |
| 12 | Distance slider while browsing: regroup; if the source's group drops below 2, **exit browse**. |

## Global Constraints

- Do not `git commit`. Suggested commit messages are for the user.
- Every user-visible string is `lang.L("...")` with the same key in `translations/en.json` **and** `translations/de.json`. Guard: `TestTranslations_EveryLocaleCoversEnglish` in `main_test.go`.
- Feature packages talk through their own `Host`. Do not import `internal/ui` from `grid`. Do not pass `appState`. **Grid `Host` gains only `ShowToast(msg string)`** — required so Shift+D can raise the analyzing toast from both the grid and the viewer. Do not add any other Host methods.
- Cross-feature decisions stay in `internal/ui`. Grid-local `D` / `Shift+D` / Escape staging stay in `grid.HandleKey`. Viewer `Shift+D` lives in `keys.go` when the grid is **not** visible (keys already forward to `HandleKey` when it is).
- `browseHost` **must** be initialized to **`-1`** in `New`. Go's zero value is `0`, which would browse file 0 on every new overview.
- Uniform-color JPEGs all dHash to `0`. Grid tests must use **patterned** JPEGs (`pairAndUnique` / `openPatterned` / `uitest.PatternedJPEGURI`).
- Tests: TDD. No `time.Sleep`. Use `Warm` / `Settle` / existing harness. Fyne test driver runs `fyne.Do` inline.
- Open work in `todos.md`; no `TODO`/`FIXME` in source. Do not mark the todo done until Florian accepts.
- English comments; match surrounding style. Verify identifiers against the files you open.
- Extra background work goes through `g.decodes.Go` so `Settle` waits. `hashRemaining` completion must `applyFilter` when **either** hide **or** browse is on.

## Subagent assignment

Run **strictly in order**. Do not dispatch implementers in parallel. Do not commit.

| Task | What | Type | Model |
|------|------|------|-------|
| 1 | Browse filter + API + top bar + translations | `go-expert` | `cursor-grok-4.6-xhigh` |
| 2 | Grid `Shift+D`, Escape staging, Close/G, hashRemaining, distance | `go-expert` | `cursor-grok-4.6-xhigh` |
| 3 | Viewer `Shift+D` opens grid | `go-expert` | `cursor-grok-4.6-xhigh` |
| 4 | Manual EN/DE + `ARCHITECTURE.md` | `generalPurpose` | `composer-2.5-fast` |
| Per-task review | Spec + quality | `generalPurpose` | `claude-sonnet-5-thinking-high` |
| Parent fix-up | After each task | controller (this session) | inherit |
| Final review | Whole change | `generalPurpose` | `claude-opus-5-thinking-high` |

## File map

| File | Role |
|------|------|
| `internal/ui/grid/grid.go` | `browseHost int` field (`-1` off); `New` initializes it; `Host.ShowToast` |
| `internal/ui/grid/harness_test.go` | `fakeHost.ShowToast` records messages |
| `internal/ui/grid/dupes.go` | `BrowsingDuplicates`, `SetBrowsingDuplicates`, `ToggleBrowseDuplicates`, `finishBrowse`, `groupMembers`; toast when `hashRemaining` is pending |
| `internal/ui/grid/dupes_test.go` | Filter / API / toast tests |
| `internal/ui/grid/search.go` | `applyFilter` third predicate; `syncTopBar` chrome |
| `internal/ui/grid/nav.go` | `HandleKey` Shift+D; Escape staging |
| `internal/ui/grid/grid.go` `Close` | Clear browse (not hide) |
| `internal/ui/keys.go` | Viewer `Shift+D` when grid closed |
| `internal/ui/step_test.go` | Viewer key tests (next to existing `D` tests) |
| `translations/en.json`, `de.json` | `"Showing duplicates"`, `"The images are currently being analyzed"` |
| `internal/ui/help/manual.md`, `manual_de.md` | Docs |
| `ARCHITECTURE.md` | Browse-duplicates sentence |

### Identifier lock (verify in the files you open)

As of this plan, these names exist. If the file differs, use the file.

```go
// Host (internal/ui/grid/grid.go + harness_test.go)
FileCount() int
FileAt(i int) fyne.URI
CurrentIndex() int
Generation() uint64
ShowImage(i int)
HighlightChanged(i int)
ForceRepaint()
Unfocus()
Modifiers() fyne.KeyModifier
ShowToast(msg string) // ADD — viewer already has this; fakeHost records toasts []string

// Overview
HideDuplicates() bool
SetHideDuplicates(on bool)
IsHiddenExtra(hostIndex int) bool
RepresentativeOf(hostIndex int) int
groupSize(hostIndex int) int
rebuildGroups()
hashRemaining() int
hashOf(u fyne.URI) (uint64, bool)
applyFilter()
syncTopBar()
HandleKey(ev *fyne.KeyEvent)
HandleRune(r rune)
Toggle()
Close()
Visible() bool
Warm() error
Settle()
fileIndex(display int) int
displayIndexOf(hostIdx int) int
count() int
setHighlight(id int)
escape()
clearSearch()
hideDupes bool
matches []int
searching bool
query string
highlight int
visible bool
sel *selection.Set
wrap *widget.GridWrap
searchLabel, countLabel *widget.Label
searchBar *fyne.Container

// Test helpers
pairAndUnique(t) (*Overview, *fakeHost)
openPatterned / hostPatterned / hostWith / newOverview
fakeHost.mods fyne.KeyModifier; fakeHost.index; fakeHost.files; fakeHost.toasts
loadPatternedTriple(t) *viewer
stubKeyModifiers(t, v, mods)
waitUntilLoaded(t, v)
HandleRune(r rune)
Highlight() int / setHighlight(id int)
ClearSelection()
Query() string
Warm() error / Settle()
rememberHash / SetDuplicateDistance
searchLabel / countLabel
lang.L("Hiding duplicates") / lang.L("Search: %s") / lang.L("%d of %d") / lang.L("The images are currently being analyzed")
```

---

### Task 1: Browse filter, API, top bar, translations

**Subagent:** `go-expert` @ `cursor-grok-4.6-xhigh`

**Files:** Modify `grid.go` (field + `New` + `Host.ShowToast`), `harness_test.go` (`fakeHost.ShowToast` / `toasts []string`), `dupes.go`, `dupes_test.go`, `search.go` (`applyFilter`, `syncTopBar`), `translations/en.json`, `translations/de.json`. **Do not** wire keys, Escape, Close, or `keys.go`.

**Interfaces:**

```go
// browseHost is the host index Shift+D was pressed on, or -1 when browse is
// off. Zero is a valid file index — New MUST set this to -1.
browseHost int

func (g *Overview) BrowsingDuplicates() bool
func (g *Overview) SetBrowsingDuplicates(on bool)
func (g *Overview) ToggleBrowseDuplicates()
func (g *Overview) finishBrowse() // unexported; called when hashes are ready
func (g *Overview) groupMembers(hostIndex int) []int // host indices in the same group, size >= 2; nil otherwise
```

**Filter rule** (`applyFilter`):

```
browse := g.browseHost >= 0
nameFilter := g.searching && g.query != ""
hide := g.hideDupes && !browse   // browse overrides hide so extras of this group show
if nameFilter || hide || browse {
    for i := range FileCount {
        if nameFilter && name does not contain query { continue }
        if browse && RepresentativeOf(i) != RepresentativeOf(g.browseHost) { continue }
        if hide && IsHiddenExtra(i) { continue }
        append i
    }
}
```

**`SetBrowsingDuplicates(true)`:**

1. `src := g.host.CurrentIndex()`; if `g.visible` then `src = g.fileIndex(g.highlight)`.
2. If `src < 0`, return.
3. `pending := g.hashRemaining()`.
4. If `pending > 0`, call `g.host.ShowToast(lang.L("The images are currently being analyzed"))`. Do this **only** on the browse path, never from `SetHideDuplicates`.
5. Set `g.browseHost = src` **before** pending work so completion can finish the browse.
6. If `pending == 0`, call `finishBrowse()`.

Do **not** toast when `pending == 0` (hashes already ready), whether the source is unique or a pair. Do **not** toast again when hashing finishes — `finishBrowse` either shows the group or silently leaves browse off.

**`finishBrowse`:** if `browseHost < 0` return; if `groupSize(browseHost) < 2` set `browseHost = -1` and `applyFilter`; else `applyFilter`, then if visible `setHighlight(displayIndexOf(browseHost))` and `wrap.ScrollTo` that index.

**`SetBrowsingDuplicates(false)`:** if already off, return; `browseHost = -1`; `applyFilter`.

**`ToggleBrowseDuplicates`:** if `BrowsingDuplicates()` then off, else on.

**`hashRemaining` completion** (in this task, so filter tests with un-Warmed files can work if you add one; Task 2 will add dedicated tests): when the last job runs `fyne.Do`, apply if `(g.hideDupes || g.browseHost >= 0) && gen == g.host.Generation()`. If browsing, call `finishBrowse()` instead of only `applyFilter`+`jumpIfHiddenExtra`. If hide is also on, still `jumpIfHiddenExtra` **only when not browsing** (browse shows extras).

**Top bar** (`syncTopBar`): after the `searching` case, before `hideDupes`:

```go
case g.browseHost >= 0:
    g.searchLabel.SetText(lang.L("Showing duplicates"))
    g.countLabel.SetText(fmt.Sprintf(lang.L("%d of %d"), g.count(), g.host.FileCount()))
    g.searchLabel.Show()
    g.countLabel.Show()
```

Show the bar when browsing even with no selection/search/hide. Empty notice stays search-only.

**Translations:**

- EN: `"Showing duplicates"` → `"Showing duplicates"`
- DE: `"Showing duplicates"` → `"Duplikate anzeigen"`
- EN: `"The images are currently being analyzed"` → `"The images are currently being analyzed"`
- DE: `"The images are currently being analyzed"` → `"Die Bilder werden gerade analysiert"`

- [ ] **Step 1: Write failing tests** in `dupes_test.go` (use `pairAndUnique`; it already `Warm`s and `Toggle`s):

```go
func TestSetBrowsingDuplicates_ShowsOnlyTheGroup(t *testing.T) {
	g, host := pairAndUnique(t) // host 0,1 pair; host 2 unique

	g.SetBrowsingDuplicates(true)

	if !g.BrowsingDuplicates() {
		t.Fatal("BrowsingDuplicates() = false after SetBrowsingDuplicates(true)")
	}
	if g.count() != 2 {
		t.Fatalf("count() = %d, want 2 (pair only)", g.count())
	}
	if g.fileIndex(0) != 0 || g.fileIndex(1) != 1 {
		t.Fatalf("visible = [%d, %d], want [0, 1]", g.fileIndex(0), g.fileIndex(1))
	}
	if g.Highlight() != 0 {
		t.Errorf("Highlight() = %d, want 0 (source file)", g.Highlight())
	}
	if len(host.toasts) != 0 {
		t.Errorf("toasts = %v, want none (already hashed)", host.toasts)
	}
}

func TestSetBrowsingDuplicates_NoopOnUnique(t *testing.T) {
	g, host := pairAndUnique(t)
	g.setHighlight(2) // moon.jpg

	g.SetBrowsingDuplicates(true)

	if g.BrowsingDuplicates() {
		t.Fatal("unique file must not enter browse")
	}
	if g.count() != 3 {
		t.Fatalf("count() = %d, want 3 (unfiltered)", g.count())
	}
	if len(host.toasts) != 0 {
		t.Errorf("toasts = %v, want none (hashes already ready)", host.toasts)
	}
}

func TestSetBrowsingDuplicates_ShowsExtrasEvenWhenHideOn(t *testing.T) {
	g, _ := pairAndUnique(t)
	g.SetHideDuplicates(true)
	if g.count() != 2 {
		t.Fatalf("setup hide count() = %d, want 2", g.count())
	}
	// Display 0 is host 0 (representative). Extra host 1 is hidden.
	g.setHighlight(0)

	g.SetBrowsingDuplicates(true)

	if g.count() != 2 {
		t.Fatalf("count() = %d, want 2 (both pair members, unique excluded)", g.count())
	}
	seen := map[int]bool{g.fileIndex(0): true, g.fileIndex(1): true}
	if !seen[0] || !seen[1] {
		t.Fatalf("visible hosts = %v, want 0 and 1 (extra must be shown)", seen)
	}
	if g.HideDuplicates() != true {
		t.Fatal("browse must not clear the hide flag")
	}
}

func TestSetBrowsingDuplicates_IntersectsSearch(t *testing.T) {
	g, _ := pairAndUnique(t)
	g.HandleRune('/')
	for _, r := range "sunset" {
		g.HandleRune(r)
	}
	g.SetBrowsingDuplicates(true)
	if g.count() != 2 {
		t.Fatalf("count() = %d, want 2 (sunset pair)", g.count())
	}
	g.clearSearch()
	if !g.BrowsingDuplicates() || g.count() != 2 {
		t.Fatal("clearing search must leave browse on")
	}
}

func TestSyncTopBar_ShowingDuplicates(t *testing.T) {
	g, _ := pairAndUnique(t)
	g.SetBrowsingDuplicates(true)
	if got, want := g.searchLabel.Text, lang.L("Showing duplicates"); got != want {
		t.Errorf("searchLabel = %q, want %q", got, want)
	}
	if got, want := g.countLabel.Text, fmt.Sprintf(lang.L("%d of %d"), 2, 3); got != want {
		t.Errorf("countLabel = %q, want %q", got, want)
	}
}

func TestGroupMembers_Pair(t *testing.T) {
	g, _ := pairAndUnique(t)
	got := g.groupMembers(1)
	if len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Fatalf("groupMembers(1) = %v, want [0 1]", got)
	}
	if g.groupMembers(2) != nil {
		t.Fatalf("groupMembers(unique) = %v, want nil", g.groupMembers(2))
	}
}
```

Assert the top-bar the same way existing hide tests assert `"Hiding duplicates"` if those tests exist; otherwise check `searchLabel.Visible()`.

- [ ] **Step 2: Run tests — expect FAIL**

```bash
go test -count=1 ./internal/ui/grid/ -run 'TestSetBrowsingDuplicates_|TestSyncTopBar_ShowingDuplicates|TestGroupMembers_' -v
```

- [ ] **Step 3: Implement**

In `New`, set `browseHost: -1`.

In `dupes.go`, add the methods above. `groupMembers`: if `groupSize(hostIndex) < 2` return nil; else collect every `i` with `RepresentativeOf(i) == RepresentativeOf(hostIndex)` in host-index order.

Wire `applyFilter` and `syncTopBar` as specified. Update `hashRemaining`'s last-job `fyne.Do` as specified.

- [ ] **Step 4: Run tests — expect PASS**

```bash
go test -count=1 ./internal/ui/grid/ -run 'TestSetBrowsingDuplicates_|TestSyncTopBar_ShowingDuplicates|TestGroupMembers_|TestSetHideDuplicates_|TestHandleKey_D|TestHandleKey_Escape' -v
go test -count=1 ./ -run TestTranslations_EveryLocaleCoversEnglish
```

Existing Escape/hide tests must still pass (browse is off).

- [ ] **Step 5: Suggested commit (do not commit):** `grid: add browse-duplicates filter for one group's members`

---

### Task 2: Grid Shift+D, Escape, Close, hashRemaining, distance

**Subagent:** `go-expert` @ `cursor-grok-4.6-xhigh`

**Files:** `nav.go` (`HandleKey`, `escape`), `grid.go` (`Close`), `dupes.go` (`SetDuplicateDistance` live browse), `dupes_test.go`.

**Interfaces:** Consumes Task 1 API. Produces: `HandleKey` `KeyD` with Shift via `g.host.Modifiers()&fyne.KeyModifierShift`; Escape staging; `Close`/`G` clear browse.

**`HandleKey` `KeyD`** (not searching):

```go
case fyne.KeyD:
    if g.host.Modifiers()&fyne.KeyModifierShift != 0 {
        g.ToggleBrowseDuplicates()
    } else {
        g.SetHideDuplicates(!g.hideDupes)
    }
```

Search branch: do **not** handle `KeyD` (existing: `d`/`D` is a query rune).

**`escape`:**

```go
switch {
case g.sel.Len() > 0:
    g.ClearSelection()
case g.searching:
    g.clearSearch()
case g.browseHost >= 0:
    g.SetBrowsingDuplicates(false)
case g.hideDupes:
    g.SetHideDuplicates(false)
default:
    g.Close()
}
```

**`Close`:** after `clearSearch()`, if `browseHost >= 0` set it to `-1` and `applyFilter()` (or call `SetBrowsingDuplicates(false)` **before** hiding — but `SetBrowsingDuplicates(false)` calls `applyFilter`, which is OK while still `visible`). Must **not** clear `hideDupes`. `G` already calls `Close`.

**`SetDuplicateDistance`:** after regroup, if `browseHost >= 0` call `finishBrowse()` (exits if group &lt; 2). If hide is on and not browsing, keep existing `applyFilter` + `jumpIfHiddenExtra`.

- [ ] **Step 1: Write failing tests**

```go
func TestHandleKey_ShiftDTogglesBrowseDuplicates(t *testing.T) {
	g, host := pairAndUnique(t)
	host.mods = fyne.KeyModifierShift

	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyD})
	if !g.BrowsingDuplicates() || g.count() != 2 {
		t.Fatalf("after Shift+D: browse=%v count=%d, want browse=true count=2", g.BrowsingDuplicates(), g.count())
	}

	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyD})
	if g.BrowsingDuplicates() || g.count() != 3 {
		t.Fatalf("second Shift+D: browse=%v count=%d, want browse=false count=3", g.BrowsingDuplicates(), g.count())
	}
}

func TestHandleKey_PlainDStillTogglesHideWhileNotSearching(t *testing.T) {
	g, host := pairAndUnique(t)
	host.mods = 0
	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyD})
	if !g.HideDuplicates() || g.BrowsingDuplicates() {
		t.Fatal("plain D must toggle hide, not browse")
	}
}

func TestHandleKey_ShiftDWhileSearchingDoesNotBrowse(t *testing.T) {
	g, host := pairAndUnique(t)
	g.HandleRune('/')
	g.HandleRune('x')
	host.mods = fyne.KeyModifierShift
	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyD})
	if g.BrowsingDuplicates() {
		t.Fatal("Shift+D while searching must not browse")
	}
	if g.Query() != "x" {
		t.Errorf("Query() = %q, want %q (KeyD is not a typed rune)", g.Query(), "x")
	}
}

func TestHandleKey_EscapeTurnsOffBrowseBeforeHide(t *testing.T) {
	g, host := pairAndUnique(t)
	g.SetHideDuplicates(true)
	g.SetBrowsingDuplicates(true)
	host.mods = 0

	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if g.BrowsingDuplicates() {
		t.Fatal("first Escape should leave browse")
	}
	if !g.HideDuplicates() || !g.Visible() {
		t.Fatal("first Escape should leave hide on and the grid up")
	}

	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if g.HideDuplicates() {
		t.Fatal("second Escape should turn hide off")
	}
	if !g.Visible() {
		t.Fatal("second Escape should not close the grid")
	}
}

func TestClose_ClearsBrowseLeavesHide(t *testing.T) {
	g, _ := pairAndUnique(t)
	g.SetHideDuplicates(true)
	g.SetBrowsingDuplicates(true)

	g.Close()

	if g.Visible() {
		t.Fatal("Close should hide the grid")
	}
	if g.BrowsingDuplicates() {
		t.Error("Close must clear browse")
	}
	if !g.HideDuplicates() {
		t.Error("Close must not clear hide-duplicates")
	}
}

func TestHandleKey_GClearsBrowse(t *testing.T) {
	g, host := pairAndUnique(t)
	g.SetBrowsingDuplicates(true)
	host.mods = 0
	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyG})
	if g.Visible() {
		t.Fatal("G should close")
	}
	if g.BrowsingDuplicates() {
		t.Error("G/Close must clear browse")
	}
}

func TestSetBrowsingDuplicates_HashesRemainingWithoutWarm(t *testing.T) {
	host := hostPatterned(t,
		[]string{"sunset-a.jpg", "sunset-b.jpg", "moon.jpg"},
		[]int{1, 1, 99},
	)
	g := newOverview(t, host)
	g.Toggle()
	host.index = 0

	g.SetBrowsingDuplicates(true)
	if len(host.toasts) != 1 || host.toasts[0] != lang.L("The images are currently being analyzed") {
		t.Fatalf("toasts = %v, want [%q] while hashing", host.toasts, lang.L("The images are currently being analyzed"))
	}
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

func TestSetDuplicateDistance_ExitsBrowseWhenGroupSplits(t *testing.T) {
	host := hostWith(t, "a.jpg", "b.jpg")
	g := newOverview(t, host)
	a, b := nearGrayPair()
	g.rememberHash(host.files[0], a)
	g.rememberHash(host.files[1], b)
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

Copy helper usage from `TestSetHideDuplicates_HashesRemainingWithoutWarm` and `TestSetDuplicateDistance_RegroupsLive` in `dupes_test.go`.

- [ ] **Step 2: Run — expect FAIL**

```bash
go test -count=1 ./internal/ui/grid/ -run 'TestHandleKey_ShiftD|TestHandleKey_PlainD|TestHandleKey_ShiftDWhileSearching|TestHandleKey_EscapeTurnsOffBrowse|TestClose_ClearsBrowse|TestHandleKey_GClearsBrowse|TestSetBrowsingDuplicates_HashesRemaining|TestSetDuplicateDistance_ExitsBrowse' -v
```

- [ ] **Step 3: Implement** the HandleKey / escape / Close / distance / hashRemaining pieces.

- [ ] **Step 4: Run — expect PASS**, including existing Escape hide tests (they now have one extra Escape stage only when browse is on):

```bash
go test -count=1 ./internal/ui/grid/ -v
```

- [ ] **Step 5: Suggested commit (do not commit):** `grid: Shift+D browses one duplicate group`

---

### Task 3: Viewer Shift+D opens the grid

**Subagent:** `go-expert` @ `cursor-grok-4.6-xhigh`

**Files:** `internal/ui/keys.go` (`KeyD` case), `internal/ui/step_test.go` (next to `TestHandleKeyEvent_DTogglesHideDuplicatesWhenGridClosed`).

**Interfaces:** Consumes `ToggleBrowseDuplicates`, `BrowsingDuplicates`, `Visible`, `Toggle`. When the grid **is** visible, `handleKeyEvent` already returns after `HandleKey` — do **not** duplicate Shift+D there.

**`keys.go` `KeyD`:**

```go
case fyne.KeyD:
    if v.keyModifiers()&fyne.KeyModifierShift != 0 {
        if !v.slides.Active() {
            v.grid.ToggleBrowseDuplicates()
            if v.grid.BrowsingDuplicates() && !v.grid.Visible() {
                v.grid.Toggle()
            }
        }
        return
    }
    v.grid.SetHideDuplicates(!v.grid.HideDuplicates())
    return
```

If browse is a no-op (unique), do **not** open the grid.

- [ ] **Step 1: Write failing tests** in `step_test.go` using `loadPatternedTriple` + `stubKeyModifiers` from `rotate_test.go`:

```go
func TestHandleKeyEvent_ShiftDOpensGridOnCurrentGroup(t *testing.T) {
	v := loadPatternedTriple(t)
	stubKeyModifiers(t, v, fyne.KeyModifierShift)
	if v.grid.Visible() {
		t.Fatal("setup: grid closed")
	}

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyD})
	v.grid.Settle()

	if !v.grid.Visible() {
		t.Fatal("Shift+D on a duplicated file should open the grid")
	}
	if !v.grid.BrowsingDuplicates() {
		t.Fatal("grid should be in browse mode")
	}
	if v.grid.count() != 2 { // unexported — if count is unexported, assert via exported API or test from grid package only
		// Prefer: keep this assertion in grid tests; here assert Visible + BrowsingDuplicates + HideDuplicates()==false
	}
	if v.grid.HideDuplicates() {
		t.Fatal("Shift+D must not turn hide on")
	}
}

func TestHandleKeyEvent_ShiftDNoopOnUniqueDoesNotOpenGrid(t *testing.T) {
	v := loadPatternedTriple(t)
	v.ShowImage(2)
	waitUntilLoaded(t, v)
	stubKeyModifiers(t, v, fyne.KeyModifierShift)

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyD})
	v.grid.Settle()

	if v.grid.Visible() {
		t.Fatal("Shift+D on a unique file must not open the grid")
	}
	if v.grid.BrowsingDuplicates() {
		t.Fatal("must not enter browse")
	}
}

func TestHandleKeyEvent_PlainDStillHidesWhenGridClosed(t *testing.T) {
	v := loadPatternedTriple(t)
	stubKeyModifiers(t, v, 0)
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyD})
	if !v.grid.HideDuplicates() || v.grid.Visible() {
		t.Fatal("plain D with grid closed should hide extras, not open the grid")
	}
}
```

`count()` is unexported. **Do not** export it. Viewer tests assert `Visible()`, `BrowsingDuplicates()`, `HideDuplicates()` only. Group-size coverage is Task 1–2.

If `BrowsingDuplicates` is on `Overview` and `step_test.go` is package `ui`, it can call it — it is exported.

- [ ] **Step 2: Run — expect FAIL**

```bash
go test -count=1 ./internal/ui/ -run 'TestHandleKeyEvent_ShiftD|TestHandleKeyEvent_PlainDStillHides' -v
```

- [ ] **Step 3: Implement** the `KeyD` branch in `keys.go`.

- [ ] **Step 4: Run — expect PASS**

```bash
go test -count=1 ./internal/ui/ -run 'TestHandleKeyEvent_|TestStepImage_SkipsHidden' -v
```

- [ ] **Step 5: Suggested commit (do not commit):** `ui: Shift+D from the viewer opens the duplicate group in the grid`

---

### Task 4: Manual, ARCHITECTURE.md

**Subagent:** `generalPurpose` @ `composer-2.5-fast`

**Files:** `internal/ui/help/manual.md`, `manual_de.md`, `ARCHITECTURE.md`. Do **not** mark `todos.md` done.

**English (grid overview + cheatsheet), after the `D` hide paragraph:**

- `Shift+D` shows every copy of the **highlighted** shot (grid) or the **current** shot (image view). The grid lists only that group, including extras `D` would hide.
- If thumbnails are still being hashed, an info toast says **The images are currently being analyzed**; the group appears when hashing finishes. A unique shot (already hashed, no copies) does nothing.
- `Esc` leaves browse before it turns hide off. `G`/Close leave hide on but **end** browse.
- While `/` search is open, `Shift+D` is not browse (`D` is a letter).
- Picture-frame: `Shift+D` does nothing, like `G`.

**German:** same facts; `Shift+D` / `Duplikate dieser Aufnahme anzeigen` / keep existing hide wording.

**ARCHITECTURE.md** hide-duplicates FAQ: add that `Shift+D` is a grid-local group filter (`browseHost`), cleared on Close; `Host` gained `ShowToast` for the analyzing toast; viewer `Shift+D` in `keys.go` opens the grid.

- [ ] **Step 1:** Edit the three files.

- [ ] **Step 2:**

```bash
go test -count=1 ./internal/ui/help/ -run Manual
go test -count=1 ./ -run TestTranslations_EveryLocaleCoversEnglish
```

- [ ] **Step 3: Suggested commit (do not commit):** `docs: describe Shift+D browse-duplicates`

---

## Parent review checklist (after every task)

- `browseHost` default is `-1`, never left at `0` from `New`.
- Browse overrides hide in `applyFilter` but does **not** clear `hideDupes`.
- `Host` gained **only** `ShowToast`. No second decode pool. No `time.Sleep`.
- Analyzing toast fires iff `hashRemaining` returned `pending > 0` on the **browse** path; exact English key `The images are currently being analyzed`. No toast for unique-already-hashed. No toast on hide `D`. No second toast when hashing finishes.
- `hashRemaining` completion applies browse, not only hide.
- Shift is read from `Modifiers()` / `keyModifiers()`, not from `KeyEvent` (Fyne events carry none).
- Searching still swallows `KeyD`.
- Close clears browse; hide survives.
- Both translation JSON files updated (Task 1), including the analyzing toast.
- Identifiers match the files on disk.

## Suggested overall commit (user)

```
grid: Shift+D shows the current shot's duplicate group
```
