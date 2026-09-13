# Microsoft Store combined release

Route: Standard. One publisher capability across two tooling packages; no Windows
runtime changes. Pico owns design, implementation, review and fixes. Ronin chose
one approval covering reconciliation and submission, retaining the separate modes.

## Contract and scope

- Add `release` alongside `check`, `preview`, `reconcile` and `submit`. Automatic
  Store build completion selects `release`; standalone commands keep their scope.
- GitHub-only preparation freezes the current tag/run/artifact/notes plus the
  exact active previous receipt when present. Its summary explicitly authorizes
  reconciling that receipt and submitting the selected release only after the
  previous version is confirmed published. No post-approval release discovery.
- One protected job consumes that immutable combined approval, under the existing
  workflow serialization and one local claim. Microsoft credentials stay in the
  protected environment. No changes to reviewer policy or credential scope.
- Processing yields a clear waiting result without submitting the next version;
  failure, ambiguous state and drift stop the sequence. A later run needs fresh
  preparation/approval. No certification polling or additional release selection.
- No eligible next artifact means no combined approval; separate `reconcile`
  remains available. Implementation makes no workflow dispatches or live Store
  writes. Ronin subsequently authorized committing and pushing the change.

## Acceptance and tasks

1. Preparation freezes both releases and the conditional published base without
   Microsoft access; absent/inactive receipts follow ordinary submission.
2. Execution reconciles only the frozen previous receipt, submits only the frozen
   new release after publication, and rejects changed approval/receipt/artifact.
3. Processing/failure/conflicts cannot create the next submission. Existing
   `submit` still rejects an active receipt; `reconcile` remains one release only.
4. Workflow keeps one approval job, immutable artifact ID/hash, trusted main,
   environment checks and producer-run binding; all manual modes remain exposed.

Seams: existing `storepublish.run` with persistent fake HTTP services and existing
MSIX workflow contract tests. These are the repository's established public CLI
and workflow boundaries. Use vertical red/green slices, then focused race checks.

Implementation files: `scripts/storepublish/{main.go,approval.go,lifecycle.go,
main_test.go}`, `.github/workflows/microsoft-store-publish.yml`,
`scripts/msixstage/msixstage_test.go`, `docs/microsoft-store.md`. Keep the package
map and this evidence record/todos current; add no dependencies or test files.

Verify: `go test ./scripts/storepublish ./scripts/msixstage`; focused race tests,
format/vet/build checks, `make verify` platform preflight and GoLand inspections.
Live GitHub environment enforcement and Microsoft effects remain unverified
until an approved run after landing; fake services establish the local contract.

Delegation: one read-only scout mapped existing test locations while Pico read
the lifecycle. Bounded question, file/line output oracle, no mutations, no overlap
with lead code reading, smaller context than implementation; no review delegated.
The same scout later located the isolated GoLand CLI invocation; Pico executed
inspections and reviewed every finding.

## Evidence

Implemented locally; native Linux/amd64 CI and an approved live Store run remain
outstanding. Implementation made no workflow dispatches, Store mutations or changes
to the pre-existing artwork/transparent-PNG work. The authorized handoff uses
`codex/store-combined-release`; unrelated local edits are excluded from its commit.

- Red/green: combined preparation first rejected `--mode release`; combined
  execution first rejected the unknown command; workflow contract failed for all
  three missing combined-mode requirements. Each passed after implementation.
- A separate regression caught `no_update` incorrectly labelling the unconfirmed
  prior release as published; the result now labels its expected base instead.
- `go test ./scripts/storepublish ./scripts/msixstage` passes. The persistent
  fake-service cases cover both frozen releases/notes/base, no Microsoft access
  before approval, exact producer admission, publish-then-submit without new
  discovery, first release, waiting/failed/ambiguous/unknown prior status,
  changed digest/receipt/nested mode/base/projected receipt/tag/artifact, replay
  rejection and standalone submission scope. Existing standalone reconciliation
  coverage still passes and never selects the waiting newer version.
- `go test -race ./scripts/storepublish ./scripts/msixstage` passes; targeted
  `go vet` and builds pass. The projected-receipt guard was temporarily disabled:
  `reject_changed_projected_receipt` failed because the changed receipt was
  accepted. The exact original source was restored and the race suite passed.
- `make verify-build` passes (format, TUF, Qodana exclusions, generated tag/art
  checks, repository-wide vet and build). The native linker reports the existing
  duplicate `-lobjc` warning. Final `make fmt-check` and `git diff --check` pass.
- `make verify` stops at the required platform preflight: the selected Docker
  daemon reports `linux/aarch64`, not native Linux/amd64. Worker isolation remains
  enforced. The full repository race suite must run on native amd64 CI.
- GoLand 2026.2.1.1, build GO-262.9437.286, inspects an isolated copy with Go
  1.27.1 and Project Default plus `GoStructLayout` at weak warning. The five
  changed Go files have no confirmed defects after review. Scoped source
  suppressions document the existing product-name capitalization warnings and
  the nil-analysis false positive: the earlier reconcile admission explicitly
  rejects nil, covered by `TestStorePublishApproval`. Remaining spelling flags
  are reviewed existing API/tool/product identifiers or German fixture text
  (`frathe`, `Fyne`, `marketSpecificPricings`, `Beschreibung`, etc.).
  Evidence and the exact profile are retained under
  `/private/tmp/picfetch-store-combined-inspection/`. The workflow inspection
  completes without code findings (four existing product/tool spelling flags).
  Later headless passes (both publisher-only and complete scripts scope) report
  five false `GoUnusedFunction` findings: `prepareApproval`, `loadApproval`,
  `previewRelease`, `checkStore` and `drive` are directly invoked by `run` in
  `main.go` at lines 93, 99, 95, 97 and 107 respectively. All command-boundary tests
  exercise those callers. The temporary inspection profile
  `Project_Verified_CLI.xml` excludes only that inspection in `approval.go` and
  `lifecycle.go`, via the `StorePublisherCLI` scope; other inspections remain
  enabled and no unused-function suppressions were added to source. Raw findings,
  the exact scoped profile and its final rerun are retained in the evidence folder.
  The mitigated rerun completes with only the 11 reviewed spelling flags in the
  four publisher files. MSIX tests have 20 existing spelling flags; the workflow
  has four. All six inspected code files match the final checkout byte for byte.

The GitHub job still uses one required-reviewer environment approval and the
downloaded artifact's immutable ID plus SHA-256. Its summary names the prior
reconciliation and conditional new submission. Manual `check`, `preview`,
`reconcile` and `submit` remain available. No dependency closure changed.

| Task | Spawns budget/actual | Review rounds | Full suite | Notes |
| --- | --- | --- | --- | --- |
| Test map | 1/1 | Lead-owned | No | Existing test seams located |
| Combined mode | 0/0 | 1 plus inspection follow-up | Platform preflight blocked | Lead inline; affected package race suites pass |

## Consolidation onto the Spiral bugfix branch

On September 13, the delta from `codex/store-combined-release` at `4927666`
was applied to `bugfix/intel-not-rendering-the-easter-egg-spiral` at `f03d2b7`,
including its documentation and this plan. The preceding evidence describes the
source branch; its commit authorization does not apply to this consolidation.
The working tree was clean before application. No branch switch, reset, commit,
push, workflow dispatch or Store operation was performed.

- The source patch applied cleanly, and its reverse applicability was checked
  before this evidence was appended. The workflow, publisher code/tests, MSIX
  tests and Store guide match the source branch. Architecture and TODO additions
  preserve the target branch's newer entries.
- Direct comparisons against `HEAD` confirm the Spiral fix/tests/diagnosis,
  release and Store build workflows, version metadata and third-party notices
  remain unchanged. The existing local Windows executable retains SHA-256
  `6742f444eda1c2f2198264cebeb01772dcb18e3b1696b4571c41625a549ffe3f`.
- Windows `go test -count=1 ./scripts/storepublish ./scripts/msixstage` and
  targeted `go vet` pass. One bounded test-execution subagent ran these commands;
  the lead reviewed the results and all imported changes.
- An isolated native Linux/amd64 Docker checkout passes `make verify-build`
  and `go test -race -count=1 -v ./scripts/storepublish ./scripts/msixstage`.
  Both package suites complete without skips, including the shell-based
  environment-policy and cross-packaging guards. Final formatting and
  `git diff --check` pass.
- GoLand inspections, including weak warnings, complete for all five changed
  Go files and the workflow. Only duplicate-code warnings remain: an existing
  ignored source mirror under `.scratch/macos-x64-runtime/format-check/` matches
  unchanged code. The isolated verification checkout excludes that mirror, and
  the two test files retain their exact-file Qodana duplication exclusions.
  No confirmed code defects were found or source suppressions added.
- The full repository race gate remains unverified here: Docker exposes
  15.45 GiB, below its required 16 GiB. An approved live Store run also remains
  outstanding. Local logs and the applied patches are retained under
  `.scratch/store-ci-consolidation/`.
