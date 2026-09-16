# PicFetch threat model

**Updated:** 2026-09-16

**Source baseline:** PR #28 at commit
52ed2dfb2500d8a6b8b0895e0cba353c934915e0, with the history-reconciliation changes
in this PR. This adapts the historical model from 73cb3c9 and its fc127b44
implementation baseline; it does not import the old worker implementation.

This model records assets, trust boundaries, implemented controls and residual
risks. It is a static source/documentation assessment, not an exhaustive audit
or verification of every installed binary. Risk categories are not confirmed
vulnerability findings. [ARCHITECTURE.md](ARCHITECTURE.md) is the package map;
[HEIC qualification](docs/heic/qualification.md) records runtime evidence and
[history reconciliation](docs/heic/history-reconciliation.md) records carryover.

**HEIC viewing is default-off and requires a restart after an explicit
Experimental Settings opt-in.** Startup validates the executable-derived helper
package before constructing an immutable shared capability. Missing/corrupt
packages or failed isolation remain unavailable while ordinary viewing works.
Windows private staging and standalone standard-user activation pass native
x64/ARM64 qualification at 69fef1a. Installed-MSIX activation remains blocked
before the test process starts; standalone helper results do not qualify it.
Distribution clearance, production signing and broad color/camera qualification
remain release gates. See the [activation record](docs/heic/experimental-opt-in.md).
The accepted macOS design lacks a hard total native-memory cap; memory pressure
and application/system crashes remain possible. Capability isolation and finite
WASM/input/output/job/time limits remain mandatory.

## 1. Application and assets

PicFetch is a Go/Fyne desktop image viewer. Files arrive through command-line
paths, drag-and-drop, native pickers and macOS Open With events. Opening a file
can also discover sibling images; folder opening can recursively scan a tree.
Production supports common raster formats, SVG, AVIF and embedded JPEG
previews from camera RAW containers. HEIC/HEIF is an experimental opt-in. PicFetch
also handles EXIF metadata, save/export, clipboard, Trash, wallpaper, favorites,
mosaics and session paths.

Optional features add network and native-code dependencies: Similarity Explorer
installs model/runtime assets and analyzes locally in a subprocess; EXIF maps
fetch tiles; standalone builds offer GitHub updates. Microsoft Store builds use
their separate distribution policy. PicFetch has no account system, listening
application server or privileged daemon. Desktop input and optional outbound
traffic, rather than web sessions or server-side roles, define this model.

Assets to protect include:

- Images, GPS and other metadata, paths, previews and similarity representations.
- Original files and user-selected destinations during writes and deletion.
- Application integrity, installed native libraries and release provenance.
- Desktop and host availability, including memory, CPU and storage.
- Release, signing and Store credentials and the authority to distribute updates.

## 2. Attacker capabilities and trust assumptions

**Untrusted content:** an attacker can supply image/container bytes, filenames,
metadata, dimensions, animation data and SVG structure through a download,
message, removable device or shared folder. Opening or previewing a file does
not make its internals trusted. A writable shared tree can contain symlinks and
can change while PicFetch reads or mutates it. Network services and their
responses are inputs too, subject to HTTPS and feature-specific integrity checks.

**Local configuration:** selected paths, preferences, environment variables,
`PATH`, proxy/CA configuration and asset overrides are normally controlled by
the operator. Installed OS helpers and their output are part of the local trust
base. These assumptions can change when launch configuration or directories are
shared with a less-trusted party. App-local state should be protected by suitable
OS directory permissions; PicFetch does not encrypt it as a separate boundary.

**Process authority:** normal operation assumes PicFetch is not elevated. Each
process has the authority actually granted by its OS account, access controls,
entitlements, sandbox and privacy permissions. Two processes with the same user
ID need not have identical access: macOS TCC/App Sandbox and Windows application
write controls can matter. A same-user prerequisite does not by itself dismiss
a defect that lets PicFetch act with authority the attacker lacks.

**User intent:** opening content authorizes reading for viewing, not arbitrary
file mutation or disclosure. Save, export, metadata removal, Trash and wallpaper
actions must respect the user's selected operation and target. Confirmation is
an interaction boundary, not proof that all affected paths are safe.

**Supply chain:** the OS, Go/runtime implementations, shipped dependencies,
checksum pins and update trust roots are trusted components. Maintainers control
source, build tags, embedded assets and workflows. Repository permissions,
release accounts, signing services and protected environments are separate
operational boundaries. Integrity verification against a trusted pin establishes
which bytes were obtained; it does not prove those bytes are free of defects.

## 3. Content parsing and resource use

Most image work runs inside the GUI process. Decoder code executes with its
host process's effective privileges. A memory-safety defect that enables
arbitrary code execution could expose that authority; an ordinary panic or
out-of-memory failure primarily affects availability and does not itself prove
code execution or arbitrary filesystem access. Go memory safety, third-party
native code, runtime trust, resource limits and OS confinement are distinct
parts of the assessment.

The canonical [loader](internal/imaging/loader.go) defaults to a **512 MiB
encoded-file limit**, configurable in settings, and checks probed raster
dimensions against **200 million pixels**. [GIF handling](internal/imaging/gif.go)
bounds animation storage and frame count before full animation decode.
[SVG handling](internal/imaging/vector.go) and
[raster sizing](internal/imaging/svg.go) include encoded-source, XML-depth,
expansion-work and rasterization limits, plus recovery around rasterizer panics.
ICO admission validates the selected
directory entry and embedded dimensions. AVIF build guards require its WASM
implementation. These are scoped controls, not a claim
that all parser paths recover panics or all intermediate allocations are bounded.

Large accepted images, concurrent operations and decoder/runtime overhead can
still exhaust resources. A header check is not a bound on every operation needed
to read that header; many in-process decoder calls cannot be interrupted in
place. Image-cache budgets are not process-wide memory quotas. No universal CPU
or total-memory bound is established here. Codec/runtime selection must also be
checked for the actual build; this model does not claim every format runs in WASM.

### Experimental HEIC helper boundary

The [image reader](internal/imaging/source.go) performs bounded leading-brand
recognition and routes HEIC to an explicitly injected shared client. The default
reader refuses it. HEIC configuration, pixels and Exif parsing run in a fixed
WASI guest inside a dedicated, disposable [helper](cmd/picfetch-heic-worker/main.go).
The guest is built from the exact maintained h265 production source; image bytes
are data, not executable modules. No native codec is imported into the viewer.
HEIC refusal does not trigger parent metadata parsing or RAW/JPEG fallback.

The [runtime](internal/heicdecode/worker/runtime.go) supplies bounded standard
streams without guest filesystem mounts, environment, clock, randomness or
sockets. It caps WASM linear memory at **1 GiB**. Growth can temporarily retain
old and replacement backing buffers; the linear-memory ceiling does not cover
that overlap, the native Go/wazero process or parent/cached pixels. Eager
reservation was rejected after it broke native Linux qualification. The native
Go memory target remains soft.

[Limits](internal/heicdecode/limits.go) permit at most **64 MiB encoded input**,
**64 million pixels**, **256,000,000 output bytes**, **64 KiB normalized metadata**,
**4096 diagnostic bytes**, and **60 seconds** per request after admission.
Readers apply the current user file-size limit to each new read, bounded by the
shared owner's fixed 64 MiB ceiling. Raising the live preference cannot raise
that hard HEIC ceiling or replace the owner.
The output cap admits at most 32 million NRGBA64 pixels. One shared lane covers
the app and its analysis descendants; bounded queues prioritize interactive work
without indefinitely starving background requests. Bulk reads wait for admission
and helper readiness. Cancellation retires the process and pipes before releasing
the lane. Validated pixels and metadata cross back through the codec-free
[protocol](internal/heicdecode/protocol.go); helper identity is pinned to the
trusted installed package. Hashes are not an independent package signature.

| Platform | Candidate native helper boundary |
| --- | --- |
| Linux x64/ARM64 | Thread-synchronized default-deny seccomp, resource limits and a 2 GiB address-space ceiling before input. This is not a physical-RAM/cgroup limit. |
| Windows x64/ARM64 | Suspended zero-capability AppContainer setup, explicit inherited handles, private Job Object committed-memory/CPU/process limits and kill-on-close. Standalone standard-user native guards pass on both architectures; installed-MSIX and production distribution remain unqualified. |
| macOS Intel/Apple Silicon | Separately entitled App Sandbox helper and Hardened Runtime with verified owned file/network denial; helper-only executable-memory entitlement for wazero. No guaranteed hard total native-memory ceiling. |

The saved `experimentalHEIC` value describes user intent, separately from the
session's configured reader and its latest availability failure. Scanning,
restored sessions and Favorites use that same reader predicate. A failed helper
never selects a different decoder or replaces the service. Existing package
format queries and OS associations remain unchanged; sequence admission and
HEIC encoding stay unsupported.

Windows copies the verified installed helper into a dedicated user/SYSTEM-only
cache. A cross-process file lock serializes publication and cleanup. Copies are
bounded and hashed again before atomic publication; only the AppContainer's
read/execute grant is prepared, on the copied executable and its containing
directory. Reparse points and hard-linked executable entries are refused before
permission changes. Client-held file/directory handles deny replacement/deletion
until Stop/Wait has joined work. Each launch still verifies the pinned hash and
native readiness. A damaged inactive copy is rebuilt; obsolete live copies are
retained until a later safe cleanup. Cache/staging failure refuses activation.
The installed manifest remains part of the enclosing authenticated package;
a content hash does not establish publisher authenticity. Store source files
and their ACLs are never staging targets, and Store update authority is unchanged.

Windows helper startup supplies only `GOMAXPROCS` and the OS-reported
`SystemRoot`/`LOCALAPPDATA`; it does not inherit the parent's environment or
search path. Windows redirects the profile directory for the AppContainer.
Profile creation uses a bounded per-user/session cross-process mutex. These
startup controls pass standalone standard-user x64/ARM64 CI at `69fef1a`, including
the launch-time loopback permission query; they grant no additional AppContainer
capabilities or filesystem rights. Hosted runner results do not qualify every
Windows installation. Installed-MSIX qualification now checks desktop-session
ownership and obtains a fresh standard-user logon of that owner on disposable
hosted VMs. The earlier alternate-user launch failed before application startup;
package-context behavior remains unverified until both native CI targets pass.

Windows can drop blocked loopback traffic instead of returning a permission
error. Both parent-owned listeners are positively checked before launch, and
UDP has a bounded one-byte echo worker. A helper timeout qualifies only with a
live request and a verified zero-capability AppContainer identity. Before
creating each helper, the parent must confirm through the native loopback
exemption API that this identity is not exempt; the AppContainer itself cannot
perform this privileged query. Successful
communication, an absent listener or an API
failure refuses readiness. The listeners close before image input; their worker
is joined on every path. Linux/macOS still require explicit permission refusal.

Actual platform controls are verified before helper readiness; unsupported or
failed setup refuses input. OS controls and WASI restrictions protect different
boundaries. A decoder failure, runtime compromise and OS sandbox escape are
separate events. The fixed module, Go, wazero and kernel are trusted components;
containment is not proof of absence of defects. See the qualification record for
native execution evidence, executable-memory tradeoff and incomplete package
checks. Insufficient native headroom may refuse a job; raising limits or
dropping isolation is not a fallback.

Container transforms run in the guest. ICC/wide-gamut/HDR presentation is not
implemented: some such files can decode into untagged pixels rather than being
rejected. Successful decode and 16-bit transport do not establish color fidelity.
These HEIC controls do not isolate other codecs, rendering, or native ONNX work.

## 4. Filesystem discovery and mutation

[Folder scanning](internal/filescan/filescan.go) resolves symlinks for cycle
detection and deduplication, supports cancellation and defaults to **200,000
results**. It does not confine discovery beneath the originally selected folder:
a symlink can lead to other accessible files. A selected tree is not a stable
snapshot, especially on shared storage.

[Save/export/metadata writes](internal/imaging/save.go) use temporary files,
file synchronization and rename replacement, coordinated by
[in-process path transactions](internal/imaging/mutations.go). Save-through-link
intentionally follows the target; export resolves parent links and refuses an
existing symlink leaf. These controls reduce partial writes and conflicts among
PicFetch operations; they do not lock out another process changing the filesystem.
[Deletion](internal/ui/deletion/deletion.go) uses an OS Trash operation after a
confirmation that defaults to Cancel. Recoverability depends on the OS/storage.

OS integration adapters handle paths through platform-specific argument, URI or
data transport. Their correctness and the trustworthiness of installed helpers
remain part of the boundary; a filename is not authorization for a command.
Assess any unintended access or mutation by its actual target, user consent and
authority crossed, including shared-folder and application-specific permissions.

## 5. Similarity analysis, assets and caches

[Similarity Explorer](internal/similarity/client.go) sends source paths and
requests over local pipes to its own analysis subprocess. It uses native ONNX
Runtime, checks source versions and verifies pinned assets before loading.
Native inference requires cgo; the no-cgo implementation reports it unavailable.
HEIC requests from analysis use explicit inherited pipes to the same
application-owned HEIC admission lane; analysis workers cannot choose a helper
executable or start an independent lane. The analysis process itself retains
separate privileges and is not a general sandbox for other image codecs.

| Platform | Analysis-worker network control |
| --- | --- |
| Linux x64/ARM64 | Worker installs a thread-synchronized seccomp network-denial policy; setup failure stops startup. |
| macOS | Worker launches under a `sandbox-exec` network-denial profile. |
| Windows | Ordinary subprocess; no OS network isolation is configured. |

[Runtime checks](internal/similarity/offline.go) require actual TCP/UDP denial
before analysis on macOS/Linux; merely being disconnected does not satisfy them.
These network policies do not establish comprehensive filesystem confinement.
The worker retains filesystem access subject to its effective OS permissions;
Windows also permits networking. The [encoder](internal/similarity/encoder.go)
requests the telemetry opt-out before library loading and requires the runtime
API opt-out to succeed before session creation. This is not proof that every
dependency or build can never communicate.

[Asset installation](internal/similarity/assets_install.go) requires a user
download action, uses pinned sources, checks expected sizes and SHA-256, restricts
HTTPS redirects to listed provider domains, and extracts named runtime members
into staging before publication. Analysis itself does not start downloads.
Store builds use verified bundled runtime DLLs and download model data; model
cache overrides cannot replace the bundled runtime. These checks still trust
the selected upstream model/runtime and the shipped pins.

Worker events cross back into the parent and are part of its input-validation
boundary. [Cache records](internal/similarity/cache_payload.go) have size,
version, vector and preview checks, but cached paths, image facts, previews and
representations remain sensitive local data. Hash-based filenames do not
anonymize those contents. Filesystem protection and user-selected persistence
matter independently of cache integrity checks.

## 6. Network and privacy

Normal viewing and similarity analysis do not intentionally upload images or
representations. Optional downloads, update checks and maps do generate outbound
traffic and disclose ordinary connection information. This local-processing
design is not a guarantee against disclosure following a compromised component.

The [EXIF map](internal/ui/exifwin/exifwin.go) starts collapsed in a fresh window.
Expansion requests tiles for the image's location; while expanded it follows
navigation to subsequent GPS images. The configured
[tile origin](internal/ui/exifwin/tiles.go) is OpenStreetMap. Requests include
tile coordinates and a PicFetch user agent, not the source image, but the tile
area can reveal sensitive location information alongside the client's network
address. A configured origin alone is not a guarantee about redirect recipients.
The fetcher limits concurrency, response bytes and cache storage and uses
timeouts; these limits are not a total decode-memory guarantee.

Favorites, sessions, thumbnails, presets and analysis caches are unencrypted
local state. Their exposure depends on directory permissions, backups, storage
sharing and the authority of other processes. Metadata removal/omission is
privacy-sensitive: failure to honor the selected export or removal behavior can
expose location or identity when a user shares the result. GPS disclosure can
have serious consequences even when only one image is affected.

## 7. Updates and release supply chain

Standalone builds support manual update checks and optional automatic checks,
which default off. Disabling checks does not discard an already completed stage
scheduled for installation at shutdown. [Store builds](internal/distribution/distribution_store.go)
disable the GitHub update channel.

[Fresh downloads](internal/update/download.go) have a **200 MiB archive-download
cap** and require successful Sigstore release-attestation verification before
extraction, independently of an optional GitHub API digest. The
[verification policy](internal/update/attest.go) binds the archive digest, asset
name, repository, tag and package URL to the expected GitHub release-service
identity, using TUF trust and a signed timestamp. This authenticates the release
artifact; it does not establish that its source was reviewed or its behavior is
safe. [Extraction](internal/update/extract.go) rejects nonlocal paths and archive
links. The download cap is not an aggregate expanded-size quota.

Persisted staging stores provenance fields and the full extracted companion-file
inventory locally. Reuse/apply checks helper and manifest identity, rejects
missing/changed/unexpected companion files and validates the platform;
it relies on trusted local staging state rather than cryptographically sealed
metadata. Any claimed abuse must establish the attacker's existing permissions,
PicFetch's destination authority and the boundary actually crossed.

The candidate updater installs Linux/Windows helper files with executable
replacement and rollback, and replaces the entire signed macOS app bundle.
The currently released updater cannot perform the first helper-bearing upgrade:
it drops the helper and deletes staging, invalidating the enclosing macOS
signature. The first transition needs a complete reinstall or a separately
qualified bridge. New updater tests do not fix an already installed old updater.

The [release workflow](.github/workflows/release.yml) configures CI gates,
Windows signing/verification and release publication dependencies.
The protected Windows signing job has no repository checkout/toolchain and
executes no repository programs. Fixed PowerShell data operations refresh the
post-signing helper digest; both executable signatures are checked afterward.
[Signing documentation](docs/release-signing.md) describes required environment
protection and credential handling. Configuration does not prove that live
environment approvals are enabled, a particular CI run passed, or a downloaded
artifact is correctly signed. macOS packaging verifies ad-hoc bundle/helper
signatures; that does not establish Developer ID/notarization or App Store
qualification. Manual-download and Store distribution have their own
provenance/installation paths; the in-app updater's
checks must not be attributed to every installation method.

Compromise of trusted source, dependencies, workflows, signing accounts or
release services can affect many installations. Signatures and checksums do
not make a malicious authorized release safe. Operational release qualification
must be tied to the actual tag, build and distributed artifacts.

## 8. Severity guidance

These are project triage starting points, not automatic or universal ratings.
Use demonstrated impact, exploitability, user interaction, persistence,
recoverability, data sensitivity and the real permission boundary. Distinguish
a confirmed defect from an untested hypothesis.

| Starting point | Example impact to assess |
| --- | --- |
| Critical | Reliable code execution through ordinary content opening, or a compromised update/release path distributing attacker code broadly. Scope and prerequisites still matter. |
| High | Substantial unauthorized file access, destructive mutation, sensitive disclosure or code installation across a meaningful OS/application permission boundary. |
| Medium | Repeatable application resource exhaustion or crash with limited, recoverable impact; limited integrity/privacy failures when the actual consequences justify this level. |
| Low | Recoverable local-state errors or non-sensitive diagnostic/thumbnail defects that add little authority or impact beyond what the attacker already possesses. |

A same-user prerequisite lowers concern only when the attacker already has
equivalent relevant authority. Conversely, a nominally local issue can cross
TCC, sandbox, write-control or release boundaries. GPS exposure may warrant
high severity depending on context; availability failures can be more serious
when persistent, destructive or capable of destabilizing the host. Neither a
single-process crash nor user interaction determines the rating by itself.

## 9. Maintenance and reporting

Update the baseline and review this model when decoder/runtime choices, worker
permissions, network behavior, file mutation, persistence or distribution change.
Record implementation and qualification separately; planned safeguards and
successful source checks are not substitutes for deployment evidence.

Use the repository's existing [security policy](.github/SECURITY.md) to report
suspected vulnerabilities privately. Its preferred route is
[GitHub Security Advisories](https://github.com/frathe/picfetch/security/advisories/new).
Include the affected version/build, platform and impact; do not post sensitive
reports or private sample images in a public issue. This model adds no separate
disclosure policy or response-time commitment.
