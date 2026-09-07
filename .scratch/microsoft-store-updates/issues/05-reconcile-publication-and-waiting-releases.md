# 05: Track publication and advance the newest waiting release

**What to build:** After the initial submission run exits, a later reconciliation invocation observes Microsoft certification/publication and automatically selects the newest eligible waiting release when the active submission is finished. No additional tag event is needed to make progress.

**Blocked by:** [04: Recover interrupted submissions without creating duplicates](04-recover-interrupted-submissions.md)

**Status:** ready-for-agent

**Parent:** [Automatic Microsoft Store updates](../spec.md).

**Coverage:** User stories US12, US13, US14, US15, US17, US21; AC5, AC6 (certification and terminal states).

## Acceptance criteria

- [ ] Bound each check and persist state across invocations. Correctly report processing, certification, publishing, Published, retryable waits and terminal failure without keeping a runner alive for the certification period. Unexpected provider states remain visible and cannot be mistaken for success.
  Verification: `go test ./scripts/storepublish -run '^TestStorePublishStatus$'`.

- [ ] While one release is active, discover and retain waiting eligible candidates without canceling its work. When Store state permits another submission, select the numerically newest eligible waiting version, revalidate its original run/artifact evidence and submit it. Missing, expired or mismatched artifacts must produce a reported failure rather than substitution.
  Verification: `go test ./scripts/storepublish -run '^TestStorePublishReconcile$'`.

- [ ] If intermediate versions are skipped, generate notes covering the range from the actual last published Store version to the selected candidate. Freeze the new note snapshot once its submission begins. Exercise at least three waiting versions, delayed build completion and no further release events.
  Verification: `go test ./scripts/storepublish -run '^TestStorePublishReconcile$'` and `go test ./scripts/storepublish -run '^TestStorePublishNotes$'`.

- [ ] A fresh scheduled or competing invocation neither duplicates the active submission nor loses waiting work. Published releases are no-ops, rejected identical releases are not retried forever, and unowned/conflicting drafts stop mutation while preserving useful status. Verify durable state and retained artifacts across ordinary run completion and restarts.
  Verification: `go test -race ./scripts/storepublish -run '^TestStorePublishReconcile$'`.

## Implementation notes

This ticket supplies a complete reconcile command and durable waiting-work discovery; ticket 06 supplies its production schedule. A GitHub concurrency group alone is not durable Store state. Store ownership and crash recovery come from ticket 04. Record the selected persistence and retention mechanism in the implementation plan, including how it survives ephemeral runners.

The spec's proposed defaults remain defaults, not newly confirmed user decisions.
Use the command boundary and fake-service testing strategy from the spec.
Verification commands referring to publisher tests or commands describe future
implementation requirements; they have not been executed as part of ticket creation.

## Comments

2026-09-07: Created from the implementation spec. No implementation or live
verification has been performed for this ticket.
