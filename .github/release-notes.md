## What's Changed

### New Features

- When only a single image is selected or dropped,
  the left and right arrow keys now move to the next/previous image in the same folder.

- **Loop duplicate variants after picking one from the grid.**
  Return from the variants grid kept losing the chosen extra to the
  highest-resolution stand-in, and arrows then skipped the rest of the
  group. Committing a variant now inspects that file, wraps arrows inside
  the group, and uses Escape to walk back to the variants grid.

### Bugfix

- When duplicates are hidden, the highest resolution image will now be selected by default.

### Internal

- `make release` writes GitHub release notes from this Done section (empty categories dropped) and appends the Full Changelog compare link.

**Full Changelog**: https://github.com/frathe/picfetch/compare/v0.2.7...v0.2.8
