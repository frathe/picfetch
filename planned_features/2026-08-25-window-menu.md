# Window Menu Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Parent protocol (this session):** Do **not** start until Florian says to start the subagents. Dispatch **one implementer at a time**, in task order. After each task the **parent reads the diff, re-runs that task's verification command, and fixes drift** before dispatching a reviewer and then the next implementer. Do **not** `git commit` (`AGENTS.md`); the parent suggests one message at the very end.
>
> Task subagents: read only your task, Global Constraints, and that task's Interfaces. Do not read this whole file. **If an identifier in this plan disagrees with a file you open, the file wins.**

**Goal:** Add a **Window** menu that can open the viewer, EXIF panel, grid, picture-frame mode, and the manual, greying out whichever surface is already showing.

**Architecture:** Cross-feature composition stays in `internal/ui` (`menu.go`), the same rule as File + Favorites + Help. Feature packages do not learn about menus. They grow small **observation** APIs (`Open` / `ManualOpen` / visibility and close hooks) so the menu can grey items when the user closes a window with Escape or the window chrome, not only when they used the menu. Menu actions **show/enter** a surface; they do not toggle it off. Leaving grid or picture-frame is **Window → Viewer**.

**Tech Stack:** Go 1.26.7, Fyne v2.8.0, `lang.L` + `translations/{en,de}.json`, packages `internal/ui`, `internal/ui/help`, `internal/ui/exifwin`, `internal/ui/grid`, `internal/ui/slideshow`.

**Source todo:** `todos.md` — **Menu Window** (not Menu Actions).

## Recommended defaults (locked unless Florian overrides)

These are the defaults the tasks implement. Open questions in the parent message can change them **before Task 1 starts**.

| # | Decision |
|---|----------|
| 1 | **Scope is Menu Window only.** Sort submenu, hide-duplicates checkmark, and “show variant” belong to **Menu Actions** and must not land here. |
| 2 | **Top-level menu label** `lang.L("Window")`. Bar order: **File, Favorites, Window, Help**. |
| 3 | **Items, in this order:** Viewer, EXIF Data, Grid View, Picture-frame mode, Help. |
| 4 | **Labels:** Viewer → `lang.L("Viewer")`; EXIF → reuse `lang.L("EXIF Data")` (same as the panel title); Grid → `lang.L("Grid View")`; Picture-frame → `lang.L("Picture-frame mode")` (matches the manual heading / existing hyphenation); Help → reuse `lang.L("Help")`. Window → Help opens the **manual** (`help.ShowManual`), not About. |
| 5 | **Grey-out means `Disabled = true`**, not `Checked`. A showing surface cannot be re-chosen from this menu. `G` / `P` still **toggle**. `V` is show-only (same as Window → Viewer). `Esc` and window chrome still close as they do today. |
| 6 | **Actions show/enter; they do not toggle off.** `grid.Toggle` / `slides.Toggle` must not be the menu callbacks — a Disabled item whose Action still ran would *leave* the mode. Defensive guards in the actions even if Fyne skips Disabled clicks. |
| 7 | **Viewer** means “the main image view”: `grid.Close()`, `slides.Exit()` + `resetFade` if picture-frame was on, then `v.win.RequestFocus()`. Disabled when `!grid.Visible() && !slides.Active()` (already there). Does **not** close EXIF, Settings, or Help. |
| 8 | **EXIF** calls `v.exif.Show()` (same as `E` and the info-overlay link). Disabled when `v.exif.Open()` **or** `DisplayedFile` is missing (Show is already a no-op then). |
| 9 | **Grid View** opens the grid if it is closed. Disabled when the grid is visible, `FileCount()==0`, or `slides.Active()` (same as the `G` key). Does not close the grid. |
| 10 | **Picture-frame mode** enters via `togglePictureFrameMode` if it is off. Disabled when `slides.Active()` or `FileCount()==0`. Entering still closes the grid (existing glue). Does not exit the mode — that is Viewer (or `P` / `Esc`). |
| 11 | **Help** calls `v.help.ShowManual()` (same as `F1` / Help → Manual). Disabled when `v.help.ManualOpen()`. About stays only on the Help menu. Settings stays only on File. Spiral is not in this menu. |
| 12 | **Every Window item shows its key** (same display-only `desktop.CustomShortcut` pattern as File → Open / Help → Manual). Viewer `V`, EXIF `E`, Grid `G`, Picture-frame `P`, Help `F1`. Modifier `0` on all five. `E`/`G`/`P`/`F1` keep their existing bindings. **`V` is new:** it calls `showViewer()` (leave grid or picture-frame; no-op in the image view). It is **not** a toggle. While the grid search bar is up, `V` must **not** steal the letter `v` from the query (`HandleRune` still types it; `HandleKey` ignores `KeyV`). |
| 13 | **Do not add Settings, About, or a second EXIF entry.** File → Settings… and Help → Manual / About stay. |

## Why not a new package

A Window menu is the app deciding how features compose (`ARCHITECTURE.md`). `help.Menu()` already returns a `*fyne.Menu` so `menu.go` can place it. Grid and slideshow must not import each other; the Viewer / Grid / Picture-frame mutual exclusion stays in `internal/ui`, next to `togglePictureFrameMode` and the `G` key guard.

## Global Constraints

- Do **not** `git commit`. `AGENTS.md` forbids it; suggested messages are for the user.
- Do **not** add `TODO`/`FIXME` comments to source. Open work stays in `todos.md`. Do **not** move Menu Window into Done until Florian accepts.
- Every user-visible string is `lang.L("English text")` with that exact key in `translations/en.json` **and** `translations/de.json`. Guard: `TestTranslations_EveryLocaleCoversEnglish` in `main_test.go`.
- Feature packages talk through their own `Host`. Do **not** import `internal/ui` from `help`, `exifwin`, `grid`, or `slideshow`. Do **not** pass `appState`. Do **not** add menu types to any `Host` interface.
- Cross-feature decisions stay in `internal/ui`. Menu enablement is window-geometry’s sibling in that package (`ARCHITECTURE.md`), not a feature package.
- Do **not** add mutable **package-level** test seams. Close/visibility callbacks are per-instance fields (what `AGENTS.md` permits).
- Tests: TDD. No `time.Sleep`. Use `newTestViewer` / `newTestUI`, `dropAndWait`, `waitUntilLoaded`, feature `Settle`, existing completion channels. Fyne’s test driver runs `fyne.Do` inline.
- **Do not call `ShowManual` / `F1` from `internal/ui` tests.** Rendering `manual.md` panics under Fyne’s test theme (bold inside a code span). Help grey-out is proven in `internal/ui/help` (`ManualOpen` + `SetOnManualClosed`). Viewer tests still assert the Help item’s label, `F1` hint, non-nil Action, and that `updateWindowMenuState` assigns `windowHelpItem.Disabled = v.help.ManualOpen()`.
- `gofmt -l -w` every touched file. Match CI before handoff: `gofmt`, `go vet ./...`, `go build ./...`, then focused tests; the parent runs `go test -race ./...` after the last code task.
- Work from `/Users/ronin/Projects/picfetch`. Golden screenshots are out of scope: **do not** run `make golden` and do not touch `internal/ui/testdata/`.
- English comments; match surrounding style. Verify identifiers against the files you open.
- Menu actions must **show/enter**, never `Toggle` off. See default 6.

## Subagent roster

Cursor’s Task tool does **not** offer Opus. Use `cursor-grok-4.6-xhigh` wherever this plan would have used Opus (judgment, easy-to-miss call sites, final review).

Run **strictly in order**. Do not dispatch implementers in parallel. Do not commit.

| Task | Role | `subagent_type` | Model | Why |
| --- | --- | --- | --- | --- |
| 1 | Implementer | `go-expert` | `cursor-grok-4.6-xhigh` | Four packages, close/visibility contracts, must not double-fire. |
| 1 | Reviewer | `generalPurpose` | `cursor-grok-4.5-high-fast` | API/layering check. |
| 2 | Implementer | `go-expert` | `composer-2.5-fast` | Structure + translations; follow File menu patterns. |
| 2 | Reviewer | `generalPurpose` | `cursor-grok-4.5-high-fast` | Labels/order/shortcuts. |
| 3 | Implementer | `go-expert` | `cursor-grok-4.6-xhigh` | Grey-out engine; easy to miss a Close path. |
| 3 | Reviewer | `generalPurpose` | `cursor-grok-4.6-xhigh` | State matrix. |
| 4 | Implementer | `go-expert` | `cursor-grok-4.6-xhigh` | Show-only actions vs Toggle is the bug this menu will ship if wrong. |
| 4 | Reviewer | `generalPurpose` | `cursor-grok-4.6-xhigh` | Same reason. |
| 5 | Implementer | `generalPurpose` | `composer-2.5-fast` | Manual EN/DE + `ARCHITECTURE.md`. |
| 5 | Reviewer | `generalPurpose` | `cursor-grok-4.5-high-fast` | Doc/spec check. |
| Final | Whole-branch review | `generalPurpose` | `cursor-grok-4.6-xhigh` | Cross-package + missed call sites. |
| Parent | Fix-up after each task | this session | inherit | Read the diff, re-run the task’s tests, fix, then review. |

## File map

| File | Change | Task |
| --- | --- | --- |
| `internal/ui/help/help.go` | `ManualOpen() bool`; `onManualClosed func()`; `SetOnManualClosed`. | 1 |
| `internal/ui/help/manual.go` | Existing ShowManual `onClosed` also runs `onManualClosed`. | 1 |
| `internal/ui/help/manual_search_test.go` (or `help_test.go`) | ManualOpen + close hook tests (stub `currentManual`). | 1 |
| `internal/ui/exifwin/exifwin.go` | `onClosed func()`; `SetOnClosed`; call from existing Singleton `onClosed`. | 1 |
| `internal/ui/exifwin/exifwin_test.go` | SetOnClosed fires on window close. | 1 |
| `internal/ui/grid/grid.go` | `onVisibility func()`; `SetOnVisibilityChanged`; fire from `Toggle` open path and `Close` (not the no-op Close). | 1 |
| `internal/ui/grid/grid_test.go` | Visibility callback counts. | 1 |
| `internal/ui/slideshow/slideshow.go` | `onActive func()`; `SetOnActiveChanged`; fire from `enter` and from `Exit` when it actually left. | 1 |
| `internal/ui/slideshow/slideshow_test.go` | Active callback counts. | 1 |
| `internal/ui/viewer.go` | Five `*fyne.MenuItem` fields (`windowViewerItem` … `windowHelpItem`). | 2 |
| `internal/ui/menu.go` | Build Window menu; assign item fields; display-only shortcuts on **all five** items; wire actions (filled in Task 4). | 2, 4 |
| `internal/ui/keys.go` | `KeyV` → `showViewer` (before the navigation-length guard, same as `P`/`G`). | 4 |
| `internal/ui/keys_test.go` | `V` leaves picture-frame. | 4 |
| `internal/ui/grid/nav.go` | `HandleKey` `KeyV` closes the grid unless `searching`. | 4 |
| `internal/ui/grid/nav_test.go` | `V` closes; `V` while searching does not. | 4 |
| `internal/ui/menu_test.go` | Structure: 4 top-level menus; Window item labels/order/shortcuts. | 2 |
| `translations/en.json`, `de.json` | New keys: `"Window"`, `"Viewer"`, `"Grid View"`, `"Picture-frame mode"`. | 2 |
| `internal/ui/windowmenu.go` | `updateWindowMenuState` / `applyWindowMenuState`; `showViewer`; show-only open helpers. | 3, 4 |
| `internal/ui/windowmenu_test.go` | Grey-out matrix + action tests. | 3, 4 |
| `internal/ui/features.go` or `build.go` | Wire the four observation callbacks to `updateWindowMenuState`. | 3 |
| `internal/ui/save.go` | `updateFileMenuState` also `applyWindowMenuState` before the existing `Refresh` (FileCount / DisplayedFile). | 3 |
| `ARCHITECTURE.md` | `menu.go` blurb: Window menu composition + grey-out. | 5 |
| `internal/ui/help/manual.md`, `manual_de.md` | §11 `V`, §12 Window menu, §18 quick reference. | 5 |

### Identifier lock (verify in the files you open)

As of this plan:

- `help.Help` has unexported `manualWin widgets.Singleton`. Add `ManualOpen() bool { return h.manualWin.Open() }`.
- `exifwin.Window` already has `Open() bool`. Do not add a second Open.
- `grid.Overview` already has `Visible()`, `Toggle()`, `Close()`.
- `slideshow.Controller` already has `Active()`, `Toggle()`, `Exit()`, unexported `enter()`.
- Viewer File items are `saveItem`, `exportItem`, `wallpaperItem`, `closeFilesItem`. Window items use the `window` prefix so they cannot be confused with `v.exif` / `v.help`.
- `viewer.DisplayedFile()` already exists (`viewer.go`).
- `togglePictureFrameMode` lives in `internal/ui/slideshow.go` (package `ui`), not in `internal/ui/slideshow`.

## Disabled matrix

`applyWindowMenuState` implements exactly this. Nil-guard every item pointer (construction order: `registerFeatures` before `buildMainMenu`).

| Item | `Disabled` when |
| --- | --- |
| Viewer | `!v.grid.Visible() && !v.slides.Active()` |
| EXIF Data | `v.exif.Open() \|\| !displayed` where `_, displayed := v.DisplayedFile()` |
| Grid View | `v.grid.Visible() \|\| v.FileCount() == 0 \|\| v.slides.Active()` |
| Picture-frame mode | `v.slides.Active() \|\| v.FileCount() == 0` |
| Help | `v.help.ManualOpen()` |

Fresh `newTestViewer` (no files): Viewer disabled, EXIF disabled, Grid disabled, Picture-frame disabled, Help enabled.

---

### Task 1: Observation APIs (close / visibility)

**Files:**
- Modify: `internal/ui/help/help.go`, `internal/ui/help/manual.go`
- Modify: `internal/ui/exifwin/exifwin.go`
- Modify: `internal/ui/grid/grid.go`
- Modify: `internal/ui/slideshow/slideshow.go`
- Test: `internal/ui/help/manual_search_test.go` (or new `internal/ui/help/window_test.go`)
- Test: `internal/ui/exifwin/exifwin_test.go`
- Test: `internal/ui/grid/grid_test.go`
- Test: `internal/ui/slideshow/slideshow_test.go`

**Interfaces:**
- Consumes: existing `widgets.Singleton.Open` / `Show` `onClosed`; existing `Overview.Toggle`/`Close`; existing `Controller.enter`/`Exit`.
- Produces:
  - `func (h *Help) ManualOpen() bool`
  - `func (h *Help) SetOnManualClosed(f func())`
  - `func (w *Window) SetOnClosed(f func())` (`exifwin`)
  - `func (g *Overview) SetOnVisibilityChanged(f func())`
  - `func (c *Controller) SetOnActiveChanged(f func())`
  Callbacks are stored on the receiver and **read at fire time**, not captured once at `Show`. A `Set*` after the window/mode is already up must still run on the next close/transition. `nil` means no hook. Do not panic on nil.

- [ ] **Step 1: Write the failing help tests**

Stub `currentManual` the way `TestShowManual_OpensSearchBarAndFocusesIt` already does (`"# Head\n\nplain text"`). Do not render `manual.md`.

```go
func TestHelp_ManualOpenAndOnClosed(t *testing.T) {
	h := New(test.NewApp(), "PicFetch", nil)
	orig := currentManual
	t.Cleanup(func() { currentManual = orig })
	currentManual = func() string { return "# Head\n\nplain text" }

	if h.ManualOpen() {
		t.Fatal("manual should start closed")
	}

	var closed int
	h.SetOnManualClosed(func() { closed++ })

	h.ShowManual()
	if !h.ManualOpen() {
		t.Fatal("ManualOpen should be true after ShowManual")
	}
	if closed != 0 {
		t.Fatalf("close hook fired early: %d", closed)
	}

	win := h.manualWin.Window()
	if win == nil {
		t.Fatal("expected an open manual window")
	}
	win.Close()

	if h.ManualOpen() {
		t.Error("ManualOpen should be false after close")
	}
	if closed != 1 {
		t.Errorf("close hook calls = %d, want 1", closed)
	}
}
```

Also cover: `SetOnManualClosed` **after** `ShowManual` still fires (field read at close time), and a second `ShowManual` after close opens again.

- [ ] **Step 2: Run the help test to verify it fails**

Run: `go test -count=1 -run TestHelp_ManualOpenAndOnClosed ./internal/ui/help/`

Expected: FAIL (`ManualOpen` / `SetOnManualClosed` undefined).

- [ ] **Step 3: Implement help observation**

On `Help`:

```go
onManualClosed func()

func (h *Help) ManualOpen() bool { return h.manualWin.Open() }

func (h *Help) SetOnManualClosed(f func()) { h.onManualClosed = f }
```

In `ShowManual`’s existing `onClosed` (the one that sets `h.manual = nil`), append:

```go
if h.onManualClosed != nil {
    h.onManualClosed()
}
```

Do not replace the existing teardown. Do not hook About.

- [ ] **Step 4: Write the failing EXIF close-hook test**

Follow an existing `exifwin` test that already `Show()`s a window with a real JPEG (any `TestShow_*` that leaves `w.Open()` true). Pattern:

```go
func TestWindow_SetOnClosedFiresWhenThePanelCloses(t *testing.T) {
	// same New + host + Show setup as an existing open-window test in this file
	var closed int
	w.SetOnClosed(func() { closed++ })
	w.Show()
	if !w.Open() {
		t.Fatal("premises: panel should be open")
	}
	w.Window().Close()
	if w.Open() {
		t.Error("panel should be closed")
	}
	if closed != 1 {
		t.Errorf("close hook calls = %d, want 1", closed)
	}
}
```

Use the file’s existing host fake and JPEG helper. If `Window()` is already exported (it is), use it. Do not break tile/warm teardown: the new hook runs **after** the existing Singleton `onClosed` body.

- [ ] **Step 5: Run the EXIF test to verify it fails**

Run: `go test -count=1 -run TestWindow_SetOnClosedFiresWhenThePanelCloses ./internal/ui/exifwin/`

Expected: FAIL (`SetOnClosed` undefined).

- [ ] **Step 6: Implement EXIF `SetOnClosed`**

Field `onClosed func()` on `exifwin.Window`. Setter assigns it. At the **end** of the existing `Show` `onClosed` closure (after `w.text = nil` and friends), call it if non-nil.

- [ ] **Step 7: Write the failing grid visibility tests**

Use `newOverview` from `harness_test.go` with at least one file so `Toggle` actually opens.

```go
func TestOverview_SetOnVisibilityChanged(t *testing.T) {
	g := newOverview(t, 1) // use the harness helper that already exists; adjust args to match the file
	var n int
	g.SetOnVisibilityChanged(func() { n++ })

	g.Toggle()
	if !g.Visible() || n != 1 {
		t.Fatalf("after open: visible=%v n=%d, want true/1", g.Visible(), n)
	}

	g.Close()
	if g.Visible() || n != 2 {
		t.Fatalf("after close: visible=%v n=%d, want false/2", g.Visible(), n)
	}

	g.Close()
	if n != 2 {
		t.Errorf("no-op Close fired the hook: n=%d", n)
	}
}
```

Also: `HandleKey` `G` while visible (and no selection) must fire the hook once via `Close`. `Toggle` while visible goes through `Close` — **one** fire, not two.

- [ ] **Step 8: Run the grid test to verify it fails**

Run: `go test -count=1 -run TestOverview_SetOnVisibilityChanged ./internal/ui/grid/`

Expected: FAIL (`SetOnVisibilityChanged` undefined).

- [ ] **Step 9: Implement grid visibility hook**

Field `onVisibility func()` on `Overview`.

```go
func (g *Overview) SetOnVisibilityChanged(f func()) { g.onVisibility = f }

func (g *Overview) fireVisibility() {
	if g.onVisibility != nil {
		g.onVisibility()
	}
}
```

Call `fireVisibility()` at the end of a successful **open** in `Toggle` (after `overlay.Show()` / `ForceRepaint`). Call it at the end of `Close` **after** `g.visible` is set false, and only on the path that actually closed (the early `if !g.visible { return }` stays hook-free).

`Toggle` when already visible only calls `Close()` and returns — do **not** fire again in `Toggle`.

- [ ] **Step 10: Write the failing slideshow active-hook tests**

Use `newController(t, 2)` from `slideshow_test.go`.

```go
func TestController_SetOnActiveChanged(t *testing.T) {
	c, _ := newController(t, 2)
	var n int
	c.SetOnActiveChanged(func() { n++ })

	c.Toggle()
	if !c.Active() || n != 1 {
		t.Fatalf("after enter: active=%v n=%d, want true/1", c.Active(), n)
	}

	c.Exit()
	if c.Active() || n != 2 {
		t.Fatalf("after exit: active=%v n=%d, want false/2", c.Active(), n)
	}

	c.Exit()
	if n != 2 {
		t.Errorf("no-op Exit fired the hook: n=%d", n)
	}

	c2, host := newController(t, 0)
	c2.SetOnActiveChanged(func() { n = 99 })
	c2.Toggle()
	if host.files != 0 {
		t.Fatal("sanity")
	}
	if c2.Active() || n == 99 {
		t.Error("Toggle with no files must not enter or fire")
	}
}
```

`Toggle` while active calls `Exit()` — one fire from `Exit`, not a second from `Toggle`.

- [ ] **Step 11: Run the slideshow test to verify it fails**

Run: `go test -count=1 -run TestController_SetOnActiveChanged ./internal/ui/slideshow/`

Expected: FAIL (`SetOnActiveChanged` undefined).

- [ ] **Step 12: Implement slideshow active hook**

Field `onActive func()` on `Controller`. Setter. Unexported `fireActive()` with a nil check.

Call `fireActive()` at the end of `enter` (after the run goroutine is started). Call it at the end of `Exit` only after the `if !c.active.Load() { return }` guard has been passed (the mode actually left).

`Toggle` must not call `fireActive` itself.

- [ ] **Step 13: Format and run Task 1 tests**

```bash
gofmt -l -w internal/ui/help/help.go internal/ui/help/manual.go internal/ui/exifwin/exifwin.go internal/ui/grid/grid.go internal/ui/slideshow/slideshow.go
gofmt -l -w internal/ui/help/*_test.go internal/ui/exifwin/exifwin_test.go internal/ui/grid/grid_test.go internal/ui/slideshow/slideshow_test.go
go test -count=1 ./internal/ui/help/ ./internal/ui/exifwin/ ./internal/ui/grid/ ./internal/ui/slideshow/
```

Expected: PASS. Do not run the whole module suite yet.

- [ ] **Step 14: Skip git commit.** Note for the parent: `feat: add window-surface observation hooks for the Window menu`.

---

### Task 2: Window menu structure, translations, item fields

**Files:**
- Modify: `internal/ui/menu.go`
- Modify: `internal/ui/viewer.go` (five `*fyne.MenuItem` fields next to `closeFilesItem`)
- Modify: `internal/ui/menu_test.go` (`TestBuildMainMenu_Structure` and a new accelerator test)
- Modify: `translations/en.json`, `translations/de.json`

**Interfaces:**
- Consumes: Task 1 APIs exist but this task does not wire callbacks yet. Actions may be empty funcs or already call the Task 4 helpers if those files exist — **prefer wiring labels/shortcuts/Disabled seeds here**, and set `Action` to a named `viewer` method stub that Task 4 fills (`showViewer`, `showWindowExif`, `showWindowGrid`, `showWindowPictureFrame`, `showWindowHelp`). If those methods do not exist yet, put `Action: func() {}` and a comment that Task 4 replaces them — **actually no**: define the five methods in `windowmenu.go` in this task as empty bodies (`// filled in Task 4` is a placeholder — forbidden). **This task creates `windowmenu.go` with method names and `panic("unreachable")` is also forbidden.**
- **Do this instead:** create `internal/ui/windowmenu.go` with the five methods as real one-liners that already call the underlying feature (`ShowManual`, `exif.Show`, etc.) even before Task 4’s guards. Task 4 tightens them to show-only. Empty Viewer can be `v.win.RequestFocus()` only until Task 4 adds Close/Exit.
- Produces: `buildMainMenu` returns File, Favorites, **Window**, Help. Fields on `viewer` listed below.

Viewer fields (place with the File menu item fields, same comment shape):

```go
windowViewerItem       *fyne.MenuItem
windowExifItem         *fyne.MenuItem
windowGridItem         *fyne.MenuItem
windowPictureFrameItem *fyne.MenuItem
windowHelpItem         *fyne.MenuItem
```

- [ ] **Step 1: Add translation keys**

`translations/en.json` (identity map):

```json
"Window": "Window",
"Viewer": "Viewer",
"Grid View": "Grid View",
"Picture-frame mode": "Picture-frame mode"
```

`translations/de.json`:

```json
"Window": "Fenster",
"Viewer": "Bildanzeige",
"Grid View": "Rasteransicht",
"Picture-frame mode": "Bilderrahmen-Modus"
```

Do **not** re-add `"Help"` or `"EXIF Data"`. Keep JSON valid (commas).

- [ ] **Step 2: Write the failing structure tests**

Update `TestBuildMainMenu_Structure`:

- `len(menu.Items) == 4`
- labels File, Favorites, **Window**, Help
- Window has **5** items, labels exactly: `Viewer`, `EXIF Data`, `Grid View`, `Picture-frame mode`, `Help`
- each has a non-nil `Action`
- none is a separator

New test `TestBuildMainMenu_WindowItemsDisplayTheirAccelerators`:

- Viewer: `*desktop.CustomShortcut{KeyName: fyne.KeyV, Modifier: 0}`
- EXIF: `*desktop.CustomShortcut{KeyName: fyne.KeyE, Modifier: 0}`
- Grid: `KeyG`, modifier 0
- Picture-frame: `KeyP`, modifier 0
- Help: `KeyF1`, modifier 0 (same as Help → Manual)

Update the File-menu count comment in `TestBuildMainMenu_Structure` only if you touch File — **do not change File’s 7 items**.

- [ ] **Step 3: Run structure tests to verify they fail**

Run: `go test -count=1 -run 'TestBuildMainMenu_Structure|TestBuildMainMenu_WindowItemsDisplayTheirAccelerators' ./internal/ui/`

Expected: FAIL (still 3 top-level menus, or Window missing).

- [ ] **Step 4: Implement `windowmenu.go` method stubs that compile**

```go
package ui

func (v *viewer) showViewer() {
	v.win.RequestFocus()
}

func (v *viewer) showWindowExif() {
	v.exif.Show()
}

func (v *viewer) showWindowGrid() {
	if !v.grid.Visible() && !v.slides.Active() {
		v.grid.Toggle()
	}
}

func (v *viewer) showWindowPictureFrame() {
	if !v.slides.Active() {
		v.togglePictureFrameMode()
	}
}

func (v *viewer) showWindowHelp() {
	v.help.ShowManual()
}
```

Task 4 will add Viewer Close/Exit, EXIF/Grid/PF no-file guards, and `updateWindowMenuState` calls. Shipping Toggle-on-open here is OK if the `Visible`/`Active` guards stay (that is already show-only for grid/PF).

- [ ] **Step 5: Implement `buildMainMenu` Window section**

After the File menu is built, before `return`:

```go
viewerItem := fyne.NewMenuItem(lang.L("Viewer"), view.showViewer)
viewerItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyV}
view.windowViewerItem = viewerItem

exifItem := fyne.NewMenuItem(lang.L("EXIF Data"), view.showWindowExif)
exifItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyE}
view.windowExifItem = exifItem

gridItem := fyne.NewMenuItem(lang.L("Grid View"), view.showWindowGrid)
gridItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyG}
view.windowGridItem = gridItem

pfItem := fyne.NewMenuItem(lang.L("Picture-frame mode"), view.showWindowPictureFrame)
pfItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyP}
view.windowPictureFrameItem = pfItem

helpItem := fyne.NewMenuItem(lang.L("Help"), view.showWindowHelp)
helpItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyF1}
view.windowHelpItem = helpItem

windowMenu := fyne.NewMenu(lang.L("Window"),
	viewerItem, exifItem, gridItem, pfItem, helpItem)

return fyne.NewMainMenu(fileMenu, view.favorites.Menu(), windowMenu, view.help.Menu())
```

Seed Disabled on the items that start grey on a fresh window (no files): Viewer `true`, EXIF `true`, Grid `true`, Picture-frame `true`, Help `false`. Task 3 will keep them in sync; the seeds stop a one-frame flash of enabled items.

Update the file comment at the top of `menu.go` (File, Favorites, Window, Help).

- [ ] **Step 6: Run Task 2 tests plus translation parity**

```bash
gofmt -l -w internal/ui/menu.go internal/ui/viewer.go internal/ui/windowmenu.go internal/ui/menu_test.go
go test -count=1 -run 'TestBuildMainMenu_|TestTranslations_EveryLocaleCoversEnglish' ./internal/ui/ ./
```

`TestTranslations_EveryLocaleCoversEnglish` lives in package main (`main_test.go`): `go test -count=1 -run TestTranslations_EveryLocaleCoversEnglish .`

Expected: PASS.

- [ ] **Step 7: Skip git commit.** Parent note: `feat: add Window menu structure and translations`.

---

### Task 3: Grey-out engine and callback wiring

**Files:**
- Modify: `internal/ui/windowmenu.go` — add `updateWindowMenuState` / `applyWindowMenuState`
- Modify: `internal/ui/save.go` — `updateFileMenuState` calls `applyWindowMenuState` before `MainMenu().Refresh()`
- Modify: `internal/ui/features.go` **or** `internal/ui/build.go` — wire Task 1 setters to `view.updateWindowMenuState` **after** `buildMainMenu` (items must exist). Prefer the end of `buildMainMenu` or immediately after `window.SetMainMenu(buildMainMenu(view))` in `build.go`.
- Test: `internal/ui/windowmenu_test.go` (new)
- Test: existing File-menu tests must still pass (`updateFileMenuState` still Refreshs once)

**Interfaces:**
- Consumes: Task 1 setters; Task 2 item fields; `DisplayedFile`, `FileCount`, `grid.Visible`, `slides.Active`, `exif.Open`, `help.ManualOpen`
- Produces: `func (v *viewer) applyWindowMenuState()` (no Refresh); `func (v *viewer) updateWindowMenuState()` (apply + Refresh)

`applyWindowMenuState` must no-op when `v.windowViewerItem == nil` (or check all five). Never nil-deref `v.win.MainMenu()`: if `MainMenu()` is nil, skip Refresh.

```go
func (v *viewer) applyWindowMenuState() {
	if v.windowViewerItem == nil {
		return
	}
	_, displayed := v.DisplayedFile()
	v.windowViewerItem.Disabled = !v.grid.Visible() && !v.slides.Active()
	v.windowExifItem.Disabled = v.exif.Open() || !displayed
	v.windowGridItem.Disabled = v.grid.Visible() || v.FileCount() == 0 || v.slides.Active()
	v.windowPictureFrameItem.Disabled = v.slides.Active() || v.FileCount() == 0
	v.windowHelpItem.Disabled = v.help.ManualOpen()
}

func (v *viewer) updateWindowMenuState() {
	v.applyWindowMenuState()
	if v.win != nil && v.win.MainMenu() != nil {
		v.win.MainMenu().Refresh()
	}
}
```

Wire (after menu exists):

```go
view.help.SetOnManualClosed(view.updateWindowMenuState)
view.exif.SetOnClosed(view.updateWindowMenuState)
view.grid.SetOnVisibilityChanged(view.updateWindowMenuState)
view.slides.SetOnActiveChanged(view.updateWindowMenuState)
```

`updateFileMenuState` already Refreshes. Change it to:

```go
func (v *viewer) updateFileMenuState() {
	v.saveItem.Disabled = !v.canSaveRotation()
	v.exportItem.Disabled = !v.canExport()
	v.wallpaperItem.Disabled = !v.canSetWallpaper()
	v.closeFilesItem.Disabled = v.FileCount() == 0
	v.favorites.SetHasFiles(v.FileCount() > 0)
	v.applyWindowMenuState()
	v.win.MainMenu().Refresh()
}
```

Call `updateWindowMenuState()` once at the end of `buildMainMenu` so seeds match the matrix even if later construction changes file state (it should not yet).

Also call `updateWindowMenuState` after `showWindowExif` / `showWindowHelp` in Task 4; for Task 3, opening EXIF via `v.exif.Show()` in a test must grey the item because `SetOnClosed` does not fire on **open**. **Opens are not close hooks.** So `showWindowExif` and keys `E` / F1 / info link must call `updateWindowMenuState` after Show.

For Task 3, add those calls on:

- `internal/ui/keys.go` after `v.help.ShowManual()`, `v.exif.Show()`, `v.grid.Toggle()` (the `G` branch when the grid is **not** already forwarding to `HandleKey`), and after picture-frame toggle / Escape `slides.Exit`
- `internal/ui/build.go` info-overlay EXIF link: wrap `view.exif.Show()` with `updateWindowMenuState`
- Grid open via `HandleKey` is covered by `SetOnVisibilityChanged`
- Picture-frame via `SetOnActiveChanged` covers `Toggle`/`Exit` — **then keys.go does not need a second call** for P/Escape if the slideshow hook is wired. Prefer the hook; do not double-Refresh unless it is cheaper to be redundant (Refresh is cheap; double is OK).

**Do not miss:** `drop.go` `grid.Close` (hook covers it), `RemoveFiles` empty-set `grid.Close` (hook), `clearToDropzone` `slides.Exit` (hook), `togglePictureFrameMode` (hook + grid Close hook).

`E` / `F1` / info link do **not** go through a visibility hook on open — they **must** call `updateWindowMenuState` after Show.

- [ ] **Step 1: Write the failing grey-out tests** in `windowmenu_test.go`

Use `newTestViewer` / `dropAndWait` / `waitUntilLoaded`. Helper:

```go
func windowMenu(v *viewer) *fyne.Menu {
	bar := v.win.MainMenu()
	if bar == nil || len(bar.Items) < 3 {
		return nil
	}
	return bar.Items[2] // File, Favorites, Window
}
```

Tests (each in its own `TestWindowMenu_*`):

1. **Fresh viewer:** Viewer/EXIF/Grid/PF disabled; Help enabled.
2. **After one JPEG drop:** Viewer disabled; EXIF, Grid, PF enabled; Help enabled.
3. **`v.exif.Show()` then Open:** EXIF disabled; close via `v.exif.Window().Close()` (or `v.exif`’s exported `Window()`); after close EXIF enabled again. Drain nothing extra unless the existing EXIF tests wait on warm — if `Show` starts a map warm, follow `exif_test.go` and do not expand the map (no GPS JPEG).
4. **`v.grid.Toggle()`:** Grid disabled, Viewer enabled; `Toggle` again: Grid enabled, Viewer disabled.
5. **`v.togglePictureFrameMode()`:** PF disabled, Viewer enabled, Grid disabled; toggle off: PF enabled, Viewer disabled, Grid enabled. `settleSlideshow` / existing slideshow settle helper if the file has one.
6. **`G` key while grid closed** (handleKeyEvent): same as (4). Use the same key-dispatch helper other `keys_test.go` tests use (`v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyG})` or the exported test path in that file).
7. **Picture-frame then Grid item stays disabled** (item.Disabled, do not rely on Action).
8. **`closeFiles`:** Grid/PF/EXIF disabled again; Viewer disabled.

Do **not** open the manual in these tests.

- [ ] **Step 2: Run grey-out tests to verify they fail**

Run: `go test -count=1 -run 'TestWindowMenu_' ./internal/ui/`

Expected: FAIL (`updateWindowMenuState` missing or items not tracking).

- [ ] **Step 3: Implement apply/update, wiring, key/link open updates, `updateFileMenuState` fold**

- [ ] **Step 4: Run Task 3 tests plus File menu enablement**

```bash
gofmt -l -w internal/ui/windowmenu.go internal/ui/windowmenu_test.go internal/ui/save.go internal/ui/features.go internal/ui/build.go internal/ui/keys.go
go test -count=1 -run 'TestWindowMenu_|TestBuildMainMenu_|TestCloseFilesItem_|TestSaveItem_|TestExportItem_' ./internal/ui/
```

Expected: PASS.

- [ ] **Step 5: Skip git commit.** Parent note: `feat: grey out Window menu items for the showing surface`.

---

### Task 4: Show-only actions and Viewer leave-mode

**Files:**
- Modify: `internal/ui/windowmenu.go` (`showViewer` and guards)
- Modify: `internal/ui/windowmenu_test.go` (action tests)
- Modify: `internal/ui/keys.go` (`case fyne.KeyV: v.showViewer()`)
- Modify: `internal/ui/keys_test.go` (`V` leaves picture-frame / no-ops in the image view)
- Modify: `internal/ui/grid/nav.go` (`KeyV` → `Close` unless `searching`)
- Modify: `internal/ui/grid/nav_test.go` (`V` closes; `V` while searching does not)
- Possibly `internal/ui/menu.go` if Actions are not already bound

**Interfaces:**
- Consumes: Task 3 `updateWindowMenuState`; `togglePictureFrameMode`; `grid.Toggle`/`Close`; `slides.Exit`
- Produces: the five `show*` methods with the contracts below; `fyne.KeyV` bound to `showViewer` in `keys.go` and `grid.HandleKey`

Final method bodies:

```go
func (v *viewer) showViewer() {
	v.grid.Close()
	if v.slides.Active() {
		v.slides.Exit()
		v.resetFade()
	}
	v.win.RequestFocus()
	v.updateWindowMenuState()
}

func (v *viewer) showWindowExif() {
	v.exif.Show()
	v.updateWindowMenuState()
}

func (v *viewer) showWindowGrid() {
	if v.grid.Visible() || v.slides.Active() || v.FileCount() == 0 {
		return
	}
	v.grid.Toggle()
	v.updateWindowMenuState()
}

func (v *viewer) showWindowPictureFrame() {
	if v.slides.Active() || v.FileCount() == 0 {
		return
	}
	v.togglePictureFrameMode()
	v.updateWindowMenuState()
}

func (v *viewer) showWindowHelp() {
	v.help.ShowManual()
	v.updateWindowMenuState()
}
```

`grid.Toggle` / `togglePictureFrameMode` already fire visibility/active hooks → `updateWindowMenuState`. Extra call at the end is redundant and required to be harmless.

- [ ] **Step 1: Write the failing action tests**

```go
func TestWindowMenu_ViewerLeavesGrid(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	v.grid.Toggle()
	if !v.grid.Visible() {
		t.Fatal("premises: grid up")
	}
	v.windowViewerItem.Action()
	if v.grid.Visible() {
		t.Error("Viewer should close the grid")
	}
	if !v.windowViewerItem.Disabled {
		t.Error("Viewer should grey once the image view is back")
	}
}

func TestWindowMenu_ViewerLeavesPictureFrame(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	v.togglePictureFrameMode()
	if !v.slides.Active() {
		t.Fatal("premises: picture-frame on")
	}
	v.windowViewerItem.Action()
	if v.slides.Active() {
		t.Error("Viewer should exit picture-frame mode")
	}
}

func TestWindowMenu_GridActionOpensAndDoesNotToggleOff(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	v.windowGridItem.Action()
	if !v.grid.Visible() {
		t.Fatal("Grid View should open the grid")
	}
	v.windowGridItem.Action() // even if Disabled, Action is callable from tests
	if !v.grid.Visible() {
		t.Error("Grid View must not toggle the grid closed")
	}
}

func TestWindowMenu_PictureFrameActionEntersAndDoesNotToggleOff(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	v.windowPictureFrameItem.Action()
	if !v.slides.Active() {
		t.Fatal("should enter picture-frame")
	}
	v.windowPictureFrameItem.Action()
	if !v.slides.Active() {
		t.Error("Picture-frame mode must not toggle off from the menu")
	}
}

func TestWindowMenu_GridActionNoopsWithoutFiles(t *testing.T) {
	v := newTestViewer(t)
	v.windowGridItem.Action()
	if v.grid.Visible() {
		t.Error("no files: grid must stay closed")
	}
}

func TestWindowMenu_ExifActionOpensWhenAFileIsDisplayed(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	v.windowExifItem.Action()
	if !v.exif.Open() {
		t.Error("EXIF Data should open the panel")
	}
	if !v.windowExifItem.Disabled {
		t.Error("EXIF item should grey while the panel is open")
	}
}

func TestWindowMenu_VKeyLeavesGrid(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	v.grid.Toggle()
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyV})
	if v.grid.Visible() {
		t.Error("V should leave the grid, same as Window -> Viewer")
	}
}

func TestWindowMenu_VKeyLeavesPictureFrame(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	v.togglePictureFrameMode()
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyV})
	if v.slides.Active() {
		t.Error("V should leave picture-frame mode, same as Window -> Viewer")
	}
}

func TestWindowMenu_GridActionNoopsDuringPictureFrame(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	v.togglePictureFrameMode()
	v.windowGridItem.Action()
	if v.grid.Visible() {
		t.Error("must not open the grid over picture-frame mode")
	}
}
```

Import `image/color` and `uitest` like `menu_test.go`. Use the package’s slideshow settle helper if `togglePictureFrameMode` tests elsewhere wait (see `slideshow_test.go` in `internal/ui`).

- [ ] **Step 2: Run action tests to verify they fail**

Run: `go test -count=1 -run 'TestWindowMenu_ViewerLeaves|TestWindowMenu_GridAction|TestWindowMenu_PictureFrameAction|TestWindowMenu_ExifAction' ./internal/ui/`

Expected: FAIL on Viewer not closing grid / Grid Action toggling off, depending on Task 2 stubs.

- [ ] **Step 3: Implement the final `show*` bodies**

- [ ] **Step 3b: Bind `V`**

In `handleKeyEvent`, in the same early switch as `P`/`G`/`E` (before the navigation-length guard):

```go
case fyne.KeyV:
	// Window -> Viewer: leave the grid or picture-frame; no-op in the
	// image view. Not a toggle. When the grid is up this case is not
	// reached — HandleKey owns V there (and ignores it while searching
	// so the letter v can still be typed).
	v.showViewer()
	return
```

When `v.grid.Visible()`, keys never hit that switch. In `grid.HandleKey`:

```go
case fyne.KeyV:
	if !g.searching {
		g.Close()
	}
```

Unlike `G`, `V` still closes while a selection is pending: it is “go to the image view”, not “toggle the grid”. `Close` already drops the selection.

Grid tests (pattern-match `nav_test.go`’s existing `G` cases): `V` closes a visible grid; with `/` search open, `V` leaves the grid up and does not clear the query (rune `v` may still be appended via `HandleRune` — assert Close did not run).

`keys.go` tests: `V` with picture-frame on calls `showViewer` (grid is not visible, so the main switch runs).

- [ ] **Step 4: Run all Window menu tests plus neighbors**

```bash
gofmt -l -w internal/ui/windowmenu.go internal/ui/windowmenu_test.go
go test -count=1 -run 'TestWindowMenu_|TestBuildMainMenu_|TestTogglePictureFrame|TestGrid' ./internal/ui/
```

If `-run TestGrid` is too broad, use `./internal/ui/` with `-count=1` for the whole ui package **without** `-race` first, then:

```bash
go test -count=1 -race -run 'TestWindowMenu_|TestBuildMainMenu_' ./internal/ui/
```

Expected: PASS.

- [ ] **Step 5: Skip git commit.** Parent note: `feat: Window menu opens surfaces and Viewer leaves overlay modes`.

---

### Task 5: Docs (ARCHITECTURE + manuals)

**Files:**
- Modify: `ARCHITECTURE.md` — `menu.go` row: Window menu items, grey-out via `updateWindowMenuState`, composition still in `internal/ui`.
- Modify: `internal/ui/help/manual.md` sections **11. Keyboard shortcuts**, **12. Menu**, and **18. Quick reference**
- Modify: `internal/ui/help/manual_de.md` matching sections
- Do **not** mark `todos.md` Menu Window as Done

**Interfaces:** none.

English manual, add after File / Favorites bullets in §12 (keep ASCII `->`):

```markdown
- **Window -> Viewer** (`V`) — shows the normal image view. Closes the
  grid or leaves picture-frame mode if either is up. Greyed out while you
  are already in that view. `V` is not a toggle. While the grid search
  (`/`) is open, typing `v` still goes into the query
- **Window -> EXIF Data** (`E`) — opens the EXIF panel for the image on
  screen, same as the info overlay's "Show EXIF data" link. Greyed out
  while that panel is already open, or when nothing is displayed
- **Window -> Grid View** (`G`) — opens the thumbnail overview. Greyed
  out while the grid is up, while picture-frame mode is on, or when no
  files are loaded. Closing the grid is Viewer / `V`, `G`, or `Esc` — not
  this item
- **Window -> Picture-frame mode** (`P`) — enters full-screen
  picture-frame mode. Greyed out while it is already on, or when no files
  are loaded. Leaving is Viewer / `V`, `P`, or `Esc`
- **Window -> Help** (`F1`) — opens this manual, same as Help -> Manual.
  Greyed out while the manual window is already open
```

Also add under §11 Keyboard shortcuts, after the `G` bullet:

```markdown
- **`V`** — return to the normal image view (closes the grid or leaves
  picture-frame mode). Not a toggle. While a grid search is open, types
  the letter `v` instead
```

German: **Fenster -> Bildanzeige / EXIF-Daten / Rasteransicht / Bilderrahmen-Modus / Hilfe**, same semantics. Match existing `manual_de.md` tone (`Sie`, **`G`**, ASCII `->`).

ARCHITECTURE `menu.go` cell: mention Window between Favorites and Help, grey-out, and that grid/slideshow still do not import each other.

- [ ] **Step 1: Edit the three doc files**
- [ ] **Step 2: No new `lang.L` keys.** Manual prose is not in `translations/*.json`.
- [ ] **Step 3:** `gofmt` N/A for md. Parent will run the full suite after this task.

- [ ] **Step 4: Skip git commit.** Parent note: `docs: document the Window menu`.

---

## Parent verification after Task 5

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
feat: add a Window menu for viewer, EXIF, grid, picture-frame, and help

Grey out the surface that is already showing. Menu items open or enter;
leaving the grid or picture-frame is Window -> Viewer (or V). G and P
still toggle.
```

## Out of scope

- **Menu Actions** (sort submenu, hide-duplicates checkmark, show variant)
- Settings / About / Spiral in the Window menu
- Changing `G` / `P` / `E` / `F1` **toggle/open** behavior (`V` is the only new key)
- Checkmarks instead of Disabled
- Golden screenshots

## Spec coverage (self-review)

| Requirement | Task |
| --- | --- |
| Viewer item | 2, 4 |
| EXIF information item | 2, 4 |
| Grid view item | 2, 4 |
| Picture-frame mode item | 2, 4 |
| Help item | 2 (structure), 1+3 (grey-out without opening full manual in ui tests) |
| Keyboard hints on every Window item (`V`/`E`/`G`/`P`/`F1`) | 2 |
| `V` key → `showViewer` (not a toggle; ignored while grid-searching) | 4 |
| Showing surfaces greyed out | 3 |
| Show-only, not toggle-off | 4 |
| Translations (`Bildanzeige` for Viewer) | 2 |
| Manual + ARCHITECTURE | 5 |
| Menu Actions | none (out of scope) |
