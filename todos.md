# PicFetch — TODOs

## Done

- possibility to drag a selection rectangle in grid view to select multiple images.

## ACTIVE DEVELOPMENT

## TODO

- when only a single image is selected or dropped
  the left and right arrow keys should move to the next/previous image in the same folder.

- if duplicates are hidden, the highest resolution image should be shown.

- if variants a dupe are shown in the grid and one of the variants is selected with return
  the selected variant should be shown in the viewer and not the highest resolution image.
  when using the arrow keys in this view it should loop over the variants.
  ESC -> back to variants grid -> ESC -> back to normal grid.

## not deemed worth implementing (edge cases)

- There is a bug in the Windows Version: WHen in Gridview, multiselect via
  the space key works, but when trying it with mouse and Ctrl key, it does not.
  Holding the Ctrl key down and clicking on an image does not select it but
  instead opens it. Observation, when pushing the Ctrl key at exactly the same time
  as clicking on the image, it actually works, and the image is selected.
  (this seems to be a bug in fyne, created an issue, sorry Windows users)
