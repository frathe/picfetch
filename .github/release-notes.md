## What's Changed

### New Features

- Help -> Release Notes shows the installed release's text offline and offers
  a link to browse all older releases on GitHub. Menu and link labels are
  available in English and German; notes retain their published language.
  Markdown images load online without blocking the window; notes without
  images start no image requests. The post-update What's New window reads
  the same bundled release-notes file.

### Bugfix

![trane security superhero](https://github.com/frathe/picfetch/blob/main/assets/trane/trane_bsod_eyes.png?raw=true)

- Windows Make targets automatically discover the existing portable MinGW
  compiler under `.tools/windows/mingw64/bin`, including from fresh terminals.

- Fixed Windows analysis-cache validation rejecting Fyne's forward-slash paths:
  Favorite representations can be reused and general representations can be
  persisted. Native regression tests cover reopening and stale cleanup.

**Full Changelog**: https://github.com/frathe/picfetch/compare/v1.1.4...v1.1.5
