# Isolated HEIC/HEIF restoration

**Current status:** production HEIC remains disabled. This chronological plan's
original 30-second/all-platform hard-memory requirements were superseded by
the recorded 60-second ceiling and accepted macOS native-memory limitation.
Use [qualification](../docs/heic/qualification.md) and
[history reconciliation](../docs/heic/history-reconciliation.md) for the current
implementation and the disposition of the original branch's changes.

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

## Production continuation task graph

P1 runtime/helper -> P2 native launch and sandbox -> P3 family admission ->
P4 canonical source integration -> P5 packaging/platform verification. The lead
owns all implementation, review and fixes. Each boundary lands with focused
tests before consumers are changed; platform refusal remains explicit.

| Task | Files / contract | Required evidence |
| --- | --- | --- |
| P1 | `internal/heicdecode/worker`: fixed embedded WASI artifact, bounded streaming stdin/stdout/stderr, linear memory ceiling and context termination. `cmd/picfetch-heic-worker`: minimal entry point, no Fyne/native codec. Move the artifact into the worker package; maintain provenance. | Owned runtime boundary controls and ordinary fixtures with `go test -tags no_emoji,nodynamic ./internal/heicdecode/worker ./scripts/heicbuild`; import and reproducibility guards. |
| P2 | Native platform launch establishes restrictions before image input; parent verifies helper identity/readiness and owns timeout, pipes and process join. Capability evidence distinguishes actual OS controls from soft native-memory targets. | Native owned file/socket controls plus ordinary decode; absent sandbox and invalid identity fail closed; cancellation joins helper and all pipe work. Cross-builds are recorded separately from native execution. |
| P3 | One app-family admission owner serves GUI and analysis consumers. Acquire before bulk source reads; bounded waiting and cancelled queues; release only after helper/output cleanup. | Concurrent GUI/analysis fixture consumers never exceed one admitted HEIC request; queue cancellation and shutdown join are observable. |
| P4 | Canonical imaging source carries either ordinary encoded bytes or already validated HEIC pixels/metadata, so probe/decode do not retain a second queued HEIC buffer or decode twice. | Display, thumbnail, metadata and similarity focused regressions; ordinary formats unchanged. |
| P5 | Build/package helper identity and notices; truthful per-platform admission and format declarations; native Linux CI and available macOS tests, GoLand inspections. | Package checks, import/provenance guards, native platform tests and full required CI. Missing native evidence is unverified. |

One additional bounded read-only scout maps source retention and subprocess/UI
lifetime owners across consumers while the lead implements P1. This independent
breadth question needs call-path context beyond a grep; it performs no edits or
review and chooses no architecture. P1-P5 budget: one scout, zero implementation
spawns, one focused review per boundary plus the final gate. The prior whole
worker cap gate above is historical and superseded only for the explicitly
approved macOS limitation.

Publication: candidate `b64cca35430917cec6e59bd1c26ee4e6c86310b0` is pushed
and GitHub reports its signature valid. The cloud foundation `a3c4d92` is
unsigned. No history rewrite is authorized or performed; the full PR range
requires signing cleanup before merge.

### P1/P2 continuation evidence (2026-09-16)

- Fixed artifact moved to the helper-only worker package; native import guard
  rejects accidentally linking it into the viewer. The parent owns hash pinning,
  bounded waiting/streams, pre-read admission, readiness, process-group teardown
  before leader reap, and joined pipe workers. Shared GUI/analysis admission is
  still P3; the new Client alone does not satisfy that criterion.
- Owned WASM initial/growth/deadline/diagnostic controls and ordinary decode pass.
  Owned parent peers pass changed identity, blocked writer timeout, crash,
  diagnostic overflow, active/queued cancellation and Stop/Wait under race.
- Real macOS 27.0 arm64 helper bundle verifies App Sandbox and denied owned
  file-read/create/TCP/UDP before input. Ordinary ten-bit decode and cancellation
  pass with strict bundle signature verification. Native other-platform startup
  still fails closed. Direct no-cgo macOS also fails closed.
- Hardened Runtime without an executable-memory exception refused compiler
  decoding. Pinned wazero uses RW -> RX without MAP_JIT; Apple's broader unsigned
  executable memory entitlement is needed by this implementation. It is confined
  to the helper; no library-validation, DYLD, network or user-file exception.
- Interpreter comparison with unchanged finite bounds: basic 320x240 1.5206 s;
  owned valid 4032x3024 gradient reached 30.0093 s and was terminated/joined.
  Compiler: 1.6174 s and 7.4040 s respectively. Keep compiler on this evidence,
  document increased trusted native executable-memory authority and own-container
  rights. These two fixtures are not broad camera qualification.
- Candidate b64cca3 CI failed only on the race-instrumented development
  interpreter's 15-second basic/main10 deadline. Fixture ABI tests now use the
  production compiler, one test-owned compiled module and the actual 30-second
  contract. No production resource bound was increased. Other CI jobs passed.
- The independent macOS research has completed; public App Sandbox supports the
  selected bundle route. No new memory-risk approval is needed. User-owned
  historical THREAT-MODEL.md in the separate saved checkout is not imported or
  edited by this feature work.

P2's platform continuation uses one further read-only scout to map the pinned
Windows API surface and official launch/job/AppContainer semantics while the
lead completes macOS validation. It writes no code and performs no review or
security testing. This bounded independent breadth survey replaces repeated
cold exploration; all design and fixes stay with the lead. Cumulative scout
count is three including the original source/OS phase; no implementation spawns.

### Worker publication gate and updated source requirement

- Focused race tests pass: protocol 1.205 s, client 1.718 s, worker 15.419 s,
  guest/build guards 16.506 s, nativeguard tooling 1.236 s.
- `make verify-build` passes formatting, generated assets/notices, exact Qodana
  exclusions, source/artifact reproducibility, native import guards, vet and
  build. The first restricted run could not write Go's module stat cache; the
  same target passed with authorized standard cache access. Full native
  Linux/amd64 race and platform CI remain required; no local Docker daemon.
- `make heic-native-macos` passes both required native guards, 46.377 s total.
  Missing entitlement refuses before source read; real ten-bit decode,
  read/create/TCP/UDP denial, cancellation and owned descendant cleanup pass.
  Canonical compressed gradient: compiler 7.1366 s; interpreter terminated and
  joined at 30.0108 s under the unchanged 30-second deadline.
- Owned preopen control passes without filesystem access and traps when given
  a temporary directory. Temporarily changing group termination to leader-only
  makes the descendant test fail at its bounded EOF assertion; exact source
  restored and the focused/native suites pass afterward.
- GoLand worktree access is resolved. All 34 changed Go files were inspected,
  including weak warnings. Fixed fixture cleanup, ignored errors, nullable test
  metadata and redundant types/conversions. Guest-only protocol enum values use
  a verified `GoUnusedConst` suppression on that declaration; literal owned
  WASM controls use verified function-scoped duplication suppressions and the
  existing exact test-file Qodana exclusion. Final re-inspections are clear.
- Qodana run 35024568046's post-suppression SARIF was retrieved and its b64cca3
  revision verified. All seven results were inspected: five corrected issues
  and the two separately compiled guest-status false positives above. This is
  disposition of the prior commit, not a clean report for the pending commit.
- One read-only evidence-retrieval scout downloaded that report while the lead
  inspected/fixed code. Another bounded source inventory now maps historical
  local patches; it performs no review, edits or upstream test/fuzz execution.
  These two additional independent scouts exceed the prior count explicitly;
  all architecture, equivalence assessment and implementation remain with T0.

The coordinator relayed newer user source instructions: retain the maintained
`third_party/h265` hardening until equivalent upstream checks are verified;
never add native/in-process fallback. Read the historical `PICFETCH.md` in the
separate saved checkout. It records additional coded-work, extent/table, NAL,
frame-retention and transformed-config changes beyond this candidate's source
selection. P0 now reconciles that source and records each patch disposition
before P4/P5 activation; the independently verified P1/P2 worker boundary can
be published while HEIC remains disabled. Do not copy or edit the user's
historical `THREAT-MODEL.md`. No source-equivalence conclusion follows solely
from the newer upstream version or the updated AGENTS wording.

### P0 source preservation and native deadline diagnosis

P0 files: `third_party/h265`, root/guest module replacements,
`scripts/heicbuild/source.go`, provenance, ordinary guest checks and the source
qualification record. Owner T0; no implementation delegation. Contract: exact
maintained production bytes from fc127b44, WASI-only application use, baseline
and replacement validation before builds. Verify with `make check-heic-wasm
test-h265`, focused worker/guest race tests and native guards. One source
inventory scout and the earlier evidence-retrieval scout were reused; source
assessment, patches and findings stayed with the lead.

- Read historical `third_party/h265/PICFETCH.md`; source subtree was clean at
  saved HEAD 73cb3c9, last source change fc127b44. Restored 107 production/source
  notice files exactly (57 Go, 47 assembly, LICENSE, README and go.mod). No
  historical test corpus or THREAT-MODEL.md was imported or edited.
- Preserve all local hardening instead of substituting unmodified v0.2.3.
  Reviewed v0.2.2 -> v0.2.3 production diff: missing metadata, coded-frame bounds
  and transform bounds already exist locally; decoder/deblock changes are
  comments. Keep the complete local bounded sequence implementation without
  claiming equivalence to upstream's different representation. Patch inventory
  and exact source manifest are in the maintained directory.
- The new input-inventory/native-import test first failed because maintained
  source was absent from provenance. It passes after source inventory and
  module isolation are recognized. Missing or redirected replacements are
  refused. An actual appended comment in the maintained budget source made
  `heicbuild build` fail on its baseline hash before compilation; exact bytes
  restored. `make test-h265` then passed source/rebuild/import checks and the
  ordinary WASI fixture test (1.563 s).
- The maintained worker and guest pass focused race tests (14.730 / 15.802 s).
  All imported Go/assembly files were inspected. Assembly dialect errors and
  retained declarations/algorithms have exact Qodana exclusions; existing weak
  style suggestions and partial switch/EOF behavior were assessed and recorded
  in PICFETCH.md. The IDE does not consume Qodana scopes: those reports are
  reviewed dispositions, not a claim of zero generic-assembly diagnostics.
- Published c4a89d1 Qodana had one result: worker Main appeared unused although
  cmd/picfetch-heic-worker calls it directly. Added a function-scoped suppression
  with that caller documented; GoLand re-inspection is clear. Fresh source-phase
  Qodana evidence remains pending.
- c4a89d1 CI passed six jobs; Linux race and Intel native guards failed on the
  30-second timing proposal. Linux's real guest compilation under race took
  37.08 s; every small owned runtime boundary control passed. Intel sandbox,
  small ordinary decode and cancellation passed, while the 12MP compiler job
  reached 30.060 s and its interpreter reached 30.049 s. No guard was skipped.
- Diagnosis retained the original deadline. The maintained x86 helper under
  Rosetta decoded basic at 2.290 s and 12MP at 5.967 s. Its first diagnostic
  launch hit the installed xcrun's missing x86 slice; using ARM clang to build
  the x86 helper resolved setup. This exercises x86 transport/compiler behavior
  but is not native Intel performance qualification. A non-race real worker
  fixture test completed in 0.89 s; instrumentation/host throughput explain the
  observed timing evidence better than a general transport stall.
- The coordinator clarified that 30 seconds was an engineering proposal.
  Proposed a finite 60-second ceiling to measure common-camera completion on
  native Intel, with no byte/memory/job/sandbox relaxation and no test removals.
  The updated default/max-duration tests went red at the prior 30-second limit,
  then green; values above 60 seconds are refused. Fresh native Intel completion
  time is still required before qualification. Longer single-lane occupation
  is the availability cost, recorded in the qualification document.
- Final local P0 checks: `make verify-build test-h265` passes. Focused race
  protocol/client/worker/guest/nativeguard packages pass in 1.310 / 1.911 /
  14.998 / 15.903 / 1.225 s. Native Apple Silicon suite passes in 75.328 s;
  compiler 12MP 6.713 s, interpreter finite rejection at 60.012 s. All native
  file/network, missing-entitlement, cancellation and descendant guards remain.
  The new source tooling and changed boundary files re-inspect clean in GoLand.

### P2 Linux candidate (native execution pending)

Files: worker Linux policy/installation and owned native controls, client native
fixture test, nativeguard runner, CI matrix, exact test exclusions and platform
qualification record. Lead owns design/implementation/review. Verify with
`make heic-native-linux` on native Linux amd64 and arm64, the portable filter
test on every host, cross-builds and GoLand. No Linux production exposure follows
from cross-build success alone.

- Reused one read-only scout for installed Go 1.27.1/wazero 1.12.0 syscall
  facts while the lead implemented policy. A second read-only Windows scout
  mapped existing package/CI launch points. Neither wrote policy, reviewed code,
  ran security tests or made edits; all implementation remained with the lead.
- Exact pure-Go thread flags differ: amd64 0xd0f00 includes CLONE_SETTLS,
  arm64 0x50f00 does not. Reject all other clone forms, clone3, exec, file/network
  operations, io_uring, new policies/namespaces and unspecified syscalls.
  Runtime memory/signal/poll/stdio operations are the explicit allowlist.
- Portable BPF tests first failed on the refusing placeholder, then passed
  required runtime calls and denied authority for both ABIs. Native-suite
  registration/CI tests first failed for the absent Linux suite and invocation,
  then passed after wiring. Native guards cannot be executed on this macOS host;
  no local Docker daemon is available. Their execution is still unverified.
- RLIMIT_AS is a virtual-address-space ceiling, not a measured RSS quota.
  Read bounded /proc/self/statm before installation and reject an oversized
  starting process. Read back every hard/soft limit, then require ENOMEM from
  an oversized PROT_NONE reservation before input; no RAM pressure test.
  CPU/core/file-size/descriptor limits are finite; actual process creation is
  denied, while debug.SetMaxThreads remains only a Go runtime thread ceiling.
- Native controls retain one thread across TSYNC and create enough locked
  threads to exercise inheritance. They also observe syscall denial, a small
  owned WASI cancellation and the omitted-address-limit negative control.
  The real helper must decode ordinary ten-bit and 12MP fixtures after its
  parent-owned file/TCP/UDP readiness controls under the unchanged 60 seconds.
- Signed P0 publication is temporarily waiting on Secretive's signature request.
  Its original staged checkpoint remains separate from these working edits.

### P3 family admission and pipe transport

Lead-only implementation in `internal/heicdecode/client`. One application-owned
Client admits foreground and background operations; FIFO within class and no
more than three foreground grants before a waiting background grant. A bounded
pipe service shares that Client with analysis processes. The remote sends a
small intent first and reads encoded bytes only after the owner's native-ready
grant. Framed canonical responses use existing validation, without a second
image encoding or an image-sized envelope copy. Retain admission through
downstream response delivery, bound idle connections and queue entries, close
both stream directions on cancellation, and join their service work at Stop.

Tests: controlled local/remote source callbacks cannot overlap, remote bulk
reads follow the grant, blocked output retains admission until cancellation or
delivery, cancelled queues never read sources, repeated responses remain
framed, and Stop joins active/idle service work. Verify focused client race
tests, GoLand and native Mac regression after this boundary. Pipe inheritance
and app/similarity wiring are subsequent P3/P4 work, not satisfied by these
component tests alone. Existing platform refusal remains unchanged.

- The fairness test went red on the refusing queue placeholder, then passed
  FIFO selection, the three-foreground maximum, cancellation and idempotent
  release. Existing client lifecycle regressions pass after lane replacement.
- Repeated remote requests went red on the refusing transport placeholder,
  then passed canonical NRGBA64 transfer. A queued-disconnect regression
  exposed a service waiting behind active foreground work; one bounded control
  byte now observes disconnect, cancels admission and is joined on every exit.
  Pre-admission refusal retires the connection without reading source bytes.
- The focused client race suite passes (5.200 s), including bounds for frame
  lengths/grants, eight idle connections, Stop/Wait and held output. An actual
  early-release mutation made the held-output test fail; exact source restored.
  GoLand inspections are clear after moving deliberate retained test peers to
  one explicit cleanup. Final post-fix native/local checks are recorded next.

- Final component client race run: 5.275 s. `make verify-build` passes. Real
  macOS App Sandbox helper/cancellation regression passes in 3.343 s; the
  runtime-choice measurements remain the earlier unchanged-runtime evidence.
- Pipe inheritance now proceeds through a Client-owned attachment and a small
  descriptor/limits configuration. UNIX appends explicit ExtraFiles and makes
  inherited child pipes pollable; Windows uses explicit inherited handles.
  Attachments must register service work before asynchronous dispatch, close
  parent copies of child endpoints immediately after Start, and stop/join on
  every failure/exit. A reused read-only Windows scout checks pinned Go pipe
  Close/handle-list semantics while the lead implements the UNIX side.

- Actual inherited subprocess decoding, child cancellation, and cleanup before
  process Start pass under the race detector. The full client race suite passes
  in 8.841 s; nativeguard tests and `make verify-build` pass. Latest changed Go
  files have no GoLand findings. Windows amd64/arm64 and Linux arm64 test binaries
  cross-build; the new Windows native pipe CI suite is still unexecuted.

### P4 source contract and first integration gate

Owner: lead for design, tests, implementation and review. Reused one read-only
caller scout to locate constructor and shutdown injection points across GUI
features while the lead inspected imaging/analysis. No edits or review delegated.

- `imaging.Reader` is an immutable value containing an optional HEIC decode
  function. Its zero value keeps HEIC unavailable. `Read` returns a `Source`
  with bounds, complete decode, metadata, encoded digest, thumbnail and preview
  methods. Ordinary sources retain encoded bytes and the existing algorithms;
  HEIC sources retain validated pixels/normalized facts and discard source bytes.
- Dispatch reads at most 64 KiB before admission, from one open source stream.
  HEIC extensions route directly; renamed HEIC and ambiguous BMFF file types
  route to the isolated decoder. Only positively identified AVIF/CR3 brands
  enter existing native metadata/preview handling. No full container walk occurs
  in the parent. Bulk HEIC reads and digest calculation occur inside admission.
- The guest's transformed pixels are authoritative. The source adapter does
  not apply Exif orientation again; ordinary image orientation stays unchanged.
  Native camera/container/Exif precedence still needs qualification before
  production enablement. Existing normalized GPS absence ambiguity is preserved.
- Metadata-only reads use the guest metadata operation. Cancellation closes
  the open stream and joins that close callback. Opening or a storage backend
  that cannot interrupt a blocked read retains the existing I/O limitation.
- First gate: owned decoder callbacks prove no bulk read before admission,
  renamed dispatch, exact digest/size, one decode, 16-bit pixels, no repeated
  orientation, source cancellation and metadata-only routing. Ordinary JPEG,
  AVIF, CR3, GIF/SVG, thumbnails and capture-date regressions must pass.
  Verify `go test -race -tags no_emoji,nodynamic ./internal/imaging`, then the
  affected consumer packages, `make verify-build`, and changed-file GoLand.
  App/similarity injection and shutdown tests are a separate subsequent gate.

- First source gate passed: image-source contracts went red against the refusing
  placeholder, then green. Native byte-metadata refusal first failed on the
  existing ordinary HEIC fixture, then passed after dispatch guards. An actual
  early full-read mutation failed the renamed-file admission test; exact source
  restored. Full imaging race tests pass in 23.429 s; expanded focused source
  tests pass in 1.386 s. GoLand is clear on new source/dispatch/tests and changed
  preview, metadata, RAW and raster files. The loader import retains the already
  recorded IDE module-tag error (no_emoji/nodynamic not applied to its live
  module); command-line tagged builds validate that import. No suppression was
  added for the missing IDE configuration. The first build gate caught import
  grouping in two new files; corrected with repository goimports settings.

P4 analysis connection gate: `similarity.Client` accepts the app-owned HEIC
client; Analyze and retained Search attach explicit pipe pairs before process
start and close/join them on every exit. Only descriptor/limit values enter the
request. Worker request setup owns one Remote and injects `imaging.Reader` into
analysis and search preparation; disposal runs before WorkerMain can exit.
Source hashes/facts use Source methods, with no raw HEIC metadata path. Owned
analysis/search subprocesses must observe a typed owner refusal without reading
source bytes, including start failure and cancellation. Pixel transport itself
is covered by P3's real inherited subprocess tests. Verify focused similarity
protocol/source tests, affected regressions, Windows cross-build and GoLand.

- Analysis/search owned-process connection tests went red for missing setup and
  missing inherited handles, then green. Search preview injection went red on
  the old byte path, then green after Source migration. Full similarity race
  suite passes in 13.543 s. Expanded cancellation/start-failure/preview tests pass
  in 3.754 s. Windows ARM64 test binary cross-builds; native execution remains
  separate. All eight changed analysis code/test files re-inspect clear in
  GoLand. No inference model or real HEIC camera qualification follows from
  these owned transport and source tests.

P4 GUI injection gate: immutable foreground/background Readers enter display
Config and feature constructors before workers start. Viewer owns both readers
and the one optional HEIC Client; Explorer and visual search receive that same
Client. Stop closes HEIC admission before feature teardown and Wait joins it off
UI in production and the harness. Comparison, EXIF/file reconciliation, grid
hash/thumb work, Favorite previews, sorting, Spiral and mosaic use those Readers.
Zero-valued readers keep production HEIC disabled until package qualification.
Verify feature-level HEIC callback contracts, root wiring/shutdown assertions,
focused package regressions, exact UI shard assignments, and GoLand.

P4 GUI evidence (2026-09-16):

- Root source-consumer and owner-shutdown race tests pass (2.890 s); display
  suite 2.696 s, grid 10.780 s, EXIF 5.517 s, Spiral 2.523 s, filesort 1.397 s,
  Favorite previews 1.679 s and mosaic 51.267 s. Expanded Spiral HEIC callback
  test passes (1.882 s); source/metadata/date regressions pass (1.903 s).
- Focused viewer capture-sort, EXIF navigation, Favorite, comparison, save/export
  and HEIC regressions pass (45.573 s). The first run exposed a pre-existing
  synctest ownership mismatch: a viewer created outside the bubble acquired a
  context Done channel inside it after source cancellation became observable.
  Its cleanup cancelled that channel outside the bubble. Moving that test's
  viewer/source setup inside its existing bubble makes creation and cleanup
  share the lifetime; no production cancellation was weakened. The minimized
  test failed before the change and passes (2.624 s) afterward.
- All 29 changed GUI files were inspected, confirmed issues corrected, and
  touched files re-inspected clear. The source loader retains the known IDE
  build-tag configuration diagnostic for avifpolicy; tagged command-line
  verification passes. Live IDE tag refresh is still recorded as incomplete.
- Both new top-level UI tests are assigned to ui-1 and its count is 238.
  Qodana exact-file exclusions are synchronized. Format/provenance/import/vet/
  build gates pass. Canonical Linux shard validation and the complete race
  suite remain unverified locally because the Docker daemon is unavailable.
  Direct host shard checking selects an existing Darwin-only test absent from
  the Linux manifest; that does not justify changing the canonical manifest.

P2 Windows launch gate: the lead owns implementation of suspended AppContainer
creation with zero granted capabilities, an explicit three-handle stdio list,
one-process Job Object, finite committed-memory/CPU limits and kill-on-close.
The child verifies its token and active job before readiness; failures retire
all handles and the suspended child. No native image parser is added. Native
owned controls and ordinary WASI fixtures must pass on both Windows architectures
before activation. Cross-builds alone are not native evidence. The read-only
Windows scout was reused for pinned standard-library/x/sys API inventory while
the lead verified P4 (G1 bounded API question; G2 source declarations/official
links; G3 no edits; G4 narrow platform facts; G5 no lead implementation overlap).

P2 Windows candidate evidence (2026-09-16):

- Added `internal/heicdecode/winisolation`: suspended zero-capability AppContainer
  launch, explicit stdio handle list, child-process restriction and private job
  with process-count/commit-memory/user-CPU/kill-on-close limits. The worker
  checks token identity/capabilities and immediate job before the common owned
  file/network denial probes. Provisioning grants read/execute only to a
  dedicated helper and its directory, without ACL inheritance; launch itself
  does not change permissions.
- The owned native control has a 64 MiB budget, checks a fixed 65 MiB commitment
  refusal without touching pages, rejects a second inert copy and checks that
  a parent environment sentinel is absent. An unsandboxed process must fail
  verification. These Windows tests first failed to compile for the missing
  boundary; native red/green execution remains pending, not inferred from that
  compile failure. AMD64/ARM64 test binaries and the AMD64 helper cross-build.
- Native suite registration tests failed for missing Windows guards/CI wiring,
  then pass (0.267 s). `heic-windows` replaces the pipe-only suite, retaining
  its analysis/pipe controls and adding token/job controls and the same ordinary
  ten-bit/12MP helper fixtures as Linux. The shared fixture test is now
  `client/native_platform_test.go`; native Linux inventory is updated with it.
- A portable denial-classification control failed for standard permission
  errors, then passes (0.263 s). Windows file access-denied and Winsock WSAEACCES
  are recognized; connection failures/timeouts are not isolation evidence.
- Client process-lifetime regressions pass under race (8.935 s). Windows HEIC
  packages pass vet; Linux AMD64 and Windows ARM64 native client inventories
  cross-build. All 19 affected existing/new code files inspected clear in
  GoLand. `make verify-build` passes including format, source/artifact proof,
  imports, notices, vet and build; exact Qodana exclusions are synchronized.
- Packaging discovery reused the read-only caller scout for twelve-file-bounded
  artifact-layout facts while the lead implemented Windows isolation. G1-G5:
  bounded inventory, file/line oracle, no edits, cold independent package paths,
  no lead design/review overlap. P5 retains all decisions and implementation
  with the lead. Additional scout call recorded rather than hidden in budget.

- Real signed Apple Silicon App Sandbox helper still passes after the process
  abstraction/permission changes (3.385 s).

P5 packaging contract: helper and exact notices live at fixed package-relative
paths, with a bounded manifest recording the post-signing executable SHA-256,
platform and embedded-guest SHA-256. This manifest belongs to the trusted
installed package (TUF archive or platform package/bundle signing); it is not
an independent signature or protection against replacement of the whole trusted
installation by its owner. No image, environment override or manifest field
chooses an executable path. Readiness and isolation checks still run for every
request. Package construction finalizes the manifest after helper signing, and
any later helper change must fail the executable pin. Production admission and
format exposure remain disabled until native/package qualification is complete.
The next acceptance seam is package loading with absent, malformed, wrong-target
and changed-helper controls, followed by actual staged-bundle qualification.

P5 packaging evidence in progress:

- Fixed package paths and bounded manifests now have race-tested absent,
  malformed, wrong-architecture and changed-helper controls (1.368 s).
  `scripts/heicpackage` builds the minimal helper, signs/verifies its macOS
  bundle, copies exact h265/Go/wazero notice files and writes the final hash
  manifest. Wrong binary format is refused before manifest creation; package
  tool controls pass (1.367 s). Linux AMD64 and Windows ARM64 artifacts stage
  successfully here; those are cross-builds, not native isolation results.
- Native macOS acceptance now uses this same staged package; ordinary decode,
  missing-sandbox refusal and cancellation pass (3.070 s). `make package-mac`
  builds the complete app with its nested helper/notices. Strict deep codesign
  verification passes for both outer app and nested helper. The automatic
  Fyne build-number increment (469 to 470) was restored after verification.
- Release ZIP/tar and MSIX assembly include the helper/notices. Windows signing
  signs both executables, finalizes the helper manifest afterward, and retains
  the helper directory when rebuilding the ZIP. Real Windows Authenticode/MSIX
  package execution remains unverified.
- Packaging inspection found a further required P5 path: the in-app updater
  currently installs only the main executable and macOS plist. It will need
  authenticated companion-file staging/install (with rollback) or a complete
  package transaction before HEIC activation. No updater installation behavior
  has been changed yet. Production remains disabled while this work and native
  qualification are incomplete.

GoLand configuration follow-up: using the live project Build Tags settings,
set the required `no_emoji nodynamic` editor tags. The changed imaging loader
now re-inspects with no findings; its previously reported avifpolicy import
warning is resolved without a suppression. This is project-local IDE state,
not a source change. The known nested guest module resolution issue is separate
from the native loader diagnostic.

P5 updater continuation contract: after authenticated archive extraction, retain
and persist exact companion-file digests. Validate every staged companion again
before installation. Linux/Windows install the dedicated `heic` directory with
rollback around the existing executable transaction. macOS installs a verified
complete app bundle so the outer signature and nested helper remain consistent.
Legacy stages without companion provenance retain their existing behavior;
this also means the first upgrade from an older updater can omit the helper and
must remain fail-closed until a complete package is installed. The shutdown path
must join the HEIC owner before replacing any helper files. Acceptance tests use
owned archives/files: companion tampering after Save/Load is rejected; install
preserves the package; preparation/apply failures restore prior installed bytes;
existing update/relaunch controls still pass. Native Windows/MSIX execution and
the old-updater transition stay explicit qualification limits.

P5 updater evidence (2026-09-16):

- The companion-provenance test failed because a changed notice was accepted
  after Save/Load, then passes after exact bounded inventories were added.
  Each authenticated extraction now owns a fresh payload directory so stale
  cached files cannot be incorporated into a newer archive's companion proof.
- The install test failed with a new main executable paired to the old helper,
  then passes after the helper-directory transaction/rollback. Full updater
  race suite passes (4.204 s), updater UI suite (1.627 s) and focused root
  shutdown/update regressions (4.446 s). Windows AMD64/ARM64 updater test
  binaries cross-build; native execution remains a gate.
- Native macOS whole-bundle install and an injected final-rename failure both
  preserve coherent signatures and all expected bytes, without modifying a
  neighboring owned user-data file. An initial fixture had unchanged helper
  resources and incorrectly survived the negative binary-only mutation; it was
  strengthened to change helper metadata/signature, its manifest and another
  sealed resource. The same real mutation now fails for package mismatch;
  restored implementation passes (1.639 s). This supersedes the weaker fixture
  evidence. Strict deep codesign verification covers the enclosing bundle and
  nested helper, not merely the standalone helper signature.
- The retained legacy updater path was tested separately with an owned initial
  bundle that has no helper. It omits the new helper, removes the staged payload
  and fails enclosing codesign verification (invalid resource directory). The
  expected-limitation control passes (1.748 s). This proves no cache-based
  automatic migration is available after the old updater's normal cleanup.
  A complete reinstall or separately qualified bridge is a first-release
  requirement; production HEIC remains disabled. The originating task received
  this concrete finding and the fact that work continues locally.

- Removing a helper-bearing stage's companion provenance initially passed
  validation. The new control fails before the fix and passes after rejecting
  helper packages without their verified digests. Legacy stages with no helper
  remain supported. Full updater race regressions pass again (4.091 s).
- Final updater/native-guard GoLand inspections are clear, and the three
  modified workflow documents parse as YAML. Explicit Windows companion
  provenance/install/rollback requirements were missing from the guard list;
  a failing registration control exposed this and the list is corrected.
- A final read-only fixture inventory reused the caller scout while the lead
  verified updater changes. G1-G5: repository-only ordinary-fixture coverage,
  named tests/file-line evidence, no writes or decoder execution, independent
  compatibility facts, and no delegated implementation or review. It confirms
  that the existing fixtures cover basic/alpha/explicit ten-bit transport and
  synthetic 12MP throughput, but no recorded real-camera provenance or
  HDR/color reference. Adapter metadata stubs are not camera extraction proof.
- Final consolidated `make heic-native-macos` passes all four mandatory guards:
  sandbox helper, interpreter/compiler comparison, signed bundle install and
  rollback, and the legacy first-upgrade limitation. Compiler 12MP decode is
  6.772 s; the interpreter is terminated/joined at 60.015 s. The two native
  update guards pass in 1.310 s and 1.220 s. Raw events are retained in
  `.scratch/heic-qualification/native-macos.json`. Native guard registration
  regressions pass under race in 1.344 s; changed guard files re-inspect clear.
- Final `make verify-build` passes again after the updater provenance and CI
  guard corrections. Publication remains blocked on the pending Secretive
  signing request; subsequent work stays unstaged to preserve that in-flight
  checkpoint. A refreshed PR description is prepared locally for publication.

### Ordinary compatibility continuation (2026-09-16)

The originating task requested bounded normal-camera and color qualification
before treating signing as the only remaining local dependency. The lead owns
all checks and conclusions. A read-only scout searched first-party records for
at most two ordinary licensed samples while the lead tested existing fixtures;
it downloaded no image bytes and executed no decoder. This is the existing
compatibility gate, with no new HDR feature, negative corpus or stress work.

- Real sandboxed helper checks pass for the existing synthetic TestCam metadata
  and container rotation, alpha, basic color pattern and owned ten-bit ramp.
  ImageIO/ExifTool provide independent metadata/orientation and eight-bit SDR
  comparisons. The known generated ramp is the numeric oracle for ten-bit
  values, because ImageIO expands/clips that fixture's range differently.
- Added the metadata/config/decode and known-ramp assertions to the existing
  native sandbox guard. The complete guard passes in 5.906 s, and GoLand is
  clear. Full details, reference methodology and observed pixel differences
  are in `docs/heic/compatibility-2026-09-16.md`.
- Static maintained-source/API evidence establishes an implementation limit:
  ICC/primaries/transfer handling and HDR gain-map/tone-map presentation are
  absent, and the guest drops the color description. Those color classes are
  not explicitly rejected. This supersedes the earlier vague description of
  HDR/color as merely unqualified; no implementation behavior was expanded.
- One photographer-released public-domain Samsung S23 Ultra sample with an
  immutable source manifest and paired SDR reference was located. Only that
  ordinary file is being retrieved with pinned size/hash and finite transfer
  limits; incomplete bytes will never be decoded. All native OS CI and signed
  publication requirements remain unchanged.
- The camera sample was fully retrieved and matched its pinned 2,872,198-byte
  size and SHA-256 before decoding. The real helper succeeds in 20.414 s,
  preserving upright 3000x4000 pixels and expected Samsung metadata. It carries
  a P3 profile; comparison with color-managed ImageIO yields RGB MAE
  7.529/8.283/7.644, so this is decode/metadata/orientation evidence, not faithful
  P3 display. The public paired PNG was identified but not downloaded or used.
- The bounded compatibility continuation is complete. Native tests, GoLand
  and `make verify-build` pass. Remaining camera breadth, color/HDR support,
  native CI, distribution and signing requirements are recorded explicitly;
  no production activation or new color implementation was added.

### Complete checkpoint publication (2026-09-16)

Ronin requested publication of the entire owned checkpoint to the existing
draft PR #28. Fresh inspection confirms the old signing process has exited;
the configured SSH key fingerprint matches a currently available Secretive
key. Signing stays enabled and the configured signer is used without a key
override. The prior P0-only staged subset is superseded by all 226 related
changed/new files; no unrelated local changes were identified.

The lead reviewed the intended source, platform, shared-admission, imaging,
packaging/updater, CI/test and documentation scope against the existing plan
and verification record. Source/artifact/import guards and exact Qodana test
exclusions are rechecked. The UI manifest has unique assignments for both new
root tests, sorted blocks and matching counts (238/248/202); complete Linux
build-selected inventory and race verification remain native CI gates.
Prior focused/native/GoLand/build evidence remains applicable; broad local
tests are not rerun merely for publication.

A read-only scout inventoried the two separately owned documentation commits
while the lead handled the checkpoint (bounded commit/path facts; no edits,
review or historical code import). They are intentionally absent here:
`cac90b9e4b2e7039c1489bc84e96327ec1e84f46` adds
`docs/heic/macos-memory-protection-research.md`; root-local
`73cb3c9c568728ecd4cd36df10dd6907f3d47a3c` adds `THREAT-MODEL.md` scoped to the
historical prototype. Neither branch nor the historical hardening commit is
merged. The missing documents are reported for separate intentional integration.
HEIC activation remains disabled, all qualification limitations remain explicit,
and no history rewrite, merge or release is authorized by this checkpoint.


### Historical omission reconciliation (2026-09-16)

Ronin authorized restoring useful omissions from `feature/heic-hardening`
commits `fc127b44` and `73cb3c9` into the existing PR, including signed commit
and push. The earlier checkpoint's deliberate document omission is superseded
by this authorization. The CI/review agent remains paused; no merge, release,
activation or history rewrite is included.

The lead restored capped one-time guest backing storage, YAML validation before
Qodana path checks, four unchanged ordinary positive fixtures, stronger alpha
coverage, the generated Fyne metadata ignore and an adapted app-wide threat
model. The 107 maintained production/source-license files were already present
byte for byte; the current helper architecture replaces the older worker.
[The reconciliation ledger](../docs/heic/history-reconciliation.md) accounts
for every historical non-vendored path and the selected/omitted test coverage.

The lead owned implementation, all review and fixes. One bounded read-only
scout checked ordinary fixture provenance and final source/fixture identity
while the lead reviewed code and ran checks (no writes, decoder execution or
review delegation). It confirmed 107/107 source/license files and 4/4 restored
fixtures with no mismatches. This adds evidence to the existing Deep plan.

The Make gate first accepted malformed YAML; the restored parser rejects it.
Both memory-growth cases first replaced the backing buffer; preallocation makes
them pass without raising the ceiling. Focused worker/build-tool/validator race
regressions and interpreter memory/cancellation tests pass. All four native
Apple Silicon guards pass after preallocation. The guest regenerates unchanged;
`make verify-build` and GoLand inspections of all changed Go files pass.

Canonical shard validation could not complete locally: the Linux-only direct
target was accidentally run on macOS, and the documented Docker target then
found no running daemon. No UI test or assignment changed. Current baseline CI
has passing Linux/macOS native guards and UI shards, but failing Windows HEIC
and non-UI Linux race jobs. Fresh CI must qualify the new configuration. These
and distribution/color/upgrade/signing gates remain explicit in `todos.md`;
this reconciliation does not claim feature or review-loop completion.
