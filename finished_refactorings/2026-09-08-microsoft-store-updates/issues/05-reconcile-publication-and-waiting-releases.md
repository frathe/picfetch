# 05: Track publication and approve the next waiting release

**What to build:** After the initial submission run exits, a later reconciliation invocation observes Microsoft certification/publication and stops at the selected receipt. A separate submit dispatch prepares the newest eligible waiting release for its own approval after the active receipt is finished. No additional tag event is needed to make progress.

**Blocked by:** [04: Recover interrupted submissions without creating duplicates](04-recover-interrupted-submissions.md)

**Status:** implemented; local verification passed

**Parent:** [Approved Microsoft Store updates](../spec.md).

**Coverage:** User stories US12, US13, US14, US15, US17, US21; AC5, AC6 (certification and terminal states).

## Acceptance criteria

- [x] Bound each check and persist state across invocations. Correctly report processing, certification, publishing, Published, retryable waits and terminal failure without keeping a runner alive for the certification period. Unexpected provider states remain visible and cannot be mistaken for success.
  Verification: `go test ./scripts/storepublish -run '^TestStorePublishStatus$'`.

- [x] While one release is active, discover and retain waiting eligible candidates without canceling its work. When Store state permits another submission, a separate invocation selects the numerically newest eligible waiting version before approval, then revalidates its exact run/artifact evidence and submits it after approval. Missing, expired or mismatched artifacts must produce a reported failure rather than substitution.
  Verification: `go test ./scripts/storepublish -run '^TestStorePublishReconcile$'`.

- [x] If intermediate versions are skipped, generate notes covering the range from the actual last published Store version to the selected candidate. Freeze the new note snapshot before approval. Exercise at least three waiting versions, delayed build completion and no further release events.
  Verification: `go test ./scripts/storepublish -run '^TestStorePublishReconcile$'` and `go test ./scripts/storepublish -run '^TestStorePublishNotes$'`.

- [x] A fresh manually approved or competing invocation neither duplicates the active submission nor loses waiting work. Published releases are no-ops, rejected identical releases are not retried forever, and unowned/conflicting drafts stop mutation while preserving useful status. Verify durable state and retained artifacts across ordinary run completion and restarts.
  Verification: `go test -race ./scripts/storepublish -run '^TestStorePublishReconcile$'`.

## Implementation notes

This ticket supplies a complete reconcile command and durable waiting-work discovery; ticket 06 supplies manual approved operations. There is no periodic polling with protected secrets. A GitHub concurrency group alone is not durable Store state. Store ownership and crash recovery come from ticket 04. Record the selected persistence and retention mechanism in the implementation plan, including how it survives ephemeral runners.

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
