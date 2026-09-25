## What's Changed

### New Features

Explore your photos by where they were taken. Open **Window -> Location Map**
(`Shift+L`) to see photos with location information on an interactive map.
Nearby photos are grouped together: open a group to browse all its pictures,
then return to the same map position. Use the arrow keys to select a photo or
group, Enter to open it, and Shift+arrow keys to move around the map. Zoom with
`+`/`-` or press `0` to fit everything. The map follows your light or dark theme,
keeps photos visible while dragging, and shows progress while checking duplicates.

![Trane exploring a map of Europe with photo pins](https://raw.githubusercontent.com/frathe/picfetch/9521edb7087c1d229355419e3d3ce0f5299089fc/assets/trane/trane_europe_map.png)

- The EXIF Location panel also follows dark mode without changing photo colors
  or the OpenStreetMap provider.
- Similarity Explorer now shows progress while checking duplicate groups.
- HEIC photo support is back on systems with a compatible built-in decoder.
  PicFetch checks availability and remembers the result for later launches.

### Bugfix

- Reduce map flicker when dragging by keeping the current background and photos
  visible until the next section of the map is ready.
- Switching between light and dark mode now updates panel backgrounds correctly
  in Location Map, Grid View and Similarity Explorer.

- Open HEIC photos on Windows using installed Microsoft image extensions, with
  the correct photo, orientation, camera details and transparency. Photo loading
  runs in the background with resource limits.
- Restore camera details for HEIC photos on macOS and refresh previously saved
  information without repeating the slower similarity analysis.

### Internal

- Additional checks of Windows HEIC support and Microsoft Store installation
  are deferred to testing after release; these checks are not yet complete.
- Update automated tests for HEIC photos in Similarity Explorer.

- Add a three-minute video showcasing PicFetch with Trane, animated scenes and
  original music. See the [production notes](https://github.com/frathe/picfetch/blob/main/finished_refactorings/2026-09-22-picfetch-promo.md).

- Improve release checks so harmless compiler messages do not flag a working
  build as broken, while actual build failures and missing tests are still caught.

**Full Changelog**: https://github.com/frathe/picfetch/compare/v1.1.8...v1.1.9
