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

### `viewer` god object — 87 fields and still growing

- **Impact 4 · Risk 3 · Effort 4 → priority 14**
- Where: the struct definition alone spans
  [viewer.go:38–478](internal/ui/viewer.go:38); its methods spread across
  ~30 files of `internal/ui`. (Grew by one field, `dupes *dupes.Model`,
  when the duplicate-visibility model moved out from under the grid — item
  2's resolution; the navigation helpers it replaced were methods, not
  fields, so they didn't shrink the count on the way out.)
- The comments are exemplary and the tests thorough, which is why this is
  Risk 3 and not 5 — but the growth pattern is intact: autoupdate landed 6
  new fields, the info overlay 7, menu items account for **16 fields** on
  their own. Every feature keeps paying a "where in the 430-line struct does
  my field go" tax, and the concurrency notes per field only get harder to
  hold in one head.
- **Fix**: continue the existing feature-split practice with field-cluster
  extractions that have clean seams already visible in the comments:
  - menu-item state (`saveItem` … `actionsTrashItem`, 16 fields) → a
    `menus` type with a single recompute entry point (pairs with item 5);
  - updater state (`update`, `updateDir`, `updateOp`, `updateDone`,
    `updateCurrentVersion`, `updateDayMu`) → an `updater` type in
    autoupdate.go;
  - info-overlay state (`infoVisible`, `infoText`, `exifLink`, `infoCard`,
    `currentFileSize`, `currentHasEXIF`, `currentPreview`) → info.go;
  - display state (`displayFrames`, `displayFrameIdx`, `rotation`,
    `fadeAnim`) → load.go/rotate.go.

## not deemed worth implementing (edge cases)

- There is a bug in the Windows Version: WHen in Gridview, multiselect via
  the space key works, but when trying it with mouse and Ctrl key, it does not.
  Holding the Ctrl key down and clicking on an image does not select it but
  instead opens it. Observation, when pushing the Ctrl key at exactly the same time
  as clicking on the image, it actually works, and the image is selected.
  (this seems to be a bug in fyne, created an issue, sorry Windows users)
