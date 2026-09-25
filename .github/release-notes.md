## What's Changed

### New Features

- Browse GPS-tagged photos with Window -> Location Map (`Shift+L`). Explore
  geographic clusters, open their exact members in Grid View, and return to the
  same map position. Location scanning is progressive and reuses completed facts;
  saved Favorites can retain their own location cache.
- Use arrow keys to select the nearest map photo or cluster, with a highlighted
  border, and Enter to open it. Offscreen targets are brought into view;
  Shift+arrow keys pan without changing selection.
- Show duplicate-check progress while Location Map or Similarity Explorer waits
  for grouping to finish.
- OpenStreetMap tiles now follow dark appearance through a local display filter
  in both Location Map and the EXIF Location panel, without changing provider or
  photo colors.

- *delegate heic image rendering to the OS*
  Add back support for HEIC image formats, on start check if the system supports rendering of heic images. if yes save
  that information to the settings so we don't have to check that on every launch. when the os does support the 
  rendering we delegate the rendering to the OS and enable HEIC support.

### Bugfix

- Keep loaded map tiles and photo previews visible together while dragging, and
  replace the background only when the next tile scene is complete. Correct
  live Light/Dark panel backgrounds in Location Map, Grid View and Similarity
  Explorer.

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
