## What's Changed

### Bugfix

![Trane drag and drop](https://raw.githubusercontent.com/frathe/picfetch/main/assets/trane/draganddrop2.png)

- macOS "Open With", dropping a file on the Dock icon, `open -a`, and
  double-clicking a file already associated with PicFetch now actually open
  it in the app — every one of those was silently ignored before, whether
  PicFetch was already running or being launched cold by the click itself.
  These paths also now cover the app's whole supported format list, not just
  the seven extensions Finder used to offer them for: HEIC, AVIF, TIFF, SVG,
  ICO, BMP, and every RAW format PicFetch reads are all included now, and a
  folder can be dropped on the Dock icon the same as a file.

### Internal

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

**Full Changelog**: https://github.com/frathe/picfetch/compare/v0.2.9...v0.2.10
