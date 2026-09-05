# PicFetch

PicFetch is a desktop image-viewing context in which a user can act on either
files in Grid View or pixels in the single-image viewer.

## Language

**Grid selection**:
A set of image files selected in Grid View as the subject of a batch action.
_Avoid_: Selection, image selection

**Image-region selection**:
A single rectangular area of displayed image content, expressed in the
image's oriented coordinate space.
_Avoid_: Crop, grid selection

**Copy Selection mode**:
A transient single-image-viewer mode for defining an image-region selection
to copy as image data.
_Avoid_: Crop mode, screenshot mode

**Picture-frame mode**:
A full-screen single-image-viewer mode that advances through the file set on
a timed interval.
_Avoid_: Slideshow, kiosk mode (the `--slideshow` launch flag keeps the
common word on purpose - do not rename it to match this entry)

**Kiosk mode**:
Unclaimed. A locked appliance state that suppresses the app's own exits is
not a PicFetch concept - picture-frame mode can always be left.
_Avoid_: Using this as another name for Picture-frame mode

**Linked comparison**:
A two-photo comparison state where view adjustments affect both photos
together.
_Avoid_: Locked comparison

**Unlinked comparison**:
A two-photo comparison state where view adjustments affect one targeted photo
without changing the other.
_Avoid_: Unlocked comparison

**Grid result**:
The complete set of image files currently represented by Grid View after its
active filtering, including files outside the visible scroll area.
_Avoid_: Visible cells, current files

**Target display**:
The attached display chosen as a mosaic's native-pixel output and wallpaper
destination.
_Avoid_: Screen, monitor, default display

**Source pool**:
The fixed set of images available to supply cards for one mosaic generation.
_Avoid_: Sources, image list, Grid result
