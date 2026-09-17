# JPEG metadata removal qualification

Qualification date: 2026-09-17. The supported operation is original-file metadata
removal; metadata-preserving Save Changes and export retain their separate policy.

## Supported inputs

| Dimension | Qualified behavior |
| --- | --- |
| JPEG | 8-bit Huffman SOF0 baseline and SOF2 progressive, one or three components, 8-bit quantization tables; complete coefficient progression, including separate-component sequential scans. Other processes and four-component color are refused. |
| Scan boundaries | All supported scans are followed through structural EOI. APP/COM metadata is removed between scans as well as in the header. Missing boundaries and unsupported structures are refused. |
| JFIF/JFXX | Validate JFIF 1.00-1.02 immediately after SOI, retain its 14-byte interpretation/density header with zero thumbnail dimensions; remove thumbnail bytes, extensions and unclaimed payload. |
| Adobe | Validate version 100, zero flags and qualified gray/RGB transform. Retain only the 12-byte declaration; conflicts with JFIF or RGB component identifiers are refused. |
| ICC | v2/v4 input/display (`scnr`/`mntr`) RGB matrix/TRC and gray/TRC with XYZ PCS and D50 header illuminant. Required descriptions/copyright/white point and model-specific transform tags must exist. |
| ICC transform tags | Exact XYZ columns, white/black points, `chad`, `chrm`, and monotonic `curv` or qualified gamma/sRGB `para` types 0/3. Other tags, LUT models, ambiguous assembly, partial overlaps, wrong models and malformed lengths are refused. Maximum assembled source profile: 4 MiB. |
| ICC identity | Rebuild the tag table/data with zero padding; retain exact transform payloads. Replace description/copyright with neutral text, remove manufacturer/model descriptions and unclaimed bytes, normalize creation date, clear creator/manufacturer/model/platform/CMM/profile ID. Preserve qualified rendering intent and device attributes. |
| Orientation | Identity/absent: primary encoded bytes and decoded samples remain identical. Orientations 2-8: apply the existing quality-95 re-encode, retain gray/RGB model and normalized profile, preserve JFIF density/pixel aspect with axes swapped for orientations 5-8, and validate the proposed result before replacement. |

Unknown input is not presented as clean. Inspection and mutation share the policy;
mutation reads the current file after transaction admission. Clean sources return
an uncommitted result. Cancellation/refusal before replacement leaves bytes intact;
committed results preserve their existing host-reconciliation semantics.
Cancellation is checked throughout scan traversal, between orientation rows and
at buffered JPEG encoder output boundaries; the encoder's pixel loops unwind on
cancellation instead of finishing a discarded image.

## Corpus and independent color reference

The five synthetic 16x12 JPEGs and four ICC profile fixtures are in
[`internal/imaging/testdata/jpeg-removal`](../internal/imaging/testdata/jpeg-removal/README.md).
That record contains recipes, exact development-tool versions, source provenance
and license obligations. No production dependency or shipped native runtime was
added. Samples cover baseline/progressive RGB and gray, separate-component RGB,
ICC 2.1/4.3 RGB sRGB matrix/TRC and D50 gray gamma 2.2, plus input-class derivatives.

The public mutation tests compare outputs with independent clean JPEG fixtures,
exact decoded samples and exact numerical profile tag payloads. The combined
matrix covers each applicable JPEG/profile family. Orientation tests cover all
values 2-8 with RGB/gray v2/v4 profiles; the deliberately high-frequency fixture
allows mean channel error at most 12/255 for the documented lossy re-encode.

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

All six `TestJPEGMetadataRemoval*` acceptance tests are registered in imaging and
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
image coding/numerical color transforms, erase filesystem metadata or other copies,
or qualify every JPEG/ICC extension. Structural qualification plus decoding is
not a claim that a file contains no hidden information. Native Windows/macOS UI
behavior is not established by Linux tests; final gate evidence belongs in the
[implementation record](../plans/2026-09-17-jpeg-metadata-privacy.md).
