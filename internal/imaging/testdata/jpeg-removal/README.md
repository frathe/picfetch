# Synthetic JPEG removal references

`reference.py` generates small fixtures from formulas in this directory's source
and checks ICC color transforms with an independent LittleCMS implementation.
It does not import PicFetch code. Generated files belong in a temporary directory
until deliberately selected for test coverage.

```sh
python3 internal/imaging/testdata/jpeg-removal/reference.py generate /tmp/picfetch-jpeg-reference
python3 internal/imaging/testdata/jpeg-removal/reference.py self-check /tmp/picfetch-jpeg-reference
python3 internal/imaging/testdata/jpeg-removal/reference.py check ORIGINAL.icc OUTPUT.icc
```

`self-check` regenerates the corpus and compares each of the four profiles with
itself. `check` opens both supplied profiles and independently transforms the
same samples to XYZ doubles using all four ICC intents: perceptual, relative
colorimetric, saturation and absolute colorimetric. RGB samples cover the full
343-point Cartesian grid of 0, 0.1, 0.25, 0.5, 0.75, 0.9 and 1. Gray samples
cover 257 equally spaced values from 0 through 1. Transform optimization is
disabled. Unreadable profiles, unsupported or different input color spaces,
failed transforms, non-finite output and a maximum absolute XYZ difference
greater than `1e-7` fail the command. Passing this finite sample check provides
reference evidence; it is not proof of equivalence at every possible input.

The generator emits these 16 x 12 pixel JPEGs, using quality 90:

| File | Encoding |
| --- | --- |
| `baseline-rgb.jpg` | Baseline, one scan, synthetic RGB source encoded as YCbCr |
| `progressive-rgb.jpg` | Progressive, synthetic RGB source encoded as YCbCr |
| `multiscan-rgb.jpg` | Sequential, one full coefficient scan per Y/Cb/Cr component |
| `baseline-gray.jpg` | Baseline grayscale, one scan |
| `progressive-gray.jpg` | Progressive grayscale |

The original patterned PPM/PGM files and sequential scan script are also emitted.
Five additional `entropy-*.jpg` fixtures use the same formulas at 35x27 pixels
and quality 90, covering partial MCU edges and exact entropy boundaries:

| File | Encoding |
| --- | --- |
| `entropy-baseline-420-restart1.jpg` | Baseline 4:2:0, restart every MCU |
| `entropy-progressive-420.jpg` | Progressive 4:2:0, no restarts |
| `entropy-progressive-444-restart1.jpg` | Progressive 4:4:4, restart every MCU |
| `entropy-multiscan-444-restart1.jpg` | Separate-component sequential 4:4:4, restart every MCU |
| `entropy-progressive-gray-restart1.jpg` | Progressive grayscale, restart every MCU |

`reference.py generate` reproduces all ten JPEGs; restart options use `-restart
1B`. These fixtures pass the pinned `djpeg -strict` and Go 1.27 decoder. They
exercise restart cycling, progressive EOB/refinement, ordinary stuffed bytes,
and legal final `FF00` immediately before scan/restart markers. Public tests
require exact clean output and reject additional bytes at every interval end.
Exploratory progressive/separate-component 4:2:0 fixtures with restart every
MCU pass `djpeg -strict` but fail the Go decoder; they are outside this operation's
qualified set and are not part of the committed supported corpus.

JPEGs contain no embedded ICC profile; tests can insert each applicable profile
and metadata independently. `rgb-v2.icc` and `rgb-v4.icc` are LittleCMS-generated
sRGB matrix/TRC profiles. `gray-v2.icc` and `gray-v4.icc` are D50 gray gamma 2.2
profiles. ICC versions are 2.1 and 4.3, respectively. All profiles contain a
PicFetch description and copyright text to exercise removal of descriptive
fields. Their creation date is fixed to 2026-09-17 for reproducibility.

## Tools, provenance and licenses

Generation and checking were prepared with these explicit locally installed
development tools; the script deliberately does not resolve tools from `PATH`:

| Tool | Pinned installation and upstream source | License |
| --- | --- | --- |
| libjpeg-turbo 3.2.0, build 20260630 | `/home/linuxbrew/.linuxbrew/Cellar/jpeg-turbo/3.2.0/bin/cjpeg`; [upstream source archive](https://github.com/libjpeg-turbo/libjpeg-turbo/releases/download/3.2.0/libjpeg-turbo-3.2.0.tar.gz) | IJG and BSD-3-Clause; see installed `LICENSE.md` and `share/doc/libjpeg-turbo/README.ijg` |
| LittleCMS package 2.19.1 | `/home/linuxbrew/.linuxbrew/Cellar/little-cms2/2.19.1/lib/liblcms2.so`; [upstream source archive](https://downloads.sourceforge.net/project/lcms/lcms/2.19.1/lcms2-2.19.1.tar.gz) | MIT, Copyright 2023 Marti Maria Saguer; installed package `LICENSE` |

These source URLs and archive SHA-256 values come from the installed packages'
`.brew/jpeg-turbo.rb` and `.brew/little-cms2.rb` formula records:

- libjpeg-turbo: `6f30092cef9fb839779646608f4ee14ae3cbac989c47fa05e841b0841f09878e`
- LittleCMS: `bfc54f7bab59fbc921012014a8032e4cba4abd46db47d46b76416a8c0b2815c8`

The libjpeg-turbo license record also identifies Zlib for SIMD code and zlib,
libpng-2.0 for parts of libspng, and BSD-2-Clause for most of libspng. Its
`LICENSE.md` explains that these terms are subsumed by the IJG/BSD license
requirements for the tool distribution.

This LittleCMS build reports encoded API version `2190` (2.19); the package
revision is 2.19.1. Both versions are recorded deliberately. Python's standard
library `ctypes` binds the public LittleCMS API. No Pillow or Python package
installation is needed.

The tools and their libraries are not shipped with PicFetch and are not runtime
dependencies. This directory adds no third-party photos, preexisting profiles,
decoder source or native binaries. Pixels, profile descriptions and scan scripts
are synthetic project data; the utility and generated corpus use the repository's
[MIT license](../../../../LICENSE). The generator uses standard sRGB colorimetry
and a numeric gray gamma rather than copying a third-party ICC asset. Retain this
provenance record when adopting generated fixtures. Distribution of the tools
themselves would require their own notices; the tool licenses identified above
are not a license for bundled software added elsewhere.
