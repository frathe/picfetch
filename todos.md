# PicFetch — TODOs

## Done

### What's Changed

#### New Features

#### Bugfix

- macOS "Open With", dropping a file on the Dock icon, `open -a`, and
  double-clicking a file already associated with PicFetch now actually open
  it in the app — every one of those was silently ignored before, whether
  PicFetch was already running or being launched cold by the click itself.
  These paths also now cover the app's whole supported format list, not just
  the seven extensions Finder used to offer them for: HEIC, AVIF, TIFF, SVG,
  ICO, BMP, and every RAW format PicFetch reads are all included now, and a
  folder can be dropped on the Dock icon the same as a file.

#### Internal

- Extracted the duplicate-visibility model out of `internal/ui/grid` into a
  new, viewer-independent `internal/dupes` package: dHashes and native pixel
  sizes keyed by file, generation-scoped wipe-vs-adopt, the Hamming
  threshold, the group snapshot (`Compute`/`Install`), the hide-duplicates
  and inspect modes, and the visibility queries (`IsVisible`/`NextVisible`/
  `FirstVisible`/`LastVisible`/`VisibleIndexesExcept`) plain navigation
  needs. The viewer now owns the model and answers arrow-key/Home/End/
  shuffle questions directly from it (`internal/ui/visibility.go`) instead
  of polling a closed grid overlay. `internal/ui/grid` keeps presentation
  and browse-duplicates, and boxes its hashing pass behind a new
  `hashEngine` type (`grid/hashengine.go`) rather than more fields on
  `Overview`.

## ACTIVE DEVELOPMENT

## TODO

### `viewer.FileAt` is unguarded and read off the UI goroutine

- **Impact 4 · Risk 2 · Effort 3 → priority 18**
- Where: [viewer.go:882](internal/ui/viewer.go:882) — `FileAt(i)` is a bare
  `v.state.files[i]`. `dupes.Model.Compute` calls `Count()` then `KeyAt(i)`
  (`dupeFileSet`, [visibility.go:35](internal/ui/visibility.go:35)) from a
  hash-pool worker while the UI goroutine can replace `v.state.files`
  concurrently.
- A shrink landing between the two reads panics the worker, and the slice is
  read there with no synchronization at all. The pre-extraction `hostSet`
  had the identical shape, so this is not new — the generation check is the
  only thing making it rare in practice, not a real guard.
- **Fix**: hoist the key slice out from under the lock inside `Compute`
  (snapshot `Count()`/`KeyAt` results once, up front) rather than adding
  locking to `FileAt` itself, which every other Host caller also uses on
  the assumption that it is cheap and synchronous.

### `hideApplyAt` is never cleared between hashing passes

- **Impact 1 · Risk 1 · Effort 1 → priority 10**
- Where: [hashengine.go:202](internal/ui/grid/hashengine.go:202)
  `shouldScheduleHideApply`. Setting `remaining == 0` sets `hideApply` but
  leaves `hideApplyAt` at the previous pass's last mid-window timestamp, so
  a new pass starting within `hideApplyMinInterval` (250 ms) of the old
  one's last apply skips its own first mid-window apply.
- The last job always applies regardless, so nothing is lost permanently —
  this is a latency wart, not a correctness bug.
- **Fix**: reset `hideApplyAt` to 0 when a pass starts (`Run`'s top), or
  when `hashJobs` transitions from 0.

### The hashing pass has no nil-URI guard

- **Impact 3 · Risk 2 · Effort 1 → priority 25**
- Where: [hashengine.go:111](internal/ui/grid/hashengine.go:111) —
  `u := e.host.FileAt(i)` then `u.String()` three lines later with no nil
  check, unlike every neighboring helper (`rememberHash`, `hashOf`, …in
  `grid/dupes.go`), which all guard `u == nil` first.
- `Host.FileAt(i)` returning nil panics the hashing pass. Deliberately left
  as-is during the extraction (no behaviour change, locked decision 7) since
  the loop already cannot survive a nil URI — the key it dedups and caches
  by is that URI's own string — so a guard here is a genuine gap, not a
  moved one.
- **Fix**: `if u == nil { continue }` right after the `FileAt` call.

### `NextVisible` is O(n) per arrow key

- **Impact 2 · Risk 2 · Effort 3 → priority 12**
- Where: [visible.go:127](internal/dupes/visible.go:127) `NextVisible` →
  `InspectMembers` → `InspectSource` ([visible.go:61](internal/dupes/visible.go:61))
  scans the whole file set for the inspect key on every single step, and the
  skip-hidden-extras walk calls `IsHiddenExtra` (one model-mutex
  acquisition) per candidate.
- Unchanged from the pre-extraction cost — it is the same shape it always
  was — but now concentrated in one place and visible at scale: a user
  moving through a 50k-file drop with inspect or hide-duplicates on pays
  this on every keystroke.
- **Fix**: cache the inspect key's index instead of re-scanning
  (`BeginInspect` already knows it at the moment of the call); batch the
  hidden-extra check by reading the `Groups` snapshot once per `NextVisible`
  call instead of once per candidate index.

### `applyVisibleFilter` computes `hostRep` unconditionally

- **Impact 1 · Risk 1 · Effort 1 → priority 10**
- Where: [search.go:131](internal/ui/grid/search.go:131) —
  `hostRep := g.dupes.RepresentativeOf(g.browseHost)` runs whenever
  `nameFilter || hide || browseFilter`, but is only read inside the
  `browseFilter` branch below it. When browse is off (`browseHost == -1`,
  the common case while just searching or hiding duplicates), this is a
  wasted model-mutex acquisition per filter pass.
- **Fix**: move the `hostRep` computation inside `if browseFilter`.

## not deemed worth implementing (edge cases)

- There is a bug in the Windows Version: WHen in Gridview, multiselect via
  the space key works, but when trying it with mouse and Ctrl key, it does not.
  Holding the Ctrl key down and clicking on an image does not select it but
  instead opens it. Observation, when pushing the Ctrl key at exactly the same time
  as clicking on the image, it actually works, and the image is selected.
  (this seems to be a bug in fyne, created an issue, sorry Windows users)
