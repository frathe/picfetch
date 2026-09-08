# Approved Microsoft Store updates

Route: Deep. Status: implemented locally; live activation pending. No git commit or new release is part of this run.

Current policy: the final user amendment at the end of this plan supersedes the
original unattended design and setup status below. Required reviewer frathe approves
every prepared release. Environment metadata is now verified; live access is pending.

## Original implementation history (before final approval requirement)

Implement the six local tickets in `.scratch/microsoft-store-updates/issues/`.
The source spec's defaults apply: stable tags, existing notes, English notes in
existing locales, newest waiting release and immediate publication after certification.
Existing unrelated UI/Trane edits are preserved, including shared documentation changes.

## Contracts and decisions

- One Go command package, `scripts/storepublish`, with `record`, `preview`, `check`,
  `submit`, `reconcile` operations. Tests invoke the command with per-call external
  HTTP, clock/wait, git-process and filesystem inputs. No UI integration or new dependency.
- Microsoft MSIX REST API, documented client-secret OAuth resource flow. Secret
  names: `MSSTORE_TENANT_ID`, `MSSTORE_CLIENT_ID`, `MSSTORE_CLIENT_SECRET` in the
  `microsoft-store` environment. Existing GitHub configuration has none of these.
- The existing producer records source/run identity, inner MSIX versions, bundle
  digest and WACK report digest with its validated bundle. Consumers verify the
  producing workflow/run via GitHub and resolve the tag against trusted main.
- A separate publisher workflow checks out trusted main, follows completed Store
  builds, and reconciles on a schedule. It owns one fixed concurrency group with
  cancellation disabled. Branch/PR builds never become eligible candidates.
- Use GitHub deployment payloads as a durable append-only receipt journal, with
  `auto_merge: false` and no repository writes. Save intent before each remote
  mutation. Workflow serialization is the distributed admission boundary; local
  command invocations also use an exclusive filesystem claim.
- Receipt contract: schema, release tag/commit/version, producing run/artifact,
  bundle/notes digests, frozen notes/base Store version, submission ID, phase and
  preserved-metadata digest. No SAS URL, credentials or token is persisted.
- Discover waiting releases from successful Store producer runs and immutable
  artifacts, not from GitHub Releases or an in-memory queue. Re-read tag/ref and
  provenance at every resumption. Retention is explicit; expiry is a visible error.
- A lost create response has no documented ownership/idempotency token. An
  ambiguous new pending submission is a conflict requiring diagnosis, never an
  excuse to adopt/delete a draft or retry creation blindly. A commit ambiguity
  is reconciled using the already recorded submission ID and provider status.
- New source files: command/config and record/admission, notes/metadata,
  Microsoft client, lifecycle/journal, GitHub discovery/persistence, and command
  tests. Extend existing MSIX workflow tests; register new tests in Qodana and
  the command in ARCHITECTURE. Operational docs explain setup and recovery.

## Tasks, ownership and verification

| Task | Files / contract | Owner | Depends | Verify | Spawn budget |
| --- | --- | --- | --- | --- | --- |
| 01 Account discovery/setup | GitHub environment and Store operations docs | Lead | None | credential-name/rule reads; future live `check` | 0 |
| 02 Preview | new command/admission/notes plus command tests; build evidence contract above | Lead | None | spec AC1, AC2, AC7 | 0 |
| 03 Submission | Microsoft client/lifecycle and fake HTTP command tests | Lead | 02 | spec AC3, AC6, AC7 | 0 |
| 04 Recovery | journal and command restart/failure tests | Lead | 03 | spec AC4, AC6, focused race | 0 |
| 05 Reconcile | GitHub discovery/journal adapter and command tests | Lead | 04 | spec AC5, AC6 | up to 1 bounded adapter implementation after contracts/tests are fixed |
| 06 CI and handoff | producer/publisher workflows, existing MSIX guards, docs, Qodana, architecture | Lead | 01 setup + 05 code | focused packages then `make verify`; actual live evidence separately | 0 |

Task graph: 02 -> 03 -> 04 -> 05 -> 06; account setup 01 is independent until 06.
Missing Entra access cannot block offline code work. The final live-publication
criterion remains open until the next ordinary authorized stable release is observed.

## Delegation and cost ledger

GitHub adapter delegation gate: G1 bounded to one file and three methods; G2
`go test ./scripts/storepublish -run TestStorePublishGitHub` proves command input
comes from the expected trusted artifact (lead-authored fixture, red observed);
G3 only `github.go`, no shared edits; G4 needs API/receipt contracts, not the full
Microsoft lifecycle; G5 lead has not implemented the adapter. S/W: not a mechanical
substitution and no implementation is supplied. Lead implements the command and
Microsoft transport concurrently, owns integration and reviews the final adapter.

One factual scout researches current Microsoft API contracts while the lead designs
and writes code. Read-only source research is independent of implementation and
does not delegate architecture, spec, review or user-visible text. Any adapter
delegation will document G1-G5 before dispatch; no agent edits shared files.

| Work | Spawns budget / actual | Review | Full suite |
| --- | --- | --- | --- |
| Microsoft API facts | 1 / 1 (existing scout; one follow-up on real bundle versions) | Lead checked primary sources and actual artifact | no |
| Implementation | 1 / 1 (bounded GitHub adapter) | Lead reviewed, integrated and fixed inline | no |
| Final gate | 0 / 0 | Lead | 1 full attempt (Docker OOM), 1 isolated non-UI rerun, 1 clean testshards rerun after diagnostic override interference |

## Checklist

- [x] Read spec, tickets, architecture and working agreement.
- [x] Select command/fake-service seam already specified by the user-requested spec.
- [x] Read GitHub credential names and environment rules; Store credentials absent.
- [x] Write and observe failing command tests before each implementation slice.
- [x] Complete preview, submission, recovery and reconciliation.
- [x] Verify guards negatively, then restore and re-run.
- [x] Review final diff, register tests and update package map/operations docs.
- [x] Run focused regression checks and `make verify` once; record OOM and isolated reruns honestly.
- [x] Update tickets with actual evidence and remaining external prerequisites.

## Evidence and limits

GitHub API reads succeeded on 2026-09-07: repository secrets are QODANA_TOKEN and
WINGET_TOKEN; signing environment has Certum-only secrets; no Store environment
or Microsoft credentials exist. Neither Azure CLI nor msstore is installed here.
Docker 29.6.1 and Go 1.27.1 are available. Partner Center account access and first
automated Store publication are unverified. The user has been asked whether an
Entra application already exists, without requesting secret values in chat.

### Lead review and verification evidence (2026-09-08)

- Focused `go test ./scripts/releasenotes ./scripts/msixstage ./scripts/storepublish`
  passed after integration and guard restoration. Publisher `-race` passed.
- Observed red assertions for the missing preview path, missing note range,
  accumulating poll receipts, metadata drift before upload, shortened Retry-After,
  oldest-first discovery, missing workflow integration and treating a bundle
  timestamp as the application version. Each was fixed inline and rerun green.
- Eight deliberate mutations proved provenance, bundle digest, WACK digest,
  serialization, exclusive claim, final metadata validation, disabled cancellation
  and trusted-main checkout guards. Every mutation was restored, followed by a
  successful focused regression run.
- Both workflows parse as YAML. Wizard `bash -n` passes; its shared library is
  unchanged, all three values map to the workflow's environment secrets, and it
  refuses to create/change environment rules. No end-to-end wizard run occurred.
- The command cross-builds with `GOOS=windows GOARCH=amd64 CGO_ENABLED=0`.
- Downloaded the accepted 1.0.2 artifact from Store run `33983485123`, then ran
  the actual new recorder successfully against its bundle, WACK report and tagged
  source files. No upload or Store mutation occurred. New Windows SDK/WACK runs
  remain CI evidence after this implementation is merged.
- The combined `make verify` run recorded a Docker OOM event; all UI race shards and all non-UI race-test packages passed across the isolated reruns. The original combined command did not exit successfully. Formatting, TUF/Qodana checks, host vet and build
  have passed; All UI race shards and all non-UI packages passed across the isolated reruns described below.

Review changes: preserve full tagged provenance on receipt resume; bound command
runtime and respect cancellation/retry delays; do not repeat ambiguous PUT/create/
commit; recheck metadata before upload and commit; avoid one receipt per poll;
select newest artifacts before looking at obsolete builds; retain journal history
without the artifact-discovery page cap. All review and fixes remained with Lead.

The real MakeAppx bundle revealed outer version `2026.905.1829.0`, distinct from
inner `1.0.2.0`. The initial mapping is pinned to this user-confirmed accepted
release; future bundle and public versions bind through the submission receipt.
Unknown published mappings fail for inspection. Both versions must advance and
the original producer versioning stays intact. Deployment receipts use ref `main`
for the restricted environment; exact source SHA remains in their payload.

### External activation boundaries

Automatic approval review rejected the attempted creation of GitHub environment
`microsoft-store`: changing deployment approval/ref controls needs approval of
that exact scope. Nothing was created. The prepared request has no required
reviewer/wait timer and permits only main through its separate branch rule.
Do not bypass the rejection. Entra credentials/account access are unavailable.
The setup wizard only runs after the environment exists and verifies read-only
access before installing secrets. Ticket 01 and ticket 06's CI/live-publication
criteria remain open. No commit, tag, release or Store submission was created.

Final review added a read-only workflow configuration job before the job that
references the environment. It verifies environment existence, exact main-only
branch policy and no reviewer/timer/custom approval rules. This prevents GitHub
from implicitly creating an unrestricted environment. Its new guard failed before
implementation and passed afterward; final YAML parsing and format checks passed.
No Microsoft transport or lifecycle code changed during full-suite verification.

The initial full run recorded a Docker OOM event (container
`4f186cabbb7c7f7f3a16ad4999ef6a84bb0d6bc172402a0765e1ba8c8c22463e`).
All three UI partitions passed (302.142s, 306.126s, 276.510s). An isolated Linux
non-UI rerun passed 52 packages, including storepublish. Its only failure was the
existing testshards Make-contract test inheriting the diagnostic command-line
TEST_CAPTURE override; the actual output shows /evidence/non-ui.json where the
fixture requires its default /tmp path. A clean focused Linux testshards rerun
without that override passed (13.653s). This diagnostic issue caused no repository
code change. Raw rerun evidence lives in /private/tmp/picfetch-store-verification.

Final evidence: 52 non-UI packages passed in the retained Linux race stream;
`scripts/testshards` then passed its clean Linux race rerun in 13.653s. Together
with all three passing UI race shards, every race-test package has passed.
Formatting, TUF/Qodana checks, host vet/build, Windows command cross-build,
workflow YAML parsing, wizard syntax and diff whitespace checks also passed.
The combined `make verify` exit was still failure after the OOM and is not
represented as a successful single run. Remaining work is external activation,
CI credential proof and the next ordinary Store publication (tickets 01/06).

## Approval requirement amendment (2026-09-08; supersedes unattended policy above)

Deep route, Lead inline, zero new delegates. Keep this plan open for user review.
Deliverable: each Store mutation is bound to a reviewed artifact and frozen notes;
all live operations remain outside this local change. Existing command/fake-service
and workflow contract tests are the accepted seams; no new testing seam is needed.

- Add a GitHub-only `prepare` operation. It freezes release/run/attempt/artifact,
  bundle/WACK digests, generated notes, expected Store base and receipt identity
  in a JSON approval artifact before the environment job starts. Initial base is
  the confirmed live 1.0.2; later bases come from published receipts. An active
  receipt requires explicit reconciliation first. No Microsoft credentials here.
- `submit` and `reconcile` require that exact manifest and its SHA-256 from the
  preparation job. Submission restores the exact artifact; reconciliation only
  resumes the selected receipt and never selects another release. Stale receipts,
  bases, tags or artifacts fail closed. Rebuilds/new tags cannot replace approval.
- Preserve main-only trusted code, pinned to the run's main SHA across both jobs,
  one workflow concurrency group, durable intent/receipt recovery, version checks
  and metadata preservation. Successful tag builds prepare their producing run;
  manual submission can select the newest candidate before approval.
- Remove cron. Microsoft certifies/publishes after an approved commit without a
  runner. Manual `check` and `reconcile` each require environment approval because
  the same protected secrets are needed; no periodic approval requests or copied
  credentials. Reconcile, then dispatch submit for a waiting release, with a fresh
  approval. Result collection is deliberately manual after the job exits.
- Live metadata verified: required reviewer frathe, main-only branch rule, self
  review allowed, bypass/timer/custom rules disabled, all three secret names exist.
  User confirmed the linked application uses Developer and rotated its key.
  Existing main has no publisher workflow; no Microsoft credential test is possible
  here without landing the trusted workflow and approving a read-only check job.

| Task | Files | Test / verification | Owner / budget |
| --- | --- | --- | --- |
| Approval contract | scripts/storepublish command, lifecycle, GitHub adapter, approval.go, existing tests | `go test ./scripts/storepublish -run 'TestStorePublish(Approval|Reconcile|Recovery|Submission)'` | Lead; 0 spawns; focused only |
| Workflow gate | publisher YAML, msixstage existing tests, environment jq policy | `go test ./scripts/msixstage -run TestStoreWorkflowPublishingContract`; negative guard checks; YAML/actionlint | Lead; 0 spawns |
| Operational amendment | setup wizard, docs, spec/tickets/README, architecture, todos | `bash -n .scratch/microsoft-store-updates/configure-access.sh`; policy/terminology review | Lead; 0 spawns |
| Final gate | changed tooling and repository checks | focused race; Make fmt/TUF/Qodana/vet/build/shards | Lead; 0 spawns; reuse prior full race partition evidence per user instruction, no repeated Docker OOM run |

Task graph: approval contract -> workflow gate -> handoff; setup/docs follow the
same decisions. Final review and fixes stay with Lead. No commit, push, tag,
release, submission or remote protection changes are authorized.

### Approval amendment verification and handoff

- Red: `TestStorePublishApproval` demonstrated that unapproved submission previously
  succeeded and `prepare` was absent. The workflow contract also failed against
  cron, floating-main checkout and missing approval-artifact binding. Implemented
  and reran green at the established command/workflow seams.
- `go test -race ./scripts/storepublish ./scripts/msixstage ./scripts/releasenotes`
  passed, including the executable jq policy fixtures (valid policy plus eight
  invalid rule variants), frozen approval tampering/staleness, exact triggering run,
  no postapproval discovery, recovery, metadata/version guards and separate approval
  for a newer waiting release. The loopback-listener sandbox denial was resolved by
  running the local fake-service tests with approved sandbox escalation.
- Five deliberate mutations (approval digest, approved operation, fixed publisher
  checkout, no cron, required reviewer identity) failed their test assertions. Every
  file was restored and the focused race suite reran green. New tests remain in
  existing test files; the earlier Qodana registration is unchanged.
- `make fmt-check check-tuf-root check-qodana-test-exclusions vet build
  check-test-shards` passed. The shard inventory ran in Linux/amd64 Docker and
  reported 671 runnables across 3 shards. The host linker emitted its existing
  duplicate `-lobjc` warning but the build succeeded.
- `actionlint v1.7.12` passed both Store workflows. `bash -n` passed the amended
  wizard, local document links resolve, and `git diff --check` is clean. The wizard
  was not run interactively and no credential values were requested or captured.
- The restored command cross-build passed with `GOOS=windows GOARCH=amd64
  CGO_ENABLED=0 go build -o /private/tmp/picfetch-storepublish-approval-windows.exe
  ./scripts/storepublish`. This is compile evidence, not Windows SDK/WACK execution.
- Read-only GitHub metadata matched the exact new jq policy. All three secret names
  exist. The installed Store package workflow has no publisher/credential-check
  step, and the publisher workflow is absent from the installed workflow list.
  No trusted existing workflow supports the requested read-only Microsoft check.
- No full race repeat: per the user instruction, retain the original combined
  `make verify` Docker OOM and the successful isolated race partitions as prior
  evidence. Do not describe the old combined invocation as successful or imply that
  this amendment reran the complete application race suite.

Remaining live work: review/land the uncommitted implementation on main, dispatch
and approve `check`, record actual product read access, then approve the next normal
stable release and observe submission permission and eventual Published state.
The narrower Developer role remains deliberate; a read-only success cannot prove
mutation permission. No permission elevation or live settings change was performed.
All local requested work is complete. No user architecture decision is pending.
Changes remain uncommitted in the existing checkout and branch; nothing was pushed,
tagged, released or submitted to Microsoft.

| Amendment work | Spawns budget / actual | Review | Full suite |
| --- | --- | --- | --- |
| Approval contract and workflow | 0 / 0 | Lead, inline fixes and negative guards | focused race only |
| Policy/docs/setup and handoff | 0 / 0 | Lead, live metadata read and static checks | prior full-partition evidence retained |
