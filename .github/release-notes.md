## What's Changed

### New Features

 - Copy selection: rectangular image-region copy from the normal viewer
   (Actions -> Copy selection, Opt/Alt+Shift+C).

 - Copy Selection: Cmd/Ctrl+C copies the active image-region selection.

 - Settings: maximum window width, maximum window height, and fixed window
   size now live under Appearance.

### Internal

 - Copy Selection crop/encode uses one `copyselection.Source` pipeline, even
   with the test encoder seam. Raster capture pins the displayed frame instead
   of allocating a duplicate rotation; SVG capture and crop share one oriented
   logical-size definition.

 - Copy Selection yield is enforced at command entry and at the `ShowImage`
   chokepoint. Favorites menu actions use the same host runner, callback
   completeness is guarded by reflection, and a busy-copy refusal now shows
   feedback. `0` yields because it resets rotation; pure zoom keys still keep
   the mode.

 - Settings window is seeded from `settingsState()` (`preferences.State`)
   and sends the snapshot before and after each edit to `ApplySettings`, which
   applies only that patch. Main-window shortcut changes made while Settings
   is open are no longer reverted by the next unrelated form edit.

 - Zoom geometry skips Copy Selection dispatch entirely while the mode is
   inactive and refreshes only the selection chrome while active.

 - `make test` and the race-test phase of `make verify` run in cached
   Linux/amd64 Docker environments, so local golden comparisons match CI.

**Full Changelog**: https://github.com/frathe/picfetch/compare/v0.2.14...v0.2.15
