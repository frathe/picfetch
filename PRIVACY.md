# PicFetch privacy policy

Effective date: September 25, 2026

PicFetch is a free and open-source desktop image viewer. It has no accounts,
advertising, analytics, or telemetry. The PicFetch developer does not collect,
store, sell, or share personal information from the app.

## Local files and metadata

Images and their metadata are opened and processed on the user's device.
PicFetch does not upload images or their EXIF metadata to the developer or to a
PicFetch service.

Similarity Explorer analysis also runs locally. Image representations,
suggested tags, previews, saved cohorts, and optional Favorite analysis caches
stay on the device. They are not sent to a model provider or feedback service.

Explorer communicates with the viewer through local process pipes. It does not
open a localhost server or expose a model API on a network port. PicFetch
disables ONNX Runtime telemetry before creating an analysis session; if the
telemetry opt-out fails, analysis does not start.

PicFetch can display GPS coordinates already embedded in images. Reading these
coordinates is local. Opening an image or its EXIF information does not by itself
request map tiles; the EXIF Location section is collapsed by default.

## OpenStreetMap location map

PicFetch uses `tile.openstreetmap.org` for both the collection-wide **Location
Map** and the **Location** section of an image's EXIF window. Tile requests begin
when the user opens Location Map, or expands the EXIF Location section for an
image with GPS coordinates. Panning, zooming, resizing, or navigating while a map
is open can request additional tiles. The EXIF map also loads nearby tiles around
the image's location.

Requests contain map tile coordinates (zoom level, column, and row), not the
original GPS metadata. They nevertheless disclose the map areas being viewed,
which can reveal or approximate where images were taken. The tile service also
receives the device's IP address, a PicFetch user-agent string, and ordinary HTTP
connection information. PicFetch does not send images, previews, filenames,
local paths, or other EXIF fields with tile requests.

Downloaded map tiles are cached in memory. The collection map also reads image
metadata locally and can save versioned metadata, including GPS coordinates and
source paths, in local caches for saved Favorites. These records are not uploaded.
The dark-mode filter for both maps changes map colors locally; it does not change
the provider, upload photos, or request a separate dark map service.

The OpenStreetMap Foundation processes those requests under its own
[privacy policy](https://osmfoundation.org/wiki/Privacy_Policy). To avoid OSM tile
requests, do not open Location Map and leave the EXIF Location section collapsed.
Leaving the collection map, collapsing the EXIF section, or closing its window
cancels that map's pending tile work; requests already sent cannot be recalled.

## Similarity Explorer downloads

On first use, the Explorer explains its local analysis and offers to download
the required model data. Standalone builds also download the runtime; the
Microsoft Store version includes it, as described below. Downloads begin only
when the user chooses Download. PicFetch obtains the pinned SigLIP 2 model from
Hugging Face and, in standalone builds, the ONNX Runtime package from Microsoft's
ONNX Runtime releases on GitHub. These
services may redirect downloads through their content-delivery providers.

As with other internet downloads, the services receive the device's IP address,
the requested asset URL, and ordinary connection/request information, including
a PicFetch user-agent string. Requests do not contain images, filenames, local
paths, EXIF metadata, image representations, or a PicFetch user identifier.
PicFetch does not require an account or send analytics or usage reports.

Hugging Face and GitHub handle download requests under their own
[Hugging Face privacy policy](https://huggingface.co/privacy) and
[GitHub privacy statement](https://docs.github.com/en/site-policy/privacy-policies/github-general-privacy-statement).
PicFetch verifies the downloaded files against pinned checksums and stores them
locally. Once the required files are installed, image analysis works without
an internet connection. Declining the download leaves the rest of the image
viewer available.

The Microsoft Store version includes the ONNX Runtime DLLs and their notices
in its x64/ARM64 package. Microsoft Store installs and updates those DLLs and
the required Microsoft C++ runtime framework. Explorer setup in that version
downloads only the model and processor configuration from Hugging Face (about
372 MB); it does not download executable code from GitHub or place runtime DLLs
in the model cache.

On supported macOS and Linux systems, the operating system blocks the analysis
worker's network access. On Windows 11, analysis runs in a normal local worker
process with ONNX Runtime telemetry disabled. PicFetch does not configure an
operating-system network block for that Windows process. Its lack of a listening
port does not prevent it from making outgoing connections; local processing and
the telemetry opt-out are not a guarantee of network isolation. The Windows
setup page explains this difference before analysis starts.

## Voluntary community discussions

The GitHub Discussions link opens
[PicFetch's public discussion page](https://github.com/frathe/picfetch/discussions)
in the user's browser. PicFetch does not submit a feedback form or attach images,
logs, system information, or usage data to that link. Users choose whether to
post and what to share. Browser visits and posts are handled by GitHub under its
privacy statement; public posts can be read by other people.

## Updates

The Microsoft Store build uses Microsoft Store delivery and updates. In other
distributions, users can optionally enable PicFetch's update checker; when
enabled, it contacts GitHub to check and download PicFetch releases. GitHub
processes those requests under its own
[privacy statement](https://docs.github.com/en/site-policy/privacy-policies/github-general-privacy-statement).

## Contact

Questions or requests about this policy can be submitted through
[PicFetch's public issue tracker](https://github.com/frathe/picfetch/issues).

This policy may be updated when PicFetch's data-handling behavior changes. The
current version is maintained with the PicFetch source code.
