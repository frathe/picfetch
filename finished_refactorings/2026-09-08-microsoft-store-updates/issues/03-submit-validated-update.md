# 03: Submit one validated update and report its Store state

**What to build:** One publisher invocation authenticates, reads the current product, revalidates a previewed release, uploads the original validated bundle, commits the update for certification and reports the submission ID and observed state. This complete path is demonstrable against a fake Store.

**Blocked by:** [02: Preview a validated release with generated Store notes](02-preview-validated-store-release.md)

**Status:** implemented; local verification passed

**Parent:** [Approved Microsoft Store updates](../spec.md).

**Coverage:** User stories US3, US4, US9, US10, US11, US12, US19, US20; AC3, AC6 (initial state reporting), AC7 (connectivity).

## Acceptance criteria

- [x] Provide an authenticated read-only check and live-snapshot preview using the supported MSIX client. Report missing/invalid access clearly, expose the current product state, and issue no submission/upload mutations in either read-only operation. Authentication and transport remain per invocation.
  Verification: `go test ./scripts/storepublish -run '^TestStorePublishReadOnly$'`.

- [x] Re-read and validate Store state before mutation. For an eligible update, copy the published submission, apply the previewed notes/package changes and automatic publication settings, upload the API-required archive containing byte-for-byte the validated bundle, and commit. Preserve unrelated metadata, actual locales and original artifact evidence.
  Verification: `go test ./scripts/storepublish -run '^TestStorePublishSubmission$'`.

- [x] Establish the durable receipt contract for release, source/run identity, artifact digest, note snapshot/digest, operation phase and submission ID. Record intent before mutations and the returned ID as soon as known. Serialize product mutations; refuse unknown pending drafts. The happy path and refusal cases must be externally observable without depending on internal helper structure.
  Verification: `go test ./scripts/storepublish -run '^TestStorePublishSubmission$'`.

- [x] Produce redacted summaries distinguishing accepted-for-processing, certification, publishing and Published. Never report upload/commit success as publication. Keep tokens, credentials and upload SAS URLs out of errors, stdout, summaries and retained evidence.
  Verification: `go test ./scripts/storepublish -run '^TestStorePublishStatus$'`.

- [x] Exercise the complete flow with fake authentication, Store and upload endpoints and a fresh command invocation. Confirm the original artifact remains available on submission failure. Preserve existing release-note and packaging behavior.
  Verification: `go test ./scripts/releasenotes ./scripts/msixstage ./scripts/storepublish`.

## Implementation notes

Ticket 01 is intentionally not a blocker: use fake endpoints and configurable credentials for development. The lead records supported authentication and client choices before integration with real account setup. Automatic production triggers stay unwired until ticket 06; this is implementation sequencing, superseded by the final per-rollout approval requirement. Interruption handling is completed by ticket 04 using the receipt contract established here.

The final user amendment requires REDACTED_REVIEWER approval of each specific validated release.
Use the command boundary and fake-service testing strategy from the spec.
Verification commands define the existing command/fake-service acceptance seam.

## Comments

2026-09-07: Created from the implementation spec. No implementation or live
verification has been performed for this ticket.

2026-09-08: Implemented through the `scripts/storepublish` command and fake
GitHub/Microsoft/blob services. `go test ./scripts/releasenotes ./scripts/msixstage ./scripts/storepublish` and `go test -race ./scripts/storepublish` passed.
Admission, metadata, artifact-digest and serialization guards were deliberately
broken, observed failing and restored; green focused reruns passed. No live Store
submission was made. Account setup and live publication belong to 01/06.

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
