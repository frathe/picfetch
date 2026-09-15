# Image decoder alternatives

Research date: 2026-09-15. PicFetch is MIT-licensed. Ronin's preference is to
avoid gen2brain, including replacements from the same maintainer. This is
source research and a shortlist, not approval to change dependencies or a
release qualification.

## Recommendation

1. **AVIF:** investigate a small PicFetch-owned adapter built directly against
   **libavif 1.4.2 + libaom 3.14.1**. Both decoding and encoding can use libaom.
   These are independent upstream projects with permissive licenses. A
   reproducible WASM build would preserve the existing no-cgo decoder path;
   the adapter and containment still need implementation and verification.
2. **HEIC:** **libheif 1.23.4 + libde265 1.1.3** is the established independent
   option. Both libraries are LGPL-3.0-or-later. PicFetch's own code can remain
   MIT, but distribution must satisfy LGPL source/relinking requirements.
   Keep HEIC disabled until that work and security qualification are complete.
3. **Small permissive HEIC experiment:** **awxkee/hpvcd 0.3.2** merits a
   prototype, after tracing its copied-table provenance. It has no external
   runtime crate dependencies but is young, uses unsafe Rust, and needs a Go
   interface. It is not yet a qualified alternative to the native pair.
4. **Easy extra format:** **PCX**, using `samuel/go-pcx/pcx v1.0.0`.
   **QOI** is a second candidate with some hardening; the inspected Netpbm and
   TGA packages need more work than a straightforward import.

These rankings weigh security evidence and maintenance before dependency
counts. An empty module manifest does not establish safe parsing or complete
licensing provenance.

## Subsequent pure-Go gen2brain evaluation

At Ronin's later direction, `gen2brain/h265 v0.2.3` at
`b2d46ba787d8f0a2025bd106443ab1b1c7cd010f` and `gen2brain/gav1d v0.2.5` at
`7aa50e4e898ebdb4794ae3df88ce8ad6cbb74608` were evaluated as narrow
exceptions to the preference above. Targeted tests and bounded fuzzing found
multiple defects. The final patched admission and WASM prototype suites each
passed 9/9 scoped checks.

Neither the published snapshots nor the locally patched candidates qualify for
PicFetch. Gav1d still has two aggregate-memory paths plus incorrect AV1 level
signaling. H265 still needs explicit sequence rejection, containment,
translated-source provenance resolution and HEVC patent clearance.

The [full security evaluation](purego-codec-security-evaluation-2026-09-15.md)
records the findings, residual risks, license closure, production worker
contract and exact [review patches](codec-hardening/). The patches are audit
evidence, not dependency selections. The preferred AVIF direction and the
decision to keep HEIC disabled are unchanged.

## What the current integration requires

The current [`loader.go`](../internal/imaging/loader.go) registers AVIF;
[`exif.go`](../internal/imaging/exif.go) calls `avif.DecodeExif`; and
[`save.go`](../internal/imaging/save.go) encodes AVIF for save/export.
A replacement must cover all three, including probing, thumbnails and
metadata on the canonical image path. Merely replacing pixel decoding leaves
the existing dependency linked.

The pinned `gen2brain/avif v0.6.0` recipe builds libavif 1.4.2, libaom 3.14.1,
dav1d 1.5.3 and an unpinned `libyuv stable`. Its Go dependencies include wazero,
purego and x/sys. The wrapper prefers an installed native library and falls
back to WASM; `APP_TAGS := no_emoji` does not disable native loading. Its wazero
path uses background contexts without an explicit memory ceiling or
close-on-context-done configuration. These are source observations, not a
new exploit demonstration. See the pinned
[recipe](https://github.com/gen2brain/avif/blob/v0.6.0/lib/Makefile),
[runtime](https://github.com/gen2brain/avif/blob/v0.6.0/avif_wazero.go), and
[existing distribution audit](find-more-like-this/dependency-qualification.md).

## AVIF alternatives

### Direct libavif: preferred established route

Inspected **libavif v1.4.2**, released May 26, 2026, and its upstream-selected
**libaom v3.14.1**. A deliberately small library build can retain only the AOM
encoder and decoder and turn off apps, extra codecs, external libyuv,
libsharpyuv and libxml2. Build flags must be explicit so a host-installed
optional library cannot silently enlarge the shipped dependency set.
[Release](https://github.com/AOMediaCodec/libavif/releases/tag/v1.4.2),
[build options](https://github.com/AOMediaCodec/libavif/blob/v1.4.2/CMakeLists.txt),
[AOM pin](https://github.com/AOMediaCodec/libavif/blob/v1.4.2/cmake/Modules/LocalAom.cmake).

The source-license inventory is more than two top-level labels:

| Component | License / required record |
| --- | --- |
| libavif | BSD-2-Clause; preserve its complete applicable notices, including dav1d-derived `src/obu.c`. |
| libaom | BSD-2-Clause and the accompanying AOM patent grant; retain copyright, license and `PATENTS`, and inspect the configured build's included third-party files. |
| libavif's bundled libyuv scaling subset | BSD-3-Clause. **Still compiled when `AVIF_LIBYUV=OFF`.** Its recorded upstream source is `def473f501acbd652cd4593fd2a90a067e8c9f1a`; no floating `stable` fetch is needed for this subset. |
| WASM execution, if selected | Existing wazero v1.12.0, Apache-2.0. Inventory the exact WASI/LLVM runtime code linked into the guest as part of its build. |
| Native execution, if selected | Inventory the actual C/compiler runtimes for each packaged target. |

Sources: [libavif combined license](https://github.com/AOMediaCodec/libavif/blob/v1.4.2/LICENSE),
[bundled source provenance](https://github.com/AOMediaCodec/libavif/blob/v1.4.2/third_party/README.md),
[AOM license](https://aomedia.googlesource.com/aom/+/refs/tags/v3.14.1/LICENSE),
[AOM patent grant](https://aomedia.googlesource.com/aom/+/refs/tags/v3.14.1/PATENTS).

Thus the minimum is **two codec projects, plus bundled source and execution
runtime obligations**, not two notices or a zero-dependency binary. Retaining
**dav1d** as the decoder adds one BSD-2-Clause library and is a performance
option to benchmark. AOM-only decoding is a dependency reduction, not a
demonstrated speed improvement. This research built neither configuration.

Security evidence includes an upstream private reporting process, OSS-Fuzz
integration and maintained malformed-input fixes. The 1.4.2 changelog includes
memory-leak, allocation-failure and decoder crash fixes. Native C remains
memory-unsafe; use a constrained worker or bounded WASM execution rather than
treating upstream maturity as isolation.
[Security policy](https://github.com/AOMediaCodec/libavif/blob/v1.4.2/SECURITY.md),
[OSS-Fuzz integration](https://github.com/google/oss-fuzz/tree/master/projects/libavif),
[changelog](https://github.com/AOMediaCodec/libavif/blob/v1.4.2/CHANGELOG.md).

### Other AVIF routes examined

| Candidate / inspected revision | Dependency and license facts | Assessment |
| --- | --- | --- |
| `KarpelesLab/goavif`, `b494d13c5a28cbd9d22f5b5305bedced03fdb6b5` (April 18, 2026; newer than v0.1.0) | MIT; module has no requirements, core uses Go standard library. | Not an equivalent replacement yet: its roadmap leaves filter-intra/palette/intrabc, quantization and restoration work open. README's broad completeness claim is insufficient. No fuzz target found in inspected sources. |
| `vegidio/avif-go`, tag `26.7.0`, `dbb32e4e0094990df3cee6ec3026509bbaa524ea` | Apache-2.0 wrapper, cgo, bundled static libraries. Link flags name libavif, SVT-AV1, dav1d, jpeg, turbojpeg, yuv and fastfeat, plus platform runtime. Go test dependencies are separate. | Supports our six architecture/OS combinations in build files, but not a dependency reduction. Exact native archive provenance and every component's notices still require qualification; the wrapper license alone does not clear them. |
| `memorysafety/rav1d` | BSD-2-Clause Rust port of dav1d, C-compatible API; still needs AVIF container handling and an encoder. | Useful future backend; adds a Rust toolchain/bridge and a crate closure. Not the smallest complete replacement. |
| `imazen/zenavif`, inspected manifest version 0.2.0 | AGPL-3.0-only OR commercial; additional crates. | Reintroduces the kind of distribution decision behind the HEIC removal. Do not select as a permissive replacement. |

Sources: [goavif manifest](https://github.com/KarpelesLab/goavif/blob/b494d13c5a28cbd9d22f5b5305bedced03fdb6b5/go.mod),
[goavif roadmap](https://github.com/KarpelesLab/goavif/blob/b494d13c5a28cbd9d22f5b5305bedced03fdb6b5/ROADMAP.md),
[vegidio native linkage](https://github.com/vegidio/avif-go/blob/dbb32e4e0094990df3cee6ec3026509bbaa524ea/build_linux_amd64.go),
[rav1d](https://github.com/memorysafety/rav1d),
[zenavif manifest](https://github.com/imazen/zenavif/blob/main/Cargo.toml).

## HEIC alternatives

### libheif + libde265

Inspected **libheif v1.23.4** and **libde265 v1.1.3**. Both actual library
headers specify LGPL-3.0-or-later. A minimal decode-only build needs these two
libraries and the platform C/C++ runtime; **x265 is not needed**. Explicitly
disable every other codec, plugin loading, libsharpyuv, header compression,
uncompressed codecs and application/UI/example targets. Default builds enable
more than this.
[libheif license header](https://github.com/strukturag/libheif/blob/v1.23.4/libheif/api/libheif/heif.h),
[libde265 license header](https://github.com/strukturag/libde265/blob/v1.1.3/libde265/de265.h),
[configuration](https://github.com/strukturag/libheif/blob/v1.23.4/CMakeLists.txt).

LGPL can coexist with PicFetch's MIT application code. It requires library
notices, GPL/LGPL texts, appropriate corresponding source and a compliant way
to modify/relink or replace the covered library. Static linking or embedding
WASM does not remove those requirements. An actual packaging design must
satisfy them, including applicable installation information. HEVC patent
rights are a separate question from these source copyright licenses.
[LGPL section 4](https://github.com/strukturag/libheif/blob/v1.23.4/COPYING).

**One shared backend is possible:** libheif + libde265 + libaom can cover HEIC
reading and AVIF reading/writing with three codec projects. That reduces
duplication if HEIC is restored, while extending the LGPL-dependent path to
AVIF too. Assess fidelity, metadata and packaging before choosing it over
libavif. The official
[libheif-go binding](https://github.com/strukturag/libheif-go) uses cgo and LGPL;
it does not preserve the existing no-cgo AVIF path.

Security maintenance is active, not evidence of freedom from defects:
libheif 1.23.4 is a security-maintenance release; recent libde265 releases
include memory-safety fixes. Pin current source, preserve the library limits,
and isolate the complete parse/decode operation.
[libheif release](https://github.com/strukturag/libheif/releases/tag/v1.23.4),
[libde265 releases](https://github.com/strukturag/libde265/releases).

### hpvcd: small, permissive candidate with unresolved provenance

Inspected **0.3.2**, commit
`de8b3a38d25b35431141b175562e1488758c3c61` (July 25, 2026). The manifest declares
**BSD-3-Clause OR Apache-2.0** and an empty runtime `[dependencies]` section.
The separate app/fuzz workspace members have their own dependencies; do not
count those as the library runtime or accidentally ship them.
[Manifest](https://github.com/awxkee/hpvcd/blob/de8b3a38d25b35431141b175562e1488758c3c61/Cargo.toml),
[license](https://github.com/awxkee/hpvcd/blob/de8b3a38d25b35431141b175562e1488758c3c61/LICENSE.md).

It handles HEIC containers, grids, orientation, alpha, metadata, gain-map
planes and 8/10/12-bit samples. Parser settings bound dimensions, pixels,
items, boxes, extents and metadata. Fuzz targets and scalar Linux/macOS tests
exist; a WASM cross-build is in CI. No Windows execution evidence was found.
Rust 1.93 and a Go bridge or worker/guest ABI are additional engineering work.
[API](https://github.com/awxkee/hpvcd/blob/de8b3a38d25b35431141b175562e1488758c3c61/src/lib.rs),
[limits](https://github.com/awxkee/hpvcd/blob/de8b3a38d25b35431141b175562e1488758c3c61/src/limits.rs),
[CI](https://github.com/awxkee/hpvcd/blob/de8b3a38d25b35431141b175562e1488758c3c61/.github/workflows/build_push.yml).

Two qualifications matter:

- Disabling SIMD does **not** make it entirely safe Rust: `decode.rs`,
  `plane.rs`, `threadpool.rs` and `wpp.rs` also contain unsafe code.
- `src/cabac/contexts.rs` explicitly attributes initialization tables to
  libde265's `contextmodel.cc`. These may be normative codec tables; this
  observation does not establish an LGPL violation. It does mean we must
  trace their specification/source provenance before declaring the whole
  payload cleared under its root permissive license.

[Table attributions](https://github.com/awxkee/hpvcd/blob/de8b3a38d25b35431141b175562e1488758c3c61/src/cabac/contexts.rs),
[plane implementation](https://github.com/awxkee/hpvcd/blob/de8b3a38d25b35431141b175562e1488758c3c61/src/plane.rs).

Other independent Rust projects examined were `dan335/heif-oxide 0.1.0`
(limited profile support and an additional crate tree) and `tbraun96/heic-rs
0.1.1` (created September 11; security policy says fuzzing has not run).
Neither has stronger readiness evidence for this task.
[heif-oxide](https://docs.rs/crate/heif-oxide/0.1.0),
[heic-rs security policy](https://github.com/tbraun96/heic-rs/blob/4f0d4df474c773dfc3b14fa80e7219be6898866e/SECURITY.md).

Apple ImageIO can avoid a bundled decoder on macOS. Windows WIC's HEIF/HEVC
availability varies by installed codecs, and neither provides Linux coverage.
These are platform-specific options, not one portable replacement. Nokia's
reference library is container handling rather than a complete HEVC decoder,
and its custom noncommercial evaluation/research terms do not meet this use.
[Apple](https://developer.apple.com/documentation/imageio),
[Microsoft](https://learn.microsoft.com/en-us/windows/win32/wic/heif-codec),
[Nokia license](https://github.com/nokiatech/heif/blob/master/LICENSE.TXT).

## Additional formats: only small candidates

Existing JPEG, PNG, GIF, WebP, BMP, TIFF, ICO, XPM, AVIF, SVG and RAW embedded
JPEG previews are not new formats. JPEG XL, full RAW processing and broad
native image toolkits are outside the requested low-effort scope.

| Format / package | License and dependency closure | Readiness |
| --- | --- | --- |
| **PCX**: `samuel/go-pcx/pcx v1.0.0`, `18116d57b65c81ebede2ad47a0d7e16fe3cc7762` | BSD-3-Clause; Go standard library only, no module requirements/native payload. | Best small addition. `Decode`, header-only `DecodeConfig`, image registration and encode; malformed-header/RLE/truncation tests and fuzz corpus. Uncompressed PCX is unsupported. Keep PicFetch's pixel cap: the library's own cap still permits large allocations. |
| **QOI**: `hchargois/qoi v1.0.0` | MIT; codec uses stdlib. Module also requires BSD-3-Clause x/sync for other tooling. | Small follow-up after hardening/tests. `Width*Height*4` uses uint32 arithmetic; direct calls can overflow. PicFetch's existing 200-Mpx preflight would block that particular overflow when consistently applied. Header validation and malformed streams still need qualification. |
| **PBM/PGM/PPM/PAM**: `spakin/netpbm v1.3.2` | BSD-3-Clause; stdlib and its own color package only. | Parser fix first. Header comments ending at EOF can remain in a loop appending zero runes, before dimension admission. File-byte bounds alone do not stop that loop. Source-derived finding; not reproduced in this task. |
| **TGA**: `ftrvxmtrx/tga`, `bd8e8d5be13a2bda61c29a47ca41ecc8f20ee731` | MIT codec, stdlib only. Test-fixture licensing needs separate treatment. | Last upstream commit May 2015. Registers empty magic and reads the full file in `DecodeConfig`; needs explicit dispatch and hardening. Lower priority. |
| **CUR** via current ICO decoder | Reuses existing dependency if adapted. | Not extension-only: `fyne-io/image v0.1.1` rejects cursor type 2. Needs a cursor-header adapter and tests. |

Sources: [PCX license](https://github.com/samuel/go-pcx/blob/v1.0.0/LICENSE),
[PCX decoder](https://github.com/samuel/go-pcx/blob/v1.0.0/pcx/decoder.go),
[PCX tests](https://github.com/samuel/go-pcx/blob/v1.0.0/pcx/invalid_test.go),
[PCX fuzzing](https://github.com/samuel/go-pcx/blob/v1.0.0/.github/workflows/ci.yml),
[QOI arithmetic](https://github.com/hchargois/qoi/blob/v1.0.0/qoi.go#L193),
[QOI manifest](https://github.com/hchargois/qoi/blob/v1.0.0/go.mod),
[Netpbm comment parser](https://github.com/spakin/netpbm/blob/v1.3.2/netpbm.go#L183),
[TGA decoder](https://github.com/ftrvxmtrx/tga/blob/bd8e8d5be13a2bda61c29a47ca41ecc8f20ee731/decode.go#L92),
[ICO reader](https://github.com/fyne-io/image/blob/v0.1.1/ico/reader.go).

## Qualification still needed before implementation ships

- Lock the selected source, full compiled dependency inventory, flags and
  runtime artifacts. Deliver applicable notices in the actual packages;
  document source/relinking delivery for LGPL components.
- Put container parsing, metadata and pixels under the same resource limits.
  WASM needs explicit memory limits and interruptible calls; a worker needs
  deadlines, kill/join and bounded input/output. Pure Go/Rust still permits
  resource exhaustion, panics and unsafe-code defects.
- Compare real images against the existing implementation/reference decoder:
  color, alpha, high bit depth, grids, orientation, metadata and AVIF export.
  Run malformed-input, cancellation, memory and relevant fuzz regressions.
- Recheck published advisories for the selected versions and the built native
  or Rust payload, not only `govulncheck` on the Go wrapper. Verify packaged
  Linux, macOS and Windows behavior on the supported architectures.

No application source, dependency selection or format support changed. This
shortlist began as source research; the subsequent, separately documented
pure-Go evaluation added disposable decoder builds, targeted regressions,
bounded fuzzing and isolation tests. Neither phase is an exhaustive security
or conformance audit. The durable repository changes are research records,
the implementation plan, standing todos and exact review patches.
