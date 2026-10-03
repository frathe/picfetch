# Apple Store privacy and dependency audit

Checked 2026-09-30 against the local qualification bundle and primary sources.
This record does not certify App Store acceptance.

## Native closure and listed SDKs

The package uses ONNX Runtime **1.29.0 arm64** and **1.23.2 x86_64**, staged
through the existing pinned size/hash verifier. Both contain only system-linked
native dependencies according to the package Mach-O checks. That check does not
exclude statically linked third-party code.

| Runtime | Abseil evidence | Protobuf evidence |
|---|---|---|
| arm64 1.29.0 | Binary strings include `absl::` and `lts_20250814`, plus an Abseil 20250814.0 build path | Binary strings include `google::protobuf` and a `v3.21.12` build path |
| x86_64 1.23.2 | Binary strings include `lts_20250512` | Binary RTTI includes `google8protobuf`; upstream pins v21.12 |

The exact upstream dependency declarations are
[ONNX 1.29.0 deps.txt](https://raw.githubusercontent.com/microsoft/onnxruntime/v1.29.0/cmake/deps.txt)
and [ONNX 1.23.2 deps.txt](https://raw.githubusercontent.com/microsoft/onnxruntime/v1.23.2/cmake/deps.txt).
Their Abseil versions agree with the binary symbols; both pin Protobuf v21.12.
The runtime's bundled ThirdPartyNotices.txt also names both libraries.

Apple's [listed-SDK requirements](https://developer.apple.com/support/third-party-SDK-requirements/)
include Abseil and Protobuf and cover SDKs repackaging them. Binary dependencies
also have SDK-signature requirements. **The current local bundle is not cleared
for this gate.** Neither pinned runtime archive contains an `.xcprivacy` file;
its `Privacy.md` is explanatory text, not an Apple privacy manifest.

Both exact Abseil manifests are now retained in
[privacy/abseil](privacy/abseil/provenance.json). The authenticated GitHub read API
succeeded where the earlier research-browser/raw requests failed. Versions
20250512.0 and 20250814.0 contain identical bytes, git blob
`3ff4a9d98b13eafdc813fb5394796afd6cb8486b`, SHA-256
`f232217ae9edf2ab6a541a28eff50cfe05303c2b4756fff95cca72eccbc3b898`.
Their upstream Apache-2.0 LICENSE files are also identical. The builder preserves
the declaration, license and provenance once in
`Contents/Resources/AbseilPrivacy.bundle/Contents/Resources`; the surrounding
resource bundle has macOS BNDL metadata. It checks pinned hashes before staging
and during artifact validation, including validly re-signed app tampering. This
retains Abseil's declaration; it does not declare PicFetch or ONNX as collecting
no data and does not establish Apple's acceptance of the completed SDK assembly.

The complete, non-truncated Protobuf v21.12 source tree contains no privacy
manifest. Current upstream CocoaPods manifest packaging is specifically for its
Objective-C runtime; it does not by itself qualify the older C++ runtime in
these ONNX binaries. No replacement declaration has been invented or copied
from another version. Protobuf/ONNX manifest coverage and final Apple validation
remain open.

Original dylibs from both pinned archives have Developer ID Application
signatures from Microsoft Corporation, Team `UBF8T346G9`, identifier
`libonnxruntime.1`. Before replacing them with local test signatures, packaging
now verifies strict/all-architecture code integrity with an Apple-anchored
requirement for that exact Team and identifier. The output manifest records the
verified identity, requirement and original library SHA-256 per architecture.
An otherwise valid ad-hoc signature fails this check. This closes the previously
unverified input-identity step; it does not establish final Store distribution
trust or substitute for Apple's SDK/submission validation.

Next technical work: obtain an applicable declaration for the pinned Protobuf
C++/ONNX assembly, or qualify a reviewed source build/new runtime with complete
privacy artifacts. Then validate final manifest placement and SDK requirements
with Apple's tooling. Source builds/upgrades require renewed license review,
architecture/minimum-OS, model, performance and sandbox qualification. Keep
Similarity Explorer available while resolving this release gate.

The native libraries live in Contents/Frameworks. Complete runtime LICENSE,
ThirdPartyNotices.txt and Privacy.md files live under versioned directories in
Contents/Resources. PicFetch's LICENSE, THIRD-PARTY-NOTICES.md and PRIVACY.md are
also included there. Existing Go/Fyne/font/AVIF notices remain required; the
Make verification gate checks the retained AVIF notice inputs.

## Network and local-data inventory

| Behavior | Trigger and destination | Information exposed |
|---|---|---|
| Image viewing, EXIF, similarity, local caches | Local processing; isolated Apple analysis worker | No image or metadata upload |
| Location Map and EXIF map | User opens map; `tile.openstreetmap.org` | Tile coordinates, IP, user agent and connection data; map area can reveal photo location |
| Model setup | User accepts download; pinned Hugging Face model/configuration URLs with permitted provider redirects | Asset URL, IP, user agent and connection data |
| Release-note artwork | Displayed notes contain remote images | Image URL, IP and ordinary HTTP details; current bundled 1.1.11 notes contain no images |
| Help links | User opens public GitHub pages in their browser | Browser/provider policies apply |
| App updates | Mac App Store delivery | Apple-managed; GitHub self-update is disabled for this channel |

Implementation evidence: `internal/similarity/assets_install.go`,
`internal/ui/locationmap/tiles.go`, `internal/ui/exifwin/tiles.go`,
`internal/ui/help/releaseart.go`, `internal/ui/help/releaseimage.go` and the
captured launch/distribution policy. Permitted asset redirect host families are
huggingface.co, hf.co, github.com and githubusercontent.com; executable runtime
downloads are excluded from Store setup.

OSMF explicitly describes network access records, including IP/request data,
and third-party tile CDN processing in its
[privacy policy](https://osmfoundation.org/wiki/Privacy_Policy). Therefore a
blanket claim that all app-related requests are unretained is unsupported.
Apple's [app privacy guidance](https://developer.apple.com/app-store/app-privacy-details/)
distinguishes on-device processing from data transmitted and retained by third
parties. The exact App Store Connect categories/purposes/linkage answers need
assessment of map-derived location and provider retention. This record does not
select answers or claim that every optional request is exempt from disclosure.

The app privacy policy now explains Apple bundled runtime delivery, temporary
worker access and conditional release-note image requests. The public policy
URL must be rechecked after the preparation branch reaches the public main branch.

The pinned ONNX 1.29.0 privacy documentation warns that an API-only opt-out may
follow initialization events. `internal/similarity/encoder.go` already sets
`ORT_DISABLE_TELEMETRY=1` before loading the runtime and also disables telemetry
through the API before session creation. The worker network sandbox is a separate
verified protection; no opt-out code change was necessary.

## Verification boundaries

Local artifact checks, retained ARM/Intel-Rosetta worker results and binary
samples are under `.scratch/apple-store-package/`; the implementation plan
records their commands and limits. No SDK privacy upload validation, physical
Intel run, minimum-OS run, final Store signing, account questionnaire or App
Review has completed. Required-reason API rules for other Apple platforms should
not be copied wholesale onto native macOS; independently apply the current
[privacy manifest documentation](https://developer.apple.com/documentation/bundleresources/privacy-manifest-files).
