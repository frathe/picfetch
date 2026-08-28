# PicFetch — TODOs

## Done

### What's Changed

#### New Features
- Changed a few labels to make them less confusing

#### Bugfix

- Hashing a folder no longer crashes when the file set shrinks underneath
  the pool. The duplicate model reads an immutable snapshot of the file
  set instead of the live slice — the old live `Count()`/`KeyAt(i)` pair
  could index past the end, reproducible in about a second on a 48-file
  drop under `-race`.
- A file the scanner produced no URI for no longer panics the hashing
  pass.
- A crash when pressing 0 after rotating an image and returning to the
  drop zone. `clearToDropzone` cleared the frames but left the rotation
  set, and the 0 key is handled ahead of the navigation guard, so
  `resetRotation` indexed an empty frame slice. Reproducible in four
  keystrokes: open, R, Escape, 0.

#### Internal

- `internal/dupes.FileSet` is a single `Snapshot()` method; every `Model`
  method takes one snapshot at entry.
- Arrow keys with inspect or hide-duplicates on no longer rescan the whole
  file set per step (`InspectSource` is an O(1) map lookup; the
  hidden-extra walk takes the model mutex once per call, not once per
  candidate).
- A new hashing pass no longer inherits the previous pass's hide-apply
  throttle floor.
- Deleted `grid/dupes.go`'s `rememberHashFail` and `hashFailedOf`, two
  duplicate-hash wrappers with no callers anywhere in the tree; the
  hashing pass already calls `dupes.Model.PutFailed`/`Failed` directly.
  Dropped the stale `hashFailedOf` mention from `hashengine.go`'s
  wrapper comment.
- `internal/dupes` now exports `Visibility()`, a frozen read of the
  hide flag plus the installed group snapshot taken in one model-mutex
  acquisition, with `HiddenExtra`/`Visible`/`RepresentativeOf`/`Size`
  methods for testing many indices against it without re-locking per
  index. `applyVisibleFilter`'s filter pass and `jumpIfHiddenExtra` now
  take one such read instead of one (in fact two) model-mutex
  acquisitions per file.
- Split the `viewer` god object from 87 fields to 55: four new
  subpackages, each reading a value `State` snapshot built in exactly one
  `internal/ui` function rather than a `Host` interface —
  `internal/ui/menus` (the 20 File/Window/Actions menu items and their
  whole Checked/Disabled matrix as `Apply(State)`, −19 fields),
  `internal/ui/autoupdate` (the release check/download policy, staged-
  update lifecycle, and What's-New cache, −4 fields), `internal/ui/infoview`
  (the persistent info overlay's widgets and text formatting, −6 fields),
  and `internal/ui/display` (decoded frames, frame index, view-only
  rotation, and crossfade, −3 fields). A `Host` for `menus` alone would
  have needed roughly a dozen methods, which is why these four read a
  snapshot instead. Also lands needs_refactoring.md item 5 (`menuState()`/
  `syncMenus()` replace the per-site Checked/Disabled push and
  `HighlightChanged`'s four-boolean hand-diff with one recompute-and-diff
  entry point, called from 16 choke points instead of the previous 23 call
  sites) and the rest of item 6 (`appState` now evicts the image cache
  itself on file removal via an `onRemove` hook, wired in `build.go`).

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
