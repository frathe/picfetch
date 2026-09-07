# 03: Submit one validated update and report its Store state

**What to build:** One publisher invocation authenticates, reads the current product, revalidates a previewed release, uploads the original validated bundle, commits the update for certification and reports the submission ID and observed state. This complete path is demonstrable against a fake Store.

**Blocked by:** [02: Preview a validated release with generated Store notes](02-preview-validated-store-release.md)

**Status:** ready-for-agent

**Parent:** [Automatic Microsoft Store updates](../spec.md).

**Coverage:** User stories US3, US4, US9, US10, US11, US12, US19, US20; AC3, AC6 (initial state reporting), AC7 (connectivity).

## Acceptance criteria

- [ ] Provide an authenticated read-only check and live-snapshot preview using the supported MSIX client. Report missing/invalid access clearly, expose the current product state, and issue no submission/upload mutations in either read-only operation. Authentication and transport remain per invocation.
  Verification: `go test ./scripts/storepublish -run '^TestStorePublishReadOnly$'`.

- [ ] Re-read and validate Store state before mutation. For an eligible update, copy the published submission, apply the previewed notes/package changes and automatic publication settings, upload the API-required archive containing byte-for-byte the validated bundle, and commit. Preserve unrelated metadata, actual locales and original artifact evidence.
  Verification: `go test ./scripts/storepublish -run '^TestStorePublishSubmission$'`.

- [ ] Establish the durable receipt contract for release, source/run identity, artifact digest, note snapshot/digest, operation phase and submission ID. Record intent before mutations and the returned ID as soon as known. Serialize product mutations; refuse unknown pending drafts. The happy path and refusal cases must be externally observable without depending on internal helper structure.
  Verification: `go test ./scripts/storepublish -run '^TestStorePublishSubmission$'`.

- [ ] Produce redacted summaries distinguishing accepted-for-processing, certification, publishing and Published. Never report upload/commit success as publication. Keep tokens, credentials and upload SAS URLs out of errors, stdout, summaries and retained evidence.
  Verification: `go test ./scripts/storepublish -run '^TestStorePublishStatus`.`.

- [ ] Exercise the complete flow with fake authentication, Store and upload endpoints and a fresh command invocation. Confirm the original artifact remains available on submission failure. Preserve existing release-note and packaging behavior.
  Verification: `go test ./scripts/releasenotes ./scripts/msixstage ./scripts/storepublish`.

## Implementation notes

Ticket 01 is intentionally not a blocker: use fake endpoints and configurable credentials for development. The lead records supported authentication and client choices before integration with real account setup. Automatic production triggers stay unwired until ticket 06; this is implementation sequencing, not a recurring human approval. Interruption handling is completed by ticket 04 using the receipt contract established here.

The spec's proposed defaults remain defaults, not newly confirmed user decisions.
Use the command boundary and fake-service testing strategy from the spec.
Verification commands referring to publisher tests or commands describe future
implementation requirements; they have not been executed as part of ticket creation.

## Comments

2026-09-07: Created from the implementation spec. No implementation or live
verification has been performed for this ticket.
