## What's Changed

### Bugfix

![trane pest control](https://github.com/frathe/picfetch/raw/main/assets/trane/trane_pest_control.png?raw=true)

- Favorites handle large collections more smoothly. Automatic preview loading now
  respects your configured file limit (1,000 by default), reuses previews that
  are already available, and can stop promptly when you switch views.
- When viewing a single image, PicFetch scans nearby files only up to your
  configured limit, keeps them in filename order, and accurately tells you when
  the folder scan was limited.
- Copy Selection now respects the **Max file size (MB)** setting throughout PNG
  creation, reducing memory use while still allowing a failed copy to be retried.
- Release-note images are loaded only from approved, secure GitHub sources.
- Notifications keep their correct appearance as views change, and repeatedly
  opening invalid images no longer causes unnecessary memory retention in very
  large collections.

### Behind the scenes

- Microsoft Store publishing now has stronger safeguards to ensure reviewed
  releases are the ones approved and published.
- **Find more like this** now uses your configured image cache to improve
  responsiveness, while keeping its Favorites and background work independent.

**Full Changelog**: https://github.com/frathe/picfetch/compare/v1.1.5...v1.1.6
