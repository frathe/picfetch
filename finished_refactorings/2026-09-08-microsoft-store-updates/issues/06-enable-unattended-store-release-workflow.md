# 06: Enable approved CI updates and verify the first publication

**What to build:** The maintainer's next ordinary stable release builds, generates notes, uploads, submits, tracks certification and publishes through the existing Store workflow after approval of the exact prepared artifact and notes, without manual transfer. Approved reconciliation records publication; a separate approved submit handles waiting work, and the full path has recorded live evidence.

**Blocked by:** [01: Configure Store access and prove read-only connectivity](01-configure-store-access.md); [05: Track publication and approve the next waiting release](05-reconcile-publication-and-waiting-releases.md)

**Status:** implementation complete; activation and live evidence pending

**Parent:** [Approved Microsoft Store updates](../spec.md).

**Coverage:** User stories US1, US2, US3, US11, US12, US13, US19, US20, US21, US22, US23; AC8, AC9, AC10, Live publication evidence.

## Acceptance criteria

- [x] Connect the completed publisher to successful trusted stable-tag builds after existing CI, packaging and WACK. Bind the original artifact/run/commit and contained package versions. Keep branch/manual build-only, fork and prerelease routes unable to publish. A manual retry names an eligible release and passes the same admission. GitHub Release publication is not a dependency.
  Verification: `go test ./scripts/msixstage -run '^TestStoreWorkflowPublishingContract$'`.

- [x] Wire manual approved reconciliation, product-wide mutation serialization without canceling active work, protected credential delivery, durable receipts and artifact retention. Publishing or credential failure leaves the validated bundle/report downloadable and other distribution workflows functional. Repository settings require REDACTED_REVIEWER approval of each protected operation; no cron creates periodic requests.
  Verification: `go test ./scripts/msixstage -run '^TestStoreWorkflowPublishingContract$'` and `go test ./scripts/storepublish -run '^TestStorePublishReconcile$'`.

- [ ] Integrate the exact identity/environment from ticket 01. Inspect remote protection rules and run the finished publisher's check from CI, proving that configured credentials can read the intended live product and eligible runs require the configured REDACTED_REVIEWER reviewer. Do not infer remote settings solely from workflow YAML.
  Verification: `go run ./scripts/storepublish check` in the protected CI context; inspect `gh api repos/frathe/picfetch/environments/microsoft-store --jq '{name, protection_rules, deployment_branch_policy}'` and retain the corresponding diagnostic run result.

- [ ] Finish release/recovery documentation with generated-note policy, submission status meanings, credential renewal, safe retries, certification failure inspection and manual fallback. Run focused regression tests, negatively verify release guards, and complete the full repository gate once after integration. Include architecture and Qodana registration changes introduced by preceding tickets.
  Verification: `go test ./scripts/releasenotes ./scripts/msixstage ./scripts/storepublish`, deliberate guard-failure evidence followed by green reruns, and `make verify`.

- [ ] Observe the first subsequent normally authorized stable release. Retain producing-run identity, original artifact digest, submitted package version, generated-note evidence, submission ID, processing/certification status and eventual Store-reported Published state. Keep this criterion open while certification or the next release event is pending; never call upload success publication.
  Verification: `gh run view <release-run-id> --repo frathe/picfetch --json conclusion,jobs,url` and `go run ./scripts/storepublish check` in protected CI; compare the resulting receipt and Store state to the selected release.

## Implementation notes

The lead owns integration review, the final gate and evaluation of live evidence. Do not manufacture a release or resubmit already-live 1.0.2 for acceptance. Observe the next normal release with the required approval of its prepared artifact and notes. A missing release event, certification delay or rejected submission must remain an explicit incomplete live criterion, not a claim that publication passed. Account setup and offline implementation can proceed in parallel, but this ticket needs both.

The final user amendment requires REDACTED_REVIEWER approval of each specific validated release.
Use the command boundary and fake-service testing strategy from the spec.
Verification commands define the existing command/fake-service acceptance seam.

## Comments

2026-09-07: Created from the implementation spec. No implementation or live
verification has been performed for this ticket.

2026-09-08: Producer and trusted-main publisher workflows are implemented.
`go test ./scripts/msixstage -run '^TestStoreWorkflowPublishingContract$'` passes;
YAML parsing succeeds. The cancellation and trusted-checkout guards failed under
deliberate violations and passed after restoration. The Windows command cross-build
and focused publisher race suite pass. The combined `make verify` run recorded a Docker OOM event; all UI race shards and all non-UI race-test packages passed across the isolated reruns. The original combined command did not exit successfully. Environment
creation was rejected by automatic approval review and credentials are unavailable;
no protected CI read or first automated Store publication is claimed.

Final local result: 52 non-UI packages passed with race checking, then the
clean Linux `scripts/testshards` race rerun passed (13.653s). All three UI shards
also passed. The original combined `make verify` still exited unsuccessfully
following a Docker OOM event; no single-run green result is claimed. The full-gate
checkbox stays open for that exact command, alongside the live activation checks.

2026-09-08 approval amendment (supersedes earlier unattended/setup comments):
The final user requirement is approval of every specific release via GitHub Required
reviewers. GitHub metadata now verifies main-only, REDACTED_REVIEWER reviewer, self-review
allowed, no bypass/timer/custom rules, and all three secret names. The user confirmed
the linked Developer application and replacement key. Credentials remain exclusively
in GitHub Secrets. The publisher is not yet on main, so the approved CI read check,
actual submission permission and first ordinary approved publication remain open.
Preparation freezes artifact/notes/base/receipt before the protected job; submit
uses only that selection and reconcile never admits a newer release. Cron is removed;
status/recovery require manual approved dispatch. Local approval and workflow tests
cover the amendment; see the plan for verification. No Store update was submitted.
