# Pure-Go HEIC and AVIF codec security evaluation

Evaluation date: 2026-09-15.

This evaluation covers `github.com/gen2brain/h265` and
`github.com/gen2brain/gav1d` against attacker-controlled image data. It
examines parsing, metadata, decoding, encoding, resource limits, dependency
licenses, source provenance and a scalar WASI isolation prototype.

No PicFetch dependency, decoder registration or supported format changed.
HEIC remains disabled.

## Decision

| Candidate | Exact source | Decision |
| --- | --- | --- |
| `gen2brain/h265` | v0.2.3, `b2d46ba787d8f0a2025bd106443ab1b1c7cd010f` | Do not use as published. A malformed HEIC can return an empty image without an error. The local fix addresses that case, but source provenance and HEVC patent clearance remain release blockers. Consider a pinned, patched snapshot only as a contingency and only inside a dedicated WASM worker. |
| `gen2brain/gav1d` | v0.2.5, `7aa50e4e898ebdb4794ae3df88ce8ad6cbb74608` | Do not use as published or with only the local patch. The audit confirmed header allocation denial of service, a grid pixel-limit bypass and invalid output from supported-looking encode calls. Sequences accepted by the patched parser and multi-picture item payloads can still exhaust a bounded worker, and the encoder signals the wrong AV1 level for many accepted dimensions. |
| Scalar WASI worker | Local prototype | The prototype contained tested guest memory exhaustion, CPU loops and output floods, and denied guest filesystem and environment access. Its TCP attempt failed and the host provides no socket extension. This is useful architecture evidence, not a production sandbox or security certification. |

The preferred AVIF direction remains a small PicFetch-owned adapter around the
established libavif/libaom stack described in
[the alternatives review](image-codec-alternatives-2026-09-15.md). Isolation
reduces the effect of a decoder defect; it does not correct the defect or make
the library release-ready.

The review fixes are retained as exact, auditable evidence:

- [gav1d v0.2.5 security patch](codec-hardening/gav1d-v0.2.5-security.patch),
  SHA-256 `e24da837a0ea3772a21e01de05ab8de8b3e7a817e759238923799c3372769661`.
- [h265 v0.2.3 security patch](codec-hardening/h265-v0.2.3-security.patch),
  SHA-256 `ca233b2ab81a36d80aee9f23a1ad596d563fd187c760f7d30ae6b1c7f76bf634`.

These patches have not received upstream or independent review. Shipping them
would make PicFetch responsible for maintaining security-sensitive forks.

## Exact environment and review boundary

| Component | Audited value |
| --- | --- |
| h265 | v0.2.3, commit `b2d46ba787d8f0a2025bd106443ab1b1c7cd010f`, dated 2026-09-15 |
| gav1d | v0.2.5, commit `7aa50e4e898ebdb4794ae3df88ce8ad6cbb74608`, dated 2026-08-18 |
| WASM host | wazero v1.12.0 |
| Host transitive module | `golang.org/x/sys v0.44.0` |
| Audit toolchain | Go 1.27.1, Linux/amd64, `CGO_ENABLED=0` |
| Codec manifests | Go 1.26.4; neither codec module declares another Go module |

Both codec repositories began in August 2026. Their native, scalar and fuzz
workflows are useful evidence, but their age and empty module manifests do not
establish parser maturity. The inspected trees contain about 59,500 production
Go lines, 23,800 Go test lines and 65,000 assembly lines. This was a targeted
review rather than an exhaustive line-by-line audit.

The h265 production packages had no `unsafe` import in the inspected snapshot.
Gav1d uses `unsafe` for slice reinterpretation and decoder operations even with
the `noasm` build tag; that tag removes architecture assembly, not every unsafe
operation. Neither production codec path imported networking, process
execution or syscall packages.

The source pins rely on archived upstream metadata because the disposable
source snapshots do not contain their Git histories. A release process would
need a PicFetch-controlled mirror, tree hash and reproducible build record.

## Threat model

An image can control container box counts, offsets, dimensions, grid layout,
sequence length, metadata and compressed bitstream contents. PicFetch must
therefore treat probing, configuration, metadata extraction, pixel decoding,
sequence handling and encoding as hostile operations.

The required boundary covers the complete codec operation. The parent may
perform a fixed, allocation-free magic check and enforce an input byte limit.
It must not call either codec's container parser, metadata reader,
`DecodeConfig`, decoder or encoder directly.

The primary risks considered here are process memory exhaustion, unbounded
CPU, panics, malformed output and violations of size invariants. Supply-chain
controls require an exact source mirror and reproducible guest digest. This
evaluation does not establish that either maintainer or a future release can
be trusted.

## Confirmed findings and red-green fixes

### GAV-01: header-controlled sample-table allocations

Gav1d v0.2.5 allocates from attacker-declared `stts`, `stsz`, `stco` and `co64`
counts before proving that the table payload or referenced media data exists.
Its frame-size limit is installed after container parsing, so it cannot protect
these paths.

Representative malformed inputs and cumulative Go allocation were:

| Table | Input size | Allocation per public operation |
| --- | ---: | ---: |
| `stts` | 88 bytes | about 393,457,000 bytes |
| `stsz` | 84 bytes | about 67,110,000 bytes |
| `stco` | 84 bytes | about 134,219,000 bytes |
| `co64` | 80 bytes | about 134,219,000 bytes |

The behavior reproduced through configuration, metadata, single-image decode
and sequence decode. It is an availability vulnerability in a process that
parses an untrusted AVIF.

The local patch:

- validates each table's declared entry count against its remaining payload
  before allocation;
- stores `stts` as cumulative timing runs instead of expanding every sample;
- retains a uniform `stsz` value without allocating one entry per sample;
- validates chunk runs, offsets, sample counts, integer arithmetic and complete
  file extents before materializing the sample layout.

The red harness observed the allocations above. The regression
`TestSampleTableAllocationBounds` covers all four table forms and malformed
layout relationships. The final patched probe rejects those headers with about
1.3 to 1.7 KiB of cumulative allocation.

### GAV-02: AVIF grid bypasses the frame pixel limit

Gav1d v0.2.5 applies `FrameSizeLimit` to each grid tile. It then retains every
tile and allocates the stitched output without applying the limit to the whole
canvas.

The repository's 2x2 fixture has four 160x120 tiles. Each tile fits a
32,768-pixel limit, while the returned 320x240 image contains 76,800 pixels.
The unpatched decoder returned that image successfully under the smaller
limit.

The local patch validates the complete grid area, aggregate coded tile area,
tile count, tile coverage, per-tile area and integer headroom before retaining
all tiles or allocating the output. The aggregate coded-area rule deliberately
counts padding outside the visible canvas because those pixels still consume
decode resources. `TestGridFrameSizeLimit` and `TestGridAllocationLimit` cover
rejection and an allowed grid.

### GAV-03: AVIF encoder reports success for an invalid bitstream

Encoding a 3648x2736 image with v0.2.5 returned a nil error and 6,538 bytes.
The same library then rejected those bytes as `av1: invalid bitstream`. A
3072x3072 control image, exactly 2,304 64x64 superblocks, encoded and decoded
successfully.

The encoder currently emits one AV1 tile. The local patch rejects dimensions
that exceed the single-tile 4,096-pixel width or 2,304-superblock constraints
until multi-tile encoding exists. It also prevents the four-bit encoded height
length from truncating dimensions above 65,536. `TestEncodeSingleTileLimit`
covers the rejected and supported boundaries. The finding concerns output
integrity; the audit did not demonstrate memory corruption.

The patch does not close a separate conformance problem: the encoder always
signals `seq_level_idx=0`, AV1 Level 2.0, while accepting frames beyond that
level's maximum 147,456 pixels, 2,048-pixel width and 1,152-pixel height. A
future encoder must calculate a valid level or reject dimensions that exceed
the signaled level. See [AV1 specification, Annex A](https://aomediacodec.github.io/av1-spec/av1-spec.pdf).

### H265-01: zero display dimension returns an empty image

Native and scalar fuzzing independently found malformed HEIC inputs whose
primary `ispe` property has a zero display dimension. `Decode` returned a
zero-sized image with a nil error. `DecodeConfig` already rejected the same
invalid property, so the public operations enforced different invariants.

The deterministic red case changed the width of `basic.heic` to zero and
reported `zero ispe width: <nil>, want ErrInvalid`. The local patch calls the
existing size validation before bitstream decoding or result cropping. Width
and height regressions then return `ErrInvalid`, and saved fuzz cases remain in
the post-fix corpus.

This is a parser/result-invariant defect. Downstream failure is plausible, but
the audit did not demonstrate a PicFetch crash or memory-safety violation from
this input.

### HEIC sample tables

The v0.2.3 h265 snapshot already contains a correction for the large
sample-table allocation behavior present in its preceding release. The four
malformed table forms stayed below about 10 KiB through configuration,
metadata, image decode and sequence decode. This result is limited to the
tested table shapes.

## Known residual codec risks

The patched gav1d snapshot still has two resource-exhaustion paths that make a
worker mandatory and block adoption:

- A sequence accepted by the patched parser ultimately materializes a 16-byte
  extent for every sample. A file near the 16 MiB input cap can describe almost
  16 million one-byte samples and request about 256 MiB before AV1 decoding
  rejects their contents. The PicFetch adapter must reject sequence tracks
  unless it gains an independent sample, frame, duration and aggregate byte
  budget.
- An AVIF item payload can cause `DecodeOBUs` to return multiple pictures. The
  container decoder retains every returned picture before selecting one.
  Per-frame and grid limits do not bound that aggregate.

These two paths are source and arithmetic review findings. The audit did not
retain conforming files that exercise them end to end.

The prototype's tested WASM memory ceiling is expected to turn allocations of
this class into bounded worker failures. It does not make the inputs succeed
safely, and they can still consume the worker's entire time and memory budget.
A released still-image adapter should request one picture only and reject any
additional output before retaining it.

Additional unresolved source-review items include inconsistent handling of
duplicate `stbl` children, no equality check between `stts` and `stsz` sample
totals, and delayed picture release that can increase peak memory in a reused
native process. These require specification review and hardening before
adoption.

H265 `Decode` may fall back to sequence decoding. A production HEIC adapter
must explicitly reject sequences; calling the still-image convenience API is
not by itself proof that only one frame will be decoded.

## Isolation prototype

The prototype compiles both patched codecs into a scalar `wasip1` guest. A
separate Go subprocess hosts that guest with wazero. The PicFetch UI process
would never link or call either codec under this design.

| Control | Prototype setting |
| --- | --- |
| Guest linear memory | 1,024 WASM pages, 64 MiB |
| Input | at most 16 MiB |
| Standard output | at most 16 KiB |
| Standard error | at most 4 KiB |
| Guest execution deadline | 2 seconds, with module closure on context completion |
| Host watchdog | 45 seconds; 40-second overall context |
| Guest concurrency | scalar build; one operation per process |
| Guest capabilities | arguments, standard streams, clocks, sleep and cryptographic randomness |
| Guest filesystem/environment | no filesystem or environment configuration |
| Guest network | no host socket extension; the test TCP attempt failed |
| Audit container | no network, read-only root, all capabilities dropped, no new privileges, UID/GID 65534, PID/CPU limits and a 2 GiB outer memory limit |

The final guest and host binaries used for the retained tests had these
SHA-256 digests:

- patched guest: `e0ce63220f9a8b59a89d5596fcedf851ae945de748030b4e98e24974bf7349eb`;
- host: `cc28716f0c9388bfee1478a8e102b313aad787e29a2521dd915a4348af5fed5a`;
- patched native probe: `cec3779a7526fc2202e0bf8935d103484d4009cd8665f5d6f34866d392d614e2`.

Within the Linux/amd64 Docker evaluation:

- the unpatched AVIF allocation input exhausted the 64 MiB guest with about
  46.8 MiB in use; the host remained able to report the closed module;
- an explicit memory bomb failed with about 56.8 MiB in use;
- an endless guest loop was canceled after about two seconds;
- an output flood stopped at exactly 16,384 captured stdout bytes;
- the guest could not see the host environment or read and write the
  host-mounted test paths, and its TCP attempt failed;
- a real 320x240 AVIF decoded with about 4.47 MiB cumulative guest allocation;
- a real 320x240 HEIC decoded with about 0.86 MiB cumulative guest allocation.

The compiler engine took about three seconds to compile the fixed guest per
fresh host process in these runs; valid decode execution took roughly 0.15 to
0.19 seconds. Interpreter smoke tests compiled in about 0.3 seconds and
decoded the same fixtures in about 1.0 to 1.3 seconds. These single-machine
observations are not performance benchmarks. They suggest compiling once in a
persistent host, while recycling guest instances after a bounded number of
requests or any fault.

### What the prototype does not guarantee

- The 64 MiB limit covers WASM linear memory. It does not cap the host Go heap,
  wazero compiler memory, input buffering or total process RSS.
- Docker supplied the hard process limit for this evaluation. PicFetch has no
  equivalent production worker implemented across Linux, macOS and Windows.
- The prototype decodes pixels inside the guest but returns audit JSON. It
  does not prove a bounded production pixel or metadata transport.
- The 16 MiB input, 64 MiB guest and two-second deadline are proof values. A
  representative corpus must determine product limits.
- The existing similarity worker is insufficient for this codec boundary:
  Linux mainly denies socket-related syscalls, macOS uses an allow-by-default
  profile with network denial, and Windows starts a normal child process.
- Scalar WASM excludes native SIMD and assembly. Native execution on other
  architectures remains outside this containment evidence.
- A guest panic or memory corruption can still corrupt that guest's own state
  and result. The parent must reject malformed output and recycle the worker.
- A fixed guest must be embedded or installed under a verified digest. A
  production host must never load a caller-selected WASM module.

A production child needs an operating-system boundary in addition to wazero:
cgroup v2 memory, PID and CPU policy plus syscall and filesystem restrictions
on Linux; Job Object limits and a restricted token or AppContainer on Windows;
and a proven App Sandbox or XPC arrangement on macOS. The relevant platform
controls are documented by the
[Linux kernel for cgroup v2](https://docs.kernel.org/admin-guide/cgroup-v2.html)
and [seccomp](https://www.kernel.org/doc/html/latest/userspace-api/seccomp_filter.html),
[Microsoft for Job Objects](https://learn.microsoft.com/en-us/windows/win32/api/winnt/ns-winnt-jobobject_extended_limit_information)
and [AppContainer](https://learn.microsoft.com/en-us/windows/win32/secauthz/appcontainer-isolation),
and [Apple for App Sandbox](https://developer.apple.com/documentation/security/app_sandbox).
A hard, portable whole-process memory mechanism for packaged macOS builds was
not established in this work.

## Production worker contract

If a later decision permits a pinned codec fork, the worker boundary should
enforce this contract:

1. The parent recognizes only fixed AVIF or HEIC magic and rejects an input
   above the byte cap. All other parsing occurs in the guest.
2. The host loads one embedded guest whose digest matches the build manifest.
   It does not accept a module path from the file, preferences or environment.
3. Each request has fixed-size framing and explicit limits for input, metadata,
   dimensions, pixel count, output bytes and time. Queue time and I/O time are
   included in the deadline.
4. The guest permits one still picture and one thread. It rejects image
   sequences and additional decoded pictures before retaining them.
5. The parent validates positive dimensions, overflow-safe dimension products,
   pixel format, stride, exact pixel byte length and bounded metadata lengths
   before allocating or publishing a result.
6. A deadline, malformed response, panic, memory exhaustion or excess output
   kills and joins the child. The host is recycled after any failure and after
   a bounded number of successful requests.
7. The packaged worker runs with no ambient filesystem, network or inherited
   environment access, and with hard total-process memory, PID and CPU limits.
8. Tests prove cancellation, no orphan process, capability denial, total-memory
   enforcement and result validation on every packaged operating system.

## Verification matrix

Counts below are emitted pass and skip markers from Linux/amd64 Docker tests.
Skipped reference tools and external conformance corpora remain unverified.

| Source | Package/build | Pass | Skip | Fail |
| --- | --- | ---: | ---: | ---: |
| upstream h265 | HEIC native | 174 | 6 | 0 |
| upstream h265 | HEIC `noasm` | 174 | 6 | 0 |
| upstream h265 | HEVC native | 1,477 | 7 | 0 |
| upstream h265 | HEVC `noasm` | 1,444 | 21 | 0 |
| upstream gav1d | AVIF native | 139 | 7 | 0 |
| upstream gav1d | AVIF `noasm` | 139 | 7 | 0 |
| upstream gav1d | AV1 native | 300 | 6 | 0 |
| upstream gav1d | AV1 `noasm` | 292 | 14 | 0 |
| locally patched h265 | HEIC native | 174 | 6 | 0 |
| locally patched h265 | HEIC `noasm` | 174 | 6 | 0 |
| locally patched gav1d | AVIF native | 156 | 7 | 0 |
| locally patched gav1d | AVIF `noasm` | 156 | 7 | 0 |
| locally patched gav1d | AV1 native | 301 | 6 | 0 |
| locally patched gav1d | AV1 `noasm` | 293 | 14 | 0 |

The four patched gav1d package/build combinations also passed with
`-gcflags=all=-d=checkptr=2`, with the same pass and skip counts.

| Check | Result | Limit |
| --- | --- | --- |
| Unpatched admission harness | Four of five initial checks failed; the expanded eight-test run contained 20 failing cases | Targeted malicious inputs |
| Final patched admission harness | 9/9 passed across 40 probe invocations | Native scalar probe |
| Final patched WASM isolation | 9/9 passed | Linux/amd64 Docker prototype |
| Patched h265/HEVC guided fuzz | Six 20-second campaigns passed | Native and `noasm`, container and bitstream decode, bitstream encode |
| Final patched gav1d guided fuzz | Four 20-second campaigns passed | AVIF reached only 31-32 executions; AV1 reached about 56,000-100,000 executions |
| Interpreter smoke | 3/3 passed | Two valid fixtures and CPU-loop cancellation |
| `go vet` | Passed for patched codec packages in native and `noasm` builds, the probe and the host | Offline, Linux/amd64 |
| GoLand inspections | Probe and host clean; patched-source findings exactly matched the unpatched files | Candidate sources retain duplicate-code and test-hygiene warnings |
| Linux/386 compile | Four patched `noasm` package test binaries compiled | Compile only; the current host sandbox rejected 386 execution |
| `govulncheck` v1.7.0 | No finding records at module scan level; database dated 2026-09-10 | Does not cover the new source defects found here |

The upstream suites passed while targeted tests found four defects. These
results therefore cannot serve as conformance or security certification. A
20-second fuzz campaign is a regression smoke test; it is especially shallow
for full AVIF decoding.

## License and provenance review

Both codec manifests have no runtime Go module requirements. Their copied and
ported implementations, patent terms, the Go runtime and the isolation host
still form a distribution closure.

| Component | Compatibility and obligations | Open issue |
| --- | --- | --- |
| h265 | Root MIT license. The identified `rust_h265` source is MIT OR Apache-2.0; `oxideav-h265` is MIT. Preserve the selected upstream copyright, permission and notice terms after mapping the imported source. | Its README identifies both ports but no exact translated revisions. Map the imported source and applicable roticv, OxideAV and Karpeles Lab notices before distribution. |
| HEVC implementation | The source copyright licenses do not themselves establish HEVC patent clearance. | Product/legal review must decide territorial and distribution obligations. |
| gav1d | BSD-2-Clause. Source distributions retain the notice, terms and disclaimer; binary distributions reproduce them in documentation or other supplied materials. | Exact dav1d and libaom source revisions are not recorded. |
| AOM-derived encoder code | Reproduce the AOM Patent License. A distributor benefiting from it must make its Necessary Claims available under that license; defensive termination applies to specified patent litigation. | Confirm the shipped code is an Implementation covered by the grant and deliver the patent text. |
| dav1d-derived code | Corresponding upstream dav1d arithmetic/table source has a Two Orioles LLC notice that is not visible in gav1d's root notice. | This is a notice-completeness question, not a conclusion of infringement. Map every ported file to an exact upstream revision and preserve applicable notices. |
| wazero v1.12.0 | Apache-2.0. Include the license and applicable NOTICE text; mark modified files if wazero is modified. | Record the exact host build and notice bundle. |
| x/sys v0.44.0 | BSD-3-Clause notice, disclaimer and non-endorsement condition; retain its patent text in the distribution record. | Include it in the worker SBOM and notices. |
| Go/WASI runtime | Go's source license and patent files apply to the compiled runtime payload. | Inventory the actual guest and host toolchain payload. |

MIT, BSD-2-Clause, BSD-3-Clause and Apache-2.0 components can coexist with
PicFetch's MIT application license when their notices and terms are met. That
copyright-license compatibility does not resolve the source-mapping gaps or
HEVC patent question. The inspected upstream terms are retained in the
[h265 LICENSE](https://github.com/gen2brain/h265/blob/b2d46ba787d8f0a2025bd106443ab1b1c7cd010f/LICENSE),
[gav1d COPYING](https://github.com/gen2brain/gav1d/blob/7aa50e4e898ebdb4794ae3df88ce8ad6cbb74608/COPYING)
and [gav1d PATENTS](https://github.com/gen2brain/gav1d/blob/7aa50e4e898ebdb4794ae3df88ce8ad6cbb74608/PATENTS).

## Adoption gates

Neither library should enter PicFetch until all applicable gates are complete:

1. Create a PicFetch-controlled fork or vendor snapshot at an exact tree hash.
   Preserve the red tests and reviewed fixes. Do not follow a floating tag or
   load an installed native library.
2. Produce a reproducible scalar WASM guest, SBOM, build recipe and expected
   digest. Verify the digest before execution.
3. Put magic probing, container parsing, configuration, Exif/XMP/ICC
   extraction, still decoding and encoding inside the worker.
4. Decode one requested still frame with one guest thread. Reject sequence
   tracks and extra pictures until aggregate frame count, pixels, duration and
   output bytes have independent limits.
5. Implement the bounded result protocol and worker lifecycle described above.
6. Enforce measured input, output, pixel, metadata, time and concurrency limits.
7. Add hard total-process memory, PID/CPU and capability/filesystem restrictions
   for packaged Linux, Windows and macOS builds. Prove each packaged boundary.
8. Run sustained coverage-guided fuzzing and retained regression corpora over
   container, metadata, AV1/HEVC and worker protocol paths. Add differential
   tests against independent reference decoders.
9. Qualify color, alpha, 8/10/12-bit data, grids, orientation, ICC, Exif, XMP,
   truncation and malformed offsets on a licensed real-image corpus. Verify
   encoded AVIF output with independent decoders.
10. Correct the gav1d level signal or restrict the encoder to its declared
    level, then run an independent AV1/AVIF conformance suite.
11. Map every ported source file to an exact upstream revision and close the
    notice gaps. Deliver MIT/BSD/Apache/AOM/Go notices and obtain a decision on
    HEVC patents.
12. Recheck vulnerability databases and upstream history at the final pins.
    Run formatting, vet, `checkptr`, GoLand inspections and the applicable
    PicFetch verification suite after integration.
13. Update `ARCHITECTURE.md`, packaging manifests and the HEIC removal record
    only when an implementation satisfies these gates.

## Claims this evaluation does not make

This work does not establish the absence of exploitable defects, decoder
conformance, production performance, complete image fidelity, reproducible
builds, cross-platform containment, total-process memory safety, license
clearance or HEVC patent clearance. It did not review every Go or assembly
path, run a long fuzz campaign, execute external conformance suites, build a
production pixel protocol or contact upstream.

The local patches show that the confirmed cases can be addressed. They are
evidence for a future decision, not approval to restore HEIC or replace the
current AVIF implementation.
