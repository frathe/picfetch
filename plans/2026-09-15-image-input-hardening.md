# Image input hardening

Status: implemented and full verification passed; live IDE tag refresh pending.
Authorized by Ronin on 2026-09-15.
Route: Deep, because decoder admission and build selection cross packaging and platforms.

## Contract

Close the four static-review findings against `367c78fe691dd883cd000c726d92940dafa0ebe6`:

1. ICO configuration and decoding select the same single image. Validate the directory,
   every referenced span and embedded dimensions before decoding. Bound ICO bytes and
   entries independently. Preserve PNG icons and common uncompressed DIB icons.
2. Preflight SVG bytes, XML nesting and expansion work before calling oksvg. Permit
   direct definition reuse; reject use elements inside definitions (including recursive
   reuse). Bound parsing and check cancellation at parser reads and compositing boundaries.
3. Require `nodynamic` and prohibit `wasm2go` in every imaging build, with supported tags
   propagated through Make, CI, packaging and nested verification commands.
4. Admit animated GIFs using a frame cap and conservative estimates of palettes,
   frame objects, source pixels, output pixels and compositing scratch. Preserve static
   fallback and preview downscaling. Propagate cancellation into decoding/compositing.

No HEIC restoration, new codec dependency, upstream publication, fuzzing, exploit
reproduction, commits or desktop launch is part of this change. Limits reduce exposure;
they do not certify decoder correctness or provide an OS memory sandbox.

## Decisions and dependency obligations

| Decision | Reason |
| --- | --- |
| Local bounded ICO adapter | Avoid the registered dependency decoder's allocation and selection behavior. |
| Conservative SVG preflight | The pinned parser's definition storage differs from DOM subtrees; rejecting nested reuse avoids a competing reference interpreter. |
| Build-time AVIF gate | A runtime check occurs after native library initialization. |
| Shared GIF admission estimates | Full viewing and previews must account for source allocations before DecodeAll. |

Dependency versions remain unchanged: fyne-io/image v0.1.1 (BSD-3-Clause; XPM
remains), fyne-io/oksvg v0.2.0 (BSD-3-Clause), x/image v0.46.0 (BSD-3-Clause),
gen2brain/avif v0.6.0 (MIT wrapper plus embedded native component obligations),
and Go 1.27.1 (BSD-3-Clause). No upstream code is copied or forked. Existing notices
remain required. The AVIF component notice/libyuv provenance gaps already recorded
in `todos.md` remain release blockers; selecting embedded WASM does not resolve them.

## Tasks and proof

All implementation, design and review: lead inline. One read-only scout maps build
routes while the lead handles parser admission; no review or fix delegation.
Graph: ICO, SVG, AVIF and GIF are independent; all feed final verification.

| Task | Files / contract | Acceptance command |
| --- | --- | --- |
| ICO | imaging/ico.go, loader imports, ICO tests/fixtures; bounded explicit decoder dispatch | `go test -tags no_emoji,nodynamic ./internal/imaging -run 'ICO|Ico'` |
| SVG | imaging/svg_limits.go, vector.go, svg.go, thumbnail.go, loader.go; ParseVectorContext | `go test -tags no_emoji,nodynamic ./internal/imaging -run 'SVG|Vector'` |
| AVIF | internal/avifpolicy, Makefile, workflows, nativeguards/testshards, packaging contracts and command docs | `go test -tags no_emoji,nodynamic ./scripts/nativeguards ./scripts/testshards ./scripts/msixstage ./internal/imaging -run 'AVIF|Avif|Make|Suite|Tags|Packaging'` |
| GIF | imaging/gif.go, preview.go, loader.go and tests; shared admission and context | `go test -tags no_emoji,nodynamic ./internal/imaging -run 'GIF|AnimatedPreview'` |
| Final | ARCHITECTURE, Qodana exclusions, todos and this record | `make verify`; GoLand inspections of changed code |

Use ordinary small fixtures and boundary arithmetic tests. Never run a red test that
intentionally invokes unbounded recursion or allocation: those paths retain static
evidence, and refusal checks execute only after the admission guard exists. No runtime
exploitability claim is made. Host focused tests use the cached Go 1.27.1 toolchain;
canonical full tests use the repository's native Linux/amd64 Docker runner.

## Evidence and cost ledger

- Baseline working tree: user edits in todos.md and unrelated untracked codec evaluation
  documents; preserve those edits.
- One read-only build-route scout; zero implementation/review spawns.
- Focused red/green, local build checks and final Docker gate passed.
- Budget: one review per task plus final integration review; one full suite after
  focused checks, repeated only to resolve failures or changes.

### Implementation and focused evidence

- ICO regression first failed with probe 8x8 versus decoded 16x12. It also failed
  when the test imported the older ICO registration used by Fyne's desktop driver.
  Explicit probe/decode dispatch now passes both cases, validates all spans and
  embedded dimensions, and decodes only the selected image. Invalid ICO admission
  cannot fall through to the RAW embedded-preview path. Limits: 16 MiB, 256 entries,
  256 pixels per axis; PNG and uncompressed 1/2/4/8/24/32-bit Windows DIB layouts.
- SVG's harmless unused definition-reuse test failed before the preflight guard.
  Direct reuse still renders; nested reuse, excessive XML depth/source/expanded
  work, non-finite sizes and cancellation are covered. Limits: 8 MiB source, 64
  XML levels, 100,000 expanded elements and 16 MiB conservatively expanded source.
  Renderer reads observe cancellation; one bounded path/raster operation may finish.
- GIF's memory-boundary test failed under pixel-only accounting and passes with
  source pixels, two canvases, 64 KiB decoder scratch and 8 KiB per-frame allowance
  plus output pixels. Both animation paths enforce 4,096 frames. Previews additionally
  reserve 128 bytes per retained frame; full decoding/compositing observes context.
- Build-command regression tests failed before `nodynamic` propagation and pass
  afterward. Six-target AVIF file-selection tests select `avif_wazero.go` and
  `purego_other.go`; native and wasm2go implementations are excluded. Actual
  `go list -deps` rejects both missing-nodynamic and nodynamic+wasm2go builds.
- All tests in imaging, nativeguards, testshards, msixstage, plistdoctypes and
  tagvectors pass. Imaging also passed the focused race run.
- `make verify-build` passed formatting, TUF freshness, Qodana test exclusions,
  generated assets/vectors, six-target updater notices, vet and complete build.
- Windows amd64 and arm64 `CGO_ENABLED=0 go build -tags no_emoji,nodynamic
  ./internal/...` both passed, matching the existing cross-build CI scope.
- First full Docker run: all three UI shards passed; the only failing package
  was tagvectors' old Explorer setup command expectation. That expectation was
  corrected and its package passed. The final run includes that correction and
  the ICO registration-order integration fix.
- Final `make verify` passed (exit 0): formatting/TUF/exclusions/assets/notices,
  vet/build, shard inventory and all four Docker race partitions. Evidence:
  `.scratch/race-runs/20260915T153053Z-mvk0Sm/`; complete command log:
  `/tmp/picfetch-hardening-verify-final.log`. All 686 top-level UI runnables were
  assigned to the three validated shards. No source changes followed this run.

Implementation review is complete. No commits, pushes, PRs, dependency upgrades,
desktop launches or fuzz campaigns were made for this viewer change. Existing
codec-evaluation documents and their todos were preserved. The plan remains here
until the live IDE verification below is finished; distribution qualification and
native Windows/macOS execution remain separate release work.

### IDE and external verification

Every changed Go file was inspected, including weak warnings, and changed code
was re-inspected after fixes. Remaining test-duplication warnings in gif_test.go,
loader_test.go and scripts/testshards/main_test.go are covered by their existing
exact-file Qodana exclusions; repeated fixture setup remains intentional.

The running GoLand instance has not loaded the updated module build tags and still
reports the expected build-constraint error at imaging's avifpolicy import. The
ignored local module settings now contain `no_emoji nodynamic`; completing IDE
verification requires applying/reloading those settings and re-inspecting. The
user has been asked asynchronously while tests continue. No source suppression
weakens the build guard. Fresh Qodana module templates are prepared by its workflow;
their XML/tags validate, but hosted Qodana execution remains unverified. See the
[GoLand setting](https://www.jetbrains.com/help/go/configuring-build-constraints-and-vendoring.html)
and [Qodana plugin configuration](https://www.jetbrains.com/help/qodana/extending-qodana-plugins.html).
The IDE also reports unresolved action-input metadata on unchanged GitHub Action
parameters; hosted workflow analysis is not claimed as verified locally.

The first escalation for full verification was rejected because the automatic
approval reviewer reported model capacity exhaustion. Local verification continued;
the subsequent reviewed invocation was approved and ran normally.
