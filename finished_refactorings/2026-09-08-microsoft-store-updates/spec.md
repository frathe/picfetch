# Approved Microsoft Store updates

Status: implemented locally; activation and live publication evidence pending
Updated: 2026-09-08
Implementation status: approval amendment implemented locally; protected environment
metadata verified, Microsoft read/submission access and live publication unverified.

## Problem Statement

PicFetch 1.0.2 is available in Microsoft Store, as confirmed by the maintainer.
CI already builds the Microsoft Store edition for x64 and ARM64, creates one
MSIX bundle, and validates it with the Windows App Certification Kit (WACK).
It stops at a downloadable GitHub Actions artifact. The maintainer must still
prepare Store change notes, upload the bundle, and submit the update.

This repeated work should disappear from normal releases. The existing product
must receive the exact validated package and accurate change notes, then become
available automatically after Microsoft certification.

## Solution

Extend the existing release process with automatic build validation and release-note
preparation, followed by **REDACTED_REVIEWER's explicit GitHub Required reviewers approval for
every rollout**. Approval covers a specific validated release/artifact and frozen
notes. The approved operation submits only that selection to product `9P0DM0KTH01K`
and requests immediate publication after certification. Recovery never selects a
newer release under an older approval.

### Confirmed requirement amendment

The final user instruction supersedes the earlier unattended-release requirement.
Keep the existing `microsoft-store` environment: branch main only, required reviewer
REDACTED_REVIEWER, self-review permitted, no administrator bypass, timer or custom rules.
Keep secrets there only. Do not weaken protection or copy secrets to avoid approval.

| Decision | Current contract |
| --- | --- |
| Release trigger | Successful stable-tag CI, packaging and WACK prepare that producing run automatically. |
| Approval | GitHub-only preparation freezes tag/commit/run/attempt/artifact/hash/notes/base/receipt identity before the protected job. That job downloads by immutable ID and verifies its hash. |
| Trusted code | Both jobs check out the workflow run's fixed main SHA. No code from a release tag runs with Store credentials. |
| GitHub release dependency | No dependency on GitHub Release publication. |
| Notes | Tagged existing notes, English across existing locales, skipped-release range included; freeze before approval. |
| Overlapping versions | An active receipt requires a manual approved reconcile first. Then a separate submit dispatch prepares the newest/requested waiting release for its own approval. |
| Publication | Microsoft continues immediate publication after successful certification without a runner. |
| Polling | No cron: protected secrets require approval even for reads. Manually dispatch and approve check/reconcile when needed. No automatic status/failure notification after the approved job exits. |
| Role | Keep the user-selected Developer role. Role-table permission and the API setup guide's Manager instruction differ; read access is not submission proof. |

Preparation needs no Microsoft secrets. It uses confirmed live 1.0.2 initially and
published receipts thereafter as its expected base. The protected job revalidates
the actual Store base before any mutation; stale state needs fresh preparation and
approval. The live metadata snapshot is read and preserved after approval; the
reviewable summary covers release/artifact/notes and intended update policy.

## User Stories

1. As a maintainer, I want my normal stable release to initiate the Store update, so that build and note preparation finish before I approve the rollout.
2. As a maintainer, I want existing tests and WACK checks to remain prerequisites, so that automation preserves release quality.
3. As a maintainer, I want to upload the exact bundle that passed validation, so that the tested and submitted builds agree.
4. As a Windows user, I want the Store update to contain the appropriate x64 or ARM64 package, so that it runs on my device.
5. As a maintainer, I want release notes reused automatically, so that I do not maintain a separate Store changelog.
6. As a Store customer, I want concise notes describing the update, so that I can understand what changed.
7. As a Store customer, I want notes to exclude internal engineering work and explicitly unrelated platform changes, so that the description is relevant.
8. As a maintainer, I want notes to fit the Store's length limit automatically, so that long releases do not require manual editing.
9. As a Store customer, I want the note version to match the delivered package, so that I am not shown stale release information.
10. As a maintainer, I want existing listing text, images, pricing and declarations preserved, so that an update does not reset the product setup.
11. As a Store customer, I want certified updates to become available automatically, so that publication does not wait for another maintainer action.
12. As a maintainer, I want upload, certification and publication reported separately, so that I can tell whether the update is actually live.
13. As a maintainer, I want a release arriving during certification to wait for its own approval, so that overlapping releases do not conflict or share authorization.
14. As a Store customer, I want notes to cover changes since the previous Store version when intermediate versions were skipped, so that the description remains complete in scope.
15. As a maintainer, I want retries to recognize completed work, so that rerunning a workflow does not create duplicate submissions.
16. As a maintainer, I want interrupted operations reconciled against Store state, so that recovery does not blindly repeat mutations.
17. As a maintainer, I want unrelated or manually created drafts preserved, so that automation does not destroy someone else's work.
18. As a maintainer, I want obsolete releases and mismatched artifacts rejected before upload, so that a delayed run cannot replace a newer version.
19. As a maintainer, I want Store credentials limited to publishing operations, so that builds and pull requests do not need publishing access.
20. As a maintainer, I want a read-only connectivity check and a preview, so that setup can be verified without changing the live product.
21. As a maintainer, I want useful failure details and retained artifacts, so that I can diagnose certification or infrastructure failures.
22. As a maintainer, I want Store outages to leave other release channels functional, so that one provider does not block every distribution.
23. As a maintainer, I want credential setup and recovery documented, so that exceptional maintenance is understandable without recreating the release process.

## Implementation Decisions

These are the implementation contract proposed by this spec. They extend the
existing packaging and release-note tooling; application UI behavior is unchanged.

1. **One publisher module.** Add a Store publishing command in repository tooling
   that owns candidate admission, note preparation, submission and reconciliation.
   Keep workflow orchestration thin. Use Microsoft's supported MSIX submission
   API; a pinned official CLI adapter is acceptable if it satisfies the same
   command-boundary tests. Supply authentication, transport and persistence per
   invocation, with no mutable global test seams.

2. **Admit an identified release.** Accept canonical stable versions newer than
   the currently published Store version. Require agreement among tag, source
   commit, public application version, package manifests and captured notes.
   The Store version is the public version with a trailing zero; the Fyne build
   counter is irrelevant. Check trusted repository and release-branch provenance.
   Branch builds, pull requests, forks and prereleases cannot submit. A manual
   retry must identify an eligible release and pass the same checks; a
   branch-based manual build remains build-only.

3. **Bind publication to build evidence.** Carry the producing run, commit,
   artifact identity and digest, contained package versions, and CI/WACK evidence
   into submission. Verify those bindings on retry or approved recovery.
   Upload the original bundle bytes without rebuilding or re-signing. Missing,
   expired or mismatched artifacts prevent submission and produce an actionable
   failure; another artifact with the same filename is not a substitute.

4. **Generate notes deterministically.** Use captured release notes and, when
   waiting versions were skipped, intervening release notes since the last
   published Store version. The source remains maintained change descriptions
   already converted into notes by the normal release command. Keep user-facing
   sections; omit Internal sections and entries explicitly scoped solely to
   Linux or macOS. Keep shared, mixed-platform and unmarked entries rather than
   guessing their scope. Convert Markdown to plain text without inventing claims.
   Order releases newest first and retain entry order within each release.
   Include the target version and a full-change link covering the Store version
   range. Reserve space for these, then include complete entries that fit within
   1,500 characters, conservatively also bounded to 1,500 UTF-16 code units.
   Never split a character or entry. If no entry fits or is relevant, publish the
   version and full-change link without a generic bug-fix claim. Missing or stale
   source notes cause a pre-submission failure. Save the generated note snapshot
   so a retry uses the same text.

5. **Preserve listing languages and metadata.** Discover actual published listing
   locales rather than inferring them from the package manifest. Apply generated
   English notes to each existing locale under this spec's language default;
   preserve the locale set. Copy the current published submission and change only
   package references, release-note fields, required submission bookkeeping and
   the publication settings selected above. Preserve localized descriptions,
   screenshots, pricing, availability, age ratings and capability explanations.
   Ensure platform-specific note overrides cannot keep displaying an older
   version. Verify intended differences before upload and commit: the API sends
   complete submission data, not a sparse patch.

6. **Authenticate after initial setup.** Discover existing tenant/application
   configuration before creating credentials. Use a dedicated Partner Center
   application with Microsoft's required submission permissions and protected CI
   credential storage with required reviewers for every protected operation. Keep the
   chosen Developer role; do not silently elevate to Manager. Select
   authentication supported by the chosen client and account; OIDC support has
   not been established. Document expiry and renewal when using a secret or
   certificate. Keep tokens, credentials and upload SAS URLs out of logs,
   summaries and retained evidence. Refresh expiring access tokens during long
   operations. Credential failure must leave packaging artifacts available.

7. **Submit and distinguish outcomes.** Create an update from the published
   submission, prepare package references and notes, upload the API's required
   archive containing the exact validated bundle, then commit for certification
   with automatic publication enabled. Record the submission ID as soon as known.
   Distinguish accepted-for-processing, certification-in-progress, publishing,
   published and terminal failure. Upload or commit success never means the
   update is already live.

8. **Serialize and resume.** Allow one mutating operation for this product at a
   time; new releases must not cancel active uploads or submissions. Keep a
   durable association among release, artifact digest, note digest and submission
   ID. Use bounded checks and manually approved reconciliation without keeping a
   runner alive through certification. Reconcile only the selected receipt. After
   it reaches a terminal state, prepare the newest eligible waiting release in a
   separate invocation that needs its own approval. A workflow
   concurrency group alone neither tracks Store certification nor guarantees
   durable waiting work. Storage and artifact retention must support delayed
   submissions and restarts.

9. **Reconcile before retrying.** Read current Store state and recorded evidence
   before repeating a mutation. Resume known pending work, observe a submission
   already processing, and treat the exact already-published release as a
   successful no-op. Reject a digest mismatch for the same release, an older
   version, or a moved tag. Retry transient failures with bounded backoff and
   honor rate-limit delays. Reconcile ambiguous create/commit responses through
   reads; report a conflict when identity cannot be established. Preserve unknown
   or manually created drafts. Certification rejection is a reported terminal
   failure, not a reason to resubmit identical bits forever. A newer version may
   proceed only when Store state permits it; clearing a conflicting draft is
   outside automatic recovery.

10. **Expose useful operations and evidence.** Provide read-only connectivity,
    preview, GitHub-only prepare, approved submit and approved reconcile operations. Connectivity verifies product and
    account access. Preview reports the selected release, generated notes and
    intended field changes without creating, uploading or committing anything.
    Summaries identify version, producing run, bundle digest, submission ID and
    observed state. Record retryable waits, terminal failures and conflicts
    explicitly. Document initial configuration, credential renewal, retry,
    certification inspection and the existing manual recovery procedure.

## Testing Decisions

### Primary seam and prior art

Test the publishing command as its caller would: provide a tagged release
fixture, validated bundle evidence and a fake external service; invoke the
command; assert its outputs, uploaded bytes and externally visible submission
changes. One command boundary covers generation and lifecycle behavior. A local
HTTP server supplies token, Store and upload responses. Temporary storage
supplies durable state and artifacts, with controllable time and delays per
invocation. Tests never contact the real Store.

Test external behavior rather than helper structure. Request counts matter when
proving that invalid candidates cause no mutation or reruns create no duplicate
submission. Restart tests use a fresh invocation with persisted evidence.
Exercise competing candidates and failure after a remote side effect but before
local recording. Workflow guards additionally verify dependency and credential
boundaries, which command tests cannot establish.

Prior art: release-note command tests use temporary repository inputs and verify
written outputs; MSIX staging tests cover versions and packaging workflow
contracts; WinGet guards cover release provenance; updater HTTP tests use local
service fixtures and injected transport. Extend those patterns rather than
adding UI seams or requiring desktop integration.

### Acceptance criteria

The test names below define the command-level acceptance contract. Preserve this command-level coverage if the
implementation plan selects a different module name.

| ID | Observable acceptance condition | Verification command |
| --- | --- | --- |
| AC1 | Stable releases with matching provenance/version/evidence are admitted; branch, fork, prerelease, stale-note, moved-tag, older-version and mismatched-artifact fixtures cause no Store mutation. | `go test ./scripts/storepublish -run '^TestStorePublishAdmission$'` |
| AC2 | Notes use the captured Store release range, omit Internal and explicitly exclusive non-Windows entries, preserve mixed entries, fit the length limit, handle Unicode and empty/oversized entries, and produce reproducible text for existing English/German listing fixtures. | `go test ./scripts/storepublish -run '^TestStorePublishNotes$'` |
| AC3 | The uploaded archive contains the exact validated bundle; the resulting submission has the expected package and notes, preserves unrelated metadata and selects automatic publication. | `go test ./scripts/storepublish -run '^TestStorePublishSubmission$'` |
| AC4 | Retries and restarts resume known work, handle lost create/upload/commit responses, avoid duplicate publication and refuse conflicting identity or unowned drafts. | `go test ./scripts/storepublish -run '^TestStorePublishRecovery$'` |
| AC5 | Competing candidates cannot mutate concurrently; reconciliation never submits another release; a separate prepared and approved submission selects the newest waiting candidate and covers skipped notes without a new tag event. | `go test ./scripts/storepublish -run '^TestStorePublishReconcile$'` |
| AC6 | Processing, certification, publication, timeout, token expiry, throttling, rejection and unknown states produce accurate results, bounded retries and redacted diagnostics. | `go test ./scripts/storepublish -run '^TestStorePublishStatus$'` |
| AC7 | Connectivity check and preview issue no mutating Store/upload requests; preview exposes notes and intended changes, and missing access is reported clearly. | `go test ./scripts/storepublish -run '^TestStorePublishReadOnly$'` and `go test ./scripts/storepublish -run '^TestStorePublishSubmission$'` |
| AC8 | Workflow guards verify trusted release admission, successful packaging/WACK prerequisites, artifact binding, credential scope, required reviewer policy, immutable preapproval artifact selection, no scheduled secret-bearing runs, and retained artifacts on publishing failure. | `go test ./scripts/msixstage -run '^TestStoreWorkflowPublishingContract$'` |
| AC9 | Existing packaging and release-note behavior remains covered alongside new tooling. | `go test ./scripts/releasenotes ./scripts/msixstage ./scripts/storepublish` |
| AC10 | The implementation passes repository verification: formatting, TUF/Qodana checks, vet, build and Linux/amd64 Docker race tests. | `make verify` |
| AC11 | Preparation reads GitHub only and freezes exact artifacts/notes before approval; mutations require that file/hash, reject changed state, and never rediscover a replacement release. | `go test ./scripts/storepublish -run '^TestStorePublishApproval$'` |

For guards, deliberately violate the protected behavior, observe the expected
failure, restore it, and rerun. Add new test files to Qodana's exact-path
exclusions. Update the architecture map if a tooling package is introduced.
No new UI tests or golden changes are expected.

Local fixtures cannot prove remote environment settings or Microsoft acceptance.
Before production enablement, inspect the actual GitHub environment to establish
that routine releases require REDACTED_REVIEWER approval, and after the workflow lands on main
approve its read-only `check` job with protected credentials. A read-only success
proves access to product data, not permission to upload or submit.
Record product access and current published/pending submission state. The first
subsequent normal stable release supplies live acceptance evidence: its artifact
digest, package version, submission ID and eventually Store-reported Published
state. Report each stage only when observed; do not resubmit already-live 1.0.2
just to test automation.

## Out of Scope

- Rebuilding Windows packaging or changing Microsoft Store edition application behavior.
- Replacing the normal release command, maintained change descriptions or GitHub/WinGet distribution.
- Inferring summaries from commits, adding an LLM dependency or automatic translation in this iteration.
- Changing product identity, price, descriptions, screenshots, locales, age ratings or capability explanations.
- Gradual rollouts, package flights, scheduled holds or forced customer updates.
- Automatically deleting unrelated drafts, fixing certification rejection or rolling customers back to a lower package version.
- Bypassing Microsoft certification, account administration or provider outages.

## Further Notes

### Repository navigation and sources

- Current build workflow: `.github/workflows/microsoft-store.yml`.
- Existing note generation: `scripts/releasenotes`, the `release` Make target,
  and `.github/release-notes.md` in the release commit.
- Version and packaging guards: `scripts/msixstage`.
- Relevant tests: `TestRun_WriteAndClearDone`,
  `TestStoreVersion_UsesSemanticVersionAndStoreReservedRevision`,
  `TestMicrosoftStoreWorkflowAndBuildTarget`,
  `TestWorkflowGatesUntrustedReleaseTag`, and the updater's `TestCheck`.
- Operational guidance and listing content: `docs/microsoft-store.md` and
  `packaging/microsoft-store/listing.md`.
- Process, architecture and glossary: `AGENTS.md`,
  `.agents/skills/improved_sdd_tdd_cycle.md`, `ARCHITECTURE.md` and `CONTEXT.md`.

Microsoft sources checked during discovery:

- [Manage MSIX app submissions](https://learn.microsoft.com/en-us/windows/uwp/monetize/manage-app-submissions):
  copying published submissions, archive upload, commit, per-locale release notes,
  publication modes and status values. Microsoft warns that editing an API-created
  submission through Partner Center can prevent subsequent API updates or commit.
- [Store listing fields](https://learn.microsoft.com/en-us/windows/apps/publish/publish-your-app/msix/add-and-edit-store-listing-info):
  separate listing languages and the 1,500-character What's new limit.
- [Official Store CLI commands](https://learn.microsoft.com/en-us/windows/apps/publish/msstore-dev-cli/commands):
  secret/certificate authentication and submission operations; MSIX updates send
  complete submission JSON. This source did not establish GitHub OIDC support.

### External prerequisites and honest limits

The maintainer's report establishes initial publication, the linked Developer
application and key rotation. GitHub environment protection and secret names were
verified directly. Live Microsoft API access, current listing locales, pending
drafts and credential expiry were not inspected. The read-only CI check requires
the trusted publisher on main and the user's environment approval. Keep actual
submission permission and the next approved publication as separate live criteria.

The language default is English across existing listings. It prevents stale
version notes without adding a translation service, but does not provide localized
German change notes. This and the other defaults were selected during synthesis
and remain reviewable policy choices.

Normal releases require approval of the prepared artifact and notes. Credential renewal,
certification rejection, unowned drafts and missing retained artifacts can still
require maintenance. Passing local tests does not establish live certification
or publication.

### Handoff

This is the amended implementation contract. Changes remain uncommitted for review.
No commit, push, tag, release, Store submission, workflow dispatch or live protection
change is authorized by this local task. The first live check needs the trusted
workflow on main and the user's approval; never resubmit already-live 1.0.2 as a test.

## Implementation record — 2026-09-08

Implementation uses the direct MSIX REST API with client-secret authentication,
a trusted-main publisher workflow and append-only GitHub deployment receipts.
The initial accepted bundle has outer version `2026.905.1829.0`, distinct from
inner app version `1.0.2.0`; the bootstrap mapping and subsequent receipt bindings
preserve both. The command fails on an unmapped published version instead of
silently concluding no update is available. This resolves a real-artifact finding
without changing the user's public version or rebuilding the accepted release.

Local command/race tests and workflow guards pass; The combined `make verify` run recorded a Docker OOM event; all UI race shards and all non-UI race-test packages passed across the isolated reruns. The original combined command did not exit successfully.
Environment metadata and secret names were verified on 2026-09-08. The user
confirmed the linked Developer application and replacement key. The publisher is
not installed on main. CI credential validity/read access, actual submission
permission and the next ordinary approved release's publication remain open. See the ticket index and implementation plan for evidence.
