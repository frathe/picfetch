# Variant inspect loop Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
>
> **Controller extra (this session):** After every task, the parent agent reviews the diff and fixes it before dispatching the next task. Do not start Task N+1 until that review lands. Do not commit (`AGENTS.md`). End with a suggested commit message for the user. Do not dispatch Task 1 until Florian has confirmed the locked product decisions (or explicitly said to proceed with the defaults).

**Goal:** Committing a cell from the variants grid (Return or click) shows that file in the viewer — even when it is a hidden extra — then Left/Right loop only that duplicate group, and Escape walks back variants-grid → hide-duplicates grid.

**Architecture:** Add an inspect session on `grid.Overview` (`inspectKey` = URI string of the committed file). Return/click from browse begins inspect and closes the overlay without clearing it; `jumpIfHiddenExtra` no-ops while inspecting so a late hash apply cannot yank the viewer to the highest-resolution representative. Viewer navigation (`nextVisibleIndex` / Home / End) walks `InspectMembers()` with wrap. Viewer Escape, while inspecting and the grid is closed, reopens browse on the current file instead of `reset()`.

**Tech Stack:** Go, Fyne, existing `internal/ui/grid` browse/hide-duplicates, `internal/ui` `handleKeyEvent` / `StepImage`.

## Why this approach

Today `OnSelected` does `Close()` then `ShowImage(clicked)`. `Close()` ends browse. Hide-duplicates stays on. `IsHiddenExtra` is true for the clicked extra. `hashRemaining`'s UI apply then calls `jumpIfHiddenExtra` because `browseHost < 0` — that is the “Return shows the highest-resolution image” bug, even when hashes were already running when the user opened variants.

`StepImage` / Home / End key off `IsHiddenExtra`, so once the extra is on screen the next arrow leaves the group (skips to the next representative). Escape in the image view calls `reset()` and wipes the session.

The fix is a third mode, **inspect**, that exists only while the grid is closed after a browse commit:

```
hide-duplicates grid  --Shift+D-->  variants grid  --Return/click-->  inspect viewer (arrows loop group)
        ^                                ^                                      |
        +----- Esc (existing) -----------+----- Esc (new, not reset) -----------+
```

Alternatives rejected:

- **Keep `browseHost` set after Close.** `BrowsingDuplicates()` would mean “grid is filtered to a group” *and* “viewer is looping a group”. Menu checkmarks, `applyFilter`, and hash-apply early-returns would all lie while the overlay is hidden.
- **Copy member indices onto `viewer`.** Sort and delete invalidate host indices. The grid already owns `groupMembers`. Cross-feature glue stays in `internal/ui` (`reopenVariantGrid`, `nextVisibleIndex`); the flag lives on `Overview`.
- **Change `IsHiddenExtra` to false while inspecting.** Badges, hide filter, and “D jumps to representative” would all break. Skip only `jumpIfHiddenExtra` and the viewer walk.

## Locked product decisions

These are the defaults. Change them only if Florian says so before Task 1.

1. **Click = Return.** Plain click on a variants cell is the same commit as Return (`OnSelected` default branch). Modifier-click still selects without opening.
2. **Inspect starts only from browse.** Return/click from the hide-duplicates grid (not browsing) or the unfiltered grid does **not** begin inspect.
3. **Hide on or off.** Inspect works whenever the commit came from browse, including `Shift+D` with hide off. Arrows still loop only the group.
4. **Escape from inspect viewer** reopens the variants grid (browse on, highlight = current file). It does **not** `reset()` the session.
5. **Escape from the variants grid** stays the existing staged `escape()`: end browse, leave hide on, grid stays up. That is “back to normal grid”.
6. **G from inspect viewer** reopens the variants grid, same as Escape (`reopenVariantGrid`). It does **not** open the hide-duplicates overview.
7. **V from inspect viewer** is still a no-op in the image view. Inspect stays on.
8. **Home / End** while inspecting jump to the first / last *visible* file of the whole set (existing `firstVisibleIndex` / `lastVisibleIndex`, hide-duplicates skip). They do **not** wrap inside the group. Left/Right still loop the group.
9. **P is disabled** while `BrowsingDuplicates() || InspectingDuplicates()`. Key, Window → Picture-frame, and `togglePictureFrameMode` are no-ops. Do not enter picture-frame; do not ClearInspect via P.
10. **D (hide toggle) is disabled** while `BrowsingDuplicates() || InspectingDuplicates()`. Grid `HandleKey` plain `D`, viewer `D`, and Actions → Show/Hide duplicates are no-ops. `Shift+D` still toggles browse (it is not D). `jumpIfHiddenExtra` stays skipped while inspecting so a late hash apply cannot yank to the representative.
11. **No new `lang.L` keys.** Manual EN/DE wording only.
12. **No golden screenshot regeneration.**
13. **Out of scope:** Windows Ctrl+click; changing Hamming grouping; changing which file is the hide-duplicates representative.

## Open points (answered 2026-08-26)

1. **G vs Escape.** G also reopens variants (locked #6).
2. **Home / End.** Whole-set first/last visible (locked #8).
3. **D.** Disabled while browsing or inspecting variants (locked #10).
4. **P.** Disabled while browsing or inspecting variants (locked #9).

## Global Constraints

- Do not commit. `AGENTS.md`: “Do not run `git commit`. End with a suggested commit message for the user.”
- Do not add `TODO`/`FIXME` comments. Open work stays in `todos.md`.
- Every user-visible string is `lang.L("English text")` with the same key in every `translations/*.json` bundle. This feature adds none.
- Do not pass `appState` into `internal/ui/grid`. Inspect state lives on `Overview`.
- `g.ui.Do` stays inside the `g.decodes.Go` body that owns it.
- Drive grid tests with `Warm` / `Settle` / `parkDecodes`. Drive viewer tests with `dropAndWait` / `waitUntilLoaded` / `v.grid.Settle()`. Never `time.Sleep` to guess completion.
- Preserve `gofmt` / `goimports -local github.com/frathe/picfetch`. Tabs, not spaces.
- Subagents must not start Task N+1 themselves. They stop after their task’s verification and report.
- Do not change `imaging.DuplicateGroups` or the highest-resolution representative pick.
- `TestClose_ClearsBrowseLeavesHide` must stay green: G/Close still end browse. Inspect is a separate flag.
- `handleKeyEvent` Escape still cancels scan/sort and still leaves picture-frame before any inspect/reset branch.

## Subagent models

Use the least powerful listed model that can handle the role. Available slugs: `composer-2.5-fast`, `cursor-grok-4.5-high-fast`, `cursor-grok-4.6-xhigh`, `claude-opus-5-thinking-high`.

| Role | Model | Why |
|------|--------|-----|
| Task 1 implementer | `cursor-grok-4.6-xhigh` | Inspect flag + Close split + jump skip. Easy to clear inspect on the commit path or leave the hash-apply jump in. |
| Task 2 implementer | `cursor-grok-4.6-xhigh` | `OnSelected` / Return wiring; must not break modifier-click or filtered Return. |
| Task 3 implementer | `cursor-grok-4.6-xhigh` | Viewer walk + wrap; hide-dupes skip must keep working when inspect is off. |
| Task 4 implementer | `cursor-grok-4.6-xhigh` | Escape currently `reset()`s the session. Wrong branch order closes the window or wipes files. |
| Task 5 implementer | `cursor-grok-4.6-xhigh` | Delete / empty-set / G-from-viewer edges. |
| Task 6 implementer | `cursor-grok-4.5-high-fast` | Manual / ARCHITECTURE / todos copy. |
| Task reviewer (Task 6) | `cursor-grok-4.5-high-fast` | Mid-tier floor. |
| Task reviewer (Tasks 1–5) | `cursor-grok-4.6-xhigh` | Navigation-stack bugs are easy to miss. |
| Parent review / fix after each task | this session (do not dispatch) | Review and fix after every step. |
| Final whole-branch review | `claude-opus-5-thinking-high` | Cross-task: Return keeps the extra, arrows wrap the group, Escape/G reopen variants, D and P are inert while browsing or inspecting. |

Subagent type: `generalPurpose` for implementers and reviewers. Do not use `go-expert` to write the code (it is for design questions). Do not dispatch two implementers in parallel.

If Task 1 or 4 reports `BLOCKED` on a real design hole (not missing context), re-dispatch **that task only** with `claude-opus-5-thinking-high`.

## File structure

- Modify: `internal/ui/grid/grid.go` — `inspectKey` on `Overview`; `closeOverlay`; `OnSelected` commit (Task 2); `Toggle` open clears inspect
- Modify: `internal/ui/grid/dupes.go` — `BeginInspect` / `ClearInspect` / `InspectingDuplicates` / `InspectMembers` / `inspectSource`; `jumpIfHiddenExtra` and hash-apply skip
- Modify: `internal/ui/grid/selection.go` — `FilesChanged` retargets or clears inspect (Task 5)
- Test: `internal/ui/grid/dupes_test.go` — inspect flag, jump skip, Return from browse
- Modify: `internal/ui/viewer.go` — `nextVisibleIndex` / `firstVisibleIndex` / `lastVisibleIndex` / `randomVisibleOther` / `stepInMembers`
- Modify: `internal/ui/actionmenu.go` — `reopenVariantGrid` next to `browseCurrentDuplicates`
- Modify: `internal/ui/keys.go` — Escape inspect branch
- Modify: `internal/ui/viewer.go` `clearToDropzone` — `ClearInspect` (Task 5)
- Test: `internal/ui/step_test.go` — Return extra, arrow wrap; Home/End stay whole-set
- Test: `internal/ui/keys.go` coverage via `step_test.go` or `keys_test.go` — Escape stack
- Modify: `internal/ui/help/manual.md`, `internal/ui/help/manual_de.md`
- Modify: `ARCHITECTURE.md` — hide-duplicates / keys locator
- Modify: `todos.md` — pointer to this plan; do **not** move the item to Done

Do not add a new grid file. Do not regenerate goldens. Do not change `Host`.

## Current code the implementers must not break

`OnSelected` default branch today (Task 2 replaces only this arm):

```go
			i := g.fileIndex(id)

			g.Close()
			if i >= 0 {
				g.host.ShowImage(i)
			}
```

`jumpIfHiddenExtra` today (Task 1 adds an inspect guard at the top):

```go
func (g *Overview) jumpIfHiddenExtra() {
	if i := g.host.CurrentIndex(); g.IsHiddenExtra(i) {
		g.host.ShowImage(g.RepresentativeOf(i))
	}
}
```

`hashRemaining` UI apply today — when `browseHost < 0` and `hideDupes`, it jumps. After Task 1 the jump must also be skipped while inspecting:

```go
					if g.browseHost >= 0 {
						if remaining == 0 {
							g.finishBrowse()
							g.fireDupeState()
						}
						return
					}
					if g.hideDupes {
						keepHost := g.fileIndex(g.highlight)
						g.groupSizes, g.groupReps = sizes, reps
						g.applyVisibleFilter(false, keepHost)
						g.jumpIfHiddenExtra()
					}
```

`Close` today returns immediately when `!g.visible`, so `togglePictureFrameMode` → `v.grid.Close()` does **not** currently clear anything. Task 1’s `closeOverlay(true)` must `ClearInspect` even when the overlay is already hidden.

`handleKeyEvent` Escape today (Task 4 inserts inspect **after** slides/scan/sort and **before** empty-set / `reset()`):

```go
		if v.slides.Active() {
			v.slides.Exit()
			v.resetFade()
		} else if v.scanOp.active {
			v.cancelScan()
		} else if v.sortOp.active {
			v.cancelSort()
		} else if len(v.state.files) == 0 {
			v.win.Close()
		} else {
			v.reset()
		}
```

`nextVisibleIndex` today returns `from + delta` when hide is off (wrap is `ShowImage`’s job). Inspect must be checked **before** that early return, or hide-off browse commits would not loop.

---

### Task 1: Inspect session flag and skip `jumpIfHiddenExtra`

**Files:**
- Modify: `internal/ui/grid/grid.go` (`Overview` fields, `New`, `Close` → `closeOverlay`, `Toggle` open)
- Modify: `internal/ui/grid/dupes.go`
- Test: `internal/ui/grid/dupes_test.go`

**Interfaces:**
- Consumes: existing `groupMembers`, `IsHiddenExtra`, `RepresentativeOf`, `Close` body
- Produces (exported, same package):
  - `func (g *Overview) BeginInspect(hostIndex int)`
  - `func (g *Overview) ClearInspect()`
  - `func (g *Overview) InspectingDuplicates() bool`
  - `func (g *Overview) InspectMembers() []int` — host indices in host-index order, or nil
- Produces (unexported):
  - `inspectKey string` on `Overview` — URI `.String()` of the committed file; `""` means off
  - `func (g *Overview) inspectSource() int` — host index of `inspectKey`, or -1
  - `func (g *Overview) closeOverlay(clearInspect bool)`
- `Close()` becomes `g.closeOverlay(true)`
- `jumpIfHiddenExtra` returns immediately when `InspectingDuplicates()`
- `hashRemaining` UI apply: call `jumpIfHiddenExtra` only when `!g.InspectingDuplicates()` (the method itself also guards; both)
- `Toggle()` when **opening** (`!g.visible` path, after the FileCount==0 guard): `g.ClearInspect()`
- This task does **not** change `OnSelected`. Direct `BeginInspect` in tests is the red phase.

`BeginInspect` / `ClearInspect` / `inspectSource` / `InspectMembers`:

```go
func (g *Overview) BeginInspect(hostIndex int) {
	if hostIndex < 0 || hostIndex >= g.host.FileCount() {
		g.inspectKey = ""
		return
	}
	u := g.host.FileAt(hostIndex)
	if u == nil {
		g.inspectKey = ""
		return
	}
	g.inspectKey = u.String()
}

func (g *Overview) ClearInspect() {
	g.inspectKey = ""
}

func (g *Overview) InspectingDuplicates() bool {
	return g.inspectKey != ""
}

func (g *Overview) inspectSource() int {
	if g.inspectKey == "" {
		return -1
	}
	for i := 0; i < g.host.FileCount(); i++ {
		if g.host.FileAt(i).String() == g.inspectKey {
			return i
		}
	}
	return -1
}

func (g *Overview) InspectMembers() []int {
	src := g.inspectSource()
	if src < 0 {
		return nil
	}
	return g.groupMembers(src)
}
```

`closeOverlay` is the current `Close` body plus:

1. If `clearInspect`, `g.ClearInspect()` **first**, including when `!g.visible`.
2. If `!g.visible`: if `g.browseHost >= 0`, `g.SetBrowsingDuplicates(false)`; then return. (Drop / picture-frame / inspect-clear with the overlay already down.)
3. If visible: keep today’s body (selection, marquee, search, end browse, hide overlay, Unfocus, fireVisibility). Do **not** `ClearInspect` a second time in the visible body when `clearInspect` is false — that is the commit path Task 2 will use.

`New` does not need to set `inspectKey`; the zero value `""` is off. Document the field next to `browseHost`:

```go
	// inspectKey is the URI string of a file committed from the
	// variants grid, or "" when inspect is off. The viewer loops
	// InspectMembers and Escape reopens browse. Distinct from
	// browseHost: browse filters the overlay; inspect survives Close.
	inspectKey string
```

- [ ] **Step 1: Write the failing tests**

Append to `internal/ui/grid/dupes_test.go`:

```go
func TestBeginInspect_ReportsMembersAndSkipsJump(t *testing.T) {
	g, host := pairAndUnique(t)
	g.SetHideDuplicates(true)
	host.index = 1 // extra; equal-size pair, representative is 0

	if g.InspectingDuplicates() {
		t.Fatal("inspect should start off")
	}
	g.BeginInspect(1)
	if !g.InspectingDuplicates() {
		t.Fatal("BeginInspect(1) should turn inspect on")
	}
	got := g.InspectMembers()
	if len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Fatalf("InspectMembers() = %v, want [0 1]", got)
	}

	host.shown = nil
	g.jumpIfHiddenExtra()
	if len(host.shown) != 0 {
		t.Errorf("ShowImage calls = %v, want none while inspecting", host.shown)
	}
}

func TestJumpIfHiddenExtra_StillJumpsWhenNotInspecting(t *testing.T) {
	g, host := pairAndUnique(t)
	g.SetHideDuplicates(true)
	host.index = 1
	host.shown = nil

	g.jumpIfHiddenExtra()
	if len(host.shown) != 1 || host.shown[0] != 0 {
		t.Errorf("ShowImage calls = %v, want jump to representative 0", host.shown)
	}
}

func TestClose_ClearsInspectEvenWhenAlreadyHidden(t *testing.T) {
	g, _ := pairAndUnique(t)
	g.BeginInspect(1)
	if g.Visible() {
		g.Close()
	}
	if g.Visible() {
		t.Fatal("premises: grid closed")
	}
	g.BeginInspect(1)
	g.Close()
	if g.InspectingDuplicates() {
		t.Fatal("Close must clear inspect while the overlay is already hidden")
	}
}

func TestToggleOpen_ClearsInspect(t *testing.T) {
	g, _ := pairAndUnique(t)
	if g.Visible() {
		g.Close()
	}
	g.BeginInspect(1)
	g.Toggle()
	if !g.Visible() {
		t.Fatal("Toggle should open")
	}
	if g.InspectingDuplicates() {
		t.Fatal("opening the grid with G must end inspect")
	}
}

func TestClearInspect_StopsSkippingJump(t *testing.T) {
	g, host := pairAndUnique(t)
	g.SetHideDuplicates(true)
	host.index = 1
	g.BeginInspect(1)
	g.ClearInspect()
	host.shown = nil
	g.jumpIfHiddenExtra()
	if len(host.shown) != 1 || host.shown[0] != 0 {
		t.Errorf("after ClearInspect, ShowImage calls = %v, want [0]", host.shown)
	}
}
```

`pairAndUnique` already Warms and Toggles (grid visible, equal-size seed-1 pair + unique). Representative of the pair is index 0.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -count=1 -run 'TestBeginInspect_ReportsMembersAndSkipsJump|TestJumpIfHiddenExtra_StillJumpsWhenNotInspecting|TestClose_ClearsInspectEvenWhenAlreadyHidden|TestToggleOpen_ClearsInspect|TestClearInspect_StopsSkippingJump' ./internal/ui/grid/`

Expected: FAIL (`BeginInspect` undefined).

- [ ] **Step 3: Implement inspectKey, closeOverlay, jump skip, Toggle-open ClearInspect**

Keep `TestClose_ClearsBrowseLeavesHide` passing: visible `Close` still ends browse. `closeOverlay(false)` is unused until Task 2; still implement it so Task 2 only wires `OnSelected`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -count=1 ./internal/ui/grid/`

Expected: PASS (full grid package).

- [ ] **Step 5: Do not commit**

Report files changed and the test command output.

---

### Task 2: Return/click from browse begins inspect and does not jump

**Files:**
- Modify: `internal/ui/grid/grid.go` (`OnSelected` default branch only)
- Test: `internal/ui/grid/dupes_test.go`
- Test: `internal/ui/grid/nav_test.go` only if Task 2’s Return tests belong next to `TestHandleKey_ReturnOpensHighlightedAndCloses` — prefer `dupes_test.go` so browse fixtures stay together

**Interfaces:**
- Consumes: `BeginInspect`, `closeOverlay(false)`, `Close`, `BrowsingDuplicates`, `groupSize`, `fileIndex`
- Produces: committing a cell while `BrowsingDuplicates()` and `groupSize(i) >= 2` calls `BeginInspect(i)` then `closeOverlay(false)` then `ShowImage(i)`
- Modifier-click / Shift-click unchanged
- Return from hide-duplicates (not browsing) still `Close()` (clears inspect) then `ShowImage`
- After commit: `InspectingDuplicates()==true`, `BrowsingDuplicates()==false`, `Visible()==false`, last `ShowImage` is the clicked host index (the extra, not the representative)

`OnSelected` default branch:

```go
		default:
			i := g.fileIndex(id)
			if g.BrowsingDuplicates() && i >= 0 && g.groupSize(i) >= 2 {
				g.BeginInspect(i)
				g.closeOverlay(false)
			} else {
				g.Close()
			}
			if i >= 0 {
				g.host.ShowImage(i)
			}
```

Resolve `i` **before** closing, same comment as today.

- [ ] **Step 1: Write the failing tests**

`click` lives in `selection_test.go` (same package). `setHighlight` is unexported in package `grid` — tests in this package may call it. Do not press Escape; that would leave browse.

```go
func TestHandleKey_ReturnFromBrowseCommitsExtra(t *testing.T) {
	g, host := pairAndUnique(t)
	g.SetHideDuplicates(true)
	g.SetBrowsingDuplicates(true)
	if g.displayIndexOf(1) < 0 {
		t.Fatal("extra should be visible while browsing")
	}
	g.setHighlight(g.displayIndexOf(1))
	host.shown = nil
	host.index = 1 // what the viewer would set after ShowImage; jump uses CurrentIndex

	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyReturn})

	if g.Visible() {
		t.Fatal("Return should close the grid")
	}
	if g.BrowsingDuplicates() {
		t.Fatal("Return should end browse")
	}
	if !g.InspectingDuplicates() {
		t.Fatal("Return from browse should begin inspect")
	}
	if len(host.shown) != 1 || host.shown[0] != 1 {
		t.Fatalf("ShowImage calls = %v, want [1] (the extra, not representative 0)", host.shown)
	}

	g.jumpIfHiddenExtra()
	if len(host.shown) != 1 || host.shown[0] != 1 {
		t.Fatalf("after jumpIfHiddenExtra ShowImage = %v, want still [1]", host.shown)
	}
}

func TestOnSelected_ClickFromBrowseCommitsExtra(t *testing.T) {
	g, host := pairAndUnique(t)
	g.SetHideDuplicates(true)
	g.SetBrowsingDuplicates(true)
	host.shown = nil
	host.index = 1
	click(g, host, g.displayIndexOf(1), 0)

	if !g.InspectingDuplicates() || g.Visible() || g.BrowsingDuplicates() {
		t.Fatalf("inspect=%v visible=%v browse=%v", g.InspectingDuplicates(), g.Visible(), g.BrowsingDuplicates())
	}
	if len(host.shown) != 1 || host.shown[0] != 1 {
		t.Fatalf("ShowImage = %v, want [1]", host.shown)
	}
}

func TestHandleKey_ReturnFromHideGridDoesNotInspect(t *testing.T) {
	g, host := pairAndUnique(t)
	g.SetHideDuplicates(true)
	if g.BrowsingDuplicates() {
		t.Fatal("premises: not browsing")
	}
	g.setHighlight(0)
	host.shown = nil
	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if g.InspectingDuplicates() {
		t.Fatal("Return from hide-duplicates (not browse) must not begin inspect")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -count=1 -run 'TestHandleKey_ReturnFromBrowseCommitsExtra|TestOnSelected_ClickFromBrowseCommitsExtra|TestHandleKey_ReturnFromHideGridDoesNotInspect' ./internal/ui/grid/`

Expected: FAIL (`InspectingDuplicates` false after Return — still `Close()` which clears inspect, and/or `jumpIfHiddenExtra` not in this path yet; the inspect assertion is the red).

- [ ] **Step 3: Change only the `OnSelected` default branch**

- [ ] **Step 4: Run tests to verify they pass**

Run:

```
go test -count=1 ./internal/ui/grid/
```

Expected: PASS, including `TestOnSelected_PlainClickStillOpensTheImage`, `TestOnSelected_ModifierClickSelectsWithoutOpening`, `TestHandleKey_ReturnOpensHighlightedAndCloses`, `TestClose_ClearsBrowseLeavesHide`.

- [ ] **Step 5: Do not commit**

Report files changed and the test command output.

---

### Task 3: Viewer arrows loop inspect members

**Files:**
- Modify: `internal/ui/viewer.go` (`nextVisibleIndex`; add `stepInMembers`. Do not change `firstVisibleIndex` / `lastVisibleIndex` / `randomVisibleOther`)
- Test: `internal/ui/step_test.go`

**Interfaces:**
- Consumes: `v.grid.InspectMembers() []int`, `v.grid.InspectingDuplicates()`
- Produces: `func stepInMembers(members []int, from, delta int) int`
  - `members` is non-empty, sorted host indices
  - `delta == 0` returns `from`
  - If `from` is not in `members`, treat its position as `0` then apply `delta`
  - Wrap with `(pos+step+n)%n` per unit of `delta` (same shape as hide-dupes skip)
- `nextVisibleIndex`: if `len(InspectMembers()) >= 2` and `delta != 0`, return `stepInMembers(InspectMembers(), from, delta)` **before** the hide-duplicates early return
- Do **not** change `firstVisibleIndex`, `lastVisibleIndex`, or `randomVisibleOther`. Home/End stay whole-set (hide-duplicates skip). Picture-frame shuffle is disabled while inspecting in Task 5, so `randomVisibleOther` is not on the inspect path.
- Hide-duplicates skip path unchanged when inspect is off
- `TestStepImage_SkipsHiddenExtras` and `TestStepImage_HideDuplicatesShowsHighestResolution` stay green (they never Return from browse)

`stepInMembers`:

```go
func stepInMembers(members []int, from, delta int) int {
	n := len(members)
	if n == 0 {
		return from
	}
	if delta == 0 {
		return from
	}
	pos := 0
	found := false
	for i, m := range members {
		if m == from {
			pos = i
			found = true
			break
		}
	}
	if !found {
		pos = 0
	}
	step := 1
	if delta < 0 {
		step = -1
	}
	for k := 0; k < absInt(delta); k++ {
		pos = (pos + step + n) % n
	}
	return members[pos]
}
```

Put it next to `absInt` in `viewer.go`.

`nextVisibleIndex` starts:

```go
func (v *viewer) nextVisibleIndex(from, delta int) int {
	n := len(v.state.files)
	if n == 0 {
		return 0
	}
	if members := v.grid.InspectMembers(); len(members) >= 2 && delta != 0 {
		return stepInMembers(members, from, delta)
	}
	if !v.grid.HideDuplicates() || delta == 0 {
		return from + delta
	}
	// existing skip loop unchanged
```

- [ ] **Step 1: Write the failing tests**

Add a helper and tests in `internal/ui/step_test.go`:

```go
func loadBrowsePair(t *testing.T) *viewer {
	t.Helper()
	v := loadPatternedTriple(t)
	v.grid.SetHideDuplicates(true)
	v.grid.Settle()
	waitUntilLoaded(t, v)
	v.grid.Toggle()
	v.grid.SetBrowsingDuplicates(true)
	v.grid.Settle()
	if !v.grid.Visible() || !v.grid.BrowsingDuplicates() {
		t.Fatal("premises: variants grid up")
	}
	return v
}

func TestStepInMembers_Wraps(t *testing.T) {
	members := []int{0, 3, 5}
	if got := stepInMembers(members, 3, 1); got != 5 {
		t.Errorf("step +1 from 3 = %d, want 5", got)
	}
	if got := stepInMembers(members, 5, 1); got != 0 {
		t.Errorf("wrap +1 from 5 = %d, want 0", got)
	}
	if got := stepInMembers(members, 0, -1); got != 5 {
		t.Errorf("wrap -1 from 0 = %d, want 5", got)
	}
	if got := stepInMembers(members, 99, 1); got != 3 {
		t.Errorf("from missing, +1 from pos 0 = %d, want 3", got)
	}
}

func TestStepImage_InspectLoopsVariantsNotUniques(t *testing.T) {
	v := loadBrowsePair(t)
	// finishBrowse highlights browseHost (current, index 0). One Right is the extra.
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
	waitUntilLoaded(t, v)

	if v.state.index != 1 {
		t.Fatalf("index = %d, want 1 (committed extra)", v.state.index)
	}
	if !v.grid.InspectingDuplicates() {
		t.Fatal("inspect should be on after Return from browse")
	}

	v.StepImage(1)
	waitUntilLoaded(t, v)
	if v.state.index != 0 {
		t.Fatalf("after StepImage(1) index = %d, want 0 (other variant, not unique 2)", v.state.index)
	}

	v.StepImage(1)
	waitUntilLoaded(t, v)
	if v.state.index != 1 {
		t.Fatalf("after wrap index = %d, want 1", v.state.index)
	}

	v.StepImage(-1)
	waitUntilLoaded(t, v)
	if v.state.index != 0 {
		t.Fatalf("after StepImage(-1) index = %d, want 0", v.state.index)
	}
}

func TestHandleKeyEvent_HomeEndWhileInspectingUseWholeSet(t *testing.T) {
	v := loadBrowsePair(t)
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
	waitUntilLoaded(t, v)

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEnd})
	waitUntilLoaded(t, v)
	if v.state.index != 2 {
		t.Fatalf("End index = %d, want 2 (last visible of the set, unique)", v.state.index)
	}
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyHome})
	waitUntilLoaded(t, v)
	if v.state.index != 0 {
		t.Fatalf("Home index = %d, want 0 (first visible representative)", v.state.index)
	}
}
```

Package `ui` tests must not call unexported `setHighlight` / `displayIndexOf`. `loadBrowsePair` only uses exported grid methods (`SetHideDuplicates`, `Toggle`, `SetBrowsingDuplicates`, `Settle`). Keep `stepInMembers` tests in package `ui` (same file as `stepInMembers`).

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -count=1 -run 'TestStepInMembers_Wraps|TestStepImage_InspectLoopsVariantsNotUniques|TestHandleKeyEvent_HomeEndWhileInspectingUseWholeSet' ./internal/ui/`

Expected: FAIL (`stepInMembers` undefined, and/or StepImage from extra 1 goes to unique 2).

- [ ] **Step 3: Implement `stepInMembers` and the four navigation helpers**

Do not change Escape.

- [ ] **Step 4: Run tests to verify they pass**

Run:

```
go test -count=1 -run 'TestStepImage_|TestHandleKeyEvent_|TestStepInMembers_' ./internal/ui/
go test -count=1 ./internal/ui/grid/
```

Expected: PASS.

- [ ] **Step 5: Do not commit**

Report files changed and the test command output.

---

### Task 4: Escape from inspect viewer reopens the variants grid

**Files:**
- Modify: `internal/ui/actionmenu.go` — `reopenVariantGrid` immediately after `browseCurrentDuplicates`
- Modify: `internal/ui/keys.go` — Escape inspect branch
- Test: `internal/ui/step_test.go`

**Interfaces:**
- Consumes: `InspectingDuplicates`, `ClearInspect`, `SetBrowsingDuplicates`, `Toggle`, `BrowsingDuplicates`, `Visible`
- Produces: `func (v *viewer) reopenVariantGrid()`
- Escape order (grid already not visible — `HandleKey` owns Escape while it is):
  1. picture-frame Exit
  2. cancel scan
  3. cancel sort
  4. **new:** if `v.grid.InspectingDuplicates()`, `reopenVariantGrid()` and return
  5. empty set → `win.Close()`
  6. `reset()`
- `reopenVariantGrid`:
  - no-op if `v.slides.Active()` (defensive; Escape already exited picture-frame first)
  - `v.grid.ClearInspect()` **first** (so a dissolved group cannot trap Escape in a loop)
  - `v.grid.SetBrowsingDuplicates(true)` (source = `CurrentIndex()` because the grid is closed)
  - if `v.grid.BrowsingDuplicates() && !v.grid.Visible()`, `v.grid.Toggle()`
  - `v.updateWindowMenuState()`
  - if browse did not stick (unique / unhashed), do **not** `reset()` and do **not** open the hide-duplicates grid; stay on the current image
- After a successful reopen: grid visible, browsing, inspect off, hide still on, `state.files` unchanged (not reset)
- Second Escape is existing `escape()`: end browse, hide stays, grid stays

```go
func (v *viewer) reopenVariantGrid() {
	if v.slides.Active() {
		return
	}
	v.grid.ClearInspect()
	v.grid.SetBrowsingDuplicates(true)
	if v.grid.BrowsingDuplicates() && !v.grid.Visible() {
		v.grid.Toggle()
	}
	v.updateWindowMenuState()
}
```

Put `reopenVariantGrid` in `actionmenu.go` immediately after `browseCurrentDuplicates` (same “open browse + maybe Toggle” shape). Do not add it to `keys.go` beyond the Escape call.

keys.go Escape:

```go
		if v.slides.Active() {
			v.slides.Exit()
			v.resetFade()
		} else if v.scanOp.active {
			v.cancelScan()
		} else if v.sortOp.active {
			v.cancelSort()
		} else if v.grid.InspectingDuplicates() {
			v.reopenVariantGrid()
		} else if len(v.state.files) == 0 {
			v.win.Close()
		} else {
			v.reset()
		}
```

- [ ] **Step 1: Write the failing tests**

```go
func TestHandleKeyEvent_EscapeFromInspectReopensVariantsThenHideGrid(t *testing.T) {
	v := loadBrowsePair(t)
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
	waitUntilLoaded(t, v)
	if v.grid.Visible() || !v.grid.InspectingDuplicates() {
		t.Fatal("premises: inspect viewer, grid closed")
	}
	startFiles := len(v.state.files)

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
	v.grid.Settle()

	if len(v.state.files) != startFiles {
		t.Fatalf("files = %d, want %d (Escape must not reset the session)", len(v.state.files), startFiles)
	}
	if !v.grid.Visible() {
		t.Fatal("Escape from inspect should reopen the grid")
	}
	if !v.grid.BrowsingDuplicates() {
		t.Fatal("reopened grid should be the variants (browse) filter")
	}
	if v.grid.InspectingDuplicates() {
		t.Fatal("inspect ends when the variants grid is back")
	}
	if !v.grid.HideDuplicates() {
		t.Fatal("hide should still be on")
	}

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if v.grid.BrowsingDuplicates() {
		t.Fatal("second Escape should leave browse")
	}
	if !v.grid.Visible() {
		t.Fatal("second Escape should stay on the hide-duplicates grid")
	}
	if !v.grid.HideDuplicates() {
		t.Fatal("second Escape must not turn hide off")
	}
}

func TestHandleKeyEvent_EscapeWithoutInspectStillResets(t *testing.T) {
	v := loadPatternedTriple(t)
	if v.grid.InspectingDuplicates() || v.grid.Visible() {
		t.Fatal("premises: image view, not inspecting")
	}
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if len(v.state.files) != 0 {
		t.Fatal("Escape in the image view without inspect should reset the session")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -count=1 -run 'TestHandleKeyEvent_EscapeFromInspectReopensVariantsThenHideGrid|TestHandleKeyEvent_EscapeWithoutInspectStillResets' ./internal/ui/`

Expected: FAIL (first Escape resets `files` to 0).

- [ ] **Step 3: Implement `reopenVariantGrid` and the Escape branch**

Do not change `escape()` in `nav.go`. The second Escape is already correct.

- [ ] **Step 4: Run tests to verify they pass**

Run:

```
go test -count=1 -run 'TestHandleKeyEvent_|TestStepImage_' ./internal/ui/
go test -count=1 ./internal/ui/grid/
```

Expected: PASS, including `TestHandleKeyEvent_IgnoredWhileAFyneDialogIsUp` (dialog guard still first).

- [ ] **Step 5: Do not commit**

Report files changed and the test command output.

---

### Task 5: G reopens variants; disable D/P; delete/drop

**Files:**
- Modify: `internal/ui/keys.go` — G from inspect calls `reopenVariantGrid`; D no-op while inspect; P no-op while inspect
- Modify: `internal/ui/grid/nav.go` — plain `D` no-op while `BrowsingDuplicates()` (Shift+D still toggles browse)
- Modify: `internal/ui/actionmenu.go` / `internal/ui/windowmenu.go` — disable Show/Hide duplicates and Picture-frame while `BrowsingDuplicates() || InspectingDuplicates()`
- Modify: `internal/ui/grid/selection.go` — `FilesChanged`
- Modify: `internal/ui/viewer.go` — `clearToDropzone` calls `v.grid.ClearInspect()`
- Test: `internal/ui/step_test.go`
- Test: `internal/ui/grid/dupes_test.go` (FilesChanged, D while browse)
- Test: `internal/ui/actionmenu_test.go` if menu Disabled needs a lock

**Interfaces:**
- Consumes: `InspectingDuplicates`, `BrowsingDuplicates`, `reopenVariantGrid` (Task 4)
- Produces: G from inspect ≡ Escape from inspect (`reopenVariantGrid`)
- Produces: plain D ignored while browsing or inspecting; hide flag unchanged; index unchanged
- Produces: P ignored while browsing or inspecting; `slides.Active()` stays false; inspect unchanged
- Produces: `FilesChanged` retargets inspect when the inspect URI left the set or its group shrank below 2
- `clearToDropzone` always `ClearInspect`
- `showWindowGrid` while inspecting also `reopenVariantGrid` (same as G)
- `showWindowPictureFrame` no-op while browsing or inspecting
- Do **not** change deletion’s confirm UI. `RemoveFiles` already calls `FilesChanged`
- Shift+D is **not** disabled

Helper used by keys and menus (put next to `reopenVariantGrid` in `actionmenu.go`):

```go
func (v *viewer) variantsSession() bool {
	return v.grid.BrowsingDuplicates() || v.grid.InspectingDuplicates()
}
```

keys.go `KeyG` (still before the navigation-length guard):

```go
	case fyne.KeyG:
		if v.slides.Active() {
			return
		}
		if v.grid.InspectingDuplicates() {
			v.reopenVariantGrid()
			v.updateWindowMenuState()
			return
		}
		v.grid.Toggle()
		v.updateWindowMenuState()
		return
```

keys.go `KeyD` after the Shift+D branch, before `toggleHideDuplicates`:

```go
		if v.keyModifiers()&fyne.KeyModifierShift != 0 {
			v.browseCurrentDuplicates()
			return
		}
		if v.grid.InspectingDuplicates() {
			return
		}
		v.toggleHideDuplicates()
		return
```

keys.go `KeyP`: if `v.grid.InspectingDuplicates()`, return before `togglePictureFrameMode`. (Grid-visible P never reaches this switch.)

nav.go `KeyD`:

```go
	case fyne.KeyD:
		if g.host.Modifiers()&fyne.KeyModifierShift != 0 {
			g.ToggleBrowseDuplicates()
		} else if !g.BrowsingDuplicates() {
			g.SetHideDuplicates(!g.hideDupes)
		}
```

`FilesChanged`:

```go
func (g *Overview) FilesChanged() {
	g.sel.Clear()
	if g.inspectKey != "" {
		src := g.inspectSource()
		if src < 0 || len(g.groupMembers(src)) < 2 {
			cur := g.host.CurrentIndex()
			if len(g.groupMembers(cur)) >= 2 {
				g.BeginInspect(cur)
			} else {
				g.ClearInspect()
			}
		}
	}
	g.applyFilter()
}
```

`clearToDropzone`: `v.grid.ClearInspect()` once (start or after `clearFiles`). Do not open or close the overlay here beyond what already happens.

Menu: `actionsHideItem.Disabled` includes `v.variantsSession()`. `windowPictureFrameItem.Disabled` includes `v.variantsSession()`.

- [ ] **Step 1: Write the failing tests**

```go
func TestHandleKeyEvent_GFromInspectReopensVariants(t *testing.T) {
	v := loadBrowsePair(t)
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
	waitUntilLoaded(t, v)

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyG})
	v.grid.Settle()
	if !v.grid.Visible() || !v.grid.BrowsingDuplicates() {
		t.Fatal("G from inspect should reopen the variants grid")
	}
	if v.grid.InspectingDuplicates() {
		t.Fatal("inspect ends when variants reopen")
	}
}

func TestHandleKeyEvent_DNoopWhileInspecting(t *testing.T) {
	v := loadBrowsePair(t)
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
	waitUntilLoaded(t, v)
	if !v.grid.HideDuplicates() {
		t.Fatal("premises: hide on")
	}
	idx := v.state.index
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyD})
	if !v.grid.HideDuplicates() {
		t.Fatal("D while inspecting must not toggle hide")
	}
	if v.state.index != idx {
		t.Fatalf("index = %d, want %d (D must not jump)", v.state.index, idx)
	}
}

func TestHandleKeyEvent_PNoopWhileInspecting(t *testing.T) {
	v := loadBrowsePair(t)
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
	waitUntilLoaded(t, v)
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyP})
	if v.slides.Active() {
		t.Fatal("P while inspecting must not enter picture-frame")
	}
	if !v.grid.InspectingDuplicates() {
		t.Fatal("P must leave inspect on")
	}
}

func TestHandleKey_DNoopWhileBrowsing(t *testing.T) {
	g, _ := pairAndUnique(t)
	g.SetHideDuplicates(true)
	g.SetBrowsingDuplicates(true)
	g.HandleKey(&fyne.KeyEvent{Name: fyne.KeyD})
	if !g.HideDuplicates() || !g.BrowsingDuplicates() {
		t.Fatal("plain D while browsing must not toggle hide or leave browse")
	}
}

func TestFilesChanged_ClearsInspectWhenGroupDissolves(t *testing.T) {
	g, host := pairAndUnique(t)
	g.SetHideDuplicates(true)
	g.BeginInspect(1)
	host.files = host.files[2:] // only moon.jpg left
	host.index = 0
	g.FilesChanged()
	if g.InspectingDuplicates() {
		t.Fatal("inspect must end when the group no longer has two members")
	}
}
```

For `TestFilesChanged_ClearsInspectWhenGroupDissolves`, shrinking `host.files` rebuilds groups via `applyFilter`. If flaky, `Warm` after shrinking.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -count=1 -run 'TestHandleKeyEvent_GFromInspectReopensVariants|TestHandleKeyEvent_DNoopWhileInspecting|TestHandleKeyEvent_PNoopWhileInspecting|TestHandleKey_DNoopWhileBrowsing|TestFilesChanged_ClearsInspectWhenGroupDissolves' ./internal/ui/ ./internal/ui/grid/`

Expected: FAIL (G Toggle-opens hide overview; D toggles hide; P enters picture-frame).

- [ ] **Step 3: Implement G/D/P guards, menu Disabled, FilesChanged, clearToDropzone**

Keep `TestHandleKey_DTogglesHideDuplicates` green (not browsing). Keep `TestHandleKeyEvent_GIsIgnoredDuringPictureFrameMode` green.

- [ ] **Step 4: Run tests to verify they pass**

Run:

```
go test -count=1 ./internal/ui/grid/
go test -count=1 -run 'TestHandleKeyEvent_|TestStepImage_|TestFilesChanged_|TestActionsMenu_' ./internal/ui/
```

Expected: PASS.

- [ ] **Step 5: Do not commit**

Report files changed and the test command output.

---

### Task 6: Docs and todos

**Files:**
- Modify: `internal/ui/help/manual.md` (browse bullets ~391–398 and cheatsheet ~905–907)
- Modify: `internal/ui/help/manual_de.md` (matching ~438–448 and ~1038–1041)
- Modify: `ARCHITECTURE.md` — hide-duplicates “Where to look” line
- Modify: `todos.md` — leave the item under TODO, add a pointer to this plan; do **not** move it to Done

**Interfaces:** none.

English, insert after the existing Shift+D / Esc / G bullets in section 8 (~after “`G`/Close leave hide on but **end** browse.”):

> Committing a variant (`Return` or a click) shows **that** file, even when hide-duplicates would otherwise keep the highest-resolution copy on screen. Left/Right then loop only the group; Home/End still jump to the first/last visible file of the whole set. `Esc` or `G` from that view reopens the variants grid; `Esc` again returns to the hide-duplicates grid. `D` and `P` do nothing while variants are showing or that loop is active.

Cheatsheet **Browse duplicates** bullet — append:

> Return/click keeps the chosen copy on screen and loops the group with Left/Right; `Esc` or `G` returns to the variants grid, then `Esc` to the hide-duplicates grid. `D` and picture-frame stay off during that loop.

German, after the `G`/Schließen sentence (~447–448):

> Eine Variante mit `Return` oder einem Klick zu öffnen zeigt **diese** Datei, auch wenn das Ausblenden sonst die Kopie mit der höchsten Auflösung behalten würde. Links/Rechts laufen dann nur durch die Gruppe; Home/Ende springen weiter zum ersten/letzten sichtbaren Bild der ganzen Menge. `Esc` oder `G` in dieser Ansicht öffnet wieder das Varianten-Raster; ein weiteres `Esc` kehrt zum Raster mit ausgeblendeten Extra-Kopien zurück. `D` und der Bilderrahmen-Modus tun nichts, solange Varianten angezeigt werden oder diese Schleife aktiv ist.

Cheatsheet **Duplikate anzeigen** — matching append.

ARCHITECTURE hide-duplicates locator:

> "How does hide-duplicates work?" → `internal/imaging/dhash.go` + `internal/ui/grid/dupes.go` (inspect loop: `BeginInspect` / `InspectMembers`; Escape reopen: `internal/ui` `reopenVariantGrid`).

todos.md: under the TODO item, add one line: `Plan: docs/superpowers/plans/2026-08-26-variant-inspect-loop.md`.

- [ ] **Step 1: Edit the four files**

No tests. Do not add `lang.L` keys. Do not touch README.

- [ ] **Step 2: Locale / fmt**

Run: `make fmt-check`

If you touched only markdown, `make fmt-check` still must pass on the branch’s Go files.

- [ ] **Step 3: Do not commit**

Report files changed.

---

## Execution notes for the parent agent

1. Open points are locked (Florian 2026-08-26): G reopens variants; Home/End whole-set; D and P disabled while browsing or inspecting.
2. Dispatch Task 1 → review/fix → Task 2 → review/fix → … → Task 6. Never two implementers at once.
3. After Task 6, run CI-matching verification from the repo root:

```
make fmt-check
go vet ./...
go build ./...
go test -timeout 20m -race ./...
```

4. Suggested commit message (user commits):

```
Loop duplicate variants after picking one from the grid.

Return from the variants grid kept losing the chosen extra to the
highest-resolution stand-in, and arrows then skipped the rest of the
group. Committing a variant now inspects that file, wraps arrows inside
the group, and uses Escape to walk back to the variants grid.
```

## Self-review

- Spec coverage: Return/click keeps extra, jump skipped, arrow wrap, Home/End whole-set, Escape/G reopen variants, D/P disabled, delete dissolve, docs — each has a task.
- No TBD / “add tests later”.
- `inspectKey` / `BeginInspect` / `ClearInspect` / `InspectMembers` / `closeOverlay` / `reopenVariantGrid` / `stepInMembers` names are used consistently.
- `OnSelected` still resolves `fileIndex` before closing.
- Viewer tests do not call unexported `setHighlight` (they use Right then Return).
