# PicFetch — TODOs

## Done

### What's Changed

#### New Features

#### Bugfix

#### Internal

## ACTIVE DEVELOPMENT

## TODO

### Favorites un-merges the macOS native menu bar

`internal/ui/favorites` calls `f.menu.Refresh()` from two sites
(`SetHasFiles`, `refreshMenu`) with no `syncNativeMenuBar` follow-up.
`fyne.Menu.Refresh` is a `SetMainMenu`, which rebuilds the Darwin native
bar; only `refreshMainMenu` folds it back together afterwards. So adding,
renaming, or deleting a favorite already leaves a duplicate "Window" menu
and Command-prefixed accelerators on the unmodified letters until the next
`refreshMainMenu`. Pre-existing, not caused by the `viewer` field-cluster
refactor. The invariant worth stating: nothing outside `refreshMainMenu`
may call `Refresh()` on a menu that lives in the main bar.

### The manual-opened observer registration is untested

After `52af098` reduced the menu push sites to choke points, F1 and
Window → Help have no hand-written `syncMenus`; they rely on
`help.SetOnManualOpened(view.syncMenus)` staying registered in
`buildMainMenu`. Deleting that registration passes the entire test suite —
verified by mutation. No viewer-level test is possible because
`ShowManual` panics under Fyne's test theme (see the note at the tail of
`internal/ui/e2e_test.go`). `internal/ui/help`'s own
`TestHelp_SetOnManualOpenedFiresOnShow` pins that the hook *fires*, not
that `internal/ui` subscribes to it.

### `maybeStartUpdateCheck` begins its lifecycle token before the verifier is built

`9fb2f79` moved the update client construction inside
`autoupdate.Updater.Start`, so on `NewSigstoreVerifier()` failure the
token has already been superseded, where previously it had not.
Currently unobservable — the build path only runs when the client is nil,
which is the first call, when no check goroutine exists — but it is a
real ordering difference from the pre-extraction code.

## not deemed worth implementing (edge cases)

- There is a bug in the Windows Version: WHen in Gridview, multiselect via
  the space key works, but when trying it with mouse and Ctrl key, it does not.
  Holding the Ctrl key down and clicking on an image does not select it but
  instead opens it. Observation, when pushing the Ctrl key at exactly the same time
  as clicking on the image, it actually works, and the image is selected.
  (this seems to be a bug in fyne, created an issue, sorry Windows users)
