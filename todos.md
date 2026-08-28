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
- `applyVisibleFilter` only computes the browse representative when
  browsing.

## ACTIVE DEVELOPMENT

## TODO

### Dead duplicate-hash wrapper functions in `grid/dupes.go`

- **Impact 1 · Risk 1 · Effort 1 → priority 10**
- Where: [dupes.go:33](internal/ui/grid/dupes.go:33) `rememberHashFail` and
  [dupes.go:59](internal/ui/grid/dupes.go:59) `hashFailedOf` have no
  callers anywhere in the tree. Verified dead at commit `21b3630` too, so
  this predates the snapshot work — it is not something the `Snapshot`
  migration introduced. `hashengine.go`'s comment still lists
  `hashFailedOf` among the wrappers "which exist to nil-guard the
  `fyne.URI` the cell and Warm paths hand them", but nothing calls it.
- **Fix**: delete both functions, and drop the stale `hashFailedOf`
  mention from `hashengine.go`'s comment.

### `applyVisibleFilter` takes the model mutex once per index

- **Impact 2 · Risk 2 · Effort 3 → priority 12**
- Where: [search.go:147](internal/ui/grid/search.go:147) —
  `applyVisibleFilter`'s loop over `g.host.FileCount()` calls
  `g.dupes.IsHiddenExtra(i)` per index, one model-mutex acquisition per
  file. This is the same one-lock-per-candidate pattern the `Snapshot`
  work just removed from the navigation walks in
  `internal/dupes/visible.go` (`visibility()` + `hiddenExtra()`), but on
  the grid's filter pass instead of arrow-key navigation — at 50k files
  with hide or browse on, this is the per-keystroke/per-filter cost of a
  full-set scan under lock.
- Not folded into that work because the fix needs a new batched entry
  point exported from `internal/dupes` — something like a `VisibleMask()`
  or a `Visibility()` that hands back the hide flag plus the installed
  `Groups` so a caller can test many indices itself without re-locking per
  index. That is a package-API design decision, not a local edit, which is
  why it is a separate backlog item rather than part of the snapshot
  migration.
- **Fix**: export a batched visibility accessor from `internal/dupes` and
  have `applyVisibleFilter` read it once per filter pass instead of
  calling `IsHiddenExtra` in the loop.

## not deemed worth implementing (edge cases)

- There is a bug in the Windows Version: WHen in Gridview, multiselect via
  the space key works, but when trying it with mouse and Ctrl key, it does not.
  Holding the Ctrl key down and clicking on an image does not select it but
  instead opens it. Observation, when pushing the Ctrl key at exactly the same time
  as clicking on the image, it actually works, and the image is selected.
  (this seems to be a bug in fyne, created an issue, sorry Windows users)
