# PicFetch — TODOs

## Done

### Menu Window
Menu points
- viewer
- exif information
- grid view
- picture frame mode
- help
  Windows that are currently showing are grayed out in the menu

## ACTIVE DEVELOPMENT

## TODO

### Menu Actions
Menu points
 - sort with sub menu of sorting options the currently active has a checkmark or is grayed out
 - hide duplicates checkmarked when active (toggable)
 - show variant available it the current item has dupes
 - exif information
Windows that are currently showing are grayed out in the menu

## not deemed worth implementing (edge cases)

- There is a bug in the Windows Version: WHen in Gridview, multiselect via
  the space key works, but when trying it with mouse and Ctrl key, it does not.
  Holding the Ctrl key down and clicking on an image does not select it but
  instead opens it. Observation, when pushing the Ctrl key at exactly the same time
  as clicking on the image, it actually works, and the image is selected.
  (this seems to be a bug in fyne, created an issue, sorry Windows users)
