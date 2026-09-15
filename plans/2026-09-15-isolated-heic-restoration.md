# Isolated HEIC/HEIF restoration

Date: 2026-09-15. Route: **Deep SDD/TDD**. The deliverable is decode-only
HEIC/HEIF support through one bounded, disposable WASI decoder helper, admitted
only on platforms where both process-family memory enforcement and capability
isolation have been exercised at runtime. This plan is also the durable evidence
record. It does not authorize merge or release.

## Intent and honest limit

Restore HEIC/HEIF without loading a codec, container parser, or native HEIC
library into the desktop process. Untrusted container, sequence, tile and pixel
work belongs in a pinned WASI guest inside a minimal helper. The desktop parent
does bounded brand recognition and validates a versioned response before it
allocates or publishes pixels.

Compilation is not platform qualification. Until a platform's mandatory
isolation and process-family memory controls pass in the packaged application,
admission on that platform fails closed and PicFetch does not advertise HEIC
there. The current Codex Cloud runner cannot provide Docker, Windows, macOS,
cgroup delegation, signing, MSIX, AppContainer, Job Object, or XPC runtime
evidence. HEVC may require patent licences in some jurisdictions; this work does
not provide patent clearance and does not change PicFetch's MIT licence.

## Decisions (do not relitigate)

| Topic | Decision |
| --- | --- |
| Decoder | `github.com/gen2brain/h265/heic`; never `github.com/gen2brain/heic`, `gav1d`, the removed Rust payload, or native fallback. |
| Boundary | Dedicated `cmd/picfetch-heic-worker`; no Fyne self-exec initialization. Codec/parser/sequences/tiles/pixels remain in its WASI guest. |
| Protocol | Versioned, length-prefixed binary IPC over explicitly inherited pipes. One request and response per disposable helper; reject unknown versions, statuses, fields and trailing bytes. No paths. |
| Capabilities | Guest receives byte streams only: no preopened filesystem, environment, clock beyond runtime need, secrets, or network. Helper receives only IPC/control handles. |
| Admission | One live lane per app instance. Acquire before the expensive source read. FIFO aging with bounded foreground preference; cancellation removes a waiter. No retry loop. |
| Proposed ceilings | 30 s total; 1 GiB WASM linear memory; 2 GiB OS-enforced process-family committed memory; input `min(user positive limit, 64 MiB)`; 64M pixels; checked RGBA output derivation; 64 KiB metadata; 4096-byte diagnostics; one guest thread/process and one live job. Lower after measurements, never make unbounded. |
| Output | Still image only. Reject sequences, unsupported bit depth/pixel format, zero/oversized dimensions, invalid stride, length mismatch, excessive metadata/diagnostics and extra data. Parent converts only a completely validated result. |
| Integration | `internal/imaging` remains canonical for probe, decode, record, capture date, metadata, thumbnails and previews. Similarity keeps its existing capability behavior and never gains a fallback parser. |
| Exposure | Chooser, MIME, manuals, translations and packaging change only for runtime-qualified fail-closed availability. |

## Dependency, source and licence inventory

The selected source must be pinned by immutable commit and Go module checksum,
not merely a tag. Qualification is incomplete until every row has an exact
source digest, artifact digest, licence and shipped notice location.

| Component | Exact source / artifact | Licence and obligations | Status |
| --- | --- | --- | --- |
| `github.com/gen2brain/h265/heic` | v0.2.3 at `b2d46ba787d8f0a2025bd106443ab1b1c7cd010f`; module/archive/source hashes and static v0.2.2 comparison in `docs/heic/qualification.md`. | Upstream licence and complete generated/translated-source provenance must be inspected; ship its notice. | Source recovered locally; MIT notice retained. Exact translated-source lineage and full decoder qualification remain incomplete. Sequence fallback is refused inside the guest. |
| WASI guest | Reproducible build recipe will name compiler version, target, flags and source tree digest; checked-in artifact gets SHA-256. | Same closure as selected decoder plus compiler/runtime notices where distributed. | Development artifact built and reproduced; production qualification pending. |
| wazero | Existing wazero v1.12.0; now directly used by the development fixture generator and ABI tests. No version upgrade. | Apache-2.0 notice and NOTICE obligations. | Apache-2.0 license and NOTICE preserved in `docs/heic/notices`; no production HEIC runtime. |
| PicFetch host/helper/protocol | This repository and guest manifest input hashes. | MIT. | Protocol/guest implemented; production client/helper pending OS controls. |
| OS restriction helpers | Standard-library/syscall use plus existing pinned `golang.org/x/sys`; no new native runtime. | Existing BSD-3-Clause notice. | Planned per platform. |

The old removed implementation used `github.com/gen2brain/heic` with a Rust
`heic` 0.1.6 payload under AGPL-3.0-only or commercial terms. It is excluded
from this design and must not reappear in either host or guest closure.

## File map

| Path | Responsibility |
| --- | --- |
| `internal/heicdecode/` | Context-bearing `Client` (`Decode`, `DecodeConfig`, `DecodeExif`), validated `Limits`, admission, typed failures, protocol validation, process lifecycle. |
| `internal/heicdecode/platform_*.go` | Platform availability and mandatory restriction setup; failure is an unavailable typed error, never unrestricted execution. |
| `cmd/picfetch-heic-worker/` | Minimal private helper dispatch and guest invocation; no UI imports. |
| `internal/imaging/{loader,exif,thumbnail,preview}.go` | Canonical HEIC routing after runtime qualification; ordinary 200M-pixel and encoded-input semantics stay unchanged. |
| `scripts/heicprovenance/`, `scripts/heicimports/` | Reproducible source/artifact and forbidden-import checks backing Make targets. |
| `scripts/nativeguards/` | Actual Linux, Windows, macOS/store isolation acceptance. |
| `docs/heic/robustness-testing.md` | Permitted defensive scope, budgets, observables and explicit omissions. |
| `THIRD-PARTY-NOTICES.md`, packaging/manual/translation files | Exact notices and exposure, only after qualification. |
| `ARCHITECTURE.md`, `todos.md`, this plan | Package map, remaining work and evidence. |

Every new `_test.go` receives an exact `qodana.yaml` DuplicatedCode exclusion.
Any new top-level `internal/ui` test receives a sorted shard assignment and
updated count.

## Acceptance criteria and commands

1. **Provenance and closure.** Exact immutable source and guest artifact are
   reproducible, checksummed, licensed and noticed; forbidden decoders/native
   backends are absent.
   `make heic-check-provenance && make heic-check-imports`
2. **Bounded protocol/client.** Owned protocol property tests and fake helpers
   cover malformed/trailing/unknown replies, output arithmetic, metadata and
   diagnostic caps, cancellation, timeout, crash, startup failure, queue
   priority/no-starvation, cleanup and next-request recovery.
   `go test -race -tags=no_emoji -count=1 ./internal/heicdecode/...`
3. **Canonical integration.** Valid licensed fixtures cover probe/decode,
   renamed input, metadata/orientation/alpha/bit depth, grid and existing routes;
   ordinary formats, AVIF, 200M pixels, cache generations and encoded limits
   remain intact.
   `go test -race -tags=no_emoji -count=1 ./internal/imaging ./internal/favthumbs ./internal/filescan`
4. **Similarity/UI lifecycle.** Existing capability reporting is preserved and
   root/UI work settles through existing queues and shutdown barriers. Focused
   named tests are recorded in the evidence ledger before completion.
5. **Packaging and locale.** Associations reflect runtime-qualified availability
   only; no missing locale key or Unicode arrow.
   `go test ./scripts/plistdoctypes ./scripts/msixstage` plus the repository's
   named locale/manual guards.
6. **Platform isolation.** Native guards exercise Linux cgroup v2 memory.max
   (including swap/overshoot/permissions) and all-thread seccomp; Windows
   restricted token/AppContainer, explicit handles, no network/user files and
   Job Object memory/CPU/process/kill-on-close in direct executable and MSIX;
   macOS signed minimal sandbox/XPC helper, denied capabilities and supported
   hard-memory control on Intel and Apple Silicon. Missing mandatory enforcement
   produces a deterministic unavailable error.
7. **Defensive security.** `make security-govulncheck` includes host and guest;
   static/import/provenance checks pass. Testing remains within
   `docs/heic/robustness-testing.md`; no exploit corpus or vulnerability discovery.
8. **Repository gates.** `make fmt`; `make check-qodana-test-exclusions`;
   `make check-test-shards-direct`; `make verify-build`; and `make verify` on
   native Linux/amd64 Docker/CI. Changed Go files receive GoLand inspections.

## Task graph and budgets

`T0 -> T1 -> T2 -> T3 -> T4 -> T5`. Platform implementations may proceed
independently only after T1 contracts are fixed. The lead owns architecture,
review and every fix; no review agent is used.

### T0 - feasibility and provenance
Owner: T0 inline. Files: this plan, source/licence records, provenance scripts.
Contract: exact reviewed source/artifact pins or an explicit blocker; no decoder
code enters the graph first. Test: provenance/import guards fail for missing or
forbidden closure. Verify: AC1. Budget: 0 spawns, 1 review, no full suite.

### T1 - protocol, client, helper and runtime
Owner: T0 inline. Files: `internal/heicdecode`, helper, tests. Depends: T0.
Contract: context-bearing API and typed deterministic failures under validated
limits. Test: AC2 with owned fake helper/guest. Verify: AC2. Budget: 0 spawns,
2 reviews, no full suite.

### T2 - restrictions and admission
Owner: T0 inline. Files: platform pairs and nativeguards. Depends: T1. Contract:
one fair priority lane; mandatory controls fail closed and kill/join descendants.
Test: benign capability/memory probes. Verify: AC6. Budget: 0 spawns, one review
per executable platform, no full suite.

### T3 - imaging and process-family integration
Owner: T0 inline. Files: canonical imaging callers and existing tests. Depends:
T1-T2. Contract: one path for full images and all derived consumers; no rich
parent parsing or fallback. Test/verify: AC3-AC4. Budget: 0 spawns, 2 reviews.

### T4 - native/package qualification and exposure
Owner: T0 inline. Files: native guards, packaging, docs/manual/translations.
Depends: T3 and actual runtime evidence. Contract: expose only qualified runtime
targets. Test/verify: AC5-AC7. Budget: 0 spawns, platform evidence required.

### T5 - final review, land and publication
Owner: T0. Depends: all tasks. Update architecture/todos/ledger; inspect changed
Go files; run AC8 once; commit and publish draft PR without merge/release.
Budget: 0 spawns, one final review and one full suite.

Delegation gate: all tasks stay with the lead because architecture, platform
security, dependency judgement and cross-package contracts fail G2/G3/G4; review
and fixes are never delegated. Rule S applies only to deterministic catalogue or
manifest generation. Rule W is satisfied.

## Safe validation scope

Use ordinary licensed HEIC photographs and PicFetch-owned bounded fake
helpers/guests. Benign probes may request denied file/network access, bounded
over-allocation, hang, crash and malformed IPC. Do not create malicious images,
run historical exploit corpora, conduct third-party vulnerability discovery, or
route rejected work through another model. Protocol property tests generate
only bounded PicFetch-owned byte messages. No fuzz coverage is claimed unless an
actual bounded fuzz command and duration are entered below.

## Evidence ledger

| Date | Task/check | Result and evidence |
| --- | --- | --- |
| 2026-09-15 | Repository/base | User supplied trusted connector evidence: public `frathe/picfetch`, default `main`, pull/push allowed, `main` = `0335a44384435a7680773c323b43016e839c07de`, and no HEIC branch collision. Local HEAD matched and branch `codex/heic-isolated-decoder` was created. |
| 2026-09-15 | Historical context | Read `finished_refactorings/2026-09-14-remove-heic-decoder.md`; recovered the removed research records from commit `aeb7176` for context. Historical prototype commit `fc127b44e1c99447b8d150256563c6d4f99b8ad3` is absent locally, as permitted by the task. |
| 2026-09-15 | Upstream feasibility | `go list -m -json github.com/gen2brain/h265@v0.2.3` failed at `proxy.golang.org` with HTTP 403. No h265 source exists in the module cache. Exact upstream diff, immutable revision, source checksum, licence closure and reproducible guest artifact therefore remain unverified; no decoder dependency or artifact may be added yet. |
| 2026-09-15 | T1 bounded response foundation | Red test first failed on absent `Limits`/protocol identifiers. Added finite-limit validation and a strict response reader that checks version/status, dimensions, multiplication, stride, bit depth, pixel format, every length and EOF before publishing RGBA pixels. `go test -race -tags=no_emoji -count=1 ./internal/heicdecode/...` passed in 1.010s. The client, helper and guest remain pending T0. |
| 2026-09-15 | Repository guards | `make check-qodana-test-exclusions`, `make check-test-shards-direct`, `make fmt`, and `git diff --check` passed. No top-level UI test was added. |
| 2026-09-15 | Existing imaging regressions | The brief's literal `go test -race -tags=no_emoji ...` failed at setup because the repository deliberately requires `nodynamic` for imaging (`internal/avifpolicy` otherwise has no selected file). The convention-compliant `go test -race -tags=no_emoji,nodynamic -count=1 ./internal/imaging ./internal/favthumbs ./internal/filescan` passed: 42.165s, 1.114s, 1.067s. |
| 2026-09-15 | Packaging/build | `go test -tags=no_emoji,nodynamic ./scripts/plistdoctypes ./scripts/msixstage` passed (0.005s, 0.297s). `make verify-build` passed its TUF, vector, asset, updater-notice and repository-wide vet checks. |

## Local continuation (2026-09-15)

Recovered the cloud's eight-file, 533-line foundation through its supported
publication control. Draft PR #28 exposes the same foundation as `a3c4d92`
(the cloud-local commit was `fb2c4a9`); the published branch is
`codex/implement-authorized-heic-restoration-in-codex`. Continue this branch.
Remote `main` still equals `0335a44`. The local host is macOS 27.0 build 26A428,
arm64, Go 1.27.1. Docker is installed but has no running daemon.

The source-access blocker is resolved: the standard Go module registry returned
v0.2.3 at `b2d46ba787d8f0a2025bd106443ab1b1c7cd010f`, checksum
`h1:+fEP2Xf1CoZ21SxA2YpqnPZb6Y/hAEkcgjU1gMsOhrk=`. v0.2.2 is available for
static comparison. No historical reproducer or upstream test corpus is run.

Local work sequence, before adding a production runtime:

1. Record source/license/API differences and actual OS memory semantics in
   `docs/heic/qualification.md`. A small disposable native macOS probe tests
   whether the proposed address-space budget can even be installed; it never
   allocates up to that budget. Public API/source evidence does not count as
   runtime qualification. Lead owns conclusions.
2. Complete the independent wire contract: straight-alpha NRGBA8/NRGBA64,
   explicit operations and failure statuses, bounded structured metadata,
   request framing, exact output/EOF validation. Share the existing codec-free
   `internal/heicdecode` package with the separate guest module. Preserve 16-bit output
   without increasing the 256,000,000-byte output ceiling (32M pixels at 16 bits).
   Verify with `go test -race -tags no_emoji,nodynamic ./internal/heicdecode/...`.
3. If source inspection permits a development guest, add a separate WASI-only
   module at `scripts/heicguest`, reproducible build/provenance/import guards,
   and tests using only ordinary licensed fixtures. No shipped/production helper
   runs without qualified OS controls. The guest accepts byte streams and emits
   the shared wire contract. Source digests and notices accompany the artifact.
4. Integrate production decoding only after T2 can be enforced and measured.
   A missing whole-worker memory mechanism blocks that dependent work; no soft
   substitute, per-process semaphore, or compilation-only platform claim.

One read-only OS-documentation scout was used alongside cloud recovery. G1:
bounded primary-source question; G2: cited public APIs/kernel implementation;
G3: zero file writes; G4/G5: independent OS source sweep before lead review.
Rule S cannot resolve accounting semantics from a textual substitution; Rule W
holds. No implementation/review/fix delegation. This revises T0's spawn budget
from zero to one. All reviews/fixes remain with the lead.

The literal imaging commands above require `-tags no_emoji,nodynamic` under the
current repository policy. Native platform execution and GoLand availability
will be recorded separately from build-only checks.

## Cloud cost ledger (historical)

| Task | Spawns budget/actual | Review rounds | Full suite | Notes |
| --- | --- | --- | --- | --- |
| T0 | 0 / 0 | 0 | no | Source retrieval blocked; independent plan/recon completed. |
| T1 | 0 / 0 | 1 | no | Limits and response validation complete; client/helper/guest pending T0 immutable source qualification. |
| T2 | 0 / 0 | 0 | no | Pending fixed runtime contract. |
| T3 | 0 / 0 | 0 | no | Pending qualified platform admission. |
| T4 | 0 / 0 | 0 | no | Native platform evidence unavailable in Cloud. |
| T5 | 0 / 0 | 0 | no | Pending. |

## Local evidence ledger

| Check | Actual result |
| --- | --- |
| Source recovery/publication | Draft PR #28 exposes the cloud foundation at `a3c4d92`; continued locally on its published branch. Upstream module and v0.2.2 comparison retrieved through the standard registry, with exact source/archive/module pins recorded in `docs/heic/qualification.md`. |
| Test-led protocol corrections | New tests first rejected premultiplied RGBA interpretation and absent 16-bit layout, then passed after NRGBA8/NRGBA64 support. Request/operation/metadata tests first failed for missing contracts. Additional limits tests rejected previously accepted fractional WASM pages and guest memory above the outer budget. |
| Bounded native feasibility probe | macOS 27.0 arm64: `setrlimit(RLIMIT_AS, 2 GiB)` returned EINVAL below existing virtual mappings. Tiny disposable C process, no large allocation. This does not rule out other macOS strategies; separate research is ongoing. |
| Guest/source guards | `make heic-build heic-check-provenance heic-check-imports` passed. Independent byte-identical rebuild; known module/archive/source/notice checks and native-graph checks. Actual artifact-byte mutation made the provenance command fail; removing the fixture generator's WASI-only build guard made the import command fail. Both exact files restored. Unit guards also refuse altered source content, missing artifact and native codec import. |
| Positive guest ABI | Four cases / three distinct licensed images; upstream basic/main10 are byte-identical. RGB, alpha and PicFetch-owned explicit ten-bit gradient pass Decode, DecodeConfig and DecodeExif. NRGBA64 and alpha preserved. No camera compatibility, colorimetric/orientation precedence or native sandbox conclusion follows from these checks. |
| Focused race regression | `go test -race -tags no_emoji,nodynamic -count=1 ./internal/heicdecode ./scripts/heicbuild ./internal/imaging ./internal/favthumbs ./internal/filescan ./scripts/plistdoctypes ./scripts/msixstage` passed. Imaging 28.130s, favorites 1.682s, scanner 1.505s, plist 1.283s, MSIX staging 5.460s. After final protocol/guard refinements, boundary and guest repeated and passed in 1.243s / 58.194s. |
| Advisory scan | `make security-govulncheck` passed for the native graph and separate WASI module. No reachable or imported-package vulnerability found. Native verbose scan identified existing module-only GO-2026-5932 in unimported `golang.org/x/crypto/openpgp`; this change neither imports it nor changes x/crypto. Guest scan found no vulnerabilities. This is known-advisory coverage, not decoder security certification. |
| Local build | `make verify-build` completed format/TUF/assets/notices/provenance/import/vet/build. The appended direct shard command selected macOS tests and failed on the pre-existing `TestMergeWindowMenus_FoldsEveryDuplicate`; the Make target explicitly requires a prepared Linux/amd64 runner. Do not change the Linux manifest or bypass the guard for a macOS run. Canonical shard validation remains a CI gate. |
| Full suite | `make verify` could not start: Docker daemon socket absent. Native Linux/amd64 CI is required. No seccomp, shard or isolation guard was relaxed. |
| GoLand | Connected IDE reports only `/Users/ronin/Projects/picfetch`; this isolated worktree is not an open GoLand project. Inspections of changed files remain unverified. Qodana is a separate CI check, not a replacement claim. |
| Production/platform status | No helper/client/broker/OS sandbox or memory control installed. Linux, Windows/MSIX and macOS production HEIC admission all remain disabled. No main/imaging/UI integration, format declarations or packaging exposure changed. |

### Local cost/status ledger

One bounded read-only scout, all implementation/review/fixes by the lead.
T0 source/build provenance is implemented; translated-source lineage and OS
qualification are incomplete. T1 protocol and development guest are implemented;
production client/helper lifecycle is pending the resource decision. T2-T4
remain blocked on qualified enforcement and then shared admission, compatibility
and packaging. T5 publishes a reviewable candidate only, without merge/release.
No native capability/crash/timeout/memory-exhaustion campaign or upstream fuzz
corpus was executed. CI outcomes/publication are recorded below when observed.

## Accepted macOS memory tradeoff (supersedes the original T0/T2 gate)

After a separate explicit decision, Ronin approved enabling macOS HEIC once
sandboxing and decoder memory limits are verified, accepting that the native
helper has no guaranteed hard total-memory cap and can still cause system
memory pressure or crashes. Unauthorized file/network actions remain
unacceptable. The former 2 GiB value was an engineering proposal, not his numeric
requirement. This approval does not permit weaker sandboxing, unvalidated IPC,
native codec loading, unbounded decoder/input/output/jobs/time, or false claims
about undiscovered vulnerabilities.

Continue beyond this candidate into a minimal disposable protected helper,
family-wide app/analysis admission, deterministic cancellation/join and canonical
imaging integration. macOS must fail closed if its actual sandbox cannot be
established. Windows/Linux use strong practical restrictions and OS memory
controls where available, with truthful per-platform reporting. Native/package
qualification remains required before exposure. All commits/pushes remain
permitted; no merge/release. The dedicated macOS research task supplies further
primary-source implementation guidance. No alternative runtime migration is
selected by this approval.
