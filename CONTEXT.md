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

**Linked comparison**:
A two-photo comparison state where view adjustments affect both photos
together.
_Avoid_: Locked comparison

**Unlinked comparison**:
A two-photo comparison state where view adjustments affect one targeted photo
without changing the other.
_Avoid_: Unlocked comparison
