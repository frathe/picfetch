# JPEG metadata removal qualification

Qualification date: 2026-09-17. The supported operation is original-file metadata
removal; metadata-preserving Save Changes and export retain their separate policy.

## Supported inputs

| Dimension | Qualified behavior |
| --- | --- |
| JPEG | 8-bit Huffman SOF0 baseline and SOF2 progressive, one or three components, 8-bit quantization tables; complete coefficient progression, including separate-component sequential scans. Other processes and four-component color are refused. |
| Component interpretation | Three-component JPEGs without JFIF or Adobe declarations require ordered component IDs `1, 2, 3` (YCbCr) or `R, G, B` (RGB). Other undeclared layouts are refused for both upright and oriented removal. |
| Scan boundaries | Huffman syntax consumes exactly the expected blocks through EOI, including progressive refinement. Scan/restart boundaries require one-bit padding and the expected restart sequence; unused bytes, malformed padding and resynchronization are refused. Legal stuffed bytes and marker fill are retained. APP/COM metadata is removed between scans as well as in the header. |
| JFIF/JFXX | Validate JFIF 1.00-1.02 immediately after SOI, retain its 14-byte interpretation/density header with zero thumbnail dimensions; remove thumbnail bytes, extensions and unclaimed payload. |
| Adobe | Validate version 100, zero flags and qualified gray/RGB transform. Retain only the 12-byte declaration; conflicts with JFIF or RGB component identifiers are refused. |
| SPIFF | APP8 SPIFF declarations are refused because their base-image color interpretation is not qualified. |
| EXIF color | Rebuild validated `ColorSpace=1`/`65535` and `InteroperabilityIndex=R98`/`R03` alongside the sanitized ICC profile, preserving their existing interpretation without assuming agreement or precedence. Duplicate, malformed or other enumerations are refused. Explicit TransferFunction, WhitePoint, PrimaryChromaticities, YCbCrCoefficients, ReferenceBlackWhite and Gamma remain refused. |
| EXIF chroma positioning | Preserve co-sited `YCbCrPositioning=2` with the unchanged compressed image; centered `1` is the default and needs no retained tag. Reserved values and malformed or duplicate declarations remain refused. |
| ICC | v2/v4 input/display (`scnr`/`mntr`) RGB matrix/TRC and gray/TRC with XYZ PCS and D50 header illuminant. Required descriptions/copyright/white point and model-specific transform tags must exist. |
| ICC transform tags | Exact XYZ columns, white/black points, `chad`, `chrm`, and monotonic `curv` or qualified gamma/sRGB `para` types 0/3. Also retain standard `lumi` (nonnegative Y, zero X/Z), fixed-size `meas` with bounded standard observer/geometry/flare/illuminant values, and enumerated `tech`. Other tags, LUT models, ambiguous assembly, partial overlaps, wrong models and malformed lengths are refused. Maximum assembled source profile: 4 MiB. |
| ICC identity | Rebuild the tag table/data with zero padding; retain exact numerical payloads. Replace description/copyright with neutral text, remove manufacturer/model/viewing descriptions and unclaimed bytes, normalize creation date, clear creator/manufacturer/model/platform/CMM/profile ID and vendor-specific device attributes. Preserve qualified rendering intent and the four standard media-attribute bits; reserved bits remain refused. |
| Retained markers | Preserve legal fill on retained JFIF, Adobe and EOI markers. When the assembled ICC profile already equals its qualified normalized form, preserve its original marker fill, chunking and scan placement; packet packaging alone is not removable private data. |
| Orientation | All orientations retain exact encoded image bytes and decoded samples. Rebuild Orientation 2-8 without recompression; absent/identity needs no retained tag. JFIF density/pixel aspect and ICC transforms stay in the original image coordinate system. The rebuilt EXIF contains at most 104 payload bytes with fresh pointers, zero padding and no descriptive values, thumbnails or source data gaps. |
| Memory | The separate 256 MiB working-memory budget reserves encoded copies, ICC scratch, padded/subsampled component planes, progressive coefficient arrays and nonzero masks, plus RGB conversion only when required by the decoder. Flexible sampling retains its full-plane estimate. Header-only admission precedes copied JPEG data, entropy masks and full decoding; encoded sources remain limited to 60 MiB or the configured file limit, whichever is lower. Ordinary 24 MP 4:2:0 progressive headers with 10 MiB encoded storage are admitted; oversized working sets remain refused before decoding. |

Unknown input is not presented as clean. Inspection and mutation share the policy;
mutation reads the current file after transaction admission. Clean sources return
an uncommitted result. Cancellation/refusal before replacement leaves bytes intact;
committed results preserve their existing host-reconciliation semantics.
The retained EXIF is reconstructed from finite enumerations: orientation 2-8,
co-sited chroma placement, the two supported ColorSpace values and the two
supported InteroperabilityIndex values. Identity/centered defaults are omitted.
Keeping these declarations with unchanged compressed samples and ICC numerical
transforms preserves the reader's interpretation, including when EXIF and ICC
would select different color spaces. No numerical equivalence or precedence is
assumed. Camera/date/GPS/MakerNote/thumbnail directories, text, source padding,
unclaimed bytes and next-IFD links are never copied. Malformed rendering fields
still cause refusal; unqualified numerical EXIF transforms remain unsupported.
The orientation/chroma/color fields are defined in section 4.6 of
[CIPA DC-008-2010](https://www.cipa.jp/std/documents/e/DC-008-2010_E.pdf).
Cancellation remains checked during source reads, marker/scan traversal,
validation decoding and atomic output writes; this operation has no encoder.
Entropy qualification follows [ITU-T T.81](https://www.w3.org/Graphics/JPEG/itu-t81.pdf),
sections B.1.1.5, E.1.2 and F.1.2.3 and Annexes C/G. Unused scan bytes are outside
the qualified image coding syntax; the steganography exclusion does not cover
them. Pixel reconstruction must also succeed with the shipped Go decoder. Its
existing refusal of some subsampled noninterleaved restart layouts remains:
the independent 35x27 4:2:0 progressive and separate-component fixtures with a
restart every MCU pass `djpeg -strict` but fail Go 1.27 `image/jpeg.Decode`.

## Corpus and independent color reference

The five synthetic 16x12 JPEGs, five 35x27 entropy fixtures and four ICC profiles are in
[`internal/imaging/testdata/jpeg-removal`](../internal/imaging/testdata/jpeg-removal/README.md).
That record contains recipes, exact development-tool versions, source provenance
and license obligations. No production dependency or shipped native runtime was
added. Samples cover baseline/progressive RGB and gray, separate-component RGB,
ICC 2.1/4.3 RGB sRGB matrix/TRC and D50 gray gamma 2.2, plus input-class derivatives.

The public mutation tests compare outputs with independent clean JPEG fixtures,
exact decoded samples and exact numerical profile tag payloads. The combined
matrix covers each applicable JPEG/profile family. Orientation tests cover all
values 2-8 with RGB/gray v2/v4 profiles and require exact decoded/displayed sample
equality. A 288-case camera-declaration matrix combines three scan families, all
eight orientations, both chroma placements, both color declarations and absent,
v2 or v4 ICC profiles. It asserts exact compressed-image bytes, retained rendering
values, removed private payloads and idempotence. Both TIFF byte orders and
conflicting EXIF/ICC interpretations also have coverage.

A supplied 4048x3036 camera JPEG exposed another compatibility gap: its ordinary
v2 sRGB profile contains optional luminance, measurement, technology and viewing
description tags, plus vendor attribute bits. Synthetic v2/v4 regressions now
cover those fields individually and together, and malformed variants must leave
the source untouched. The tag layouts/enumerations and header attributes follow
[ICC.1:2022](https://www.color.org/specification/ICC.1-2022-05.pdf), sections
7.2.14, 9.2.33-34, 9.2.49-50 and 10.14. Only standard numerical/enumerated data
survives; vendor identity and descriptions are removed. Nonstandard vendor CMM
interpretation is not qualified.

On a temporary copy of the supplied JPEG, removal succeeds and ExifTool finds
only `ColorSpace=sRGB` in EXIF, with no XMP/IPTC. Strict `djpeg` output before and
after has identical SHA-256
`455e13d4c2071e11bb7b10feaceb3aefcd06c86eb4209a9a3569739f91d43258`.
LittleCMS comparison of the extracted profiles yields zero XYZ difference for
343 samples under every rendering intent. The private photo/profile are not
repository fixtures; the source attachment remains unchanged.

`reference.py check` independently opens original and output profiles with
LittleCMS 2.19.1 and transforms 343 RGB grid samples or 257 gray samples into XYZ
doubles under all four rendering intents. Transform optimization is disabled.
Tolerance is 1e-7 absolute XYZ, allowing numerical rounding without accepting a
visible transform change. All tested normalized profiles produced **zero**
difference. This sample evidence supplements exact preservation of qualified
transform parameters; it does not prove arbitrary profiles equivalent.

Reproduce profile artifacts with:

```sh
PICFETCH_ICC_EVIDENCE=/tmp/picfetch-icc-output go test -tags no_emoji,nodynamic ./internal/imaging -run '^TestJPEGMetadataRemovalProfiles$' -count=1
python3 internal/imaging/testdata/jpeg-removal/reference.py check /tmp/picfetch-icc-output/rgb-v4-mntr-source.icc /tmp/picfetch-icc-output/rgb-v4-mntr.icc
```

Repeat the checker for `rgb`/`gray`, `v2`/`v4`, and `mntr`/`scnr`. A changed gray
gamma reference failed with a 0.073410135908 XYZ difference; unreadable profiles
also fail. Every generated JPEG decoded with independent `djpeg -strict`.

## Acceptance commands and limits

The `TestJPEGMetadataRemoval*` acceptance tests are registered in imaging and
the EXIF window. Run them with:

```sh
go test -tags no_emoji,nodynamic -count=1 ./internal/imaging ./internal/ui/exifwin -run '^TestJPEGMetadataRemoval'
go test -tags no_emoji,nodynamic -count=1 ./internal/imaging ./internal/ui/exifwin -run '^(TestFileMutation|TestFileTransaction|TestMetadataRemoval|TestMetadataResultsApplyOnceOnUIAndRejectQueuedOldData)'
make verify
```

The real window matrix checks action membership, default cancellation, committed
notification, post-removal clean status, unsupported status and no false success
when a source becomes clean before confirmation. Existing lifetime regressions
cover cancellation, navigation, close and retry. The wrapped confirmation was
rendered and visually inspected at the ordinary 420-pixel panel width.

This operation does not anonymize visible content, remove steganographic data in
image coding/numerical color transforms, erase other copies, or qualify every
JPEG/ICC extension. Atomic replacement may reset filesystem attributes or
timestamps; neither their preservation nor their removal is guaranteed. The
working-memory estimate bounds this operation's admitted work, not whole-process
RSS or the independently retained foreground image cache. Structural qualification plus decoding is
not a claim that a file contains no hidden information. Native Windows/macOS UI
behavior is not established by Linux tests; final gate evidence belongs in the
[original implementation record](../plans/2026-09-17-jpeg-metadata-privacy.md)
and the [compatibility follow-up](../finished_refactorings/2026-09-17-jpeg-removal-compatibility.md).
The [camera-profile and clipboard follow-up](../finished_refactorings/2026-09-17-jpeg-and-clipboard-followup.md)
records the supplied-photo qualification and latest full-gate results.
