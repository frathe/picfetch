## What's Changed

### New Features

- Added the Microsoft Store MSIX delivery path: exact reserved package
  identity, generated x64/ARM64 manifests and assets, a Store-managed build
  mode that disables GitHub self-updates, Windows CI packaging/WACK checks,
  localized listing copy, and an explicit privacy disclosure for the optional
  OpenStreetMap EXIF-location view.

### Internal

- Unified GitHub and Microsoft Store release versioning for the 1.0 launch:
  `FyneApp.toml` is the single public version source, and MSIX appends only its
  Store-reserved fourth zero component.

- Reduced the measured median reusable CI gate from 22:17 to 7:52 while
  retaining Linux race coverage, native Windows tests, validation, and release
  failure gating. The final topology can use six concurrent required runners
  instead of two and raises median runner time from 23:26 to 30:43 (+31.1%).
  Its exact manifest proves UI coverage and non-overlap, but no one process now
  exercises the whole UI suite's ordering; see the
  [completed measurement record](finished_refactorings/2026-09-03-measured-ci-test-sharding.md).

**Full Changelog**: https://github.com/frathe/picfetch/compare/v0.2.17...v1.0.0
