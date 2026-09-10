# PicFetch privacy policy

Effective date: September 10, 2026

PicFetch is a free and open-source desktop image viewer. It has no accounts,
advertising, analytics, or telemetry. The PicFetch developer does not collect,
store, sell, or share personal information from the app.

## Local files and metadata

Images and their metadata are opened and processed on the user's device.
PicFetch does not upload images or their EXIF metadata to the developer or to a
PicFetch service.

Visual Similarity Explorer analysis also runs locally. Image representations,
suggested tags, previews, saved cohorts, and optional Favorite analysis caches
stay on the device. They are not sent to a model provider or feedback service.

PicFetch can display GPS coordinates already embedded in an image. The Location
map is collapsed by default, and no map request is made merely by opening the
image or its EXIF information.

## OpenStreetMap location map

When the user explicitly expands the Location section for an image containing
GPS coordinates, PicFetch requests the map tiles needed to display that area
from `tile.openstreetmap.org`. Those requests disclose the requested map area,
the device's IP address, and a PicFetch user-agent string to the OpenStreetMap
tile service. PicFetch does not send the image, filename, or other EXIF fields.

The OpenStreetMap Foundation processes those requests under its own
[privacy policy](https://osmfoundation.org/wiki/Privacy_Policy). The map can be
avoided entirely by leaving the Location section collapsed.

## Visual Similarity Explorer downloads

On first use, the Explorer explains its local analysis and offers to download
the required model and runtime. Downloads begin only when the user chooses
Download. PicFetch obtains the pinned SigLIP 2 model from Hugging Face and the
ONNX Runtime package from Microsoft's ONNX Runtime releases on GitHub. These
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
locally. Once the required files are installed, image analysis works offline;
the analysis worker has network access disabled. Declining the download leaves
the rest of the image viewer available.

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
