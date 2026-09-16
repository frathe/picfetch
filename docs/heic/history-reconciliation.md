# HEIC branch history reconciliation

Date: 2026-09-16. This record accounts for the two commits on
`feature/heic-hardening` against PR #28's published `52ed2df` baseline and the
reconciliation change that contains this document. Neither historical commit
is an ancestor of the PR. Their source remains accessible on the original
branch; no history is rewritten.

## Scope and acceptance

Ronin authorized restoring useful omissions into the same PR. The lead owns
code, documentation, review and fixes. One read-only scout checked ordinary
fixture provenance; the CI/review agent remains paused. This extends the
existing Deep plan rather than replacing the new helper architecture.

| Acceptance | Verification |
| --- | --- |
| Preserved decoder hardening | Compare all 107 `PICFETCH-SOURCE.json` entries byte for byte against `fc127b44`; `make check-heic-wasm`. |
| Qodana YAML rejected before path checks | `go test ./scripts/qodanaconfig`; actual Make gate rejects malformed YAML in an isolated temporary checkout. |
| Fixed guest backing storage and unchanged ceiling | `TestRuntimeMemoryGrowthReusesBacking` under compiler and interpreter; owned runtime regression suite and native macOS helper tests. |
| Useful ordinary compatibility coverage restored | `make test-h265` through the fixed WASI artifact; exact fixture hashes/provenance in testdata README. |
| Current app-wide threat model | Check relative links and review every changed assertion against the current source/qualification record. |
| Reviewable publication | Focused tests, build/provenance/import/Qodana/shard checks, GoLand inspections, signed commit and matching GitHub PR head. |

## `fc127b44` — Harden and isolate HEIC decoding

All **107 production/source-license files** in the maintained-source manifest
are byte-identical to the historical commit. These include the decoder's local
input/work/allocation checks. The following table accounts for every remaining
non-vendored path changed by that commit, grouped by responsibility.

| Historical paths | Current disposition |
| --- | --- |
| `.gitignore` | Restore `/fyne_metadata_init.go`; generated Fyne metadata remains local. |
| `AGENTS.md` | Restore source-maintenance and WASI-only ownership guidance, using current commands and honest test scope; omit the obsolete RSS-test command. |
| `ARCHITECTURE.md` | Current helper/client/guest map supersedes the historical module map; add the restored YAML-validator package and threat-model pointer. |
| `Makefile` | Current guest/build/native/package targets replace old targets. Restore YAML syntax validation. Host and guest advisory scans already remain present. |
| `THIRD-PARTY-NOTICES.md` | Current h265/Go/wazero notices and packaging inventory replace historical inline notices. |
| `docs/find-more-like-this/dependency-qualification.md` | Current source/license/runtime limitations live in `qualification.md` and `third_party/h265/PICFETCH.md`; old qualification is historical evidence, not current release approval. |
| `go.mod`, `go.sum` | Current root/guest replacements select the exact maintained h265 source; removed Rust decoder remains absent. |
| `internal/heicdecode/client.go`, `command_unix.go`, `command_windows.go`, `worker.go` | Replaced by dedicated `cmd/picfetch-heic-worker`, `client/`, `worker/` and `winisolation/`, with shared app-family admission and native restrictions. Do not restore self-execution/ordinary-privilege launch. |
| `internal/heicdecode/register.go`, `internal/imaging/heic.go` | Replaced by explicit injected `imaging.Reader`/`Source`; production activation remains disabled instead of globally registering HEIC. |
| `internal/heicdecode/runtime.go` | Replaced by `worker/runtime.go`; restore capped one-time backing storage with a real memory-growth regression. |
| `internal/heicdecode/wire/wire.go` | Current codec-free request/response/readiness/metadata protocol replaces it; retains bounded NRGBA8/NRGBA64 transport. |
| `internal/heicdecode/decoder.json`, `decoder.wasm`, `scripts/heicwasm/main.go`, `scripts/heicwasmbuild/main.go` | Superseded by separate `scripts/heicguest`, `scripts/heicbuild`, its exact manifest and helper-only embedded artifact. |
| `internal/heicdecode/client_test.go`, `runtime_test.go`, `testdata/capabilities/main.go` | Current owned process/IPC/memory/cancellation/capability tests cover the new boundary. Restore the omitted memory-growth allocation property; old test binaries and protocol are not current tests. |
| `internal/imaging/exif.go`, `loader.go`, `internal/similarity/analyze.go`, `facts.go`, `search_worker.go`, `internal/ui/exifwin/metadata.go`, `filework.go` | Current injected reader and shared analysis pipes propagate context, validated pixels and metadata; byte-only helpers refuse HEIC/native preview fallback. |
| `internal/imaging/heic_test.go` | Current source, native-helper and guest tests cover admission, upright metadata/pixels, cancellation, alpha and explicit ten-bit output. Extend ordinary container/chroma coverage below; no claim of full historical test equivalence. |
| `internal/imaging/heic_leak_test.go` | Legacy in-process/RSS comparison does not qualify the new process-family budget. Native memory/cleanup guards and platform qualification replace that claim. |
| `internal/imaging/save.go` | HEIC encoding remains unavailable; current ordinary-image save behavior does not require the old decoder comment. |
| `plans/2026-09-14-heic-hardening.md`, `plans/2026-09-14-heic-isolation.md`, `todos.md` | Historical evidence remains in Git; current restoration plan, qualification and todos own incomplete work. |
| `qodana.yaml` | Current exact exclusions cover present files, including restored validator tests; old exclusions for absent paths are obsolete. |
| `scripts/qodanaconfig/main.go` | Restore actual YAML parsing before Make's text inventory; reuse the existing pinned YAML dependency. |

The historical vendored `PICFETCH.md` is replaced by the current provenance,
upgrade and qualification record. An additional manifest pins the unchanged
production copy.

### Historical test-tree disposition

The old vendored tree also contained 21 test files, 90 fixture/corpus files and
one upstream workflow absent at `52ed2df`. Copying that entire tree would restore
native decoder/assembly and encoder testing plus historical reproducer/fuzz
inputs, rather than qualify PicFetch's current WASI-only path.

Restore four unchanged ordinary positive fixtures (`chroma422`, `chroma444`,
`lossless`, `thumb`) and their configuration/decode agreement checks through
the current bounded guest. Strengthen the existing alpha fixture to check its
left/middle/right ramp. Keep existing ordinary metadata, orientation, ten-bit,
input-admission and owned containment tests. This is useful additional coverage,
**not** the complete historical decoder suite or a security certification.
The remaining historical tests/fixtures remain available in the old commit;
none is represented as having passed or being covered by these eight fixtures.

## `73cb3c9` — added threat model

Restore a current root `THREAT-MODEL.md`, retaining its application assets,
attacker/operator/user-intent assumptions, filesystem and cache/privacy risks,
release authority, severity guidance and private-reporting policy.

Replace the historical HEIC section: the new helper has per-platform OS
restrictions, 64 MiB input, a 60-second ceiling, shared GUI/analysis admission,
new paths and explicit disabled production status. Update package/updater
boundaries and document the first-upgrade limitation. The old document remains
historical evidence; it is not copied as an assertion of current behavior.

## Verification evidence

- Before changes, the actual Make exclusion-check recipe accepted malformed
  `exclude: [` YAML (exit 0) in an isolated temporary repository.
- Before memory preallocation, both owned memory-growth regression cases
  failed because growth replaced the backing buffer; restoring the setting
  passes while preserving refusal above the two-page ceiling.
- The YAML validator's five cases pass. The actual Make gate now rejects the
  same malformed YAML and accepts valid configuration. Exact Qodana exclusions
  include the new test file.
- Focused race runs pass for `internal/heicdecode/worker`, `scripts/heicbuild`
  and `scripts/qodanaconfig` (14.833 s, 17.929 s and 1.184 s). The fixture run
  exercises all eight ordinary inputs. Owned memory/cancellation regressions
  also pass with `heicinterpreter` (0.416 s).
- `make generate-heic-wasm` refreshes the input manifest without changing the
  guest artifact. `make verify-build` passes formatting, TUF, Qodana, generated
  assets/notices, source and artifact reproducibility, native import guards,
  vet and build. The initial restricted-cache failure was rerun with access to
  the existing Go cache; no source or policy was weakened.
- `make heic-native-macos` passes all four required Apple Silicon guards with
  preallocation enabled: sandbox/readiness/decode/cancellation, both runtime
  engines, signed bundle install/rollback, and the legacy upgrade limitation.
  The interpreter's expected deadline is a successful containment check, not
  a claim that it decoded the 12MP image.
- GoLand inspections, including weak warnings, are clear for all five changed
  Go files. The runtime test was re-inspected after its owned module ceiling
  was corrected. All local threat-model links resolve; the lead checked
  changed behavior claims against source and qualification evidence.
- The final read-only inventory confirms 107/107 source/license hashes and
  4/4 recovered fixture bytes match `fc127b44`, with no mismatches.
- Local canonical shard validation remains unverified: the Linux-only direct
  target was mistakenly invoked on macOS, then the documented Docker target
  could not start because the daemon was unavailable. No UI test or shard
  assignment changed. Native Linux CI remains the canonical gate.

Publication uses the existing PR branch and configured commit signer, without
merging or rewriting either historical branch. The final task response records
the resulting commit and remote-head verification. Production activation and
the separately paused CI/GitHub review loop are outside this reconciliation's
completion claim. At baseline `52ed2df`, native Linux/macOS guards and all UI
race shards passed, while Windows HEIC guards and the non-UI Linux race job
failed. Those results do not qualify the new memory configuration on other
platforms; fresh CI is required.
