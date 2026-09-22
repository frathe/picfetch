# HEIC through system-provided decoders

Status: accepted product design; implementation and native qualification pending.
Date: 2026-09-22

The `/grill-with-docs` interview selected conditional HEIC decoding on macOS,
Windows and Linux. The user explicitly required Linux and two Settings buttons,
then accepted the remaining defaults. This document records those decisions and
the engineering defaults selected under that instruction. It does not claim
that HEIC works in the current build.

## Decisions

| Question | Decision |
| --- | --- |
| Q1: platforms | Include macOS, Windows and Linux. Do not defer Linux. |
| Q2: feature coverage | Decode into PicFetch's shared image path, supporting viewing, thumbnails, comparison, clipboard, JPEG/PNG export, mosaics and similarity analysis. HEIC writing is outside scope. |
| Q3: capability and help | Persist refreshable capability results. Settings has a support-check button and a button that directly opens the current OS's Markdown installation guide. |
| Q4: Linux dependency | Use system-installed libheif with a working HEVC decoder. Another application's ability to open HEIC does not establish PicFetch support. |
| Q5: recheck | A successful check enables subsequent HEIC opens immediately. It does not silently rescan or replace the current collection. |
| Q6: unavailable decoder | Explicit opens explain unavailable support and offer the guide. Mixed scans retain supported images and report skipped HEIC once per scan. |

The architectural trade-off is recorded in
[ADR 0002](adr/0002-system-provided-heic-decoding.md). The local interview
evidence is under `.scratch/os-heic/`. No further product question is pending;
the verification tasks below are work for the implementation phase.

## System decoder boundary

Use Apple ImageIO on macOS, Microsoft WIC with the required official extensions
on Windows, and an installed shared libheif plus HEVC backend on Linux.
PicFetch does not bundle or download a HEIC decoder, launch another image viewer,
or run installation commands. Ordinary non-HEIC use must still launch and work
when these optional facilities are absent. AVIF retains its existing decoder.

Windows qualification must establish which actual HEIF/HEVC components the WIC
adapter uses; do not silently accept arbitrary registered third-party decoders
as equivalent to the selected Microsoft path. Linux qualification must establish
the supported ABI, system library/plugin discovery and minimum versions. An
absent, incompatible or incomplete installation is an availability result, not
a loader error that prevents PicFetch itself from starting.

Linux guidance covers Debian/Ubuntu, Fedora and Arch families where a verified
recipe exists for the detected distribution, release and architecture. This is
not a promise of every Linux distribution. Unknown systems receive generic
requirements and upstream/package links rather than guessed shell commands.
Fedora guidance must distinguish Fedora repositories from optional RPM Fusion;
opening the guide never adds a repository or installs a package.

## What an image can do

Recognize `.heic` and HEVC-backed `.heif` still images, case-insensitively, with
content validation. Decode the declared primary image at full resolution,
including tiled images; multiple auxiliary items do not make the first stored
item the primary photo. Cover ordinary 8-bit and 10-bit camera photos and alpha
where present. A capability probe is evidence for that tested path, not a
promise that every HEIF container or codec profile works.

Apply container orientation/mirroring and EXIF interpretation exactly once.
Probe dimensions, thumbnails, full-image pixels and image-consuming features
must agree on the oriented result. Produce a color-correct SDR representation
for the existing viewer, with native-platform fixture evidence for wide-gamut
and HDR sources. Native HDR display, gain/depth inspection, auxiliary-image
browsing and sequence playback are outside scope. If a source cannot provide a
qualified SDR result, report a file-specific unsupported case rather than show
known incorrect colors. Do not fall back from a still decoder to movie/sequence
decoding, even for a renamed source.

Existing metadata inspection remains best effort and must never double-apply
orientation. Export and clipboard use the same canonical pixels as the viewer.
JPEG/PNG export follows existing metadata-omission behavior for non-JPEG
sources; do not promise HEIC metadata preservation. Save Changes into HEIC,
HEIC encoding, and original HEIC metadata rewriting remain unavailable.

## Capability lifecycle

The Settings display distinguishes not checked, checking, available, unavailable
and check failed. An operational error or timeout is distinct from proving that
the codec is missing. Store successful available/unavailable observations with
their check time, OS identity, architecture and decoder/probe revision. A
stored available result is not permission to skip validation of real files.

On startup, reuse a matching stored observation. A first run, incompatible
record, OS change or decoder/probe revision change triggers one background
check. Ordinary launches do not enumerate codecs or decode a sample again.
An unrelated application version change need not invalidate the record.
Installing a codec without changing that identity is handled by the explicit
Settings check; the design does not promise package-change notifications.

The check must obtain and validate pixels from licensed, small representative
HEIC fixtures through the actual platform adapter, including the promised
8-bit/10-bit paths. A filename, registry entry, library load, source object or
header-only probe cannot establish availability. Bound its execution time and
memory; distinguish unsupported capability from a crashed or timed-out probe.

Only one check is active per app instance. Disable the check button while it
runs. Closing Settings suppresses view-bound callbacks while a successfully
completed check may still update app capability and persistence. Shutdown
cancels and joins its worker off UI. Tests use the feature's observable
completion and drainable queue, never sleeps or real codec installation.

During an initial check, keep the UI responsive. A scan/open encountering HEIC
waits for that check before deciding HEIC admission; mixed scans may make
progress on other files. Cancellation retires pending delivery. Do not drop the
initial HEIC request because the background check happened to finish later.

A successful manual check publishes capability for subsequent operations in
the same run. It does not mutate the source list, revisit previously skipped
files, or silently restart active analysis. Existing in-flight work retains its
captured capability generation. If the OS cannot yet observe an installation,
explain that restart/recheck may be needed rather than reporting false success.

Distinguish a per-file failure from missing system capability. A malformed,
oversized or unsupported photo must not switch off HEIC globally. Evidence of
a missing backend invalidates availability and coalesces one recheck; it must
not create an automatic retry loop. A cancelled or transient failed check does
not overwrite a prior valid observation as "unavailable". If the prior result
was already invalidated, a failed recheck cannot restore it as valid.

## Settings, guide and unavailable files

Place the HEIC status and two actions together in General Settings. Proposed
English labels are "Check HEIC support" and "HEIC installation instructions";
final UI keys and their exact translations follow the normal locale rules.

The guide is an embedded, offline, scrollable Markdown singleton. Reopening
raises it; Escape closes it. Use the current OS and, on Linux, the detected
distribution to select guidance. Follow the existing manual's English/German
content and fallback convention. Buttons/status/error text use all translation
bundles. Avoid Markdown tables and Unicode arrows in rendered help.

Explain requirements, verified install steps or official links, any possible
Windows Store cost, and the instruction to return to Settings and check again.
On macOS explain the built-in support/OS requirement rather than inventing a
codec download. Show only current-OS instructions by default. External links
open only when clicked, and documentation access starts no downloads or probes.

An explicit HEIC open without support gives a specific explanation and access
to the guide. A scan with skipped HEIC files emits one aggregate notice, not a
message per file. An all-HEIC scan gives the specific unavailable explanation.
Existing non-HEIC results and current-view behavior retain normal scan rules.
Saved sessions and Favorites keep their stored HEIC members when unavailable;
temporary filtering is not permission to rewrite the saved collection without
those members. A later explicit reopen after a successful check can admit them.

## Integration, containment and associations

Keep capability ownership with app startup and standing preferences, and
format-specific probe/decode behind a viewer-independent boundary. The existing
canonical image path must serve every consumer, including Explorer and visual
search subprocesses. Private worker startup must receive the same capability
and limits without depending on desktop/Fyne initialization or rereading UI
preferences. Do not add mutable package-level test seams.

Keep untrusted native HEIC parsing and decoding outside the desktop process in
a bounded, cancellable worker. Preserve encoded-byte and pixel admission,
validate dimensions/stride/pixel length before publication, bound concurrency,
and ensure cancellation or timeout can retire a hung decode. Use the same path
for probes, metadata extraction that parses HEIC, thumbnails and full images.
Decoded payload validation remains the parent's responsibility. Existing Linux
and macOS analysis isolation must remain effective when the decoder is added.
Process separation alone is not an OS sandbox; the existing Windows analysis
worker has no network-denial guarantee. The implementation plan must specify
and verify HEIC worker restrictions without implying stronger protection than
the evidence establishes.

Separate formats recognized by the build from formats available on this
machine. Package associations and saved Explorer rule validity must not depend
on which codecs the packaging machine happens to have installed. Once platform
qualification passes, advertise `.heic`/`.heif` in that platform's generated
associations; an unavailable runtime open then follows the explanatory path.
Persisted HEIC format filters remain valid across machines even when one
machine cannot decode HEIC. Ordinary scan admission follows live capability.

## Acceptance and verification

The following are required future test contracts, not tests executed during
this documentation task. Add the named HEIC cases and confirm their inventory
before using a green package command as evidence. Preserve the unavailable
branch of today's HEIC refusal tests, including renamed input, while adding
the available branch. Do not silently skip required native cases.

| Criterion | Required coverage | Verification command |
| --- | --- | --- |
| AC1: capability lifecycle | Unknown, available, unavailable, failed, cached restart, changed identity, manual refresh, coalescing and stale delivery | `go test -tags no_emoji,nodynamic ./internal/preferences ./internal/ui/...` |
| AC2: decoder behavior | Real licensed fixtures for 8/10-bit stills, primary selection, orientation/mirroring, grids, alpha, SDR color and malformed/renamed/sequence input; no HEIC encoder | `go test -tags no_emoji,nodynamic ./internal/imaging` |
| AC3: feature parity | Thumbnails, comparison, captures/export, mosaic and similarity subprocesses consume the same oriented pixels; no AVIF regression | `go test -tags no_emoji,nodynamic ./internal/imaging ./internal/mosaic ./internal/similarity ./internal/ui/...` |
| AC4: admission and recovery | Initial open while probing, mixed/all-HEIC scans, targeted guide, immediate refresh, saved-list preservation and unavailable format rules | `go test -tags no_emoji,nodynamic ./internal/filescan ./internal/explorerpresets ./internal/session ./internal/favstore ./internal/ui/...` |
| AC5: Settings/help | Both buttons are reachable in the widget tree; correct current-OS content, offline singleton lifecycle, translated states and no Unicode arrows | `go test -tags no_emoji,nodynamic ./internal/ui/settingswin ./internal/ui/help .` |
| AC6: packaged formats | Association generation independent of host codec availability; correct macOS UTIs and Windows declarations | `go test -tags no_emoji,nodynamic ./scripts/plistdoctypes ./scripts/msixstage` |
| AC7: worker lifetime | Probe/decode cancellation, timeout, crash, malformed response, input/output bounds and shutdown without stale UI mutation | `go test -race -tags no_emoji,nodynamic ./internal/imaging ./internal/similarity ./internal/ui/...` |
| AC8: native evidence | Required fixture/probe/decode cases actually run, with and without codecs, on shipped OS/architecture combinations | `make test-native`, with implementation-time HEIC required-test inventory and retained native logs |
| AC9: repository gate | Exact test exclusions/shards, locales, format/build/vet/race checks and changed-code GoLand inspection | `make verify`; inspect changed code with GoLand separately |

During implementation, replace broad iteration commands with focused named
HEIC cases once those cases exist. Run the complete gate once at handoff under
the repository's native Linux/amd64 policy. Native platform qualification must
cover macOS Intel/Apple Silicon and Linux/Windows x64/ARM64, including packaged
Windows Store behavior. A cross-build or a stub test is not native evidence.

## Qualification work and honest limits

- Establish exact API/ABI compatibility, primary-image selection and transform/
  color semantics on each native backend. Record real 10-bit/HDR-to-SDR evidence
  before advertising that case; the design requirement is not proof of support.
- Validate decoder discovery after installation and removal, including a
  running app, retained analysis workers and Store packaging. Inspect failed
  probes without treating every failure as a missing codec.
- Verify distribution/version-specific Linux recipes and their architectures.
  Do not equate Arch x86_64 package evidence with Arch Linux ARM evidence.
- Record the binding/header sources, versions, loaded dependency closure,
  applicable notices/distribution obligations and fixture rights in the Deep
  implementation plan. Using a system library is not a legal-clearance claim.
  Do not restore the removed embedded decoder or its dependency.
- Specify worker resource limits, launch policy and testable stop signals in
  that plan; qualify native API access under those restrictions. Reopen this
  design if feasibility requires weakening the agreed behavior or containment.

HEIC remains unsupported until implementation and qualification are complete.
All non-HEIC behavior must continue to work without the optional system codecs.

## Primary sources checked during design

- [Apple HEIF support](https://support.apple.com/en-au/116944) and
  [ImageIO source APIs](https://developer.apple.com/documentation/imageio/cgimagesource)
  establish the native path. [Apple's HDR explanation](https://developer.apple.com/videos/play/wwdc2023/10181/)
  describes explicit SDR decoding/tone mapping; it is not native test evidence.
- [Microsoft HEIF codec](https://learn.microsoft.com/en-us/windows/win32/wic/heif-codec)
  documents optional codec dependencies, RGB output and gain-map access, with
  a prerelease caveat for some content. It does not establish universal HDR
  tone mapping or primary-frame indexing for this adapter.
  [WIC transformations](https://learn.microsoft.com/en-us/windows/win32/wic/-wic-bitmapsources)
  distinguish color conversion from pixel-format conversion.
- [libheif documentation](https://github.com/strukturag/libheif/blob/master/README.md)
  describes decoder backends and plugin loading. Distribution package evidence:
  [Debian](https://packages.debian.org/trixie/libheif1),
  [Ubuntu](https://packages.ubuntu.com/resolute/libheif-plugin-libde265),
  [Arch](https://archlinux.org/packages/extra/x86_64/libheif/),
  [Fedora](https://packages.fedoraproject.org/pkgs/libheif/libheif/fedora-44.html),
  and the separate [RPM Fusion source](https://github.com/rpmfusion/libheif-freeworld/blob/master/libheif-freeworld.spec).
  These are research inputs to version-specific installation guidance, not
  confirmation of what is installed on a user's machine.
