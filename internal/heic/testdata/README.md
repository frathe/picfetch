# Authored HEIC fixtures

These images were created for PicFetch from the literal mathematical pattern in
`generate.py`, under the repository's MIT license (copyright 2026 Florian Rathe).
No external photograph, bundled codec, or borrowed test image is used.

The source is 64 by 64 pixels with a black left half and white right half,
neutral chroma, BT.709 primaries and matrix, and the sRGB transfer function.
Limited-range luma is 16/235 for 8-bit and 64/940 for 10-bit. HEVC encoding uses QP 1 and explicit Main/Main10 profiles. Independently expected RGBA is black `(0,0,0,255)` and white
`(255,255,255,255)` away from the border; tests allow two integer levels for
color conversion rounding. Capability checks decode both bit depths.

Reproduce with `python3 internal/heic/testdata/generate.py`. It invokes the
already installed Ubuntu FFmpeg `6.1.1-3ubuntu5` with libx265 `3.5-2build1`,
records one HEVC sample in a temporary MP4, then creates a minimal HEIF
container carrying the sample and its `hvcC` configuration. The MP4 and raw
source bytes are temporary; only generated still fixtures are retained. The
script contains the complete FFmpeg argument vector and container construction.
Neither encoder binary is shipped. Fixture construction is a maintenance action;
ordinary tests do not execute encoders or install anything.

| File | Contract | SHA-256 |
| --- | --- | --- |
| `probe8.heic` | Full designated 8-bit primary | `ae5080bcd30e1c54de08b32e16a56c88273dd957647077b2ce48dc4a6ee3f3cf` |
| `probe10.heic` | Full designated 10-bit primary | `dcb0e856cbc8aea3c9286d049803d7c6eb3b1b24fdd7ee8fc2bfbebaf0e85d97` |
| `container-rotate.heic` | `irot=1`: white top, black bottom | `fb447a4c1d80c3a854ed70da1d263fdd523523ce1928084d931987cb1b882b36` |
| `exif-rotate.heic` | EXIF orientation 6, no container rotation: black top, white bottom | `7db8fe805930464a4ff3839c1a929996f4620e9760b318067600e7db11971708` |
| `container-and-exif.heic` | Container rotation wins over contradictory EXIF: white top, black bottom | `7d30b1f370993628e89bf88ff8160ce6a580681cee179a191f89973203ccd660` |

The two probe files are embedded as runtime capability fixtures. Remaining files
are native test inputs. The native tests mutate owned container fields to check
sequence/AVIF refusal, native handling of wide-gamut/HDR declarations, malformed
box sizes and oversized dimensions. Color management is delegated to the installed
OS provider; these tests make no exact ICC/HDR conversion promise. A native pass
qualifies only the provider and platform actually exercised.

The final `hvcC.general_profile_idc` values are 1 (Main) and 2 (Main10).
`keyint=2` prevents x265 from selecting an Intra Range Extensions profile for
the single encoded frame. Initial lossless/keyint=1 experiments selected profile
4; those bytes were replaced before handoff. `info=0` omits encoder-info SEI.

`generate_primary.py` adds `nonfirst-primary.heic` for primary selection. It
contains two equal-size images: item 1 is black/white, item 2 is white/black,
and `pitm` selects item 2. The independent expected primary is therefore
white/black. Its SHA-256 is
`c5718b8677d2598660913a657f11d6dff3a83507bce4399683951c65a0216525`.
Linux native decoding confirms that oracle; the Windows WIC experiment uses it
to investigate the undocumented relationship between item IDs and frame indices.

## Authored color, grid, transform, and alpha corpus

Run `python3 internal/heic/testdata/generate_corpus.py` to reproduce the additional
corpus with the same FFmpeg/x265 versions. It creates all source pixels from
literal values and reuses only the two original probe samples for ICC-tagged
grayscale fixtures. Original fixture bytes are unchanged. Source and generated
images have the repository's MIT license; no external artwork is included.

The colored source has four equal quadrants, in reading order:
`(192,32,64)`, `(32,160,64)`, `(32,64,192)`, `(160,128,32)`. The generator computes
limited-range BT.709 YCbCr directly from these RGB values, writes 8-bit or 10-bit
planar 4:2:0 samples, and encodes Main/Main10 at QP 1. Pixel tests sample well
inside each solid quadrant and allow three integer levels for quantization and
color conversion. The expected colors come from the source constants, not from
another decoder's output.

Each grid has four independently encoded, hidden 64-by-64 tiles with those
colors. Its designated primary is item 5, a 128-by-128 `grid` item, following the
tiles in item order. Its descriptor is version 0, flags 0, rows-minus-one 1,
columns-minus-one 1, then two 16-bit output dimensions; `dimg` references items
1 through 4. Geometry and all four output tile centers are checked. This is an
ordinary even-size image, not a malformed-input or decoder-crash reproduction.

Mirror fixtures set `imir=1` (horizontal reflection). `mirror-rotate8` associates
`irot=1` before `imir=1`, giving quadrant order 4, 2, 3, 1 after counterclockwise
rotation and horizontal reflection. Installed libheif 1.17.6 renders both 8-bit
cases but explicitly rejects the 10-bit mirror operation. That provider failure
must remain visible in qualification evidence; it is not a successful rendition.

Alpha fixtures pair a color primary with an auxiliary image identified by
`auxC` URN `urn:mpeg:hevc:2015:auxid:1`; `auxl` references the primary from the
auxiliary image. Alpha values in reading order are 0, 128, 192, and 255, encoded
as full-range luma with neutral dummy chroma. Straight-color fixtures use
`(160,80,40)` throughout. Premultiplied fixtures encode each color channel as
`round(channel * alpha / 255)` and add a `prem` reference from the primary to the
alpha image. Native output must be straight RGBA: alpha tolerance is two levels,
and nontransparent color tolerance is six levels, accounting for quantization
amplified by unpremultiplication. RGB under zero alpha is deliberately unspecified.

ICC fixtures carry real serialized profiles, not placeholder bytes.
`icc-srgb8/10` wrap the original grayscale samples in profiles produced with
`cmsCreate_sRGBProfile`. `icc-p3-linear8/10` carry the colored samples and a profile
created with `cmsCreateRGBProfile`, D65 white point `(0.3127,0.3290)`, Display P3
primaries `(0.68,0.32)`, `(0.265,0.69)`, `(0.15,0.06)`, and linear tone curves.
Tests require successful full decode, opaque alpha, and recognizable color
quadrants; they do not require matching a separate color-management engine.

Generation uses the already installed Ubuntu Little CMS `2.14-2ubuntu0.1`
library at `/lib/x86_64-linux-gnu/liblcms2.so.2`. It is a maintenance-time tool
only, not a PicFetch runtime dependency or bundled binary. Little CMS is MIT
licensed; its profile creator writes the profile copyright tag "No copyright,
use freely". The generated ICC timestamp is pinned to 2026-09-22 for reproducible
bytes. No Little CMS implementation source is copied into this repository.

Primary references for container and profile construction:

- [libheif v1.17.6 grid descriptors, alpha relationships and alpha decode](https://github.com/strukturag/libheif/blob/v1.17.6/libheif/context.cc)
- [libheif v1.17.6 `auxC`, `imir`, and item references](https://github.com/strukturag/libheif/blob/v1.17.6/libheif/box.cc)
- [libheif v1.17.6 transform order](https://github.com/strukturag/libheif/blob/v1.17.6/libheif/file.cc)
- [libheif v1.17.6 8-bit mirror limitation](https://github.com/strukturag/libheif/blob/v1.17.6/libheif/pixelimage.cc)
- [Little CMS 2.14 profile creation and profile copyright tag](https://github.com/mm2/Little-CMS/blob/lcms2.14/src/cmsvirt.c), [public API](https://github.com/mm2/Little-CMS/blob/lcms2.14/include/lcms2.h), [license](https://github.com/mm2/Little-CMS/blob/lcms2.14/COPYING)
- [W3C CSS Color 4 Display P3 definitions and conversion data](https://www.w3.org/TR/css-color-4/#color-conversion-code)

| File | SHA-256 |
| --- | --- |
| `color8.heic` | `d20cbd2ce9f3751cd55921a7cf2dce7cabe8475a60b28c26bf44b16d6dfa7179` |
| `color10.heic` | `827aeeae6b9682fc3b2eb9aa93f78e16e81d7060cc61a7ccf207ac8387b00ae4` |
| `grid8.heic` | `0ba8c9fabd14115b81363174a1e3d12e7d66a0c98c35fa79df6f5ff4a86e4304` |
| `grid10.heic` | `86b9f460c0c36bf914d68a31e49d4430769a9d22f8d96d7c854d9d45f2e60015` |
| `mirror-horizontal8.heic` | `cd969b3ab80d4b661eaa6c1b585987921802a780be80b2b6d6a955d84577c85e` |
| `mirror-horizontal10.heic` | `d4c375d8244aa33953d5296148b1045f6ff85a641f18b921685b16e9cc8ff8ac` |
| `mirror-rotate8.heic` | `bec76a809df67c5dce7444fa1327b1c54771f6702a4ff14eb87d02f68f7e33a6` |
| `alpha-straight8.heic` | `037a9f4f3b80e1eb6cf34015da36181d22260f4be469db7ed174e15ba485337e` |
| `alpha-straight10.heic` | `5cbeb6fb313c2b5633a7e196779b389daf0450159d01b03a0d489f9dbe957645` |
| `alpha-premultiplied8.heic` | `4e0de28954e12f0e308bf459fce8fd365cb6ecaa5f05e24e11284237e495e638` |
| `alpha-premultiplied10.heic` | `e27c4eab021508c3b99b66f0c8ee84904e69f36a6ee60c857fcde70e46005c4e` |
| `icc-srgb8.heic` | `9a6b65b673ef5a2d1463e88687ddbf5cfcde98197b64809b01323a5f4682148c` |
| `icc-srgb10.heic` | `2d63e03e06482e8c62c584899425d1bae2e4c9ecdce8cc8cd6dc401706a43c9e` |
| `icc-p3-linear8.heic` | `a10746e1975242021bf31f65527a53d3a6800e1e7d4708d2639d8564f5a20863` |
| `icc-p3-linear10.heic` | `8e6fb2b758e6ce3fdf5fe3a39451a653d9582bf5fc96eb3f65cb68ed7f27a201` |
