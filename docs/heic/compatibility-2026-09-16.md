# Ordinary HEIC compatibility checks, 2026-09-16

## Scope and method

These checks use ordinary existing licensed fixtures and the current real,
ad-hoc-signed App Sandbox helper on macOS 27.0 (26A428), Apple Silicon. The
helper uses the maintained WASI artifact and unchanged production candidate
limits: one admitted job, 64 MiB input, 64M pixels, 256 MB output, 1 GiB WASM
memory and a 60-second deadline. Source bytes enter after native readiness.
There is no native decoder fallback in PicFetch.

Independent reference work runs outside PicFetch, only on these ordinary
fixtures: macOS ImageIO applies orientation and draws into an eight-bit sRGB
bitmap; ExifTool 13.55 reports metadata. PNG comparisons use premultiplied RGBA,
with errors expressed on a 0–255 scale. The eight-bit reference cannot establish
sixteen-bit precision, HDR correctness or broad colorimetric fidelity. Local
tools/results are retained under `.scratch/heic-qualification/`; the reference
tool is `/private/tmp/picfetch-heic-reference.swift`.

## Existing fixture results

| Fixture | Real helper result | Independent check and limit |
| --- | --- | --- |
| `internal/imaging/testdata/test_exif.heic` | 480×640 NRGBA, 1.204 s; orientation 6, TestCam/Model123, f/5.6, ISO 800 | ImageIO and ExifTool agree on metadata and upright dimensions. RGB mean absolute error 0.893/0.517/0.586; maximum 3/2/2. This is a synthetic metadata fixture, not a camera photo. |
| `scripts/heicbuild/testdata/basic.heic` | 320×240 NRGBA, 1.023 s | Same dimensions and recognizable test pattern as ImageIO. Six unobstructed bottom-row color bars differ by at most two levels per channel. Full-image RGB mean absolute error 3.515/1.863/2.339; maxima 134/74/117 occur outside those flat samples. This is not pixel-equivalence evidence; texture/edge discrepancies are not diagnosed here. |
| `scripts/heicbuild/testdata/alpha.heic` | 64×64 NRGBA, 0.911 s | Alpha samples match the independent reference exactly. Premultiplied RGB mean absolute error 0.535/0.326/0.469. |
| `scripts/heicbuild/testdata/tenbit.heic` | 16×16 NRGBA64, 0.917 s | The owned full-range grayscale ramp matches its known generated values within one ten-bit quantization step; equal RGB channels and opaque alpha survive. ImageIO expands/clips this fixture's range differently, so its eight-bit output is not accepted as the numeric oracle. |

Fixture origin and retained notices are recorded in
[`scripts/heicbuild/testdata/README.md`](../../scripts/heicbuild/testdata/README.md)
and [`internal/imaging/testdata/README.md`](../../internal/imaging/testdata/README.md).
The synthetic EXIF fixture SHA-256 is
`49d881a7a87d91cdf79d4599a3f7ecf331fa9d940743954856dc1c66fdd2eae3`.

The native guard now checks Decode, DecodeConfig and DecodeExif against those
known synthetic metadata and single-container-rotation expectations. Its
metadata subtest passes in 3.850 s including package/test setup. The strengthened
ordinary ten-bit subtest passes in 1.893 s. GoLand reports no findings in the
changed test file. These assertions establish additional positive behavior;
they do not claim a fixed production defect or new camera support.
The complete native sandbox-helper test passes in 5.906 s with both additions;
`make verify-build` also passes afterward.

## One real camera photograph

The selected subject is a bee on a flower, Imazen sample 1427, photographed by
contributor `lilith` with a Samsung S23 Ultra. The publisher's pinned
[per-photo manifest](https://raw.githubusercontent.com/imazen/imazen-26/ce09d338b3f68f9bdc8d222f40017ea71ca6bfa4/1400-lilith-nature/MANIFEST.tsv)
records `PD-own`; its
[license statement](https://raw.githubusercontent.com/imazen/imazen-26/ce09d338b3f68f9bdc8d222f40017ea71ca6bfa4/README.md)
defines that as the photographer's own work released to the public domain,
while describing the inventory as best-effort. No private photo library was
accessed. The sample is retained only in local ignored qualification output.

The [immutable test manifest](https://raw.githubusercontent.com/imazen/imazen-26/ce09d338b3f68f9bdc8d222f40017ea71ca6bfa4/manifests/test.tsv)
pins the [ordinary original](https://codec-corpus.r2.imazen.org/imazen-26-unprocessed/1400-lilith-nature/1427_nature_bee-on-pink-flower_monument-inochi-kyoto_s23u_iso64-f4p9_20231107-111746_4000x3000.heic):

- Actual file size: 2,872,198 bytes.
- SHA-256: `b18db35022b1d1d0617b166eef381464cf0d07dbd5f34afeb11d01820ad08f40`.
- The complete size and digest were verified before any decoder read. Stalled
  long transfers were completed with sequential bounded 64 KiB range requests;
  no incomplete image was decoded.
- Real sandboxed helper: successful NRGBA decode in **20.414 seconds**, upright
  **3000×4000**, within the unchanged 60-second/1-GiB-WASM/one-job limits.
- Metadata agrees with independent ImageIO/ExifTool: orientation 6, `samsung`,
  `Galaxy S23 Ultra`, original date `2023:11:07 11:17:46`, f/4.9, ISO 64,
  exposure 1/60 s and focal length 27.2 mm.

The public image contains an eight-bit P3 profile with sRGB transfer. Against
the orientation-aware, color-managed ImageIO sRGB output, RGB mean absolute
errors are **7.529 / 8.283 / 7.644**, RMSE **9.017 / 9.524 / 9.465**, and maxima
**29 / 30 / 70** on a 0–255 scale. Alpha agrees exactly. Visual inspection
confirms the same complete upright scene; these pixel differences do not
establish color fidelity. Profile omission is a known implementation limit;
the experiment does not attribute every pixel difference solely to that cause.

A publisher-generated upright SDR PNG was also identified through a pinned
[LFS pointer](https://raw.githubusercontent.com/imazen/imazen-26/0264d4d8a7e283c046ba2d1febb6e1fc547d511e/png-v3/1400-lilith-nature/1427_nature_bee-on-pink-flower_monument-inochi-kyoto_s23u_iso64-f4p9_20231107-111746_3000x4000.sdr.png):
17,752,719 bytes, SHA-256
`dd7b9491fe6bf115820b28dc945f58090866de872f365eddcb50e5751825129b`.
It was not downloaded or used as a pixel oracle; the actual reference above is
local ImageIO. That distinction avoids claiming an unperformed comparison.

This establishes normal decode, metadata and orientation for **one** Samsung
file on Apple Silicon. It does not qualify every Samsung image, other camera
families, iPhone files, HDR, or faithful wide-gamut display.

## Color/HDR support boundary

The maintained decoder's documented contract applies YUV matrix/range conversion
but leaves primaries and transfer functions in the source color space. ICC data
is only exposed by DecodeColor. PicFetch's guest calls Decode, and its protocol
carries no ICC/CICP profile or HDR metadata. See
[`third_party/h265/heic/heic.go`](../../third_party/h265/heic/heic.go) and
[`scripts/heicguest/main.go`](../../scripts/heicguest/main.go).

ICC-managed/wide-gamut display, PQ/HLG transfer handling, HDR tone mapping and
gain-map composition are therefore not implemented. The guest does not reject
files merely because they use those color classes; it can return untagged
source-space pixels. Successful decode or sixteen-bit transport must not be
described as faithful HDR/wide-gamut display. This work does not add a color
engine or change that behavior. Production HEIC remains disabled.

## What remains unverified

- Broad camera/device compatibility and other valid crop/mirror/rotation
  combinations. The decoder supports still-container transforms; the checked
  cases cover only the ordinary inputs listed here.
- Exact SDR color equivalence, wide-gamut color management and HDR presentation.
  No suitable HDR reference was selected; the missing presentation behavior is
  established independently by the current code contract and is not cured by
  adding a fixture.
- Native Intel/macOS, Linux and Windows camera results, distribution execution,
  and native worker memory measurements. Apple Silicon runs and cross-builds
  do not substitute for those CI/package gates.

The bounded compatibility continuation is complete. No new decoder, color
engine, production activation, commit, push or signing change was made.
