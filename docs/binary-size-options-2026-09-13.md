# Binary size options — 2026-09-13

Research observations, not an implementation plan. Ronin declined MA-024;
upstream Sigstore/TUF verification remains the chosen approach. The question
here is whether ordinary application data can be smaller without executable
packing, extra startup decompression, or changes to signing and verification.

Ronin subsequently authorized the asset/font/vector changes. Work and current
verification are tracked in the separate
[implementation record](../finished_refactorings/2026-09-13-embedded-asset-size.md).

## Recommendation

Prioritize app-only character atlases containing the frames actually displayed,
then an exact binary representation of the semantic tag vectors. Both can retain
the existing source assets in the repository and generate smaller embedded data
before compilation. Neither requires a new third-party dependency. No production
code, assets, build flags, dependencies, or signing configuration were changed
during this investigation.

## Measurements

The existing native baseline is 50,629,138 bytes, built with Go 1.27.1 for
darwin/arm64 using `make build`, `-trimpath`, and `-ldflags="-s -w"` at source
revision `87c84b0c1fb7a7bd18d20b218f0d87e3e737ae12`. Its SHA-256 is
`a546b94973dbce45917716508c10530faf43483c89c2fc579e0b89867d5a2144`.
It remains locally under `.scratch/ma-024/2026-09-12/bin-current/picfetch`;
that ignored artifact is not required to build PicFetch.

Each complete file below was found byte-for-byte inside that executable. These
are embedded data sizes, not measured savings from candidate release builds.
MB means 1,000,000 bytes.

| Embedded data | Current bytes | Opportunity and limits |
| --- | ---: | --- |
| [Trane atlas](../internal/ui/assets/trane.webp) | 2,342,454 | Only 17 of 88 cells are used. |
| [Finis atlas](../internal/ui/help/finis.webp) | 1,961,722 | Only 17 of 88 cells are used. |
| [Tag vectors](../internal/similarity/tag-vectors.json) | 1,020,288 | Exact float32 values occupy 230,400 bytes. |
| [Explorer illustration](../internal/ui/assets/explorer-intro.png) | 2,076,221 | 1448 × 1086 pixels; smaller dimensions need visual validation. |
| Fyne 2.8.0 `theme/font/EmojiOneColor.otf` | 4,232,712 | Supported `no_emoji` build tag removes the bundled font, with a text-rendering trade-off. |

### Character frames

[DecodeGazeAtlas](../internal/ui/widgets/gaze.go) decodes an 8 × 11 atlas of
192 × 208 cells, then retains the 16 gaze directions from rows 9 and 10 and
one neutral cell from row 0. Thus 71 cells per atlas are unused by PicFetch.
The two embedded atlases occupy 4,304,176 bytes together. Compressed file size
does not scale directly with cell count, so no exact saving is claimed.

An app-only atlas could retain all 17 frames with identical decoded pixels and
use the existing WebP decoder. Preserve the full Codex pet originals, which
serve a different purpose. Keep Trane's existing spill correction and verify
all displayed frames against the originals using the coverage in
[Trane tests](../internal/ui/trane_test.go) and
[Finis tests](../internal/ui/help/finis_test.go).
Fewer decoded pixels should reduce decoding work; timing has not been measured.

### Exact vector storage

[NewTagger](../internal/similarity/tags.go) already converts JSON numbers to
float32 and hashes their little-endian bits in catalogue order. Serializing
the same 75 × 768 values directly requires 230,400 bytes, a reduction of
789,888 bytes (about 0.75 MiB) in vector data, before loader and linker effects.

The raw representation was computed in memory using Python's
`struct.pack('<f', value)` in [catalogue](../internal/similarity/tag-catalog.json)
order. Its SHA-256 is
`a805e2d7b5f51afe13ef9cd3fc05a9a5dc911fc34a344c900dc79270e10fc107`,
exactly matching the existing `vectorsSHA256`. This preserves the current
canonical float32 values without quantization or compression.

The readable JSON can remain a source artifact for the
[generator](../scripts/explorertags/README.md); only the generated binary data
would be embedded. Retain identity, dimension, norm, and checksum validation.
The tagger is constructed in the [analysis path](../internal/similarity/analyze.go).
Binary loading removes JSON number parsing, but no timing result is claimed.

### Options with visible trade-offs

The Explorer image appears in the [setup dialog](../internal/ui/explorersetup.go)
with a 320 × 240 minimum size. That is not a maximum rendered size: resizing
and high-DPI displays must inform any reduction. Its PNG contains only image
chunks, with no ancillary metadata to remove. Savings are unmeasured.

Fyne 2.8.0 explicitly supports `no_emoji` through `theme/bundled-emoji.go`,
`theme/unbundled-emoji.go`, and the nil-font guard in `internal/painter/font.go`.
These were inspected in the pinned module cache. Omitting the 4.23 MB font
could degrade emoji in filenames or other text. This is a functionality
trade-off, so it is not a preferred first step.

## Build and packaging findings

The ordinary stripping options are already in use. Native builds explicitly
pass `-s -w` and `-trimpath` in the [Makefile](../Makefile). macOS packages use
Fyne `-release`; normal Windows and Linux cross builds also receive it through
fyne-cross's default `StripDebug` setting. The pinned implementations are
[fyne-cross 1.6.3](https://github.com/fyne-io/fyne-cross/blob/v1.6.3/internal/command/container.go#L213)
and [Fyne tools 1.7.2](https://github.com/fyne-io/tools/blob/v1.7.2/cmd/fyne/internal/commands/build.go#L178),
verified from their local module sources. Go documents the stripping flags in
its [linker reference](https://pkg.go.dev/cmd/link).

No redundant shipped content was identified in the inspected packaging paths.
The [release workflow](../.github/workflows/release.yml) builds separate
architectures and puts one executable plus license, notice, and privacy files
in each Windows/Linux archive. macOS packaging adds normal bundle metadata,
icon, and those documents. Store packages intentionally include the ONNX Runtime
DLLs and their notices; the [runtime allowlist](../internal/similarity/assets.go)
already excludes PDB files. This was source inspection, not a fresh audit of
every platform's finished archive.

## Signing, startup, and antivirus limits

The preferred candidates change embedded data before the ordinary compile and
sign steps. They introduce no executable unpacker, loader, or modification of
an already signed binary. Apple's
[code-signing documentation](https://developer.apple.com/library/archive/technotes/tn2206/_index.html)
explains why signed bundle contents must remain intact. Existing asset
provenance and shipped notices should accompany any derived assets.

No technique guarantees unchanged antivirus verdicts for a new binary; Go's
[FAQ](https://go.dev/doc/faq#virus) documents false positives in ordinary Go
programs. These candidates avoid adding packing or obfuscation behavior.
Actual size, startup/first-use timing, pixel/vector equivalence, and normal
signature verification would need checking on candidate builds before making
a release claim. No candidate builds or timing tests were performed here.
