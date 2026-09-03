# PicFetch privacy policy

Effective date: September 3, 2026

PicFetch is a free and open-source desktop image viewer. It has no accounts,
advertising, analytics, or telemetry. The PicFetch developer does not collect,
store, sell, or share personal information from the app.

## Local files and metadata

Images and their metadata are opened and processed on the user's device.
PicFetch does not upload images or their EXIF metadata to the developer or to a
PicFetch service.

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
