# HEIC restoration qualification, 2026-09-15

## Current disposition

**HEIC viewing is still disabled on every production platform.** This branch
contains a reproducible WASI guest, codec-free protocol, bounded parent launcher
and a minimal helper. Native Apple Silicon tests exercise a signed App Sandbox
bundle and real decoding; Linux/Windows enforcement, shared app/analysis
admission, final packaging and canonical imaging integration are incomplete.
Those remain activation gates. No release or merge is authorized by this record.

The subsequent source-retention instruction also requires reconciling the
maintained `third_party/h265` copy recorded in the historical checkout with this
candidate's upstream v0.2.3. Its local hardening must be retained unless equivalent
upstream checks are verified. The historical patch record has been read; this
candidate does not yet establish that equivalence or select final shipped source.

## Exact source and distribution inventory

| Component | Reviewed input | Distribution record |
| --- | --- | --- |
| h265 HEIC/HEVC code | `github.com/gen2brain/h265` v0.2.3, commit `b2d46ba787d8f0a2025bd106443ab1b1c7cd010f`; module sum `h1:+fEP2Xf1CoZ21SxA2YpqnPZb6Y/hAEkcgjU1gMsOhrk=`; module ZIP SHA-256 `2838bcb83b8da357a19788ad5d8a9d55c16bd4a12961fa81e3ab282974f1ed7a` | Unmodified upstream. MIT text and both upstream copyright notices preserved in `notices/h265-LICENSE`. |
| Guest compiler/runtime | Go 1.27.1, `wasip1/wasm`, no CGo, no assembly | BSD license and patent grant in `notices/Go-LICENSE` and `notices/Go-PATENTS`. The selected guest graph has no vendored third-party standard-library imports. |
| PicFetch adapter/protocol | This branch's `scripts/heicguest` and `internal/heicdecode`; exact input hashes in `scripts/heicguest/decoder.json` | PicFetch MIT. Adapter changes do not patch or vendor the upstream source. |
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

### Changes since v0.2.2

The previous module is pinned at `665fd95984177afef4a7efca7d50638e4b695c7a`.
Static source comparison found six production files changed: `heic/exif.go`,
`heic/heic.go`, `heic/sequence.go`, `hevc/deblock.go`, `hevc/decoder.go`, and
`hevc/headers.go`. Upstream changes include a missing-metadata guard, propagating
coded-frame limits into the HEVC decoder, bounded/delayed sequence-table work,
transform-size validation and slice/deblocking maintenance. Upstream test
changes were inventoried but not executed or copied.

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
output bytes, 64 KiB normalized metadata, 4096-byte diagnostics, 30 seconds,
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
as metadata. Their precedence in the eventual imaging adapter, color-profile/HDR
handling, representative camera compatibility and memory measurements remain
unqualified. No application integration may silently assume they are solved.

## OS enforcement status

| Platform | Documented candidate | Implemented/runtime verified here | Admission |
| --- | --- | --- | --- |
| Linux x64/ARM64 | Delegated cgroup v2 with `memory.max`, `memory.swap.max=0`, process limits and restricted filesystem/network/syscalls installed across threads before input | No Linux kernel/daemon in this worktree; no delegated hierarchy or packaged helper tested | Disabled |
| Windows x64/ARM64 | AppContainer/restricted capabilities plus Job Object aggregate commit-memory/process/CPU/kill-on-close limits, explicit inherited pipes and suspended setup | No native Windows or MSIX execution; no helper/restriction implementation | Disabled |
| macOS Intel/Apple Silicon | Independently entitled App Sandbox helper bundle, hardened runtime, bounded WASI, parent-owned pipes/deadline/group termination | Apple Silicon native ordinary decode, read/create/TCP/UDP denial and cancellation verified below. Intel/packaged application qualification pending. Hard total native cap absent by explicit approval. | Viewer integration pending |

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

The parent owns one admitted operation, at most 64 waiters, the 30-second
deadline, every pipe and process wait. It kills the Unix group before reaping
its leader (preventing PID reuse before a late kill) and joins pipe work before
releasing admission. An owned peer and descendant pass cancellation with an
observable connection close; deliberately killing only the leader makes that
guard fail. App-family sharing remains pending; one Client is not yet a complete
cross-process broker.

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
