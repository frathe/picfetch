# Favorites Native Menu-Bar Fix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> `superpowers:subagent-driven-development` (recommended) or
> `superpowers:executing-plans` to implement this plan task-by-task. Steps use
> checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop adding or deleting a favorite from leaving a duplicate "Window"
menu and Command-prefixed accelerators in the macOS native menu bar, and make
the invariant behind it hold structurally instead of by convention.

**Architecture:** `internal/ui/favorites` is viewer-independent and reaches the
viewer through a narrow consumer-side `Host` interface, so it cannot call
`refreshMainMenu` itself. One method joins that interface. In exchange, the
feature stops calling `fyne.Menu.Refresh` at all, which is what makes the
invariant true rather than merely documented.

**Tech Stack:** Go 1.27.0 toolchain on a Go 1.26.7 module, Fyne v2, GoLand
inspections through the `goland` MCP server.

**Spec:** `todos.md`, section
`Favorites un-merges the macOS native menu bar`.

## The Bug

`fyne.Menu.Refresh` is `SetMainMenu` underneath, which on Darwin tears down
and rebuilds the native bar (`clearNativeMenu` plus a fresh Fyne Window next
to GLFW's). Only `refreshMainMenu` folds that bar back together afterwards,
via `syncNativeMenuBar`, which merges the Window items into
`NSApp.windowsMenu` and clears AppKit's default Command mask from unmodified
letter accelerators.

So any `Refresh()` on a menu living in the main bar, made from outside
`refreshMainMenu`, leaves the bar un-merged until the next sync.

### Corrected diagnosis

`todos.md` names `SetHasFiles` and `refreshMenu` as the two unguarded sites,
and lists "adding, renaming, or deleting". Tracing every caller shows that is
not right, and the correction narrows the change:

Tree-wide there are exactly three `.Refresh()` calls on a main-bar menu:
`internal/ui/windowmenu.go:22` (inside `refreshMainMenu`, correct),
`internal/ui/favorites/favorites.go:116` (`SetHasFiles`), and
`internal/ui/favorites/favorites.go:163` (`refreshMenu`).

| Path | Folds afterwards? |
|---|---|
| `SetHasFiles` ← `menu.go:125` (`syncMenus`) | **Yes** — `refreshMainMenu()` is the next line, deliberately, per the comment at `menu.go:100-115`. |
| `refreshMenu` ← `SetDir` ← `run.go:68` | **Yes** — startup folds at `run.go:45` and again at `run.go:47`. |
| `refreshMenu` ← `AddCurrentList` (`favorites.go:249`) | **No. This is the bug.** |
| `refreshMenu` ← `performRemove` (`manage.go:362`) | **No. This is the bug.** |

There is also **no rename path**: `internal/favstore` exposes `Save` and
`Remove` and nothing else, and `manage.go` has only `performRemove`. The
user-visible symptom is therefore: add a favorite, or delete one, and the
menu bar stays un-merged until the next `syncMenus` that reports a change.

`todos.md` must be corrected when this closes, not just ticked off.

## Resolved Decisions

Answered by the user during planning:

1. **Fix shape** — add the hook to `Host`, and remove *both* `f.menu.Refresh()`
   calls. `SetHasFiles` only sets `Disabled`; its single caller folds on the
   very next line. This leaves zero `Refresh()` calls on a main-bar menu
   outside `refreshMainMenu`, so the invariant is structural.
   Rejected: calling the hook from both sites (double native rebuild per
   sync); keeping `Refresh()` and folding after it (invariant still violated).
2. **Host method name** — `RefreshMenus()`.
   Rejected: `MenuChanged()`.
3. **Tests** — assert the hook at both levels: a counting fake `Host` in
   `favorites`, and an `internal/ui` test pinning that the viewer's
   implementation reaches `refreshMainMenu`.
   Rejected: feature-level only; a source-scanning invariant test.

## Global Constraints

- Read `AGENTS.md` and `ARCHITECTURE.md` before editing.
- `internal/ui/favorites` stays viewer-independent. It must not import
  `internal/ui`, learn about `fyne.MainMenu`, or call `SetMainMenu`.
- Keep `Host` narrow. One method is added; nothing else.
- No behaviour change off Darwin. `syncNativeMenuBar`'s two steps are already
  no-ops there.
- Do not touch `qodana.yaml` and add no `//goland:noinspection`.
- Comments in this area are load-bearing and were written deliberately. Where
  a comment states something this change makes untrue, rewrite it; do not
  delete it and do not leave it stale.
- No `git commit` (`AGENTS.md`). The parent session ends with a suggested
  message.

## Known Risk to Verify, Not to Design Around

After this change, `SetDir` at `run.go:68` reaches `refreshMainMenu` during
`startViewerRuntime`, which runs **before** `window.Show()`. `run.go:38-40`
warns that a `fyne.Do` fold before `application.Run()` executes inline and can
merge while the Fyne Window title is still absent.

This is expected to be harmless, and strictly better than today: the previous
code did a bare `menu.Refresh()` there with no fold at all, and `run.go:45`
re-folds after `Show()` regardless. `refreshMainMenu` also returns early when
`v.win == nil`. But it is a real ordering change on the user's own platform,
so it gets an explicit manual macOS check in the final verification rather
than an assumption.

## File Map

| File | Change | Task |
|---|---|---|
| `internal/ui/favorites/favorites.go` | `Host` gains `RefreshMenus`; both `Refresh()` calls go | 1 |
| `internal/ui/windowmenu.go` | new exported `RefreshMenus` on `viewer` | 1 |
| `internal/ui/menu.go` | rewrite the stale half of the `syncMenus` comment | 1 |
| `internal/ui/favorites/favorites_test.go` | `fakeHost` gains the method (task 1), then the new tests (task 2) | 1, 2 |
| `internal/ui/menu_test.go` | viewer-level wiring test | 3 |

## Subagent Routing

All tasks go to the `go-expert` agent.

| Task | Model | Rationale |
|---|---|---|
| 1 | `sonnet` | An interface change that must land atomically with both implementers, plus rewriting a comment whose current text is load-bearing and partly invalidated. |
| 2 | `sonnet` | Test authoring against the existing `fakeHost`/`newFeature` harness, with one counting subtlety (see the task). |
| 3 | `opus` | Genuine investigation: whether Fyne's test driver exposes any observable signal for a main-menu republish is unknown, and the honest fallback has to be chosen and reported rather than faked. |

**Step 1** runs Task 1 alone — every later task depends on the interface
existing.
**Step 2** runs Tasks 2 and 3 in parallel; they touch disjoint files
(`internal/ui/favorites/*_test.go` versus `internal/ui/menu_test.go`).
**Step 3** is the parent session's verification and backlog close.

The parent session waits for the user's go-signal before each step.

---

## Task 1: Add the Hook and Remove Both `Refresh()` Calls

**Subagent:** `go-expert`, model `sonnet`

**Files:**
- `internal/ui/favorites/favorites.go`
- `internal/ui/windowmenu.go`
- `internal/ui/menu.go`
- `internal/ui/favorites/favorites_test.go` (the `fakeHost` method only)

All four edits must land together: adding a method to `Host` breaks every
implementer until each one has it.

**Steps:**

- [ ] `favorites.go` — add to the `Host` interface, after
      `SyncFavoritePreviews`:

      ```go
      	// RefreshMenus re-publishes the main menu bar. This feature calls it
      	// after changing its own menu's items, because fyne.Menu.Refresh is
      	// SetMainMenu underneath: on Darwin that rebuilds the native bar, and
      	// only the host knows how to fold it back together afterwards.
      	RefreshMenus()
      ```

- [ ] `favorites.go:113-117` — `SetHasFiles` drops its `Refresh`:

      ```go
      // SetHasFiles enables adding the current list when it is non-empty. It
      // deliberately does not re-publish the menu: its one caller is
      // internal/ui's syncMenus, which folds the bar on the very next line.
      func (f *Feature) SetHasFiles(has bool) {
      	f.addItem.Disabled = !has
      }
      ```

- [ ] `favorites.go:163` — inside `refreshMenu`, replace `f.menu.Refresh()`
      with `f.host.RefreshMenus()`. The `f.menu.Items = items` assignment on
      the line above stays exactly as it is.

- [ ] `internal/ui/windowmenu.go` — add the viewer's implementation directly
      below `refreshMainMenu`:

      ```go
      // RefreshMenus is the favorites feature's way to ask for the fold, since
      // internal/ui/favorites is viewer-independent and cannot reach
      // refreshMainMenu itself. Adding or deleting a favorite rewrites that
      // menu's items, and every such rewrite has to end here.
      func (v *viewer) RefreshMenus() { v.refreshMainMenu() }
      ```

- [ ] `internal/ui/menu.go` — the comment at lines 100-115 opens with
      "SetHasFiles is inside the changed branch, and before refreshMainMenu,
      for two reasons that are both load-bearing - do not lift it out", and
      its first reason says SetHasFiles "ends in fyne.Menu.Refresh". That is
      no longer true. Rewrite **only that first bullet** so it says what is
      now true: `SetHasFiles` no longer refreshes anything itself, so the
      `refreshMainMenu` on the next line is what publishes the item's new
      `Disabled` state, and the ordering still matters for that reason.
      Leave the second bullet (the `FileCount()`/`NoFiles` complement
      argument) and the closing startup-sync paragraph untouched — both are
      still correct.

- [ ] `favorites_test.go` — give `fakeHost` the new method so the package
      compiles. Add a counter field; Task 2 will assert on it:

      ```go
      	refreshMenus int
      ```

      ```go
      func (h *fakeHost) RefreshMenus() { h.refreshMenus++ }
      ```

      Add no tests in this task.

- [ ] Search for any other implementer of `favorites.Host` in the tree
      (`grep -rn 'favorites.Host' .` and any test fake in `internal/ui`) and
      give it the method too. Report what you found.

**Verification:**

- [ ] `make fmt-check`
- [ ] `go build ./...` succeeds.
- [ ] `go vet ./...` is clean.
- [ ] `go test ./internal/ui/favorites/... ./internal/ui/` passes — the
      existing suite must stay green with no test changes.
- [ ] `grep -rn '\.menu\.Refresh()' internal/ui/favorites/` prints nothing.
- [ ] `grep -rn -E '\.Refresh\(\)' internal/ui/windowmenu.go` shows exactly
      one hit, the `bar.Refresh()` inside `refreshMainMenu`.

---

## Task 2: Feature-Level Tests for the Hook

**Subagent:** `go-expert`, model `sonnet`

**Files:**
- `internal/ui/favorites/favorites_test.go`
- `internal/ui/favorites/manage_test.go`

**The counting subtlety:** `newFeature` calls `f.SetDir(t.TempDir())`, which
calls `refreshMenu`, which now fires the hook. Every feature therefore starts
with `host.refreshMenus == 1` before the test body runs. Assert on the
*delta*: record the count after construction and compare against that, or
reset the counter to zero as the first line of the test body. Do not
hard-code `1` as the post-add expectation without accounting for this.

**Steps:**

- [ ] In `favorites_test.go`, add a test that adding a favorite fires the hook
      exactly once. Drive it through the same entry point the existing add
      tests use (`AddCurrentList` and its dialog, or the save path they call)
      rather than reaching into `refreshMenu` directly — the point is that the
      *user-facing* path ends in a fold.

- [ ] Add a test that `SetHasFiles` does **not** fire the hook, pinning the
      decision that its caller owns the fold. Without this, someone
      re-adding a `Refresh` there would go unnoticed.

- [ ] In `manage_test.go`, add a test that removing a favorite fires the hook
      exactly once. `performRemove` does its work in a goroutine and marshals
      back through `fyne.Do`; the existing remove tests in that file already
      handle that waiting correctly — follow whatever they do (`f.pending`,
      or the file's own helper) rather than sleeping.

- [ ] Name the tests in the style of their neighbours in each file, and give
      each a short comment saying what breaks in the real app when it fails:
      a duplicate "Window" menu and Command-prefixed accelerators on the
      unmodified letters, until the next sync.

- [ ] Change no existing test.

**Verification:**

- [ ] `make fmt-check`
- [ ] `go test ./internal/ui/favorites/...` passes.
- [ ] `go test -race ./internal/ui/favorites/...` passes — `performRemove` is
      concurrent.
- [ ] Mutation check, then revert it: change `refreshMenu`'s
      `f.host.RefreshMenus()` back to `f.menu.Refresh()` and confirm the new
      add and remove tests fail while the rest of the package still passes.
      Paste that output. If they do not fail, the tests are not testing what
      they claim — say so rather than adjusting the expectation.

---

## Task 3: Viewer-Level Wiring Test

**Subagent:** `go-expert`, model `opus`

**Files:**
- `internal/ui/menu_test.go`

**The open question this task must answer:** whether Fyne's test driver
exposes anything observable when the main menu is re-published. `bar.Refresh()`
is `SetMainMenu`, and `syncNativeMenuBar`'s two steps are no-ops off Darwin,
so there may be no behavioural signal available on the test platform at all.

Investigate first, then pick the strongest assertion that is actually true,
in this order of preference:

1. A behavioural assertion that the bar was re-published — if the test driver
   records `SetMainMenu` calls, or the window's `MainMenu()` identity or
   contents change observably.
2. Failing that, a test that drives the real viewer through a favorite add or
   delete and asserts the viewer's `RefreshMenus` was reached, using a seam
   that already exists in the harness. Do **not** add a mutable package-level
   test seam — `AGENTS.md` forbids them; runtime/test-configurable values
   belong on `viewer` or the owning feature.
3. Failing that, a compile-time `var _ favorites.Host = (*viewer)(nil)`
   assertion plus a direct `v.RefreshMenus()` call-through test.

**Steps:**

- [ ] Read `internal/ui/harness_test.go` for how `newTestUI`/`newTestViewer`
      build a viewer, and `internal/ui/menu_test.go:290-340` for how the
      existing favorites-menu tests drive `v.favorites.SetDir`.
- [ ] Determine which of the three assertion levels above is achievable.
      **Report which one you used and why the stronger ones were not
      available.** An honest level-3 test is the correct outcome if levels 1
      and 2 are not reachable; a level-1 test that does not actually observe
      anything is not.
- [ ] Write the test. Give it a comment naming the real-world failure it
      guards: a favorite added or deleted leaves a duplicate "Window" menu and
      Command-prefixed accelerators until the next sync.
- [ ] Change no existing test.

**Verification:**

- [ ] `make fmt-check`
- [ ] `go test ./internal/ui/` passes.
- [ ] `go test -race -run <YourTestName> -v ./internal/ui/` passes; paste the
      output.
- [ ] If your test is level 1 or 2, mutation-check it: make
      `viewer.RefreshMenus` a no-op body and confirm the test fails. Paste
      that output, then revert. If it does not fail, say so and drop to the
      honest level.

---

## Parent Controller Protocol After Every Task

The parent session, not the subagent, owns verification. After each task
returns:

1. `git diff -- <the task's files>` read in full.
2. `mcp__goland__lint_files` over the task's files.
3. `make fmt-check`, `go vet ./...`, `go build ./...`.
4. Re-run the task's own mutation check independently where it claimed one.
5. Fix up anything wrong directly rather than redispatching, unless the task
   needs redoing wholesale.
6. Report and wait for the user's go-signal before the next step.

## Final Repository Verification

- [ ] `mcp__goland__lint_files` over all touched files.
- [ ] `make fmt-check`
- [ ] `go vet ./...`
- [ ] `go build ./...`
- [ ] `go test -timeout 20m -race ./...`
- [ ] `grep -rn '\.Refresh()' internal/ui/favorites/` prints nothing.
- [ ] **Manual macOS check, by the user** — this is the only place the actual
      bug is visible, since `syncNativeMenuBar` is a Darwin-only cgo path:
      1. `make run`
      2. Open a folder so the file list is non-empty.
      3. Favorites → Add Current List to Favorites…, save one.
      4. Look at the menu bar: exactly one "Window" menu, and the unmodified
         letter accelerators (G, V, R, …) show without a Command prefix.
      5. Favorites → Manage Favorites…, delete it. Check the bar again.
      6. Also confirm startup is unchanged — see the ordering risk above:
         relaunch and check the bar before touching any menu.

## Backlog Close

- [ ] In `todos.md`, delete the
      `### Favorites un-merges the macOS native menu bar` section from
      `## TODO`.
- [ ] Add an entry under `## Done` → `#### Internal` that **corrects the
      original diagnosis** as well as recording the fix: `SetHasFiles` was
      never an unguarded site (`syncMenus` folds on the next line), `SetDir`
      is covered by the startup fold, and no rename path exists. The two real
      paths were adding a favorite and deleting one. State the invariant in
      its now-structural form: `internal/ui/favorites` calls
      `Host.RefreshMenus` and never `fyne.Menu.Refresh`, so `refreshMainMenu`
      is the only place a main-bar menu is re-published.
- [ ] `ARCHITECTURE.md` — check whether it describes `favorites.Host`'s
      method set. Update it in the same change if so; leave it if not.
- [ ] Move this plan to `finished_refactorings/`.
- [ ] Do not commit. Hand the user a suggested commit message.

## Out of Scope

- The remaining `todos.md` items: the untested manual-opened observer, the
  `maybeStartUpdateCheck` ordering note, and the two Qodana CI items.
- `internal/ui/menus`' `Apply`, which does not refresh anything itself.
- The `internal/ui/favorites/manage.go:200` `p.Refresh()` — that is the
  manage dialog's own widget, not a main-bar menu.
- Any change to `syncNativeMenuBar`, `mergeNativeWindowMenu`, or
  `applyUnmodifiedNativeAccelerators`. The native code is correct; only its
  call sites were incomplete.
