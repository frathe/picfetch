## What's Changed

### New Features

![sorting_images.png](assets/trane/sorting_images.png)

- Added current app version and build number to Settings > Updates.
- Split Settings into General, Appearance, and Updates tabs so related options
  are easier to find.
- Added manual update checks in Settings: **Check now** bypasses the automatic
  daily gate, shows checking/current/download/error states, verifies and
  stages updates, and offers **Later** or **Perform update** to apply and
  relaunch with What's New on the updated launch.
- Added an **Appearance** choice in Settings with Light, Dark, and System
  default modes. The System default follows operating-system theme changes.
- Added a **Keep a fixed window size** toggle in Settings > General. When it
  is on, PicFetch no longer resizes the window to fit each image, zoom step,
  or rotation; a size you set by dragging the edge is kept across launches.
  Escape back to the welcome screen also leaves that size alone.

### Bugfix

- Windows: if PicFetch lived in a protected folder (Documents, Music, and the 
  like), Controlled Folder Access blocked every in-app update. The update now 
  applies without going through cmd.exe, so that block is gone.
  If Windows still refuses the write (the app isn't signed), you get a message
  on the next launch with a button to the download page instead of a silent fail. 
  That dialog actually fits the window, and German text no longer shows garbage 
  characters.

### Internal

- Made macOS native-menu title reads return caller-owned C strings, removing
  shared-buffer aliasing and truncation. Native-menu test fixtures now release
  their retained AppKit menus after each test.

**Full Changelog**: https://github.com/frathe/picfetch/compare/v0.2.12...v0.2.13
