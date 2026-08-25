# Darwin Window-menu merge

**Date:** 2026-08-25
**Status:** accepted
**Problem:** On macOS the menu bar shows two adjacent **Window** menus. One is PicFetch’s (Viewer, EXIF Data, Grid View, Picture-frame mode, Help). The other is GLFW/AppKit’s system Window menu (Minimize, Zoom, Full Screen, window list).

## Decision

Merge PicFetch’s items into the system Window menu on Darwin. Linux and Windows keep a single PicFetch **Window** menu as they do today.

## Root cause

GLFW always creates an `NSMenu` titled `Window`, calls `[NSApp setWindowsMenu:]`, and fills Minimize / Zoom / Bring All to Front. Fyne 2.8.0’s Darwin driver special-cases **Help**, **About**, and **Settings** only. A `fyne.Menu` whose label is `lang.L("Window")` is inserted as an extra top-level menu. `MainMenu.Refresh()` rebuilds that extra menu every time (`SetMainMenu` → `setupNativeMenu`), so the duplicate survives grey-out updates.

## Behaviour

After every native menu rebuild on Darwin, the bar has exactly one **Window** menu, which is `[NSApp windowsMenu]`, with this order:

1. Viewer
2. EXIF Data
3. Grid View
4. Picture-frame mode
5. Help
6. separator
7. GLFW/AppKit items (Minimize, Zoom, …, window list)

Grey-out, show/enter actions, `V`/`E`/`G`/`P`/`F1` bindings, and File / Favorites / Help stay unchanged. Window → Help still opens the manual.

Linux / Windows: no merge, no cgo. The Fyne bar is still File, Favorites, Window, Help.

## Architecture

Cross-feature composition stays in `internal/ui`. Feature packages do not learn about menus or AppKit.

Keep `fyne.NewMenu(lang.L("Window"), …)` in `buildMainMenu` on every platform so tests, `Disabled` updates, and Fyne’s native item callbacks stay on the same `*fyne.MenuItem` pointers.

Darwin-only cgo (same style as `internal/winpos/darwin.go`: AppKit, `-fobjc-arc`) finds the extra top-level menu whose submenu is **not** `[NSApp windowsMenu]` and whose title equals `lang.L("Window")` (English `Window` or German `Fenster`). It moves those `NSMenuItem`s onto `windowsMenu` (tags, target, and action stay, so Fyne `menuEnabled` / `menuCallback` still work), inserts one separator after them, and removes the extra top-level item.

The separator must carry a Fyne-range tag (`>= 5000`, same as `menuTagMin` in Fyne’s `menu_darwin.m`) so the next `clearNativeMenu` strips it. Untagged separators would accumulate on every Refresh.

## When it runs

`setupNativeMenu` is inside Fyne; we cannot hook it. Merge must run **after** every native rebuild:

- `updateWindowMenuState` and `updateFileMenuState` both `MainMenu().Refresh()`. Route that through one helper, e.g. `(v *viewer) refreshMainMenu()`, that `Refresh()`s then calls `mergeNativeWindowMenu()`.
- After `window.SetMainMenu(buildMainMenu(view))` in `build.go`, queue the same merge on the UI thread (`fyne.Do`) so it runs after `runOnMainWhenCreated` has executed `setupNativeMenu`.

`mergeNativeWindowMenu` is a no-op when `windowsMenu` is nil, when there is no duplicate submenu, and on non-Darwin builds.

Idempotent: a second call with no duplicate top-level Window is a no-op.

## Out of scope

- Renaming the menu to View
- Fyne Darwin drawing ⌘ on modifier-0 letter shortcuts (`⌘V` next to Viewer)
- Removing Window → Help
- Changing G/P toggle behaviour
- `todos.md` Menu Window Done (still Florian’s call)

## Tests

- Existing `TestBuildMainMenu_Structure` stays: Fyne `MainMenu` still has a Window menu with five items. That is the Linux/Windows bar and the Darwin source of the moved items.
- Darwin-only test (`//go:build darwin`) builds a fake `NSApp` main menu (PicFetch Window + GLFW Window), runs the merge, asserts one Window and PicFetch items above a separator above Minimize. No `ShowManual` / `F1` from `internal/ui` tests.
- Manual check on a real Mac: launch, one Window menu, items work, Minimize still works, open a file so grey-out Refresh runs, still one Window menu.

## Docs

- `ARCHITECTURE.md` `menu.go` / `windowmenu.go` cell: Darwin merge into `NSApp.windowsMenu`.
- Manual §12: one sentence that on macOS these items live in the system Window menu, above Minimize.
