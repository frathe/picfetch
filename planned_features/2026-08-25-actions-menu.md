# Actions Menu Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Parent protocol (this session):** Do **not** start until Florian says to start the subagents. Dispatch **one implementer at a time**, in task order. After each task the **parent reads the diff, re-runs that task's verification command, and fixes drift** before dispatching a reviewer and then the next implementer. Do **not** `git commit` (`AGENTS.md`); the parent suggests one message at the very end.
>
> Task subagents: read only your task, Global Constraints, and that task's Interfaces. Do not read this whole file. **If an identifier in this plan disagrees with a file you open, the file wins.**

**Goal:** Add an **Actions** menu: sort, duplicate filters, then rotate/zoom, merge/info, and copy/trash — the same commands as `S`/`D`/`Shift+D`/`R`/`+`/`-`/`M`/`I`/`Cmd+C`/`Cmd+Shift+C`/`Shift+Delete`, with checkmarks on modes and grey-out when an item cannot run.

**Architecture:** Cross-feature composition stays in `internal/ui` (`menu.go` + new `actionmenu.go`). `internal/ui/grid` stays menu-ignorant aside from `SourceDuplicateGroupSize` / `SetOnDupeStateChanged`. Sort jumps through `SetSortMode`. Duplicate items share helpers with `D` / `Shift+D` (Show variants is **stricter** than the key: hide must be on). Rotate/zoom/merge/info/copy/trash call the existing viewer methods (`rotateBy`, `zoom.In`/`Out`, `toggleMergeMode`, `toggleInfoOverlay`, `copySelection`, `copyPathToClipboard`, `requestDelete`) so the menu cannot drift from the keys/shortcuts.

**Tech Stack:** Go 1.26.7, Fyne v2.8.0 (`MenuItem.Checked`, `MenuItem.ChildMenu`, `MenuItem.Disabled`), `lang.L` + `translations/{en,de}.json`, packages `internal/ui`, `internal/ui/grid`, `internal/filesort`.

**Source todo:** `todos.md` — **Menu Actions** (not Menu Window). Window menu already exists and is out of scope except where adding a fifth top-level menu shifts test indices.

## Recommended defaults (locked unless Florian overrides)

These are the defaults the tasks implement. Open questions in the parent message can change them **before Task 1 starts**.

| # | Decision |
|---|----------|
| 1 | **Scope is Menu Actions only.** Do **not** add EXIF, Viewer, Grid, Picture-frame, Help, Settings, or Shuffle here. EXIF stays on **Window → EXIF Data**. `Shift+R` (counter-clockwise), `0`/`1` zoom, and `Shift+P` shuffle stay **keys only**. |
| 2 | **Top-level menu label** `lang.L("Actions")` / DE `Aktionen`. Bar order: **File, Favorites, Actions, Window, Help**. Darwin Window-merge is unchanged; Actions is not merged into `NSApp.windowsMenu`. |
| 3 | **Items, in this order** (separators between groups): Sort order (submenu); Show/Hide duplicates; Show variants; **separator**; Rotate image; Zoom in; Zoom out; **separator**; Toggle merge mode; Show/Hide info overlay; **separator**; Copy image; Copy image path; Move image to Trash. |
| 4 | **Sort submenu:** parent `lang.L("Sort order")`. Children `filesort.Modes()` / `DisplayName`. **Current mode `Checked`; others enabled.** Re-choosing current is a no-op. Jump via `SetSortMode`. Parent accelerator `S` (modifier `0`). |
| 5 | **Show/Hide duplicates** checkbox: `Checked` iff `HideDuplicates()`. Action = `D`. Accelerator `D`. Disabled when `FileCount()==0`. |
| 6 | **Show variants** checkbox: `Checked` iff `BrowsingDuplicates()`. Enabled only when hide is on **and** `SourceDuplicateGroupSize()>=2`, or while already browsing. Accelerator `Shift+D`. Menu is stricter than the `Shift+D` key (key still works with hide off). |
| 7 | **Do not rebuild duplicate groups on every menu refresh.** `SourceDuplicateGroupSize` reads last `groupSizes`. `0` = unknown → Show variants stays disabled until hide has built a group of size ≥2. |
| 8 | **Native-menu refresh:** `updateActionsMenuState` and `updateWindowMenuState` each apply **both** Window and Actions state, then one `refreshMainMenu`. `HighlightChanged` refreshes only when Show/Hide duplicates or Show variants Checked/Disabled actually changed. |
| 9 | **New `lang.L` keys** (do not re-add `"Sort order"`, DisplayNames, or `"Move to Trash"`): `"Actions"`, `"Show/Hide duplicates"`, `"Show variants"`, `"Rotate image"`, `"Zoom in"`, `"Zoom out"`, `"Toggle merge mode"`, `"Show/Hide info overlay"`, `"Copy image"`, `"Copy image path"`, `"Move image to Trash"`. German: `Aktionen`, `Duplikate ein-/ausblenden`, `Varianten anzeigen`, `Bild drehen`, `Vergrößern`, `Verkleinern`, `Zusammenführen-Modus umschalten`, `Info-Overlay ein-/ausblenden`, `Bild kopieren`, `Bildpfad kopieren`, `Bild in den Papierkorb legen`. |
| 10 | Feature packages do not learn about menus. Do **not** add menu types to `grid.Host`. |
| 11 | **Rotate image** (`R`, modifier `0`): `rotateBy(1)` — clockwise 90°, view-only, same as the `R` key. Not `Shift+R`. **Disabled** when `len(displayFrames)==0` or `grid.Visible()` (keys never reach rotate while the grid owns the keyboard). Enabled in picture-frame. |
| 12 | **Zoom in** (`+`, `KeyPlus`, modifier `0`) / **Zoom out** (`-`, `KeyMinus`, modifier `0`): `v.zoom.In()` / `v.zoom.Out()`. Same Disabled rule as rotate. `KeyEqual` stays a key alias only; the menu shows `+`. |
| 13 | **Toggle merge mode** checkbox: `Checked` iff `MergeMode()`. Action `toggleMergeMode`. Accelerator `M`. **Always enabled** (same as the `M` key, including before any drop). `SetMergeMode` (Settings) must update the checkmark. |
| 14 | **Show/Hide info overlay** checkbox: `Checked` iff `v.infoVisible` (the preference, even if the card is hidden for lack of an image). Action `toggleInfoOverlay`. Accelerator `I`. **Disabled** when `grid.Visible()`; enabled with no files (card stays hidden until there is an image, same as `I`). |
| 15 | **Copy image** (`Cmd/Ctrl+C`): `copySelection()` — pixels in the image view, file references in the grid. Display-only `CustomShortcut{KeyC, KeyModifierShortcutDefault}` (the real binding remains `fyne.ShortcutCopy` in `shortcuts.go`; do not register a second handler). **Disabled** when `FileCount()==0`. |
| 16 | **Copy image path** (`Cmd/Ctrl+Shift+C`): `copyPathToClipboard()`. Display-only `CustomShortcut{KeyC, ShortcutDefault\|Shift}`. **Disabled** when `FileCount()==0`. |
| 17 | **Move image to Trash** (`Shift+Delete` / German Entf): `requestDelete()` — current file, or the grid selection/highlight. Display-only `CustomShortcut{KeyDelete, KeyModifierShift}` (the real binding remains `ShortcutCut` + `deletion.ShortcutHandler`; do not register a second handler). **Disabled** when `FileCount()==0`. App vocabulary is Trash, not “bin”. |
| 18 | Copy/trash tests that invoke `Action` must stub clipboard (`uitest.StubClipboardCopy`) or only assert `deletion.Visible()`; never hit the real desktop. |

## Why not a new package

An Actions menu is the app deciding how sort + grid duplicate filters compose (`ARCHITECTURE.md`). Settings already jumps sort through `viewer.SetSortMode`. Grid already owns hide/browse. `internal/ui` is the composition point, next to `menu.go` / `windowmenu.go`.

## Global Constraints

- Do **not** `git commit`. `AGENTS.md` forbids it; suggested messages are for the user.
- Do **not** add `TODO`/`FIXME` comments to source. Open work stays in `todos.md`. Do **not** move Menu Actions into Done until Florian accepts.
- Every user-visible string is `lang.L("English text")` with that exact key in `translations/en.json` **and** `translations/de.json`. Guard: `TestTranslations_EveryLocaleCoversEnglish` in `main_test.go`.
- Feature packages talk through their own `Host`. Do **not** import `internal/ui` from `grid`. Do **not** pass `appState`. Do **not** add menu types to any `Host` interface.
- Cross-feature decisions stay in `internal/ui`. Menu enablement is window-geometry’s sibling in that package, not a feature package.
- Do **not** add mutable **package-level** test seams. Close/visibility/dupe-state callbacks are per-instance fields.
- Tests: TDD. No `time.Sleep`. Use `newTestViewer` / `newTestUI`, `dropAndWait`, `waitUntilLoaded`, `waitForSort`, `grid.Settle`, `loadPatternedTriple` (in `step_test.go`) for hashed duplicate fixtures. Fyne’s test driver runs `fyne.Do` inline.
- Do **not** call `ShowManual` / `F1` from `internal/ui` tests (manual.md panics under the test theme).
- `gofmt -l -w` every touched file. Match CI before handoff: `gofmt`, `go vet ./...`, `go build ./...`, then focused tests; the parent runs `go test -race ./...` after the last code task.
- Work from `/Users/ronin/Projects/picfetch`. Golden screenshots are out of scope: **do not** run `make golden` and do not touch `internal/ui/testdata/`.
- English comments; match surrounding style. Verify identifiers against the files you open.
- Do **not** change `S` / `D` / `Shift+D` semantics except extracting shared helpers that those keys then call.
- Do **not** change Darwin Window-merge (`windowmenu_darwin.go`) except insofar as tests that index the Window menu must move from `Items[2]` to `Items[3]`.

## Subagent roster

Cursor’s Task tool does **not** offer Opus. Use `cursor-grok-4.6-xhigh` wherever this plan would have used Opus (judgment, easy-to-miss call sites, final review).

Run **strictly in order**. Do not dispatch implementers in parallel. Do not commit.

| Task | Role | `subagent_type` | Model | Why |
| --- | --- | --- | --- | --- |
| 1 | Implementer | `go-expert` | `cursor-grok-4.6-xhigh` | Hash-apply vs finishBrowse fire sites are easy to miss; last-job-only. |
| 1 | Reviewer | `generalPurpose` | `cursor-grok-4.5-high-fast` | API/layering check. |
| 2 | Implementer | `go-expert` | `composer-2.5-fast` | Structure + translations; follow Window/File menu patterns. |
| 2 | Reviewer | `generalPurpose` | `cursor-grok-4.5-high-fast` | Labels/order/shortcuts/bar index. |
| 3 | Implementer | `go-expert` | `cursor-grok-4.6-xhigh` | State engine; miss a SetSortMode / HighlightChanged / updateFileMenuState site and the menu lies. |
| 3 | Reviewer | `generalPurpose` | `cursor-grok-4.6-xhigh` | State matrix. |
| 4 | Implementer | `go-expert` | `cursor-grok-4.6-xhigh` | Menu Show variants vs Shift+D; extract D helpers. |
| 4 | Reviewer | `generalPurpose` | `cursor-grok-4.6-xhigh` | Same reason. |
| 5 | Implementer | `go-expert` | `cursor-grok-4.6-xhigh` | Rotate/zoom/merge/info/copy/trash enablement + existing shortcut display-only traps (Copy/Cut). |
| 5 | Reviewer | `generalPurpose` | `cursor-grok-4.6-xhigh` | Easy to double-bind Cmd+C or Shift+Delete. |
| 6 | Implementer | `generalPurpose` | `composer-2.5-fast` | Manual EN/DE + `ARCHITECTURE.md`. |
| 6 | Reviewer | `generalPurpose` | `cursor-grok-4.5-high-fast` | Doc/spec check. |
| Final | Whole-branch review | `generalPurpose` | `cursor-grok-4.6-xhigh` | Cross-package + missed call sites. |
| Parent | Fix-up after each task | this session | inherit | Read the diff, re-run the task’s tests, fix, then review. |

## File map

| File | Change | Task |
| --- | --- | --- |
| `internal/ui/grid/dupes.go` | `SourceDuplicateGroupSize`; `onDupeState` / `SetOnDupeStateChanged` / `fireDupeState`; fire from the listed sites. | 1 |
| `internal/ui/grid/dupes_test.go` | Size + callback tests (use `pairAndUnique` / existing inject patterns). | 1 |
| `internal/ui/viewer.go` | `actionsSortItems`, `actionsHideItem`, `actionsShowVariantItem`, plus rotate/zoom/merge/info/copy/path/trash item fields. | 2 |
| `internal/ui/menu.go` | Build Actions menu; bar order; wire `SetOnDupeStateChanged`. | 2, 3 |
| `internal/ui/actionmenu.go` | **Create.** apply/update; all Actions helpers. | 2, 3, 4, 5 |
| `internal/ui/actionmenu_test.go` | **Create.** Structure, state, actions. | 2, 3, 4, 5 |
| `internal/ui/menu_test.go` | Five top-level menus; Actions labels/order/separators. | 2 |
| `internal/ui/windowmenu_test.go` | `windowMenu` helper: `Items[3]`; `len < 4`. | 2 |
| `translations/en.json`, `de.json` | All new keys in default 9. | 2 |
| `internal/ui/sort.go` | Both `SetSortMode` branches update Actions. | 3 |
| `internal/ui/save.go` | `updateFileMenuState` applies Actions before `refreshMainMenu`. | 3 |
| `internal/ui/windowmenu.go` | `updateWindowMenuState` also `applyActionsMenuState` before Refresh. | 3 |
| `internal/ui/viewer.go` | `HighlightChanged`; `SetMergeMode` updates merge checkmark. | 3, 5 |
| `internal/ui/info.go` | `toggleInfoOverlay` updates info checkmark. | 5 |
| `internal/ui/keys.go` | `KeyD` uses extracted hide/browse helpers (Task 4 only). | 4 |
| `ARCHITECTURE.md` | `menu.go` row: Actions between Favorites and Window. | 6 |
| `internal/ui/help/manual.md`, `manual_de.md` | §12 Actions bullets; §18. | 6 |

### Identifier lock (verify in the files you open)

As of this plan:

- `filesort.Modes()` returns `{ByName, ByCaptureDate, ByModTime, BySize, ByDropOrder}` — that is both `Next`’s cycle and the submenu order. `DisplayName` is already translated.
- `viewer.SetSortMode` already jumps (Settings). `toggleSort` is `SetSortMode(Next())`. `S` is behind the `len(files)<2` guard in `keys.go`; the menu is not.
- `toggleMergeMode` / `SetMergeMode` / `MergeMode` live on `viewer` (`viewer.go`). `infoVisible` / `toggleInfoOverlay` live in `info.go`.
- `rotateBy(1)` is clockwise `R`. `v.zoom.In()` / `Out()` are `+`/`-`. `copySelection` / `copyPathToClipboard` / `requestDelete` already match the shortcuts.
- Copy’s real shortcut is `&fyne.ShortcutCopy{}`; delete’s is `&fyne.ShortcutCut{}` via `deletion.ShortcutHandler`. Menu `Shortcut` fields are **display-only**.
- `len(v.displayFrames)==0` is rotateBy’s own no-op guard — use that, not `img.Image==nil` (welcome art can fill `img`).
- `grid.Overview` already has `HideDuplicates`, `SetHideDuplicates`, `BrowsingDuplicates`, `SetBrowsingDuplicates`, `ToggleBrowseDuplicates`. `groupSize` is unexported: `0` unhashed, `1` hashed unique, `≥2` a group.
- `SetBrowsingDuplicates` source index: `host.CurrentIndex()`, or `g.fileIndex(g.highlight)` while `g.visible`. `SourceDuplicateGroupSize` must use that same source.
- Window item fields are `windowViewerItem` … `windowHelpItem`. Actions fields use the `actions` prefix.
- `refreshMainMenu` in `windowmenu.go` already Refresh + `mergeNativeWindowMenu`. Reuse it. Do not add a second Refresh path.
- `updateFileMenuState` (`save.go`) already `applyWindowMenuState` then `refreshMainMenu`. Fold `applyActionsMenuState` in next to `applyWindowMenuState`.
- `loadPatternedTriple` (`step_test.go`) drops three patterned JPEGs and `Warm()`s the grid — use it for viewer-level hide/browse menu tests.
- `waitForSort` lives in `internal/ui/harness_test.go`.
- `stubKeyModifiers` lives in `internal/ui/rotate_test.go` (same package).

## Disabled / Checked matrix

`applyActionsMenuState` implements exactly this. Nil-guard `v.actionsHideItem == nil` (construction order: `registerFeatures` before `buildMainMenu`). Never nil-deref `v.grid`.

| Item | `Checked` | `Disabled` |
| --- | --- | --- |
| Sort child `i` | `v.SortMode() == filesort.Modes()[i]` | always `false` |
| Show/Hide duplicates | `v.grid.HideDuplicates()` | `v.FileCount()==0` |
| Show variants | `v.grid.BrowsingDuplicates()` | `v.FileCount()==0 \|\| v.slides.Active() \|\| !((v.grid.HideDuplicates() && v.grid.SourceDuplicateGroupSize() >= 2) \|\| v.grid.BrowsingDuplicates())` |
| Rotate image / Zoom in / Zoom out | n/a | `len(v.displayFrames)==0 \|\| v.grid.Visible()` |
| Toggle merge mode | `v.MergeMode()` | always `false` |
| Show/Hide info overlay | `v.infoVisible` | `v.grid.Visible()` |
| Copy image / Copy image path / Move image to Trash | n/a | `v.FileCount()==0` |

`hasLoadedImage` for rotate/zoom: `len(v.displayFrames) > 0`.

Fresh `newTestViewer`: Name checked; Show/Hide duplicates + Show variants + rotate/zoom/copy/path/trash **disabled**; merge **enabled** unchecked; info **enabled** unchecked (grid closed).

After `dropAndWait` + `waitUntilLoaded` one JPEG: rotate/zoom/copy/path/trash **enabled**; Show/Hide duplicates enabled; Show variants still disabled (hide off).

After `grid.Toggle()`: rotate/zoom/info **disabled**; copy/path/trash stay enabled.

After `loadPatternedTriple` + `SetHideDuplicates(true)`: Show/Hide checked; Show variants **enabled** on the pair (`size>=2`), **disabled** on the unique (`size==1`).

---

### Task 1: Grid observation (`SourceDuplicateGroupSize` + dupe-state hook)

**Files:**
- Modify: `internal/ui/grid/dupes.go`
- Test: `internal/ui/grid/dupes_test.go`

**Interfaces:**
- Consumes: existing `groupSize`, `fileIndex`, `visible`, `highlight`, `HideDuplicates` / `SetHideDuplicates`, `SetBrowsingDuplicates` / `finishBrowse`, `hashRemaining` last-job `ui.Do`, `SetDuplicateDistance`.
- Produces:
  - `func (g *Overview) SourceDuplicateGroupSize() int`
  - `func (g *Overview) SetOnDupeStateChanged(f func())`
  - unexported `func (g *Overview) fireDupeState()`
  Callbacks are stored on the receiver and **read at fire time**. `nil` means no hook. Do not panic on nil. Do **not** call `rebuildGroups` from `SourceDuplicateGroupSize`.

Source index (copy from `SetBrowsingDuplicates`, do not invent a third rule):

```go
src := g.host.CurrentIndex()
if g.visible {
    src = g.fileIndex(g.highlight)
}
return g.groupSize(src)
```

**Fire `fireDupeState` from exactly these sites** (after the state change, on paths that actually changed something):

1. `SetHideDuplicates` — after `applyFilter` / `jumpIfHiddenExtra`, **not** on the `hideDupes == on` early return.
2. `SetBrowsingDuplicates(false)` — after `applyFilter`, **not** when `browseHost` was already `< 0`.
3. `SetBrowsingDuplicates(true)` — if `hashRemaining() > 0`, fire once after setting `browseHost` (checkmark while analyzing). If pending is `0`, `finishBrowse` fires; do **not** fire twice.
4. `finishBrowse` — at the end of both the unique-bail path (after `browseHost = -1; applyFilter`) and the success path. If `browseHost < 0` on entry, return without firing.
5. `hashRemaining` last-job `ui.Do` (`remaining == 0`) after installing groups / `finishBrowse` / hide apply. If that callback already called `finishBrowse`, the extra fire is required to be **harmless** (idempotent). Mid-interval hide applies (`remaining != 0`) must **not** fire.
6. `SetDuplicateDistance` — after existing hide/browse work. If neither hide nor browse is on, still `rebuildGroups()` then fire so Show-variant availability tracks a live slider change. Early return when `n == g.dupeDist` does not fire.

- [ ] **Step 1: Write the failing size tests**

Use `pairAndUnique` (grid already visible, Warm already hashed, but groups may be empty until hide/browse). Pattern:

```go
func TestSourceDuplicateGroupSize_UnknownUntilGroupsBuilt(t *testing.T) {
	g, host := pairAndUnique(t)
	host.index = 0
	if got := g.SourceDuplicateGroupSize(); got != 0 {
		t.Fatalf("before hide/browse rebuild: size = %d, want 0 (unknown)", got)
	}

	g.SetHideDuplicates(true)
	if got := g.SourceDuplicateGroupSize(); got != 2 {
		t.Fatalf("pair representative size = %d, want 2", got)
	}

	host.index = 2
	// grid is visible: source is the highlight, not host.index.
	// Move the ring to the unique cell (display index of host 2).
	g.setHighlight(g.displayIndexOf(2))
	if got := g.SourceDuplicateGroupSize(); got != 1 {
		t.Fatalf("unique cell size = %d, want 1", got)
	}
}

func TestSourceDuplicateGroupSize_ClosedGridUsesCurrentIndex(t *testing.T) {
	g, host := pairAndUnique(t)
	g.SetHideDuplicates(true)
	g.Close()
	host.index = 2
	if g.Visible() {
		t.Fatal("premises: grid closed")
	}
	if got := g.SourceDuplicateGroupSize(); got != 1 {
		t.Fatalf("closed grid unique current = %d, want 1", got)
	}
}
```

If `displayIndexOf` is unexported, you are in package `grid` — call it. If `setHighlight` needs the grid visible, the first test already has `pairAndUnique` toggling it open.

- [ ] **Step 2: Run size tests to verify they fail**

Run: `go test -count=1 -run 'TestSourceDuplicateGroupSize_' ./internal/ui/grid/`

Expected: FAIL (`SourceDuplicateGroupSize` undefined).

- [ ] **Step 3: Write the failing callback tests**

```go
func TestSetOnDupeStateChanged_HideAndBrowse(t *testing.T) {
	g, _ := pairAndUnique(t)
	var n int
	g.SetOnDupeStateChanged(func() { n++ })

	g.SetHideDuplicates(true)
	if n != 1 {
		t.Fatalf("after hide on: n=%d, want 1", n)
	}
	g.SetHideDuplicates(true)
	if n != 1 {
		t.Fatalf("idempotent hide fired: n=%d", n)
	}
	g.SetHideDuplicates(false)
	if n != 2 {
		t.Fatalf("after hide off: n=%d, want 2", n)
	}

	n = 0
	g.SetOnDupeStateChanged(func() { n++ }) // still read at fire time
	g.SetBrowsingDuplicates(true)
	if !g.BrowsingDuplicates() {
		t.Fatal("premises: pair should browse")
	}
	if n < 1 {
		t.Fatalf("browse on did not fire: n=%d", n)
	}

	was := n
	g.SetBrowsingDuplicates(false)
	if g.BrowsingDuplicates() {
		t.Fatal("browse should be off")
	}
	if n <= was {
		t.Fatalf("browse off did not fire: n=%d was=%d", n, was)
	}
}

func TestSetOnDupeStateChanged_SetAfterHideStillFiresOnNextChange(t *testing.T) {
	g, _ := pairAndUnique(t)
	g.SetHideDuplicates(true)
	var n int
	g.SetOnDupeStateChanged(func() { n++ })
	g.SetHideDuplicates(false)
	if n != 1 {
		t.Fatalf("hook registered after hide-on must still run on hide-off: n=%d", n)
	}
}

func TestSetOnDupeStateChanged_NilIsNoop(t *testing.T) {
	g, _ := pairAndUnique(t)
	g.SetOnDupeStateChanged(nil)
	g.SetHideDuplicates(true) // must not panic
}
```

Also cover `SetDuplicateDistance` while hide is on: changing the slider fires once (use a distance that actually changes, e.g. `g.SetDuplicateDistance(0)` from the default).

- [ ] **Step 4: Run callback tests to verify they fail**

Run: `go test -count=1 -run 'TestSetOnDupeStateChanged_' ./internal/ui/grid/`

Expected: FAIL (`SetOnDupeStateChanged` undefined).

- [ ] **Step 5: Implement**

On `Overview` (next to `onVisibility`):

```go
onDupeState func()

func (g *Overview) SetOnDupeStateChanged(f func()) { g.onDupeState = f }

func (g *Overview) fireDupeState() {
	if g.onDupeState != nil {
		g.onDupeState()
	}
}

func (g *Overview) SourceDuplicateGroupSize() int {
	src := g.host.CurrentIndex()
	if g.visible {
		src = g.fileIndex(g.highlight)
	}
	return g.groupSize(src)
}
```

Wire the fire sites listed above. Do not hook `applyFilter` in general (too hot). Do not fire from `Toggle` / `Close`.

- [ ] **Step 6: Format and run Task 1 tests**

```bash
gofmt -l -w internal/ui/grid/dupes.go internal/ui/grid/dupes_test.go
go test -count=1 ./internal/ui/grid/
```

Expected: PASS. Do not run the whole module suite yet.

- [ ] **Step 7: Skip git commit.** Parent note: `feat: add grid duplicate-state observation for the Actions menu`.

---

### Task 2: Actions menu structure, translations, item fields

**Files:**
- Create: `internal/ui/actionmenu.go`
- Create: `internal/ui/actionmenu_test.go` (structure tests can live here or in `menu_test.go`; prefer `actionmenu_test.go` for Actions-specific tests, keep bar-shape in `menu_test.go`)
- Modify: `internal/ui/menu.go`
- Modify: `internal/ui/viewer.go` (fields next to `windowHelpItem`)
- Modify: `internal/ui/menu_test.go` (`TestBuildMainMenu_Structure` and a new accelerator test)
- Modify: `internal/ui/windowmenu_test.go` (`windowMenu` index)
- Modify: `translations/en.json`, `translations/de.json`

**Interfaces:**
- Consumes: Task 1 APIs exist but this task does **not** wire `SetOnDupeStateChanged` yet (Task 3).
- Produces: `buildMainMenu` returns File, Favorites, **Actions**, Window, Help. Fields:

```go
actionsSortItems       []*fyne.MenuItem // len 5, index matches filesort.Modes()
actionsHideItem        *fyne.MenuItem
actionsShowVariantItem *fyne.MenuItem
actionsRotateItem      *fyne.MenuItem
actionsZoomInItem      *fyne.MenuItem
actionsZoomOutItem     *fyne.MenuItem
actionsMergeItem       *fyne.MenuItem
actionsInfoItem        *fyne.MenuItem
actionsCopyItem        *fyne.MenuItem
actionsCopyPathItem    *fyne.MenuItem
actionsTrashItem       *fyne.MenuItem
```

This task creates `actionmenu.go` with real one-liners (not panics):

```go
func (v *viewer) setActionsSort(m filesort.Mode) { v.SetSortMode(m) }
func (v *viewer) toggleActionsHideDuplicates() {
	v.grid.SetHideDuplicates(!v.grid.HideDuplicates())
}
func (v *viewer) showActionsVariant() {
	v.grid.ToggleBrowseDuplicates()
	if v.grid.BrowsingDuplicates() && !v.grid.Visible() {
		v.grid.Toggle()
	}
}
func (v *viewer) rotateActionsImage()       { v.rotateBy(1) }
func (v *viewer) zoomActionsIn()            { v.zoom.In() }
func (v *viewer) zoomActionsOut()           { v.zoom.Out() }
func (v *viewer) toggleActionsMergeMode()   { v.toggleMergeMode() }
func (v *viewer) toggleActionsInfoOverlay() { v.toggleInfoOverlay() }
func (v *viewer) copyActionsImage()         { v.copySelection() }
func (v *viewer) copyActionsPath()          { v.copyPathToClipboard() }
func (v *viewer) trashActionsImage()        { v.requestDelete() }
```

Task 4–5 add guards. Structure tests this task only assert labels, order, shortcuts, and seed Disabled.

- [ ] **Step 1: Add translation keys**

`translations/en.json` (identity map):

```json
"Actions": "Actions",
"Show/Hide duplicates": "Show/Hide duplicates",
"Show variants": "Show variants",
"Rotate image": "Rotate image",
"Zoom in": "Zoom in",
"Zoom out": "Zoom out",
"Toggle merge mode": "Toggle merge mode",
"Show/Hide info overlay": "Show/Hide info overlay",
"Copy image": "Copy image",
"Copy image path": "Copy image path",
"Move image to Trash": "Move image to Trash"
```

`translations/de.json`:

```json
"Actions": "Aktionen",
"Show/Hide duplicates": "Duplikate ein-/ausblenden",
"Show variants": "Varianten anzeigen",
"Rotate image": "Bild drehen",
"Zoom in": "Vergrößern",
"Zoom out": "Verkleinern",
"Toggle merge mode": "Zusammenführen-Modus umschalten",
"Show/Hide info overlay": "Info-Overlay ein-/ausblenden",
"Copy image": "Bild kopieren",
"Copy image path": "Bildpfad kopieren",
"Move image to Trash": "Bild in den Papierkorb legen"
```

Do **not** re-add `"Sort order"`, the five `DisplayName` keys, or `"Move to Trash"` (confirmation button). Keep JSON valid (commas).

- [ ] **Step 2: Write the failing structure tests**

Update `TestBuildMainMenu_Structure`:

- `len(menu.Items) == 5`
- labels File, Favorites, **Actions**, Window, Help
- Actions has **14** items: `Sort order`, `Show/Hide duplicates`, `Show variants`, separator, `Rotate image`, `Zoom in`, `Zoom out`, separator, `Toggle merge mode`, `Show/Hide info overlay`, separator, `Copy image`, `Copy image path`, `Move image to Trash`
- items 3, 7, 10 are separators (`IsSeparator`); every other item has a non-nil `Action` except Sort parent (nil or empty func, never `toggleSort`)
- Sort order `ChildMenu` has **5** children, `filesort.DisplayName` of `filesort.Modes()` in order; Name starts `Checked`
- On a fresh viewer: Show/Hide duplicates, Show variants, Rotate, Zoom in, Zoom out, Copy, Copy path, Move to Trash start **Disabled**; Toggle merge mode and Show/Hide info overlay start **enabled**; merge and info unchecked

New test `TestBuildMainMenu_ActionsItemsDisplayTheirAccelerators` (skip separators; match by label):

- Sort parent: `KeyS`, modifier `0`
- Show/Hide duplicates: `KeyD`, `0`
- Show variants: `KeyD`, `KeyModifierShift`
- Rotate image: `KeyR`, `0`
- Zoom in: `KeyPlus`, `0`
- Zoom out: `KeyMinus`, `0`
- Toggle merge mode: `KeyM`, `0`
- Show/Hide info overlay: `KeyI`, `0`
- Copy image: `KeyC`, `fyne.KeyModifierShortcutDefault`
- Copy image path: `KeyC`, `KeyModifierShortcutDefault|KeyModifierShift`
- Move image to Trash: `KeyDelete`, `KeyModifierShift`

Update `windowMenu` in `windowmenu_test.go`:

```go
func windowMenu(v *viewer) *fyne.Menu {
	bar := v.win.MainMenu()
	if bar == nil || len(bar.Items) < 4 {
		return nil
	}
	return bar.Items[3] // File, Favorites, Actions, Window
}
```

Existing `TestWindowMenu_*` tests must keep passing after this index change. Run them in Step 6.

- [ ] **Step 3: Run structure tests to verify they fail**

Run: `go test -count=1 -run 'TestBuildMainMenu_Structure|TestBuildMainMenu_ActionsItemsDisplayTheirAccelerators|TestWindowMenu_' ./internal/ui/`

Expected: FAIL (still 4 top-level menus, or Window tests looking at Actions).

- [ ] **Step 4: Implement fields, `actionmenu.go` stubs, and `buildMainMenu`**

Loop **must** capture `mode` (classic Go loop-variable bug):

```go
modes := filesort.Modes()
sortItems := make([]*fyne.MenuItem, len(modes))
for i, m := range modes {
	mode := m
	it := fyne.NewMenuItem(filesort.DisplayName(mode), func() { view.setActionsSort(mode) })
	if mode == view.SortMode() {
		it.Checked = true
	}
	sortItems[i] = it
}
view.actionsSortItems = sortItems

sortParent := fyne.NewMenuItem(lang.L("Sort order"), nil)
sortParent.ChildMenu = fyne.NewMenu("", sortItems...)
sortParent.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyS}

hideItem := fyne.NewMenuItem(lang.L("Show/Hide duplicates"), view.toggleActionsHideDuplicates)
hideItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyD}
hideItem.Disabled = true
view.actionsHideItem = hideItem

variantItem := fyne.NewMenuItem(lang.L("Show variants"), view.showActionsVariant)
variantItem.Shortcut = &desktop.CustomShortcut{
	KeyName:  fyne.KeyD,
	Modifier: fyne.KeyModifierShift,
}
variantItem.Disabled = true
view.actionsShowVariantItem = variantItem

rotateItem := fyne.NewMenuItem(lang.L("Rotate image"), view.rotateActionsImage)
rotateItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyR}
rotateItem.Disabled = true
view.actionsRotateItem = rotateItem

zoomIn := fyne.NewMenuItem(lang.L("Zoom in"), view.zoomActionsIn)
zoomIn.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyPlus}
zoomIn.Disabled = true
view.actionsZoomInItem = zoomIn

zoomOut := fyne.NewMenuItem(lang.L("Zoom out"), view.zoomActionsOut)
zoomOut.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyMinus}
zoomOut.Disabled = true
view.actionsZoomOutItem = zoomOut

mergeItem := fyne.NewMenuItem(lang.L("Toggle merge mode"), view.toggleActionsMergeMode)
mergeItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyM}
view.actionsMergeItem = mergeItem

infoItem := fyne.NewMenuItem(lang.L("Show/Hide info overlay"), view.toggleActionsInfoOverlay)
infoItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyI}
view.actionsInfoItem = infoItem

copyItem := fyne.NewMenuItem(lang.L("Copy image"), view.copyActionsImage)
copyItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyC, Modifier: fyne.KeyModifierShortcutDefault}
copyItem.Disabled = true
view.actionsCopyItem = copyItem

copyPath := fyne.NewMenuItem(lang.L("Copy image path"), view.copyActionsPath)
copyPath.Shortcut = &desktop.CustomShortcut{
	KeyName:  fyne.KeyC,
	Modifier: fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift,
}
copyPath.Disabled = true
view.actionsCopyPathItem = copyPath

trashItem := fyne.NewMenuItem(lang.L("Move image to Trash"), view.trashActionsImage)
trashItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyDelete, Modifier: fyne.KeyModifierShift}
trashItem.Disabled = true
view.actionsTrashItem = trashItem

actionsMenu := fyne.NewMenu(lang.L("Actions"),
	sortParent, hideItem, variantItem,
	fyne.NewMenuItemSeparator(),
	rotateItem, zoomIn, zoomOut,
	fyne.NewMenuItemSeparator(),
	mergeItem, infoItem,
	fyne.NewMenuItemSeparator(),
	copyItem, copyPath, trashItem,
)
```

Return:

```go
return fyne.NewMainMenu(fileMenu, view.favorites.Menu(), actionsMenu, windowMenu, view.help.Menu())
```

Keep the existing Window-menu construction and `SetOnManualClosed` / visibility hooks. Update the file comment at the top of `menu.go`.

If `NewMenuItem(..., nil)` is inconvenient, use `func() {}` for the Sort parent; **never** `toggleSort`.

- [ ] **Step 5: Format and run Task 2 tests plus translation parity plus Window menu**

```bash
gofmt -l -w internal/ui/menu.go internal/ui/viewer.go internal/ui/actionmenu.go internal/ui/menu_test.go internal/ui/windowmenu_test.go internal/ui/actionmenu_test.go
go test -count=1 -run 'TestBuildMainMenu_|TestWindowMenu_|TestTranslations_EveryLocaleCoversEnglish' ./internal/ui/ ./
```

`TestTranslations_EveryLocaleCoversEnglish` lives in package main: `go test -count=1 -run TestTranslations_EveryLocaleCoversEnglish .`

Expected: PASS.

- [ ] **Step 6: Skip git commit.** Parent note: `feat: add Actions menu structure and translations`.

---

### Task 3: Checkmark / enablement engine and callback wiring

**Files:**
- Modify: `internal/ui/actionmenu.go` — `applyActionsMenuState` / `updateActionsMenuState`
- Modify: `internal/ui/save.go` — `updateFileMenuState` calls `applyActionsMenuState` before the existing `refreshMainMenu`
- Modify: `internal/ui/sort.go` — both `SetSortMode` branches
- Modify: `internal/ui/viewer.go` — `HighlightChanged`; `SetMergeMode` calls `updateActionsMenuState`
- Modify: `internal/ui/info.go` — `toggleInfoOverlay` calls `updateActionsMenuState`
- Modify: `internal/ui/menu.go` — `view.grid.SetOnDupeStateChanged(view.updateActionsMenuState)` after the menu items exist (end of `buildMainMenu`, next to the Window hooks)
- Modify: `internal/ui/windowmenu.go` — `updateWindowMenuState` applies Actions too
- Test: `internal/ui/actionmenu_test.go`

**Interfaces:**
- Consumes: Task 1–2 fields; `SortMode`, `FileCount`, `displayFrames`, `grid.Visible`, `MergeMode`, `infoVisible`, plus the duplicate APIs
- Produces: `applyActionsMenuState` (no Refresh); `updateActionsMenuState` = `applyWindowMenuState` + `applyActionsMenuState` + `refreshMainMenu`

```go
func (v *viewer) applyActionsMenuState() {
	if v.actionsHideItem == nil {
		return
	}
	modes := filesort.Modes()
	cur := v.SortMode()
	for i, item := range v.actionsSortItems {
		if item == nil || i >= len(modes) {
			continue
		}
		item.Checked = modes[i] == cur
		item.Disabled = false
	}
	noFiles := v.FileCount() == 0
	gridUp := v.grid.Visible()
	noImage := len(v.displayFrames) == 0

	v.actionsHideItem.Checked = v.grid.HideDuplicates()
	v.actionsHideItem.Disabled = noFiles
	v.actionsShowVariantItem.Checked = v.grid.BrowsingDuplicates()
	canShowVariants := v.grid.HideDuplicates() && v.grid.SourceDuplicateGroupSize() >= 2
	v.actionsShowVariantItem.Disabled = noFiles || v.slides.Active() || !(canShowVariants || v.grid.BrowsingDuplicates())

	rotZoomOff := noImage || gridUp
	v.actionsRotateItem.Disabled = rotZoomOff
	v.actionsZoomInItem.Disabled = rotZoomOff
	v.actionsZoomOutItem.Disabled = rotZoomOff

	v.actionsMergeItem.Checked = v.MergeMode()
	v.actionsMergeItem.Disabled = false
	v.actionsInfoItem.Checked = v.infoVisible
	v.actionsInfoItem.Disabled = gridUp

	v.actionsCopyItem.Disabled = noFiles
	v.actionsCopyPathItem.Disabled = noFiles
	v.actionsTrashItem.Disabled = noFiles
}

func (v *viewer) updateActionsMenuState() {
	v.applyWindowMenuState()
	v.applyActionsMenuState()
	v.refreshMainMenu()
}
```

Change `updateWindowMenuState` the same way (apply both, one Refresh) so opening the grid greys rotate/zoom/info without a second native rebuild from a second function.

`updateFileMenuState`: apply Window **and** Actions, then one `refreshMainMenu`.

`SetSortMode`: `updateActionsMenuState()` on both branches.

`SetMergeMode`: after `applyTitle()`, `updateActionsMenuState()` so Settings and `M` keep the checkmark in sync. (Task 5 can add this if Task 3 only wires sort/dupes — **do it in Task 3** so merge seed is correct after `M`.)

`HighlightChanged`: snapshot Hide/Show-variants Checked+Disabled, apply Actions, Refresh only if those four bools changed.

Nil-guard the new item pointers the same way as `actionsHideItem` (all assigned together in `buildMainMenu`).

Call `applyActionsMenuState()` at the end of `buildMainMenu`.

- [ ] **Step 1: Write the failing state tests** in `actionmenu_test.go`

Helper `actionsMenu` as below (Items[2]). Also a `actionsItem(m *fyne.Menu, label string) *fyne.MenuItem` that skips separators.

```go
func actionsMenu(v *viewer) *fyne.Menu {
	bar := v.win.MainMenu()
	if bar == nil || len(bar.Items) < 3 {
		return nil
	}
	return bar.Items[2] // File, Favorites, Actions
}
```

Tests (each `TestActionsMenu_*`):

1. **Fresh viewer:** Name checked; Hide+variants+rotate+zoom+copy+path+trash disabled; merge enabled unchecked; info enabled unchecked.
2. **`SetSortMode(filesort.BySize)` with no files:** File size checked, Name unchecked. No sort wait.
3. **After `dropAndWait` + `waitUntilLoaded` one JPEG:** Hide enabled; variants disabled; rotate/zoom/copy/path/trash **enabled**.
4. **`S` with two files:** Capture date checked after one `S` + `waitForSort`.
5. **`loadPatternedTriple` + `SetHideDuplicates(true)`:** Hide checked; variants enabled on pair, disabled on unique (`ShowImage(2)`).
6. **hide on + `ToggleBrowseDuplicates`:** variants checked and enabled.
7. **`togglePictureFrameMode` after a drop:** variants disabled; Hide still enabled; rotate still **enabled** (picture-frame is not the grid).
8. **`closeFiles`:** Hide/variants/rotate/zoom/copy/path/trash disabled; sort checkmark unchanged; merge checkmark unchanged.
9. **`grid.Toggle()` after a loaded JPEG:** rotate, zoom in/out, and info **disabled**; copy/path/trash still enabled. Toggle grid closed: rotate/zoom/info enabled again.
10. **`toggleMergeMode`:** merge item Checked tracks on/off (no files required).
11. **`toggleInfoOverlay`:** info item Checked tracks on/off with no files.

Do **not** open the manual.

- [ ] **Step 2: Run state tests to verify they fail**

Run: `go test -count=1 -run 'TestActionsMenu_' ./internal/ui/`

Expected: FAIL (checkmarks / Disabled not tracking).

- [ ] **Step 3: Implement apply/update, wiring, SetSortMode, HighlightChanged, updateFileMenuState fold**

- [ ] **Step 4: Run Task 3 tests plus neighbors**

```bash
gofmt -l -w internal/ui/actionmenu.go internal/ui/actionmenu_test.go internal/ui/save.go internal/ui/sort.go internal/ui/menu.go internal/ui/viewer.go
go test -count=1 -run 'TestActionsMenu_|TestBuildMainMenu_|TestWindowMenu_|TestToggleSort_|TestSetSortMode_|TestCloseFilesItem_' ./internal/ui/
```

Expected: PASS.

- [ ] **Step 5: Skip git commit.** Parent note: `feat: keep Actions menu checkmarks and enablement in sync`.

---

### Task 4: Actions (sort jump, hide toggle, show variant = Shift+D)

**Files:**
- Modify: `internal/ui/actionmenu.go` (final helpers)
- Modify: `internal/ui/actionmenu_test.go` (action tests)
- Modify: `internal/ui/keys.go` (`KeyD` uses the extracted helpers)
- Modify: `internal/ui/keys_test.go` or rely on existing `step_test.go` Shift+D tests (must still pass)

**Interfaces:**
- Consumes: Task 3 `updateActionsMenuState`; `SetSortMode`; `togglePictureFrameMode`
- Produces: the three methods with the contracts below; `keys.go` `case fyne.KeyD` calls them so the menu cannot drift

Final bodies:

```go
func (v *viewer) setActionsSort(m filesort.Mode) {
	if v.SortMode() == m {
		return
	}
	v.SetSortMode(m)
}

func (v *viewer) toggleHideDuplicates() {
	v.grid.SetHideDuplicates(!v.grid.HideDuplicates())
}

func (v *viewer) browseCurrentDuplicates() {
	if v.slides.Active() {
		return
	}
	v.grid.ToggleBrowseDuplicates()
	if v.grid.BrowsingDuplicates() && !v.grid.Visible() {
		v.grid.Toggle()
	}
}

func (v *viewer) toggleActionsHideDuplicates() {
	if v.FileCount() == 0 {
		return
	}
	v.toggleHideDuplicates()
}

func (v *viewer) showActionsVariant() {
	if v.FileCount() == 0 || v.slides.Active() {
		return
	}
	if v.grid.BrowsingDuplicates() {
		v.browseCurrentDuplicates() // leave browse even if hide is now off
		return
	}
	if !v.grid.HideDuplicates() || v.grid.SourceDuplicateGroupSize() < 2 {
		return
	}
	v.browseCurrentDuplicates()
}
```

Rename Task 2’s `toggleActionsHideDuplicates` body onto `toggleHideDuplicates`. Keep the menu item bound to `toggleActionsHideDuplicates` (the FileCount guard). Bind Show variants to `showActionsVariant`. The menu item is **stricter** than `Shift+D`: it no-ops unless hide is on and the current file has a group (or browse is already on). `keys.go` must **not** gain that extra guard.

In `keys.go` `case fyne.KeyD:` replace the inline Shift/plain bodies with:

```go
if v.keyModifiers()&fyne.KeyModifierShift != 0 {
	v.browseCurrentDuplicates()
	return
}
v.toggleHideDuplicates()
return
```

Do **not** add a FileCount==0 guard on the key path (today `D` with no files still flips the flag). The menu is what greys out.

`SetOnDupeStateChanged` already refreshes after hide/browse. Extra `updateActionsMenuState` at the end of the menu methods is redundant and required to be harmless if you add it; prefer relying on the hook so keys and menu share one refresh.

- [ ] **Step 1: Write the failing action tests**

```go
func TestActionsMenu_SortItemJumpsWithoutCycling(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White), uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White))
	waitForSort(t, v)
	waitUntilLoaded(t, v)

	// Modes()[3] is BySize — do not cycle S three times.
	v.actionsSortItems[3].Action()
	waitForSort(t, v)
	if v.SortMode() != filesort.BySize {
		t.Fatalf("SortMode = %v, want BySize", v.SortMode())
	}
	if !v.actionsSortItems[3].Checked || v.actionsSortItems[0].Checked {
		t.Fatal("File size should be the only checked sort item")
	}
}

func TestActionsMenu_SortItemNoopWhenAlreadySelected(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White), uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White))
	waitForSort(t, v)
	waitUntilLoaded(t, v)
	if p := filesort.Label(v.SortMode()); p != "" {
		t.Fatalf("premises: default ByName must have an empty title prefix, got %q", p)
	}
	v.actionsSortItems[0].Action() // Name, the default
	if v.SortMode() != filesort.ByName {
		t.Fatal("re-choosing Name must leave ByName")
	}
	if p := filesort.Label(v.SortMode()); p != "" {
		t.Fatalf("re-choosing Name must not start a sort or change the title prefix, got %q", p)
	}
}

func TestActionsMenu_HideTogglesLikeD(t *testing.T) {
	v := loadPatternedTriple(t)
	v.actionsHideItem.Action()
	if !v.grid.HideDuplicates() || !v.actionsHideItem.Checked {
		t.Fatal("Show/Hide duplicates should turn hide on and checkmark")
	}
	v.actionsHideItem.Action()
	if v.grid.HideDuplicates() || v.actionsHideItem.Checked {
		t.Fatal("second click should turn hide off")
	}
}

func TestActionsMenu_HideNoopsWithoutFiles(t *testing.T) {
	v := newTestViewer(t)
	v.actionsHideItem.Action()
	if v.grid.HideDuplicates() {
		t.Fatal("no files: hide must stay off")
	}
}

func TestActionsMenu_ShowVariantsOpensGridOnPairAfterHide(t *testing.T) {
	v := loadPatternedTriple(t)
	v.actionsHideItem.Action()
	if v.actionsShowVariantItem.Disabled {
		t.Fatal("premises: hide on + pair should enable Show variants")
	}
	v.actionsShowVariantItem.Action()
	v.grid.Settle()
	if !v.grid.Visible() || !v.grid.BrowsingDuplicates() {
		t.Fatal("Show variants on a duplicate (hide on) should browse and open the grid")
	}
	if !v.grid.HideDuplicates() {
		t.Fatal("Show variants must leave hide on")
	}
	if !v.actionsShowVariantItem.Checked {
		t.Fatal("Show variants should be checked while browsing")
	}
}

func TestActionsMenu_ShowVariantsNoopsWhileHideOff(t *testing.T) {
	v := loadPatternedTriple(t)
	if !v.actionsShowVariantItem.Disabled {
		t.Fatal("premises: hide off must grey Show variants")
	}
	v.actionsShowVariantItem.Action()
	v.grid.Settle()
	if v.grid.Visible() || v.grid.BrowsingDuplicates() {
		t.Fatal("hide off: Show variants must not start browse")
	}
}

func TestActionsMenu_ShowVariantsNoopOnUnique(t *testing.T) {
	v := loadPatternedTriple(t)
	v.grid.SetHideDuplicates(true) // rebuild groups so size==1 is known
	v.ShowImage(2)
	waitUntilLoaded(t, v)
	if !v.actionsShowVariantItem.Disabled {
		t.Fatal("premises: unique should grey Show variants")
	}
	v.actionsShowVariantItem.Action()
	v.grid.Settle()
	if v.grid.Visible() || v.grid.BrowsingDuplicates() {
		t.Fatal("unique: must not open browse")
	}
}

func TestActionsMenu_ShowVariantsNoopsDuringPictureFrame(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	v.togglePictureFrameMode()
	v.actionsShowVariantItem.Action()
	if v.grid.Visible() || v.grid.BrowsingDuplicates() {
		t.Fatal("picture-frame: Show variants must no-op, like Shift+D")
	}
}

func TestActionsMenu_ShowVariantsSecondClickLeavesBrowse(t *testing.T) {
	v := loadPatternedTriple(t)
	v.actionsHideItem.Action()
	v.actionsShowVariantItem.Action()
	v.grid.Settle()
	v.actionsShowVariantItem.Action()
	v.grid.Settle()
	if v.grid.BrowsingDuplicates() {
		t.Fatal("second click should leave browse, like Shift+D")
	}
}
```

Existing `TestHandleKeyEvent_ShiftDOpensGridOnCurrentGroup` and `TestHandleKeyEvent_DTogglesHideDuplicatesWhenGridClosed` must still pass after the extract.

- [ ] **Step 2: Run action tests to verify they fail**

Run: `go test -count=1 -run 'TestActionsMenu_SortItem|TestActionsMenu_Hide|TestActionsMenu_ShowVariants' ./internal/ui/`

Expected: FAIL on missing no-op / picture-frame guard / extract drift.

- [ ] **Step 3: Implement final helpers and switch `keys.go` to them**

- [ ] **Step 4: Run all Actions tests plus key neighbors**

```bash
gofmt -l -w internal/ui/actionmenu.go internal/ui/actionmenu_test.go internal/ui/keys.go
go test -count=1 -run 'TestActionsMenu_|TestBuildMainMenu_|TestHandleKeyEvent_D|TestHandleKeyEvent_ShiftD|TestToggleSort_|TestWindowMenu_' ./internal/ui/
```

Expected: PASS.

- [ ] **Step 5: Skip git commit.** Parent note: `feat: Actions menu drives sort, show/hide duplicates, and show variants`.

---

---

### Task 5: Rotate, zoom, merge, info, copy, trash

**Files:**
- Modify: `internal/ui/actionmenu.go` (guards on the Task 2 one-liners)
- Modify: `internal/ui/actionmenu_test.go`
- Modify: `internal/ui/viewer.go` `SetMergeMode` — `updateActionsMenuState` after `applyTitle` if Task 3 did not already
- Modify: `internal/ui/info.go` `toggleInfoOverlay` — `updateActionsMenuState` after the toggle if Task 3 did not already
- Do **not** change `shortcuts.go` bindings

**Interfaces:**
- Consumes: Task 3 matrix; `rotateBy`, `zoom.In`/`Out`, `toggleMergeMode`, `toggleInfoOverlay`, `copySelection`, `copyPathToClipboard`, `requestDelete`
- Produces: the eight `*Actions*` methods with the same no-op guards as `Disabled` (defensive: tests may invoke Action while Disabled)

```go
func (v *viewer) rotateActionsImage() {
	if len(v.displayFrames) == 0 || v.grid.Visible() {
		return
	}
	v.rotateBy(1)
}

func (v *viewer) zoomActionsIn() {
	if len(v.displayFrames) == 0 || v.grid.Visible() {
		return
	}
	v.zoom.In()
}

func (v *viewer) zoomActionsOut() {
	if len(v.displayFrames) == 0 || v.grid.Visible() {
		return
	}
	v.zoom.Out()
}

func (v *viewer) toggleActionsMergeMode() { v.toggleMergeMode() }

func (v *viewer) toggleActionsInfoOverlay() {
	if v.grid.Visible() {
		return
	}
	v.toggleInfoOverlay()
}

func (v *viewer) copyActionsImage() {
	if v.FileCount() == 0 {
		return
	}
	v.copySelection()
}

func (v *viewer) copyActionsPath() { v.copyPathToClipboard() }

func (v *viewer) trashActionsImage() {
	if v.FileCount() == 0 {
		return
	}
	v.requestDelete()
}
```

If `toggleInfoOverlay` already calls `updateActionsMenuState`, `toggleActionsInfoOverlay` must not skip that when it returns early on grid.

- [ ] **Step 1: Write the failing action tests**

```go
func TestActionsMenu_RotateTurnsImageClockwise(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	waitUntilLoaded(t, v)
	if v.rotation != 0 {
		t.Fatal("premises: unrotated")
	}
	v.actionsRotateItem.Action()
	if v.rotation != 1 {
		t.Fatalf("rotation = %d, want 1 (clockwise R)", v.rotation)
	}
}

func TestActionsMenu_RotateNoopsWithoutImage(t *testing.T) {
	v := newTestViewer(t)
	v.actionsRotateItem.Action()
	if v.rotation != 0 {
		t.Fatal("no image: rotate must no-op")
	}
}

func TestActionsMenu_RotateNoopsWhileGridVisible(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	waitUntilLoaded(t, v)
	v.grid.Toggle()
	v.actionsRotateItem.Action()
	if v.rotation != 0 {
		t.Fatal("grid up: rotate must no-op")
	}
}

func TestActionsMenu_ZoomInThenOutChangesPercent(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 8, 8, color.White))
	waitUntilLoaded(t, v)
	start := v.zoom.Percent()
	v.actionsZoomInItem.Action()
	if v.zoom.Percent() <= start {
		t.Fatalf("zoom in percent = %d, want > %d", v.zoom.Percent(), start)
	}
	afterIn := v.zoom.Percent()
	v.actionsZoomOutItem.Action()
	if v.zoom.Percent() >= afterIn {
		t.Fatalf("zoom out percent = %d, want < %d", v.zoom.Percent(), afterIn)
	}
}

func TestActionsMenu_MergeToggleChecksItem(t *testing.T) {
	v := newTestViewer(t)
	v.actionsMergeItem.Action()
	if !v.MergeMode() || !v.actionsMergeItem.Checked {
		t.Fatal("merge should turn on and checkmark")
	}
	v.actionsMergeItem.Action()
	if v.MergeMode() || v.actionsMergeItem.Checked {
		t.Fatal("second click should turn merge off")
	}
}

func TestActionsMenu_InfoToggleChecksItem(t *testing.T) {
	v := newTestViewer(t)
	v.actionsInfoItem.Action()
	if !v.infoVisible || !v.actionsInfoItem.Checked {
		t.Fatal("info overlay preference should turn on")
	}
}

func TestActionsMenu_CopyPathWritesClipboard(t *testing.T) {
	v := newTestViewer(t)
	u := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	dropAndWait(t, v, u)
	waitUntilLoaded(t, v)
	v.actionsCopyPathItem.Action()
	if got := v.app.Clipboard().Content(); got != u.Path() {
		t.Fatalf("clipboard = %q, want %q", got, u.Path())
	}
}

func TestActionsMenu_CopyImageUsesStub(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	waitUntilLoaded(t, v)
	called := make(chan struct{}, 1)
	uitest.StubClipboardCopy(t, func([]byte) error { called <- struct{}{}; return nil })
	v.actionsCopyItem.Action()
	select {
	case <-called:
	case <-time.After(testTimeout):
		t.Fatal("Copy image should encode and dispatch clipboard.CopyImage")
	}
}

func TestActionsMenu_TrashOpensConfirmation(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	waitUntilLoaded(t, v)
	v.actionsTrashItem.Action()
	if !v.deletion.Visible() {
		t.Fatal("Move image to Trash should open the confirmation")
	}
}

func TestActionsMenu_TrashNoopsWithoutFiles(t *testing.T) {
	v := newTestViewer(t)
	v.actionsTrashItem.Action()
	if v.deletion.Visible() {
		t.Fatal("no files: trash must not open")
	}
}
```

Import `time` if `testTimeout` is already in package `ui` tests (it is). Use `uitest.StubClipboardCopy`; do not call the real clipboard.

- [ ] **Step 2: Run to verify fail / implement guards / run**

```bash
gofmt -l -w internal/ui/actionmenu.go internal/ui/actionmenu_test.go internal/ui/viewer.go internal/ui/info.go
go test -count=1 -run 'TestActionsMenu_|TestCopyPathToClipboard_|TestHandleKeyEvent_D|TestWindowMenu_' ./internal/ui/
```

Expected: PASS. Existing `TestCopyImageToClipboard_*` still pass (no second Cmd+C handler).

- [ ] **Step 3: Skip git commit.** Parent note: `feat: Actions menu rotate, zoom, merge, info, copy, and trash`.

---

### Task 6: Docs (ARCHITECTURE + manuals)

**Files:**
- Modify: `ARCHITECTURE.md` — `menu.go` row: Actions between Favorites and Window; checkmarks vs Window’s grey-out; `actionmenu.go` composition; grid stays menu-ignorant via `SetOnDupeStateChanged`.
- Modify: `internal/ui/help/manual.md` sections **12. Menu** and **18. Quick reference**
- Modify: `internal/ui/help/manual_de.md` matching sections
- Do **not** mark `todos.md` Menu Actions as Done

**Interfaces:** none.

English manual, insert in §12 after the Favorites bullets and **before** Window (keep ASCII `->`):

```markdown
- **Actions -> Sort order** (`S`) — submenu of the same five orders as
  Settings: Name, Capture date, Modified date, File size, Drop order. The
  current order has a checkmark. Choosing one jumps to it (it does not
  cycle). `S` still cycles. Re-choosing the current order does nothing
- **Actions -> Show/Hide duplicates** (`D`) — same as `D`: hides extra
  copies of the same shot and checkmarks while hide is on. Greyed out
  when no files are loaded. Works from the menu even while a grid search
  is open
- **Actions -> Show variants** (`Shift+D`) — shows every copy of the
  current/highlighted shot in the grid, same as `Shift+D` once it runs.
  Checkmarked while that browse filter is on. Greyed out until
  Show/Hide duplicates is on **and** the current file has duplicates,
  and also when no files are loaded or picture-frame mode is on. The
  `Shift+D` key still works with hide off; this menu item does not
- **Actions -> Rotate image** (`R`) — 90° clockwise, view-only, same as
  `R`. Greyed out with no image loaded or while the grid is up.
  `Shift+R` stays keyboard-only
- **Actions -> Zoom in** (`+`) / **Zoom out** (`-`) — same as the `+`/`-`
  keys. Greyed out with no image loaded or while the grid is up
- **Actions -> Toggle merge mode** (`M`) — same as `M`. Checkmarked while
  merge is on. Works before any files are loaded
- **Actions -> Show/Hide info overlay** (`I`) — same as `I`. Checkmarked
  while the overlay preference is on. Greyed out while the grid is up
- **Actions -> Copy image** (`Cmd/Ctrl+C`) — the displayed pixels, or the
  grid selection as files. Greyed out when no files are loaded
- **Actions -> Copy image path** (`Cmd/Ctrl+Shift+C`) — the current
  file's path. Greyed out when no files are loaded
- **Actions -> Move image to Trash** (`Shift+Delete`) — same as
  `Shift+Delete`: confirms, then moves the current file (or the grid
  selection) to the Trash. Greyed out when no files are loaded
```

§18: mention the Actions menu next to the matching key bullets (sort, D, Shift+D, R, zoom, M, I, copy, delete).

German: **Aktionen ->** Sortierreihenfolge / Duplikate ein-/ausblenden / Varianten anzeigen / Bild drehen / Vergrößern / Verkleinern / Zusammenführen-Modus umschalten / Info-Overlay ein-/ausblenden / Bild kopieren / Bildpfad kopieren / Bild in den Papierkorb legen. Match `manual_de.md` tone (`Sie`, ASCII `->`). `Shift+Entf` may be mentioned as the German key name alongside `Shift+Delete`.

- [ ] **Step 1: Edit the three doc files**
- [ ] **Step 2: No new `lang.L` keys.** Manual prose is not in `translations/*.json`.
- [ ] **Step 3:** `gofmt` N/A for md. Parent will run the full suite after this task.
- [ ] **Step 4: Skip git commit.** Parent note: `docs: document the Actions menu`.

---

## Parent verification after Task 6

From the repo root:

```bash
gofmt -l .
go vet ./...
go build ./...
go test -race ./...
```

Expected: all pass. Do not run `make golden`.

Suggested commit message (user only):

```
feat: add an Actions menu for sort, duplicates, rotate, zoom, and clipboard

Checkmarked sort, merge, hide-duplicates, variants, and info. Show
variants stays off until hide is on and the current file has a group.
Rotate, zoom, copy, path, and trash match the existing keys.
```

## Out of scope

- EXIF / Viewer / Grid / Picture-frame / Help in this menu (Window menu)
- Shuffle (`Shift+P`); counter-clockwise rotate (`Shift+R`); zoom 0/1
- Changing `S` so it works with 0–1 files (menu/Settings already can)
- Checkmarks on Window items
- Golden screenshots
- Eager hashing just to enable Show variants

## Spec coverage (self-review)

| Requirement | Task |
| --- | --- |
| Sort submenu of all sort options | 2 |
| Current sort checkmarked (not greyed) | 3 |
| Show/Hide duplicates checkmarked when hide is on, togglable | 3, 4 |
| Show variants available iff hide is on and current item has a group | 1, 3, 4 |
| Show variants action = Shift+D once enabled (open grid, unique/hide-off no-op, ignore picture-frame) | 4 |
| Rotate / zoom in / zoom out | 2, 3, 5 |
| Toggle merge mode checkmark | 3, 5 |
| Show/Hide info overlay | 3, 5 |
| Copy image / copy path / move to Trash | 2, 3, 5 |
| Keyboard hints including Cmd+C and Shift+Delete | 2 |
| Translations | 2 |
| Manual + ARCHITECTURE | 6 |
| EXIF / showing windows greyed | none (Window menu; default 1) |
| Darwin still one Window menu | 2 (index only; merge code untouched) |
