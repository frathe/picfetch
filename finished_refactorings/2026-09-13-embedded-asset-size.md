# Smaller embedded application assets

Status: complete, accepted by Ronin on 2026-09-13. Owner: Pico.
The native Linux/amd64 race suites subsequently passed in PR #21 CI; see the
[CI follow-up](#pr-21-lossless-optimization-check--2026-09-13).
Route: Deep, because the
change spans UI packages, generated data and every platform's packaging route.

Deliverable: smaller ordinary executables through app-specific gaze atlases,
appropriately sized raster artwork, exact generated tag-vector data, and
Fyne's supported `no_emoji` option.

## Decisions and scope

| Decision | Contract |
| --- | --- |
| Security verification | Keep upstream Sigstore/TUF and all existing verification. MA-024 remains declined. |
| Character artwork | Retain 16 gaze directions and neutral pose; preserve their decoded pixels and Trane's existing fringe correction. Preserve full source atlases outside the embed. |
| Other raster artwork | Derive display-sized images from retained sources, accounting for high-DPI rendering and layout constraints. No app-icon degradation. |
| Emoji | Ronin explicitly accepts removal of the bundled emoji font, including loss of that font's fallback coverage. |
| Vectors | JSON remains authoritative. A standalone offline Go tool emits catalogue-ordered little-endian float32 bytes with the existing digest. Commit generated output for direct Go builds; Make regenerates it for app builds, CI checks freshness. |
| Packaging | Normal compilation/signing; no executable packing, post-sign modification, new runtime dependency, or precision reduction. |

Ronin explicitly approved deterministic local image processing on September 13.
No further image-processing approval is needed for this scope.

## Acceptance and task graph

`T1 vectors` and `T2 artwork` are independent; `T3 build integration` consumes
their assets; `T4 verification/documentation` joins all three.

### T1 — Exact generated vectors

Owner: T0 inline. Files: `scripts/tagvectors/`, `internal/similarity/tags.go`,
`tags_test.go`, generated `.bin`, generator README.
Contract: `NewTagger` retains numeric/identity/digest validation and scores;
generator supports write and check modes without loading models or UI.
Tests: source/binary bit equivalence, unchanged tag results, corrupted/truncated
data and invalid catalogue rejection, deterministic conversion, stale output.
Verify: `go test ./scripts/tagvectors ./internal/similarity`; `make check-tag-vectors`.
Budget: zero implementation spawns; one review round, fixes inline; no full suite.

### T2 — Only displayed artwork

Owner: T0 inline. Files: `internal/ui/widgets/gaze.go`, atlas embeds, retained
sources and deterministic tooling, relevant UI/help tests, resized image embeds.
Contract: compact gaze frame layout; original 192 × 208 frame pixels and all
17 poses preserved. Non-atlas dimensions follow measured display constraints.
Tests: compare all compact frames against original cells; reject old/invalid
layout; existing pointer/rest/reopen/fringe tests; visual inspection of resized art.
Verify: `go test -tags no_emoji ./internal/ui/widgets ./internal/ui/help`;
`go test -tags no_emoji ./internal/ui -run 'TestTrane|TestExplorer.*Setup'`;
generator freshness/dimension checks and rendered image inspection.
Budget: two read-only Scout passes, no implementation spawns; one review plus fixes;
no full suite.

Scout gate: G1 bounded asset-use/source sweep; G2 returned locators checked with
shell; G3 zero edited files; G4 breadth avoids loading every artwork consumer;
G5 these non-atlas layout details were not yet understood. Rule S handles file
sizes; Scout only follows consumers. Rule W: no implementation handed off.

### T3 — Consistent build inputs

Owner: T0 inline. Files: Makefile, CI validation, packaging/build documentation
and relevant existing build-contract tests.
Contract: supported app build/package routes include `no_emoji`; Store retains
`microsoftstore`; vector generation precedes builds; CI rejects stale output.
Verify: Make dry runs of native/macOS/Linux/Windows/Store routes; focused build
contract tests; native `make build`; inspect final dependency embeds/binary.
Budget: zero spawns; one review plus fixes; no full suite.

### T4 — Final verification and evidence

Owner: T0 inline. Files: this record, ARCHITECTURE.md, todos.md, research note,
Qodana test exclusions and UI shard manifest if necessary.
Verify: focused tests/race checks, GoLand inspections of changed code (including
weak warnings), `make verify`, same-toolchain baseline/candidate sizes, ordinary
signature verification on locally produced artifacts where applicable.
Budget: zero spawns; one full final gate. Complete Linux/amd64 race validation
requires a native amd64 daemon or CI; unavailable gates remain explicit.

## Licenses and provenance

No dependency additions or upgrades. Fyne remains `fyne.io/fyne/v2 v2.8.0`
(BSD-3-Clause); removing its bundled emoji resource does not remove other Fyne
notice obligations. Tag vectors retain the pinned SigLIP2 export and Apache-2.0
provenance recorded in `scripts/explorertags/README.md`. Preserve asset source
provenance and existing LICENSE/THIRD-PARTY-NOTICES delivery in release archives
and Store packages. The separate updater-notice backlog item remains open.

## Evidence and cost ledger

No commits authorized. All implementation tasks are complete. All ten generated
images pass exact decoded-pixel checks against the source transformation. The
manual's three static images target height 440 for their 220-unit rows. Original
atlases and artwork remain outside the application's embed closure.

### Final size and performance

The final native build uses the same Go 1.27.1 darwin/arm64 toolchain,
`-trimpath`, `-s -w`, and `GOFLAGS='-mod=readonly -buildvcs=false'` as the baseline,
with the intended new `no_emoji` tag. `make build
BIN_DIR=.scratch/asset-size/final` produces **40,836,338 bytes**, down from
**50,629,138 bytes**: **9,792,800 bytes (19.34%) smaller**.
SHA-256: `866fbe9d7190148ca9979aa8c9ab1ff97a2bb65d8341166b06e8e1b7c41a67a9`.
This is the native executable measurement, not a claim about every platform's
signed package size.

| Data | Before bytes | After bytes |
| --- | ---: | ---: |
| Trane atlas | 2,342,454 | 480,466 |
| Finis atlas | 1,961,722 | 417,680 |
| Explorer PNG | 2,076,221 | 369,022 |
| Welcome fallback | 56,446 | 131,868 |
| Error placeholder | 47,172 | 111,954 |
| Scan illustration | 61,742 | 96,290 |
| About illustration | 67,928 | 238,870 |
| Manual picture frame | 74,926 | 211,226 |
| Manual digging | 278,846 | 289,460 |
| Manual wagging | 211,912 | 144,236 |
| Tag vectors | 1,020,288 | 230,400 |
| Bundled emoji font | 4,232,712 | 0 |

The ten image payloads save 4,688,297 bytes together. Some previously lossy
WebPs grow because resized pixels are encoded losslessly, avoiding another
lossy pass. All new image payloads and binary vectors were found byte-for-byte
inside the final executable; the old images, JSON vectors and emoji font were
absent. No Go dependency versions or verifier implementation changed.

Three sequential native atlas benchmark samples, using each actual old/new
decoder with its matching source data: Trane fell from 54.8–57.0 ms to
10.7–11.0 ms per decode; Finis from 51.8–53.8 ms to 10.9–11.0 ms.
Allocations fell from about 17.3 MB to 5.8 MB per atlas. These measure atlas
decoding and extraction, not whole-app startup or Trane's unchanged fringe pass.
Raw evidence: `.scratch/asset-size/{before,after}-gaze.txt`.

### Final checks

- `make check-app-assets` and `make check-tag-vectors` pass. The artwork checker
  is also tested against a same-sized output with wrong pixels.
- Both compact-atlas shape and old-layout rejection tests were observed failing
  before implementation. All 17 frames for each character now compare exactly
  against the preserved original cells; pointer/rest/reopen and fringe tests pass.
- Final focused race checks pass for `internal/similarity`, both generators,
  packaging/shard tooling, `internal/ui/widgets`, `internal/ui/help`, and
  `internal/ui`'s Trane and Visual Similarity Explorer tests.
- `make golden TEST_IMAGE=picfetch-mosaic-verify:local` passes under Linux/amd64
  emulation in the inspected Ubuntu 24.04 image. Only the two placeholder
  goldens needed updating; their changes are confined to the illustration
  (170 × 191 pixels at 340,73). No files under `testdata/failed` are included.
- Manual, About and Finis captures were inspected in light/dark themes at 1x
  and 2x scale, with all manual illustrations intact. Explorer setup captures
  were inspected at 720 × 660 and its existing smaller-window regression ran.
  Captures: `.scratch/asset-size/visuals/`; temporary overlay helpers remain
  outside the repository and add no UI shard entries.
- All 13 changed Go files pass the pinned formatter and have complete GoLand
  inspection results. No new findings; existing test-duplication warnings are
  documented below and covered by the standing Qodana exclusion.
- `make vet`, `go build -tags no_emoji ./...`, Windows/amd64 internal-package
  vet, and Windows/arm64 internal-package build pass. These do not substitute
  for native Windows GUI/signing tests.
- TUF expiry and Qodana exclusion checks pass. Final local
  `codesign --verify --strict --verbose=2` passes. No signed release was produced.
- `make verify` was attempted and stopped at native amd64 Docker admission.
  The full Linux/amd64 race gate remains pending CI; local global formatting
  also retains the unrelated scratch-file limitation described below.

Ronin requested that this plan be marked done on September 13, 2026; it is
archived in `finished_refactorings/`. The verification limits above remain
recorded, with the full native amd64 gate tracked separately in `todos.md`.
Measurements and check results describe the implementation validation snapshot;
subsequent image optimizations are not covered by that evidence. No commit,
push, signing configuration change, or release publication was performed.

### Earlier implementation evidence

- Red: `TestEmbeddedTagVectorsMatchJSON` failed on 1,020,288 embedded bytes
  instead of 230,400; the updated asset/loader passes exact bit comparisons.
- Red: Linux packaging fixture rejected missing vector generation and missing
  `no_emoji`; all five cross-packaging routes now pass with both requirements.
- Focused race tests pass for `internal/similarity`, `scripts/tagvectors`,
  `scripts/appassets`, `scripts/msixstage`, and `scripts/testshards`.
- GoLand inspections completed for the eight changed/new Go files. No new
  findings. Four existing weak duplicate-code findings in unchanged portions
  of `scripts/testshards/main_test.go` remain covered by its exact-path Qodana
  test exclusion; no production refactor or new suppression is warranted.
- Native intermediate build: `make build BIN_DIR=.scratch/asset-size/working
  GOFLAGS='-mod=readonly -buildvcs=false'`, Go 1.27.1 darwin/arm64. Size
  45,559,346 bytes, versus 50,629,138 baseline: 5,069,792 bytes smaller.
  SHA-256 `0b2668899e4fddbb5b64bb7f8e3204d213dcef385770f4df54ae1292ffed1063`.
  Complete old emoji/JSON payloads are absent; exact generated vector bytes
  are present. `codesign --verify --strict --verbose=2` passes on this local
  executable; this is not a claim of Developer ID or Windows release signing.
- Native tagger microbenchmarks (three 300 ms samples): original JSON loader
  3.87–4.00 ms/op and about 834 KB/op; binary loader 0.153–0.156 ms/op and
  about 249 KB/op. Evidence: `.scratch/asset-size/{before,after}-tagger.txt`.
  Baseline source was supplied with a Go overlay; identical benchmark bodies
  exercised each actual `NewTagger`. This is not a whole-app startup benchmark.
- `make check-test-platform` reports `linux/aarch64`; native Linux/amd64 full
  race validation remains unavailable locally. Worker policy is unchanged.
- `make vet` passes with `no_emoji`. The repository-wide `make fmt-check`
  stops at the pre-existing `.scratch/mosaic-wallpaper/capture_test.go`; changed
  Go files were formatted with the pinned goimports tool. The unrelated scratch
  file was not changed.
- Local cwebp is 1.6.0; its BSD-3-Clause terms were verified in the installed
  `/opt/homebrew/Cellar/webp/1.6.0/COPYING`. It is development-only tooling.

| Task | Spawns budget/actual | Review rounds | Full suite |
| --- | --- | --- | --- |
| T1 | 0/0 | 1 | no |
| T2 | 2/2 read-only passes | 1 plus inline encoder fix | no |
| T3 | 0/0 | 1 | no |
| T4 | 0/0 | 1 | attempted; native amd64 CI pending |

### PR #21 lossless optimization check — 2026-09-13

On `34ab671`, the [validation job](https://github.com/frathe/picfetch/actions/runs/34725724787/job/103639378598?pr=21)
failed at `make check-app-assets`: the Explorer PNG differed from its source
transformation after ImageOptim processing. A comparison of the committed
source/output isolated 140,048 output pixels with different RGB beneath alpha
zero. There were no visible RGB or alpha differences. The current locally
optimized source gives the same result.

The checker now clears RGB beneath alpha zero in its temporary comparison
buffers for ordinary illustrations. Dimensions, every alpha value, and RGB
where alpha is nonzero must still match exactly. Gaze atlases retain the
original exact-byte comparison, including hidden RGB. Generation, runtime
decoding, artwork files, dependencies, and signing are unchanged by this fix.

Regression tests reproduce the optimizer mismatch before the fix and pass
afterward. They also reject one-channel changes at opaque and alpha-one
pixels, alpha changes in either direction, and hidden RGB changes in gaze
atlases. No ImageOptim installation is needed to run these tests.

Validation of the fix:

- `make check-app-assets` passes on the existing optimized assets.
- The Linux/amd64 build of the checker passes in the existing Docker image,
  with a read-only repository mount and networking disabled. This focused
  image check does not exercise or bypass worker isolation.
- `go test -tags no_emoji -race ./scripts/appassets -count=1` and
  `go vet ./scripts/appassets` pass.
- Both changed Go files pass the pinned formatter and GoLand inspections,
  including weak warnings. The IDE build reports success with limited build
  diagnostics; the explicit Go build and vet commands provide the build evidence.
- Tag-vector freshness, Qodana test exclusions, and TUF expiry checks pass.
- The original hosted run already passed all four Linux race partitions,
  Windows tests, and macOS arm64/amd64 native guards. This closes the earlier
  native amd64 race follow-up for the asset implementation at `34ab671`.
  The local checker fix still needs a fresh hosted validation run after push.

The user’s additional image edits were preserved. No commit or push was made.

### PR #21 Codex review loop — 2026-09-13

Ronin invoked the GitHub Codex review loop, authorizing fix commits, pushes and
review-thread dispositions. The earlier no-commit statements describe their
original sessions. The artwork fix on `cbf106d` now passes
[hosted validation](https://github.com/frathe/picfetch/actions/runs/34726451266/job/103641292187).

Route: Standard follow-up within this accepted Deep plan. Lead owns assessment,
the Makefile fix, regression and final review. The confirmed P2 finding omitted
vector generation from Explorer profiling and acceptance targets. The same gap
also affected the real-asset install test, which runs production analysis.
All seven Explorer targets now depend on generation, so setup/download builds
also share the same current embedded data.

Files: Makefile, the existing `scripts/tagvectors/main_test.go`, this record and
`todos.md`. No new test files, UI tests, dependencies or package moves.
Acceptance: every Explorer target's Make dry run generates vectors before its
Go build/run/test command. Verify with `go test ./scripts/tagvectors -run
'^TestMakeGeneratesVectorsBeforeExplorerBuilds$' -count=1`.
The regression failed before the fix on all six previously omitted targets;
the existing `explorer-evaluate` path passed.

Verification after the fix:

- `go test -tags no_emoji -race ./scripts/tagvectors ./scripts/appassets -count=1`
  passes (1.582s and 4.543s respectively).
- `go vet ./scripts/tagvectors ./scripts/appassets`, the pinned formatter for
  the changed Go file, and `git diff --check` pass.
- `make check-tag-vectors check-app-assets check-qodana-test-exclusions
  check-tuf-root` passes.
- GoLand inspections of the changed Go file and Makefile complete with no
  findings, including weak warnings.
- Full race and native platform suites run in GitHub CI under this workflow's
  explicit local-test exception. The final commit's code/security review and
  CI results are retained in [PR #21](https://github.com/frathe/picfetch/pull/21).

Delegation: one read-only Scout collects Qodana artifacts and CodeQL alert
metadata while the Lead fixes the prerequisite. G1 bounded report collection;
G2 retained API JSON and actual SARIF; G3 zero repository edits; G4 artifact
collection is independent of the fix; G5 Lead has not loaded the report details.
Rule S scripts JSON extraction; the Scout follows report locations and run
identity. No assessment or fix is delegated. Budget/actual: 1/1 Scout; local
full suites 0; fresh GitHub review rounds continue until the latest head is clean.
