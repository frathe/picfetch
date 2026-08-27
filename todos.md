# PicFetch — TODOs

## Done

### What's Changed

#### New Features

#### Bugfix

#### Internal

## ACTIVE DEVELOPMENT

## TODO

### The duplicate-visibility model lives inside the grid feature

- **Impact 4 · Risk 4 · Effort 4 → priority 16**
- Where: `internal/ui/grid` owns hide-duplicates state, group membership, and
  inspect/browse mode; the core viewer reaches into it at **29 call sites**
  across [viewer.go](internal/ui/viewer.go), [keys.go](internal/ui/keys.go),
  [actionmenu.go](internal/ui/actionmenu.go), and
  [windowmenu.go](internal/ui/windowmenu.go)
  (`IsHiddenExtra`, `InspectMembers`, `InspectingDuplicates`,
  `BrowsingDuplicates`, `HideDuplicates`).
- Plain navigation — arrow keys, Home/End, shuffle — must poll the grid
  overlay's state per index even while the overlay is closed:
  `nextVisibleIndex` / `firstVisibleIndex` / `lastVisibleIndex` /
  `randomVisibleOther` ([viewer.go:940–1003](internal/ui/viewer.go:940))
  are filter-aware iteration implemented by poking a feature package.
  This inverts the codebase's own rule ("features expose state; `internal/ui`
  composes them" — ARCHITECTURE.md): *which files are visible* is file-set
  model state, not overlay state.
- **Why it matters**: this seam is where the bugs actually are — the last
  three fix branches (variant loop after grid pick, highest-res
  representative, variants badge/hover) all patched interactions across it.
  Every mode added near it (slideshow shuffle, inspect, hide-dupes) multiplies
  the guard combinations in `handleKeyEvent` and the menu-state code.
- **Fix (staged)**: extract a visibility/grouping model (e.g. `internal/ui`'s
  own `visibleSet`, or an `internal/dupegroups` package) owning
  hidden-extras, group membership, and representative choice, fed by the
  grid's hashing pass and consumed by both the viewer's navigation and the
  grid's rendering. The grid keeps presentation (badges, filter display,
  marquee); the viewer stops asking the grid who exists.

## not deemed worth implementing (edge cases)

- There is a bug in the Windows Version: WHen in Gridview, multiselect via
  the space key works, but when trying it with mouse and Ctrl key, it does not.
  Holding the Ctrl key down and clicking on an image does not select it but
  instead opens it. Observation, when pushing the Ctrl key at exactly the same time
  as clicking on the image, it actually works, and the image is selected.
  (this seems to be a bug in fyne, created an issue, sorry Windows users)
