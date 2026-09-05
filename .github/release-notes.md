## What's Changed

### New Features

![Trane building a mosaic](https://raw.githubusercontent.com/frathe/picfetch/main/assets/trane/trane_mosaic.webp)

- **Create an Image Mosaic from Grid View.** Turn your selected images—or all images matching your current filters—into a collage sized to fill your screen. Everything is generated locally on your device.
- Generate a new arrangement, save it as a PNG or JPEG, or set it as your wallpaper.
- On Windows and macOS, choose which monitor gets the wallpaper. On Linux, you can save the mosaic or use the existing wallpaper option, but targeting a specific monitor isn’t supported.
- Use the mosaic controls with a keyboard.
- Choose **Start Over** from the finished preview to create another mosaic with your settings preserved—handy for setting up another monitor.
- Customize the look, including optional drop shadows, under **Advanced**.
- Added the hidden Finis companion: type `finis` in manual search, then move the
  pointer in his window to guide his gaze. Escape closes the companion.

### Fixes and Improvements

- Fixed mosaic controls that appeared disabled or didn’t respond properly.
- Fixed overlapping text while generating a mosaic.
- Reduced excessive image overlap so more of each photo stays visible.
- Smoothed jagged borders and rotated image edges while keeping photos sharp.
- Improved image handling: photos follow their saved orientation, all unique images are used before any repeat, and selected duplicates use the highest-resolution version available in Grid View.
- Improved sizing for high-resolution Mac displays and handling of monitor changes before generation.
- Fixed target selection for identical monitors and translated fallback display names.
- Disconnected Windows monitors no longer block mosaic creation or accept wallpaper changes.
- Bounded the memory retained by repeated mosaic sources and preserved the full canvas of partial-frame GIFs.
- Removed artificial fading where rotated photos extend past the canvas boundary.

### Behind the Scenes

- Improved error reporting and cleaned up internal code and documentation to make the mosaic feature easier to maintain.

**Full Changelog**: https://github.com/frathe/picfetch/compare/v1.0.0...v1.0.1
