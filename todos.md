# PicFetch — TODOs

## Done

- Split `internal/ui/toast.go` import groups so stdlib, `fyne.io`, and
  `github.com/frathe/picfetch` are blank-line-separated like neighbouring
  `internal/ui` files (2026-08-25).

- `make fmt` / `make fmt-check` / CI use `goimports -local github.com/frathe/picfetch`
  (pinned as a Go `tool` in `go.mod`) so that grouping cannot regress (2026-08-25).

- Trimmed `ARCHITECTURE.md` from per-function commentary to a locator
  package map plus the “Where to look for X” index (2026-08-25).

### Menu Window
Menu points
- viewer
- exif information
- grid view
- picture frame mode
- help
  Windows that are currently showing are grayed out in the menu

### Menu Actions
Menu points
 - sort with sub menu of sorting options the currently active has a checkmark or is grayed out
 - hide duplicates checkmarked when active (toggable)
 - show variant available it the current item has dupes
 - exif information
Windows that are currently showing are grayed out in the menu

### Never-started canary
Before the `completion.Signal` migration, waiting on an operation that had never begun blocked on a nil channel until
the test timed out with a named message. `Signal.Wait` on a never-begun signal returns nil immediately - which is
exactly what lets `drain` drop its nil-guard, but also meant a helper that used to fail loudly then returned silently.

The guard went back in unevenly. Helpers that carried an *explicit* `== nil` check kept it as `Begun()`:
`settleChooser`, `settleWallpaper`, `settleFavoritePreviews`, and `settleToast` - the last being the best of the four,
since its `stop == nil` answers "pending *now*" rather than "ever begun". Helpers that relied on the *implicit*
nil-channel block lost theirs silently: `waitUntilLoaded`, `waitForScan`, `waitForSort`, `waitForAnimStopped`,
`waitForClipboard`.

Restored: those five named wait helpers now fatal via `Begun()`; `drain` and `waitFor` still do not.
`go test -race ./...` passed with no canary fatals (2026-08-25).

## ACTIVE DEVELOPMENT

## TODO

- `finishLoad` (`internal/ui/load.go:192-305`) is a 114-line do-everything pipeline (vector setup, fade, overlay, zoom,
  resize, title, animation, preload). It is linear and well-commented; decompose into named steps only if it needs to
  change anyway.

- `internal/imaging/exif.go` (687 lines) holds two parsers plus IFD walking plus display formatting. Cohesive and
  well-tested; a parse/format file split is cosmetic.

## not deemed worth implementing (edge cases)

- There is a bug in the Windows Version: WHen in Gridview, multiselect via
  the space key works, but when trying it with mouse and Ctrl key, it does not.
  Holding the Ctrl key down and clicking on an image does not select it but
  instead opens it. Observation, when pushing the Ctrl key at exactly the same time
  as clicking on the image, it actually works, and the image is selected.
  (this seems to be a bug in fyne, created an issue, sorry Windows users)
