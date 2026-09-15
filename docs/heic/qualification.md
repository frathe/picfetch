# HEIC restoration qualification, 2026-09-15

## Current disposition

**HEIC viewing is still disabled on every production platform.** This branch
contains a reproducible development WASI guest and codec-free protocol. It has
no production helper launcher, family-wide admission broker, signed helper,
packaging integration, or mandatory OS memory/capability enforcement. Those are
required before enabling formats. The application and similarity paths retain
their current behavior. No release or merge is authorized by this record.

## Exact source and distribution inventory

| Component | Reviewed input | Distribution record |
| --- | --- | --- |
| h265 HEIC/HEVC code | `github.com/gen2brain/h265` v0.2.3, commit `b2d46ba787d8f0a2025bd106443ab1b1c7cd010f`; module sum `h1:+fEP2Xf1CoZ21SxA2YpqnPZb6Y/hAEkcgjU1gMsOhrk=`; module ZIP SHA-256 `2838bcb83b8da357a19788ad5d8a9d55c16bd4a12961fa81e3ab282974f1ed7a` | Unmodified upstream. MIT text and both upstream copyright notices preserved in `notices/h265-LICENSE`. |
| Guest compiler/runtime | Go 1.27.1, `wasip1/wasm`, no CGo, no assembly | BSD license and patent grant in `notices/Go-LICENSE` and `notices/Go-PATENTS`. The selected guest graph has no vendored third-party standard-library imports. |
| PicFetch adapter/protocol | This branch's `scripts/heicguest` and `internal/heicdecode`; exact input hashes in `scripts/heicguest/decoder.json` | PicFetch MIT. Adapter changes do not patch or vendor the upstream source. |
| Development WASI host | Existing wazero v1.12.0; used by the fixed fixture generator and guest ABI tests | Apache-2.0 license and NOTICE in `notices/wazero-LICENSE` and `notices/wazero-NOTICE`. No version upgrade. |
| Production helper / OS library | Not implemented or shipped | Complete native-helper closure remains a gate; no new native decoder/runtime has been admitted. |

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
1 GiB WASM linear memory, and a separately enforced 2 GiB worker-family budget.
The output cap allows at most 32M NRGBA64 pixels. The protocol accepts only
positive finite limits; resource fields are contracts, not OS enforcement.
Guest decoding sets upstream `Threads: 1`. A native Go helper's runtime thread
count is a separate, still-unimplemented OS limit.

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
| macOS Intel/Apple Silicon | Separate sandboxed helper/XPC for privileges; a supported hard-memory mechanism must still be established | One bounded Apple Silicon address-space feasibility probe only; no decoder or sandbox qualification | Disabled |

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
not prove all macOS resource strategies impossible. Separate macOS protection
research is ongoing at Ronin's request. No weaker protection has been accepted.

## Subsequent explicit macOS decision

Ronin subsequently approved the practical macOS tradeoff: activation may proceed
once sandboxing and finite decoder memory are verified, even though the native
helper lacks a guaranteed hard total-memory cap and can cause memory pressure or
crashes. Capability restrictions remain mandatory. This supersedes the original
whole-worker-cap gate for macOS; the proposed 2 GiB number was an engineering
starting point. The current candidate still does not launch a production helper.
Implementation continues with bounded host allocations, byte-only validated IPC,
finite jobs/time, cancellation/cleanup, shared admission and platform sandboxing.
