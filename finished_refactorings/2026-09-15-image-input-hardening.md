# Image input hardening

Status: implemented; review and hosted verification evidence tracked in
[PR #27](https://github.com/frathe/picfetch/pull/27). Local full verification
passed; live IDE tag refresh remains pending.
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
reproduction or desktop launch is part of this change. The September 15 review-loop
request authorizes PR creation, fix commits, pushes and review replies. Limits reduce exposure;
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

Implementation review is complete. No dependency upgrades, desktop launches or
fuzz campaigns were made for this viewer change. Existing
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
their XML/tags validate and PR #27's hosted Qodana report has no findings. See the
[GoLand setting](https://www.jetbrains.com/help/go/configuring-build-constraints-and-vendoring.html)
and [Qodana plugin configuration](https://www.jetbrains.com/help/qodana/extending-qodana-plugins.html).
The IDE also reports unresolved action-input metadata on unchanged GitHub Action
parameters; hosted workflow analysis is not claimed as verified locally.

The first escalation for full verification was rejected because the automatic
approval reviewer reported model capacity exhaustion. Local verification continued;
the subsequent reviewed invocation was approved and ran normally.

### GitHub Codex review loop

The user invoked the repository review loop on September 15. Deliverable: an open
PR with a fresh clean Codex code review on its latest commit, completed security
review, no actionable Qodana/CodeQL results and passing required CI. Merge and
release require separate authorization. The lead owns findings and fixes; each
confirmed behavior defect receives a failing bounded regression before its fix.
Focused local tests prove fixes; CI owns the complete race suite in this loop.

- Initial branch `feature/sec-hardening` was clean at `11a0644`, but its imaging
  test could not build because the AVIF policy package was absent. All nine omitted
  files were already committed in its direct child `77a6fc0` on
  `feature/dep-hardening`. Fast-forwarding recovered the original implementation,
  Qodana templates and linked research records without rewriting either commit.
- One read-only scout located that commit while the lead verified the build and
  GitHub state. Gate: bounded file recovery question, Git tree evidence as oracle,
  no writes/shared ownership, smaller context than implementation, no delegated
  review. The lead confirmed the tree diff and performed the fast-forward.
- Proof commands: focused imaging/build-tool tests; `gh pr checks` and workflow
  logs for CI; GraphQL review threads and REST reviews for Codex; downloaded
  post-suppression `qodana.sarif.json` and CodeQL analyses/alerts for static analysis.
  Review dispositions and live head/check evidence belong in the PR and this record.
- Patch-file whitespace is intentional unified-diff context; production-source
  whitespace is checked separately without rewriting the retained patches.
- [PR #27](https://github.com/frathe/picfetch/pull/27), initial review head
  `681562e`: focused imaging and five build-tool packages passed after the
  fast-forward. [CI run 34994040147](https://github.com/frathe/picfetch/actions/runs/34994040147)
  passed validation, all four Linux race partitions, Windows tests and both macOS
  native-guard jobs. The local full race suite was not duplicated for this loop.
- [Qodana run 34994040172](https://github.com/frathe/picfetch/actions/runs/34994040172)
  passed. Its final `/qodana.sarif.json` identifies `681562e` and contains zero
  results, as do `/end/qodana.sarif.json` and the rendered report's result set.
  The `/start/` baseline is a different report and is not the acceptance result.
- [CodeQL run 34994040263](https://github.com/frathe/picfetch/actions/runs/34994040263)
  passed Go and Actions analysis. Both downloaded SARIFs contain zero results;
  the analyses target merge commit `39926be1`, whose head is `681562e`, and report
  no extraction error or warning. No open PR code-scanning alerts were returned.
- Codex security review completed at 16:22:43 UTC with no findings. Code review
  completed at 16:27:39 UTC with one P2 finding: contributor-facing vet/test
  commands omitted the newly required build tags. The lead confirmed that
  untagged `go vet ./internal/imaging` fails at the policy import, then updated
  CONTRIBUTING and the PR checklist to use `make verify`, and the root guide's
  direct commands to supply the tags. `make vet` and the documented
  `go test -tags no_emoji,nodynamic -run TestE2E -v ./internal/ui/...` command
  passed after the correction (all 13 E2E tests). GoLand re-inspection of the
  three corrected guidance files reports no findings.
  This documentation fix reuses the existing build-policy tests and validates
  the documented commands directly; no test merely mirrors Markdown text.
- Live GoLand still reports the module-tag configuration error; fresh hosted
  Qodana verifies the committed template configuration. The review-loop changes
  add no source suppression. An existing Markdown directory-link warning was
  resolved by removing the trailing slash from the translations link.
- After this documentation/evidence commit, a new code review and security
  review are required on the latest head, alongside its CI and static-analysis
  results. Final head, run IDs and dispositions are recorded on PR #27 so
  recording completion does not itself invalidate the reviewed commit.
