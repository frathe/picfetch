# PicFetch — TODOs

## Done

### What's Changed

#### New Features

#### Bugfix

- Reject zip/tar entries that are not `filepath.IsLocal` during update extract (GitHub CodeQL go/zipslip).

#### Internal

- Drop deprecated `tar.TypeRegA` in update extract (still accept the historic NUL regular-file typeflag).
- Install CI's Linux GUI packages in CodeQL so `internal/winpos/linux.go` can compile.

## ACTIVE DEVELOPMENT

## TODO

## not deemed worth implementing (edge cases)

- There is a bug in the Windows Version: WHen in Gridview, multiselect via
  the space key works, but when trying it with mouse and Ctrl key, it does not.
  Holding the Ctrl key down and clicking on an image does not select it but
  instead opens it. Observation, when pushing the Ctrl key at exactly the same time
  as clicking on the image, it actually works, and the image is selected.
  (this seems to be a bug in fyne, created an issue, sorry Windows users)
