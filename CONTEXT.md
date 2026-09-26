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

**Requested image**:
The image chosen for the current loading attempt, which may not yet supply
the content of the single-image view.
_Avoid_: Displayed image, current image

**Displayed image**:
The image whose content supplies the single-image view, including its current
animation frame and view-only rotation. It can remain while another requested
image is loading.
_Avoid_: Requested image, current file

**Image capture**:
Image content retained in a fixed orientation for a particular operation,
independently of later changes to the single-image view.
_Avoid_: Screenshot, live displayed image

**HEIC support check**:
A check of whether the current system can supply HEIC decoding for PicFetch,
available on request from Settings. A successful check does not promise that
every HEIC file can be decoded.
_Avoid_: Codec installation, file validation

**HEIC installation guide**:
In-app instructions for obtaining HEIC decoding support on the current
operating system.
_Avoid_: Installer, support check

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

**Metadata removal**:
Rewriting an original image file in place so its identifying metadata and
secondary media are gone while the primary image remains.
_Avoid_: Strip, scrub, sanitize

**Primary image**:
The main photograph retained by JPEG metadata removal, as distinct from its
embedded previews, additional pictures or motion-photo content.
_Avoid_: Displayed image (a viewing state), first preview

**Secondary media**:
Previews, additional pictures, video or audio stored alongside the primary
image within the same file.
_Avoid_: Metadata (too broad), other files

**Metadata omission**:
Writing an exported copy without the source's identifying tags. The source
keeps everything it had.
_Avoid_: Strip, remove, sanitize, metadata removal

**Export size limit**:
The longest-edge ceiling applied to an exported copy. A photo already inside
the ceiling is exported at its own size, never enlarged to meet it.
_Avoid_: Resize, max dimension, scale, downsample

**Microsoft Store edition**:
The edition of PicFetch installed and updated through Microsoft Store.
_Avoid_: Windows edition (PicFetch also has portable Windows downloads)

**Content similarity**:
Relatedness in what images depict, including across different visual media
or styles.

**Search result**:
The ordered matching images that form the active list for browsing and file
actions during Find more like this exploration.
_Avoid_: Search origin, search scope

**Search origin**:
The initial image list and browsing visit retained for return when search
results replace the active list during exploration.
_Avoid_: Search result, oldest result

**Search scope**:
The original collection of images against which every reference in one
exploration is compared, regardless of the active search result.
_Avoid_: Search result, Grid result

**Analysis cache**:
Reusable per-image analysis retained on disk for later content-similarity
exploration, owned by a Favorite or the general cache for loose image lists.
_Avoid_: Search history, saved search result, thumbnail cache

**Image cohort**:
A group of images related under the map's chosen grouping criterion.

**Cohort pile**:
The loose arrangement of sampled images that represents an image cohort on
the explorer map. The sample represents the cohort's contents, not its full
membership.
_Avoid_: Image cohort (the group itself), mosaic

**Location Map**:
A geographic view of the loaded collection using recorded GPS coordinates and
honoring duplicate hiding. A duplicate group's highest-resolution image can be
located using coordinates recorded by another member of that group.
_Avoid_: Similarity map, explorer map

**Location cluster**:
A group of mapped images represented by one counted pin because their
positions are too close to display separately at the current map zoom.
_Avoid_: Duplicate group, image cohort

**Location source**:
The image whose recorded GPS coordinates position an image on Location Map.
It can be a hidden duplicate of the image shown by the map.
_Avoid_: Displayed image, inferred location

**GPS cache**:
Remembered per-image location metadata used by Location Map, retained on disk
for Favorite collections and in memory for live collections.
_Avoid_: Tile cache, thumbnail cache, analysis cache

**Trane**:
PicFetch's dog mascot, including the character on the welcome screen.

**Finis**:
The hidden dog companion shown in a separate window, whose gaze follows the
pointer.
_Avoid_: Trane

**Hypno Spiral**:
PicFetch's hidden animated spiral scene, which can carry images from the
loaded collection.
_Avoid_: Picture-frame mode, Finis

**Mascot-circle gesture**:
A sequence of pointer revolutions around Trane's or Finis's head, forming
one step in discovering Finis or the Hypno Spiral clue.
_Avoid_: Spiral window gesture (which moves the application window)
