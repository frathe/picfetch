# HEIC restoration qualification, 2026-09-15

## Current disposition

**HEIC viewing is still disabled on every production platform.** This branch
contains a reproducible WASI guest, codec-free protocol, bounded parent launcher
and a minimal helper. Native Apple Silicon tests exercise a signed App Sandbox
bundle and real decoding. Shared app/analysis admission and canonical imaging
integration now have focused regression coverage. Linux and Windows restrictions are
implemented but await native CI. Package construction and updater integration
are implemented, with native macOS signed-bundle evidence. Native distribution
qualification, the first-upgrade transition and representative camera
compatibility remain activation gates. No release or merge is authorized by this record.

The subsequent source-retention instruction is implemented by restoring the
exact maintained production copy in `third_party/h265`. The earlier unmodified
v0.2.3 guest is superseded: it did not contain every local hardening change.
The new source/artifact is undergoing native and compatibility qualification;
preserving those checks is not itself permission to activate HEIC.

## Exact source and distribution inventory

| Component | Reviewed input | Distribution record |
| --- | --- | --- |
| h265 HEIC/HEVC code | Maintained PicFetch production copy from `fc127b44e1c99447b8d150256563c6d4f99b8ad3`, based on upstream v0.2.2 `665fd95984177afef4a7efca7d50638e4b695c7a`. Exact 107-file baseline in `third_party/h265/PICFETCH-SOURCE.json`; its SHA-256 is `583c5ab3b021675acfac0bc7344a7ff7acdc8f0ae1ab94e804d7b931e3618b9b`. | Local hardening retained unchanged; MIT text and both copyright notices in the maintained LICENSE and `notices/h265-LICENSE`. |
| Guest compiler/runtime | Go 1.27.1, `wasip1/wasm`, no CGo, no assembly | BSD license and patent grant in `notices/Go-LICENSE` and `notices/Go-PATENTS`. The selected guest graph has no vendored third-party standard-library imports. |
| PicFetch adapter/protocol | This branch's `scripts/heicguest` and `internal/heicdecode`; exact input hashes in `scripts/heicguest/decoder.json` | PicFetch MIT; adapter and maintained decoder are separately inventoried. |
| WASI host | Existing wazero v1.12.0; fixed guest, helper and qualification tools | Apache-2.0 license and NOTICE in `notices/wazero-LICENSE` and `notices/wazero-NOTICE`. No version upgrade. |
| Native helper / OS library | `cmd/picfetch-heic-worker`, codec-free boundary, Go/wazero; macOS uses system Security/CoreFoundation frameworks via cgo | Separate minimal executable, no Fyne or native HEIC codec. Final shipped closure and notice placement remain packaging gates. |

The [pinned upstream README](https://github.com/gen2brain/h265/blob/b2d46ba787d8f0a2025bd106443ab1b1c7cd010f/README.md)
describes a pure-Go implementation without module dependencies and names
rust_h265 and oxideav-h265 as source projects. Its LICENSE credits roticv (2025)
and Karpeles Lab Inc. (2026). The reviewed module contains that license, but does
not identify exact translated-source revisions or a file-by-file lineage map.
This records the distributed notice and the remaining traceability gap; it does
not invent upstream provenance or claim the whole translation is qualified.
The prior removed `github.com/gen2brain/heic` Rust payload is unrelated to this
new module and remains excluded.

The upstream README explicitly separates its software copyright license from
HEVC patent rights. This work claims no patent clearance, purchases no license,
and does not change PicFetch's MIT license. Existing project-wide dependency
qualification obligations also remain open.

### Source reconciliation with the earlier v0.2.3 candidate

The previous module is pinned at `665fd95984177afef4a7efca7d50638e4b695c7a`.
Static source comparison found six production files changed: `heic/exif.go`,
`heic/heic.go`, `heic/sequence.go`, `hevc/deblock.go`, `hevc/decoder.go`, and
`hevc/headers.go`. Upstream changes include a missing-metadata guard, propagating
coded-frame limits into the HEVC decoder, bounded/delayed sequence-table work,
transform-size validation; the slice/deblocking differences are comments. Upstream test
changes were inventoried but not executed or copied.

The maintained v0.2.2-based source already contains the missing-metadata,
coded-frame and 32-sample transform checks. It additionally retains aggregate
coded-work accounting, metadata/extent limits, streaming bounded NAL traversal,
finite sequence retention and transformed configuration. Its stricter sequence
implementation is preserved in full; no complete equivalence to upstream's
different table representation is claimed. All 107 production/source-notice
files are byte-identical to the clean historical subtree, including unused
native assembly. The WASI guest selects `noasm`; native application imports
remain prohibited. The historical test corpus and THREAT-MODEL.md are not copied.

The exact upstream base module sum is
`h1:rnpfo8I4PFhohbij8ySYZlJFn07TTwV+/uWBrKsOEas=` and its ZIP SHA-256 is
`57a197c95e25b481abcd6d20773ebfb17d859e6d864017304aff6ec0e0f19757`.
Root and guest replacements select the maintained directory. Build guards
refuse missing/redirected replacements, changed baseline bytes and uninventoried
source files before reproducing the guest.

Two API facts affect this adapter:

- The still API can fall back to decoding an image sequence. PicFetch's guest
  therefore refuses every top-level `moov` box before calling upstream. This is
  also conservative for a still image accompanied by movie data.
- Decoding returns straight-alpha NRGBA8/NRGBA64; treating those bytes as
  premultiplied RGBA changes translucent pixels. Protocol v2 identifies the
  layout and preserves the full sixteen-bit samples. The earlier v1 boundary
  was never shipped and has no compatibility promise.

## Reproduction and guards

From the repository root, with Go 1.27.1:

```sh
make heic-build
make heic-check-provenance
make heic-check-imports
make heic-check-guest
```

`heic-build` checks the reviewed source revision, module and archive hashes,
verifies the module cache and notices, then builds with `GOOS=wasip1`,
`GOARCH=wasm`, `CGO_ENABLED=0`, `GOWORK=off`, `GOTOOLCHAIN=local`, empty
`GOEXPERIMENT`/`GOFLAGS`/`GOWASM`, `-mod=readonly -tags=noasm -trimpath
-buildvcs=false -ldflags='-s -w -buildid='`. It writes the artifact and its
input/hash manifest. `heic-check-provenance` refuses input inventory or hash
changes and independently rebuilds to a temporary file for byte comparison.
The manifest is a review record, not a signature or protection from a committer
who changes both sources and recorded hashes.

The import guard rejects direct native codec imports across build tags, checks
the selected native imaging/analysis graph, and permits only the exact h265,
PicFetch adapter and standard-library guest closure. The guest's entry point is
WASI-only in a separate Go module. The artifact is not embedded in the viewer.
Source/notice changes require an intentional rebuild and review. Guard tests
include mismatched source content, missing artifacts and forbidden imports.

`make heic-fixture` reproduces one ordinary 16x16 ten-bit gradient inside WASI.
Fixture provenance and scope are in `scripts/heicbuild/testdata/README.md`.
Upstream `basic.heic` and `main10.heic` are byte-identical in the pinned archive;
only the generated explicit-depth gradient is used as ten-bit evidence.

## Resource and compatibility limits

The candidate production maxima remain 64 MiB input, 64M pixels, 256,000,000
output bytes, 64 KiB normalized metadata, 4096-byte diagnostics, 60 seconds,
1 GiB WASM linear memory, and a requested 2 GiB native worker budget.
The output cap allows at most 32M NRGBA64 pixels. The protocol accepts only
positive finite limits; resource fields are contracts, not OS enforcement.
Guest decoding sets upstream `Threads: 1`. The helper sets Go's thread ceiling
to 32 and GOMAXPROCS to 1; this is not an OS process-family thread quota. Its
1.5 GiB Go memory target is soft. Readiness explicitly reports zero hard native
memory bytes on macOS under Ronin's accepted availability tradeoff.

The development fixture tests use 1 MiB input, 1M pixels, 8 MiB output, 128 MiB
WASM linear memory, bounded stderr/stdout, and a deadline. They provide no
untrusted files, filesystem preopens, environment, real clock or sockets to the
guest. The host test process is not claimed to have the production OS cap.
These are positive ABI checks with ordinary images, not memory-exhaustion tests.

Configuration currently decodes pixels to report the same transformed dimensions
as the displayed image. Optimizing this requires a qualified transformed-header
API. Container transforms are applied in the guest; Exif orientation is returned
as metadata. The imaging adapter keeps these validated pixels without applying
Exif orientation again. Color-profile/HDR handling, representative camera
compatibility and memory measurements remain unqualified.

The color limitation is stronger than missing test evidence: the maintained
decoder applies the YUV matrix and range, but deliberately leaves primaries and
transfer in the source color space (`third_party/h265/heic/heic.go`, Color
contract). Its ICC profile is reported only by `DecodeColor`; the guest calls
`Decode`, and the response protocol carries neither ICC nor CICP descriptions.
Consequently, ICC-managed, wide-gamut, PQ/HLG and gain-map HDR display are not
implemented. These files are **not deliberately rejected by color class**;
some can decode into untagged pixels without faithful display semantics.
Sixteen-bit sample transport is not an HDR or color-management claim. No color
engine redesign or new HDR feature is part of this qualification work.
Additional real-helper metadata/orientation, alpha and known-ramp checks are
recorded in the [ordinary compatibility report](compatibility-2026-09-16.md).

## OS enforcement status

| Platform | Documented candidate | Implemented/runtime verified here | Admission |
| --- | --- | --- | --- |
| Linux x64/ARM64 | All-thread seccomp, hard RLIMIT_AS virtual-address-space ceiling, CPU/core/file/descriptor limits before input; bounded WASI and parent lifetime | Candidate cross-builds; portable syscall-policy controls pass. Native helper execution and packaged application qualification await CI. No RSS or cgroup limit is claimed. | Disabled |
| Windows x64/ARM64 | AppContainer/restricted capabilities plus Job Object aggregate commit-memory/process/CPU/kill-on-close limits, explicit inherited pipes and suspended setup | AppContainer/suspended launch and job controls cross-build; owned native controls are wired in CI. Native Windows and MSIX execution remain unverified. | Disabled |
| macOS Intel/Apple Silicon | Independently entitled App Sandbox helper bundle, hardened runtime, bounded WASI, parent-owned pipes/deadline/group termination | Apple Silicon native ordinary decode, read/create/TCP/UDP denial and cancellation verified below. Intel/packaged application qualification pending. Hard total native cap absent by explicit approval. | Disabled pending qualification |

Linux's [cgroup v2 documentation](https://docs.kernel.org/admin-guide/cgroup-v2.html)
describes charged memory and explicitly permits temporary `memory.max` overshoot.
Swap is separately controlled, and unprivileged use depends on actual delegation.
Neither cgroup existence nor a CLI flag proves the worker is governed by it.

Windows [Job Object extended limits](https://learn.microsoft.com/en-us/windows/win32/api/winnt/ns-winnt-jobobject_extended_limit_information)
control aggregate committed memory, not an exact physical-RAM measurement.
[AppContainer](https://learn.microsoft.com/en-us/windows/win32/secauthz/implementing-an-appcontainer)
is a separate privilege boundary. Both must be established before input and
verified in the direct executable and MSIX distribution.

Apple's [published XNU resource implementation](https://github.com/apple-oss-distributions/xnu/blob/xnu-12377.121.6/bsd/kern/kern_resource.c)
sets an address-space map limit for `RLIMIT_AS`, rejecting one below existing
mappings. It is incorrect to say this API is universally ignored. This source
is evidence about address-space accounting, not a whole-helper resident-memory
limit or supported-version matrix. [Sandboxed XPC services](https://developer.apple.com/library/archive/documentation/MacOSX/Conceptual/BPSystemStartup/Chapters/CreatingXPCServices.html)
provide privilege/crash separation, not a configurable hard RAM ceiling by
themselves. Private or privileged jetsam controls are not an established shipping
solution for an ordinary app.

### Local macOS feasibility observation

On macOS 27.0 (26A428), arm64, a disposable C executable queried
`MACH_TASK_BASIC_INFO`, then attempted `setrlimit(RLIMIT_AS, {2 GiB, 2 GiB})`.
It did not allocate a large buffer, test an image, or change the parent process.
Result:

```text
virtual_bytes=500512735232 resident_bytes=1835008
setrlimit_2GiB=-1 errno=22 (Invalid argument)
```

The probe source is retained in `macos-rlimit-probe.c.txt`. Reproduce with:

```sh
clang -x c -Wall -Wextra -Werror docs/heic/macos-rlimit-probe.c.txt -o /tmp/picfetch-heic-rlimit-probe
/tmp/picfetch-heic-rlimit-probe
```

This eliminates this specific 2 GiB address-space setting on this host; it does
not prove all macOS resource strategies impossible. The later approval below
supersedes that hard-native-memory gate, without removing sandbox requirements.

## Subsequent explicit macOS decision

Ronin subsequently approved the practical macOS tradeoff: activation may proceed
once sandboxing and finite decoder memory are verified, even though the native
helper lacks a guaranteed hard total-memory cap and can cause memory pressure or
crashes. Capability restrictions remain mandatory. This supersedes the original
whole-worker-cap gate for macOS; the proposed 2 GiB number was an engineering
starting point. The current candidate launches a helper through its isolated
client tests; the application has not yet adopted that client.
Implementation continues with bounded host allocations, byte-only validated IPC,
finite jobs/time, cancellation/cleanup, shared admission and platform sandboxing.

## Native macOS helper evidence, 2026-09-16

The selected public route is an independently entitled helper `.app` bundle.
`SecTaskCopyValueForEntitlement` verifies App Sandbox at startup; owned external
read/create and owned loopback TCP/UDP probes must all return EPERM/EACCES before
the helper sends readiness or reads image input. A plain ad-hoc signed executable
aborted before main on this host; the signed bundle starts and strict codesign
verification passes. No deprecated `sandbox_init`/`sandbox-exec` route is used.
Apple recommends App Sandbox for computation helpers and describes separate
container rights in its [secure-helper guide](https://developer.apple.com/library/archive/documentation/Security/Conceptual/SecureCodingGuide/DesigningSecureHelpers/DesigningSecureHelpers.html)
and [current sandbox guide](https://developer.apple.com/documentation/security/protecting-user-data-with-app-sandbox).

App Sandbox is not zero native filesystem authority: the helper can access its
own container and some system resources. It also does not prove zero possible
native descendants. The guest gets no filesystem preopens, environment, socket
descriptors or native execution bridge. Generic WASI imports are not granted
capabilities. The trusted native host compiles only the fixed embedded module;
image bytes enter solely through bounded stdin. Parent code uses an absolute
hash-pinned executable, explicit stdio, a minimal environment, and no user image
paths in helper arguments. Trusted probe paths contain no image data.

The parent owns one admitted operation, at most 64 waiters, the 60-second
deadline, every pipe and process wait. It kills the Unix group before reaping
its leader (preventing PID reuse before a late kill) and joins pipe work before
releasing admission. An owned peer and descendant pass cancellation with an
observable connection close; deliberately killing only the leader makes that
guard fail. The shared Client now serves bounded analysis connections through
explicit inherited pipes. GUI and analysis construction inject that same owner;
production construction still supplies no owner until qualification completes.

### Executable-memory decision

Pinned wazero v1.12.0 maps anonymous code pages read/write and then changes them
to read/execute; it does not use MAP_JIT. See
[allocation](https://github.com/tetratelabs/wazero/blob/v1.12.0/internal/platform/mmap_other.go)
and [protection](https://github.com/tetratelabs/wazero/blob/v1.12.0/internal/platform/mmap_unix.go).
Apple's [allow-jit entitlement](https://developer.apple.com/documentation/bundleresources/entitlements/com.apple.security.cs.allow-jit)
specifically concerns MAP_JIT. The working compiler bundle therefore needs the
broader [unsigned-executable-memory exception](https://developer.apple.com/documentation/bundleresources/entitlements/com.apple.security.cs.allow-unsigned-executable-memory).
No dynamic-library validation, DYLD-environment, executable-page-protection,
network or user-file exception is added.

Actual controls on macOS 27.0 (26A428), arm64, Go 1.27.1:

| Hardened, App Sandbox helper | Ordinary 320x240 image | Owned 4032x3024 lossless gradient |
| --- | --- | --- |
| Interpreter, no executable-memory exception | 1.5206 s, decoded | Parent deadline at 30.0093 s; terminated and joined |
| Compiler, required unsigned-executable-memory exception | 1.6174 s, decoded | 7.4040 s, correct dimensions |

These are single cold runs including startup and compilation, not a general
performance guarantee. The initial comparison used the raw fixture; its checked
gzip wrapper now saves repository space without changing HEIC bytes. A separate
real ten-bit decode plus read/create/TCP/UDP denial and cancellation test passed
in 1.94 s with the hardened compiler bundle. Hardened Runtime without the
required exception refused the real decode, while its cancellation check passed.

Keep the compiler in the disposable helper: the interpreter failed the existing
deadline on an ordinary image of a common camera size. The broader entitlement
increases native executable-memory authority and therefore the trusted runtime
surface; it does not grant file or network access. Images are data, never modules
to compile. App Sandbox, WASI validation, fixed code, bounded IPC and process
termination remain separate protections, not a guarantee against every unknown
native/runtime/kernel defect. Developer ID signing/notarization, Intel execution,
own-container identity across updates and final packaged launch remain to verify.

Reproduce with `make heic-native-macos`; the native suite refuses skipped guards.
It uses only owned peers/probes and ordinary fixtures. Pure boundary tests also
verify initial/growing WASM memory limits, an owned loop's deadline, bounded
diagnostics, changed-helper refusal, blocked-writer timeout, crash, and Stop/Wait.
An owned WASI capability control confirms no filesystem preopen and detects a
deliberately installed temporary directory. The canonical native target passed
in 46.377 s after adding missing-sandbox refusal and descendant cleanup. Its
gzip-wrapped 12-megapixel fixture took 7.1366 s with the compiler; the interpreter
reached the unchanged deadline at 30.0108 s. Raw HEIC bytes are unchanged.
Full app functionality and broad camera/color/orientation qualification remain
open. The separate macOS research informed this record; native results above
come from this implementation task, not from documentation alone.

### Finite lifetime tuning after native Intel evidence

Signed worker `c4a89d1` passed native Intel App Sandbox, small-image decoding,
read/create/TCP/UDP denial and cancellation. Its 12-megapixel compiler comparison
reached the 30-second ceiling (30.060 s); the interpreter also timed out. This
was a controlled deadline rejection, not successful large-image qualification.
The same x86 transport/compiler path with the maintained source completed the
fixture in 5.967 s under Rosetta after using the installed ARM clang toolchain.
Rosetta is diagnostic evidence only and does not replace native Intel timing.

The lifetime ceiling is now proposed as a finite 60 seconds, with an explicit
rejection test above that value. The initial 30 seconds was an engineering
choice; native Intel ordinary-camera usefulness requires measuring beyond it.
The pending native Intel run must show its actual completion time before that
platform's common-camera performance is qualified. Longer occupation of the
single lane is the availability cost; file/network restrictions, WASM memory,
input/output/metadata/diagnostic sizes, thread settings and job count do not
change. Earlier 30-second measurements above remain historical observations.

The prior Linux race failure was compilation of the fixed real guest under
instrumentation: 37.08 seconds exceeded 30. The same focused ordinary-worker
test took 0.89 seconds without race instrumentation on this host; all owned
memory, capability, cancellation and stream tests passed in CI. The new finite
ceiling retains that real compiler test and all native guards; none is skipped.

The maintained-source candidate passes `make verify-build test-h265`, the
focused boundary/guest race suite, and native Apple Silicon qualification.
With the 60-second ceiling, its compiler decoded the 12MP fixture in 6.713 s;
the interpreter reached 60.012 s and was terminated/joined. The helper-only
executable-memory entitlement remains justified by this observed runtime
difference. Native Intel timing at the new ceiling is still pending CI.

## Linux native-helper candidate

The no-cgo amd64/arm64 helper now has a default-deny seccomp policy synchronized
across threads. It allows explicit Go scheduling, private futexes, stdio/poll,
signal and anonymous-memory operations. Exact architecture-specific CLONE_THREAD
flags permit runtime threads; process/exec, file opens, network, io_uring,
namespace and resource-policy changes remain denied. Wazero's private RW
mapping followed by RX protection is allowed; RWX and file-backed mappings are
not. Optional huge-page allocation and mapping-name decoration may be refused.
The compiled runtime and kernel remain trusted. This reduces native authority;
seccomp by itself is not a complete security guarantee. See the
[kernel seccomp documentation](https://docs.kernel.org/userspace-api/seccomp_filter.html).

Before input, the helper verifies finite hard/soft limits for virtual address
space (2 GiB), CPU time (rounded request seconds), core files (zero), file size
(zero), and descriptors (32). It refuses an already oversized starting address
space and requires ENOMEM from an oversized PROT_NONE reservation after policy
installation. The reservation touches no RAM. RLIMIT_AS constrains virtual
mappings, not physical RSS; the Go thread ceiling remains runtime enforcement,
not a per-family kernel PID quota. See [Linux resource-limit semantics](https://man7.org/linux/man-pages/man2/getrlimit.2.html).

Pure Go is part of this candidate's contract: cgo Linux builds and unsupported
architectures refuse startup. The `heic-linux` native suite records actual
thread synchronization/inheritance, address-limit negative control, owned
runtime cancellation, file/network denial and ordinary ten-bit/12MP decoding.
CI selects native amd64 and arm64 hosts using documented
[GitHub runner labels](https://docs.github.com/en/actions/reference/runners/github-hosted-runners).
Cross-build and portable BPF results are available locally; native execution
remains pending publication and CI. No production HEIC activation is included.


## Shared source and Windows integration checkpoint

The GUI owns one optional HEIC Client shared by foreground/background Readers
and both analysis subprocess types. Remote connections inherit explicit pipes,
wait for owner admission/native readiness before bulk reads, and retain the lane
through validated output delivery. The imaging Source retains ordinary encoded
data or validated HEIC pixels/metadata, so HEIC probing and decoding do not run
twice. Container transforms remain applied once. Display/preloads, comparison,
grid, EXIF, capture sorting, Favorite/Spiral/mosaic previews and similarity use
the injected readers. Production still supplies no owner and advertises no HEIC
formats while qualification is incomplete. Focused race results are in the plan.

The Windows candidate creates a zero-capability AppContainer process suspended,
with three explicit inherited stdio handles and a child-process restriction.
Before resuming it, the parent assigns a private Job Object with active-process
limit one, requested process/job committed-memory ceilings, finite user CPU time
and kill-on-close. The worker independently checks its AppContainer SID, empty
capability list and immediate job, then performs owned file/TCP/UDP denial
probes before readiness. [Microsoft documents creation attributes](https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-updateprocthreadattribute)
and [immediate-job queries](https://learn.microsoft.com/en-us/windows/win32/api/jobapi2/nf-jobapi2-queryinformationjobobject).

Dedicated helper provisioning grants that AppContainer read/execute on only the
helper and its immediate directory; the grant does not propagate to children.
Launch does not mutate ACLs. The profile can access its own storage and permitted
system resources; these controls do not establish zero native filesystem access.
The stable profile and immutable MSIX installation/staging path still need native
distribution qualification.

The `heic-windows` CI matrix requires owned token/job and unsandboxed-refusal
controls, inherited-pipe cancellation, analysis attachment cleanup, and ordinary
ten-bit/12-megapixel decode. The memory control uses a 64 MiB job and a fixed
65 MiB commitment attempt, touches no pages, and frees any unexpected allocation
immediately. It is a bounded OS-policy control, not an image decoder stress test.
Cross-build/vet/IDE results establish compilation only; no Windows execution has
been observed on this macOS host.


## Package updates and first-upgrade limitation

Authenticated archives now extract into a fresh payload directory. Stage
provenance records every companion-file digest and revalidates the exact file
set after persistence and before install. Linux/Windows prepare the dedicated
helper directory before replacing it, then roll it back if the existing binary
transaction fails. macOS installs the complete verified app bundle, including
its enclosing signature resources, nested helper, manifest, notices and sealed
resources. The viewer joins HEIC process/pipe retirement before applying updates.
Removing companion provenance from a helper-bearing stage is rejected; only
legacy stages without helper content may omit that proof.

Owned native macOS bundles with different helper signatures, manifests and
sealed resources pass strict deep codesign verification after successful
installation and after an injected final-rename failure restores the original
bundle. Neighboring owned user-data bytes remain unchanged. Reverting to the
binary/plist-only path deliberately makes this guard fail; the restored
implementation passes. These are signed fixture/package transaction checks,
not a claim of Windows or production distribution execution.

The final local `make heic-native-macos` run on 2026-09-16 passes all four
mandatory helper/runtime/update/legacy-transition guards. The compiler's 12MP
decode takes 6.772 s; the interpreter reaches 60.015 s and is terminated/joined.
The signed install/rollback guard passes in 1.310 s, and the expected legacy
transition limitation is observed in 1.220 s. The raw local event stream is
`.scratch/heic-qualification/native-macos.json`. Full updater race regressions
pass in 4.091 s; native guard registration regressions pass in 1.344 s.

**The currently released updater cannot perform the first helper-bearing
upgrade completely.** Its binary/plist replacement omits the helper and then
deletes the staged archive payload. The new app cannot recover the helper from
that cache. An owned native reproduction also leaves the enclosing macOS bundle
failing strict signature verification with an invalid resource-directory error.
The first helper-bearing version therefore requires complete package
reinstallation, or a separately designed and qualified bridge before release.
New updater code only governs subsequent upgrades. Missing/unqualified helper
availability remains closed; no release, repair download, or external user
notification is authorized or performed by this work.
