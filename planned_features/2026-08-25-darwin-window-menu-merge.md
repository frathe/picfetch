# Darwin Window-menu merge Implementation Plan

> **For agentic workers:** Execute in this session. Do **not** `git commit` (`AGENTS.md`).

**Goal:** On macOS, show one Window menu: PicFetch items above a separator, then Minimize / Zoom / the rest.

**Architecture:** Keep the Fyne Window menu on every platform. After each native rebuild, Darwin cgo moves those items onto `[NSApp windowsMenu]` and removes the extra top-level menu. Linux/Windows unchanged.

**Tech Stack:** Go, Fyne v2.8.0, AppKit cgo (`-fobjc-arc`), same pattern as `internal/winpos/darwin.go`.

## Global Constraints

- Do **not** `git commit`.
- Do **not** mark `todos.md` Menu Window as Done.
- Do **not** call `ShowManual` / `F1` from `internal/ui` tests.
- Do **not** add `TODO`/`FIXME` in source. `gofmt` every touched `.go` file.
- Feature packages stay menu-ignorant. Handles cross the cgo boundary as `uintptr_t`, not `unsafe.Pointer`.
- Separator tag must be `>= 5000` (Fyne `menuTagMin`) and far from Fyne callback IDs so `clearNativeMenu` strips it without colliding with `menuCallback`.

---

### Task 1: Merge primitive (TDD)

**Files:**
- Create: `internal/ui/windowmenu_darwin.go`
- Create: `internal/ui/windowmenu_notdarwin.go`
- Create: `internal/ui/windowmenu_darwin_test.go`

**Interfaces:**
- Produces: `func mergeNativeWindowMenu()` (all platforms; no-op off Darwin)
- Produces (Darwin): `func mergeWindowMenus(main, system uintptr, label string) bool`

- [ ] **Step 1:** Darwin tests for both-titled-Window merge, Fenster-vs-Window + label `Fenster`, idempotent second call, no-op when there is no duplicate. Watch them fail (missing symbol or no-op).
- [ ] **Step 2:** Implement cgo merge + test menu builders. `gofmt`. `go test -count=1 -run TestMergeWindowMenus ./internal/ui/`
- [ ] **Step 3:** Skip commit.

### Task 2: Call merge after every native rebuild

**Files:**
- Modify: `internal/ui/windowmenu.go` (`refreshMainMenu`)
- Modify: `internal/ui/save.go` (`updateFileMenuState` uses it)
- Modify: `internal/ui/build.go` (`fyne.Do(mergeNativeWindowMenu)` after `SetMainMenu`)

- [ ] **Step 1:** Existing Window/File menu tests still pass (they already Refresh).
- [ ] **Step 2:** `refreshMainMenu` = Refresh then `mergeNativeWindowMenu`. Wire both update paths and `build.go`.
- [ ] **Step 3:** `go test -count=1 -run 'TestBuildMainMenu_|TestWindowMenu_' ./internal/ui/`

### Task 3: Docs

**Files:** `ARCHITECTURE.md`, `internal/ui/help/manual.md`, `internal/ui/help/manual_de.md`

- [ ] One sentence in §12: on macOS these items live in the system Window menu, above Minimize.
- [ ] `ARCHITECTURE.md` `menu.go` cell: Darwin merge into `NSApp.windowsMenu`.
