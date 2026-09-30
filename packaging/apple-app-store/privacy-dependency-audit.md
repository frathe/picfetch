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
its `Privacy.md` is explanatory text, not an Apple privacy manifest. An ad-hoc
app signature is not evidence of upstream SDK provenance or distribution trust.

The [Abseil 20250814.0 tree](https://github.com/abseil/abseil-cpp/tree/20250814.0)
lists a PrivacyInfo.xcprivacy file. Retrieving exact manifest files and recursive
source trees failed in this environment (GitHub API timeouts; research-browser
cache/fetch failures). A newer default-branch manifest is not silently substituted
for an exact pinned dependency. The Protobuf CocoaPods manifest packaging found
in current upstream is specifically for its Objective-C runtime; it does not by
itself qualify the older C++ runtime in these ONNX binaries.

Next technical work: obtain and hash manifests for the exact shipped components,
verify their applicable declarations and intended bundle placement, and resolve
upstream binary-signature provenance or use a reproducible, reviewed source build.
Then validate the final package through Apple's tools. Do not invent empty
manifests or infer compliance from an archive's license notice. This is an
external-artifact/research blocker, not a request to remove Similarity Explorer.

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
URL must be rechecked after these uncommitted changes are published.

## Verification boundaries

Local artifact checks, retained ARM/Intel-Rosetta worker results and binary
samples are under `.scratch/apple-store-package/`; the implementation plan
records their commands and limits. No SDK privacy upload validation, physical
Intel run, minimum-OS run, final Store signing, account questionnaire or App
Review has completed. Required-reason API rules for other Apple platforms should
not be copied wholesale onto native macOS; independently apply the current
[privacy manifest documentation](https://developer.apple.com/documentation/bundleresources/privacy-manifest-files).
