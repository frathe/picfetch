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
| `github.com/gen2brain/h265/heic` | Candidate module `github.com/gen2brain/h265` v0.2.3; immutable commit, `h1:` checksum and changes since v0.2.2 still must be obtained from the trusted source. | Upstream licence and complete generated/translated-source provenance must be inspected; ship its notice. | **Blocked:** outbound module/source access is HTTP 403 in this environment; do not invent a pin or digest. Prior repository audit records an invalid result invariant and sequence fallback at v0.2.3. |
| WASI guest | Reproducible build recipe will name compiler version, target, flags and source tree digest; checked-in artifact gets SHA-256. | Same closure as selected decoder plus compiler/runtime notices where distributed. | Pending source qualification. |
| wazero | Existing selected dependency is transitive through AVIF; exact selected version and licence will be recorded from `go list -m all` when the guest wrapper is added. | Apache-2.0 notice and NOTICE obligations. | Existing dependency; HEIC usage not yet added. |
| PicFetch host/helper/protocol | This repository and committed revision. | MIT. | Planned. |
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

## Cost ledger

| Task | Spawns budget/actual | Review rounds | Full suite | Notes |
| --- | --- | --- | --- | --- |
| T0 | 0 / 0 | 0 | no | Source retrieval blocked; independent plan/recon completed. |
| T1 | 0 / 0 | 1 | no | Limits and response validation complete; client/helper/guest pending T0 immutable source qualification. |
| T2 | 0 / 0 | 0 | no | Pending fixed runtime contract. |
| T3 | 0 / 0 | 0 | no | Pending qualified platform admission. |
| T4 | 0 / 0 | 0 | no | Native platform evidence unavailable in Cloud. |
| T5 | 0 / 0 | 0 | no | Pending. |
