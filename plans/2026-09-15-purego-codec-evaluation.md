# Evaluate pure-Go HEIC and AVIF codecs

Ronin requested a careful security evaluation of gen2brain/h265 and
gen2brain/gav1d and strong sandboxing/isolation. This authorizes evaluation
and disposable local prototypes, not restoring HEIC or replacing production
AVIF before the findings are assessed. It supersedes the earlier preference
to exclude these projects from research.

Route: Deep evaluation, because parser security and isolation span operating
systems. Deliverable: a reproducible source/test audit and concrete isolation
assessment, with shipping blockers and unverified properties explicit.

## Acceptance criteria and evidence

1. Record exact source revisions, published fixes/issues, module/import
   inventory and license provenance. Verify source metadata and manifests
   against upstream GitHub and inspect the selected source trees.
2. Assess header/container, metadata, pixel/sequence, unsafe/assembly and
   encoder paths. Exercise suspicious paths with a standalone harness under
   Docker's hard memory/CPU/pid limits and no network; retain JSON results.
3. Run the upstream native and scalar suites and bounded fuzz smoke tests in
   the same isolated environment. Record skipped external/reference corpus
   checks rather than treating them as verified conformance.
4. Compile a minimal WASI guest and test limits/cancellation through wazero
   if the local toolchain permits. Distinguish guest limits from process
   limits and an audit Docker container from a production desktop sandbox.
5. Assess reusable PicFetch worker code and specific Linux/macOS/Windows
   isolation gaps. Cite source/platform documentation for each guarantee.
6. Preserve application dependencies and format support. Update the report,
   existing research disposition and todos; run `git diff --check` and
   validate local report links. No commits, pushes or upstream reports.

## Ownership and delegation

Lead owns all security assessment, findings, harnesses, architecture and final
review. Two read-only scouts gather (a) historical license/source provenance
and (b) existing worker/source/platform facts. These are bounded searches with
exact file/commit/URL evidence, no shared writes, and no delegated security
review or design. The lead verifies decisive reported sources. This breadth
is independent of the lead's codec-path analysis and cannot be replaced by a
single local text search. Budget: two scouts, targeted isolated runs; no
unrelated application test suite for an evaluation-only change.

## Source pins

- h265 v0.2.3: `b2d46ba787d8f0a2025bd106443ab1b1c7cd010f` (September 15).
- gav1d v0.2.5: `7aa50e4e898ebdb4794ae3df88ce8ad6cbb74608` (August 18).

Disposable original and patched source trees are staged under
`/tmp/picfetch-purego-audit/`; the h265 working tree now contains its reviewed
fix and gav1d retains separate original and patched trees. Audit harnesses and
human-readable logs live under `.scratch/purego-codec-audit/`; final JSON is
under `/tmp/picfetch-purego-audit/logs/`. The durable assessment and exact
patches are in `docs/`.

## Progress

- Exact snapshots, manifests, licenses, patent texts and relevant upstream
  sources were inspected. Native Linux/amd64 Docker and Go 1.27.1 were used;
  tool execution called the Go binary directly because the snap launcher was
  not usable inside the editor sandbox.
- Four targeted defects were reproduced: gav1d header-driven sample-table
  allocations, gav1d grid limit bypass, gav1d invalid supported-looking encoder
  output, and h265 zero-display-dimension success. Red tests precede the local
  fixes retained in [gav1d](../docs/codec-hardening/gav1d-v0.2.5-security.patch)
  and [h265](../docs/codec-hardening/h265-v0.2.3-security.patch) patch form.
- All upstream native/scalar suites passed. Final patched suites, 9/9 native
  admission checks, 9/9 WASM isolation checks and ten bounded 20-second guided
  fuzz campaigns passed. Strict `checkptr` passed for patched gav1d's AVIF and
  AV1 packages in native and `noasm` builds. Skipped reference tools and
  external conformance corpora remain unverified.
- The WASI prototype contained a 64 MiB guest-memory exhaustion, a CPU loop and
  output flooding; it denied guest filesystem/environment access and its TCP
  attempt failed. Docker provided the outer process limit, so this is not a
  packaged desktop sandbox.
- Final patch review found two residual gav1d aggregate-memory paths and
  incorrect AV1 level signaling. Isolation turns the memory paths into bounded
  worker failures but does not make the codec release-ready.
- Source and license review found no declared Go dependencies in either codec,
  but both retain copied-source and runtime obligations. Exact translated
  revisions are unrecorded, notice completeness remains unresolved, and HEVC
  patent clearance remains open.
- Native and `noasm` vet passed for both patched codec package pairs; the audit
  host and probe also passed vet and clean GoLand inspection. Before/after
  GoLand comparison found no warning introduced by the review patches. All four
  scalar package test binaries compiled for Linux/386; this host's seccomp
  policy prevented executing 386 binaries.

## Decision and handoff

Neither candidate is qualified for production. HEIC remains disabled and the
preferred AVIF route remains a small PicFetch-owned libavif/libaom adapter. A
future exception for either pure-Go codec requires the worker contract and all
adoption gates in the
[security evaluation](../docs/purego-codec-security-evaluation-2026-09-15.md).

No application source, dependency, package map or supported format changed.
The plan remains active until Ronin accepts the evaluation; acceptance can move
it to `finished_refactorings/` without further implementation.
