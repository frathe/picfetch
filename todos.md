# PicFetch — TODOs

## Done

### What's Changed

#### New Features

#### Bugfix

- Hashing a folder no longer crashes when the file set shrinks underneath
  the pool. The duplicate model reads an immutable snapshot of the file
  set instead of the live slice — the old live `Count()`/`KeyAt(i)` pair
  could index past the end, reproducible in about a second on a 48-file
  drop under `-race`.
- A file the scanner produced no URI for no longer panics the hashing
  pass.

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

## ACTIVE DEVELOPMENT

## TODO

## not deemed worth implementing (edge cases)

- There is a bug in the Windows Version: WHen in Gridview, multiselect via
  the space key works, but when trying it with mouse and Ctrl key, it does not.
  Holding the Ctrl key down and clicking on an image does not select it but
  instead opens it. Observation, when pushing the Ctrl key at exactly the same time
  as clicking on the image, it actually works, and the image is selected.
  (this seems to be a bug in fyne, created an issue, sorry Windows users)
