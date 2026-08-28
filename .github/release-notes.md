## What's Changed

### New Features
- Changed a few labels to make them less confusing

### Bugfix

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

### Internal

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

**Full Changelog**: https://github.com/frathe/picfetch/compare/v0.2.10...v0.2.11
