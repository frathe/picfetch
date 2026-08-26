# PicFetch — TODOs

## Done

### What's Changed

#### New Features

- When only a single image is selected or dropped,
  the left and right arrow keys now move to the next/previous image in the same folder.

- **Loop duplicate variants after picking one from the grid.**
  Return from the variants grid kept losing the chosen extra to the
  highest-resolution stand-in, and arrows then skipped the rest of the
  group. Committing a variant now inspects that file, wraps arrows inside
  the group, and uses Escape to walk back to the variants grid.

#### Bugfix

- When duplicates are hidden, the highest resolution image will now be selected by default.

#### Internal

- `make release` writes GitHub release notes from this Done section (empty categories dropped) and appends the Full Changelog compare link.

## ACTIVE DEVELOPMENT

## TODO

## not deemed worth implementing (edge cases)

- There is a bug in the Windows Version: WHen in Gridview, multiselect via
  the space key works, but when trying it with mouse and Ctrl key, it does not.
  Holding the Ctrl key down and clicking on an image does not select it but
  instead opens it. Observation, when pushing the Ctrl key at exactly the same time
  as clicking on the image, it actually works, and the image is selected.
  (this seems to be a bug in fyne, created an issue, sorry Windows users)
