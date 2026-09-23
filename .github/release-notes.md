## What's Changed

### New Features

- *delegate heic image rendering to the OS*
  Add back support for HEIC image formats, on start check if the system supports rendering of heic images. if yes save
  that information to the settings so we don't have to check that on every launch. when the os does support the 
  rendering we delegate the rendering to the OS and enable HEIC support.

### Bugfix

- Enable Windows HEIC decoding through installed Microsoft extensions, preserving
  primary-image selection, EXIF orientation, metadata and transparency. Keep
  decoder workers hidden and enforce memory, CPU and child-process limits.
- Refresh Explorer test fixtures after the HEIC analysis-facts version change.
- Restore macOS HEIC camera metadata and refresh previously cached image facts
  without repeating similarity inference.

### Internal

- Close the Windows/Store HEIC pre-release qualification items by maintainer
  decision (2026-09-23). Native x64/ARM64 evidence, installed MSIX behavior,
  codec installation/recheck recovery and final-package qualification are
  accepted as deferred to the maintainer's testing after rollout. This records
  acceptance of the deferral, not successful execution of the outstanding tests.

- Create a three-minute 1080p PicFetch feature promo with original electronic
  music, Trane artwork, 22 animated scenes and verified media output. see the
  [production record](finished_refactorings/2026-09-22-picfetch-promo.md).

- Accept Go build diagnostics in the native qualification runner while still
  rejecting build failures and missing or skipped required tests.

**Full Changelog**: https://github.com/frathe/picfetch/compare/v1.1.8...v1.1.9
